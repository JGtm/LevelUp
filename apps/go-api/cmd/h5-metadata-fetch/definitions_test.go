package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/i18n"
)

// newMetadataDB crée les tables de définitions telles qu'elles sont APRÈS la migration
// metadata_definition_texts_to_translations_v1 : aucune colonne de texte, tables par langue.
func newMetadataDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	db.SetMaxOpenConns(1)
	for _, q := range []string{
		`CREATE TABLE team_colors (team_id INTEGER PRIMARY KEY, color VARCHAR, icon_url VARCHAR)`,
		`CREATE TABLE commendation_definitions (commendation_id VARCHAR PRIMARY KEY, commendation_type VARCHAR,
			category VARCHAR, icon_url VARCHAR, tier_targets VARCHAR)`,
		`CREATE TABLE medal_definitions (medal_name_id BIGINT PRIMARY KEY, difficulty_index INTEGER, difficulty VARCHAR,
			medal_type VARCHAR, sprite_sheet_url VARCHAR, sprite_left INTEGER, sprite_top INTEGER,
			sprite_width INTEGER, sprite_height INTEGER)`,
		`CREATE TABLE asset_translations (asset_id VARCHAR NOT NULL, asset_type VARCHAR NOT NULL, lang VARCHAR NOT NULL,
			name VARCHAR NOT NULL, description VARCHAR, PRIMARY KEY (asset_id, asset_type, lang))`,
		`CREATE TABLE medal_translations (medal_name_id BIGINT NOT NULL, lang VARCHAR NOT NULL, name VARCHAR NOT NULL,
			description VARCHAR, PRIMARY KEY (medal_name_id, lang))`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("schéma: %v\nSQL: %s", err, q)
		}
	}
	return db
}

// textRow relit la ligne (nom, description) d'un actif dans une langue ; ok=false si absente.
func textRow(t *testing.T, db *sql.DB, assetType, assetID string, l i18n.Locale) (name, desc string, ok bool) {
	t.Helper()
	var d sql.NullString
	err := db.QueryRow(`SELECT name, description FROM asset_translations WHERE asset_type = ? AND asset_id = ? AND lang = ?`,
		assetType, assetID, haloTag(l)).Scan(&name, &d)
	if err == sql.ErrNoRows {
		return "", "", false
	}
	if err != nil {
		t.Fatalf("lecture %s/%s/%s: %v", assetType, assetID, l, err)
	}
	return name, d.String, true
}

// Les couleurs d'équipe écrivent une ligne par langue qui a un nom : la langue de l'API depuis la
// réponse sans en-tête, les autres depuis leur propre passe. Idempotent, iconUrl null toléré.
func TestPersistTeamColors(t *testing.T) {
	db := newMetadataDB(t)
	ctx := context.Background()

	// Forme RÉELLE de la réponse de l'API /team-colors : `id` est sérialisé en STRING ("0","1"),
	// iconUrl nullable. Verrouille le parsing string→int.
	payload := `[
		{"id":"0","name":"Red","description":"Red team","color":"#E64C4C","iconUrl":"https://cdn/red.png","contentId":"a"},
		{"id":"1","name":"Blue","description":"Blue team","color":"#4C7FE6","iconUrl":null,"contentId":"b"}
	]`
	var colors []apiTeamColor
	if err := json.Unmarshal([]byte(payload), &colors); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	localized := map[i18n.Locale]map[string]string{i18n.LocaleFR: {"0": "Rouge", "1": "Bleu"}}

	for i := 0; i < 2; i++ { // 2e passe : aucune ligne en double
		if got := persistTeamColors(ctx, db, colors, localized); got != 2 {
			t.Fatalf("passe %d : %d couleurs écrites, want 2", i, got)
		}
	}
	var color, icon string
	if err := db.QueryRow(`SELECT color, COALESCE(icon_url,'') FROM team_colors WHERE team_id = 1`).Scan(&color, &icon); err != nil {
		t.Fatal(err)
	}
	if color != "#4C7FE6" || icon != "" {
		t.Errorf("team 1 = {%q %q}, want {#4C7FE6 ''}", color, icon)
	}
	for id, want := range map[string]map[i18n.Locale]string{
		"0": {i18n.LocaleFR: "Rouge", i18n.LocaleEN: "Red"},
		"1": {i18n.LocaleFR: "Bleu", i18n.LocaleEN: "Blue"},
	} {
		for l, name := range want {
			if got, _, ok := textRow(t, db, "team_color", id, l); !ok || got != name {
				t.Errorf("team %s [%s] = %q (ok=%v), want %q", id, l, got, ok, name)
			}
		}
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM asset_translations WHERE asset_type = 'team_color'`).Scan(&n); err != nil || n != 4 {
		t.Errorf("lignes team_color = %d (err %v), want 4", n, err)
	}
}

// Sans nom localisé, la langue n'écrit RIEN (pas de copie de l'anglais dans la langue) : la
// lecture se replie sur la langue de l'API.
func TestPersistTeamColors_LangueSansNomNEcritRien(t *testing.T) {
	db := newMetadataDB(t)
	colors := []apiTeamColor{{ID: "2", Name: "Green", Color: "#4CE67F"}}
	persistTeamColors(context.Background(), db, colors, map[i18n.Locale]map[string]string{})
	if name, _, ok := textRow(t, db, "team_color", "2", i18n.Fallback); !ok || name != "Green" {
		t.Errorf("langue de l'API = %q (ok=%v), want Green", name, ok)
	}
	if _, _, ok := textRow(t, db, "team_color", "2", i18n.LocaleFR); ok {
		t.Error("une ligne `fr` a été écrite sans nom localisé")
	}
}

func TestPersistTeamColors_IDNonNumeriqueIgnore(t *testing.T) {
	db := newMetadataDB(t)
	got := persistTeamColors(context.Background(), db, []apiTeamColor{{ID: "abc", Name: "X"}}, nil)
	if got != 0 {
		t.Errorf("écrites = %d, want 0", got)
	}
}

// Commendations : l'override TOML l'emporte sur le nom localisé de l'API, qui l'emporte sur rien ;
// la description d'une langue est celle de l'API dans cette langue, jamais une copie d'une autre.
func TestPersistCommendation(t *testing.T) {
	db := newMetadataDB(t)
	ctx := context.Background()
	labels := assetLabels{Commendations: map[string]i18n.Text{
		"Spartan Slayer": {i18n.LocaleFR: "Tueur de Spartans"},
	}}
	localized := map[i18n.Locale]map[string]commText{i18n.LocaleFR: {
		"c1": {name: "Bourreau de Spartans", desc: "Tuez des Spartans."},
		"c2": {name: "Chef des tirs à la tête"},
	}}
	comms := []apiCommendation{
		{ID: "c1", Name: "Spartan Slayer  ", Description: "Kill Spartans.", Type: "Progressive"},
		{ID: "c2", Name: "Headshot Honcho", Description: "Get headshots.", Type: "Meta"},
		{ID: "c3", Name: "Lone Wolf", Type: "Daily"}, // ni override ni API localisée, ni description
	}
	for i := 0; i < 2; i++ { // idempotent
		for _, c := range comms {
			if err := persistCommendation(ctx, db, c, labels, localized); err != nil {
				t.Fatalf("passe %d, %s : %v", i, c.ID, err)
			}
		}
	}
	type want struct {
		l          i18n.Locale
		name, desc string
		present    bool
	}
	cases := map[string][]want{
		"c1": {{i18n.LocaleFR, "Tueur de Spartans", "Tuez des Spartans.", true}, {i18n.LocaleEN, "Spartan Slayer", "Kill Spartans.", true}},
		"c2": {{i18n.LocaleFR, "Chef des tirs à la tête", "", true}, {i18n.LocaleEN, "Headshot Honcho", "Get headshots.", true}},
		"c3": {{i18n.LocaleFR, "", "", false}, {i18n.LocaleEN, "Lone Wolf", "", true}},
	}
	for id, ws := range cases {
		for _, w := range ws {
			name, desc, ok := textRow(t, db, "commendation", id, w.l)
			if ok != w.present || name != w.name || desc != w.desc {
				t.Errorf("%s [%s] = {%q %q ok=%v}, want {%q %q ok=%v}", id, w.l, name, desc, ok, w.name, w.desc, w.present)
			}
		}
	}
	var typ string
	if err := db.QueryRow(`SELECT commendation_type FROM commendation_definitions WHERE commendation_id = 'c2'`).Scan(&typ); err != nil || typ != "Meta" {
		t.Errorf("type c2 = %q (err %v), want Meta", typ, err)
	}
}

// Médailles : la langue de l'API vient de l'API, les autres du TOML seul ; une médaille sans entrée
// TOML n'a que sa ligne de la langue de l'API.
func TestPersistMedal(t *testing.T) {
	db := newMetadataDB(t)
	ctx := context.Background()
	labels := assetLabels{
		Medals:            map[string]i18n.Text{"Splatter": {i18n.LocaleFR: "Écrasement"}},
		MedalDescriptions: map[string]i18n.Text{"Splatter": {i18n.LocaleFR: "Écrasez un ennemi."}},
	}
	for i := 0; i < 2; i++ {
		if err := persistMedal(ctx, db, 7, apiMedal{ID: "7", Name: "Splatter", Description: "Run over an enemy.", Classification: "Style"}, labels); err != nil {
			t.Fatal(err)
		}
		if err := persistMedal(ctx, db, 8, apiMedal{ID: "8", Name: "Unlisted", Description: "Something."}, labels); err != nil {
			t.Fatal(err)
		}
	}
	read := func(id int64, l i18n.Locale) (string, string, bool) {
		var n string
		var d sql.NullString
		err := db.QueryRow(`SELECT name, description FROM medal_translations WHERE medal_name_id = ? AND lang = ?`, id, haloTag(l)).Scan(&n, &d)
		return n, d.String, err == nil
	}
	if n, d, ok := read(7, i18n.LocaleFR); !ok || n != "Écrasement" || d != "Écrasez un ennemi." {
		t.Errorf("7 fr = {%q %q ok=%v}", n, d, ok)
	}
	if n, d, ok := read(7, i18n.LocaleEN); !ok || n != "Splatter" || d != "Run over an enemy." {
		t.Errorf("7 en = {%q %q ok=%v}", n, d, ok)
	}
	if _, _, ok := read(8, i18n.LocaleFR); ok {
		t.Error("médaille sans entrée TOML : aucune ligne `fr` attendue")
	}
	if n, _, ok := read(8, i18n.LocaleEN); !ok || n != "Unlisted" {
		t.Errorf("8 en = %q (ok=%v)", n, ok)
	}
	var cnt int
	if err := db.QueryRow(`SELECT COUNT(*) FROM medal_definitions`).Scan(&cnt); err != nil || cnt != 2 {
		t.Errorf("medal_definitions = %d (err %v), want 2", cnt, err)
	}
}

// L'en-tête de langue vient de config/locales.toml : aucune pour la langue de l'API, la première
// étiquette Halo pour les autres.
func TestAcceptLanguage(t *testing.T) {
	if got := acceptLanguage(i18n.Fallback); got != "" {
		t.Errorf("langue de l'API : %q, want vide", got)
	}
	for _, l := range i18n.Locales() {
		if l == i18n.Fallback {
			continue
		}
		if got, want := acceptLanguage(l), i18n.HaloTags(l)[0]; got != want {
			t.Errorf("acceptLanguage(%s) = %q, want %q", l, got, want)
		}
	}
}
