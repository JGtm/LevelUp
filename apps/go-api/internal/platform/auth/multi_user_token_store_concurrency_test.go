// Package auth — multi_user_token_store_concurrency_test.go : tests de
// concurrence et corner cases du store (race conditions, corruption, path
// traversal). Sans dépendance cgo.
package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ─── Concurrence ──────────────────────────────────────────────────────────

func TestMultiUserTokenStore_ConcurrentUpdateOAuthRT_SameXUID(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))
	const N = 100
	const xuid = "111"

	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			rt := "rt-" + string(rune('A'+i%26))
			if err := s.UpdateOAuthRefreshToken(xuid, rt); err != nil {
				t.Errorf("Update[%d]: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	// L'état final doit être cohérent (RT non vide, fichier lisible).
	user, err := s.Load(xuid)
	if err != nil {
		t.Fatalf("Load after concurrent writes: %v", err)
	}
	if user.OAuthRefreshToken == "" {
		t.Error("RT vide après 100 writes concurrents")
	}
	if !strings.HasPrefix(user.OAuthRefreshToken, "rt-") {
		t.Errorf("RT format inattendu : %q", user.OAuthRefreshToken)
	}
}

func TestMultiUserTokenStore_ConcurrentUpsertAndUpdate(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))
	const N = 50
	const xuid = "222"

	var wg sync.WaitGroup
	wg.Add(N * 2)

	// N goroutines Upsert (recrée l'entrée complète)
	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			_ = s.Upsert(&UserTokens{
				XUID:      xuid,
				Gamertag:  "Alice",
				XSTSToken: "xsts-" + string(rune('A'+i%26)),
			})
		}(i)
	}

	// N goroutines UpdateOAuthRefreshToken (préserve XSTS, change RT)
	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			_ = s.UpdateOAuthRefreshToken(xuid, "rt-"+string(rune('a'+i%26)))
		}(i)
	}

	wg.Wait()

	// Vérifier que l'entrée existe et est cohérente (pas de corruption JSON).
	user, err := s.Load(xuid)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if user.XUID != xuid {
		t.Errorf("XUID corrompu : %q", user.XUID)
	}
	// Gamertag peut être vide selon l'ordre (UpdateOAuth ne le set pas), c'est OK
	if user.Gamertag != "" && user.Gamertag != "Alice" {
		t.Errorf("Gamertag inattendu : %q", user.Gamertag)
	}
}

func TestMultiUserTokenStore_ConcurrentLoadDuringUpsert(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))
	const N = 50
	const xuid = "333"

	// Seed une entrée pour avoir quelque chose à lire
	if err := s.Upsert(&UserTokens{XUID: xuid, Gamertag: "Bob", XSTSToken: "initial"}); err != nil {
		t.Fatal(err)
	}

	var readErrors atomic.Int64
	var wg sync.WaitGroup
	wg.Add(N * 2)

	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			_ = s.Upsert(&UserTokens{XUID: xuid, Gamertag: "Bob", XSTSToken: "v" + string(rune('A'+i%26))})
		}(i)
	}

	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			if _, err := s.Load(xuid); err != nil {
				readErrors.Add(1)
			}
		}()
	}

	wg.Wait()

	if readErrors.Load() != 0 {
		t.Errorf("Load errors during concurrent Upsert: %d (devrait être 0, RWMutex protège)", readErrors.Load())
	}
}

func TestMultiUserTokenStore_ConcurrentLoadByGamertagAndUpsert(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))
	const N = 30

	// Seed 3 joueurs
	for i, xuid := range []string{"111", "222", "333"} {
		if err := s.Upsert(&UserTokens{
			XUID:     xuid,
			Gamertag: "Player" + string(rune('A'+i)),
		}); err != nil {
			t.Fatal(err)
		}
	}

	var wg sync.WaitGroup
	wg.Add(N * 2)

	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			_ = s.Upsert(&UserTokens{
				XUID:      "111",
				Gamertag:  "PlayerA",
				XSTSToken: "tok-" + string(rune('A'+i%26)),
			})
		}(i)
	}

	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			_, _ = s.LoadByGamertag("PlayerA")
		}()
	}

	wg.Wait()

	// Vérification finale : entrée toujours cohérente
	user, err := s.LoadByGamertag("PlayerA")
	if err != nil {
		t.Fatalf("LoadByGamertag final: %v", err)
	}
	if user.XUID != "111" {
		t.Errorf("XUID corrompu : %q", user.XUID)
	}
}

// ─── Corruption fichier ───────────────────────────────────────────────────

func TestMultiUserTokenStore_CorruptedJSONFile_LoadReturnsError(t *testing.T) {
	dir := tempTokenDir(t)
	s := NewMultiUserTokenStore(dir)

	// Créer un fichier 999.json avec du JSON invalide
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "999.json"), []byte("{not valid json"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := s.Load("999")
	if err == nil {
		t.Error("Load JSON corrompu → erreur attendue")
	}
}

func TestMultiUserTokenStore_CorruptedJSONFile_LoadAllSkipsAndWarns(t *testing.T) {
	dir := tempTokenDir(t)
	s := NewMultiUserTokenStore(dir)

	// Seed une entrée valide
	if err := s.Upsert(&UserTokens{XUID: "111", Gamertag: "Alice", XSTSToken: "tok"}); err != nil {
		t.Fatal(err)
	}

	// Ajouter un fichier corrompu
	if err := os.WriteFile(filepath.Join(dir, "999.json"), []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}

	all, err := s.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll devrait skip les fichiers corrompus sans erreur globale, got %v", err)
	}
	if _, ok := all["111"]; !ok {
		t.Error("111 devrait être chargé malgré la corruption de 999")
	}
	if _, ok := all["999"]; ok {
		t.Error("999 ne devrait pas être dans le résultat (corrompu)")
	}
}

func TestMultiUserTokenStore_OrphanTmpFile_IgnoredByLoadAll(t *testing.T) {
	dir := tempTokenDir(t)
	s := NewMultiUserTokenStore(dir)

	if err := s.Upsert(&UserTokens{XUID: "111", Gamertag: "Alice", XSTSToken: "tok"}); err != nil {
		t.Fatal(err)
	}

	// Créer un .tmp orphelin (simule crash mid-write)
	if err := os.WriteFile(filepath.Join(dir, "999.json.tmp"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	all, err := s.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if _, ok := all["999"]; ok {
		t.Error("Fichier .tmp ne devrait pas être dans LoadAll")
	}
	if _, ok := all["999.json"]; ok {
		t.Error("Fichier .tmp avec nom contenant .json ne doit pas matcher")
	}
}

// ─── Path traversal & xuid validation ─────────────────────────────────────

func TestMultiUserTokenStore_RejectsPathTraversal(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))

	badXUIDs := []string{
		"../escape",
		"../../etc/passwd",
		"sub/dir/x",
		"\\windows\\path",
		".dotfile",
		"file.json",
		"file.txt",
		"foo bar",  // espace
		"123;456",  // semicolon
		"a@b",      // @
		"unicodeé", // accent (rejeté car pas dans [0-9-_])
	}

	for _, xuid := range badXUIDs {
		t.Run(xuid, func(t *testing.T) {
			if err := s.Upsert(&UserTokens{XUID: xuid, Gamertag: "x"}); err == nil {
				t.Errorf("Upsert xuid=%q devrait être refusé", xuid)
			}
			if _, err := s.Load(xuid); err == nil {
				t.Errorf("Load xuid=%q devrait être refusé", xuid)
			}
			if err := s.UpdateOAuthRefreshToken(xuid, "rt"); err == nil {
				t.Errorf("UpdateOAuth xuid=%q devrait être refusé", xuid)
			}
		})
	}
}

func TestMultiUserTokenStore_AcceptsValidXUIDFormats(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))

	// xuidIsSafe accepte uniquement [0-9_-] — pas de lettres.
	goodXUIDs := []string{
		"123",
		"2533274858283686", // format Xbox Live typique
		"0",
		"9999999999999999",
		"123-456-789",
		"123_456",
		"_____",
		"-----",
	}

	for _, xuid := range goodXUIDs {
		t.Run(xuid, func(t *testing.T) {
			if err := s.Upsert(&UserTokens{XUID: xuid, Gamertag: "x"}); err != nil {
				t.Errorf("Upsert xuid=%q devrait passer : %v", xuid, err)
			}
		})
	}
}

func TestMultiUserTokenStore_RejectsLettersInXUID(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))

	// Confirme explicitement que les lettres sont rejetées (xuids Xbox = pure digits)
	letterXUIDs := []string{
		"abc",
		"123abc",
		"X12345",
		"player1",
	}

	for _, xuid := range letterXUIDs {
		t.Run(xuid, func(t *testing.T) {
			if err := s.Upsert(&UserTokens{XUID: xuid, Gamertag: "x"}); err == nil {
				t.Errorf("Upsert xuid=%q devrait être refusé (contient des lettres)", xuid)
			}
		})
	}
}

// ─── Permissions filesystem (POSIX only) ──────────────────────────────────

func TestMultiUserTokenStore_FilePermissions_StrictMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permissions POSIX non applicables sur Windows")
	}
	dir := tempTokenDir(t)
	s := NewMultiUserTokenStore(dir)
	if err := s.Upsert(&UserTokens{XUID: "111", Gamertag: "alice", XSTSToken: "tok"}); err != nil {
		t.Fatal(err)
	}

	// Verify dir perms == 0700 (no group/other access)
	infoDir, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if mode := infoDir.Mode().Perm(); mode&0o077 != 0 {
		t.Errorf("dir perms = %#o, want strict 0700 (no group/other bits)", mode)
	}

	// Verify file perms == 0600
	infoFile, err := os.Stat(filepath.Join(dir, "111.json"))
	if err != nil {
		t.Fatal(err)
	}
	if mode := infoFile.Mode().Perm(); mode&0o077 != 0 {
		t.Errorf("file perms = %#o, want strict 0600", mode)
	}
}

// ─── Roundtrip serialization ──────────────────────────────────────────────

func TestMultiUserTokenStore_AllFieldsRoundtrip(t *testing.T) {
	s := NewMultiUserTokenStore(tempTokenDir(t))

	original := &UserTokens{
		XUID:              "111",
		Gamertag:          "Madina97294",
		XSTSToken:         "xsts-token-value",
		XSTSUserHash:      "user-hash-abc",
		XSTSExpiresAt:     time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
		AccessToken:       "access-token",
		OAuthExpiresAt:    time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC),
		OAuthRefreshToken: "rt-value",
	}
	if err := s.Upsert(original); err != nil {
		t.Fatal(err)
	}

	loaded, err := s.Load("111")
	if err != nil {
		t.Fatal(err)
	}

	if loaded.XUID != original.XUID {
		t.Errorf("XUID = %q", loaded.XUID)
	}
	if loaded.Gamertag != original.Gamertag {
		t.Errorf("Gamertag = %q", loaded.Gamertag)
	}
	if loaded.XSTSToken != original.XSTSToken {
		t.Errorf("XSTSToken = %q", loaded.XSTSToken)
	}
	if loaded.XSTSUserHash != original.XSTSUserHash {
		t.Errorf("XSTSUserHash = %q", loaded.XSTSUserHash)
	}
	if !loaded.XSTSExpiresAt.Equal(original.XSTSExpiresAt) {
		t.Errorf("XSTSExpiresAt = %v", loaded.XSTSExpiresAt)
	}
	if loaded.AccessToken != original.AccessToken {
		t.Errorf("AccessToken = %q", loaded.AccessToken)
	}
	if !loaded.OAuthExpiresAt.Equal(original.OAuthExpiresAt) {
		t.Errorf("OAuthExpiresAt = %v", loaded.OAuthExpiresAt)
	}
	if loaded.OAuthRefreshToken != original.OAuthRefreshToken {
		t.Errorf("OAuthRefreshToken = %q", loaded.OAuthRefreshToken)
	}
}

func TestMultiUserTokenStore_JSONStructure_HasOAuthRefreshTokenField(t *testing.T) {
	// Vérifie que le marshalling JSON inclut bien le nouveau champ oauth_refresh_token
	// (régression test : si on retire `json:"oauth_refresh_token,omitempty"` par
	// accident, ce test détecte).
	dir := tempTokenDir(t)
	s := NewMultiUserTokenStore(dir)

	if err := s.UpdateOAuthRefreshToken("111", "rt-value"); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "111.json"))
	if err != nil {
		t.Fatal(err)
	}

	var asMap map[string]any
	if err := json.Unmarshal(raw, &asMap); err != nil {
		t.Fatal(err)
	}

	rt, ok := asMap["oauth_refresh_token"].(string)
	if !ok {
		t.Fatal("clé oauth_refresh_token absente du JSON sur disque")
	}
	if rt != "rt-value" {
		t.Errorf("oauth_refresh_token = %q, want rt-value", rt)
	}
}
