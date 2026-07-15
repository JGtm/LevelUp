// Package auth — device_token_test.go : tests unitaires pour device_token.go.
package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRequestDeviceToken_Success vérifie le cas nominal avec un serveur de test.
func TestRequestDeviceToken_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Vérifier les headers attendus
		if r.Header.Get("x-xbl-contract-version") != "2" {
			t.Errorf("x-xbl-contract-version attendu '2', obtenu %q", r.Header.Get("x-xbl-contract-version"))
		}
		if r.Header.Get("Signature") == "" {
			t.Error("header Signature absent")
		}
		if r.Method != http.MethodPost {
			t.Errorf("méthode attendue POST, obtenu %s", r.Method)
		}

		// Vérifier que le body est JSON valide avec les champs requis
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("body JSON invalide: %v", err)
		}
		if body["RelyingParty"] != "http://auth.xboxlive.com" {
			t.Errorf("RelyingParty inattendu: %v", body["RelyingParty"])
		}
		props, ok := body["Properties"].(map[string]any)
		if !ok {
			t.Error("champ Properties absent ou mauvais type")
		}
		if props["AuthMethod"] != "ProofOfPossession" {
			t.Errorf("AuthMethod inattendu: %v", props["AuthMethod"])
		}
		// "Android" obligatoire : SISU /authorize ne fait pas confiance aux
		// device tokens Win32 (attestation TPM requise) — fix 401 2026-07-15.
		if props["DeviceType"] != "Android" {
			t.Errorf("DeviceType inattendu: %v", props["DeviceType"])
		}
		if _, present := props["Version"]; present {
			t.Error("Version ne doit plus être envoyée (aligné MinecraftAuth)")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"Token": "device-token-test-123"}) //nolint:errcheck
	}))
	defer srv.Close()

	kp, err := GeneratePoPKeyPair()
	if err != nil {
		t.Fatalf("GeneratePoPKeyPair: %v", err)
	}

	// Remplacer l'URL réelle par le serveur de test en monkey-patching temporaire.
	// On appelle la fonction interne buildDeviceTokenRequest pour tester avec un client custom.
	token, err := requestDeviceTokenWithURL(context.Background(), srv.Client(), kp, srv.URL+"/device/authenticate")
	if err != nil {
		t.Fatalf("RequestDeviceToken: %v", err)
	}
	if token != "device-token-test-123" {
		t.Errorf("token attendu 'device-token-test-123', obtenu %q", token)
	}
}

// TestRequestDeviceToken_HTTPError vérifie le comportement sur un HTTP 401.
func TestRequestDeviceToken_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	kp, _ := GeneratePoPKeyPair()
	_, err := requestDeviceTokenWithURL(context.Background(), srv.Client(), kp, srv.URL+"/device/authenticate")
	if err == nil {
		t.Fatal("attendu une erreur sur HTTP 401")
	}
}

// TestRequestDeviceToken_MissingTokenField vérifie le comportement si Token est absent.
func TestRequestDeviceToken_MissingTokenField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"NotAToken": "value"}) //nolint:errcheck
	}))
	defer srv.Close()

	kp, _ := GeneratePoPKeyPair()
	_, err := requestDeviceTokenWithURL(context.Background(), srv.Client(), kp, srv.URL+"/device/authenticate")
	if err == nil {
		t.Fatal("attendu une erreur si champ Token absent")
	}
}

// TestRequestDeviceToken_DeviceIDFormat vérifie que l'ID de device est au bon format.
func TestRequestDeviceToken_DeviceIDFormat(t *testing.T) {
	var capturedID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
		if props, ok := body["Properties"].(map[string]any); ok {
			capturedID, _ = props["Id"].(string)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"Token": "tok"}) //nolint:errcheck
	}))
	defer srv.Close()

	kp, _ := GeneratePoPKeyPair()
	requestDeviceTokenWithURL(context.Background(), srv.Client(), kp, srv.URL+"/device/authenticate") //nolint:errcheck
	// Format attendu : {UUID-EN-MAJUSCULES}
	if len(capturedID) != 38 { // {xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx}
		t.Errorf("longueur ID attendue 38, obtenu %d: %q", len(capturedID), capturedID)
	}
	if capturedID[0] != '{' || capturedID[37] != '}' {
		t.Errorf("ID doit être entouré d'accolades: %q", capturedID)
	}
}
