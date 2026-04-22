package analysis_test

import (
	"testing"

	"levelup/go-api/internal/analysis"
)

func TestComputeTugOfWar_Empty(t *testing.T) {
	result := analysis.ComputeTugOfWar(nil, 60_000, 0)
	if result != nil {
		t.Errorf("attendu nil, obtenu %v", result)
	}
}

func TestComputeTugOfWar_ZeroDuration(t *testing.T) {
	events := []analysis.TugOfWarEvent{{TimeMS: 1000, IsAlly: true}}
	result := analysis.ComputeTugOfWar(events, 0, 0)
	if result != nil {
		t.Errorf("attendu nil avec durée=0, obtenu %v", result)
	}
}

func TestComputeTugOfWar_AllAllyKills(t *testing.T) {
	events := []analysis.TugOfWarEvent{
		{TimeMS: 5_000, IsAlly: true},
		{TimeMS: 20_000, IsAlly: true},
	}
	bins := analysis.ComputeTugOfWar(events, 60_000, 30_000)
	if len(bins) == 0 {
		t.Fatal("attendu au moins un bin")
	}
	// Les deux events sont dans le bin 0 (0–30s)
	if bins[0].Delta != 2 {
		t.Errorf("attendu Delta=2 dans bin 0, obtenu %d", bins[0].Delta)
	}
}

func TestComputeTugOfWar_MixedKills(t *testing.T) {
	events := []analysis.TugOfWarEvent{
		{TimeMS: 5_000, IsAlly: true},
		{TimeMS: 10_000, IsAlly: false},
		{TimeMS: 40_000, IsAlly: true},
	}
	bins := analysis.ComputeTugOfWar(events, 90_000, 30_000)
	if bins[0].Delta != 0 {
		t.Errorf("attendu Delta=0 dans bin 0 (1 ally - 1 enemy), obtenu %d", bins[0].Delta)
	}
	// bin 1 (30s–60s) : 1 ally → Delta=1, CumDelta=1
	if bins[1].Delta != 1 {
		t.Errorf("attendu Delta=1 dans bin 1, obtenu %d", bins[1].Delta)
	}
	if bins[1].CumDelta != 1 {
		t.Errorf("attendu CumDelta=1 dans bin 1, obtenu %d", bins[1].CumDelta)
	}
}

func TestComputeTugOfWar_CumDeltaProgresses(t *testing.T) {
	events := []analysis.TugOfWarEvent{
		{TimeMS: 5_000, IsAlly: true},
		{TimeMS: 35_000, IsAlly: true},
	}
	bins := analysis.ComputeTugOfWar(events, 90_000, 30_000)
	// bin0 : delta=1, cum=1 ; bin1 : delta=1, cum=2
	if bins[0].CumDelta != 1 {
		t.Errorf("bin 0 CumDelta attendu=1, obtenu %d", bins[0].CumDelta)
	}
	if bins[1].CumDelta != 2 {
		t.Errorf("bin 1 CumDelta attendu=2, obtenu %d", bins[1].CumDelta)
	}
}
