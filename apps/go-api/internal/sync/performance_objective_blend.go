package sync

// performance_objective_blend.go — VARIANTES de la note de performance des matchs à objectif
// (décision D-7 du plan des compléments 8.2.0, corrigée par les décisions du user des
// 2026-11-15), À L'ÉTUDE : aucun chargeur de production ne pose historyRow.ObjectiveBlend,
// batchComputePerformanceScores ne l'appelle pas et skill.PerformanceScoreRev n'a pas bougé. La
// simulation sur le parc (performance_objective_simulation_research_test.go) les compare à la
// note actuelle avant que les poids ne soient figés.
//
// # Ce qui change par rapport à la note actuelle
//
// Aujourd'hui, sur les chaînes à objectif, la métrique `objective_participation` (ospm) pèse
// 0,12 sur 1,04 et vaut les points d'awards `objective` par minute : un porteur de drapeau
// sans frag sort avec une note faible, parce que le reste du profil est du combat. Ici, des
// moyennes SYMÉTRIQUES — l'objectif peut monter ou baisser la note, aucune règle à sens unique,
// aucun plancher.
//
// # V2 (la base retenue)
//
//	note = (1 − α) × C + α × O_obj        α = AlphaMin + (AlphaMax − AlphaMin) × part
//
//   - C : le combat — le profil de poids de la chaîne SANS la métrique d'objectif, renormalisé
//     par la somme des poids restants (weightedPercentileMean) ;
//   - O_obj : le rang de l'indice r_f BRUT du joueur (narrative.RawIndex, sans l'écrêtage de
//     1,25 que les radars gardent pour l'affichage) parmi les matchs de la MÊME famille de mode
//     de la fenêtre de la chaîne, les égalités classées à mi-rang (percentileRankMid) ; sous
//     MinFamilyMatches matchs de famille, repli sur les matchs à objectif de la fenêtre (r_f est
//     normalisé par les repères P80 de sa famille, donc comparable d'une famille à l'autre) ;
//     sous MinFamilyMatches même là, la métrique est ABSENTE ;
//   - part : la part du joueur dans l'objectif de son équipe pour CE match
//     (narrative.ComputeObjectiveTeamShare).
//
// # V5 (V2 + le soutien) : UseSupport
//
// Une mesure d'objectif propre au soutien, indépendante du temps de port : les points de
// soutien du match par minute (narrative.ObjectiveSupportPoints, analysis.PerMinute), rangés à
// mi-rang parmi les matchs de la même famille du joueur (O_sout), pesant
// α_sout = SupportSlope × part du joueur dans les points de soutien de son équipe :
//
//	note = (1 − α_obj − α_sout) × C + α_obj × O_obj + α_sout × O_sout
//
// α_obj + α_sout est plafonné à AlphaMax en réduisant les deux en proportion : le combat garde
// toujours au moins 1 − AlphaMax du poids. Sans référence de soutien, α_sout = 0 (V2).
//
// # V6 (V5 + la part) : ShareRank
//
// O_obj devient la moyenne de deux rangs à mi-rang : celui de l'indice brut et celui de la
// part de l'objectif de l'équipe, chacun contre l'historique du joueur dans la même famille.
// Celui qui porte seul une équipe peu active à l'objectif n'est plus puni par un indice brut
// que sa propre histoire (plus active) écrase.
//
// Métrique absente (titre sans match_objective_stats, match sans bloc d'objectif, référence
// trop courte) : la note est C, jamais une note à zéro d'objectif. Pas de « points requis pour
// gagner » : le résultat est déjà dans rank_perf.

import (
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
)

// objectiveBlendFacts : ce qu'un match porte pour les variantes, posé par le chargeur.
type objectiveBlendFacts struct {
	Family narrative.ObjectiveFamily
	// Index : indice r_f BRUT du joueur pour ce match (>= 0, sans écrêtage).
	Index float64
	// Share : part du joueur dans l'objectif de son équipe, dans [0, 1].
	Share float64
	// SupportShare : sa part dans les points de soutien de son équipe, dans [0, 1].
	SupportShare float64
	// SupportPerMin : ses points de soutien par minute jouée.
	SupportPerMin float64
}

// objectiveBlendFactsOf pose les faits d'un match depuis la part calculée par narrative et la
// durée jouée (analysis.PerMinute : durée inconnue = 0 point par minute).
func objectiveBlendFactsOf(share narrative.ObjectiveTeamShare, playedSeconds float64) *objectiveBlendFacts {
	return &objectiveBlendFacts{
		Family: share.Family, Index: share.RawIndex, Share: share.Share, SupportShare: share.SupportShare,
		SupportPerMin: analysis.PerMinute(share.SupportPoints, playedSeconds),
	}
}

// objectiveBlendParams : les réglages des variantes.
type objectiveBlendParams struct {
	// AlphaMin, AlphaMax : poids de l'objectif à part nulle et à part pleine ; AlphaMax plafonne
	// aussi α_obj + α_sout.
	AlphaMin, AlphaMax float64
	// SupportSlope : poids du soutien à part de soutien pleine (V5).
	SupportSlope float64
	// MinFamilyMatches : matchs de référence minimum, par famille puis en repli sur la
	// fenêtre de la chaîne.
	MinFamilyMatches int
	// UseSupport : V5, le soutien pèse à côté de l'objectif.
	UseSupport bool
	// ShareRank : V6, O_obj est la moyenne du rang de l'indice et du rang de la part.
	ShareRank bool
}

// defaultObjectiveBlendParams : V2, alpha de 0,12 (le poids actuel d'ospm) à 0,50 (décision du
// user, 2026-11-15).
func defaultObjectiveBlendParams() objectiveBlendParams {
	return objectiveBlendParams{AlphaMin: 0.12, AlphaMax: 0.50, SupportSlope: 0.38, MinFamilyMatches: MinMatchesForRelative}
}

// objectiveBlendResult détaille une note des variantes (la simulation les rapporte).
type objectiveBlendResult struct {
	Score float64 // arrondi au dixième
	// Combat : C, non arrondi.
	Combat float64
	// HasObjective : false quand la métrique est absente (Score == Combat arrondi).
	HasObjective bool
	// Percentile : O_obj, 0-100 ; Alpha : son poids ; Part : la part qui a réglé Alpha.
	Percentile, Alpha, Part float64
	// SupportPercentile, SupportAlpha : O_sout et son poids (0 hors V5, ou sans référence).
	SupportPercentile, SupportAlpha float64
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

// objectiveReferenceFacts rend les faits de référence : les matchs de la famille du match
// d'abord, les matchs à objectif de la fenêtre en repli. ok == false sous le minimum.
func objectiveReferenceFacts(cur *objectiveBlendFacts, history []historyRow, minMatches int) (facts []*objectiveBlendFacts, familyScope, ok bool) {
	var inFamily, any []*objectiveBlendFacts
	for i := range history {
		f := history[i].ObjectiveBlend
		if f == nil {
			continue
		}
		any = append(any, f)
		if f.Family == cur.Family {
			inFamily = append(inFamily, f)
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

// objectiveRank classe une valeur à mi-rang parmi un champ des faits de référence.
func objectiveRank(value float64, facts []*objectiveBlendFacts, field func(*objectiveBlendFacts) float64) float64 {
	series := make([]float64, len(facts))
	for i, f := range facts {
		series[i] = field(f)
	}
	return percentileRankMid(value, series)
}

// computeObjectiveBlend calcule la note des variantes pour un match. `weights` est le profil
// de la chaîne (WeightsForChain) : les variantes ne valent que pour un profil qui porte la
// métrique d'objectif ; sur tout autre profil, ok == false (l'appelant garde la note actuelle).
// ok == false aussi sous MinMatchesForRelative matchs d'historique ou sans aucune métrique de
// combat calculable.
func computeObjectiveBlend(current *historyRow, history []historyRow, weights map[string]float64, p objectiveBlendParams) (objectiveBlendResult, bool) {
	if _, inProfile := profileWeight(weights, MetricKeyObjectiveParticipation); !inProfile {
		return objectiveBlendResult{}, false
	}
	combat, ok := weightedPercentileMean(current, history, profileWithoutObjective(weights))
	if !ok {
		return objectiveBlendResult{}, false
	}
	res := objectiveBlendResult{Combat: combat, Score: roundPerfScore(combat)}
	cur := current.ObjectiveBlend
	if cur == nil {
		return res, true
	}
	ref, familyScope, hasRef := objectiveReferenceFacts(cur, history, p.MinFamilyMatches)
	if !hasRef {
		return res, true
	}
	oObj := objectiveRank(cur.Index, ref, func(f *objectiveBlendFacts) float64 { return f.Index })
	if p.ShareRank {
		oShare := objectiveRank(cur.Share, ref, func(f *objectiveBlendFacts) float64 { return f.Share })
		oObj = (oObj + oShare) / 2
	}
	part := clampF(cur.Share, 0, 1)
	alphaObj := p.AlphaMin + (p.AlphaMax-p.AlphaMin)*part
	var alphaSup, oSup float64
	if p.UseSupport {
		alphaSup = p.SupportSlope * clampF(cur.SupportShare, 0, 1)
		oSup = objectiveRank(cur.SupportPerMin, ref, func(f *objectiveBlendFacts) float64 { return f.SupportPerMin })
		if total := alphaObj + alphaSup; total > p.AlphaMax {
			alphaObj *= p.AlphaMax / total
			alphaSup *= p.AlphaMax / total
		}
	}
	res.HasObjective, res.FamilyScope = true, familyScope
	res.Percentile, res.Alpha, res.Part = oObj, alphaObj, part
	res.SupportPercentile, res.SupportAlpha = oSup, alphaSup
	res.Score = roundPerfScore((1-alphaObj-alphaSup)*combat + alphaObj*oObj + alphaSup*oSup)
	return res, true
}
