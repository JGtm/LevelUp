// Package analysis — home_canonical_test.go : tests parité entre les wrappers
// `*FromCanonical` et leurs versions legacy (P4.3b, ADR 0011).
//
// La conversion canonical → HomeMatchRow est encapsulée dans le wrapper. Les
// tests vérifient que appeler le wrapper sur des canonical rows produit le
// même résultat que appeler la version legacy directement sur les
// HomeMatchRow équivalents.
package analysis

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// fixturePairForHome construit la même donnée match dans les 2 formats.
func fixturePairForHome(matchID string, kills, deaths, assists int, outcome int, isRanked, isWithFriends bool) (domain.HomeMatchRow, canonical.PlayerMatchRow) {
	startTime := time.Date(2026, 4, 29, 14, 0, 0, 0, time.UTC)
	kda := float64(kills+assists) / float64(maxInt(deaths, 1))
	accuracy := 0.42
	timePlayed := 600
	perfScore := 75.5
	ratio := float64(kills) / float64(maxInt(deaths, 1))

	canonicalOutcome := canonical.OutcomeWin
	switch outcome {
	case domain.OutcomeLoss:
		canonicalOutcome = canonical.OutcomeLoss
	case domain.OutcomeDraw:
		canonicalOutcome = canonical.OutcomeTie
	case domain.OutcomeDNF:
		canonicalOutcome = canonical.OutcomeDNF
	}

	domainRow := domain.HomeMatchRow{
		MatchID:          matchID,
		StartTime:        startTime,
		Outcome:          outcome,
		Kills:            kills,
		Deaths:           deaths,
		Assists:          assists,
		KDA:              &kda,
		Ratio:            &ratio,
		Accuracy:         &accuracy,
		TimePlayedSecs:   &timePlayed,
		PerformanceScore: &perfScore,
		IsRanked:         isRanked,
		IsFirefight:      false,
		IsWithFriends:    isWithFriends,
	}

	canonicalRow := canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			MatchID:      matchID,
			StartedAtUTC: startTime,
			IsRanked:     boolPtr(isRanked),
			IsPvE:        boolPtr(false),
			Outcome:      canonicalOutcome,
		},
		Self: canonical.MatchParticipant{
			Kills:      &kills,
			Deaths:     &deaths,
			Assists:    &assists,
			KDA:        &kda,
			Accuracy:   &accuracy,
			TimePlayed: &timePlayed,
			Outcome:    canonicalOutcome,
		},
		Enrichment: canonical.PlayerMatchEnrichment{
			IsWithFriends:    isWithFriends,
			PerformanceScore: &perfScore,
		},
	}
	return domainRow, canonicalRow
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// TestHomeMatchRowFromCanonical_RoundtripFields garantit que les champs clés
// (kills/deaths/outcome/ratio/KDA) survivent à la conversion canonical → HomeMatchRow.
func TestHomeMatchRowFromCanonical_RoundtripFields(t *testing.T) {
	_, canonicalRow := fixturePairForHome("m-1", 15, 6, 3, domain.OutcomeWin, true, true)
	out := HomeMatchRowFromCanonical(canonicalRow)

	if out.MatchID != "m-1" {
		t.Errorf("MatchID: got %q, want m-1", out.MatchID)
	}
	if out.Kills != 15 || out.Deaths != 6 || out.Assists != 3 {
		t.Errorf("K/D/A: got %d/%d/%d, want 15/6/3", out.Kills, out.Deaths, out.Assists)
	}
	if out.Outcome != domain.OutcomeWin {
		t.Errorf("Outcome: got %d, want %d", out.Outcome, domain.OutcomeWin)
	}
	if !out.IsRanked || out.IsFirefight || !out.IsWithFriends {
		t.Errorf("flags: ranked=%v firefight=%v friends=%v", out.IsRanked, out.IsFirefight, out.IsWithFriends)
	}
	if out.Ratio == nil || *out.Ratio != 15.0/6.0 {
		t.Errorf("Ratio: got %v, want 2.5", out.Ratio)
	}
}

// TestInferHomeSkillHistoryFromCanonical_ParityWithLocal vérifie que la
// version canonical produit le même résultat que la version locale legacy.
func TestInferHomeSkillHistoryFromCanonical_ParityWithLocal(t *testing.T) {
	dataset := []struct {
		isRanked, isPvE, isWithFriends bool
	}{
		{true, false, true},
		{false, false, false},
		{false, true, false}, // PvE doit être exclu
		{true, false, false},
	}
	domainRows := make([]domain.HomeMatchRow, 0, len(dataset))
	canonicalRows := make([]canonical.PlayerMatchRow, 0, len(dataset))
	for i, d := range dataset {
		dr := domain.HomeMatchRow{
			MatchID:     "m-" + string(rune('0'+i)),
			IsRanked:    d.isRanked,
			IsFirefight: d.isPvE,
		}
		domainRows = append(domainRows, dr)
		canonicalRows = append(canonicalRows, canonical.PlayerMatchRow{
			Summary: canonical.MatchSummary{
				MatchID:  "m-" + string(rune('0'+i)),
				IsRanked: boolPtr(d.isRanked),
				IsPvE:    boolPtr(d.isPvE),
			},
		})
	}

	gotR, gotU := InferHomeSkillHistoryFromCanonical(canonicalRows)
	// Replique la logique legacy de inferHomeSkillHistory.
	wantR, wantU := false, false
	for _, m := range domainRows {
		if m.IsFirefight {
			continue
		}
		if m.IsRanked {
			wantR = true
		} else {
			wantU = true
		}
	}

	if gotR != wantR || gotU != wantU {
		t.Errorf("InferHomeSkillHistoryFromCanonical = (%v,%v), want (%v,%v)", gotR, gotU, wantR, wantU)
	}
}
