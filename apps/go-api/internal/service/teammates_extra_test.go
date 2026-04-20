package service

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// ---------- computeKPIsFromSquadMatches ----------

func TestComputeKPIsFromSquadMatches_Empty_NoCount(t *testing.T) {
	got := computeKPIsFromSquadMatches(nil)
	if got.MatchCount != 0 {
		t.Errorf("expected 0 matches, got %d", got.MatchCount)
	}
}

func TestComputeKPIsFromSquadMatches_WithAccuracy(t *testing.T) {
	acc := 0.5
	matches := []domain.SquadMatchRow{
		{MatchID: "m1", Outcome: domain.OutcomeWin, Kills: 10, Deaths: 5, Assists: 3, Accuracy: &acc},
		{MatchID: "m2", Outcome: domain.OutcomeLoss, Kills: 6, Deaths: 8, Assists: 2, Accuracy: &acc},
	}
	got := computeKPIsFromSquadMatches(matches)
	if got.MatchCount != 2 {
		t.Errorf("expected 2, got %d", got.MatchCount)
	}
	if got.Wins != 1 {
		t.Errorf("expected 1 win, got %d", got.Wins)
	}
	if got.KDRatio == nil || *got.KDRatio < 1.0 {
		t.Errorf("expected KD > 1, got %v", got.KDRatio)
	}
	if got.Accuracy == nil {
		t.Error("expected accuracy non-nil")
	}
}

// ---------- computeKPIsFromSynthesisExcluding ----------

func TestComputeKPIsFromSynthesisExcluding_AllOut(t *testing.T) {
	matches := []domain.SynthesisMatchRow{
		{MatchID: "m1", Outcome: domain.OutcomeWin, Kills: 10, Deaths: 5},
	}
	exclude := map[string]bool{"m1": true}
	got := computeKPIsFromSynthesisExcluding(matches, exclude)
	if got.MatchCount != 0 {
		t.Errorf("expected 0, got %d", got.MatchCount)
	}
}

func TestComputeKPIsFromSynthesisExcluding_PartialFilter(t *testing.T) {
	matches := []domain.SynthesisMatchRow{
		{MatchID: "m1", Outcome: domain.OutcomeWin, Kills: 10, Deaths: 5},
		{MatchID: "m2", Outcome: domain.OutcomeLoss, Kills: 3, Deaths: 7},
	}
	exclude := map[string]bool{"m1": true}
	got := computeKPIsFromSynthesisExcluding(matches, exclude)
	if got.MatchCount != 1 {
		t.Errorf("expected 1, got %d", got.MatchCount)
	}
}

// ---------- computeSoloReference ----------

func TestComputeSoloReference_NoSoloMatches(t *testing.T) {
	matches := []domain.SynthesisMatchRow{
		{MatchID: "m1", IsWithFriends: true, Kills: 10, Deaths: 5},
	}
	got := computeSoloReference(matches)
	if got != nil {
		t.Error("expected nil when no solo matches")
	}
}

func TestComputeSoloReference_WithSoloFiltered(t *testing.T) {
	matches := []domain.SynthesisMatchRow{
		{MatchID: "m1", IsWithFriends: false, Outcome: domain.OutcomeWin, Kills: 10, Deaths: 5},
		{MatchID: "m2", IsWithFriends: true, Outcome: domain.OutcomeWin, Kills: 8, Deaths: 3},
	}
	got := computeSoloReference(matches)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if got.MatchCount != 1 {
		t.Errorf("expected 1, got %d", got.MatchCount)
	}
}

// ---------- safeDiv ----------

func TestSafeDiv_NormalDiv(t *testing.T) {
	if got := safeDiv(10, 5); got != 2.0 {
		t.Errorf("got %f", got)
	}
}

func TestSafeDiv_ZeroDenomReturnsZero(t *testing.T) {
	if got := safeDiv(10, 0); got != 10 {
		t.Errorf("expected 10 (returns a when b=0), got %f", got)
	}
}

// ---------- round2 ----------

func TestRound2_Precision(t *testing.T) {
	if got := round2(1.2345); got != 1.23 {
		t.Errorf("got %f", got)
	}
}

// ---------- buildTeammateOptions ----------

func TestBuildTeammateOptions(t *testing.T) {
	rows := []domain.TopTeammateRow{
		{Gamertag: "Player1", GamesTogether: 10},
		{Gamertag: "Player2", GamesTogether: 5},
	}
	opts := buildTeammateOptions(rows)
	if len(opts) != 2 {
		t.Fatalf("expected 2, got %d", len(opts))
	}
	if opts[0].Gamertag == "" {
		t.Error("expected non-empty gamertag")
	}
}

// ---------- computeKPIsFromSquadMatches — HeadshotKills / PerfectKills ----------

func TestComputeKPIsFromSquadMatches_HeadshotAndPerfectKills(t *testing.T) {
	acc := 0.6
	matches := []domain.SquadMatchRow{
		{MatchID: "m1", Outcome: domain.OutcomeWin, Kills: 10, Deaths: 3, Assists: 1, HeadshotKills: 4, PerfectKills: 2, Accuracy: &acc},
		{MatchID: "m2", Outcome: domain.OutcomeLoss, Kills: 6, Deaths: 7, Assists: 2, HeadshotKills: 2, PerfectKills: 1, Accuracy: &acc},
	}
	got := computeKPIsFromSquadMatches(matches)
	if got.HeadshotKillsPerGame == nil {
		t.Fatal("expected HeadshotKillsPerGame non-nil")
	}
	if *got.HeadshotKillsPerGame != 3.0 { // (4+2)/2
		t.Errorf("expected 3.0 headshot kills/game, got %f", *got.HeadshotKillsPerGame)
	}
	if got.PerfectKillsPerGame == nil {
		t.Fatal("expected PerfectKillsPerGame non-nil")
	}
	if *got.PerfectKillsPerGame != 1.5 { // (2+1)/2
		t.Errorf("expected 1.5 perfect kills/game, got %f", *got.PerfectKillsPerGame)
	}
}

func TestComputeKPIsFromSquadMatches_ZeroHeadshots(t *testing.T) {
	matches := []domain.SquadMatchRow{
		{MatchID: "m1", Outcome: domain.OutcomeWin, Kills: 5, Deaths: 2, HeadshotKills: 0, PerfectKills: 0},
	}
	got := computeKPIsFromSquadMatches(matches)
	if got.HeadshotKillsPerGame == nil || *got.HeadshotKillsPerGame != 0.0 {
		t.Errorf("expected 0.0 headshot kills/game, got %v", got.HeadshotKillsPerGame)
	}
}

// ---------- extractSynthesisSessionLabels ----------

func TestExtractSynthesisSessionLabels_SeparatesSoloSquad(t *testing.T) {
	s1 := "session-solo-1"
	s2 := "session-squad-1"
	matches := []domain.SynthesisMatchRow{
		{MatchID: "m1", IsWithFriends: false, SessionLabel: &s1},
		{MatchID: "m2", IsWithFriends: true, SessionLabel: &s2},
		{MatchID: "m3", IsWithFriends: false, SessionLabel: &s1}, // doublon → 1 seul
	}
	got := extractSynthesisSessionLabels(matches)
	if len(got.Solo) != 1 {
		t.Errorf("expected 1 solo session, got %d", len(got.Solo))
	}
	if len(got.Squad) != 1 {
		t.Errorf("expected 1 squad session, got %d", len(got.Squad))
	}
	if got.Solo[0] != s1 {
		t.Errorf("expected %s, got %s", s1, got.Solo[0])
	}
}

func TestExtractSynthesisSessionLabels_SkipsEmptyLabel(t *testing.T) {
	empty := ""
	matches := []domain.SynthesisMatchRow{
		{MatchID: "m1", IsWithFriends: false, SessionLabel: &empty},
		{MatchID: "m2", IsWithFriends: false, SessionLabel: nil},
	}
	got := extractSynthesisSessionLabels(matches)
	if len(got.Solo) != 0 || len(got.Squad) != 0 {
		t.Errorf("expected empty labels, got solo=%v squad=%v", got.Solo, got.Squad)
	}
}

// ---------- filterSynthesisBySession ----------

func TestFilterSynthesisBySession_NilFiltersReturnsAll(t *testing.T) {
	s := "s1"
	matches := []domain.SynthesisMatchRow{
		{MatchID: "m1", IsWithFriends: false, SessionLabel: &s},
		{MatchID: "m2", IsWithFriends: true, SessionLabel: &s},
	}
	got := filterSynthesisBySession(matches, nil, nil)
	if len(got) != 2 {
		t.Errorf("expected all 2, got %d", len(got))
	}
}

func TestFilterSynthesisBySession_SoloFilter(t *testing.T) {
	solo := "session-solo"
	squad := "session-squad"
	matches := []domain.SynthesisMatchRow{
		{MatchID: "m1", IsWithFriends: false, SessionLabel: &solo},
		{MatchID: "m2", IsWithFriends: true, SessionLabel: &squad},
		{MatchID: "m3", IsWithFriends: false, SessionLabel: &squad},
	}
	got := filterSynthesisBySession(matches, &solo, nil)
	if len(got) != 1 || got[0].MatchID != "m1" {
		t.Errorf("expected only m1, got %v", got)
	}
}

func TestFilterSynthesisBySession_SquadFilter(t *testing.T) {
	solo := "session-solo"
	squad := "session-squad"
	matches := []domain.SynthesisMatchRow{
		{MatchID: "m1", IsWithFriends: false, SessionLabel: &solo},
		{MatchID: "m2", IsWithFriends: true, SessionLabel: &squad},
		{MatchID: "m3", IsWithFriends: true, SessionLabel: &solo},
	}
	got := filterSynthesisBySession(matches, nil, &squad)
	if len(got) != 1 || got[0].MatchID != "m2" {
		t.Errorf("expected only m2, got %v", got)
	}
}

// ---------- computeMapBreakdown ----------

func TestComputeMapBreakdown_WinRateCalculation(t *testing.T) {
	matches := []domain.SquadMatchRow{
		{MatchID: "m1", MapUI: "Bazaar", Outcome: domain.OutcomeWin},
		{MatchID: "m2", MapUI: "Bazaar", Outcome: domain.OutcomeWin},
		{MatchID: "m3", MapUI: "Bazaar", Outcome: domain.OutcomeLoss},
		{MatchID: "m4", MapUI: "Recharge", Outcome: domain.OutcomeLoss},
	}
	rows := computeMapBreakdown(matches)
	if len(rows) != 2 {
		t.Fatalf("expected 2 maps, got %d", len(rows))
	}
	byMap := map[string]domain.MapBreakdownRow{}
	for _, r := range rows {
		byMap[r.MapUI] = r
	}
	bazaar := byMap["Bazaar"]
	if bazaar.MatchCount != 3 {
		t.Errorf("Bazaar: expected 3 matches, got %d", bazaar.MatchCount)
	}
	if bazaar.WinRate != 66.67 {
		t.Errorf("Bazaar: expected win rate 66.67, got %f", bazaar.WinRate)
	}
	recharge := byMap["Recharge"]
	if recharge.WinRate != 0.0 {
		t.Errorf("Recharge: expected 0 win rate, got %f", recharge.WinRate)
	}
}

func TestComputeMapBreakdown_EmptyMapUIFallback(t *testing.T) {
	matches := []domain.SquadMatchRow{
		{MatchID: "m1", MapUI: "", Outcome: domain.OutcomeWin},
	}
	rows := computeMapBreakdown(matches)
	if len(rows) != 1 || rows[0].MapUI != "Unknown" {
		t.Errorf("expected MapUI=Unknown, got %v", rows)
	}
}

// ---------- buildMatchSeries ----------

func TestBuildMatchSeries_FieldMapping(t *testing.T) {
	label := "session-1"
	perf := 78.5
	ts := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	matches := []domain.SquadMatchRow{
		{
			MatchID:          "match-abc",
			StartTime:        ts,
			Outcome:          domain.OutcomeWin,
			PerformanceScore: &perf,
			TeamMMR:          1500.0,
			SessionLabel:     &label,
		},
	}
	got := buildMatchSeries(matches)
	if len(got) != 1 {
		t.Fatalf("expected 1 point, got %d", len(got))
	}
	p := got[0]
	if p.MatchID != "match-abc" {
		t.Errorf("MatchID: got %s", p.MatchID)
	}
	if p.StartTime != "2025-01-15T10:30:00Z" {
		t.Errorf("StartTime: got %s", p.StartTime)
	}
	if p.Outcome != domain.OutcomeWin {
		t.Errorf("Outcome: got %d", p.Outcome)
	}
	if p.PerformanceScore == nil || *p.PerformanceScore != perf {
		t.Errorf("PerformanceScore: got %v", p.PerformanceScore)
	}
	if p.TeamMMRAvg != 1500.0 {
		t.Errorf("TeamMMRAvg: got %f", p.TeamMMRAvg)
	}
	if p.SessionLabel == nil || *p.SessionLabel != label {
		t.Errorf("SessionLabel: got %v", p.SessionLabel)
	}
}

func TestBuildMatchSeries_Empty(t *testing.T) {
	got := buildMatchSeries(nil)
	if len(got) != 0 {
		t.Errorf("expected empty, got %d", len(got))
	}
}

// ---------- GetPage avec SelectedGamertags (couvre buildTeammateRowWithMatches) ----------

func TestTeammatesService_GetPage_WithSelectedGamertag(t *testing.T) {
	repo := &mockSquadRepo{
		topRows: []domain.TopTeammateRow{
			{XUID: "xuid-ally1", Gamertag: "Ally1", GamesTogether: 15, WinsTogether: 9, WinRate: 0.6, AvgKDA: 1.2},
		},
		squadRows: []domain.SquadMatchRow{
			{MatchID: "m1", MapUI: "Bazaar", Outcome: domain.OutcomeWin, Kills: 10, Deaths: 4, Assists: 2, HeadshotKills: 3, PerfectKills: 1, StartTime: time.Now(), TeamMMR: 1400.0},
			{MatchID: "m2", MapUI: "Bazaar", Outcome: domain.OutcomeLoss, Kills: 6, Deaths: 8, Assists: 1, HeadshotKills: 1, PerfectKills: 0, StartTime: time.Now(), TeamMMR: 1350.0},
		},
	}
	svc := NewTeammatesService(repo)

	resp, err := svc.GetPage(context.Background(), "player-xuid", domain.TeammatesQueryRequest{
		SelectedGamertags: []string{"Ally1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Teammates) != 1 {
		t.Fatalf("expected 1 teammate, got %d", len(resp.Teammates))
	}
	tm := resp.Teammates[0]
	if tm.Gamertag != "Ally1" {
		t.Errorf("expected Ally1, got %s", tm.Gamertag)
	}
	if tm.WithKPIs.HeadshotKillsPerGame == nil || *tm.WithKPIs.HeadshotKillsPerGame != 2.0 {
		t.Errorf("expected headshot_kills_per_game=2.0, got %v", tm.WithKPIs.HeadshotKillsPerGame)
	}
	if tm.WithKPIs.PerfectKillsPerGame == nil || *tm.WithKPIs.PerfectKillsPerGame != 0.5 {
		t.Errorf("expected perfect_kills_per_game=0.5, got %v", tm.WithKPIs.PerfectKillsPerGame)
	}
	// MapBreakdown doit être calculé
	if len(resp.MapBreakdown) == 0 {
		t.Error("expected non-empty MapBreakdown")
	}
	// MatchSeries doit contenir l'entrée pour Ally1
	if _, ok := resp.MatchSeries["Ally1"]; !ok {
		t.Error("expected MatchSeries entry for Ally1")
	}
	if len(resp.MatchSeries["Ally1"]) != 2 {
		t.Errorf("expected 2 match series points, got %d", len(resp.MatchSeries["Ally1"]))
	}
}

func TestTeammatesService_GetPage_UnknownGamertag_Skipped(t *testing.T) {
	repo := &mockSquadRepo{
		topRows: []domain.TopTeammateRow{
			{XUID: "xuid-ally1", Gamertag: "Ally1", GamesTogether: 5},
		},
	}
	svc := NewTeammatesService(repo)

	resp, err := svc.GetPage(context.Background(), "player-xuid", domain.TeammatesQueryRequest{
		SelectedGamertags: []string{"Unknown"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Unknown gamertag non présent dans topRows → skippé
	if len(resp.Teammates) != 0 {
		t.Errorf("expected 0 teammates, got %d", len(resp.Teammates))
	}
}
