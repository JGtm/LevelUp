package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
)

// fakeDirectory : port.PlayerDirectory en double de test. Le handler n'a aucune
// logique — il n'y a donc que trois comportements à tenir : le corps passe, les
// erreurs deviennent des 500, et l'annuaire non câblé un 503.
type fakeDirectory struct {
	resp domain.AdminIdentitiesResponse
	err  error
}

func (f *fakeDirectory) List(context.Context) (domain.AdminIdentitiesResponse, error) {
	return f.resp, f.err
}

func (f *fakeDirectory) Get(context.Context, string) (domain.IdentityRecord, error) {
	return domain.IdentityRecord{}, nil
}

func (f *fakeDirectory) HasTrackedProfile(context.Context, string, string) (bool, error) {
	return false, nil
}

func identitiesFixture() domain.AdminIdentitiesResponse {
	return domain.AdminIdentitiesResponse{
		GeneratedAt: "2026-09-15T12:00:00Z",
		Identities: []domain.IdentityRecord{
			{
				XUID:     "999",
				Gamertag: "Inconnu",
				Profiles: []domain.ProfileRef{},
				Account:  &domain.AccountRef{Username: "inconnu", Role: domain.RoleUser},
				Watched:  []string{},
				Anomalies: []domain.IdentityAnomaly{
					{Code: domain.AnomalyAccountWithoutProfile, Severity: domain.AnomalySeverityWarning, Detail: "inconnu"},
				},
			},
			{
				XUID:      "111",
				Gamertag:  "Spartan",
				Profiles:  []domain.ProfileRef{{TitleSlug: "halo_infinite", Key: "Spartan", SyncEnabled: true}},
				Watched:   []string{"halo_infinite"},
				Anomalies: []domain.IdentityAnomaly{},
			},
		},
		Counts: map[string]int{"identities": 2, "warnings": 1, "infos": 0},
	}
}

func TestAdminIdentitiesHandler_Get_OK(t *testing.T) {
	h := NewAdminIdentitiesHandler(&fakeDirectory{resp: identitiesFixture()})
	r := chi.NewRouter()
	r.Route("/admin", func(r chi.Router) { h.Mount(r) })

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/identities", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got domain.AdminIdentitiesResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Identities) != 2 {
		t.Fatalf("identités = %d, want 2", len(got.Identities))
	}
	if got.Identities[0].XUID != "999" || len(got.Identities[0].Anomalies) != 1 {
		t.Fatalf("première identité = %+v", got.Identities[0])
	}
	if got.Identities[0].Anomalies[0].Code != domain.AnomalyAccountWithoutProfile {
		t.Fatalf("anomalie = %+v", got.Identities[0].Anomalies[0])
	}
	if got.Counts["warnings"] != 1 {
		t.Fatalf("compteurs = %v", got.Counts)
	}
	if got.GeneratedAt != "2026-09-15T12:00:00Z" {
		t.Fatalf("generated_at = %q", got.GeneratedAt)
	}
}

func TestAdminIdentitiesHandler_Get_Error(t *testing.T) {
	h := NewAdminIdentitiesHandler(&fakeDirectory{err: errors.New("boom")})
	r := chi.NewRouter()
	r.Route("/admin", func(r chi.Router) { h.Mount(r) })

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/identities", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

// TestAdminIdentitiesHandler_Get_SansAnnuaire : annuaire non câblé → 503, jamais
// une panique ni un corps vide qui passerait pour « aucune identité ».
func TestAdminIdentitiesHandler_Get_SansAnnuaire(t *testing.T) {
	h := NewAdminIdentitiesHandler(nil)
	r := chi.NewRouter()
	r.Route("/admin", func(r chi.Router) { h.Mount(r) })

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/identities", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

// TestAdminIdentitiesHandler_Gating_401_403 : la route est gatée par la MÊME
// chaîne que les autres routes /admin (RequireAuth + RequireAdmin).
func TestAdminIdentitiesHandler_Gating_401_403(t *testing.T) {
	h := NewAdminIdentitiesHandler(&fakeDirectory{resp: identitiesFixture()})
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(false, "password"))
		r.Use(middleware.RequireAdmin(false, "password"))
		r.Route("/admin", func(r chi.Router) { h.Mount(r) })
	})

	serve := func(sess *domain.SessionData) int {
		req := httptest.NewRequest(http.MethodGet, "/admin/identities", nil)
		if sess != nil {
			req = req.WithContext(middleware.InjectSession(req.Context(), sess))
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	if code := serve(nil); code != http.StatusUnauthorized {
		t.Errorf("sans session : status=%d, want 401", code)
	}
	user := "bob"
	roleUser := "user"
	if code := serve(&domain.SessionData{Username: &user, Role: &roleUser}); code != http.StatusForbidden {
		t.Errorf("non-admin : status=%d, want 403", code)
	}
	roleAdmin := "admin"
	if code := serve(&domain.SessionData{Username: &user, Role: &roleAdmin}); code != http.StatusOK {
		t.Errorf("admin : status=%d, want 200", code)
	}
}
