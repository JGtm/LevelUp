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

func TestGetByXUID_FoundAndMissing(t *testing.T) {
	s := NewStore(tempStorePath(t))
	_, _ = s.Create(testUser, testPass, domain.RoleUser)
	_ = s.LinkIdentity(testUser, "AliceGT", "xuid-alice-123")

	user, err := s.GetByXUID("xuid-alice-123")
	if err != nil {
		t.Fatalf("GetByXUID existing: %v", err)
	}
	if user.Username != testUser {
		t.Errorf("username = %q, want %q", user.Username, testUser)
	}

	_, err = s.GetByXUID("xuid-absent")
	if err != ErrUserNotFound {
		t.Errorf("GetByXUID absent: err = %v, want ErrUserNotFound", err)
	}

	_, err = s.GetByXUID("")
	if err != ErrUserNotFound {
		t.Errorf("GetByXUID empty: err = %v, want ErrUserNotFound", err)
	}
}

func TestCreateFromXbox_Basic(t *testing.T) {
	s := NewStore(tempStorePath(t))

	user, err := s.CreateFromXbox("XboxUser42", "xuid-xbox-42")
	if err != nil {
		t.Fatalf("CreateFromXbox: %v", err)
	}
	if user.Role != domain.RoleUser {
		t.Errorf("role = %q, want user (admin promotion doit passer par /auth/register en first_launch)", user.Role)
	}
	if user.Gamertag != "XboxUser42" {
		t.Errorf("gamertag = %q, want XboxUser42 (original conservé)", user.Gamertag)
	}
	if user.XUID != "xuid-xbox-42" {
		t.Errorf("xuid = %q, want xuid-xbox-42", user.XUID)
	}
	if user.Username != "xboxuser42" {
		t.Errorf("username = %q, want xboxuser42 (slug normalisé)", user.Username)
	}
	if user.PasswordHash != "" {
		t.Errorf("password_hash devrait être vide pour un user Xbox SSO")
	}

	// Doit être retrouvable par XUID.
	found, err := s.GetByXUID("xuid-xbox-42")
	if err != nil {
		t.Fatalf("GetByXUID après CreateFromXbox: %v", err)
	}
	if found.Username != "xboxuser42" {
		t.Errorf("found username = %q, want xboxuser42", found.Username)
	}
}

func TestCreateFromXbox_SlugifyGamertag(t *testing.T) {
	s := NewStore(tempStorePath(t))

	// Gamertag avec espace et casse → slug compact lowercase.
	user, err := s.CreateFromXbox("Mr Banana", "xuid-banana")
	if err != nil {
		t.Fatalf("CreateFromXbox espace: %v", err)
	}
	if user.Username != "mrbanana" {
		t.Errorf("username = %q, want mrbanana", user.Username)
	}
	if user.Gamertag != "Mr Banana" {
		t.Errorf("gamertag = %q, want 'Mr Banana' (original)", user.Gamertag)
	}
}

func TestCreateFromXbox_CollisionFallback(t *testing.T) {
	s := NewStore(tempStorePath(t))

	// Pré-créer un user password avec username Alice (slug "alice").
	_, err := s.Create(testUser, testPass, domain.RoleUser)
	if err != nil {
		t.Fatalf("setup Create: %v", err)
	}

	// CreateFromXbox avec gamertag "Alice" (slug "alice") → collision → fallback "alice_xbox".
	user, err := s.CreateFromXbox("Alice", "xuid-alice-xbox")
	if err != nil {
		t.Fatalf("CreateFromXbox collision: %v", err)
	}
	if user.Username != "alice_xbox" {
		t.Errorf("username = %q, want alice_xbox (suffixe collision)", user.Username)
	}

	// Le user password original existe toujours.
	original, err := s.Get(testUser)
	if err != nil {
		t.Fatalf("Get original: %v", err)
	}
	if original.PasswordHash == "" {
		t.Error("le user password original devrait avoir conservé son hash")
	}
}

func TestCreateFromXbox_DoubleCollisionFails(t *testing.T) {
	s := NewStore(tempStorePath(t))

	// Pré-créer "alice" (password) ET "alice_xbox" (xbox).
	_, _ = s.Create(testUser, testPass, domain.RoleUser)
	_, _ = s.CreateFromXbox("Alice", "xuid-1")

	// Une 3ème tentative collisionne sur les deux slots.
	_, err := s.CreateFromXbox("Alice", "xuid-2")
	if err != ErrUserAlreadyExists {
		t.Errorf("double collision: err = %v, want ErrUserAlreadyExists", err)
	}
}

func TestCreateFromXbox_RequiresXUID(t *testing.T) {
	s := NewStore(tempStorePath(t))

	_, err := s.CreateFromXbox("ValidGamertag", "")
	if err == nil {
		t.Error("CreateFromXbox sans xuid devrait échouer")
	}
}

func TestCreateFromXbox_InvalidGamertag(t *testing.T) {
	s := NewStore(tempStorePath(t))

	// Gamertag entièrement non-alphanum → slug vide → erreur.
	_, err := s.CreateFromXbox("!!!", "xuid-1")
	if err != ErrInvalidUsername {
		t.Errorf("gamertag invalide: err = %v, want ErrInvalidUsername", err)
	}
}

func TestAuthenticateByXUID_TouchesLastLogin(t *testing.T) {
	s := NewStore(tempStorePath(t))
	created, _ := s.CreateFromXbox("XboxAlice", "xuid-alice")
	if created.LastLoginAt != "" {
		t.Fatal("LastLoginAt devrait être vide juste après création")
	}

	user, err := s.AuthenticateByXUID("xuid-alice")
	if err != nil {
		t.Fatalf("AuthenticateByXUID: %v", err)
	}
	if user.LastLoginAt == "" {
		t.Error("AuthenticateByXUID devrait toucher LastLoginAt")
	}

	// Persistance : nouveau Store doit voir LastLoginAt.
	persisted, err := s.GetByXUID("xuid-alice")
	if err != nil {
		t.Fatalf("GetByXUID après auth: %v", err)
	}
	if persisted.LastLoginAt == "" {
		t.Error("LastLoginAt non persisté")
	}
}

func TestAuthenticateByXUID_UnknownXUID(t *testing.T) {
	s := NewStore(tempStorePath(t))

	_, err := s.AuthenticateByXUID("xuid-inexistant")
	if err != ErrUserNotFound {
		t.Errorf("xuid inconnu: err = %v, want ErrUserNotFound", err)
	}

	_, err = s.AuthenticateByXUID("")
	if err != ErrUserNotFound {
		t.Errorf("xuid vide: err = %v, want ErrUserNotFound", err)
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
