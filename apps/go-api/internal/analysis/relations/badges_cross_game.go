package relations

// colorTokenCrossGame : token couleur du badge cross-jeu. Token DÉDIÉ
// narrative-encounter-cross-game (ardoise), distinct des autres pills de relation
// (chaque pill a sa couleur propre — cf. _encounterColors.ts côté front).
const colorTokenCrossGame = "narrative-encounter-cross-game"

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
