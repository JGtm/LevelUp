// Package auth — cli_refresh_test.go : tests RefreshHaloTokensViaStoreFirst.
package auth

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

// fakeProvider est un TokenProvider stub configurable pour les tests cli_refresh.
// Distinct de stubProvider (watcher_refresh_test) qui ne supporte pas Exchange ok.
type fakeProvider struct {
	// silentResp/silentErr : valeurs retournées par TrySilentRefresh
	silentResp string
	silentErr  error
	// oauthAccess/oauthRotated/oauthErr : valeurs de TryOAuthRefreshWithRotation
	oauthAccess  string
	oauthRotated string
	oauthErr     error
	// exchangeResult/exchangeErr : valeurs d'Exchange
	exchangeResult *ExchangeResult
	exchangeErr    error

	// Trace des appels pour les assertions
	silentCalls   int
	oauthCalls    int
	exchangeCalls int
	lastOAuthRT   string
}

func (p *fakeProvider) InitDeviceFlow(_ context.Context) (DeviceFlow, error) {
	return nil, errors.New("not implemented")
}

func (p *fakeProvider) TrySilentRefresh(_ context.Context, _ string) (string, error) {
	p.silentCalls++
	return p.silentResp, p.silentErr
}

func (p *fakeProvider) TryOAuthRefresh(_ context.Context, rt string) (string, error) {
	p.lastOAuthRT = rt
	return p.oauthAccess, p.oauthErr
}

func (p *fakeProvider) TryOAuthRefreshWithRotation(_ context.Context, rt string) (string, string, error) {
	p.oauthCalls++
	p.lastOAuthRT = rt
	return p.oauthAccess, p.oauthRotated, p.oauthErr
}

func (p *fakeProvider) Exchange(_ context.Context, _ string) (*ExchangeResult, error) {
	p.exchangeCalls++
	return p.exchangeResult, p.exchangeErr
}

func okExchangeResult() *ExchangeResult {
	return &ExchangeResult{
		Tokens: &domain.HaloTokens{SpartanToken: "spartan-X", ClearanceToken: "clearance-X"},
	}
}

func TestRefreshHaloTokensViaStoreFirst_StoreMSAL_SilentRefresh(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	_ = store.UpdateMSALCache("111", `{"cache":"data"}`)

	prov := &fakeProvider{
		silentResp:     "access-from-msal",
		exchangeResult: okExchangeResult(),
	}

	result, err := RefreshHaloTokensViaStoreFirst(context.Background(), store, prov, "111", "Alice", LegacyAuthInputs{})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if result == nil || result.Tokens == nil {
		t.Fatal("result vide")
	}
	if result.Tokens.SpartanToken != "spartan-X" {
		t.Errorf("Spartan = %q", result.Tokens.SpartanToken)
	}
	if prov.silentCalls != 1 {
		t.Errorf("silent appels = %d, want 1", prov.silentCalls)
	}
	if prov.oauthCalls != 0 {
		t.Errorf("oauth appels = %d, want 0 (MSAL doit suffire)", prov.oauthCalls)
	}
}

func TestRefreshHaloTokensViaStoreFirst_StoreOAuth_RotationPersisted(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	_ = store.UpdateOAuthRefreshToken("111", "rt-v1")

	prov := &fakeProvider{
		oauthAccess:    "access-from-oauth",
		oauthRotated:   "rt-v2-rotated",
		exchangeResult: okExchangeResult(),
	}

	result, _ := RefreshHaloTokensViaStoreFirst(context.Background(), store, prov, "111", "Alice", LegacyAuthInputs{})
	if result == nil {
		t.Fatal("result nil")
	}
	if prov.lastOAuthRT != "rt-v1" {
		t.Errorf("lastOAuthRT = %q, want rt-v1", prov.lastOAuthRT)
	}

	// Vérifier que le RT rotaté a été persisté dans le store.
	user, _ := store.Load("111")
	if user.OAuthRefreshToken != "rt-v2-rotated" {
		t.Errorf("RT après rotation = %q, want rt-v2-rotated", user.OAuthRefreshToken)
	}
}

func TestRefreshHaloTokensViaStoreFirst_StoreOAuth_NoRotationKeepsOriginal(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	_ = store.UpdateOAuthRefreshToken("111", "rt-v1")

	prov := &fakeProvider{
		oauthAccess:    "access",
		oauthRotated:   "", // Microsoft ne rote pas
		exchangeResult: okExchangeResult(),
	}

	_, _ = RefreshHaloTokensViaStoreFirst(context.Background(), store, prov, "111", "Alice", LegacyAuthInputs{})

	user, _ := store.Load("111")
	if user.OAuthRefreshToken != "rt-v1" {
		t.Errorf("RT non rotaté devrait rester rt-v1, got %q", user.OAuthRefreshToken)
	}
}

func TestRefreshHaloTokensViaStoreFirst_StoreOAuth_RotationIdenticalNoWrite(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	_ = store.UpdateOAuthRefreshToken("111", "rt-v1")
	created, _ := store.Load("111")

	prov := &fakeProvider{
		oauthAccess:    "access",
		oauthRotated:   "rt-v1", // identique à l'original
		exchangeResult: okExchangeResult(),
	}

	_, _ = RefreshHaloTokensViaStoreFirst(context.Background(), store, prov, "111", "Alice", LegacyAuthInputs{})

	user, _ := store.Load("111")
	// UpdatedAt ne devrait PAS avoir bougé (rotation === original → no-op).
	if !user.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("UpdatedAt a bougé alors que la rotation est identique")
	}
}

func TestRefreshHaloTokensViaStoreFirst_StoreFailsFallsBackToLegacy(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	// Store n'a rien — fallback legacy attendu.

	prov := &fakeProvider{
		oauthAccess:    "access-from-legacy",
		oauthRotated:   "rt-legacy-rotated",
		exchangeResult: okExchangeResult(),
	}

	result, _ := RefreshHaloTokensViaStoreFirst(context.Background(), store, prov, "111", "Alice", LegacyAuthInputs{
		OAuthRT: "rt-legacy",
		Source:  "test_legacy",
	})
	if result == nil {
		t.Fatal("result nil — devrait fallback sur legacy")
	}

	// Le RT rotaté devrait être persisté dans le store (auto-réparation).
	user, err := store.Load("111")
	if err != nil {
		t.Fatalf("store devrait contenir 111 après rotation legacy : %v", err)
	}
	if user.OAuthRefreshToken != "rt-legacy-rotated" {
		t.Errorf("RT rotaté = %q, want rt-legacy-rotated (auto-migration)", user.OAuthRefreshToken)
	}
}

func TestRefreshHaloTokensViaStoreFirst_LegacyMSALFirst(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))

	prov := &fakeProvider{
		silentResp:     "access-msal",
		exchangeResult: okExchangeResult(),
	}

	result, _ := RefreshHaloTokensViaStoreFirst(context.Background(), store, prov, "111", "Alice", LegacyAuthInputs{
		MSALCache: `{"legacy":"cache"}`,
		Source:    "duckdb",
	})
	if result == nil {
		t.Fatal("result nil")
	}
	if prov.silentCalls != 1 {
		t.Errorf("MSAL legacy devrait être essayé, silent calls = %d", prov.silentCalls)
	}
	if prov.oauthCalls != 0 {
		t.Errorf("OAuth ne devrait pas être appelé si MSAL marche, got %d", prov.oauthCalls)
	}
}

func TestRefreshHaloTokensViaStoreFirst_BothEmptyReturnsNil(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	prov := &fakeProvider{}

	result, err := RefreshHaloTokensViaStoreFirst(context.Background(), store, prov, "111", "Alice", LegacyAuthInputs{})
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if result != nil {
		t.Errorf("result = %v, want nil", result)
	}
	if prov.oauthCalls != 0 || prov.silentCalls != 0 || prov.exchangeCalls != 0 {
		t.Errorf("aucun call attendu, got silent=%d oauth=%d exchange=%d",
			prov.silentCalls, prov.oauthCalls, prov.exchangeCalls)
	}
}

func TestRefreshHaloTokensViaStoreFirst_ProviderNilError(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	_, err := RefreshHaloTokensViaStoreFirst(context.Background(), store, nil, "111", "Alice", LegacyAuthInputs{})
	if err == nil {
		t.Error("provider nil devrait retourner une erreur")
	}
}

func TestRefreshHaloTokensViaStoreFirst_NilStoreFallsBackLegacy(t *testing.T) {
	prov := &fakeProvider{
		oauthAccess:    "access",
		oauthRotated:   "rt-rotated",
		exchangeResult: okExchangeResult(),
	}

	result, _ := RefreshHaloTokensViaStoreFirst(context.Background(), nil, prov, "111", "Alice", LegacyAuthInputs{
		OAuthRT: "rt-legacy",
	})
	if result == nil {
		t.Error("store nil ne devrait pas bloquer le fallback legacy")
	}
	// Pas d'écriture store puisque store nil — pas de panic attendu.
}

func TestRefreshHaloTokensViaStoreFirst_EmptyXUIDSkipsStore(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	_ = store.UpdateOAuthRefreshToken("111", "rt-in-store")

	prov := &fakeProvider{
		oauthAccess:    "access-legacy",
		oauthRotated:   "rt-rotated-legacy",
		exchangeResult: okExchangeResult(),
	}

	// xuid vide → store skipped → fallback direct sur legacy.
	_, _ = RefreshHaloTokensViaStoreFirst(context.Background(), store, prov, "", "Alice", LegacyAuthInputs{
		OAuthRT: "rt-legacy",
	})
	if prov.lastOAuthRT != "rt-legacy" {
		t.Errorf("xuid vide devrait skipper le store, lastOAuthRT = %q", prov.lastOAuthRT)
	}
}

func TestRefreshHaloTokensViaStoreFirst_LegacyRotationPersistedIfStoreAvailable(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	// Store vide.

	prov := &fakeProvider{
		oauthAccess:    "access",
		oauthRotated:   "rt-rotated-from-legacy",
		exchangeResult: okExchangeResult(),
	}

	_, _ = RefreshHaloTokensViaStoreFirst(context.Background(), store, prov, "111", "Alice", LegacyAuthInputs{
		OAuthRT: "rt-legacy-original",
	})

	user, err := store.Load("111")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if user.OAuthRefreshToken != "rt-rotated-from-legacy" {
		t.Errorf("RT persisté = %q, want rt-rotated-from-legacy (auto-migration)", user.OAuthRefreshToken)
	}
}

func TestRefreshHaloTokensViaStoreFirst_ExchangeFailureSkipsToLegacy(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	_ = store.UpdateOAuthRefreshToken("111", "rt-store")

	prov := &fakeProvider{
		oauthAccess: "access",
		oauthErr:    nil,
		// Exchange échoue côté store → on devrait tomber sur legacy.
		exchangeResult: nil,
		exchangeErr:    errors.New("halo exchange down"),
	}

	result, _ := RefreshHaloTokensViaStoreFirst(context.Background(), store, prov, "111", "Alice", LegacyAuthInputs{
		OAuthRT: "rt-legacy",
	})
	// Avec Exchange qui échoue partout, on attend nil. Le test vérifie que le code
	// ne panique pas et que les deux paths sont tentés.
	if result != nil {
		t.Errorf("Exchange échoue partout → result devrait être nil, got %v", result)
	}
	if prov.oauthCalls < 2 {
		t.Errorf("les deux sources devraient être tentées, oauth calls = %d", prov.oauthCalls)
	}
}

func TestHaloTokensFromExchange_Nil(t *testing.T) {
	if got := HaloTokensFromExchange(nil); got != nil {
		t.Errorf("HaloTokensFromExchange(nil) = %v, want nil", got)
	}
}

func TestHaloTokensFromExchange_Valid(t *testing.T) {
	r := &ExchangeResult{Tokens: &domain.HaloTokens{SpartanToken: "x"}}
	got := HaloTokensFromExchange(r)
	if got == nil || got.SpartanToken != "x" {
		t.Errorf("HaloTokensFromExchange = %v", got)
	}
}
