// Package domain — types de la page Carrière.
package domain

import "time"

// ---------------------------------------------------------------------------
// Types de transfert bruts (repo → service, pas de tags JSON)
// ---------------------------------------------------------------------------

// CareerRankData est le type de transfert brut pour la progression de rang.
type CareerRankData struct {
	RankNumber    int
	CurrentXP     int
	RecordedAt    time.Time
	RankLabel     *string
	RankName      *string
	RankTier      *string
	XPForNextRank *int
	XPTotal       *int
	IsMaxRank     bool
}

// TopMatchRawRow est le type de transfert brut pour les top matches.
type TopMatchRawRow struct {
	MatchID          string
	PerformanceScore float64
	StartTime        *time.Time
	MapName          *string
	PairName         *string
	PlaylistName     *string
	Outcome          int
	Kills            int
	Deaths           int
	KDA              *float64
	TeamMMR          *float64
	EnemyMMR         *float64
	DominanceFlag    int
}

// EncounterRawRow est le type de transfert brut pour les encounters.
type EncounterRawRow struct {
	XUID       string
	Gamertag   string
	MatchCount int
	AsTeammate int
	AsEnemy    int
	AvgKDA     *float64
}

// ---------------------------------------------------------------------------
// DTOs API (service → handler → JSON)
// ---------------------------------------------------------------------------

// CareerRankSummary représente le rang actuel du joueur.
type CareerRankSummary struct {
	RankNumber    int        `json:"rank_number"`
	RankLabel     string     `json:"rank_label"`
	RankNameRaw   string     `json:"rank_name_raw"`
	RankTier      string     `json:"rank_tier"`
	CurrentXP     int        `json:"current_xp"`
	XPForNextRank int        `json:"xp_for_next_rank"`
	XPTotal       int        `json:"xp_total"`
	ProgressPct   float64    `json:"progress_pct"`
	IsMaxRank     bool       `json:"is_max_rank"`
	RecordedAt    *time.Time `json:"recorded_at"`
}

// HeroProgress représente la progression vers le rang maximum.
type HeroProgress struct {
	XPTotalRequired int     `json:"xp_total_required"`
	XPRemaining     int     `json:"xp_remaining"`
	Percentage      float64 `json:"percentage"`
	CurrentRank     int     `json:"current_rank"`
}

// CareerProjections représente les projections de date d'atteinte du rang max.
type CareerProjections struct {
	XPPerDayActive       float64 `json:"xp_per_day_active"`
	XPPerDayFallback     float64 `json:"xp_per_day_fallback"`
	EstimatedHeroDate    *string `json:"estimated_hero_date"`
	EstimatedRankCapDate *string `json:"estimated_rank_cap_date"`
}

// XPHistoryPoint représente un point dans l'historique XP.
// Champs alignés avec les colonnes de Q7CareerXPHistory.
type XPHistoryPoint struct {
	RecordedAt   time.Time `json:"recorded_at"`
	RankNumber   int       `json:"rank_number"`
	CurrentXP    int       `json:"current_xp"`
	XPTotalCumul int       `json:"xp_total_cumul"`
}

// LUSRCheckpointDTO représente un checkpoint LUSR pour l'API.
// Champs alignés avec les colonnes de Q8LUSRHistory.
type LUSRCheckpointDTO struct {
	MatchID       string     `json:"-"`
	RatingValue   float64    `json:"rating_value"`
	TierLabel     *string    `json:"tier_label"`
	PlaylistGroup *string    `json:"playlist_group"`
	RecordedAt    *time.Time `json:"recorded_at"`
	RatingDelta   *float64   `json:"rating_delta,omitempty"`
}

// LUSRSummary résume le rating LUSR actuel.
type LUSRSummary struct {
	CurrentRating        *float64            `json:"current_rating"`
	CurrentTierLabel     *string             `json:"current_tier_label"`
	CurrentPlaylistGroup *string             `json:"current_playlist_group"`
	TrendLabel           *string             `json:"trend_label"`
	Checkpoints          []LUSRCheckpointDTO `json:"checkpoints"`
}

// CareerPageResponse est la réponse de GET /pages/career.
type CareerPageResponse struct {
	Summary       CareerRankSummary    `json:"summary"`
	HeroProgress  HeroProgress         `json:"hero_progress"`
	Projections   CareerProjections    `json:"projections"`
	XPHistory     []XPHistoryPoint     `json:"xp_history"`
	LUSR          LUSRSummary          `json:"lusr"`
	CurrentSeason *CurrentSeasonResult `json:"current_season,omitempty"` // Sprint 54-A7
}

// TopMatchDTO représente un match dans le top/pire performance.
type TopMatchDTO struct {
	MatchID          string   `json:"match_id"`
	PerformanceScore float64  `json:"performance_score"`
	MapUI            *string  `json:"map_ui"`
	ModeUI           *string  `json:"mode_ui"`
	OutcomeCode      int      `json:"outcome_code"`
	OutcomeLabel     string   `json:"outcome_label"`
	Kills            int      `json:"kills"`
	Deaths           int      `json:"deaths"`
	KDA              *float64 `json:"kda"`
}

// CareerTopMatchesResponse est la réponse de GET /pages/career/top-matches.
type CareerTopMatchesResponse struct {
	BestMatches  []TopMatchDTO `json:"best_matches"`
	WorstMatches []TopMatchDTO `json:"worst_matches"`
}

// EncounterDTO représente un adversaire/coéquipier fréquent.
type EncounterDTO struct {
	Gamertag   string   `json:"gamertag"`
	XUID       string   `json:"xuid"`
	MatchCount int      `json:"match_count"`
	AsTeammate int      `json:"as_teammate"`
	AsEnemy    int      `json:"as_enemy"`
	AvgKDA     *float64 `json:"avg_kda"`
}

// CareerEncountersResponse est la réponse de GET /pages/career/encounters.
type CareerEncountersResponse struct {
	Teammates []EncounterDTO `json:"teammates"`
	Enemies   []EncounterDTO `json:"enemies"`
	Total     int            `json:"total"`
}

// GamertagSearchResult est un résultat de la recherche gamertag.
type GamertagSearchResult struct {
	Gamertag   string  `json:"gamertag"`
	XUID       string  `json:"xuid"`
	Score      float64 `json:"score"`
	ExactMatch bool    `json:"exact_match"`
}

// GamertagSearchResponse est la réponse de GET /directory/gamertags/search.
type GamertagSearchResponse struct {
	Query string                 `json:"query"`
	Items []GamertagSearchResult `json:"items"`
}
