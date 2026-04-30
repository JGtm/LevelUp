package service

import (
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// ─── helpers ───────────────────────────────────────────────────────────────────

// makeTranslatedRows produces 10 FilterMatchRows with FR mode translations applied:
//   - 5 rows with PairName="Arena:Slayer", PairNameFR="Assassin"
//   - 5 rows with PairName="BTB:CTF",      PairNameFR="Capture de drapeau"
func makeTranslatedRows() []domain.FilterMatchRow {
	now := time.Now()
	rows := make([]domain.FilterMatchRow, 10)
	for i := range 5 {
		rows[i] = domain.FilterMatchRow{
			MatchID:      fmt.Sprintf("mt%d", i),
			StartTime:    &now,
			MapName:      strPtr("Aquarius"),
			MapNameFR:    strPtr("Aquarius"),
			PairName:     strPtr("Arena:Slayer"),
			PairNameFR:   strPtr("Assassin"),
			PlaylistName: strPtr("Ranked Arena"),
		}
	}
	for i := range 5 {
		rows[5+i] = domain.FilterMatchRow{
			MatchID:      fmt.Sprintf("mt%d", 5+i),
			StartTime:    &now,
			MapName:      strPtr("Streets"),
			MapNameFR:    strPtr("Streets"),
			PairName:     strPtr("BTB:CTF"),
			PairNameFR:   strPtr("Capture de drapeau"),
			PlaylistName: strPtr("Quick Play"),
		}
	}
	return rows
}

// makeUntranslatedRows produces rows where PairNameFR == PairName (COALESCE fallback, no DB translation).
func makeUntranslatedRows() []domain.FilterMatchRow {
	now := time.Now()
	rows := make([]domain.FilterMatchRow, 10)
	for i := range 5 {
		rows[i] = domain.FilterMatchRow{
			MatchID:      fmt.Sprintf("mu%d", i),
			StartTime:    &now,
			MapName:      strPtr("Aquarius"),
			PairName:     strPtr("Arena:Slayer"),
			PairNameFR:   strPtr("Arena:Slayer"),
			PlaylistName: strPtr("Ranked Arena"),
		}
	}
	for i := range 5 {
		rows[5+i] = domain.FilterMatchRow{
			MatchID:      fmt.Sprintf("mu%d", 5+i),
			StartTime:    &now,
			MapName:      strPtr("Streets"),
			PairName:     strPtr("BTB:CTF"),
			PairNameFR:   strPtr("BTB:CTF"),
			PlaylistName: strPtr("Quick Play"),
		}
	}
	return rows
}

// ─── buildModeTranslationMap ───────────────────────────────────────────────────

func TestBuildModeTranslationMap_NilRows(t *testing.T) {
	t.Parallel()
	if m := buildModeTranslationMap(nil); len(m) != 0 {
		t.Errorf("expected empty map for nil rows, got %d entries", len(m))
	}
}

func TestBuildModeTranslationMap_NilPairName(t *testing.T) {
	t.Parallel()
	rows := []domain.FilterMatchRow{{PairName: nil, PairNameFR: nil}}
	if m := buildModeTranslationMap(rows); len(m) != 0 {
		t.Errorf("expected empty map for nil fields, got %v", m)
	}
}

func TestBuildModeTranslationMap_SameNormalized(t *testing.T) {
	t.Parallel()
	rows := []domain.FilterMatchRow{
		{PairName: strPtr("Arena:Slayer"), PairNameFR: strPtr("Arena:Slayer")},
		{PairName: strPtr("Slayer"), PairNameFR: strPtr("Slayer")},
	}
	if m := buildModeTranslationMap(rows); len(m) != 0 {
		t.Errorf("no translation when EN==FR, but got: %v", m)
	}
}

func TestBuildModeTranslationMap_WithTranslations(t *testing.T) {
	t.Parallel()
	rows := makeTranslatedRows()
	m := buildModeTranslationMap(rows)
	cases := [][2]string{
		{"Slayer", "Assassin"},
		{"CTF", "Capture de drapeau"},
	}
	for _, c := range cases {
		if m[c[0]] != c[1] {
			t.Errorf("tr[%q] = %q, want %q", c[0], m[c[0]], c[1])
		}
	}
	if len(m) != 2 {
		t.Errorf("expected 2 entries, got %d: %v", len(m), m)
	}
}

// ─── modeUI ───────────────────────────────────────────────────────────────────

func TestModeUI_FrenchPairNameFR(t *testing.T) {
	t.Parallel()
	row := domain.FilterMatchRow{
		PairName:   strPtr("Arena:Slayer"),
		PairNameFR: strPtr("Assassin"),
	}
	if got := modeUI(row); got != "Assassin" {
		t.Errorf("modeUI = %q, want Assassin", got)
	}
}

func TestModeUI_EnglishFallback(t *testing.T) {
	t.Parallel()
	row := domain.FilterMatchRow{
		PairName:   strPtr("Arena:Slayer"),
		PairNameFR: strPtr("Arena:Slayer"),
	}
	if got := modeUI(row); got != "Slayer" {
		t.Errorf("modeUI = %q, want Slayer", got)
	}
}

// ─── ResolveFiltersFromRows — mode translations ────────────────────────────────

func TestResolveFilters_TranslatedModes_AvailableOptionsFR(t *testing.T) {
	t.Parallel()
	result := ResolveFiltersFromRows(makeTranslatedRows(), domain.FilterContextInput{})

	modes := result.AvailableOptions.Modes
	if len(modes) != 2 {
		t.Fatalf("expected 2 mode options, got %d: %v", len(modes), modes)
	}
	// sorted: "Assassin" < "Capture de drapeau"
	if modes[0].Label != "Assassin" || modes[0].Value != "Assassin" {
		t.Errorf("modes[0] = %+v, want {Assassin Assassin}", modes[0])
	}
	if modes[1].Label != "Capture de drapeau" || modes[1].Value != "Capture de drapeau" {
		t.Errorf("modes[1] = %+v, want {Capture de drapeau Capture de drapeau}", modes[1])
	}
}

func TestResolveFilters_TranslatedModes_CountIsCorrect(t *testing.T) {
	t.Parallel()
	result := ResolveFiltersFromRows(makeTranslatedRows(), domain.FilterContextInput{})

	if result.Counts.TotalMatchesBeforeFilters != 10 {
		t.Errorf("TotalMatchesBeforeFilters = %d, want 10", result.Counts.TotalMatchesBeforeFilters)
	}
	if result.Counts.TotalMatchesAfterFilters != 10 {
		t.Errorf("TotalMatchesAfterFilters = %d, want 10 (no filters active)", result.Counts.TotalMatchesAfterFilters)
	}
}

func TestResolveFilters_LegacyEnglishModeFilter_MigratedToFR(t *testing.T) {
	t.Parallel()
	// User has old English "Slayer" stored — must be transparently migrated to "Assassin"
	input := domain.FilterContextInput{
		Cascade: domain.CascadeFilter{Modes: []string{"Slayer"}},
	}
	result := ResolveFiltersFromRows(makeTranslatedRows(), input)

	if result.Counts.TotalMatchesAfterFilters != 5 {
		t.Errorf("TotalMatchesAfterFilters = %d, want 5 (Slayer→Assassin migration)", result.Counts.TotalMatchesAfterFilters)
	}
}

func TestResolveFilters_FrenchModeFilter_WorksDirectly(t *testing.T) {
	t.Parallel()
	input := domain.FilterContextInput{
		Cascade: domain.CascadeFilter{Modes: []string{"Assassin"}},
	}
	result := ResolveFiltersFromRows(makeTranslatedRows(), input)

	if result.Counts.TotalMatchesAfterFilters != 5 {
		t.Errorf("TotalMatchesAfterFilters = %d, want 5", result.Counts.TotalMatchesAfterFilters)
	}
}

func TestResolveFilters_NoTranslation_EnglishFilterStillWorks(t *testing.T) {
	t.Parallel()
	// Rows have no translation (PairNameFR == PairName, typical when DB has no pair_name_fr)
	input := domain.FilterContextInput{
		Cascade: domain.CascadeFilter{Modes: []string{"Slayer"}},
	}
	result := ResolveFiltersFromRows(makeUntranslatedRows(), input)

	if result.Counts.TotalMatchesAfterFilters != 5 {
		t.Errorf("TotalMatchesAfterFilters = %d, want 5 (EN filter without translations)", result.Counts.TotalMatchesAfterFilters)
	}
}

func TestResolveFilters_MultipleLegacyModes_AllMigrated(t *testing.T) {
	t.Parallel()
	// Both stored as English — both should be migrated and filtering should give all 10 rows
	input := domain.FilterContextInput{
		Cascade: domain.CascadeFilter{Modes: []string{"Slayer", "CTF"}},
	}
	result := ResolveFiltersFromRows(makeTranslatedRows(), input)

	if result.Counts.TotalMatchesAfterFilters != 10 {
		t.Errorf("TotalMatchesAfterFilters = %d, want 10 (Slayer+CTF migrated to Assassin+Capture de drapeau)", result.Counts.TotalMatchesAfterFilters)
	}
}

func TestResolveFilters_UnknownLegacyMode_Preserved(t *testing.T) {
	t.Parallel()
	// "Oddball" has no translation — kept as-is, matches nothing → 0 results
	input := domain.FilterContextInput{
		Cascade: domain.CascadeFilter{Modes: []string{"Oddball"}},
	}
	result := ResolveFiltersFromRows(makeTranslatedRows(), input)

	if result.Counts.TotalMatchesAfterFilters != 0 {
		t.Errorf("TotalMatchesAfterFilters = %d, want 0 (unknown mode matches no rows)", result.Counts.TotalMatchesAfterFilters)
	}
}
