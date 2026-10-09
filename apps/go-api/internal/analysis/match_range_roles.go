// Package analysis — match_range_roles.go : LES SEUILS DE RÔLE DE PORTÉE D'UNE PÉRIODE
// (plan .ai/V7.5/PLAN_AJUSTEMENTS_SUPP_PRE_V75_2026-09-21.md, lot U, décision D23-4).
//
// # LES SEUILS SONT DES QUANTILES, JAMAIS DES MÈTRES EN DUR
//
// Tiers bas (1/3) et tiers haut (2/3) des écarts au lobby de la période. Un joueur qui joue
// toute sa période au fusil de précision garde donc trois bandes internes — c'est le seul
// moyen de lire « ce soir j'ai joué plus loin que d'habitude » ; des mètres en dur ne
// diraient que la playlist.
//
// # SEULS LES POINTS PLEINS COMPTENT
//
// Une médiane sur trois frags n'est pas une médiane : un match sous
// [MatchRangeRoleMinMeasured] frags mesurés ne fabrique aucune bande. C'est exactement le
// `PLANCHER_MESURE` du nuage de l'Escouade (`squadRangeRoles.logic.ts`) — les deux pages
// doivent tracer les MÊMES bandes sur les mêmes données, donc le seuil et la règle de
// quantile sont les mêmes des deux côtés (interpolation linéaire, `percentileLinear` /
// `quantileLineaire`).
//
// # RIEN PLUTÔT QUE DES BANDES FABRIQUÉES
//
// Sous [MatchRangeRoleMinPoints] points pleins, les trois valeurs sont ABSENTES : deux
// matchs ne définissent pas un habituel, et des bandes calculées dessus se liraient comme
// une mesure de période. Une seule porte pour les trois — deux gardes auraient laissé une
// médiane de période sans les bandes qui lui donnent son sens.
package analysis

import (
	"sort"

	"levelup/go-api/internal/domain"
)

// MatchRangeRoleMinMeasured est le nombre de frags mesurés au-dessous duquel la médiane
// d'un match ne compte pas dans les bandes (jumeau de `PLANCHER_MESURE`, web).
const MatchRangeRoleMinMeasured = 5

// MatchRangeRoleMinPoints est le nombre de points pleins au-dessous duquel la période ne
// porte aucun repère (jumeau de `MIN_BATONS_POUR_BANDES`, web).
const MatchRangeRoleMinPoints = 3

// MatchRangeRoleThresholds sont les repères d'une période. Les trois champs sont nil
// ENSEMBLE quand la période ne porte pas assez de points pleins.
type MatchRangeRoleThresholds struct {
	LowM    *float64
	HighM   *float64
	MedianM *float64
	// FullPoints est le nombre de points pleins retenus — le dénominateur des trois
	// repères, toujours renseigné (même quand les repères sont absents).
	FullPoints int
}

// MatchRangeRoleBands calcule les repères de rôle d'une période, sur les écarts au lobby du
// joueur `xuid`. `xuid` vide = tous les joueurs publiés (le cas de l'Escouade).
func MatchRangeRoleBands(profiles []domain.MatchRangeProfile, xuid string) MatchRangeRoleThresholds {
	ecarts := make([]float64, 0, len(profiles))
	for _, p := range profiles {
		for _, j := range p.Players {
			if xuid != "" && j.XUID != xuid {
				continue
			}
			if j.Measured < MatchRangeRoleMinMeasured {
				continue
			}
			ecarts = append(ecarts, j.LobbyDeltaM)
		}
	}
	out := MatchRangeRoleThresholds{FullPoints: len(ecarts)}
	if len(ecarts) < MatchRangeRoleMinPoints {
		return out
	}
	sort.Float64s(ecarts)
	bas := percentileLinear(ecarts, 100.0/3.0)
	haut := percentileLinear(ecarts, 200.0/3.0)
	med := percentileLinear(ecarts, 50)
	out.LowM, out.HighM, out.MedianM = &bas, &haut, &med
	return out
}
