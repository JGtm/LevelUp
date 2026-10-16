// Package handlers_test — invite_issuer_test.go : une invitation n'existe que par
// un compte réel. Créer un groupe ou émettre une invitation exige un compte du
// magasin d'utilisateurs ; une invitation dont l'émetteur n'existe plus ne vaut
// rien ; un code ne crée qu'un seul compte, même sous requêtes concurrentes.
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

// TestGroups_CreateRequiresExistingAccount : une session qui porte une identité Halo
// sans compte (utilisateur synthétisé) ne crée pas de groupe.
func TestGroups_CreateRequiresExistingAccount(t *testing.T) {
	r, sessStore, _, _ := newGroupsRouter(t)
	cookie := authCookie(t, sessStore, "ghost-x", "Ghost")

	req := httptest.NewRequest(http.MethodPost, "/groups", strings.NewReader(`{"name":"Fantome"}`))
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("création de groupe sans compte : %d, attendu 403 (corps %s)", w.Code, w.Body.String())
	}
}

// TestGroups_InviteRequiresExistingAccount : membre d'un groupe mais sans compte :
// aucune invitation émise.
func TestGroups_InviteRequiresExistingAccount(t *testing.T) {
	r, sessStore, groups, _ := newGroupsRouter(t)
	g, err := groups.Create("Fam", "ghost-x", "Ghost")
	if err != nil {
		t.Fatalf("groups.Create : %v", err)
	}
	cookie := authCookie(t, sessStore, "ghost-x", "Ghost")

	req := httptest.NewRequest(http.MethodPost, "/groups/"+g.ID+"/invites", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("invitation sans compte : %d, attendu 403 (corps %s)", w.Code, w.Body.String())
	}
}

// newInviteRegisterRouter : inscription sur invitation (mode password).
func newInviteRegisterRouter(t *testing.T) (*chi.Mux, *userstore.Store, *userstore.InviteStore) {
	t.Helper()
	dir := t.TempDir()
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	invites := userstore.NewInviteStore(filepath.Join(dir, "invites.json"), users)
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	h := handlers.NewUserAuthHandler(users, invites, sessStore, "invite").WithAuthMode("password")
	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessStore, middleware.SecureCookiePolicy{}))
	h.Mount(r)
	return r, users, invites
}

// TestRegister_InviteFromDeletedIssuerRefused : l'admin qui a émis l'invitation a été
// supprimé : l'invitation ne crée plus de compte.
func TestRegister_InviteFromDeletedIssuerRefused(t *testing.T) {
	r, users, invites := newInviteRegisterRouter(t)
	if _, err := users.Create("keeper", testUserAuthPass, domain.RoleAdmin); err != nil {
		t.Fatalf("Create keeper : %v", err)
	}
	if _, err := users.Create("boss", testUserAuthPass, domain.RoleAdmin); err != nil {
		t.Fatalf("Create boss : %v", err)
	}
	inv, err := invites.Generate("boss", 7, "")
	if err != nil {
		t.Fatalf("Generate : %v", err)
	}
	if err := users.Delete("boss"); err != nil {
		t.Fatalf("Delete boss : %v", err)
	}

	w := postJSON(t, r, "/auth/register", domain.RegisterRequest{
		Username: "newcomer", Password: testUserAuthPass, InviteCode: inv.Code,
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("inscription avec l'invitation d'un émetteur supprimé : %d, attendu 403 (corps %s)", w.Code, w.Body.String())
	}
	if _, err := users.Get("newcomer"); err == nil {
		t.Error("compte créé avec l'invitation d'un émetteur supprimé")
	}
}

// TestRegister_InviteIsSingleUseUnderConcurrency : six inscriptions simultanées avec
// le même code : un seul compte créé.
func TestRegister_InviteIsSingleUseUnderConcurrency(t *testing.T) {
	r, users, invites := newInviteRegisterRouter(t)
	if _, err := users.Create("boss", testUserAuthPass, domain.RoleAdmin); err != nil {
		t.Fatalf("Create boss : %v", err)
	}
	inv, err := invites.Generate("boss", 7, "")
	if err != nil {
		t.Fatalf("Generate : %v", err)
	}

	const n = 6
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			postJSON(t, r, "/auth/register", domain.RegisterRequest{
				Username: "racer" + string(rune('a'+i)), Password: testUserAuthPass, InviteCode: inv.Code,
			})
		}()
	}
	wg.Wait()

	list, err := users.List()
	if err != nil {
		t.Fatalf("List : %v", err)
	}
	if created := len(list) - 1; created != 1 {
		t.Fatalf("comptes créés avec un seul code : %d, attendu 1", created)
	}
}
