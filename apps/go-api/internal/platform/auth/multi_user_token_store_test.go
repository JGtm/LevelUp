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

// TestUpsert_PreservesRefreshTokenOnPartialWrite : un Upsert PARTIEL (XSTS/access
// seulement, RT vide — comme le mirror ou le link AddPlayer) ne doit PAS
// effacer le refresh_token déjà persisté. Régression incident
// 2026-06-13/14 : RT e1cb35ab frais écrasé à vide → migration refill RT mort →
// AADSTS70000 en boucle (la reconnexion ne tenait jamais).
func TestUpsert_PreservesRefreshTokenOnPartialWrite(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))

	// 1) Semer un RT frais (comme le callback SSO).
	if err := s.Upsert(&UserTokens{
		XUID: "111", Gamertag: "Alice",
		OAuthRefreshToken: "rt_frais",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// 2) Upsert PARTIEL : XSTS/access seulement, RT vide.
	if err := s.Upsert(&UserTokens{
		XUID: "111", Gamertag: "Alice",
		XSTSToken: "xsts_new", AccessToken: "at_new",
	}); err != nil {
		t.Fatalf("upsert partiel: %v", err)
	}

	// 3) RT PRÉSERVÉ, XSTS mis à jour.
	got, err := s.Load("111")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.OAuthRefreshToken != "rt_frais" {
		t.Errorf("RT effacé = %q, want rt_frais (préservé)", got.OAuthRefreshToken)
	}
	if got.XSTSToken != "xsts_new" {
		t.Errorf("XSTSToken = %q, want xsts_new (mis à jour)", got.XSTSToken)
	}
}

func TestMultiUserTokenStore_ReauthMarkClear(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))

	// Absent → false, et Clear est un no-op.
	if s.IsReauthRequired("111") {
		t.Error("absent doit donner false")
	}
	if err := s.ClearReauthRequired("111"); err != nil {
		t.Errorf("Clear sur absent: %v", err)
	}

	// Mark sur un xuid sans entrée préalable → crée l'entrée marquée.
	newly, err := s.MarkReauthRequired("111", "Alice")
	if err != nil {
		t.Fatalf("Mark: %v", err)
	}
	if !newly {
		t.Error("première Mark devrait retourner newlyMarked=true")
	}
	if !s.IsReauthRequired("111") {
		t.Fatal("après Mark, IsReauthRequired devrait être true")
	}
	got, _ := s.Load("111")
	if got.Gamertag != "Alice" || got.ReauthDetectedAt.IsZero() {
		t.Errorf("Mark devrait poser gamertag + ReauthDetectedAt, got %+v", got)
	}

	// Mark idempotent : ne réécrit pas ReauthDetectedAt.
	first := got.ReauthDetectedAt
	newlyAgain, _ := s.MarkReauthRequired("111", "Bob")
	if newlyAgain {
		t.Error("Mark répété ne doit pas retourner newlyMarked=true")
	}
	got2, _ := s.Load("111")
	if !got2.ReauthDetectedAt.Equal(first) {
		t.Error("Mark répété ne doit pas changer ReauthDetectedAt")
	}
	if got2.Gamertag != "Alice" {
		t.Error("Mark répété ne doit pas écraser un gamertag existant")
	}

	// Clear → false + ReauthDetectedAt remis à zéro.
	if err := s.ClearReauthRequired("111"); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if s.IsReauthRequired("111") {
		t.Error("après Clear, IsReauthRequired devrait être false")
	}
	cleared, _ := s.Load("111")
	if !cleared.ReauthDetectedAt.IsZero() {
		t.Error("Clear devrait remettre ReauthDetectedAt à zéro")
	}
}

func TestMultiUserTokenStore_AuthErrorRecordClear(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))

	// Clear sur absent : no-op sans erreur.
	if err := s.ClearAuthError("111"); err != nil {
		t.Errorf("Clear sur absent: %v", err)
	}

	// Record sans entrée préalable → crée l'entrée avec classe + message + date.
	if err := s.RecordAuthError("111", "Alice", "config", "AADSTS90023: refused"); err != nil {
		t.Fatalf("Record: %v", err)
	}
	got, err := s.Load("111")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.LastAuthErrorClass != "config" || got.LastAuthError != "AADSTS90023: refused" {
		t.Errorf("champs erreur: %+v", got)
	}
	if got.LastAuthErrorAt.IsZero() {
		t.Error("Record devrait poser LastAuthErrorAt")
	}
	if got.Gamertag != "Alice" {
		t.Error("Record devrait compléter le gamertag vide")
	}

	// Record préserve les autres champs (RT existant).
	if err := s.UpdateOAuthRefreshToken("111", "rt-1"); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordAuthError("111", "Bob", "revoked", "invalid_grant"); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Load("111")
	if got.OAuthRefreshToken != "rt-1" {
		t.Error("Record ne doit pas écraser le RT")
	}
	if got.Gamertag != "Alice" {
		t.Error("Record ne doit pas écraser un gamertag existant")
	}
	if got.LastAuthErrorClass != "revoked" {
		t.Errorf("classe mise à jour attendue, got %q", got.LastAuthErrorClass)
	}

	// Clear → champs vidés, RT intact.
	if err := s.ClearAuthError("111"); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	got, _ = s.Load("111")
	if got.LastAuthErrorClass != "" || got.LastAuthError != "" || !got.LastAuthErrorAt.IsZero() {
		t.Errorf("Clear devrait vider les champs erreur: %+v", got)
	}
	if got.OAuthRefreshToken != "rt-1" {
		t.Error("Clear ne doit pas toucher le RT")
	}
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

func TestMultiUserTokenStore_UpdateOAuthRefreshToken_PreservesOtherFields(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))

	original := &UserTokens{
		XUID:          "2535471234567890",
		Gamertag:      "Madina97294",
		XSTSToken:     "xsts-original",
		XSTSUserHash:  "hash-original",
		XSTSExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if err := s.Upsert(original); err != nil {
		t.Fatalf("Upsert original: %v", err)
	}
	createdAt, _ := s.Load("2535471234567890")

	if err := s.UpdateOAuthRefreshToken("2535471234567890", "rt-rotated-v1"); err != nil {
		t.Fatalf("UpdateOAuthRefreshToken: %v", err)
	}

	loaded, err := s.Load("2535471234567890")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.OAuthRefreshToken != "rt-rotated-v1" {
		t.Errorf("OAuthRefreshToken = %q, want rt-rotated-v1", loaded.OAuthRefreshToken)
	}
	if loaded.Gamertag != "Madina97294" {
		t.Errorf("Gamertag écrasé : %q", loaded.Gamertag)
	}
	if loaded.XSTSToken != "xsts-original" {
		t.Errorf("XSTSToken écrasé : %q", loaded.XSTSToken)
	}
	if !loaded.CreatedAt.Equal(createdAt.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v (préservé)", loaded.CreatedAt, createdAt.CreatedAt)
	}
}

func TestMultiUserTokenStore_UpdateOAuthRefreshToken_CreatesIfAbsent(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))

	if err := s.UpdateOAuthRefreshToken("111", "rt-first"); err != nil {
		t.Fatalf("UpdateOAuthRefreshToken sur xuid absent: %v", err)
	}

	loaded, err := s.Load("111")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.XUID != "111" {
		t.Errorf("XUID = %q, want 111", loaded.XUID)
	}
	if loaded.OAuthRefreshToken != "rt-first" {
		t.Errorf("OAuthRefreshToken = %q, want rt-first", loaded.OAuthRefreshToken)
	}
	if loaded.CreatedAt.IsZero() {
		t.Error("CreatedAt devrait être défini")
	}
}

func TestMultiUserTokenStore_UpdateOAuthRefreshToken_RejectsEmpty(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))

	if err := s.UpdateOAuthRefreshToken("111", ""); err == nil {
		t.Error("UpdateOAuthRefreshToken avec rt vide devrait être refusé")
	}
	if err := s.UpdateOAuthRefreshToken("", "rt"); err == nil {
		t.Error("UpdateOAuthRefreshToken avec xuid vide devrait être refusé")
	}
	if err := s.UpdateOAuthRefreshToken("../escape", "rt"); err == nil {
		t.Error("UpdateOAuthRefreshToken avec xuid unsafe devrait être refusé")
	}
}

func TestMultiUserTokenStore_UpdateOAuthRefreshToken_Idempotent(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))

	if err := s.UpdateOAuthRefreshToken("111", "rt-v1"); err != nil {
		t.Fatalf("UpdateOAuthRefreshToken v1: %v", err)
	}
	if err := s.UpdateOAuthRefreshToken("111", "rt-v1"); err != nil {
		t.Fatalf("UpdateOAuthRefreshToken v1 bis: %v", err)
	}
	loaded, _ := s.Load("111")
	if loaded.OAuthRefreshToken != "rt-v1" {
		t.Errorf("OAuthRefreshToken après double update = %q, want rt-v1", loaded.OAuthRefreshToken)
	}
}

func TestMultiUserTokenStore_LoadByGamertag(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))

	_ = s.Upsert(&UserTokens{XUID: "111", Gamertag: "Madina97294", XSTSToken: "tok-madina"})
	_ = s.Upsert(&UserTokens{XUID: "222", Gamertag: "Chocoboflor", XSTSToken: "tok-choco"})

	// Match exact
	loaded, err := s.LoadByGamertag("Madina97294")
	if err != nil {
		t.Fatalf("LoadByGamertag exact: %v", err)
	}
	if loaded.XUID != "111" {
		t.Errorf("XUID = %q, want 111", loaded.XUID)
	}

	// Case insensitive
	loaded, err = s.LoadByGamertag("madina97294")
	if err != nil {
		t.Fatalf("LoadByGamertag lower: %v", err)
	}
	if loaded.XUID != "111" {
		t.Errorf("case-insensitive: XUID = %q, want 111", loaded.XUID)
	}

	// Trim whitespace
	loaded, err = s.LoadByGamertag("  Chocoboflor  ")
	if err != nil {
		t.Fatalf("LoadByGamertag trim: %v", err)
	}
	if loaded.XUID != "222" {
		t.Errorf("trim: XUID = %q, want 222", loaded.XUID)
	}
}

func TestMultiUserTokenStore_LoadByGamertag_NotFound(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))
	_ = s.Upsert(&UserTokens{XUID: "111", Gamertag: "alice", XSTSToken: "tok"})

	_, err := s.LoadByGamertag("bob")
	if !errors.Is(err, ErrUserTokensNotFound) {
		t.Errorf("err = %v, want ErrUserTokensNotFound", err)
	}

	_, err = s.LoadByGamertag("")
	if err == nil {
		t.Error("LoadByGamertag avec gamertag vide devrait être refusé")
	}
}

func TestMultiUserTokenStore_LoadByGamertag_EmptyDir(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))

	_, err := s.LoadByGamertag("alice")
	if !errors.Is(err, ErrUserTokensNotFound) {
		t.Errorf("err = %v, want ErrUserTokensNotFound", err)
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
