package domain

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// TierPlacement est la valeur STOCKÉE du tier d'une ligne de rang en phase de placement (colonne
// `tier`) et le début du libellé stocké (`tier_label`) : une clé de donnée, jamais un texte
// d'affichage. Le web la reconnaît par `isPlacementTierLabel` et affiche son propre texte.
const TierPlacement = "Placement"

// PlacementDefaultRemaining est le nombre de matchs restants quand le libellé ne le porte pas
// (joueur en placement sans match joué) ; aussi le plafond d'une valeur lue.
const PlacementDefaultRemaining = 10

// placementRemainingRe extrait N de « Placement (N) » et de l'ancienne forme « Placement (N restants) »
// toujours présente dans les lignes déjà écrites : seule la valeur numérique compte.
var placementRemainingRe = regexp.MustCompile(`\((\d+)`)

// PlacementLabel construit le `tier_label` stocké d'une ligne en placement : le tier et le nombre
// de matchs restants, sans mot d'une langue. Seule source d'écriture ; ParsePlacementRemaining la relit.
func PlacementLabel(remaining int) string {
	return fmt.Sprintf("%s (%d)", TierPlacement, remaining)
}

// IsPlacementLabel indique si un `tier_label` stocké décrit la phase de placement.
func IsPlacementLabel(label string) bool {
	return strings.HasPrefix(label, TierPlacement)
}

// ParsePlacementRemaining extrait le nombre de matchs restants d'un `tier_label` de placement.
// Retourne PlacementDefaultRemaining quand le libellé n'en porte pas ou le porte hors bornes
// (plafonné à PlacementDefaultRemaining).
func ParsePlacementRemaining(label string) int {
	m := placementRemainingRe.FindStringSubmatch(label)
	if len(m) < 2 {
		return PlacementDefaultRemaining
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n > PlacementDefaultRemaining {
		return PlacementDefaultRemaining
	}
	return n
}
