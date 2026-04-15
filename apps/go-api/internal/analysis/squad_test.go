// Package analysis — squad_test.go : tests unitaires pour les algorithmes escouade.
package analysis

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// =============================================================================
// Tests ComputeSquadPerformanceScore
// =============================================================================

func TestComputeSquadPerformanceScore_NoScores(t *testing.T) {
	result := ComputeSquadPerformanceScore(nil, nil, nil, nil)
	if result.Score != nil {
		t.Errorf("score doit être nil quand pas de données, got %v", result.Score)
	}
	if result.Grade != "N/A" {
		t.Errorf("grade doit être N/A, got %s", result.Grade)
	}
}

func TestComputeSquadPerformanceScore_Basic(t *testing.T) {
	s1, s2 := 70.0, 80.0
	result := ComputeSquadPerformanceScore(
		[]*float64{&s1, &s2},
		[]float64{65.0, 70.0}, // win rate > 60 → +5
		[]float64{1.5, 2.0},   // min KDA > 1 → +5
		[]float64{8.0, 9.0},   // std < 3 → +3
	)
	if result.Score == nil {
		t.Fatal("score ne doit pas être nil")
	}
	// base = 75, bonus = 13 → 88 (clamp 100)
	expected := 88.0
	if *result.Score != expected {
		t.Errorf("score attendu %.1f, got %.1f", expected, *result.Score)
	}
}

func TestComputeSquadPerformanceScore_NoBonus(t *testing.T) {
	s1, s2 := 50.0, 60.0
	result := ComputeSquadPerformanceScore(
		[]*float64{&s1, &s2},
		[]float64{40.0},        // win rate ≤ 60 → pas de bonus
		[]float64{0.8},         // min KDA ≤ 1 → pas de bonus
		[]float64{5.0, 15.0},   // std > 3 → pas de bonus
	)
	if result.Score == nil {
		t.Fatal("score ne doit pas être nil")
	}
	if *result.Score != 55.0 {
		t.Errorf("score attendu 55.0, got %.1f", *result.Score)
	}
}

func TestComputeSquadPerformanceScore_Clamp(t *testing.T) {
	s1, s2 := 95.0, 98.0
	result := ComputeSquadPerformanceScore(
		[]*float64{&s1, &s2},
		[]float64{65.0},  // +5
		[]float64{2.0},   // +5
		[]float64{3.0, 4.0}, // std > 3 → pas de bonus
	)
	if result.Score == nil {
		t.Fatal("score ne doit pas être nil")
	}
	if *result.Score > 100.0 {
		t.Errorf("score ne doit pas dépasser 100, got %.1f", *result.Score)
	}
}

// =============================================================================
// Tests resolveSquadGrade
// =============================================================================

func TestResolveSquadGrade(t *testing.T) {
	cases := []struct{ score float64; grade string }{
		{95, "S"}, {85, "A"}, {70, "B"}, {55, "C"}, {40, "D"}, {20, "F"},
	}
	for _, tc := range cases {
		g := resolveSquadGrade(tc.score)
		if g != tc.grade {
			t.Errorf("score %.0f → grade attendu %s, got %s", tc.score, tc.grade, g)
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
			t.Errorf("record doit être nil pour données vides")
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
		t.Error("impact doit être unavailable quand pas d'events")
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
		t.Error("impact doit être available")
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
		t.Errorf("match_count doit être 0 pour données vides")
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
	// 2 matchs dans une semaine → ne doit pas apparaître (min 3).
	t0 := time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC) // lundi
	rows := []domain.SquadMatchRow{
		{StartTime: t0, Outcome: 2, Kills: 10},
		{StartTime: t0.Add(24 * time.Hour), Outcome: 3, Kills: 5},
	}
	weeks := ComputeTopWeeks(rows)
	if len(weeks) != 0 {
		t.Errorf("semaine < 3 matchs ne doit pas apparaître, got %d", len(weeks))
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
		t.Fatalf("attendu ≥ 2 semaines, got %d", len(weeks))
	}
	if weeks[0].WinRate < weeks[1].WinRate {
		t.Errorf("semaines mal triées : %v", weeks)
	}
}
