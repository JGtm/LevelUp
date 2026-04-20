package auth

import (
	"testing"
	"time"
)

// --- XSTSResult tests ---

func TestXSTSResult_AuthHeader(t *testing.T) {
	r := &XSTSResult{
		Token:    "xsts-token-abc",
		UserHash: "hash123",
		Gamertag: "TestPlayer",
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

// --- TokenStore tests ---

func TestTokenStore_LoadSave_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/tokens.json"
	store := NewTokenStore(path)

	// Load from non-existent file → empty
	tokens, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if tokens.AccessToken != "" {
		t.Errorf("expected empty AccessToken, got %q", tokens.AccessToken)
	}

	// Save
	tokens.AccessToken = "at-123"
	tokens.RefreshToken = "rt-456"
	tokens.XSTSToken = "xsts-789"
	tokens.XSTSUserHash = "hash-abc"
	tokens.XSTSGamertag = "TestGT"
	tokens.XSTSXUID = "xuid-999"
	tokens.XSTSExpiresAt = time.Date(2026, 4, 20, 16, 0, 0, 0, time.UTC)
	tokens.OAuthExpiresAt = time.Date(2026, 4, 20, 15, 0, 0, 0, time.UTC)
	if err := store.Save(tokens); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load back
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.AccessToken != "at-123" {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, "at-123")
	}
	if loaded.RefreshToken != "rt-456" {
		t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, "rt-456")
	}
	if loaded.XSTSToken != "xsts-789" {
		t.Errorf("XSTSToken = %q, want %q", loaded.XSTSToken, "xsts-789")
	}
	if loaded.XSTSUserHash != "hash-abc" {
		t.Errorf("XSTSUserHash = %q, want %q", loaded.XSTSUserHash, "hash-abc")
	}
	if loaded.XSTSGamertag != "TestGT" {
		t.Errorf("XSTSGamertag = %q, want %q", loaded.XSTSGamertag, "TestGT")
	}
}

func TestTokenStore_NestedDir(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/deep/nested/tokens.json"
	store := NewTokenStore(path)

	tokens := &StoredTokens{AccessToken: "nested-test"}
	if err := store.Save(tokens); err != nil {
		t.Fatalf("Save() nested dir error = %v", err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.AccessToken != "nested-test" {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, "nested-test")
	}
}

func TestTokenStore_UpdateXSTS(t *testing.T) {
	dir := t.TempDir()
	store := NewTokenStore(dir + "/tokens.json")

	// Save initial
	if err := store.Save(&StoredTokens{RefreshToken: "keep-this"}); err != nil {
		t.Fatal(err)
	}

	result := &XSTSResult{
		Token:    "new-xsts",
		UserHash: "new-hash",
		Gamertag: "GT",
		XUID:     "X",
	}
	if err := store.UpdateXSTS(result, 90*time.Minute); err != nil {
		t.Fatalf("UpdateXSTS() error = %v", err)
	}

	loaded, _ := store.Load()
	if loaded.RefreshToken != "keep-this" {
		t.Errorf("RefreshToken lost: %q", loaded.RefreshToken)
	}
	if loaded.XSTSToken != "new-xsts" {
		t.Errorf("XSTSToken = %q, want %q", loaded.XSTSToken, "new-xsts")
	}
	if !loaded.IsXSTSValid(0) {
		t.Error("XSTS should be valid after UpdateXSTS")
	}
}

func TestTokenStore_UpdateOAuth(t *testing.T) {
	dir := t.TempDir()
	store := NewTokenStore(dir + "/tokens.json")

	if err := store.Save(&StoredTokens{XSTSToken: "keep-xsts"}); err != nil {
		t.Fatal(err)
	}

	if err := store.UpdateOAuth("new-at", "new-rt", 60*time.Minute); err != nil {
		t.Fatalf("UpdateOAuth() error = %v", err)
	}

	loaded, _ := store.Load()
	if loaded.XSTSToken != "keep-xsts" {
		t.Errorf("XSTSToken lost: %q", loaded.XSTSToken)
	}
	if loaded.AccessToken != "new-at" {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, "new-at")
	}
	if loaded.RefreshToken != "new-rt" {
		t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, "new-rt")
	}
	if !loaded.IsOAuthValid(0) {
		t.Error("OAuth should be valid after UpdateOAuth")
	}
}

// --- StoredTokens validity tests ---

func TestStoredTokens_IsXSTSValid(t *testing.T) {
	tests := []struct {
		name   string
		token  string
		exp    time.Time
		margin time.Duration
		want   bool
	}{
		{"empty_token", "", time.Now().Add(time.Hour), 0, false},
		{"expired", "tok", time.Now().Add(-time.Minute), 0, false},
		{"valid", "tok", time.Now().Add(time.Hour), 0, true},
		{"valid_with_margin", "tok", time.Now().Add(20 * time.Minute), 15 * time.Minute, true},
		{"expired_with_margin", "tok", time.Now().Add(10 * time.Minute), 15 * time.Minute, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &StoredTokens{XSTSToken: tt.token, XSTSExpiresAt: tt.exp}
			if got := s.IsXSTSValid(tt.margin); got != tt.want {
				t.Errorf("IsXSTSValid(%v) = %v, want %v", tt.margin, got, tt.want)
			}
		})
	}
}

func TestStoredTokens_IsOAuthValid(t *testing.T) {
	s := &StoredTokens{AccessToken: "at", OAuthExpiresAt: time.Now().Add(time.Hour)}
	if !s.IsOAuthValid(0) {
		t.Error("should be valid")
	}
	s2 := &StoredTokens{AccessToken: "", OAuthExpiresAt: time.Now().Add(time.Hour)}
	if s2.IsOAuthValid(0) {
		t.Error("empty token should not be valid")
	}
}

// --- RefreshLoop tests ---

func TestRefreshLoop_New(t *testing.T) {
	store := NewTokenStore(t.TempDir() + "/tokens.json")
	called := false
	rl := NewRefreshLoop(store, func(_ *XSTSResult) { called = true })
	if rl == nil {
		t.Fatal("NewRefreshLoop returned nil")
	}
	if rl.store != store {
		t.Error("store mismatch")
	}
	if called {
		t.Error("callback should not have been called yet")
	}
}

func TestTokenStore_Path(t *testing.T) {
	s := NewTokenStore("/some/path/tokens.json")
	if s.Path() != "/some/path/tokens.json" {
		t.Errorf("Path() = %q", s.Path())
	}
}
