// Package domain — squad.go : types pour les pages Escouade et Synthèse.
//
// Sprint 12 :
//
//	GET /api/v1/players/{slug}/pages/squad          → SquadPageResponse
//	GET /api/v1/players/{slug}/pages/squad?teammate= → SquadPageResponse (filtered)
//	GET /api/v1/players/{slug}/pages/synthesis       → SynthesisPageResponse
package domain

import "time"

// ---------------------------------------------------------------------------
// Lignes brutes DuckDB — Squad
// ---------------------------------------------------------------------------

// TopTeammateRow est une ligne brute chargée depuis Q29 (coéquipiers fréquents).
type TopTeammateRow struct {
	XUID          string
	Gamertag      string
	GamesTogether int
	WinsTogether  int
	WinRate       float64
	AvgKills      float64
	AvgDeaths     float64
	AvgKDA        float64
}

// SquadMatchRow est une ligne brute chargée depuis Q30 (matchs communs avec coéquipier).
type SquadMatchRow struct {
	MatchID          string
	StartTime        time.Time
	MapName          string
	MapUI            string
	PairName         string
	PlaylistName     string
	IsFirefight      bool
	IsRanked         bool
	Outcome          int
	Kills            int
	Deaths           int
	Assists          int
	KDA              *float64
	Accuracy         *float64
	TimePlayedSecs   int
	TeamMMR          float64
	SessionID        *int
	SessionLabel     *string
	PerformanceScore *float64
	IsWithFriends    bool
	// Sprint N : métriques précision avancée
	HeadshotKills int
	PerfectKills  int
}

// TeammateMatchRow est une ligne brute chargée depuis Q31 (stats d'un coéquipier).
type TeammateMatchRow struct {
	MatchID        string
	StartTime      time.Time
	MapUI          string
	PairName       string
	Outcome        int
	Kills          int
	Deaths         int
	Assists        int
	Ratio          *float64
	TimePlayedSecs int
	TeamMMR        float64
	Accuracy       *float64
	MyTeamScore    *int
	EnemyTeamScore *int
}

// ImpactEventRow est une ligne brute chargée depuis Q32 (events highlight escouade).
type ImpactEventRow struct {
	MatchID   string
	XUID      string
	Gamertag  string
	EventType string
	TimeMS    int64
}

// SynthesisHeatmapRow est une ligne brute chargée depuis Q33 (heatmap map×mode).
type SynthesisHeatmapRow struct {
	MapName    string
	ModeName   string
	MatchCount int
	Wins       int
}

// ---------------------------------------------------------------------------
// Types de réponse — Escouade
// ---------------------------------------------------------------------------

// TopTeammate est un coéquipier agrégé pour la liste des plus fréquents.
type TopTeammate struct {
	XUID          string  `json:"xuid"`
	Gamertag      string  `json:"gamertag"`
	GamesTogether int     `json:"games_together"`
	WinsTogether  int     `json:"wins_together"`
	WinRate       float64 `json:"win_rate"`
	AvgKDA        float64 `json:"avg_kda"`
	AvgKills      float64 `json:"avg_kills"`
}

// SquadPerformanceScore est le score collectif d'une escouade (0-100).
// Miroir de Python compute_squad_performance_score().
type SquadPerformanceScore struct {
	Score      *float64               `json:"score"`
	Grade      string                 `json:"grade"`
	Components map[string]interface{} `json:"components"`
}

// ParticipationProfile est le profil radar à 6 axes d'un joueur.
// Miroir de Python ParticipationProfile dataclass.
type ParticipationProfile struct {
	Name   string             `json:"name"`
	Color  string             `json:"color"`
	Values map[string]float64 `json:"values"`
}

// ImpactEventSummary résume les événements d'impact pour 2 joueurs en escouade.
type ImpactEventSummary struct {
	Me       int `json:"me"`
	Teammate int `json:"teammate"`
}

// SquadImpact contient les 4 types d'événements d'impact analysés.
type SquadImpact struct {
	FirstBloods ImpactEventSummary `json:"first_bloods"`
	ClutchKills ImpactEventSummary `json:"clutch_kills"`
	LastKills   ImpactEventSummary `json:"last_kills"`
	FirstDeaths ImpactEventSummary `json:"first_deaths"`
	Available   bool               `json:"available"`
}

// SquadRecord est le record d'un joueur pour une métrique donnée.
type SquadRecord struct {
	Me       *float64 `json:"me"`
	Teammate *float64 `json:"teammate"`
}

// SquadTimeseriesPoint est un point de la série temporelle escouade.
type SquadTimeseriesPoint struct {
	BucketLabel string   `json:"bucket_label"`
	SquadPerf   *float64 `json:"squad_perf"`
	TeamMMRAvg  *float64 `json:"team_mmr_avg,omitempty"`
	MatchCount  int      `json:"match_count"`
	WinRate     *float64 `json:"win_rate,omitempty"`
}

// SquadBreakdownStats résume les stats solo ou escouade.
type SquadBreakdownStats struct {
	MatchCount int     `json:"match_count"`
	WinRate    float64 `json:"win_rate"`
	AvgKDA     float64 `json:"avg_kda"`
	AvgKills   float64 `json:"avg_kills"`
}

// SelectedTeammateData contient les données calculées pour le coéquipier sélectionné.
type SelectedTeammateData struct {
	Gamertag      string                 `json:"gamertag"`
	XUID          string                 `json:"xuid"`
	GamesTogether int                    `json:"games_together"`
	SquadScore    *SquadPerformanceScore `json:"squad_score,omitempty"`
	RadarMe       ParticipationProfile   `json:"radar_me"`
	RadarTeammate ParticipationProfile   `json:"radar_teammate"`
	Impact        SquadImpact            `json:"impact"`
	Records       map[string]SquadRecord `json:"records"`
	Timeseries    []SquadTimeseriesPoint `json:"timeseries"`
}

// SquadPageResponse est la réponse complète de la page Escouade.
type SquadPageResponse struct {
	TopTeammates     []TopTeammate         `json:"top_teammates"`
	SelectedTeammate *SelectedTeammateData `json:"selected_teammate,omitempty"`
	SoloStats        SquadBreakdownStats   `json:"solo_stats"`
	SquadStats       SquadBreakdownStats   `json:"squad_stats"`
}

// ---------------------------------------------------------------------------
// Types de réponse — Synthèse
// ---------------------------------------------------------------------------

// HeatmapCell est une cellule de la heatmap carte × mode de jeu.
type HeatmapCell struct {
	RowKey string  `json:"row_key"`
	ColKey string  `json:"col_key"`
	Value  float64 `json:"value"`
	Count  int     `json:"count"`
}

// TopWeekEntry est une semaine performante dans l'historique du joueur.
type TopWeekEntry struct {
	WeekLabel  string  `json:"week_label"`
	WinRate    float64 `json:"win_rate"`
	AvgKills   float64 `json:"avg_kills"`
	AvgDeaths  float64 `json:"avg_deaths"`
	AvgKDA     float64 `json:"avg_kda"`
	MatchCount int     `json:"match_count"`
}

// SynthesisKPIs contient les métriques agrégées solo ou escouade (Sprint 43).
type SynthesisKPIs struct {
	MatchCount       int      `json:"match_count"`
	Wins             int      `json:"wins"`
	KDRatio          *float64 `json:"kd_ratio"`
	WinRate          float64  `json:"win_rate"`
	Accuracy         *float64 `json:"accuracy"`
	KillsPerMin      *float64 `json:"kills_per_min"`
	AvgLifeSeconds   *float64 `json:"avg_life_seconds"`
	PerformanceScore *float64 `json:"performance_score"`
}

// ComparisonMetricItem est une métrique bipolaire solo / escouade.
type ComparisonMetricItem struct {
	Label      string  `json:"label"`
	SoloValue  float64 `json:"solo_value"`
	SquadValue float64 `json:"squad_value"`
	SoloText   string  `json:"solo_text"`
	SquadText  string  `json:"squad_text"`
}

// TemporalHeatmapCell est une cellule de la heatmap jour × heure (Sprint 43).
type TemporalHeatmapCell struct {
	DOW   int `json:"dow"`  // 0 = lundi … 6 = dimanche
	Hour  int `json:"hour"` // 0–23
	Count int `json:"count"`
}

// SynthesisPageResponse est la réponse de la page Synthèse.
// Sprint 43 : enrichi avec KPIs détaillés, comparison_metrics et heatmap temporelle.
type SynthesisPageResponse struct {
	Period            string                 `json:"period"`
	TotalMatches      int                    `json:"total_matches"`
	SoloKPIs          SynthesisKPIs          `json:"solo_kpis"`
	SquadKPIs         SynthesisKPIs          `json:"squad_kpis"`
	ComparisonMetrics []ComparisonMetricItem `json:"comparison_metrics"`
	HeatmapData       []TemporalHeatmapCell  `json:"heatmap_data"`
	TopWeeks          []TopWeekEntry         `json:"top_weeks"`
}

// ---------------------------------------------------------------------------
// Types de requête — Synthèse (POST body)
// ---------------------------------------------------------------------------

// SynthesisPageRequest : corps de POST /pages/synthesis.
// Contient les filtres optionnels de la requête.
type SynthesisPageRequest struct {
	Filters FilterContextInput `json:"filters"`
}

// ---------------------------------------------------------------------------
// Lignes brutes DuckDB — Synthèse (simplified)
// ---------------------------------------------------------------------------

// SynthesisMatchRow est une ligne brute chargée depuis Q33b.
// Sprint 43 : enrichi avec accuracy, time_played, performance_score pour les KPIs bipolaires.
// Sprint N : ajout SessionLabel pour les filtres de session.
// Sprint N+1 : ajout IsRanked, IsFirefight, PlaylistName pour le câblage cascade.
type SynthesisMatchRow struct {
	MatchID          string
	StartTime        time.Time
	Outcome          int
	Kills            int
	Deaths           int
	KDA              *float64
	IsWithFriends    bool
	Accuracy         *float64
	TimePlayedSecs   *int
	PerformanceScore *float64
	SessionLabel     *string
	IsRanked         bool
	IsFirefight      bool
	PlaylistName     string
}
