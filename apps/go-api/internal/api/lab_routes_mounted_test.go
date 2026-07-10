//go:build cgo

// Package api_test — lab_routes_mounted_test.go : anti-régression du montage
// du diagnostic d'instance (ex-Lab).
//
// Historique : le backend du Lab existait mais n'était jamais monté dans
// server.go → /lab/* renvoyait 404 en prod (PMT-14 volet C). Ce test construit
// le VRAI routeur (buildTestRouter, mode démo) et vérifie via chi.Walk que la
// route survivante est enregistrée.
//
// A3.5 (DC-9, 2026-07-10) : le Lab est retiré de l'app — seule
// GET /lab/diagnostics doit rester montée (panneau Diagnostics de l'onglet
// admin Données). Les explorateurs /lab/{resources,contracts,waypoint} doivent
// être ABSENTS (garde-rail anti-résurrection du code supprimé).
package api_test

import (
	"strings"
	"testing"
)

func TestLabRoutesMounted(t *testing.T) {
	routes := chiRoutes(buildTestRouter(t))

	// La route survivante doit être montée.
	found := false
	for route := range routes {
		// route au format "METHOD /chemin/complet" (ex. "GET /api/v1/lab/diagnostics").
		if strings.HasPrefix(route, "GET ") && strings.HasSuffix(route, "/lab/diagnostics") {
			found = true
			break
		}
	}
	if !found {
		t.Error("route diagnostic non montée (GET .../lab/diagnostics absent) — régression : le diagnostic d'instance doit rester câblé dans server.go")
	}

	// Les routes supprimées ne doivent PAS réapparaître.
	removedSuffixes := []string{"/lab/resources", "/lab/contracts", "/lab/waypoint"}
	for route := range routes {
		for _, suffix := range removedSuffixes {
			if strings.HasSuffix(route, suffix) {
				t.Errorf("route Lab supprimée re-montée : %s (A3.5/DC-9 — le Lab est retiré de l'app)", route)
			}
		}
	}
}
