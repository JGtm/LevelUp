package wire

// registry_auth_enrich_test.go — garde-fou de l'invariant anti-intermittence du rang
// carrière : enrichWithHaloTokens ne réutilise un token de SESSION que si son expiry est
// CONNUE et fraîche (TokensFreshStrict). Un token de session d'expiry inconnue (session
// pré-A1, SpartanExpiresAt=0) — parfois réellement périmé — est RE-RÉSOLU frais, ce qui
// rend le chemin DÉTERMINISTE (plus de flip-flop « parfois 200 / parfois 401 »).
//
// enrichWithHaloTokens n'utilise aucun champ du registry → testable avec &ServiceRegistry{}.

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/halo"
)

// Token de session d'expiry INCONNUE (zéro) → NON cru frais → re-mint déterministe.
// C'est l'invariant qui tue l'intermittence du rang carrière.
func TestEnrichWithHaloTokens_ZeroExpirySession_ResolvesFresh(t *testing.T) {
	const xuid = "xuid-enrich-zero"
	halo.InvalidateCachedPlayerTokens(xuid)
	defer halo.SetPlayerTokenRefresher(nil)
	halo.SetPlayerTokenRefresher(func(_ context.Context, x string) (*domain.HaloTokens, error) {
		return &domain.HaloTokens{SpartanToken: "fresh-" + x, SpartanExpiresAt: time.Now().Add(time.Hour)}, nil
	})

	reg := &ServiceRegistry{}
	pdb := &duckdb.PlayerDB{XUID: xuid}
	// Token de session SANS expiry (pré-A1) — TokensFresh le croirait frais, TokensFreshStrict non.
	ctx := ctxkeys.WithHaloAuth(context.Background(), &domain.HaloTokens{SpartanToken: "stale-session"}, xuid)

	got := ctxkeys.HaloTokens(reg.enrichWithHaloTokens(ctx, pdb))
	if got == nil || got.SpartanToken != "fresh-"+xuid {
		t.Errorf("token de session expiry=0 doit être REMPLACÉ par un token frais re-minté, got %v", got)
	}
}

// Token de session d'expiry CONNUE et fraîche → réutilisé tel quel (pas de re-mint inutile).
func TestEnrichWithHaloTokens_KnownFreshSession_Reused(t *testing.T) {
	const xuid = "xuid-enrich-fresh"
	halo.InvalidateCachedPlayerTokens(xuid)
	defer halo.SetPlayerTokenRefresher(nil)
	halo.SetPlayerTokenRefresher(func(_ context.Context, x string) (*domain.HaloTokens, error) {
		return &domain.HaloTokens{SpartanToken: "should-not-be-used-" + x, SpartanExpiresAt: time.Now().Add(time.Hour)}, nil
	})

	reg := &ServiceRegistry{}
	pdb := &duckdb.PlayerDB{XUID: xuid}
	sess := &domain.HaloTokens{SpartanToken: "session-fresh", SpartanExpiresAt: time.Now().Add(time.Hour)}
	ctx := ctxkeys.WithHaloAuth(context.Background(), sess, xuid)

	got := ctxkeys.HaloTokens(reg.enrichWithHaloTokens(ctx, pdb))
	if got == nil || got.SpartanToken != "session-fresh" {
		t.Errorf("token de session expiry connue+fraîche doit être RÉUTILISÉ (pas de re-mint), got %v", got)
	}
}

// Token de session expiry CONNUE mais PÉRIMÉE → re-mint déterministe (pas de réutilisation).
func TestEnrichWithHaloTokens_ExpiredKnownSession_ResolvesFresh(t *testing.T) {
	const xuid = "xuid-enrich-expired"
	halo.InvalidateCachedPlayerTokens(xuid)
	defer halo.SetPlayerTokenRefresher(nil)
	halo.SetPlayerTokenRefresher(func(_ context.Context, x string) (*domain.HaloTokens, error) {
		return &domain.HaloTokens{SpartanToken: "fresh-" + x, SpartanExpiresAt: time.Now().Add(time.Hour)}, nil
	})

	reg := &ServiceRegistry{}
	pdb := &duckdb.PlayerDB{XUID: xuid}
	sess := &domain.HaloTokens{SpartanToken: "session-expired", SpartanExpiresAt: time.Now().Add(-time.Hour)}
	ctx := ctxkeys.WithHaloAuth(context.Background(), sess, xuid)

	got := ctxkeys.HaloTokens(reg.enrichWithHaloTokens(ctx, pdb))
	if got == nil || got.SpartanToken != "fresh-"+xuid {
		t.Errorf("token de session périmé doit être REMPLACÉ par un token frais, got %v", got)
	}
}

// Re-mint impossible (SSO-only hors pool) + token de session présent → on GARDE le token de
// session (dégradation, pas de régression : mieux que rien).
func TestEnrichWithHaloTokens_RemintFails_KeepsSessionToken(t *testing.T) {
	const xuid = "xuid-enrich-nopool"
	halo.InvalidateCachedPlayerTokens(xuid)
	defer halo.SetPlayerTokenRefresher(nil)
	halo.SetPlayerTokenRefresher(nil) // aucun refresher câblé → ResolveFreshPlayerTokens échoue

	reg := &ServiceRegistry{}
	pdb := &duckdb.PlayerDB{XUID: xuid}
	sess := &domain.HaloTokens{SpartanToken: "session-only"} // expiry inconnue + re-mint KO
	ctx := ctxkeys.WithHaloAuth(context.Background(), sess, xuid)

	got := ctxkeys.HaloTokens(reg.enrichWithHaloTokens(ctx, pdb))
	if got == nil || got.SpartanToken != "session-only" {
		t.Errorf("re-mint impossible → fallback sur le token de session existant, got %v", got)
	}
}
