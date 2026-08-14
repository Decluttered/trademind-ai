package platformcredential

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/pkg/metrics"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Actor struct {
	TenantID  int64
	UserID    uuid.UUID
	RequestID string
}

type CredentialStatus string

type CredentialRotation struct {
	CredentialID     uuid.UUID `json:"credentialId"`
	ExpectedRevision int       `json:"expectedRevision"`
}

type CredentialRevocation struct {
	CredentialID     uuid.UUID `json:"credentialId"`
	ExpectedRevision int       `json:"expectedRevision"`
}

type OAuthProvider interface {
	BuildAuthorizationURL(context.Context, Actor, string, uuid.UUID, string) (string, error)
	ExchangeAuthorizationCode(context.Context, Actor, string) (*CredentialMetadata, error)
	RefreshCredential(context.Context, Actor, uuid.UUID, int) (*CredentialMetadata, error)
	RevokeCredential(context.Context, Actor, uuid.UUID, int) (*CredentialMetadata, error)
}

type CredentialMetadata struct {
	CredentialID uuid.UUID  `json:"credentialId"`
	TenantID     int64      `json:"tenantId"`
	Platform     string     `json:"platform"`
	ShopID       uuid.UUID  `json:"shopId"`
	Status       string     `json:"status"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	RotatedAt    *time.Time `json:"rotatedAt,omitempty"`
	RevokedAt    *time.Time `json:"revokedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	Version      int        `json:"version"`
	KeyReference string     `json:"keyReference,omitempty"`
	Algorithm    string     `json:"algorithm,omitempty"`
}

type Service struct {
	DB                *gorm.DB
	Cipher            CredentialCipher
	Now               func() time.Time
	RedirectAllowlist []string
	OAuthStateTTL     time.Duration
	OfflineEnabled    bool
	Metrics           *metrics.Catalog
	Environment       string
}

func (s *Service) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s *Service) ready() error {
	if s == nil || s.DB == nil || s.Cipher == nil {
		return ErrKeyUnavailable
	}
	return nil
}

func credentialAAD(tenantID int64, platform string, credentialID uuid.UUID, version int) []byte {
	return []byte(fmt.Sprintf("tenant=%d;platform=%s;credential=%s;version=%d", tenantID, strings.ToLower(strings.TrimSpace(platform)), credentialID.String(), version))
}

func randomSecret(bytes int) (string, error) {
	raw := make([]byte, bytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// CreateOfflineCredential creates development-only fixture material without accepting tokens in an API DTO.
func (s *Service) CreateOfflineCredential(ctx context.Context, actor Actor, platform string, shopID uuid.UUID) (*CredentialMetadata, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if !s.OfflineEnabled || actor.TenantID <= 0 || actor.UserID == uuid.Nil || shopID == uuid.Nil {
		return nil, ErrCredentialUseDenied
	}
	platform = strings.ToLower(strings.TrimSpace(platform))
	if platform != "douyin" {
		return nil, ErrCredentialUseDenied
	}
	bearer, err := randomSecret(32)
	if err != nil {
		return nil, fmt.Errorf("create offline credential: %w", err)
	}
	renewal, err := randomSecret(32)
	if err != nil {
		return nil, fmt.Errorf("create offline credential: %w", err)
	}
	clientSecret, err := randomSecret(32)
	if err != nil {
		return nil, fmt.Errorf("create offline credential: %w", err)
	}
	expires := s.now().Add(time.Hour)
	root := PlatformCredential{TenantID: actor.TenantID, Platform: platform, ShopID: shopID, Status: StatusActive, ActiveVersion: 1, ExpiresAt: &expires, Revision: 1}
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing PlatformCredential
		lookup := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND platform = ? AND shop_id = ?", actor.TenantID, platform, shopID).First(&existing)
		if lookup.Error == nil {
			return ErrCredentialConflict
		}
		if !errors.Is(lookup.Error, gorm.ErrRecordNotFound) {
			return lookup.Error
		}
		if err := tx.Create(&root).Error; err != nil {
			return err
		}
		plain, err := encodeSecretPayload(RuntimeCredential{ClientID: "offline-fixture-app", ClientSecret: []byte(clientSecret), Bearer: []byte(bearer), Renewal: []byte(renewal), Version: 1, ExpiresAt: expires.Unix()})
		if err != nil {
			return err
		}
		envelope, err := s.Cipher.Encrypt(ctx, plain, credentialAAD(root.TenantID, root.Platform, root.ID, 1))
		if err != nil {
			return err
		}
		now := s.now()
		version := CredentialVersion{TenantID: root.TenantID, PlatformCredentialID: root.ID, Version: 1, Status: StatusActive, Ciphertext: envelope.Ciphertext, Nonce: envelope.Nonce, Algorithm: envelope.Algorithm, KeyReference: envelope.KeyReference, EnvelopeVersion: envelope.Version, ExpiresAt: &expires, ActivatedAt: &now}
		if err := tx.Create(&version).Error; err != nil {
			return err
		}
		if err := tx.Create(&OAuthCredential{TenantID: root.TenantID, PlatformCredentialID: root.ID, GrantType: "offline_fixture", Scopes: datatypes.JSON([]byte(`[]`))}).Error; err != nil {
			return err
		}
		if err := tx.Create(&CredentialBinding{TenantID: root.TenantID, PlatformCredentialID: root.ID, ShopID: root.ShopID, Platform: root.Platform}).Error; err != nil {
			return err
		}
		return appendLifecycleEvent(tx, actor, root.ID, 1, "credential_created", map[string]any{"platform": root.Platform, "shopId": root.ShopID.String(), "developmentOnly": true})
	})
	if err != nil {
		return nil, err
	}
	return s.metadata(ctx, actor.TenantID, root.ID)
}

func (s *Service) RotateOfflineCredential(ctx context.Context, actor Actor, credentialID uuid.UUID, expectedRevision int) (*CredentialMetadata, error) {
	if err := s.ready(); err != nil || !s.OfflineEnabled {
		return nil, ErrCredentialUseDenied
	}
	var root PlatformCredential
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND id = ?", actor.TenantID, credentialID).First(&root).Error; err != nil {
			return mapNotFound(err)
		}
		if root.Status == StatusRevoked || root.Revision != expectedRevision {
			return ErrCredentialConflict
		}
		bearer, err := randomSecret(32)
		if err != nil {
			return err
		}
		renewal, err := randomSecret(32)
		if err != nil {
			return err
		}
		clientSecret, err := randomSecret(32)
		if err != nil {
			return err
		}
		now := s.now()
		expires := now.Add(time.Hour)
		newVersion := root.ActiveVersion + 1
		plain, err := encodeSecretPayload(RuntimeCredential{ClientID: "offline-fixture-app", ClientSecret: []byte(clientSecret), Bearer: []byte(bearer), Renewal: []byte(renewal), Version: newVersion, ExpiresAt: expires.Unix()})
		if err != nil {
			return err
		}
		envelope, err := s.Cipher.Encrypt(ctx, plain, credentialAAD(root.TenantID, root.Platform, root.ID, newVersion))
		if err != nil {
			return err
		}
		if err := tx.Model(&CredentialVersion{}).Where("tenant_id = ? AND platform_credential_id = ? AND version = ? AND status = ?", root.TenantID, root.ID, root.ActiveVersion, StatusActive).Updates(map[string]any{"status": StatusRetired, "retired_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		version := CredentialVersion{TenantID: root.TenantID, PlatformCredentialID: root.ID, Version: newVersion, Status: StatusActive, Ciphertext: envelope.Ciphertext, Nonce: envelope.Nonce, Algorithm: envelope.Algorithm, KeyReference: envelope.KeyReference, EnvelopeVersion: envelope.Version, ExpiresAt: &expires, ActivatedAt: &now}
		if err := tx.Create(&version).Error; err != nil {
			return err
		}
		res := tx.Model(&PlatformCredential{}).Where("tenant_id = ? AND id = ? AND revision = ?", root.TenantID, root.ID, expectedRevision).Updates(map[string]any{"active_version": newVersion, "status": StatusActive, "expires_at": expires, "rotated_at": now, "revision": expectedRevision + 1, "updated_at": now})
		if res.Error != nil || res.RowsAffected != 1 {
			return ErrCredentialConflict
		}
		root.ActiveVersion = newVersion
		root.Revision = expectedRevision + 1
		return appendLifecycleEvent(tx, actor, root.ID, newVersion, "credential_rotated", map[string]any{"previousVersion": newVersion - 1})
	})
	if err != nil {
		return nil, err
	}
	return s.metadata(ctx, actor.TenantID, root.ID)
}

func (s *Service) Revoke(ctx context.Context, actor Actor, credentialID uuid.UUID, expectedRevision int) (*CredentialMetadata, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	var root PlatformCredential
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND id = ?", actor.TenantID, credentialID).First(&root).Error; err != nil {
			return mapNotFound(err)
		}
		if root.Revision != expectedRevision || root.Status == StatusRevoked {
			return ErrCredentialConflict
		}
		now := s.now()
		if err := tx.Model(&CredentialVersion{}).Where("tenant_id = ? AND platform_credential_id = ? AND status = ?", root.TenantID, root.ID, StatusActive).Updates(map[string]any{"status": StatusRevoked, "revoked_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		res := tx.Model(&PlatformCredential{}).Where("tenant_id = ? AND id = ? AND revision = ?", root.TenantID, root.ID, expectedRevision).Updates(map[string]any{"status": StatusRevoked, "revoked_at": now, "revision": expectedRevision + 1, "updated_at": now})
		if res.Error != nil || res.RowsAffected != 1 {
			return ErrCredentialConflict
		}
		root.Revision = expectedRevision + 1
		return appendLifecycleEvent(tx, actor, root.ID, root.ActiveVersion, "credential_revoked", map[string]any{"localOnly": true})
	})
	if err != nil {
		return nil, err
	}
	return s.metadata(ctx, actor.TenantID, root.ID)
}

func (s *Service) ResolveActive(ctx context.Context, tenantID int64, platform string, shopID uuid.UUID) (RuntimeCredential, error) {
	if err := s.ready(); err != nil {
		return RuntimeCredential{}, err
	}
	var root PlatformCredential
	if err := s.DB.WithContext(ctx).Where("tenant_id = ? AND platform = ? AND shop_id = ?", tenantID, strings.ToLower(strings.TrimSpace(platform)), shopID).First(&root).Error; err != nil {
		return RuntimeCredential{}, mapNotFound(err)
	}
	if root.Status != StatusActive || root.ExpiresAt == nil || !root.ExpiresAt.After(s.now()) {
		return RuntimeCredential{}, ErrCredentialUseDenied
	}
	if root.ExpiresAt.Before(s.now().Add(24 * time.Hour)) {
		s.Metrics.ObserveProductionCapabilities(s.Environment, root.Platform, "credential_resolve", "expiring", false, false, true, false, false, -1)
	}
	var version CredentialVersion
	if err := s.DB.WithContext(ctx).Where("tenant_id = ? AND platform_credential_id = ? AND version = ? AND status = ?", tenantID, root.ID, root.ActiveVersion, StatusActive).First(&version).Error; err != nil {
		return RuntimeCredential{}, mapNotFound(err)
	}
	plain, err := s.Cipher.Decrypt(ctx, EncryptedCredential{Ciphertext: version.Ciphertext, Nonce: version.Nonce, Algorithm: version.Algorithm, KeyReference: version.KeyReference, Version: version.EnvelopeVersion}, credentialAAD(root.TenantID, root.Platform, root.ID, version.Version))
	if err != nil {
		return RuntimeCredential{}, ErrCredentialUseDenied
	}
	expiresAt := int64(0)
	if version.ExpiresAt != nil {
		expiresAt = version.ExpiresAt.Unix()
	}
	return decodeSecretPayload(plain, version.Version, expiresAt)
}

func (s *Service) List(ctx context.Context, tenantID int64, allowedShopIDs []uuid.UUID) ([]CredentialMetadata, error) {
	if allowedShopIDs != nil && len(allowedShopIDs) == 0 {
		return []CredentialMetadata{}, nil
	}
	var rows []PlatformCredential
	query := s.DB.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if allowedShopIDs != nil {
		query = query.Where("shop_id IN ?", allowedShopIDs)
	}
	if err := query.Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]CredentialMetadata, 0, len(rows))
	for _, row := range rows {
		meta, err := s.metadataFromRoot(ctx, row)
		if err != nil {
			return nil, err
		}
		out = append(out, *meta)
	}
	return out, nil
}

func (s *Service) metadata(ctx context.Context, tenantID int64, id uuid.UUID) (*CredentialMetadata, error) {
	var root PlatformCredential
	if err := s.DB.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&root).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return s.metadataFromRoot(ctx, root)
}

func (s *Service) metadataFromRoot(ctx context.Context, root PlatformCredential) (*CredentialMetadata, error) {
	var version CredentialVersion
	if root.ActiveVersion > 0 {
		if err := s.DB.WithContext(ctx).Select("algorithm", "key_reference").Where("tenant_id = ? AND platform_credential_id = ? AND version = ?", root.TenantID, root.ID, root.ActiveVersion).First(&version).Error; err != nil {
			return nil, mapNotFound(err)
		}
	}
	status := root.Status
	if status == StatusActive && root.ExpiresAt != nil && !root.ExpiresAt.After(s.now()) {
		status = StatusExpired
	}
	return &CredentialMetadata{CredentialID: root.ID, TenantID: root.TenantID, Platform: root.Platform, ShopID: root.ShopID, Status: status, ExpiresAt: root.ExpiresAt, RotatedAt: root.RotatedAt, RevokedAt: root.RevokedAt, CreatedAt: root.CreatedAt, UpdatedAt: root.UpdatedAt, Version: root.Revision, KeyReference: version.KeyReference, Algorithm: version.Algorithm}, nil
}

func (s *Service) BuildOfflineAuthorizationURL(ctx context.Context, actor Actor, platform string, shopID uuid.UUID, redirectURI string) (string, error) {
	if err := s.ready(); err != nil {
		return "", err
	}
	if !s.OfflineEnabled || !redirectAllowed(redirectURI, s.RedirectAllowlist) || actor.TenantID <= 0 || actor.UserID == uuid.Nil || shopID == uuid.Nil || strings.ToLower(strings.TrimSpace(platform)) != "douyin" {
		return "", ErrRedirectNotAllowed
	}
	state, err := randomSecret(32)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(state))
	ttl := s.OAuthStateTTL
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	row := OAuthState{StateHash: hex.EncodeToString(hash[:]), TenantID: actor.TenantID, UserID: actor.UserID, Platform: "douyin", ShopID: shopID, RedirectURI: strings.TrimSpace(redirectURI), ExpiresAt: s.now().Add(ttl)}
	if err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return appendLifecycleEvent(tx, actor, uuid.Nil, 0, "oauth_authorization_started", map[string]any{
			"platform": "douyin", "shopId": shopID.String(), "redirectUri": row.RedirectURI, "developmentOnly": true,
		})
	}); err != nil {
		return "", err
	}
	u, _ := url.Parse("https://offline.invalid/oauth/authorize")
	query := u.Query()
	query.Set("state", state)
	query.Set("redirect_uri", row.RedirectURI)
	query.Set("mode", "fixture")
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func (s *Service) CompleteOfflineAuthorization(ctx context.Context, actor Actor, rawState string) (*CredentialMetadata, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if !s.OfflineEnabled {
		return nil, ErrCredentialUseDenied
	}
	hash := sha256.Sum256([]byte(strings.TrimSpace(rawState)))
	var state OAuthState
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("state_hash = ? AND tenant_id = ? AND user_id = ?", hex.EncodeToString(hash[:]), actor.TenantID, actor.UserID).First(&state).Error; err != nil {
			return ErrOAuthStateInvalid
		}
		if state.ConsumedAt != nil || !state.ExpiresAt.After(s.now()) {
			return ErrOAuthStateInvalid
		}
		now := s.now()
		res := tx.Model(&OAuthState{}).Where("id = ? AND consumed_at IS NULL", state.ID).Updates(map[string]any{"consumed_at": now, "updated_at": now})
		if res.Error != nil || res.RowsAffected != 1 {
			return ErrOAuthStateInvalid
		}
		return appendLifecycleEvent(tx, actor, uuid.Nil, 0, "oauth_callback_received", map[string]any{
			"platform": state.Platform, "shopId": state.ShopID.String(), "redirectUri": state.RedirectURI, "developmentOnly": true,
		})
	})
	if err != nil {
		s.Metrics.ObserveProductionCapabilities(s.Environment, "douyin", "oauth_callback", "failure", false, true, false, false, false, -1)
		return nil, err
	}
	return s.CreateOfflineCredential(ctx, actor, state.Platform, state.ShopID)
}

func redirectAllowed(candidate string, allowlist []string) bool {
	candidate = strings.TrimSpace(candidate)
	for _, allowed := range allowlist {
		if candidate != "" && candidate == strings.TrimSpace(allowed) {
			return true
		}
	}
	return false
}

func appendLifecycleEvent(tx *gorm.DB, actor Actor, credentialID uuid.UUID, version int, action string, metadata map[string]any) error {
	raw, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	return tx.Create(&CredentialLifecycleEvent{TenantID: actor.TenantID, CredentialID: credentialID, Version: version, Action: action, ActorID: actor.UserID, RequestID: strings.TrimSpace(actor.RequestID), Metadata: datatypes.JSON(raw)}).Error
}

func mapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrCredentialNotFound
	}
	return err
}
