// Package service — synthesis_service_test.go : tests pour SynthesisService.
// Sprint 55 D9.
package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

var errSynthTest = errors.New("synthesis repo error")

// --- mock SynthesisRepository ---

type mockSynthesisRepo struct {
	synthRows     []domain.SynthesisMatchRow
	synthErr      error
	heatmapRows   []domain.SynthesisHeatmapRow
	heatmapErr    error
	encounterRows []domain.EncounterRawRow
	encounterErr  error
}

func (m *mockSynthesisRepo) LoadSynthesisMatches(_ context.Context, _ string) ([]domain.SynthesisMatchRow, error) {
	return m.synthRows, m.synthErr
}

func (m *mockSynthesisRepo) LoadSynthesisHeatmap(_ context.Context, _ string) ([]domain.SynthesisHeatmapRow, error) {
	return m.heatmapRows, m.heatmapErr
}

func (m *mockSynthesisRepo) LoadEncounters(_ context.Context, _ string) ([]domain.EncounterRawRow, error) {
	return m.encounterRows, m.encounterErr
}

// --- helpers ---

func ptrFloat(v float64) *float64 { return &v }

func makeSynthRows(n int) []domain.SynthesisMatchRow {
	rows := make([]domain.SynthesisMatchRow, n)
	for i := range rows {
		kills := 10 - i%10
		deaths := 3 + i%5
		kda := float64(kills) / float64(deaths+1)
		rows[i] = domain.SynthesisMatchRow{
			MatchID:   fmt.Sprintf("match-%d", i),
			StartTime: time.Now().UTC().Add(-time.Duration(i) * time.Hour),
			Outcome:   2 + i%2, // alternance WIN/LOSS
			Kills:     kills,
			Deaths:    deaths,
			KDA:       &kda,
		}
	}
	return rows
}

// --- filterSynthesisByPeriod ---

func TestFilterSynthesisByPeriod_All(t *testing.T) {
	rows := makeSynthRows(10)
	got, applied, ignored := filterSynthesisByPeriod(rows, "all", domain.FilterContextInput{})
	if len(got) != 10 {
		t.Fatalf("want 10 rows, got %d", len(got))
	}
	if len(applied) != 0 {
		t.Errorf("want 0 applied, got %v", applied)
	}
	if len(ignored) != 0 {
		t.Errorf("want 0 ignored, got %v", ignored)
	}
}

func TestFilterSynthesisByPeriod_1w(t *testing.T) {
	rows := []domain.SynthesisMatchRow{
		{MatchID: "recent", StartTime: time.Now().UTC().Add(-24 * time.Hour), Outcome: 2, Kills: 5},
		{MatchID: "old", StartTime: time.Now().UTC().Add(-30 * 24 * time.Hour), Outcome: 3, Kills: 3},
	}
	got, applied, _ := filterSynthesisByPeriod(rows, "1w", domain.FilterContextInput{})
	if len(got) != 1 {
		t.Fatalf("want 1 row in 1w period, got %d", len(got))
	}
	if got[0].MatchID != "recent" {
		t.Errorf("want recent match, got %s", got[0].MatchID)
	}
	if len(applied) == 0 {
		t.Error("want applied filter for 1w")
	}
}

func TestFilterSynthesisByPeriod_Unknown(t *testing.T) {
	rows := makeSynthRows(5)
	got, _, _ := filterSynthesisByPeriod(rows, "unknown_period", domain.FilterContextInput{})
	if len(got) != 5 {
		t.Fatalf("unknown period should return all rows, got %d", len(got))
	}
}

// --- buildHighlightsPreview ---

func TestBuildHighlightsPreview_Empty(t *testing.T) {
	h := buildHighlightsPreview(nil)
	if len(h.TopByKills) != 0 || len(h.TopByKDA) != 0 || len(h.WorstByDeaths) != 0 {
		t.Error("empty rows should return empty highlights")
	}
}

func TestBuildHighlightsPreview_TopByKills(t *testing.T) {
	rows := []domain.SynthesisMatchRow{
		{MatchID: "a", Kills: 5},
		{MatchID: "b", Kills: 12},
		{MatchID: "c", Kills: 8},
	}
	h := buildHighlightsPreview(rows)
	if len(h.TopByKills) == 0 {
		t.Fatal("want top by kills, got empty")
	}
	if h.TopByKills[0].Kills != 12 {
		t.Errorf("top kill should be 12, got %d", h.TopByKills[0].Kills)
	}
}

func TestBuildHighlightsPreview_WorstByDeaths(t *testing.T) {
	rows := []domain.SynthesisMatchRow{
		{MatchID: "a", Deaths: 2},
		{MatchID: "b", Deaths: 15},
		{MatchID: "c", Deaths: 7},
	}
	h := buildHighlightsPreview(rows)
	if len(h.WorstByDeaths) == 0 {
		t.Fatal("want worst by deaths, got empty")
	}
	if h.WorstByDeaths[0].Deaths != 15 {
		t.Errorf("worst deaths should be 15, got %d", h.WorstByDeaths[0].Deaths)
	}
}

func TestBuildHighlightsPreview_LimitTopN(t *testing.T) {
	rows := makeSynthRows(20)
	h := buildHighlightsPreview(rows)
	if len(h.TopByKills) > highlightTopN {
		t.Errorf("top by kills capped at %d, got %d", highlightTopN, len(h.TopByKills))
	}
	if len(h.WorstByDeaths) > highlightTopN {
		t.Errorf("worst by deaths capped at %d, got %d", highlightTopN, len(h.WorstByDeaths))
	}
}

// --- buildRivalriesPreview ---

func TestBuildRivalriesPreview_Empty(t *testing.T) {
	r := buildRivalriesPreview(nil)
	if r.Total != 0 {
		t.Errorf("empty encounters should have total=0, got %d", r.Total)
	}
}

func TestBuildRivalriesPreview_SplitTeamEnemy(t *testing.T) {
	avk := 1.5
	rows := []domain.EncounterRawRow{
		{XUID: "x1", Gamertag: "PlayerA", MatchCount: 10, AsTeammate: 8, AsEnemy: 2, AvgKDA: &avk},
		{XUID: "x2", Gamertag: "PlayerB", MatchCount: 6, AsTeammate: 1, AsEnemy: 5, AvgKDA: &avk},
	}
	r := buildRivalriesPreview(rows)
	if len(r.TopTeammates) != 1 {
		t.Errorf("want 1 teammate, got %d", len(r.TopTeammates))
	}
	if len(r.TopEnemies) != 1 {
		t.Errorf("want 1 enemy, got %d", len(r.TopEnemies))
	}
	if r.Total != 2 {
		t.Errorf("total should be 2, got %d", r.Total)
	}
}

// --- buildBreakdowns ---

func TestBuildBreakdowns_Empty(t *testing.T) {
	b := buildBreakdowns(nil)
	if len(b.TopMaps) != 0 || len(b.TopModes) != 0 {
		t.Error("empty heatmap should return empty breakdowns")
	}
}

func TestBuildBreakdowns_Aggregates(t *testing.T) {
	rows := []domain.SynthesisHeatmapRow{
		{MapName: "Aquarius", ModeName: "Slayer", MatchCount: 5, Wins: 3},
		{MapName: "Aquarius", ModeName: "Oddball", MatchCount: 2, Wins: 1},
		{MapName: "Bazaar", ModeName: "Slayer", MatchCount: 3, Wins: 2},
	}
	b := buildBreakdowns(rows)

	// Aquarius doit avoir 7 matchs (5+2)
	var aquarius *domain.SynthesisMapEntry
	for i := range b.TopMaps {
		if b.TopMaps[i].MapName == "Aquarius" {
			aquarius = &b.TopMaps[i]
			break
		}
	}
	if aquarius == nil {
		t.Fatal("Aquarius not found in breakdowns")
	}
	if aquarius.MatchCount != 7 {
		t.Errorf("Aquarius match count should be 7, got %d", aquarius.MatchCount)
	}
	if aquarius.Wins != 4 {
		t.Errorf("Aquarius wins should be 4, got %d", aquarius.Wins)
	}
}

// --- GetSynthesisPage ---

func TestGetSynthesisPage_Success(t *testing.T) {
	kda := 2.0
	repo := &mockSynthesisRepo{
		synthRows: []domain.SynthesisMatchRow{
			{MatchID: "m1", StartTime: time.Now().UTC(), Outcome: 2, Kills: 10, Deaths: 3, KDA: &kda},
			{MatchID: "m2", StartTime: time.Now().UTC().Add(-time.Hour), Outcome: 3, Kills: 4, Deaths: 8, KDA: &kda},
		},
		heatmapRows: []domain.SynthesisHeatmapRow{
			{MapName: "Aquarius", ModeName: "Slayer", MatchCount: 2, Wins: 1},
		},
		encounterRows: []domain.EncounterRawRow{},
	}

	svc := NewSynthesisService(repo)
	resp, err := svc.GetSynthesisPage(context.Background(), "xuid-test", domain.SynthesisRequest{Period: "all"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Scope.MatchCount != 2 {
		t.Errorf("scope match count = %d, want 2", resp.Scope.MatchCount)
	}
	if resp.Overview.TotalMatches != 2 {
		t.Errorf("overview total matches = %d, want 2", resp.Overview.TotalMatches)
	}
	if len(resp.Breakdowns.TopMaps) == 0 {
		t.Error("expected non-empty map breakdown")
	}
}

func TestGetSynthesisPage_DefaultPeriodAll(t *testing.T) {
	repo := &mockSynthesisRepo{synthRows: []domain.SynthesisMatchRow{}}
	svc := NewSynthesisService(repo)

	resp, err := svc.GetSynthesisPage(context.Background(), "xuid", domain.SynthesisRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Scope.Period != "all" {
		t.Errorf("default period should be 'all', got %q", resp.Scope.Period)
	}
}

func TestGetSynthesisPage_RepoError(t *testing.T) {
	repo := &mockSynthesisRepo{synthErr: errSynthTest}
	svc := NewSynthesisService(repo)

	_, err := svc.GetSynthesisPage(context.Background(), "xuid", domain.SynthesisRequest{})
	if err == nil {
		t.Error("expected error from repo, got nil")
	}
}
