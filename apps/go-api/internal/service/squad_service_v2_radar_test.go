package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/games/canonical"
)

func mkPMRowRadar(
	gt string,
	startedAt time.Time,
	kills, deaths, assists, personalScore int,
	avgLife, perfScore *float64,
) canonical.PlayerMatchRow {
	k, d, a, ps := kills, deaths, assists, personalScore
	return canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{StartedAtUTC: startedAt},
		Self: canonical.MatchParticipant{
			Identity:       canonical.PlayerIdentity{Gamertag: gt},
			Kills:          &k,
			Deaths:         &d,
			Assists:        &a,
			PersonalScore:  &ps,
			AvgLifeSeconds: avgLife,
		},
		Enrichment: canonical.PlayerMatchEnrichment{
			PerformanceScore: perfScore,
		},
	}
}

func TestBuildRadarSeries_OneSeriesPerPlayer(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	life := 12.0
	perf := 75.0
	rows := map[string][]canonical.PlayerMatchRow{
		"main": {
			mkPMRowRadar("main", t0, 20, 10, 5, 100, &life, &perf),
			mkPMRowRadar("main", t0.Add(time.Hour), 30, 8, 7, 120, &life, &perf),
		},
		"f1": {
			mkPMRowRadar("f1", t0, 5, 15, 2, 50, &life, &perf),
		},
	}
	series := BuildRadarSeries(rows, "slayer")
	if len(series) != 2 {
		t.Fatalf("want 2 series, got %d", len(series))
	}
	// Tri alpha
	if series[0].Meta["gamertag"] != "f1" || series[1].Meta["gamertag"] != "main" {
		t.Errorf("series order want [f1, main], got [%v, %v]",
			series[0].Meta["gamertag"], series[1].Meta["gamertag"])
	}
	// 6 axes attendus
	if len(series[0].Axes) != 6 {
		t.Errorf("want 6 axes, got %d", len(series[0].Axes))
	}
	// Ordre axes : Combat, Survival, Support, Score, Objective, Impact
	wantOrder := []narrative.ParticipationAxis{
		narrative.AxisCombat, narrative.AxisSurvival, narrative.AxisSupport,
		narrative.AxisScore, narrative.AxisObjective, narrative.AxisImpact,
	}
	for i, want := range wantOrder {
		if series[0].Axes[i].Axis != want {
			t.Errorf("[0] axis[%d] want %s, got %s", i, want, series[0].Axes[i].Axis)
		}
	}
}

func TestBuildRadarSeries_NormalizesAxes(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	// Slayer thresholds : Combat=250, Survival=50, Score=80, Impact=100
	// 1 match avec kills=125 -> Combat raw=125, value=50.0 (slayer)
	rows := map[string][]canonical.PlayerMatchRow{
		"main": {mkPMRowRadar("main", t0, 125, 0, 0, 0, nil, nil)},
	}
	series := BuildRadarSeries(rows, "slayer")
	combat := series[0].Axes[0]
	if combat.Axis != narrative.AxisCombat {
		t.Fatalf("first axis want combat, got %s", combat.Axis)
	}
	if combat.Raw != 125 {
		t.Errorf("Combat raw want 125, got %f", combat.Raw)
	}
	if combat.Value != 50.0 {
		t.Errorf("Combat value want 50.0 (125/250*100), got %f", combat.Value)
	}
}

func TestBuildRadarSeries_ObjectiveDeferredToZero(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	rows := map[string][]canonical.PlayerMatchRow{
		"main": {mkPMRowRadar("main", t0, 50, 10, 5, 999_999, nil, nil)},
	}
	series := BuildRadarSeries(rows, "ctf")
	// Objective axis = 0 (deferred awards-based)
	for _, ax := range series[0].Axes {
		if ax.Axis == narrative.AxisObjective {
			if ax.Raw != 0 || ax.Value != 0 {
				t.Errorf("Objective want raw=0 value=0 (deferred), got raw=%f value=%f",
					ax.Raw, ax.Value)
			}
		}
	}
}

func TestBuildRadarSeries_SkipsRowsWithoutOptional(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	life := 10.0
	rows := map[string][]canonical.PlayerMatchRow{
		"main": {
			// 1 row avec life, 1 sans
			mkPMRowRadar("main", t0, 10, 0, 0, 0, &life, nil),
			mkPMRowRadar("main", t0.Add(time.Hour), 10, 0, 0, 0, nil, nil),
		},
	}
	series := BuildRadarSeries(rows, "slayer")
	for _, ax := range series[0].Axes {
		if ax.Axis == narrative.AxisSurvival {
			// Survival = avg(10) sur 1 row valide = 10.0
			if ax.Raw != 10.0 {
				t.Errorf("Survival raw want 10.0 (avg sur 1 valide), got %f", ax.Raw)
			}
		}
		if ax.Axis == narrative.AxisImpact {
			// Impact = 0 (aucune row avec perf score)
			if ax.Raw != 0 {
				t.Errorf("Impact raw want 0 (no perf score), got %f", ax.Raw)
			}
		}
	}
}

func TestBuildRadarSeries_EmptyInput(t *testing.T) {
	t.Parallel()
	if got := BuildRadarSeries(nil, "slayer"); got != nil {
		t.Errorf("nil rows: want nil, got %v", got)
	}
}

func TestBuildRadarSeries_DefaultModeFamily(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	rows := map[string][]canonical.PlayerMatchRow{
		"main": {mkPMRowRadar("main", t0, 50, 0, 0, 0, nil, nil)},
	}
	// Mode family vide -> custom thresholds (Combat=150)
	series := BuildRadarSeries(rows, "")
	combat := series[0].Axes[0]
	// 50/150 * 100 = 33.33
	if combat.Value < 33.0 || combat.Value > 34.0 {
		t.Errorf("custom Combat value want ~33.3, got %f", combat.Value)
	}
}
