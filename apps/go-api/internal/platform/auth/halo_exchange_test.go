// Package auth — halo_exchange_test.go : tests unitaires des helpers d'échange Halo.
// Package auth (non _test) pour accéder aux fonctions privées.
package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// redirectTransport redirige toutes les requêtes vers une URL cible fixe (pour mocker les URLs hardcodées).
type redirectTransport struct {
	targetBase string
	wrapped    http.RoundTripper
}

func (t *redirectTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r2 := r.Clone(r.Context())
	base, _ := url.Parse(t.targetBase)
	r2.URL.Scheme = base.Scheme
	r2.URL.Host = base.Host
	return t.wrapped.RoundTrip(r2)
}

func mockClient(baseURL string) *http.Client {
	return &http.Client{
		Transport: &redirectTransport{
			targetBase: baseURL,
			wrapped:    http.DefaultTransport,
		},
	}
}

// extractDisplayClaims

func TestExtractDisplayClaims_Valid(t *testing.T) {
	resp := map[string]any{
		"DisplayClaims": map[string]any{
			"xui": []any{map[string]any{"gtg": "Player1", "xid": "1234567890"}},
		},
	}
	gamertag, xuid := extractDisplayClaims(resp)
	if gamertag != "Player1" {
		t.Errorf("expected Player1, got %q", gamertag)
	}
	if xuid != "1234567890" {
		t.Errorf("expected 1234567890, got %q", xuid)
	}
}

func TestExtractDisplayClaims_Missing(t *testing.T) {
	g, x := extractDisplayClaims(map[string]any{})
	if g != "" || x != "" {
		t.Errorf("expected empty, got %q %q", g, x)
	}
}

func TestExtractDisplayClaims_EmptyXUI(t *testing.T) {
	resp := map[string]any{"DisplayClaims": map[string]any{"xui": []any{}}}
	g, x := extractDisplayClaims(resp)
	if g != "" || x != "" {
		t.Errorf("expected empty, got %q %q", g, x)
	}
}

func TestExtractDisplayClaims_WrongType(t *testing.T) {
	resp := map[string]any{"DisplayClaims": "not_a_map"}
	g, x := extractDisplayClaims(resp)
	if g != "" || x != "" {
		t.Errorf("expected empty, got %q %q", g, x)
	}
}

// postJSON

func TestPostJSON_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"Token": "test_token"})
	}))
	defer srv.Close()
	resp, err := postJSON(context.Background(), srv.Client(), srv.URL, nil, map[string]string{"k": "v"})
	if err != nil {
		t.Fatal(err)
	}
	if resp["Token"] != "test_token" {
		t.Errorf("expected test_token, got %v", resp["Token"])
	}
}

func TestPostJSON_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()
	_, err := postJSON(context.Background(), srv.Client(), srv.URL, nil, nil)
	if err == nil {
		t.Error("expected error for HTTP 401")
	}
}

func TestPostJSON_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{invalid"))
	}))
	defer srv.Close()
	_, err := postJSON(context.Background(), srv.Client(), srv.URL, nil, nil)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestPostJSON_WithHeaders(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("x-custom")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer srv.Close()
	_, err := postJSON(context.Background(), srv.Client(), srv.URL, map[string]string{"x-custom": "test-value"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "test-value" {
		t.Errorf("expected test-value, got %q", got)
	}
}

// requestUserToken

func TestRequestUserToken_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"Token": "user_tok_abc"})
	}))
	defer srv.Close()
	token, err := requestUserToken(context.Background(), mockClient(srv.URL), "fake_access_token")
	if err != nil {
		t.Fatal(err)
	}
	if token != "user_tok_abc" {
		t.Errorf("expected user_tok_abc, got %q", token)
	}
}

// TestRequestUserToken_RetryTPrefixOn401 : les deux familles de tokens (app
// Azure → d=, client Xbox natif/SISU → t=) sont indistinguables par format —
// sur un 401 en d=, l'échange DOIT être retenté en t= (fix 2026-07-15 : une
// heuristique par format envoyait t= aux tokens Azure et cassait le pool).
func TestRequestUserToken_RetryTPrefixOn401(t *testing.T) {
	var prefixes []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		props, _ := body["Properties"].(map[string]any)
		ticket, _ := props["RpsTicket"].(string)
		prefixes = append(prefixes, ticket[:2])
		if ticket[:2] == "d=" {
			http.Error(w, "", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"Token": "user_tok_msa"})
	}))
	defer srv.Close()

	token, err := requestUserToken(context.Background(), mockClient(srv.URL), "EwAmsa_ticket")
	if err != nil {
		t.Fatalf("retry t= attendu après 401 en d= : %v", err)
	}
	if token != "user_tok_msa" {
		t.Errorf("token = %q, want user_tok_msa", token)
	}
	if len(prefixes) != 2 || prefixes[0] != "d=" || prefixes[1] != "t=" {
		t.Errorf("préfixes essayés = %v, want [d= t=]", prefixes)
	}
}

// TestRequestUserToken_NoRetryOnOtherError : un 429/5xx ne déclenche PAS le
// retry de famille (seul un 401 signale un mauvais préfixe).
func TestRequestUserToken_NoRetryOnOtherError(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		http.Error(w, "throttled", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	_, err := requestUserToken(context.Background(), mockClient(srv.URL), "tok")
	if err == nil {
		t.Fatal("erreur attendue")
	}
	if calls != 1 {
		t.Errorf("appels = %d, want 1 (pas de retry hors 401)", calls)
	}
}

func TestRequestUserToken_MissingToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"other": "value"})
	}))
	defer srv.Close()
	_, err := requestUserToken(context.Background(), mockClient(srv.URL), "fake_access_token")
	if err == nil {
		t.Error("expected error when Token absent")
	}
}

// requestXSTSToken

func TestRequestXSTSToken_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"Token": "xsts_tok",
			"DisplayClaims": map[string]any{
				"xui": []any{map[string]any{"gtg": testGamertagInternal, "xid": "9876"}},
			},
		})
	}))
	defer srv.Close()
	token, gamertag, xuid, err := requestXSTSToken(context.Background(), mockClient(srv.URL), "user_tok", "https://xsts.example.com/")
	if err != nil {
		t.Fatal(err)
	}
	if token != "xsts_tok" {
		t.Errorf("expected xsts_tok, got %q", token)
	}
	if gamertag != testGamertagInternal {
		t.Errorf("expected %s, got %q", testGamertagInternal, gamertag)
	}
	if xuid != "9876" {
		t.Errorf("expected 9876, got %q", xuid)
	}
}

func TestRequestXSTSToken_MissingToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"other": "value"})
	}))
	defer srv.Close()
	_, _, _, err := requestXSTSToken(context.Background(), mockClient(srv.URL), "user_tok", "https://xsts.example.com/")
	if err == nil {
		t.Error("expected error when Token absent")
	}
}

// requestSpartanToken

func TestRequestSpartanToken_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"SpartanToken": "spartan_xyz"})
	}))
	defer srv.Close()
	token, _, err := requestSpartanToken(context.Background(), mockClient(srv.URL), "xsts_tok")
	if err != nil {
		t.Fatal(err)
	}
	if token != "spartan_xyz" {
		t.Errorf("expected spartan_xyz, got %q", token)
	}
}

func TestRequestSpartanToken_Missing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"other": "value"})
	}))
	defer srv.Close()
	_, _, err := requestSpartanToken(context.Background(), mockClient(srv.URL), "xsts_tok")
	if err == nil {
		t.Error("expected error when SpartanToken absent")
	}
}

// requestClearanceToken

func TestRequestClearanceToken_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-343-authorization-spartan") == "" {
			http.Error(w, "missing spartan", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"FlightConfigurationId": "flight_123"})
	}))
	defer srv.Close()
	token, err := requestClearanceToken(context.Background(), mockClient(srv.URL), "spartan_tok")
	if err != nil {
		t.Fatal(err)
	}
	if token != "flight_123" {
		t.Errorf("expected flight_123, got %q", token)
	}
}

func TestRequestClearanceToken_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()
	_, err := requestClearanceToken(context.Background(), mockClient(srv.URL), "spartan_tok")
	if err == nil {
		t.Error("expected error for HTTP 403")
	}
}

func TestRequestClearanceToken_MissingField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"other": "value"})
	}))
	defer srv.Close()
	_, err := requestClearanceToken(context.Background(), mockClient(srv.URL), "spartan_tok")
	if err == nil {
		t.Error("expected error when FlightConfigurationId absent")
	}
}

// --- Garde-fou A1 : capture de l'expiry réel du Spartan (ExpiresUtc) ---

func TestParseSpartanExpiry(t *testing.T) {
	// ExpiresUtc.ISO8601Date présent → parsé.
	resp := map[string]any{
		"SpartanToken": "tok",
		"ExpiresUtc":   map[string]any{"ISO8601Date": "2030-01-02T03:04:05Z"},
	}
	got := parseSpartanExpiry(resp)
	if got.IsZero() {
		t.Fatal("expiry devrait être parsé depuis ExpiresUtc.ISO8601Date")
	}
	if got.Year() != 2030 || got.Month() != 1 || got.Day() != 2 {
		t.Errorf("expiry mal parsé: %v", got)
	}

	// Champ absent → zéro (expiry inconnu, fallback côté cache).
	if exp := parseSpartanExpiry(map[string]any{"SpartanToken": "tok"}); !exp.IsZero() {
		t.Errorf("ExpiresUtc absent → expiry zéro attendu, got %v", exp)
	}
	// Date illisible → zéro.
	if exp := parseSpartanExpiry(map[string]any{
		"ExpiresUtc": map[string]any{"ISO8601Date": "pas-une-date"},
	}); !exp.IsZero() {
		t.Errorf("date illisible → expiry zéro attendu, got %v", exp)
	}
}
