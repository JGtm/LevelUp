// Package domain — types pour l'explorer (matchs communs, recherche joueurs).
//
// Port Go de apps/api/app/schemas/explorer.py.
// Routes : POST /players/{slug}/pages/explorer/player-query
//
//	POST /players/{slug}/pages/explorer/matches-query
//	GET  /directory/gamertags/search (déjà implémenté dans gamertag_repo)
package domain

import "time"

// ---------------------------------------------------------------------------
// Explorer — Player Query
// ---------------------------------------------------------------------------

// PageSizeCommonMatches est la taille de page fixe pour l'historique commun.
const PageSizeCommonMatches = 20

// ExplorerPlayerQueryRequest : corps de la requête POST player-query.
type ExplorerPlayerQueryRequest struct {
	TargetGamertag string `json:"target_gamertag"`
	Page           int    `json:"page,omitempty"` // 1-indexé, défaut 1
}

// CommonMatchRow : un match en commun entre 2 joueurs.
type CommonMatchRow struct {
	MatchID       string    `json:"match_id"`
	StartTime     time.Time `json:"start_time"`
	MapUI         string    `json:"map_ui"`
	ModeUI        string    `json:"mode_ui"`
	WereTeammates bool      `json:"were_teammates"`
	PlayerOutcome int       `json:"player_outcome"`
	OutcomeLabel  string    `json:"outcome_label"`
	Kills         int       `json:"kills"`
	Deaths        int       `json:"deaths"`
	KDA           float64   `json:"kda"`
}

// ExplorerPlayerQueryResponse : réponse de la requête player-query.
type ExplorerPlayerQueryResponse struct {
	TargetGamertag string                `json:"target_gamertag"`
	TargetXUID     string                `json:"target_xuid"`
	CommonMatches  []CommonMatchRow      `json:"common_matches"`
	Badges         []MatchEncounterBadge `json:"badges,omitempty"`
	Total          int                   `json:"total"`       // items sur la page courante
	TotalCount     int                   `json:"total_count"` // total tous matchs confondus
	WinsTogether   int                   `json:"wins_together"`
	LossesTogether int                   `json:"losses_together"`
	Page           int                   `json:"page"`
	PageSize       int                   `json:"page_size"`
}

// KillerVictimAggregate : kills croisés agrégés entre deux joueurs.
type KillerVictimAggregate struct {
	KillsDealt     int // kills du joueur courant sur le joueur cible
	DeathsSuffered int // kills du joueur cible sur le joueur courant
}

// ---------------------------------------------------------------------------
// Explorer — Matches Query
// ---------------------------------------------------------------------------

// ExplorerMatchesQueryRequest : corps de POST matches-query.
// Accepte les mêmes filtres que MatchHistoryQueryRequest.
type ExplorerMatchesQueryRequest struct {
	Filters    FilterContextInput `json:"filters"`
	Pagination PaginationRequest  `json:"pagination"`
	SortField  string             `json:"sort_field"`
	SortDir    string             `json:"sort_dir"`
	// PerfTiers filtre par palier de performance (1=Excellent … 5=Mauvais).
	// Liste vide = pas de filtre.
	PerfTiers []int `json:"perf_tiers,omitempty"`
	// SkillTiers filtre par tier skill ("Bronze"…"Onyx"). Requiert RankedContext.
	SkillTiers []string `json:"skill_tiers,omitempty"`
	// RankedContext : "ranked" | "unranked" | "" (tous). Requiert pour activer SkillTiers.
	RankedContext string `json:"ranked_context,omitempty"`
	// OutcomeFilter : codes résultat acceptés (1=Égalité,2=Victoire,3=Défaite,4=Abandon).
	OutcomeFilter []int `json:"outcome_filter,omitempty"`
	// Filtres Explorer additionnels (date, type d'expérience, playlist, carte, mode, squad, match ID).
	MatchStartDate  *time.Time `json:"match_start_date,omitempty"`
	MatchEndDate    *time.Time `json:"match_end_date,omitempty"`
	ExperienceTypes []string   `json:"experience_types,omitempty"`
	Playlists       []string   `json:"playlists,omitempty"`
	MapNames        []string   `json:"map_names,omitempty"`
	ModeNames       []string   `json:"mode_names,omitempty"`
	SquadScope      string     `json:"squad_scope,omitempty"`
	MatchIDSearch   string     `json:"match_id_search,omitempty"`
}

// ExplorerMatchesRow : une ligne dans la liste des matchs filtrés (Explorer).
type ExplorerMatchesRow struct {
	MatchID             string    `json:"match_id"`
	StartTime           time.Time `json:"start_time"`
	StartTimeLabel      string    `json:"start_time_label"`
	MapUI               *string   `json:"map_ui"`
	ModeUI              *string   `json:"mode_ui"`
	PlaylistLabel       *string   `json:"playlist_label"`
	OutcomeCode         int       `json:"outcome_code"`
	OutcomeLabel        string    `json:"outcome_label"`
	ScoreLabel          string    `json:"score_label"`
	IsWithFriends       bool      `json:"is_with_friends"`
	ExperienceTypeLabel string    `json:"experience_type_label"`
	MatchURL            string    `json:"match_url"`
	// Combat stats
	Kills   int `json:"kills,omitempty"`
	Deaths  int `json:"deaths,omitempty"`
	Assists int `json:"assists,omitempty"`
	// PerfScore : score de performance 0-100 (nil si non calculé).
	PerfScore *int `json:"perf_score,omitempty"`
	// PerfTier : palier de performance 1-5 (0 si score absent).
	PerfTier int `json:"perf_tier,omitempty"`
	// DeltaPerf : déviation du score de perf depuis la médiane (perf_score - 50).
	DeltaPerf *int `json:"delta_perf,omitempty"`
	// SkillTierLabel : label formaté du tier ranked/LUSR (ex. "Diamant IV"), nil si absent.
	SkillTierLabel *string `json:"skill_tier_label,omitempty"`
	// DeltaMMR : variation de MMR/CSR/LUSR pour ce match (nil si non rankée).
	DeltaMMR *float64 `json:"delta_mmr,omitempty"`
}

// ExplorerMatchesSummary : résumé de la requête Explorer.
type ExplorerMatchesSummary struct {
	TotalMatches int `json:"total_matches"`
	// Options disponibles pour les filtres Explorer (valeurs distinctes triées).
	AvailableExperienceTypes []string `json:"available_experience_types,omitempty"`
	AvailablePlaylists       []string `json:"available_playlists,omitempty"`
	AvailableMaps            []string `json:"available_maps,omitempty"`
	AvailableModes           []string `json:"available_modes,omitempty"`
}

// ExplorerMatchesTable : table paginée de l'Explorer.
type ExplorerMatchesTable struct {
	Items      []ExplorerMatchesRow `json:"items"`
	Pagination PaginationMeta       `json:"pagination"`
}

// ExplorerMatchesQueryResponse : réponse de POST matches-query.
type ExplorerMatchesQueryResponse struct {
	Summary ExplorerMatchesSummary `json:"summary"`
	Table   ExplorerMatchesTable   `json:"table"`
}

// ---------------------------------------------------------------------------
// Common matches raw DB row
// ---------------------------------------------------------------------------

// CommonMatchRaw : données brutes de Q19.
type CommonMatchRaw struct {
	MatchID        string
	StartTime      time.Time
	MapUI          string
	ModeUI         string
	Player1TeamID  *int
	Player2TeamID  *int
	Player1Outcome int
	Player1Kills   int
	Player1Deaths  int
	Player1KDA     float64
}
