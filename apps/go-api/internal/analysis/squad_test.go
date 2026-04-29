// Package analysis â€” squad_test.go : tests unitaires pour les algorithmes escouade.
package analysis

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// =============================================================================
// Tests ComputeSquadPerformanceScore
// =============================================================================

func TestComputeSquadPerformanceScore_NoScores(t *testing.T) {
	result := ComputeSquadPerformanceScore(nil, nil, nil, nil)
	if result.Score != nil {
		t.Errorf("score doit Ãªtre nil quand pas de donnÃ©es, got %v", result.Score)
	}
	if result.Grade != "N/A" {
		t.Errorf("grade doit Ãªtre N/A, got %s", result.Grade)
	}
}

func TestComputeSquadPerformanceScore_Basic(t *testing.T) {
	s1, s2 := 70.0, 80.0
	result := ComputeSquadPerformanceScore(
		[]*float64{&s1, &s2},
		[]float64{65.0, 70.0}, // win rate > 60 â†’ +5
		[]float64{1.5, 2.0},   // min KDA > 1 â†’ +5
		[]float64{8.0, 9.0},   // std < 3 â†’ +3
	)
	if result.Score == nil {
		t.Fatal("score ne doit pas Ãªtre nil")
	}
	// base = 75, bonus = 13 â†’ 88 (clamp 100)
	expected := 88.0
	if *result.Score != expected {
		t.Errorf("score attendu %.1f, got %.1f", expected, *result.Score)
	}
}

func TestComputeSquadPerformanceScore_NoBonus(t *testing.T) {
	s1, s2 := 50.0, 60.0
	result := ComputeSquadPerformanceScore(
		[]*float64{&s1, &s2},
		[]float64{40.0},      // win rate â‰¤ 60 â†’ pas de bonus
		[]float64{0.8},       // min KDA â‰¤ 1 â†’ pas de bonus
		[]float64{5.0, 15.0}, // std > 3 â†’ pas de bonus
	)
	if result.Score == nil {
		t.Fatal("score ne doit pas Ãªtre nil")
	}
	if *result.Score != 55.0 {
		t.Errorf("score attendu 55.0, got %.1f", *result.Score)
	}
}

func TestComputeSquadPerformanceScore_Clamp(t *testing.T) {
	s1, s2 := 95.0, 98.0
	result := ComputeSquadPerformanceScore(
		[]*float64{&s1, &s2},
		[]float64{65.0},     // +5
		[]float64{2.0},      // +5
		[]float64{3.0, 4.0}, // std > 3 â†’ pas de bonus
	)
	if result.Score == nil {
		t.Fatal("score ne doit pas Ãªtre nil")
	}
	if *result.Score > 100.0 {
		t.Errorf("score ne doit pas dÃ©passer 100, got %.1f", *result.Score)
	}
}

// =============================================================================
// Tests resolveSquadGrade
// =============================================================================

func TestResolveSquadGrade(t *testing.T) {
	cases := []struct {
		score float64
		grade string
	}{
		{95, "S"}, {85, "A"}, {70, "B"}, {55, "C"}, {40, "D"}, {20, "F"},
	}
	for _, tc := range cases {
		g := resolveSquadGrade(tc.score)
		if g != tc.grade {
			t.Errorf("score %.0f â†’ grade attendu %s, got %s", tc.score, tc.grade, g)
		}
	}
}

// =============================================================================
// Tests ComputeParticipationProfile
// =============================================================================

func TestComputeParticipationProfile_Empty(t *testing.T) {
	profile := ComputeParticipationProfile(nil, "Test", "#ff0000")
	if len(profile.Values) != 0 {
		t.Errorf("profil vide doit avoir 0 valeurs, got %d", len(profile.Values))
	}
}

func TestComputeParticipationProfile_Values(t *testing.T) {
	kda := 2.0
	acc := 60.0
	rows := []domain.SquadMatchRow{
		{Kills: 10, Deaths: 5, Assists: 3, KDA: &kda, Accuracy: &acc, TimePlayedSecs: 600},
		{Kills: 8, Deaths: 4, Assists: 2, KDA: &kda, Accuracy: &acc, TimePlayedSecs: 600},
	}
	profile := ComputeParticipationProfile(rows, "Player", "#blue")
	if profile.Values["kills"] != 9.0 {
		t.Errorf("avg kills attendu 9.0, got %.2f", profile.Values["kills"])
	}
	if profile.Values["kda"] != 2.0 {
		t.Errorf("avg kda attendu 2.0, got %.2f", profile.Values["kda"])
	}
}

// =============================================================================
// Tests ComputeSquadRecords
// =============================================================================

func TestComputeSquadRecords_Empty(t *testing.T) {
	records := ComputeSquadRecords(nil)
	for _, v := range records {
		if v != nil {
			t.Errorf("record doit Ãªtre nil pour donnÃ©es vides")
		}
	}
}

func TestComputeSquadRecords_MaxKills(t *testing.T) {
	rows := []domain.SquadMatchRow{
		{Kills: 5},
		{Kills: 12},
		{Kills: 3},
	}
	records := ComputeSquadRecords(rows)
	if records["kills"] == nil || *records["kills"] != 12.0 {
		t.Errorf("record max kills attendu 12.0, got %v", records["kills"])
	}
}

func TestComputeSquadRecords_MinDeaths(t *testing.T) {
	rows := []domain.SquadMatchRow{
		{Deaths: 8},
		{Deaths: 2},
		{Deaths: 5},
	}
	records := ComputeSquadRecords(rows)
	if records["deaths"] == nil || *records["deaths"] != 2.0 {
		t.Errorf("record min deaths attendu 2.0, got %v", records["deaths"])
	}
}

// =============================================================================
// Tests ComputeImpactSummary
// =============================================================================

func TestComputeImpactSummary_Empty(t *testing.T) {
	impact := ComputeImpactSummary(nil, "xuid_me", "xuid_tm")
	if impact.Available {
		t.Error("impact doit Ãªtre unavailable quand pas d'events")
	}
}

func TestComputeImpactSummary_FirstBlood(t *testing.T) {
	events := []domain.ImpactEventRow{
		{MatchID: "m1", XUID: "xuid_me", Gamertag: "Me", EventType: "kill", TimeMS: 1000},
		{MatchID: "m1", XUID: "xuid_tm", Gamertag: "Tm", EventType: "kill", TimeMS: 2000},
		{MatchID: "m2", XUID: "xuid_tm", Gamertag: "Tm", EventType: "kill", TimeMS: 500},
		{MatchID: "m2", XUID: "xuid_me", Gamertag: "Me", EventType: "kill", TimeMS: 1500},
	}
	impact := ComputeImpactSummary(events, "xuid_me", "xuid_tm")
	if !impact.Available {
		t.Error("impact doit Ãªtre available")
	}
	if impact.FirstBloods.Me != 1 {
		t.Errorf("first bloods me attendu 1, got %d", impact.FirstBloods.Me)
	}
	if impact.FirstBloods.Teammate != 1 {
		t.Errorf("first bloods teammate attendu 1, got %d", impact.FirstBloods.Teammate)
	}
}

// =============================================================================
// Tests ComputeSquadBreakdown
// =============================================================================

func TestComputeSquadBreakdown_Empty(t *testing.T) {
	stats := ComputeSquadBreakdown(nil)
	if stats.MatchCount != 0 {
		t.Errorf("match_count doit Ãªtre 0 pour donnÃ©es vides")
	}
}

func TestComputeSquadBreakdown_WinRate(t *testing.T) {
	kda1, kda2 := 1.5, 2.0
	rows := []domain.SquadMatchRow{
		{Outcome: 2, Kills: 10, KDA: &kda1},
		{Outcome: 2, Kills: 8, KDA: &kda2},
		{Outcome: 3, Kills: 5, KDA: &kda1},
		{Outcome: 3, Kills: 6, KDA: &kda2},
	}
	stats := ComputeSquadBreakdown(rows)
	if stats.WinRate != 50.0 {
		t.Errorf("win rate attendu 50.0, got %.1f", stats.WinRate)
	}
	if stats.MatchCount != 4 {
		t.Errorf("match count attendu 4, got %d", stats.MatchCount)
	}
}

// =============================================================================
// Tests ComputeSynthesisHeatmap
// =============================================================================

func TestComputeSynthesisHeatmap(t *testing.T) {
	rows := []domain.SynthesisHeatmapRow{
		{MapName: "Aquarius", ModeName: "Slayer", MatchCount: 10, Wins: 6},
		{MapName: "Aquarius", ModeName: "CTF", MatchCount: 5, Wins: 2},
	}
	cells := ComputeSynthesisHeatmap(rows)
	if len(cells) != 2 {
		t.Fatalf("attendu 2 cellules, got %d", len(cells))
	}
	if cells[0].Value != 60.0 {
		t.Errorf("win rate Aquarius/Slayer attendu 60.0, got %.1f", cells[0].Value)
	}
}

// =============================================================================
// Tests ComputeTopWeeks
// =============================================================================

func TestComputeTopWeeks_MinimumMatches(t *testing.T) {
	// 2 matchs dans une semaine â†’ ne doit pas apparaÃ®tre (min 3).
	t0 := time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC) // lundi
	rows := []domain.SquadMatchRow{
		{StartTime: t0, Outcome: 2, Kills: 10},
		{StartTime: t0.Add(24 * time.Hour), Outcome: 3, Kills: 5},
	}
	weeks := ComputeTopWeeks(rows)
	if len(weeks) != 0 {
		t.Errorf("semaine < 3 matchs ne doit pas apparaÃ®tre, got %d", len(weeks))
	}
}

func TestComputeTopWeeks_Sorting(t *testing.T) {
	// Semaine 1 : 4 matchs, 3 victoires (75%)
	// Semaine 2 : 4 matchs, 2 victoires (50%)
	t1 := time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC)  // lundi semaine 1
	t2 := time.Date(2025, 1, 13, 12, 0, 0, 0, time.UTC) // lundi semaine 2
	rows := []domain.SquadMatchRow{
		{StartTime: t1, Outcome: 2, Kills: 10}, {StartTime: t1.Add(time.Hour), Outcome: 2, Kills: 8},
		{StartTime: t1.Add(2 * time.Hour), Outcome: 2, Kills: 6}, {StartTime: t1.Add(3 * time.Hour), Outcome: 3, Kills: 4},
		{StartTime: t2, Outcome: 2, Kills: 9}, {StartTime: t2.Add(time.Hour), Outcome: 2, Kills: 7},
		{StartTime: t2.Add(2 * time.Hour), Outcome: 3, Kills: 5}, {StartTime: t2.Add(3 * time.Hour), Outcome: 3, Kills: 3},
	}
	weeks := ComputeTopWeeks(rows)
	if len(weeks) < 2 {
		t.Fatalf("attendu â‰¥ 2 semaines, got %d", len(weeks))
	}
	if weeks[0].WinRate < weeks[1].WinRate {
		t.Errorf("semaines mal triÃ©es : %v", weeks)
	}
}

// --- ComputeSynthesisBreakdown ---

func TestComputeSynthesisBreakdown_Empty(t *testing.T) {
	result := ComputeSynthesisBreakdown(nil, false)
	if result.MatchCount != 0 {
		t.Errorf("expected 0, got %d", result.MatchCount)
	}
}

func TestComputeSynthesisBreakdown_Solo(t *testing.T) {
	kda := 1.5
	rows := []legacymatch.SynthesisMatchRow{
		{Outcome: domain.OutcomeWin, Kills: 10, KDA: &kda, IsWithFriends: false},
		{Outcome: domain.OutcomeLoss, Kills: 5, KDA: &kda, IsWithFriends: false},
		{Outcome: domain.OutcomeWin, Kills: 8, KDA: &kda, IsWithFriends: true}, // excluded
	}
	result := ComputeSynthesisBreakdown(rows, false)
	if result.MatchCount != 2 {
		t.Errorf("MatchCount = %d, want 2", result.MatchCount)
	}
	if result.WinRate == 0 {
		t.Error("expected non-zero WinRate")
	}
}

func TestComputeSynthesisBreakdown_Squad(t *testing.T) {
	kda := 2.0
	rows := []legacymatch.SynthesisMatchRow{
		{Outcome: domain.OutcomeWin, Kills: 15, KDA: &kda, IsWithFriends: true},
		{Outcome: domain.OutcomeWin, Kills: 12, KDA: &kda, IsWithFriends: true},
	}
	result := ComputeSynthesisBreakdown(rows, true)
	if result.MatchCount != 2 {
		t.Errorf("MatchCount = %d, want 2", result.MatchCount)
	}
	if result.AvgKDA == 0 {
		t.Error("expected non-zero AvgKDA")
	}
}

// â”€â”€â”€ fmtPct â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func TestFmtPct(t *testing.T) {
	got := fmtPct(0.66666)
	if got != "66.7%" {
		t.Errorf("fmtPct(0.66666) = %q, want 66.7%%", got)
	}
}

func TestFmtPct_Zero(t *testing.T) {
	got := fmtPct(0)
	if got != "0.0%" {
		t.Errorf("fmtPct(0) = %q", got)
	}
}

// â”€â”€â”€ ComputeSynthesisHeatmap â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func TestComputeSynthesisHeatmap_Empty(t *testing.T) {
	result := ComputeSynthesisHeatmap(nil)
	if len(result) != 0 {
		t.Error("expected empty")
	}
}

func TestComputeSynthesisHeatmap_WithData(t *testing.T) {
	rows := []domain.SynthesisHeatmapRow{
		{MapName: "Aquarius", ModeName: "Slayer", Wins: 5, MatchCount: 10},
		{MapName: "Aquarius", ModeName: "CTF", Wins: 3, MatchCount: 8},
	}
	result := ComputeSynthesisHeatmap(rows)
	if len(result) != 2 {
		t.Errorf("expected 2 cells, got %d", len(result))
	}
}

// â”€â”€â”€ ComputeSynthesisKPIs â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func TestComputeSynthesisKPIs_Empty(t *testing.T) {
	kpis := ComputeSynthesisKPIs(nil, false)
	if kpis.MatchCount != 0 {
		t.Errorf("MatchCount = %d, want 0", kpis.MatchCount)
	}
}

func TestComputeSynthesisKPIs_WithData(t *testing.T) {
	kda1 := 2.0
	acc1 := 0.5
	perf := 1500.0
	dur := 600
	rows := []legacymatch.SynthesisMatchRow{
		{Kills: 10, Deaths: 5, KDA: &kda1, Accuracy: &acc1, PerformanceScore: &perf, TimePlayedSecs: &dur, Outcome: domain.OutcomeWin, IsWithFriends: true},
		{Kills: 8, Deaths: 8, KDA: &kda1, Accuracy: &acc1, PerformanceScore: &perf, TimePlayedSecs: &dur, Outcome: domain.OutcomeLoss, IsWithFriends: true},
	}
	kpis := ComputeSynthesisKPIs(rows, true)
	if kpis.MatchCount != 2 {
		t.Errorf("MatchCount = %d, want 2", kpis.MatchCount)
	}
	if kpis.Wins != 1 {
		t.Errorf("Wins = %d, want 1", kpis.Wins)
	}
}

// â”€â”€â”€ ComputeComparisonMetrics â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func TestComputeComparisonMetrics_ZeroKPIs(t *testing.T) {
	solo := domain.SynthesisKPIs{}
	squad := domain.SynthesisKPIs{}
	metrics := ComputeComparisonMetrics(solo, squad)
	if len(metrics) == 0 {
		t.Error("expected metrics even for zero KPIs")
	}
}

// â”€â”€â”€ ComputeTemporalHeatmap â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func TestComputeTemporalHeatmap_Empty(t *testing.T) {
	result := ComputeTemporalHeatmap(nil)
	if len(result) != 0 {
		t.Error("expected empty")
	}
}

func TestComputeTemporalHeatmap_WithData(t *testing.T) {
	// Monday 14:00
	monday := time.Date(2024, 3, 4, 14, 0, 0, 0, time.UTC)
	rows := []legacymatch.SynthesisMatchRow{
		{StartTime: monday, Outcome: domain.OutcomeWin},
		{StartTime: monday.Add(time.Hour), Outcome: domain.OutcomeLoss},
	}
	result := ComputeTemporalHeatmap(rows)
	if len(result) == 0 {
		t.Error("expected cells")
	}
}

// â”€â”€â”€ ComputeSquadBreakdown (additional) â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func TestComputeSquadBreakdown_WithMixedOutcomes(t *testing.T) {
	kda := 2.0
	rows := []domain.SquadMatchRow{
		{Kills: 10, Deaths: 5, Outcome: domain.OutcomeWin, KDA: &kda, IsWithFriends: true},
		{Kills: 5, Deaths: 10, Outcome: domain.OutcomeLoss, KDA: &kda, IsWithFriends: true},
		{Kills: 8, Deaths: 8, Outcome: domain.OutcomeDraw, KDA: &kda, IsWithFriends: true},
	}
	result := ComputeSquadBreakdown(rows)
	if result.MatchCount != 3 {
		t.Errorf("MatchCount = %d, want 3", result.MatchCount)
	}
}

// â”€â”€â”€ squad_profiles.go â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func TestComputeParticipationProfile_WithData(t *testing.T) {
	kda := 2.5
	acc := 0.45
	rows := []domain.SquadMatchRow{
		{Kills: 15, Deaths: 5, Assists: 3, KDA: &kda, Accuracy: &acc, TimePlayedSecs: 600},
	}
	profile := ComputeParticipationProfile(rows, "Player1", "#ff0000")
	if len(profile.Values) == 0 {
		t.Error("expected values")
	}
}

func TestComputeSquadRecords_WithData(t *testing.T) {
	kda := 5.0
	acc := 0.8
	rows := []domain.SquadMatchRow{
		{Kills: 20, Deaths: 2, Assists: 5, KDA: &kda, Accuracy: &acc, TimePlayedSecs: 600},
		{Kills: 5, Deaths: 15, Assists: 1, KDA: &kda, Accuracy: &acc, TimePlayedSecs: 300},
	}
	records := ComputeSquadRecords(rows)
	if len(records) == 0 {
		t.Error("expected records")
	}
}

func TestComputeTeammateProfile_Empty(t *testing.T) {
	profile := ComputeTeammateProfile(nil, "Teammate", "#00ff00")
	if profile.Name != "Teammate" {
		t.Errorf("Name = %q", profile.Name)
	}
}

func TestComputeTeammateRecords_Empty(t *testing.T) {
	records := ComputeTeammateRecords(nil)
	// Returns default keys even for nil input
	if records == nil {
		t.Error("expected non-nil")
	}
}
