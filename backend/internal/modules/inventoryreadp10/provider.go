package inventoryreadp10

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/config"
	"github.com/trademind-ai/trademind/backend/internal/modules/credentialp10"
	"github.com/trademind-ai/trademind/backend/internal/modules/inventorysyncp9"
	"github.com/trademind-ai/trademind/backend/internal/modules/productioncontrolp10"
	"github.com/trademind-ai/trademind/backend/internal/modules/productpublish"
	"github.com/trademind-ai/trademind/backend/internal/providers/platform/douyinshop"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const ProviderModeRealReadOnly = "real_readonly"

type ErrorCode string

const (
	ErrorUnauthorized        ErrorCode = "Unauthorized"
	ErrorCredentialExpired   ErrorCode = "CredentialExpired"
	ErrorRateLimited         ErrorCode = "RateLimited"
	ErrorProviderUnavailable ErrorCode = "ProviderUnavailable"
	ErrorInvalidRequest      ErrorCode = "InvalidRequest"
	ErrorProviderProtocol    ErrorCode = "ProviderProtocolError"
)

type ProviderError struct {
	Code              ErrorCode
	InternalRequestID string
	ProviderRequestID string
	RetryAfter        time.Duration
	Timeout           bool
	Cause             error
}

func (e *ProviderError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("p10 provider error: code=%s requestId=%s providerRequestId=%s", e.Code, e.InternalRequestID, e.ProviderRequestID)
}

func (e *ProviderError) Unwrap() error { return e.Cause }

type CredentialResolver interface {
	ResolveActive(context.Context, int64, string, uuid.UUID) (credentialp10.RuntimeCredential, error)
}

type ReadGuard interface {
	EvaluateRead(context.Context, int64, uuid.UUID, int) error
}

type publicationPage struct {
	PublicationID     uuid.UUID
	ExternalProductID string
	ProductTitle      string
	SKUCodes          map[string]string
}

type PublicationSource struct{ DB *gorm.DB }

func (s PublicationSource) CountSKU(ctx context.Context, tenantID int64, shopID uuid.UUID) (int, error) {
	var count int64
	err := s.DB.WithContext(ctx).Table("product_publication_skus AS sku").Joins("JOIN product_publications AS pub ON pub.id = sku.publication_id").Joins("JOIN shops AS shop ON shop.id = pub.shop_id").Where("shop.tenant_id = ? AND pub.shop_id = ? AND pub.platform IN ? AND pub.external_product_id <> ''", tenantID, shopID, []string{"douyin", "douyin_shop"}).Count(&count).Error
	return int(count), err
}

func (s PublicationSource) Page(ctx context.Context, tenantID int64, shopID uuid.UUID, offset, limit int) ([]publicationPage, error) {
	var publications []productpublish.ProductPublication
	if err := s.DB.WithContext(ctx).Table("product_publications AS pub").Select("pub.*").Joins("JOIN shops AS shop ON shop.id = pub.shop_id").Where("shop.tenant_id = ? AND pub.shop_id = ? AND pub.platform IN ? AND pub.external_product_id <> ''", tenantID, shopID, []string{"douyin", "douyin_shop"}).Order("pub.id ASC").Offset(offset).Limit(limit).Find(&publications).Error; err != nil {
		return nil, err
	}
	out := make([]publicationPage, 0, len(publications))
	for _, publication := range publications {
		var skus []productpublish.ProductPublicationSKU
		if err := s.DB.WithContext(ctx).Where("publication_id = ?", publication.ID).Order("id ASC").Find(&skus).Error; err != nil {
			return nil, err
		}
		codes := make(map[string]string, len(skus))
		for _, sku := range skus {
			if externalID := strings.TrimSpace(sku.ExternalSKUID); externalID != "" {
				codes[externalID] = strings.TrimSpace(sku.SKUCode)
			}
		}
		out = append(out, publicationPage{PublicationID: publication.ID, ExternalProductID: strings.TrimSpace(publication.ExternalProductID), ProductTitle: strings.TrimSpace(publication.Title), SKUCodes: codes})
	}
	return out, nil
}

type providerCursor struct {
	Version int `json:"version"`
	Offset  int `json:"offset"`
}

type DouyinReadOnlyInventoryProvider struct {
	Config      *config.Config
	Credentials CredentialResolver
	Guard       ReadGuard
	Source      PublicationSource
}

type requestIDContextKey struct{}

func withInternalRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, strings.TrimSpace(requestID))
}

func internalRequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDContextKey{}).(string)
	return strings.TrimSpace(value)
}

var _ inventorysyncp9.InventoryProvider = (*DouyinReadOnlyInventoryProvider)(nil)

func (p *DouyinReadOnlyInventoryProvider) Key() inventorysyncp9.InventoryProviderKey {
	return inventorysyncp9.InventoryProviderKey{Platform: inventorysyncp9.PlatformDouyin, ProviderMode: ProviderModeRealReadOnly}
}

func (p *DouyinReadOnlyInventoryProvider) Capabilities() inventorysyncp9.InventoryProviderCapabilities {
	return inventorysyncp9.InventoryProviderCapabilities{NetworkAccess: true, OAuth: true, Credentials: true, RealCredentials: true, RealPlatformRead: true, RealInventoryRead: true}
}

func (p *DouyinReadOnlyInventoryProvider) FetchInventoryPage(ctx context.Context, request inventorysyncp9.InventoryFetchRequest) (inventorysyncp9.InventoryFetchPageResult, error) {
	requestID := internalRequestID(ctx)
	if p == nil || p.Config == nil || p.Credentials == nil || p.Guard == nil || p.Source.DB == nil || request.TenantID <= 0 {
		return inventorysyncp9.InventoryFetchPageResult{}, &ProviderError{Code: ErrorProviderUnavailable, InternalRequestID: requestID}
	}
	shopID, err := uuid.Parse(strings.TrimSpace(request.ShopConnectionID))
	if err != nil || strings.ToLower(strings.TrimSpace(request.Platform)) != inventorysyncp9.PlatformDouyin || strings.ToLower(strings.TrimSpace(request.ProviderMode)) != ProviderModeRealReadOnly {
		return inventorysyncp9.InventoryFetchPageResult{}, &ProviderError{Code: ErrorInvalidRequest, InternalRequestID: requestID, Cause: err}
	}
	skuCount, err := p.Source.CountSKU(ctx, request.TenantID, shopID)
	if err != nil {
		return inventorysyncp9.InventoryFetchPageResult{}, &ProviderError{Code: ErrorProviderUnavailable, InternalRequestID: requestID, Cause: err}
	}
	if err := p.Guard.EvaluateRead(ctx, request.TenantID, shopID, skuCount); err != nil {
		return inventorysyncp9.InventoryFetchPageResult{}, &ProviderError{Code: ErrorUnauthorized, InternalRequestID: requestID, Cause: err}
	}
	credential, err := p.Credentials.ResolveActive(ctx, request.TenantID, inventorysyncp9.PlatformDouyin, shopID)
	if err != nil {
		return inventorysyncp9.InventoryFetchPageResult{}, &ProviderError{Code: ErrorCredentialExpired, InternalRequestID: requestID, Cause: err}
	}
	cursor, err := decodeCursor(request.Cursor)
	if err != nil {
		return inventorysyncp9.InventoryFetchPageResult{}, &ProviderError{Code: ErrorInvalidRequest, InternalRequestID: requestID, Cause: err}
	}
	pageSize := request.PageSize
	if pageSize <= 0 {
		pageSize = p.Config.P10.SKUPageSize
	}
	if pageSize < 1 || pageSize > 100 {
		return inventorysyncp9.InventoryFetchPageResult{}, &ProviderError{Code: ErrorInvalidRequest, InternalRequestID: requestID}
	}
	// Pagination is over locally bound products because the repository has no official all-inventory endpoint contract.
	publications, err := p.Source.Page(ctx, request.TenantID, shopID, cursor.Offset, pageSize)
	if err != nil {
		return inventorysyncp9.InventoryFetchPageResult{}, &ProviderError{Code: ErrorProviderUnavailable, InternalRequestID: requestID, Cause: err}
	}
	client := p.client(shopID, credential)
	items := make([]inventorysyncp9.InventoryProviderItem, 0)
	for _, publication := range publications {
		detail, err := client.GetProductDetail(ctx, shopID.String(), publication.ExternalProductID)
		if err != nil {
			return inventorysyncp9.InventoryFetchPageResult{}, mapDouyinError(err, requestID)
		}
		for _, sku := range detail.SKUs {
			code := strings.TrimSpace(sku.OuterSKUID)
			if code == "" {
				code = publication.SKUCodes[strings.TrimSpace(sku.PlatformSKUID)]
			}
			items = append(items, inventorysyncp9.InventoryProviderItem{ExternalProductID: detail.PlatformProductID, ExternalSKUID: sku.PlatformSKUID, ExternalProductCode: publication.ExternalProductID, ExternalSKUCode: code, ProductTitle: firstNonEmpty(detail.Name, publication.ProductTitle), VariantTitle: sku.SpecName, AvailableQuantity: sku.Stock, TotalQuantity: sku.Stock, SafeMetadata: map[string]string{"operation": douyinshop.MethodProductDetail, "requestId": requestID, "providerRequestId": detail.RequestID, "publicationId": publication.PublicationID.String(), "readOnly": "true"}})
			if len(items) > 100 || (request.MaxItemsPerPage > 0 && len(items) > request.MaxItemsPerPage) {
				return inventorysyncp9.InventoryFetchPageResult{}, &ProviderError{Code: ErrorProviderProtocol, InternalRequestID: requestID, Cause: errors.New("provider page exceeds approved SKU limit")}
			}
		}
	}
	next := providerCursor{Version: 1, Offset: cursor.Offset + len(publications)}
	nextRaw, _ := json.Marshal(next)
	currentRaw, _ := json.Marshal(cursor)
	hasMore := len(publications) == pageSize
	return inventorysyncp9.InventoryFetchPageResult{Items: items, Cursor: datatypes.JSON(currentRaw), NextCursor: datatypes.JSON(nextRaw), HasMore: hasMore, Scenario: "real_readonly", FixtureHash: "", NetworkCalls: len(publications)}, nil
}

func (p *DouyinReadOnlyInventoryProvider) client(shopID uuid.UUID, credential credentialp10.RuntimeCredential) *douyinshop.Client {
	transport := &http.Transport{Proxy: http.ProxyFromEnvironment, DialContext: (&net.Dialer{Timeout: p.Config.P10.ProviderConnectTimeout, KeepAlive: 30 * time.Second}).DialContext, ForceAttemptHTTP2: true, MaxIdleConns: 20, MaxIdleConnsPerHost: p.Config.P10.ProviderConcurrency, MaxConnsPerHost: p.Config.P10.ProviderConcurrency, IdleConnTimeout: 60 * time.Second, TLSHandshakeTimeout: p.Config.P10.ProviderConnectTimeout, ResponseHeaderTimeout: p.Config.P10.ProviderResponseHeaderTime}
	doer := &limitedHTTPDoer{client: &http.Client{Transport: transport, Timeout: p.Config.P10.ProviderRequestTimeout}, maxBytes: p.Config.P10.ProviderMaxResponseBytes}
	return &douyinshop.Client{ShopID: shopID.String(), Config: douyinshop.RuntimeConfig{AppKey: credential.ClientID, AppSecret: string(credential.ClientSecret), APIBaseURL: p.Config.P10.DouyinAPIBaseURL, Environment: p.Config.AppEnv, HTTPTimeout: p.Config.P10.ProviderRequestTimeout, RealAPIEnabled: true, InventoryEnabled: false, WriteOperationsEnabled: false, ScheduledInventorySyncEnabled: false}, HTTP: doer, AccessToken: string(credential.Bearer), RefreshTokenValue: ""}
}

type limitedHTTPDoer struct {
	client   *http.Client
	maxBytes int64
}

func (d *limitedHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.Body != nil && d.maxBytes > 0 {
		resp.Body = &limitedReadCloser{Reader: io.LimitReader(resp.Body, d.maxBytes+1), closer: resp.Body, remaining: d.maxBytes + 1}
	}
	return resp, nil
}

type limitedReadCloser struct {
	io.Reader
	closer    io.Closer
	remaining int64
}

func (r *limitedReadCloser) Read(buffer []byte) (int, error) {
	n, err := r.Reader.Read(buffer)
	r.remaining -= int64(n)
	if r.remaining <= 0 && err == nil {
		return n, errors.New("provider response exceeds configured size limit")
	}
	return n, err
}

func (r *limitedReadCloser) Close() error { return r.closer.Close() }

func decodeCursor(raw datatypes.JSON) (providerCursor, error) {
	if len(raw) == 0 || string(raw) == "{}" || string(raw) == "null" {
		return providerCursor{Version: 1}, nil
	}
	var cursor providerCursor
	if err := json.Unmarshal(raw, &cursor); err != nil || cursor.Version != 1 || cursor.Offset < 0 {
		return providerCursor{}, errors.New("invalid provider cursor")
	}
	return cursor, nil
}

func mapDouyinError(err error, requestID string) error {
	var de *douyinshop.Error
	if !douyinshop.AsError(err, &de) || de == nil {
		return &ProviderError{Code: ErrorProviderUnavailable, InternalRequestID: requestID, Timeout: errors.Is(err, context.DeadlineExceeded), Cause: err}
	}
	code := ErrorProviderProtocol
	switch {
	case de.AuthExpired:
		code = ErrorCredentialExpired
	case de.PermissionDenied:
		code = ErrorUnauthorized
	case de.RateLimited:
		code = ErrorRateLimited
	case de.ErrorClass == douyinshop.ErrorClassTimeout || de.ErrorClass == douyinshop.ErrorClassNetwork:
		code = ErrorProviderUnavailable
	case de.ErrorClass == douyinshop.ErrorClassValidation:
		code = ErrorInvalidRequest
	}
	return &ProviderError{Code: code, InternalRequestID: requestID, ProviderRequestID: de.RequestID, RetryAfter: time.Duration(de.RetryAfter) * time.Second, Timeout: de.ErrorClass == douyinshop.ErrorClassTimeout, Cause: err}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

var _ ReadGuard = (*productioncontrolp10.Service)(nil)
