// Package duckdb — engagement_map_fr_test.go : test interne de resolveMapNameFR
// (resolution du nom de map FR via metadata.asset_translations, car
// match_registry.map_name_fr est systematiquement NULL).
//
// Test interne (package duckdb) pour acceder a la methode non exportee +
// construire un EngagementScoreRepo minimal avec seulement Metadata cable.
package duckdb

import (
	"context"
	"testing"
)

func TestResolveMapNameFR(t *testing.T) {
	meta, err := OpenReadWrite(":memory:")
	if err != nil {
		t.Fatalf("OpenReadWrite metadata: %v", err)
	}
	t.Cleanup(func() { _ = meta.Close() })

	ctx := context.Background()
	ddl := []string{
		`CREATE TABLE asset_translations (
			asset_id    VARCHAR,
			asset_type  VARCHAR,
			lang        VARCHAR,
			name        VARCHAR,
			description VARCHAR,
			fetched_at  TIMESTAMP
		)`,
		// The Pit : EN "The Pit" / FR "La fosse" (cas EN != FR).
		`INSERT INTO asset_translations VALUES ('648ae7aa','map','fr-FR','La fosse','',now())`,
		`INSERT INTO asset_translations VALUES ('648ae7aa','map','en-US','The Pit','',now())`,
		// Aquarius : FR == EN (cas degenere mais valide).
		`INSERT INTO asset_translations VALUES ('33c0766c','map','fr-FR','Aquarius','',now())`,
		// Map avec seulement un fallback 'fr' (pas 'fr-FR').
		`INSERT INTO asset_translations VALUES ('ffff0001','map','fr','Repli','',now())`,
		// Bruit : un playlist FR ne doit PAS matcher asset_type='map'.
		`INSERT INTO asset_translations VALUES ('648ae7aa','playlist','fr-FR','NE PAS PRENDRE','',now())`,
	}
	for _, s := range ddl {
		if _, err := meta.Exec(ctx, s); err != nil {
			t.Fatalf("seed %q: %v", s, err)
		}
	}

	repo := &EngagementScoreRepo{pdb: &PlayerDB{Metadata: meta}}

	cases := []struct {
		name      string
		mapID     string
		wantFR    string
		wantFound bool
	}{
		{"EN!=FR -> FR", "648ae7aa", "La fosse", true},
		{"FR==EN", "33c0766c", "Aquarius", true},
		{"fallback lang 'fr'", "ffff0001", "Repli", true},
		{"map_id inconnu -> fallback EN", "deadbeef", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := repo.resolveMapNameFR(ctx, tc.mapID)
			if ok != tc.wantFound {
				t.Fatalf("found = %v, want %v (got %q)", ok, tc.wantFound, got)
			}
			if ok && got != tc.wantFR {
				t.Errorf("FR = %q, want %q", got, tc.wantFR)
			}
		})
	}
}

// Metadata nil -> best-effort ("", false), jamais de panic.
func TestResolveMapNameFR_NilMetadata(t *testing.T) {
	repo := &EngagementScoreRepo{pdb: &PlayerDB{}}
	if fr, ok := repo.resolveMapNameFR(context.Background(), "648ae7aa"); ok || fr != "" {
		t.Errorf("attendu (\"\", false) sans Metadata, got (%q, %v)", fr, ok)
	}
}
