package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/userstore"
)

// TestRequireAdmin_RereadsRoleFromStore : rétrograder puis rejouer la session → 403 ;
// supprimer le compte → 403 ; la session seule ne fait jamais un admin.
func TestRequireAdmin_RereadsRoleFromStore(t *testing.T) {
	users := userstore.NewStore(filepath.Join(t.TempDir(), "users.json"))
	// « keeper » reste admin : l'instance garde toujours un administrateur (ErrLastAdmin).
	if _, err := users.Create("keeper", "keeper-pass-123", domain.RoleAdmin); err != nil {
		t.Fatalf("Create : %v", err)
	}
	if _, err := users.Create("boss", "boss-pass-1234", domain.RoleAdmin); err != nil {
		t.Fatalf("Create : %v", err)
	}
	username, role := "boss", string(domain.RoleAdmin)
	sess := &domain.SessionData{SessionID: "s-boss", Username: &username, Role: &role}
	guarded := middleware.RequireAdmin(false, "xbox", users)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	replay := func() int {
		req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
		req = req.WithContext(middleware.InjectSession(req.Context(), sess))
		w := httptest.NewRecorder()
		guarded.ServeHTTP(w, req)
		return w.Code
	}

	if got := replay(); got != http.StatusOK {
		t.Fatalf("admin du magasin : %d, attendu 200", got)
	}
	if err := users.SetRole("boss", domain.RoleUser); err != nil {
		t.Fatalf("SetRole : %v", err)
	}
	if got := replay(); got != http.StatusForbidden {
		t.Errorf("admin rétrogradé, session rejouée : %d, attendu 403", got)
	}
	if err := users.SetRole("boss", domain.RoleAdmin); err != nil {
		t.Fatalf("SetRole : %v", err)
	}
	if err := users.Delete("boss"); err != nil {
		t.Fatalf("Delete : %v", err)
	}
	if got := replay(); got != http.StatusForbidden {
		t.Errorf("admin supprimé, session rejouée : %d, attendu 403", got)
	}
}
