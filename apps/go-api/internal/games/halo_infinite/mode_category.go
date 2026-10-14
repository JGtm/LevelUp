// Package halo_infinite - mode_category.go : façade Halo Infinite de la règle de catégorie
// parente d'un pair_name. La règle (table des préfixes, grammaires normale et inversée) vit
// UNE fois dans `analysis/modelabel/category.go`, paquet feuille que sync et migration peuvent
// importer sans dépendre du titre ; ce fichier l'expose sous les noms historiques du titre
// (taxonomie injectée par api/wire, adapter de catalogue, skillchain).
//
// Deux niveaux orthogonaux : le SOUS-MODE affiché (analysis.NormalizeModeLabel, "Arena:Slayer on
// Bazaar" -> "Slayer") et la CATÉGORIE parente ("Arena:Slayer on Bazaar" -> "Assassin"), seule
// concernée ici. Voir modelabel/category.go pour la table et les cas.
package halo_infinite

import "levelup/go-api/internal/analysis/modelabel"

// Catégories canoniques : alias des constantes de modelabel.
const (
	ModeCategoryAssassin    = modelabel.CategoryAssassin
	ModeCategoryFiesta      = modelabel.CategoryFiesta
	ModeCategorySuperFiesta = modelabel.CategorySuperFiesta
	ModeCategoryHuskyRaid   = modelabel.CategoryHuskyRaid
	ModeCategoryBTB         = modelabel.CategoryBTB
	ModeCategoryRanked      = modelabel.CategoryRanked
	ModeCategoryFirefight   = modelabel.CategoryFirefight
	ModeCategoryOther       = modelabel.CategoryOther
)

// Préfixes de playlist repris par les appelants du titre.
const (
	ModePrefixCastleWars     = modelabel.PrefixCastleWars
	ModePrefixBTBHeavies     = modelabel.PrefixBTBHeavies
	ModePrefixSuperHuskyRaid = modelabel.PrefixSuperHuskyRaid
)

// InferModeCategoryFromPairName rend la catégorie parente d'un pair_name brut.
func InferModeCategoryFromPairName(pairName string) string {
	return modelabel.InferCategory(pairName)
}

// PairNamePrefixesForCategory rend les préfixes d'une catégorie (nil pour Other et "").
func PairNamePrefixesForCategory(category string) []string {
	return modelabel.PrefixesForCategory(category)
}

// AllKnownPairNamePrefixes rend tous les préfixes rangés dans une catégorie autre que Other.
func AllKnownPairNamePrefixes() []string { return modelabel.KnownPrefixes() }
