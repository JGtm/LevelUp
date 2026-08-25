// Package auth — token_health.go : calcul de la santé des tokens (Accès / XSTS /
// Refresh) à partir de l'état PERSISTÉ d'un UserTokens, SANS appel réseau.
// Sert le dashboard admin « Santé des tokens ».
//
// La famille « Accès » s'appelait « MSAL » jusqu'à ADR 0023 Phase 5 : le champ
// mesurait déjà l'expiration de l'access_token Microsoft persisté
// (OAuthExpiresAt), pas un cache MSAL — lequel n'existe plus depuis le retrait
// de MSAL (2026-07-15).
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
	Access  string // ok | expiring | expired | absent
	XSTS    string // ok | expiring | expired | absent
}

// Health calcule la santé des tokens à partir de l'état persisté, SANS appel
// réseau. `now` et `margin` (fenêtre « expire bientôt ») sont injectés pour la
// testabilité.
//
//   - Refresh : capacité à rafraîchir. absent (pas de RT) → reauth
//     (ReauthRequired, RT révoqué) → ok.
//   - Access : access_token Microsoft persisté, qualifié par OAuthExpiresAt.
//   - XSTS : validité du token XSTS (XSTSExpiresAt).
func (u *UserTokens) Health(now time.Time, margin time.Duration) TokenHealth {
	return TokenHealth{
		Refresh: u.refreshStatus(),
		Access:  u.accessStatus(now, margin),
		XSTS:    expiryStatus(u.XSTSToken != "", u.XSTSExpiresAt, now, margin),
	}
}

func (u *UserTokens) refreshStatus() string {
	if u.OAuthRefreshToken == "" {
		return TokenAbsent
	}
	if u.ReauthRequired {
		return TokenReauth
	}
	return TokenOK
}

func (u *UserTokens) accessStatus(now time.Time, margin time.Duration) string {
	if u.AccessToken == "" {
		return TokenAbsent
	}
	// Access token persisté sans horodatage connu → OK (rien ne permet de le
	// juger expiré ; l'échec réel remonte par le statut Refresh).
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
