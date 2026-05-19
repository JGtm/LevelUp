package records

// extractors.go — métriques trackées par le détecteur de PB.
//
// Chaque métrique est identifiée par une clé string (cf. canonical fields)
// et lue depuis le `Metrics` map d'un MatchInput. L'orchestrateur (post-sync
// hook, commit 6) est responsable de remplir ce map depuis la DB.
//
// Pour ajouter une métrique : 1) la lister dans DefaultTrackedMetrics, 2) faire
// en sorte que l'orchestrateur la peuple dans MatchInput.Metrics. Pas de code
// à toucher dans detector.go.

// TrackedMetric identifie une métrique pour laquelle on suit un PB.
type TrackedMetric string

const (
	// MetricPerformanceScore : score de performance 0-100 (relatif à
	// l'historique personnel, déjà calculé dans player_match_enrichment).
	MetricPerformanceScore TrackedMetric = "performance_score"
	// MetricKDA : K/D/A par match.
	MetricKDA TrackedMetric = "kda"
	// MetricKPM : kills par minute (normalisé par durée de match).
	MetricKPM TrackedMetric = "kpm"
	// MetricAccuracy : précision (0-1).
	MetricAccuracy TrackedMetric = "accuracy"
	// MetricPSPM : personal score par minute.
	MetricPSPM TrackedMetric = "pspm"
)

// DefaultTrackedMetrics liste les métriques suivies par défaut. L'orchestrateur
// peut surcharger cette liste pour des évaluations partielles.
func DefaultTrackedMetrics() []TrackedMetric {
	return []TrackedMetric{
		MetricPerformanceScore,
		MetricKDA,
		MetricKPM,
		MetricAccuracy,
		MetricPSPM,
	}
}
