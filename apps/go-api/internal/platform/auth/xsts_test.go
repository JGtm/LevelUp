package auth

import (
	"context"
	"testing"
	"time"
)

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

// --- TokenStore UpdateXSTS tests ---

func TestTokenStore_UpdateXSTS(t *testing.T) {
	dir := t.TempDir()
	store := NewTokenStore(dir + "/tokens.json")

	// Save initial
	if err := store.Save(&StoredTokens{AccessToken: "keep-this"}); err != nil {
		t.Fatal(err)
	}

	// Sans NotAfter → fallback sur le TTL passé
	result := &XSTSResult{
		Token:    "new-xsts",
		UserHash: "new-hash",
		Gamertag: "GT",
		XUID:     "X",
	}
	if err := store.UpdateXSTS(result, 55*time.Minute); err != nil {
		t.Fatalf("UpdateXSTS() error = %v", err)
	}

	loaded, _ := store.Load()
	if loaded.AccessToken != "keep-this" {
		t.Errorf("AccessToken lost: %q", loaded.AccessToken)
	}
	if loaded.XSTSToken != "new-xsts" {
		t.Errorf("XSTSToken = %q, want %q", loaded.XSTSToken, "new-xsts")
	}
	if !loaded.IsXSTSValid(0) {
		t.Error("XSTS should be valid after UpdateXSTS")
	}
}

func TestTokenStore_UpdateXSTS_UsesNotAfter(t *testing.T) {
	dir := t.TempDir()
	store := NewTokenStore(dir + "/tokens.json")

	notAfter := time.Now().Add(45 * time.Minute)
	result := &XSTSResult{
		Token:    "xsts",
		UserHash: "uh",
		NotAfter: notAfter,
	}
	if err := store.UpdateXSTS(result, 90*time.Minute); err != nil { // fallback ignoré
		t.Fatalf("UpdateXSTS() error = %v", err)
	}

	loaded, _ := store.Load()
	// L'expiration doit correspondre à NotAfter (±1s), pas à time.Now()+90min
	diff := loaded.XSTSExpiresAt.Sub(notAfter)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("XSTSExpiresAt = %v, want ~%v (diff %v)", loaded.XSTSExpiresAt, notAfter, diff)
	}
}

func TestTokenStore_UpdateOAuth(t *testing.T) {
	dir := t.TempDir()
	store := NewTokenStore(dir + "/tokens.json")

	if err := store.Save(&StoredTokens{XSTSToken: "keep-xsts"}); err != nil {
		t.Fatal(err)
	}

	if err := store.UpdateOAuth("new-at", 60*time.Minute); err != nil {
		t.Fatalf("UpdateOAuth() error = %v", err)
	}

	loaded, _ := store.Load()
	if loaded.XSTSToken != "keep-xsts" {
		t.Errorf("XSTSToken lost: %q", loaded.XSTSToken)
	}
	if loaded.AccessToken != "new-at" {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, "new-at")
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

// --- RefreshLoop.check logic tests ---

// refreshLoopWithMock crée un RefreshLoop avec une fonction XSTS mockée et un
// MultiUserTokenStore attaché (source unique du refresh_token, ADR 0023 Phase 5).
func refreshLoopWithMock(t *testing.T, mockFn XSTSAcquireFn, callback RefreshCallback) (*RefreshLoop, *TokenStore, *MultiUserTokenStore) {
	t.Helper()
	dir := t.TempDir()
	store := NewTokenStore(dir + "/tokens.json")
	multi := NewMultiUserTokenStore(dir + "/watcher_tokens")
	rl := NewRefreshLoop(store, callback).WithMultiUserMirror(multi)
	rl.acquireXSTSFn = mockFn
	return rl, store, multi
}

func TestRefreshLoop_Check_SkipsWhenNoRefreshToken(t *testing.T) {
	called := false
	rl, store, _ := refreshLoopWithMock(t,
		func(_ context.Context, _ string) (*XSTSResult, error) {
			called = true
			return &XSTSResult{Token: "new"}, nil
		},
		nil,
	)
	// MultiUserTokenStore vide → pas de refresh_token pour le tracker.
	_ = store.Save(&StoredTokens{
		AccessToken:   "at",
		XSTSXUID:      "1234",
		XSTSToken:     "old",
		XSTSExpiresAt: time.Now().Add(5 * time.Minute), // < margin → devrait refresh, mais pas de RT
	})
	rl.check(context.Background())
	if called {
		t.Error("acquireXSTSFn ne doit pas être appelé sans refresh_token")
	}
}

func TestRefreshLoop_Check_SkipsWhenXSTSValid(t *testing.T) {
	called := false
	rl, store, multi := refreshLoopWithMock(t,
		func(_ context.Context, _ string) (*XSTSResult, error) {
			called = true
			return &XSTSResult{Token: "new"}, nil
		},
		nil,
	)
	_ = multi.UpdateOAuthRefreshToken("1234", "rt")
	_ = store.Save(&StoredTokens{
		AccessToken:    "at",
		XSTSXUID:       "1234",
		OAuthExpiresAt: time.Now().Add(time.Hour),
		XSTSToken:      "old",
		XSTSExpiresAt:  time.Now().Add(30 * time.Minute), // > 20min margin → pas de refresh
	})
	rl.check(context.Background())
	if called {
		t.Error("acquireXSTSFn ne doit pas être appelé si XSTS encore valide > margin")
	}
}

func TestRefreshLoop_Check_RefreshesWhenXSTSNearExpiry(t *testing.T) {
	var callbackResult *XSTSResult
	rl, store, multi := refreshLoopWithMock(t,
		func(_ context.Context, _ string) (*XSTSResult, error) {
			return &XSTSResult{
				Token:    "new-xsts",
				UserHash: "uh",
				NotAfter: time.Now().Add(60 * time.Minute),
			}, nil
		},
		func(r *XSTSResult) { callbackResult = r },
	)
	_ = multi.UpdateOAuthRefreshToken("1234", "rt")
	_ = store.Save(&StoredTokens{
		AccessToken:    "at",
		XSTSXUID:       "1234",
		OAuthExpiresAt: time.Now().Add(time.Hour),
		XSTSToken:      "old",
		XSTSExpiresAt:  time.Now().Add(15 * time.Minute), // 15min < 20min margin → refresh
	})
	rl.check(context.Background())
	if callbackResult == nil {
		t.Fatal("callback doit être appelé après refresh XSTS")
	}
	if callbackResult.Token != "new-xsts" {
		t.Errorf("callback reçu token=%q, want %q", callbackResult.Token, "new-xsts")
	}
	// Le store doit contenir le nouveau token
	loaded, _ := store.Load()
	if loaded.XSTSToken != "new-xsts" {
		t.Errorf("store XSTSToken = %q, want %q", loaded.XSTSToken, "new-xsts")
	}
	// L'expiration doit utiliser NotAfter, pas xstsDefaultTTL
	if time.Until(loaded.XSTSExpiresAt) < 55*time.Minute {
		t.Error("XSTSExpiresAt devrait refléter le NotAfter (~60min), pas le fallback 55min")
	}
}

func TestRefreshLoop_Check_NoCallbackWhenNil(t *testing.T) {
	rl, store, multi := refreshLoopWithMock(t,
		func(_ context.Context, _ string) (*XSTSResult, error) {
			return &XSTSResult{Token: "new"}, nil
		},
		nil, // callback nil → ne doit pas paniquer
	)
	_ = multi.UpdateOAuthRefreshToken("1234", "rt")
	_ = store.Save(&StoredTokens{
		AccessToken:    "at",
		XSTSXUID:       "1234",
		OAuthExpiresAt: time.Now().Add(time.Hour),
		XSTSToken:      "old",
		XSTSExpiresAt:  time.Now().Add(5 * time.Minute),
	})
	// Ne doit pas paniquer
	rl.check(context.Background())
}

// --- IsXSTSValid avec la marge réelle 20min ---

func TestIsXSTSValid_Margin20min_BoundaryBefore(t *testing.T) {
	// Token avec 21min restants, marge 20min → encore valide (juste au-dessus)
	s := &StoredTokens{XSTSToken: "tok", XSTSExpiresAt: time.Now().Add(21 * time.Minute)}
	if !s.IsXSTSValid(20 * time.Minute) {
		t.Error("21min restants avec marge 20min → doit être valide")
	}
}

func TestIsXSTSValid_Margin20min_BoundaryAfter(t *testing.T) {
	// Token avec 15min restants, marge 20min → expiré (sous la marge → refresh déclenché)
	s := &StoredTokens{XSTSToken: "tok", XSTSExpiresAt: time.Now().Add(15 * time.Minute)}
	if s.IsXSTSValid(20 * time.Minute) {
		t.Error("15min restants avec marge 20min → doit être invalide (refresh requis)")
	}
}

func TestTokenStore_Path(t *testing.T) {
	s := NewTokenStore("/some/path/tokens.json")
	if s.Path() != "/some/path/tokens.json" {
		t.Errorf("Path() = %q", s.Path())
	}
}
