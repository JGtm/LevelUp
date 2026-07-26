package narrative

// ParticipationAxis represente l'un des 6 axes du radar de participation.
// Stable : ces valeurs sont des cles d'i18n et de tokens couleur.
type ParticipationAxis string

// Constantes des 6 axes. Ordre d'affichage canonique : Combat -> Survival ->
// Support -> Score -> Objective -> Impact (sens horaire sur le radar).
const (
	AxisCombat    ParticipationAxis = "combat"
	AxisSurvival  ParticipationAxis = "survival"
	AxisSupport   ParticipationAxis = "support"
	AxisScore     ParticipationAxis = "score"
	AxisObjective ParticipationAxis = "objective"
	AxisImpact    ParticipationAxis = "impact"
)

// AllParticipationAxes retourne les 6 axes dans l'ordre canonique d'affichage.
func AllParticipationAxes() []ParticipationAxis {
	return []ParticipationAxis{
		AxisCombat, AxisSurvival, AxisSupport,
		AxisScore, AxisObjective, AxisImpact,
	}
}

// ParticipationScore est la valeur d'un axe : Value normalisee 0..100, Raw
// inchangee pour debug / tooltip.
type ParticipationScore struct {
	Axis  ParticipationAxis
	Value float64 // 0..100 (cap a 100 si Raw depasse le seuil)
	Raw   float64
}

// ParticipationThresholds porte les valeurs au-dessus desquelles un axe est
// considere a 100. Les valeurs sont calibrees par famille de mode parce qu'un
// "Combat" a 250 points fait sens en Slayer mais pas en CTF.
type ParticipationThresholds struct {
	Combat    float64
	Survival  float64
	Support   float64
	Score     float64
	Objective float64
	Impact    float64
}

// DefaultThresholds retourne les seuils par defaut pour une famille de mode.
// Les familles supportees sont : "slayer", "ctf", "strongholds", "oddball",
// "custom" (defaut). Calibration initiale heuristique.
//
// NOTE (2026-07-26, plan PLAN_AXE_OBJECTIFS_INDEX) : le champ Objective de ces
// seuils n'est PLUS consomme par les surfaces vivantes — elles alimentent l'axe
// via ComputeObjectiveIndex et ecrasent le seuil par ObjectiveIndexThreshold
// (cf. profile.aggregateNarrative, qui surcharge DefaultThresholds("custom")).
// Les autres champs restent les defauts des heuristiques V1.
func DefaultThresholds(modeFamily string) ParticipationThresholds {
	switch modeFamily {
	case "ctf":
		return ParticipationThresholds{
			Combat: 100, Survival: 50, Support: 80,
			Score: 200, Objective: 300, Impact: 80,
		}
	case "strongholds":
		return ParticipationThresholds{
			Combat: 120, Survival: 50, Support: 80,
			Score: 250, Objective: 250, Impact: 80,
		}
	case "oddball":
		return ParticipationThresholds{
			Combat: 100, Survival: 80, Support: 60,
			Score: 250, Objective: 200, Impact: 80,
		}
	case "slayer":
		return ParticipationThresholds{
			Combat: 250, Survival: 50, Support: 80,
			Score: 80, Objective: 0, Impact: 100,
		}
	}
	// custom / unknown : neutre.
	return ParticipationThresholds{
		Combat: 150, Survival: 50, Support: 80,
		Score: 150, Objective: 100, Impact: 80,
	}
}

// ComputeParticipationProfile normalise des points agreges par axe en scores
// 0..100 selon les seuils fournis. Le slice retourne contient toujours 6
// scores dans l'ordre AllParticipationAxes() (Combat -> Impact), avec Value=0
// pour les axes absents de rawByAxis.
//
// Si threshold == 0 sur un axe (cas Slayer.Objective), Value reste a 0
// (l'axe est neutre, n'apporte rien dans cette famille de mode).
//
// L'agregation award_name -> ParticipationAxis est laissee au consommateur :
// elle depend des definitions title-specific de personal_score_awards et
// devra etre alimentee par une table de mapping (Phase 1 pilote).
func ComputeParticipationProfile(
	rawByAxis map[ParticipationAxis]float64,
	thresholds ParticipationThresholds,
) []ParticipationScore {
	out := make([]ParticipationScore, 0, 6)
	for _, axis := range AllParticipationAxes() {
		raw := rawByAxis[axis]
		threshold := thresholdFor(axis, thresholds)
		var value float64
		if threshold > 0 {
			value = (raw / threshold) * 100
			if value > 100 {
				value = 100
			}
			if value < 0 {
				value = 0
			}
		}
		out = append(out, ParticipationScore{
			Axis:  axis,
			Value: value,
			Raw:   raw,
		})
	}
	return out
}

// thresholdFor route un axe vers le champ correspondant de la struct.
func thresholdFor(axis ParticipationAxis, t ParticipationThresholds) float64 {
	switch axis {
	case AxisCombat:
		return t.Combat
	case AxisSurvival:
		return t.Survival
	case AxisSupport:
		return t.Support
	case AxisScore:
		return t.Score
	case AxisObjective:
		return t.Objective
	case AxisImpact:
		return t.Impact
	}
	return 0
}
