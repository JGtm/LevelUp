package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/session"
	"levelup/go-api/internal/platform/userstore"
)

// TestRegister_FirstAdminRaceCreatesOneAdmin : inscriptions simultanées sur une
// instance vide, inscriptions fermées : un seul compte, administrateur ; les autres
// passent par les gardes ordinaires et sont refusés.
func TestRegister_FirstAdminRaceCreatesOneAdmin(t *testing.T) {
	dir := t.TempDir()
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	h := handlers.NewUserAuthHandler(users, userstore.NewInviteStore(filepath.Join(dir, "invites.json"), users), sessStore, "closed").
		WithAuthMode("password")
	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessStore, middleware.SecureCookiePolicy{}))
	h.Mount(r)

	const n = 6
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			postJSON(t, r, "/auth/register", domain.RegisterRequest{
				Username: "first" + string(rune('a'+i)), Password: testUserAuthPass,
			})
		}()
	}
	wg.Wait()

	list, err := users.List()
	if err != nil {
		t.Fatalf("List : %v", err)
	}
	if len(list) != 1 || list[0].Role != domain.RoleAdmin {
		t.Fatalf("comptes après la course = %+v, attendu un seul admin", list)
	}
}

// TestAdmin_DeleteLastAdminIs409 : l'interface d'administration ne supprime ni ne
// rétrograde le dernier administrateur.
func TestAdmin_DeleteLastAdminIs409(t *testing.T) {
	dir := t.TempDir()
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	if _, err := users.Create("boss", testUserAuthPass, domain.RoleAdmin); err != nil {
		t.Fatalf("Create : %v", err)
	}
	h := handlers.NewAdminHandler(users, userstore.NewInviteStore(filepath.Join(dir, "invites.json"), users))
	r := chi.NewRouter()
	r.Route("/admin", func(r chi.Router) { h.Mount(r) })

	if w := serveJSON(r, "DELETE", "/admin/users/boss", ""); w.Code != 409 {
		t.Errorf("suppression du dernier admin : %d, attendu 409 (%s)", w.Code, w.Body.String())
	}
	if w := serveJSON(r, "PATCH", "/admin/users/boss/role", `{"role":"user"}`); w.Code != 409 {
		t.Errorf("rétrogradation du dernier admin : %d, attendu 409 (%s)", w.Code, w.Body.String())
	}
}

func serveJSON(r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
