// Package domain — session.go : types pour la gestion des sessions web.
//
// Sprint 14 :
//
//	POST /session/context → SessionContextResponse
package domain

// HaloIdentity est l'identité Halo liée à une session après Device Code Flow.
// Stockée côté serveur uniquement, jamais exposée au navigateur.
type HaloIdentity struct {
	Gamertag string `json:"gamertag"`
	XUID     string `json:"xuid"`
}

// SessionData est le contenu d'une session serveur.
// Stocké dans data/sessions/<session_id>.json — jamais exposé au navigateur.
type SessionData struct {
	SessionID          string        `json:"session_id"`
	CreatedAt          int64         `json:"created_at"`   // unix timestamp
	LastSeenAt         int64         `json:"last_seen_at"` // unix timestamp
	CurrentPlayerSlug  *string       `json:"current_player_slug,omitempty"`
	CurrentTitleSlug   string        `json:"current_title_slug"` // Sprint 44 : titre courant (default: "halo_infinite")
	Locale             string        `json:"locale"`
	HintsVisible       bool          `json:"hints_visible"`
	AuthReady          bool          `json:"auth_ready"`
	LinkedHaloIdentity *HaloIdentity `json:"linked_halo_identity,omitempty"`
	ActiveSyncJobID    *string       `json:"active_sync_job_id,omitempty"`
	// HaloTokens contient les tokens Halo obtenus après échange (Sprint 18).
	// Jamais exposés au navigateur. TTL ~4h (Spartan token).
	HaloTokens *HaloTokens `json:"halo_tokens,omitempty"`
}

// SessionContextRequest est le body de POST /session/context.
type SessionContextRequest struct {
	PlayerSlug *string `json:"player_slug"`
	TitleSlug  *string `json:"title_slug"` // Sprint 44 : switch titre
	Locale     *string `json:"locale"`
}

// SessionContextResponse est la réponse de POST /session/context.
// Sprint 49 : enrichie pour inclure les titres disponibles (bootstrap complet).
type SessionContextResponse struct {
	CurrentPlayerSlug *string        `json:"current_player_slug,omitempty"`
	CurrentTitleSlug  string         `json:"current_title_slug"`
	AvailableTitles   []TitleSummary `json:"available_titles"`
	Locale            string         `json:"locale"`
	HintsVisible      bool           `json:"hints_visible"`
	AuthReady         bool           `json:"auth_ready"`
}
