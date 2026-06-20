// Package analysis â€” synthesis_canonical_test.go : tests paritÃ© entre
// ComputeSynthesisKPIs (legacy legacymatch.SynthesisMatchRow) et
// ComputeSynthesisKPIsFromCanonical (P4, ADR 0011).
//
// Garde-fou : les 2 implÃ©mentations doivent produire des KPIs Ã©quivalents
// pour le mÃªme match. Permet la migration progressive sans changement de
// comportement observable.
package analysis

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
)

// intPtr/deref/float64Ptr dÃ©jÃ  dÃ©clarÃ©s ailleurs dans le package â€” utilisÃ©s tels quels.

// fixturePairForSynthesis construit la mÃªme donnÃ©e match dans les 2 formats.
func fixturePairForSynthesis() (legacymatch.SynthesisMatchRow, canonical.PlayerMatchRow) {
	startTime := time.Date(2026, 4, 29, 14, 0, 0, 0, time.UTC)
	kda := 3.17
	accuracy := 0.42
	perfScore := 75.5
	timePlayed := 600
	avgLife := 42.5
	sessionLabel := "session-1"

	domainRow := legacymatch.SynthesisMatchRow{
		MatchID:          "m-1",
		StartTime:        startTime,
		Outcome:          domain.OutcomeWin,
		Kills:            15,
		Deaths:           6,
		KDA:              &kda,
		IsWithFriends:    true,
		Accuracy:         &accuracy,
		TimePlayedSecs:   &timePlayed,
		AvgLifeSeconds:   &avgLife,
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
			Kills:          &kills,
			Deaths:         &deaths,
			KDA:            &kda,
			Accuracy:       &accuracy,
			TimePlayed:     &timePlayed,
			AvgLifeSeconds: &avgLife,
			Outcome:        canonical.OutcomeWin,
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
		d legacymatch.SynthesisMatchRow
		c canonical.PlayerMatchRow
	}, 5)
	for i := range pairs {
		pairs[i].d, pairs[i].c = fixturePairForSynthesis()
	}

	domainRows := make([]legacymatch.SynthesisMatchRow, len(pairs))
	canonicalRows := make([]canonical.PlayerMatchRow, len(pairs))
	for i, p := range pairs {
		domainRows[i] = p.d
		canonicalRows[i] = p.c
	}

	domainKPIs := ComputeSynthesisKPIs(domainRows, true)
	canonicalKPIs := ComputeSynthesisKPIsFromCanonical(canonicalRows, true, 225)

	// Comparaison : les KPIs doivent Ãªtre bit-identiques.
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
	got := ComputeSynthesisKPIsFromCanonical(nil, false, 225)
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
	domainRows := []legacymatch.SynthesisMatchRow{d1, d2}

	// isSquad=true â†’ seul c1 compte.
	gotSquad := ComputeSynthesisKPIsFromCanonical(canonicalRows, true, 225)
	wantSquad := ComputeSynthesisKPIs(domainRows, true)
	if gotSquad.MatchCount != wantSquad.MatchCount {
		t.Errorf("isSquad=true: got %d matches, want %d", gotSquad.MatchCount, wantSquad.MatchCount)
	}

	// isSquad=false â†’ seul c2 compte.
	gotSolo := ComputeSynthesisKPIsFromCanonical(canonicalRows, false, 225)
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

// derefFloat est une variante locale (le helper `deref` existe dÃ©jÃ  dans
// le package non-test pour des string pointers).
func derefFloat(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

// =============================================================================
// P4.3 (ADR 0011) : tests paritÃ© TopWeeks / Breakdown / TemporalHeatmap
// =============================================================================

// fixtureMixedSynthesisDataset crÃ©e un dataset variÃ© pour stresser les agrÃ©gations
// (plusieurs semaines, mix Win/Loss, KDA et Kills variables).
func fixtureMixedSynthesisDataset() ([]legacymatch.SynthesisMatchRow, []canonical.PlayerMatchRow) {
	base := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	specs := []struct {
		dayOffset, kills, deaths int
		kda                      float64
		outcome                  int
		canonicalOutcome         canonical.Outcome
		isWithFriends            bool
		hasKDA                   bool
	}{
		{0, 15, 5, 3.0, domain.OutcomeWin, canonical.OutcomeWin, true, true},
		{0, 8, 12, 0.83, domain.OutcomeLoss, canonical.OutcomeLoss, true, true},
		{1, 20, 3, 7.0, domain.OutcomeWin, canonical.OutcomeWin, true, true},
		{2, 10, 10, 1.5, domain.OutcomeWin, canonical.OutcomeWin, false, true},
		{8, 12, 8, 1.75, domain.OutcomeLoss, canonical.OutcomeLoss, true, true},
		{9, 5, 15, 0.5, domain.OutcomeLoss, canonical.OutcomeLoss, true, false},
	}
	domainRows := make([]legacymatch.SynthesisMatchRow, len(specs))
	canonicalRows := make([]canonical.PlayerMatchRow, len(specs))
	for i, s := range specs {
		st := base.AddDate(0, 0, s.dayOffset).Add(time.Duration(i) * time.Hour)
		var kdaPtr *float64
		if s.hasKDA {
			v := s.kda
			kdaPtr = &v
		}
		domainRows[i] = legacymatch.SynthesisMatchRow{
			MatchID:       fmtMatchID(i),
			StartTime:     st,
			Kills:         s.kills,
			Deaths:        s.deaths,
			KDA:           kdaPtr,
			Outcome:       s.outcome,
			IsWithFriends: s.isWithFriends,
		}
		k, d := s.kills, s.deaths
		canonicalRows[i] = canonical.PlayerMatchRow{
			Summary: canonical.MatchSummary{MatchID: fmtMatchID(i), StartedAtUTC: st},
			Self: canonical.MatchParticipant{
				Kills:   &k,
				Deaths:  &d,
				KDA:     kdaPtr,
				Outcome: s.canonicalOutcome,
			},
			Enrichment: canonical.PlayerMatchEnrichment{IsWithFriends: s.isWithFriends},
		}
	}
	return domainRows, canonicalRows
}

func fmtMatchID(i int) string {
	return "m-" + string(rune('0'+i))
}

// TestComputeSynthesisTopWeeksFromCanonical_RankBased vérifie le comportement
// rank-based de la variante canonique (rank=1 = "Top 1", pas win/loss outcome).
// La parité avec ComputeSynthesisTopWeeks n'est plus attendue : la version
// canonique compte les tops par RankInMatch=1 et trie chronologiquement.
func TestComputeSynthesisTopWeeksFromCanonical_RankBased(t *testing.T) {
	base := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC) // mercredi → semaine du 30/03
	rank1 := 1
	rank2 := 2
	rows := []canonical.PlayerMatchRow{
		{Summary: canonical.MatchSummary{StartedAtUTC: base}, Self: canonical.MatchParticipant{RankInMatch: &rank1}},
		{Summary: canonical.MatchSummary{StartedAtUTC: base.Add(time.Hour)}, Self: canonical.MatchParticipant{RankInMatch: &rank2}},
		{Summary: canonical.MatchSummary{StartedAtUTC: base.Add(2 * time.Hour)}, Self: canonical.MatchParticipant{RankInMatch: &rank1}},
		{Summary: canonical.MatchSummary{StartedAtUTC: base.Add(3 * time.Hour)}, Self: canonical.MatchParticipant{RankInMatch: &rank2}},
	}
	got := ComputeSynthesisTopWeeksFromCanonical(rows)
	if len(got) != 1 {
		t.Fatalf("expected 1 week, got %d", len(got))
	}
	w := got[0]
	if w.MatchCount != 4 {
		t.Errorf("MatchCount = %d, want 4", w.MatchCount)
	}
	if w.Wins != 2 {
		t.Errorf("Wins = %d, want 2 (rank=1 rows)", w.Wins)
	}
	// WinRate = 2/4 * 100 = 50%
	if w.WinRate != 50 {
		t.Errorf("WinRate = %v, want 50", w.WinRate)
	}
}

func TestComputeSynthesisBreakdownFromCanonical_ParityWithDomain(t *testing.T) {
	domainRows, canonicalRows := fixtureMixedSynthesisDataset()
	for _, isSquad := range []bool{true, false} {
		want := ComputeSynthesisBreakdown(domainRows, isSquad)
		got := ComputeSynthesisBreakdownFromCanonical(canonicalRows, isSquad)
		if want != got {
			t.Errorf("isSquad=%v: domain=%+v, canonical=%+v", isSquad, want, got)
		}
	}
}

func TestComputeTemporalHeatmapFromCanonical_ParityWithDomain(t *testing.T) {
	domainRows, canonicalRows := fixtureMixedSynthesisDataset()
	want := ComputeTemporalHeatmap(domainRows)
	got := ComputeTemporalHeatmapFromCanonical(canonicalRows)
	if len(want) != len(got) {
		t.Fatalf("len: domain=%d, canonical=%d", len(want), len(got))
	}
	for i := range want {
		if want[i] != got[i] {
			t.Errorf("cell %d: domain=%+v, canonical=%+v", i, want[i], got[i])
		}
	}
}
