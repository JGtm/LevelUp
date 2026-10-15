//go:build cgo

// healthz_home_admin_test.go — /healthz/home réservé à l'admin (décision utilisateur
// D-C du 2026-10-15) : la sonde construit la home d'un joueur choisi par `?player=`
// (lecture d'autrui, rafraîchissement de sa carrière avec son jeton) ; elle reste une
// sonde d'opérateur. Le câblage est vérifié sur le routeur réel (chi.Walk) : la route
// porte RequireAuth ET RequireAdmin, inertes en démo et sans auth (e2e, harnais démo).
package api_test

import (
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestHealthzHome_RequiresAdmin(t *testing.T) {
	r, ok := buildTestRouter(t).(chi.Router)
	if !ok {
		t.Fatal("le routeur n'est pas un chi.Router")
	}
	found := false
	err := chi.Walk(r, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		if method != http.MethodGet || route != "/api/v1/healthz/home" {
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
				t.Errorf("GET /api/v1/healthz/home sans %s (chaîne : %s)", guard, chain)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk : %v", err)
	}
	if !found {
		t.Fatal("route GET /api/v1/healthz/home absente du routeur")
	}
}
