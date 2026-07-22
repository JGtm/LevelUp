package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// ---------------------------------------------------------------------------
// buildCumulTab
// ---------------------------------------------------------------------------

func TestBuildCumulTab_Empty(t *testing.T) {
	tab := buildCumulTab(nil)
	if len(tab.CumulativeKD) != 0 {
		t.Errorf("expected empty CumulativeKD, got %d", len(tab.CumulativeKD))
	}
}

func TestBuildCumulTab_SingleMatch(t *testing.T) {
	matches := []legacymatch.StatsMatchRow{
		{Kills: 10, Deaths: 5, StartTime: time.Now()},
	}
	tab := buildCumulTab(matches)
	if len(tab.CumulativeKD) != 1 {
		t.Fatalf("expected 1 CumulativeKD point, got %d", len(tab.CumulativeKD))
	}
	if tab.CumulativeKD[0].Value != 2.0 {
		t.Errorf("expected cumul KD 2.0, got %v", tab.CumulativeKD[0].Value)
	}
	if tab.CumulativeNet[0].Value != 5.0 {
		t.Errorf("expected cumul net 5, got %v", tab.CumulativeNet[0].Value)
	}
}

func TestBuildCumulTab_RollingKD(t *testing.T) {
	matches := make([]legacymatch.StatsMatchRow, 25)
	now := time.Now()
	for i := range matches {
		matches[i] = legacymatch.StatsMatchRow{
			Kills:     10,
			Deaths:    5,
			StartTime: now.Add(time.Duration(i) * time.Hour),
		}
	}
	tab := buildCumulTab(matches)
	if len(tab.RollingKD) != 25 {
		t.Fatalf("expected 25 rolling KD points, got %d", len(tab.RollingKD))
	}
	// All same stats â†’ rolling KD should be 2.0 throughout.
	if tab.RollingKD[24].Value != 2.0 {
		t.Errorf("expected rolling KD 2.0, got %v", tab.RollingKD[24].Value)
	}
}

// ---------------------------------------------------------------------------
// buildIntensityTab
// ---------------------------------------------------------------------------

func TestBuildIntensityTab_Empty(t *testing.T) {
	tab := buildIntensityTab(nil)
	if tab.HeatmapData == nil || len(tab.HeatmapData) != 0 {
		t.Errorf("expected empty HeatmapData, got %v", tab.HeatmapData)
	}
}

func TestBuildIntensityTab_SingleMatch(t *testing.T) {
	dur := 300
	score := 1000
	matches := []legacymatch.StatsMatchRow{
		{
			Kills:             10,
			Deaths:            5,
			StartTime:         time.Date(2025, 1, 6, 14, 0, 0, 0, time.UTC), // Monday
			PersonalScore:     &score,
			TimePlayedSeconds: &dur,
		},
	}
	tab := buildIntensityTab(matches)
	if len(tab.HeatmapData) != 1 {
		t.Fatalf("expected 1 heatmap point, got %d", len(tab.HeatmapData))
	}
	p := tab.HeatmapData[0]
	if p.DayOfWeek != 0 { // Monday
		t.Errorf("expected day 0 (Monday), got %d", p.DayOfWeek)
	}
	if p.Hour != 14 {
		t.Errorf("expected hour 14, got %d", p.Hour)
	}
	if len(tab.ScorePerMinData) != 1 {
		t.Fatalf("expected 1 score/min point, got %d", len(tab.ScorePerMinData))
	}
	// 1000 / (300/60) = 200
	if tab.ScorePerMinData[0].Value != 200.0 {
		t.Errorf("expected score/min 200, got %v", tab.ScorePerMinData[0].Value)
	}
}

// ---------------------------------------------------------------------------
// buildDistributionsTab
// ---------------------------------------------------------------------------

func TestBuildDistributionsTab_Empty(t *testing.T) {
	tab := buildDistributionsTab(nil, true)
	if len(tab.KDABuckets) != 0 {
		t.Errorf("expected empty KDABuckets, got %d", len(tab.KDABuckets))
	}
	if len(tab.KillsBuckets) != 0 {
		t.Errorf("expected empty KillsBuckets, got %d", len(tab.KillsBuckets))
	}
}

func TestBuildDistributionsTab_CorrectBuckets(t *testing.T) {
	// KDA fourni en pointeur — buildKDABuckets lit m.KDA (colonne BDD) et
	// non plus kills/deaths (cf. ADR 0006 + revue 2026-05-10).
	kda1, kda2, kda3, kda4 := 2.0, 1.0, 3.0, 0.0
	matches := []legacymatch.StatsMatchRow{
		{Kills: 10, Deaths: 5, KDA: &kda1, StartTime: time.Now()},
		{Kills: 5, Deaths: 5, KDA: &kda2, StartTime: time.Now()},
		{Kills: 15, Deaths: 5, KDA: &kda3, StartTime: time.Now()},
		{Kills: 0, Deaths: 10, KDA: &kda4, StartTime: time.Now()},
	}
	tab := buildDistributionsTab(matches, true)
	if len(tab.KDABuckets) == 0 {
		t.Fatal("expected non-empty KDABuckets")
	}
	// buildCorrelationPoints gÃ©nÃ¨re 4 points par match (kills_vs_kd, lifespan_vs_kills,
	// lifespan_vs_deaths, kills_vs_deaths) â€” accuracy_vs_kda et mmr_team_vs_enemy sont
	// exclus car Accuracy/KDA/MMR sont nil dans ce fixture.
	if len(tab.CorrelationPoints) != 16 {
		t.Errorf("expected 4 matches Ã— 4 types = 16 correlation points, got %d", len(tab.CorrelationPoints))
	}
}

// ---------------------------------------------------------------------------
// FanoutPlan / FanoutResult (domain types)
// ---------------------------------------------------------------------------

func TestFanoutPlan_Empty(t *testing.T) {
	plan := domain.FanoutPlan{
		SourceGamertag: "TestPlayer",
	}
	if len(plan.Targets) != 0 {
		t.Errorf("expected empty targets")
	}
}

func TestFanoutResult_NoErrors(t *testing.T) {
	result := domain.FanoutResult{
		TargetsProcessed: 3,
		MatchesEnriched:  15,
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors")
	}
	if result.TargetsProcessed != 3 {
		t.Errorf("expected 3 targets processed, got %d", result.TargetsProcessed)
	}
}

// ---------------------------------------------------------------------------
// TimeseriesService (GetPage + NewTimeseriesService)
// ---------------------------------------------------------------------------

type mockTimeseriesRepo struct {
	matches []legacymatch.StatsMatchRow
	err     error
}

func (m *mockTimeseriesRepo) LoadStatsMatches(_ context.Context) ([]legacymatch.StatsMatchRow, error) {
	return m.matches, m.err
}
func (m *mockTimeseriesRepo) LoadLUSRHistory(_ context.Context) ([]domain.LUSRMatchRating, error) {
	return nil, nil
}
func (m *mockTimeseriesRepo) LoadMatchParticipants(_ context.Context) ([]domain.ParticipantRow, error) {
	return nil, nil
}

func TestNewTimeseriesService(t *testing.T) {
	svc := NewTimeseriesService(&mockTimeseriesRepo{})
	if svc == nil {
		t.Fatal("expected non-nil")
	}
}

func TestTimeseriesService_GetPage_Empty(t *testing.T) {
	svc := NewTimeseriesService(&mockTimeseriesRepo{}).
		WithPlayerMatchesRepo(newStatsMockFromRows(nil, nil), "halo_infinite", "Test")
	resp, err := svc.GetPage(context.Background(), domain.TimeseriesQueryRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.TotalMatches != 0 {
		t.Errorf("TotalMatches = %d, want 0", resp.TotalMatches)
	}
}

func TestTimeseriesService_GetPage_Error(t *testing.T) {
	svc := NewTimeseriesService(&mockTimeseriesRepo{}).
		WithPlayerMatchesRepo(newStatsMockFromRows(nil, errors.New("fail")), "halo_infinite", "Test")
	_, err := svc.GetPage(context.Background(), domain.TimeseriesQueryRequest{})
	if err == nil {
		t.Error("expected error")
	}
}

func TestTimeseriesService_GetPage_WithData(t *testing.T) {
	win := 2
	dur := 600
	acc := 0.5
	ps := 1500
	kda := 2.0
	matches := []legacymatch.StatsMatchRow{
		{Kills: 10, Deaths: 5, Assists: 3, Outcome: &win, TimePlayedSeconds: &dur, Accuracy: &acc, PersonalScore: &ps, KDA: &kda},
	}
	svc := NewTimeseriesService(&mockTimeseriesRepo{matches: matches}).
		WithPlayerMatchesRepo(newStatsMockFromRows(matches, nil), "halo_infinite", "Test")
	resp, err := svc.GetPage(context.Background(), domain.TimeseriesQueryRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.TotalMatches != 1 {
		t.Errorf("TotalMatches = %d, want 1", resp.TotalMatches)
	}
	// BriefingKPIs alimente le composant <SessionBriefing> en haut de page.
	if resp.BriefingKPIs == nil {
		t.Fatal("BriefingKPIs should be filled when matches > 0")
	}
	if resp.BriefingKPIs.MatchesCount != 1 {
		t.Errorf("BriefingKPIs.MatchesCount = %d, want 1", resp.BriefingKPIs.MatchesCount)
	}
}

func TestTimeseriesService_GetPage_BriefingKPIsEmptyWhenNoMatches(t *testing.T) {
	svc := NewTimeseriesService(&mockTimeseriesRepo{}).
		WithPlayerMatchesRepo(newStatsMockFromRows(nil, nil), "halo_infinite", "Test")
	resp, err := svc.GetPage(context.Background(), domain.TimeseriesQueryRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.BriefingKPIs != nil {
		t.Errorf("BriefingKPIs should be nil when no matches, got %+v", resp.BriefingKPIs)
	}
}

// ---------------------------------------------------------------------------
// buildTimeseriesSummaryTab
// ---------------------------------------------------------------------------

func TestBuildTimeseriesSummaryTab_Empty(t *testing.T) {
	tab := buildTimeseriesSummaryTab(nil)
	if len(tab.KpiCards) != 0 {
		t.Errorf("expected 0 cards, got %d", len(tab.KpiCards))
	}
}

func TestBuildTimeseriesSummaryTab_WithMatches(t *testing.T) {
	win := 2
	dur := 600
	matches := []legacymatch.StatsMatchRow{
		{Kills: 10, Deaths: 5, Outcome: &win, TimePlayedSeconds: &dur},
		{Kills: 15, Deaths: 3, Outcome: &win, TimePlayedSeconds: &dur},
	}
	tab := buildTimeseriesSummaryTab(matches)
	if len(tab.KpiCards) == 0 {
		t.Error("expected cards")
	}
}

// ---------------------------------------------------------------------------
// buildMatchRows
// ---------------------------------------------------------------------------

func TestBuildMatchRows_Basic(t *testing.T) {
	win := 2
	acc := 0.55
	score := 1200
	dur := 600
	now := time.Now()
	matches := []legacymatch.StatsMatchRow{
		{
			MatchID:           "abc123",
			StartTime:         now,
			Kills:             10,
			Deaths:            5,
			Assists:           2,
			Accuracy:          &acc,
			Outcome:           &win,
			PersonalScore:     &score,
			TimePlayedSeconds: &dur,
			PlaylistName:      "Ranked Arena",
		},
	}
	rows := buildMatchRows(matches, true)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	r := rows[0]
	if r.MatchID != "abc123" {
		t.Errorf("MatchID = %q, want abc123", r.MatchID)
	}
	if r.Index != 0 {
		t.Errorf("Index = %d, want 0 (0-based)", r.Index)
	}
	if r.Kills != 10 {
		t.Errorf("Kills = %d, want 10", r.Kills)
	}
	if r.PlaylistName != "Ranked Arena" {
		t.Errorf("PlaylistName = %q", r.PlaylistName)
	}
}

// ---------------------------------------------------------------------------
// buildAccuracyBuckets
// ---------------------------------------------------------------------------

func TestBuildAccuracyBuckets_Empty(t *testing.T) {
	buckets := buildAccuracyBuckets(nil)
	if len(buckets) != 0 {
		t.Errorf("expected 0 buckets, got %d", len(buckets))
	}
}

func TestBuildAccuracyBuckets_Basic(t *testing.T) {
	acc30 := 0.30
	acc60 := 0.60
	matches := []legacymatch.StatsMatchRow{
		{Accuracy: &acc30},
		{Accuracy: &acc30},
		{Accuracy: &acc60},
		{Accuracy: nil},
	}
	buckets := buildAccuracyBuckets(matches)
	// Should produce non-empty buckets for bins containing 30% and 60%
	if len(buckets) == 0 {
		t.Fatal("expected non-empty buckets")
	}
	total := 0
	for _, b := range buckets {
		total += b.Count
	}
	if total != 3 {
		t.Errorf("total count = %d, want 3 (nil excluded)", total)
	}
}

// ---------------------------------------------------------------------------
// buildCorrelationPoints (types Ã©tendus)
// ---------------------------------------------------------------------------

func TestBuildCorrelationPoints_AllTypes(t *testing.T) {
	win := 2
	acc := 0.55
	dur := 600
	kda := 2.5
	mmrTeam := 1500.0
	mmrEnemy := 1600.0
	matches := []legacymatch.StatsMatchRow{
		{
			Kills:             10,
			Deaths:            5,
			Assists:           2,
			Outcome:           &win,
			Accuracy:          &acc,
			TimePlayedSeconds: &dur,
			KDA:               &kda,
			TeamMMR:           &mmrTeam,
			EnemyMMR:          &mmrEnemy,
		},
	}
	points := buildCorrelationPoints(matches)
	if len(points) == 0 {
		t.Fatal("expected non-empty correlation points")
	}
	// P7.1 : Label composite remplacé par MetricXKey + MetricYKey.
	pairs := make(map[string]bool)
	for _, p := range points {
		pairs[p.MetricXKey+"_vs_"+p.MetricYKey] = true
	}
	for _, want := range []string{"kills_vs_kd_ratio", "kills_vs_deaths", "lifespan_vs_kills"} {
		if !pairs[want] {
			t.Errorf("missing pair %q", want)
		}
	}
}

// ---------------------------------------------------------------------------
// filterStatsMatchRows
// ---------------------------------------------------------------------------

func TestFilterStatsMatchRows_Empty(t *testing.T) {
	rows := filterStatsMatchRows(nil, domain.FilterContextInput{})
	if len(rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(rows))
	}
}

func TestFilterStatsMatchRows_NoFilter(t *testing.T) {
	now := time.Now()
	rows := []legacymatch.StatsMatchRow{
		{MatchID: "m1", StartTime: now, PlaylistName: "Arena"},
		{MatchID: "m2", StartTime: now, PlaylistName: "BTB"},
	}
	out := filterStatsMatchRows(rows, domain.FilterContextInput{})
	if len(out) != 2 {
		t.Errorf("expected 2 rows without filter, got %d", len(out))
	}
}

// TestFilterStatsMatchRows_MatchContextSolo verifie que le filtre
// match_context='solo' est bien applique par la pipeline applyAllFilters
// (page Stats : exclure les matchs en escouade IsWithFriends=true).
func TestFilterStatsMatchRows_MatchContextSolo(t *testing.T) {
	now := time.Now()
	rows := []legacymatch.StatsMatchRow{
		{MatchID: "solo1", StartTime: now, IsWithFriends: false},
		{MatchID: "squad1", StartTime: now, IsWithFriends: true},
		{MatchID: "solo2", StartTime: now, IsWithFriends: false},
	}
	out := filterStatsMatchRows(rows, domain.FilterContextInput{MatchContext: domain.MatchContextSolo})
	if len(out) != 2 {
		t.Fatalf("solo filter: expected 2 rows, got %d", len(out))
	}
	for _, r := range out {
		if r.IsWithFriends {
			t.Errorf("solo filter leaked squad match: %s", r.MatchID)
		}
	}
}

// TestFilterStatsMatchRows_MatchContextSquad verifie le pendant squad.
func TestFilterStatsMatchRows_MatchContextSquad(t *testing.T) {
	now := time.Now()
	rows := []legacymatch.StatsMatchRow{
		{MatchID: "solo1", StartTime: now, IsWithFriends: false},
		{MatchID: "squad1", StartTime: now, IsWithFriends: true},
	}
	out := filterStatsMatchRows(rows, domain.FilterContextInput{MatchContext: domain.MatchContextSquad})
	if len(out) != 1 || out[0].MatchID != "squad1" {
		t.Errorf("squad filter: expected only squad1, got %+v", out)
	}
}

// TestFilterStatsMatchRows_PlaylistFRPreferred : la Value playlist des options du
// filtre est FR-canonique → filterStatsMatchRows doit dériver la playlist depuis
// PlaylistNameFR (sinon une sélection FR ne matcherait jamais les rows, ex. H5).
func TestFilterStatsMatchRows_PlaylistFRPreferred(t *testing.T) {
	now := time.Now()
	rows := []legacymatch.StatsMatchRow{
		{MatchID: "m1", StartTime: now, PlaylistName: "Ranked Arena", PlaylistNameFR: "Arène classée"},
		{MatchID: "m2", StartTime: now, PlaylistName: "Big Team Battle", PlaylistNameFR: "Bataille en équipe"},
	}
	f := domain.FilterContextInput{Cascade: domain.CascadeFilter{Playlists: []string{"Arène classée"}}}
	out := filterStatsMatchRows(rows, f)
	if len(out) != 1 || out[0].MatchID != "m1" {
		t.Fatalf("playlist FR: attendu m1 uniquement, obtenu %+v", out)
	}
}

// TestFilterStatsMatchRows_ModeGameVariantFallback : titre sans pair (H5) — la
// cascade modes matche sur le game_variant normalisé quand PairName est vide ;
// non-régression Infinite : le pair prime sur le game_variant.
func TestFilterStatsMatchRows_ModeGameVariantFallback(t *testing.T) {
	now := time.Now()
	rows := []legacymatch.StatsMatchRow{
		{MatchID: "h5", StartTime: now, GameVariantNameFR: "Assassin"},                        // pair vide → variant
		{MatchID: "inf", StartTime: now, PairName: "Strongholds", GameVariantName: "Slayer"}, // pair présent
	}
	f := domain.FilterContextInput{Cascade: domain.CascadeFilter{Modes: []string{"Assassin"}}}
	out := filterStatsMatchRows(rows, f)
	if len(out) != 1 || out[0].MatchID != "h5" {
		t.Fatalf("mode variant fallback: attendu h5 uniquement, obtenu %+v", out)
	}

	fInf := domain.FilterContextInput{Cascade: domain.CascadeFilter{Modes: []string{"Strongholds"}}}
	outInf := filterStatsMatchRows(rows, fInf)
	if len(outInf) != 1 || outInf[0].MatchID != "inf" {
		t.Fatalf("mode pair Infinite: attendu inf uniquement (game_variant ignoré), obtenu %+v", outInf)
	}
}
