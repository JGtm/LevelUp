//go:build cgo

// player_mutations_guard_ratchet_test.go — Ratchet « écritures joueur gardées »
// (audit RequireAuth du groupe /players/{player_slug}, 2026-08-04).
//
// POURQUOI CE GARDE-RAIL. L'audit a montré que le groupe player-scoped n'avait
// jamais porté de garde d'AUTHENTIFICATION : la seule route qui refusait un
// anonyme était /media/likes, gatée à la main le 2026-08-02. La correction est un
// middleware de groupe — donc une protection qui tient tant que les routes
// restent montées SOUS ce groupe. Rien n'empêche structurellement un futur
// `r.Post("/api/v1/players/{player_slug}/...")` monté ailleurs, et il serait à
// nouveau ouvert sans que personne ne le voie. Ce test est ce filet.
//
// COMMENT. Même technique que bare_routes_ratchet_test.go : on construit le VRAI
// routeur en mode démo et on identifie les middlewares par le nom runtime de leur
// fonction. En démo la garde est inerte à l'EXÉCUTION, mais son closure reste
// présent dans la chaîne exposée par chi.Walk — c'est bien le CÂBLAGE qu'on
// vérifie ici, pas le comportement (couvert par require_auth_mutations_test.go).
package api_test

import (
	"net/http"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// playerScopedPrefix — le groupe dont toutes les routes doivent être gardées.
const playerScopedPrefix = "/api/v1/players/{player_slug}"

// mutationGuardMarker — sous-chaîne du nom runtime du middleware d'écriture.
const mutationGuardMarker = "RequireAuthForMutations"

// routeHasMutationGuard — vrai si la chaîne de middlewares porte la garde.
func routeHasMutationGuard(mws []func(http.Handler) http.Handler) bool {
	for _, mw := range mws {
		if strings.Contains(runtime.FuncForPC(reflect.ValueOf(mw).Pointer()).Name(), mutationGuardMarker) {
			return true
		}
	}
	return false
}

// TestPlayerRoutesRatchet_AllUnderMutationGuard : toute route player-scoped passe
// par RequireAuthForMutations.
//
// La garde est posée sur le GROUPE et laisse passer les lectures : l'exiger sur
// TOUTES les routes (et pas seulement sur celles qu'on croit mutantes) évite
// d'avoir à re-classer chaque route ici — la classification vit à un seul endroit,
// middleware.IsMutatingRequest.
func TestPlayerRoutesRatchet_AllUnderMutationGuard(t *testing.T) {
	t.Setenv("LEVELUP_DEMO_MODE", "true")
	t.Setenv("PRESTIGE_ENABLED", "true")

	router := buildTestRouter(t)
	r, ok := router.(chi.Router)
	if !ok {
		t.Fatalf("le routeur n'est pas un chi.Router (%T)", router)
	}

	var unguarded []string
	seen := 0
	err := chi.Walk(r, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		if !strings.HasPrefix(route, playerScopedPrefix) {
			return nil
		}
		seen++
		if !routeHasMutationGuard(mws) {
			unguarded = append(unguarded, method+" "+route)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}

	// Self-check : un walk qui ne verrait aucune route player-scoped rendrait ce
	// test vert pour de mauvaises raisons (routeur mal construit, préfixe changé).
	if seen == 0 {
		t.Fatalf("aucune route sous %s n'a été walkée — le ratchet ne vérifie rien", playerScopedPrefix)
	}

	if len(unguarded) > 0 {
		sort.Strings(unguarded)
		t.Errorf("%d route(s) player-scoped SANS garde d'écriture — les monter sous le "+
			"groupe /players/{player_slug} de server_apiv1.go :", len(unguarded))
		for _, u := range unguarded {
			t.Errorf("  - %s", u)
		}
	}
}
