//go:build integration

package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/migration"
)

// insertCitationTexts écrit le nom et la description d'une citation par langue dans
// asset_translations (type citation), là où les lecteurs les lisent.
func insertCitationTexts(t *testing.T, db *DB, norm string, names, descriptions i18n.Text) {
	t.Helper()
	if err := migration.UpsertAssetTexts(context.Background(), db.SQLDb(), AssetTypeCitation, norm, names, descriptions); err != nil {
		t.Fatalf("insertCitationTexts(%s): %v", norm, err)
	}
}
