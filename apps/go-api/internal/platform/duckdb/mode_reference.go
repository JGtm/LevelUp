package duckdb

import (
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/canonical"
)

// ModeAssetReference rend la référence du MODE d'un match (`canonical.MatchSummary.PairMode`)
// selon la source déclarée par le manifeste du titre (title.TitleDescriptor.ModeSource). Une partie présente au
// registre garde toujours la priorité ; sans partie, un titre à source « type de jeu » porte
// une COPIE de la référence du type de jeu, avec son `Kind` (game_variant) : l'enrichissement
// des libellés (EnrichCanonicalAssetTranslations) y applique les dictionnaires des types de
// jeu. Un type de jeu sans nom n'expose aucun identifiant brut : les libellés restent vides
// jusqu'à ce que le dictionnaire le nomme.
//
// Seule définition de la règle « où est le mode », partagée par la source générique
// (projectMatchSummary) et par la source Halo 5 (halo5.projectH5MatchSummary).
func ModeAssetReference(src title.ModeSource, pair, variant *canonical.AssetReference) *canonical.AssetReference {
	if pair != nil || src != title.ModeSourceGameVariant {
		return pair
	}
	return cloneAssetRef(variant)
}
