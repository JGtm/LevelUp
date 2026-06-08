// Package handlers_test — user_auth_test.go : tests cohabitation D3 (mode xbox).
package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/session"
	"levelup/go-api/internal/platform/userstore"
)

const (
	testUserAuthPass = "Yp4kR9wQ"
)

// newUserAuthRouter assemble un routeur de test avec UserAuthHandler dans le mode auth donné.
func newUserAuthRouter(t *testing.T, authMode string) (*chi.Mux, *userstore.Store) {
	t.Helper()
	dir := t.TempDir()
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	invites := userstore.NewInviteStore(filepath.Join(dir, "invites.json"))
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")

	h := handlers.NewUserAuthHandler(users, invites, sessStore, "open").WithAuthMode(authMode)

	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessStore, middleware.SecureCookiePolicy{}))
	r.Post("/auth/login", h.Login)
	r.Post("/auth/register", h.Register)
	return r, users
}

// newLockedUserAuthRouter assemble un routeur avec le verrou « instance fermée » actif.
func newLockedUserAuthRouter(t *testing.T, authMode string) (*chi.Mux, *userstore.Store) {
	t.Helper()
	dir := t.TempDir()
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	invites := userstore.NewInviteStore(filepath.Join(dir, "invites.json"))
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")

	h := handlers.NewUserAuthHandler(users, invites, sessStore, "open").
		WithAuthMode(authMode).
		WithInstanceLock(func() bool { return true })

	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessStore, middleware.SecureCookiePolicy{}))
	r.Post("/auth/register", h.Register)
	return r, users
}

// TestUserAuth_Register_InstanceLockedBlocksNewUsers : instance fermée → register
// refusé (403 instance_locked) dès qu'un utilisateur existe déjà.
func TestUserAuth_Register_InstanceLockedBlocksNewUsers(t *testing.T) {
	r, users := newLockedUserAuthRouter(t, "password")
	_, _ = users.Create("existing", testUserAuthPass, domain.RoleAdmin) // store non vide

	w := postJSON(t, r, "/auth/register", domain.RegisterRequest{
		Username: "newcomer", Password: testUserAuthPass,
	})

	if w.Code != http.StatusForbidden {
		t.Fatalf("register sous lockdown : status = %d, want 403. Body: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("instance_locked")) {
		t.Errorf("code d'erreur attendu instance_locked, body: %s", w.Body.String())
	}
}

// TestUserAuth_Register_InstanceLockedExemptsFirstAdmin : le tout premier admin
// (store vide) reste autorisé même sous lockdown — sinon instance non amorçable.
func TestUserAuth_Register_InstanceLockedExemptsFirstAdmin(t *testing.T) {
	r, _ := newLockedUserAuthRouter(t, "password") // store vide

	w := postJSON(t, r, "/auth/register", domain.RegisterRequest{
		Username: "firstadmin", Password: testUserAuthPass,
	})

	if w.Code != http.StatusCreated {
		t.Fatalf("bootstrap 1er admin sous lockdown : status = %d, want 201. Body: %s", w.Code, w.Body.String())
	}
}

func postJSON(t *testing.T, r http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestUserAuth_XboxMode_LoginAdminAllowed : en mode xbox, un admin peut se logger
// par password (fallback si SSO down).
func TestUserAuth_XboxMode_LoginAdminAllowed(t *testing.T) {
	r, users := newUserAuthRouter(t, "xbox")
	_, _ = users.Create("admin", testUserAuthPass, domain.RoleAdmin)

	w := postJSON(t, r, "/auth/login", domain.LoginRequest{
		Username: "admin",
		Password: testUserAuthPass,
	})

	if w.Code != http.StatusOK {
		t.Fatalf("admin login en mode xbox : status = %d, want 200. Body: %s", w.Code, w.Body.String())
	}
}

// TestUserAuth_XboxMode_LoginWithPasswordAllowed : PR-C — en mode xbox, un user
// NON-admin qui a défini un mot de passe peut désormais se logger par password
// (opt-in). Avant PR-C : 403 password_login_admin_only.
func TestUserAuth_XboxMode_LoginWithPasswordAllowed(t *testing.T) {
	r, users := newUserAuthRouter(t, "xbox")
	_, _ = users.Create("normaluser", testUserAuthPass, domain.RoleUser) // a un mot de passe

	w := postJSON(t, r, "/auth/login", domain.LoginRequest{
		Username: "normaluser",
		Password: testUserAuthPass,
	})

	if w.Code != http.StatusOK {
		t.Fatalf("login password non-admin (avec MDP) en mode xbox : status = %d, want 200. Body: %s", w.Code, w.Body.String())
	}
}

// TestUserAuth_SetPassword_OptIn : un compte SSO (sans mot de passe) en définit un
// via POST /auth/password, puis peut s'authentifier avec.
func TestUserAuth_SetPassword_OptIn(t *testing.T) {
	dir := t.TempDir()
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	invites := userstore.NewInviteStore(filepath.Join(dir, "invites.json"))
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	u, _ := users.CreateFromXbox("Spartan", "xuid-1") // compte SSO sans mot de passe

	h := handlers.NewUserAuthHandler(users, invites, sessStore, "open").WithAuthMode("xbox")

	req := httptest.NewRequest(http.MethodPost, "/auth/password", bytes.NewReader([]byte(`{"password":"NewPass123"}`)))
	req.Header.Set("Content-Type", "application/json")
	sess := &domain.SessionData{Username: &u.Username}
	req = req.WithContext(middleware.InjectSession(req.Context(), sess))
	w := httptest.NewRecorder()
	h.SetPassword(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("SetPassword : status = %d, want 204. Body: %s", w.Code, w.Body.String())
	}
	if _, err := users.Authenticate(u.Username, "NewPass123"); err != nil {
		t.Errorf("Authenticate après SetPassword devrait réussir, got %v", err)
	}
}

// TestUserAuth_SetPassword_NoSession : sans session authentifiée → 401.
func TestUserAuth_SetPassword_NoSession(t *testing.T) {
	dir := t.TempDir()
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	invites := userstore.NewInviteStore(filepath.Join(dir, "invites.json"))
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	h := handlers.NewUserAuthHandler(users, invites, sessStore, "open")

	req := httptest.NewRequest(http.MethodPost, "/auth/password", bytes.NewReader([]byte(`{"password":"NewPass123"}`)))
	sess := &domain.SessionData{} // pas de Username
	req = req.WithContext(middleware.InjectSession(req.Context(), sess))
	w := httptest.NewRecorder()
	h.SetPassword(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("SetPassword sans session : status = %d, want 401", w.Code)
	}
}

// TestUserAuth_PasswordMode_LoginUserAllowed : en mode password, un user normal
// peut se logger normalement (comportement préservé, pas de régression).
func TestUserAuth_PasswordMode_LoginUserAllowed(t *testing.T) {
	r, users := newUserAuthRouter(t, "password")
	_, _ = users.Create("normaluser", testUserAuthPass, domain.RoleUser)

	w := postJSON(t, r, "/auth/login", domain.LoginRequest{
		Username: "normaluser",
		Password: testUserAuthPass,
	})

	if w.Code != http.StatusOK {
		t.Fatalf("user login en mode password : status = %d, want 200. Body: %s", w.Code, w.Body.String())
	}
}

// TestUserAuth_XboxMode_RegisterBootstrapAllowed : en mode xbox, le register password
// est autorisé UNIQUEMENT pour le bootstrap admin initial (users.json vide).
func TestUserAuth_XboxMode_RegisterBootstrapAllowed(t *testing.T) {
	r, _ := newUserAuthRouter(t, "xbox")

	w := postJSON(t, r, "/auth/register", domain.RegisterRequest{
		Username: "firstadmin",
		Password: testUserAuthPass,
	})

	if w.Code != http.StatusCreated {
		t.Fatalf("bootstrap admin en mode xbox : status = %d, want 201. Body: %s", w.Code, w.Body.String())
	}
	var resp domain.RegisterResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Role != domain.RoleAdmin {
		t.Errorf("role = %q, want admin (premier user)", resp.Role)
	}
}

// TestUserAuth_XboxMode_RegisterNonBootstrapBlocked : en mode xbox, register password
// avec users.json non vide est refusé — les autres comptes doivent passer par SSO.
func TestUserAuth_XboxMode_RegisterNonBootstrapBlocked(t *testing.T) {
	r, users := newUserAuthRouter(t, "xbox")
	_, _ = users.Create("existingadmin", testUserAuthPass, domain.RoleAdmin)

	w := postJSON(t, r, "/auth/register", domain.RegisterRequest{
		Username: "newuser",
		Password: testUserAuthPass,
	})

	if w.Code != http.StatusForbidden {
		t.Fatalf("register hors bootstrap en mode xbox : status = %d, want 403. Body: %s", w.Code, w.Body.String())
	}
	var errResp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp["code"] != "register_xbox_mode" {
		t.Errorf("error code = %v, want register_xbox_mode", errResp["code"])
	}
}

// TestUserAuth_PasswordMode_RegisterFlow : en mode password, register fonctionne
// normalement (premier = admin, suivants = user, regMode appliqué).
func TestUserAuth_PasswordMode_RegisterFlow(t *testing.T) {
	r, users := newUserAuthRouter(t, "password")

	// Premier user → admin auto, pas besoin d'invite (regMode "open" en plus).
	w := postJSON(t, r, "/auth/register", domain.RegisterRequest{
		Username: "firstuser",
		Password: testUserAuthPass,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("premier register password : status = %d, want 201", w.Code)
	}

	// User existant.
	if _, err := users.Get("firstuser"); err != nil {
		t.Fatalf("firstuser absent : %v", err)
	}

	// 2e user en regMode "open" → OK.
	w2 := postJSON(t, r, "/auth/register", domain.RegisterRequest{
		Username: "seconduser",
		Password: testUserAuthPass,
	})
	if w2.Code != http.StatusCreated {
		t.Fatalf("2e register password (open) : status = %d, want 201. Body: %s", w2.Code, w2.Body.String())
	}
}
