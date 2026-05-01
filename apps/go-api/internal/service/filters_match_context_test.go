// filters_match_context_test.go — tests Phase I plan catalogue : application
// du filtre match_context dans ResolveFiltersFromRows.
package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// helpers locaux pour fabriquer des rows.
func mcRow(matchID string, withFriends bool) domain.FilterMatchRow {
	t := time.Date(2025, 5, 1, 12, 0, 0, 0, time.UTC)
	return domain.FilterMatchRow{
		MatchID:       matchID,
		StartTime:     &t,
		IsWithFriends: withFriends,
	}
}

func TestApplyMatchContextFilter_Solo(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mcRow("m1", false),
		mcRow("m2", true),
		mcRow("m3", false),
		mcRow("m4", true),
	}
	got := applyMatchContextFilter(rows, domain.MatchContextSolo)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (solo only)", len(got))
	}
	for _, r := range got {
		if r.IsWithFriends {
			t.Errorf("row %q is_with_friends=true ne devrait pas être dans solo", r.MatchID)
		}
	}
}

func TestApplyMatchContextFilter_Squad(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mcRow("m1", false),
		mcRow("m2", true),
		mcRow("m3", false),
		mcRow("m4", true),
	}
	got := applyMatchContextFilter(rows, domain.MatchContextSquad)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (squad only)", len(got))
	}
	for _, r := range got {
		if !r.IsWithFriends {
			t.Errorf("row %q is_with_friends=false ne devrait pas être dans squad", r.MatchID)
		}
	}
}

func TestApplyMatchContextFilter_All(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mcRow("m1", false),
		mcRow("m2", true),
	}
	got := applyMatchContextFilter(rows, domain.MatchContextAll)
	if len(got) != 2 {
		t.Errorf("all : len = %d, want 2 (no filter)", len(got))
	}
}

func TestApplyMatchContextFilter_Empty(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mcRow("m1", false),
		mcRow("m2", true),
	}
	got := applyMatchContextFilter(rows, "")
	if len(got) != 2 {
		t.Errorf("empty : len = %d, want 2 (no filter par défaut)", len(got))
	}
}

// Test d'intégration : le filtre est bien appliqué à travers
// ResolveFiltersFromRows et impacte les counts.
func TestResolveFiltersFromRows_AppliesMatchContext(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mcRow("m1", false), // solo
		mcRow("m2", true),  // squad
		mcRow("m3", false), // solo
		mcRow("m4", true),  // squad
		mcRow("m5", true),  // squad
	}

	// Squad : 3 matchs.
	res := ResolveFiltersFromRows(rows, domain.FilterContextInput{MatchContext: domain.MatchContextSquad})
	if res.Counts.TotalMatchesBeforeFilters != 3 {
		t.Errorf("squad TotalBefore = %d, want 3", res.Counts.TotalMatchesBeforeFilters)
	}

	// Solo : 2 matchs.
	res = ResolveFiltersFromRows(rows, domain.FilterContextInput{MatchContext: domain.MatchContextSolo})
	if res.Counts.TotalMatchesBeforeFilters != 2 {
		t.Errorf("solo TotalBefore = %d, want 2", res.Counts.TotalMatchesBeforeFilters)
	}

	// All : 5 matchs.
	res = ResolveFiltersFromRows(rows, domain.FilterContextInput{MatchContext: domain.MatchContextAll})
	if res.Counts.TotalMatchesBeforeFilters != 5 {
		t.Errorf("all TotalBefore = %d, want 5", res.Counts.TotalMatchesBeforeFilters)
	}

	// Vide (default) : 5 matchs (pas de filtre par défaut).
	res = ResolveFiltersFromRows(rows, domain.FilterContextInput{})
	if res.Counts.TotalMatchesBeforeFilters != 5 {
		t.Errorf("empty TotalBefore = %d, want 5", res.Counts.TotalMatchesBeforeFilters)
	}
}
