// Package handlers_test — admin_role_reread_test.go : un rôle admin se lit dans le
// compte (magasin d'utilisateurs), jamais dans la session. Un admin rétrogradé qui
// rejoue sa session perd la main sur les décisions réservées : verrou d'instance,
// suppression et association de médias d'autrui, exemption du provisionnement.
package handlers_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/port"
)

// staleAdminSession : la session d'un compte qui était admin à l'ouverture.
func staleAdminSession() *domain.SessionData {
	username, role := "boss", string(domain.RoleAdmin)
	return &domain.SessionData{SessionID: "s-boss", Username: &username, Role: &role}
}

// demotedBoss : le magasin, où « boss » n'est plus qu'utilisateur.
func demotedBoss(t *testing.T) *userstore.Store {
	t.Helper()
	users := userstore.NewStore(filepath.Join(t.TempDir(), "users.json"))
	if _, err := users.Create("boss", testUserAuthPass, domain.RoleUser); err != nil {
		t.Fatalf("Create boss : %v", err)
	}
	return users
}

// withSession pose la session donnée sur chaque requête.
func withSession(sess *domain.SessionData) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(middleware.InjectSession(r.Context(), sess)))
		})
	}
}

func TestSettings_InstanceLockToggle_DemotedAdminIs403(t *testing.T) {
	dir := t.TempDir()
	users := demotedBoss(t)
	cfg := &config.AppConfig{AuthMode: "xbox", DBProfilesPath: filepath.Join(dir, "db_profiles.json")}
	h := handlers.NewSettingsHandlerWithIndexer(cfg, settings_platform.NewStore(filepath.Join(dir, "app_settings.json")),
		jobs.NewStore(filepath.Join(dir, "jobs.json")), &mockMediaIndexer{}).WithUserLookup(users)
	r := chi.NewRouter()
	r.Use(withSession(staleAdminSession()))
	h.Mount(r)

	patch := func() int {
		req := httptest.NewRequest(http.MethodPatch, "/settings", bytes.NewReader([]byte(`{"instance_locked": false}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}
	if got := patch(); got != http.StatusForbidden {
		t.Fatalf("admin rétrogradé ouvre l'instance : %d, attendu 403", got)
	}
	if err := users.SetRole("boss", domain.RoleAdmin); err != nil {
		t.Fatalf("SetRole : %v", err)
	}
	if got := patch(); got != http.StatusOK {
		t.Fatalf("admin du magasin : %d, attendu 200", got)
	}
}

func TestMediaDelete_DemotedAdminIsNotAdmin(t *testing.T) {
	users := demotedBoss(t)
	mock := &mockMediaService{}
	factory := func(context.Context, string) (port.MediaService, error) { return mock, nil }
	h := handlers.NewMediaHandler(factory, nil, "").WithAuthEnforced(true).
		WithRequester(requesterIs("Boss")).WithUserLookup(users)
	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(r chi.Router) { h.Mount(r) })

	deleteMedia(r, "/api/v1/players/"+testPlayerSlug+"/media/files/JGtm/clip.mp4", staleAdminSession())
	if mock.delReq == nil {
		t.Fatal("suppression non transmise au service")
	}
	if mock.delReq.RequesterIsAdmin {
		t.Error("un admin rétrogradé est encore traité en admin pour supprimer le média d'autrui")
	}
}
