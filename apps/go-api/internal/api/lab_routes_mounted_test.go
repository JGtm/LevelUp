//go:build cgo

// Package api_test — lab_routes_mounted_test.go : anti-régression du montage Lab.
//
// Le backend du Lab interne (handlers/service/provider) existait mais n'était
// jamais monté dans server.go → /lab/* renvoyait 404 en prod. La casse était
// masquée par les mocks MSW du front + les tests chi-local du handler, qui ne
// vérifiaient jamais l'intégration serveur réelle. Ce test construit le VRAI
// routeur (buildTestRouter, mode démo) et vérifie via chi.Walk que les trois
// routes Lab sont enregistrées — c'est précisément le test absent qui aurait
// attrapé la casse (PMT-14 volet C).
package api_test

import (
	"strings"
	"testing"
)

func TestLabRoutesMounted(t *testing.T) {
	routes := chiRoutes(buildTestRouter(t))

	wantSuffixes := []string{
		"/lab/resources",
		"/lab/contracts",
		"/lab/diagnostics",
		"/lab/waypoint",
	}
	for _, suffix := range wantSuffixes {
		found := false
		for route := range routes {
			// route au format "METHOD /chemin/complet" (ex. "GET /api/v1/lab/resources").
			if strings.HasPrefix(route, "GET ") && strings.HasSuffix(route, suffix) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("route Lab non montée (GET ...%s absent) — régression : le backend Lab doit rester câblé dans server.go", suffix)
		}
	}
}
