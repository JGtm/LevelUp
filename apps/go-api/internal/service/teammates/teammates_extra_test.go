package teammates

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
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
	matches := []legacymatch.SynthesisMatchRow{
		{MatchID: "m1", Outcome: domain.OutcomeWin, Kills: 10, Deaths: 5},
	}
	exclude := map[string]bool{"m1": true}
	got := computeKPIsFromSynthesisExcluding(matches, exclude)
	if got.MatchCount != 0 {
		t.Errorf("expected 0, got %d", got.MatchCount)
	}
}

func TestComputeKPIsFromSynthesisExcluding_PartialFilter(t *testing.T) {
	matches := []legacymatch.SynthesisMatchRow{
		{MatchID: "m1", Outcome: domain.OutcomeWin, Kills: 10, Deaths: 5},
		{MatchID: "m2", Outcome: domain.OutcomeLoss, Kills: 3, Deaths: 7},
	}
	exclude := map[string]bool{"m1": true}
	got := computeKPIsFromSynthesisExcluding(matches, exclude)
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

// ---------- computeKPIsFromSquadMatches â€” HeadshotKills / PerfectKills ----------

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
	ts := time.Now()
	matches := []legacymatch.SynthesisMatchRow{
		{MatchID: "m1", IsWithFriends: false, SessionLabel: &s1, StartTime: ts},
		{MatchID: "m2", IsWithFriends: true, SessionLabel: &s2, StartTime: ts},
		{MatchID: "m3", IsWithFriends: false, SessionLabel: &s1, StartTime: ts}, // doublon â†’ 1 seul
	}
	got := extractSynthesisSessionLabels(matches)
	if len(got.Solo) != 1 {
		t.Errorf("expected 1 solo session, got %d", len(got.Solo))
	}
	if len(got.Squad) != 1 {
		t.Errorf("expected 1 squad session, got %d", len(got.Squad))
	}
	if got.Solo[0].Label != s1 {
		t.Errorf("expected label %s, got %s", s1, got.Solo[0].Label)
	}
}

func TestExtractSynthesisSessionLabels_SkipsEmptyLabel(t *testing.T) {
	empty := ""
	ts := time.Now()
	matches := []legacymatch.SynthesisMatchRow{
		{MatchID: "m1", IsWithFriends: false, SessionLabel: &empty, StartTime: ts},
		{MatchID: "m2", IsWithFriends: false, SessionLabel: nil, StartTime: ts},
	}
	got := extractSynthesisSessionLabels(matches)
	if len(got.Solo) != 0 || len(got.Squad) != 0 {
		t.Errorf("expected empty labels, got solo=%v squad=%v", got.Solo, got.Squad)
	}
}

func TestExtractSynthesisSessionLabels_TimestampsMinMax(t *testing.T) {
	label := "sess-a"
	t1 := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	t3 := time.Date(2025, 1, 1, 14, 0, 0, 0, time.UTC)
	matches := []legacymatch.SynthesisMatchRow{
		{MatchID: "m1", IsWithFriends: true, SessionLabel: &label, StartTime: t2},
		{MatchID: "m2", IsWithFriends: true, SessionLabel: &label, StartTime: t1},
		{MatchID: "m3", IsWithFriends: true, SessionLabel: &label, StartTime: t3},
	}
	got := extractSynthesisSessionLabels(matches)
	if len(got.Squad) != 1 {
		t.Fatalf("expected 1 squad session, got %d", len(got.Squad))
	}
	e := got.Squad[0]
	if !e.StartedAt.Equal(t1) {
		t.Errorf("StartedAt: expected %v, got %v", t1, e.StartedAt)
	}
	if !e.EndedAt.Equal(t3) {
		t.Errorf("EndedAt: expected %v, got %v", t3, e.EndedAt)
	}
}

func TestExtractSynthesisSessionLabels_SortedByStartedAtDesc(t *testing.T) {
	s1, s2, s3 := "sess-old", "sess-mid", "sess-new"
	tOld := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	tMid := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	tNew := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	matches := []legacymatch.SynthesisMatchRow{
		{MatchID: "m1", IsWithFriends: true, SessionLabel: &s1, StartTime: tOld},
		{MatchID: "m2", IsWithFriends: true, SessionLabel: &s3, StartTime: tNew},
		{MatchID: "m3", IsWithFriends: true, SessionLabel: &s2, StartTime: tMid},
	}
	got := extractSynthesisSessionLabels(matches)
	if len(got.Squad) != 3 {
		t.Fatalf("expected 3 squad sessions, got %d", len(got.Squad))
	}
	if got.Squad[0].Label != s3 {
		t.Errorf("expected newest first (%s), got %s", s3, got.Squad[0].Label)
	}
	if got.Squad[2].Label != s1 {
		t.Errorf("expected oldest last (%s), got %s", s1, got.Squad[2].Label)
	}
}

// ---------- extractSynthesisSessionLabels â€” expÃ©riences & playlists ----------

func TestExtractSynthesisSessionLabels_AggregatesExperiences(t *testing.T) {
	label := "sess-mixed"
	ts := time.Now()
	matches := []legacymatch.SynthesisMatchRow{
		{MatchID: "m1", IsWithFriends: true, SessionLabel: &label, StartTime: ts, IsRanked: true, IsFirefight: false, PlaylistName: "Ranked Arena"},
		{MatchID: "m2", IsWithFriends: true, SessionLabel: &label, StartTime: ts, IsRanked: false, IsFirefight: false, PlaylistName: "Quick Play"},
		{MatchID: "m3", IsWithFriends: true, SessionLabel: &label, StartTime: ts, IsRanked: false, IsFirefight: true, PlaylistName: "Firefight"},
	}
	got := extractSynthesisSessionLabels(matches)
	if len(got.Squad) != 1 {
		t.Fatalf("expected 1 squad session, got %d", len(got.Squad))
	}
	e := got.Squad[0]
	if len(e.Experiences) != 3 {
		t.Errorf("expected 3 experiences, got %v", e.Experiences)
	}
	if len(e.Playlists) != 3 {
		t.Errorf("expected 3 playlists, got %v", e.Playlists)
	}
}

func TestExtractSynthesisSessionLabels_PureRankedSession(t *testing.T) {
	label := "sess-ranked"
	ts := time.Now()
	matches := []legacymatch.SynthesisMatchRow{
		{MatchID: "m1", IsWithFriends: true, SessionLabel: &label, StartTime: ts, IsRanked: true, IsFirefight: false, PlaylistName: "Ranked Arena"},
		{MatchID: "m2", IsWithFriends: true, SessionLabel: &label, StartTime: ts, IsRanked: true, IsFirefight: false, PlaylistName: "Ranked Arena"},
	}
	got := extractSynthesisSessionLabels(matches)
	e := got.Squad[0]
	if len(e.Experiences) != 1 || e.Experiences[0] != "PVP classé" {
		t.Errorf("expected [PVP classé], got %v", e.Experiences)
	}
}

// ---------- filterSynthesisByCascade ----------

func TestFilterSynthesisByCascade_NoFiltersReturnsAll(t *testing.T) {
	matches := []legacymatch.SynthesisMatchRow{
		{MatchID: "m1", IsRanked: true},
		{MatchID: "m2", IsRanked: false},
	}
	got := filterSynthesisByCascade(matches, domain.CascadeFilter{})
	if len(got) != 2 {
		t.Errorf("expected 2, got %d", len(got))
	}
}

func TestFilterSynthesisByCascade_ExperienceFilter(t *testing.T) {
	matches := []legacymatch.SynthesisMatchRow{
		{MatchID: "m1", IsRanked: true, IsFirefight: false},
		{MatchID: "m2", IsRanked: false, IsFirefight: false},
		{MatchID: "m3", IsRanked: false, IsFirefight: true},
	}
	got := filterSynthesisByCascade(matches, domain.CascadeFilter{ExperienceTypes: []string{"PVP classé"}})
	if len(got) != 1 || got[0].MatchID != "m1" {
		t.Errorf("expected only m1 (ranked), got %v", got)
	}
}

func TestFilterSynthesisByCascade_PlaylistFilter(t *testing.T) {
	matches := []legacymatch.SynthesisMatchRow{
		{MatchID: "m1", PlaylistName: "Ranked Arena"},
		{MatchID: "m2", PlaylistName: "Quick Play"},
	}
	got := filterSynthesisByCascade(matches, domain.CascadeFilter{Playlists: []string{"Ranked Arena"}})
	if len(got) != 1 || got[0].MatchID != "m1" {
		t.Errorf("expected only m1, got %v", got)
	}
}

// ---------- filterSynthesisBySession ----------

func TestFilterSynthesisBySession_EmptyFiltersReturnsAll(t *testing.T) {
	s := "s1"
	matches := []legacymatch.SynthesisMatchRow{
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
	matches := []legacymatch.SynthesisMatchRow{
		{MatchID: "m1", IsWithFriends: false, SessionLabel: &solo},
		{MatchID: "m2", IsWithFriends: true, SessionLabel: &squad},
		{MatchID: "m3", IsWithFriends: false, SessionLabel: &squad},
	}
	got := filterSynthesisBySession(matches, []string{solo}, nil)
	if len(got) != 1 || got[0].MatchID != "m1" {
		t.Errorf("expected only m1, got %v", got)
	}
}

func TestFilterSynthesisBySession_SquadFilter(t *testing.T) {
	solo := "session-solo"
	squad := "session-squad"
	matches := []legacymatch.SynthesisMatchRow{
		{MatchID: "m1", IsWithFriends: false, SessionLabel: &solo},
		{MatchID: "m2", IsWithFriends: true, SessionLabel: &squad},
		{MatchID: "m3", IsWithFriends: true, SessionLabel: &solo},
	}
	got := filterSynthesisBySession(matches, nil, []string{squad})
	if len(got) != 1 || got[0].MatchID != "m2" {
		t.Errorf("expected only m2, got %v", got)
	}
}

func TestFilterSynthesisBySession_MultiSquadLabels(t *testing.T) {
	s1 := "session-a"
	s2 := "session-b"
	s3 := "session-c"
	matches := []legacymatch.SynthesisMatchRow{
		{MatchID: "m1", IsWithFriends: true, SessionLabel: &s1},
		{MatchID: "m2", IsWithFriends: true, SessionLabel: &s2},
		{MatchID: "m3", IsWithFriends: true, SessionLabel: &s3},
		{MatchID: "m4", IsWithFriends: false, SessionLabel: &s1},
	}
	// SÃ©lection de 2 sessions escouade sur 3 â†’ union des matchs
	got := filterSynthesisBySession(matches, nil, []string{s1, s2})
	if len(got) != 2 {
		t.Errorf("expected 2 matches (m1+m2), got %d: %v", len(got), got)
	}
	ids := map[string]bool{}
	for _, m := range got {
		ids[m.MatchID] = true
	}
	if !ids["m1"] || !ids["m2"] {
		t.Errorf("expected m1 and m2, got %v", ids)
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
	if bazaar.WinRate != 0.67 {
		t.Errorf("Bazaar: expected win rate 0.67, got %f", bazaar.WinRate)
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

// TestComputeMapBreakdown_OrderedByFirstAppearance verrouille le tri des
// cartes par ordre CHRONOLOGIQUE de première apparition, calqué sur
// TestBuildSquadMapHeatmap_MapsOrderedByFirstAppearance
// (teammates_squad_charts_test.go). "Early" (1 match) est jouée avant "Mid"
// (2 matchs) avant "Late" (3 matchs) : l'ordre fréquence (match_count DESC)
// serait l'inverse (Late>Mid>Early), donc ce test distingue les deux tris.
func TestComputeMapBreakdown_OrderedByFirstAppearance(t *testing.T) {
	base := time.Date(2026, 5, 1, 18, 0, 0, 0, time.UTC)
	matches := []domain.SquadMatchRow{
		{MatchID: "l1", StartTime: base.Add(2 * time.Hour), MapUI: "Late", Outcome: domain.OutcomeWin},
		{MatchID: "l2", StartTime: base.Add(150 * time.Minute), MapUI: "Late", Outcome: domain.OutcomeWin},
		{MatchID: "l3", StartTime: base.Add(3 * time.Hour), MapUI: "Late", Outcome: domain.OutcomeWin},
		{MatchID: "m1", StartTime: base.Add(1 * time.Hour), MapUI: "Mid", Outcome: domain.OutcomeWin},
		{MatchID: "m2", StartTime: base.Add(90 * time.Minute), MapUI: "Mid", Outcome: domain.OutcomeWin},
		{MatchID: "e1", StartTime: base, MapUI: "Early", Outcome: domain.OutcomeWin},
	}
	rows := computeMapBreakdown(matches)
	want := []string{"Early", "Mid", "Late"}
	if len(rows) != len(want) {
		t.Fatalf("want %d maps, got %d (%v)", len(want), len(rows), rows)
	}
	for i, w := range want {
		if rows[i].MapUI != w {
			t.Errorf("rows[%d]: want %q, got %q (ordre chronologique cassé, got order: %v)",
				i, w, rows[i].MapUI, rows)
		}
	}
}

// ---------- enrichMapBreakdownWithSquadStats ----------

// TestEnrichMapBreakdownWithSquadStats_JoinByMapID vérifie le chemin nominal :
// jointure par MapID, injection de HistoricalWinRate (= Wins/Total) et
// HistoricalPerformanceAvg en une seule passe.
func TestEnrichMapBreakdownWithSquadStats_JoinByMapID(t *testing.T) {
	perf := 65.5
	rows := []domain.MapBreakdownRow{
		{MapID: "bazaar_id", MapUI: "Bazaar", MatchCount: 3, WinRate: 0.67},
	}
	stats := map[string]domain.MapSquadStats{
		"bazaar_id": {Wins: 11, Total: 20, PerfAvg: &perf},
	}
	enriched := enrichMapBreakdownWithSquadStats(rows, stats)
	if enriched[0].HistoricalWinRate == nil || *enriched[0].HistoricalWinRate != 0.55 {
		t.Errorf("HistoricalWinRate: want 0.55, got %v", enriched[0].HistoricalWinRate)
	}
	if enriched[0].HistoricalPerformanceAvg == nil || *enriched[0].HistoricalPerformanceAvg != 65.5 {
		t.Errorf("HistoricalPerformanceAvg: want 65.5, got %v", enriched[0].HistoricalPerformanceAvg)
	}
	if enriched[0].HistoricalMatchCount == nil || *enriched[0].HistoricalMatchCount != 20 {
		t.Errorf("HistoricalMatchCount: want 20 (= Total), got %v", enriched[0].HistoricalMatchCount)
	}
}

// TestEnrichMapBreakdownWithSquadStats_FallbackMapUI vérifie le fallback
// MapUI quand MapID est vide (cas dégradé sans UUID exposé).
func TestEnrichMapBreakdownWithSquadStats_FallbackMapUI(t *testing.T) {
	rows := []domain.MapBreakdownRow{
		{MapID: "", MapUI: "Bazaar", MatchCount: 3, WinRate: 0.67},
	}
	stats := map[string]domain.MapSquadStats{
		"Bazaar": {Wins: 1, Total: 2},
	}
	enriched := enrichMapBreakdownWithSquadStats(rows, stats)
	if enriched[0].HistoricalWinRate == nil || *enriched[0].HistoricalWinRate != 0.5 {
		t.Errorf("expected 0.5 via MapUI fallback, got %v", enriched[0].HistoricalWinRate)
	}
}

// TestEnrichMapBreakdownWithSquadStats_NoMatchLeaveNil : si la clé n'est pas
// dans le map de stats (carte jamais jouée avec ce squad), les deux champs
// historiques restent nil — la cellule front affiche "—".
func TestEnrichMapBreakdownWithSquadStats_NoMatchLeaveNil(t *testing.T) {
	rows := []domain.MapBreakdownRow{
		{MapID: "newmap_id", MapUI: "NewMap", MatchCount: 1, WinRate: 1.0},
	}
	stats := map[string]domain.MapSquadStats{"bazaar_id": {Wins: 1, Total: 2}}
	enriched := enrichMapBreakdownWithSquadStats(rows, stats)
	if enriched[0].HistoricalWinRate != nil {
		t.Errorf("HistoricalWinRate: want nil, got %v", enriched[0].HistoricalWinRate)
	}
	if enriched[0].HistoricalPerformanceAvg != nil {
		t.Errorf("HistoricalPerformanceAvg: want nil, got %v", enriched[0].HistoricalPerformanceAvg)
	}
	if enriched[0].HistoricalMatchCount != nil {
		t.Errorf("HistoricalMatchCount: want nil, got %v", enriched[0].HistoricalMatchCount)
	}
}

// TestEnrichMapBreakdownWithSquadStats_NilStats : map nil ne crashe pas et
// laisse les rows intactes (cas LoadMapStatsForSquad échoue ou squad vide).
func TestEnrichMapBreakdownWithSquadStats_NilStats(t *testing.T) {
	rows := []domain.MapBreakdownRow{{MapID: "x", MapUI: "X", MatchCount: 1, WinRate: 1.0}}
	enriched := enrichMapBreakdownWithSquadStats(rows, nil)
	if enriched[0].HistoricalWinRate != nil {
		t.Errorf("nil stats: HistoricalWinRate should remain nil")
	}
}

// TestSquadStatsToWinTotal : conversion vers le format historique
// map[mapID][2]int{wins,total} attendu par buildSquadMatchHistory.
func TestSquadStatsToWinTotal(t *testing.T) {
	stats := map[string]domain.MapSquadStats{
		"a": {Wins: 5, Total: 10},
		"b": {Wins: 0, Total: 3},
	}
	got := squadStatsToWinTotal(stats)
	if got["a"] != [2]int{5, 10} {
		t.Errorf("a: want [5 10], got %v", got["a"])
	}
	if got["b"] != [2]int{0, 3} {
		t.Errorf("b: want [0 3], got %v", got["b"])
	}
	if squadStatsToWinTotal(nil) != nil {
		t.Errorf("nil input: want nil output")
	}
}

// ---------- PerformanceAvg session + historique (teammates.13) ----------

// TestComputeMapBreakdown_PerformanceAvg vérifie l'agrégation de PerformanceScore
// par carte : moyenne des non-nil, nil si aucun score.
func TestComputeMapBreakdown_PerformanceAvg(t *testing.T) {
	p1, p2, p3 := 60.0, 80.0, 50.0
	matches := []domain.SquadMatchRow{
		{MatchID: "m1", MapUI: "Bazaar", Outcome: domain.OutcomeWin, PerformanceScore: &p1},
		{MatchID: "m2", MapUI: "Bazaar", Outcome: domain.OutcomeWin, PerformanceScore: &p2},
		{MatchID: "m3", MapUI: "Bazaar", Outcome: domain.OutcomeLoss, PerformanceScore: nil},
		{MatchID: "m4", MapUI: "Recharge", Outcome: domain.OutcomeLoss, PerformanceScore: &p3},
		{MatchID: "m5", MapUI: "Aquarius", Outcome: domain.OutcomeLoss, PerformanceScore: nil},
	}
	rows := computeMapBreakdown(matches)
	byMap := map[string]domain.MapBreakdownRow{}
	for _, r := range rows {
		byMap[r.MapUI] = r
	}
	bazaar := byMap["Bazaar"]
	if bazaar.PerformanceAvg == nil {
		t.Fatal("Bazaar: expected PerformanceAvg, got nil")
	}
	if *bazaar.PerformanceAvg != 70.0 {
		t.Errorf("Bazaar: expected PerformanceAvg=70 ((60+80)/2), got %f", *bazaar.PerformanceAvg)
	}
	recharge := byMap["Recharge"]
	if recharge.PerformanceAvg == nil || *recharge.PerformanceAvg != 50.0 {
		t.Errorf("Recharge: expected PerformanceAvg=50, got %v", recharge.PerformanceAvg)
	}
	aquarius := byMap["Aquarius"]
	if aquarius.PerformanceAvg != nil {
		t.Errorf("Aquarius: expected PerformanceAvg=nil (no scores), got %v", aquarius.PerformanceAvg)
	}
}

// Note : les anciens tests sur computeHistoricalMap{Stats,WR,Perf}ByLabel ont
// été retirés en même temps que ces fonctions (le filtre IsWithFriends agrégeait
// "matchs avec n'importe quel ami" et non "matchs avec l'escouade exacte"). La
// référence est désormais calculée côté repo via LoadMapStatsForSquad et
// vérifiée par les tests TestEnrichMapBreakdownWithSquadStats_*.

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
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test")

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
	// MapBreakdown doit Ãªtre calculÃ©
	if len(resp.MapBreakdown) == 0 {
		t.Error("expected non-empty MapBreakdown")
	}
	// MatchSeries doit contenir l'entrÃ©e pour Ally1
	if _, ok := resp.MatchSeries["Ally1"]; !ok {
		t.Error("expected MatchSeries entry for Ally1")
	}
	if len(resp.MatchSeries["Ally1"]) != 2 {
		t.Errorf("expected 2 match series points, got %d", len(resp.MatchSeries["Ally1"]))
	}
}

// SessionBriefing — Header populated tests
// =============================================================================

func TestTeammatesService_GetPage_HeaderSoloKPIs_NoSelectedGamertags(t *testing.T) {
	repo := &mockSquadRepo{
		topRows: []domain.TopTeammateRow{},
		synthRows: func() []legacymatch.SynthesisMatchRow {
			tp600, tp400 := 600, 400
			return []legacymatch.SynthesisMatchRow{
				{MatchID: "m1", Outcome: domain.OutcomeWin, Kills: 10, Deaths: 5, TimePlayedSecs: &tp600, StartTime: time.Now()},
				{MatchID: "m2", Outcome: domain.OutcomeLoss, Kills: 6, Deaths: 8, TimePlayedSecs: &tp400, StartTime: time.Now()},
			}
		}(),
	}
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test")

	resp, err := svc.GetPage(context.Background(), "player-xuid", domain.TeammatesQueryRequest{
		// SelectedGamertags vide → mode solo
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Header == nil {
		t.Fatal("Header should be non-nil with matches")
	}
	if resp.Header.SoloKPIs == nil {
		t.Fatal("SoloKPIs should be populated in solo mode")
	}
	if resp.Header.SoloKPIs.MatchesCount != 2 {
		t.Errorf("SoloKPIs.MatchesCount: want 2, got %d", resp.Header.SoloKPIs.MatchesCount)
	}
	// Mode solo → pas de squad fields
	if resp.Header.SquadScore != nil {
		t.Errorf("SquadScore should be nil in solo mode, got %+v", resp.Header.SquadScore)
	}
	if len(resp.Header.PlayerCards) != 0 {
		t.Errorf("PlayerCards should be empty in solo mode, got %d", len(resp.Header.PlayerCards))
	}
}

// directRowsMainMock retourne directement des canonical.PlayerMatchRow via
// LoadPlayerMatches (utile pour les tests qui veulent contrôler XUID/Identity
// du main, contrairement au mockSynthPlayerMatches qui ne set pas Identity).
type directRowsMainMock struct{ rows []canonical.PlayerMatchRow }

func (m *directRowsMainMock) LoadPlayerMatches(_ context.Context, _, _ string, _ port.PlayerMatchFilters) ([]canonical.PlayerMatchRow, error) {
	return m.rows, nil
}
func (m *directRowsMainMock) InvalidatePlayer(_, _ string) {}

func TestTeammatesService_GetPage_HeaderSquadMode_DrillDownDataIsolated(t *testing.T) {
	// Ce test cadenasse le bug du PlayerMatchesAdapter bound-to-main : sans
	// SquadV2Loader, charger les rows d'un teammate retournait silencieusement
	// les rows du main → kpis_by_xuid contenait une seule entree (le main) et
	// le drill-down click n'avait aucun effet. Avec WithSquadLoader, chaque
	// teammate a ses propres canonical rows → entree distincte dans kpis_by_xuid.
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	mainRows := []canonical.PlayerMatchRow{
		rowWithStatsXUID("xuid-main", "m1", t0, canonical.OutcomeWin, 10, 5, 2, 600, 50, 80),
	}
	teammateRows := []canonical.PlayerMatchRow{
		rowWithStatsXUID("xuid-friend1", "m1", t0, canonical.OutcomeWin, 6, 8, 3, 600, 40, 50),
	}
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main":    mainRows,
			"friend1": teammateRows,
		},
	}
	repo := &mockSquadRepo{
		topRows: []domain.TopTeammateRow{{XUID: "xuid-friend1", Gamertag: "friend1", GamesTogether: 5}},
		synthRows: func() []legacymatch.SynthesisMatchRow {
			tp := 600
			return []legacymatch.SynthesisMatchRow{
				{MatchID: "m1", Outcome: domain.OutcomeWin, Kills: 10, Deaths: 5, TimePlayedSecs: &tp, StartTime: t0},
			}
		}(),
	}
	// playerMatchesRepo doit retourner des canonical rows AVEC Identity.XUID
	// pour que extractSquadXUIDs trouve le xuid du main (cf. fix bound-to-main).
	mainMock := &directRowsMainMock{rows: mainRows}
	svc := NewTeammatesService(repo, nil).
		WithPlayerMatchesRepo(mainMock, "halo_infinite", "main").
		WithSquadLoader(loader)
	resp, err := svc.GetPage(context.Background(), "xuid-main", domain.TeammatesQueryRequest{
		SelectedGamertags: []string{"friend1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Header == nil {
		t.Fatal("Header should be non-nil in squad mode")
	}
	// Doit contenir 2 entrees (main + friend1) avec des KPIs distincts
	if len(resp.Header.KPIsByXUID) != 2 {
		t.Fatalf("KPIsByXUID: want 2 entries (main + friend1), got %d — bug bound-to-main probable", len(resp.Header.KPIsByXUID))
	}
	mainKPIs := resp.Header.KPIsByXUID["xuid-main"]
	friendKPIs := resp.Header.KPIsByXUID["xuid-friend1"]
	if mainKPIs == nil || friendKPIs == nil {
		t.Fatalf("Both xuids should have KPIs: main=%v friend=%v", mainKPIs, friendKPIs)
	}
	// Les KPIs doivent etre distincts (sinon le drill-down est cassé)
	if mainKPIs.KillsPerGame == friendKPIs.KillsPerGame {
		t.Errorf("KillsPerGame identique (%v) → bug bound-to-main : les rows teammate sont les rows main",
			mainKPIs.KillsPerGame)
	}
	// PlayerCards doit avoir XUID renseigne pour chaque entree
	for _, c := range resp.Header.PlayerCards {
		if c.XUID == "" {
			t.Errorf("PlayerScoreCard %q : XUID vide → drill-down click sera disabled", c.Gamertag)
		}
	}
}

func TestTeammatesService_GetPage_HeaderEmptyWhenNoMatches(t *testing.T) {
	repo := &mockSquadRepo{
		topRows:   []domain.TopTeammateRow{},
		synthRows: []legacymatch.SynthesisMatchRow{},
	}
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test")
	resp, err := svc.GetPage(context.Background(), "player-xuid", domain.TeammatesQueryRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Pas de match → Header nil (rien à afficher)
	if resp.Header != nil {
		t.Errorf("Header should be nil with no matches, got %+v", resp.Header)
	}
}

func TestTeammatesService_GetPage_UnknownGamertag_Skipped(t *testing.T) {
	repo := &mockSquadRepo{
		topRows: []domain.TopTeammateRow{
			{XUID: "xuid-ally1", Gamertag: "Ally1", GamesTogether: 5},
		},
	}
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test")

	resp, err := svc.GetPage(context.Background(), "player-xuid", domain.TeammatesQueryRequest{
		SelectedGamertags: []string{"Unknown"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Unknown gamertag non prÃ©sent dans topRows â†’ skippÃ©
	if len(resp.Teammates) != 0 {
		t.Errorf("expected 0 teammates, got %d", len(resp.Teammates))
	}
}

// ---------- buildSquadMatchHistory : dominance_flag (F1) ----------

// TestBuildSquadMatchHistory_PropagatesDominanceFlag : le badge narratif chargé
// par le repo (player_match_enrichment) doit arriver tel quel dans la ligne
// d'historique — c'est la source du marqueur de dominance de la bande de
// résultats Escouade.
func TestBuildSquadMatchHistory_PropagatesDominanceFlag(t *testing.T) {
	rows := []domain.SquadMatchRow{
		{MatchID: "m1", MapUI: "Aquarius", DominanceFlag: 3},
		{MatchID: "m2", MapUI: "Recharge"}, // aucun badge
	}
	hist := buildSquadMatchHistory(rows, nil, "halo_infinite", nil, nil)
	if len(hist) != 2 {
		t.Fatalf("want 2 rows, got %d", len(hist))
	}
	byID := map[string]int{}
	for _, h := range hist {
		byID[h.MatchID] = h.DominanceFlag
	}
	if byID["m1"] != 3 {
		t.Errorf("m1 DominanceFlag = %d, want 3 (remontada)", byID["m1"])
	}
	if byID["m2"] != 0 {
		t.Errorf("m2 DominanceFlag = %d, want 0 (aucun badge)", byID["m2"])
	}
}
