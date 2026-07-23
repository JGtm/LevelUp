package service

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// fakeSquadLoaderFull est un loader qui supporte les 4 sources (matchs,
// events, weapons, medals) — utilise pour tester GetSquadPage end-to-end.
type fakeSquadLoaderFull struct {
	rowsByGT map[string][]canonical.PlayerMatchRow
	events   []canonical.HighlightEvent
	weapons  []port.WeaponKillRow
	medals   []port.MedalRow
}

func (f *fakeSquadLoaderFull) LoadFor(_ context.Context, _, gt string, _ port.PlayerMatchFilters) ([]canonical.PlayerMatchRow, error) {
	return f.rowsByGT[gt], nil
}

func (f *fakeSquadLoaderFull) LoadHighlightEvents(_ context.Context, _ string, _ port.HighlightEventFilters) ([]canonical.HighlightEvent, error) {
	return f.events, nil
}

func (f *fakeSquadLoaderFull) LoadWeaponKills(_ context.Context, _ string, _ port.WeaponKillFilters) ([]port.WeaponKillRow, error) {
	return f.weapons, nil
}

func (f *fakeSquadLoaderFull) LoadKillMechanics(_ context.Context, _ string, _ port.WeaponKillFilters) ([]port.KillMechanicsRow, error) {
	return nil, nil
}

func (f *fakeSquadLoaderFull) LoadMedals(_ context.Context, _ string, _ port.MedalsByXUIDFilters) ([]port.MedalRow, error) {
	return f.medals, nil
}

func (f *fakeSquadLoaderFull) LoadEmblemURLs(_ context.Context, _ string, _ []string) map[string]string {
	return nil
}

func (f *fakeSquadLoaderFull) LoadMapStatsForSquad(_ context.Context, _, _ string, _ []string) (map[string]domain.MapSquadStats, error) {
	return nil, nil
}

func (f *fakeSquadLoaderFull) LoadObjectiveScores(_ context.Context, _, _ string, _ []string) (map[string]int, error) {
	return map[string]int{}, nil
}

func (f *fakeSquadLoaderFull) LoadPlayerAssistsModel(_ context.Context, _, _, _ string) (*domain.PlayerAssistsModel, error) {
	return nil, nil
}

func (f *fakeSquadLoaderFull) LoadPopulationalAssistsCoef(_ context.Context, _, _ string) (float64, float64, bool, error) {
	return 0, 0, false, nil
}

func mkRowFull(gt, xuid, matchID string, t time.Time, outcome canonical.Outcome, kills, deaths int) canonical.PlayerMatchRow {
	k, d := kills, deaths
	a := 3
	dur := 600
	perfScore := 70.0
	return canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			MatchID:         matchID,
			StartedAtUTC:    t,
			DurationSeconds: &dur,
			Outcome:         outcome,
			Map: &canonical.AssetReference{
				ID: "map_aquarius", DefaultLabel: "Aquarius",
			},
		},
		Self: canonical.MatchParticipant{
			Identity: canonical.PlayerIdentity{Gamertag: gt, XUID: xuid},
			Outcome:  outcome,
			Kills:    &k,
			Deaths:   &d,
			Assists:  &a,
		},
		Enrichment: canonical.PlayerMatchEnrichment{
			PerformanceScore: &perfScore,
		},
	}
}

func TestGetSquadPage_FullComposition(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(time.Hour)

	xuidMain := "x_main"
	xuidF1 := "x_f1"

	loader := &fakeSquadLoaderFull{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main": {
				mkRowFull("main", xuidMain, "m1", t0, canonical.OutcomeWin, 10, 5),
				mkRowFull("main", xuidMain, "m2", t1, canonical.OutcomeLoss, 8, 9),
			},
			"f1": {
				mkRowFull("f1", xuidF1, "m1", t0, canonical.OutcomeWin, 6, 7),
				mkRowFull("f1", xuidF1, "m2", t1, canonical.OutcomeLoss, 5, 8),
			},
		},
		events: []canonical.HighlightEvent{
			{
				MatchID: "m1", EventType: string(canonical.EventKill),
				TimeMS: 5000, KillerXUID: &xuidMain, VictimXUID: &xuidF1,
			},
		},
		weapons: []port.WeaponKillRow{
			{XUID: xuidMain, WeaponID: 100, Kills: 10, Label: "BR75"},
			{XUID: xuidF1, WeaponID: 100, Kills: 6, Label: "BR75"},
		},
		medals: []port.MedalRow{
			{XUID: xuidMain, MatchID: "m1", MedalID: 200, Count: 1, Label: "Killing Spree"},
		},
	}

	svc := NewSquadServiceV2(loader)
	resp, err := svc.GetSquadPage(context.Background(), "halo_infinite", "main", []string{"f1"}, temporal.PeriodAll, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("GetSquadPage: %v", err)
	}

	// Vérifications de base
	if resp.SharedMatchesCount != 2 {
		t.Errorf("SharedMatchesCount want 2, got %d", resp.SharedMatchesCount)
	}
	if resp.Header == nil {
		t.Error("Header want non-nil (chunk S2)")
	}
	if resp.Charts == nil {
		t.Fatal("Charts want non-nil")
	}
	if resp.Tables == nil {
		t.Fatal("Tables want non-nil")
	}

	// Synergies (S3+S4)
	if resp.Charts.MapBreakdownLollipop == nil {
		t.Error("MapBreakdownLollipop want non-nil")
	}
	if resp.Charts.HeatmapPlayerMap == nil {
		t.Error("HeatmapPlayerMap want non-nil")
	}
	if len(resp.Charts.TimelineMultiPlayer) == 0 {
		t.Error("TimelineMultiPlayer want non-empty")
	}

	// Cadence + Intensité (S6) — nécessite events
	if resp.Charts.Cadence == nil {
		t.Error("Cadence want non-nil (events fournis)")
	}
	if resp.Charts.IntensityHeatmap == nil {
		t.Error("IntensityHeatmap want non-nil")
	}

	// Impact (S5)
	if resp.Charts.ImpactMatrix == nil {
		t.Error("ImpactMatrix want non-nil")
	}
	if len(resp.Charts.ImpactRanking) != 8 {
		t.Errorf("ImpactRanking want 8 cols, got %d", len(resp.Charts.ImpactRanking))
	}

	// Contributions (S7)
	if resp.Charts.PerMinuteStats == nil {
		t.Error("PerMinuteStats want non-nil")
	}
	if len(resp.Charts.AssistsTimeseries) == 0 {
		t.Error("AssistsTimeseries want non-empty")
	}

	// Radar (S8) — 2 joueurs (main + f1) → 2 séries
	if len(resp.Charts.Radar) != 2 {
		t.Errorf("Radar want 2 series, got %d", len(resp.Charts.Radar))
	}

	// Tables (S9)
	if len(resp.Tables.History) != 2 {
		t.Errorf("History want 2 rows, got %d", len(resp.Tables.History))
	}
	if len(resp.Tables.Weapons) == 0 {
		t.Error("Weapons want non-empty")
	}
	if len(resp.Tables.Medals) == 0 {
		t.Error("Medals want non-empty")
	}
}

func TestGetSquadPage_NoEvents_OmitsCadenceImpact(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)

	loader := &fakeSquadLoaderFull{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main": {mkRowFull("main", "x_main", "m1", t0, canonical.OutcomeWin, 10, 5)},
		},
		// Pas d'events
	}

	svc := NewSquadServiceV2(loader)
	resp, err := svc.GetSquadPage(context.Background(), "halo_infinite", "main", nil, temporal.PeriodAll, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("GetSquadPage: %v", err)
	}

	// Cadence/Impact omis (events vides)
	if resp.Charts.Cadence != nil {
		t.Error("Cadence want nil (no events)")
	}
	if resp.Charts.ImpactMatrix != nil {
		t.Error("ImpactMatrix want nil (no events)")
	}
	// Synergies/Contributions/Radar/History toujours buildables
	if resp.Charts.MapBreakdownLollipop == nil {
		t.Error("MapBreakdownLollipop want non-nil")
	}
	if len(resp.Tables.History) != 1 {
		t.Errorf("History want 1 row, got %d", len(resp.Tables.History))
	}
}

func TestGetSquadPage_EmptyIntersection_NoChartsBuilt(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(time.Hour)

	// main et f1 ont des matchs disjoints
	loader := &fakeSquadLoaderFull{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main": {mkRowFull("main", "x_main", "m1", t0, canonical.OutcomeWin, 10, 5)},
			"f1":   {mkRowFull("f1", "x_f1", "m2", t1, canonical.OutcomeLoss, 5, 8)},
		},
	}

	svc := NewSquadServiceV2(loader)
	resp, err := svc.GetSquadPage(context.Background(), "halo_infinite", "main", []string{"f1"}, temporal.PeriodAll, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("GetSquadPage: %v", err)
	}

	if resp.SharedMatchesCount != 0 {
		t.Errorf("SharedMatchesCount want 0 (no intersection), got %d", resp.SharedMatchesCount)
	}
	// Charts/Tables doivent être nil (pas de matchs partagés → on n'invoque pas les builders)
	if resp.Charts != nil {
		t.Error("Charts want nil (intersection vide)")
	}
	if resp.Tables != nil {
		t.Error("Tables want nil (intersection vide)")
	}
}
