//go:build cgo

// spa_handler_routes_test.go — la liste des routes du routeur (isServerRoutePath) couvre le
// VRAI routeur : pendant le démarrage, une route serveur absente de cette liste serait
// servie en index.html au lieu de la réponse « serveur en démarrage ».
package api_test

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api"
)

// routeParam : segment variable d'un motif chi ({param} ou {param:regex}).
var routeParam = regexp.MustCompile(`\{[^/]+\}`)

func TestServerRoutePath_CouvreLeRouteur(t *testing.T) {
	t.Setenv("LEVELUP_DEMO_MODE", "true")
	t.Setenv("PRESTIGE_ENABLED", "true")
	r, ok := buildTestRouter(t).(chi.Router)
	if !ok {
		t.Fatal("le routeur n'est pas un chi.Router")
	}
	walked := 0
	err := chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if route == "/*" { // catch-all de la page (mountSPA)
			return nil
		}
		walked++
		sample := strings.ReplaceAll(routeParam.ReplaceAllString(route, "x"), "*", "x")
		if !api.IsServerRoutePath(sample) {
			t.Errorf("%s %s : route du routeur absente de serverRoutePrefixes/serverRoutePaths", method, route)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}
	if walked == 0 {
		t.Fatal("aucune route parcourue : le test ne vérifie rien")
	}
}
