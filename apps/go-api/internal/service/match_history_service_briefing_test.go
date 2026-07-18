package service

import (
	"context"
	"math"
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
	b := svcWithRanked(true).buildExplorerBriefing(context.Background(), filtered, filtered)
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
	if b.Baseline != nil || b.Dimensions != nil || b.Ranked != nil {
		t.Error("low sample: modules must be nil")
	}
}

func TestBuildExplorerBriefing_EmptyScope(t *testing.T) {
	if got := svcWithRanked(false).buildExplorerBriefing(context.Background(), nil, nil); got != nil {
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
	b := svcWithRanked(false).buildExplorerBriefing(context.Background(), filtered, filtered)
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
	b := svcWithRanked(false).buildExplorerBriefing(context.Background(), filtered, filtered)
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

// rankedRawOpts paramètre une raw row rangée pour les tests du module « Classement »
// (struct pour rester ≤ 5 params — CLAUDE.md §5). group renseigne PlaylistGroup.
type rankedRawOpts struct {
	id             string
	daysAgo        int
	ratingType     string
	group          string
	tierLabel      *string
	ratingValue    *float64
	placementDone  *int
	placementTotal *int
}

// briefingRankedRaw construit une raw row rangée (type + chaîne + palier/rating
// optionnels). L'outcome est fixe (Win) — non pertinent au module Classement.
func briefingRankedRaw(o rankedRawOpts) domain.MatchHistoryRawRow {
	r := briefingRaw(o.id, o.daysAgo, domain.OutcomeWin, 10, 5, 2, 60, "map1", "Aquarius", "Slayer", "Arène classée")
	rt := o.ratingType
	r.SkillRatingType = &rt
	if o.group != "" {
		g := o.group
		r.PlaylistGroup = &g
	}
	r.SkillTierLabel = o.tierLabel
	r.RatingValue = o.ratingValue
	r.PlacementDone = o.placementDone
	r.PlacementTotal = o.placementTotal
	return r
}

func TestBuildExplorerBriefing_RankedMonoChainProgression(t *testing.T) {
	bronze, platine := "Bronze I", "Platine VI"
	// 15 matchs CSR chaîne unique "ranked". Palier de départ (plus ancien, daysAgo=14,
	// rating 100) et d'arrivée (plus récent, daysAgo=0, rating 220) posés ; net +120.
	var filtered []domain.MatchHistoryRawRow
	for i := 0; i < 15; i++ {
		var label *string
		rv := float64Ptr(150)
		switch i {
		case 14:
			label, rv = &bronze, float64Ptr(100) // plus ancien
		case 0:
			label, rv = &platine, float64Ptr(220) // plus récent
		}
		filtered = append(filtered, briefingRankedRaw(rankedRawOpts{
			id: "m" + string(rune('a'+i)), daysAgo: i, ratingType: "CSR",
			group: "ranked", tierLabel: label, ratingValue: rv,
		}))
	}
	// rankedCapable=false -> Ranked nil.
	if b := svcWithRanked(false).buildExplorerBriefing(context.Background(), filtered, filtered); b.Ranked != nil {
		t.Error("rankedCapable=false must yield Ranked nil")
	}
	b := svcWithRanked(true).buildExplorerBriefing(context.Background(), filtered, filtered)
	if b.Ranked == nil || len(b.Ranked.Kinds) != 1 {
		t.Fatalf("want exactly 1 CSR chain, got %+v", b.Ranked)
	}
	k := b.Ranked.Kinds[0]
	if k.Kind != "csr" || k.PlaylistGroup != "ranked" || k.Matches != 15 {
		t.Errorf("chain identity: want csr/ranked/15, got %q/%q/%d", k.Kind, k.PlaylistGroup, k.Matches)
	}
	if k.DeltaPerMatch == nil || math.Abs(*k.DeltaPerMatch-8.0) > 1e-9 {
		t.Errorf("DeltaPerMatch want 8.0 (net +120 / 15), got %v", k.DeltaPerMatch)
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

func TestBuildExplorerBriefing_RankedMultiChainNeverCrossed(t *testing.T) {
	csrLo, csrHi := "Or I", "Or III"
	arLo, arHi := "Argent II", "Argent V"
	btbHi, btbLo := "Or III", "Or I"
	var filtered []domain.MatchHistoryRawRow
	// Isolation des paliers par chaîne (« never crossed ») avec toutes les chaînes
	// AU-DESSUS du seuil de pertinence DP-5 (>= MinRankedChainMatches = 10) : CSR 25
	// (type majoritaire), arena_slayer 12, btb 10 → LUSR total 22 < 25, ordre
	// csr -> arena -> btb, plafond 3 non mordant.
	// 25 CSR chaîne "ranked" : plus ancien i=24, plus récent i=0.
	for i := 0; i < 25; i++ {
		var label *string
		rv := float64Ptr(230)
		switch i {
		case 24:
			label, rv = &csrLo, float64Ptr(200)
		case 0:
			label, rv = &csrHi, float64Ptr(260)
		}
		filtered = append(filtered, briefingRankedRaw(rankedRawOpts{
			id: "c" + string(rune('a'+i)), daysAgo: 100 + i, ratingType: "CSR",
			group: "ranked", tierLabel: label, ratingValue: rv,
		}))
	}
	// 12 LUSR chaîne "arena_slayer" : rating monte (Argent II -> Argent V).
	for i := 0; i < 12; i++ {
		var label *string
		rv := float64Ptr(15)
		switch i {
		case 11:
			label, rv = &arLo, float64Ptr(10)
		case 0:
			label, rv = &arHi, float64Ptr(20)
		}
		filtered = append(filtered, briefingRankedRaw(rankedRawOpts{
			id: "a" + string(rune('a'+i)), daysAgo: 200 + i, ratingType: "LUSR",
			group: "arena_slayer", tierLabel: label, ratingValue: rv,
		}))
	}
	// 10 LUSR chaîne "btb" : rating baisse (Or III -> Or I).
	for i := 0; i < 10; i++ {
		var label *string
		rv := float64Ptr(42)
		switch i {
		case 9:
			label, rv = &btbHi, float64Ptr(50)
		case 0:
			label, rv = &btbLo, float64Ptr(35)
		}
		filtered = append(filtered, briefingRankedRaw(rankedRawOpts{
			id: "b" + string(rune('a'+i)), daysAgo: 300 + i, ratingType: "LUSR",
			group: "btb", tierLabel: label, ratingValue: rv,
		}))
	}
	b := svcWithRanked(true).buildExplorerBriefing(context.Background(), filtered, filtered)
	// CSR majoritaire (25) d'abord, puis chaînes LUSR par nb de matchs desc :
	// arena_slayer (12) avant btb (10). Total 3 entrées (1 CSR + 2 LUSR), toutes >=
	// seuil → aucune omise, plafond 3 non atteint.
	if b.Ranked == nil || len(b.Ranked.Kinds) != 3 {
		t.Fatalf("want 3 chains (csr/ranked + lusr/arena_slayer + lusr/btb), got %+v", b.Ranked)
	}
	csr, arena, btb := b.Ranked.Kinds[0], b.Ranked.Kinds[1], b.Ranked.Kinds[2]
	if csr.Kind != "csr" || csr.PlaylistGroup != "ranked" || csr.TierStartLabel == nil || *csr.TierStartLabel != "Or I" || csr.TierEndLabel == nil || *csr.TierEndLabel != "Or III" {
		t.Errorf("CSR chain: paliers CSR uniquement (Or I -> Or III), got %+v", csr)
	}
	if arena.Kind != "lusr" || arena.PlaylistGroup != "arena_slayer" || arena.TierStartLabel == nil || *arena.TierStartLabel != "Argent II" || arena.TierEndLabel == nil || *arena.TierEndLabel != "Argent V" {
		t.Errorf("arena_slayer chain: Argent II -> Argent V, jamais mélangé, got %+v", arena)
	}
	if btb.Kind != "lusr" || btb.PlaylistGroup != "btb" || btb.TierStartLabel == nil || *btb.TierStartLabel != "Or III" || btb.TierEndLabel == nil || *btb.TierEndLabel != "Or I" {
		t.Errorf("btb chain: Or III -> Or I, jamais mélangé, got %+v", btb)
	}
	// Co-signage pt/match ↔ progression : arena monte (>0), btb baisse (<0).
	if arena.DeltaPerMatch == nil || *arena.DeltaPerMatch <= 0 {
		t.Errorf("arena_slayer DeltaPerMatch want > 0 (rating monte), got %v", arena.DeltaPerMatch)
	}
	if btb.DeltaPerMatch == nil || *btb.DeltaPerMatch >= 0 {
		t.Errorf("btb DeltaPerMatch want < 0 (rating baisse), got %v", btb.DeltaPerMatch)
	}
}

func TestBuildExplorerBriefing_RankedSmallSecondaryChainOmitted(t *testing.T) {
	// DP-5 : seuil de pertinence par chaîne (MinRankedChainMatches = 10). Une petite
	// chaîne LUSR (3 matchs < seuil) coexiste avec la chaîne CSR qualifiée (15) — la
	// petite est OMISE, seule la CSR reste (le fallback ne se déclenche pas puisqu'au
	// moins une chaîne qualifie).
	var filtered []domain.MatchHistoryRawRow
	for i := 0; i < 15; i++ {
		filtered = append(filtered, briefingRankedRaw(rankedRawOpts{
			id: "c" + string(rune('a'+i)), daysAgo: i, ratingType: "CSR",
			group: "ranked", ratingValue: float64Ptr(float64(100 + i)),
		}))
	}
	for i := 0; i < 3; i++ { // petite chaîne LUSR btb (< seuil → omise)
		filtered = append(filtered, briefingRankedRaw(rankedRawOpts{
			id: "l" + string(rune('a'+i)), daysAgo: 50 + i, ratingType: "LUSR",
			group: "btb", ratingValue: float64Ptr(float64(1 + i)),
		}))
	}
	b := svcWithRanked(true).buildExplorerBriefing(context.Background(), filtered, filtered)
	if b.Ranked == nil || len(b.Ranked.Kinds) != 1 {
		t.Fatalf("want 1 chain (csr/ranked ; lusr/btb 3 < seuil omise), got %+v", b.Ranked)
	}
	if b.Ranked.Kinds[0].Kind != "csr" || b.Ranked.Kinds[0].PlaylistGroup != "ranked" {
		t.Errorf("chaîne retenue : csr/ranked, got %+v", b.Ranked.Kinds)
	}
}

func TestBuildExplorerBriefing_RankedFallbackKeepsPrincipalChain(t *testing.T) {
	// DP-5 fallback : AUCUNE chaîne n'atteint le seuil (CSR 8, LUSR btb 5, tous < 10)
	// mais au moins une progression existe → la chaîne principale (type majoritaire =
	// CSR, 8 > 5, première en ordre canonique) est conservée pour ne jamais tout
	// masquer. Une seule entrée. (13 matchs au total → pas low_sample.)
	bronze, argent := "Bronze I", "Argent III"
	var filtered []domain.MatchHistoryRawRow
	for i := 0; i < 8; i++ { // CSR "ranked" : 8 matchs < seuil (majoritaire)
		var label *string
		switch i {
		case 7:
			label = &bronze
		case 0:
			label = &argent
		}
		filtered = append(filtered, briefingRankedRaw(rankedRawOpts{
			id: "c" + string(rune('a'+i)), daysAgo: i, ratingType: "CSR",
			group: "ranked", tierLabel: label, ratingValue: float64Ptr(float64(100 + i)),
		}))
	}
	for i := 0; i < 5; i++ { // LUSR "btb" : 5 matchs < seuil (minoritaire)
		filtered = append(filtered, briefingRankedRaw(rankedRawOpts{
			id: "l" + string(rune('a'+i)), daysAgo: 50 + i, ratingType: "LUSR",
			group: "btb", ratingValue: float64Ptr(float64(1 + i)),
		}))
	}
	b := svcWithRanked(true).buildExplorerBriefing(context.Background(), filtered, filtered)
	if b.Ranked == nil || len(b.Ranked.Kinds) != 1 {
		t.Fatalf("fallback : want exactly 1 principal chain, got %+v", b.Ranked)
	}
	k := b.Ranked.Kinds[0]
	if k.Kind != "csr" || k.PlaylistGroup != "ranked" {
		t.Errorf("fallback : chaîne principale = type majoritaire csr/ranked, got %q/%q", k.Kind, k.PlaylistGroup)
	}
	if k.Matches != 8 {
		t.Errorf("fallback : Matches want 8, got %d", k.Matches)
	}
}

func TestBuildExplorerBriefing_RankedCapsToMostPlayed(t *testing.T) {
	// DP-5 plafond : 4 chaînes qualifiées (toutes >= seuil) au-delà de
	// RankedChainMaxCount = 3 → seules les 3 LES PLUS JOUÉES sont émises, restituées
	// dans l'ordre canonique. CSR 40 (majoritaire) + arena_slayer 15 + arena_objectif
	// 12 + btb 10 (LUSR total 37 < 40). Top 3 par matchs : csr, arena_slayer,
	// arena_objectif ; btb (10, la moins jouée) écartée. Ordre : csr -> arena_slayer
	// -> arena_objectif.
	add := func(prefix, ratingType, group string, n, dayBase int) []domain.MatchHistoryRawRow {
		rows := make([]domain.MatchHistoryRawRow, 0, n)
		for i := 0; i < n; i++ {
			rows = append(rows, briefingRankedRaw(rankedRawOpts{
				id: prefix + string(rune('a'+i)), daysAgo: dayBase + i, ratingType: ratingType,
				group: group, ratingValue: float64Ptr(float64(100 + i)),
			}))
		}
		return rows
	}
	var filtered []domain.MatchHistoryRawRow
	filtered = append(filtered, add("c", "CSR", "ranked", 40, 0)...)
	filtered = append(filtered, add("a", "LUSR", "arena_slayer", 15, 100)...)
	filtered = append(filtered, add("o", "LUSR", "arena_objectif", 12, 200)...)
	filtered = append(filtered, add("b", "LUSR", "btb", 10, 300)...)
	b := svcWithRanked(true).buildExplorerBriefing(context.Background(), filtered, filtered)
	if b.Ranked == nil || len(b.Ranked.Kinds) != 3 {
		t.Fatalf("plafond : want 3 chains (top N par matchs), got %+v", b.Ranked)
	}
	got := []string{
		b.Ranked.Kinds[0].Kind + "/" + b.Ranked.Kinds[0].PlaylistGroup,
		b.Ranked.Kinds[1].Kind + "/" + b.Ranked.Kinds[1].PlaylistGroup,
		b.Ranked.Kinds[2].Kind + "/" + b.Ranked.Kinds[2].PlaylistGroup,
	}
	want := []string{"csr/ranked", "lusr/arena_slayer", "lusr/arena_objectif"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ordre canonique plafonné[%d] = %q, want %q (full: %+v)", i, got[i], want[i], got)
		}
	}
	for _, k := range b.Ranked.Kinds { // btb (la moins jouée) écartée
		if k.PlaylistGroup == "btb" {
			t.Errorf("btb (10 matchs) doit être écartée par le plafond, got %+v", b.Ranked.Kinds)
		}
	}
}

func TestBuildExplorerBriefing_RankedNoTierLabels(t *testing.T) {
	// Aucun palier mais rating_value présents : progression omise, pt/match calculé.
	var filtered []domain.MatchHistoryRawRow
	for i := 0; i < 15; i++ {
		rv := float64Ptr(120)
		switch i {
		case 14:
			rv = float64Ptr(100)
		case 0:
			rv = float64Ptr(145)
		}
		filtered = append(filtered, briefingRankedRaw(rankedRawOpts{
			id: "m" + string(rune('a'+i)), daysAgo: i, ratingType: "CSR",
			group: "ranked", ratingValue: rv,
		}))
	}
	b := svcWithRanked(true).buildExplorerBriefing(context.Background(), filtered, filtered)
	if b.Ranked == nil || len(b.Ranked.Kinds) != 1 {
		t.Fatalf("want 1 chain, got %+v", b.Ranked)
	}
	k := b.Ranked.Kinds[0]
	if k.DeltaPerMatch == nil || math.Abs(*k.DeltaPerMatch-3.0) > 1e-9 {
		t.Errorf("DeltaPerMatch want 3.0 (net +45 / 15), got %v", k.DeltaPerMatch)
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
		o := rankedRawOpts{id: "m" + string(rune('a'+i)), daysAgo: i, ratingType: "CSR", group: "ranked", ratingValue: float64Ptr(float64(140 + i))}
		switch i {
		case 14: // plus ancien : en placement
			pl := "Placement (7 restants)"
			o.tierLabel, o.placementDone, o.placementTotal, o.ratingValue = &pl, &done, &total, nil
		case 0: // plus récent : palier résolu
			o.tierLabel = &platine
			o.ratingValue = float64Ptr(180)
		}
		filtered = append(filtered, briefingRankedRaw(o))
	}
	b := svcWithRanked(true).buildExplorerBriefing(context.Background(), filtered, filtered)
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
		o := rankedRawOpts{id: "m" + string(rune('a'+i)), daysAgo: i, ratingType: "CSR", group: "ranked", ratingValue: float64Ptr(float64(40 + i))}
		switch i {
		case 14: // plus ancien : palier résolu
			o.tierLabel = &bronze
		case 0: // plus récent : encore en placement
			pl := "Placement (6 restants)"
			o.tierLabel, o.placementDone, o.placementTotal, o.ratingValue = &pl, &done, &total, nil
		}
		filtered = append(filtered, briefingRankedRaw(o))
	}
	b := svcWithRanked(true).buildExplorerBriefing(context.Background(), filtered, filtered)
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

func TestBuildExplorerBriefing_RankedNilWhenNoRatedRows(t *testing.T) {
	// Aucune row rangée (pas de SkillRatingType) → aucun sample → module nil.
	var filtered []domain.MatchHistoryRawRow
	for i := 0; i < 15; i++ {
		filtered = append(filtered, briefingRaw("m"+string(rune('a'+i)), i, domain.OutcomeWin, 10, 5, 2, 60, "map1", "Aquarius", "Slayer", "Arène"))
	}
	b := svcWithRanked(true).buildExplorerBriefing(context.Background(), filtered, filtered)
	if b.Ranked != nil {
		t.Errorf("Ranked must be nil without any rated row, got %+v", b.Ranked)
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
	b := svcWithRanked(false).buildExplorerBriefing(context.Background(), scope, all)
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

	b := svcWithRanked(false).buildExplorerBriefing(context.Background(), scope, scope)
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

	b := svcWithRanked(false).buildExplorerBriefing(context.Background(), scope, all)
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

// peakRankRaw : raw row portant les champs du « Pic rang » (type + palier EN +
// sous-palier + label FR).
func peakRankRaw(id string, daysAgo int, ratingType, tierEN string, subTier int, tierLabel string) domain.MatchHistoryRawRow {
	r := briefingRaw(id, daysAgo, domain.OutcomeWin, 10, 5, 2, 60, "map1", "Aquarius", "Slayer", "Arène")
	r.SkillRatingType = strPtr(ratingType)
	r.SkillTier = strPtr(tierEN)
	r.SubTier = intPtr(subTier)
	r.SkillTierLabel = strPtr(tierLabel)
	return r
}

func TestBuildBriefingScope_DurationAndPeaks(t *testing.T) {
	mk := func(id string, daysAgo int, dur *int, kda, mmr *float64) domain.MatchHistoryRawRow {
		r := briefingRaw(id, daysAgo, domain.OutcomeWin, 10, 5, 2, 60, "map1", "Aquarius", "Slayer", "Arène")
		r.DurationSeconds, r.KDA, r.TeamMMR = dur, kda, mmr
		return r
	}
	rows := []domain.MatchHistoryRawRow{
		mk("a", 2, intPtr(600), float64Ptr(1.5), float64Ptr(1400)),
		mk("b", 1, intPtr(900), float64Ptr(3.0), float64Ptr(1600)),
		mk("c", 0, nil, float64Ptr(2.0), nil), // durée + mmr absents sur ce match
	}
	s := buildBriefingScope(rows)
	if s.TotalDurationSeconds == nil || *s.TotalDurationSeconds != 1500 {
		t.Errorf("TotalDurationSeconds want 1500 (600+900), got %v", s.TotalDurationSeconds)
	}
	if s.PeakKDA == nil || *s.PeakKDA != 3.0 {
		t.Errorf("PeakKDA want 3.0 (max), got %v", s.PeakKDA)
	}
	if s.PeakTeamMMR == nil || *s.PeakTeamMMR != 1600 {
		t.Errorf("PeakTeamMMR want 1600 (max), got %v", s.PeakTeamMMR)
	}
}

func TestBuildBriefingScope_DurationAndMMRNilWhenAbsent(t *testing.T) {
	// briefingRaw ne pose ni durée ni team_mmr → agrégats nil (omission par le front).
	rows := []domain.MatchHistoryRawRow{
		briefingRaw("a", 1, domain.OutcomeWin, 10, 5, 2, 60, "map1", "Aquarius", "Slayer", "Arène"),
		briefingRaw("b", 0, domain.OutcomeLoss, 5, 10, 1, 40, "map1", "Aquarius", "Slayer", "Arène"),
	}
	s := buildBriefingScope(rows)
	if s.TotalDurationSeconds != nil {
		t.Errorf("TotalDurationSeconds want nil (aucune durée), got %v", s.TotalDurationSeconds)
	}
	if s.PeakTeamMMR != nil {
		t.Errorf("PeakTeamMMR want nil (aucun team_mmr), got %v", s.PeakTeamMMR)
	}
}

func TestBuildBriefingScope_PeakRankLUSROnly(t *testing.T) {
	// Diamond (ordinal 5) > Gold (3) ; parmi Diamond, sous-palier 4 > 1 → « Diamant IV ».
	rows := []domain.MatchHistoryRawRow{
		peakRankRaw("a", 3, "LUSR", "Gold", 2, "Or II"),
		peakRankRaw("b", 2, "LUSR", "Diamond", 4, "Diamant IV"),
		peakRankRaw("c", 1, "LUSR", "Diamond", 1, "Diamant I"),
	}
	s := buildBriefingScope(rows)
	if len(s.PeakRanks) != 1 {
		t.Fatalf("want 1 peak rank (LUSR seul), got %+v", s.PeakRanks)
	}
	if s.PeakRanks[0].RatingType != "lusr" || s.PeakRanks[0].TierLabel != "Diamant IV" {
		t.Errorf("want lusr / Diamant IV (palier max atteint), got %+v", s.PeakRanks[0])
	}
}

func TestBuildBriefingScope_PeakRankBothSystems(t *testing.T) {
	rows := []domain.MatchHistoryRawRow{
		peakRankRaw("c1", 4, "CSR", "Onyx", 0, "Onyx"), // pic CSR
		peakRankRaw("c2", 3, "CSR", "Platinum", 3, "Platine III"),
		peakRankRaw("l1", 2, "LUSR", "Gold", 5, "Or V"), // pic LUSR
		peakRankRaw("l2", 1, "LUSR", "Silver", 2, "Argent II"),
	}
	s := buildBriefingScope(rows)
	if len(s.PeakRanks) != 2 {
		t.Fatalf("want 2 peak ranks (lusr + csr), got %+v", s.PeakRanks)
	}
	// Ordre déterministe : LUSR puis CSR.
	if s.PeakRanks[0].RatingType != "lusr" || s.PeakRanks[0].TierLabel != "Or V" {
		t.Errorf("PeakRanks[0] want lusr / Or V, got %+v", s.PeakRanks[0])
	}
	if s.PeakRanks[1].RatingType != "csr" || s.PeakRanks[1].TierLabel != "Onyx" {
		t.Errorf("PeakRanks[1] want csr / Onyx, got %+v", s.PeakRanks[1])
	}
}

func TestBuildBriefingScope_PeakRankNilWhenNoTier(t *testing.T) {
	rows := []domain.MatchHistoryRawRow{
		briefingRaw("a", 0, domain.OutcomeWin, 10, 5, 2, 60, "map1", "Aquarius", "Slayer", "Arène"),
	}
	s := buildBriefingScope(rows)
	if s.PeakRanks != nil {
		t.Errorf("PeakRanks want nil (aucun palier), got %+v", s.PeakRanks)
	}
}

func TestBuildBriefingScope_MinMaxTriptych(t *testing.T) {
	// Triptyques FDA & Perf (DP-1/DEC-MINMAX). Dataset hétérogène ; r.KDA posé = net
	// natif par match (k + a/3 − d) pour que l'agrégat (moyenne des nets) soit encadré
	// par min/max — sinon l'ordre serait purement cosmétique (cf. Découverte-2 du plan).
	mk := func(id string, daysAgo, kills, deaths, assists int, perf float64) domain.MatchHistoryRawRow {
		r := briefingRaw(id, daysAgo, domain.OutcomeWin, kills, deaths, assists, perf, "map1", "Aquarius", "Slayer", "Arène")
		net := float64(kills) + float64(assists)/3.0 - float64(deaths)
		r.KDA = &net
		return r
	}
	rows := []domain.MatchHistoryRawRow{
		mk("a", 2, 8, 6, 6, 40),  // net 4.0
		mk("b", 1, 16, 4, 3, 90), // net 13.0
		mk("c", 0, 6, 9, 9, 62),  // net 0.0
	}
	s := buildBriefingScope(rows)
	// FDA : min = plus bas net (0.0) ; max (peak) = plus haut net (13.0).
	if s.MinKDA == nil || *s.MinKDA != 0.0 {
		t.Errorf("MinKDA want 0.0 (min r.KDA), got %v", s.MinKDA)
	}
	if s.PeakKDA == nil || *s.PeakKDA != 13.0 {
		t.Errorf("PeakKDA want 13.0 (max r.KDA), got %v", s.PeakKDA)
	}
	// Perf : min 40, max 90, moyenne (40+90+62)/3 = 64.
	if s.MinPerf == nil || *s.MinPerf != 40 {
		t.Errorf("MinPerf want 40, got %v", s.MinPerf)
	}
	if s.MaxPerf == nil || *s.MaxPerf != 90 {
		t.Errorf("MaxPerf want 90, got %v", s.MaxPerf)
	}
	if s.AvgPerf == nil || math.Abs(*s.AvgPerf-64) > 1e-9 {
		t.Errorf("AvgPerf want 64, got %v", s.AvgPerf)
	}
	// Ordre des triptyques min ≤ moyenne ≤ max. Perf EXACT (moyenne arithmétique) ;
	// FDA garanti par r.KDA = net natif (la moyenne des nets est encadrée par min/max).
	if !(*s.MinPerf <= *s.AvgPerf && *s.AvgPerf <= *s.MaxPerf) {
		t.Errorf("ordre triptyque Perf min ≤ moy ≤ max violé : %v / %v / %v", *s.MinPerf, *s.AvgPerf, *s.MaxPerf)
	}
	if !(*s.MinKDA <= s.KDA && s.KDA <= *s.PeakKDA) {
		t.Errorf("ordre triptyque FDA min ≤ kda ≤ peak violé : %v / %v / %v", *s.MinKDA, s.KDA, *s.PeakKDA)
	}
}

func TestBuildBriefingScope_MinMaxNilWhenAbsent(t *testing.T) {
	// Aucun r.KDA ni PerformanceScore (rows nues) → bornes min/max ET moyenne perf nil
	// (le front n'affiche alors que la moyenne FDA agrégée, sans « — » parasite).
	rows := []domain.MatchHistoryRawRow{
		{MatchID: "a", Outcome: domain.OutcomeWin, Kills: 10, Deaths: 5, Assists: 2},
		{MatchID: "b", Outcome: domain.OutcomeLoss, Kills: 5, Deaths: 10, Assists: 1},
	}
	s := buildBriefingScope(rows)
	if s.MinKDA != nil || s.PeakKDA != nil {
		t.Errorf("MinKDA/PeakKDA want nil (aucun r.KDA), got %v / %v", s.MinKDA, s.PeakKDA)
	}
	if s.MinPerf != nil || s.MaxPerf != nil || s.AvgPerf != nil {
		t.Errorf("MinPerf/MaxPerf/AvgPerf want nil (aucun score), got %v / %v / %v", s.MinPerf, s.MaxPerf, s.AvgPerf)
	}
}
