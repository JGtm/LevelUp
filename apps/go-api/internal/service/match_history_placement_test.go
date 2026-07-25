package service

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/sync"
)

func TestParsePlacementRemaining(t *testing.T) {
	cases := []struct {
		label string
		want  int
	}{
		{"Placement (4 restants)", 4},
		{"Placement (10 restants)", 10},
		{"Placement (0 restant)", 0},
		{"Placement", 10},  // fallback : pas de "(N restant)"
		{"Diamant IV", 10}, // fallback : pas un label de placement
		{"", 10},
		{"Placement (999 restants)", 10}, // out of bounds → fallback
	}
	for _, c := range cases {
		if got := parsePlacementRemaining(c.label); got != c.want {
			t.Errorf("parsePlacementRemaining(%q) = %d, want %d", c.label, got, c.want)
		}
	}
}

func TestApplyCSRPlacements_threshold5(t *testing.T) {
	rows := []domain.MatchHistoryRawRow{
		{MatchID: "m1", SkillRatingType: strPtr("CSR"), SkillTierLabel: strPtr("Placement (4 restants)"), SeasonID: strPtr("CsrSeason13-1")},
		{MatchID: "m2", SkillRatingType: strPtr("CSR"), SkillTierLabel: strPtr("Diamant IV"), SeasonID: strPtr("CsrSeason13-1")}, // tier officiel → pas placement
		{MatchID: "m3", SkillRatingType: strPtr("LUSR"), SkillTierLabel: strPtr("Placement (3 restants)")},                       // LUSR → pas concerné par applyCSRPlacements
		{MatchID: "m4"}, // pas de rating → ignore
	}
	csrResolver := func(_ context.Context, _ string) int { return 5 }
	if got := applyCSRPlacements(context.Background(), rows, csrResolver); got != 1 {
		t.Errorf("count want 1, got %d", got)
	}

	if rows[0].PlacementDone == nil || *rows[0].PlacementDone != 1 {
		t.Errorf("m1: PlacementDone want 1 (= 5-4), got %v", rows[0].PlacementDone)
	}
	if rows[0].PlacementTotal == nil || *rows[0].PlacementTotal != 5 {
		t.Errorf("m1: PlacementTotal want 5, got %v", rows[0].PlacementTotal)
	}
	if rows[1].PlacementDone != nil {
		t.Errorf("m2: tier officiel ne doit pas être en placement, got %v", rows[1].PlacementDone)
	}
	if rows[2].PlacementDone != nil {
		t.Errorf("m3: LUSR ne doit pas être affecté par applyCSRPlacements, got %v", rows[2].PlacementDone)
	}
}

func TestApplyCSRPlacements_threshold10_legacySeason(t *testing.T) {
	rows := []domain.MatchHistoryRawRow{
		{MatchID: "m1", SkillRatingType: strPtr("CSR"), SkillTierLabel: strPtr("Placement (7 restants)"), SeasonID: strPtr("CsrSeason1")},
	}
	csrResolver := func(_ context.Context, seasonID string) int {
		if seasonID == "CsrSeason1" {
			return 10
		}
		return 5
	}
	applyCSRPlacements(context.Background(), rows, csrResolver)
	if rows[0].PlacementDone == nil || *rows[0].PlacementDone != 3 {
		t.Errorf("PlacementDone want 3 (= 10-7), got %v", rows[0].PlacementDone)
	}
	if rows[0].PlacementTotal == nil || *rows[0].PlacementTotal != 10 {
		t.Errorf("PlacementTotal want 10, got %v", rows[0].PlacementTotal)
	}
}

func TestApplyLUSRPlacements_first10PerChain(t *testing.T) {
	// 12 matchs Slayer Arena (chaîne "arena_slayer") sans LUSR, ordre temporel
	// inverse pour valider le tri ASC dans la fonction.
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := []domain.MatchHistoryRawRow{}
	for i := 12; i >= 1; i-- {
		ts := base.Add(time.Duration(i) * time.Hour)
		rows = append(rows, domain.MatchHistoryRawRow{
			MatchID:   "slayer-" + string(rune('a'+i-1)),
			StartTime: &ts,
			PairName:  strPtr("Arena:Slayer on Bazaar"),
		})
	}

	if got := applyLUSRPlacements(rows); got != 10 {
		t.Errorf("applyLUSRPlacements returned %d, want 10", got)
	}

	// Les 10 matchs les plus anciens doivent être marqués 1/10 → 10/10.
	// Tri attendu : matchs i=1..10 (les plus anciens car +i heures), donc on
	// trie d'abord les rows par start_time ASC pour vérifier.
	placementCount := 0
	for _, r := range rows {
		if r.PlacementDone != nil {
			placementCount++
			if r.PlacementTotal == nil || *r.PlacementTotal != 10 {
				t.Errorf("%s: PlacementTotal want 10, got %v", r.MatchID, r.PlacementTotal)
			}
		}
	}
	if placementCount != 10 {
		t.Errorf("want 10 matchs en placement, got %d", placementCount)
	}
}

func TestApplyMatchPlacements_csrAndLusrCombined(t *testing.T) {
	ts := time.Now()
	rows := []domain.MatchHistoryRawRow{
		// CSR placement → done = 5-2 = 3, total = 5
		{MatchID: "csr1", StartTime: &ts, PairName: strPtr("Ranked:Slayer"),
			SkillRatingType: strPtr("CSR"), SkillTierLabel: strPtr("Placement (2 restants)"),
			SeasonID: strPtr("CsrSeason13-1")},
		// LUSR placement (chaîne arena_slayer, premier match) → 1/10
		{MatchID: "lusr1", StartTime: &ts, PairName: strPtr("Arena:Slayer on Bazaar")},
		// Tier officiel CSR → ignoré
		{MatchID: "csr_full", StartTime: &ts, PairName: strPtr("Ranked:Slayer"),
			SkillRatingType: strPtr("CSR"), SkillTierLabel: strPtr("Diamant IV")},
	}
	csrResolver := func(_ context.Context, _ string) int { return 5 }
	applyMatchPlacements(context.Background(), rows, csrResolver)

	if rows[0].PlacementDone == nil || *rows[0].PlacementDone != 3 || *rows[0].PlacementTotal != 5 {
		t.Errorf("csr1 want 3/5, got done=%v total=%v", rows[0].PlacementDone, rows[0].PlacementTotal)
	}
	if rows[1].PlacementDone == nil || *rows[1].PlacementDone != 1 || *rows[1].PlacementTotal != 10 {
		t.Errorf("lusr1 want 1/10, got done=%v total=%v", rows[1].PlacementDone, rows[1].PlacementTotal)
	}
	if rows[2].PlacementDone != nil {
		t.Errorf("csr_full (tier officiel) ne doit pas être en placement, got %v", rows[2].PlacementDone)
	}
}

func TestApplyMatchPlacements_nilCsrResolver(t *testing.T) {
	rows := []domain.MatchHistoryRawRow{
		{MatchID: "m1", SkillRatingType: strPtr("CSR"), SkillTierLabel: strPtr("Placement (2 restants)")},
	}
	// csrThreshold = nil → fallback à defaultCSRPlacementThreshold (5)
	applyMatchPlacements(context.Background(), rows, nil)
	if rows[0].PlacementTotal == nil || *rows[0].PlacementTotal != defaultCSRPlacementThreshold {
		t.Errorf("fallback threshold want %d, got %v", defaultCSRPlacementThreshold, rows[0].PlacementTotal)
	}
	if rows[0].PlacementDone == nil || *rows[0].PlacementDone != 3 {
		t.Errorf("done want 3 (5-2), got %v", rows[0].PlacementDone)
	}
}

func TestApplyLUSRPlacements_skipMatchesWithLUSRorCSR(t *testing.T) {
	ts := time.Now()
	rows := []domain.MatchHistoryRawRow{
		{MatchID: "m1", StartTime: &ts, PairName: strPtr("Arena:Slayer"), SkillRatingType: strPtr("LUSR")},
		{MatchID: "m2", StartTime: &ts, PairName: strPtr("Arena:Slayer"), SkillRatingType: strPtr("CSR")},
		{MatchID: "m3", StartTime: &ts, PairName: strPtr("Arena:Slayer")}, // sans rating → placement
	}
	applyLUSRPlacements(rows)
	if rows[0].PlacementDone != nil {
		t.Errorf("m1 (LUSR existant) ne doit pas être en placement")
	}
	if rows[1].PlacementDone != nil {
		t.Errorf("m2 (CSR existant) ne doit pas être en placement")
	}
	if rows[2].PlacementDone == nil || *rows[2].PlacementDone != 1 {
		t.Errorf("m3 (sans rating, chaîne LUSR) doit être placement 1/10, got %v", rows[2].PlacementDone)
	}
}

func TestApplyLUSRPlacements_skipRankedAndFirefight(t *testing.T) {
	ts := time.Now()
	rows := []domain.MatchHistoryRawRow{
		{MatchID: "ranked", StartTime: &ts, PairName: strPtr("Ranked:Slayer")},     // chaîne "" (Ranked → CSR)
		{MatchID: "firefight", StartTime: &ts, PairName: strPtr("Firefight:KOTH")}, // chaîne "" (Firefight → PvE)
		{MatchID: "btb", StartTime: &ts, PairName: strPtr("BTB:CTF on Highpower")}, // chaîne "btb"
	}
	applyLUSRPlacements(rows)
	if rows[0].PlacementDone != nil {
		t.Errorf("ranked ne doit pas être placement LUSR")
	}
	if rows[1].PlacementDone != nil {
		t.Errorf("firefight ne doit pas être placement LUSR")
	}
	if rows[2].PlacementDone == nil || *rows[2].PlacementDone != 1 {
		t.Errorf("btb doit être placement 1/10, got %v", rows[2].PlacementDone)
	}
}

// TestApplyLUSRPlacements_bigTeamBattle_fewerThan10_userReportedScenario reproduit
// le signalement V72-32 : un joueur ayant joué SEULEMENT quelques matchs Big Team
// Battle (chaîne "btb") au total, dont plusieurs "hier". Confirme le mécanisme
// exact — < sync.LUSRPlacementThreshold (10) matchs dans LA CHAÎNE BTB, pas un
// total global de matchs joués — et que TOUS les matchs BTB de la chaîne (donc
// ceux d'hier) reçoivent PlacementDone/PlacementTotal, jamais nil silencieux.
// C'est ce signal que consomme ExplorerMatchesTable.placement.tsx (front) pour
// afficher « En placement » sur Perf/ΔPerf/Note à la place du "-".
func TestApplyLUSRPlacements_bigTeamBattle_fewerThan10_userReportedScenario(t *testing.T) {
	base := time.Date(2026, 7, 20, 18, 0, 0, 0, time.UTC)
	// 6 matchs BTB au total (< 10) : 3 "il y a plusieurs jours" + 3 "hier"
	// (24/07), aucun n'a encore de LUSR (chaîne trop jeune).
	rows := []domain.MatchHistoryRawRow{
		{MatchID: "btb-old-1", StartTime: tPtr(base), PairName: strPtr("BTB:CTF on Highpower")},
		{MatchID: "btb-old-2", StartTime: tPtr(base.Add(24 * time.Hour)), PairName: strPtr("BTB:Slayer on Behemoth")},
		{MatchID: "btb-old-3", StartTime: tPtr(base.Add(48 * time.Hour)), PairName: strPtr("BTB:Stockpile on Deadlock")},
		{MatchID: "btb-yday-1", StartTime: tPtr(base.Add(96 * time.Hour)), PairName: strPtr("BTB:CTF on Fragmentation")},
		{MatchID: "btb-yday-2", StartTime: tPtr(base.Add(97 * time.Hour)), PairName: strPtr("BTB:Slayer on Highpower")},
		{MatchID: "btb-yday-3", StartTime: tPtr(base.Add(98 * time.Hour)), PairName: strPtr("BTB:Stockpile on Behemoth")},
		// Un match Ranked le même jour ne doit pas polluer la chaîne "btb".
		{MatchID: "ranked-same-day", StartTime: tPtr(base.Add(50 * time.Hour)), PairName: strPtr("Ranked:Slayer")},
	}
	count := applyLUSRPlacements(rows)
	if count != 6 {
		t.Fatalf("6 matchs BTB attendus en placement (< 10 dans la chaîne), got %d", count)
	}
	for i, want := range []int{1, 2, 3, 4, 5, 6} {
		r := rows[i]
		if r.PlacementDone == nil || *r.PlacementDone != want {
			t.Errorf("%s: PlacementDone want %d, got %v", r.MatchID, want, r.PlacementDone)
		}
		if r.PlacementTotal == nil || *r.PlacementTotal != sync.LUSRPlacementThreshold {
			t.Errorf("%s: PlacementTotal want %d, got %v", r.MatchID, sync.LUSRPlacementThreshold, r.PlacementTotal)
		}
	}
	if rows[6].PlacementDone != nil {
		t.Errorf("ranked-same-day (chaîne différente) ne doit pas être en placement BTB, got %v", rows[6].PlacementDone)
	}
}

func tPtr(t time.Time) *time.Time { return &t }
