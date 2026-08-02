// Package analysis — citations_test.go : tests unitaires pour les algorithmes citations.
package analysis

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// =============================================================================
// Tests MergeCitationTotals
// =============================================================================

func TestMergeCitationTotals_Empty(t *testing.T) {
	items := MergeCitationTotals(nil, nil)
	if len(items) != 0 {
		t.Errorf("attendu 0 items pour données vides, got %d", len(items))
	}
}

func TestMergeCitationTotals_ZeroTotalVisible(t *testing.T) {
	totals := []domain.CitationTotalRow{
		{NameNorm: "triple_kill", Total: 0},
		{NameNorm: "double_kill", Total: 5},
	}
	mappings := []domain.CitationMappingRow{
		{NameNorm: "triple_kill", NameDisplay: "Triple Kill", Category: "kills"},
		{NameNorm: "double_kill", NameDisplay: "Double Kill", Category: "kills"},
	}
	items := MergeCitationTotals(totals, mappings)
	// Les citations non commencées (Total=0) doivent apparaître — filtre supprimé.
	if len(items) != 2 {
		t.Errorf("attendu 2 items (toutes les citations du catalogue), got %d", len(items))
	}
	found := false
	for _, it := range items {
		if it.NameNorm == "triple_kill" && it.Total == 0 {
			found = true
		}
	}
	if !found {
		t.Error("triple_kill avec Total=0 doit être présent")
	}
}

func TestMergeCitationTotals_OrphanTotalIgnored(t *testing.T) {
	// Une citation avec des données dans match_citations mais absente du catalogue
	// (mappings vide) ne doit PAS s'afficher — le catalogue est la source de vérité.
	totals := []domain.CitationTotalRow{
		{NameNorm: "unknown_citation", Total: 3},
	}
	items := MergeCitationTotals(totals, nil)
	if len(items) != 0 {
		t.Errorf("attendu 0 items (orphan ignoré), got %d", len(items))
	}
}

func TestMergeCitationTotals_Sorted(t *testing.T) {
	totals := []domain.CitationTotalRow{
		{NameNorm: "flag_cap", Total: 2},
		{NameNorm: "triple_kill", Total: 10},
		{NameNorm: "double_kill", Total: 5},
	}
	img := "/path/img.png"
	mappings := []domain.CitationMappingRow{
		// Libellés FR du seed Infinite : normalisés en clés canoniques (item 1.4).
		{NameNorm: "triple_kill", NameDisplay: "Triple Kill", Category: "Arme"},
		{NameNorm: "double_kill", NameDisplay: "Double Kill", Category: "Arme"},
		{NameNorm: "flag_cap", NameDisplay: "Flag Capture", Category: "Mode de jeu", ImagePath: &img},
	}
	items := MergeCitationTotals(totals, mappings)
	// Ordre produit canonique : game_mode avant weapon. Dans weapon :
	// triple_kill (10) avant double_kill (5).
	if items[0].Category != canonical.CommendationCategoryGameMode {
		t.Errorf("première catégorie attendue game_mode, got %s", items[0].Category)
	}
	if items[1].Category != canonical.CommendationCategoryWeapon || items[1].NameNorm != "triple_kill" {
		t.Errorf("attendu weapon/triple_kill en 2e position, got %s/%s", items[1].Category, items[1].NameNorm)
	}
	// Aucun libellé humain ne subsiste dans la sortie.
	for _, it := range items {
		if it.Category != canonical.NormalizeCommendationCategory(it.Category) {
			t.Errorf("catégorie non canonique %q pour %s", it.Category, it.NameNorm)
		}
	}
}

// =============================================================================
// Tests ExtractCategories
// =============================================================================

func TestExtractCategories_Dedup(t *testing.T) {
	items := []domain.CitationItem{
		{Category: "kills"},
		{Category: "objective"},
		{Category: "kills"},
		{Category: "misc"},
	}
	cats := ExtractCategories(items)
	if len(cats) != 3 {
		t.Errorf("attendu 3 catégories, got %d", len(cats))
	}
}

// =============================================================================
// Tests MergeMedalSummary
// =============================================================================

func TestMergeMedalSummary_Empty(t *testing.T) {
	items := MergeMedalSummary(nil, nil)
	if len(items) != 0 {
		t.Errorf("attendu 0 items, got %d", len(items))
	}
}

func TestMergeMedalSummary_Enrichment(t *testing.T) {
	medals := []domain.MedalEarnedRow{
		{MedalID: 1001, TotalCount: 25},
		{MedalID: 1002, TotalCount: 10},
	}
	mappings := []domain.MedalCitationRow{
		{MedalID: 1001, NameDisplay: "Killing Spree", Category: "Multijoueur"},
		// 1002 sans mapping → Unknown, catégorie sentinelle "misc" → "other"
	}
	items := MergeMedalSummary(medals, mappings)
	if len(items) != 2 {
		t.Fatalf("attendu 2 items, got %d", len(items))
	}
	// Ordre canonique : multiplayer avant other (le sans-mapping).
	if items[0].Category != canonical.CommendationCategoryMultiplayer {
		t.Errorf("première catégorie attendue multiplayer, got %s — %v", items[0].Category, items)
	}
	if items[1].Category != canonical.CommendationCategoryOther {
		t.Errorf("seconde catégorie attendue other (misc normalisé), got %s — %v", items[1].Category, items)
	}
	// Trouver Killing Spree
	found := false
	for _, item := range items {
		if item.MedalName == "Killing Spree" && item.Count == 25 {
			found = true
		}
	}
	if !found {
		t.Error("Killing Spree non trouvé dans les items")
	}
}

// =============================================================================
// Tests GroupCommendationsByCategory
// =============================================================================

func TestGroupCommendationsByCategory_Empty(t *testing.T) {
	cats := GroupCommendationsByCategory(nil)
	if cats != nil {
		t.Errorf("attendu nil pour données vides, got %v", cats)
	}
}

func TestGroupCommendationsByCategory_GroupAndTotal(t *testing.T) {
	items := []domain.CommendationItem{
		{MedalID: 1, MedalName: "Spree", Count: 10, Category: "spree"},
		{MedalID: 2, MedalName: "Running Riot", Count: 5, Category: "spree"},
		{MedalID: 3, MedalName: "Flag Cap", Count: 8, Category: "objective"},
	}
	cats := GroupCommendationsByCategory(items)
	if len(cats) != 2 {
		t.Fatalf("attendu 2 catégories, got %d", len(cats))
	}
	// Trouver spree
	var spreeTotal int
	for _, c := range cats {
		if c.Category == "spree" {
			spreeTotal = c.Total
			break
		}
	}
	if spreeTotal != 15 {
		t.Errorf("total spree attendu 15, got %d", spreeTotal)
	}
}

// =============================================================================
// Tests ComputeFullMatchCitations — moteur complet
// =============================================================================

func makeMedalMapping(norm, mappingType string, medalID int64) domain.CitationFullMapping {
	return domain.CitationFullMapping{
		NameNorm:    norm,
		NameDisplay: norm,
		MappingType: mappingType,
		MedalID:     &medalID,
	}
}

func makeStatMapping(norm, stat string) domain.CitationFullMapping {
	s := stat
	return domain.CitationFullMapping{
		NameNorm:    norm,
		NameDisplay: norm,
		MappingType: "stat",
		StatName:    &s,
	}
}

func makeAwardMapping(norm, award string) domain.CitationFullMapping {
	a := award
	return domain.CitationFullMapping{
		NameNorm:    norm,
		NameDisplay: norm,
		MappingType: "award",
		AwardName:   &a,
	}
}

func TestComputeFullMatchCitations_MedalType(t *testing.T) {
	mappings := []domain.CitationFullMapping{
		makeMedalMapping("triple_kill", "medal", 1001),
	}
	ctx := domain.CitationContext{
		Medals: map[int64]int{1001: 3},
		Stats:  map[string]float64{},
		Awards: map[string]int{},
	}
	deltas := ComputeFullMatchCitations(CitationProgressInput{Ctx: ctx}, mappings)
	if len(deltas) != 1 {
		t.Fatalf("attendu 1 delta, got %d", len(deltas))
	}
	if deltas[0].Value != 3 {
		t.Errorf("valeur attendue 3, got %d", deltas[0].Value)
	}
}

func TestComputeFullMatchCitations_StatType(t *testing.T) {
	mappings := []domain.CitationFullMapping{
		makeStatMapping("kills_stat", "kills"),
	}
	ctx := domain.CitationContext{
		Medals: map[int64]int{},
		Stats:  map[string]float64{"kills": 12.0},
		Awards: map[string]int{},
	}
	deltas := ComputeFullMatchCitations(CitationProgressInput{Ctx: ctx}, mappings)
	if len(deltas) != 1 || deltas[0].Value != 12 {
		t.Errorf("attendu 12, got %v", deltas)
	}
}

func makeObjectiveStatMapping(norm, stat string) domain.CitationFullMapping {
	s := stat
	return domain.CitationFullMapping{
		NameNorm:    norm,
		NameDisplay: norm,
		MappingType: domain.CitationMappingTypeObjectiveStat,
		StatName:    &s,
	}
}

// TestComputeFullMatchCitations_ObjectiveStatType : une citation objective_stat lit
// sa colonne (StatName) depuis ctx.Stats (injectée par sync.loadObjectiveStats).
func TestComputeFullMatchCitations_ObjectiveStatType(t *testing.T) {
	mappings := []domain.CitationFullMapping{
		makeObjectiveStatMapping("charge", "zone_captures"),
	}
	ctx := domain.CitationContext{
		Medals: map[int64]int{},
		Stats:  map[string]float64{"zone_captures": 7},
		Awards: map[string]int{},
	}
	deltas := ComputeFullMatchCitations(CitationProgressInput{Ctx: ctx}, mappings)
	if len(deltas) != 1 || deltas[0].NameNorm != "charge" || deltas[0].Value != 7 {
		t.Errorf("attendu charge=7, got %v", deltas)
	}
}

// TestComputeFullMatchCitations_ObjectiveStatMissingColumn : colonne absente de Stats
// (match non-objectif / colonne non peuplée) → valeur 0 → aucun delta (pas de crash).
func TestComputeFullMatchCitations_ObjectiveStatMissingColumn(t *testing.T) {
	mappings := []domain.CitationFullMapping{
		makeObjectiveStatMapping("flag_carrier_hunter", "flag_carriers_killed"),
	}
	ctx := domain.CitationContext{
		Medals: map[int64]int{},
		Stats:  map[string]float64{}, // colonne absente → 0
		Awards: map[string]int{},
	}
	deltas := ComputeFullMatchCitations(CitationProgressInput{Ctx: ctx}, mappings)
	if len(deltas) != 0 {
		t.Errorf("colonne objective_stat absente → 0, aucun delta attendu, got %v", deltas)
	}
}

func TestComputeFullMatchCitations_AwardType(t *testing.T) {
	mappings := []domain.CitationFullMapping{
		makeAwardMapping("hijack_cit", "hijacked_mongoose"),
	}
	ctx := domain.CitationContext{
		Medals: map[int64]int{},
		Stats:  map[string]float64{},
		Awards: map[string]int{"hijacked_mongoose": 2},
	}
	deltas := ComputeFullMatchCitations(CitationProgressInput{Ctx: ctx}, mappings)
	if len(deltas) != 1 || deltas[0].Value != 2 {
		t.Errorf("attendu 2, got %v", deltas)
	}
}

func TestComputeFullMatchCitations_CompositeSkipped(t *testing.T) {
	mappings := []domain.CitationFullMapping{
		{NameNorm: "composite_c", MappingType: "composite"},
	}
	ctx := domain.CitationContext{
		Medals: map[int64]int{},
		Stats:  map[string]float64{},
		Awards: map[string]int{},
	}
	deltas := ComputeFullMatchCitations(CitationProgressInput{Ctx: ctx}, mappings)
	if len(deltas) != 0 {
		t.Errorf("composite doit être ignoré par-match, got %d deltas", len(deltas))
	}
}

func TestComputeFullMatchCitations_ZeroValuesExcluded(t *testing.T) {
	mappings := []domain.CitationFullMapping{
		makeMedalMapping("orphan", "medal", 9999),
	}
	ctx := domain.CitationContext{
		Medals: map[int64]int{},
		Stats:  map[string]float64{},
		Awards: map[string]int{},
	}
	deltas := ComputeFullMatchCitations(CitationProgressInput{Ctx: ctx}, mappings)
	if len(deltas) != 0 {
		t.Errorf("valeur 0 ne doit pas produire de delta, got %d", len(deltas))
	}
}

// Tests fonctions custom déplacés vers internal/games/halo_infinite/
// citations_custom_test.go (P5.4 — Halo-only adapters extraction).
