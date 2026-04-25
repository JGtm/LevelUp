package service

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

func ptr(s string) *string { return &s }

func makeMatch(label string, kills, deaths int, outcome *int) domain.StatsMatchRow {
	return domain.StatsMatchRow{
		SessionLabel: &label,
		Kills:        kills,
		Deaths:       deaths,
		Outcome:      outcome,
		StartTime:    time.Now(),
	}
}

func TestExtractSessionLabels(t *testing.T) {
	matches := []domain.StatsMatchRow{
		makeMatch("S1", 10, 5, nil),
		makeMatch("S2", 8, 6, nil),
		makeMatch("S1", 12, 4, nil),
	}
	labels := extractSessionLabels(matches)
	if len(labels) != 2 {
		t.Fatalf("expected 2, got %d", len(labels))
	}
}

func TestExtractSessionLabels_NoLabels(t *testing.T) {
	labels := extractSessionLabels(nil)
	if len(labels) != 0 {
		t.Fatalf("expected 0, got %d", len(labels))
	}
}

func TestLastOrNil(t *testing.T) {
	labels := []string{"S1", "S2", "S3"}
	if got := lastOrNil(labels, nil); got != "S3" {
		t.Fatalf("expected S3, got %s", got)
	}
	if got := lastOrNil(labels, ptr("override")); got != "override" {
		t.Fatalf("expected override, got %s", got)
	}
	if got := lastOrNil(nil, nil); got != "" {
		t.Fatalf("expected empty, got %s", got)
	}
}

func TestSecondLastOrNil(t *testing.T) {
	labels := []string{"S1", "S2", "S3"}
	if got := secondLastOrNil(labels, nil); got != "S2" {
		t.Fatalf("expected S2, got %s", got)
	}
	if got := secondLastOrNil(labels, ptr("override")); got != "override" {
		t.Fatalf("expected override, got %s", got)
	}
	if got := secondLastOrNil([]string{"S1"}, nil); got != "" {
		t.Fatalf("expected empty, got %s", got)
	}
}

func TestFilterBySession(t *testing.T) {
	matches := []domain.StatsMatchRow{
		makeMatch("S1", 10, 5, nil),
		makeMatch("S2", 8, 6, nil),
		makeMatch("S1", 12, 4, nil),
	}
	filtered := filterBySession(matches, "S1")
	if len(filtered) != 2 {
		t.Fatalf("expected 2, got %d", len(filtered))
	}
}

func TestFilterBySession_NoLabel(t *testing.T) {
	filtered := filterBySession(nil, "S1")
	if filtered != nil {
		t.Fatal("expected nil")
	}
}

func TestBuildCompareEntry_Nil(t *testing.T) {
	entry := buildCompareEntry(nil, "S1")
	if entry != nil {
		t.Fatal("expected nil for empty matches")
	}
}

func TestBuildCompareEntry_WithMatches(t *testing.T) {
	win := analysis.OutcomeWin
	loss := analysis.OutcomeLoss
	matches := []domain.StatsMatchRow{
		makeMatch("S1", 15, 5, &win),
		makeMatch("S1", 10, 8, &loss),
		makeMatch("S1", 20, 3, &win),
	}
	entry := buildCompareEntry(matches, "S1")
	if entry == nil {
		t.Fatal("expected non-nil")
	}
	if entry.TotalMatches != 3 {
		t.Fatalf("expected 3, got %d", entry.TotalMatches)
	}
	if entry.Wins != 2 {
		t.Fatalf("expected 2 wins, got %d", entry.Wins)
	}
}

func TestWinRate(t *testing.T) {
	win := analysis.OutcomeWin
	loss := analysis.OutcomeLoss
	matches := []domain.StatsMatchRow{
		makeMatch("S1", 10, 5, &win),
		makeMatch("S1", 8, 6, &loss),
	}
	rate := winRate(matches)
	if rate != 50 {
		t.Fatalf("expected 50, got %f", rate)
	}
}

func TestAvgKD(t *testing.T) {
	matches := []domain.StatsMatchRow{
		makeMatch("S1", 10, 5, nil),
		makeMatch("S1", 20, 10, nil),
	}
	kd := avgKD(matches)
	if kd != 2.0 {
		t.Fatalf("expected 2.0, got %f", kd)
	}
}

func TestBuildCompareMetrics_TwoSessions(t *testing.T) {
	win := analysis.OutcomeWin
	a := []domain.StatsMatchRow{makeMatch("S1", 15, 5, &win)}
	b := []domain.StatsMatchRow{makeMatch("S2", 10, 10, nil)}
	metrics := buildCompareMetrics(a, b)
	if len(metrics) < 4 {
		t.Fatalf("expected >=4 metrics, got %d", len(metrics))
	}
}

func TestSessionCompareService_Compare_AutoSelectsLatestSessions(t *testing.T) {
	repo := &mockSessionPageStatsRepo{matches: makeSessionPageDataset()}
	svc := NewSessionCompareService(nil, repo)

	resp, err := svc.Compare(context.Background(), domain.SessionCompareRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SessionA == nil || resp.SessionA.SessionLabel != "2026-04-21 19h30" {
		t.Fatalf("unexpected session A: %#v", resp.SessionA)
	}
	if resp.SessionB == nil || resp.SessionB.SessionLabel != "2026-04-21 18h" {
		t.Fatalf("unexpected session B: %#v", resp.SessionB)
	}
	if len(resp.Metrics) == 0 {
		t.Fatal("expected comparison metrics")
	}
	assertSessionMetricPresent(t, resp.Metrics, "score")
}

func TestSessionCompareService_Compare_WithFilterAndSingleSession(t *testing.T) {
	repo := &mockSessionPageStatsRepo{matches: makeSessionPageDataset()}
	svc := NewSessionCompareService(nil, repo)
	start := time.Date(2026, 4, 21, 19, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 21, 21, 0, 0, 0, time.UTC)

	resp, err := svc.Compare(context.Background(), domain.SessionCompareRequest{
		Filters: domain.FilterContextInput{
			FilterMode: "period",
			Period: domain.PeriodInput{
				StartDate: &start,
				EndDate:   &end,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.AvailableSessions) != 1 {
		t.Fatalf("expected one filtered session, got %v", resp.AvailableSessions)
	}
	if resp.SessionA != nil || resp.SessionB != nil {
		t.Fatalf("expected no compare entries when fewer than two sessions remain, got %#v %#v", resp.SessionA, resp.SessionB)
	}
	if len(resp.Metrics) != 0 {
		t.Fatalf("expected no metrics, got %d", len(resp.Metrics))
	}
}

func TestEffectiveKDA(t *testing.T) {
	precomputed := 1.75
	if got := effectiveKDA(domain.StatsMatchRow{KDA: &precomputed}); got == nil || *got != precomputed {
		t.Fatalf("expected precomputed KDA, got %#v", got)
	}
	if got := effectiveKDA(domain.StatsMatchRow{Kills: 9, Deaths: 0}); got == nil || *got != 9 {
		t.Fatalf("expected kills fallback, got %#v", got)
	}
	if got := effectiveKDA(domain.StatsMatchRow{Kills: 9, Deaths: 4}); got == nil || *got != 2.25 {
		t.Fatalf("expected computed fallback, got %#v", got)
	}
}

func TestClassifySessionCategory(t *testing.T) {
	tests := []struct {
		name  string
		match domain.StatsMatchRow
		want  string
	}{
		{name: "firefight", match: domain.StatsMatchRow{PlaylistName: "Firefight Normal"}, want: "Firefight"},
		{name: "ranked", match: domain.StatsMatchRow{IsRanked: true, PlaylistName: "Arena"}, want: "Ranked"},
		{name: "btb", match: domain.StatsMatchRow{PlaylistName: "Big Team Battle"}, want: "BTB"},
		{name: "arena", match: domain.StatsMatchRow{PlaylistName: "Quick Play"}, want: "Arena"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifySessionCategory(tt.match); got != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, got)
			}
		})
	}
}
