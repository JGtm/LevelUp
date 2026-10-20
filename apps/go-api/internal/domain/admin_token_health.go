// Package domain — admin_token_health.go : payload du dashboard admin
// « Santé des tokens » (Accès / XSTS / Refresh par joueur).
//
// Statuts calculés à partir de l'état persisté (MultiUserTokenStore, ADR 0023),
// SANS déclencher de refresh réseau.
package domain

// PlayerTokenHealth est la santé des tokens auth d'un joueur suivi.
type PlayerTokenHealth struct {
	PlayerSlug string `json:"player_slug"`
	Gamertag   string `json:"gamertag"`
	XUID       string `json:"xuid"`
	// Statuts : "ok" | "expiring" | "expired" | "absent" | "reauth".
	Refresh string `json:"refresh"`
	// Access : fraîcheur de l'access_token Microsoft persisté (ex-champ "msal",
	// renommé ADR 0023 Phase 5 — il ne mesure plus aucun cache MSAL).
	Access string `json:"access"`
	XSTS   string `json:"xsts"`
	// Horodatages d'expiration (RFC3339, vides si inconnus) — pour affichage.
	XSTSExpiresAt  string `json:"xsts_expires_at,omitempty"`
	OAuthExpiresAt string `json:"oauth_expires_at,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
	// LoadError non vide = tokens illisibles pour ce joueur (≠ « sain »).
	LoadError string `json:"load_error,omitempty"`
	// LastAuthError* : dernier échec OAuth permanent (classe "config" |
	// "revoked", message court, horodatage RFC3339). Vides si aucun échec
	// mémorisé. Persistés par le resolver (plan anti-bruit 2026-06-11).
	LastAuthErrorClass string `json:"last_auth_error_class,omitempty"`
	LastAuthError      string `json:"last_auth_error,omitempty"`
	LastAuthErrorAt    string `json:"last_auth_error_at,omitempty"`
	// CredentialSource : source de credentials retenue au dernier scan du pool.
	// Depuis ADR 0023 Phase 5, la seule valeur possible est "watcher_oauth" ;
	// "unknown" si aucun scan depuis le boot.
	CredentialSource string `json:"credential_source,omitempty"`
}

// TokenHealthResponse est la réponse de GET /admin/token-health.
type TokenHealthResponse struct {
	GeneratedAt string              `json:"generated_at"` // RFC3339
	Players     []PlayerTokenHealth `json:"players"`
	// StoreUnavailable = pas de MultiUserTokenStore câblé (mode legacy) → aucun
	// signal token possible.
	StoreUnavailable bool `json:"store_unavailable,omitempty"`
}
