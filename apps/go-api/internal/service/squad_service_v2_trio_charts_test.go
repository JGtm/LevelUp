package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/games/canonical"
)

func mkPMRowTrio(
	gt string,
	startedAt time.Time,
	assists *int,
	kda *float64,
	accuracy *float64,
	avgLife *float64,
	perfScore *float64,
) canonical.PlayerMatchRow {
	row := canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{StartedAtUTC: startedAt},
		Self: canonical.MatchParticipant{
			Identity: canonical.PlayerIdentity{Gamertag: gt},
		},
	}
	row.Self.Assists = assists
	row.Self.KDA = kda
	row.Self.Accuracy = accuracy
	row.Self.AvgLifeSeconds = avgLife
	row.Enrichment.PerformanceScore = perfScore
	return row
}

func TestBuildAssistsChart_OneTracePerPlayer(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	a1, a2 := 5, 3
	rows := map[string][]canonical.PlayerMatchRow{
		"main": {mkPMRowTrio("main", t0, &a1, nil, nil, nil, nil)},
		"f1":   {mkPMRowTrio("f1", t0, &a2, nil, nil, nil, nil)},
	}
	out := BuildAssistsChart(rows)
	if len(out) != 2 {
		t.Fatalf("want 2 traces, got %d", len(out))
	}
	// Tri alpha
	if out[0].Meta["gamertag"] != "f1" || out[1].Meta["gamertag"] != "main" {
		t.Errorf("trace order want [f1, main], got [%v, %v]",
			out[0].Meta["gamertag"], out[1].Meta["gamertag"])
	}
	if out[0].Datapoints[0].Y != 3 {
		t.Errorf("f1 assists want 3, got %f", out[0].Datapoints[0].Y)
	}
	if out[1].Datapoints[0].Y != 5 {
		t.Errorf("main assists want 5, got %f", out[1].Datapoints[0].Y)
	}
}

func TestBuildKDAChart_SkipsNilValues(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	kda := 1.5
	rows := map[string][]canonical.PlayerMatchRow{
		"main": {
			mkPMRowTrio("main", t0, nil, nil, nil, nil, nil), // pas de KDA
			mkPMRowTrio("main", t0.Add(time.Hour), nil, &kda, nil, nil, nil),
		},
	}
	out := BuildKDAChart(rows)
	if len(out) != 1 {
		t.Fatalf("want 1 trace, got %d", len(out))
	}
	if len(out[0].Datapoints) != 1 {
		t.Errorf("want 1 datapoint (1 row sans KDA skip), got %d", len(out[0].Datapoints))
	}
	if out[0].Datapoints[0].Y != 1.5 {
		t.Errorf("Y want 1.5, got %f", out[0].Datapoints[0].Y)
	}
}

func TestBuildAccuracyChart(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	acc := 42.0 // accuracy en pourcentage 0..100 (passe-through, pas de conversion)
	rows := map[string][]canonical.PlayerMatchRow{
		"main": {mkPMRowTrio("main", t0, nil, nil, &acc, nil, nil)},
	}
	out := BuildAccuracyChart(rows)
	if len(out) != 1 {
		t.Fatalf("want 1 trace, got %d", len(out))
	}
	if out[0].Datapoints[0].Y != 42.0 {
		t.Errorf("accuracy want 42.0, got %f", out[0].Datapoints[0].Y)
	}
	if out[0].Key != "squad.contrib.accuracy.main" {
		t.Errorf("Key want squad.contrib.accuracy.main, got %s", out[0].Key)
	}
}

func TestBuildAvgLifeChart(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	life := 12.5
	rows := map[string][]canonical.PlayerMatchRow{
		"main": {mkPMRowTrio("main", t0, nil, nil, nil, &life, nil)},
	}
	out := BuildAvgLifeChart(rows)
	if len(out) != 1 {
		t.Fatalf("want 1 trace, got %d", len(out))
	}
	if out[0].Datapoints[0].Y != 12.5 {
		t.Errorf("avg_life want 12.5, got %f", out[0].Datapoints[0].Y)
	}
}

func TestBuildPerformanceChart_SortedChronologically(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	p1, p2, p3 := 70.0, 80.0, 90.0
	rows := map[string][]canonical.PlayerMatchRow{
		// Ordre non-chrono volontaire
		"main": {
			mkPMRowTrio("main", t0.Add(2*time.Hour), nil, nil, nil, nil, &p3),
			mkPMRowTrio("main", t0, nil, nil, nil, nil, &p1),
			mkPMRowTrio("main", t0.Add(time.Hour), nil, nil, nil, nil, &p2),
		},
	}
	out := BuildPerformanceChart(rows)
	if len(out) != 1 {
		t.Fatalf("want 1 trace, got %d", len(out))
	}
	if len(out[0].Datapoints) != 3 {
		t.Fatalf("want 3 datapoints, got %d", len(out[0].Datapoints))
	}
	// Tri chrono attendu
	wantY := []float64{70.0, 80.0, 90.0}
	for i, want := range wantY {
		if out[0].Datapoints[i].Y != want {
			t.Errorf("[%d] Y want %f, got %f", i, want, out[0].Datapoints[i].Y)
		}
	}
}

func TestBuildTrioCharts_EmptyInputs(t *testing.T) {
	t.Parallel()
	if got := BuildAssistsChart(nil); got != nil {
		t.Errorf("AssistsChart nil: want nil, got %v", got)
	}
	if got := BuildKDAChart(nil); got != nil {
		t.Errorf("KDAChart nil: want nil, got %v", got)
	}
	if got := BuildAccuracyChart(nil); got != nil {
		t.Errorf("AccuracyChart nil: want nil, got %v", got)
	}
	if got := BuildAvgLifeChart(nil); got != nil {
		t.Errorf("AvgLifeChart nil: want nil, got %v", got)
	}
	if got := BuildPerformanceChart(nil); got != nil {
		t.Errorf("PerformanceChart nil: want nil, got %v", got)
	}
}

func TestBuildTrioCharts_AllPlayersAllNil_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	rows := map[string][]canonical.PlayerMatchRow{
		"main": {mkPMRowTrio("main", t0, nil, nil, nil, nil, nil)},
	}
	if got := BuildKDAChart(rows); len(got) != 0 {
		t.Errorf("all-nil KDA: want 0 traces, got %d", len(got))
	}
}
