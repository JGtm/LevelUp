// xsts_rta_test.go : tests du protocole XSTS RTA (déplacés de platform/auth/xsts_test.go, plan D3
// lot 2 ; les tests du magasin et de la boucle de rafraîchissement y restent).

package xblauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- XSTSResult tests ---

func TestXSTSResult_AuthHeader(t *testing.T) {
	r := &XSTSResult{
		Token:    "xsts-token-abc",
		UserHash: "hash123",
		Gamertag: testGamertagInternal,
		XUID:     "12345",
	}
	want := "XBL3.0 x=hash123;xsts-token-abc"
	if got := r.AuthHeader(); got != want {
		t.Errorf("AuthHeader() = %q, want %q", got, want)
	}
}

func TestXSTSResult_AuthHeader_EmptyHash(t *testing.T) {
	r := &XSTSResult{Token: "tok", UserHash: ""}
	want := "XBL3.0 x=;tok"
	if got := r.AuthHeader(); got != want {
		t.Errorf("AuthHeader() = %q, want %q", got, want)
	}
}

// --- extractUserHash tests ---

func TestExtractUserHash_Valid(t *testing.T) {
	resp := map[string]any{
		"DisplayClaims": map[string]any{
			"xui": []any{
				map[string]any{"uhs": "abc123", "gtg": "Player1"},
			},
		},
	}
	got := extractUserHash(resp)
	if got != "abc123" {
		t.Errorf("extractUserHash() = %q, want %q", got, "abc123")
	}
}

func TestExtractUserHash_Missing(t *testing.T) {
	tests := []struct {
		name string
		resp map[string]any
	}{
		{"nil", nil},
		{"empty", map[string]any{}},
		{"no_dc", map[string]any{"Token": "x"}},
		{"no_xui", map[string]any{"DisplayClaims": map[string]any{}}},
		{"empty_xui", map[string]any{"DisplayClaims": map[string]any{"xui": []any{}}}},
		{"no_uhs", map[string]any{"DisplayClaims": map[string]any{
			"xui": []any{map[string]any{"gtg": "P"}},
		}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractUserHash(tt.resp)
			if got != "" {
				t.Errorf("extractUserHash() = %q, want empty", got)
			}
		})
	}
}

// --- extractNotAfter tests ---

func TestExtractNotAfter_Valid(t *testing.T) {
	resp := map[string]any{
		"NotAfter": "2026-04-21T14:00:00.0000000Z",
	}
	got := extractNotAfter(resp)
	if got.IsZero() {
		t.Fatal("extractNotAfter() returned zero, want parsed time")
	}
	want := time.Date(2026, 4, 21, 14, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("extractNotAfter() = %v, want %v", got, want)
	}
}

func TestExtractNotAfter_Missing(t *testing.T) {
	got := extractNotAfter(map[string]any{"Token": "x"})
	if !got.IsZero() {
		t.Errorf("extractNotAfter() = %v, want zero", got)
	}
}

// --- extractNotAfter edge cases ---

func TestExtractNotAfter_WrongType(t *testing.T) {
	// NotAfter est un nombre, pas une string
	resp := map[string]any{"NotAfter": 12345}
	got := extractNotAfter(resp)
	if !got.IsZero() {
		t.Errorf("extractNotAfter() = %v, want zero for wrong type", got)
	}
}

func TestExtractNotAfter_InvalidFormat(t *testing.T) {
	// String présente mais non parseable
	resp := map[string]any{"NotAfter": "not-a-date"}
	got := extractNotAfter(resp)
	if !got.IsZero() {
		t.Errorf("extractNotAfter() = %v, want zero for invalid format", got)
	}
}

// --- requestXSTSTokenFull populates NotAfter ---

func TestRequestXSTSTokenFull_PopulatesNotAfter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"Token":    "xsts_live_tok",
			"NotAfter": "2026-04-21T15:00:00.0000000Z",
			"DisplayClaims": map[string]any{
				"xui": []any{map[string]any{"uhs": "hash123", "gtg": "Player", "xid": "111"}},
			},
		})
	}))
	defer srv.Close()
	result, err := requestXSTSTokenFull(context.Background(), mockClient(srv.URL), "user_tok", "http://xboxlive.com")
	if err != nil {
		t.Fatalf("requestXSTSTokenFull() error = %v", err)
	}
	if result.Token != "xsts_live_tok" {
		t.Errorf("Token = %q, want %q", result.Token, "xsts_live_tok")
	}
	if result.NotAfter.IsZero() {
		t.Fatal("NotAfter doit être renseigné quand la réponse contient NotAfter")
	}
	want := time.Date(2026, 4, 21, 15, 0, 0, 0, time.UTC)
	if !result.NotAfter.Equal(want) {
		t.Errorf("NotAfter = %v, want %v", result.NotAfter, want)
	}
	if result.UserHash != "hash123" {
		t.Errorf("UserHash = %q, want %q", result.UserHash, "hash123")
	}
}

func TestRequestXSTSTokenFull_NoNotAfter_ZeroTime(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"Token": "xsts_tok",
			// pas de NotAfter
			"DisplayClaims": map[string]any{
				"xui": []any{map[string]any{"uhs": "uh"}},
			},
		})
	}))
	defer srv.Close()
	result, err := requestXSTSTokenFull(context.Background(), mockClient(srv.URL), "user_tok", "http://xboxlive.com")
	if err != nil {
		t.Fatalf("requestXSTSTokenFull() error = %v", err)
	}
	if !result.NotAfter.IsZero() {
		t.Errorf("NotAfter doit être zero quand absent de la réponse, got %v", result.NotAfter)
	}
}
