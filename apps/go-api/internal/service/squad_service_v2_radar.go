// Package service — squad_service_v2_radar.go : helper Radar 6 axes pour la
// page Squad V2 (cf. PLAN_SQUAD_GO_PORTAGE Phase P8, section 15 audit).
//
// Construit 1 trace par joueur du squad sur les 6 axes canoniques :
// Combat / Survival / Support / Score / Objective / Impact (cf.
// narrative.AllParticipationAxes()). Chaque axe est normalise [0..100] via
// narrative.ComputeParticipationProfile + thresholds par famille de mode.
//
// Strategie d'agregation des axes pour la Phase 1 pilote :
//
//	Combat    = sum(kills)         / N_matches
//	Survival  = avg(avg_life_secs) sur les rows ou disponible
//	Support   = sum(assists)       / N_matches
//	Score     = sum(personal_score) / N_matches
//	Objective = 0 (proxy en attendant le repo PersonalScoreAwardsRepo, audit § 7.X)
//	Impact    = avg(performance_score) sur les rows ou disponible
//
// L'axe Objective restera a 0 jusqu'a la livraison du repo awards (Phase 2
// ou chunk dedie post-S9). Le commentaire JSDoc cote front explicite le
// behavior pour eviter qu'un utilisateur le lise comme "absence d'objectif".
package service

import (
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/games/canonical"
)

// RadarChartSeries est le payload de la serie Radar :
//
//	Key       : "squad.radar.<gamertag>" stable, utilisable comme cle React.
//	LabelKey  : cle i18n du libelle (gamertag-friendly).
//	Axes      : 6 elements ordonnees par AllParticipationAxes() (Combat...Impact).
//	Meta      : metadonnees (mode_family, raw_by_axis pour debug).
//
// Le wrapper <Radar> cote front s'attend a une serie par joueur, axes alignes
// (les axes sont identiques pour tous les joueurs).
type RadarChartSeries struct {
	Key      string                         `json:"key"`
	LabelKey string                         `json:"label_key"`
	Axes     []narrative.ParticipationScore `json:"axes"`
	Meta     map[string]any                 `json:"meta,omitempty"`
}

// BuildRadarSeries construit 1 trace radar par joueur du squad.
//
// modeFamily : "slayer" / "ctf" / "strongholds" / "oddball" / "" (default).
// Calibre les seuils via narrative.DefaultThresholds. La detection de la
// famille majoritaire est laissee au service amont (peut etre passee depuis
// la playlist principale de la session).
func BuildRadarSeries(
	rowsByPlayer map[string][]canonical.PlayerMatchRow,
	modeFamily string,
) []RadarChartSeries {
	if len(rowsByPlayer) == 0 {
		return nil
	}
	thresholds := narrative.DefaultThresholds(modeFamily)
	gts := sortedGamertags(rowsByPlayer)

	out := make([]RadarChartSeries, 0, len(gts))
	for _, gt := range gts {
		raw := aggregateAxesFromRows(rowsByPlayer[gt])
		axes := narrative.ComputeParticipationProfile(raw, thresholds)

		// Reformater raw_by_axis en clefs JSON-friendly (sans le prefix type).
		rawDebug := make(map[string]float64, len(raw))
		for axis, v := range raw {
			rawDebug[string(axis)] = v
		}

		out = append(out, RadarChartSeries{
			Key:      "squad.radar." + gt,
			LabelKey: "squad.radar.player",
			Axes:     axes,
			Meta: map[string]any{
				chartMetaGamertag: gt,
				"mode_family":     modeFamily,
				"raw_by_axis":     rawDebug,
			},
		})
	}
	return out
}

// aggregateAxesFromRows agrege les rows en valeurs par axe.
//
// Les rows sans valeur sur un axe ne contribuent pas (n'introduit pas de
// biais zero). Pour Combat / Support / Score, on somme et on divise par
// N_matches (moyenne implicite). Pour Survival / Impact, on calcule une
// moyenne sur les rows ou la valeur est disponible.
func aggregateAxesFromRows(rows []canonical.PlayerMatchRow) map[narrative.ParticipationAxis]float64 {
	out := make(map[narrative.ParticipationAxis]float64, 6)
	if len(rows) == 0 {
		return out
	}
	n := float64(len(rows))

	var totalKills, totalAssists, totalScore int
	var sumLife, sumPerf float64
	var nLife, nPerf int

	for _, r := range rows {
		totalKills += derefInt(r.Self.Kills)
		totalAssists += derefInt(r.Self.Assists)
		totalScore += derefInt(r.Self.PersonalScore)
		if r.Self.AvgLifeSeconds != nil {
			sumLife += *r.Self.AvgLifeSeconds
			nLife++
		}
		if r.Enrichment.PerformanceScore != nil {
			sumPerf += *r.Enrichment.PerformanceScore
			nPerf++
		}
	}

	out[narrative.AxisCombat] = float64(totalKills) / n
	out[narrative.AxisSupport] = float64(totalAssists) / n
	out[narrative.AxisScore] = float64(totalScore) / n
	out[narrative.AxisObjective] = 0 // deferred (cf. doc fichier)
	if nLife > 0 {
		out[narrative.AxisSurvival] = sumLife / float64(nLife)
	}
	if nPerf > 0 {
		out[narrative.AxisImpact] = sumPerf / float64(nPerf)
	}
	return out
}
