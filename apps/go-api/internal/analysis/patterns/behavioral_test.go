package patterns

import (
	"fmt"
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

// TestDetectAccuracyPlateau_HighAccuracy vérifie qu'une précision haute n'est pas signalée.
func TestDetectAccuracyPlateau_HighAccuracy(t *testing.T) {
	cfg := DefaultPatternConfig()
	var rows []MatchRow
	// 30 matchs avec précision à 55% > AccuracyPlateauMax (0.45)
	for i := 0; i < 30; i++ {
		rows = append(rows, MatchRow{Accuracy: 0.55 + float64(i%3)*0.002})
	}
	_, ok := detectAccuracyPlateau(rows, cfg)
	if ok {
		t.Error("accuracy plateau ne doit pas être détecté pour une précision haute (>= AccuracyPlateauMax)")
	}
}

// TestDetectTilt_NotEnoughLosses vérifie qu'une suite courte ne déclenche pas le tilt.
func TestDetectTilt_NotEnoughLosses(t *testing.T) {
	cfg := DefaultPatternConfig() // TiltLossRun = 3
	var rows []MatchRow
	for i := 0; i < 10; i++ {
		rows = append(rows, makeRowWithKDA(2, 2.0)) // WIN
	}
	// Seulement 2 défaites consécutives (< TiltLossRun=3)
	rows = append(rows, makeRowWithKDA(3, 0.5))
	rows = append(rows, makeRowWithKDA(3, 0.5))

	_, ok := detectTilt(rows, cfg)
	if ok {
		t.Error("tilt ne doit pas être détecté pour moins de TiltLossRun défaites consécutives")
	}
}

// TestDetectSessionFatigue_NoLongSessions vérifie qu'aucune session assez longue = pas de détection.
func TestDetectSessionFatigue_NoLongSessions(t *testing.T) {
	cfg := DefaultPatternConfig() // FatigueMinSession = 4
	now := time.Now().UTC()
	var rows []MatchRow
	// 3 sessions de 3 matchs chacune (< FatigueMinSession=4)
	for s := 0; s < 3; s++ {
		for i := 0; i < 3; i++ {
			rows = append(rows, MatchRow{
				SessionID: fmt.Sprintf("sess%d", s),
				KDA:       2.5 - float64(i)*0.5, // KDA décroissant dans chaque session
				Outcome:   2,
				PlayedAt:  now.Add(time.Duration(s*3+i) * time.Minute),
			})
		}
	}
	_, ok := detectSessionFatigue(rows, cfg)
	if ok {
		t.Error("session fatigue ne doit pas être détectée quand toutes les sessions sont trop courtes")
	}
}

// TestDetectPerfCeiling_FlatLowess vérifie la détection de plafond via LOWESS plate.
func TestDetectPerfCeiling_FlatLowess(t *testing.T) {
	cfg := DefaultPatternConfig()
	var rows []MatchRow
	// 25 matchs avec PerfScore stable autour de 70 (pente ≈ 0, max-meanTop < 5)
	for i := 0; i < 25; i++ {
		v := 70.0 + float64(i%3) // oscillation 70/71/72 → max=72, top10 mean ≈ 72
		rows = append(rows, MatchRow{PerfScore: ptr64(v)})
	}

	p, ok := detectPerfCeiling(rows, cfg)
	if !ok {
		t.Fatal("perf ceiling non détecté")
	}
	if p.Type != BehaviorPerfCeiling {
		t.Errorf("type = %q, want perf_ceiling", p.Type)
	}
	if p.Severity != SeverityMedium {
		t.Errorf("severity = %q, want Medium", p.Severity)
	}
}

// TestDetectPerfCeiling_NotEnoughRows vérifie qu'un faible historique ne déclenche pas.
func TestDetectPerfCeiling_NotEnoughRows(t *testing.T) {
	cfg := DefaultPatternConfig() // PerfCeilingMinRows = 20
	var rows []MatchRow
	// Seulement 15 matchs avec PerfScore (< PerfCeilingMinRows=20)
	for i := 0; i < 15; i++ {
		rows = append(rows, MatchRow{PerfScore: ptr64(70.0)})
	}
	_, ok := detectPerfCeiling(rows, cfg)
	if ok {
		t.Error("perf ceiling ne doit pas être détecté avec moins de PerfCeilingMinRows matchs")
	}
}
