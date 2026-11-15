package sync

// performance_objective_blend.go — VARIANTE de la note de performance des matchs à objectif
// (décision D-7 du plan des compléments 8.2.0, corrigée par les décisions du user du
// 2026-11-15), À L'ÉTUDE : aucun chargeur de production ne pose historyRow.ObjectiveBlend,
// batchComputePerformanceScores ne l'appelle pas et skill.PerformanceScoreRev n'a pas bougé. La
// simulation sur le parc (performance_objective_simulation_research_test.go) la compare à la
// note actuelle avant que les poids ne soient figés.
//
// # Ce qui change par rapport à la note actuelle
//
// Aujourd'hui, sur les chaînes à objectif, la métrique `objective_participation` (ospm) pèse
// 0,12 sur 1,04 et vaut les points d'awards `objective` par minute : un porteur de drapeau
// sans frag sort avec une note faible, parce que le reste du profil est du combat. Ici :
//
//	note = C + α × max(0, O − C)
//	α    = AlphaMin + (AlphaMax − AlphaMin) × part
//
//   - C : le combat — le profil de poids de la chaîne SANS la métrique d'objectif, renormalisé
//     par la somme des poids restants (weightedPercentileMean) ;
//   - O : le percentile objectif — le rang de l'indice r_f BRUT du joueur (narrative.RawIndex,
//     sans l'écrêtage de 1,25 que les radars gardent pour l'affichage) parmi les matchs de la
//     MÊME famille de mode de la fenêtre de la chaîne, les égalités classées à mi-rang
//     (percentileRankMid) ; sous MinFamilyMatches matchs de famille, repli sur les matchs à
//     objectif de la fenêtre (r_f est normalisé par les repères P80 de sa famille, donc
//     comparable d'une famille à l'autre) ; sous MinFamilyMatches même là, la métrique est
//     ABSENTE ;
//   - part : la part du joueur dans l'objectif de son équipe pour CE match
//     (narrative.ComputeObjectiveTeamShareWith). Avec UseSupportShare, max(part de l'objectif,
//     part du soutien), la part du soutien étant calculée sur les seules actions de soutien
//     de l'équipe (narrative.ObjectiveSupportColumns).
//
// L'objectif n'enlève jamais rien : note >= C. Une part nulle rend C + AlphaMin × max(0, O − C),
// soit la note du combat seul dès que O <= C.
//
// Métrique absente (titre sans match_objective_stats, match sans bloc d'objectif, référence
// trop courte) : la note est C, jamais une note à zéro d'objectif. Pas de « points requis pour
// gagner » : le résultat est déjà dans rank_perf.

import (
	"levelup/go-api/internal/analysis/narrative"
)

// objectiveBlendFacts : ce qu'un match porte pour la variante, posé par le chargeur.
type objectiveBlendFacts struct {
	Family narrative.ObjectiveFamily
	// Index : indice r_f BRUT du joueur pour ce match (>= 0, sans écrêtage).
	Index float64
	// Share : part du joueur dans l'objectif de son équipe, dans [0, 1].
	Share float64
	// SupportShare : sa part dans les seules actions de soutien de l'équipe, dans [0, 1].
	SupportShare float64
}

// objectiveBlendParams : les réglages de la variante.
type objectiveBlendParams struct {
	AlphaMin, AlphaMax float64
	// MinFamilyMatches : matchs de référence minimum, par famille puis en repli sur la
	// fenêtre de la chaîne.
	MinFamilyMatches int
	// UseSupportShare : α suit max(part de l'objectif, part du soutien) (variante d'escorte).
	UseSupportShare bool
}

// defaultObjectiveBlendParams : α de 0,12 (le poids actuel d'ospm) à 0,50 (D1 du user, le
// 2026-11-15), parts de l'objectif seules.
func defaultObjectiveBlendParams() objectiveBlendParams {
	return objectiveBlendParams{AlphaMin: 0.12, AlphaMax: 0.50, MinFamilyMatches: MinMatchesForRelative}
}

// objectiveBlendResult détaille une note de la variante (la simulation les rapporte).
type objectiveBlendResult struct {
	Score float64 // arrondi au dixième
	// Combat : C, non arrondi.
	Combat float64
	// HasObjective : false quand la métrique est absente (Score == Combat arrondi).
	HasObjective bool
	// Percentile : O, 0-100 ; Alpha : son poids ; Part : la part qui a réglé Alpha.
	Percentile, Alpha, Part float64
	// Floored : vrai quand O <= C, l'objectif n'ajoute alors rien.
	Floored bool
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
	part := current.ObjectiveBlend.Share
	if p.UseSupportShare {
		part = max(part, current.ObjectiveBlend.SupportShare)
	}
	part = clampF(part, 0, 1)
	alpha := p.AlphaMin + (p.AlphaMax-p.AlphaMin)*part
	pct := percentileRankMid(current.ObjectiveBlend.Index, series)
	res.HasObjective = true
	res.FamilyScope = familyScope
	res.Percentile = pct
	res.Alpha = alpha
	res.Part = part
	res.Floored = pct <= combat
	res.Score = roundPerfScore(combat + alpha*max(0, pct-combat))
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
