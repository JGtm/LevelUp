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

// ExplorerPlayerQueryRequest : corps de la requête POST player-query.
type ExplorerPlayerQueryRequest struct {
	TargetGamertag string `json:"target_gamertag"`
	Limit          int    `json:"limit,omitempty"`
}

// CommonMatchRow : un match en commun entre 2 joueurs.
type CommonMatchRow struct {
	MatchID       string    `json:"match_id"`
	StartTime     time.Time `json:"start_time"`
	MapUI         string    `json:"map_ui"`
	ModeUI        string    `json:"mode_ui"`
	WereTeammates bool      `json:"were_teammates"`
	PlayerOutcome int       `json:"player_outcome"`
}

// ExplorerPlayerQueryResponse : réponse de la requête player-query.
type ExplorerPlayerQueryResponse struct {
	TargetGamertag string           `json:"target_gamertag"`
	TargetXUID     string           `json:"target_xuid"`
	CommonMatches  []CommonMatchRow `json:"common_matches"`
	Total          int              `json:"total"`
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
}

// ExplorerMatchesRow : une ligne dans la liste des matchs filtrés (Explorer).
type ExplorerMatchesRow struct {
	MatchID      string    `json:"match_id"`
	StartTime    time.Time `json:"start_time"`
	MapUI        *string   `json:"map_ui"`
	ModeUI       *string   `json:"mode_ui"`
	OutcomeCode  int       `json:"outcome_code"`
	OutcomeLabel string    `json:"outcome_label"`
	Kills        int       `json:"kills"`
	Deaths       int       `json:"deaths"`
	KDA          *float64  `json:"kda"`
	MatchURL     string    `json:"match_url"`
}

// ExplorerMatchesQueryResponse : réponse de POST matches-query.
type ExplorerMatchesQueryResponse struct {
	Matches    []ExplorerMatchesRow `json:"matches"`
	Pagination PaginationMeta       `json:"pagination"`
	Total      int                  `json:"total"`
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
}
