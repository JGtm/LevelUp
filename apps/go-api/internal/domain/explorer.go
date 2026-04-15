// Package domain — types pour l'explorer (matchs communs, recherche joueurs).
//
// Port Go de apps/api/app/schemas/explorer.py.
// Routes : POST /players/{slug}/pages/explorer/player-query
//          GET  /directory/gamertags/search (déjà implémenté dans gamertag_repo)
package domain

import "time"

// ---------------------------------------------------------------------------
// Explorer — Player Query
// ---------------------------------------------------------------------------

// ExplorerPlayerQueryRequest : corps de la requête POST player-query.
type ExplorerPlayerQueryRequest struct {
	OtherGamertag string `json:"other_gamertag"`
	Limit         int    `json:"limit,omitempty"`
}

// CommonMatchRow : un match en commun entre 2 joueurs.
type CommonMatchRow struct {
	MatchID        string     `json:"match_id"`
	StartTime      time.Time  `json:"start_time"`
	MapUI          string     `json:"map_ui"`
	ModeUI         string     `json:"mode_ui"`
	WereTeammates  bool       `json:"were_teammates"`
	PlayerOutcome  int        `json:"player_outcome"`
}

// ExplorerPlayerQueryResponse : réponse de la requête player-query.
type ExplorerPlayerQueryResponse struct {
	OtherGamertag string           `json:"other_gamertag"`
	OtherXUID     string           `json:"other_xuid"`
	CommonMatches []CommonMatchRow `json:"common_matches"`
	Total         int              `json:"total"`
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
