// Package domain — types de la page Carrière.
package domain

import "time"

// ---------------------------------------------------------------------------
// Types de transfert bruts (repo → service, pas de tags JSON)
// ---------------------------------------------------------------------------

// CareerRankData est le type de transfert brut pour la progression de rang.
//
// XPHeroTotal / RankMax (optionnels) portent les bornes de progression « Héros »
// PAR TITRE (HINF : 9 319 350 XP / 272 rangs ; Halo 5 : 50 000 000 XP / 152 SR).
// Nil = non fourni par la source → le service retombe sur les constantes par
// défaut (Halo Infinite). Title-agnostic : aucune valeur de jeu en dur dans le
// chemin de calcul du service.
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
	XPHeroTotal   *int
	RankMax       *int
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

// ---------------------------------------------------------------------------
// CSR — classements compétitifs par playlist
// ---------------------------------------------------------------------------

// CareerCSRRank est un instantané de classement CSR (current/season/alltime).
type CareerCSRRank struct {
	Value                       float64 `json:"value"`
	Tier                        string  `json:"tier"`
	SubTier                     int     `json:"sub_tier"`
	MeasurementMatchesRemaining int     `json:"measurement_matches_remaining"`
	BadgeImageURL               *string `json:"badge_image_url,omitempty"`
	// PlacementTotal = seuil placement de la saison du snapshot (5 depuis S3,
	// 10 historique). Toujours présent depuis Phase 6 du plan pipeline CSR.
	PlacementTotal int `json:"placement_total"`
}

// CareerPlaylistCSR regroupe les classements d'un joueur pour une playlist ranked.
type CareerPlaylistCSR struct {
	PlaylistID   string        `json:"playlist_id"`
	PlaylistName string        `json:"playlist_name"`
	Queue        string        `json:"queue"`
	Input        string        `json:"input"`
	Current      CareerCSRRank `json:"current"`
	Season       CareerCSRRank `json:"season"`
	AllTime      CareerCSRRank `json:"all_time"`
}

// CSRSeasonOption décrit une saison CSR sélectionnable dans le menu déroulant
// "Classements" (page Carrière). Une saison apparaît si le joueur y a des
// données classées (snapshot CSR), plus la saison courante.
type CSRSeasonOption struct {
	SeasonID  string `json:"season_id"`            // ex: "CsrSeason13-1"
	Label     string `json:"label"`                // ex: "Saison 13" (FR)
	IsCurrent bool   `json:"is_current,omitempty"` // saison active du jour
}

// CareerCSRResponse est la réponse de GET /pages/career/csrs.
// SeasonID = saison effectivement retournée (sélectionnée ou courante par défaut).
// AvailableSeasons = saisons proposables dans le menu (CSR uniquement ; LUSR est
// cumulatif, hors saison).
type CareerCSRResponse struct {
	Playlists        []CareerPlaylistCSR `json:"playlists"`
	SeasonID         string              `json:"season_id"`
	AvailableSeasons []CSRSeasonOption   `json:"available_seasons"`
}

// TopMatchDTO représente un match dans le top/pire performance.
type TopMatchDTO struct {
	MatchID          string   `json:"match_id"`
	StartTime        *string  `json:"start_time"`
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
// match_id + outcome + section (1=best, 2=worst) + had_bot_teammate. Le
// service délègue ensuite l'enrichissement à MatchHistoryService via la
// whitelist MatchIDs.
//
// HadBotTeammate : propagé jusqu'au front pour affichage d'un badge sur les
// best_matches (un LOSS avec bot est déjà exclu côté repo, le flag est donc
// toujours pertinent en best_matches uniquement).
type HighlightMatchIDRow struct {
	MatchID        string
	Outcome        int
	Section        int // 1 = best, 2 = worst
	HadBotTeammate bool
}

// HighlightMatchPoolRow : ligne légère du pool éligible (mêmes filtres
// d'éligibilité que Q9b mais sans contrainte d'outcome ni LIMIT). Utilisée
// pour calculer les cascade counts (available_experience, available_seasons,
// available_modes, available_playlists).
//
// ModeUISource     = COALESCE(pair_name_fr, pair_name) brut (sert au filtre SQL).
// ModeUI           = label normalisé final affiché côté front
//
//	(analysis.NormalizeModeLabel + override mode_name_tr FR).
//
// PlaylistNameRaw  = COALESCE(playlist_name_fr, playlist_name) brut (filtre SQL).
// PlaylistName     = label final affiché côté front (override asset_translations FR
//
//	si PlaylistNameRaw est encore EN après COALESCE).
//
// PlaylistID       = playlist_id (UUID) pour le lookup asset_translations.
type HighlightMatchPoolRow struct {
	MatchID         string
	IsRanked        bool
	StartTime       *time.Time
	ModeUI          string
	ModeUISource    string
	PlaylistName    string
	PlaylistNameRaw string
	PlaylistID      string
}

// CareerHighlightFilters : filtres résolus passés au repo. Champs zéro = pas de filtre.
type CareerHighlightFilters struct {
	// Experience : "all" / "ranked" / "unranked". Vide ou "all" = pas de filtre.
	Experience string
	// SeasonRanges : list de fenêtres temporelles pré-résolues par le service
	// depuis la sélection de saisons + le SeasonsCatalog. Vide = pas de filtre.
	SeasonRanges []SeasonTimeRange
	// ModeRawSources : valeurs brutes COALESCE(pair_name_fr, pair_name) expandées
	// par le service depuis la sélection utilisateur (labels normalisés) via le
	// pool. Utilisées tel quel dans la clause SQL `COALESCE IN (...)`.
	ModeRawSources []string
	// PlaylistNamesRaw : valeurs brutes COALESCE(playlist_name_fr, playlist_name)
	// expandées par le service depuis la sélection utilisateur (labels FR via
	// asset_translations) via le pool. Utilisées tel quel dans la clause SQL.
	PlaylistNamesRaw []string
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
	Experience    string   // "all" | "ranked" | "unranked"
	SeasonIDs     []string // ex. ["season6", "season10_op1"]
	ModeUIs       []string // valeurs pair_name (= mode_ui) sélectionnées
	PlaylistNames []string // valeurs playlist_name sélectionnées
}

// HighlightMatchesData : résultat agrégé de service.GetHighlightMatchIDs
// incluant les rows brutes (best+worst, à enrichir par le handler) et les
// cascade counts pour les dropdowns Expérience / Saisons / Modes / Playlists.
type HighlightMatchesData struct {
	Rows                []HighlightMatchIDRow
	AvailableExperience []HighlightExperienceCount
	AvailableSeasons    []HighlightSeasonCount
	AvailableModes      []HighlightModeCount
	AvailablePlaylists  []HighlightPlaylistCount
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

// HighlightModeCount : count par mode (pair_name) pour la dropdown Modes.
// Value = pair_name (= mode_ui dans ExplorerMatchRow).
type HighlightModeCount struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// HighlightPlaylistCount : count par playlist pour la dropdown Playlists.
// Value = playlist_name (= playlist_label dans ExplorerMatchRow).
type HighlightPlaylistCount struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// CareerHighlightMatchesResponse est la réponse de
// GET /pages/career/highlight-matches : tableau Explorer-format pour les
// 15 meilleurs et 15 pires matchs (toggle Best/Worst côté front).
//
// Tous les champs Available* sont cascade-aware : les counts respectent
// tous les autres filtres actifs — alimentent les dropdowns à gauche du toggle.
type CareerHighlightMatchesResponse struct {
	BestMatches         []ExplorerMatchesRow       `json:"best_matches"`
	WorstMatches        []ExplorerMatchesRow       `json:"worst_matches"`
	AvailableExperience []HighlightExperienceCount `json:"available_experience"`
	AvailableSeasons    []HighlightSeasonCount     `json:"available_seasons"`
	AvailableModes      []HighlightModeCount       `json:"available_modes"`
	AvailablePlaylists  []HighlightPlaylistCount   `json:"available_playlists"`
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
