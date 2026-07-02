// Package handlers — security_lot_s_test.go : tests httptest du LOT S (sécurité,
// audit 2026-07-02).
//
// Prouve que les endpoints re-gardés dans server.go rejettent une requête anonyme
// (401) une fois montés sous le middleware d'auth utilisé par server.go, que le
// mode demo/single-user reste transparent (no-op), et que le probe auto-sync ne
// sérialise plus aucun fragment de refresh token.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
)

// lotSRouter monte `mount` sous /api/v1 avec RequireAuth (+ RequireAdmin si admin),
// en mode enforced (authMode=password) ou demo selon demoMode — reproduit le
// câblage de server.go (Lot S).
func lotSRouter(mount func(chi.Router), admin, demoMode bool) http.Handler {
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(demoMode, "password"))
			if admin {
				r.Use(middleware.RequireAdmin(demoMode, "password"))
			}
			mount(r)
		})
	})
	return r
}

// withAdminSession injecte une session admin (username + role=admin) dans le
// contexte, comme le ferait WithSession après un login réussi.
func withAdminSession(req *http.Request) *http.Request {
	u, role := "boss", "admin"
	return req.WithContext(middleware.InjectSession(req.Context(),
		&domain.SessionData{Username: &u, Role: &role}))
}

// --- S1 (audit B1) : PATCH /settings + POST associés → 401 anonyme ---

func TestLotS_Settings_RequireAuthAdmin(t *testing.T) {
	h := NewSettingsHandler(&config.AppConfig{}, nil, nil)
	srv := lotSRouter(func(r chi.Router) { h.Mount(r) }, true, false)

	for _, tc := range []struct{ method, path string }{
		{http.MethodPatch, "/api/v1/settings"},
		{http.MethodPost, "/api/v1/settings/media/scan"},
		{http.MethodPost, "/api/v1/settings/sessions/recalculate"},
		{http.MethodPost, "/api/v1/settings/backup/run"},
		{http.MethodPost, "/api/v1/settings/media/reset-index"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rr := httptest.NewRecorder()
		srv.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("%s %s anonyme: status = %d, want 401", tc.method, tc.path, rr.Code)
		}
	}
}

// --- S2 (audit B2) : POST /_admin/progression/backfill/{slug} → 401 anonyme,
// admin et demo passent (no-op) ---

func TestLotS_ProgressionBackfill_RequireAuthAdmin(t *testing.T) {
	factory := func(_ context.Context, _ string) (ProgressionBackfiller, error) {
		return &stubProgressionBackfiller{diag: &domain.ProgressionDiag{}}, nil
	}
	mount := func(r chi.Router) {
		NewProgressionBackfillHandler(factory).Mount(r.With(middleware.NoStore))
	}
	path := "/api/v1/_admin/progression/backfill/jgtm"

	// Anonyme enforced → 401.
	rr := httptest.NewRecorder()
	lotSRouter(mount, true, false).ServeHTTP(rr, httptest.NewRequest(http.MethodPost, path, nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("anonyme: status = %d, want 401", rr.Code)
	}

	// Admin enforced → ni 401 ni 403 (le backfill s'exécute).
	rrAdmin := httptest.NewRecorder()
	lotSRouter(mount, true, false).ServeHTTP(rrAdmin, withAdminSession(httptest.NewRequest(http.MethodPost, path, nil)))
	if rrAdmin.Code == http.StatusUnauthorized || rrAdmin.Code == http.StatusForbidden {
		t.Fatalf("admin: status = %d, ne doit pas être 401/403", rrAdmin.Code)
	}

	// Demo → middlewares transparents (invariant single-user).
	rrDemo := httptest.NewRecorder()
	lotSRouter(mount, true, true).ServeHTTP(rrDemo, httptest.NewRequest(http.MethodPost, path, nil))
	if rrDemo.Code == http.StatusUnauthorized || rrDemo.Code == http.StatusForbidden {
		t.Fatalf("demo no-op: status = %d, ne doit pas être 401/403", rrDemo.Code)
	}
}

// --- S6 (audit M4) : diagnostics par joueur → 401 anonyme ---

func TestLotS_Diagnostics_RequireAuthAdmin(t *testing.T) {
	cases := []struct {
		name  string
		mount func(chi.Router)
		path  string
	}{
		{"health-home", func(r chi.Router) { NewHealthHomeHandler(nil).Mount(r.With(middleware.NoStore)) }, "/api/v1/healthz/home?player=jgtm"},
		{"diag-csr", func(r chi.Router) { NewDiagCSRHandler(nil).Mount(r.With(middleware.NoStore)) }, "/api/v1/_diag/csr-coverage/jgtm"},
		{"diag-progression", func(r chi.Router) { NewDiagProgressionHandler(nil).Mount(r.With(middleware.NoStore)) }, "/api/v1/_diag/progression/jgtm"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			lotSRouter(tc.mount, true, false).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if rr.Code != http.StatusUnauthorized {
				t.Errorf("%s anonyme: status = %d, want 401", tc.name, rr.Code)
			}
		})
	}
}

// --- S8 (audit A4-m2) : /setup/players & /setup/smoke-test → 401 anonyme
// (RequireAuth seul, self-provision autorisé) ---

func TestLotS_Setup_RequireAuth(t *testing.T) {
	h := NewSetupHandler(&config.AppConfig{}, nil, nil, nil, nil)
	srv := lotSRouter(func(r chi.Router) { h.Mount(r) }, false, false)
	for _, path := range []string{"/api/v1/setup/players", "/api/v1/setup/smoke-test"} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		rr := httptest.NewRecorder()
		srv.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("%s anonyme: status = %d, want 401", path, rr.Code)
		}
	}
}

// --- S5 (audit M3) : le probe auto-sync ne sérialise plus aucun fragment de RT ---

func TestLotS_TokenProbeResult_NeverLeaksFragments(t *testing.T) {
	res := TokenProbeResult{
		Gamertag:           "jgtm",
		HasRefreshToken:    true,
		RefreshTokenLen:    40,
		RefreshTokenSHA256: "deadbeefcafe0000",
	}
	b, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := m["refresh_token_head"]; ok {
		t.Errorf("refresh_token_head ne doit jamais être sérialisé : %s", b)
	}
	if _, ok := m["refresh_token_tail"]; ok {
		t.Errorf("refresh_token_tail ne doit jamais être sérialisé : %s", b)
	}
	if _, ok := m["refresh_token_sha256"]; !ok {
		t.Errorf("refresh_token_sha256 doit rester présent (empreinte non réversible) : %s", b)
	}
}

func TestLotS_FingerprintToken_ShaOnly(t *testing.T) {
	if got := fingerprintToken(""); got != "" {
		t.Errorf("token vide → \"\", obtenu %q", got)
	}
	if got := fingerprintToken("a-secret-refresh-token"); len(got) != 16 {
		t.Errorf("empreinte sha256[:8] attendue = 16 hex, obtenu %q (len %d)", got, len(got))
	}
}
