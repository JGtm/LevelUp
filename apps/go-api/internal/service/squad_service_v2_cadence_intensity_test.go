package service

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
)

func TestBuildCadenceChart_AggregatesAcrossPlayersAndMatches(t *testing.T) {
	t.Parallel()
	// 2 matchs, 2 joueurs. PhaseSeconds=60 -> chaque bucket = 60_000 ms.
	events := []canonical.HighlightEvent{
		killEv("m1", "x_p1", "x_v1", 1_000),   // m1, p1, bucket 0
		killEv("m1", "x_p1", "x_v2", 65_000),  // m1, p1, bucket 1
		killEv("m1", "x_p2", "x_v3", 5_000),   // m1, p2, bucket 0
		killEv("m2", "x_p1", "x_v4", 130_000), // m2, p1, bucket 2
	}
	squadXUIDs := map[string]string{
		"main": "x_p1",
		"f1":   "x_p2",
	}
	chart := BuildCadenceChart(events, squadXUIDs, 60, nil)

	if chart.Key != "squad.synergies.cadence" {
		t.Errorf("Key want squad.synergies.cadence, got %s", chart.Key)
	}
	// MaxBuckets = 3 (m2 a un bucket 2)
	if len(chart.Datapoints) != 3 {
		t.Fatalf("want 3 datapoints (max buckets), got %d", len(chart.Datapoints))
	}
	// Bucket 0 : main=1, f1=1
	if chart.Datapoints[0].Components["main"] != 1 || chart.Datapoints[0].Components["f1"] != 1 {
		t.Errorf("bucket 0 expected main=1 f1=1, got %v", chart.Datapoints[0].Components)
	}
	// Bucket 1 : main=1, f1=0
	if chart.Datapoints[1].Components["main"] != 1 || chart.Datapoints[1].Components["f1"] != 0 {
		t.Errorf("bucket 1 expected main=1 f1=0, got %v", chart.Datapoints[1].Components)
	}
	// Bucket 2 : main=1, f1=0
	if chart.Datapoints[2].Components["main"] != 1 || chart.Datapoints[2].Components["f1"] != 0 {
		t.Errorf("bucket 2 expected main=1 f1=0, got %v", chart.Datapoints[2].Components)
	}
	// Categories doivent etre triees stable phase_00, phase_01, phase_02
	wantCats := []string{"phase_00", "phase_01", "phase_02"}
	for i, want := range wantCats {
		if chart.Datapoints[i].Category != want {
			t.Errorf("Category[%d] want %s, got %s", i, want, chart.Datapoints[i].Category)
		}
	}
}

func TestBuildCadenceChart_EmptySquadReturnsEmpty(t *testing.T) {
	t.Parallel()
	chart := BuildCadenceChart(
		[]canonical.HighlightEvent{killEv("m1", "x_p1", "x_v1", 1_000)},
		nil,
		60,
		nil,
	)
	if len(chart.Datapoints) != 0 {
		t.Errorf("empty squad: want 0 datapoints, got %d", len(chart.Datapoints))
	}
}

func TestBuildCadenceChart_DefaultPhaseSeconds(t *testing.T) {
	t.Parallel()
	chart := BuildCadenceChart(
		[]canonical.HighlightEvent{killEv("m1", "x_p1", "x_v1", 1_000)},
		map[string]string{"main": "x_p1"},
		0,
		nil,
	)
	if chart.Meta["phase_seconds"] != 60 {
		t.Errorf("default phase_seconds want 60, got %v", chart.Meta["phase_seconds"])
	}
}

func TestBuildIntensityHeatmap_NormalizesPerMatch(t *testing.T) {
	t.Parallel()
	// Match m1 max=100 ; nBuckets=10 -> chaque bucket = 10ms.
	// 1 event au bucket 0, 2 events au bucket 9.
	events := []canonical.HighlightEvent{
		evtIntensity("m1", string(canonical.EventKill), 5),
		evtIntensity("m1", string(canonical.EventKill), 95),
		evtIntensity("m1", string(canonical.EventDeath), 100),
	}
	chart := BuildIntensityHeatmap(events, 10, nil)

	if chart.Key != "squad.synergies.intensity" {
		t.Errorf("Key want squad.synergies.intensity, got %s", chart.Key)
	}
	if len(chart.Datapoints) != 10 {
		t.Fatalf("want 10 datapoints (1 match x 10 buckets), got %d", len(chart.Datapoints))
	}
	// Bucket 0 : 1 event ; max bucket = bucket 9 avec 2 events ; normalise = 0.5
	if chart.Datapoints[0].Value != 0.5 {
		t.Errorf("bucket 0 normalized want 0.5, got %f", chart.Datapoints[0].Value)
	}
	// Bucket 9 : 2 events ; normalise = 1.0
	if chart.Datapoints[9].Value != 1.0 {
		t.Errorf("bucket 9 normalized want 1.0, got %f", chart.Datapoints[9].Value)
	}
	// Detail.raw doit refleter le brut
	if chart.Datapoints[9].Detail["raw"] != 2 {
		t.Errorf("bucket 9 raw want 2, got %v", chart.Datapoints[9].Detail["raw"])
	}
	// Y = MatchID
	if chart.Datapoints[0].Y != "m1" {
		t.Errorf("Y want m1, got %s", chart.Datapoints[0].Y)
	}
}

func TestBuildIntensityHeatmap_MultiMatchSorted(t *testing.T) {
	t.Parallel()
	events := []canonical.HighlightEvent{
		evtIntensity("m_b", string(canonical.EventKill), 100),
		evtIntensity("m_a", string(canonical.EventKill), 100),
	}
	chart := BuildIntensityHeatmap(events, 5, nil)
	// 2 matchs x 5 buckets = 10 datapoints
	if len(chart.Datapoints) != 10 {
		t.Fatalf("want 10 datapoints, got %d", len(chart.Datapoints))
	}
	// Tri par MatchID asc -> les 5 premiers Y = m_a, les 5 suivants = m_b
	for i := 0; i < 5; i++ {
		if chart.Datapoints[i].Y != "m_a" {
			t.Errorf("[%d] Y want m_a, got %s", i, chart.Datapoints[i].Y)
		}
	}
	for i := 5; i < 10; i++ {
		if chart.Datapoints[i].Y != "m_b" {
			t.Errorf("[%d] Y want m_b, got %s", i, chart.Datapoints[i].Y)
		}
	}
}

func TestBuildIntensityHeatmap_EmptyEvents(t *testing.T) {
	t.Parallel()
	chart := BuildIntensityHeatmap(nil, 10, nil)
	if len(chart.Datapoints) != 0 {
		t.Errorf("empty events: want 0 datapoints, got %d", len(chart.Datapoints))
	}
	if chart.Meta["n_buckets"] != 10 {
		t.Errorf("Meta.n_buckets want 10, got %v", chart.Meta["n_buckets"])
	}
}

func evtIntensity(matchID, eventType string, timeMS int64) canonical.HighlightEvent {
	return canonical.HighlightEvent{
		MatchID:   matchID,
		EventType: eventType,
		TimeMS:    timeMS,
	}
}
