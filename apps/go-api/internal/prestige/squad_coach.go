package prestige

// squad_coach.go — cœur du coach d'escouade (Phase C, point 1 & 3 du plan).
//
// Pure logique : agrège les profils 6-axes des membres (ADR 0004 : combat,
// survival, support, score, objective, impact), en déduit l'axe focal (le plus
// faible — celui à renforcer), et biaise un pool de templates vers cet axe.
//
// Les valeurs d'axes par membre sont fournies par l'orchestration (réutilise la
// même base 6-axes que le Synergy Radar côté analytics). Le wiring dans
// RefreshSquadPool + l'émission d'un signal coach escouade viennent ensuite.

// canonicalRadarAxes liste les 6 axes radar dans l'ordre canonique (ADR 0004).
var canonicalRadarAxes = []string{"combat", "survival", "support", "score", "objective", "impact"}

// AggregateSquadAxes calcule la moyenne par axe sur les profils des membres.
// Chaque membre peut n'exposer qu'un sous-ensemble d'axes ; la moyenne se fait
// sur les valeurs présentes.
func AggregateSquadAxes(memberAxes []map[string]float64) map[string]float64 {
	sums := make(map[string]float64)
	counts := make(map[string]int)
	for _, axes := range memberAxes {
		for axis, v := range axes {
			sums[axis] += v
			counts[axis]++
		}
	}
	out := make(map[string]float64, len(sums))
	for axis, s := range sums {
		out[axis] = s / float64(counts[axis])
	}
	return out
}

// SquadFocusAxis retourne l'axe canonique le plus faible du profil agrégé —
// l'orientation à renforcer pour l'escouade. Retourne "" si aucun axe canonique
// n'est présent. En cas d'égalité, l'ordre canonique tranche (premier rencontré).
func SquadFocusAxis(axes map[string]float64) string {
	focus := ""
	var min float64
	for _, axis := range canonicalRadarAxes {
		v, ok := axes[axis]
		if !ok {
			continue
		}
		if focus == "" || v < min {
			focus = axis
			min = v
		}
	}
	return focus
}

// MetricToRadarAxis mappe une métrique de défi vers un des 6 axes radar.
// Cohérent avec coach_advisor.axisForMetric (ADR 0004). Retourne "" si la
// métrique n'a pas d'axe direct.
func MetricToRadarAxis(metric string) string {
	switch metric {
	case "kills_total", "kills", "FieldKills", "headshot_kills", "FieldHeadshotKills",
		"power_weapon_kills", "FieldPowerWeaponKills", "melee_kills", "FieldMeleeKills",
		"grenade_kills", "FieldGrenadeKills", "kda", "FieldKDA", "kd", "FieldKDR",
		"max_killing_spree", "FieldMaxKillingSpree":
		return "combat"
	case "deaths_total", "deaths", "FieldDeaths", "survival_time_per_life", "avg_life_seconds":
		return "survival"
	case "assists", "assists_total", "FieldAssists":
		return "support"
	case "personal_score", "FieldPersonalScore", "score":
		return "score"
	case "objective_carry_time", "flag_captures_total", "objective_completion_rate":
		return "objective"
	case "damage_dealt", "FieldDamageDealt":
		return "impact"
	}
	return ""
}

// BiasTemplatesByAxis réordonne un pool de templates pour placer en tête ceux
// dont la métrique cible l'axe focal de l'escouade — l'alternative au shuffle
// pur dans RefreshSquadPool. Ordre stable (les non-correspondants gardent leur
// ordre relatif). Si focusAxis == "", retourne `templates` inchangé.
func BiasTemplatesByAxis(templates []Template, focusAxis string) []Template {
	if focusAxis == "" {
		return templates
	}
	matched := make([]Template, 0, len(templates))
	rest := make([]Template, 0, len(templates))
	for _, t := range templates {
		if MetricToRadarAxis(t.Metric) == focusAxis {
			matched = append(matched, t)
		} else {
			rest = append(rest, t)
		}
	}
	return append(matched, rest...)
}
