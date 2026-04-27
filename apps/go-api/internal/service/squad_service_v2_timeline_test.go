package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/games/canonical"
)

func mkTimelineRow(matchID string, ts time.Time, outcome canonical.Outcome, perf *float64) canonical.PlayerMatchRow {
	r := canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			MatchID:      matchID,
			StartedAtUTC: ts,
			Outcome:      outcome,
		},
		Self: canonical.MatchParticipant{Outcome: outcome},
	}
	if perf != nil {
		r.Enrichment.PerformanceScore = perf
	}
	return r
}

func fptrTL(v float64) *float64 { return &v }

func TestBuildTimelineMultiPlayer_Empty(t *testing.T) {
	t.Parallel()
	got := BuildTimelineMultiPlayer(nil)
	if got != nil {
		t.Errorf("nil input: want nil, got %v", got)
	}
}

func TestBuildTimelineMultiPlayer_TwoPlayersChrono(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	rowsByPlayer := map[string][]canonical.PlayerMatchRow{
		"main": {
			mkTimelineRow("m2", t0.Add(time.Hour), canonical.OutcomeWin, fptrTL(70)),
			mkTimelineRow("m1", t0, canonical.OutcomeLoss, fptrTL(50)),
		},
		"f1": {
			mkTimelineRow("m1", t0, canonical.OutcomeLoss, fptrTL(45)),
			mkTimelineRow("m2", t0.Add(time.Hour), canonical.OutcomeWin, fptrTL(65)),
		},
	}
	got := BuildTimelineMultiPlayer(rowsByPlayer)
	if len(got) != 2 {
		t.Fatalf("want 2 series (1 par joueur), got %d", len(got))
	}
	// Ordre alphabetique des series : f1, main
	if got[0].Meta["gamertag"] != "f1" {
		t.Errorf("first series should be f1, got %v", got[0].Meta["gamertag"])
	}
	// Tri chrono ASC dans chaque serie : m1 puis m2
	for _, serie := range got {
		if !serie.Datapoints[0].X.(time.Time).Before(serie.Datapoints[1].X.(time.Time)) {
			t.Errorf("series %s not in chrono ASC order", serie.Key)
		}
	}
	// Outcome embarque dans Label
	if got[0].Datapoints[0].Label == nil || *got[0].Datapoints[0].Label != "loss" {
		t.Errorf("first dp label want 'loss', got %v", got[0].Datapoints[0].Label)
	}
}

func TestBuildTimelineMultiPlayer_SkipsRowsWithoutPerfScore(t *testing.T) {
	t.Parallel()
	t0 := time.Now()
	rowsByPlayer := map[string][]canonical.PlayerMatchRow{
		"main": {
			mkTimelineRow("m1", t0, canonical.OutcomeWin, nil),         // skip
			mkTimelineRow("m2", t0, canonical.OutcomeLoss, fptrTL(50)), // ok
		},
	}
	got := BuildTimelineMultiPlayer(rowsByPlayer)
	if len(got) != 1 || len(got[0].Datapoints) != 1 {
		t.Errorf("want 1 series with 1 dp, got %v", got)
	}
}

func TestBuildTimelineMultiPlayer_PlayerWithNoPerfOmitted(t *testing.T) {
	t.Parallel()
	rowsByPlayer := map[string][]canonical.PlayerMatchRow{
		"main": {mkTimelineRow("m1", time.Now(), canonical.OutcomeWin, fptrTL(50))},
		"f1":   {mkTimelineRow("m1", time.Now(), canonical.OutcomeWin, nil)},
	}
	got := BuildTimelineMultiPlayer(rowsByPlayer)
	if len(got) != 1 {
		t.Errorf("player without any perf should be omitted, got %d series", len(got))
	}
	if got[0].Meta["gamertag"] != "main" {
		t.Errorf("only main should remain, got %v", got[0].Meta["gamertag"])
	}
}

func TestBuildFormScore_Empty(t *testing.T) {
	t.Parallel()
	got := BuildFormScore(nil, 0.3)
	if got.Datapoints != nil {
		t.Errorf("nil input: want no dp, got %v", got.Datapoints)
	}
	if got.Key != "squad.synergies.form_score" {
		t.Errorf("Key: %q", got.Key)
	}
}

func TestBuildFormScore_LinearSeriesPreserved(t *testing.T) {
	t.Parallel()
	// Serie lineaire : LOWESS doit la preserver (a tolerance pres).
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var rows []canonical.PlayerMatchRow
	for i := 0; i < 30; i++ {
		val := float64(50 + i)
		rows = append(rows, mkTimelineRow("m", t0.Add(time.Duration(i)*time.Hour),
			canonical.OutcomeWin, &val))
	}
	got := BuildFormScore(rows, 0.3)
	if len(got.Datapoints) != 30 {
		t.Errorf("want 30 dp, got %d", len(got.Datapoints))
	}
	// Au centre, le lissage doit etre tres proche de la valeur originale.
	mid := got.Datapoints[15]
	expected := 50.0 + 15.0
	if abs(mid.Y-expected) > 1.0 {
		t.Errorf("linear preserved at center: want ~%v, got %v", expected, mid.Y)
	}
}

func TestBuildFormScore_SkipsRowsWithoutPerf(t *testing.T) {
	t.Parallel()
	t0 := time.Now()
	rows := []canonical.PlayerMatchRow{
		mkTimelineRow("m1", t0, canonical.OutcomeWin, nil),
		mkTimelineRow("m2", t0.Add(time.Hour), canonical.OutcomeWin, fptrTL(50)),
		mkTimelineRow("m3", t0.Add(2*time.Hour), canonical.OutcomeWin, fptrTL(60)),
		mkTimelineRow("m4", t0.Add(3*time.Hour), canonical.OutcomeWin, fptrTL(55)),
	}
	got := BuildFormScore(rows, 0.5)
	// 3 rows valides, doit produire <= 3 dp (selon LOWESS minPoints).
	if len(got.Datapoints) > 3 {
		t.Errorf("want <= 3 dp (skipped m1), got %d", len(got.Datapoints))
	}
}

func TestBuildFormScore_AlphaDefault(t *testing.T) {
	t.Parallel()
	t0 := time.Now()
	rows := []canonical.PlayerMatchRow{
		mkTimelineRow("m1", t0, canonical.OutcomeWin, fptrTL(50)),
		mkTimelineRow("m2", t0.Add(time.Hour), canonical.OutcomeWin, fptrTL(60)),
		mkTimelineRow("m3", t0.Add(2*time.Hour), canonical.OutcomeWin, fptrTL(70)),
	}
	got := BuildFormScore(rows, 0) // 0 -> default 0.3
	if got.Meta["alpha"] != 0.3 {
		t.Errorf("alpha default 0.3 expected, got %v", got.Meta["alpha"])
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
