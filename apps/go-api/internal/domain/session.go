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
	// Auth locale : username et rôle de l'utilisateur connecté (mode password).
	Username *string `json:"username,omitempty"`
	Role     *string `json:"role,omitempty"` // "admin" | "user"
	// PR 4 — Authorization Code Flow SSO Xbox : state CSRF généré par LoginRedirect
	// et vérifié par Callback. Stocké en session pour résister à un attaquant qui
	// forcerait un callback avec un code volé sur une victime.
	OAuthState string `json:"oauth_state,omitempty"`
	// PKCE (RFC 7636) : code_verifier généré par LoginRedirect (le code_challenge
	// S256 part dans l'URL /authorize), consommé par Callback pour l'échange du code.
	// Lie le code d'autorisation au client initiateur → un code intercepté est
	// inexploitable sans ce verifier secret.
	OAuthCodeVerifier string `json:"oauth_code_verifier,omitempty"`
	// DeviceFlowAttemptID lie la session à sa tentative Device Code Flow en cours.
	// Posé par StartDeviceFlow : rend la session « significative » (IsMeaningful) donc
	// PERSISTÉE, sinon le SessionID anonyme n'est jamais sauvé et le poll de statut
	// (clé = SessionID) ne retrouve jamais la tentative → login device-flow cassé.
	DeviceFlowAttemptID *string `json:"device_flow_attempt_id,omitempty"`
	// PendingDeviceFlowAttempt : id de la tentative Device Code Flow en cours,
	// posé par StartDeviceFlow. Rend la session « significative » → un cookie
	// stable est posé dès le démarrage du flow, indispensable pour que les
	// requêtes de statut (et le single-flight) retrouvent la MÊME session.
	// Sans ça, chaque requête repart sur une session anonyme distincte →
	// attempt introuvable (404) → le device-flow login ne peut plus aboutir.
	// Posé en parallèle de DeviceFlowAttemptID (les deux pris en compte par
	// IsMeaningful) — cf. handlers/auth.go.
	PendingDeviceFlowAttempt string `json:"pending_device_flow_attempt,omitempty"`
	// PendingInviteCode : code d'invitation "rejoindre un groupe" capté par
	// LoginRedirect (query ?invite=) et consommé par la LinkStrategy après login
	// Xbox SSO réussi (bypass instance lock + AddMember au groupe + Consume).
	// Voyage avec la session à travers l'aller-retour OAuth.
	PendingInviteCode string `json:"pending_invite_code,omitempty"`
}

// IsMeaningful indique si la session porte un état qui mérite d'être persisté sur
// disque. Une session anonyme « vierge » (telle que créée par Store.New : locale
// "fr", hints visibles, auth_ready false, rien d'autre) ne vaut pas un fichier :
// la persister à chaque requête sans cookie (bots, sondes, chargements d'assets)
// sature data/sessions/. On ne persiste qu'à partir du moment où elle porte de
// l'auth, un flow OAuth en cours, une identité/des tokens Halo, un joueur/titre
// courant, un job de sync, ou une préférence explicitement modifiée.
func (s *SessionData) IsMeaningful() bool {
	return s.Username != nil ||
		s.Role != nil ||
		s.OAuthState != "" ||
		s.OAuthCodeVerifier != "" ||
		s.DeviceFlowAttemptID != nil ||
		s.PendingDeviceFlowAttempt != "" ||
		s.PendingInviteCode != "" ||
		s.LinkedHaloIdentity != nil ||
		s.HaloTokens != nil ||
		s.CurrentPlayerSlug != nil ||
		s.ActiveSyncJobID != nil ||
		s.AuthReady ||
		s.CurrentTitleSlug != "" ||
		s.Locale != "fr" ||
		!s.HintsVisible
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
