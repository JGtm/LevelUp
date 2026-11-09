//go:build integration

// halo5_commendation_defs_test.go — test in-memory (DuckDB :memory:) de la lecture
// du référentiel commendation_definitions (metadata h5) et de ses textes par langue
// (asset_translations, type « commendation »). NE TOUCHE JAMAIS la vraie metadata h5.
//
// Lancer : go test -tags=integration ./internal/platform/duckdb/halo5/ -run Halo5Commendation

package halo5

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/ctxkeys"

	_ "github.com/duckdb/duckdb-go/v2"
)

// seedCommendationDefs crée les tables telles qu'elles sont après la migration des textes de
// définitions : commendation_definitions sans colonne de texte, noms dans asset_translations.
func seedCommendationDefs(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	for _, q := range []string{
		`CREATE TABLE commendation_definitions (
			commendation_id   VARCHAR PRIMARY KEY,
			commendation_type VARCHAR,
			category          VARCHAR,
			icon_url          VARCHAR,
			tier_targets      VARCHAR
		)`,
		`CREATE TABLE asset_translations (
			asset_id VARCHAR NOT NULL, asset_type VARCHAR NOT NULL, lang VARCHAR NOT NULL,
			name VARCHAR NOT NULL, description VARCHAR,
			PRIMARY KEY (asset_id, asset_type, lang)
		)`,
	} {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatalf("ddl: %v", err)
		}
	}
	rows := []struct{ id, icon string }{
		{"uuid-1", "https://cdn/1.png"},
		{"uuid-2", "https://cdn/2.png"},
		{"uuid-3", ""}, // icon vide
	}
	for _, r := range rows {
		if _, err := db.ExecContext(ctx, `INSERT INTO commendation_definitions
			(commendation_id, commendation_type, category, icon_url)
			VALUES (?,?,?,?)`, r.id, "Progressive", "MULTIPLAYER", r.icon); err != nil {
			t.Fatalf("insert %s: %v", r.id, err)
		}
	}
	texts := []struct{ id, lang, name string }{
		{"uuid-1", "fr-FR", "Tueur de Spartans"}, {"uuid-1", "en-US", "Spartan Slayer"},
		{"uuid-2", "en-US", "Headshot Honcho"}, // pas de ligne fr-FR : repli sur l'anglais
		{"uuid-3", "fr-FR", "Sans Icône"}, {"uuid-3", "en-US", "No Icon"},
	}
	for _, r := range texts {
		if _, err := db.ExecContext(ctx, `INSERT INTO asset_translations (asset_id, asset_type, lang, name)
			VALUES (?, 'commendation', ?, ?)`, r.id, r.lang, r.name); err != nil {
			t.Fatalf("insert texte %s/%s: %v", r.id, r.lang, err)
		}
	}
}

func TestHalo5CommendationDefs_LookupResolvesNameIconFallback(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	seedCommendationDefs(t, db)

	src := NewHalo5CommendationDefSource(db)
	// uuid-2 demandé en double (dédup) + uuid-inconnu (absent du résultat).
	got, err := src.LookupCommendations(context.Background(),
		[]string{"uuid-1", "uuid-2", "uuid-2", "uuid-3", "uuid-unknown", ""})
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("résultat = %d, want 3 (uuid-1,2,3 ; inconnu+vide exclus) — %+v", len(got), got)
	}
	// Langue par défaut : le texte de la langue quand il existe.
	if got["uuid-1"].Name != "Tueur de Spartans" {
		t.Errorf("uuid-1 name = %q, want FR 'Tueur de Spartans'", got["uuid-1"].Name)
	}
	if got["uuid-1"].IconURL != "https://cdn/1.png" {
		t.Errorf("uuid-1 icon = %q", got["uuid-1"].IconURL)
	}
	// Pas de ligne dans la langue → repli sur la langue de repli.
	if got["uuid-2"].Name != "Headshot Honcho" {
		t.Errorf("uuid-2 name = %q, want fallback EN 'Headshot Honcho'", got["uuid-2"].Name)
	}
	// icon vide → chaîne vide (l'adapter ne posera pas d'IconURL).
	if got["uuid-3"].IconURL != "" {
		t.Errorf("uuid-3 icon = %q, want vide", got["uuid-3"].IconURL)
	}
	if _, ok := got["uuid-unknown"]; ok {
		t.Error("uuid-unknown ne doit pas être dans le résultat")
	}
}

// TestHalo5CommendationDefs_LocaleAware verrouille la résolution par locale : le texte de la
// langue demandée, sinon celui de la langue de repli ; une langue inconnue du contexte prend la
// langue par défaut (i18n.Parse), jamais un texte inventé.
func TestHalo5CommendationDefs_LocaleAware(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	seedCommendationDefs(t, db)
	src := NewHalo5CommendationDefSource(db)
	ids := []string{"uuid-1", "uuid-2", "uuid-3"}

	cases := []struct {
		name   string
		ctx    context.Context
		wanted map[string]string
	}{
		{"fr (contexte par défaut)", context.Background(),
			map[string]string{"uuid-1": "Tueur de Spartans", "uuid-2": "Headshot Honcho", "uuid-3": "Sans Icône"}},
		{"fr explicite", ctxkeys.WithLocale(context.Background(), "fr"),
			map[string]string{"uuid-1": "Tueur de Spartans", "uuid-2": "Headshot Honcho", "uuid-3": "Sans Icône"}},
		{"en : jamais de français", ctxkeys.WithLocale(context.Background(), "en"),
			map[string]string{"uuid-1": "Spartan Slayer", "uuid-2": "Headshot Honcho", "uuid-3": "No Icon"}},
		{"en-US normalisé", ctxkeys.WithLocale(context.Background(), "en-US"),
			map[string]string{"uuid-1": "Spartan Slayer", "uuid-2": "Headshot Honcho", "uuid-3": "No Icon"}},
		{"langue inconnue : langue par défaut", ctxkeys.WithLocale(context.Background(), "zz"),
			map[string]string{"uuid-1": "Tueur de Spartans", "uuid-2": "Headshot Honcho", "uuid-3": "Sans Icône"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := src.LookupCommendations(c.ctx, ids)
			if err != nil {
				t.Fatalf("lookup: %v", err)
			}
			for id, want := range c.wanted {
				if got[id].Name != want {
					t.Errorf("%s = %q, want %q", id, got[id].Name, want)
				}
			}
		})
	}
}

func TestHalo5CommendationDefs_NilAndEmpty(t *testing.T) {
	// meta nil → map vide, pas d'erreur.
	nilSrc := NewHalo5CommendationDefSource(nil)
	got, err := nilSrc.LookupCommendations(context.Background(), []string{"x"})
	if err != nil || len(got) != 0 {
		t.Errorf("meta nil: got=%v err=%v, want map vide sans erreur", got, err)
	}
	// ids vide → map vide.
	db, _ := sql.Open("duckdb", ":memory:")
	defer db.Close()
	seedCommendationDefs(t, db)
	src := NewHalo5CommendationDefSource(db)
	got, err = src.LookupCommendations(context.Background(), nil)
	if err != nil || len(got) != 0 {
		t.Errorf("ids vide: got=%v err=%v, want map vide", got, err)
	}
}
