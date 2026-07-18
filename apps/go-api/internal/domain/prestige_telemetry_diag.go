// Package domain — prestige_telemetry_diag.go : type métier pour l'endpoint diag
// d'agrégation de la télémétrie Prestige (/api/v1/_diag/prestige/telemetry/{slug}).
//
// ADR 0020 (coach→pont Prestige). Agrège la table append-only prestige_telemetry
// (stats.duckdb) par origine du défi (source) pour mesurer si les défis proposés
// par le coach sont acceptés/complétés davantage que ceux d'autres origines.
package domain

// PrestigeTelemetryDiag est l'agrégat de la télémétrie Prestige d'un joueur,
// ventilé par origine du défi. Émis par l'endpoint diag.
type PrestigeTelemetryDiag struct {
	PlayerSlug  string                         `json:"player_slug"`
	TotalEvents int                            `json:"total_events"`
	BySource    []PrestigeTelemetrySourceStats `json:"by_source"`
}

// PrestigeTelemetrySourceStats agrège les compteurs et taux d'une origine
// (ChallengeSource*: "coach" | "user" | "pilot_mode", ou "unknown" pour les
// lignes historiques sans source).
type PrestigeTelemetrySourceStats struct {
	Source    string `json:"source"`
	Created   int    `json:"created"`
	Rejected  int    `json:"rejected"` // auto-reject "too_easy" à la création
	Completed int    `json:"completed"`
	Expired   int    `json:"expired"`
	Abandoned int    `json:"abandoned"`

	// AcceptanceRate = created / (created + rejected). Part des défis matérialisés
	// vs auto-rejetés à la création (stretch trop faible). -1 si dénominateur nul.
	AcceptanceRate float64 `json:"acceptance_rate"`
	// CompletionRate = completed / created. -1 si aucun défi créé pour l'origine.
	CompletionRate float64 `json:"completion_rate"`
	// AbandonRate = abandoned / created. -1 si aucun défi créé pour l'origine.
	AbandonRate float64 `json:"abandon_rate"`
}
