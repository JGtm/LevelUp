// Package auth — multi_user_token_store_test.go : tests MultiUserTokenStore.
package auth

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func tempTokenDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "watcher_tokens")
}

func TestMultiUserTokenStore_UpsertAndLoad(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))

	tokens := &UserTokens{
		XUID:          "2535471234567890",
		Gamertag:      "Spartan42",
		XSTSToken:     "xsts-token-xyz",
		XSTSUserHash:  "user-hash-abc",
		XSTSExpiresAt: time.Now().Add(55 * time.Minute),
	}
	if err := s.Upsert(tokens); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	loaded, err := s.Load("2535471234567890")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Gamertag != "Spartan42" {
		t.Errorf("gamertag = %q, want Spartan42", loaded.Gamertag)
	}
	if loaded.XSTSToken != "xsts-token-xyz" {
		t.Errorf("XSTSToken = %q", loaded.XSTSToken)
	}
	if loaded.CreatedAt.IsZero() {
		t.Error("CreatedAt devrait être défini après Upsert")
	}
	if loaded.UpdatedAt.IsZero() {
		t.Error("UpdatedAt devrait être défini après Upsert")
	}
}

func TestMultiUserTokenStore_Load_NotFound(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))

	_, err := s.Load("123456789")
	if !errors.Is(err, ErrUserTokensNotFound) {
		t.Errorf("err = %v, want ErrUserTokensNotFound", err)
	}
}

func TestMultiUserTokenStore_Upsert_PreservesCreatedAtOnUpdate(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))

	xuid := "111"
	first := &UserTokens{XUID: xuid, Gamertag: "Alice", XSTSToken: "v1"}
	if err := s.Upsert(first); err != nil {
		t.Fatalf("Upsert v1: %v", err)
	}
	loaded1, _ := s.Load(xuid)
	created := loaded1.CreatedAt

	time.Sleep(5 * time.Millisecond)
	second := &UserTokens{XUID: xuid, Gamertag: "Alice", XSTSToken: "v2"}
	if err := s.Upsert(second); err != nil {
		t.Fatalf("Upsert v2: %v", err)
	}
	loaded2, _ := s.Load(xuid)

	if !loaded2.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt = %v, want %v (préservé sur update)", loaded2.CreatedAt, created)
	}
	if !loaded2.UpdatedAt.After(loaded2.CreatedAt) {
		t.Errorf("UpdatedAt (%v) devrait être > CreatedAt (%v)", loaded2.UpdatedAt, loaded2.CreatedAt)
	}
	if loaded2.XSTSToken != "v2" {
		t.Errorf("XSTSToken = %q, want v2", loaded2.XSTSToken)
	}
}

func TestMultiUserTokenStore_LoadAll(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))

	xuids := []string{"111", "222", "333"}
	for _, xuid := range xuids {
		_ = s.Upsert(&UserTokens{XUID: xuid, Gamertag: "user-" + xuid, XSTSToken: "tok"})
	}

	all, err := s.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("len(all) = %d, want 3", len(all))
	}
	for _, xuid := range xuids {
		if _, ok := all[xuid]; !ok {
			t.Errorf("xuid %q absent du map", xuid)
		}
	}
}

func TestMultiUserTokenStore_LoadAll_EmptyDir(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))
	all, err := s.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("len(all) = %d, want 0 (dir absent)", len(all))
	}
}

func TestMultiUserTokenStore_LoadAll_IgnoresInvalidFiles(t *testing.T) {
	dir := tempTokenDir(t)
	s := NewMultiUserTokenStore(dir)
	_ = s.Upsert(&UserTokens{XUID: "111", Gamertag: "alice", XSTSToken: "tok"})

	// Ajouter un fichier au nom invalide (caractères non autorisés).
	_ = os.WriteFile(filepath.Join(dir, "../escape.json"), []byte("{}"), 0o600)
	// Et un fichier .json mais avec contenu invalide.
	_ = os.WriteFile(filepath.Join(dir, "999.json"), []byte("not json"), 0o600)

	all, err := s.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if _, ok := all["111"]; !ok {
		t.Error("xuid valide '111' devrait être chargé")
	}
	if _, ok := all["999"]; ok {
		t.Error("fichier au JSON invalide ne devrait pas apparaître dans LoadAll")
	}
}

func TestMultiUserTokenStore_Remove(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))
	_ = s.Upsert(&UserTokens{XUID: "111", Gamertag: "alice", XSTSToken: "tok"})

	if err := s.Remove("111"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	_, err := s.Load("111")
	if !errors.Is(err, ErrUserTokensNotFound) {
		t.Errorf("après Remove, Load err = %v, want ErrUserTokensNotFound", err)
	}

	// Remove inexistant (xuid valide mais absent) = no-op.
	if err := s.Remove("99999999"); err != nil {
		t.Errorf("Remove inexistant devrait être no-op, got %v", err)
	}
}

func TestMultiUserTokenStore_RejectsUnsafeXUID(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))

	for _, bad := range []string{"", "../escape", "abc/def", "x.json", ".secret", "foo bar"} {
		err := s.Upsert(&UserTokens{XUID: bad, Gamertag: "x"})
		if err == nil {
			t.Errorf("Upsert(xuid=%q) devrait être refusé", bad)
		}
		_, err = s.Load(bad)
		if err == nil {
			t.Errorf("Load(xuid=%q) devrait être refusé", bad)
		}
	}
}

func TestMultiUserTokenStore_AuthHeader(t *testing.T) {
	tokens := &UserTokens{XSTSToken: "tok", XSTSUserHash: "hash"}
	got := tokens.AuthHeader()
	want := "XBL3.0 x=hash;tok"
	if got != want {
		t.Errorf("AuthHeader = %q, want %q", got, want)
	}

	empty := &UserTokens{}
	if empty.AuthHeader() != "" {
		t.Errorf("AuthHeader vide attendu si XSTS manquant")
	}
}

func TestMultiUserTokenStore_FilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permissions POSIX non applicables sur Windows")
	}
	dir := tempTokenDir(t)
	s := NewMultiUserTokenStore(dir)
	_ = s.Upsert(&UserTokens{XUID: "111", Gamertag: "alice", XSTSToken: "tok"})

	infoDir, _ := os.Stat(dir)
	if infoDir.Mode().Perm()&0o077 != 0 {
		t.Errorf("dir perms = %o, want 0700 strict", infoDir.Mode().Perm())
	}

	infoFile, _ := os.Stat(filepath.Join(dir, "111.json"))
	if infoFile.Mode().Perm()&0o077 != 0 {
		t.Errorf("file perms = %o, want 0600 strict", infoFile.Mode().Perm())
	}
}
