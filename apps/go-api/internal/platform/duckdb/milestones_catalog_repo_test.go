//go:build integration

// Tests d'intégration de MilestoneCatalogRepo (metadata.duckdb).

package duckdb

import (
	"context"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/progression/milestones"
)

func newMilestoneCatalogRepoForTest(t *testing.T) *MilestoneCatalogRepo {
	t.Helper()
	db := setupPrestigeDB(t, migration.TargetMetadata)
	return NewMilestoneCatalogRepo(db)
}

func TestMilestoneCatalogRepo_UpsertAndList(t *testing.T) {
	repo := newMilestoneCatalogRepoForTest(t)
	ctx := context.Background()

	entries := []milestones.CatalogEntry{
		{ID: "h.matches.100", TitleSlug: "halo_infinite", Metric: "matches_played", Threshold: 100, TitleEN: "Centurion", TitleFR: "Centurion", Icon: "icon_100"},
		{ID: "h.matches.500", TitleSlug: "halo_infinite", Metric: "matches_played", Threshold: 500, TitleEN: "Veteran", TitleFR: "Vétéran"},
		{ID: "h.wins.50", TitleSlug: "halo_infinite", Metric: "wins", Threshold: 50, TitleEN: "Winner", TitleFR: "Vainqueur"},
		// Autre titre (devrait être exclu)
		{ID: "other.x.1", TitleSlug: "other_title", Metric: "x", Threshold: 1, TitleEN: "X", TitleFR: "X"},
	}
	for _, e := range entries {
		if err := repo.Upsert(ctx, e); err != nil {
			t.Fatalf("Upsert %s: %v", e.ID, err)
		}
	}

	got, err := repo.ListByTitle(ctx, "halo_infinite")
	if err != nil {
		t.Fatalf("ListByTitle: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("ListByTitle len = %d, want 3", len(got))
	}
	// Tri attendu : (metric, threshold) — matches_played avant wins, 100 avant 500.
	if got[0].ID != "h.matches.100" || got[1].ID != "h.matches.500" || got[2].ID != "h.wins.50" {
		t.Errorf("ordre incorrect : %s %s %s", got[0].ID, got[1].ID, got[2].ID)
	}
	// Icon round-trip
	if got[0].Icon != "icon_100" {
		t.Errorf("Icon = %q, want icon_100", got[0].Icon)
	}
}

func TestMilestoneCatalogRepo_UpsertOverwrites(t *testing.T) {
	repo := newMilestoneCatalogRepoForTest(t)
	ctx := context.Background()

	original := milestones.CatalogEntry{
		ID: "h.matches.100", TitleSlug: "halo_infinite", Metric: "matches_played", Threshold: 100,
		TitleEN: "Centurion", TitleFR: "Centurion",
	}
	if err := repo.Upsert(ctx, original); err != nil {
		t.Fatalf("Upsert v1: %v", err)
	}

	// Update libellé FR.
	updated := original
	updated.TitleFR = "Centurion (révisé)"
	if err := repo.Upsert(ctx, updated); err != nil {
		t.Fatalf("Upsert v2: %v", err)
	}

	got, err := repo.ListByTitle(ctx, "halo_infinite")
	if err != nil || len(got) != 1 {
		t.Fatalf("ListByTitle: got=%v err=%v", got, err)
	}
	if got[0].TitleFR != "Centurion (révisé)" {
		t.Errorf("TitleFR = %q, want updated", got[0].TitleFR)
	}
}
