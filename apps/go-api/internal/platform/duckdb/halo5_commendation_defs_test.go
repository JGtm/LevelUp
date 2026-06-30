//go:build integration

// halo5_commendation_defs_test.go — test in-memory (DuckDB :memory:) de la lecture
// du référentiel commendation_definitions (metadata h5). NE TOUCHE JAMAIS la vraie
// metadata h5.
//
// Lancer : go test -tags=integration ./internal/platform/duckdb/ -run Halo5Commendation

package duckdb

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/ctxkeys"

	_ "github.com/duckdb/duckdb-go/v2"
)

func seedCommendationDefs(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `CREATE TABLE commendation_definitions (
		commendation_id   VARCHAR PRIMARY KEY,
		name_en           VARCHAR NOT NULL,
		name_fr           VARCHAR NOT NULL,
		description_en    VARCHAR DEFAULT '',
		description_fr    VARCHAR DEFAULT '',
		commendation_type VARCHAR,
		category          VARCHAR,
		icon_url          VARCHAR,
		tier_targets      VARCHAR
	)`); err != nil {
		t.Fatalf("ddl: %v", err)
	}
	rows := []struct {
		id, nameEN, nameFR, icon string
	}{
		{"uuid-1", "Spartan Slayer", "Tueur de Spartans", "https://cdn/1.png"},
		{"uuid-2", "Headshot Honcho", "", "https://cdn/2.png"}, // name_fr vide → fallback EN
		{"uuid-3", "No Icon", "Sans Icône", ""},                // icon vide
	}
	for _, r := range rows {
		if _, err := db.ExecContext(ctx, `INSERT INTO commendation_definitions
			(commendation_id, name_en, name_fr, commendation_type, category, icon_url)
			VALUES (?,?,?,?,?,?)`, r.id, r.nameEN, r.nameFR, "Progressive", "MULTIPLAYER", r.icon); err != nil {
			t.Fatalf("insert %s: %v", r.id, err)
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
	// name_fr prioritaire quand non vide.
	if got["uuid-1"].Name != "Tueur de Spartans" {
		t.Errorf("uuid-1 name = %q, want FR 'Tueur de Spartans'", got["uuid-1"].Name)
	}
	if got["uuid-1"].IconURL != "https://cdn/1.png" {
		t.Errorf("uuid-1 icon = %q", got["uuid-1"].IconURL)
	}
	// name_fr vide → fallback name_en.
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

// TestHalo5CommendationDefs_LocaleAware verrouille la résolution par locale : EN →
// name_en, FR (défaut) → name_fr, avec repli sur l'autre langue si vide. C'est le
// correctif du bug "titres FR en UI anglaise" (le reader servait toujours name_fr).
func TestHalo5CommendationDefs_LocaleAware(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	seedCommendationDefs(t, db)
	src := NewHalo5CommendationDefSource(db)
	ids := []string{"uuid-1", "uuid-2", "uuid-3"}

	// FR (contexte par défaut) : name_fr prioritaire ; uuid-2 fr vide → fallback EN.
	fr, err := src.LookupCommendations(context.Background(), ids)
	if err != nil {
		t.Fatalf("lookup FR: %v", err)
	}
	if fr["uuid-1"].Name != "Tueur de Spartans" {
		t.Errorf("FR uuid-1 = %q, want 'Tueur de Spartans'", fr["uuid-1"].Name)
	}
	if fr["uuid-2"].Name != "Headshot Honcho" {
		t.Errorf("FR uuid-2 = %q, want fallback EN 'Headshot Honcho'", fr["uuid-2"].Name)
	}

	// EN : name_en prioritaire (uuid-1 anglais malgré un name_fr non vide).
	en, err := src.LookupCommendations(ctxkeys.WithLocale(context.Background(), "en"), ids)
	if err != nil {
		t.Fatalf("lookup EN: %v", err)
	}
	if en["uuid-1"].Name != "Spartan Slayer" {
		t.Errorf("EN uuid-1 = %q, want 'Spartan Slayer'", en["uuid-1"].Name)
	}
	if en["uuid-3"].Name != "No Icon" {
		t.Errorf("EN uuid-3 = %q, want 'No Icon'", en["uuid-3"].Name)
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
