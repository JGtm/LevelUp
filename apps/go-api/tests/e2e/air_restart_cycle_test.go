// Package e2e_test — air_restart_cycle_test.go : test PIVOT du refactor ADR 0023.
//
// Ce test simule le scénario complet qui a déclenché le refactor (bug Madina) :
// Air hot-reload kill + restart le serveur en boucle, et avec l'ancien code,
// le RT de l'env.local était consommé par le Pool au boot et l'env var devenait
// stale, causant invalid_grant sur les requêtes HTTP suivantes.
//
// Le test reproduit ce scénario en isolation (sans Microsoft réel) via un
// TokenProvider stub qui simule strictement la politique de rotation de
// Microsoft : chaque RT utilisé une fois est invalidé, retournant invalid_grant
// s'il est ré-utilisé.
//
// Si ce test passe sur 10 cycles consécutifs sans invalid_grant, le bug Madina
// est résolu architecturalement.
//
// Build tag cgo car le Pool dépend transitivement de DuckDB (via auth/pool →
// config → duckdb).
//
//go:build cgo

package e2e_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/capturecli"
	"levelup/go-api/internal/platform/auth/pool"
)

// rotationProvider est un TokenProvider qui mime strictement la politique de
// rotation de Microsoft Azure AD :
//   - Chaque RT consommé est marqué "used"
//   - Le nouveau RT retourné devient le RT actif (rotation)
//   - Ré-utiliser un RT déjà "used" retourne invalid_grant (= ErrInvalidGrant)
//
// Le compteur invalidGrantCount permet d'asserter à la fin du test qu'aucun
// invalid_grant n'a été émis sur l'ensemble du scénario.
type rotationProvider struct {
	mu              sync.Mutex
	usedRTs         map[string]bool // RTs déjà consommés
	rtCounter       atomic.Int64    // génère "rt-v1", "rt-v2", ...
	exchangeCounter atomic.Int64    // génère spartan tokens uniques

	invalidGrantCount atomic.Int64 // sentinel : doit rester à 0
	refreshCallCount  atomic.Int64
	exchangeCallCount atomic.Int64
}

func newRotationProvider() *rotationProvider {
	return &rotationProvider{usedRTs: make(map[string]bool)}
}

func (p *rotationProvider) InitDeviceFlow(_ context.Context) (auth.DeviceFlow, error) {
	return nil, errors.New("not used in air restart test")
}

func (p *rotationProvider) TryOAuthRefresh(ctx context.Context, refreshToken string) (string, error) {
	at, _, err := p.TryOAuthRefreshWithRotation(ctx, refreshToken)
	return at, err
}

func (p *rotationProvider) TryOAuthRefreshWithRotation(_ context.Context, refreshToken string) (string, string, error) {
	p.refreshCallCount.Add(1)

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.usedRTs[refreshToken] {
		p.invalidGrantCount.Add(1)
		return "", "", fmt.Errorf("invalid_grant: refresh_token already used (%s)", refreshToken[:min(10, len(refreshToken))])
	}
	p.usedRTs[refreshToken] = true

	nextN := p.rtCounter.Add(1)
	rotatedRT := fmt.Sprintf("rt-v%d-rotated", nextN)
	accessToken := fmt.Sprintf("access-v%d", nextN)
	return accessToken, rotatedRT, nil
}

func (p *rotationProvider) Exchange(_ context.Context, _ string) (*auth.ExchangeResult, error) {
	p.exchangeCallCount.Add(1)
	n := p.exchangeCounter.Add(1)
	return &auth.ExchangeResult{
		Tokens: &domain.HaloTokens{
			SpartanToken:   fmt.Sprintf("spartan-v%d", n),
			ClearanceToken: fmt.Sprintf("clearance-v%d", n),
		},
		Gamertag: "Madina97294",
		XUID:     "2533274858283686",
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// rotationCallback retourne un onRotated callback qui écrit dans le store.
func rotationCallback(t *testing.T, store *auth.MultiUserTokenStore, xuid string) pool.TokenRotationCallback {
	t.Helper()
	return func(ctx context.Context, gamertag, newRT string) error {
		return store.UpdateOAuthRefreshToken(xuid, newRT)
	}
}

// ─────────────────────────────────────────────────────────────────────────
// TEST PIVOT — TestAirRestartCycle_FullScenario
// ─────────────────────────────────────────────────────────────────────────

func TestAirRestartCycle_FullScenario(t *testing.T) {
	tempDir := t.TempDir()
	storeDir := filepath.Join(tempDir, "data", "auth", "watcher_tokens")
	const xuid = "2533274858283686"
	const gamertag = "Madina97294"

	provider := newRotationProvider()

	// ─── Step 1 : token-capture (simule l'injection initiale du RT) ────────
	t.Log("Step 1 : capturecli.PersistRefreshToken('rt-initial')")
	store := auth.NewMultiUserTokenStore(storeDir)
	if err := capturecli.PersistRefreshToken(store, xuid, gamertag, "rt-initial", nil); err != nil {
		t.Fatalf("PersistRefreshToken: %v", err)
	}

	user, err := store.Load(xuid)
	if err != nil {
		t.Fatalf("Load post-capture: %v", err)
	}
	if user.OAuthRefreshToken != "rt-initial" {
		t.Errorf("après capture : RT = %q, want rt-initial", user.OAuthRefreshToken)
	}

	// ─── Step 2 : 1er boot — Resolver consomme rt-initial, rotation → rt-v1 ─
	t.Log("Step 2 : 1er boot Pool/Resolver — consomme rt-initial → rotation persistée")

	src := pool.CredentialSource{
		Gamertag:     gamertag,
		XUID:         xuid,
		RefreshToken: user.OAuthRefreshToken,
		Source:       "watcher_oauth",
	}
	resolver1 := pool.NewResolver(provider, 0, rotationCallback(t, store, xuid))

	resolved, err := resolver1.Resolve(context.Background(), src)
	if err != nil {
		t.Fatalf("Resolve 1: %v", err)
	}
	if resolved == nil || resolved.Tokens == nil {
		t.Fatal("Resolve 1 retourne nil")
	}

	user, _ = store.Load(xuid)
	if user.OAuthRefreshToken == "rt-initial" {
		t.Error("store n'a pas été mis à jour avec le RT rotaté")
	}
	if user.OAuthRefreshToken != "rt-v1-rotated" {
		t.Errorf("store RT après Resolve 1 = %q, want rt-v1-rotated", user.OAuthRefreshToken)
	}

	// ─── Step 3 : Stress 10 restarts Air ──────────────────────────────────
	t.Log("Step 3 : 10 cycles de restart Air (rotation propre attendue)")

	for i := 1; i <= 10; i++ {
		// Simule Air kill + restart : nouveau MultiUserTokenStore et nouveau Resolver
		// pointant sur le même répertoire (état partagé via filesystem).
		newStore := auth.NewMultiUserTokenStore(storeDir)
		current, err := newStore.Load(xuid)
		if err != nil {
			t.Fatalf("cycle %d : Load après restart: %v", i, err)
		}
		newSrc := pool.CredentialSource{
			Gamertag:     gamertag,
			XUID:         xuid,
			RefreshToken: current.OAuthRefreshToken,
			Source:       "watcher_oauth",
		}
		newResolver := pool.NewResolver(provider, 0, rotationCallback(t, newStore, xuid))
		_, err = newResolver.Resolve(context.Background(), newSrc)
		if err != nil {
			t.Fatalf("cycle %d : Resolve échoué: %v (refresh_count=%d, invalid_grant_count=%d)",
				i, err, provider.refreshCallCount.Load(), provider.invalidGrantCount.Load())
		}

		updated, _ := newStore.Load(xuid)
		expected := fmt.Sprintf("rt-v%d-rotated", i+1) // i=1 → rt-v2-rotated, etc.
		if updated.OAuthRefreshToken != expected {
			t.Errorf("cycle %d : RT après Resolve = %q, want %q", i, updated.OAuthRefreshToken, expected)
		}
	}

	// ─── Step 4 : Vérifier ZÉRO invalid_grant sur tout le scénario ─────────
	t.Log("Step 4 : vérification finale — aucune erreur invalid_grant")

	if got := provider.invalidGrantCount.Load(); got != 0 {
		t.Errorf("invalidGrantCount = %d, want 0 (le bug Madina serait revenu)", got)
	}
	if got := provider.refreshCallCount.Load(); got != 11 {
		t.Errorf("refreshCallCount = %d, want 11 (1 boot + 10 restarts)", got)
	}

	t.Logf("Bilan : %d refresh calls, %d exchanges, %d invalid_grant — toutes les rotations OK",
		provider.refreshCallCount.Load(), provider.exchangeCallCount.Load(), provider.invalidGrantCount.Load())
}

// ─────────────────────────────────────────────────────────────────────────
// Test régression spécifique Madina — env var stale n'interfère pas
// ─────────────────────────────────────────────────────────────────────────

func TestAirRestartCycle_StaleEnvVarIgnoredAfterFirstRotation(t *testing.T) {
	tempDir := t.TempDir()
	storeDir := filepath.Join(tempDir, "data", "auth", "watcher_tokens")
	const xuid = "2533274858283686"
	const gamertag = "Madina97294"

	// Setup : env var stale (= scénario .env.local jamais updaté)
	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_MADINA97294", "rt-stale-from-env-NEVER_USE")

	provider := newRotationProvider()
	store := auth.NewMultiUserTokenStore(storeDir)

	// Setup : store contient un RT valide (post-rotation initiale)
	if err := capturecli.PersistRefreshToken(store, xuid, gamertag, "rt-fresh-from-capture", nil); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	// Marquer rt-stale comme "déjà utilisé" — si jamais le code essayait de
	// l'utiliser, le provider renverrait invalid_grant.
	provider.usedRTs["rt-stale-from-env-NEVER_USE"] = true

	// Resolve via le RT du STORE (pas l'env)
	src := pool.CredentialSource{
		Gamertag:     gamertag,
		XUID:         xuid,
		RefreshToken: "rt-fresh-from-capture",
		Source:       "watcher_oauth",
	}
	resolver := pool.NewResolver(provider, 0, rotationCallback(t, store, xuid))
	_, err := resolver.Resolve(context.Background(), src)
	if err != nil {
		t.Fatalf("Resolve devrait fonctionner avec le RT du store : %v", err)
	}

	if got := provider.invalidGrantCount.Load(); got != 0 {
		t.Errorf("invalidGrantCount = %d : le code a essayé d'utiliser le RT stale de l'env ?", got)
	}
}

// ─────────────────────────────────────────────────────────────────────────
// Test : rotation idempotente (Microsoft retourne le même RT)
// ─────────────────────────────────────────────────────────────────────────

type identityRotationProvider struct {
	rotationProvider
}

func (p *identityRotationProvider) TryOAuthRefreshWithRotation(_ context.Context, rt string) (string, string, error) {
	p.refreshCallCount.Add(1)
	// Microsoft renvoie le MÊME RT (pas de rotation) — cas où Azure n'applique
	// pas la rotation stricte.
	return "access-v1", rt, nil
}

func (p *identityRotationProvider) TryOAuthRefresh(ctx context.Context, rt string) (string, error) {
	at, _, err := p.TryOAuthRefreshWithRotation(ctx, rt)
	return at, err
}

func TestAirRestartCycle_RotationNoOp_StoreUnchanged(t *testing.T) {
	tempDir := t.TempDir()
	storeDir := filepath.Join(tempDir, "data", "auth", "watcher_tokens")
	const xuid = "2533274858283686"

	store := auth.NewMultiUserTokenStore(storeDir)
	if err := store.UpdateOAuthRefreshToken(xuid, "rt-same"); err != nil {
		t.Fatal(err)
	}
	pre, _ := store.Load(xuid)
	originalUpdated := pre.UpdatedAt

	provider := &identityRotationProvider{}
	provider.usedRTs = make(map[string]bool)
	// On utilise un Exchange réussi via le wrapper rotationProvider
	provider.rotationProvider = *newRotationProvider()
	// Mais notre TryOAuthRefreshWithRotation surchargé ne consomme pas le RT
	// (pas de marquage usedRTs), donc rt-same reste utilisable.

	src := pool.CredentialSource{
		Gamertag:     "Madina97294",
		XUID:         xuid,
		RefreshToken: "rt-same",
		Source:       "watcher_oauth",
	}

	// Callback compatible : ne devrait JAMAIS être appelé puisque RT identique
	var callbackCalled atomic.Bool
	cb := func(ctx context.Context, gamertag, newRT string) error {
		callbackCalled.Store(true)
		return store.UpdateOAuthRefreshToken(xuid, newRT)
	}

	resolver := pool.NewResolver(provider, 0, cb)
	if _, err := resolver.Resolve(context.Background(), src); err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if callbackCalled.Load() {
		t.Error("callback onRotated appelé alors que la rotation est no-op (rt identique)")
	}

	post, _ := store.Load(xuid)
	if !post.UpdatedAt.Equal(originalUpdated) {
		t.Errorf("UpdatedAt a bougé alors que la rotation est identique : pre=%v post=%v",
			originalUpdated, post.UpdatedAt)
	}
}

// ─────────────────────────────────────────────────────────────────────────
// Test : concurrent Resolves SAME gamertag dédupliqués par singleflight
// ─────────────────────────────────────────────────────────────────────────
//
// Régression du test retiré initialement (sans singleflight, 19 invalid_grant
// sur 20 Resolves concurrents). Avec sfGroup, on attend : 1 refresh, 19 dédup.

func TestAirRestartCycle_ConcurrentResolves_SingleflightDedup(t *testing.T) {
	tempDir := t.TempDir()
	storeDir := filepath.Join(tempDir, "data", "auth", "watcher_tokens")
	const xuid = "2533274858283686"
	const gamertag = "Madina97294"

	provider := newRotationProvider()
	store := auth.NewMultiUserTokenStore(storeDir)
	if err := store.UpdateOAuthRefreshToken(xuid, "rt-initial"); err != nil {
		t.Fatal(err)
	}

	src := pool.CredentialSource{
		Gamertag:     gamertag,
		XUID:         xuid,
		RefreshToken: "rt-initial",
		Source:       "watcher_oauth",
	}
	resolver := pool.NewResolver(provider, 1*time.Hour, rotationCallback(t, store, xuid))

	// 20 résolutions concurrentes du MÊME gamertag.
	const N = 20
	var wg sync.WaitGroup
	wg.Add(N)
	errs := make(chan error, N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			if _, err := resolver.Resolve(context.Background(), src); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("Resolve concurrent échoué (singleflight devrait dédup) : %v", err)
	}

	refreshes := provider.refreshCallCount.Load()
	exchanges := provider.exchangeCallCount.Load()
	invalidGrants := provider.invalidGrantCount.Load()

	t.Logf("─── singleflight dedup (N=%d Resolves concurrents même gamertag) ───", N)
	t.Logf("  refresh_calls       : %d (attendu : 1, sf déduplique)", refreshes)
	t.Logf("  exchange_calls      : %d (attendu : 1)", exchanges)
	t.Logf("  invalid_grants      : %d (attendu : 0)", invalidGrants)

	if refreshes != 1 {
		t.Errorf("refresh_calls = %d, want 1 (singleflight devrait dédup en 1 seul OAuth call)", refreshes)
	}
	if exchanges != 1 {
		t.Errorf("exchange_calls = %d, want 1 (singleflight devrait dédup l'exchange aussi)", exchanges)
	}
	if invalidGrants != 0 {
		t.Errorf("invalid_grants = %d (devait être 0 grâce à singleflight)", invalidGrants)
	}
}

// ─────────────────────────────────────────────────────────────────────────
// Test : Resolver.Refresh() force re-acquisition + rotation persistée
// ─────────────────────────────────────────────────────────────────────────

func TestAirRestartCycle_ResolverRefresh_PersistsRotation(t *testing.T) {
	tempDir := t.TempDir()
	storeDir := filepath.Join(tempDir, "data", "auth", "watcher_tokens")
	const xuid = "2533274858283686"
	const gamertag = "Madina97294"

	provider := newRotationProvider()
	store := auth.NewMultiUserTokenStore(storeDir)
	if err := store.UpdateOAuthRefreshToken(xuid, "rt-initial"); err != nil {
		t.Fatal(err)
	}

	src := pool.CredentialSource{
		Gamertag:     gamertag,
		XUID:         xuid,
		RefreshToken: "rt-initial",
		Source:       "watcher_oauth",
	}
	resolver := pool.NewResolver(provider, 1*time.Hour, rotationCallback(t, store, xuid))

	// 1ère Resolve : cache miss → refresh → rotation persistée
	if _, err := resolver.Resolve(context.Background(), src); err != nil {
		t.Fatalf("Resolve 1: %v", err)
	}
	user1, _ := store.Load(xuid)
	rt1 := user1.OAuthRefreshToken
	if rt1 == "rt-initial" {
		t.Fatal("store n'a pas été mis à jour avec rotation 1")
	}

	// Refresh forcé : invalide le cache, ré-exchange
	if _, err := resolver.Refresh(context.Background(), gamertag); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	user2, _ := store.Load(xuid)
	if user2.OAuthRefreshToken == rt1 {
		t.Errorf("après Refresh, store devrait avoir un nouveau RT, got %q (= rt1)", user2.OAuthRefreshToken)
	}

	if got := provider.invalidGrantCount.Load(); got != 0 {
		t.Errorf("invalid_grant pendant Refresh forcé : %d", got)
	}
}

// Imports utilisés conditionnellement pour éviter les warnings unused
// (compile-time check).
var (
	_ = sync.WaitGroup{}
	_ = atomic.Int64{}
)
