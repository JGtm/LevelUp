//go:build integration

package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

// TestMediaFilters_Audience : l'audience borne la galerie sans casse, l'emporte sur le
// filtre d'auteur du client, et une audience vide ne rend rien.
func TestMediaFilters_Audience(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMediaRepo(pdb)
	load := func(f domain.MediaFilters) []domain.MediaFileRow {
		t.Helper()
		rows, err := repo.LoadMediaFiles(context.Background(), f, 100, 0)
		if err != nil {
			t.Fatalf("LoadMediaFiles: %v", err)
		}
		return rows
	}

	aliceOnly := domain.NewMediaAudience("alice")
	assertOrder(t, load(domain.MediaFilters{Audience: &aliceOnly}),
		[]string{"/A2.png", "/A1.mp4"}, "audience=[alice]")

	// Le client demande Bob : hors audience, il ne l'obtient pas.
	assertOrder(t, load(domain.MediaFilters{Audience: &aliceOnly, AuthorSlugs: []string{"Alice", "Bob"}}),
		[]string{"/A2.png", "/A1.mp4"}, "audience=[alice], authorSlugs=[Alice,Bob]")

	nobody := domain.NewMediaAudience()
	if rows := load(domain.MediaFilters{Audience: &nobody}); len(rows) != 0 {
		t.Fatalf("audience vide : got %v, want []", pathsOf(rows))
	}
}
