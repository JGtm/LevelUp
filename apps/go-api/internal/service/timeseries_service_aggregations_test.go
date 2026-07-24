package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/legacymatch"
)

// ---------------------------------------------------------------------------
// buildSoloMapBreakdown
// ---------------------------------------------------------------------------

// TestBuildSoloMapBreakdown_OrderedByFirstAppearance verrouille le tri des
// cartes par ordre CHRONOLOGIQUE de première apparition, calqué sur
// TestComputeMapBreakdown_OrderedByFirstAppearance (teammates_extra_test.go,
// commit 59e705690 — I12). "Early" (1 match) est jouée avant "Mid" (2 matchs)
// avant "Late" (3 matchs) : l'ordre fréquence (match_count DESC, ancien tri)
// serait l'inverse (Late>Mid>Early), donc ce test distingue les deux tris.
func TestBuildSoloMapBreakdown_OrderedByFirstAppearance(t *testing.T) {
	base := time.Date(2026, 5, 1, 18, 0, 0, 0, time.UTC)
	win := analysis.OutcomeWin
	current := []legacymatch.StatsMatchRow{
		{MatchID: "l1", StartTime: base.Add(2 * time.Hour), MapNameFR: "Late", Outcome: &win},
		{MatchID: "l2", StartTime: base.Add(150 * time.Minute), MapNameFR: "Late", Outcome: &win},
		{MatchID: "l3", StartTime: base.Add(3 * time.Hour), MapNameFR: "Late", Outcome: &win},
		{MatchID: "m1", StartTime: base.Add(1 * time.Hour), MapNameFR: "Mid", Outcome: &win},
		{MatchID: "m2", StartTime: base.Add(90 * time.Minute), MapNameFR: "Mid", Outcome: &win},
		{MatchID: "e1", StartTime: base, MapNameFR: "Early", Outcome: &win},
	}

	rows := buildSoloMapBreakdown(current, nil)

	want := []string{"Early", "Mid", "Late"}
	if len(rows) != len(want) {
		t.Fatalf("want %d maps, got %d (%v)", len(want), len(rows), rows)
	}
	for i, w := range want {
		if rows[i].MapUI != w {
			t.Errorf("rows[%d]: want %q, got %q (ordre chronologique cassé, got order: %v)",
				i, w, rows[i].MapUI, rows)
		}
	}
}

// TestBuildSoloMapBreakdown_TieBreakMapUI verrouille le tie-break déterministe
// (MapUI asc) quand deux cartes partagent le même firstSeen — l'itération
// d'une map Go n'étant pas ordonnée, un tri sans tie-break serait non
// déterministe d'un run à l'autre (même piège que computeMapBreakdown).
func TestBuildSoloMapBreakdown_TieBreakMapUI(t *testing.T) {
	same := time.Date(2026, 5, 1, 18, 0, 0, 0, time.UTC)
	win := analysis.OutcomeWin
	current := []legacymatch.StatsMatchRow{
		{MatchID: "z1", StartTime: same, MapNameFR: "Zanzibar", Outcome: &win},
		{MatchID: "a1", StartTime: same, MapNameFR: "Aquarius", Outcome: &win},
	}

	rows := buildSoloMapBreakdown(current, nil)

	if len(rows) != 2 || rows[0].MapUI != "Aquarius" || rows[1].MapUI != "Zanzibar" {
		t.Fatalf("want [Aquarius, Zanzibar] (tie-break MapUI asc), got %v", rows)
	}
}
