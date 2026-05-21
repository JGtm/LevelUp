package analysis_test

import (
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// helpers

func strPtr(s string) *string { return &s }

func mappingComposite(norm, children string) domain.CitationFullMapping {
	return domain.CitationFullMapping{
		NameNorm:          norm,
		MappingType:       "composite",
		CompositeChildren: strPtr(children),
	}
}

func mappingStat(norm string, tiers *string) domain.CitationFullMapping {
	m := domain.CitationFullMapping{
		NameNorm:    norm,
		MappingType: "stat",
		StatName:    strPtr(norm),
		TierTargets: tiers,
	}
	return m
}

// ---------------------------------------------------------------------------
// computeCompositeCitations — via ComputeFullMatchCitations avec ctx vide
// ---------------------------------------------------------------------------

// buildMappings crée les mappings pour les tests composites.
func buildMappings(statMappings []domain.CitationFullMapping, compositeMappings []domain.CitationFullMapping) []domain.CitationFullMapping {
	out := make([]domain.CitationFullMapping, 0, len(statMappings)+len(compositeMappings))
	out = append(out, statMappings...)
	out = append(out, compositeMappings...)
	return out
}

// injectStats crée un CitationContext minimal avec des stats injectées.
func injectStats(stats map[string]float64) domain.CitationContext {
	return domain.CitationContext{Stats: stats, Medals: map[int64]int{}, Awards: map[string]int{}}
}

// injectProgress crée un CitationProgressInput avec stats et cumulPre.
func injectProgress(stats map[string]float64, cumulPre map[string]int) analysis.CitationProgressInput {
	return analysis.CitationProgressInput{
		Ctx:      injectStats(stats),
		CumulPre: cumulPre,
	}
}

// assertDelta vérifie qu'un delta précis est présent dans la liste.
func assertDelta(t *testing.T, deltas []domain.CitationMatchDelta, norm string, want int) {
	t.Helper()
	for _, d := range deltas {
		if d.NameNorm == norm {
			if d.Value != want {
				t.Errorf("%s: attendu %d, obtenu %d", norm, want, d.Value)
			}
			return
		}
	}
	t.Errorf("%s: absent des deltas (attendu %d)", norm, want)
}

// assertNoDelta vérifie qu'un delta est absent.
func assertNoDelta(t *testing.T, deltas []domain.CitationMatchDelta, norm string) {
	t.Helper()
	for _, d := range deltas {
		if d.NameNorm == norm {
			t.Errorf("%s: ne devrait pas apparaître (obtenu %d)", norm, d.Value)
		}
	}
}

// TestComposite_AllChildrenFire vérifie qu'un composite vaut 2 si 2 enfants matchent.
func TestComposite_AllChildrenFire(t *testing.T) {
	// Enfants : wins_slayer (tier [1]), wins_ctf (tier [1]) — tous les deux >= 1 this match
	tiers1 := "1"
	mappings := buildMappings(
		[]domain.CitationFullMapping{
			mappingStat("wins_slayer", &tiers1),
			mappingStat("wins_ctf", &tiers1),
		},
		[]domain.CitationFullMapping{
			mappingComposite("combo_wins", `["wins_slayer","wins_ctf"]`),
		},
	)

	// Inject stats so that wins_slayer et wins_ctf ont une valeur >= 1.
	// Le moteur lit Stats["wins_slayer"] via dispatchFull/stat.
	ctx := injectStats(map[string]float64{"wins_slayer": 1, "wins_ctf": 1})
	deltas := analysis.ComputeFullMatchCitations(analysis.CitationProgressInput{Ctx: ctx}, mappings)

	found := false
	for _, d := range deltas {
		if d.NameNorm == "combo_wins" {
			if d.Value != 2 {
				t.Errorf("combo_wins: attendu 2, obtenu %d", d.Value)
			}
			found = true
		}
	}
	if !found {
		t.Error("combo_wins absent des deltas")
	}
}

// TestComposite_OneChildMissing vérifie qu'un composite vaut 1 si un seul enfant matche.
func TestComposite_OneChildMissing(t *testing.T) {
	tiers := "1"
	mappings := buildMappings(
		[]domain.CitationFullMapping{
			mappingStat("wins_slayer", &tiers),
			mappingStat("wins_ctf", &tiers),
		},
		[]domain.CitationFullMapping{
			mappingComposite("combo_wins", `["wins_slayer","wins_ctf"]`),
		},
	)
	ctx := injectStats(map[string]float64{"wins_slayer": 1, "wins_ctf": 0})
	deltas := analysis.ComputeFullMatchCitations(analysis.CitationProgressInput{Ctx: ctx}, mappings)
	for _, d := range deltas {
		if d.NameNorm == "combo_wins" && d.Value != 1 {
			t.Errorf("combo_wins: attendu 1, obtenu %d", d.Value)
		}
	}
}

// TestComposite_NoChildFires vérifie qu'un composite absent si aucun enfant ne matche.
func TestComposite_NoChildFires(t *testing.T) {
	tiers := "5"
	mappings := buildMappings(
		[]domain.CitationFullMapping{
			mappingStat("wins_slayer", &tiers),
		},
		[]domain.CitationFullMapping{
			mappingComposite("combo_wins", `["wins_slayer"]`),
		},
	)
	// wins_slayer = 2 < tiers(5) → not masterised
	ctx := injectStats(map[string]float64{"wins_slayer": 2})
	deltas := analysis.ComputeFullMatchCitations(analysis.CitationProgressInput{Ctx: ctx}, mappings)
	for _, d := range deltas {
		if d.NameNorm == "combo_wins" {
			t.Errorf("combo_wins ne doit pas apparaître si aucun enfant masterisé, obtenu %d", d.Value)
		}
	}
}

// TestComposite_NoTierTargets vérifie que sans tier_targets sur l'enfant,
// le composite ne détecte pas de transition (palier final inconnu).
// La leaf elle-même s'incrémente librement (pas de cap).
func TestComposite_NoTierTargets(t *testing.T) {
	mappings := buildMappings(
		[]domain.CitationFullMapping{
			mappingStat("kills_badge", nil), // pas de tier_targets
		},
		[]domain.CitationFullMapping{
			mappingComposite("combo_any", `["kills_badge"]`),
		},
	)
	ctx := injectStats(map[string]float64{"kills_badge": 3})
	deltas := analysis.ComputeFullMatchCitations(analysis.CitationProgressInput{Ctx: ctx}, mappings)
	assertDelta(t, deltas, "kills_badge", 3)  // leaf sans tier → écrite librement
	assertNoDelta(t, deltas, "combo_any")     // composite sans max sur l'enfant → pas de transition
}

// ---------------------------------------------------------------------------
// scoreboard_extremes — MVP/LVP multi-colonnes
// ---------------------------------------------------------------------------

func f64(v float64) *float64 { return &v }

func TestMVPLVP_MultiColumn(t *testing.T) {
	// A : top kills, top assists → devrait être MVP (2+ best cells)
	// B : top deaths → seul worst → devrait être LVP (1 worst cell, pas assez)
	// C : middle everywhere
	scoreboard := []domain.ScoreboardRaw{
		{XUID: "A", Kills: 15, Deaths: 2, Assists: 8, KDA: f64(3.0)},
		{XUID: "B", Kills: 5, Deaths: 12, Assists: 1, KDA: f64(0.4)},
		{XUID: "C", Kills: 8, Deaths: 6, Assists: 4, KDA: f64(1.2)},
	}
	ext := analysis.ComputeMVPLVP(scoreboard)
	if ext.MVPXUID != "A" {
		t.Errorf("attendu MVP=A (plus de kills+assists), obtenu %q", ext.MVPXUID)
	}
	// B a deaths élevé (worst cell pour deaths), KDA bas (worst cell pour kda) → LVP
	if ext.LVPXUID != "B" {
		t.Errorf("attendu LVP=B (deaths+KDA worst), obtenu %q", ext.LVPXUID)
	}
}

func TestMVPLVP_BotExcluded(t *testing.T) {
	// bid( préfixe = bot → exclu du calcul MVP/LVP
	scoreboard := []domain.ScoreboardRaw{
		{XUID: "bid(12345)", Kills: 50, Deaths: 0, Assists: 30, KDA: f64(10.0)},
		{XUID: "human1", Kills: 10, Deaths: 5, Assists: 3, KDA: f64(1.5)},
		{XUID: "human2", Kills: 5, Deaths: 8, Assists: 1, KDA: f64(0.6)},
	}
	ext := analysis.ComputeMVPLVP(scoreboard)
	if ext.MVPXUID == "bid(12345)" {
		t.Error("un bot ne doit pas être MVP")
	}
}

func TestMVPLVP_TooFewPlayers(t *testing.T) {
	// 1 seul joueur → pas de MVP/LVP
	ext := analysis.ComputeMVPLVP([]domain.ScoreboardRaw{
		{XUID: "solo", Kills: 10, Deaths: 5},
	})
	if ext.MVPXUID != "" || ext.LVPXUID != "" {
		t.Error("pas de MVP/LVP avec un seul joueur")
	}
}

// =============================================================================
// Tests T1-T10 — sémantique progression per-match (R1-R8)
// =============================================================================

// T1 : leaf progression simple, pas au max.
func TestProgression_T1_LeafSimpleProgress(t *testing.T) {
	tiers := "25,50,100,200,500"
	mappings := []domain.CitationFullMapping{mappingStat("br75", &tiers)}
	in := injectProgress(map[string]float64{"br75": 5}, map[string]int{"br75": 50})
	assertDelta(t, analysis.ComputeFullMatchCitations(in, mappings), "br75", 5)
}

// T2 : leaf capée par max tier.
func TestProgression_T2_LeafCappedAtMax(t *testing.T) {
	tiers := "25,50,100,200,500"
	mappings := []domain.CitationFullMapping{mappingStat("br75", &tiers)}
	in := injectProgress(map[string]float64{"br75": 10}, map[string]int{"br75": 498})
	assertDelta(t, analysis.ComputeFullMatchCitations(in, mappings), "br75", 2) // 500-498=2
}

// T3 : leaf déjà masterisée → delta 0 (R3).
func TestProgression_T3_LeafAlreadyMastered(t *testing.T) {
	tiers := "25,50,100,200,500"
	mappings := []domain.CitationFullMapping{mappingStat("br75", &tiers)}
	in := injectProgress(map[string]float64{"br75": 5}, map[string]int{"br75": 500})
	assertNoDelta(t, analysis.ComputeFullMatchCitations(in, mappings), "br75")
}

// T4 : composite — enfant traverse son palier final dans ce match (R4).
func TestProgression_T4_CompositeChildCrossesMax(t *testing.T) {
	tiers := "25,50,100,200,500"
	mappings := buildMappings(
		[]domain.CitationFullMapping{mappingStat("br75", &tiers)},
		[]domain.CitationFullMapping{mappingComposite("unsc", `["br75"]`)},
	)
	in := injectProgress(map[string]float64{"br75": 10}, map[string]int{"br75": 495})
	deltas := analysis.ComputeFullMatchCitations(in, mappings)
	assertDelta(t, deltas, "br75", 5)    // 500-495 = 5 (capé)
	assertDelta(t, deltas, "unsc", 1)    // transition ✓
}

// T5 : composite — enfant déjà masterisé avant le match → pas de transition.
func TestProgression_T5_CompositeChildAlreadyMastered(t *testing.T) {
	tiers := "25,50,100,200,500"
	mappings := buildMappings(
		[]domain.CitationFullMapping{mappingStat("br75", &tiers)},
		[]domain.CitationFullMapping{mappingComposite("unsc", `["br75"]`)},
	)
	in := injectProgress(map[string]float64{"br75": 5}, map[string]int{"br75": 500})
	deltas := analysis.ComputeFullMatchCitations(in, mappings)
	assertNoDelta(t, deltas, "br75")
	assertNoDelta(t, deltas, "unsc")
}

// T6 : composite — 2 enfants traversent dans le même match.
func TestProgression_T6_CompositeTwoChildrenCross(t *testing.T) {
	tiers := "10"
	mappings := buildMappings(
		[]domain.CitationFullMapping{mappingStat("c_a", &tiers), mappingStat("c_b", &tiers)},
		[]domain.CitationFullMapping{mappingComposite("parent", `["c_a","c_b"]`)},
	)
	in := injectProgress(
		map[string]float64{"c_a": 5, "c_b": 5},
		map[string]int{"c_a": 8, "c_b": 8},
	)
	assertDelta(t, analysis.ComputeFullMatchCitations(in, mappings), "parent", 2)
}

// T7 : méta cascade — composite de composites (R6).
func TestProgression_T7_MetaCascade(t *testing.T) {
	tiers := "10"
	mappings := buildMappings(
		[]domain.CitationFullMapping{mappingStat("leaf_a", &tiers)},
		[]domain.CitationFullMapping{
			mappingComposite("mid", `["leaf_a"]`),
			mappingComposite("meta", `["mid"]`),
		},
	)
	// leaf_a : pre=8, raw=5 → post=10=max → transition sur mid
	// mid : pre=0 < 1 (max=len(children)=1), post=1 → transition sur meta
	in := injectProgress(map[string]float64{"leaf_a": 5}, map[string]int{"leaf_a": 8})
	deltas := analysis.ComputeFullMatchCitations(in, mappings)
	assertDelta(t, deltas, "leaf_a", 2) // capé à 10-8=2
	assertDelta(t, deltas, "mid", 1)
	assertDelta(t, deltas, "meta", 1)
}

// T8 : composite sans tier_targets — max = len(children) (R7).
func TestProgression_T8_CompositeR7MaxIsChildCount(t *testing.T) {
	tiers := "5"
	mappings := buildMappings(
		[]domain.CitationFullMapping{
			mappingStat("c1", &tiers), mappingStat("c2", &tiers), mappingStat("c3", &tiers),
		},
		[]domain.CitationFullMapping{mappingComposite("parent_r7", `["c1","c2","c3"]`)},
	)
	// c1 : pre=3, raw=5 → post=5=max → transition ; c2, c3 : raw < max
	in := injectProgress(
		map[string]float64{"c1": 5, "c2": 2, "c3": 1},
		map[string]int{"c1": 3, "c2": 0, "c3": 0},
	)
	assertDelta(t, analysis.ComputeFullMatchCitations(in, mappings), "parent_r7", 1)
}

// T9 : idempotence — deux appels consécutifs avec le même CumulPre donnent les mêmes deltas.
func TestProgression_T9_Idempotent(t *testing.T) {
	tiers := "10"
	mappings := buildMappings(
		[]domain.CitationFullMapping{mappingStat("leaf", &tiers)},
		[]domain.CitationFullMapping{mappingComposite("comp", `["leaf"]`)},
	)
	in := injectProgress(map[string]float64{"leaf": 5}, map[string]int{"leaf": 8})
	d1 := analysis.ComputeFullMatchCitations(in, mappings)
	d2 := analysis.ComputeFullMatchCitations(in, mappings)
	if len(d1) != len(d2) {
		t.Errorf("idempotence: len %d != %d", len(d1), len(d2))
	}
}

// T10 : l'ordre des mappings n'affecte pas le résultat.
func TestProgression_T10_OrderIndependence(t *testing.T) {
	tiers := "10"
	leaf := mappingStat("leaf", &tiers)
	comp := mappingComposite("comp", `["leaf"]`)
	in := injectProgress(map[string]float64{"leaf": 5}, map[string]int{"leaf": 8})

	findComp := func(ds []domain.CitationMatchDelta) int {
		for _, d := range ds {
			if d.NameNorm == "comp" {
				return d.Value
			}
		}
		return 0
	}
	d1 := findComp(analysis.ComputeFullMatchCitations(in, []domain.CitationFullMapping{leaf, comp}))
	d2 := findComp(analysis.ComputeFullMatchCitations(in, []domain.CitationFullMapping{comp, leaf}))
	if d1 != d2 {
		t.Errorf("order independence: comp d1=%d d2=%d", d1, d2)
	}
}

func TestMVPLVP_InsufficientBestCells(t *testing.T) {
	// A n'a qu'une seule best cell (kills) → pas de MVP (besoin ≥ 2)
	scoreboard := []domain.ScoreboardRaw{
		{XUID: "A", Kills: 10, Deaths: 5, Assists: 3},
		{XUID: "B", Kills: 5, Deaths: 5, Assists: 5},
		{XUID: "C", Kills: 3, Deaths: 5, Assists: 8},
	}
	ext := analysis.ComputeMVPLVP(scoreboard)
	// A a top kills (+1), C a top assists (+1), aucun n'a 2+ → MVP=""
	if ext.MVPXUID != "" {
		t.Errorf("attendu MVP vide (aucun joueur avec ≥2 best cells), obtenu %q", ext.MVPXUID)
	}
}
