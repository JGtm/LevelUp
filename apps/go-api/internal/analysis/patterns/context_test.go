package patterns

import (
	"testing"
	"time"
)

func baseTime() time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}

func makeRow(mode, mapID string, outcome int, kda, oc, dr float64, isWithFriends bool) MatchRow {
	return MatchRow{
		Mode:          mode,
		MapID:         mapID,
		Outcome:       outcome,
		KDA:           kda,
		OC:            oc,
		DR:            dr,
		IsWithFriends: isWithFriends,
		PlayedAt:      baseTime(),
	}
}

// TestAnalyze_EmptyInput vérifie qu'une entrée vide retourne un PatternReport vide.
func TestAnalyze_EmptyInput(t *testing.T) {
	report := Analyze(AnalyzeInput{
		Rows:   nil,
		Config: DefaultPatternConfig(),
		Now:    baseTime(),
	})
	if report.WindowSize != 0 {
		t.Errorf("window_size = %d, want 0", report.WindowSize)
	}
	if len(report.ContextPatterns) != 0 {
		t.Errorf("context_patterns non vide sur entrée vide")
	}
	if len(report.BehaviorPatterns) != 0 {
		t.Errorf("behavior_patterns non vide sur entrée vide")
	}
	if len(report.Levers) != 0 {
		t.Errorf("levers non vide sur entrée vide")
	}
}

// TestAnalyze_TooFewMatchesPerMode vérifie qu'un groupe < 5 matchs n'émet pas de pattern.
func TestAnalyze_TooFewMatchesPerMode(t *testing.T) {
	var rows []MatchRow
	// 4 matchs en Slayer — en dessous du seuil MinMatchesPerGroup = 5
	for i := 0; i < 4; i++ {
		rows = append(rows, makeRow("Slayer", "map1", 2, 1.5, 1.0, 1.0, false))
	}
	report := Analyze(AnalyzeInput{
		Rows:   rows,
		Config: DefaultPatternConfig(),
		Now:    baseTime(),
	})
	// Aucun pattern contextuel ne doit être émis
	for _, cp := range report.ContextPatterns {
		if cp.Type == ContextByMode && cp.Key == "Slayer" {
			t.Errorf("pattern émis pour groupe < MinMatchesPerGroup")
		}
	}
}

// TestAnalyze_StrengthAndWeakness vérifie la détection de force/faiblesse.
// 30 Slayer winRate=0.70 (strength) + 15 CTF winRate=0.30 (weakness).
func TestAnalyze_StrengthAndWeakness(t *testing.T) {
	var rows []MatchRow
	// Slayer : 21 wins, 9 losses = 70%
	for i := 0; i < 21; i++ {
		rows = append(rows, makeRow("Slayer", "map1", 2, 1.5, 1.3, 1.0, false))
	}
	for i := 0; i < 9; i++ {
		rows = append(rows, makeRow("Slayer", "map1", 3, 1.0, 1.0, 0.9, false))
	}
	// CTF : 4 wins, 11 losses ~27%
	for i := 0; i < 4; i++ {
		rows = append(rows, makeRow("CTF", "map2", 2, 0.8, 0.7, 0.8, false))
	}
	for i := 0; i < 11; i++ {
		rows = append(rows, makeRow("CTF", "map2", 3, 0.5, 0.6, 0.7, false))
	}
	// WR global = (21+4)/(30+15) = 25/45 ≈ 0.556
	// Slayer WR = 0.70, delta = 0.144 > 0.12 → Strength (count=30 >= 10)
	// CTF WR ≈ 0.267, delta = -0.289 < -0.12 → Weakness

	report := Analyze(AnalyzeInput{
		Rows:   rows,
		Config: DefaultPatternConfig(),
		Now:    baseTime(),
	})

	slayerFound, ctfFound := false, false
	for _, cp := range report.ContextPatterns {
		if cp.Type != ContextByMode {
			continue
		}
		if cp.Key == "Slayer" {
			slayerFound = true
			if cp.Signal != SignalStrength {
				t.Errorf("Slayer signal = %q, want Strength", cp.Signal)
			}
		}
		if cp.Key == "CTF" {
			ctfFound = true
			if cp.Signal != SignalWeakness {
				t.Errorf("CTF signal = %q, want Weakness", cp.Signal)
			}
		}
	}
	if !slayerFound {
		t.Error("pattern Slayer absent")
	}
	if !ctfFound {
		t.Error("pattern CTF absent")
	}
}

// TestAnalyze_SquadStrength vérifie la détection de force en squad.
// squad wr=0.65 (15 matchs) + solo wr=0.40 (15 matchs).
func TestAnalyze_SquadStrength(t *testing.T) {
	var rows []MatchRow
	// squad : 10 wins / 5 losses = 66.7%
	for i := 0; i < 10; i++ {
		rows = append(rows, makeRow("Slayer", "map1", 2, 1.5, 1.2, 1.0, true))
	}
	for i := 0; i < 5; i++ {
		rows = append(rows, makeRow("Slayer", "map1", 3, 1.0, 1.0, 0.9, true))
	}
	// solo : 6 wins / 9 losses = 40%
	for i := 0; i < 6; i++ {
		rows = append(rows, makeRow("Slayer", "map1", 2, 1.2, 1.1, 1.0, false))
	}
	for i := 0; i < 9; i++ {
		rows = append(rows, makeRow("Slayer", "map1", 3, 0.8, 0.9, 0.8, false))
	}
	// globalWR = 16/30 ≈ 0.533
	// squad delta ≈ 0.667 - 0.533 = 0.134 > 0.12, count=15 >= 10 → Strength
	// solo delta = 0.40 - 0.533 = -0.133 < -0.12 → Weakness

	report := Analyze(AnalyzeInput{
		Rows:   rows,
		Config: DefaultPatternConfig(),
		Now:    baseTime(),
	})

	squadFound := false
	for _, cp := range report.ContextPatterns {
		if cp.Type == ContextBySquad && cp.Key == "with_friends" {
			squadFound = true
			if cp.Signal != SignalStrength {
				t.Errorf("with_friends signal = %q, want Strength", cp.Signal)
			}
		}
	}
	if !squadFound {
		t.Error("pattern with_friends absent")
	}
}

// TestAnalyze_DeltaCSRNilWhenNoCSR vérifie que AvgDeltaCSR est nil si aucun row n'a DeltaCSR.
func TestAnalyze_DeltaCSRNilWhenNoCSR(t *testing.T) {
	var rows []MatchRow
	// 15 matchs sans DeltaCSR
	for i := 0; i < 15; i++ {
		r := makeRow("Slayer", "map1", 2, 1.5, 1.2, 1.0, false)
		r.DeltaCSR = nil
		rows = append(rows, r)
	}
	report := Analyze(AnalyzeInput{
		Rows:   rows,
		Config: DefaultPatternConfig(),
		Now:    baseTime(),
	})
	for _, cp := range report.ContextPatterns {
		if cp.AvgDeltaCSR != nil {
			t.Errorf("AvgDeltaCSR devrait être nil quand aucune row n'a DeltaCSR")
		}
	}
}
