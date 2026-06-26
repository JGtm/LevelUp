package relations

// colorTokenCrossGame : token couleur du badge cross-jeu. Réutilise
// narrative-encounter-cameleon (sémantiquement « joue les deux jeux » =
// caméléon cross-titre) → zéro ajout de token, zéro churn palette / snapshot.
const colorTokenCrossGame = colorTokenCameleon

// CrossGameBadge construit le badge « Aussi sur {game} » (style solid) si la
// co-occurrence sur l'autre titre atteint le seuil. Pur, testable : retourne nil
// sous le seuil OU si le nom de titre est vide. gameDisplayName nomme l'autre
// titre (résolu via TitleRegistry, jamais un littéral). matchesTogether est le
// nombre de matchs communs sur cet autre titre.
func CrossGameBadge(gameDisplayName string, matchesTogether int) *Badge {
	if gameDisplayName == "" || matchesTogether < CrossGameMinMatchesTogether {
		return nil
	}
	return &Badge{
		LabelKey:   "narrative.encounter.cross_game",
		ColorToken: colorTokenCrossGame,
		Style:      BadgeStyleSolid,
		Detail: map[string]any{
			"game":             gameDisplayName,
			"matches_together": matchesTogether,
		},
	}
}
