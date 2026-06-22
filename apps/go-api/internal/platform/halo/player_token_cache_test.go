// Package halo — player_token_cache_test.go : tests cache HaloTokens
// (revue 2026-04-29 P3.6).
//
// Cache process-level pur (pas d'appel HTTP). Tests focuses sur le TTL,
// la concurrence, et le miss/hit pattern.
package halo

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

func TestPlayerTokenCache_MissEmpty(t *testing.T) {
	// Reset du store global pour isoler ce test.
	resetPlayerTokenStore()

	got := GetCachedPlayerTokens("unknown-xuid")
	if got != nil {
		t.Errorf("GetCachedPlayerTokens(unknown) = %v, want nil", got)
	}
}

func TestPlayerTokenCache_HitAfterSet(t *testing.T) {
	resetPlayerTokenStore()

	want := &domain.HaloTokens{SpartanToken: "test-spartan", ClearanceToken: "test-clearance"}
	SetCachedPlayerTokens("xuid-42", want)

	got := GetCachedPlayerTokens("xuid-42")
	if got == nil {
		t.Fatal("GetCachedPlayerTokens(xuid-42) = nil après Set")
	}
	if got.SpartanToken != want.SpartanToken {
		t.Errorf("SpartanToken = %q, want %q", got.SpartanToken, want.SpartanToken)
	}
}

func TestPlayerTokenCache_IsolationParXUID(t *testing.T) {
	resetPlayerTokenStore()

	tokensA := &domain.HaloTokens{SpartanToken: "spartan-A"}
	tokensB := &domain.HaloTokens{SpartanToken: "spartan-B"}
	SetCachedPlayerTokens("xuid-A", tokensA)
	SetCachedPlayerTokens("xuid-B", tokensB)

	if got := GetCachedPlayerTokens("xuid-A"); got.SpartanToken != "spartan-A" {
		t.Errorf("xuid-A = %q, want spartan-A", got.SpartanToken)
	}
	if got := GetCachedPlayerTokens("xuid-B"); got.SpartanToken != "spartan-B" {
		t.Errorf("xuid-B = %q, want spartan-B", got.SpartanToken)
	}
}

func TestPlayerTokenCache_ExpiryReturnsNil(t *testing.T) {
	resetPlayerTokenStore()

	// Insérer une entrée déjà expirée (manipule directement le store).
	playerTokenStore.mu.Lock()
	playerTokenStore.store["expired-xuid"] = cachedTokenEntry{
		tokens:    &domain.HaloTokens{SpartanToken: "old"},
		expiresAt: time.Now().Add(-1 * time.Minute),
	}
	playerTokenStore.mu.Unlock()

	got := GetCachedPlayerTokens("expired-xuid")
	if got != nil {
		t.Errorf("GetCachedPlayerTokens(expired) = %v, want nil (TTL expiré)", got)
	}
}

func TestPlayerTokenCache_OverrideUpdatesEntry(t *testing.T) {
	resetPlayerTokenStore()

	SetCachedPlayerTokens("xuid-X", &domain.HaloTokens{SpartanToken: "v1"})
	SetCachedPlayerTokens("xuid-X", &domain.HaloTokens{SpartanToken: "v2"})

	got := GetCachedPlayerTokens("xuid-X")
	if got == nil || got.SpartanToken != "v2" {
		t.Errorf("override : got %v, want v2", got)
	}
}

func TestPlayerTokenCache_ConcurrencyReadWrite(t *testing.T) {
	resetPlayerTokenStore()

	// Test de race condition : 100 writes et 100 reads concurrents sur le
	// même xuid. Pas de panic ni d'invariant cassé attendu.
	var wg sync.WaitGroup
	const N = 100
	for i := 0; i < N; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			SetCachedPlayerTokens("xuid-concurrent", &domain.HaloTokens{
				SpartanToken: "v",
			})
		}(i)
		go func() {
			defer wg.Done()
			_ = GetCachedPlayerTokens("xuid-concurrent")
		}()
	}
	wg.Wait()

	// Le test passe si pas de panic ; vérifier une dernière lecture valide.
	got := GetCachedPlayerTokens("xuid-concurrent")
	if got == nil {
		t.Errorf("après concurrent writes, expected hit, got nil")
	}
}

// resetPlayerTokenStore vide le cache global pour isoler les tests.
// (pas exposé en API publique — utilitaire test-only).
func resetPlayerTokenStore() {
	playerTokenStore.mu.Lock()
	defer playerTokenStore.mu.Unlock()
	playerTokenStore.store = make(map[string]cachedTokenEntry)
}

// --- Garde-fous C1 : cache expiry-aware (jamais servir un token périmé) ---

// Le cache cale son expiry sur l'expiry RÉEL du Spartan (SpartanExpiresAt) et applique
// une marge : un token dont l'expiry tombe dans la marge est traité comme expiré.
func TestPlayerTokenCache_HonorsRealSpartanExpiry(t *testing.T) {
	resetPlayerTokenStore()

	// Expiry réel dans 30 min → servi (au-delà de la marge de 5 min).
	SetCachedPlayerTokens("xuid-future", &domain.HaloTokens{
		SpartanToken:     "valid",
		SpartanExpiresAt: time.Now().Add(30 * time.Minute),
	})
	if got := GetCachedPlayerTokens("xuid-future"); got == nil {
		t.Error("token avec expiry futur (30min) devrait être servi")
	}

	// Expiry réel dans 2 min → DANS la marge (5min) → traité comme expiré → nil.
	SetCachedPlayerTokens("xuid-soon", &domain.HaloTokens{
		SpartanToken:     "about-to-expire",
		SpartanExpiresAt: time.Now().Add(2 * time.Minute),
	})
	if got := GetCachedPlayerTokens("xuid-soon"); got != nil {
		t.Error("token expirant dans 2min (< marge 5min) ne doit JAMAIS être servi")
	}

	// Expiry réel déjà passé → nil.
	SetCachedPlayerTokens("xuid-past", &domain.HaloTokens{
		SpartanToken:     "expired",
		SpartanExpiresAt: time.Now().Add(-1 * time.Minute),
	})
	if got := GetCachedPlayerTokens("xuid-past"); got != nil {
		t.Error("token déjà expiré ne doit jamais être servi")
	}
}

func TestPlayerTokenCache_FallbackTTLWhenExpiryUnknown(t *testing.T) {
	resetPlayerTokenStore()
	// Expiry inconnu (0) → TTL de repli conservateur → servi immédiatement après Set.
	SetCachedPlayerTokens("xuid-legacy", &domain.HaloTokens{SpartanToken: "no-expiry"})
	if got := GetCachedPlayerTokens("xuid-legacy"); got == nil {
		t.Error("expiry inconnu → fallback TTL → token servi juste après Set")
	}
}

func TestTokensFresh(t *testing.T) {
	if TokensFresh(nil) {
		t.Error("nil → pas frais")
	}
	if TokensFresh(&domain.HaloTokens{}) {
		t.Error("SpartanToken vide → pas frais")
	}
	if !TokensFresh(&domain.HaloTokens{SpartanToken: "x"}) {
		t.Error("expiry inconnu (0) → considéré frais (legacy, couvert par le filet 401)")
	}
	if !TokensFresh(&domain.HaloTokens{SpartanToken: "x", SpartanExpiresAt: time.Now().Add(time.Hour)}) {
		t.Error("expiry +1h → frais")
	}
	if TokensFresh(&domain.HaloTokens{SpartanToken: "x", SpartanExpiresAt: time.Now().Add(2 * time.Minute)}) {
		t.Error("expiry +2min (< marge) → pas frais")
	}
}

// --- Garde-fou C5 : re-mint dédupliqué par xuid (singleflight, anti thundering-herd) ---

func TestResolveFreshPlayerTokens_SingleflightOneMint(t *testing.T) {
	resetPlayerTokenStore()
	prev := playerTokenRefresher
	defer SetPlayerTokenRefresher(prev)

	var mintCount int32
	SetPlayerTokenRefresher(func(_ context.Context, xuid string) (*domain.HaloTokens, error) {
		atomic.AddInt32(&mintCount, 1)
		time.Sleep(20 * time.Millisecond) // fenêtre de collision
		return &domain.HaloTokens{SpartanToken: "fresh-" + xuid, SpartanExpiresAt: time.Now().Add(time.Hour)}, nil
	})

	const N = 30
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			_, _ = ResolveFreshPlayerTokens(context.Background(), "xuid-herd")
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt32(&mintCount); got != 1 {
		t.Errorf("attendu 1 seul re-mint (singleflight), got %d", got)
	}
	if got := GetCachedPlayerTokens("xuid-herd"); got == nil || got.SpartanToken != "fresh-xuid-herd" {
		t.Errorf("token frais devrait être en cache après résolution, got %v", got)
	}
}

func TestResolveFreshPlayerTokens_NoRefresher_Errors(t *testing.T) {
	resetPlayerTokenStore()
	prev := playerTokenRefresher
	defer SetPlayerTokenRefresher(prev)
	SetPlayerTokenRefresher(nil)

	if _, err := ResolveFreshPlayerTokens(context.Background(), "xuid-no-refresher"); err == nil {
		t.Error("sans refresher câblé, ResolveFreshPlayerTokens doit renvoyer une erreur (nil-safe)")
	}
}

func TestPlayerTokenCache_InvalidateRemovesEntry(t *testing.T) {
	resetPlayerTokenStore()
	SetCachedPlayerTokens("xuid-rm", &domain.HaloTokens{SpartanToken: "to-remove"})

	if got := GetCachedPlayerTokens("xuid-rm"); got == nil {
		t.Fatal("pré-condition échec : entrée pas en cache")
	}

	InvalidateCachedPlayerTokens("xuid-rm")

	if got := GetCachedPlayerTokens("xuid-rm"); got != nil {
		t.Errorf("après Invalidate : got %v, want nil", got)
	}
}

func TestPlayerTokenCache_InvalidateNoopWhenAbsent(t *testing.T) {
	resetPlayerTokenStore()
	// Pas de panic ni erreur sur xuid absent ou vide.
	InvalidateCachedPlayerTokens("never-existed")
	InvalidateCachedPlayerTokens("")
}

func TestPlayerTokenCache_InvalidateIsolatedToTargetXUID(t *testing.T) {
	resetPlayerTokenStore()
	SetCachedPlayerTokens("xuid-A", &domain.HaloTokens{SpartanToken: "keep-A"})
	SetCachedPlayerTokens("xuid-B", &domain.HaloTokens{SpartanToken: "remove-B"})

	InvalidateCachedPlayerTokens("xuid-B")

	if got := GetCachedPlayerTokens("xuid-A"); got == nil || got.SpartanToken != "keep-A" {
		t.Errorf("xuid-A devrait être préservé, got %v", got)
	}
	if got := GetCachedPlayerTokens("xuid-B"); got != nil {
		t.Errorf("xuid-B devrait être invalidé, got %v", got)
	}
}
