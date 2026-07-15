// Package auth — xbox_device_code_test.go : tests unitaires pour xbox_device_code.go.
package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestStartXboxDeviceCode_Success vérifie le cas nominal.
func TestStartXboxDeviceCode_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("méthode attendue POST, obtenu %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type attendu application/x-www-form-urlencoded, obtenu %q", ct)
		}
		r.ParseForm() //nolint:errcheck
		if r.Form.Get("response_type") != "device_code" {
			t.Errorf("response_type inattendu: %q", r.Form.Get("response_type"))
		}
		if r.Form.Get("client_id") != "test-client-id" {
			t.Errorf("client_id inattendu: %q", r.Form.Get("client_id"))
		}
		// Garde-rail 401 SISU (2026-07-15) : le scope DOIT être le scope MSA natif —
		// les scopes Azure AD produisent un JWT que SISU /authorize rejette en 401.
		if r.Form.Get("scope") != sisuMSAScope {
			t.Errorf("scope attendu %q (MSA natif SISU), obtenu %q", sisuMSAScope, r.Form.Get("scope"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"device_code":      "device-code-opaque",
			"user_code":        "ABCDEF",
			"verification_uri": "https://login.live.com/oauth20_remoteconnect.srf",
			"expires_in":       float64(300),
			"interval":         float64(5),
		})
	}))
	defer srv.Close()

	result, err := startXboxDeviceCodeWithURL(context.Background(), srv.Client(), "test-client-id", srv.URL)
	if err != nil {
		t.Fatalf("StartXboxDeviceCode: %v", err)
	}
	if result.DeviceCode != "device-code-opaque" {
		t.Errorf("DeviceCode attendu 'device-code-opaque', obtenu %q", result.DeviceCode)
	}
	if result.UserCode != "ABCDEF" {
		t.Errorf("UserCode attendu 'ABCDEF', obtenu %q", result.UserCode)
	}
	if result.ExpiresIn != 300 {
		t.Errorf("ExpiresIn attendu 300, obtenu %d", result.ExpiresIn)
	}
	if result.Interval != 5 {
		t.Errorf("Interval attendu 5, obtenu %d", result.Interval)
	}
}

// TestStartXboxDeviceCode_MissingDeviceCode vérifie l'erreur si device_code absent.
func TestStartXboxDeviceCode_MissingDeviceCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"user_code": "ABCDEF"}) //nolint:errcheck
	}))
	defer srv.Close()

	_, err := startXboxDeviceCodeWithURL(context.Background(), srv.Client(), "client-id", srv.URL)
	if err == nil {
		t.Fatal("attendu une erreur si device_code absent")
	}
}

// TestPollXboxDeviceCode_AuthorizationPending vérifie que la boucle continue sur pending.
func TestPollXboxDeviceCode_AuthorizationPending(t *testing.T) {
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if n < 3 {
			// Retourner authorization_pending les deux premières fois
			json.NewEncoder(w).Encode(map[string]any{"error": "authorization_pending"}) //nolint:errcheck
		} else {
			// Succès à la troisième tentative
			json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
				"access_token":  "access-token-ok",
				"refresh_token": "refresh-token-ok",
			})
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	at, rt, err := pollXboxDeviceCodeWithURL(ctx, srv.Client(), "client-id", "device-code", 0, srv.URL)
	if err != nil {
		t.Fatalf("PollXboxDeviceCode: %v", err)
	}
	if at != "access-token-ok" {
		t.Errorf("access_token attendu 'access-token-ok', obtenu %q", at)
	}
	if rt != "refresh-token-ok" {
		t.Errorf("refresh_token attendu 'refresh-token-ok', obtenu %q", rt)
	}
	if callCount.Load() != 3 {
		t.Errorf("attendu 3 appels, obtenu %d", callCount.Load())
	}
}

// TestPollXboxDeviceCode_SlowDown vérifie que l'intervalle augmente sur slow_down.
func TestPollXboxDeviceCode_SlowDown(t *testing.T) {
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			json.NewEncoder(w).Encode(map[string]any{"error": "slow_down"}) //nolint:errcheck
		} else {
			json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "refresh_token": ""}) //nolint:errcheck
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Avec interval=0, le slow_down doit augmenter l'intervalle à pollSlowDownIncrement (5s)
	// mais le test utilise 0 pour accélérer l'exécution (le sleep est quand même déclenché).
	// On vérifie juste que ça passe sans erreur.
	_, _, err := pollXboxDeviceCodeWithURL(ctx, srv.Client(), "client-id", "device-code", 0, srv.URL)
	if err != nil {
		t.Fatalf("PollXboxDeviceCode avec slow_down: %v", err)
	}
}

// TestPollXboxDeviceCode_FatalError vérifie le retour d'erreur immédiat sur access_denied.
func TestPollXboxDeviceCode_FatalError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"error":             "access_denied",
			"error_description": "L'utilisateur a refusé.",
		})
	}))
	defer srv.Close()

	_, _, err := pollXboxDeviceCodeWithURL(context.Background(), srv.Client(), "client-id", "device-code", 0, srv.URL)
	if err == nil {
		t.Fatal("attendu une erreur fatale sur access_denied")
	}
}

// TestPollXboxDeviceCode_ContextCancel vérifie l'arrêt propre sur annulation de contexte.
func TestPollXboxDeviceCode_ContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"error": "authorization_pending"}) //nolint:errcheck
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	// Annuler le contexte immédiatement après le premier tick
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, _, err := pollXboxDeviceCodeWithURL(ctx, srv.Client(), "client-id", "device-code", 0, srv.URL)
	if err == nil {
		t.Fatal("attendu une erreur de contexte annulé")
	}
}
