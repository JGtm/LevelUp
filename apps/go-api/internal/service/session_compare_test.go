package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

func ptr(s string) *string { return &s }
func ptrInt(i int) *int    { return &i }

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
