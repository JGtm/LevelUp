package userstore

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"levelup/go-api/internal/domain"
)

func tempStorePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "users.json")
}

// Credentials de test — valeurs fictives, pas de vrais secrets.
const (
	testPass  = "Xk9mP2vL"
	testPass2 = "Qr7nW4jT"
	testUser  = "Alice"
)

func TestNewStore_IsEmpty(t *testing.T) {
	s := NewStore(tempStorePath(t))
	empty, err := s.IsEmpty()
	if err != nil {
		t.Fatalf("IsEmpty: %v", err)
	}
	if !empty {
		t.Fatal("nouveau store devrait être vide")
	}
}

func TestCreate_And_Authenticate(t *testing.T) {
	s := NewStore(tempStorePath(t))
	user, err := s.Create(testUser, testPass, domain.RoleAdmin)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if user.Username != testUser {
		t.Errorf("username = %q, want Alice", user.Username)
	}
	if user.Role != domain.RoleAdmin {
		t.Errorf("role = %q, want admin", user.Role)
	}

	// Store n'est plus vide.
	empty, _ := s.IsEmpty()
	if empty {
		t.Fatal("store ne devrait plus être vide après Create")
	}

	// Authenticate — succès.
	auth, err := s.Authenticate(testUser, testPass)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if auth.Username != testUser {
		t.Errorf("auth username = %q, want Alice", auth.Username)
	}
}

func TestCreate_DuplicateSlug(t *testing.T) {
	s := NewStore(tempStorePath(t))
	_, _ = s.Create(testUser, testPass, domain.RoleUser)
	_, err := s.Create("alice", testPass2, domain.RoleUser) // même slug
	if err != ErrUserAlreadyExists {
		t.Errorf("err = %v, want ErrUserAlreadyExists", err)
	}
}

func TestCreate_ValidationErrors(t *testing.T) {
	s := NewStore(tempStorePath(t))

	// Username trop court.
	_, err := s.Create("ab", testPass, domain.RoleUser)
	if err != ErrInvalidUsername {
		t.Errorf("short username: err = %v, want ErrInvalidUsername", err)
	}

	// Password trop court.
	_, err = s.Create("ValidUser", "shrt", domain.RoleUser)
	if err != ErrPasswordTooShort {
		t.Errorf("short pass: err = %v, want ErrPasswordTooShort", err)
	}

	// Username invalide (caractères spéciaux).
	_, err = s.Create("user@name", testPass, domain.RoleUser)
	if err != ErrInvalidUsername {
		t.Errorf("invalid chars: err = %v, want ErrInvalidUsername", err)
	}
}

func TestAuthenticate_WrongPassword(t *testing.T) {
	s := NewStore(tempStorePath(t))
	_, _ = s.Create(testUser, testPass, domain.RoleUser)

	_, err := s.Authenticate(testUser, "wrongXk9")
	if err != ErrInvalidCredentials {
		t.Errorf("wrong pass: err = %v, want ErrInvalidCredentials", err)
	}
}

func TestAuthenticate_UnknownUser(t *testing.T) {
	s := NewStore(tempStorePath(t))

	_, err := s.Authenticate("Nobody", testPass)
	if err != ErrInvalidCredentials {
		t.Errorf("unknown user: err = %v, want ErrInvalidCredentials", err)
	}
}

func TestGet_ExistingAndMissing(t *testing.T) {
	s := NewStore(tempStorePath(t))
	_, _ = s.Create(testUser, testPass, domain.RoleUser)

	user, err := s.Get(testUser)
	if err != nil {
		t.Fatalf("Get existing: %v", err)
	}
	if user.Username != testUser {
		t.Errorf("username = %q, want Alice", user.Username)
	}

	// Case-insensitive slug.
	user2, err := s.Get("alice")
	if err != nil {
		t.Fatalf("Get lowercase: %v", err)
	}
	if user2.Username != testUser {
		t.Errorf("username = %q, want Alice", user2.Username)
	}

	_, err = s.Get("Bob")
	if err != ErrUserNotFound {
		t.Errorf("Get missing: err = %v, want ErrUserNotFound", err)
	}
}

func TestList(t *testing.T) {
	s := NewStore(tempStorePath(t))
	_, _ = s.Create(testUser, testPass, domain.RoleAdmin)
	_, _ = s.Create("Bob", testPass2, domain.RoleUser)

	list, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want 2", len(list))
	}
}

func TestDelete(t *testing.T) {
	s := NewStore(tempStorePath(t))
	_, _ = s.Create(testUser, testPass, domain.RoleUser)

	if err := s.Delete(testUser); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	empty, _ := s.IsEmpty()
	if !empty {
		t.Fatal("store devrait être vide après Delete")
	}

	// Delete utilisateur inexistant.
	err := s.Delete(testUser)
	if err != ErrUserNotFound {
		t.Errorf("Delete missing: err = %v, want ErrUserNotFound", err)
	}
}

func TestSetRole(t *testing.T) {
	s := NewStore(tempStorePath(t))
	_, _ = s.Create(testUser, testPass, domain.RoleUser)

	if err := s.SetRole(testUser, domain.RoleAdmin); err != nil {
		t.Fatalf("SetRole: %v", err)
	}
	user, _ := s.Get(testUser)
	if user.Role != domain.RoleAdmin {
		t.Errorf("role = %q, want admin", user.Role)
	}

	// SetRole user inexistant.
	err := s.SetRole("Nobody", domain.RoleAdmin)
	if err != ErrUserNotFound {
		t.Errorf("SetRole missing: err = %v, want ErrUserNotFound", err)
	}
}

func TestResetPassword(t *testing.T) {
	s := NewStore(tempStorePath(t))
	_, _ = s.Create(testUser, testPass, domain.RoleUser)

	if err := s.ResetPassword(testUser, testPass2); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}

	// Ancien mot de passe ne fonctionne plus.
	_, err := s.Authenticate(testUser, testPass)
	if err != ErrInvalidCredentials {
		t.Errorf("old pass should fail: err = %v", err)
	}

	// Nouveau mot de passe fonctionne.
	_, err = s.Authenticate(testUser, testPass2)
	if err != nil {
		t.Errorf("new pass should work: err = %v", err)
	}

	// Password trop court.
	err = s.ResetPassword(testUser, "shrt")
	if err != ErrPasswordTooShort {
		t.Errorf("short pass: err = %v, want ErrPasswordTooShort", err)
	}
}

func TestLinkIdentity(t *testing.T) {
	s := NewStore(tempStorePath(t))
	_, _ = s.Create(testUser, testPass, domain.RoleUser)

	if err := s.LinkIdentity(testUser, "AliceGT", "xuid123"); err != nil {
		t.Fatalf("LinkIdentity: %v", err)
	}
	user, _ := s.Get(testUser)
	if user.Gamertag != "AliceGT" {
		t.Errorf("gamertag = %q, want AliceGT", user.Gamertag)
	}
	if user.XUID != "xuid123" {
		t.Errorf("xuid = %q, want xuid123", user.XUID)
	}

	// LinkIdentity user inexistant.
	err := s.LinkIdentity("Nobody", "GT", "xuid")
	if err != ErrUserNotFound {
		t.Errorf("LinkIdentity missing: err = %v, want ErrUserNotFound", err)
	}
}

func TestUpdateLastLogin(t *testing.T) {
	s := NewStore(tempStorePath(t))
	_, _ = s.Create(testUser, testPass, domain.RoleUser)

	if err := s.UpdateLastLogin(testUser); err != nil {
		t.Fatalf("UpdateLastLogin: %v", err)
	}
	user, _ := s.Get(testUser)
	if user.LastLoginAt == "" {
		t.Error("LastLoginAt devrait être défini")
	}
}

func TestPersistence(t *testing.T) {
	path := tempStorePath(t)
	s1 := NewStore(path)
	_, _ = s1.Create(testUser, testPass, domain.RoleAdmin)

	// Nouvel objet Store sur le même fichier.
	s2 := NewStore(path)
	user, err := s2.Get(testUser)
	if err != nil {
		t.Fatalf("persistence Get: %v", err)
	}
	if user.Username != testUser {
		t.Errorf("persistence username = %q, want Alice", user.Username)
	}
}

func TestFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permissions POSIX non applicables sur Windows")
	}
	path := tempStorePath(t)
	s := NewStore(path)
	_, _ = s.Create(testUser, testPass, domain.RoleUser)

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm()&0o002 != 0 {
		t.Errorf("fichier world-writable : %o", info.Mode().Perm())
	}
}
