// Package auth — sisu_provider_test.go : tests unitaires pour sisu_provider.go.
package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ─── sisuDeviceFlow getters ────────────────────────────────────────────────────

func TestSISUDeviceFlow_Getters(t *testing.T) {
	f := &sisuDeviceFlow{
		verificationURL: "https://microsoft.com/link",
		userCode:        "ABC123",
		deviceCode:      "device-opaque",
		interval:        5,
		appID:           "000000004c20a908",
		expiresIn:       900,
	}

	if got := f.GetUserCode(); got != "ABC123" {
		t.Errorf("GetUserCode = %q, want ABC123", got)
	}
	if got := f.GetVerificationURL(); got != "https://microsoft.com/link" {
		t.Errorf("GetVerificationURL = %q, want https://microsoft.com/link", got)
	}
	if got := f.GetExpiresIn(); got != 900 {
		t.Errorf("GetExpiresIn = %d, want 900", got)
	}
	if got := f.GetFlowType(); got != "sisu" {
		t.Errorf("GetFlowType = %q, want sisu", got)
	}
	if msg := f.GetMessage(); msg == "" {
		t.Error("GetMessage retourne une chaîne vide")
	}
}

// TestSISUDeviceFlow_VerificationURLFallback vérifie que si MsaOauthRedirect est vide
// la verificationURL est celle du Device Code Xbox.
func TestSISUDeviceFlow_VerificationURLFallback(t *testing.T) {
	f := &sisuDeviceFlow{verificationURL: "https://xbox.com/activate"}
	if got := f.GetVerificationURL(); got != "https://xbox.com/activate" {
		t.Errorf("fallback URL = %q, want https://xbox.com/activate", got)
	}
}

// ─── NewSISUProviderWithIDs ───────────────────────────────────────────────────

func TestNewSISUProviderWithIDs(t *testing.T) {
	p := NewSISUProviderWithIDs("custom-app", "custom-title")
	if p.appID != "custom-app" {
		t.Errorf("appID = %q, want custom-app", p.appID)
	}
	if p.titleID != "custom-title" {
		t.Errorf("titleID = %q, want custom-title", p.titleID)
	}
}

func TestNewSISUProvider_DefaultIDs(t *testing.T) {
	p := NewSISUProvider()
	if p.appID != SISUDefaultAppID {
		t.Errorf("appID = %q, want %q", p.appID, SISUDefaultAppID)
	}
	if p.titleID != SISUDefaultTitleID {
		t.Errorf("titleID = %q, want %q", p.titleID, SISUDefaultTitleID)
	}
}

// ─── TrySilentRefresh ─────────────────────────────────────────────────────────

func TestSISUProvider_TrySilentRefresh_AlwaysEmpty(t *testing.T) {
	p := NewSISUProvider()
	token, err := p.TrySilentRefresh(context.Background(), "some-cache-json")
	if err != nil {
		t.Fatalf("TrySilentRefresh erreur inattendue : %v", err)
	}
	if token != "" {
		t.Errorf("TrySilentRefresh retourne %q, want vide", token)
	}
}

// ─── TryOAuthRefresh ──────────────────────────────────────────────────────────

func TestSISUProvider_TryOAuthRefresh_EmptyToken(t *testing.T) {
	p := NewSISUProvider()
	token, err := p.TryOAuthRefresh(context.Background(), "")
	if err != nil {
		t.Fatalf("TryOAuthRefresh(empty) erreur inattendue : %v", err)
	}
	if token != "" {
		t.Errorf("TryOAuthRefresh(empty) = %q, want vide", token)
	}
}

// ─── ExchangeWithoutInit ──────────────────────────────────────────────────────

// TestSISUProvider_ExchangeWithoutInit vérifie qu'Exchange appelé sans
// InitDeviceFlow préalable NE panique PAS mais bascule sur l'échange stateless
// (access_token Microsoft → XSTS Halo), exactement comme MSALProvider. C'est le
// chemin emprunté par le pool auto-sync / scheduler / watcher, qui fournit un
// access_token déjà obtenu via OAuth refresh sans passer par le device flow.
//
// On utilise un contexte déjà annulé pour que la chaîne HTTP échoue immédiatement
// (erreur de contexte) sans appel réseau réel : le test reste déterministe et
// vérifie le contrat — pas de panic, erreur propre retournée.
func TestSISUProvider_ExchangeWithoutInit(t *testing.T) {
	p := NewSISUProvider()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // annulé immédiatement → la 1ère requête HTTP échoue sans I/O réseau

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Exchange sans InitDeviceFlow ne doit PLUS paniquer (fallback stateless) ; panic reçue : %v", r)
		}
	}()

	_, err := p.Exchange(ctx, "fake-access-token")
	if err == nil {
		t.Fatal("Exchange stateless avec access_token invalide devrait retourner une erreur")
	}
}

// ─── InitDeviceFlow happy path (URLs mockées) ─────────────────────────────────

// deviceTokenResponse retourne un corps JSON minimal pour le Device Token endpoint.
func deviceTokenResponseBody(token string) []byte {
	b, _ := json.Marshal(map[string]any{"Token": token})
	return b
}

// TestSISUProvider_InitDeviceFlowWithURLs_HappyPath vérifie le flow complet
// InitDeviceFlow avec trois serveurs HTTP de test.
func TestSISUProvider_InitDeviceFlowWithURLs_HappyPath(t *testing.T) {
	// 1. Serveur Device Token Xbox
	srvDeviceToken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(deviceTokenResponseBody("device-token-test")) //nolint:errcheck
	}))
	defer srvDeviceToken.Close()

	// 2. Serveur Xbox Device Code (login.live.com mock)
	srvXboxDeviceCode := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"device_code":      "dc-opaque",
			"user_code":        "TEST99",
			"verification_uri": "https://microsoft.com/link",
			"expires_in":       900,
			"interval":         5,
		}
		b, _ := json.Marshal(resp)
		w.Write(b) //nolint:errcheck
	}))
	defer srvXboxDeviceCode.Close()

	// 3. Serveur SISU authenticate
	srvSISU := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-SessionId", "session-abc")
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"MsaOauthRedirect": "https://login.live.com/oauth20_authorize.srf?sisu=1",
		}
		b, _ := json.Marshal(resp)
		w.Write(b) //nolint:errcheck
	}))
	defer srvSISU.Close()

	p := NewSISUProviderWithIDs("000000004c20a908", "144209987")
	urls := sisuProviderURLs{
		deviceAuth: srvDeviceToken.URL,
		xboxDevice: srvXboxDeviceCode.URL,
		sisuAuth:   srvSISU.URL,
	}

	flow, err := p.initDeviceFlowWithURLs(context.Background(), urls)
	if err != nil {
		t.Fatalf("initDeviceFlowWithURLs erreur inattendue : %v", err)
	}
	if flow == nil {
		t.Fatal("flow est nil")
	}

	// Vérification des champs du flow retourné.
	if got := flow.GetUserCode(); got != "TEST99" {
		t.Errorf("GetUserCode = %q, want TEST99", got)
	}
	// La verification URL doit être celle du DEVICE flow (page de saisie du code),
	// PAS le MsaOauthRedirect SISU (URL d'authorize PKCE qui ne demande jamais le
	// code) — incohérence UX corrigée le 2026-07-13.
	if got := flow.GetVerificationURL(); got != "https://microsoft.com/link" {
		t.Errorf("GetVerificationURL = %q, want https://microsoft.com/link (page de saisie du code, pas l'authorize SISU)", got)
	}
	if got := flow.GetFlowType(); got != "sisu" {
		t.Errorf("GetFlowType = %q, want sisu", got)
	}
	if got := flow.GetExpiresIn(); got != 900 {
		t.Errorf("GetExpiresIn = %d, want 900", got)
	}

	// Vérification que le contexte SISU a bien été stocké DANS LE FLOW (per-flow,
	// pas un slot partagé sur le provider — cf. suppression de p.current).
	sf, ok := flow.(*sisuDeviceFlow)
	if !ok {
		t.Fatalf("flow n'est pas un *sisuDeviceFlow : %T", flow)
	}
	if sf.flowCtx == nil {
		t.Fatal("sisuFlowContext non stocké dans le flow après InitDeviceFlow")
	}
	if sf.flowCtx.sessionID != "session-abc" {
		t.Errorf("sessionID = %q, want session-abc", sf.flowCtx.sessionID)
	}
	if sf.flowCtx.deviceToken != "device-token-test" {
		t.Errorf("deviceToken = %q, want device-token-test", sf.flowCtx.deviceToken)
	}
}

// TestSISUProvider_PerFlowContextIsolation : deux InitDeviceFlow concurrents portent
// chacun LEUR propre contexte SISU. Régression du slot global p.current (revue
// adversariale 2026-07-15) : avant le fix, le 2e InitDeviceFlow écrasait le contexte
// du 1er, et un Exchange stateless (pool auto-sync) consommait le contexte du
// device-flow interactif. Ici chaque flow reste indépendant.
func TestSISUProvider_PerFlowContextIsolation(t *testing.T) {
	newSISUServers := func(sessionID string) (urls sisuProviderURLs, closeAll func()) {
		srvDeviceToken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(deviceTokenResponseBody("dt-" + sessionID)) //nolint:errcheck
		}))
		srvXbox := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			b, _ := json.Marshal(map[string]any{
				"device_code": "dc-" + sessionID, "user_code": "UC-" + sessionID,
				"verification_uri": "https://microsoft.com/link", "expires_in": 900, "interval": 5,
			})
			w.Write(b) //nolint:errcheck
		}))
		srvSISU := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("X-SessionId", sessionID)
			w.Header().Set("Content-Type", "application/json")
			b, _ := json.Marshal(map[string]any{"MsaOauthRedirect": "https://login.live.com/x"})
			w.Write(b) //nolint:errcheck
		}))
		return sisuProviderURLs{deviceAuth: srvDeviceToken.URL, xboxDevice: srvXbox.URL, sisuAuth: srvSISU.URL},
			func() { srvDeviceToken.Close(); srvXbox.Close(); srvSISU.Close() }
	}

	p := NewSISUProvider()

	urlsA, closeA := newSISUServers("sess-A")
	defer closeA()
	flowA, err := p.initDeviceFlowWithURLs(context.Background(), urlsA)
	if err != nil {
		t.Fatalf("flowA init: %v", err)
	}

	urlsB, closeB := newSISUServers("sess-B")
	defer closeB()
	flowB, err := p.initDeviceFlowWithURLs(context.Background(), urlsB)
	if err != nil {
		t.Fatalf("flowB init: %v", err)
	}

	sfA := flowA.(*sisuDeviceFlow)
	sfB := flowB.(*sisuDeviceFlow)

	// Le 2e init NE DOIT PAS avoir écrasé le contexte du 1er.
	if sfA.flowCtx == nil || sfB.flowCtx == nil {
		t.Fatal("un des deux flows a un contexte nil")
	}
	if sfA.flowCtx == sfB.flowCtx {
		t.Fatal("les deux flows partagent le MÊME contexte (slot global non éliminé)")
	}
	if sfA.flowCtx.sessionID != "sess-A" {
		t.Errorf("flowA sessionID = %q, want sess-A (écrasé par flowB ?)", sfA.flowCtx.sessionID)
	}
	if sfB.flowCtx.sessionID != "sess-B" {
		t.Errorf("flowB sessionID = %q, want sess-B", sfB.flowCtx.sessionID)
	}
	if sfA.flowCtx.deviceToken != "dt-sess-A" || sfB.flowCtx.deviceToken != "dt-sess-B" {
		t.Errorf("device tokens croisés : A=%q B=%q", sfA.flowCtx.deviceToken, sfB.flowCtx.deviceToken)
	}
}

// TestSISUProvider_InitDeviceFlow_VerificationURLFallback vérifie que si la réponse
// device n'expose AUCUNE verification_uri (défensif), on retombe sur le
// MsaOauthRedirect SISU plutôt que de renvoyer une URL vide.
func TestSISUProvider_InitDeviceFlow_VerificationURLFallback(t *testing.T) {
	srvDeviceToken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(deviceTokenResponseBody("tok")) //nolint:errcheck
	}))
	defer srvDeviceToken.Close()

	// Réponse device SANS verification_uri.
	srvXboxDeviceCode := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.Marshal(map[string]any{
			"device_code": "dc",
			"user_code":   "FALLBACK",
			"expires_in":  600,
			"interval":    5,
		})
		w.Write(b) //nolint:errcheck
	}))
	defer srvXboxDeviceCode.Close()

	srvSISU := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-SessionId", "s1")
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.Marshal(map[string]any{"MsaOauthRedirect": "https://login.live.com/oauth20_authorize.srf?sisu=1"})
		w.Write(b) //nolint:errcheck
	}))
	defer srvSISU.Close()

	p := NewSISUProvider()
	flow, err := p.initDeviceFlowWithURLs(context.Background(), sisuProviderURLs{
		deviceAuth: srvDeviceToken.URL,
		xboxDevice: srvXboxDeviceCode.URL,
		sisuAuth:   srvSISU.URL,
	})
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if got := flow.GetVerificationURL(); got != "https://login.live.com/oauth20_authorize.srf?sisu=1" {
		t.Errorf("URL fallback = %q, want MsaOauthRedirect (secours quand la réponse device n'a pas d'URL)", got)
	}
}

// TestSISUProvider_InitDeviceFlow_DeviceTokenError vérifie la propagation d'erreur
// si le Device Token endpoint retourne une erreur.
func TestSISUProvider_InitDeviceFlow_DeviceTokenError(t *testing.T) {
	srvFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srvFail.Close()

	p := NewSISUProvider()
	_, err := p.initDeviceFlowWithURLs(context.Background(), sisuProviderURLs{
		deviceAuth: srvFail.URL,
		xboxDevice: "http://unused",
		sisuAuth:   "http://unused",
	})
	if err == nil {
		t.Fatal("erreur attendue quand Device Token endpoint renvoie 500")
	}
}

// TestSISUProvider_InitDeviceFlow_XboxDeviceCodeError vérifie la propagation d'erreur
// si le Xbox Device Code endpoint retourne une erreur.
func TestSISUProvider_InitDeviceFlow_XboxDeviceCodeError(t *testing.T) {
	srvDeviceToken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(deviceTokenResponseBody("tok")) //nolint:errcheck
	}))
	defer srvDeviceToken.Close()

	srvFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srvFail.Close()

	p := NewSISUProvider()
	_, err := p.initDeviceFlowWithURLs(context.Background(), sisuProviderURLs{
		deviceAuth: srvDeviceToken.URL,
		xboxDevice: srvFail.URL,
		sisuAuth:   "http://unused",
	})
	if err == nil {
		t.Fatal("erreur attendue quand XboxDeviceCode endpoint renvoie 400")
	}
}

// TestSISUProvider_InitDeviceFlow_SISUSessionError vérifie la propagation d'erreur
// si le SISU authenticate endpoint retourne une erreur.
func TestSISUProvider_InitDeviceFlow_SISUSessionError(t *testing.T) {
	srvDeviceToken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(deviceTokenResponseBody("tok")) //nolint:errcheck
	}))
	defer srvDeviceToken.Close()

	srvXboxDeviceCode := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.Marshal(map[string]any{
			"device_code": "dc", "user_code": "U1",
			"verification_uri": "https://x.com", "expires_in": 600, "interval": 5,
		})
		w.Write(b) //nolint:errcheck
	}))
	defer srvXboxDeviceCode.Close()

	srvFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "sisu error", http.StatusUnauthorized)
	}))
	defer srvFail.Close()

	p := NewSISUProvider()
	_, err := p.initDeviceFlowWithURLs(context.Background(), sisuProviderURLs{
		deviceAuth: srvDeviceToken.URL,
		xboxDevice: srvXboxDeviceCode.URL,
		sisuAuth:   srvFail.URL,
	})
	if err == nil {
		t.Fatal("erreur attendue quand SISU authenticate endpoint renvoie 401")
	}
}
