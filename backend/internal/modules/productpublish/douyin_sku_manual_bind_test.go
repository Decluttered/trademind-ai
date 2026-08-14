package productpublish

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	platformdouyin "github.com/trademind-ai/trademind/backend/internal/providers/platform/douyinshop"
	"gorm.io/gorm"
)

func TestDouyinInventorySyncReadyAllBound(t *testing.T) {
	rows := []DouyinSKUBindingRow{
		{PublicationSKUID: uuid.New(), ExternalSKUID: "a", BindStatus: BindStatusBound},
		{PublicationSKUID: uuid.New(), ExternalSKUID: "b", BindStatus: BindStatusSkipped},
	}
	ready, reason := DouyinInventorySyncReady(rows)
	if !ready || reason != "" {
		t.Fatalf("expected ready, got ready=%v reason=%q", ready, reason)
	}
}

func TestDouyinInventorySyncReadyUnmatched(t *testing.T) {
	rows := []DouyinSKUBindingRow{
		{PublicationSKUID: uuid.New(), ExternalSKUID: "a", BindStatus: BindStatusBound},
		{PublicationSKUID: uuid.New(), BindStatus: BindStatusUnmatched},
	}
	ready, _ := DouyinInventorySyncReady(rows)
	if ready {
		t.Fatal("expected not ready for unmatched")
	}
}

func TestDouyinInventorySyncReadyAmbiguous(t *testing.T) {
	rows := []DouyinSKUBindingRow{
		{PublicationSKUID: uuid.New(), BindStatus: BindStatusAmbiguous},
	}
	ready, reason := DouyinInventorySyncReady(rows)
	if ready {
		t.Fatal("expected not ready for ambiguous")
	}
	if reason == "" {
		t.Fatal("expected block reason")
	}
}

func TestDouyinInventorySyncReadyFailed(t *testing.T) {
	rows := []DouyinSKUBindingRow{
		{PublicationSKUID: uuid.New(), BindStatus: BindStatusFailed},
	}
	ready, _ := DouyinInventorySyncReady(rows)
	if ready {
		t.Fatal("expected not ready for failed")
	}
}

func TestValidateDouyinSKUBindingForInventorySync(t *testing.T) {
	if err := ValidateDouyinSKUBindingForInventorySync("douyin_shop", "123", BindStatusBound); err != nil {
		t.Fatalf("bound should pass: %v", err)
	}
	if err := ValidateDouyinSKUBindingForInventorySync("douyin_shop", "", BindStatusUnmatched); err == nil {
		t.Fatal("expected binding required")
	}
	if err := ValidateDouyinSKUBindingForInventorySync("douyin_shop", "123", BindStatusAmbiguous); err == nil {
		t.Fatal("expected ambiguous error")
	}
	if err := ValidateDouyinSKUBindingForInventorySync("tiktok", "", ""); err != nil {
		t.Fatalf("non-douyin should skip: %v", err)
	}
}

func TestAnnotatePlatformSkuCandidates(t *testing.T) {
	pubSku := uuid.New()
	cands := []DouyinPlatformSKUCandidate{{PlatformSKUID: "p1", SpecName: "红色 / L"}}
	rows := []DouyinSKUBindingRow{{PublicationSKUID: pubSku, ExternalSKUID: "p1", BindStatus: BindStatusBound}}
	out := annotatePlatformSkuCandidates(cands, rows)
	if out[0].BoundToPublicationSkuID == nil || *out[0].BoundToPublicationSkuID != pubSku {
		t.Fatalf("expected bound annotation, got %+v", out[0].BoundToPublicationSkuID)
	}
}

func TestPlatformSkusFromPublicationRaw(t *testing.T) {
	raw := mergePublicationRawPlatformSkus(nil, []platformdouyin.PlatformProductSKU{
		{PlatformSKUID: "111", SpecName: "红色 / L", PriceYuan: 99, Stock: 10},
	})
	cands := platformSkusFromPublicationRaw(raw)
	if len(cands) != 1 || cands[0].PlatformSKUID != "111" {
		t.Fatalf("unexpected candidates: %+v", cands)
	}
}

func TestCheckDouyinSKUBindingConflictPropagatesDatabaseErrors(t *testing.T) {
	db := newDouyinPublishTestDB(t)
	require.NoError(t, db.Migrator().DropTable(&ProductPublicationSKU{}))
	svc := &Service{DB: db}

	err := svc.checkDouyinSKUBindingConflict(context.Background(), uuid.New(), uuid.New(), "platform-sku-1")
	require.Error(t, err)
	require.NotErrorIs(t, err, gorm.ErrRecordNotFound)
	require.Contains(t, err.Error(), "check douyin sku binding conflict")
}
