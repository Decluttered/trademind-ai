package catalog

import (
	"context"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
	"time"
)

func catalogTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&SourceProduct{}, &ProductSnapshot{}, &OpportunityAssessment{}, &Collection{}, &CollectionProduct{}))
	return db
}
func TestCaptureDeduplicatesASINAndAppendsImmutableSnapshots(t *testing.T) {
	db := catalogTestDB(t)
	svc := &Service{DB: db}
	price := int64(1999)
	input := NormalizedProduct{Source: "amazon.de", SourceURL: "https://www.amazon.de/dp/B012345678", SourceProductID: "B012345678", Title: "Fixture", Currency: "EUR", PriceCents: &price, Availability: "Auf Lager", Raw: map[string]any{"revision": 1}}
	first, err := svc.Capture(context.Background(), 7, input)
	require.NoError(t, err)
	input.Raw = map[string]any{"revision": 2}
	second, err := svc.Capture(context.Background(), 7, input)
	require.NoError(t, err)
	require.Equal(t, first.Product.ID, second.Product.ID)
	require.NotEqual(t, first.Snapshot.ID, second.Snapshot.ID)
	var products, snapshots int64
	require.NoError(t, db.Model(&SourceProduct{}).Count(&products).Error)
	require.NoError(t, db.Model(&ProductSnapshot{}).Count(&snapshots).Error)
	require.Equal(t, int64(1), products)
	require.Equal(t, int64(2), snapshots)
	var current SourceProduct
	require.NoError(t, db.First(&current, first.Product.ID).Error)
	require.Equal(t, second.Snapshot.ID, *current.CurrentSnapshotID)
	require.Error(t, db.Model(&first.Snapshot).Update("title", "mutated").Error)
}

func TestCollectionAssignmentIsWorkspaceScopedAndIdempotent(t *testing.T) {
	db := catalogTestDB(t)
	svc := &Service{DB: db}
	collection, err := svc.CreateCollection(context.Background(), 7, CreateCollectionInput{Name: "Garten", Kind: "manual"})
	require.NoError(t, err)
	product := SourceProduct{WorkspaceID: 7, Source: "amazon.de", ExternalID: "B012345678", CanonicalURL: "https://www.amazon.de/dp/B012345678"}
	require.NoError(t, db.Create(&product).Error)
	snapshot := ProductSnapshot{WorkspaceID: 7, SourceProductID: product.ID, CapturedAt: time.Now().UTC(), Title: "Fixture", Currency: "EUR", Images: jsonData([]string{}), Attributes: jsonData(map[string]any{}), Variants: jsonData([]string{}), Raw: jsonData(map[string]any{})}
	require.NoError(t, db.Create(&snapshot).Error)
	require.NoError(t, db.Model(&product).Update("current_snapshot_id", snapshot.ID).Error)
	require.NoError(t, db.Create(&OpportunityAssessment{WorkspaceID: 7, SourceProductID: product.ID, SnapshotID: snapshot.ID, Score: 72, Confidence: 50, RuleVersion: "test", Factors: jsonData(map[string]any{})}).Error)
	first, err := svc.AddProductToCollection(context.Background(), 7, collection.ID, product.ID, "operator selection")
	require.NoError(t, err)
	second, err := svc.AddProductToCollection(context.Background(), 7, collection.ID, product.ID, "duplicate")
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	var count int64
	require.NoError(t, db.Model(&CollectionProduct{}).Count(&count).Error)
	require.Equal(t, int64(1), count)
	minScore := 70
	page, err := svc.ListProducts(context.Background(), 7, 10, "", ProductFilter{CollectionID: &collection.ID, MinScore: &minScore})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	_, err = svc.AddProductToCollection(context.Background(), 8, collection.ID, product.ID, "cross workspace")
	require.Error(t, err)
}
