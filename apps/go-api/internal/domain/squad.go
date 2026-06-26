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
	MatchID   string
	StartTime time.Time
	MapName   string
	MapUI     string
	// PairName : libellé EN brut du mode depuis match_registry (peut être un
	// UUID si la metadata n'a pas de traduction — résolu via la cascade
	// canonique analysis.ResolvePairNameFR, cf. buildSquadMatchHistory).
	PairName string
	// PairNameFR : pair_name_fr depuis match_registry (NULL/'' fréquent → la
	// cascade FR le re-résout via asset_translations[pair_id] + mode_name_tr).
	PairNameFR string
	// PairID : UUID de la paire mode/map (clé de résolution asset_translations).
	PairID string
	// GameVariantID : UUID de la variante de jeu (game_variant_id de match_registry).
	// Source du mode pour les titres SANS pair_name (ex. Halo 5 : pair_name vide
	// mais game_variant_id peuplé). Résolu read-time via asset_translations
	// (asset_type='game_variant') → GameVariantNameFR. Title-agnostic : Infinite
	// (qui a pair_name) ignore ce fallback.
	GameVariantID string
	// GameVariantNameFR : nom localisé de la variante de jeu, résolu read-time
	// depuis asset_translations[game_variant_id]. Alimente le fallback de mode
	// (squadModeUI) quand pair_name/pair_name_fr sont vides.
	GameVariantNameFR string
	PlaylistName      string
	IsFirefight       bool
	IsRanked          bool
	Outcome           int
	Kills             int
	Deaths            int
	Assists           int
	KDA               *float64
	Accuracy          *float64
	TimePlayedSecs    int
	DurationSeconds   int
	// GameplayDurationSeconds : durée réelle de gameplay (countdown retranché).
	GameplayDurationSeconds int
	// T0Ms : offset du countdown pré-match en ms (real_start_time − start_time_utc).
	// nil si real_start_time absent → T0=0 (chronologie brute). Cf. Match Timeline T0.
	T0Ms             *int64
	TeamMMR          float64
	SessionID        *int
	SessionLabel     *string
	PerformanceScore *float64
	IsWithFriends    bool
	// Sprint N : métriques précision avancée
	HeadshotKills  int
	PerfectKills   int
	EnemyMMR       *float64
	MyTeamScore    *int
	EnemyTeamScore *int
	MapID          string
	PlaylistID     string
	// ExpectedWinProb : proba de victoire pré-match ∈ [0,1] (LUSR v2), chargée
	// depuis player.match_skill_rank. Nil si pré-v2 / non disponible.
	ExpectedWinProb *float64
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

// AllyParticipant est une ligne participant côté équipe alliée d'un match.
// Chargé par SquadRepository.LoadMainTeamParticipants pour alimenter le calcul
// des badges d'impact (analysis.ComputeMatchImpactFull) en périmètre team-wide
// (toute l'équipe du joueur principal, pas seulement le squad sélectionné).
type AllyParticipant struct {
	MatchID  string
	XUID     string
	Gamertag string
	Kills    int
	Deaths   int
	Assists  int
	Outcome  int
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
//
// P7.1 (revue 2026-04-29) : champs renommés `RowKey/ColKey` (axes ECharts)
// → `MapName/ModeName` (sémantique métier — la heatmap est toujours
// map × mode pour la synthèse Squad).
type HeatmapCell struct {
	MapName  string  `json:"map_name"`
	ModeName string  `json:"mode_name"`
	Value    float64 `json:"value"`
	Count    int     `json:"count"`
}

// TopWeekEntry est une semaine performante dans l'historique du joueur.
// WeekStart est l'ISO date (YYYY-MM-DD) du lundi 00:00 UTC de la semaine,
// utilisable pour un tri chronologique côté front (week_label "DD/MM" perd l'année).
type TopWeekEntry struct {
	WeekLabel  string  `json:"week_label"`
	WeekStart  string  `json:"week_start"`
	WinRate    float64 `json:"win_rate"`
	Wins       int     `json:"wins"`
	AvgKills   float64 `json:"avg_kills"`
	AvgDeaths  float64 `json:"avg_deaths"`
	AvgKDA     float64 `json:"avg_kda"`
	MatchCount int     `json:"match_count"`
}

// SynthesisKPIs contient les métriques agrégées solo ou escouade (Sprint 43).
type SynthesisKPIs struct {
	MatchCount             int      `json:"match_count"`
	RankedMatchCount       int      `json:"ranked_match_count"`
	Wins                   int      `json:"wins"`
	TotalTimePlayedSeconds int      `json:"total_time_played_seconds"`
	KDRatio                *float64 `json:"kd_ratio"`
	WinRate                float64  `json:"win_rate"`
	Accuracy               *float64 `json:"accuracy"`
	KillsPerMin            *float64 `json:"kills_per_min"`
	AvgLifeSeconds         *float64 `json:"avg_life_seconds"`
	PerformanceScore       *float64 `json:"performance_score"`
	HeadshotsPerMatch      *float64 `json:"headshots_per_match"`
	DeathsPerMin           *float64 `json:"deaths_per_min"`
	AssistsPerMin          *float64 `json:"assists_per_min"`
	AvgMaxKillingSpree     *float64 `json:"avg_max_killing_spree"`
	AvgDamageDealt         *float64 `json:"avg_damage_dealt"`
	AvgDamageTaken         *float64 `json:"avg_damage_taken"`
	PerfectKillsPerMatch   *float64 `json:"perfect_kills_per_match"`
	AvgOffensiveConversion *float64 `json:"avg_offensive_conversion,omitempty"`
	AvgDefensiveResistance *float64 `json:"avg_defensive_resistance,omitempty"`
}

// ComparisonMetricItem est une métrique bipolaire solo / escouade.
//
// P7.1 (revue 2026-04-29) : champs `SoloText/SquadText` retirés — le
// formatage (pourcentage, décimales, suffixes) est résolu côté front via
// les helpers `formatPercent`/`formatNumber` à partir des valeurs brutes.
// Réduit le couplage formatage/transport et permet aux clients front de
// formater selon la locale active.
type ComparisonMetricItem struct {
	Label      string  `json:"label"`
	SoloValue  float64 `json:"solo_value"`
	SquadValue float64 `json:"squad_value"`
}

// TemporalHeatmapCell est une cellule de la heatmap jour × heure (Sprint 43, enrichie P9).
// Contient le count d'activité + wins et win_rate pour la spec originale (heatmap colorée par WR, count overlay).
type TemporalHeatmapCell struct {
	DOW     int     `json:"dow"`      // 0 = lundi … 6 = dimanche
	Hour    int     `json:"hour"`     // 0–23
	Count   int     `json:"count"`    // activité
	Wins    int     `json:"wins"`     // victoires dans cette cellule
	WinRate float64 `json:"win_rate"` // wins / count
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

// SynthesisMatchRow déplacé vers `internal/legacymatch` (P4.3 finale cleanup).
