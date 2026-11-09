package main

import (
	"database/sql"
	"testing"

	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/i18n"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestApplyMapIDOverrides couvre B1/B2 (résidus H5) : un canvas Forge présent dans
// maps_catalog avec name_canonical vide (l'API officielle ne le nomme pas) reçoit son
// nom canonique EN + les traductions asset_translations par langue depuis l'override
// keyé par asset_id. Idempotent (rejouable sans divergence).
func TestApplyMapIDOverrides(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	for _, q := range []string{
		`CREATE TABLE maps_catalog (title_slug VARCHAR, map_asset_id VARCHAR, name_canonical VARCHAR,
			image_url VARCHAR, last_fetched_at TIMESTAMP)`,
		`CREATE TABLE asset_translations (asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR,
			PRIMARY KEY (asset_id, asset_type, lang))`,
		`INSERT INTO maps_catalog VALUES ('` + halo5.TitleSlug + `','D67FDCB9-6D9C-403E-960D-04202E19B244','',NULL,NULL)`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("seed: %v\nSQL: %s", err, q)
		}
	}

	ov := []mapIDEntry{{ID: "d67fdcb9-6d9c-403e-960d-04202e19b244", Name: i18n.Text{i18n.Fallback: "Tidal", i18n.LocaleFR: "Tidal"}}}
	// Appliqué deux fois : idempotence.
	applyMapIDOverrides(db, ov)
	applyMapIDOverrides(db, ov)

	var nameCanonical string
	if err := db.QueryRow(
		`SELECT name_canonical FROM maps_catalog WHERE lower(map_asset_id)=lower(?)`,
		ov[0].ID).Scan(&nameCanonical); err != nil {
		t.Fatalf("read name_canonical: %v", err)
	}
	if nameCanonical != "Tidal" {
		t.Errorf("name_canonical = %q, want 'Tidal' (override par asset_id, casse tolérée)", nameCanonical)
	}

	for _, l := range i18n.Locales() {
		lang := haloTag(l)
		var name string
		if err := db.QueryRow(
			`SELECT name FROM asset_translations WHERE asset_id=? AND asset_type='map' AND lang=?`,
			ov[0].ID, lang).Scan(&name); err != nil {
			t.Fatalf("read asset_translations[%s]: %v", lang, err)
		}
		if name != "Tidal" {
			t.Errorf("asset_translations[%s] = %q, want 'Tidal'", lang, name)
		}
	}
}

// Une langue sans texte dans l'override n'écrit rien (pas de nom inventé) : la lecture se replie
// sur la langue de l'API.
func TestApplyMapIDOverrides_LangueSansTexteNEcritRien(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	for _, q := range []string{
		`CREATE TABLE maps_catalog (title_slug VARCHAR, map_asset_id VARCHAR, name_canonical VARCHAR,
			image_url VARCHAR, last_fetched_at TIMESTAMP)`,
		`CREATE TABLE asset_translations (asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR,
			PRIMARY KEY (asset_id, asset_type, lang))`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	applyMapIDOverrides(db, []mapIDEntry{{ID: "m1", Name: i18n.Text{i18n.Fallback: "Tidal"}}})
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM asset_translations WHERE asset_id = 'm1'`).Scan(&n); err != nil || n != 1 {
		t.Errorf("lignes = %d (err %v), want 1 (langue de l'API seule)", n, err)
	}
}
