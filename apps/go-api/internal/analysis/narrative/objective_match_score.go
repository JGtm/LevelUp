package narrative

// objective_match_score.go — LA NOTE D'OBJECTIF D'UN MATCH (0-100), l'axe « Objectif » des radars
// et des profils de participation (radar de la vue Match, profils Ascension, Sessions et Escouade,
// défis de campagne).
//
// Elle repose sur le même noyau que la note de performance à objectif (ComputeObjectiveTeamShare :
// mêmes familles, mêmes poids d'action, frags de soutien à ObjectiveSupportFragWeight) mais SANS
// aucune comparaison à l'historique du joueur : un match se lit seul. Le joueur est mesuré par ce
// qu'il a pris de l'objectif de SON équipe, prise ou soutien.
//
//	S    = max(part de l'objectif, part du soutien)    (une escorte n'est pas punie de ne pas porter)
//	note = 100 × min(1, S × N / ObjectiveFullShareFactor)
//
// N est le nombre de joueurs de l'équipe : S × N vaut 1 pour une part ÉQUITABLE, quelle que soit la
// taille de l'équipe (4 en Arène, 8 en grandes batailles) ; une part brute (25 % en 4 contre 4,
// 12,5 % en 8 contre 8) ne se lirait pas d'un mode à l'autre. La part équitable donne 50 ; le double
// de la part équitable, ou plus, donne 100 (au-delà, l'axe sature : il ne départage pas deux joueurs
// qui portent chacun la moitié de leur équipe). Une équipe sans aucune action d'objectif donne 0 à
// tous : c'est une mesure (personne n'a rien pris), pas une absence.
//
// Sur plusieurs matchs, la valeur est la MOYENNE des notes des matchs mesurés : la même échelle sur
// toutes les surfaces, aucun écrêtage d'un indice par famille. Un match sans donnée d'objectif
// (Halo 5, mode sans objectif, joueur absent de la table) n'est pas mesuré : il n'entre pas dans la
// moyenne, et l'axe est ABSENT, jamais zéro, quand aucun match n'est mesuré. Un match que le joueur n'a pas
// terminé (abandon) n'entre pas non plus dans la moyenne, comme dans la note de performance.
//
// AUCUN I/O : les lignes viennent du chargeur (platform/duckdb) ou de la feuille de match.

import (
	"sort"

	"levelup/go-api/internal/domain"
)

// ObjectiveScoreMax est la borne haute de la note, et le seuil à passer à
// ParticipationThresholds.Objective : la valeur brute de l'axe EST la note 0-100.
const ObjectiveScoreMax = 100.0

// ObjectiveFullShareFactor : multiple de la part équitable de l'équipe qui donne la note pleine.
const ObjectiveFullShareFactor = 2.0

// ObjectiveScoreMinPlayedSeconds : un match où le joueur a joué moins longtemps n'entre pas dans la
// moyenne de plusieurs matchs (ligne fantôme d'un abandon immédiat), comme la population de
// calibration des repères P80 (plus de 30 secondes).
const ObjectiveScoreMinPlayedSeconds = 30

// ObjectiveScoreOf rend la note d'un match depuis la part calculée : 0-100.
func ObjectiveScoreOf(share ObjectiveTeamShare) float64 {
	if share.TeamSize <= 0 {
		return 0
	}
	s := max(share.Share, share.SupportShare, 0)
	return ObjectiveScoreMax * min(1, s*float64(share.TeamSize)/ObjectiveFullShareFactor)
}

// ObjectiveMatchScore rend la note d'objectif de `xuid` pour UN match depuis les lignes de tous
// ses participants (les deux camps). ok == false : la note est absente (cf. ComputeObjectiveTeamShare).
func ObjectiveMatchScore(rows []ObjectiveTeamRow, xuid string) (float64, bool) {
	share, ok := ComputeObjectiveTeamShare(rows, xuid)
	if !ok {
		return 0, false
	}
	return ObjectiveScoreOf(share), true
}

// ObjectiveScoresByMatch rend la note de chaque match MESURÉ du joueur, par identifiant de match.
// `xuids` : les identités du joueur (un gamertag peut en porter plusieurs) ; la première présente
// dans un match est retenue. Sont écartés les matchs où le joueur a joué moins de
// ObjectiveScoreMinPlayedSeconds et ceux qu'il n'a pas terminés (domain.OutcomeDNF), comme la note de
// performance : l'abandon n'entre pas dans la moyenne, mais ses lignes comptent dans l'équipe des
// autres joueurs. C'est l'UNIQUE endroit qui décide, pour toutes les surfaces de moyenne.
func ObjectiveScoresByMatch(rowsByMatch map[string][]ObjectiveTeamRow, xuids ...string) map[string]float64 {
	out := make(map[string]float64, len(rowsByMatch))
	for matchID, rows := range rowsByMatch {
		for _, xuid := range xuids {
			if !countsInAverage(rows, xuid) {
				continue
			}
			if score, ok := ObjectiveMatchScore(rows, xuid); ok {
				out[matchID] = score
				break
			}
		}
	}
	return out
}

// countsInAverage : la ligne de `xuid` existe, dépasse ObjectiveScoreMinPlayedSeconds et n'est pas
// un match non terminé du joueur.
func countsInAverage(rows []ObjectiveTeamRow, xuid string) bool {
	for i := range rows {
		if rows[i].XUID == xuid {
			return rows[i].TimePlayedSeconds > ObjectiveScoreMinPlayedSeconds &&
				rows[i].Outcome != domain.OutcomeDNF
		}
	}
	return false
}

// ObjectiveParticipation rend la note d'objectif moyenne de `xuids` sur les matchs mesurés et leur
// nombre. n == 0 : aucun match mesuré, l'APPELANT RETIRE l'axe (jamais un zéro). La moyenne est
// déjà sur 0-100 : c'est la valeur brute de l'axe (seuil ObjectiveScoreMax).
func ObjectiveParticipation(rowsByMatch map[string][]ObjectiveTeamRow, xuids ...string) (mean float64, n int) {
	scores := ObjectiveScoresByMatch(rowsByMatch, xuids...)
	if len(scores) == 0 {
		return 0, 0
	}
	ids := make([]string, 0, len(scores))
	for id := range scores {
		ids = append(ids, id)
	}
	sort.Strings(ids) // ordre fixe : la somme flottante ne dépend pas de l'itération de la map
	var sum float64
	for _, id := range ids {
		sum += scores[id]
	}
	return sum / float64(len(ids)), len(ids)
}
