// Package service_test — tests unitaires des fonctions pures.
package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// ---------------------------------------------------------------------------
// FiltersService — ResolveFiltersFromRows
// ---------------------------------------------------------------------------

func TestResolveFiltersFromRows_Empty(t *testing.T) {
	result := ResolveFiltersFromRows(nil, domain.FilterContextInput{})
	if result.Counts.TotalMatchesBeforeFilters != 0 {
		t.Errorf("expected 0 total, got %d", result.Counts.TotalMatchesBeforeFilters)
	}
}

func TestResolveFiltersFromRows_AllRows(t *testing.T) {
	mapName := "Aquarius"
	pair := "Slayer"
	rows := []domain.FilterMatchRow{
		{MatchID: "m1", MapName: &mapName, PairName: &pair, IsFirefight: false, IsRanked: false},
		{MatchID: "m2", MapName: &mapName, PairName: &pair, IsFirefight: false, IsRanked: false},
	}
	result := ResolveFiltersFromRows(rows, domain.FilterContextInput{})
	if result.Counts.TotalMatchesBeforeFilters != 2 {
		t.Errorf("expected 2 total, got %d", result.Counts.TotalMatchesBeforeFilters)
	}
	if result.Counts.TotalMatchesAfterFilters != 2 {
		t.Errorf("expected 2 after filter, got %d", result.Counts.TotalMatchesAfterFilters)
	}
}

func TestResolveFiltersFromRows_ModeUI_StripsSuffix(t *testing.T) {
	pair := "Slayer on Aquarius"
	mapName := "Aquarius"
	rows := []domain.FilterMatchRow{
		{MatchID: "m1", PairName: &pair, MapName: &mapName},
	}
	result := ResolveFiltersFromRows(rows, domain.FilterContextInput{})
	if len(result.AvailableOptions.Modes) == 0 {
		t.Fatal("expected at least one mode")
	}
	if result.AvailableOptions.Modes[0].Label != "Slayer" {
		t.Errorf("expected stripped mode 'Slayer', got %q", result.AvailableOptions.Modes[0].Label)
	}
}

// ---------------------------------------------------------------------------
// MatchHistoryService — formatLifeSeconds
// ---------------------------------------------------------------------------

func TestFormatLifeSeconds_Zero(t *testing.T) {
	got := formatLifeSeconds(nil)
	if got != "0:00" {
		t.Errorf("expected 0:00, got %q", got)
	}
}

func TestFormatLifeSeconds_90Seconds(t *testing.T) {
	v := 90.0
	got := formatLifeSeconds(&v)
	if got != "1:30" {
		t.Errorf("expected 1:30, got %q", got)
	}
}

func TestFormatLifeSeconds_65Seconds(t *testing.T) {
	v := 65.5 // truncated to 65
	got := formatLifeSeconds(&v)
	if got != "1:05" {
		t.Errorf("expected 1:05, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// MatchHistoryService — computeMapWinRates
// ---------------------------------------------------------------------------

func TestComputeMapWinRates(t *testing.T) {
	map1 := "Aquarius"
	rows := []domain.MatchHistoryRawRow{
		{MapName: &map1, Outcome: 2},
		{MapName: &map1, Outcome: 2},
		{MapName: &map1, Outcome: 3},
	}
	rates := computeMapWinRates(rows)
	entry, ok := rates[map1]
	if !ok {
		t.Fatal("expected entry for Aquarius")
	}
	if entry[0] != 2 || entry[1] != 3 {
		t.Errorf("expected [2,3], got %v", entry)
	}
}

// ---------------------------------------------------------------------------
// MatchHistoryService — paginate
// ---------------------------------------------------------------------------

func TestPaginate_Page1(t *testing.T) {
	items := make([]domain.MatchHistoryRow, 30)
	for i := range items {
		items[i].MatchID = "m"
	}
	meta, page := paginate(items, domain.PaginationRequest{Page: 1, PageSize: 25})
	if meta.Total != 30 {
		t.Errorf("expected total=30, got %d", meta.Total)
	}
	if len(page) != 25 {
		t.Errorf("expected 25 items page 1, got %d", len(page))
	}
	if meta.HasNext != true {
		t.Error("expected HasNext=true")
	}
}

func TestPaginate_LastPage(t *testing.T) {
	items := make([]domain.MatchHistoryRow, 30)
	meta, page := paginate(items, domain.PaginationRequest{Page: 2, PageSize: 25})
	if len(page) != 5 {
		t.Errorf("expected 5 items page 2, got %d", len(page))
	}
	if meta.HasNext != false {
		t.Error("expected HasNext=false on last page")
	}
	if meta.HasPrev != true {
		t.Error("expected HasPrev=true on page 2")
	}
}

// ---------------------------------------------------------------------------
// CareerService — buildHeroProgress
// ---------------------------------------------------------------------------

func TestBuildHeroProgress_Half(t *testing.T) {
	p := buildHeroProgress(xpHeroTotal / 2)
	if p.Percentage != 50.0 {
		t.Errorf("expected 50%%, got %v", p.Percentage)
	}
	if p.XPRemaining != xpHeroTotal/2 {
		t.Errorf("expected remaining=%d, got %d", xpHeroTotal/2, p.XPRemaining)
	}
}

func TestBuildHeroProgress_MaxCap(t *testing.T) {
	p := buildHeroProgress(xpHeroTotal + 1000)
	if p.Percentage != 100.0 {
		t.Errorf("expected 100%% cap, got %v", p.Percentage)
	}
	if p.XPRemaining != 0 {
		t.Errorf("expected 0 remaining, got %d", p.XPRemaining)
	}
}

// ---------------------------------------------------------------------------
// CareerService — computeActiveXPPerDay
// ---------------------------------------------------------------------------

func TestComputeActiveXPPerDay_Simple(t *testing.T) {
	now := time.Now()
	history := []domain.XPHistoryPoint{
		{RecordedAt: now.Add(-10 * 24 * time.Hour), XPTotalCumul: 0},
		{RecordedAt: now, XPTotalCumul: 10000},
	}
	rate := computeActiveXPPerDay(history)
	// 10000 XP / 10 jours = 1000
	if rate < 900 || rate > 1100 {
		t.Errorf("expected ~1000 XP/day, got %v", rate)
	}
}

func TestComputeActiveXPPerDay_InsufficientHistory(t *testing.T) {
	history := []domain.XPHistoryPoint{{RecordedAt: time.Now(), XPTotalCumul: 5000}}
	rate := computeActiveXPPerDay(history)
	if rate != 0.0 {
		t.Errorf("expected 0 for single point, got %v", rate)
	}
}

// ---------------------------------------------------------------------------
// CareerService — computeProgressPct
// ---------------------------------------------------------------------------

func TestComputeProgressPct_Normal(t *testing.T) {
	pct := computeProgressPct(500, 2000, false)
	if pct != 25.0 {
		t.Errorf("expected 25.0%%, got %v", pct)
	}
}

func TestComputeProgressPct_MaxRank(t *testing.T) {
	pct := computeProgressPct(100, 500, true)
	if pct != 100.0 {
		t.Errorf("expected 100%% for max rank, got %v", pct)
	}
}
