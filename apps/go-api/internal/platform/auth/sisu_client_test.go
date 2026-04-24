// Package auth — sisu_client_test.go : tests unitaires pour sisu_client.go.
package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestInitSISUSession_ExtractsSessionID vérifie l'extraction du header X-SessionId.
func TestInitSISUSession_ExtractsSessionID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-xbl-contract-version") != "2" {
			t.Errorf("x-xbl-contract-version attendu '2', obtenu %q", r.Header.Get("x-xbl-contract-version"))
		}
		if r.Header.Get("Signature") == "" {
			t.Error("header Signature absent")
		}

		// Vérifier que le body contient les champs requis
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("body JSON invalide: %v", err)
		}
		if body["AppId"] != "test-app-id" {
			t.Errorf("AppId inattendu: %v", body["AppId"])
		}
		if body["Sandbox"] != "RETAIL" {
			t.Errorf("Sandbox inattendu: %v", body["Sandbox"])
		}

		w.Header().Set("X-SessionId", "session-abc-123")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"MsaOauthRedirect": "https://login.live.com/oauth20_authorize.srf?...",
		})
	}))
	defer srv.Close()

	kp, _ := GeneratePoPKeyPair()
	session, err := initSISUSessionWithURL(
		context.Background(), srv.Client(), kp,
		"device-token-test", "test-app-id", "144209987",
		"challenge", "state", srv.URL+"/authenticate",
	)
	if err != nil {
		t.Fatalf("InitSISUSession: %v", err)
	}
	if session.SessionID != "session-abc-123" {
		t.Errorf("SessionID attendu 'session-abc-123', obtenu %q", session.SessionID)
	}
	if session.MsaOauthRedirect == "" {
		t.Error("MsaOauthRedirect ne doit pas être vide")
	}
}

// TestInitSISUSession_MissingSessionIDHeader vérifie l'erreur si X-SessionId absent.
func TestInitSISUSession_MissingSessionIDHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Pas de header X-SessionId
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"MsaOauthRedirect": "https://..."}) //nolint:errcheck
	}))
	defer srv.Close()

	kp, _ := GeneratePoPKeyPair()
	_, err := initSISUSessionWithURL(
		context.Background(), srv.Client(), kp,
		"device-token", "app-id", "title-id",
		"challenge", "state", srv.URL+"/authenticate",
	)
	if err == nil {
		t.Fatal("attendu une erreur si X-SessionId absent")
	}
}

// TestInitSISUSession_HTTPError vérifie le comportement sur HTTP 401.
func TestInitSISUSession_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	kp, _ := GeneratePoPKeyPair()
	_, err := initSISUSessionWithURL(
		context.Background(), srv.Client(), kp,
		"device-token", "app-id", "title-id",
		"challenge", "state", srv.URL+"/authenticate",
	)
	if err == nil {
		t.Fatal("attendu une erreur sur HTTP 401")
	}
}

// TestCompleteSISUFlow_ExtractsXSTSFields vérifie l'extraction complète du XSTSResult.
func TestCompleteSISUFlow_ExtractsXSTSFields(t *testing.T) {
	notAfterStr := time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Vérifier le body
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
		if body["SessionId"] != "session-test" {
			t.Errorf("SessionId inattendu: %v", body["SessionId"])
		}
		// AccessToken doit être préfixé par "t="
		if at, _ := body["AccessToken"].(string); at != "t=my-access-token" {
			t.Errorf("AccessToken attendu 't=my-access-token', obtenu %q", at)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"AuthorizationToken": map[string]any{
				"Token":    "xsts-token-value",
				"NotAfter": notAfterStr,
				"DisplayClaims": map[string]any{
					"xui": []any{
						map[string]any{
							"uhs": "userhash-value",
							"gtg": "TestGamertag",
							"xid": "xuid-12345",
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	kp, _ := GeneratePoPKeyPair()
	result, err := completeSISUFlowWithURL(
		context.Background(), srv.Client(), kp,
		"device-token", "my-access-token",
		"app-id", "session-test",
		srv.URL+"/authorize",
	)
	if err != nil {
		t.Fatalf("CompleteSISUFlow: %v", err)
	}
	if result.Token != "xsts-token-value" {
		t.Errorf("Token attendu 'xsts-token-value', obtenu %q", result.Token)
	}
	if result.UserHash != "userhash-value" {
		t.Errorf("UserHash attendu 'userhash-value', obtenu %q", result.UserHash)
	}
	if result.Gamertag != "TestGamertag" {
		t.Errorf("Gamertag attendu 'TestGamertag', obtenu %q", result.Gamertag)
	}
	if result.XUID != "xuid-12345" {
		t.Errorf("XUID attendu 'xuid-12345', obtenu %q", result.XUID)
	}
	if result.NotAfter.IsZero() {
		t.Error("NotAfter ne doit pas être zéro")
	}
}

// TestCompleteSISUFlow_MissingToken vérifie l'erreur si Token absent.
func TestCompleteSISUFlow_MissingToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"AuthorizationToken": map[string]any{"NotAToken": "value"},
		})
	}))
	defer srv.Close()

	kp, _ := GeneratePoPKeyPair()
	_, err := completeSISUFlowWithURL(
		context.Background(), srv.Client(), kp,
		"device-token", "access-token",
		"app-id", "session-id",
		srv.URL+"/authorize",
	)
	if err == nil {
		t.Fatal("attendu une erreur si Token absent dans AuthorizationToken")
	}
}

// TestCompleteSISUFlow_MissingAuthorizationToken vérifie l'erreur si AuthorizationToken absent.
func TestCompleteSISUFlow_MissingAuthorizationToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"SomethingElse": "value"}) //nolint:errcheck
	}))
	defer srv.Close()

	kp, _ := GeneratePoPKeyPair()
	_, err := completeSISUFlowWithURL(
		context.Background(), srv.Client(), kp,
		"device-token", "access-token",
		"app-id", "session-id",
		srv.URL+"/authorize",
	)
	if err == nil {
		t.Fatal("attendu une erreur si AuthorizationToken absent")
	}
}

// TestGeneratePKCE vérifie la cohérence verifier/challenge.
func TestGeneratePKCE(t *testing.T) {
	verifier, challenge, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE: %v", err)
	}
	if len(verifier) == 0 {
		t.Error("verifier vide")
	}
	if len(challenge) == 0 {
		t.Error("challenge vide")
	}
	// Vérifier que challenge = base64url(sha256(verifier)) — spec PKCE S256
	import_sha256 := sha256.Sum256([]byte(verifier))
	expected := base64.RawURLEncoding.EncodeToString(import_sha256[:])
	if challenge != expected {
		t.Errorf("challenge incohérent avec verifier: attendu %q, obtenu %q", expected, challenge)
	}

	// Deux appels produisent des valeurs distinctes
	v2, c2, _ := GeneratePKCE()
	if verifier == v2 || challenge == c2 {
		t.Error("GeneratePKCE doit retourner des valeurs uniques à chaque appel")
	}
}
