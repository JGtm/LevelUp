package analysis

import (
	"math"
	"testing"
	"time"
)

// rpBase : instant de référence des tests (chronologie via daysAgo décroissant).
var rpBase = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

func rpF(v float64) *float64 { return &v }
func rpS(v string) *string   { return &v }
func rpI(v int) *int         { return &v }

// rpSample construit un RankChainSample. daysAgo pilote la chronologie (plus grand
// = plus ancien). group non vide → PlaylistGroup renseigné.
func rpSample(daysAgo int, ratingType, group string, tier *string, ratingValue, ratingDelta *float64, placementDone, placementTotal *int) RankChainSample {
	s := RankChainSample{
		RatingType:     ratingType,
		StartTime:      rpBase.AddDate(0, 0, -daysAgo),
		TierLabel:      tier,
		RatingValue:    ratingValue,
		RatingDelta:    ratingDelta,
		PlacementDone:  placementDone,
		PlacementTotal: placementTotal,
	}
	if group != "" {
		g := group
		s.PlaylistGroup = &g
	}
	return s
}

func TestComputeRankProgressionByChain_Empty(t *testing.T) {
	t.Parallel()
	if got := ComputeRankProgressionByChain(nil); got != nil {
		t.Errorf("empty samples -> nil, got %+v", got)
	}
}

func TestComputeRankProgressionByChain_MonoChain(t *testing.T) {
	t.Parallel()
	// 3 matchs CSR chaîne "ranked" : plus ancien Bronze I (rating 100), milieu sans
	// palier (rating 150), plus récent Platine VI (rating 220). Rating monte de +120.
	samples := []RankChainSample{
		rpSample(14, "CSR", "ranked", rpS("Bronze I"), rpF(100), nil, nil, nil),
		rpSample(7, "CSR", "ranked", nil, rpF(150), nil, nil, nil),
		rpSample(0, "CSR", "ranked", rpS("Platine VI"), rpF(220), nil, nil, nil),
	}
	got := ComputeRankProgressionByChain(samples)
	if len(got) != 1 {
		t.Fatalf("want 1 chain, got %d (%+v)", len(got), got)
	}
	p := got[0]
	if p.RatingType != "csr" || p.PlaylistGroup != "ranked" || p.Matches != 3 {
		t.Errorf("chain identity: want csr/ranked/3, got %s/%s/%d", p.RatingType, p.PlaylistGroup, p.Matches)
	}
	if p.TierStartLabel == nil || *p.TierStartLabel != "Bronze I" {
		t.Errorf("TierStartLabel want Bronze I, got %v", p.TierStartLabel)
	}
	if p.TierEndLabel == nil || *p.TierEndLabel != "Platine VI" {
		t.Errorf("TierEndLabel want Platine VI, got %v", p.TierEndLabel)
	}
	// Variation nette : (220-100)/3 = 40 ; POSITIVE, co-signée avec Bronze→Platine.
	if p.DeltaPerMatch == nil || math.Abs(*p.DeltaPerMatch-40.0) > 1e-9 {
		t.Errorf("DeltaPerMatch want 40 (net +120 / 3), got %v", p.DeltaPerMatch)
	}
	if p.TierStartIsPlacement || p.TierEndPlacementRemaining != nil {
		t.Error("no placement expected")
	}
}

func TestComputeRankProgressionByChain_LUSRMultiChainNeverCrossed(t *testing.T) {
	t.Parallel()
	// Deux chaînes LUSR : btb (3 matchs, rating 30->15 = baisse) et arena_slayer
	// (2 matchs, rating 10->20 = hausse). Paliers JAMAIS croisés : btb reste dans
	// « Or », arena_slayer dans « Argent ».
	samples := []RankChainSample{
		rpSample(30, "LUSR", "btb", rpS("Or III"), rpF(30), nil, nil, nil),
		rpSample(20, "LUSR", "btb", nil, rpF(22), nil, nil, nil),
		rpSample(10, "LUSR", "btb", rpS("Or I"), rpF(15), nil, nil, nil),
		rpSample(25, "LUSR", "arena_slayer", rpS("Argent II"), rpF(10), nil, nil, nil),
		rpSample(5, "LUSR", "arena_slayer", rpS("Argent V"), rpF(20), nil, nil, nil),
	}
	got := ComputeRankProgressionByChain(samples)
	if len(got) != 2 {
		t.Fatalf("want 2 chains (une par playlist_group), got %d (%+v)", len(got), got)
	}
	// Ordre : même type LUSR → par nb de matchs desc : btb (3) avant arena_slayer (2).
	btb, arena := got[0], got[1]
	if btb.PlaylistGroup != "btb" || btb.Matches != 3 {
		t.Fatalf("chain[0] want btb/3, got %s/%d", btb.PlaylistGroup, btb.Matches)
	}
	if arena.PlaylistGroup != "arena_slayer" || arena.Matches != 2 {
		t.Fatalf("chain[1] want arena_slayer/2, got %s/%d", arena.PlaylistGroup, arena.Matches)
	}
	// btb : Or III -> Or I, net (15-30)/3 = -5 (co-signé avec la baisse de palier).
	if btb.TierStartLabel == nil || *btb.TierStartLabel != "Or III" || btb.TierEndLabel == nil || *btb.TierEndLabel != "Or I" {
		t.Errorf("btb tiers want Or III -> Or I, got %v -> %v", btb.TierStartLabel, btb.TierEndLabel)
	}
	if btb.DeltaPerMatch == nil || math.Abs(*btb.DeltaPerMatch-(-5.0)) > 1e-9 {
		t.Errorf("btb DeltaPerMatch want -5, got %v", btb.DeltaPerMatch)
	}
	// arena_slayer : Argent II -> Argent V, net (20-10)/2 = +5.
	if arena.TierStartLabel == nil || *arena.TierStartLabel != "Argent II" || arena.TierEndLabel == nil || *arena.TierEndLabel != "Argent V" {
		t.Errorf("arena tiers want Argent II -> Argent V, got %v -> %v", arena.TierStartLabel, arena.TierEndLabel)
	}
	if arena.DeltaPerMatch == nil || math.Abs(*arena.DeltaPerMatch-5.0) > 1e-9 {
		t.Errorf("arena DeltaPerMatch want +5, got %v", arena.DeltaPerMatch)
	}
}

func TestComputeRankProgressionByChain_CSRSingleRankedChain(t *testing.T) {
	t.Parallel()
	// Tous les matchs CSR partagent playlist_group "ranked" → UNE seule chaîne (P-3).
	var samples []RankChainSample
	for i := 0; i < 12; i++ {
		samples = append(samples, rpSample(i, "CSR", "ranked", nil, rpF(float64(100+i)), nil, nil, nil))
	}
	got := ComputeRankProgressionByChain(samples)
	if len(got) != 1 || got[0].RatingType != "csr" || got[0].Matches != 12 {
		t.Fatalf("CSR = 1 chaîne unique ranked de 12, got %+v", got)
	}
}

func TestComputeRankProgressionByChain_StartInPlacement(t *testing.T) {
	t.Parallel()
	samples := []RankChainSample{
		// plus ancien : en placement (7 restants).
		rpSample(14, "CSR", "ranked", rpS("Placement (7 restants)"), nil, nil, rpI(3), rpI(10)),
		rpSample(0, "CSR", "ranked", rpS("Platine III"), rpF(180), nil, nil, nil),
	}
	p := ComputeRankProgressionByChain(samples)[0]
	if !p.TierStartIsPlacement {
		t.Error("TierStartIsPlacement want true")
	}
	if p.TierStartLabel != nil {
		t.Errorf("TierStartLabel must be nil in placement, got %v", p.TierStartLabel)
	}
	if p.TierEndLabel == nil || *p.TierEndLabel != "Platine III" {
		t.Errorf("TierEndLabel want Platine III, got %v", p.TierEndLabel)
	}
}

func TestComputeRankProgressionByChain_EndInPlacement(t *testing.T) {
	t.Parallel()
	samples := []RankChainSample{
		rpSample(14, "CSR", "ranked", rpS("Bronze II"), rpF(40), nil, nil, nil),
		// plus récent : encore en placement, done 4 sur 10 -> 6 restants.
		rpSample(0, "CSR", "ranked", rpS("Placement (6 restants)"), nil, nil, rpI(4), rpI(10)),
	}
	p := ComputeRankProgressionByChain(samples)[0]
	if p.TierStartLabel == nil || *p.TierStartLabel != "Bronze II" {
		t.Errorf("TierStartLabel want Bronze II, got %v", p.TierStartLabel)
	}
	if p.TierEndPlacementRemaining == nil || *p.TierEndPlacementRemaining != 6 {
		t.Errorf("TierEndPlacementRemaining want 6, got %v", p.TierEndPlacementRemaining)
	}
	if p.TierEndLabel != nil {
		t.Errorf("TierEndLabel must be nil in placement, got %v", p.TierEndLabel)
	}
}

func TestComputeRankProgressionByChain_NoTierLabels(t *testing.T) {
	t.Parallel()
	// Aucun palier mais rating_value présents : progression de paliers OMISE, pt/match
	// tout de même calculé (dégradation par omission, jamais d'erreur).
	samples := []RankChainSample{
		rpSample(9, "LUSR", "chaos", nil, rpF(1.0), nil, nil, nil),
		rpSample(6, "LUSR", "chaos", nil, rpF(1.2), nil, nil, nil),
		rpSample(3, "LUSR", "chaos", nil, rpF(1.4), nil, nil, nil),
	}
	p := ComputeRankProgressionByChain(samples)[0]
	if p.TierStartLabel != nil || p.TierEndLabel != nil || p.TierStartIsPlacement || p.TierEndPlacementRemaining != nil {
		t.Errorf("no tier/placement expected, got %+v", p)
	}
	// (1.4-1.0)/3 = 0.1333...
	if p.DeltaPerMatch == nil || math.Abs(*p.DeltaPerMatch-0.4/3.0) > 1e-9 {
		t.Errorf("DeltaPerMatch want ~0.1333, got %v", p.DeltaPerMatch)
	}
}

func TestComputeRankProgressionByChain_SingleRatedMatchEdge(t *testing.T) {
	t.Parallel()
	// Un seul match noté (rating_value) → DeltaPerMatch = son RatingDelta.
	withDelta := ComputeRankProgressionByChain([]RankChainSample{
		rpSample(0, "CSR", "ranked", rpS("Or I"), rpF(500), rpF(12), nil, nil),
	})[0]
	if withDelta.DeltaPerMatch == nil || math.Abs(*withDelta.DeltaPerMatch-12.0) > 1e-9 {
		t.Errorf("1 match noté avec delta: want 12, got %v", withDelta.DeltaPerMatch)
	}
	// Un seul match noté sans RatingDelta → 0 (pas nil).
	noDelta := ComputeRankProgressionByChain([]RankChainSample{
		rpSample(0, "CSR", "ranked", rpS("Or I"), rpF(500), nil, nil, nil),
	})[0]
	if noDelta.DeltaPerMatch == nil || *noDelta.DeltaPerMatch != 0 {
		t.Errorf("1 match noté sans delta: want 0, got %v", noDelta.DeltaPerMatch)
	}
	// Aucun match noté (rating_value nil partout) → DeltaPerMatch nil.
	unrated := ComputeRankProgressionByChain([]RankChainSample{
		rpSample(0, "CSR", "ranked", rpS("Or I"), nil, nil, nil, nil),
	})[0]
	if unrated.DeltaPerMatch != nil {
		t.Errorf("aucun match noté: DeltaPerMatch want nil, got %v", unrated.DeltaPerMatch)
	}
}

func TestComputeRankProgressionByChain_DeterministicOrder(t *testing.T) {
	t.Parallel()
	// LUSR total 4 matchs (btb 2 + arena_slayer 2) > CSR 3 → LUSR d'abord. Chaînes
	// LUSR à égalité de matchs → clé de chaîne asc : arena_slayer avant btb. Puis CSR.
	samples := []RankChainSample{
		rpSample(1, "CSR", "ranked", nil, rpF(100), nil, nil, nil),
		rpSample(2, "LUSR", "btb", nil, rpF(1), nil, nil, nil),
		rpSample(3, "CSR", "ranked", nil, rpF(101), nil, nil, nil),
		rpSample(4, "LUSR", "arena_slayer", nil, rpF(1), nil, nil, nil),
		rpSample(5, "CSR", "ranked", nil, rpF(102), nil, nil, nil),
		rpSample(6, "LUSR", "btb", nil, rpF(2), nil, nil, nil),
		rpSample(7, "LUSR", "arena_slayer", nil, rpF(2), nil, nil, nil),
	}
	got := ComputeRankProgressionByChain(samples)
	wantOrder := []struct {
		typ, group string
	}{
		{"lusr", "arena_slayer"},
		{"lusr", "btb"},
		{"csr", "ranked"},
	}
	if len(got) != len(wantOrder) {
		t.Fatalf("want %d chains, got %d (%+v)", len(wantOrder), len(got), got)
	}
	for i, w := range wantOrder {
		if got[i].RatingType != w.typ || got[i].PlaylistGroup != w.group {
			t.Errorf("order[%d]: want %s/%s, got %s/%s", i, w.typ, w.group, got[i].RatingType, got[i].PlaylistGroup)
		}
	}
	// Idempotence de l'ordre : re-calcul → même séquence.
	again := ComputeRankProgressionByChain(samples)
	for i := range got {
		if got[i].PlaylistGroup != again[i].PlaylistGroup || got[i].RatingType != again[i].RatingType {
			t.Fatalf("ordre non déterministe à l'indice %d", i)
		}
	}
}

func TestComputeRankProgressionByChain_CSRTieBreakFirst(t *testing.T) {
	t.Parallel()
	// Égalité de matchs entre CSR (2) et LUSR (2) → CSR d'abord (priorité compétitive).
	samples := []RankChainSample{
		rpSample(1, "LUSR", "btb", nil, rpF(1), nil, nil, nil),
		rpSample(2, "CSR", "ranked", nil, rpF(100), nil, nil, nil),
		rpSample(3, "LUSR", "btb", nil, rpF(2), nil, nil, nil),
		rpSample(4, "CSR", "ranked", nil, rpF(101), nil, nil, nil),
	}
	got := ComputeRankProgressionByChain(samples)
	if len(got) != 2 || got[0].RatingType != "csr" || got[1].RatingType != "lusr" {
		t.Fatalf("tie count -> CSR d'abord, got %+v", got)
	}
}
