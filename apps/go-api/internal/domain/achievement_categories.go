// Package domain — achievement_categories.go : catégorisation statique des succès Xbox.
//
// Xbox ne renvoie aucune catégorie dans les définitions de succès. Le mapping est
// maintenu en dur par titre (registre keyé par slug LevelUp), avec lookup par
// name_en normalisé (minuscules + alphanumérique seul) pour absorber les
// variations typographiques entre sources (apostrophes droites/courbes,
// ponctuation, espaces parasites en base).
package domain

import (
	"strings"
	"unicode"
)

// AchievementCategory est la catégorie produit d'un succès Xbox.
type AchievementCategory string

const (
	AchievementCategoryMultiplayer AchievementCategory = "multiplayer"
	AchievementCategoryCampaign    AchievementCategory = "campaign"
	AchievementCategoryOther       AchievementCategory = "other"
)

// achievementCategoriesByTitle : registre des mappings par slug LevelUp.
// Un titre absent du registre n'a pas de catégorisation : catégorie vide,
// le frontend masque alors le filtre (dégradation gracieuse multi-titres).
var achievementCategoriesByTitle = map[string]map[string]AchievementCategory{
	"halo_infinite": haloInfiniteAchievementCategories,
	"halo_5":        halo5AchievementCategories,
}

// AchievementCategoryFor retourne la catégorie d'un succès pour un titre.
//
// Le second retour `unmapped` vaut true quand le titre dispose d'un mapping
// mais que le nom est introuvable (succès publié après la rédaction du
// mapping) — la catégorie retombe alors sur Other et le caller peut logger.
//
//   - titre sans mapping → ("", false)
//   - nom connu          → (catégorie, false)
//   - nom inconnu        → (AchievementCategoryOther, true)
func AchievementCategoryFor(titleSlug, nameEN string) (category AchievementCategory, unmapped bool) {
	table, ok := achievementCategoriesByTitle[titleSlug]
	if !ok {
		return "", false
	}
	if cat, ok := table[normalizeAchievementName(nameEN)]; ok {
		return cat, false
	}
	return AchievementCategoryOther, true
}

// normalizeAchievementName réduit un nom à sa forme canonique de lookup :
// minuscules, lettres et chiffres uniquement.
func normalizeAchievementName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
