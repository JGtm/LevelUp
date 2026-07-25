// Package halo_5 — medal_category.go : enrichissement title-specific de la taxonomie
// des médailles Halo 5.
//
// Symétrique à halo_infinite/medal_category.go. Deux niveaux :
//   - BASELINE (tout titre, dans internal/analysis) : catégorie = medal_type
//     normalisé (multikill/spree/skill/style/mode/proficiency/other), super-section
//     "other". C'est le regroupement natif servi tant qu'aucun resolver enrichi n'est
//     enregistré.
//   - ENRICHISSEMENT Halo 5 (ici) : table statique medal_id -> {catégorie(11),
//     super-section(4), tri} bâtie depuis le référentiel du wiki halo.fr (« Médailles
//     de Halo 5 : Guardians ») croisé au catalogue medal_definitions. On RÉUTILISE les
//     noms/descriptions/sprites de LevelUp — seuls la catégorie fine et le tri viennent
//     de la table. Les 11 catégories (multikill, spree, weapons, vehicles, warzone,
//     infection, ctf, oddball, strongholds, objective, style) se répartissent sur les
//     4 super-sections partagées avec Infinite (classics, game_modes, weapons_equipment,
//     other) — d'où un rendu de page Médailles cohérent entre titres.
//
// Câblage title-agnostic : le package du titre expose MedalCategoryResolver et le boot
// l'enregistre sous halo_5.TitleSlug dans le registre du service
// (service.RegisterMedalCategoryResolver). Aucun `slug ==` : les titres non enregistrés
// utilisent la baseline (défaut du registre). Miroir EXACT du câblage Halo Infinite.
package halo_5

import "levelup/go-api/internal/analysis"

// medalCategoryEntry est une ligne de medalCategoryTable (fichier généré).
type medalCategoryEntry struct {
	category     string
	superSection string
	sort         int
}

// MedalCategoryResolver résout la catégorie/super-section/tri d'une médaille Halo 5 via
// la table wiki, avec repli baseline pour les IDs inconnus (médailles custom, nouvelles
// médailles non encore importées). Implémente port.MedalCategoryResolver de façon
// structurelle (sans importer port -> pas de cycle).
type MedalCategoryResolver struct{}

// Resolve retourne la catégorie enrichie si l'ID est connu de la table wiki, sinon le
// regroupement baseline (medal_type normalisé, "other", difficultyIndex).
func (MedalCategoryResolver) Resolve(medalID int64, medalType string, difficultyIndex int) (categoryKey, superSectionKey string, sort int) {
	if e, ok := medalCategoryTable[medalID]; ok {
		return e.category, e.superSection, e.sort
	}
	return analysis.BaselineMedalCategory(medalType, difficultyIndex)
}
