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
	oauthRefreshResult string // accessToken retourné par TryOAuthRefresh / TryOAuthRefreshWithRotation
	oauthRefreshErr    error
	rotatedRefresh     string // RT rotaté retourné par TryOAuthRefreshWithRotation (vide = pas de rotation)

	exchangeResult *auth.ExchangeResult // résultat Exchange
	exchangeErr    error

	mu             sync.Mutex
	callLog        []string // log des appels pour vérifier le pipeline (thread-safe)
	lastOauthRTArg string   // dernière valeur de refreshToken reçue par TryOAuthRefresh*
}

func (m *mockTokenProvider) InitDeviceFlow(ctx context.Context) (auth.DeviceFlow, error) {
	return nil, errors.New("not implemented")
}

func (m *mockTokenProvider) TryOAuthRefresh(ctx context.Context, refreshToken string) (string, error) {
	m.mu.Lock()
	m.callLog = append(m.callLog, "TryOAuthRefresh")
	m.mu.Unlock()
	return m.oauthRefreshResult, m.oauthRefreshErr
}

func (m *mockTokenProvider) TryOAuthRefreshWithRotation(ctx context.Context, refreshToken string) (string, string, error) {
	m.mu.Lock()
	m.callLog = append(m.callLog, "TryOAuthRefreshWithRotation")
	m.lastOauthRTArg = refreshToken
	m.mu.Unlock()
	return m.oauthRefreshResult, m.rotatedRefresh, m.oauthRefreshErr
}

func (m *mockTokenProvider) Exchange(ctx context.Context, accessToken string) (*auth.ExchangeResult, error) {
	m.mu.Lock()
	m.callLog = append(m.callLog, "Exchange")
	m.mu.Unlock()
	return m.exchangeResult, m.exchangeErr
}

// TestResolverResolve_RTDead_FiresReauthRequired : credentials présents mais
// refresh KO avec un RT RÉVOQUÉ (invalid_grant, erreur OAuth typée) →
// onReauth(required=true). PR-B slice 3.
func TestResolverResolve_RTDead_FiresReauthRequired(t *testing.T) {
	// Erreur OAuth typée invalid_grant → classe "revoked" (cf. ClassifyAuthError).
	// Un plain errors.New serait classé "transient" et NE déclencherait PAS la
	// bannière (cf. TestResolverResolve_TransientError_NoReauthSignal).
	provider := &mockTokenProvider{oauthRefreshErr: &auth.OAuthExchangeError{ErrorCode: "invalid_grant"}}

	var gotRequired *bool
	var gotGamertag, gotXUID string
	onReauth := func(_ context.Context, gamertag, xuid string, required bool) {
		v := required
		gotRequired, gotGamertag, gotXUID = &v, gamertag, xuid
	}
	resolver := NewResolverWithReauth(provider, time.Hour, nil, onReauth)

	_, err := resolver.Resolve(context.Background(), CredentialSource{Gamertag: "Alice", XUID: "111", RefreshToken: "rt-dead"})
	if err == nil {
		t.Fatal("erreur attendue (RT mort)")
	}
	if gotRequired == nil || !*gotRequired {
		t.Fatal("onReauth(required=true) attendu")
	}
	if gotGamertag != "Alice" || gotXUID != "111" {
		t.Errorf("callback args = %q/%q, want Alice/111", gotGamertag, gotXUID)
	}
}

// TestResolverResolve_TransientError_NoReauthSignal : un échec TRANSITOIRE du
// refresh (réseau / 429 / 5xx Microsoft → erreur non typée, classe "transient")
// ne doit PAS lever la bannière de reconnexion : le refresh_token n'est pas mort,
// un retry ultérieur peut réussir. Régression du faux positif « bandeau reauth
// qui revient souvent » (un simple 429 le déclenchait avant le fix).
func TestResolverResolve_TransientError_NoReauthSignal(t *testing.T) {
	provider := &mockTokenProvider{oauthRefreshErr: errors.New("dial tcp 20.190.1.1:443: i/o timeout")}

	reauthFiredTrue := false
	onReauth := func(_ context.Context, _, _ string, required bool) {
		if required {
			reauthFiredTrue = true
		}
	}
	resolver := NewResolverWithReauth(provider, time.Hour, nil, onReauth)

	_, err := resolver.Resolve(context.Background(), CredentialSource{Gamertag: "Alice", XUID: "111", RefreshToken: "rt-vivant"})
	if err == nil {
		t.Fatal("erreur attendue (échec transitoire)")
	}
	if reauthFiredTrue {
		t.Error("aucun signal reauth(true) attendu sur échec transitoire — RT vivant (faux positif)")
	}
}

// TestResolverResolve_ConfigError_NoReauthSignal : un échec de CONFIG (app Azure
// mal configurée, ex. invalid_client) ne se règle pas par une reconnexion
// utilisateur → pas de bannière (surfacé au dashboard admin à la place).
func TestResolverResolve_ConfigError_NoReauthSignal(t *testing.T) {
	provider := &mockTokenProvider{oauthRefreshErr: &auth.OAuthExchangeError{ErrorCode: "invalid_client"}}

	reauthFiredTrue := false
	onReauth := func(_ context.Context, _, _ string, required bool) {
		if required {
			reauthFiredTrue = true
		}
	}
	resolver := NewResolverWithReauth(provider, time.Hour, nil, onReauth)

	if _, err := resolver.Resolve(context.Background(), CredentialSource{Gamertag: "Alice", XUID: "111", RefreshToken: "rt"}); err == nil {
		t.Fatal("erreur attendue (échec config)")
	}
	if reauthFiredTrue {
		t.Error("aucun signal reauth(true) attendu sur échec config (problème serveur, pas RT mort)")
	}
}

// TestResolverResolve_Success_FiresReauthCleared : un refresh+échange réussi
// déclenche onReauth(required=false) pour effacer un éventuel flag.
func TestResolverResolve_Success_FiresReauthCleared(t *testing.T) {
	provider := &mockTokenProvider{
		oauthRefreshResult: "at",
		exchangeResult:     &auth.ExchangeResult{Tokens: &domain.HaloTokens{SpartanToken: "s"}},
	}
	var lastRequired *bool
	onReauth := func(_ context.Context, _, _ string, required bool) { v := required; lastRequired = &v }
	resolver := NewResolverWithReauth(provider, time.Hour, nil, onReauth)

	if _, err := resolver.Resolve(context.Background(), CredentialSource{Gamertag: "Alice", XUID: "111", RefreshToken: "rt"}); err != nil {
		t.Fatalf("err: %v", err)
	}
	if lastRequired == nil || *lastRequired {
		t.Error("onReauth(required=false) attendu sur succès")
	}
}

// TestResolverResolve_NoCreds_NoReauthSignal : aucun credential (RT vide, pas de
// cache) → pas de signal reauth (compte jamais authentifié, pas une mort).
func TestResolverResolve_NoCreds_NoReauthSignal(t *testing.T) {
	called := false
	onReauth := func(_ context.Context, _, _ string, _ bool) { called = true }
	resolver := NewResolverWithReauth(&mockTokenProvider{}, time.Hour, nil, onReauth)

	_, _ = resolver.Resolve(context.Background(), CredentialSource{Gamertag: "Alice", XUID: "111"})
	if called {
		t.Error("aucun signal reauth attendu sans credentials")
	}
}

// TestResolverResolve_PipelineOAuthRefresh teste le chemin heureux (ADR 0023
// Phase 5, source unique) : refresh_token du store → OAuth refresh → Exchange.
func TestResolverResolve_PipelineOAuthRefresh(t *testing.T) {
	provider := &mockTokenProvider{
		oauthRefreshResult: "access_token_from_oauth",
		exchangeResult: &auth.ExchangeResult{
			Tokens: &domain.HaloTokens{
				SpartanToken:   "spartan_token",
				ClearanceToken: "clearance_token",
			},
		},
	}

	resolver := NewResolver(provider, 1*time.Hour, nil)
	ctx := context.Background()

	src := CredentialSource{
		Gamertag:     "Bob",
		XUID:         "123",
		RefreshToken: "rt_store",
		Source:       credSourceWatcherOAuth,
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

	// Pipeline : TryOAuthRefreshWithRotation → Exchange (plus aucune étape MSAL).
	provider.mu.Lock()
	callLog := provider.callLog
	provider.mu.Unlock()
	expectedCalls := []string{"TryOAuthRefreshWithRotation", "Exchange"}
	if len(callLog) != len(expectedCalls) {
		t.Errorf("expected %d calls, got %d: %v", len(expectedCalls), len(callLog), callLog)
	}
	for i, expected := range expectedCalls {
		if i < len(callLog) && callLog[i] != expected {
			t.Errorf("call %d: expected %s, got %s", i, expected, callLog[i])
		}
	}
}

// TestResolverResolve_CacheTTL teste que le cache TTL fonctionne.
func TestResolverResolve_CacheTTL(t *testing.T) {
	callCount := 0
	provider := &mockTokenProvider{
		oauthRefreshResult: "access_token",
		exchangeResult: &auth.ExchangeResult{
			Tokens: &domain.HaloTokens{
				SpartanToken:   "spartan",
				ClearanceToken: "clearance",
			},
		},
	}

	resolver := NewResolver(provider, 100*time.Millisecond, nil)
	ctx := context.Background()

	src := CredentialSource{
		Gamertag:     "Carl",
		XUID:         "789",
		RefreshToken: "rt",
		Source:       "duckdb_msal",
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
		oauthRefreshResult: "token1",
		exchangeResult: &auth.ExchangeResult{
			Tokens: &domain.HaloTokens{
				SpartanToken: "token1",
			},
		},
	}

	resolver := NewResolver(provider, 10*time.Hour, nil) // long TTL, ne devrait pas expirer
	ctx := context.Background()

	src := CredentialSource{
		Gamertag:     "Dave",
		XUID:         "999",
		RefreshToken: "rt",
		Source:       "duckdb_msal",
	}

	// Première résolution.
	resolved1, err := resolver.Resolve(ctx, src)
	if err != nil {
		t.Fatalf("first Resolve failed: %v", err)
	}

	// Forcer un refresh avec un nouveau token.
	provider.oauthRefreshResult = "token2"
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

	resolver := NewResolver(provider, 1*time.Hour, nil)
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

	resolver := NewResolver(provider, 1*time.Hour, nil)
	ctx := context.Background()

	src := CredentialSource{
		Gamertag:     "NoToken",
		XUID:         "000",
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

// =============================================================================
// Tests de la rotation OAuth + callback onRotated (Phase 1b)
// =============================================================================

// TestResolverRotation_CallbackInvokedWithRotatedRT vérifie que quand Microsoft
// rotate le refresh_token (rotatedRT != "" et != source.RefreshToken), le
// callback onRotated est invoqué avec (gamertag, rotatedRT).
func TestResolverRotation_CallbackInvokedWithRotatedRT(t *testing.T) {
	provider := &mockTokenProvider{
		oauthRefreshResult: "access_token_after_refresh",
		rotatedRefresh:     "rt_v2_rotated_by_microsoft",
		exchangeResult: &auth.ExchangeResult{
			Tokens: &domain.HaloTokens{SpartanToken: "spartan_v2"},
		},
	}

	var capturedGT, capturedRT string
	var callbackCalls int
	onRotated := func(_ context.Context, gt, newRT string) error {
		callbackCalls++
		capturedGT = gt
		capturedRT = newRT
		return nil
	}

	resolver := NewResolver(provider, 1*time.Hour, onRotated)
	src := CredentialSource{
		Gamertag:     "Alice",
		XUID:         "xuid-alice",
		RefreshToken: "rt_v1_original",
		Source:       "duckdb_oauth",
	}
	resolved, err := resolver.Resolve(context.Background(), src)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if resolved.Tokens.SpartanToken != "spartan_v2" {
		t.Errorf("SpartanToken = %q, want spartan_v2", resolved.Tokens.SpartanToken)
	}

	if callbackCalls != 1 {
		t.Errorf("callback invoqué %d fois, attendu 1", callbackCalls)
	}
	if capturedGT != "Alice" {
		t.Errorf("callback gamertag = %q, want Alice", capturedGT)
	}
	if capturedRT != "rt_v2_rotated_by_microsoft" {
		t.Errorf("callback newRT = %q, want rt_v2_rotated_by_microsoft", capturedRT)
	}
}

// TestResolverRotation_SourceUpdatedInMemoryForRefresh vérifie qu'après une
// rotation, le RT mémorisé dans r.sources[gamertag] est bien le rotatedRT
// (et non plus le RT initial). Concrètement : un appel ultérieur à Refresh()
// utilisera le nouveau RT.
func TestResolverRotation_SourceUpdatedInMemoryForRefresh(t *testing.T) {
	provider := &mockTokenProvider{
		oauthRefreshResult: "access_v1",
		rotatedRefresh:     "rt_v2",
		exchangeResult: &auth.ExchangeResult{
			Tokens: &domain.HaloTokens{SpartanToken: "spartan_v1"},
		},
	}

	resolver := NewResolver(provider, 1*time.Hour, nil) // pas de callback : on teste juste l'update interne
	src := CredentialSource{
		Gamertag:     "Bob",
		RefreshToken: "rt_v1",
		Source:       "duckdb_oauth",
	}
	if _, err := resolver.Resolve(context.Background(), src); err != nil {
		t.Fatalf("first Resolve: %v", err)
	}

	// Configurer le mock pour vérifier que Refresh utilise rt_v2 (pas rt_v1).
	provider.mu.Lock()
	provider.lastOauthRTArg = ""
	provider.mu.Unlock()

	if _, err := resolver.Refresh(context.Background(), "Bob"); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	provider.mu.Lock()
	got := provider.lastOauthRTArg
	provider.mu.Unlock()
	if got != "rt_v2" {
		t.Errorf("Refresh a utilisé RT = %q, want rt_v2 (le rotatedRT du 1er Resolve)", got)
	}
}

// TestResolverRotation_CallbackErrorDoesNotFailResolve vérifie que si le
// callback de persistance retourne une erreur (ex: DB en panne), le Resolve
// continue quand même et retourne les tokens. La rotation est "best-effort"
// — perdre la rotation n'invalide pas la session courante.
func TestResolverRotation_CallbackErrorDoesNotFailResolve(t *testing.T) {
	provider := &mockTokenProvider{
		oauthRefreshResult: "access_v1",
		rotatedRefresh:     "rt_v2",
		exchangeResult: &auth.ExchangeResult{
			Tokens: &domain.HaloTokens{SpartanToken: "spartan_v1"},
		},
	}
	onRotated := func(_ context.Context, _, _ string) error {
		return errors.New("DB en panne")
	}

	resolver := NewResolver(provider, 1*time.Hour, onRotated)
	src := CredentialSource{
		Gamertag:     "Carol",
		RefreshToken: "rt_v1",
		Source:       "duckdb_oauth",
	}
	resolved, err := resolver.Resolve(context.Background(), src)
	if err != nil {
		t.Fatalf("Resolve a échoué alors que la rotation devrait être best-effort: %v", err)
	}
	if resolved == nil || resolved.Tokens == nil || resolved.Tokens.SpartanToken != "spartan_v1" {
		t.Errorf("tokens incorrects: %+v", resolved)
	}
}

// TestResolverRotation_NoRotationDoesNotInvokeCallback vérifie que si
// Microsoft ne retourne PAS de rotation (rotatedRT == ""), le callback n'est
// pas invoqué inutilement.
func TestResolverRotation_NoRotationDoesNotInvokeCallback(t *testing.T) {
	provider := &mockTokenProvider{
		oauthRefreshResult: "access_v1",
		rotatedRefresh:     "", // pas de rotation
		exchangeResult: &auth.ExchangeResult{
			Tokens: &domain.HaloTokens{SpartanToken: "spartan_v1"},
		},
	}

	var callbackCalls int
	onRotated := func(_ context.Context, _, _ string) error {
		callbackCalls++
		return nil
	}

	resolver := NewResolver(provider, 1*time.Hour, onRotated)
	src := CredentialSource{
		Gamertag:     "Dave",
		RefreshToken: "rt_v1",
		Source:       "duckdb_oauth",
	}
	if _, err := resolver.Resolve(context.Background(), src); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if callbackCalls != 0 {
		t.Errorf("callback invoqué %d fois alors qu'il n'y a pas eu de rotation", callbackCalls)
	}
}

// =============================================================================
// Tests divers (cache, concurrence)
// =============================================================================

// TestResolverResolve_ConcurrentResolve teste la thread-safety du cache.
func TestResolverResolve_ConcurrentResolve(t *testing.T) {
	provider := &mockTokenProvider{
		oauthRefreshResult: "token",
		exchangeResult: &auth.ExchangeResult{
			Tokens: &domain.HaloTokens{SpartanToken: "token"},
		},
	}

	resolver := NewResolver(provider, 1*time.Hour, nil)
	ctx := context.Background()

	src := CredentialSource{
		Gamertag:     "Concurrent",
		XUID:         "111",
		RefreshToken: "rt",
		Source:       "duckdb_msal",
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
