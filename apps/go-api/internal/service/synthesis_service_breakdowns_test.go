package service

import (
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// breakdownRow construit un match canonique pour carte `mapLabel` et mode `modeLabel`
// avec l'issue donnée ; perf <= 0 = aucun score de performance.
func breakdownRow(mapLabel, modeLabel string, outcome canonical.Outcome, perf float64) canonical.PlayerMatchRow {
	var row canonical.PlayerMatchRow
	row.Self.Outcome = outcome
	if mapLabel != "" {
		row.Summary.Map = &canonical.AssetReference{DefaultLabel: mapLabel}
	}
	if modeLabel != "" {
		row.Summary.PairMode = &canonical.AssetReference{DefaultLabel: modeLabel}
	}
	if perf > 0 {
		row.Enrichment.PerformanceScore = &perf
	}
	return row
}

func mapEntryByName(t *testing.T, entries []domain.SynthesisMapEntry, name string) domain.SynthesisMapEntry {
	t.Helper()
	for _, e := range entries {
		if e.MapName == name {
			return e
		}
	}
	t.Fatalf("carte %q absente de %+v", name, entries)
	return domain.SynthesisMapEntry{}
}

func modeEntryByName(t *testing.T, entries []domain.SynthesisModeEntry, name string) domain.SynthesisModeEntry {
	t.Helper()
	for _, e := range entries {
		if e.ModeName == name {
			return e
		}
	}
	t.Fatalf("mode %q absent de %+v", name, entries)
	return domain.SynthesisModeEntry{}
}

func TestBuildBreakdownsFromCanonical_PerformanceAvgOnlyScoredMatches(t *testing.T) {
	rows := []canonical.PlayerMatchRow{
		breakdownRow("Recharge", "Slayer", canonical.OutcomeWin, 80),
		breakdownRow("Recharge", "Slayer", canonical.OutcomeLoss, 60),
		// Non terminé, sans score : compte dans le total, pas dans la moyenne.
		breakdownRow("Recharge", "Slayer", canonical.OutcomeDNF, 0),
		// Autre carte, autre mode, un seul match noté.
		breakdownRow("Streets", "CTF", canonical.OutcomeTie, 72),
	}
	b := buildBreakdownsFromCanonical(rows, domain.GroupesDeCartes{})

	recharge := mapEntryByName(t, b.TopMaps, "Recharge")
	if recharge.PerformanceAvg == nil || *recharge.PerformanceAvg != 70 {
		t.Fatalf("Recharge PerformanceAvg = %v, want 70", recharge.PerformanceAvg)
	}
	if recharge.PerfTier != int(analysis.TierBon) {
		t.Errorf("Recharge PerfTier = %d, want %d", recharge.PerfTier, analysis.TierBon)
	}
	if recharge.MatchCount != 3 {
		t.Errorf("Recharge MatchCount = %d, want 3 (le non terminé compte)", recharge.MatchCount)
	}

	slayer := modeEntryByName(t, b.TopModes, "Slayer")
	if slayer.PerformanceAvg == nil || *slayer.PerformanceAvg != 70 {
		t.Fatalf("Slayer PerformanceAvg = %v, want 70", slayer.PerformanceAvg)
	}
	if slayer.PerfTier != int(analysis.TierBon) {
		t.Errorf("Slayer PerfTier = %d, want %d", slayer.PerfTier, analysis.TierBon)
	}

	ctf := modeEntryByName(t, b.TopModes, "CTF")
	if ctf.PerformanceAvg == nil || *ctf.PerformanceAvg != 72 {
		t.Fatalf("CTF PerformanceAvg = %v, want 72", ctf.PerformanceAvg)
	}
}

func TestBuildBreakdownsFromCanonical_NoScoreMeansNilAndTierZero(t *testing.T) {
	rows := []canonical.PlayerMatchRow{
		breakdownRow("Bazaar", "Slayer", canonical.OutcomeWin, 0),
		breakdownRow("Bazaar", "Slayer", canonical.OutcomeDNF, 0),
	}
	b := buildBreakdownsFromCanonical(rows, domain.GroupesDeCartes{})

	bazaar := mapEntryByName(t, b.TopMaps, "Bazaar")
	if bazaar.PerformanceAvg != nil || bazaar.PerfTier != 0 {
		t.Errorf("carte sans score : PerformanceAvg = %v, PerfTier = %d, want nil et 0", bazaar.PerformanceAvg, bazaar.PerfTier)
	}
	slayer := modeEntryByName(t, b.TopModes, "Slayer")
	if slayer.PerformanceAvg != nil || slayer.PerfTier != 0 {
		t.Errorf("mode sans score : PerformanceAvg = %v, PerfTier = %d, want nil et 0", slayer.PerformanceAvg, slayer.PerfTier)
	}
}

func TestBuildBreakdownsFromCanonical_PerfTierAtThresholds(t *testing.T) {
	cases := []struct {
		perf float64
		want analysis.Tier
	}{
		{80, analysis.TierExcellent},
		{79.9, analysis.TierBon},
		{65, analysis.TierBon},
		{64.9, analysis.TierCorrect},
		{50, analysis.TierCorrect},
		{49.9, analysis.TierFaible},
		{35, analysis.TierFaible},
		{34.9, analysis.TierMauvais},
	}
	for _, tc := range cases {
		b := buildBreakdownsFromCanonical([]canonical.PlayerMatchRow{
			breakdownRow("Aquarius", "Slayer", canonical.OutcomeWin, tc.perf),
		}, domain.GroupesDeCartes{})
		if got := mapEntryByName(t, b.TopMaps, "Aquarius").PerfTier; got != int(tc.want) {
			t.Errorf("carte, score %v : PerfTier = %d, want %d", tc.perf, got, tc.want)
		}
		if got := modeEntryByName(t, b.TopModes, "Slayer").PerfTier; got != int(tc.want) {
			t.Errorf("mode, score %v : PerfTier = %d, want %d", tc.perf, got, tc.want)
		}
	}
}

func TestBuildBreakdownsFromCanonical_OutcomesUnchanged(t *testing.T) {
	rows := []canonical.PlayerMatchRow{
		breakdownRow("Prism", "Slayer", canonical.OutcomeWin, 50),
		breakdownRow("Prism", "Slayer", canonical.OutcomeWin, 0),
		breakdownRow("Prism", "Slayer", canonical.OutcomeLoss, 0),
		breakdownRow("Prism", "Slayer", canonical.OutcomeTie, 0),
		breakdownRow("Prism", "Slayer", canonical.OutcomeDNF, 0),
	}
	b := buildBreakdownsFromCanonical(rows, domain.GroupesDeCartes{})

	check := func(label string, total, wins, losses, ties, unfinished int, winRate float64) {
		t.Helper()
		if total != 5 || wins != 2 || losses != 1 || ties != 1 || unfinished != 1 {
			t.Errorf("%s : total=%d wins=%d losses=%d ties=%d unfinished=%d, want 5/2/1/1/1", label, total, wins, losses, ties, unfinished)
		}
		if winRate != 0.4 {
			t.Errorf("%s : WinRate = %v, want 0.4", label, winRate)
		}
	}
	m := mapEntryByName(t, b.TopMaps, "Prism")
	check("carte", m.MatchCount, m.Wins, m.Losses, m.Ties, m.Unfinished, m.WinRate)
	d := modeEntryByName(t, b.TopModes, "Slayer")
	check("mode", d.MatchCount, d.Wins, d.Losses, d.Ties, d.Unfinished, d.WinRate)
}

func TestBuildBreakdownsFromCanonical_UnlabelledRowsAreSkippedAndEmptyIsNotNil(t *testing.T) {
	empty := buildBreakdownsFromCanonical(nil, domain.GroupesDeCartes{})
	if empty.TopMaps == nil || empty.TopModes == nil {
		t.Fatalf("liste vide : TopMaps=%v TopModes=%v, want des slices non nil", empty.TopMaps, empty.TopModes)
	}

	// Sans libellé de carte, le match n'entre que dans les modes (et inversement).
	b := buildBreakdownsFromCanonical([]canonical.PlayerMatchRow{
		breakdownRow("", "Slayer", canonical.OutcomeWin, 60),
		breakdownRow("Prism", "", canonical.OutcomeWin, 60),
	}, domain.GroupesDeCartes{})
	if len(b.TopMaps) != 1 || len(b.TopModes) != 1 {
		t.Fatalf("TopMaps=%d TopModes=%d, want 1 et 1", len(b.TopMaps), len(b.TopModes))
	}
}
