package middleware_test

// demo_read_only_ambiguity_test.go — les trois protections de la garde démo contre les
// chemins qui ne se lisent pas comme ils se routent. Chaque cas ne rougit que si SA clause
// est retirée : les autres protections le laissent sinon passer (mutations vérifiées).

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/middleware"
)

// serveRoutedDemo monte la garde démo devant des routes d'écriture réelles et rend le nom
// de la route d'écriture atteinte ("" si aucune) et le statut.
func serveRoutedDemo(target string) (string, int) {
	reached := ""
	r := chi.NewRouter()
	r.Use(middleware.DemoReadOnly(true, workerPrefixForTest))
	r.Post("/api/v1/players/{player_slug}/coach/proposals/{id}/dismiss", func(http.ResponseWriter, *http.Request) {
		reached = "dismiss"
	})
	r.Post("/api/v1/admin/monitoring/lusr-gaps/{player}/recompute", func(http.ResponseWriter, *http.Request) {
		reached = "recompute"
	})
	r.Post("/api/v1/{area}", func(http.ResponseWriter, *http.Request) { reached = "area" })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, target, nil))
	return reached, w.Code
}

func assertRefused(t *testing.T, target string) {
	t.Helper()
	reached, code := serveRoutedDemo(target)
	if reached != "" {
		t.Errorf("%s : écriture %q ATTEINTE en démo (statut %d)", target, reached, code)
	}
	if code != http.StatusForbidden {
		t.Errorf("%s : statut %d, attendu 403", target, code)
	}
}

// Sans `!AmbiguousPath(r)` devant isExemptPath (path.Clean), ce chemin nettoyé en
// /api/v1/internal/x/dismiss serait pris pour le protocole ouvrier exempté.
func TestDemoReadOnly_AmbiguousPathNeverTakesTheWorkerExemption(t *testing.T) {
	assertRefused(t, "/api/v1/players/demo-player/coach/proposals/"+
		"..%2F..%2F..%2F..%2Finternal%2Fx/dismiss")
}

// Un séparateur encodé seul (aucun segment « .. ») se lit comme une page de lecture mais se
// route vers une écriture : seule la liste d'AmbiguousPath (%2f) le distingue.
func TestDemoReadOnly_EncodedSlashAloneIsAmbiguous(t *testing.T) {
	assertRefused(t, "/api/v1/admin/monitoring/lusr-gaps/a%2Fplayers%2Fb%2Fpages%2Fc/recompute")
	assertRefused(t, "/api/v1/admin/monitoring/lusr-gaps/a%2fplayers%2fb%2fpages%2fc/recompute")
}

// `/api/v1/session%2Fcontext` se décode en l'exemption `POST /session/context` mais se
// route ailleurs : DemoAllows doit refuser la lecture d'un chemin ambigu.
func TestDemoAllows_AmbiguousPathNeverTakesTheSessionContextExemption(t *testing.T) {
	assertRefused(t, "/api/v1/session%2Fcontext")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/session%2Fcontext", nil)
	if middleware.DemoAllows(req) {
		t.Error("DemoAllows laisse passer un chemin ambigu qui se décode en /session/context")
	}
	ok := httptest.NewRequest(http.MethodPost, "/api/v1/session/context", nil)
	if !middleware.DemoAllows(ok) {
		t.Error("DemoAllows refuse le vrai POST /session/context")
	}
}
