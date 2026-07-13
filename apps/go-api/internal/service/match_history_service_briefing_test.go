package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// briefingRaw construit une raw row minimale pour les tests du briefing.
func briefingRaw(id string, daysAgo, outcome, kills, deaths, assists int, perf float64, mapID, mapFR, pairFR, playlist string) domain.MatchHistoryRawRow {
	t := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC).AddDate(0, 0, -daysAgo)
	p := perf
	r := domain.MatchHistoryRawRow{
		MatchID:          id,
		StartTime:        &t,
		Outcome:          outcome,
		Kills:            kills,
		Deaths:           deaths,
		Assists:          assists,
		PerformanceScore: &p,
	}
	if mapID != "" {
		r.MapID = &mapID
	}
	if mapFR != "" {
		r.MapNameFR = &mapFR
	}
	if pairFR != "" {
		r.PairNameFR = &pairFR
	}
	if playlist != "" {
		r.PlaylistName = &playlist
	}
	return r
}

func svcWithRanked(ranked bool) *MatchHistoryService {
	return &MatchHistoryService{rankedCapable: ranked}
}

func TestBuildExplorerBriefing_LowSample(t *testing.T) {
	var filtered []domain.MatchHistoryRawRow
	for i := 0; i < 8; i++ {
		filtered = append(filtered, briefingRaw("m"+string(rune('a'+i)), i, domain.OutcomeWin, 10, 5, 2, 60, "map1", "Aquarius", "Slayer", "Arène classée"))
	}
	kpis := &domain.KPIStats{MatchesCount: 8}
	b := svcWithRanked(true).buildExplorerBriefing(filtered, filtered, kpis)
	if b == nil {
		t.Fatal("briefing nil")
	}
	if !b.LowSample {
		t.Error("LowSample should be true for 8 < 10")
	}
	if b.KPIs == nil {
		t.Error("socle KPIs should be served")
	}
	if len(b.OutcomeSequence) != 8 {
		t.Errorf("outcome sequence len = %d, want 8", len(b.OutcomeSequence))
	}
	if b.PeriodStart == nil || b.PeriodEnd == nil {
		t.Error("period should be set")
	}
	if b.Baseline != nil || b.Dimensions != nil || b.Trend != nil || b.Ranked != nil {
		t.Error("low sample: modules must be nil")
	}
}

func TestBuildExplorerBriefing_EmptyScope(t *testing.T) {
	if got := svcWithRanked(false).buildExplorerBriefing(nil, nil, nil); got != nil {
		t.Errorf("empty scope -> nil briefing, got %+v", got)
	}
}

func TestBuildExplorerBriefing_SingleValueDimensionNotEmitted(t *testing.T) {
	// 24 matchs, TOUS sur la même carte (map1) mais 2 modes distincts.
	var filtered []domain.MatchHistoryRawRow
	for i := 0; i < 24; i++ {
		mode := "Slayer"
		if i%2 == 0 {
			mode = "CTF"
		}
		filtered = append(filtered, briefingRaw("m"+string(rune('a'+i)), i, domain.OutcomeWin, 10, 5, 2, 60, "map1", "Aquarius", mode, "Arène"))
	}
	kpis := &domain.KPIStats{MatchesCount: 24}
	b := svcWithRanked(false).buildExplorerBriefing(filtered, filtered, kpis)
	if b == nil {
		t.Fatal("nil briefing")
	}
	for _, d := range b.Dimensions {
		if d.Dimension == "map" {
			t.Error("map dimension must NOT be emitted (single distinct value)")
		}
	}
	// mode a 2 valeurs distinctes -> émis.
	var hasMode bool
	for _, d := range b.Dimensions {
		if d.Dimension == "mode" {
			hasMode = true
		}
	}
	if !hasMode {
		t.Error("mode dimension (2 distinct values) should be emitted")
	}
}

func TestBuildExplorerBriefing_SubThresholdGroupExcluded(t *testing.T) {
	// map1 : 12 matchs (qualifié) ; map2 : 5 matchs (< 10, exclu du top/flop).
	var filtered []domain.MatchHistoryRawRow
	for i := 0; i < 12; i++ {
		filtered = append(filtered, briefingRaw("a"+string(rune('a'+i)), i, domain.OutcomeWin, 10, 5, 2, 70, "map1", "Aquarius", "Slayer", "Arène"))
	}
	for i := 0; i < 5; i++ {
		filtered = append(filtered, briefingRaw("b"+string(rune('a'+i)), 20+i, domain.OutcomeLoss, 5, 10, 1, 40, "map2", "Bazaar", "Slayer", "Arène"))
	}
	kpis := &domain.KPIStats{MatchesCount: 17}
	b := svcWithRanked(false).buildExplorerBriefing(filtered, filtered, kpis)
	var mapDim *domain.ExplorerBriefingDimension
	for i := range b.Dimensions {
		if b.Dimensions[i].Dimension == "map" {
			mapDim = &b.Dimensions[i]
		}
	}
	if mapDim == nil {
		t.Fatal("map dimension should be emitted (2 distinct maps)")
	}
	for _, e := range mapDim.Entries {
		if e.Label == "Bazaar" {
			t.Error("Bazaar (5 matchs < 10) must not appear in entries")
		}
		if e.Label == "Aquarius" && e.NoteTier == nil {
			t.Error("Aquarius (12 matchs, perf 70) should have a note")
		}
	}
}

func TestBuildExplorerBriefing_RankedGating(t *testing.T) {
	var filtered []domain.MatchHistoryRawRow
	csr := "CSR"
	for i := 0; i < 15; i++ {
		r := briefingRaw("m"+string(rune('a'+i)), i, domain.OutcomeWin, 10, 5, 2, 60, "map1", "Aquarius", "Slayer", "Arène classée")
		r.SkillRatingType = &csr
		prob := 0.5
		r.SkillExpectedWinProb = &prob
		filtered = append(filtered, r)
	}
	// 15 wins sur 15 -> actual 1.0.
	kpis := &domain.KPIStats{MatchesCount: 15, RankDelta: &domain.RankDelta{Kind: "csr", Value: 120, Count: 15}}

	// rankedCapable=false -> Ranked nil.
	if b := svcWithRanked(false).buildExplorerBriefing(filtered, filtered, kpis); b.Ranked != nil {
		t.Error("rankedCapable=false must yield Ranked nil")
	}
	// rankedCapable=true -> Ranked present.
	b := svcWithRanked(true).buildExplorerBriefing(filtered, filtered, kpis)
	if b.Ranked == nil {
		t.Fatal("rankedCapable=true + CSR scope must yield Ranked")
	}
	if b.Ranked.DeltaSum != 120 {
		t.Errorf("DeltaSum = %v, want 120 (reuse RankDelta)", b.Ranked.DeltaSum)
	}
	if b.Ranked.ExpectedWinRate == nil || *b.Ranked.ExpectedWinRate != 0.5 {
		t.Errorf("ExpectedWinRate want 0.5, got %v", b.Ranked.ExpectedWinRate)
	}
	if b.Ranked.ActualWinRate != 1.0 {
		t.Errorf("ActualWinRate = %v, want 1.0", b.Ranked.ActualWinRate)
	}
	if b.Ranked.MatchesWithPrediction != 15 {
		t.Errorf("MatchesWithPrediction = %d, want 15", b.Ranked.MatchesWithPrediction)
	}
}

func TestBuildExplorerBriefing_RankedAbsentWhenLUSR(t *testing.T) {
	var filtered []domain.MatchHistoryRawRow
	for i := 0; i < 15; i++ {
		filtered = append(filtered, briefingRaw("m"+string(rune('a'+i)), i, domain.OutcomeWin, 10, 5, 2, 60, "map1", "Aquarius", "Slayer", "Arène"))
	}
	kpis := &domain.KPIStats{MatchesCount: 15, RankDelta: &domain.RankDelta{Kind: "lusr", Value: 5, Count: 15}}
	b := svcWithRanked(true).buildExplorerBriefing(filtered, filtered, kpis)
	if b.Ranked != nil {
		t.Error("Ranked must be nil when majority rating is LUSR (unranked scope)")
	}
}

func TestBuildExplorerBriefing_TrendGating(t *testing.T) {
	kpis := &domain.KPIStats{MatchesCount: 25}
	// 25 matchs sur 2 jours seulement -> span < 14j -> Trend nil.
	var narrow []domain.MatchHistoryRawRow
	for i := 0; i < 25; i++ {
		narrow = append(narrow, briefingRaw("n"+string(rune('a'+i)), i%2, domain.OutcomeWin, 10, 5, 2, 60, "map1", "Aquarius", "Slayer", "Arène"))
	}
	if b := svcWithRanked(false).buildExplorerBriefing(narrow, narrow, kpis); b.Trend != nil {
		t.Error("span < 14 days -> Trend nil")
	}
	// 25 matchs étalés sur 40 jours -> Trend présent.
	var wide []domain.MatchHistoryRawRow
	for i := 0; i < 25; i++ {
		out := domain.OutcomeWin
		if i%3 == 0 {
			out = domain.OutcomeLoss
		}
		wide = append(wide, briefingRaw("w"+string(rune('a'+i)), i*40/25, out, 10, 5, 2, 60, "map1", "Aquarius", "Slayer", "Arène"))
	}
	b := svcWithRanked(false).buildExplorerBriefing(wide, wide, kpis)
	if b.Trend == nil {
		t.Fatal("span >= 14 days & >= 20 matchs -> Trend present")
	}
	if len(b.Trend.Points) == 0 {
		t.Error("trend should have points")
	}
}

func TestBuildExplorerBriefing_BaselineDeltasSigned(t *testing.T) {
	// Scope : 15 matchs, 12 wins (WR 0.8), perf 70, KDA net positif.
	var scope []domain.MatchHistoryRawRow
	for i := 0; i < 15; i++ {
		out := domain.OutcomeWin
		if i >= 12 {
			out = domain.OutcomeLoss
		}
		scope = append(scope, briefingRaw("s"+string(rune('a'+i)), i, out, 20, 10, 3, 70, "map1", "Aquarius", "Slayer", "Arène"))
	}
	// Baseline : 100 matchs, 50 wins (WR 0.5), perf 50.
	var all []domain.MatchHistoryRawRow
	all = append(all, scope...)
	for i := 0; i < 85; i++ {
		out := domain.OutcomeWin
		if i >= 38 { // 12 (scope) + 38 = 50 wins total sur 100
			out = domain.OutcomeLoss
		}
		all = append(all, briefingRaw("h"+string(rune(i)), 100+i, out, 10, 10, 0, 50, "map1", "Aquarius", "Slayer", "Arène"))
	}
	kpis := &domain.KPIStats{MatchesCount: 15}
	b := svcWithRanked(false).buildExplorerBriefing(scope, all, kpis)
	if b.Baseline == nil {
		t.Fatal("baseline nil")
	}
	if b.Baseline.Matches != 100 {
		t.Errorf("baseline matches = %d, want 100", b.Baseline.Matches)
	}
	if b.Baseline.WinRate != 0.5 {
		t.Errorf("baseline winrate = %v, want 0.5", b.Baseline.WinRate)
	}
	// scope WR 0.8 - baseline 0.5 = +0.3.
	if d := b.Baseline.DeltaWinRate; d < 0.29 || d > 0.31 {
		t.Errorf("delta winrate = %v, want ~+0.3", d)
	}
	// baseline INCLUT le scope (DEC-3, historique complet) :
	// perf baseline = (15*70 + 85*50)/100 = 53 ; delta = 70 - 53 = +17.
	if b.Baseline.DeltaPerf == nil {
		t.Fatal("delta perf nil, want ~+17")
	}
	if dp := *b.Baseline.DeltaPerf; dp < 16.9 || dp > 17.1 {
		t.Errorf("delta perf want ~+17, got %v", dp)
	}
	if b.Baseline.DeltaKDA <= 0 {
		t.Errorf("scope KDA should exceed baseline (delta > 0), got %v", b.Baseline.DeltaKDA)
	}
}

func TestBuildExplorerBriefing_OutcomeSequenceCappedAndSorted(t *testing.T) {
	var filtered []domain.MatchHistoryRawRow
	// 80 matchs, daysAgo décroissant pour brouiller l'ordre d'insertion.
	for i := 0; i < 80; i++ {
		filtered = append(filtered, briefingRaw("m"+string(rune(i)), 80-i, domain.OutcomeWin, 10, 5, 2, 60, "map1", "Aquarius", "Slayer", "Arène"))
	}
	kpis := &domain.KPIStats{MatchesCount: 80}
	b := svcWithRanked(false).buildExplorerBriefing(filtered, filtered, kpis)
	if len(b.OutcomeSequence) != maxOutcomeSequencePoints {
		t.Errorf("outcome sequence len = %d, want %d", len(b.OutcomeSequence), maxOutcomeSequencePoints)
	}
	for i := 1; i < len(b.OutcomeSequence); i++ {
		if b.OutcomeSequence[i].StartTime.Before(b.OutcomeSequence[i-1].StartTime) {
			t.Fatal("outcome sequence must be chronologically ascending")
		}
	}
}
