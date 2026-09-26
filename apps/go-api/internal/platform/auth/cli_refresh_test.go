// Package auth — cli_refresh_test.go : tests RefreshHaloTokensViaStoreFirst.
//
// ADR 0023 Phase 5 (2026-08-25) : les cas « fallback legacy » (sync_meta, env
// var, MSAL cache) ont disparu avec les sources. Ce qui reste couvert : la
// cascade store, la persistance de rotation, la politique reauth_required, et
// les gardes défensives (provider/store nil, xuid vide).
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
	// oauthAccess/oauthRotated/oauthErr : valeurs de TryOAuthRefreshWithRotation
	oauthAccess  string
	oauthRotated string
	oauthErr     error
	// exchangeResult/exchangeErr : valeurs d'Exchange
	exchangeResult *ExchangeResult
	exchangeErr    error

	// Trace des appels pour les assertions
	oauthCalls    int
	exchangeCalls int
	lastOAuthRT   string
}

func (p *fakeProvider) InitDeviceFlow(_ context.Context) (DeviceFlow, error) {
	return nil, errors.New("not implemented")
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

// TestRefreshHaloTokensViaStoreFirst_RTDead_MarksReauth : credentials présents
// (RT) mais le refresh échoue (RT révoqué) → le store est marqué reauth_required.
func TestRefreshHaloTokensViaStoreFirst_RTDead_MarksReauth(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	_ = store.UpdateOAuthRefreshToken("111", "rt-revoked")

	// Erreur OAuth typée invalid_grant → classe "revoked" (un plain errors.New
	// serait classé "transient" → pas de marquage, cf. test transitoire ci-dessous).
	prov := &fakeProvider{oauthErr: &OAuthExchangeError{ErrorCode: "invalid_grant"}}

	result, _ := RefreshHaloTokensViaStoreFirst(context.Background(), store, prov, "111", "Alice")
	if result != nil {
		t.Fatal("result devrait être nil (refresh KO)")
	}
	if !store.IsReauthRequired("111") {
		t.Error("reauth_required attendu après mort du refresh_token")
	}
}

// TestRefreshHaloTokensViaStoreFirst_TransientError_NoMark : un échec TRANSITOIRE
// du refresh (réseau / 429 → erreur non typée, classe "transient") ne doit PAS
// marquer reauth_required : le RT n'est pas mort. Régression du faux positif
// « bannière de reconnexion qui revient souvent ».
func TestRefreshHaloTokensViaStoreFirst_TransientError_NoMark(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	_ = store.UpdateOAuthRefreshToken("111", "rt-vivant")

	prov := &fakeProvider{oauthErr: errors.New("dial tcp 20.190.1.1:443: i/o timeout")}

	result, _ := RefreshHaloTokensViaStoreFirst(context.Background(), store, prov, "111", "Alice")
	if result != nil {
		t.Fatal("result devrait être nil (refresh KO transitoire)")
	}
	if store.IsReauthRequired("111") {
		t.Error("aucun marquage reauth attendu sur échec transitoire (RT vivant, faux positif)")
	}
}

// TestRefreshHaloTokensViaStoreFirst_Success_ClearsReauth : un refresh réussi
// efface un flag reauth_required préexistant.
func TestRefreshHaloTokensViaStoreFirst_Success_ClearsReauth(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	_ = store.UpdateOAuthRefreshToken("111", "rt-v1")
	_, _ = store.MarkReauthRequired("111", "Alice") // flag préexistant

	prov := &fakeProvider{
		oauthAccess:    "access-ok",
		exchangeResult: okExchangeResult(),
	}

	result, _ := RefreshHaloTokensViaStoreFirst(context.Background(), store, prov, "111", "Alice")
	if result == nil {
		t.Fatal("result nil (refresh devait réussir)")
	}
	if store.IsReauthRequired("111") {
		t.Error("reauth_required aurait dû être effacé après refresh réussi")
	}
}

// TestRefreshHaloTokensViaStoreFirst_NoCreds_NoMark : un xuid sans credentials
// (jamais authentifié) ne doit PAS être marqué reauth_required.
func TestRefreshHaloTokensViaStoreFirst_NoCreds_NoMark(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	// Entrée présente mais sans RT (ex. gamertag seul).
	_ = store.Upsert(&UserTokens{XUID: "111", Gamertag: "Alice"})

	prov := &fakeProvider{}
	_, _ = RefreshHaloTokensViaStoreFirst(context.Background(), store, prov, "111", "Alice")

	if store.IsReauthRequired("111") {
		t.Error("pas de marquage attendu sans credentials (compte jamais authentifié)")
	}
	if prov.oauthCalls != 0 {
		t.Errorf("aucun refresh attendu sans RT, oauth calls = %d", prov.oauthCalls)
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

	result, _ := RefreshHaloTokensViaStoreFirst(context.Background(), store, prov, "111", "Alice")
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

	_, _ = RefreshHaloTokensViaStoreFirst(context.Background(), store, prov, "111", "Alice")

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

	_, _ = RefreshHaloTokensViaStoreFirst(context.Background(), store, prov, "111", "Alice")

	user, _ := store.Load("111")
	// UpdatedAt ne devrait PAS avoir bougé (rotation === original → no-op).
	if !user.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("UpdatedAt a bougé alors que la rotation est identique")
	}
}

// TestRefreshHaloTokensViaStoreFirst_StoreEmpty_NoFallback : ADR 0023 Phase 5 —
// un store vide n'a plus AUCUNE source de repli : aucun appel provider, nil.
func TestRefreshHaloTokensViaStoreFirst_StoreEmpty_NoFallback(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	prov := &fakeProvider{
		oauthAccess:    "access-jamais-servi",
		exchangeResult: okExchangeResult(),
	}

	result, err := RefreshHaloTokensViaStoreFirst(context.Background(), store, prov, "111", "Alice")
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if result != nil {
		t.Errorf("result = %v, want nil (aucune source legacy)", result)
	}
	if prov.oauthCalls != 0 || prov.exchangeCalls != 0 {
		t.Errorf("aucun call attendu, got oauth=%d exchange=%d", prov.oauthCalls, prov.exchangeCalls)
	}
}

func TestRefreshHaloTokensViaStoreFirst_ProviderNilError(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	_, err := RefreshHaloTokensViaStoreFirst(context.Background(), store, nil, "111", "Alice")
	if err == nil {
		t.Error("provider nil devrait retourner une erreur")
	}
}

// TestRefreshHaloTokensViaStoreFirst_NilStoreNoCredentials : sans store, il n'y a
// plus aucune source — retour (nil, nil) sans panic ni appel réseau.
func TestRefreshHaloTokensViaStoreFirst_NilStoreNoCredentials(t *testing.T) {
	prov := &fakeProvider{
		oauthAccess:    "access",
		exchangeResult: okExchangeResult(),
	}

	result, err := RefreshHaloTokensViaStoreFirst(context.Background(), nil, prov, "111", "Alice")
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if result != nil {
		t.Error("store nil → aucune source de credentials, result attendu nil")
	}
	if prov.oauthCalls != 0 {
		t.Errorf("aucun refresh attendu sans store, oauth calls = %d", prov.oauthCalls)
	}
}

// TestRefreshHaloTokensViaStoreFirst_EmptyXUIDNoCredentials : sans xuid, le store
// n'est pas adressable → aucune source.
func TestRefreshHaloTokensViaStoreFirst_EmptyXUIDNoCredentials(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	_ = store.UpdateOAuthRefreshToken("111", "rt-in-store")

	prov := &fakeProvider{
		oauthAccess:    "access",
		exchangeResult: okExchangeResult(),
	}

	result, _ := RefreshHaloTokensViaStoreFirst(context.Background(), store, prov, "", "Alice")
	if result != nil {
		t.Error("xuid vide → aucune entrée adressable, result attendu nil")
	}
	if prov.lastOAuthRT != "" {
		t.Errorf("aucun RT ne doit être tenté sans xuid, lastOAuthRT = %q", prov.lastOAuthRT)
	}
}

// TestRefreshHaloTokensViaStoreFirst_ExchangeFailureReturnsNil : le refresh OAuth
// réussit mais l'Exchange Halo échoue → nil sans erreur (skip non fatal).
func TestRefreshHaloTokensViaStoreFirst_ExchangeFailureReturnsNil(t *testing.T) {
	store := NewMultiUserTokenStore(tempTokenDir(t))
	_ = store.UpdateOAuthRefreshToken("111", "rt-store")

	prov := &fakeProvider{
		oauthAccess:    "access",
		exchangeResult: nil,
		exchangeErr:    errors.New("halo exchange down"),
	}

	result, _ := RefreshHaloTokensViaStoreFirst(context.Background(), store, prov, "111", "Alice")
	if result != nil {
		t.Errorf("Exchange échoue → result devrait être nil, got %v", result)
	}
	if prov.oauthCalls != 1 {
		t.Errorf("une seule source doit être tentée, oauth calls = %d", prov.oauthCalls)
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
