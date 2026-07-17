package service

import (
	"context"
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
	b := svcWithRanked(true).buildExplorerBriefing(context.Background(), filtered, filtered, kpis)
	if b == nil {
		t.Fatal("briefing nil")
	}
	if !b.LowSample {
		t.Error("LowSample should be true for 8 < 10")
	}
	if b.Scope == nil {
		t.Error("socle scope should be served")
	} else if b.Scope.Matches != 8 {
		t.Errorf("scope matches = %d, want 8 (raw filtered count)", b.Scope.Matches)
	}
	if b.PeriodStart == nil || b.PeriodEnd == nil {
		t.Error("period should be set")
	}
	if b.Baseline != nil || b.Dimensions != nil || b.Trend != nil || b.Ranked != nil {
		t.Error("low sample: modules must be nil")
	}
}

func TestBuildExplorerBriefing_EmptyScope(t *testing.T) {
	if got := svcWithRanked(false).buildExplorerBriefing(context.Background(), nil, nil, nil); got != nil {
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
	b := svcWithRanked(false).buildExplorerBriefing(context.Background(), filtered, filtered, kpis)
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
	b := svcWithRanked(false).buildExplorerBriefing(context.Background(), filtered, filtered, kpis)
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

// briefingRankedRaw : raw row rangée d'un type donné, palier / placement optionnels.
// daysAgo pilote la chronologie (plus grand = plus ancien).
func briefingRankedRaw(id string, daysAgo, outcome int, ratingType string, tierLabel *string, placementDone, placementTotal *int) domain.MatchHistoryRawRow {
	r := briefingRaw(id, daysAgo, outcome, 10, 5, 2, 60, "map1", "Aquarius", "Slayer", "Arène classée")
	rt := ratingType
	r.SkillRatingType = &rt
	r.SkillTierLabel = tierLabel
	r.PlacementDone = placementDone
	r.PlacementTotal = placementTotal
	return r
}

func TestBuildExplorerBriefing_RankedMonoTypeProgression(t *testing.T) {
	bronze, platine := "Bronze I", "Platine VI"
	// 15 matchs CSR ; palier de départ (le plus ancien, daysAgo=14) et d'arrivée
	// (le plus récent, daysAgo=0) posés, les autres sans palier.
	var filtered []domain.MatchHistoryRawRow
	for i := 0; i < 15; i++ {
		var label *string
		switch i {
		case 14:
			label = &bronze // plus ancien
		case 0:
			label = &platine // plus récent
		}
		filtered = append(filtered, briefingRankedRaw("m"+string(rune('a'+i)), i, domain.OutcomeWin, "CSR", label, nil, nil))
	}
	kpis := &domain.KPIStats{
		MatchesCount: 15,
		RankDelta:    &domain.RankDelta{Kind: "csr", Value: 120, Count: 15},
		RankDeltas:   []domain.RankDelta{{Kind: "csr", Value: 120, Count: 15}},
	}

	// rankedCapable=false -> Ranked nil.
	if b := svcWithRanked(false).buildExplorerBriefing(context.Background(), filtered, filtered, kpis); b.Ranked != nil {
		t.Error("rankedCapable=false must yield Ranked nil")
	}
	b := svcWithRanked(true).buildExplorerBriefing(context.Background(), filtered, filtered, kpis)
	if b.Ranked == nil || len(b.Ranked.Kinds) != 1 {
		t.Fatalf("want exactly 1 CSR kind, got %+v", b.Ranked)
	}
	k := b.Ranked.Kinds[0]
	if k.Kind != "csr" || k.Matches != 15 {
		t.Errorf("kind=%q matches=%d, want csr / 15", k.Kind, k.Matches)
	}
	if k.DeltaPerMatch == nil || *k.DeltaPerMatch != 8.0 {
		t.Errorf("DeltaPerMatch want 8.0 (120/15), got %v", k.DeltaPerMatch)
	}
	if k.TierStartLabel == nil || *k.TierStartLabel != "Bronze I" {
		t.Errorf("TierStartLabel want Bronze I, got %v", k.TierStartLabel)
	}
	if k.TierEndLabel == nil || *k.TierEndLabel != "Platine VI" {
		t.Errorf("TierEndLabel want Platine VI, got %v", k.TierEndLabel)
	}
	if k.TierStartIsPlacement || k.TierEndPlacementRemaining != nil {
		t.Error("no placement expected in this scope")
	}
}

func TestBuildExplorerBriefing_RankedMixedTypesNeverMerged(t *testing.T) {
	csrLo, csrHi := "Or I", "Or III"
	lusrLo, lusrHi := "Argent II", "Argent V"
	var filtered []domain.MatchHistoryRawRow
	// 13 CSR (majoritaire) : daysAgo 100..112, plus ancien = i=12, plus récent = i=0.
	for i := 0; i < 13; i++ {
		var label *string
		switch i {
		case 12:
			label = &csrLo
		case 0:
			label = &csrHi
		}
		filtered = append(filtered, briefingRankedRaw("c"+string(rune('a'+i)), 100+i, domain.OutcomeWin, "CSR", label, nil, nil))
	}
	// 11 LUSR : daysAgo 200..210, plus ancien = i=10, plus récent = i=0.
	for i := 0; i < 11; i++ {
		var label *string
		switch i {
		case 10:
			label = &lusrLo
		case 0:
			label = &lusrHi
		}
		filtered = append(filtered, briefingRankedRaw("l"+string(rune('a'+i)), 200+i, domain.OutcomeLoss, "LUSR", label, nil, nil))
	}
	kpis := &domain.KPIStats{
		MatchesCount: 24,
		RankDelta:    &domain.RankDelta{Kind: "csr", Value: 26, Count: 13},
		RankDeltas:   []domain.RankDelta{{Kind: "csr", Value: 26, Count: 13}, {Kind: "lusr", Value: 0.5, Count: 11}},
	}
	b := svcWithRanked(true).buildExplorerBriefing(context.Background(), filtered, filtered, kpis)
	if b.Ranked == nil || len(b.Ranked.Kinds) != 2 {
		t.Fatalf("want 2 kinds (csr majoritaire d'abord + lusr), got %+v", b.Ranked)
	}
	csr := b.Ranked.Kinds[0]
	if csr.Kind != "csr" || csr.TierStartLabel == nil || *csr.TierStartLabel != "Or I" || csr.TierEndLabel == nil || *csr.TierEndLabel != "Or III" {
		t.Errorf("CSR kind: paliers CSR uniquement attendus, got %+v", csr)
	}
	lusr := b.Ranked.Kinds[1]
	if lusr.Kind != "lusr" || lusr.TierStartLabel == nil || *lusr.TierStartLabel != "Argent II" || lusr.TierEndLabel == nil || *lusr.TierEndLabel != "Argent V" {
		t.Errorf("LUSR kind: paliers LUSR uniquement attendus (jamais mélangés), got %+v", lusr)
	}
}

func TestBuildExplorerBriefing_RankedSecondaryTypeBelowThresholdOmitted(t *testing.T) {
	var filtered []domain.MatchHistoryRawRow
	for i := 0; i < 15; i++ {
		filtered = append(filtered, briefingRankedRaw("c"+string(rune('a'+i)), i, domain.OutcomeWin, "CSR", nil, nil, nil))
	}
	for i := 0; i < 5; i++ { // LUSR sous minRankedKindMatches (10)
		filtered = append(filtered, briefingRankedRaw("l"+string(rune('a'+i)), 50+i, domain.OutcomeLoss, "LUSR", nil, nil, nil))
	}
	kpis := &domain.KPIStats{
		MatchesCount: 20,
		RankDelta:    &domain.RankDelta{Kind: "csr", Value: 30, Count: 15},
		RankDeltas:   []domain.RankDelta{{Kind: "csr", Value: 30, Count: 15}, {Kind: "lusr", Value: 0.1, Count: 5}},
	}
	b := svcWithRanked(true).buildExplorerBriefing(context.Background(), filtered, filtered, kpis)
	if b.Ranked == nil || len(b.Ranked.Kinds) != 1 || b.Ranked.Kinds[0].Kind != "csr" {
		t.Fatalf("LUSR (5 < 10) must be omitted, only CSR kept, got %+v", b.Ranked)
	}
}

func TestBuildExplorerBriefing_RankedNoTierLabels(t *testing.T) {
	var filtered []domain.MatchHistoryRawRow
	for i := 0; i < 15; i++ {
		filtered = append(filtered, briefingRankedRaw("m"+string(rune('a'+i)), i, domain.OutcomeWin, "CSR", nil, nil, nil))
	}
	kpis := &domain.KPIStats{
		MatchesCount: 15,
		RankDelta:    &domain.RankDelta{Kind: "csr", Value: 45, Count: 15},
		RankDeltas:   []domain.RankDelta{{Kind: "csr", Value: 45, Count: 15}},
	}
	b := svcWithRanked(true).buildExplorerBriefing(context.Background(), filtered, filtered, kpis)
	if b.Ranked == nil || len(b.Ranked.Kinds) != 1 {
		t.Fatalf("want 1 kind, got %+v", b.Ranked)
	}
	k := b.Ranked.Kinds[0]
	if k.DeltaPerMatch == nil || *k.DeltaPerMatch != 3.0 {
		t.Errorf("DeltaPerMatch want 3.0, got %v", k.DeltaPerMatch)
	}
	if k.TierStartLabel != nil || k.TierEndLabel != nil || k.TierStartIsPlacement || k.TierEndPlacementRemaining != nil {
		t.Errorf("no tier / no placement expected, got %+v", k)
	}
}

func TestBuildExplorerBriefing_RankedStartInPlacement(t *testing.T) {
	platine := "Platine III"
	done, total := 3, 10
	var filtered []domain.MatchHistoryRawRow
	for i := 0; i < 15; i++ {
		var label *string
		var pd, pt *int
		switch i {
		case 14: // plus ancien : en placement
			pl := "Placement (7 restants)"
			label, pd, pt = &pl, &done, &total
		case 0: // plus récent : palier résolu
			label = &platine
		}
		filtered = append(filtered, briefingRankedRaw("m"+string(rune('a'+i)), i, domain.OutcomeWin, "CSR", label, pd, pt))
	}
	kpis := &domain.KPIStats{
		MatchesCount: 15,
		RankDelta:    &domain.RankDelta{Kind: "csr", Value: 60, Count: 15},
		RankDeltas:   []domain.RankDelta{{Kind: "csr", Value: 60, Count: 15}},
	}
	b := svcWithRanked(true).buildExplorerBriefing(context.Background(), filtered, filtered, kpis)
	k := b.Ranked.Kinds[0]
	if !k.TierStartIsPlacement {
		t.Error("TierStartIsPlacement want true (début en placement)")
	}
	if k.TierStartLabel != nil {
		t.Errorf("TierStartLabel must be nil in placement (rendu i18n), got %v", k.TierStartLabel)
	}
	if k.TierEndLabel == nil || *k.TierEndLabel != "Platine III" {
		t.Errorf("TierEndLabel want Platine III, got %v", k.TierEndLabel)
	}
}

func TestBuildExplorerBriefing_RankedEndInPlacement(t *testing.T) {
	bronze := "Bronze II"
	done, total := 4, 10 // 6 restants
	var filtered []domain.MatchHistoryRawRow
	for i := 0; i < 15; i++ {
		var label *string
		var pd, pt *int
		switch i {
		case 14: // plus ancien : palier résolu
			label = &bronze
		case 0: // plus récent : encore en placement
			pl := "Placement (6 restants)"
			label, pd, pt = &pl, &done, &total
		}
		filtered = append(filtered, briefingRankedRaw("m"+string(rune('a'+i)), i, domain.OutcomeWin, "CSR", label, pd, pt))
	}
	kpis := &domain.KPIStats{
		MatchesCount: 15,
		RankDelta:    &domain.RankDelta{Kind: "csr", Value: 60, Count: 15},
		RankDeltas:   []domain.RankDelta{{Kind: "csr", Value: 60, Count: 15}},
	}
	b := svcWithRanked(true).buildExplorerBriefing(context.Background(), filtered, filtered, kpis)
	k := b.Ranked.Kinds[0]
	if k.TierStartLabel == nil || *k.TierStartLabel != "Bronze II" {
		t.Errorf("TierStartLabel want Bronze II, got %v", k.TierStartLabel)
	}
	if k.TierEndPlacementRemaining == nil || *k.TierEndPlacementRemaining != 6 {
		t.Errorf("TierEndPlacementRemaining want 6 (total 10 - done 4), got %v", k.TierEndPlacementRemaining)
	}
	if k.TierEndLabel != nil {
		t.Errorf("TierEndLabel must be nil in placement (rendu i18n), got %v", k.TierEndLabel)
	}
}

func TestBuildExplorerBriefing_RankedNilWhenNoRankDeltas(t *testing.T) {
	// Aucun bucket de rating (RankDeltas vide) → module nil, même avec des rows.
	var filtered []domain.MatchHistoryRawRow
	for i := 0; i < 15; i++ {
		filtered = append(filtered, briefingRaw("m"+string(rune('a'+i)), i, domain.OutcomeWin, 10, 5, 2, 60, "map1", "Aquarius", "Slayer", "Arène"))
	}
	kpis := &domain.KPIStats{MatchesCount: 15} // ni RankDelta ni RankDeltas
	b := svcWithRanked(true).buildExplorerBriefing(context.Background(), filtered, filtered, kpis)
	if b.Ranked != nil {
		t.Errorf("Ranked must be nil without any rank delta bucket, got %+v", b.Ranked)
	}
}

func TestBuildExplorerBriefing_TrendGating(t *testing.T) {
	kpis := &domain.KPIStats{MatchesCount: 25}
	// 25 matchs sur 2 jours seulement -> span < 14j -> Trend nil.
	var narrow []domain.MatchHistoryRawRow
	for i := 0; i < 25; i++ {
		narrow = append(narrow, briefingRaw("n"+string(rune('a'+i)), i%2, domain.OutcomeWin, 10, 5, 2, 60, "map1", "Aquarius", "Slayer", "Arène"))
	}
	if b := svcWithRanked(false).buildExplorerBriefing(context.Background(), narrow, narrow, kpis); b.Trend != nil {
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
	b := svcWithRanked(false).buildExplorerBriefing(context.Background(), wide, wide, kpis)
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
	b := svcWithRanked(false).buildExplorerBriefing(context.Background(), scope, all, kpis)
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

// mapDimEntries récupère les entrées de la dimension « map » d'un briefing.
func mapDimEntries(t *testing.T, b *domain.ExplorerBriefing) []domain.ExplorerBriefingDimensionEntry {
	t.Helper()
	for i := range b.Dimensions {
		if b.Dimensions[i].Dimension == "map" {
			return b.Dimensions[i].Entries
		}
	}
	t.Fatal("map dimension should be emitted")
	return nil
}

func TestBuildDimension_FullHistorySortsByWinRate(t *testing.T) {
	// Plein historique (scope == all) : deltas vs baseline tous nuls → CompareByKey
	// trie par MapID (pseudo-aléatoire). La sélection doit basculer sur le taux de
	// victoire décroissant (P-8). Les MapIDs (a1 < m1 < z1) sont volontairement dans
	// l'ordre INVERSE du WR (Alpha .9 > Charlie .6 > Bravo .3) pour prouver le re-tri.
	var scope []domain.MatchHistoryRawRow
	add := func(mapID, mapFR string, wins, losses int) {
		for i := 0; i < wins; i++ {
			scope = append(scope, briefingRaw(mapID+"w"+string(rune('a'+i)), len(scope), domain.OutcomeWin, 12, 6, 3, 60, mapID, mapFR, "Slayer", "Arène"))
		}
		for i := 0; i < losses; i++ {
			scope = append(scope, briefingRaw(mapID+"l"+string(rune('a'+i)), len(scope), domain.OutcomeLoss, 6, 12, 1, 40, mapID, mapFR, "Slayer", "Arène"))
		}
	}
	add("z1", "Alpha", 9, 1)   // WR 0.9
	add("a1", "Bravo", 3, 7)   // WR 0.3
	add("m1", "Charlie", 6, 4) // WR 0.6

	kpis := &domain.KPIStats{MatchesCount: len(scope)}
	b := svcWithRanked(false).buildExplorerBriefing(context.Background(), scope, scope, kpis)
	entries := mapDimEntries(t, b)
	want := []string{"Alpha", "Charlie", "Bravo"}
	if len(entries) != len(want) {
		t.Fatalf("entries = %d, want %d", len(entries), len(want))
	}
	for i, w := range want {
		if entries[i].Label != w {
			t.Errorf("entry[%d].Label = %q, want %q (tri WR décroissant en plein historique)", i, entries[i].Label, w)
		}
	}
}

func TestBuildDimension_FilteredSortsByDelta(t *testing.T) {
	// Sous filtre (scope ⊊ all) : l'ordre V1 par WinRateDelta est conservé.
	// MapA : scope WR .5, hist WR .1 → delta +.4. MapB : scope WR .8, hist WR .8 → delta 0.
	// Ordre par WR : B(.8) > A(.5) ; ordre par delta : A(+.4) > B(0). On attend A puis B.
	var scope, all []domain.MatchHistoryRawRow
	addScope := func(mapID, mapFR string, wins, losses int) {
		for i := 0; i < wins; i++ {
			r := briefingRaw(mapID+"sw"+string(rune('a'+i)), len(all), domain.OutcomeWin, 12, 6, 3, 60, mapID, mapFR, "Slayer", "Arène")
			scope = append(scope, r)
			all = append(all, r)
		}
		for i := 0; i < losses; i++ {
			r := briefingRaw(mapID+"sl"+string(rune('a'+i)), len(all), domain.OutcomeLoss, 6, 12, 1, 40, mapID, mapFR, "Slayer", "Arène")
			scope = append(scope, r)
			all = append(all, r)
		}
	}
	addHistOnly := func(mapID, mapFR string, wins, losses int) {
		for i := 0; i < wins; i++ {
			all = append(all, briefingRaw(mapID+"hw"+string(rune('a'+i)), 500+len(all), domain.OutcomeWin, 12, 6, 3, 60, mapID, mapFR, "Slayer", "Arène"))
		}
		for i := 0; i < losses; i++ {
			all = append(all, briefingRaw(mapID+"hl"+string(rune('a'+i)), 500+len(all), domain.OutcomeLoss, 6, 12, 1, 40, mapID, mapFR, "Slayer", "Arène"))
		}
	}
	addScope("ma", "MapA", 5, 5)     // scope WR .5
	addScope("mb", "MapB", 8, 2)     // scope WR .8
	addHistOnly("ma", "MapA", 0, 40) // A hist total : 5W / 50 = .1
	addHistOnly("mb", "MapB", 32, 8) // B hist total : 40W / 50 = .8

	kpis := &domain.KPIStats{MatchesCount: len(scope)}
	b := svcWithRanked(false).buildExplorerBriefing(context.Background(), scope, all, kpis)
	entries := mapDimEntries(t, b)
	want := []string{"MapA", "MapB"}
	if len(entries) != len(want) {
		t.Fatalf("entries = %d, want %d", len(entries), len(want))
	}
	for i, w := range want {
		if entries[i].Label != w {
			t.Errorf("entry[%d].Label = %q, want %q (tri par delta sous filtre)", i, entries[i].Label, w)
		}
	}
}
