package pool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
)

// mockTokenProvider implémente auth.TokenProvider pour les tests.
type mockTokenProvider struct {
	silentRefreshResult string // accessToken retourné par TrySilentRefresh
	silentRefreshErr    error

	oauthRefreshResult string // accessToken retourné par TryOAuthRefresh
	oauthRefreshErr    error

	exchangeResult *auth.ExchangeResult // résultat Exchange
	exchangeErr    error

	mu      sync.Mutex
	callLog []string // log des appels pour vérifier le pipeline (thread-safe)
}

func (m *mockTokenProvider) InitDeviceFlow(ctx context.Context) (auth.DeviceFlow, error) {
	return nil, errors.New("not implemented")
}

func (m *mockTokenProvider) TrySilentRefresh(ctx context.Context, cacheJSON string) (string, error) {
	m.mu.Lock()
	m.callLog = append(m.callLog, "TrySilentRefresh")
	m.mu.Unlock()
	return m.silentRefreshResult, m.silentRefreshErr
}

func (m *mockTokenProvider) TryOAuthRefresh(ctx context.Context, refreshToken string) (string, error) {
	m.mu.Lock()
	m.callLog = append(m.callLog, "TryOAuthRefresh")
	m.mu.Unlock()
	return m.oauthRefreshResult, m.oauthRefreshErr
}

func (m *mockTokenProvider) Exchange(ctx context.Context, accessToken string) (*auth.ExchangeResult, error) {
	m.mu.Lock()
	m.callLog = append(m.callLog, "Exchange")
	m.mu.Unlock()
	return m.exchangeResult, m.exchangeErr
}

// TestResolverResolve_PipelineSilentRefresh teste le chemin heureux : MSAL cache → TrySilentRefresh.
func TestResolverResolve_PipelineSilentRefresh(t *testing.T) {
	provider := &mockTokenProvider{
		silentRefreshResult: "access_token_from_msal",
		exchangeResult: &auth.ExchangeResult{
			Tokens: &domain.HaloTokens{
				SpartanToken:   "spartan_token",
				ClearanceToken: "clearance_token",
			},
		},
	}

	resolver := NewResolver(provider, 1*time.Hour)
	ctx := context.Background()

	src := CredentialSource{
		Gamertag:     "Bob",
		XUID:         "123",
		MSALCache:    "cache_json",
		RefreshToken: "",
		Source:       "duckdb_msal",
	}

	resolved, err := resolver.Resolve(ctx, src)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if resolved.Gamertag != "Bob" {
		t.Errorf("expected gamertag Bob, got %s", resolved.Gamertag)
	}
	if resolved.Tokens.SpartanToken != "spartan_token" {
		t.Errorf("expected spartan_token, got %s", resolved.Tokens.SpartanToken)
	}

	// Vérifier le pipeline : Silent → Exchange (pas OAuth car token obtenu par Silent).
	provider.mu.Lock()
	callLog := provider.callLog
	provider.mu.Unlock()
	expectedCalls := []string{"TrySilentRefresh", "Exchange"}
	if len(callLog) != len(expectedCalls) {
		t.Errorf("expected %d calls, got %d", len(expectedCalls), len(callLog))
	}
	for i, expected := range expectedCalls {
		if i < len(callLog) && callLog[i] != expected {
			t.Errorf("call %d: expected %s, got %s", i, expected, callLog[i])
		}
	}
}

// TestResolverResolve_PipelineOAuthFallback teste le fallback : MSAL échoue → TryOAuthRefresh → Exchange.
func TestResolverResolve_PipelineOAuthFallback(t *testing.T) {
	provider := &mockTokenProvider{
		silentRefreshErr:   errors.New("cache expired"),
		oauthRefreshResult: "access_token_from_oauth",
		exchangeResult: &auth.ExchangeResult{
			Tokens: &domain.HaloTokens{
				SpartanToken:   "spartan_oauth",
				ClearanceToken: "clearance_oauth",
			},
		},
	}

	resolver := NewResolver(provider, 1*time.Hour)
	ctx := context.Background()

	src := CredentialSource{
		Gamertag:     "Alice",
		XUID:         "456",
		MSALCache:    "expired_cache",
		RefreshToken: "refresh_token_value",
		Source:       "duckdb_msal+duckdb_oauth",
	}

	resolved, err := resolver.Resolve(ctx, src)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if resolved.Tokens.SpartanToken != "spartan_oauth" {
		t.Errorf("expected spartan_oauth, got %s", resolved.Tokens.SpartanToken)
	}

	// Pipeline : TrySilentRefresh → TryOAuthRefresh → Exchange.
	provider.mu.Lock()
	callLog := provider.callLog
	provider.mu.Unlock()
	expectedCalls := []string{"TrySilentRefresh", "TryOAuthRefresh", "Exchange"}
	if len(callLog) != len(expectedCalls) {
		t.Fatalf("expected %d calls, got %d: %v", len(expectedCalls), len(callLog), callLog)
	}
}

// TestResolverResolve_CacheTTL teste que le cache TTL fonctionne.
func TestResolverResolve_CacheTTL(t *testing.T) {
	callCount := 0
	provider := &mockTokenProvider{
		silentRefreshResult: "access_token",
		exchangeResult: &auth.ExchangeResult{
			Tokens: &domain.HaloTokens{
				SpartanToken:   "spartan",
				ClearanceToken: "clearance",
			},
		},
	}

	resolver := NewResolver(provider, 100*time.Millisecond)
	ctx := context.Background()

	src := CredentialSource{
		Gamertag:  "Carl",
		XUID:      "789",
		MSALCache: "cache",
		Source:    "duckdb_msal",
	}

	// Première résolution → Exchange appelé.
	_, err := resolver.Resolve(ctx, src)
	if err != nil {
		t.Fatalf("first Resolve failed: %v", err)
	}
	provider.mu.Lock()
	callCount = len(provider.callLog)
	provider.mu.Unlock()

	// Deuxième résolution immédiate → cache utilisé, Exchange NOT appelé.
	_, err = resolver.Resolve(ctx, src)
	if err != nil {
		t.Fatalf("second Resolve failed: %v", err)
	}
	provider.mu.Lock()
	count2 := len(provider.callLog)
	provider.mu.Unlock()
	if count2 != callCount {
		t.Errorf("expected cache hit (same call count %d), but got %d calls", callCount, count2)
	}

	// Attendre que le cache expire.
	time.Sleep(150 * time.Millisecond)

	// Troisième résolution → cache expiré, Exchange appelé à nouveau.
	_, err = resolver.Resolve(ctx, src)
	if err != nil {
		t.Fatalf("third Resolve failed: %v", err)
	}
	provider.mu.Lock()
	count3 := len(provider.callLog)
	provider.mu.Unlock()
	if count3 == callCount {
		t.Errorf("expected cache miss after TTL expiration, but call count didn't increase")
	}
}

// TestResolverRefresh teste Refresh() — force un re-échange.
func TestResolverRefresh(t *testing.T) {
	provider := &mockTokenProvider{
		silentRefreshResult: "token1",
		exchangeResult: &auth.ExchangeResult{
			Tokens: &domain.HaloTokens{
				SpartanToken: "token1",
			},
		},
	}

	resolver := NewResolver(provider, 10*time.Hour) // long TTL, ne devrait pas expirer
	ctx := context.Background()

	src := CredentialSource{
		Gamertag:  "Dave",
		XUID:      "999",
		MSALCache: "cache",
		Source:    "duckdb_msal",
	}

	// Première résolution.
	resolved1, err := resolver.Resolve(ctx, src)
	if err != nil {
		t.Fatalf("first Resolve failed: %v", err)
	}

	// Forcer un refresh avec un nouveau token.
	provider.silentRefreshResult = "token2"
	provider.exchangeResult = &auth.ExchangeResult{
		Tokens: &domain.HaloTokens{
			SpartanToken: "token2",
		},
	}
	provider.mu.Lock()
	provider.callLog = []string{} // reset log
	provider.mu.Unlock()

	resolved2, err := resolver.Refresh(ctx, src.Gamertag)
	if err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}

	if resolved2.Tokens.SpartanToken != "token2" {
		t.Errorf("expected token2 after refresh, got %s", resolved2.Tokens.SpartanToken)
	}

	// Vérifier que Refresh a appelé Exchange.
	provider.mu.Lock()
	callLog := provider.callLog
	provider.mu.Unlock()
	if len(callLog) == 0 || callLog[len(callLog)-1] != "Exchange" {
		t.Errorf("expected Refresh to call Exchange, got calls: %v", callLog)
	}

	// Vérifier que l'ancienne résolution était différente.
	if resolved1.Tokens.SpartanToken == resolved2.Tokens.SpartanToken {
		t.Errorf("expected different tokens after refresh")
	}
}

// TestResolverRefresh_UnknownGamertag teste Refresh() avec un gamertag inconnu.
func TestResolverRefresh_UnknownGamertag(t *testing.T) {
	provider := &mockTokenProvider{
		exchangeResult: &auth.ExchangeResult{
			Tokens: &domain.HaloTokens{SpartanToken: "token"},
		},
	}

	resolver := NewResolver(provider, 1*time.Hour)
	ctx := context.Background()

	_, err := resolver.Refresh(ctx, "UnknownPlayer")
	if err == nil {
		t.Fatal("expected error for unknown gamertag, got nil")
	}
	if !errors.Is(err, fmt.Errorf("pool/resolver: aucune source de credentials pour %s (jamais resolveé)", "UnknownPlayer")) {
		// Just check that it's an error mentioning "aucune source"
		if !errors.Is(err, errors.New("aucune source")) && !errors.Is(err, errors.New("pool/resolver")) {
			// Check string content instead
			if msg := err.Error(); !contains(msg, "aucune source") && !contains(msg, "jamais") {
				t.Errorf("expected error about missing credential source, got: %v", err)
			}
		}
	}
}

// contains is a simple helper to check if a string contains a substring.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestResolverResolve_NoTokenSources teste l'erreur quand aucune source de token n'existe.
func TestResolverResolve_NoTokenSources(t *testing.T) {
	provider := &mockTokenProvider{
		exchangeResult: &auth.ExchangeResult{
			Tokens: &domain.HaloTokens{SpartanToken: "unused"},
		},
	}

	resolver := NewResolver(provider, 1*time.Hour)
	ctx := context.Background()

	src := CredentialSource{
		Gamertag:     "NoToken",
		XUID:         "000",
		MSALCache:    "", // vide
		RefreshToken: "", // vide
		Source:       "none",
	}

	_, err := resolver.Resolve(ctx, src)
	if err == nil {
		t.Fatal("expected error for empty credentials, got nil")
	}

	// Vérifier que le message mentionne qu'aucun token n'a pu être obtenu.
	if msg := err.Error(); !contains(msg, "aucun") && !contains(msg, "accessToken") {
		t.Errorf("expected error mentioning missing token, got: %v", err)
	}
}

// TestResolverResolve_ConcurrentResolve teste la thread-safety du cache.
func TestResolverResolve_ConcurrentResolve(t *testing.T) {
	provider := &mockTokenProvider{
		silentRefreshResult: "token",
		exchangeResult: &auth.ExchangeResult{
			Tokens: &domain.HaloTokens{SpartanToken: "token"},
		},
	}

	resolver := NewResolver(provider, 1*time.Hour)
	ctx := context.Background()

	src := CredentialSource{
		Gamertag:  "Concurrent",
		XUID:      "111",
		MSALCache: "cache",
		Source:    "duckdb_msal",
	}

	// Lancer 10 goroutines qui resolvent le même gamertag.
	// Sans protection, on aurait des race conditions sur le cache.
	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := resolver.Resolve(ctx, src)
			done <- err
		}()
	}

	// Attendre et vérifier qu'aucune erreur n'est retournée.
	for i := 0; i < 10; i++ {
		if err := <-done; err != nil {
			t.Errorf("goroutine %d failed: %v", i, err)
		}
	}
}
