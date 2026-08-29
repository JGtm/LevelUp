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
	p := buildHeroProgress(xpHeroTotal/2, 100, 0, 0)
	if p.Percentage != 50.0 {
		t.Errorf("expected 50%%, got %v", p.Percentage)
	}
	if p.XPRemaining != xpHeroTotal/2 {
		t.Errorf("expected remaining=%d, got %d", xpHeroTotal/2, p.XPRemaining)
	}
	if p.CurrentRank != 100 {
		t.Errorf("expected CurrentRank=100, got %d", p.CurrentRank)
	}
}

func TestBuildHeroProgress_MaxCap(t *testing.T) {
	p := buildHeroProgress(xpHeroTotal+1000, rankMax, 0, 0)
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
		{RecordedAt: now.Add(-10 * 24 * time.Hour), XPTotal: 0},
		{RecordedAt: now, XPTotal: 10000},
	}
	rate := computeActiveXPPerDay(history)
	// 10000 XP / 10 jours = 1000
	if rate < 900 || rate > 1100 {
		t.Errorf("expected ~1000 XP/day, got %v", rate)
	}
}

func TestComputeActiveXPPerDay_InsufficientHistory(t *testing.T) {
	history := []domain.XPHistoryPoint{{RecordedAt: time.Now(), XPTotal: 5000}}
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

// ---------------------------------------------------------------------------
// MatchViewService — le score de l'en-tête (points ou MANCHES)
//
// Ex-tests de buildScoreLabelFromMeta, MIGRÉS vers applyMatchHeaderScore : le libellé n'est
// plus fabriqué ici (source unique analysis.TeamScoreLabel) et le score dépend désormais de
// la table [rounds_decide], portée par le service.
// ---------------------------------------------------------------------------

func headerScore(meta *domain.MatchMetaRaw, teamID *int, roundsDecide map[string]bool) domain.MatchViewHeader {
	var h domain.MatchViewHeader
	var stats *domain.PlayerMatchStatsRaw
	if teamID != nil {
		stats = &domain.PlayerMatchStatsRaw{TeamID: teamID}
	}
	applyMatchHeaderScore(&h, meta, stats, roundsDecide)
	return h
}

func TestApplyMatchHeaderScore_Team0(t *testing.T) {
	s0, s1 := int16(50), int16(47)
	teamID := 0
	h := headerScore(&domain.MatchMetaRaw{Team0Score: &s0, Team1Score: &s1}, &teamID, nil)
	if h.ScoreLabel != "50 - 47" || h.ScoreKind != "points" {
		t.Errorf("got %q / %q, want \"50 - 47\" / \"points\"", h.ScoreLabel, h.ScoreKind)
	}
}

func TestApplyMatchHeaderScore_Team1Swaps(t *testing.T) {
	s0, s1 := int16(50), int16(47)
	teamID := 1
	h := headerScore(&domain.MatchMetaRaw{Team0Score: &s0, Team1Score: &s1}, &teamID, nil)
	if h.ScoreLabel != "47 - 50" {
		t.Errorf("got %q, want \"47 - 50\" (l'équipe du joueur d'abord)", h.ScoreLabel)
	}
}

func TestApplyMatchHeaderScore_Nil(t *testing.T) {
	h := headerScore(nil, nil, nil)
	if h.ScoreLabel != "" || h.ScoreKind != "" {
		t.Errorf("meta nil : got %q / %q, want vides", h.ScoreLabel, h.ScoreKind)
	}
}

// Le cas qui justifie tout le chantier : témoin 293a763e (Arena:Oddball), victoire 2 manches
// à 1 alors que les POINTS disent 181-186. La variante est déclarée dans [rounds_decide].
func TestApplyMatchHeaderScore_VarianteADecideeEnManches(t *testing.T) {
	s0, s1 := int16(181), int16(186)
	r0, r1, tot := int16(2), int16(1), int16(3)
	variant := "Arena:Oddball"
	teamID := 0
	h := headerScore(&domain.MatchMetaRaw{
		Team0Score: &s0, Team1Score: &s1,
		Team0RoundsWon: &r0, Team1RoundsWon: &r1, RoundsTotal: &tot,
		GameVariantName: &variant,
	}, &teamID, map[string]bool{"Arena:Oddball": true})
	if h.ScoreLabel != "2 - 1" || h.ScoreKind != "rounds" {
		t.Errorf("got %q / %q, want \"2 - 1\" / \"rounds\"", h.ScoreLabel, h.ScoreKind)
	}
	if h.ScorePointsLabel != "181 - 186" {
		t.Errorf("score API en second plan = %q, want \"181 - 186\"", h.ScorePointsLabel)
	}
	if h.ScoreMine == nil || *h.ScoreMine != 2 || h.ScoreTheirs == nil || *h.ScoreTheirs != 1 {
		t.Errorf("nombres = %v/%v, want 2/1 (le rejeu les colore séparément)", h.ScoreMine, h.ScoreTheirs)
	}
}

// Variante NON déclarée : mêmes données de manches, mais on garde les points — c'est la
// dégradation qui protège le CTF d'arène (deux mi-temps, score = captures).
func TestApplyMatchHeaderScore_VarianteNonDeclareeGardeLesPoints(t *testing.T) {
	s0, s1 := int16(2), int16(3)
	r0, r1, tot := int16(0), int16(1), int16(2)
	variant := "CTF:Arena"
	teamID := 0
	h := headerScore(&domain.MatchMetaRaw{
		Team0Score: &s0, Team1Score: &s1,
		Team0RoundsWon: &r0, Team1RoundsWon: &r1, RoundsTotal: &tot,
		GameVariantName: &variant,
	}, &teamID, map[string]bool{"Arena:Oddball": true})
	if h.ScoreLabel != "2 - 3" || h.ScoreKind != "points" {
		t.Errorf("got %q / %q, want \"2 - 3\" / \"points\"", h.ScoreLabel, h.ScoreKind)
	}
	if h.ScorePointsLabel != "" {
		t.Errorf("aucun second plan attendu en lecture points, got %q", h.ScorePointsLabel)
	}
}

// ---------------------------------------------------------------------------
// MatchViewService — formatDateFRLong
// ---------------------------------------------------------------------------

func TestFormatDateFRLong_Known(t *testing.T) {
	// 2025-12-01 19:42 UTC → local peut varier, on teste juste que ça parse
	ts := time.Date(2025, time.December, 1, 19, 42, 0, 0, time.UTC)
	got := formatDateFRLong(ts)
	if got == "" {
		t.Error("expected non-empty string")
	}
	// Le résultat doit contenir "déc." et "2025"
	if len(got) < 10 {
		t.Errorf("result too short: %q", got)
	}
}

// ---------------------------------------------------------------------------
// MatchViewService — convertMedals
// ---------------------------------------------------------------------------

func TestConvertMedals_Empty(t *testing.T) {
	result := convertMedals(nil, "halo_infinite")
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d", len(result))
	}
}

func TestConvertMedals_Single(t *testing.T) {
	raw := []domain.MedalRaw{
		{MedalID: 42, Count: 3, Label: "Killing Spree"},
	}
	result := convertMedals(raw, "halo_infinite")
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
	if result[0].MedalNameID != 42 || result[0].Count != 3 || result[0].Name != "Killing Spree" {
		t.Errorf("unexpected medal: %+v", result[0])
	}
}

// ---------------------------------------------------------------------------
// ExplorerService — convertCommonMatches
// ---------------------------------------------------------------------------

func TestConvertCommonMatches_WereTeammates(t *testing.T) {
	team0 := 0
	raw := []domain.CommonMatchRaw{
		{
			MatchID:        "match1",
			MapUI:          "Aquarius",
			ModeUI:         "Slayer",
			Player1TeamID:  &team0,
			Player2TeamID:  &team0,
			Player1Outcome: 2,
		},
	}
	result := convertCommonMatches(raw)
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
	if !result[0].WereTeammates {
		t.Error("expected were_teammates=true when same team ID")
	}
	if result[0].PlayerOutcome != 2 {
		t.Errorf("expected outcome=2, got %d", result[0].PlayerOutcome)
	}
	if result[0].OutcomeLabel == "" {
		t.Error("OutcomeLabel vide — doit être résolu")
	}
}

func TestConvertCommonMatches_Enemies(t *testing.T) {
	team0 := 0
	team1 := 1
	raw := []domain.CommonMatchRaw{
		{
			MatchID:       "match1",
			Player1TeamID: &team0,
			Player2TeamID: &team1,
		},
	}
	result := convertCommonMatches(raw)
	if result[0].WereTeammates {
		t.Error("expected were_teammates=false when different team IDs")
	}
}
