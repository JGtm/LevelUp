//go:build integration

// Package duckdb — career_repo_q7_monotone_test.go : tests Q7CareerXPHistory
// (lecture monotone post-suppression du fallback rank*1000).
//
// Lancer : go test -tags=integration ./internal/platform/duckdb/ -run TestQ7XPHistory -v
package duckdb

import (
	"context"
	"testing"
)

// resetCareerProgression vide la table career_progression seedée par
// newTestPlayerDB pour partir d'un état propre.
func resetCareerProgression(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	if _, err := pdb.Player.Exec(context.Background(), `DELETE FROM career_progression`); err != nil {
		t.Fatalf("reset career_progression: %v", err)
	}
}

// insertCareerRow insère une row career_progression — xp_total nil = NULL en DB.
// current_xp est toujours fixé à 0 (le scan Go XPHistoryPoint.CurrentXP est int
// non-nullable ; les vraies rows partials ont current_xp défini par
// PartialFromLive même quand xp_total est nil).
func insertCareerRow(t *testing.T, pdb *PlayerDB, recordedAt string, rank int, xpTotal *int) {
	t.Helper()
	ctx := context.Background()
	if xpTotal == nil {
		_, err := pdb.Player.Exec(ctx,
			`INSERT INTO career_progression (xuid, rank, current_xp, recorded_at) VALUES (?, ?, ?, ?)`,
			pTestXUID, rank, 0, recordedAt)
		if err != nil {
			t.Fatalf("insert null xp_total: %v", err)
		}
		return
	}
	_, err := pdb.Player.Exec(ctx,
		`INSERT INTO career_progression (xuid, rank, current_xp, recorded_at, xp_total) VALUES (?, ?, ?, ?, ?)`,
		pTestXUID, rank, 0, recordedAt, *xpTotal)
	if err != nil {
		t.Fatalf("insert xp_total=%d: %v", *xpTotal, err)
	}
}

// TestQ7XPHistory_FiltersNullXPTotal vérifie qu'un row sans xp_total réel
// (xp_total NULL, ex: INSERT partial customization-only) est exclu.
func TestQ7XPHistory_FiltersNullXPTotal(t *testing.T) {
	pdb := newTestPlayerDB(t)
	resetCareerProgression(t, pdb)

	xp1 := 5_000_000
	insertCareerRow(t, pdb, "2026-05-20 10:00:00+00", 300, &xp1)
	insertCareerRow(t, pdb, "2026-05-21 10:00:00+00", 300, nil) // partial customization-only

	history, err := NewCareerRepo(pdb).GetXPHistory(context.Background())
	if err != nil {
		t.Fatalf("GetXPHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 row (NULL xp_total filtered), got %d", len(history))
	}
	if history[0].XPTotal != 5_000_000 {
		t.Errorf("expected XPTotal=5M, got %d", history[0].XPTotal)
	}
}

// TestQ7XPHistory_FiltersZeroXPTotal vérifie qu'un row avec xp_total=0
// (legacy avant la migration fix_career_xp_total_default_zero) est exclu.
func TestQ7XPHistory_FiltersZeroXPTotal(t *testing.T) {
	pdb := newTestPlayerDB(t)
	resetCareerProgression(t, pdb)

	xp1 := 5_000_000
	zero := 0
	insertCareerRow(t, pdb, "2026-05-20 10:00:00+00", 300, &xp1)
	insertCareerRow(t, pdb, "2026-05-21 10:00:00+00", 300, &zero) // legacy DEFAULT 0

	history, err := NewCareerRepo(pdb).GetXPHistory(context.Background())
	if err != nil {
		t.Fatalf("GetXPHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 row (xp_total=0 filtered), got %d", len(history))
	}
	if history[0].XPTotal != 5_000_000 {
		t.Errorf("expected XPTotal=5M, got %d", history[0].XPTotal)
	}
}

// TestQ7XPHistory_MonotoneMaskAberrantDip vérifie que MAX OVER masque un
// row aberrant qui ferait régresser la courbe (defense-in-depth).
func TestQ7XPHistory_MonotoneMaskAberrantDip(t *testing.T) {
	pdb := newTestPlayerDB(t)
	resetCareerProgression(t, pdb)

	xp1 := 5_000_000
	xp2 := 100_000 // aberrant (régression)
	xp3 := 5_100_000
	insertCareerRow(t, pdb, "2026-05-20 10:00:00+00", 300, &xp1)
	insertCareerRow(t, pdb, "2026-05-21 10:00:00+00", 300, &xp2)
	insertCareerRow(t, pdb, "2026-05-22 10:00:00+00", 300, &xp3)

	history, err := NewCareerRepo(pdb).GetXPHistory(context.Background())
	if err != nil {
		t.Fatalf("GetXPHistory: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(history))
	}
	wantXP := []int{5_000_000, 5_000_000, 5_100_000}
	for i, p := range history {
		if p.XPTotal != wantXP[i] {
			t.Errorf("row %d: expected XPTotal=%d (monotone), got %d", i, wantXP[i], p.XPTotal)
		}
	}
}

// TestQ7XPHistory_StrictlyIncreasing vérifie une suite normale monotone.
func TestQ7XPHistory_StrictlyIncreasing(t *testing.T) {
	pdb := newTestPlayerDB(t)
	resetCareerProgression(t, pdb)

	xp := []int{1_000_000, 2_000_000, 3_500_000, 5_000_000}
	dates := []string{
		"2026-05-01 10:00:00+00",
		"2026-05-10 10:00:00+00",
		"2026-05-20 10:00:00+00",
		"2026-05-25 10:00:00+00",
	}
	for i := range xp {
		insertCareerRow(t, pdb, dates[i], 100+i*50, &xp[i])
	}

	history, err := NewCareerRepo(pdb).GetXPHistory(context.Background())
	if err != nil {
		t.Fatalf("GetXPHistory: %v", err)
	}
	if len(history) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(history))
	}
	for i, p := range history {
		if p.XPTotal != xp[i] {
			t.Errorf("row %d: expected %d, got %d", i, xp[i], p.XPTotal)
		}
	}
}

// TestQ7XPHistory_RanksOnlyExcluded vérifie qu'une série avec uniquement rank
// (xp_total toujours NULL) ne renvoie aucun row — au lieu de tracer une courbe
// à rank*1000 qui était le comportement destructeur pré-fix.
func TestQ7XPHistory_RanksOnlyExcluded(t *testing.T) {
	pdb := newTestPlayerDB(t)
	resetCareerProgression(t, pdb)

	insertCareerRow(t, pdb, "2026-05-20 10:00:00+00", 10, nil)
	insertCareerRow(t, pdb, "2026-05-21 10:00:00+00", 20, nil)
	insertCareerRow(t, pdb, "2026-05-22 10:00:00+00", 30, nil)

	history, err := NewCareerRepo(pdb).GetXPHistory(context.Background())
	if err != nil {
		t.Fatalf("GetXPHistory: %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("expected 0 rows (all xp_total NULL), got %d", len(history))
	}
}

// TestQ7XPHistory_MixedRanksOnlyAndRealXP vérifie qu'un préfix de rows
// rank-only est exclu et que la monotonie démarre proprement au premier
// xp_total > 0.
func TestQ7XPHistory_MixedRanksOnlyAndRealXP(t *testing.T) {
	pdb := newTestPlayerDB(t)
	resetCareerProgression(t, pdb)

	insertCareerRow(t, pdb, "2026-05-19 10:00:00+00", 10, nil) // exclu
	xp1 := 1_000_000
	xp2 := 1_500_000
	insertCareerRow(t, pdb, "2026-05-20 10:00:00+00", 100, &xp1)
	insertCareerRow(t, pdb, "2026-05-21 10:00:00+00", 110, &xp2)

	history, err := NewCareerRepo(pdb).GetXPHistory(context.Background())
	if err != nil {
		t.Fatalf("GetXPHistory: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(history))
	}
	if history[0].XPTotal != 1_000_000 || history[1].XPTotal != 1_500_000 {
		t.Errorf("unexpected XP sequence: [%d, %d]", history[0].XPTotal, history[1].XPTotal)
	}
}

// TestQ7XPHistory_MadinaRegressionScenario reproduit le bug originel reporté
// pour Madina97294 : une row le 25 mai avec rank=300 et xp_total=0/NULL.
// Pré-fix : Q7 renvoyait rank*1000 = 300_000 → régression visuelle de 5M à 300k.
// Post-fix : la row est filtrée, projection préservée car xp_total monotone.
func TestQ7XPHistory_MadinaRegressionScenario(t *testing.T) {
	pdb := newTestPlayerDB(t)
	resetCareerProgression(t, pdb)

	xpStart := 5_000_000
	insertCareerRow(t, pdb, "2026-05-20 10:00:00+00", 300, &xpStart)
	// 25 mai : sync où API n'a pas renvoyé TotalEarned (xp_total NULL).
	// Pré-fix : COALESCE(NULL, 300*1000) = 300_000 → courbe chute.
	insertCareerRow(t, pdb, "2026-05-25 10:00:00+00", 300, nil)

	history, err := NewCareerRepo(pdb).GetXPHistory(context.Background())
	if err != nil {
		t.Fatalf("GetXPHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 row (NULL filtered), got %d", len(history))
	}
	if history[0].XPTotal != 5_000_000 {
		t.Errorf("expected XPTotal=5M (no regression), got %d", history[0].XPTotal)
	}
}
