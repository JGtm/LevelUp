package main

import (
	"database/sql"
	"encoding/json"
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

// TestPersistTeamColors couvre l'ingestion team-colors (E1) : parsing de la réponse
// API officielle /team-colors (fixture EN) + noms FR localisés (Accept-Language),
// persistance idempotente dans metadata.duckdb, iconUrl null toléré. Ces libellés
// alimentent l'affichage « Rouge/Bleu » de la Match View H5.
func TestPersistTeamColors(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE team_colors (
		team_id INTEGER PRIMARY KEY, name_en VARCHAR NOT NULL DEFAULT '',
		name_fr VARCHAR NOT NULL DEFAULT '', color VARCHAR, icon_url VARCHAR)`); err != nil {
		t.Fatalf("create team_colors: %v", err)
	}

	// Fixture : forme RÉELLE de la réponse EN de l'API /team-colors — `id` est sérialisé
	// en STRING ("0","1"), iconUrl nullable. Verrouille le parsing string→int (régression
	// : un retour de apiTeamColor.ID à int ferait échouer cet unmarshal).
	payload := `[
		{"id":"0","name":"Red","description":"Red team","color":"#E64C4C","iconUrl":"https://cdn/red.png","contentId":"a"},
		{"id":"1","name":"Blue","description":"Blue team","color":"#4C7FE6","iconUrl":null,"contentId":"b"}
	]`
	var colors []apiTeamColor
	if err := json.Unmarshal([]byte(payload), &colors); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	// Noms FR issus du pass Accept-Language: fr-FR (indexés par l'id string de l'API).
	frByID := map[string]string{"0": "Rouge", "1": "Bleu"}

	if got := persistTeamColors(db, colors, frByID); got != 2 {
		t.Fatalf("persistTeamColors a écrit %d lignes, want 2", got)
	}
	// Idempotence : rejouer ne diverge pas (INSERT OR REPLACE).
	if got := persistTeamColors(db, colors, frByID); got != 2 {
		t.Fatalf("persistTeamColors (rejeu) a écrit %d lignes, want 2", got)
	}

	var nameEN, nameFR, color string
	if err := db.QueryRow(
		`SELECT name_en, name_fr, color FROM team_colors WHERE team_id=0`).Scan(&nameEN, &nameFR, &color); err != nil {
		t.Fatalf("read team 0: %v", err)
	}
	if nameEN != "Red" || nameFR != "Rouge" || color != "#E64C4C" {
		t.Errorf("team 0 = {en=%q fr=%q color=%q}, want {Red Rouge #E64C4C}", nameEN, nameFR, color)
	}

	// team 1 : iconUrl null → chaîne vide persistée, pas d'erreur ; name_fr localisé.
	var nameFR1, iconURL string
	if err := db.QueryRow(
		`SELECT name_fr, COALESCE(icon_url,'') FROM team_colors WHERE team_id=1`).Scan(&nameFR1, &iconURL); err != nil {
		t.Fatalf("read team 1: %v", err)
	}
	if nameFR1 != "Bleu" {
		t.Errorf("team 1 name_fr = %q, want 'Bleu'", nameFR1)
	}
	if iconURL != "" {
		t.Errorf("team 1 icon_url = %q, want '' (iconUrl null)", iconURL)
	}
}

// TestPersistTeamColors_FrenchFallsBackToEN : sans nom FR de l'API, name_fr = name_en
// (jamais vide — même convention que chooseFR).
func TestPersistTeamColors_FrenchFallsBackToEN(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE team_colors (
		team_id INTEGER PRIMARY KEY, name_en VARCHAR NOT NULL DEFAULT '',
		name_fr VARCHAR NOT NULL DEFAULT '', color VARCHAR, icon_url VARCHAR)`); err != nil {
		t.Fatalf("create team_colors: %v", err)
	}
	colors := []apiTeamColor{{ID: "2", Name: "Green", Color: "#4CE67F"}}
	persistTeamColors(db, colors, map[string]string{}) // aucun nom FR
	var nameFR string
	if err := db.QueryRow(`SELECT name_fr FROM team_colors WHERE team_id=2`).Scan(&nameFR); err != nil {
		t.Fatalf("read team 2: %v", err)
	}
	if nameFR != "Green" {
		t.Errorf("name_fr = %q, want fallback 'Green'", nameFR)
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
