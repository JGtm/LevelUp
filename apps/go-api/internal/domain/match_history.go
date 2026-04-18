// Package domain — types de l'historique des parties.
package domain

import "time"

// MatchHistoryRawRow est le type de transfert entre platform/duckdb et les services.
type MatchHistoryRawRow struct {
	MatchID            string
	StartTime          *time.Time
	MapName            *string
	MapNameFR          *string
	PairName           *string
	PairNameFR         *string
	PlaylistName       *string
	IsFirefight        bool
	IsRanked           bool
	SessionID          *string
	SessionLabel       *string
	IsWithFriends      bool
	Outcome            int
	TeamMMR            *float64
	EnemyMMR           *float64
	Kills              int
	Deaths             int
	Assists            int
	KDA                *float64
	Accuracy           *float64
	PersonalScore      *int
	AverageLifeSeconds *float64
	TimePlayedSeconds  *int
	IsExcluded         bool
}

// PaginationRequest représente les paramètres de pagination d'une requête.
type PaginationRequest struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// PaginationMeta représente les métadonnées de pagination dans une réponse.
type PaginationMeta struct {
	Total    int  `json:"total"`
	Page     int  `json:"page"`
	PageSize int  `json:"page_size"`
	HasNext  bool `json:"has_next"`
	HasPrev  bool `json:"has_prev"`
}

// MatchHistoryRow représente une ligne dans la table historique des parties.
type MatchHistoryRow struct {
	MatchID                  string    `json:"match_id"`
	StartTime                time.Time `json:"start_time"`
	StartTimeLabel           string    `json:"start_time_label"`
	OutcomeCode              int       `json:"outcome_code"`
	OutcomeLabel             string    `json:"outcome_label"`
	ScoreLabel               string    `json:"score_label"`
	MapUI                    *string   `json:"map_ui"`
	ModeUI                   *string   `json:"mode_ui"`
	PlaylistLabel            *string   `json:"playlist_label"`
	TeamMMR                  *float64  `json:"team_mmr"`
	EnemyMMR                 *float64  `json:"enemy_mmr"`
	DeltaMMR                 *float64  `json:"delta_mmr"`
	WinRateHist              *float64  `json:"win_rate_hist"`
	WinRateHistTotal         *int      `json:"win_rate_hist_total"`
	PerformanceScoreRelative *int      `json:"performance_score_relative"`
	AverageLifeMMSS          string    `json:"average_life_mmss"`
	MatchURL                 string    `json:"match_url"`
	IsExcluded               bool      `json:"is_excluded"`
}

// MatchHistoryQuerySummary est le résumé de la requête historique.
type MatchHistoryQuerySummary struct {
	TotalMatchesScoped     int     `json:"total_matches_scoped"`
	TotalMatchesUnfiltered int     `json:"total_matches_unfiltered"`
	PeriodLabel            *string `json:"period_label"`
	ActiveFilterMode       string  `json:"active_filter_mode"`
}

// MatchHistoryTable est la table paginée de l'historique.
type MatchHistoryTable struct {
	Items      []MatchHistoryRow `json:"items"`
	Pagination PaginationMeta    `json:"pagination"`
	Freshness  *string           `json:"freshness"`
}

// MatchHistoryQueryRequest est le corps de POST match-history/query.
type MatchHistoryQueryRequest struct {
	Filters           FilterContextInput `json:"filters"`
	Pagination        PaginationRequest  `json:"pagination"`
	SortField         string             `json:"sort_field"`
	SortDir           string             `json:"sort_dir"`
	IncludeExportHint bool               `json:"include_export_hint"`
	// Columns permet au client de préciser les colonnes souhaitées dans la réponse.
	// Nil/vide = toutes les colonnes disponibles (comportement par défaut).
	Columns []string `json:"columns,omitempty"`
}

// MatchHistoryPageResponse est la réponse de POST match-history/query.
type MatchHistoryPageResponse struct {
	Summary             MatchHistoryQuerySummary `json:"summary"`
	Table               MatchHistoryTable        `json:"table"`
	AvailableSortFields []string                 `json:"available_sort_fields"`
	AvailableColumns    []string                 `json:"available_columns"`
	ExportHint          *ExportHint              `json:"export_hint"`
}

// ExportHint indique qu'un export CSV est disponible.
type ExportHint struct {
	FileName      string  `json:"file_name"`
	EstimatedRows int     `json:"estimated_rows"`
	Token         *string `json:"token"`
}
