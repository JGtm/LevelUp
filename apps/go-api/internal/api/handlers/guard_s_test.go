package handlers

// guard_s_test.go — Gate du lot S (sécurité) : vérifie que les endpoints mutants
// ou révélateurs d'identité renvoient 401 à une requête ANONYME quand
// l'enforcement est actif (demo=false, auth_mode=password).
//
// Le handler réel est monté sous le MÊME stack de gardes que dans server_apiv1.go ;
// la garde RequireAuth court-circuite AVANT que le handler ne s'exécute, donc des
// dépendances nil suffisent (le corps du handler n'est jamais atteint). Le câblage
// réel (server_apiv1.go monte bien ces handlers sous ces gardes) est verrouillé par
// le contrôle grep du lot S (revue route→garde S3), ce test verrouille le
// COMPORTEMENT (401 anonyme) des gardes appliquées.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
)

func TestLotS_GuardedRoutes_AnonymousUnauthorized(t *testing.T) {
	const demo = false
	const mode = "password"
	auth := middleware.RequireAuth(demo, mode)
	admin := middleware.RequireAdmin(demo, mode)

	newSettings := func(r chi.Router) { NewSettingsHandler(&config.AppConfig{}, nil, nil).Mount(r) }

	cases := []struct {
		name   string
		method string
		path   string
		mount  func(chi.Router)
		mws    []func(http.Handler) http.Handler
	}{
		// S1 — /settings mute la config (RequireAuth + RequireAdmin).
		{"S1 PATCH /settings", http.MethodPatch, "/settings", newSettings, []func(http.Handler) http.Handler{auth, admin}},
		{"S1 POST /settings/media/scan", http.MethodPost, "/settings/media/scan", newSettings, []func(http.Handler) http.Handler{auth, admin}},
		{"S1 POST /settings/media/reset-index", http.MethodPost, "/settings/media/reset-index", newSettings, []func(http.Handler) http.Handler{auth, admin}},
		{"S1 POST /settings/sessions/recalculate", http.MethodPost, "/settings/sessions/recalculate", newSettings, []func(http.Handler) http.Handler{auth, admin}},
		{"S1 POST /settings/backup/run", http.MethodPost, "/settings/backup/run", newSettings, []func(http.Handler) http.Handler{auth, admin}},
		// S2 — backfill progression MUTE des données (RequireAuth + RequireAdmin).
		{
			"S2 POST /_admin/progression/backfill/{slug}", http.MethodPost, "/_admin/progression/backfill/jgtm",
			func(r chi.Router) { NewProgressionBackfillHandler(nil).Mount(r) },
			[]func(http.Handler) http.Handler{auth, admin},
		},
		// S6 — diagnostics révélateurs d'identité (RequireAuth ; ownership en plus
		// pour les routes {player_slug}, no-op ici avec resolvers nil).
		{
			"S6 GET /_diag/csr-coverage/{slug}", http.MethodGet, "/_diag/csr-coverage/jgtm",
			func(r chi.Router) { NewDiagCSRHandler(nil).Mount(r) },
			[]func(http.Handler) http.Handler{auth},
		},
		{
			"S6 GET /_diag/progression/{slug}", http.MethodGet, "/_diag/progression/jgtm",
			func(r chi.Router) { NewDiagProgressionHandler(nil).Mount(r) },
			[]func(http.Handler) http.Handler{auth},
		},
		{
			"S6 GET /healthz/home", http.MethodGet, "/healthz/home",
			func(r chi.Router) { NewHealthHomeHandler(nil).Mount(r) },
			[]func(http.Handler) http.Handler{auth},
		},
		// S8 — /setup écrit db_profiles.json (RequireAuth).
		{
			"S8 POST /setup/players", http.MethodPost, "/setup/players",
			func(r chi.Router) { NewSetupHandler(&config.AppConfig{}, nil, nil, nil, nil).Mount(r) },
			[]func(http.Handler) http.Handler{auth},
		},
		{
			"S8 POST /setup/smoke-test", http.MethodPost, "/setup/smoke-test",
			func(r chi.Router) { NewSetupHandler(&config.AppConfig{}, nil, nil, nil, nil).Mount(r) },
			[]func(http.Handler) http.Handler{auth},
		},
		// S3 — trouvé par la revue exhaustive : import mutant sur `r` nu (RequireAuth).
		{
			"S3 POST /import/openspartan", http.MethodPost, "/import/openspartan",
			func(r chi.Router) {
				r.Post("/import/openspartan", NewOpenSpartanImportHandler(OpenSpartanImportConfig{}).StartImport)
			},
			[]func(http.Handler) http.Handler{auth},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Group(func(r chi.Router) {
				for _, mw := range tc.mws {
					r.Use(mw)
				}
				tc.mount(r)
			})
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("%s : status = %d, want 401 (auth_required)", tc.name, rec.Code)
			}
		})
	}
}
