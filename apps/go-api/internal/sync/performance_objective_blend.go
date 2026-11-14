package sync

// performance_objective_blend.go — VARIANTE de la note de performance des matchs à objectif
// (décision D-7 du plan des compléments 8.2.0), À L'ÉTUDE : aucun chargeur de production ne
// pose historyRow.ObjectiveBlend, batchComputePerformanceScores ne l'appelle pas et
// skill.PerformanceScoreRev n'a pas bougé. La simulation sur le parc
// (performance_objective_simulation_research_test.go) la compare à la note actuelle avant
// que les poids ne soient figés avec l'utilisateur.
//
// # Ce qui change par rapport à la note actuelle
//
// Aujourd'hui, sur les chaînes à objectif, la métrique `objective_participation` (ospm) pèse
// 0,12 sur 1,04 et vaut les points d'awards `objective` par minute : un porteur de drapeau
// sans frag sort avec une note faible, parce que le reste du profil est du combat. Ici :
//
//	note = (1 − α) × combat + α × percentile_objectif
//	α    = AlphaMin + (AlphaMax − AlphaMin) × part
//
//   - combat : le profil de poids de la chaîne SANS la métrique d'objectif, renormalisé par la
//     somme des poids restants (weightedPercentileMean) ;
//   - part : la part du joueur dans l'objectif de son équipe pour CE match
//     (narrative.ComputeObjectiveTeamShare : actions pondérées par
//     ObjectiveFamilyActionWeights, toutes les lignes de son équipe) ;
//   - percentile_objectif : le rang de l'indice r_f du joueur (narrative.objectiveFamilySubScore,
//     le même indice que l'axe Objectif des profils) parmi les matchs de la MÊME famille de
//     mode de la fenêtre de la chaîne ; sous MinFamilyMatches matchs de famille, repli sur les
//     matchs à objectif de la fenêtre (r_f est normalisé par les repères P80 de sa famille,
//     donc comparable d'une famille à l'autre) ; sous MinFamilyMatches même là, la métrique
//     est ABSENTE.
//
// Métrique absente (titre sans match_objective_stats, match sans bloc d'objectif, référence
// trop courte) : la note est le combat renormalisé, jamais une note à zéro d'objectif. Pas de
// « points requis pour gagner » : le résultat est déjà dans rank_perf.

import (
	"levelup/go-api/internal/analysis/narrative"
)

// objectiveBlendFacts : ce qu'un match porte pour la variante, posé par le chargeur.
type objectiveBlendFacts struct {
	Family narrative.ObjectiveFamily
	// Index : indice r_f du joueur pour ce match, dans [0, narrative.ObjectiveIndexClip].
	Index float64
	// Share : part du joueur dans l'objectif de son équipe, dans [0, 1].
	Share float64
}

// objectiveBlendParams : les trois réglages de la variante. Valeurs de la simulation,
// NON figées (cf. defaultObjectiveBlendParams).
type objectiveBlendParams struct {
	AlphaMin, AlphaMax float64
	// MinFamilyMatches : matchs de référence minimum, par famille puis en repli sur la
	// fenêtre de la chaîne.
	MinFamilyMatches int
}

// defaultObjectiveBlendParams : α de 0,12 (le poids actuel d'ospm) à 0,50 (décision D-7).
// À l'étude : les variantes de sensibilité de la simulation font varier AlphaMax (0,35 et
// 0,65) ; rien n'est figé tant que l'utilisateur n'a pas validé le tableau.
func defaultObjectiveBlendParams() objectiveBlendParams {
	return objectiveBlendParams{AlphaMin: 0.12, AlphaMax: 0.50, MinFamilyMatches: MinMatchesForRelative}
}

// objectiveBlendResult détaille une note de la variante (la simulation les rapporte).
type objectiveBlendResult struct {
	Score float64 // arrondi au dixième
	// Combat : note de combat renormalisée sans l'objectif, non arrondie.
	Combat float64
	// HasObjective : false quand la métrique est absente (Score == Combat arrondi).
	HasObjective bool
	// Percentile : rang de l'indice d'objectif, 0-100 ; Alpha : son poids.
	Percentile, Alpha float64
	// FamilyScope : vrai si la référence est la famille du match, faux en repli sur la chaîne.
	FamilyScope bool
}

// profileWithoutObjective rend une copie du profil sans la métrique d'objectif.
func profileWithoutObjective(weights map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(weights))
	for k, w := range weights {
		if k != MetricKeyObjectiveParticipation {
			out[k] = w
		}
	}
	return out
}

// objectiveReferenceSeries rend les indices de référence : la famille du match d'abord,
// les matchs à objectif de la fenêtre en repli. ok == false sous le minimum.
func objectiveReferenceSeries(cur *objectiveBlendFacts, history []historyRow, minMatches int) (series []float64, familyScope, ok bool) {
	var inFamily, any []float64
	for i := range history {
		f := history[i].ObjectiveBlend
		if f == nil {
			continue
		}
		any = append(any, f.Index)
		if f.Family == cur.Family {
			inFamily = append(inFamily, f.Index)
		}
	}
	if len(inFamily) >= minMatches {
		return inFamily, true, true
	}
	if len(any) >= minMatches {
		return any, false, true
	}
	return nil, false, false
}

// computeObjectiveBlend calcule la note de la variante pour un match. `weights` est le
// profil de la chaîne (WeightsForChain) : la variante ne vaut que pour un profil qui porte
// la métrique d'objectif ; sur tout autre profil, ok == false (l'appelant garde la note
// actuelle). ok == false aussi sous MinMatchesForRelative matchs d'historique ou sans aucune
// métrique de combat calculable.
func computeObjectiveBlend(current *historyRow, history []historyRow, weights map[string]float64, p objectiveBlendParams) (objectiveBlendResult, bool) {
	if _, inProfile := profileWeight(weights, MetricKeyObjectiveParticipation); !inProfile {
		return objectiveBlendResult{}, false
	}
	combat, ok := weightedPercentileMean(current, history, profileWithoutObjective(weights))
	if !ok {
		return objectiveBlendResult{}, false
	}
	res := objectiveBlendResult{Combat: combat, Score: roundPerfScore(combat)}
	if current.ObjectiveBlend == nil {
		return res, true
	}
	series, familyScope, hasRef := objectiveReferenceSeries(current.ObjectiveBlend, history, p.MinFamilyMatches)
	if !hasRef {
		return res, true
	}
	share := clampF(current.ObjectiveBlend.Share, 0, 1)
	alpha := p.AlphaMin + (p.AlphaMax-p.AlphaMin)*share
	pct := percentileRank(current.ObjectiveBlend.Index, series)
	res.HasObjective = true
	res.FamilyScope = familyScope
	res.Percentile = pct
	res.Alpha = alpha
	res.Score = roundPerfScore((1-alpha)*combat + alpha*pct)
	return res, true
}

// computeObjectiveBlendedScore est la note de la variante, sous la forme de
// computeRelativePerformanceScore : nil quand elle n'est pas calculable.
func computeObjectiveBlendedScore(current *historyRow, history []historyRow, weights map[string]float64, p objectiveBlendParams) *float64 {
	res, ok := computeObjectiveBlend(current, history, weights, p)
	if !ok {
		return nil
	}
	return &res.Score
}
