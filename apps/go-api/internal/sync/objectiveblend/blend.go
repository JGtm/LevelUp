// Package objectiveblend — la note de performance des matchs à objectif, retenue après simulation sur
// le parc (ADR 0047).
//
// Le paquet est PUR : il reçoit la note de combat C déjà calculée et les faits des matchs, et
// rend la note ; le calcul de C (profil de la chaîne, sans métrique d'objectif) et le chargement
// des faits restent dans internal/sync.
//
// # La note
//
// Le profil de la chaîne ne porte que du combat : l'objectif et le soutien de l'équipe se greffent
// dessus par une moyenne SYMÉTRIQUE — ils peuvent monter ou baisser la note, aucune règle à sens
// unique, aucun plancher. Un porteur de drapeau sans frag n'est ainsi pas jugé sur le seul combat :
//
//		note = (1 − α_obj − α_sout) × C + α_obj × O_obj + α_sout × O_sout
//
//	  - C : le combat, le profil de poids de la chaîne renormalisé ;
//	  - O_obj : la moyenne de deux rangs à mi-rang (RankMid) du match, contre l'historique du joueur
//	    dans la MÊME famille de mode de la fenêtre : celui de l'indice r_f BRUT (narrative.RawIndex,
//	    sans l'écrêtage de 1,25 que les radars gardent pour l'affichage) et celui de la part du
//	    joueur dans l'objectif de son équipe (narrative.ComputeObjectiveTeamShare). Celui qui porte
//	    seul une équipe peu active à l'objectif n'est plus puni par un indice que sa propre histoire
//	    écrase ;
//	  - O_sout : de même, la moyenne du rang des points de soutien par minute
//	    (narrative.ObjectiveSupportPoints, analysis.PerMinute) et du rang de la part du joueur dans
//	    les points de soutien de son équipe. Une mesure propre au soutien, indépendante du temps de
//	    port. Le débit par minute n'est classé que contre les matchs de la MÊME famille : en repli
//	    sur les autres familles (ci-dessous), O_sout ne garde que le rang de la part ;
//	  - α_obj = AlphaMin + (AlphaMax − AlphaMin) × part de l'objectif ; α_sout = SupportSlope × part
//	    du soutien ; leur somme est plafonnée à AlphaMax en réduisant les deux en proportion : le
//	    combat garde toujours au moins 1 − AlphaMax du poids.
//
// Sous MinFamilyMatches matchs de famille dans la fenêtre, repli sur les matchs à objectif de la
// fenêtre (r_f est normalisé par les repères P80 de sa famille, donc comparable d'une famille à
// l'autre) ; sous MinFamilyMatches même là, la métrique est ABSENTE. Métrique absente (titre sans
// match_objective_stats, match sans bloc d'objectif, référence trop courte) : la note est C, jamais
// une note à zéro d'objectif. Pas de « points requis pour gagner » : le résultat est déjà dans
// rank_perf.
package objectiveblend

import (
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
)

// Facts : ce qu'un match porte pour la note, posé par le chargeur.
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

// Params : les réglages de la note.
type Params struct {
	// AlphaMin, AlphaMax : poids de l'objectif à part nulle et à part pleine ; AlphaMax plafonne
	// aussi α_obj + α_sout.
	AlphaMin, AlphaMax float64
	// SupportSlope : poids du soutien à part de soutien pleine.
	SupportSlope float64
	// MinFamilyMatches : matchs de référence minimum, par famille puis en repli sur la fenêtre de
	// la chaîne.
	MinFamilyMatches int
}

// DefaultParams : alpha de 0,12 (l'ancien poids d'ospm) à 0,50, soutien jusqu'à 0,38 ; minFamilyMatches
// est le seuil de matchs de référence de la note relative.
func DefaultParams(minFamilyMatches int) Params {
	return Params{AlphaMin: 0.12, AlphaMax: 0.50, SupportSlope: 0.38, MinFamilyMatches: minFamilyMatches}
}

// Result détaille une note (la simulation et les tests la rapportent).
type Result struct {
	// Score : la note, NON arrondie (l'appelant applique l'arrondi unique de la note stockée).
	Score float64
	// HasObjective : false quand la métrique est absente (Score == combat).
	HasObjective bool
	// Percentile : O_obj, 0-100 ; Alpha : son poids ; Part : la part qui a réglé Alpha.
	Percentile, Alpha, Part float64
	// SupportPercentile, SupportAlpha : O_sout et son poids.
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
	oObj := (rank(cur.Index, ref, func(f *Facts) float64 { return f.Index }) +
		rank(cur.Share, ref, func(f *Facts) float64 { return f.Share })) / 2
	// Le débit de soutien (points par minute) n'est comparable qu'entre matchs de la MÊME famille :
	// en repli sur les autres familles, O_sout ne garde que le rang de la part du soutien.
	oSup := rank(cur.SupportShare, ref, func(f *Facts) float64 { return f.SupportShare })
	if familyScope {
		oSup = (oSup + rank(cur.SupportPerMin, ref, func(f *Facts) float64 { return f.SupportPerMin })) / 2
	}
	part := min(max(cur.Share, 0), 1)
	alphaObj := p.AlphaMin + (p.AlphaMax-p.AlphaMin)*part
	alphaSup := p.SupportSlope * min(max(cur.SupportShare, 0), 1)
	if total := alphaObj + alphaSup; total > p.AlphaMax {
		alphaObj *= p.AlphaMax / total
		alphaSup *= p.AlphaMax / total
	}
	res.HasObjective, res.FamilyScope = true, familyScope
	res.Percentile, res.Alpha, res.Part = oObj, alphaObj, part
	res.SupportPercentile, res.SupportAlpha = oSup, alphaSup
	res.Score = (1-alphaObj-alphaSup)*combat + alphaObj*oObj + alphaSup*oSup
	return res
}
