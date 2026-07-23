// Package halo_infinite — medal_category.go : enrichissement title-specific de la
// taxonomie des médailles Halo Infinite.
//
// Deux niveaux, comme mode_category.go :
//   - BASELINE (tout titre, dans internal/analysis) : catégorie = medal_type,
//     super-section = "other". C'est le regroupement natif, pas un mode dégradé.
//   - ENRICHISSEMENT Halo Infinite (ici) : table statique medal_id → {catégorie(23),
//     super-section(4), sort} extraite de SpartanRecord (AllMedals.tsx). On RÉUTILISE
//     les noms/descriptions/rareté/images de LevelUp (medal_definitions) — seuls la
//     catégorie et le sort viennent de SpartanRecord.
//
// Câblage title-agnostic : le package du titre expose MedalCategoryResolver et le
// boot l'enregistre sous halo_infinite.TitleSlug dans le registre du service
// (service.RegisterMedalCategoryResolver). Aucun `slug ==` : les titres non
// enregistrés utilisent la baseline (défaut du registre). Miroir du câblage des
// résolveurs sprite/CSR (SetMedalSpriteResolver / SetCSRBadgeResolver).
//
//go:generate go run medal_category_gen.go -src=../../../../../../SpartanRecord/src/Objects/Helpers/AllMedals.tsx -out=medal_category_table.go
package halo_infinite

import "levelup/go-api/internal/analysis"

// medalCategoryEntry est une ligne de medalCategoryTable (fichier généré).
type medalCategoryEntry struct {
	category     string
	superSection string
	sort         int
}

// MedalCategoryResolver résout la catégorie/super-section/sort d'une médaille Halo
// Infinite via la table SpartanRecord, avec repli baseline pour les IDs inconnus
// (médailles custom, nouvelles médailles 343i non encore importées). Implémente
// port.MedalCategoryResolver de façon structurelle (sans importer port → pas de cycle).
type MedalCategoryResolver struct{}

// Resolve retourne la catégorie enrichie si l'ID est connu de la table SpartanRecord,
// sinon le regroupement baseline (medal_type, "other", difficultyIndex).
func (MedalCategoryResolver) Resolve(medalID int64, medalType string, difficultyIndex int) (categoryKey, superSectionKey string, sort int) {
	if e, ok := medalCategoryTable[medalID]; ok {
		return e.category, e.superSection, e.sort
	}
	return analysis.BaselineMedalCategory(medalType, difficultyIndex)
}
