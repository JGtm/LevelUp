package coach_advisor

import (
	"math"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/progression/coach"
)

// signals.go — traduction des alertes coach (sortie du Generator existant) en
// Signals consommables par le matcher / synthesizer / arc_composer.
//
// Fonction PURE — pas d'I/O, déterministe pour des entrées identiques.
//
// Le mapping LUSR component → radar axis et metric → radar axis est inline
// pour Halo Infinite. Pour un nouveau titre, étendre les deux maps ou
// déplacer dans config/coach_advisor/axis_mapping.toml (V2.1 si besoin).

// SignalsFromAlerts convertit []coach.Alert en []Signal. Les alertes non
// pertinentes (record_broken déjà achieved, milestone_unlocked déjà débloqué,
// streak_milestone purement informatif, patterns engine stubs) sont ignorées.
//
// Retourne uniquement les signaux exploitables comme déclencheur de proposal :
//   - LOWESSPositive            ← AlertTypeLOWESSPositive
//   - CombatPatternActive       ← AlertTypeCombatPatternActif
//   - CombatPatternDiscreet     ← AlertTypeCombatPatternDiscret
//   - CombatPatternFragile      ← AlertTypeCombatPatternFragile
//   - RecordApproach            ← AlertTypeRecordNearMiss
//   - MilestoneApproach         ← AlertTypeMilestoneNearMiss
//   - ComebackWelcome           ← AlertTypeComebackWelcome
//   - LUSRTierApproach          ← AlertTypeLUSRTierApproach
func SignalsFromAlerts(alerts []coach.Alert) []Signal {
	out := make([]Signal, 0, len(alerts))
	for _, a := range alerts {
		if s, ok := signalFromAlert(a); ok {
			out = append(out, s)
		}
	}
	return out
}

func signalFromAlert(a coach.Alert) (Signal, bool) {
	switch a.Type {
	case coach.AlertTypeLOWESSPositive:
		return signalFromLOWESS(a), true
	case coach.AlertTypeLOWESSSoftNegative:
		return signalFromLOWESSSoftNegative(a), true
	case coach.AlertTypeCombatPatternActif:
		return signalFromCombatActif(a), true
	case coach.AlertTypeCombatPatternDiscret:
		return signalFromCombatDiscret(a), true
	case coach.AlertTypeCombatPatternFragile:
		return signalFromCombatFragile(a), true
	case coach.AlertTypeRecordNearMiss:
		return signalFromRecordNearMiss(a), true
	case coach.AlertTypeMilestoneNearMiss:
		return signalFromMilestoneNearMiss(a), true
	case coach.AlertTypeComebackWelcome:
		return signalFromComeback(a), true
	case coach.AlertTypeLUSRTierApproach:
		return signalFromLUSRTier(a), true
	default:
		// Ignoré : record_broken / milestone_unlocked / streak_milestone /
		// pattern_* / campaign_* — pas de proposal à émettre.
		return Signal{}, false
	}
}

// signalFromLOWESS : tendance positive sur une composante LUSR.
// Strength = clamp(slope/0.5, 0, 1) — saturé à slope=0.5/jour.
func signalFromLOWESS(a coach.Alert) Signal {
	component := stringParam(a.Params, "component")
	slope := floatParam(a.Params, "slope")
	return Signal{
		Kind:          SignalLOWESSPositive,
		LUSRComponent: component,
		RadarAxis:     axisForLUSRComponent(component),
		Strength:      clamp01(slope / 0.5),
		Source:        a,
	}
}

// signalFromLOWESSSoftNegative : tendance NÉGATIVE soutenue sur une composante
// LUSR → opportunité de stabilisation (registre soft). Strength basée sur la
// MAGNITUDE de la pente (|slope|/0.5), symétrique du positif ; le gate de
// strength en aval (MinStrength) ne déclenche une proposal que pour une baisse
// significative.
func signalFromLOWESSSoftNegative(a coach.Alert) Signal {
	component := stringParam(a.Params, "component")
	slope := floatParam(a.Params, "slope")
	return Signal{
		Kind:          SignalLOWESSSoftNegative,
		LUSRComponent: component,
		RadarAxis:     axisForLUSRComponent(component),
		Strength:      clamp01(math.Abs(slope) / 0.5),
		Source:        a,
	}
}

// signalFromCombatActif : OC basse + activité haute → joueur prêt pour
// challenge offensif. Strength basée sur l'écart de résidu (saturé à +15).
func signalFromCombatActif(a coach.Alert) Signal {
	residual := floatParam(a.Params, "avg_residual")
	return Signal{
		Kind:          SignalCombatPatternActive,
		LUSRComponent: "kills_vs_expected",
		RadarAxis:     "combat",
		Strength:      clamp01(residual / 15.0),
		Source:        a,
	}
}

// signalFromCombatDiscret : activité basse — support à valoriser.
// Strength sur |résidu négatif| saturé à 15.
func signalFromCombatDiscret(a coach.Alert) Signal {
	residual := floatParam(a.Params, "avg_residual")
	return Signal{
		Kind:          SignalCombatPatternDiscreet,
		LUSRComponent: "assists_per_kill",
		RadarAxis:     "support",
		Strength:      clamp01(math.Abs(residual) / 15.0),
		Source:        a,
	}
}

// signalFromCombatFragile : DR basse — survie à solidifier.
// Strength = 1 - clamp(MedianDR / seuil de référence DR, 0, 1). Le seuil est le
// MÊME que celui qui a déclenché l'alerte côté coach (source unique analysis).
func signalFromCombatFragile(a coach.Alert) Signal {
	medianDR := floatParam(a.Params, "median_dr")
	gapRatio := 1.0 - clamp01(medianDR/analysis.CombatDRReferenceThreshold)
	return Signal{
		Kind:          SignalCombatPatternFragile,
		LUSRComponent: "deaths_vs_expected",
		RadarAxis:     "survival",
		Strength:      clamp01(gapRatio),
		Source:        a,
	}
}

// signalFromRecordNearMiss : valeur courante à <5% du PB.
// Strength = 1 - gap_ratio (plus on est proche, plus c'est fort).
func signalFromRecordNearMiss(a coach.Alert) Signal {
	metric := stringParam(a.Params, "metric")
	target := floatParam(a.Params, "target")
	value := floatParam(a.Params, "value")
	gapRatio := 1.0
	if target > 0 {
		gapRatio = (target - value) / target
	}
	return Signal{
		Kind:      SignalRecordApproach,
		Metric:    metric,
		RadarAxis: axisForMetric(metric),
		Strength:  clamp01(1.0 - gapRatio),
		Source:    a,
	}
}

// signalFromMilestoneNearMiss : progress >= 90% du seuil.
// Strength = progress / threshold (saturé à 1).
func signalFromMilestoneNearMiss(a coach.Alert) Signal {
	metric := stringParam(a.Params, "metric")
	threshold := floatParam(a.Params, "threshold")
	progress := floatParam(a.Params, "progress")
	ratio := 0.0
	if threshold > 0 {
		ratio = progress / threshold
	}
	return Signal{
		Kind:      SignalMilestoneApproach,
		Metric:    metric,
		RadarAxis: axisForMetric(metric),
		Strength:  clamp01(ratio),
		Source:    a,
	}
}

// signalFromComeback : strength fixe à 0.6 (juste au-dessus du seuil
// synthesis_min_strength) — on veut une proposal soft à la reprise.
func signalFromComeback(a coach.Alert) Signal {
	return Signal{
		Kind:     SignalComebackWelcome,
		Strength: 0.6,
		Source:   a,
	}
}

// signalFromLUSRTier : strength = 1 - gap/delta_max (plus on est proche,
// plus c'est fort). delta_max = 10 pts (cf. coach.LUSRTierApproachDelta).
func signalFromLUSRTier(a coach.Alert) Signal {
	gap := floatParam(a.Params, "gap")
	const deltaMax = 10.0
	return Signal{
		Kind:     SignalLUSRTierApproach,
		Strength: clamp01(1.0 - gap/deltaMax),
		Source:   a,
	}
}

// ─── Mapping helpers ───

// axisForLUSRComponent retourne le radar axis associé à une composante LUSR.
// Pour Halo Infinite — basé sur ADR 0004 (6 axes : combat, survival,
// support, score, objective, impact).
//
// Retourne "" si la composante n'a pas de mapping connu (le matcher tombera
// alors sur la signature metric uniquement).
func axisForLUSRComponent(c string) string {
	switch c {
	case "accuracy_delta", "kills_vs_expected", "headshots_per_kill", "shotgun_OC", "weapon_OC":
		return "combat"
	case "deaths_vs_expected", "survival_time_per_life":
		return "survival"
	case "assists_per_kill", "callouts_per_min", "team_damage_share":
		return "support"
	case "win_rate_delta", "score_delta":
		return "score"
	case "objective_carry_time", "objective_completion_rate", "flag_captures_per_match":
		return "objective"
	case "mvp_count", "impact_kills_per_match":
		return "impact"
	}
	return ""
}

// axisForMetric retourne le radar axis associé à une métrique brute.
// Mapping pragmatique aligné sur ADR 0004 + canonical/fields.go.
func axisForMetric(m string) string {
	switch m {
	case "accuracy", "kda", "kills_total", "headshots_total", "damage_dealt":
		return "combat"
	case "deaths_total", "survival_time_per_life":
		return "survival"
	case "assists_total", "assists_per_kill", "callouts":
		return "support"
	case "win_rate", "performance_score", "score":
		return "score"
	case "objective_carry_time", "flag_captures_total":
		return "objective"
	case "mvp_count":
		return "impact"
	}
	return ""
}

// ─── Param extraction helpers ───

func stringParam(p map[string]any, key string) string {
	if v, ok := p[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// floatParam accepte float64, float32, int, int64 — robuste aux types
// génériques de map[string]any sérialisés JSON.
func floatParam(p map[string]any, key string) float64 {
	v, ok := p[key]
	if !ok {
		return 0
	}
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	}
	return 0
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}
