//go:build cgo

// recompute_coefficients_admin_test.go — POST .../engagement/recompute_coefficients est une
// opération d'administration (recalcul de l'historique, libellée « admin » au contrat) : la
// route porte RequireAuth ET RequireAdmin sur le routeur réel (chi.Walk). Aucun écran non
// admin ne l'appelle (le contrat web ne la référence que dans son instantané de surface).
package api_test

import (
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRecomputeCoefficients_RequiresAdmin(t *testing.T) {
	r, ok := buildTestRouter(t).(chi.Router)
	if !ok {
		t.Fatal("le routeur n'est pas un chi.Router")
	}
	found := false
	err := chi.Walk(r, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		if method != http.MethodPost || !strings.HasSuffix(route, "/engagement/recompute_coefficients") {
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
				t.Errorf("POST %s sans %s (chaîne : %s)", route, guard, chain)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk : %v", err)
	}
	if !found {
		t.Fatal("route POST .../engagement/recompute_coefficients absente du routeur")
	}
}
