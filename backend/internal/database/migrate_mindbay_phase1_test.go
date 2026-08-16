package database

import (
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/modules/catalog"
	"github.com/trademind-ai/trademind/backend/internal/modules/listingstudio"
	"gorm.io/gorm"
	"testing"
)

func TestMindBayImmutableTriggersRejectRawUpdates(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&catalog.ProductSnapshot{}, &listingstudio.ListingContentVersion{}))
	require.NoError(t, migrateMindBayPhase1Immutability(db))
	snap := catalog.ProductSnapshot{WorkspaceID: 1, SourceProductID: [16]byte{}, Title: "before", Currency: "EUR", Raw: []byte(`{}`)}
	require.NoError(t, db.Create(&snap).Error)
	require.Error(t, db.Exec("UPDATE product_snapshot SET title=? WHERE id=?", "after", snap.ID).Error)
}
