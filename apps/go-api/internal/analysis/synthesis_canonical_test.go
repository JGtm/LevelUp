// Package analysis — synthesis_canonical_test.go : tests parité entre
// ComputeSynthesisKPIs (legacy domain.SynthesisMatchRow) et
// ComputeSynthesisKPIsFromCanonical (P4, ADR 0011).
//
// Garde-fou : les 2 implémentations doivent produire des KPIs équivalents
// pour le même match. Permet la migration progressive sans changement de
// comportement observable.
package analysis

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// intPtr/deref/float64Ptr déjà déclarés ailleurs dans le package — utilisés tels quels.

// fixturePairForSynthesis construit la même donnée match dans les 2 formats.
func fixturePairForSynthesis() (domain.SynthesisMatchRow, canonical.PlayerMatchRow) {
	startTime := time.Date(2026, 4, 29, 14, 0, 0, 0, time.UTC)
	kda := 3.17
	accuracy := 0.42
	perfScore := 75.5
	timePlayed := 600
	sessionLabel := "session-1"

	domainRow := domain.SynthesisMatchRow{
		MatchID:          "m-1",
		StartTime:        startTime,
		Outcome:          domain.OutcomeWin,
		Kills:            15,
		Deaths:           6,
		KDA:              &kda,
		IsWithFriends:    true,
		Accuracy:         &accuracy,
		TimePlayedSecs:   &timePlayed,
		PerformanceScore: &perfScore,
		SessionLabel:     &sessionLabel,
		IsRanked:         true,
		IsFirefight:      false,
		PlaylistName:     "Ranked Arena",
	}

	kills := 15
	deaths := 6
	canonicalRow := canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			MatchID:      "m-1",
			StartedAtUTC: startTime,
			IsRanked:     boolPtr(true),
			IsPvE:        boolPtr(false),
			Outcome:      canonical.OutcomeWin,
			Playlist:     &canonical.AssetReference{DefaultLabel: "Ranked Arena"},
		},
		Self: canonical.MatchParticipant{
			Kills:      &kills,
			Deaths:     &deaths,
			KDA:        &kda,
			Accuracy:   &accuracy,
			TimePlayed: &timePlayed,
			Outcome:    canonical.OutcomeWin,
		},
		Enrichment: canonical.PlayerMatchEnrichment{
			IsWithFriends:    true,
			PerformanceScore: &perfScore,
			SessionLabel:     &sessionLabel,
		},
	}

	return domainRow, canonicalRow
}

func boolPtr(v bool) *bool { return &v }

func TestComputeSynthesisKPIsFromCanonical_ParityWithDomain(t *testing.T) {
	// Construit un dataset identique dans les 2 formats.
	pairs := make([]struct {
		d domain.SynthesisMatchRow
		c canonical.PlayerMatchRow
	}, 5)
	for i := range pairs {
		pairs[i].d, pairs[i].c = fixturePairForSynthesis()
	}

	domainRows := make([]domain.SynthesisMatchRow, len(pairs))
	canonicalRows := make([]canonical.PlayerMatchRow, len(pairs))
	for i, p := range pairs {
		domainRows[i] = p.d
		canonicalRows[i] = p.c
	}

	domainKPIs := ComputeSynthesisKPIs(domainRows, true)
	canonicalKPIs := ComputeSynthesisKPIsFromCanonical(canonicalRows, true)

	// Comparaison : les KPIs doivent être bit-identiques.
	if domainKPIs.MatchCount != canonicalKPIs.MatchCount {
		t.Errorf("MatchCount: domain=%d, canonical=%d", domainKPIs.MatchCount, canonicalKPIs.MatchCount)
	}
	if domainKPIs.Wins != canonicalKPIs.Wins {
		t.Errorf("Wins: domain=%d, canonical=%d", domainKPIs.Wins, canonicalKPIs.Wins)
	}
	if domainKPIs.WinRate != canonicalKPIs.WinRate {
		t.Errorf("WinRate: domain=%v, canonical=%v", domainKPIs.WinRate, canonicalKPIs.WinRate)
	}
	if !floatPtrEqual(domainKPIs.KDRatio, canonicalKPIs.KDRatio) {
		t.Errorf("KDRatio: domain=%v, canonical=%v", derefFloat(domainKPIs.KDRatio), derefFloat(canonicalKPIs.KDRatio))
	}
	if !floatPtrEqual(domainKPIs.Accuracy, canonicalKPIs.Accuracy) {
		t.Errorf("Accuracy: domain=%v, canonical=%v", derefFloat(domainKPIs.Accuracy), derefFloat(canonicalKPIs.Accuracy))
	}
	if !floatPtrEqual(domainKPIs.PerformanceScore, canonicalKPIs.PerformanceScore) {
		t.Errorf("PerformanceScore: domain=%v, canonical=%v", derefFloat(domainKPIs.PerformanceScore), derefFloat(canonicalKPIs.PerformanceScore))
	}
	if !floatPtrEqual(domainKPIs.AvgLifeSeconds, canonicalKPIs.AvgLifeSeconds) {
		t.Errorf("AvgLifeSeconds: domain=%v, canonical=%v", derefFloat(domainKPIs.AvgLifeSeconds), derefFloat(canonicalKPIs.AvgLifeSeconds))
	}
	if !floatPtrEqual(domainKPIs.KillsPerMin, canonicalKPIs.KillsPerMin) {
		t.Errorf("KillsPerMin: domain=%v, canonical=%v", derefFloat(domainKPIs.KillsPerMin), derefFloat(canonicalKPIs.KillsPerMin))
	}
}

func TestComputeSynthesisKPIsFromCanonical_EmptyRows(t *testing.T) {
	got := ComputeSynthesisKPIsFromCanonical(nil, false)
	if got.MatchCount != 0 {
		t.Errorf("MatchCount = %d, want 0 sur dataset vide", got.MatchCount)
	}
}

func TestComputeSynthesisKPIsFromCanonical_FilterIsSquad(t *testing.T) {
	d1, c1 := fixturePairForSynthesis()
	d2, c2 := fixturePairForSynthesis()
	d2.IsWithFriends = false
	c2.Enrichment.IsWithFriends = false

	canonicalRows := []canonical.PlayerMatchRow{c1, c2}
	domainRows := []domain.SynthesisMatchRow{d1, d2}

	// isSquad=true → seul c1 compte.
	gotSquad := ComputeSynthesisKPIsFromCanonical(canonicalRows, true)
	wantSquad := ComputeSynthesisKPIs(domainRows, true)
	if gotSquad.MatchCount != wantSquad.MatchCount {
		t.Errorf("isSquad=true: got %d matches, want %d", gotSquad.MatchCount, wantSquad.MatchCount)
	}

	// isSquad=false → seul c2 compte.
	gotSolo := ComputeSynthesisKPIsFromCanonical(canonicalRows, false)
	wantSolo := ComputeSynthesisKPIs(domainRows, false)
	if gotSolo.MatchCount != wantSolo.MatchCount {
		t.Errorf("isSquad=false: got %d matches, want %d", gotSolo.MatchCount, wantSolo.MatchCount)
	}
}

func floatPtrEqual(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// derefFloat est une variante locale (le helper `deref` existe déjà dans
// le package non-test pour des string pointers).
func derefFloat(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}
