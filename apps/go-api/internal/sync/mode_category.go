package sync

// mode_category.go — valeurs de la colonne shared.match_registry.mode_category.
//
// MT-14 : ces libellés sont des valeurs de colonne OPAQUES côté sync. L'audit
// (workflow EXT-2) a établi que la colonne est WRITE-ONLY : aucun lecteur Go n'en
// dérive de décision (le filtre « catégorie de mode » de l'UI dérive de pair_name
// via PairNamePrefixesForCategory, jamais de cette colonne stockée ; les autres
// `mode_category` Go visent des espaces de valeurs orthogonaux —
// engagement_coefficients, catalogue map_mode_pair). Elles sont donc des
// constantes LOCALES au package sync (plus de dépendance au package de titre
// halo_infinite), strictement identiques aux ex-`halo_infinite.ModeCategory*`.
//
// NB : determineModeCategory (transforms_helpers.go, basé sur strings.Contains)
// est une fonction DISTINCTE de halo_infinite.InferModeCategoryFromPairName
// (parsing par préfixe/`:`) — ne pas les confondre ni « simplifier » l'une en
// l'autre, leurs mappings diffèrent (la table de test TestDetermineModeCategoryTable
// fige le comportement byte-identique).
const (
	modeCategoryRanked    = "Ranked"
	modeCategoryFirefight = "Firefight"
	modeCategoryBTB       = "BTB"
	modeCategoryFiesta    = "Fiesta"
	modeCategoryAssassin  = "Assassin"
	modeCategoryOther     = "Other"
)
