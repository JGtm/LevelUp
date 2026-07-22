package records

import "math"

// bounds.go — bornes de vraisemblance et catalogue des métriques suivies (A4).
//
// Contexte (DEC-7) : incident prod 2026-07 — precision affichée « 7333 % »
// (accuracy lue en 0..100 au lieu de 0..1, ADR 0006), best_kda à 107. Une
// valeur hors plage est le symptôme d'une corruption d'échelle/ingestion : on
// refuse de la persister en PB (côté detector), et on ne sert pas les métriques
// hors catalogue à la lecture — sans masquer le bug amont (WarnContext).

// metricBound définit l'intervalle [Min, Max] plausible d'une métrique (bornes
// incluses).
type metricBound struct {
	Min float64
	Max float64
}

// Bornes nommées par métrique (pas de magic number). accuracy est stockée en
// 0..1 (ADR 0006). Les plafonds sont larges : filet anti-corruption, pas un
// seuil de performance — ils ne doivent rejeter que l'aberrant.
const (
	boundPerfScoreMin = 0.0
	boundPerfScoreMax = 100.0 // performance_score : score relatif 0-100
	boundKDAMin       = -20.0 // KDA agrégé peut être négatif (morts >> frags)
	boundKDAMax       = 50.0
	boundKPMMin       = 0.0
	boundKPMMax       = 20.0 // 20 tueries/minute est déjà extrême
	boundAccuracyMin  = 0.0
	boundAccuracyMax  = 1.0 // stockage 0..1 (ADR 0006)
	boundPSPMMin      = 0.0
	boundPSPMMax      = 10000.0 // score perso/minute : plafond de sécurité généreux
)

// metricBounds mappe chaque TrackedMetric vers sa borne de vraisemblance.
func metricBounds() map[TrackedMetric]metricBound {
	return map[TrackedMetric]metricBound{
		MetricPerformanceScore: {boundPerfScoreMin, boundPerfScoreMax},
		MetricKDA:              {boundKDAMin, boundKDAMax},
		MetricKPM:              {boundKPMMin, boundKPMMax},
		MetricAccuracy:         {boundAccuracyMin, boundAccuracyMax},
		MetricPSPM:             {boundPSPMMin, boundPSPMMax},
	}
}

// IsPlausibleValue retourne true si value est dans la borne de la métrique
// (bornes incluses). NaN/Inf sont toujours rejetés. Une métrique sans borne
// connue est considérée plausible (le filet ne s'applique qu'aux métriques
// cataloguées ; l'inconnu est traité par IsKnownMetric côté lecture).
func IsPlausibleValue(metric TrackedMetric, value float64) bool {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return false
	}
	b, ok := metricBounds()[metric]
	if !ok {
		return true
	}
	return value >= b.Min && value <= b.Max
}

// MetricBound expose l'intervalle de vraisemblance d'une métrique (lecture
// seule). Sert aux consommateurs qui doivent matérialiser les bornes (ex : la
// purge SQL A5) sans redéfinir les seuils.
type MetricBound struct {
	Metric TrackedMetric
	Min    float64
	Max    float64
}

// TrackedMetricBounds retourne les bornes de toutes les métriques suivies, dans
// l'ordre déterministe de DefaultTrackedMetrics. Source unique des seuils A4/A5.
func TrackedMetricBounds() []MetricBound {
	m := metricBounds()
	out := make([]MetricBound, 0, len(m))
	for _, tm := range DefaultTrackedMetrics() {
		if b, ok := m[tm]; ok {
			out = append(out, MetricBound{Metric: tm, Min: b.Min, Max: b.Max})
		}
	}
	return out
}

// IsKnownMetric retourne true si la clé de métrique fait partie du catalogue des
// métriques suivies (DefaultTrackedMetrics). Sert au filtrage read-side : une
// métrique hors catalogue (ex « best_kda » legacy) n'est ni labellisée ni
// servie à l'UI. Aucun catalogue de ce type n'existait avant A4.
func IsKnownMetric(metric string) bool {
	for _, m := range DefaultTrackedMetrics() {
		if string(m) == metric {
			return true
		}
	}
	return false
}
