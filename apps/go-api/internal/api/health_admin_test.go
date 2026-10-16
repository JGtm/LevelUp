//go:build cgo

// health_admin_test.go — /health public réduit à {status} (registre sécurité C7-2, lot L9-1) :
// versions, compteurs et outillage ne sont servis que par GET /api/v1/admin/health, derrière
// RequireAuth et RequireAdmin (câblage vérifié sur le routeur réel).
package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestAdminHealth_RequiresAdmin(t *testing.T) {
	r, ok := buildTestRouter(t).(chi.Router)
	if !ok {
		t.Fatal("le routeur n'est pas un chi.Router")
	}
	found := false
	err := chi.Walk(r, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		if method != http.MethodGet || route != "/api/v1/admin/health" {
			return nil
		}
		found = true
		var names []string
		for _, mw := range mws {
			names = append(names, runtime.FuncForPC(reflect.ValueOf(mw).Pointer()).Name())
		}
		chain := strings.Join(names, " ")
		for _, guard := range []string{"RequireAuth", "RequireAdmin"} {
			if !strings.Contains(chain, guard) {
				t.Errorf("GET /api/v1/admin/health sans %s (chaîne : %s)", guard, chain)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk : %v", err)
	}
	if !found {
		t.Fatal("route GET /api/v1/admin/health absente du routeur")
	}
}

// Sur le routeur réel, un anonyme lit {status} et rien d'autre.
func TestPublicHealth_StatusOnly(t *testing.T) {
	rec := httptest.NewRecorder()
	buildTestRouter(t).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /health : statut %d, corps %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("corps illisible : %v", err)
	}
	if len(body) != 1 || body["status"] != "ok" {
		t.Errorf("GET /health doit rendre {status:ok} seul, obtenu %v", body)
	}
}
