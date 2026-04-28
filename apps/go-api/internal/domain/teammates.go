// Package domain — teammates.go : types pour la page Teammates (contrat FastAPI).
//
// Sprint 33 :
//
//	POST /api/v1/players/{slug}/pages/teammates → TeammatesPageResponse
package domain

import "time"

// ---------------------------------------------------------------------------
// Requête
// ---------------------------------------------------------------------------

// TeammatesQueryRequest est le corps de POST /pages/teammates.
type TeammatesQueryRequest struct {
	SelectedGamertags []string            `json:"selected_gamertags"`
	Filters           *FilterContextInput `json:"filters,omitempty"`
	// Multi-sessions : l'union des labels sélectionnés est appliquée côté service.
	PickedSoloSessions  []string `json:"picked_solo_session_labels,omitempty"`
	PickedSquadSessions []string `json:"picked_squad_session_labels,omitempty"`
}

// ---------------------------------------------------------------------------
// Réponse
// ---------------------------------------------------------------------------

// TeammateOption est un coéquipier fréquent sélectionnable.
type TeammateOption struct {
	Gamertag       string     `json:"gamertag"`
	XUID           *string    `json:"xuid,omitempty"`
	EncounterCount int        `json:"encounter_count"`
	LastSeenAt     *time.Time `json:"last_seen_at,omitempty"`
}

// TeammateKPIs sont les KPIs agrégés pour un groupe de matchs.
type TeammateKPIs struct {
	MatchCount     int      `json:"match_count"`
	Wins           int      `json:"wins"`
	KDRatio        *float64 `json:"kd_ratio"`
	WinRate        float64  `json:"win_rate"`
	Accuracy       *float64 `json:"accuracy"`
	KillsPerGame   *float64 `json:"kills_per_game"`
	AssistsPerGame *float64 `json:"assists_per_game"`
	// Sprint N : précision avancée
	HeadshotKillsPerGame *float64 `json:"headshot_kills_per_game,omitempty"`
	PerfectKillsPerGame  *float64 `json:"perfect_kills_per_game,omitempty"`
}

// MapBreakdownRow est la performance par carte pour la heatmap.
type MapBreakdownRow struct {
	MapUI      string  `json:"map_ui"`
	MatchCount int     `json:"match_count"`
	WinRate    float64 `json:"win_rate"`
}

// SquadMatchSeriesPoint est un point de la série par match (perf/timeline).
type SquadMatchSeriesPoint struct {
	MatchID          string   `json:"match_id"`
	StartTime        string   `json:"start_time"` // ISO 8601
	Outcome          int      `json:"outcome"`
	PerformanceScore *float64 `json:"performance_score,omitempty"`
	TeamMMRAvg       float64  `json:"team_mmr_avg"`
	SessionLabel     *string  `json:"session_label,omitempty"`
}

// TeammateRow est une ligne de résultat (stats avec vs sans un coéquipier).
type TeammateRow struct {
	Gamertag       string        `json:"gamertag"`
	XUID           *string       `json:"xuid,omitempty"`
	EncounterCount int           `json:"encounter_count"`
	LastSeenAt     *time.Time    `json:"last_seen_at,omitempty"`
	WithKPIs       TeammateKPIs  `json:"with_kpis"`
	WithoutKPIs    *TeammateKPIs `json:"without_kpis,omitempty"`
}

// SessionLabelEntry est une session avec sa plage temporelle (pour le mini-filtre client).
type SessionLabelEntry struct {
	Label     string    `json:"label"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
}

// SessionLabelsList contient les sessions disponibles pour les deux scopes (solo/escouade).
// Triées par StartedAt DESC côté service.
type SessionLabelsList struct {
	Solo  []SessionLabelEntry `json:"solo"`
	Squad []SessionLabelEntry `json:"squad"`
}

// TeammatesPageResponse est la réponse de POST /pages/teammates.
//
// Le champ historique `solo_reference` a été retiré : la page Solo a son
// propre endpoint dédié, et la page Escouade ne compare plus contre une
// baseline solo (cf. .ai/thought_log.md 2026-04-26 refonte UX Escouade).
type TeammatesPageResponse struct {
	Options       []TeammateOption  `json:"options"`
	Teammates     []TeammateRow     `json:"teammates"`
	TotalMatches  int               `json:"total_matches"`
	SessionLabels SessionLabelsList `json:"session_labels"`
	// Sprint N : données graphiques par coéquipier sélectionné
	Timeseries   []SquadTimeseriesPoint             `json:"timeseries,omitempty"`
	MapBreakdown []MapBreakdownRow                  `json:"map_breakdown,omitempty"`
	MatchSeries  map[string][]SquadMatchSeriesPoint `json:"match_series,omitempty"`
}
