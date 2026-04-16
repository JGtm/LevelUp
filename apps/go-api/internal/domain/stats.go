// Package domain — types pour les séries temporelles, le performance score et le LUSR.
//
// Port Go de src/data/services/timeseries_service.py et src/analysis/performance_config.py.
package domain

import "time"

// ─── Entrées brutes ──────────────────────────────────────────────────────────

// StatsMatchRow est le type de transfert entre platform/duckdb et les services de stats.
// Contient toutes les métriques nécessaires au calcul du performance score (Q23).
type StatsMatchRow struct {
	MatchID           string
	StartTime         time.Time
	Outcome           *int
	Kills             int
	Deaths            int
	Assists           int
	KDA               *float64
	Accuracy          *float64
	PersonalScore     *int
	DamageDealt       *float64
	DamageTaken       *float64
	TimePlayedSeconds *int
	TeamMMR           *float64
	EnemyMMR          *float64
	KillsExpected     *float64
	DeathsExpected    *float64
	Rank              *int
	IsRanked          bool
	PlaylistName      string
	PairName          string
	TeamID            *int
	PerfScoreComputed *float64 // score stocké en DB (player_match_enrichment)
	SessionID         *string
	SessionLabel      *string
}

// LUSRMatchRating est le type de transfert pour un checkpoint LUSR.
type LUSRMatchRating struct {
	MatchID         string
	RatingValue     float64
	RatingDeviation float64
	PlaylistGroup   string
}

// ParticipantRow est le type de transfert pour un participant de match (Q25).
type ParticipantRow struct {
	MatchID        string
	XUID           string
	TeamID         *int
	KillsExpected  *float64
	DeathsExpected *float64
}

// ─── Métriques calculées ─────────────────────────────────────────────────────

// MatchMetrics regroupe les métriques normalisées par minute pour une seule partie.
// Utilisé en entrée de ComputeRelativePerformanceScore.
type MatchMetrics struct {
	MatchID          string
	StartTime        time.Time
	Outcome          *int
	KillsPerMin      float64
	DeathsPerMin     float64 // inverse : moins = mieux
	AssistsPerMin    float64
	KDA              float64
	Accuracy         *float64 // nil si absent
	ScorePerMin      *float64 // nil si absent
	DamagePerMin     *float64 // nil si absent
	RankPerfDiff     *float64 // (expected_rank - actual_rank), nil si absent
	KillsVsExpected  *float64 // actual / expected, nil si absent
	DeathsVsExpected *float64 // expected / actual (inversé), nil si absent
	// Champs pour LUSR/TrueSkill
	DamageDealtRaw *float64
	DamageTakenRaw *float64
	KillsExpected  *float64
	DeathsExpected *float64
	TeamMMR        *float64
	EnemyMMR       *float64
	Rank           *int
	IsRanked       bool
	PlaylistName   string
	PairName       string
	TeamID         *int
}

// ─── Résultats des onglets ───────────────────────────────────────────────────

// PerformancePoint est un point de la série performance score.
type PerformancePoint struct {
	MatchID   string    `json:"match_id"`
	StartTime time.Time `json:"start_time"`
	Score     *float64  `json:"score"`
}

// CumulativePoint est un point d'une série cumulative (K/D, net score).
type CumulativePoint struct {
	Index     int       `json:"index"`
	StartTime time.Time `json:"start_time"`
	Value     float64   `json:"value"`
}

// WinLossPoint est un point de la série victoires/défaites.
type WinLossPoint struct {
	StartTime time.Time `json:"start_time"`
	Outcome   int       `json:"outcome"`
	SessionID *string   `json:"session_id,omitempty"`
}

// AccuracyPoint est un point de la série précision.
type AccuracyPoint struct {
	StartTime time.Time `json:"start_time"`
	Accuracy  float64   `json:"accuracy"`
}

// ObjectivePoint est un point de la série objectif (personal score, assists).
type ObjectivePoint struct {
	StartTime     time.Time `json:"start_time"`
	PersonalScore *int      `json:"personal_score,omitempty"`
	Assists       int       `json:"assists"`
}

// LUSRPoint est un point de la série LUSR.
type LUSRPoint struct {
	MatchID         string  `json:"match_id"`
	RatingValue     float64 `json:"rating_value"`
	RatingDeviation float64 `json:"rating_deviation"`
	PlaylistGroup   string  `json:"playlist_group"`
}

// ─── Réponses des onglets ────────────────────────────────────────────────────

// StatsQueryRequest est le corps de POST stats/query.
type StatsQueryRequest struct {
	// Mode est soit "period" (toutes les parties de la période filtrée)
	// soit "sessions" (groupé par sessions).
	Mode string `json:"mode"`
	// Tab sélectionne l'onglet : "win_loss", "accuracy", "objective", "form", "lusr".
	Tab string `json:"tab"`
}

// WinLossTabResponse est la réponse de l'onglet Win/Loss.
type WinLossTabResponse struct {
	Points         []WinLossPoint    `json:"points"`
	WinRate        float64           `json:"win_rate"`
	TotalMatches   int               `json:"total_matches"`
	RollingWinRate []float64         `json:"rolling_win_rate"`
	CumulativeKD   []CumulativePoint `json:"cumulative_kd"`
	CumulativeNet  []CumulativePoint `json:"cumulative_net"`
}

// AccuracyTabResponse est la réponse de l'onglet Précision.
type AccuracyTabResponse struct {
	Points      []AccuracyPoint `json:"points"`
	Mean        float64         `json:"mean"`
	HasData     bool            `json:"has_data"`
	ScorePerMin []float64       `json:"score_per_min"`
}

// ObjectiveTabResponse est la réponse de l'onglet Objectif.
type ObjectiveTabResponse struct {
	Points     []ObjectivePoint `json:"points"`
	TotalScore int              `json:"total_score"`
	AvgAssists float64          `json:"avg_assists"`
	HasData    bool             `json:"has_data"`
}

// FormTabResponse est la réponse de l'onglet Forme (perf score).
type FormTabResponse struct {
	Points        []PerformancePoint `json:"points"`
	Mean          *float64           `json:"mean"`
	HasEnoughData bool               `json:"has_enough_data"`
}

// LUSRTabResponse est la réponse de l'onglet LUSR.
type LUSRTabResponse struct {
	Points        []LUSRPoint `json:"points"`
	CurrentRating *float64    `json:"current_rating"`
	HasData       bool        `json:"has_data"`
}

// StatsPageResponse est la réponse complète de la page stats/séries.
type StatsPageResponse struct {
	WinLoss      *WinLossTabResponse   `json:"win_loss,omitempty"`
	Accuracy     *AccuracyTabResponse  `json:"accuracy,omitempty"`
	Objective    *ObjectiveTabResponse `json:"objective,omitempty"`
	Form         *FormTabResponse      `json:"form,omitempty"`
	LUSR         *LUSRTabResponse      `json:"lusr,omitempty"`
	BucketInfo   BucketInfo            `json:"bucket_info"`
	TotalMatches int                   `json:"total_matches"`
}
