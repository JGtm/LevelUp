// Package auth — token_health.go : calcul de la santé des tokens (MSAL / XSTS /
// Refresh) à partir de l'état PERSISTÉ d'un UserTokens, SANS appel réseau.
// Sert le dashboard admin « Santé des tokens ».
package auth

import "time"

// Statuts de santé d'un token (présentés tels quels par le dashboard admin).
const (
	TokenOK       = "ok"       // présent et valide
	TokenExpiring = "expiring" // valide mais expire sous la marge
	TokenExpired  = "expired"  // expiré
	TokenAbsent   = "absent"   // jamais semé
	TokenReauth   = "reauth"   // réauthentification interactive requise (RT mort)
)

// TokenHealth agrège la santé des 3 tokens suivis par le dashboard admin.
type TokenHealth struct {
	Refresh string // ok | reauth | absent
	MSAL    string // ok | expiring | expired | absent
	XSTS    string // ok | expiring | expired | absent
}

// Health calcule la santé des tokens à partir de l'état persisté, SANS appel
// réseau. `now` et `margin` (fenêtre « expire bientôt ») sont injectés pour la
// testabilité.
//
//   - Refresh : capacité à rafraîchir. absent (ni RT ni cache MSAL) → reauth
//     (ReauthRequired, RT révoqué) → ok.
//   - MSAL : cache MSAL présent ? qualifié par l'expiry de l'access token dérivé
//     (OAuthExpiresAt).
//   - XSTS : validité du token XSTS (XSTSExpiresAt).
func (u *UserTokens) Health(now time.Time, margin time.Duration) TokenHealth {
	return TokenHealth{
		Refresh: u.refreshStatus(),
		MSAL:    u.msalStatus(now, margin),
		XSTS:    expiryStatus(u.XSTSToken != "", u.XSTSExpiresAt, now, margin),
	}
}

func (u *UserTokens) refreshStatus() string {
	if u.OAuthRefreshToken == "" && u.MSALCacheJSON == "" {
		return TokenAbsent
	}
	if u.ReauthRequired {
		return TokenReauth
	}
	return TokenOK
}

func (u *UserTokens) msalStatus(now time.Time, margin time.Duration) string {
	if u.MSALCacheJSON == "" && u.AccessToken == "" {
		return TokenAbsent
	}
	// L'expiry de l'access token MSAL dérivé est suivi via OAuthExpiresAt.
	// Cache présent sans horodatage connu → OK (refresh silencieux possible).
	if u.OAuthExpiresAt.IsZero() {
		return TokenOK
	}
	return expiryStatus(true, u.OAuthExpiresAt, now, margin)
}

// expiryStatus mappe (présence, expiry) vers ok/expiring/expired/absent.
func expiryStatus(present bool, expiresAt, now time.Time, margin time.Duration) string {
	if !present || expiresAt.IsZero() {
		return TokenAbsent
	}
	if !now.Before(expiresAt) {
		return TokenExpired
	}
	if now.Add(margin).After(expiresAt) {
		return TokenExpiring
	}
	return TokenOK
}
