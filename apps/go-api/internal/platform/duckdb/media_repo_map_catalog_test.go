//go:build integration

// media_repo_map_catalog_test.go — résolution map via maps_catalog
// (loadMapCatalogNames + enrichMediaMapTranslations) et garde anti-GUID.
//
// Couvre le bug "map = GUID" (Cliffhanger / 5324364b) : un match non enrichi
// stocke l'asset_id de map dans match_registry.map_name. Il doit être résolu via
// le catalogue (name_canonical + asset_translations FR) ou masqué — jamais
// affiché en UUID brut.
//
// CGO requis (driver DuckDB) → tag integration.
package duckdb

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"

	_ "github.com/duckdb/duckdb-go/v2"
)

func setupMetadataWithMapCatalog(t *testing.T) *DB {
	t.Helper()
	sqlDB, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	db := newTestDB(sqlDB, ":memory:")
	ctx := context.Background()
	for _, q := range []string{
		`CREATE TABLE maps_catalog (title_slug VARCHAR, map_asset_id VARCHAR, name_canonical VARCHAR, image_url VARCHAR)`,
		`CREATE TABLE asset_translations (asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR, PRIMARY KEY (asset_id, asset_type, lang))`,
		// Cliffhanger : présent au catalogue + traduction FR.
		`INSERT INTO maps_catalog VALUES ('halo_infinite','5324364b-cliff','Cliffhanger','/static/maps/halo_infinite/Cliffhanger.jpg')`,
		`INSERT INTO asset_translations VALUES ('5324364b-cliff','map','fr-FR','Dévissage')`,
		// Domicile : présent au catalogue, sans traduction FR.
		`INSERT INTO maps_catalog VALUES ('halo_infinite','921aebb1-dom','Domicile',NULL)`,
	} {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("setup %q: %v", q, err)
		}
	}
	return db
}

func TestEnrichMediaMapTranslations_ResolvesViaCatalog(t *testing.T) {
	meta := setupMetadataWithMapCatalog(t)
	repo := NewMediaRepo(&PlayerDB{Metadata: meta})

	rows := []domain.MediaFileRow{
		// Match non enrichi : map_id ET map_name = l'asset_id brut → FR via catalogue.
		{MapID: strPtr("5324364b-cliff"), MapName: strPtr("5324364b-cliff")},
		// Déjà résolu, pas de FR → name_canonical EN.
		{MapID: strPtr("921aebb1-dom"), MapName: strPtr("Domicile")},
		// GUID inconnu (absent du catalogue), map_id absent → masqué (jamais le GUID).
		{MapID: nil, MapName: strPtr("deadbeef-0000-0000-0000-000000000000")},
		// Nom propre hors catalogue → inchangé.
		{MapID: nil, MapName: strPtr("Aquarius")},
	}
	repo.enrichMediaMapTranslations(context.Background(), rows)

	assertMapName(t, "row0 (catalogue+FR)", rows[0].MapName, "Dévissage")
	assertMapName(t, "row1 (name_canonical)", rows[1].MapName, "Domicile")
	if rows[2].MapName != nil {
		t.Errorf("row2 : GUID inconnu doit être masqué, got %q", *rows[2].MapName)
	}
	assertMapName(t, "row3 (nom propre inchangé)", rows[3].MapName, "Aquarius")
}

// TestEnrichMediaMapTranslations_LocaleAware prouve GH2-B6 (« Recent media ») :
// le nom de carte suit la locale de requête — EN = name_canonical, FR =
// asset_translations (fallback croisé si l'une manque).
func TestEnrichMediaMapTranslations_LocaleAware(t *testing.T) {
	meta := setupMetadataWithMapCatalog(t)
	repo := NewMediaRepo(&PlayerDB{Metadata: meta})

	mkRows := func() []domain.MediaFileRow {
		return []domain.MediaFileRow{
			{MapID: strPtr("5324364b-cliff"), MapName: strPtr("5324364b-cliff")}, // FR dispo
			{MapID: strPtr("921aebb1-dom"), MapName: strPtr("Domicile")},         // EN seul
		}
	}

	frRows := mkRows()
	repo.enrichMediaMapTranslations(ctxkeys.WithLocale(context.Background(), "fr"), frRows)
	assertMapName(t, "FR row0", frRows[0].MapName, "Dévissage")
	assertMapName(t, "FR row1 (fallback EN)", frRows[1].MapName, "Domicile")

	enRows := mkRows()
	repo.enrichMediaMapTranslations(ctxkeys.WithLocale(context.Background(), "en"), enRows)
	assertMapName(t, "EN row0 (jamais de FR sous EN)", enRows[0].MapName, "Cliffhanger")
	assertMapName(t, "EN row1", enRows[1].MapName, "Domicile")
}

func TestLoadMapCatalogNames_PrefersFR(t *testing.T) {
	meta := setupMetadataWithMapCatalog(t)
	repo := NewMediaRepo(&PlayerDB{Metadata: meta})

	got := repo.loadMapCatalogNames(context.Background(), []string{"5324364b-cliff", "921aebb1-dom", "inconnu"})
	if got["5324364b-cliff"].fr != "Dévissage" || got["5324364b-cliff"].en != "Cliffhanger" {
		t.Errorf("Cliffhanger : got %+v, want en=Cliffhanger fr=Dévissage", got["5324364b-cliff"])
	}
	if got["921aebb1-dom"].en != "Domicile" || got["921aebb1-dom"].fr != "" {
		t.Errorf("Domicile : got %+v, want en=Domicile fr vide", got["921aebb1-dom"])
	}
	if _, ok := got["inconnu"]; ok {
		t.Errorf("id inconnu ne doit pas être présent dans le résultat")
	}
}

func TestLooksLikeAssetID(t *testing.T) {
	cases := map[string]bool{
		"5324364b-39a8-4f93-96a6-b80a1f18ce8a": true,
		"Domicile":                             false,
		"":                                     false,
		"343 Meowlnir":                         false,
		"Cliffhanger.jpg":                      false,
	}
	for in, want := range cases {
		if got := looksLikeAssetID(in); got != want {
			t.Errorf("looksLikeAssetID(%q) = %v, want %v", in, got, want)
		}
	}
}

func assertMapName(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Errorf("%s : got <nil>, want %q", label, want)
		return
	}
	if *got != want {
		t.Errorf("%s : got %q, want %q", label, *got, want)
	}
}
