// Package service - teammates_squad_charts_synergy.go : builders teammates
// synergy radar (6 axes : participation, frags, defensive, kills/min,
// assists/min, conversion offensive/defensive) + helpers. Decoupe de
// teammates_squad_charts.go (god-file split, refactor 2026-05-27).
package teammates

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// ScorePerMinuteP80 est la frontière élite (80e percentile) du score personnel
// par minute jouée, repère de normalisation de l'axe Score du radar. Mesuré sur
// shared_matches_v2 Halo Infinite (26 258 performances joueur×match,
// time_played > 30 s) : P80 ≈ 195/min (p50≈121, p90≈240, p95≈284). Comme les
// axes ratio (Survie, Impact), l'axe est normalisé par un seuil CONSTANT (pas
// n-échelonné) : un joueur au P80 marque 80/100, +25 % au-dessus plafonne à 100.
// Const Infinite-calibrée (comme DefensiveResistanceP80) ; le threading
// title-aware serait une passe dédiée (cf. Impact via games.OffensiveConversionP80).
const ScorePerMinuteP80 = 195.0

func synergyRadarThresholds(nShared int, ocP80 float64) narrative.ParticipationThresholds {
	n := float64(nShared)
	return narrative.ParticipationThresholds{
		Combat:    25.0 * n,                               // (kills+HS/2+PK/2)×(1+acc×0.4), ~25/match pour un excellent joueur
		Survival:  analysis.DefensiveResistanceP80 * 1.25, // ~1.99 ; étire le haut au-dessus du P80
		Support:   300.0 * n,                              // assists × 50, ~6 assists/match
		Score:     ScorePerMinuteP80 * 1.25,               // score personnel/min, seuil constant (repère P80 ≈ 195/min)
		Objective: 350.0 * n,                              // PSA objectif, ~350/match
		Impact:    ocP80 * 1.25,                           // P80 title-aware (0.90 Infinite / 1.264 h5)
	}
}

// SynergyOffensiveConversion calcule le rendement offensif agrégé sur l'ensemble
// des matchs : 225 × (ΣK + ΣA/3) / ΣDD. Retourne 0 si aucun dégât infligé.
func SynergyOffensiveConversion(totalKills, totalAssists int, totalDamageDlt float64, effectiveHpToKill float64) float64 {
	if totalDamageDlt <= 0 {
		return 0
	}
	return effectiveHpToKill * (float64(totalKills) + float64(totalAssists)/3.0) / totalDamageDlt
}

// SynergyDefensiveResistance calcule la résistance défensive agrégée :
// ΣDT / (225 × ΣD). Zéro mort avec damage positif → score parfait (au-delà du P80).
func SynergyDefensiveResistance(totalDamageTkn float64, totalDeaths int, effectiveHpToKill float64) float64 {
	if totalDeaths == 0 {
		if totalDamageTkn > 0 {
			return analysis.DefensiveResistanceP80 * analysis.CombatYieldClipFactor
		}
		return 0
	}
	return totalDamageTkn / (effectiveHpToKill * float64(totalDeaths))
}

// SynergyScorePerMinute calcule le score personnel par minute jouée agrégé :
// ΣPersonalScore / (Σtime_played / 60). C'est un débit (métrique intensive,
// comme la résistance/le rendement) : différenciant entre joueurs et vivant sur
// Halo Infinite (contrairement à l'ancien résiduel medals/streaks ≈ 0). Retourne
// 0 si aucun temps joué n'est renseigné. Partagé par le radar escouade et le
// radar session-compare (règle ≤2 copies : une seule définition).
func SynergyScorePerMinute(totalPersonalScore, totalTimePlayedSeconds float64) float64 {
	if totalTimePlayedSeconds <= 0 {
		return 0
	}
	return totalPersonalScore / (totalTimePlayedSeconds / 60.0)
}

// synergyMainFallbackAxes calcule combat et support depuis SquadMatchRow
// (fallback quand squadLoader est absent). Impact, Survival et Score restent à 0
// car damage_dealt/damage_taken et personal_score ne sont pas dans SquadMatchRow.
func synergyMainFallbackAxes(
	allSquadRows []domain.SquadMatchRow,
	sharedMatches map[string]struct{},
) map[narrative.ParticipationAxis]float64 {
	seen := make(map[string]struct{})
	raw := map[narrative.ParticipationAxis]float64{}
	for _, m := range allSquadRows {
		if _, ok := sharedMatches[m.MatchID]; !ok {
			continue
		}
		if _, dup := seen[m.MatchID]; dup {
			continue
		}
		seen[m.MatchID] = struct{}{}
		acc := 0.0
		if m.Accuracy != nil {
			acc = *m.Accuracy / 100.0 // DB stocke 0..100, formule attend 0..1
		}
		combat := (float64(m.Kills) + 0.5*float64(m.HeadshotKills) + 0.5*float64(m.PerfectKills)) *
			(1.0 + acc*0.4)
		raw[narrative.AxisCombat] += combat
		raw[narrative.AxisSupport] += float64(m.Assists) * 50.0
	}
	return raw
}

// loadSynergyMateAxes charge les 6 axes radar depuis canonical.PlayerMatchRow pour
// un gamertag, restreint aux matchs partagés. Formules :
//   - Combat  : (kills + HS/2 + PK/2) × (1 + accuracy × 0.4), somme sur les matchs
//   - Survival : résistance défensive agrégée ΣDT / (225 × ΣD)
//   - Support  : assists × 50, somme sur les matchs
//   - Score    : score personnel par minute jouée ΣPS / (Σtime_played / 60)
//   - Objective: PSA catégorie "objective" via LoadObjectiveScores
//   - Impact   : rendement offensif agrégé 225×(ΣK+ΣA/3)/ΣDD
func (s *TeammatesService) loadSynergyMateAxes(
	ctx context.Context,
	gt string,
	sharedMatches map[string]struct{},
	sharedMatchIDs []string,
) map[narrative.ParticipationAxis]float64 {
	raw := map[narrative.ParticipationAxis]float64{}
	if s.squadLoader == nil {
		return raw
	}
	rows, err := s.squadLoader.LoadFor(ctx, s.titleSlug, gt, port.PlayerMatchFilters{})
	if err != nil {
		slog.WarnContext(ctx, "teammates_radar_load_failed", "gamertag", gt, "err", err)
		return raw
	}

	var totalKills, totalAssists, totalDeaths int
	var totalDamageDlt, totalDamageTkn, totalPS, totalTimeSec float64
	for _, r := range rows {
		if _, ok := sharedMatches[r.Summary.MatchID]; !ok {
			continue
		}
		k := intPtrOrZero(r.Self.Kills)
		hs := intPtrOrZero(r.Self.HeadshotKills)
		pk := intPtrOrZero(r.Self.PerfectKills)
		a := intPtrOrZero(r.Self.Assists)
		acc := 0.0
		if r.Self.Accuracy != nil {
			acc = *r.Self.Accuracy / 100.0 // DB stocke 0..100, formule attend 0..1
		}
		ps := intPtrOrZero(r.Self.PersonalScore)
		if ps == 0 {
			ps = intPtrOrZero(r.Self.Score)
		}
		raw[narrative.AxisCombat] += (float64(k) + 0.5*float64(hs) + 0.5*float64(pk)) * (1.0 + acc*0.4)
		raw[narrative.AxisSupport] += float64(a) * 50.0
		totalKills += k
		totalAssists += a
		totalDeaths += intPtrOrZero(r.Self.Deaths)
		totalDamageDlt += float64(intPtrOrZero(r.Self.DamageDealt))
		totalDamageTkn += float64(intPtrOrZero(r.Self.DamageTaken))
		totalPS += float64(ps)
		totalTimeSec += float64(intPtrOrZero(r.Self.TimePlayed))
	}

	hp := games.EffectiveHpToKill(s.titleSlug)
	raw[narrative.AxisImpact] = SynergyOffensiveConversion(totalKills, totalAssists, totalDamageDlt, hp)
	raw[narrative.AxisSurvival] = SynergyDefensiveResistance(totalDamageTkn, totalDeaths, hp)

	// Objective via PSA — dégradation silencieuse si absent.
	if objScores, err := s.squadLoader.LoadObjectiveScores(ctx, s.titleSlug, gt, sharedMatchIDs); err == nil {
		var objTotal float64
		for _, v := range objScores {
			objTotal += float64(v)
		}
		raw[narrative.AxisObjective] = objTotal
	}

	// Score : score personnel par minute jouée (vivant et différenciant, vs
	// l'ancien résiduel medals/streaks ≈ 0 mesuré sur Halo Infinite).
	raw[narrative.AxisScore] = SynergyScorePerMinute(totalPS, totalTimeSec)
	return raw
}

// buildSquadSynergyRadar calcule un profil de participation 6 axes par joueur
// sur l'INTERSECTION des matchs (matchs où TOUS les coéquipiers sélectionnés
// + le main player étaient présents). Formules alignées sur participation_radar.py.
func (s *TeammatesService) buildSquadSynergyRadar(
	ctx context.Context,
	allSquadRows []domain.SquadMatchRow,
	mainGamertag string,
	selectedGamertags []string,
) []domain.SquadSynergyRadarSeries {
	if len(allSquadRows) == 0 || len(selectedGamertags) == 0 {
		return nil
	}

	// allSquadRows est déjà l'intersection composition exacte — 1 row par match,
	// dédupliquée par intersectSquadRowsByMatchID (commit 851e10ef5). L'ancien
	// comptage d'occurrences (n >= len(selectedGamertags)) datait de l'ère
	// "union avec doublons par coéquipier" : sur l'input dédupliqué il rendait
	// le radar TOUJOURS vide dès 2 coéquipiers sélectionnés (1 >= 2 faux).
	sharedMatches := make(map[string]struct{}, len(allSquadRows))
	for _, m := range allSquadRows {
		sharedMatches[m.MatchID] = struct{}{}
	}
	if len(sharedMatches) == 0 {
		return nil
	}

	sharedMatchIDs := make([]string, 0, len(sharedMatches))
	for mid := range sharedMatches {
		sharedMatchIDs = append(sharedMatchIDs, mid)
	}

	thresholds := synergyRadarThresholds(len(sharedMatches), games.OffensiveConversionP80(s.titleSlug))

	mainRaw := s.loadSynergyMateAxes(ctx, mainGamertag, sharedMatches, sharedMatchIDs)
	if len(mainRaw) == 0 {
		mainRaw = synergyMainFallbackAxes(allSquadRows, sharedMatches)
	}

	// Titre sans damage_taken (Halo 5) → l'axe Survie (résistance défensive) est
	// figé à 0 et trompeur : on le retire de TOUS les joueurs (radar 5 axes
	// cohérent ; le front aligne indicateurs+valeurs sur series[0].axes). Même
	// règle que le radar match-view (dropUncomputableRadarAxes).
	skipSurvivalAxis := !games.ProvidesDamageTaken(s.titleSlug)
	out := make([]domain.SquadSynergyRadarSeries, 0, 1+len(selectedGamertags))
	toSeries := func(player string, raw map[narrative.ParticipationAxis]float64) domain.SquadSynergyRadarSeries {
		scores := narrative.ComputeParticipationProfile(raw, thresholds)
		axes := make([]domain.SquadSynergyRadarAxis, 0, len(scores))
		for _, sc := range scores {
			if skipSurvivalAxis && sc.Axis == narrative.AxisSurvival {
				continue
			}
			axes = append(axes, domain.SquadSynergyRadarAxis{
				Axis:  string(sc.Axis),
				Value: round2(sc.Value),
				Raw:   round2(sc.Raw),
			})
		}
		return domain.SquadSynergyRadarSeries{Player: player, Axes: axes}
	}

	out = append(out, toSeries(mainGamertag, mainRaw))
	for _, gt := range selectedGamertags {
		out = append(out, toSeries(gt, s.loadSynergyMateAxes(ctx, gt, sharedMatches, sharedMatchIDs)))
	}
	return out
}

// ---------------------------------------------------------------------------
// teammates.15 — Heatmap intensité kills par phase de match (10 buckets)
// ---------------------------------------------------------------------------

const intensityBuckets = 10
const intensityMinMatches = 3

// buildSquadIntensityProfile charge les kill events highlight pour les matchs
// du scope, calcule pour chaque option (all + main + teammates) un profil
// d'intensité 10-buckets × N matchs (normalisé par match).
//
// Renvoie nil si <3 matchs (section masquée), aucun kill event, ou aucune
// option ne produit de profil.
//
//nolint:funlen // chart-builder cohésif (load kills → 10-bucket phases → profil per option).
