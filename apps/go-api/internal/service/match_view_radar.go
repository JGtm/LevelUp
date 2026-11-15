// Package service — match_view_radar.go : radar de participation du joueur actif pour MatchView,
// calculé depuis la feuille de match (colonnes match_participants + bloc objectif de la ligne Q12,
// déjà chargés dans ScoreboardRaw — zéro requête en plus), sur les axes canoniques de
// narrative.ComputeParticipationProfile.
package service

import (
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/service/teammates"
)

// MatchViewRadarSeries est la série radar exposée dans le DTO MatchView.
//
// Mirror frontend du narrative.ParticipationScore agrégé. Le wrapper
// `<RadarChart>` (S10) côté front consomme N séries (1 par joueur du match)
// avec 6 axes alignés.
type MatchViewRadarSeries struct {
	XUID     string                         `json:"xuid"`
	Gamertag string                         `json:"gamertag,omitempty"`
	Axes     []narrative.ParticipationScore `json:"axes"`
	// ModeFamily : famille de mode utilisée pour les seuils
	// (slayer / ctf / strongholds / oddball / "" custom).
	ModeFamily string `json:"mode_family,omitempty"`
}

// BuildMatchRadarFromScoreboard construit la série radar pour le JOUEUR ACTIF
// uniquement (myXUID). Calculée depuis les colonnes match_participants + le bloc
// objectif de la ligne Q12 (déjà chargés dans ScoreboardRaw — zéro requête en
// plus). Mêmes formules que le radar squad (loadSynergyMateAxes), appliquées à
// un seul match.
//
// objectiveScore : somme des award_score PSA catégorie 'objective'. Conservé
// UNIQUEMENT pour le résiduel de l'axe Score (décision 1 du plan
// PLAN_AXE_OBJECTIFS_INDEX). L'axe Objectif est la note d'objectif DU MATCH SEUL
// (matchObjectiveScore) : la part du joueur dans l'objectif de son équipe, calculée sur
// les lignes de TOUTE la feuille de match, sans comparaison à l'historique du joueur. Il
// est RETIRÉ (jamais 0) quand le match n'a aucune donnée d'objectif mesurable : Slayer,
// titre sans match.objective.stats (Halo 5), joueur sans ligne ni camp.
//
// Retourne une slice de 1 élément (ou nil si myXUID absent du scoreboard).
func BuildMatchRadarFromScoreboard(
	scoreboard []domain.ScoreboardRaw,
	myXUID string,
	objectiveScore int,
	modeFamily string,
	effectiveHpToKill float64,
	offensiveConversionP80 float64,
) []MatchViewRadarSeries {
	if len(scoreboard) == 0 || myXUID == "" {
		return nil
	}

	var me *domain.ScoreboardRaw
	for i := range scoreboard {
		if scoreboard[i].XUID == myXUID {
			me = &scoreboard[i]
			break
		}
	}
	if me == nil {
		return nil
	}

	// Mêmes seuils que synergyRadarThresholds(1) — calibration single-match.
	// Impact (rendement OC) = P80 title-aware ; Survival (DR) = const (h5 sans DR).
	thresholds := narrative.ParticipationThresholds{
		Combat:    25.0,
		Survival:  analysis.DefensiveResistanceP80 * 1.25,
		Support:   300.0,
		Score:     350.0,
		Objective: narrative.ObjectiveScoreMax,
		Impact:    offensiveConversionP80 * 1.25,
	}

	raw := computeMatchRadarRawAxes(*me, objectiveScore, effectiveHpToKill)
	objScore, objMeasured := matchObjectiveScore(scoreboard, myXUID)
	if objMeasured {
		raw[narrative.AxisObjective] = objScore
	}
	axes := narrative.ComputeParticipationProfile(raw, thresholds)
	axes = dropUncomputableRadarAxes(axes, *me, objMeasured)
	return []MatchViewRadarSeries{{
		XUID:       me.XUID,
		Gamertag:   me.Gamertag,
		Axes:       axes,
		ModeFamily: modeFamily,
	}}
}

// computeMatchRadarRawAxes calcule les valeurs brutes par axe pour 1 joueur
// sur 1 match (hors axe Objectif — note d'objectif du match posée par le caller).
// objectiveScore : score PSA catégorie 'objective' (0 si non disponible) —
// consommé UNIQUEMENT par le résiduel de l'axe Score.
func computeMatchRadarRawAxes(row domain.ScoreboardRaw, objectiveScore int, effectiveHpToKill float64) map[narrative.ParticipationAxis]float64 {
	raw := map[narrative.ParticipationAxis]float64{}

	hs := 0
	if row.HeadshotKills != nil {
		hs = *row.HeadshotKills
	}
	pk := row.PerfectKills
	acc := 0.0
	if row.Accuracy != nil {
		acc = *row.Accuracy / 100.0 // DB stocke 0..100, formule attend 0..1
	}
	dd := 0.0
	if row.DamageDealt != nil {
		dd = *row.DamageDealt
	}
	dt := 0.0
	if row.DamageTaken != nil {
		dt = *row.DamageTaken
	}
	ps := 0.0
	if row.PersonalScore != nil {
		ps = *row.PersonalScore
	}

	raw[narrative.AxisCombat] = (float64(row.Kills) + 0.5*float64(hs) + 0.5*float64(pk)) * (1.0 + acc*0.4)
	raw[narrative.AxisSupport] = float64(row.Assists) * 50.0
	raw[narrative.AxisImpact] = teammates.SynergyOffensiveConversion(row.Kills, row.Assists, dd, effectiveHpToKill)
	raw[narrative.AxisSurvival] = teammates.SynergyDefensiveResistance(dt, row.Deaths, effectiveHpToKill)

	// Score résiduel = personal_score − kills×100 − assists×50 − objectif (medals/streaks).
	// Le terme PSA objectif reste soustrait ici (décision 1 : chargement PSA conservé
	// pour ce seul résiduel).
	residual := ps - float64(row.Kills)*100.0 - float64(row.Assists)*50.0 - float64(objectiveScore)
	if residual < 0 {
		residual = 0
	}
	raw[narrative.AxisScore] = residual
	return raw
}

// dropUncomputableRadarAxes retire les axes structurellement non calculables
// faute de donnée source, pour éviter d'afficher un axe trompeur figé à 0 (le
// front rend dynamiquement la liste reçue : N axes au lieu de 6). Règle
// title-agnostic, pilotée par la donnée du joueur :
//   - Survie (résistance défensive) sans damage_taken → cas Halo 5, qui ne
//     fournit pas les dégâts subis ;
//   - Impact (rendement offensif) sans damage_dealt ;
//   - Objectif non mesuré (match Slayer, titre sans match.objective.stats, joueur sans
//     ligne d'objectif ni camp dans la feuille) → la note d'objectif n'est pas calculable.
func dropUncomputableRadarAxes(axes []narrative.ParticipationScore, me domain.ScoreboardRaw, objectiveMeasured bool) []narrative.ParticipationScore {
	drop := map[narrative.ParticipationAxis]bool{}
	if me.DamageTaken == nil {
		drop[narrative.AxisSurvival] = true
	}
	if me.DamageDealt == nil {
		drop[narrative.AxisImpact] = true
	}
	if !objectiveMeasured {
		drop[narrative.AxisObjective] = true
	}
	if len(drop) == 0 {
		return axes
	}
	kept := axes[:0]
	for _, a := range axes {
		if !drop[a.Axis] {
			kept = append(kept, a)
		}
	}
	return kept
}
