// Package sync — citations_checks_test.go : tests unitaires des fonctions pures
// de vérification d'invariants (V2-V4). Sans dépendance DB.
package sync

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// helpers

func sp(s string) *string { return &s }

func leafMapping(norm, tiers string) domain.CitationFullMapping {
	return domain.CitationFullMapping{NameNorm: norm, MappingType: "stat", TierTargets: sp(tiers)}
}

func leafMappingNoTier(norm string) domain.CitationFullMapping {
	return domain.CitationFullMapping{NameNorm: norm, MappingType: "stat"}
}

func compMapping(norm, children string) domain.CitationFullMapping {
	return domain.CitationFullMapping{
		NameNorm: norm, MappingType: "composite", CompositeChildren: sp(children),
	}
}

func compMappingWithTiers(norm, children, tiers string) domain.CitationFullMapping {
	return domain.CitationFullMapping{
		NameNorm: norm, MappingType: "composite",
		CompositeChildren: sp(children), TierTargets: sp(tiers),
	}
}

// =============================================================================
// V2 — checkV2LeafCumul
// =============================================================================

func TestCheckV2_LeafCumulOK(t *testing.T) {
	mappings := []domain.CitationFullMapping{leafMapping("br75", "100,200,500")}
	tierMax := map[string]int{"br75": 500}
	cumuls := map[string]int{"br75": 499}
	violations := checkV2LeafCumul(mappings, tierMax, cumuls)
	if len(violations) != 0 {
		t.Errorf("attendu 0 violation, obtenu %d: %v", len(violations), violations)
	}
}

func TestCheckV2_LeafCumulExceedsMax(t *testing.T) {
	mappings := []domain.CitationFullMapping{leafMapping("br75", "100,200,500")}
	tierMax := map[string]int{"br75": 500}
	cumuls := map[string]int{"br75": 501} // au-dessus du max
	violations := checkV2LeafCumul(mappings, tierMax, cumuls)
	if len(violations) != 1 {
		t.Fatalf("attendu 1 violation, obtenu %d", len(violations))
	}
	if violations[0].Rule != "V2" {
		t.Errorf("attendu rule V2, obtenu %q", violations[0].Rule)
	}
}

func TestCheckV2_LeafNoTierTargets_Ignored(t *testing.T) {
	// Feuille sans tier_targets → pas de cap → pas de vérification V2.
	mappings := []domain.CitationFullMapping{leafMappingNoTier("free_leaf")}
	tierMax := map[string]int{"free_leaf": 0}
	cumuls := map[string]int{"free_leaf": 9999}
	violations := checkV2LeafCumul(mappings, tierMax, cumuls)
	if len(violations) != 0 {
		t.Errorf("feuille sans tier_targets ne doit pas violer V2, obtenu %d", len(violations))
	}
}

func TestCheckV2_MultipleLeafs_OneViolation(t *testing.T) {
	mappings := []domain.CitationFullMapping{
		leafMapping("a", "100"),
		leafMapping("b", "200"),
	}
	tierMax := map[string]int{"a": 100, "b": 200}
	cumuls := map[string]int{"a": 100, "b": 201} // b dépasse
	violations := checkV2LeafCumul(mappings, tierMax, cumuls)
	if len(violations) != 1 {
		t.Fatalf("attendu 1 violation, obtenu %d", len(violations))
	}
	if violations[0].Rule != "V2" {
		t.Errorf("rule V2 attendue")
	}
}

// =============================================================================
// V3 — checkV3CompositeCumul
// =============================================================================

func TestCheckV3_CompositeCumulOK(t *testing.T) {
	mappings := []domain.CitationFullMapping{compMapping("meta", `["a","b"]`)}
	tierMax := map[string]int{"meta": 0}
	childCount := map[string]int{"meta": 2}
	cumuls := map[string]int{"meta": 2}
	violations := checkV3CompositeCumul(mappings, tierMax, childCount, cumuls)
	if len(violations) != 0 {
		t.Errorf("attendu 0 violation, obtenu %d", len(violations))
	}
}

func TestCheckV3_CompositeCumulExceedsChildren(t *testing.T) {
	mappings := []domain.CitationFullMapping{compMapping("comp", `["a","b"]`)}
	tierMax := map[string]int{"comp": 0}
	childCount := map[string]int{"comp": 2}
	cumuls := map[string]int{"comp": 3} // 3 > 2 enfants = violation
	violations := checkV3CompositeCumul(mappings, tierMax, childCount, cumuls)
	if len(violations) != 1 {
		t.Fatalf("attendu 1 violation, obtenu %d", len(violations))
	}
	if violations[0].Rule != "V3" {
		t.Errorf("rule V3 attendue")
	}
}

func TestCheckV3_CompositeTierTargetsOverridesChildCount(t *testing.T) {
	// tier_targets="3" avec seulement 2 enfants → effectiveMax=3, cumul=3 → OK.
	mappings := []domain.CitationFullMapping{compMappingWithTiers("comp_t", `["a","b"]`, "3")}
	tierMax := map[string]int{"comp_t": 3}
	childCount := map[string]int{"comp_t": 2}
	cumuls := map[string]int{"comp_t": 3}
	violations := checkV3CompositeCumul(mappings, tierMax, childCount, cumuls)
	if len(violations) != 0 {
		t.Errorf("tier_targets=3 → effectiveMax=3 → cumul=3 ne doit pas violer V3")
	}
}

func TestCheckV3_CompositeNoChildren_Ignored(t *testing.T) {
	// composite avec childCount=0 → effectiveMax=0 → pas de vérification.
	mappings := []domain.CitationFullMapping{compMapping("empty", `[]`)}
	tierMax := map[string]int{"empty": 0}
	childCount := map[string]int{"empty": 0}
	cumuls := map[string]int{"empty": 5}
	violations := checkV3CompositeCumul(mappings, tierMax, childCount, cumuls)
	if len(violations) != 0 {
		t.Errorf("composite sans enfants ne doit pas violer V3")
	}
}

// =============================================================================
// V4 — checkV4CompositePerMatch
// =============================================================================

func TestCheckV4_PerMatchOK(t *testing.T) {
	mappings := []domain.CitationFullMapping{compMapping("comp", `["a","b","c"]`)}
	childCount := map[string]int{"comp": 3}
	perMatch := map[string]map[string]int{
		"match1": {"comp": 2}, // 2 ≤ 3 → OK
		"match2": {"comp": 1},
	}
	violations := checkV4CompositePerMatch(mappings, childCount, perMatch)
	if len(violations) != 0 {
		t.Errorf("attendu 0 violation, obtenu %d", len(violations))
	}
}

func TestCheckV4_PerMatchExceedsChildren(t *testing.T) {
	mappings := []domain.CitationFullMapping{compMapping("comp", `["a","b"]`)}
	childCount := map[string]int{"comp": 2}
	perMatch := map[string]map[string]int{
		"match1": {"comp": 3}, // 3 > 2 → violation
	}
	violations := checkV4CompositePerMatch(mappings, childCount, perMatch)
	if len(violations) != 1 {
		t.Fatalf("attendu 1 violation, obtenu %d", len(violations))
	}
	if violations[0].Rule != "V4" {
		t.Errorf("rule V4 attendue, obtenu %q", violations[0].Rule)
	}
}

func TestCheckV4_LeafIgnored(t *testing.T) {
	// V4 ne vérifie que les composites, pas les feuilles.
	mappings := []domain.CitationFullMapping{
		{NameNorm: "leaf", MappingType: "stat"},
		compMapping("comp", `["leaf"]`),
	}
	childCount := map[string]int{"comp": 1}
	// leaf a une valeur absurde → ignorée ; comp dans les clous
	perMatch := map[string]map[string]int{
		"m1": {"leaf": 9999, "comp": 1},
	}
	violations := checkV4CompositePerMatch(mappings, childCount, perMatch)
	if len(violations) != 0 {
		t.Errorf("feuilles ne doivent pas violer V4, obtenu %d violations", len(violations))
	}
}

func TestCheckV4_MultipleMatches_OneViolation(t *testing.T) {
	mappings := []domain.CitationFullMapping{compMapping("c", `["x"]`)}
	childCount := map[string]int{"c": 1}
	perMatch := map[string]map[string]int{
		"m_ok":  {"c": 1},
		"m_bad": {"c": 2}, // 2 > 1 → violation
	}
	violations := checkV4CompositePerMatch(mappings, childCount, perMatch)
	if len(violations) != 1 {
		t.Fatalf("attendu 1 violation, obtenu %d", len(violations))
	}
}
