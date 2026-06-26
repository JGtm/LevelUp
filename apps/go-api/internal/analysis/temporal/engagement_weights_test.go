package temporal

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
)

func TestEngagementEventWeight(t *testing.T) {
	cases := map[string]float64{
		"mode":        1.5, // objectif → prime meneur
		"assist":      0.5, // support
		"death":       0.4, // subi
		"kill":        1.0,
		"medal":       1.0, // additif (intensité)
		"first_kill":  1.0,
		"first_death": 1.0,
		"finisher":    1.0,
		"":            1.0,
		"unknown_xyz": 1.0,
	}
	for et, want := range cases {
		if got := engagementEventWeight(et); got != want {
			t.Errorf("engagementEventWeight(%q) = %v, want %v", et, got, want)
		}
	}
}

func TestExtractWeightedPoints_AndSum(t *testing.T) {
	events := []canonical.HighlightEvent{
		{EventType: "kill", TimeMS: 100},
		{EventType: "death", TimeMS: 200},
		{EventType: "mode", TimeMS: 300},
		{EventType: "assist", TimeMS: 400},
		{EventType: "kill", TimeMS: 50_000}, // hors fenêtre
	}
	pts := extractWeightedPoints(events)
	if len(pts) != len(events) {
		t.Fatalf("points = %d, attendu %d", len(pts), len(events))
	}
	// Fenêtre [0, 1000] : kill(1.0)+death(0.4)+mode(1.5)+assist(0.5) = 3.4 ; le kill à
	// 50s est exclu. Un comptage NON pondéré aurait donné 4.0 → la pondération agit.
	if got := sumWeightInWindow(pts, 0, 1000); got != 3.4 {
		t.Errorf("sumWeightInWindow pondéré = %v, want 3.4 (1.0+0.4+1.5+0.5)", got)
	}
}
