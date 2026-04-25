package prestige

// stretch.go — calcul du stretch ratio (écart cible/baseline).
//
// Formule selon Annexe B du plan conceptuel :
//   - Métriques bornées (ratio/score avec plafond) : stretch normalisé sur le headroom
//   - Métriques compteurs (kills/min, damage/min) : ratio brut

// MetricKind classe les métriques selon leur type pour adapter le calcul.
type MetricKind int

const (
	// MetricCount : compteur sans plafond (kills/min, damage/min...).
	MetricCount MetricKind = iota
	// MetricRatio : ratio borné [0, ceiling] (KDA, accuracy, win_rate, performance_score).
	MetricRatio
)

// ComputeStretchRatio calcule l'écart relatif entre une cible et une baseline.
//
// Pour MetricCount (compteurs purs) : stretch = target / baseline.
//
// Pour MetricRatio (bornés) : stretch normalisé sur le headroom restant
//
//	stretch = 1 + (target - baseline) / max(ceiling - baseline, epsilon)
//
// La forme MetricRatio donne un stretch interprétable comme "fraction du
// headroom restant que l'objectif vise à conquérir" (ramené à 1+x pour
// rester comparable aux thresholds 1.08/1.25/1.50/1.85).
//
// Précondition : baseline > 0. Si baseline ≤ 0, retourne 0 (rejet implicite).
func ComputeStretchRatio(target, baseline, ceiling float64, kind MetricKind) float64 {
	if baseline <= 0 {
		return 0
	}
	if target <= baseline {
		return target / baseline // < 1, rejeté plus en aval
	}
	switch kind {
	case MetricRatio:
		headroom := ceiling - baseline
		if headroom <= epsilonHeadroom {
			headroom = epsilonHeadroom
		}
		return 1.0 + (target-baseline)/headroom
	default:
		// MetricCount
		return target / baseline
	}
}

const epsilonHeadroom = 0.01
