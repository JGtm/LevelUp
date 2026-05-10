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
	RankNumber       int        `json:"rank_number"`
	RankLabel        string     `json:"rank_label"`
	RankNameRaw      string     `json:"rank_name_raw"`
	RankTier         string     `json:"rank_tier"`
	CurrentXP        int        `json:"current_xp"`
	XPForNextRank    int        `json:"xp_for_next_rank"`
	XPTotal          int        `json:"xp_total"`
	ProgressPct      float64    `json:"progress_pct"`
	IsMaxRank        bool       `json:"is_max_rank"`
	RecordedAt       *time.Time `json:"recorded_at"`
	RankImageURL     *string    `json:"rank_image_url,omitempty"`
	NextRankNameFR   string     `json:"next_rank_name_fr,omitempty"`
	NextRankNameEN   string     `json:"next_rank_name_en,omitempty"`
	NextRankImageURL *string    `json:"next_rank_image_url,omitempty"`
}

// HeroProgress représente la progression vers le rang maximum.
type HeroProgress struct {
	XPTotalRequired int     `json:"xp_total_required"`
	XPRemaining     int     `json:"xp_remaining"`
	Percentage      float64 `json:"percentage"`
	CurrentRank     int     `json:"current_rank"`
	TotalRanks      int     `json:"total_ranks"`
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
	RecordedAt time.Time `json:"recorded_at"`
	Rank       int       `json:"rank"`
	CurrentXP  int       `json:"current_xp"`
	XPTotal    int       `json:"xp_total"`
}

// FriendXPHistory contient l'historique XP d'un ami (joueur ayant une DB locale).
type FriendXPHistory struct {
	Gamertag string           `json:"gamertag"`
	History  []XPHistoryPoint `json:"history"`
}

// LUSRCheckpointDTO représente un checkpoint LUSR pour l'API.
// Champs alignés avec les colonnes de Q8LUSRHistory.
type LUSRCheckpointDTO struct {
	MatchID       string     `json:"-"`
	RatingType    string     `json:"rating_type"`
	RatingValue   float64    `json:"rating_value"`
	TierLabel     *string    `json:"tier_label"`
	PlaylistGroup *string    `json:"playlist_group"`
	PlaylistName  string     `json:"playlist_name"`
	PlaylistID    string     `json:"-"`
	RecordedAt    *time.Time `json:"recorded_at"`
	RatingDelta   *float64   `json:"rating_delta,omitempty"`
	BadgeImageURL *string    `json:"badge_image_url,omitempty"`
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
	Summary          CareerRankSummary    `json:"summary"`
	HeroProgress     HeroProgress         `json:"hero_progress"`
	Projections      CareerProjections    `json:"projections"`
	XPHistory        []XPHistoryPoint     `json:"xp_history"`
	LUSR             LUSRSummary          `json:"lusr"`
	FriendsXPHistory []FriendXPHistory    `json:"friends_xp_history,omitempty"`
	CurrentSeason    *CurrentSeasonResult `json:"current_season,omitempty"` // Sprint 54-A7
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

// HighlightMatchIDRow est le type brut renvoyé par Q9bHighlightMatchIDs :
// match_id + outcome + section (1=best, 2=worst). Le service délègue ensuite
// l'enrichissement à MatchHistoryService via la whitelist MatchIDs.
type HighlightMatchIDRow struct {
	MatchID string
	Outcome int
	Section int // 1 = best, 2 = worst
}

// HighlightMatchPoolRow : ligne légère du pool éligible (mêmes filtres
// d'éligibilité que Q9b mais sans contrainte d'outcome ni LIMIT). Utilisée
// pour calculer les cascade counts (available_experience, available_seasons).
type HighlightMatchPoolRow struct {
	MatchID   string
	IsRanked  bool
	StartTime *time.Time
}

// CareerHighlightFilters : filtres optionnels appliqués sur la section
// "Matchs marquants" (page Carrière). Champs zéro = pas de filtre.
type CareerHighlightFilters struct {
	// Experience : "all" / "ranked" / "unranked". Vide ou "all" = pas de filtre.
	Experience string
	// SeasonRanges : list de fenêtres temporelles pré-résolues par le service
	// depuis la sélection de saisons + le SeasonsCatalog. Vide = pas de filtre.
	SeasonRanges []SeasonTimeRange
}

// SeasonTimeRange : fenêtre [Start, End) résolue depuis le SeasonsCatalog.
// End nil = saison ouverte (pas de borne supérieure).
type SeasonTimeRange struct {
	Start time.Time
	End   *time.Time
}

// HighlightFilterInput : filtres bruts venant du handler (query params) pour
// la section "Matchs marquants". Le service les résout en CareerHighlightFilters
// via SeasonsCatalog (mapping SeasonIDs → date-ranges concrètes).
type HighlightFilterInput struct {
	Experience string   // "all" | "ranked" | "unranked"
	SeasonIDs  []string // ex. ["season6", "season10_op1"]
}

// HighlightMatchesData : résultat agrégé de service.GetHighlightMatchIDs
// incluant les rows brutes (best+worst, à enrichir par le handler) et les
// cascade counts pour les dropdowns Expérience / Saisons.
type HighlightMatchesData struct {
	Rows                []HighlightMatchIDRow
	AvailableExperience []HighlightExperienceCount
	AvailableSeasons    []HighlightSeasonCount
}

// HighlightExperienceCount : count par option pour la dropdown Expérience.
// Value : "all" / "ranked" / "unranked".
type HighlightExperienceCount struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// HighlightSeasonCount : count par saison pour la dropdown Saisons.
// Value est l'ID de saison (ex. "season6"). Count est le nombre de matchs
// éligibles dans la fenêtre saison + filtre Expérience courant.
type HighlightSeasonCount struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// CareerHighlightMatchesResponse est la réponse de
// GET /pages/career/highlight-matches : tableau Explorer-format pour les
// 15 meilleurs et 15 pires matchs (toggle Best/Worst côté front).
//
// AvailableExperience et AvailableSeasons cascade-aware (counts respectent
// l'autre filtre actif) — alimentent les dropdowns à gauche du toggle.
type CareerHighlightMatchesResponse struct {
	BestMatches         []ExplorerMatchesRow       `json:"best_matches"`
	WorstMatches        []ExplorerMatchesRow       `json:"worst_matches"`
	AvailableExperience []HighlightExperienceCount `json:"available_experience"`
	AvailableSeasons    []HighlightSeasonCount     `json:"available_seasons"`
}

// CareerTopEncountersResponse est la réponse de
// GET /pages/career/top-encounters : 10 joueurs les plus croisés au niveau
// global, hors amis configurés. Réutilise MatchEncounterRow (même format que
// Match View > "Historique de rencontre").
type CareerTopEncountersResponse struct {
	Items []MatchEncounterRow `json:"items"`
}

// CareerRivalRawRow est le type brut renvoyé par Q27CareerRivals*.
type CareerRivalRawRow struct {
	XUID       string
	Gamertag   string
	Frags      int
	Deaths     int
	MatchCount int
}

// CareerRival représente une rivalité (top némésis ou top souffre-douleur)
// au niveau carrière globale du joueur.
type CareerRival struct {
	Gamertag   string  `json:"gamertag"`
	Frags      int     `json:"frags"`
	Deaths     int     `json:"deaths"`
	Ratio      float64 `json:"ratio"`
	MatchCount int     `json:"match_count"`
}

// CareerRivalsResponse est la réponse de GET /pages/career/rivals.
//
//	Nemeses = top 10 par deaths DESC (joueurs qui m'ont le plus tué).
//	Victims = top 10 par frags DESC  (joueurs que j'ai le plus tué).
type CareerRivalsResponse struct {
	Nemeses []CareerRival `json:"nemeses"`
	Victims []CareerRival `json:"victims"`
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
