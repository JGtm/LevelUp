package main

import (
	"database/sql"
	"testing"

	halo5 "levelup/go-api/internal/games/halo_5"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestChooseFR verrouille la précédence du nom FR : override TOML > nom FR de l'API
// (Accept-Language: fr-FR) > nom EN. C'est la logique qui débloque la traduction des
// titres de commendations Halo 5 (name_fr ne mirrore plus l'EN).
func TestChooseFR(t *testing.T) {
	override := map[string]string{
		"Spartan Slayer": "Tueur de Spartans",
		"Blank Override": "   ", // override en espaces → ignoré (traité comme absent)
	}
	cases := []struct {
		name  string
		en    string
		apiFR string
		want  string
	}{
		{"override prioritaire sur l'API", "Spartan Slayer", "Bourreau de Spartans", "Tueur de Spartans"},
		{"nom FR de l'API quand pas d'override", "Headshot Honcho", "Chef des tirs à la tête", "Chef des tirs à la tête"},
		{"override en espaces → nom FR de l'API", "Blank Override", "Depuis l'API", "Depuis l'API"},
		{"ni override ni API → EN", "Lone Wolf", "", "Lone Wolf"},
		{"API FR en espaces → EN", "Lone Wolf", "   ", "Lone Wolf"},
		{"nom FR de l'API trimé", "Sharp Shooter", "  Tireur d'élite  ", "Tireur d'élite"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := chooseFR(override, c.en, c.apiFR); got != c.want {
				t.Errorf("chooseFR(en=%q, apiFR=%q) = %q, want %q", c.en, c.apiFR, got, c.want)
			}
		})
	}
}

// TestFrOrPreservesLegacyBehaviour garantit que medals/weapons (qui passent par frOr,
// sans localisation API) conservent leur comportement : override sinon EN.
func TestFrOrPreservesLegacyBehaviour(t *testing.T) {
	m := map[string]string{"Sniper": "Tireur d'élite"}
	if got := frOr(m, "Sniper"); got != "Tireur d'élite" {
		t.Errorf("frOr override = %q, want 'Tireur d'élite'", got)
	}
	if got := frOr(m, "Unknown"); got != "Unknown" {
		t.Errorf("frOr fallback = %q, want clé EN", got)
	}
}

// TestApplyMapIDOverrides couvre B1/B2 (résidus H5) : un canvas Forge présent dans
// maps_catalog avec name_canonical vide (l'API officielle ne le nomme pas) reçoit son
// nom canonique EN + les traductions asset_translations en-US/fr-FR depuis l'override
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

	ov := []mapIDOverride{{ID: "d67fdcb9-6d9c-403e-960d-04202e19b244", EN: "Tidal", FR: "Tidal"}}
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

	for _, lang := range []string{"en-US", "fr-FR"} {
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
