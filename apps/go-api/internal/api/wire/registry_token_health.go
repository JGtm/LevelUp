// Package api — registry_token_health.go : runner du dashboard admin
// « Santé des tokens ». Énumère les joueurs suivis (db_profiles.json) et lit
// leur état token persisté (MultiUserTokenStore, ADR 0023) en LECTURE SEULE —
// ne déclenche aucun refresh réseau. Best-effort par joueur.
package wire

import (
	"context"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/pool"
)

// tokenHealthMargin : fenêtre « expire bientôt » pour les statuts token.
const tokenHealthMargin = 5 * time.Minute

// TokenHealth retourne la santé des tokens auth (Accès / XSTS / Refresh) par
// joueur suivi. Lecture seule de l'état persisté, sans refresh réseau.
func (r *ServiceRegistry) TokenHealth(_ context.Context, titleSlug string) (domain.TokenHealthResponse, error) {
	resp := domain.TokenHealthResponse{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Players:     []domain.PlayerTokenHealth{},
	}
	if r.authStore == nil {
		resp.StoreUnavailable = true
		return resp, nil
	}
	players, err := r.cfg.LoadPlayers(titleSlug)
	if err != nil {
		return resp, err
	}
	now := time.Now()
	for _, p := range players {
		ph := domain.PlayerTokenHealth{
			PlayerSlug:       p.PlayerSlug,
			Gamertag:         p.Gamertag,
			XUID:             p.XUID,
			Refresh:          auth.TokenAbsent,
			Access:           auth.TokenAbsent,
			XSTS:             auth.TokenAbsent,
			CredentialSource: credentialSourceFor(titleSlug, p.Gamertag),
		}
		u, lerr := r.authStore.Load(p.XUID)
		if lerr != nil || u == nil {
			// Pas de tokens semés pour ce joueur (ErrUserTokensNotFound) →
			// statuts "absent", pas une erreur du dashboard.
			resp.Players = append(resp.Players, ph)
			continue
		}
		h := u.Health(now, tokenHealthMargin)
		ph.Refresh, ph.Access, ph.XSTS = h.Refresh, h.Access, h.XSTS
		if !u.XSTSExpiresAt.IsZero() {
			ph.XSTSExpiresAt = u.XSTSExpiresAt.UTC().Format(time.RFC3339)
		}
		if !u.OAuthExpiresAt.IsZero() {
			ph.OAuthExpiresAt = u.OAuthExpiresAt.UTC().Format(time.RFC3339)
		}
		if !u.UpdatedAt.IsZero() {
			ph.UpdatedAt = u.UpdatedAt.UTC().Format(time.RFC3339)
		}
		ph.LastAuthErrorClass = u.LastAuthErrorClass
		ph.LastAuthError = u.LastAuthError
		if !u.LastAuthErrorAt.IsZero() {
			ph.LastAuthErrorAt = u.LastAuthErrorAt.UTC().Format(time.RFC3339)
		}
		resp.Players = append(resp.Players, ph)
	}
	return resp, nil
}

// credentialSourceFor retourne la source de credentials du dernier scan du
// pool pour un joueur, ou "unknown" si aucun scan n'a eu lieu depuis le boot.
func credentialSourceFor(titleSlug, gamertag string) string {
	if s, ok := pool.LastScanSource(titleSlug, gamertag); ok {
		return s.Source
	}
	return "unknown"
}
