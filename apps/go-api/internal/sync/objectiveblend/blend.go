// Package objectiveblend — les VARIANTES de la note de performance des matchs à objectif
// (décision D-7 du plan des compléments 8.2.0, corrigée par les décisions du user des
// 2026-11-15), À L'ÉTUDE : aucun chargeur de production ne pose les faits, le batch de notes de
// internal/sync ne l'appelle pas et skill.PerformanceScoreRev n'a pas bougé. La simulation sur le
// parc (internal/sync, instrument `research`) les compare à la note actuelle avant que les poids
// ne soient figés.
//
// Le paquet est PUR : il reçoit la note de combat C déjà calculée et les faits des matchs, et
// rend la note ; le calcul de C (profil de la chaîne sans la métrique d'objectif) reste dans
// internal/sync.
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
//		note = (1 − α) × C + α × O_obj        α = AlphaMin + (AlphaMax − AlphaMin) × part
//
//	  - C : le combat, renormalisé sans la métrique d'objectif ;
//	  - O_obj : le rang de l'indice r_f BRUT du joueur (narrative.RawIndex, sans l'écrêtage de
//	    1,25 que les radars gardent pour l'affichage) parmi les matchs de la MÊME famille de mode
//	    de la fenêtre de la chaîne, les égalités classées à mi-rang (RankMid) ; sous
//	    MinFamilyMatches matchs de famille, repli sur les matchs à objectif de la fenêtre (r_f est
//	    normalisé par les repères P80 de sa famille, donc comparable d'une famille à l'autre) ;
//	    sous MinFamilyMatches même là, la métrique est ABSENTE ;
//	  - part : la part du joueur dans l'objectif de son équipe pour CE match
//	    (narrative.ComputeObjectiveTeamShare).
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
// # V7 (V6 + la part du soutien) : SupportShareRank
//
// O_sout devient, comme O_obj en V6, la moyenne de deux rangs à mi-rang : celui des points de
// soutien par minute et celui de la part du joueur dans les points de soutien de son équipe,
// chacun contre l'historique du joueur dans la même famille.
//
// Métrique absente (titre sans match_objective_stats, match sans bloc d'objectif, référence
// trop courte) : la note est C, jamais une note à zéro d'objectif. Pas de « points requis pour
// gagner » : le résultat est déjà dans rank_perf.
package objectiveblend

import (
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
)

// Facts : ce qu'un match porte pour les variantes, posé par le chargeur.
type Facts struct {
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

// FactsOf pose les faits d'un match depuis la part calculée par narrative et la durée jouée
// (analysis.PerMinute : durée inconnue = 0 point par minute).
func FactsOf(share narrative.ObjectiveTeamShare, playedSeconds float64) *Facts {
	return &Facts{
		Family: share.Family, Index: share.RawIndex, Share: share.Share, SupportShare: share.SupportShare,
		SupportPerMin: analysis.PerMinute(share.SupportPoints, playedSeconds),
	}
}

// Params : les réglages des variantes.
type Params struct {
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
	// SupportShareRank : V7, O_sout est la moyenne du rang des points de soutien par minute et du
	// rang de la part de soutien de l'équipe (suppose UseSupport).
	SupportShareRank bool
}

// DefaultParams : V2, alpha de 0,12 (le poids actuel d'ospm) à 0,50 (décision du user,
// 2026-11-15) ; minFamilyMatches est le seuil de matchs de référence de la note relative.
func DefaultParams(minFamilyMatches int) Params {
	return Params{AlphaMin: 0.12, AlphaMax: 0.50, SupportSlope: 0.38, MinFamilyMatches: minFamilyMatches}
}

// Result détaille une note des variantes (la simulation les rapporte).
type Result struct {
	// Score : la note, NON arrondie (l'appelant applique l'arrondi unique de la note stockée).
	Score float64
	// HasObjective : false quand la métrique est absente (Score == combat).
	HasObjective bool
	// Percentile : O_obj, 0-100 ; Alpha : son poids ; Part : la part qui a réglé Alpha.
	Percentile, Alpha, Part float64
	// SupportPercentile, SupportAlpha : O_sout et son poids (0 hors V5, ou sans référence).
	SupportPercentile, SupportAlpha float64
	// FamilyScope : vrai si la référence est la famille du match, faux en repli sur la chaîne.
	FamilyScope bool
}

// RankMid calcule le percentile d'une valeur à MI-RANG : les valeurs strictement inférieures
// comptent pour 1, les égales pour 1/2. Sur des égalités (indice d'objectif identique), le rang
// <= rendrait le rang le plus haut à tous ; ici elles se partagent le milieu de leur plage.
// Moins de deux valeurs : 50.
func RankMid(value float64, series []float64) float64 {
	if len(series) < 2 {
		return 50.0
	}
	var below, equal float64
	for _, v := range series {
		switch {
		case v < value:
			below++
		case v == value:
			equal++
		}
	}
	return min(max((below+equal/2)/float64(len(series))*100.0, 0), 100)
}

// reference rend les faits de référence : les matchs de la famille du match d'abord, les matchs
// à objectif de la fenêtre en repli. ok == false sous le minimum.
func reference(cur *Facts, history []*Facts, minMatches int) (facts []*Facts, familyScope, ok bool) {
	var inFamily, any []*Facts
	for _, f := range history {
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

// rank classe une valeur à mi-rang parmi un champ des faits de référence.
func rank(value float64, facts []*Facts, field func(*Facts) float64) float64 {
	series := make([]float64, len(facts))
	for i, f := range facts {
		series[i] = field(f)
	}
	return RankMid(value, series)
}

// Compute calcule la note d'un match depuis son combat C (0-100), ses faits et ceux des matchs
// de la fenêtre (history : un élément nil est un match sans faits). Sans faits pour le match ou
// sans référence suffisante, la métrique est absente : Score == combat.
func Compute(combat float64, cur *Facts, history []*Facts, p Params) Result {
	res := Result{Score: combat}
	if cur == nil {
		return res
	}
	ref, familyScope, hasRef := reference(cur, history, p.MinFamilyMatches)
	if !hasRef {
		return res
	}
	oObj := rank(cur.Index, ref, func(f *Facts) float64 { return f.Index })
	if p.ShareRank {
		oShare := rank(cur.Share, ref, func(f *Facts) float64 { return f.Share })
		oObj = (oObj + oShare) / 2
	}
	part := min(max(cur.Share, 0), 1)
	alphaObj := p.AlphaMin + (p.AlphaMax-p.AlphaMin)*part
	var alphaSup, oSup float64
	if p.UseSupport {
		alphaSup = p.SupportSlope * min(max(cur.SupportShare, 0), 1)
		oSup = rank(cur.SupportPerMin, ref, func(f *Facts) float64 { return f.SupportPerMin })
		if p.SupportShareRank {
			oSupShare := rank(cur.SupportShare, ref, func(f *Facts) float64 { return f.SupportShare })
			oSup = (oSup + oSupShare) / 2
		}
		if total := alphaObj + alphaSup; total > p.AlphaMax {
			alphaObj *= p.AlphaMax / total
			alphaSup *= p.AlphaMax / total
		}
	}
	res.HasObjective, res.FamilyScope = true, familyScope
	res.Percentile, res.Alpha, res.Part = oObj, alphaObj, part
	res.SupportPercentile, res.SupportAlpha = oSup, alphaSup
	res.Score = (1-alphaObj-alphaSup)*combat + alphaObj*oObj + alphaSup*oSup
	return res
}
