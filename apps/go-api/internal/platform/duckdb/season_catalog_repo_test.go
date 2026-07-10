package duckdb

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openSeasonCatalogMemDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestLoadSeasonCatalogNames_RoundTripAndCase(t *testing.T) {
	db := openSeasonCatalogMemDB(t)
	if _, err := db.Exec(`CREATE TABLE season_catalog (
		title_slug VARCHAR, season_id VARCHAR, display_name VARCHAR, name_fr VARCHAR,
		season_major INTEGER, season_minor INTEGER, first_seen_at TIMESTAMP, last_fetched_at TIMESTAMP,
		PRIMARY KEY (title_slug, season_id))`); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO season_catalog
		(title_slug, season_id, display_name, name_fr, season_major, season_minor)
		VALUES ('halo_infinite','csrseason12-1','Shadows','Ombres',12,1)`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	names, err := LoadSeasonCatalogNames(context.Background(), db, "halo_infinite")
	if err != nil {
		t.Fatalf("LoadSeasonCatalogNames: %v", err)
	}
	sn, ok := names["csrseason12-1"] // clé minuscule
	if !ok || sn.NameFR != "Ombres" || sn.Major != 12 {
		t.Errorf("entrée = %+v (ok=%v), attendu {Shadows,Ombres,12,1}", sn, ok)
	}
	// Table absente → map vide, pas d'erreur (dégradation gracieuse).
	empty := openSeasonCatalogMemDB(t)
	got, err := LoadSeasonCatalogNames(context.Background(), empty, "halo_infinite")
	if err != nil || len(got) != 0 {
		t.Errorf("table absente : attendu (map vide, nil), got (%v, %v)", got, err)
	}
}

func TestSeasonSelectorLabel(t *testing.T) {
	names := map[string]SeasonName{
		"csrseason13-2": {DisplayName: "Infinite", NameFR: "", Major: 13, Minor: 2},
		"csrseason12-1": {DisplayName: "Shadows", NameFR: "Ombres", Major: 12, Minor: 1},
		"csrseason0-0":  {DisplayName: "Bootstrap", NameFR: "Amorce", Major: 0, Minor: 0},
	}
	cases := []struct {
		name, locale, seasonID, fallback, want string
	}{
		{"FR nom traduit", "fr", "csrseason12-1", "Saison 12", "Saison 12 · Ombres"},
		{"EN nom EN", "en", "csrseason12-1", "Season 12", "Season 12 · Shadows"},
		{"FR fallback EN si pas de NameFR", "fr", "csrseason13-2", "Saison 13", "Saison 13 · Infinite"},
		{"casse insensible (API carrière)", "fr", "CsrSeason12-1", "Saison 12", "Saison 12 · Ombres"},
		{"absent du catalogue → fallback", "fr", "csrseason9-1", "Saison 9", "Saison 9"},
		{"sans numéro → nom seul", "fr", "csrseason0-0", "brut", "Amorce"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SeasonSelectorLabel(c.locale, c.seasonID, names, c.fallback); got != c.want {
				t.Errorf("SeasonSelectorLabel(%q,%q) = %q, want %q", c.locale, c.seasonID, got, c.want)
			}
		})
	}
	// nil map (indisponible) → toujours le fallback.
	if got := SeasonSelectorLabel("fr", "csrseason12-1", nil, "Saison 12"); got != "Saison 12" {
		t.Errorf("nil map devrait retourner le fallback, got %q", got)
	}
}

func TestFallbackSeasonLabel(t *testing.T) {
	cases := []struct{ locale, id, want string }{
		{"fr", "csrseason13-2", "Saison 13"},
		{"en", "csrseason13-2", "Season 13"},
		{"fr", "garbage", "garbage"},
	}
	for _, c := range cases {
		if got := fallbackSeasonLabel(c.locale, c.id); got != c.want {
			t.Errorf("fallbackSeasonLabel(%q,%q) = %q, want %q", c.locale, c.id, got, c.want)
		}
	}
}
