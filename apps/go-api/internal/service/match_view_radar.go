// Package service — match_view_radar.go : builder radar 6 axes pour MatchView
// (chunk MV4.B).
//
// Consomme les awards (personal_score_awards) chargés via
// port.PersonalScoreAwardsRepository et les agrège sur les 6 axes canoniques
// via narrative.ComputeParticipationProfile.
//
// Mapping award_name -> ParticipationAxis : table inline pour Halo Infinite
// (Phase 1 pilote). Les awards non-mappés sont ignorés. Les définitions
// title-specific viendront d'un TOML mappings (config/titles/{slug}/mappings/
// awards.toml) en Phase 2 ou 3.
package service

import (
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/teammates"
)

// haloAwardToAxis mappe les award_name connus de Halo Infinite vers les 6
// axes canoniques. Couverture initiale Phase 1 pilote (MV4.B).
//
// Audit Python (apps/api/app/services/match_view_service.py) : les awards
// sont historiquement groupés par catégorie. Le portage Go reprend les
// catégories classiques :
//
//	combat    : kills, double_kills, triple_kills, etc.
//	survival  : avoided_kills, escapes
//	support   : assists, ally_protections, revives
//	score     : personal_score, scoring streaks
//	objective : flag_caps, hill_captures, oddball_kills, neutralizations
//	impact    : multi-kills, sprees, MVP-equivalents
//
// Awards non listés ici sont ignorés (Phase 1 — UX dégradée mais lisible).
// Phase 2 : passer par config/titles/halo_infinite/mappings/awards.toml.
var haloAwardToAxis = map[string]narrative.ParticipationAxis{
	// Combat (kills divers)
	analysis.StatLabelKills: narrative.AxisCombat,
	"double_kill":           narrative.AxisCombat,
	"triple_kill":           narrative.AxisCombat,
	"overkill":              narrative.AxisCombat,
	"killtacular":           narrative.AxisCombat,
	"killtrocity":           narrative.AxisCombat,
	"killimanjaro":          narrative.AxisCombat,
	"killtastrophe":         narrative.AxisCombat,
	"killpocalypse":         narrative.AxisCombat,
	"killionaire":           narrative.AxisCombat,
	"power_kill":            narrative.AxisCombat,
	"power_combo":           narrative.AxisCombat,
	"perfect_kill":          narrative.AxisCombat,
	"headshot":              narrative.AxisCombat,
	"melee_kill":            narrative.AxisCombat,
	"grenade_kill":          narrative.AxisCombat,
	"sniper_kill":           narrative.AxisCombat,
	"close_call":            narrative.AxisCombat,
	"highest_kills":         narrative.AxisCombat,

	// Survival (évitement / longévité)
	"avoided_kill":     narrative.AxisSurvival,
	"escape":           narrative.AxisSurvival,
	"survived":         narrative.AxisSurvival,
	"longest_lifespan": narrative.AxisSurvival,
	"avg_lifespan":     narrative.AxisSurvival,

	// Support (aide aux alliés)
	"assist":          narrative.AxisSupport,
	"emp_assist":      narrative.AxisSupport,
	"savior_kill":     narrative.AxisSupport,
	"protection":      narrative.AxisSupport,
	"protector":       narrative.AxisSupport,
	"highest_assists": narrative.AxisSupport,
	"team_player":     narrative.AxisSupport,

	// Score (objectif personnel)
	"personal_score": narrative.AxisScore,
	"highest_score":  narrative.AxisScore,
	"score_streak":   narrative.AxisScore,

	// Objective (mode-specific)
	"flag_capture":      narrative.AxisObjective,
	"flag_kill":         narrative.AxisObjective,
	"flag_return":       narrative.AxisObjective,
	"hill_capture":      narrative.AxisObjective,
	"hill_score":        narrative.AxisObjective,
	"oddball_score":     narrative.AxisObjective,
	"oddball_kill":      narrative.AxisObjective,
	"neutralization":    narrative.AxisObjective,
	"strongholds_score": narrative.AxisObjective,

	// Impact (multi-kills hauts ou MVP-equivalents)
	"killing_spree":  narrative.AxisImpact,
	"killing_frenzy": narrative.AxisImpact,
	"running_riot":   narrative.AxisImpact,
	"rampage":        narrative.AxisImpact,
	"untouchable":    narrative.AxisImpact,
	"invincible":     narrative.AxisImpact,
	"mvp":            narrative.AxisImpact,
	"top_score":      narrative.AxisImpact,
}

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

// BuildMatchRadar construit les séries radar pour les joueurs du scoreboard
// à partir des awards chargés via port.PersonalScoreAwardsRepository.
//
//	awards     : []PersonalScoreAwardRow déjà filtré sur le match courant.
//	scoreboard : utilisé pour résoudre xuid -> gamertag dans le payload.
//	modeFamily : "slayer" / "ctf" / "strongholds" / "oddball" / "".
//
// Retourne 1 série par xuid présent dans awards (l'ordre suit le scoreboard).
// Joueurs sans aucun award : skippés (pas de bruit dans le radar).
func BuildMatchRadar(
	awards []port.PersonalScoreAwardRow,
	scoreboard []domain.ScoreboardRaw,
	modeFamily string,
) []MatchViewRadarSeries {
	if len(awards) == 0 {
		return nil
	}

	// Aggréger awards par (xuid, axis).
	byXUIDAxis := make(map[string]map[narrative.ParticipationAxis]float64)
	for _, a := range awards {
		axis, ok := haloAwardToAxis[a.AwardName]
		if !ok {
			continue
		}
		if byXUIDAxis[a.XUID] == nil {
			byXUIDAxis[a.XUID] = make(map[narrative.ParticipationAxis]float64)
		}
		byXUIDAxis[a.XUID][axis] += float64(a.Total)
	}
	if len(byXUIDAxis) == 0 {
		return nil
	}

	// Résoudre xuid -> gamertag via scoreboard pour stabilité d'affichage.
	gtByXUID := make(map[string]string, len(scoreboard))
	xuidOrder := make([]string, 0, len(scoreboard))
	for _, row := range scoreboard {
		if row.XUID == "" {
			continue
		}
		if _, exists := gtByXUID[row.XUID]; !exists {
			gtByXUID[row.XUID] = row.Gamertag
			xuidOrder = append(xuidOrder, row.XUID)
		}
	}

	thresholds := narrative.DefaultThresholds(modeFamily)
	out := make([]MatchViewRadarSeries, 0, len(byXUIDAxis))
	for _, xuid := range xuidOrder {
		raw, ok := byXUIDAxis[xuid]
		if !ok {
			continue
		}
		axes := narrative.ComputeParticipationProfile(raw, thresholds)
		out = append(out, MatchViewRadarSeries{
			XUID:       xuid,
			Gamertag:   gtByXUID[xuid],
			Axes:       axes,
			ModeFamily: modeFamily,
		})
	}
	return out
}

// BuildMatchRadarFromScoreboard construit la série radar 6 axes pour le
// JOUEUR ACTIF uniquement (myXUID). Calculée depuis les colonnes
// match_participants (déjà chargées dans ScoreboardRaw) + le score
// d'objectif PSA (catégorie 'objective'). Mêmes formules que le radar
// squad (loadSynergyMateAxes), appliquées à un seul match.
//
// objectiveScore : somme des award_score PSA catégorie 'objective'.
// 0 si la table est absente ou pas de données pour ce match → l'axe
// Objective reste à 0 et le résiduel Score n'est pas amputé.
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
		Objective: 350.0,
		Impact:    offensiveConversionP80 * 1.25,
	}

	raw := computeMatchRadarRawAxes(*me, objectiveScore, effectiveHpToKill)
	axes := narrative.ComputeParticipationProfile(raw, thresholds)
	axes = dropUncomputableRadarAxes(axes, *me)
	return []MatchViewRadarSeries{{
		XUID:       me.XUID,
		Gamertag:   me.Gamertag,
		Axes:       axes,
		ModeFamily: modeFamily,
	}}
}

// computeMatchRadarRawAxes calcule les valeurs brutes par axe pour 1 joueur
// sur 1 match. Reprend les formules de loadSynergyMateAxes (squad).
// objectiveScore : score PSA catégorie 'objective' (0 si non disponible).
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
	raw[narrative.AxisObjective] = float64(objectiveScore)

	// Score résiduel = personal_score − kills×100 − assists×50 − objectif (medals/streaks).
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
//   - Impact (rendement offensif) sans damage_dealt.
func dropUncomputableRadarAxes(axes []narrative.ParticipationScore, me domain.ScoreboardRaw) []narrative.ParticipationScore {
	drop := map[narrative.ParticipationAxis]bool{}
	if me.DamageTaken == nil {
		drop[narrative.AxisSurvival] = true
	}
	if me.DamageDealt == nil {
		drop[narrative.AxisImpact] = true
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
