package ebay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ErrorClass string

const (
	ErrorValidation ErrorClass = "VALIDATION"
	ErrorAuth       ErrorClass = "AUTH"
	ErrorRateLimit  ErrorClass = "RATE_LIMIT"
	ErrorTransient  ErrorClass = "TRANSIENT"
	ErrorUnknown    ErrorClass = "UNKNOWN_RESULT"
)

type APIError struct {
	Class      ErrorClass
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("eBay %s (%d): %s", e.Class, e.StatusCode, e.Message)
}

func (e *APIError) ErrorClass() string { return string(e.Class) }

type Listing struct {
	SKU                   string
	Title                 string
	Description           string
	CategoryID            string
	Condition             string
	Currency              string
	PriceCents            int64
	Quantity              int
	ImageURLs             []string
	Aspects               map[string][]string
	Marketplace           string
	MerchantLocationKey   string
	PaymentPolicyID       string
	ReturnPolicyID        string
	FulfillmentPolicyID   string
	ManufacturerName      string
	ManufacturerAddress   string
	ResponsiblePersonName string
	ResponsibleAddress    string
	SafetyInformation     string
	SafetyStatementIDs    []string
	GPSROverridden        bool
}

type PublishResult struct {
	OfferID          string         `json:"offerId"`
	ListingID        string         `json:"listingId"`
	ListingURL       string         `json:"listingUrl"`
	RequestArtifact  map[string]any `json:"requestArtifact"`
	ResponseArtifact map[string]any `json:"responseArtifact"`
	DryRun           bool           `json:"dryRun"`
}

type OfferSnapshot struct {
	OfferID    string         `json:"offerId"`
	PriceCents int64          `json:"priceCents"`
	Quantity   int            `json:"quantity"`
	Status     string         `json:"status"`
	Raw        map[string]any `json:"raw"`
}

type Client struct {
	Config     RuntimeConfig
	HTTPClient *http.Client
}

func CentsDecimal(cents int64) string {
	negative := cents < 0
	if negative {
		cents = -cents
	}
	value := strconv.FormatInt(cents/100, 10) + "." + fmt.Sprintf("%02d", cents%100)
	if negative {
		return "-" + value
	}
	return value
}

func (c Client) http() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: c.Config.Timeout}
}

func (c Client) Publish(ctx context.Context, token, automationMode string, listing Listing) (PublishResult, error) {
	if err := validateListing(listing); err != nil {
		return PublishResult{}, err
	}
	artifact := map[string]any{"sku": listing.SKU, "marketplace": listing.Marketplace, "categoryId": listing.CategoryID, "price": CentsDecimal(listing.PriceCents), "currency": listing.Currency, "quantity": listing.Quantity, "imageCount": len(listing.ImageURLs)}
	if strings.EqualFold(strings.TrimSpace(automationMode), "DRY_RUN") {
		return PublishResult{DryRun: true, RequestArtifact: artifact, ResponseArtifact: map[string]any{"status": "DRY_RUN", "mutatingCalls": 0}}, nil
	}
	if c.Config.Environment == "production" && !strings.EqualFold(strings.TrimSpace(automationMode), "LIVE") {
		return PublishResult{}, fmt.Errorf("production eBay publishing requires AUTOMATION_MODE=LIVE")
	}
	if strings.TrimSpace(token) == "" {
		return PublishResult{}, &APIError{Class: ErrorAuth, Message: "user access token is required"}
	}
	item := map[string]any{"availability": map[string]any{"shipToLocationAvailability": map[string]any{"quantity": listing.Quantity}}, "condition": listing.Condition, "product": map[string]any{"title": listing.Title, "description": listing.Description, "aspects": listing.Aspects, "imageUrls": listing.ImageURLs}}
	if _, err := c.call(ctx, token, http.MethodPut, "/sell/inventory/v1/inventory_item/"+url.PathEscape(listing.SKU), item); err != nil {
		return PublishResult{}, err
	}
	offerBody := map[string]any{"sku": listing.SKU, "marketplaceId": listing.Marketplace, "format": "FIXED_PRICE", "categoryId": listing.CategoryID, "merchantLocationKey": listing.MerchantLocationKey, "availableQuantity": listing.Quantity, "listingDuration": "GTC", "listingPolicies": map[string]any{"paymentPolicyId": listing.PaymentPolicyID, "returnPolicyId": listing.ReturnPolicyID, "fulfillmentPolicyId": listing.FulfillmentPolicyID}, "pricingSummary": map[string]any{"price": map[string]any{"currency": listing.Currency, "value": CentsDecimal(listing.PriceCents)}}}
	if listing.ManufacturerName != "" {
		regulatory := map[string]any{"manufacturer": map[string]any{"companyName": listing.ManufacturerName, "addressLine1": listing.ManufacturerAddress}, "responsiblePersons": []any{map[string]any{"companyName": listing.ResponsiblePersonName, "addressLine1": listing.ResponsibleAddress, "types": []string{"EU_RESPONSIBLE_PERSON"}}}}
		if len(listing.SafetyStatementIDs) > 0 {
			component := listing.SafetyInformation
			if len(component) > 120 {
				component = component[:120]
			}
			regulatory["productSafety"] = map[string]any{"statements": listing.SafetyStatementIDs, "component": component}
		}
		offerBody["regulatory"] = regulatory
	}
	offerRaw, err := c.call(ctx, token, http.MethodPost, "/sell/inventory/v1/offer", offerBody)
	if err != nil {
		if apiErr, ok := err.(*APIError); !ok || apiErr.Class != ErrorUnknown {
			return PublishResult{}, err
		}
		offerRaw, err = c.findOfferBySKU(ctx, token, listing.SKU)
		if err != nil {
			return PublishResult{}, err
		}
	}
	var offer struct {
		OfferID string `json:"offerId"`
	}
	if err := json.Unmarshal(offerRaw, &offer); err != nil || offer.OfferID == "" {
		return PublishResult{}, fmt.Errorf("eBay create offer response omitted offerId")
	}
	publishRaw, err := c.call(ctx, token, http.MethodPost, "/sell/inventory/v1/offer/"+url.PathEscape(offer.OfferID)+"/publish", map[string]any{})
	if err != nil {
		return PublishResult{}, err
	}
	var published struct {
		ListingID string `json:"listingId"`
	}
	if err := json.Unmarshal(publishRaw, &published); err != nil || published.ListingID == "" {
		return PublishResult{}, fmt.Errorf("eBay publish response omitted listingId")
	}
	listingURL := "https://www.ebay.de/itm/" + url.PathEscape(published.ListingID)
	if c.Config.Environment == "sandbox" {
		listingURL = "https://www.sandbox.ebay.de/itm/" + url.PathEscape(published.ListingID)
	}
	return PublishResult{OfferID: offer.OfferID, ListingID: published.ListingID, ListingURL: listingURL, RequestArtifact: artifact, ResponseArtifact: map[string]any{"offerId": offer.OfferID, "listingId": published.ListingID}}, nil
}

func (c Client) ReadOffer(ctx context.Context, token, offerID string) (OfferSnapshot, error) {
	if strings.TrimSpace(token) == "" {
		return OfferSnapshot{}, &APIError{Class: ErrorAuth, Message: "user access token is required"}
	}
	if strings.TrimSpace(offerID) == "" {
		return OfferSnapshot{}, &APIError{Class: ErrorValidation, Message: "offerId is required"}
	}
	raw, err := c.call(ctx, token, http.MethodGet, "/sell/inventory/v1/offer/"+url.PathEscape(offerID), nil)
	if err != nil {
		return OfferSnapshot{}, err
	}
	return parseOfferSnapshot(raw, offerID)
}

func (c Client) UpdateOffer(ctx context.Context, token, automationMode, offerID, currency string, priceCents int64) (OfferSnapshot, map[string]any, bool, error) {
	artifact := map[string]any{"offerId": offerID, "price": CentsDecimal(priceCents), "currency": currency}
	if strings.EqualFold(strings.TrimSpace(automationMode), "DRY_RUN") {
		return OfferSnapshot{OfferID: offerID, PriceCents: priceCents, Status: "DRY_RUN", Raw: map[string]any{"status": "DRY_RUN"}}, artifact, true, nil
	}
	if c.Config.Environment == "production" && !strings.EqualFold(strings.TrimSpace(automationMode), "LIVE") {
		return OfferSnapshot{}, nil, false, fmt.Errorf("production eBay offer update requires AUTOMATION_MODE=LIVE")
	}
	if strings.TrimSpace(token) == "" {
		return OfferSnapshot{}, nil, false, &APIError{Class: ErrorAuth, Message: "user access token is required"}
	}
	if strings.TrimSpace(offerID) == "" || strings.TrimSpace(currency) == "" || priceCents <= 0 {
		return OfferSnapshot{}, nil, false, &APIError{Class: ErrorValidation, Message: "offerId, currency and positive price are required"}
	}
	body := map[string]any{"pricingSummary": map[string]any{"price": map[string]any{"currency": currency, "value": CentsDecimal(priceCents)}}}
	if _, err := c.call(ctx, token, http.MethodPut, "/sell/inventory/v1/offer/"+url.PathEscape(offerID), body); err != nil {
		return OfferSnapshot{}, nil, false, err
	}
	verified, err := c.ReadOffer(ctx, token, offerID)
	if err != nil {
		return OfferSnapshot{}, nil, false, err
	}
	return verified, artifact, false, nil
}

func parseOfferSnapshot(raw []byte, fallbackOfferID string) (OfferSnapshot, error) {
	var body struct {
		OfferID           string `json:"offerId"`
		Status            string `json:"status"`
		AvailableQuantity int    `json:"availableQuantity"`
		PricingSummary    struct {
			Price struct {
				Value    string `json:"value"`
				Currency string `json:"currency"`
			} `json:"price"`
		} `json:"pricingSummary"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return OfferSnapshot{}, fmt.Errorf("decode eBay offer: %w", err)
	}
	value, err := decimalCents(body.PricingSummary.Price.Value)
	if err != nil {
		return OfferSnapshot{}, fmt.Errorf("decode eBay offer price: %w", err)
	}
	if body.OfferID == "" {
		body.OfferID = fallbackOfferID
	}
	var rawMap map[string]any
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		return OfferSnapshot{}, err
	}
	return OfferSnapshot{OfferID: body.OfferID, PriceCents: value, Quantity: body.AvailableQuantity, Status: body.Status, Raw: rawMap}, nil
}

func decimalCents(value string) (int64, error) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ".")
	if len(parts) > 2 || len(parts) == 0 || parts[0] == "" {
		return 0, fmt.Errorf("invalid decimal")
	}
	euros, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || euros < 0 {
		return 0, fmt.Errorf("invalid decimal")
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if len(fraction) > 2 {
		return 0, fmt.Errorf("more than two decimal places")
	}
	for len(fraction) < 2 {
		fraction += "0"
	}
	cents := int64(0)
	if fraction != "" {
		cents, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid decimal")
		}
	}
	if euros > (math.MaxInt64-cents)/100 {
		return 0, fmt.Errorf("decimal overflows cents")
	}
	return euros*100 + cents, nil
}

func validateListing(row Listing) error {
	if row.SKU == "" || row.Title == "" || row.Description == "" || row.CategoryID == "" || row.PriceCents <= 0 || row.Quantity <= 0 || len(row.ImageURLs) == 0 || row.MerchantLocationKey == "" || row.PaymentPolicyID == "" || row.ReturnPolicyID == "" || row.FulfillmentPolicyID == "" {
		return &APIError{Class: ErrorValidation, Message: "inventory item, offer, policy, location, image and price fields are required"}
	}
	if !row.GPSROverridden && (row.ManufacturerName == "" || row.ManufacturerAddress == "" || row.ResponsiblePersonName == "" || row.ResponsibleAddress == "" || row.SafetyInformation == "") {
		return &APIError{Class: ErrorValidation, Message: "complete GPSR data is required"}
	}
	return nil
}

func (c Client) findOfferBySKU(ctx context.Context, token, sku string) ([]byte, error) {
	raw, err := c.call(ctx, token, http.MethodGet, "/sell/inventory/v1/offer?sku="+url.QueryEscape(sku), nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Offers []json.RawMessage `json:"offers"`
	}
	if err := json.Unmarshal(raw, &result); err != nil || len(result.Offers) == 0 {
		return nil, &APIError{Class: ErrorUnknown, Message: "create offer timed out and reconciliation found no offer"}
	}
	return result.Offers[0], nil
}

func (c Client) call(ctx context.Context, token, method, path string, payload any) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.Config.APIBaseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Language", "de-DE")
	res, err := c.http().Do(req)
	if err != nil {
		return nil, &APIError{Class: ErrorUnknown, Message: "request outcome is unknown; reconcile before retry"}
	}
	defer res.Body.Close()
	raw, readErr := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if readErr != nil {
		return nil, &APIError{Class: ErrorUnknown, StatusCode: res.StatusCode, Message: "response could not be read"}
	}
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return raw, nil
	}
	class := ErrorValidation
	if res.StatusCode == 401 || res.StatusCode == 403 {
		class = ErrorAuth
	} else if res.StatusCode == 429 {
		class = ErrorRateLimit
	} else if res.StatusCode >= 500 {
		class = ErrorTransient
	}
	message := strings.TrimSpace(string(raw))
	if len(message) > 500 {
		message = message[:500]
	}
	return nil, &APIError{Class: class, StatusCode: res.StatusCode, Message: message}
}

type TaxonomyAspect struct {
	Name     string   `json:"name"`
	Required bool     `json:"required"`
	Mode     string   `json:"mode"`
	Values   []string `json:"values"`
}

func (c Client) DefaultCategoryTreeID(ctx context.Context, token, marketplace string) (string, error) {
	raw, err := c.call(ctx, token, http.MethodGet, "/commerce/taxonomy/v1/get_default_category_tree_id?marketplace_id="+url.QueryEscape(marketplace), nil)
	if err != nil {
		return "", err
	}
	var result struct {
		CategoryTreeID string `json:"categoryTreeId"`
	}
	if err := json.Unmarshal(raw, &result); err != nil || strings.TrimSpace(result.CategoryTreeID) == "" {
		return "", fmt.Errorf("eBay taxonomy response omitted categoryTreeId")
	}
	return result.CategoryTreeID, nil
}

func (c Client) CategoryAspects(ctx context.Context, token, treeID, categoryID string) ([]TaxonomyAspect, json.RawMessage, error) {
	path := "/commerce/taxonomy/v1/category_tree/" + url.PathEscape(treeID) + "/get_item_aspects_for_category?category_id=" + url.QueryEscape(categoryID)
	raw, err := c.call(ctx, token, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	var response struct {
		Aspects []struct {
			Name       string `json:"localizedAspectName"`
			Constraint struct {
				Required bool   `json:"aspectRequired"`
				Mode     string `json:"itemToAspectCardinality"`
			} `json:"aspectConstraint"`
			Values []struct {
				Value string `json:"localizedValue"`
			} `json:"aspectValues"`
		} `json:"aspects"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, raw, err
	}
	out := make([]TaxonomyAspect, 0, len(response.Aspects))
	for _, aspect := range response.Aspects {
		item := TaxonomyAspect{Name: aspect.Name, Required: aspect.Constraint.Required, Mode: aspect.Constraint.Mode}
		for _, value := range aspect.Values {
			item.Values = append(item.Values, value.Value)
		}
		out = append(out, item)
	}
	return out, raw, nil
}

var _ = time.Second
