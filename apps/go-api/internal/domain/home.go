// Package domain — home.go : types pour la page d'accueil Mission Control.
//
// Correspond à l'endpoint GET /api/v1/players/{slug}/pages/home.
// Les types BattlePassResponse et ChallengesResponse sont retournés
// directement par leurs propres endpoints (best-effort, nécessite auth Sprint 15).
package domain

import "time"

// ---------------------------------------------------------------------------
// Lignes brutes DuckDB
// ---------------------------------------------------------------------------

// HomeMatchRow est une ligne brute chargée depuis Q26 (matchs du home).
type HomeMatchRow struct {
	MatchID            string
	StartTime          time.Time
	MapID              string
	MapName            string
	MapNameFR          string
	PairID             string
	PairName           string
	PairNameFR         string
	GameVariantID      string
	GameVariantName    string
	GameVariantNameFR  string
	PlaylistID         string
	PlaylistName       string
	PlaylistNameFR     string
	IsFirefight        bool
	IsRanked           bool
	SessionLabel       *string
	IsWithFriends      bool
	Outcome            int
	TeamID             int
	Team0Score         int
	Team1Score         int
	DominanceFlag      int
	Kills              int
	Deaths             int
	Assists            int
	KDA                *float64
	Ratio              *float64
	Accuracy           *float64
	AvgLifeSeconds     *float64
	TimePlayedSecs     *int
	DamageDealt        *float64
	DamageTaken        *float64
	TeamMMR            *float64
	EnemyMMR           *float64
	PerformanceScore   *float64
	SkillRatingValue   *float64
	SkillRatingType    string
	SkillTier          *string
	SkillSubTier       int
	SkillTierLabel     *string
	SkillRatingDelta   *float64
	SkillPlaylistGroup *string
	SkillRankImageURL  *string
	RankInTeam         *int
	HeadshotKills      int
	PerfectKills       int
}

// HomeSessionRow est une ligne brute chargée depuis Q27 (sessions enrichment).
type HomeSessionRow struct {
	MatchID       string
	SessionID     *int
	SessionLabel  *string
	IsWithFriends bool
	StartTime     *time.Time
}

// HomeMediaRow est une ligne brute chargée depuis Q28 (médias récents).
type HomeMediaRow struct {
	FileName       string
	MatchID        *string
	MatchStartTime *time.Time
}

// HomeSpartanIdentityRow est la projection brute de l'identité record pour la home.
type HomeSkillPeakRow struct {
	RatingValue   float64
	TierLabel     *string
	BadgeImageURL *string
}

// HomeSpartanIdentityRow est la projection brute de l'identité record pour la home.
type HomeSpartanIdentityRow struct {
	SpartanID         *string
	BannerImageURL    *string
	EmblemImageURL    *string
	BackdropImageURL  *string
	HighestCSR        *HomeSkillPeakRow
	HighestLUSR       *HomeSkillPeakRow
	RankNumber        int
	RankName          *string
	RankTier          *string
	RankTitleEN       *string
	RankTitleFR       *string
	RankImageURL      *string
	AdornmentImageURL *string
	CurrentXP         int
	XPForNextRank     int
	IsMaxRank         bool
}

// ---------------------------------------------------------------------------
// Blocs Hero Card
// ---------------------------------------------------------------------------

// HeroKPIs contient les KPIs globaux du joueur affichés dans le hero card.
type HeroKPIs struct {
	WinRate                float64  `json:"win_rate"`
	GlobalRatio            *float64 `json:"global_ratio,omitempty"`
	AvgKDA                 *float64 `json:"avg_kda,omitempty"`
	AvgAccuracy            *float64 `json:"avg_accuracy,omitempty"`
	TotalMatches           int      `json:"total_matches"`
	Wins                   int      `json:"wins"`
	Draws                  int      `json:"draws"`
	DNFs                   int      `json:"dnfs"`
	Losses                 int      `json:"losses"`
	TotalPlaytimeSecs      int      `json:"total_playtime_secs"`
	FavoriteWeaponName     string   `json:"favorite_weapon_name,omitempty"`
	FavoriteWeaponKills    int      `json:"favorite_weapon_kills,omitempty"`
	AvgOffensiveConversion *float64 `json:"avg_offensive_conversion,omitempty"`
	AvgDefensiveResistance *float64 `json:"avg_defensive_resistance,omitempty"`
}

// HeroTrend représente la variation des métriques clés sur une fenêtre glissante.
// Calcul : 5 derniers matchs vs 5 précédents.
type HeroTrend struct {
	RatioDelta    *float64 `json:"ratio_delta,omitempty"`
	AccuracyDelta *float64 `json:"accuracy_delta,omitempty"`
	WinRateDelta  *float64 `json:"win_rate_delta,omitempty"`
}

// HomeHeroCard est la carte briefing principale affichée en haut de l'accueil.
type HomeHeroCard struct {
	PlayerName string     `json:"player_name"`
	KPIs       HeroKPIs   `json:"kpis"`
	Trend      *HeroTrend `json:"trend,omitempty"`
}

// HomeCareerRankSummary représente le rang carrière courant dans la home.
type HomeCareerRankSummary struct {
	RankNumber        int     `json:"rank_number"`
	RankTitle         string  `json:"rank_title"`
	RankImageURL      *string `json:"rank_image_url,omitempty"`
	AdornmentImageURL *string `json:"adornment_image_url,omitempty"`
	CurrentXP         int     `json:"current_xp"`
	XPForNextRank     int     `json:"xp_for_next_rank"`
	ProgressPct       float64 `json:"progress_pct"`
	IsMaxRank         bool    `json:"is_max_rank"`
}

// HomeSkillPeakSummary représente un pic historique CSR ou LUSR affiché sur la home.
type HomeSkillPeakSummary struct {
	RatingValue   float64 `json:"rating_value"`
	TierLabel     *string `json:"tier_label,omitempty"`
	BadgeImageURL *string `json:"badge_image_url,omitempty"`
}

// HomePlaylistRank associe une playlist récente à son dernier rang compétitif connu.
type HomePlaylistRank struct {
	PlaylistName  string   `json:"playlist_name"`
	IsRanked      bool     `json:"is_ranked"`
	RatingType    *string  `json:"rating_type,omitempty"` // "CSR" | "LUSR" — nil si aucun rang calculé
	RatingValue   *float64 `json:"rating_value,omitempty"`
	TierLabel     *string  `json:"tier_label,omitempty"`
	BadgeImageURL *string  `json:"badge_image_url,omitempty"`
}

// HomeSpartanIdentity représente le bloc identitaire compact de la home.
type HomeSpartanIdentity struct {
	BannerImageURL   *string                `json:"banner_image_url,omitempty"`
	SpartanID        *string                `json:"spartan_id,omitempty"`
	EmblemImageURL   *string                `json:"emblem_image_url,omitempty"`
	BackdropImageURL *string                `json:"backdrop_image_url,omitempty"`
	HighestCSR       *HomeSkillPeakSummary  `json:"highest_csr,omitempty"`
	HighestLUSR      *HomeSkillPeakSummary  `json:"highest_lusr,omitempty"`
	CareerRank       *HomeCareerRankSummary `json:"career_rank,omitempty"`
}

// ---------------------------------------------------------------------------
// Signaux et highlights
// ---------------------------------------------------------------------------

// HighlightItem est un fait saillant synthétique pour la zone signaux.
type HighlightItem struct {
	Title  string `json:"title"`
	Value  string `json:"value"`
	Detail string `json:"detail"`
}

// ---------------------------------------------------------------------------
// Matchs récents
// ---------------------------------------------------------------------------

// RecentMatchItem représente un match récent dans la timeline.
type RecentMatchItem struct {
	MatchID         string     `json:"match_id"`
	Title           string     `json:"title"`
	Detail          string     `json:"detail"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	OutcomeLabel    string     `json:"outcome_label"`
	OutcomeTone     string     `json:"outcome_tone"`
	ScoreLabel      *string    `json:"score_label,omitempty"`
	NarrativeBadges []string   `json:"narrative_badges,omitempty"`
	IsFavorite      bool       `json:"is_favorite"`
	// S56 — champs enrichis pour MatchCard
	MapUI                    *string                `json:"map_ui,omitempty"`
	ModeUI                   *string                `json:"mode_ui,omitempty"`
	PlaylistUI               *string                `json:"playlist_ui,omitempty"`
	Kills                    *int                   `json:"kills,omitempty"`
	Deaths                   *int                   `json:"deaths,omitempty"`
	Assists                  *int                   `json:"assists,omitempty"`
	PerformanceScoreRelative *int                   `json:"performance_score_relative,omitempty"`
	OffensiveConversion      *float64               `json:"offensive_conversion,omitempty"`
	DefensiveResistance      *float64               `json:"defensive_resistance,omitempty"`
	DamageDealt              *float64               `json:"damage_dealt,omitempty"`
	DamageTaken              *float64               `json:"damage_taken,omitempty"`
	MapImageURL              *string                `json:"map_image_url,omitempty"`
	SkillRatingValue         *int                   `json:"skill_rating_value,omitempty"`
	SkillRatingType          *string                `json:"skill_rating_type,omitempty"`
	SkillTierLabel           *string                `json:"skill_tier_label,omitempty"`
	SkillRatingDelta         *float64               `json:"skill_rating_delta,omitempty"`
	SkillPlaylistGroup       *string                `json:"skill_playlist_group,omitempty"`
	SkillRankImageURL        *string                `json:"skill_rank_image_url,omitempty"`
	SkillProgressPct         *float64               `json:"skill_progress_pct,omitempty"`
	SkillPointsInTier        *int                   `json:"skill_points_in_tier,omitempty"`
	KDA                      *float64               `json:"kda,omitempty"`
	DurationSecs             *int                   `json:"duration_secs,omitempty"`
	Accuracy                 *float64               `json:"accuracy,omitempty"`
	AvgLifeSecs              *float64               `json:"avg_life_secs,omitempty"`
	TeamMMR                  *float64               `json:"team_mmr,omitempty"`
	EnemyMMR                 *float64               `json:"enemy_mmr,omitempty"`
	DeltaMMR                 *float64               `json:"delta_mmr,omitempty"`
	IsWithFriends            *bool                  `json:"is_with_friends,omitempty"`
	RankInTeam               *int                   `json:"rank_in_team,omitempty"`
	HeadshotKills            *int                   `json:"headshot_kills,omitempty"`
	PerfectKills             *int                   `json:"perfect_kills,omitempty"`
	TopMedals                []RecentMatchMedal     `json:"top_medals,omitempty"`
	TopCitations             []MatchCitationSnippet `json:"top_citations,omitempty"`
}

// RecentMatchMedal est une médaille compacte pour l'affichage dans MatchCard.
type RecentMatchMedal struct {
	MedalID     int64  `json:"medal_id"`
	Name        string `json:"name"`
	Count       int    `json:"count"`
	Description string `json:"description,omitempty"`
	ImageURL    string `json:"image_url"`
}

// MatchCitationSnippet est une citation progressée dans un match, pour l'affichage MatchCard.
type MatchCitationSnippet struct {
	Key             string  `json:"key"`
	Name            string  `json:"name"`
	Description     *string `json:"description,omitempty"`
	ImageURL        *string `json:"image_url,omitempty"`
	Delta           int     `json:"delta"`
	ProgressPct     float64 `json:"progress_pct"`
	IsNewlyMastered bool    `json:"is_newly_mastered,omitempty"`
}

// HomeMatchCitationRaw est une ligne brute merger depuis match_citations + citation_mappings.
type HomeMatchCitationRaw struct {
	Norm        string
	Display     string
	Description string
	ImagePath   string
	TierTargets string // CSV ex: "10,20,30,50,100"
	Delta       int    // valeur du match (mc.value)
	Cumulative  int    // SUM(value) sur tous les matchs jusqu'à ce match inclus
}

// ---------------------------------------------------------------------------
// Résumé de session
// ---------------------------------------------------------------------------

// SessionSummaryItem est le résumé d'une session (solo ou escouade).
type SessionSummaryItem struct {
	SessionLabel         string     `json:"session_label"`
	MatchCount           int        `json:"match_count"`
	WinRate              float64    `json:"win_rate"`
	GlobalRatio          *float64   `json:"global_ratio,omitempty"`
	StartedAt            *time.Time `json:"started_at,omitempty"`
	EndedAt              *time.Time `json:"ended_at,omitempty"`
	Wins                 int        `json:"wins"`
	Losses               int        `json:"losses"`
	Draws                int        `json:"draws"`
	DNFs                 int        `json:"dnfs"`
	AvgPlayerPerformance *float64   `json:"avg_player_performance,omitempty"`
	AvgTeamPerformance   *float64   `json:"avg_team_performance,omitempty"`
	AvgKDA               *float64   `json:"avg_kda,omitempty"`
	DominantPlaylist     *string    `json:"dominant_playlist,omitempty"`
	DominantMode         *string    `json:"dominant_mode,omitempty"`
}

// ---------------------------------------------------------------------------
// Médias récents
// ---------------------------------------------------------------------------

// RecentMediaItem est une entrée compacte d'un média récent.
type RecentMediaItem struct {
	Basename       string     `json:"basename"`
	MatchID        *string    `json:"match_id,omitempty"`
	MatchStartTime *time.Time `json:"match_start_time,omitempty"`
}

// ---------------------------------------------------------------------------
// Réponses API
// ---------------------------------------------------------------------------

// HomePageResponse est la réponse agrégée de la page d'accueil Mission Control.
type HomePageResponse struct {
	Hero                HomeHeroCard         `json:"hero"`
	SpartanIdentity     *HomeSpartanIdentity `json:"spartan_identity,omitempty"`
	Highlights          []HighlightItem      `json:"highlights"`
	RecentMatches       []RecentMatchItem    `json:"recent_matches"`
	FavoriteMatches     []RecentMatchItem    `json:"favorite_matches"`
	RecentMedia         []RecentMediaItem    `json:"recent_media"`
	SoloSession         *SessionSummaryItem  `json:"solo_session,omitempty"`
	SquadSession        *SessionSummaryItem  `json:"squad_session,omitempty"`
	SoloSessions        []SessionSummaryItem `json:"solo_sessions,omitempty"`
	SquadSessions       []SessionSummaryItem `json:"squad_sessions,omitempty"`
	HasRankedHistory    bool                 `json:"has_ranked_history"`
	HasUnrankedHistory  bool                 `json:"has_unranked_history"`
	RecentPlaylistRanks []HomePlaylistRank   `json:"recent_playlist_ranks,omitempty"`
}

// BattlePassResponse contient les informations Battle Pass live ou depuis le cache DB.
// available=false avec error_hint="auth_required" tant que l'auth n'est pas portée (Sprint 15).
type BattlePassResponse struct {
	Available   bool    `json:"available"`
	Rank        *int    `json:"rank,omitempty"`
	RewardTrack *string `json:"reward_track,omitempty"`
	Progress    *int    `json:"progress,omitempty"`
	FromCache   bool    `json:"from_cache,omitempty"`
	ErrorHint   *string `json:"error_hint,omitempty"`
}

// ChallengeItem représente un défi actif détaillé pour l'accueil.
type ChallengeItem struct {
	ChallengePath   string   `json:"challenge_path"`
	TrackingID      *string  `json:"tracking_id,omitempty"`
	Title           string   `json:"title"`
	Description     *string  `json:"description,omitempty"`
	ImageURL        *string  `json:"image_url,omitempty"`
	ProgressCurrent *int     `json:"progress_current,omitempty"`
	ProgressTarget  *int     `json:"progress_target,omitempty"`
	ProgressPercent *float64 `json:"progress_percent,omitempty"`
	XPReward        *int     `json:"xp_reward,omitempty"`
}

// ChallengesResponse contient le résumé des défis actifs ou depuis le cache DB.
// available=false avec error_hint="auth_required" tant que l'auth n'est pas portée (Sprint 15).
type ChallengesResponse struct {
	Available   bool            `json:"available"`
	Total       *int            `json:"total,omitempty"`
	Completed   *int            `json:"completed,omitempty"`
	XPAvailable *int            `json:"xp_available,omitempty"`
	NextExpiry  *string         `json:"next_expiry,omitempty"`
	Items       []ChallengeItem `json:"items,omitempty"`
	FromCache   bool            `json:"from_cache,omitempty"`
	ErrorHint   *string         `json:"error_hint,omitempty"`
}
