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

// ─── InitDeviceFlow (endpoint mocké) ──────────────────────────────────────────

// xboxDeviceCodeServer simule login.live.com/oauth20_connect.srf et compte les appels.
func xboxDeviceCodeServer(t *testing.T, userCode string, calls *int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*calls++
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.Marshal(map[string]any{
			"device_code":      "dc-opaque",
			"user_code":        userCode,
			"verification_uri": "https://microsoft.com/link",
			"expires_in":       900,
			"interval":         5,
		})
		w.Write(b) //nolint:errcheck
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestSISUProvider_InitDeviceFlowWithURL_HappyPath : InitDeviceFlow n'appelle que
// le démarrage du Device Code Flow (aucun device token, aucune paire PoP) et rend
// un flow autonome.
func TestSISUProvider_InitDeviceFlowWithURL_HappyPath(t *testing.T) {
	calls := 0
	srv := xboxDeviceCodeServer(t, "TEST99", &calls)

	p := NewSISUProviderWithIDs("000000004c20a908", "144209987")
	flow, err := p.initDeviceFlowWithURL(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("initDeviceFlowWithURL erreur inattendue : %v", err)
	}
	if calls != 1 {
		t.Errorf("appels au démarrage du device-flow = %d, attendu 1", calls)
	}
	if got := flow.GetUserCode(); got != "TEST99" {
		t.Errorf("GetUserCode = %q, want TEST99", got)
	}
	// La verification URL est celle du DEVICE flow (page de saisie du code).
	if got := flow.GetVerificationURL(); got != "https://microsoft.com/link" {
		t.Errorf("GetVerificationURL = %q, want https://microsoft.com/link", got)
	}
	if got := flow.GetFlowType(); got != "sisu" {
		t.Errorf("GetFlowType = %q, want sisu", got)
	}
	if got := flow.GetExpiresIn(); got != 900 {
		t.Errorf("GetExpiresIn = %d, want 900", got)
	}
	sf, ok := flow.(*sisuDeviceFlow)
	if !ok {
		t.Fatalf("flow n'est pas un *sisuDeviceFlow : %T", flow)
	}
	if sf.appID != "000000004c20a908" || sf.deviceCode != "dc-opaque" {
		t.Errorf("flow mal initialisé : appID=%q deviceCode=%q", sf.appID, sf.deviceCode)
	}
}

// TestSISUProvider_InitDeviceFlow_XboxDeviceCodeError vérifie la propagation d'erreur
// si le Xbox Device Code endpoint retourne une erreur.
func TestSISUProvider_InitDeviceFlow_XboxDeviceCodeError(t *testing.T) {
	srvFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srvFail.Close()

	p := NewSISUProvider()
	if _, err := p.initDeviceFlowWithURL(context.Background(), srvFail.URL); err == nil {
		t.Fatal("erreur attendue quand XboxDeviceCode endpoint renvoie 400")
	}
}
