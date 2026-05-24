package patterns

import (
	"testing"
	"time"
)

func ptr64(v float64) *float64 { return &v }

func makeRowWithKDA(outcome int, kda float64) MatchRow {
	return MatchRow{
		Outcome:  outcome,
		KDA:      kda,
		PlayedAt: time.Now().UTC(),
	}
}

// TestDetectTilt_FiveConsecutiveLosses vérifie la détection de tilt sur 5 défaites.
func TestDetectTilt_FiveConsecutiveLosses(t *testing.T) {
	cfg := DefaultPatternConfig()
	// 10 victoires avec KDA 2.0, puis 5 défaites avec KDA 1.0 (chute 50% > 25%)
	var rows []MatchRow
	for i := 0; i < 10; i++ {
		rows = append(rows, makeRowWithKDA(2, 2.0)) // WIN
	}
	for i := 0; i < 5; i++ {
		rows = append(rows, makeRowWithKDA(3, 1.0)) // LOSS
	}

	p, ok := detectTilt(rows, cfg)
	if !ok {
		t.Fatal("tilt non détecté")
	}
	if p.Severity != SeverityHigh {
		t.Errorf("severity = %q, want High (chute 50%% > 35%%)", p.Severity)
	}
	if p.Type != BehaviorTilt {
		t.Errorf("type = %q, want tilt", p.Type)
	}
}

// TestDetectSessionFatigue vérifie la détection sur une session de 6 matchs KDA décroissant.
func TestDetectSessionFatigue_DecreasingKDA(t *testing.T) {
	cfg := DefaultPatternConfig()
	// Session unique de 6 matchs avec KDA décroissant
	now := time.Now().UTC()
	var rows []MatchRow
	kdas := []float64{2.5, 2.2, 1.9, 1.6, 1.3, 1.0}
	for i, kda := range kdas {
		rows = append(rows, MatchRow{
			SessionID: "sess1",
			KDA:       kda,
			Outcome:   2,
			PlayedAt:  now.Add(time.Duration(i) * time.Minute),
		})
	}

	p, ok := detectSessionFatigue(rows, cfg)
	if !ok {
		t.Fatal("session fatigue non détectée")
	}
	if p.Type != BehaviorSessionFatigue {
		t.Errorf("type = %q, want session_fatigue", p.Type)
	}
}

// TestDetectEngagementDrop_BothMetricsLow vérifie la détection d'engagement drop.
// EngageScore ET ResidualBrut au P15 sur 8 matchs.
func TestDetectEngagementDrop_BothMetricsLow(t *testing.T) {
	// Créer 20 matchs "normaux" puis 8 matchs avec des valeurs très basses
	var rows []MatchRow
	for i := 0; i < 20; i++ {
		rows = append(rows, MatchRow{
			Outcome:      2,
			EngageScore:  ptr64(80.0 + float64(i)),
			ResidualBrut: ptr64(10.0 + float64(i)),
		})
	}
	// 8 matchs récents (début du slice = plus récents dans notre convention) très bas
	lowRows := make([]MatchRow, 8)
	for i := range lowRows {
		lowRows[i] = MatchRow{
			Outcome:      3,
			EngageScore:  ptr64(5.0),  // très bas
			ResidualBrut: ptr64(0.5),  // très bas
		}
	}
	// Les rows récents en premier
	rows = append(lowRows, rows...)

	p, ok := detectEngagementDrop(rows, DefaultPatternConfig())
	if !ok {
		t.Fatal("engagement drop non détecté")
	}
	if p.Type != BehaviorEngagementDrop {
		t.Errorf("type = %q, want engagement_drop", p.Type)
	}
	if p.Severity != SeverityMedium {
		t.Errorf("severity = %q, want Medium", p.Severity)
	}
}

// TestDetectEngagementDrop_OnlyEngageScore vérifie qu'EngageScore seul ne déclenche pas.
func TestDetectEngagementDrop_OnlyEngageScore(t *testing.T) {
	var rows []MatchRow
	// 30 matchs avec EngageScore bas mais ResidualBrut nil
	for i := 0; i < 30; i++ {
		rows = append(rows, MatchRow{
			Outcome:      3,
			EngageScore:  ptr64(5.0),
			ResidualBrut: nil, // pas de ResidualBrut
		})
	}

	_, ok := detectEngagementDrop(rows, DefaultPatternConfig())
	if ok {
		t.Error("engagement drop ne devrait pas être détecté sans ResidualBrut")
	}
}

// TestDetectAccuracyPlateau vérifie la détection de plateau de précision.
// Accuracy std < 0.02, mean < 0.40.
func TestDetectAccuracyPlateau_LowStableAccuracy(t *testing.T) {
	cfg := DefaultPatternConfig()
	var rows []MatchRow
	// 30 matchs avec précision stable à ~30% (σ ≈ 0.005 < 0.02)
	accuracies := make([]float64, 30)
	for i := range accuracies {
		// Légère variation autour de 0.30
		acc := 0.300 + float64(i%3)*0.003
		accuracies[i] = acc
		rows = append(rows, MatchRow{Accuracy: acc})
	}

	p, ok := detectAccuracyPlateau(rows, cfg)
	if !ok {
		t.Fatal("accuracy plateau non détecté")
	}
	if p.Type != BehaviorAccuracyPlateau {
		t.Errorf("type = %q, want accuracy_plateau", p.Type)
	}
	// mean ≈ 0.30 < 0.35 → pas Low, < 0.35 → Medium
	if p.Severity != SeverityMedium && p.Severity != SeverityHigh {
		t.Errorf("severity = %q, want Medium ou High pour mean < 0.35", p.Severity)
	}
}
