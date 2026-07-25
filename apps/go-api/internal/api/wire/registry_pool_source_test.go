package wire

import (
	"context"
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/halo"
)

// stubPoolSource : pooledTokenSource contrôlé pour tester l'ordre de résolution
// sans dépendre du pool concret.
type stubPoolSource struct {
	tokens   *domain.HaloTokens
	gamertag string
	err      error
	calls    int
}

func (s *stubPoolSource) ResolveAny(_ context.Context) (*domain.HaloTokens, string, error) {
	s.calls++
	return s.tokens, s.gamertag, s.err
}

// freshTokens fabrique des HaloTokens frais STRICTS (expiry connue et future).
func freshTokens(spartan string) *domain.HaloTokens {
	return &domain.HaloTokens{
		SpartanToken:     spartan,
		SpartanExpiresAt: time.Now().Add(2 * time.Hour),
	}
}

// resetRefresher installe un refresher global déterministe (toujours en erreur =
// "profil mort") et le remet à nil au nettoyage. Les tests utilisent des xuid
// uniques pour éviter le partage de cache/singleflight.
func resetRefresher(t *testing.T) {
	t.Helper()
	halo.SetPlayerTokenRefresher(func(_ context.Context, xuid string) (*domain.HaloTokens, error) {
		return nil, errors.New("test: profil sans token (RT mort)")
	})
	t.Cleanup(func() { halo.SetPlayerTokenRefresher(nil) })
}

// TestEnrichPublicRead_SessionFresh : un token de session frais strict est
// réutilisé tel quel — ni profil ni pool ne sont sollicités.
func TestEnrichPublicRead_SessionFresh(t *testing.T) {
	resetRefresher(t)
	stub := &stubPoolSource{tokens: freshTokens("sp-pool"), gamertag: "PoolGuy"}
	reg := &ServiceRegistry{publicReadTokenSrc: stub}
	pdb := &duckdb.PlayerDB{XUID: "profile-session"}

	ctx := ctxkeys.WithHaloAuth(context.Background(), freshTokens("sp-session"), "session-owner")
	got := reg.enrichWithHaloTokensPublicRead(ctx, pdb)

	if tok := ctxkeys.HaloTokens(got); tok == nil || tok.SpartanToken != "sp-session" {
		t.Fatalf("attendu token de session conservé, obtenu %+v", tok)
	}
	if stub.calls != 0 {
		t.Fatalf("le pool ne doit pas être sollicité quand la session est fraîche (calls=%d)", stub.calls)
	}
}

// TestEnrichPublicRead_ProfileAlive : sans session fraîche, on résout le token du
// PROFIL sélectionné (cache expiry-aware pré-semé) — le pool n'est pas sollicité.
func TestEnrichPublicRead_ProfileAlive(t *testing.T) {
	resetRefresher(t)
	const xuid = "profile-alive"
	halo.SetCachedPlayerTokens(xuid, freshTokens("sp-profile"))
	t.Cleanup(func() { halo.InvalidateCachedPlayerTokens(xuid) })

	stub := &stubPoolSource{tokens: freshTokens("sp-pool"), gamertag: "PoolGuy"}
	reg := &ServiceRegistry{publicReadTokenSrc: stub}

	got := reg.enrichWithHaloTokensPublicRead(context.Background(), &duckdb.PlayerDB{XUID: xuid})

	if tok := ctxkeys.HaloTokens(got); tok == nil || tok.SpartanToken != "sp-profile" {
		t.Fatalf("attendu token du profil, obtenu %+v", tok)
	}
	if stub.calls != 0 {
		t.Fatalf("le pool ne doit pas être sollicité quand le profil est vivant (calls=%d)", stub.calls)
	}
}

// TestEnrichPublicRead_ProfileDeadPoolHealthy : cœur du fix — RT du profil mort,
// on bascule sur un token SAIN du pool. Le contexte porte les tokens du pool et le
// SUJET reste pdb.XUID.
func TestEnrichPublicRead_ProfileDeadPoolHealthy(t *testing.T) {
	resetRefresher(t)
	const xuid = "profile-dead-pool-ok"
	halo.InvalidateCachedPlayerTokens(xuid)

	stub := &stubPoolSource{tokens: freshTokens("sp-pool"), gamertag: "PoolGuy"}
	reg := &ServiceRegistry{publicReadTokenSrc: stub}

	got := reg.enrichWithHaloTokensPublicRead(context.Background(), &duckdb.PlayerDB{XUID: xuid})

	if tok := ctxkeys.HaloTokens(got); tok == nil || tok.SpartanToken != "sp-pool" {
		t.Fatalf("attendu token du pool, obtenu %+v", tok)
	}
	if stub.calls != 1 {
		t.Fatalf("le pool doit être sollicité exactement une fois (calls=%d)", stub.calls)
	}
	if subj := ctxkeys.HaloXUID(got); subj != xuid {
		t.Fatalf("le sujet du contexte doit rester pdb.XUID=%q, obtenu %q", xuid, subj)
	}
}

// TestEnrichPublicRead_AllDead : profil mort ET pool indisponible → aucune bascule,
// le contexte ne porte aucun token (dégradation gracieuse, pas de panic).
func TestEnrichPublicRead_AllDead(t *testing.T) {
	resetRefresher(t)
	const xuid = "profile-all-dead"
	halo.InvalidateCachedPlayerTokens(xuid)

	stub := &stubPoolSource{err: errors.New("pool: aucun slot sain disponible")}
	reg := &ServiceRegistry{publicReadTokenSrc: stub}

	got := reg.enrichWithHaloTokensPublicRead(context.Background(), &duckdb.PlayerDB{XUID: xuid})

	if tok := ctxkeys.HaloTokens(got); tok != nil {
		t.Fatalf("attendu aucun token, obtenu %+v", tok)
	}
	if stub.calls != 1 {
		t.Fatalf("le pool doit être sollicité une fois avant l'abandon (calls=%d)", stub.calls)
	}
}

// TestEnrichPublicRead_NoPoolSource : parité legacy — sans source pool câblée et
// profil mort, comportement identique à enrichWithHaloTokens (contexte inchangé).
func TestEnrichPublicRead_NoPoolSource(t *testing.T) {
	resetRefresher(t)
	const xuid = "profile-no-pool"
	halo.InvalidateCachedPlayerTokens(xuid)

	reg := &ServiceRegistry{} // publicReadTokenSrc nil
	got := reg.enrichWithHaloTokensPublicRead(context.Background(), &duckdb.PlayerDB{XUID: xuid})

	if tok := ctxkeys.HaloTokens(got); tok != nil {
		t.Fatalf("attendu aucun token (parité legacy), obtenu %+v", tok)
	}
}

// publicReadCallRE : APPEL de enrichWithHaloTokensPublicRead sur un receiver
// QUELCONQUE (`<x>.enrichWithHaloTokensPublicRead(`). Le point de tête exclut la
// DÉCLARATION (`func (r *ServiceRegistry) enrichWithHaloTokensPublicRead(`), qui
// n'est jamais précédée d'un point.
//
// Durci à la contre-revue V7.2 : la version précédente cherchait le littéral
// « r.enrichWithHaloTokensPublicRead( ». Renommer le receiver (`reg`, `s`, …) —
// un refactor cosmétique que rien n'interdit — désarmait TOUT le garde-rail en
// silence : plus aucun caller détecté, l'invariant 1 aurait échoué bruyamment,
// mais un caller ajouté avec un autre receiver serait passé inaperçu.
var publicReadCallRE = regexp.MustCompile(`\.\s*enrichWithHaloTokensPublicRead\s*\(`)

// profileCallRE : appel du chemin PROFIL sur (ctx, pdb), receiver quelconque.
// `enrichWithHaloTokens\(` ne matche pas la variante PublicRead (la parenthèse
// suit immédiatement « Tokens »).
var profileCallRE = regexp.MustCompile(`\.\s*enrichWithHaloTokens\(\s*ctx\s*,\s*pdb\s*\)`)

// TestPublicReadPerimeter_Guard (A1.5) : garde-rail de PÉRIMÈTRE. Le fallback
// pool-first (enrichWithHaloTokensPublicRead) ne doit être branché QUE sur le
// player-query Explorer. Les chemins ownership-scoped (SeasonPassCtxWithAuth) et le
// Home restent sur enrichWithHaloTokens (identité du profil) — le pool ne doit
// JAMAIS servir une donnée privée. On lit les sources du package et on vérifie
// l'invariant de câblage (grep test).
func TestPublicReadPerimeter_Guard(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du répertoire package: %v", err)
	}

	callers := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, rerr := os.ReadFile(name)
		if rerr != nil {
			t.Fatalf("lecture %s: %v", name, rerr)
		}
		if publicReadCallRE.Match(src) {
			callers[name] = true
		}
	}

	// Invariant 1 : un seul caller, registry_pages_explorer.go.
	if len(callers) != 1 || !callers["registry_pages_explorer.go"] {
		t.Fatalf("enrichWithHaloTokensPublicRead ne doit être appelé QUE par registry_pages_explorer.go, callers=%v", callers)
	}

	// Invariant 2 : SeasonPass + Home (registry_auth.go) restent sur le chemin
	// profil (enrichWithHaloTokens plain, distinct du variant PublicRead).
	authSrc, rerr := os.ReadFile("registry_auth.go")
	if rerr != nil {
		t.Fatalf("lecture registry_auth.go: %v", rerr)
	}
	if !profileCallRE.Match(authSrc) {
		t.Fatal("registry_auth.go (Home/SeasonPass) doit conserver le chemin profil enrichWithHaloTokens(ctx, pdb)")
	}
}
