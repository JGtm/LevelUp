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

// forcePageIdentityXUID — garde-fou du SCOPING PAR JOUEUR de l'identité Spartan
// (régression 2026-07-16 : toutes les pages home partageaient la même identité —
// celle du compte connecté). Depuis que SISU complète l'identité de session, le
// ctx porte le xuid du compte APPELANT ; la home doit néanmoins servir l'identité
// du joueur de la PAGE. Ces tests verrouillent ce contrat.

// Compte connecté A consultant la page du joueur B (admin/groupe) : l'identité
// résolue doit être celle de B (pdb), jamais A. C'est le test qui aurait attrapé
// la régression.
func TestForcePageIdentityXUID_ThirdPartyPage_UsesPageXUID(t *testing.T) {
	const connectedXUID = "xuid-compte-A"
	const pageXUID = "xuid-joueur-B"
	// Session du compte connecté A (tokens frais + xuid A), comme après login SISO.
	ctx := ctxkeys.WithHaloAuth(context.Background(),
		&domain.HaloTokens{SpartanToken: "session-A", SpartanExpiresAt: time.Now().Add(time.Hour)},
		connectedXUID)

	got := forcePageIdentityXUID(ctx, pageXUID)
	if x := ctxkeys.HaloXUID(got); x != pageXUID {
		t.Errorf("identité de la home = joueur de la PAGE attendu %q, got %q (fuite du compte connecté)", pageXUID, x)
	}
	// Les tokens de session restent disponibles (URL careerranks/customization = xuid(page)).
	if toks := ctxkeys.HaloTokens(got); toks == nil || toks.SpartanToken != "session-A" {
		t.Errorf("les tokens de session doivent être préservés, got %v", toks)
	}
}

// Démo / crawler : aucun xuid en session → on impose celui de la page (sinon
// lectures xuid-filtrées ciblant "" → bannière/emblème absents).
func TestForcePageIdentityXUID_NoSessionXUID_UsesPageXUID(t *testing.T) {
	const pageXUID = "xuid-joueur-demo"
	got := forcePageIdentityXUID(context.Background(), pageXUID)
	if x := ctxkeys.HaloXUID(got); x != pageXUID {
		t.Errorf("sans xuid de session, la page doit imposer %q, got %q", pageXUID, x)
	}
}

// Propriétaire consultant SA propre page : xuid inchangé (pas de re-écriture inutile).
func TestForcePageIdentityXUID_OwnPage_Unchanged(t *testing.T) {
	const xuid = "xuid-proprietaire"
	ctx := ctxkeys.WithHaloAuth(context.Background(),
		&domain.HaloTokens{SpartanToken: "session-own", SpartanExpiresAt: time.Now().Add(time.Hour)}, xuid)
	got := forcePageIdentityXUID(ctx, xuid)
	if x := ctxkeys.HaloXUID(got); x != xuid {
		t.Errorf("page du propriétaire : xuid attendu %q, got %q", xuid, x)
	}
}

// A1 (revue 2026-07) — SeasonPassCtxWithAuth doit forcer l'identité de PAGE avant tout
// fetch BP/défis. Ces endpoints economy (players/xuid(<sujet>)/decks|rewardtracks) sont
// OWNERSHIP-SCOPED : le SUJET vient de ctxkeys.HaloXUID et les snapshots sont persistés
// sous le xuid de la PAGE (sink). Sans forçage, un compte connecté A consultant la page
// de B fetchait les défis de A puis les persistait sous B (pollution, 4e occurrence du
// bug PR #63). Ces tests verrouillent la composition enrich+force de SeasonPassCtxWithAuth.

// Compte connecté A consultant la page du joueur B : le sujet du fetch BP/défis doit
// être B (page), jamais A — sinon les données de A polluent la DB de B.
func TestSeasonPassEnrich_ThirdPartyViewer_SubjectIsPage(t *testing.T) {
	const connectedXUID = "xuid-compte-A-sp"
	const pageXUID = "xuid-joueur-B-sp"
	reg := &ServiceRegistry{}
	pdb := &duckdb.PlayerDB{XUID: pageXUID}
	// Session du compte connecté A (tokens frais + xuid A), comme après login SISO.
	ctx := ctxkeys.WithHaloAuth(context.Background(),
		&domain.HaloTokens{SpartanToken: "session-A", SpartanExpiresAt: time.Now().Add(time.Hour)},
		connectedXUID)

	// Composition IDENTIQUE à SeasonPassCtxWithAuth (registry_auth.go).
	enriched := forcePageIdentityXUID(reg.enrichWithHaloTokens(ctx, pdb), pdb.XUID)

	if x := ctxkeys.HaloXUID(enriched); x != pageXUID {
		t.Errorf("sujet BP/défis attendu = page %q, got %q (fuite compte connecté → pollution DB de B)", pageXUID, x)
	}
	// Le token de session A reste porté : fetch xuid(B) avec token A → 403 ownership →
	// fallback cache DB de B (jamais de persist des défis de A sous B).
	if toks := ctxkeys.HaloTokens(enriched); toks == nil || toks.SpartanToken != "session-A" {
		t.Errorf("tokens de session préservés attendus, got %v", toks)
	}
}

// Propriétaire consultant SA propre page : sujet inchangé → fetch/persist BP/défis sous
// son propre xuid (comportement nominal préservé).
func TestSeasonPassEnrich_OwnPage_SubjectUnchanged(t *testing.T) {
	const xuid = "xuid-proprietaire-sp"
	halo.InvalidateCachedPlayerTokens(xuid)
	reg := &ServiceRegistry{}
	pdb := &duckdb.PlayerDB{XUID: xuid}
	ctx := ctxkeys.WithHaloAuth(context.Background(),
		&domain.HaloTokens{SpartanToken: "session-own-sp", SpartanExpiresAt: time.Now().Add(time.Hour)}, xuid)

	enriched := forcePageIdentityXUID(reg.enrichWithHaloTokens(ctx, pdb), pdb.XUID)
	if x := ctxkeys.HaloXUID(enriched); x != xuid {
		t.Errorf("page du propriétaire : sujet BP/défis attendu %q, got %q", xuid, x)
	}
}
