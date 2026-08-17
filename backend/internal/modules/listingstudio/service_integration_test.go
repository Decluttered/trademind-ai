package listingstudio

import (
	"context"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
)

type sourceStub struct{ facts SourceFacts }

func (s sourceStub) ReadSource(context.Context, int64, uuid.UUID) (SourceFacts, error) {
	return s.facts, nil
}

type generatorStub struct{ n int }

func (g *generatorStub) Generate(context.Context, int64, SourceFacts) (GeneratedContent, string, string, error) {
	g.n++
	return GeneratedContent{Title: "Sachlicher Titel", Description: "Beschreibung aus Fakten", Specifics: map[string]string{"Marke": "Acme"}}, "fixture", "mindbay_listing_studio_v1", nil
}

type settingsStub struct{ enabled bool }

func (s settingsStub) PlainByGroup(context.Context, int64, string) (map[string]string, error) {
	if s.enabled {
		return map[string]string{"gpsr_override_enabled": "true"}, nil
	}
	return map[string]string{}, nil
}
func studioDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&ListingDraft{}, &ListingContentVersion{}, &GPSRProfile{}, &ListingGPSR{}, &ImageAsset{}, &ImageDerivative{}, &ImageSet{}))
	return db
}
func TestGenerateAppendsVersionsAndKeepsPriorContent(t *testing.T) {
	db := studioDB(t)
	gen := &generatorStub{}
	svc := &Service{DB: db, Sources: sourceStub{SourceFacts{Title: "Source"}}, Generator: gen}
	asset := ImageAsset{WorkspaceID: 3, OriginURL: "https://example.com/x.jpg", SHA256: "abc", StorageKey: "x", ContentType: "image/jpeg", SizeBytes: 1}
	require.NoError(t, db.Create(&asset).Error)
	price := int64(2999)
	draft, err := svc.Create(context.Background(), 3, CreateDraftInput{SourceProductID: uuid.New(), Category: "Home", PriceCents: &price, RequiredSpecifics: []string{"Marke"}, ImageAssetIDs: []uuid.UUID{asset.ID}})
	require.NoError(t, err)
	v1, err := svc.Generate(context.Background(), 3, draft.ID)
	require.NoError(t, err)
	v2, err := svc.Generate(context.Background(), 3, draft.ID)
	require.NoError(t, err)
	require.Equal(t, 1, v1.Version)
	require.Equal(t, 2, v2.Version)
	var count int64
	require.NoError(t, db.Model(&ListingContentVersion{}).Where("listing_draft_id=?", draft.ID).Count(&count).Error)
	require.Equal(t, int64(2), count)
	require.Error(t, db.Model(v1).Update("title", "changed").Error)
}
func TestGPSRBlocksReadyUnlessValidOrAuditedOverride(t *testing.T) {
	db := studioDB(t)
	svc := &Service{DB: db, Sources: sourceStub{}, Settings: settingsStub{enabled: true}}
	asset := ImageAsset{WorkspaceID: 4, OriginURL: "https://example.com/x.jpg", SHA256: "def", StorageKey: "x", ContentType: "image/jpeg", SizeBytes: 1}
	require.NoError(t, db.Create(&asset).Error)
	price := int64(1000)
	draft, err := svc.Create(context.Background(), 4, CreateDraftInput{SourceProductID: uuid.New(), Category: "Test", PriceCents: &price, ImageAssetIDs: []uuid.UUID{asset.ID}})
	require.NoError(t, err)
	version := ListingContentVersion{WorkspaceID: 4, ListingDraftID: draft.ID, Version: 1, Title: "Valid title", Description: "Factual", Specifics: j(map[string]string{})}
	require.NoError(t, db.Create(&version).Error)
	require.NoError(t, db.Model(&ListingDraft{}).Where("id=?", draft.ID).Updates(map[string]any{"current_content_version_id": version.ID, "state": StateNeedsReview}).Error)
	blocked, err := svc.Validate(context.Background(), 4, draft.ID, ValidationInput{})
	require.NoError(t, err)
	require.Equal(t, StateBlocked, blocked.State)
	drafting, err := svc.Validate(context.Background(), 4, draft.ID, ValidationInput{GPSROverride: true, OverrideReason: "Reviewed low-risk legacy item", AdminID: ptrUUID(uuid.New())})
	require.NoError(t, err)
	require.Equal(t, StateDrafting, drafting.State)
	require.True(t, drafting.GPSROverridden)
	review, err := svc.Validate(context.Background(), 4, draft.ID, ValidationInput{})
	require.NoError(t, err)
	require.Equal(t, StateNeedsReview, review.State)
	ready, err := svc.Validate(context.Background(), 4, draft.ID, ValidationInput{})
	require.NoError(t, err)
	require.Equal(t, StateReady, ready.State)
}
func TestGPSRProfileRequiresCompleteTraceableData(t *testing.T) {
	db := studioDB(t)
	svc := &Service{DB: db}
	_, err := svc.CreateGPSRProfile(context.Background(), 9, CreateGPSRProfileInput{Name: "Incomplete"})
	require.Error(t, err)
	profile, err := svc.CreateGPSRProfile(context.Background(), 9, CreateGPSRProfileInput{Name: "Garden supplier", ManufacturerName: "Acme GmbH", ManufacturerAddress: "Musterweg 1, Berlin", ResponsiblePersonName: "Eva Beispiel", ResponsiblePersonAddress: "Musterweg 2, Berlin", SafetyInformation: "Von Kindern fernhalten", DocumentReferences: []string{"doc://safety-1"}})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, profile.ID)
	profiles, err := svc.ListGPSRProfiles(context.Background(), 9)
	require.NoError(t, err)
	require.Len(t, profiles, 1)
}
func ptrUUID(v uuid.UUID) *uuid.UUID { return &v }
