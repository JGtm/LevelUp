//go:build integration

// prestige_arc_titles_test.go — voie cross-titre (arc_titles) côté repo :
// invariant Create, lectures ArcTitles / ArcsByTitle, fallback pré-backfill.
// Comportement strictement équivalent au mono-titre actuel
// (PLAN_CROSS_TITLE_ARCS_BACKEND Phase 2).
package prestige

import (
	"context"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/prestige"
)

// TestPrestigeArcRepo_CrossTitleJoin : Create maintient l'invariant arc_titles
// (1 ligne/arc), ArcTitles lit la jointure, et ArcsByTitle ≡ ListByUser en
// mono-titre.
func TestPrestigeArcRepo_CrossTitleJoin(t *testing.T) {
	db := setupPrestigeDB(t, migration.TargetPlayer)
	repo := NewPrestigeArcRepo(db)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	a := prestige.Arc{ID: "arc_x", UserID: "u1", TitleSlug: "halo_infinite", Title: "X", CreatedAt: now}
	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Create a inséré la ligne de jointure.
	titles, err := repo.ArcTitles(ctx, "arc_x")
	if err != nil {
		t.Fatalf("ArcTitles: %v", err)
	}
	if len(titles) != 1 || titles[0] != "halo_infinite" {
		t.Errorf("ArcTitles = %v, want [halo_infinite]", titles)
	}

	// ArcsByTitle ≡ ListByUser en mono-titre.
	byTitle, err := repo.ArcsByTitle(ctx, "u1", "halo_infinite")
	if err != nil {
		t.Fatalf("ArcsByTitle: %v", err)
	}
	byUser, err := repo.ListByUser(ctx, "u1", "halo_infinite")
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(byTitle) != 1 || len(byUser) != 1 {
		t.Fatalf("ArcsByTitle=%d ListByUser=%d, want 1 chacun", len(byTitle), len(byUser))
	}
	if byTitle[0].ID != "arc_x" {
		t.Errorf("ArcsByTitle[0].ID = %q, want arc_x", byTitle[0].ID)
	}

	// Autre titre → aucun arc.
	other, err := repo.ArcsByTitle(ctx, "u1", "autre_titre")
	if err != nil {
		t.Fatalf("ArcsByTitle other: %v", err)
	}
	if len(other) != 0 {
		t.Errorf("ArcsByTitle(autre_titre) = %d, want 0", len(other))
	}
}

// TestPrestigeArcRepo_ArcTitlesFallback : un arc présent dans `arc` mais SANS
// ligne arc_titles (état pré-backfill) → ArcTitles retombe sur le titre primaire.
func TestPrestigeArcRepo_ArcTitlesFallback(t *testing.T) {
	db := setupPrestigeDB(t, migration.TargetPlayer)
	ctx := context.Background()

	// Insert direct dans arc, SANS ligne arc_titles (simule legacy pré-backfill).
	if _, err := db.Exec(ctx,
		`INSERT INTO arc (id, user_id, title_slug, title, is_preset) VALUES ('arc_legacy','u9','halo_infinite','L',FALSE)`,
	); err != nil {
		t.Fatalf("seed legacy arc: %v", err)
	}

	repo := NewPrestigeArcRepo(db)
	titles, err := repo.ArcTitles(ctx, "arc_legacy")
	if err != nil {
		t.Fatalf("ArcTitles fallback: %v", err)
	}
	if len(titles) != 1 || titles[0] != "halo_infinite" {
		t.Errorf("ArcTitles fallback = %v, want [halo_infinite]", titles)
	}
}
