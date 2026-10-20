//go:build cgo

// player_slug_param_routes_test.go — toute route dont le chemin porte un slug ou un gamertag
// de joueur en contrôle la forme (domain.IsValidPlayerSlug) avant le handler.
//
// COMMENT. Le routeur réel (démo, cf. family_write_routes_test.go) est parcouru par chi.Walk ;
// toute route dont le motif porte un paramètre de playerSlugParams doit porter, parmi ses
// middlewares, RequirePlayerOwnership (groupes /players/{player_slug} et /profiles/…, slug
// contrôlé dans tous les modes) ou RequirePlayerSlugParam (routes d'administration et
// diagnostics). Un slug devient un segment de chemin disque (dossier du joueur).
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

// playerSlugParams : noms de paramètres de chemin qui désignent un joueur par son slug ou
// son gamertag.
var playerSlugParams = []string{"{player_slug}", "{player}", "{gamertag}"}

// playerSlugGuards : middlewares qui contrôlent la forme du slug avant le handler, reconnus au
// nom de leur fermeture (le préfixe de paquet change quand le compilateur l'inline chez
// l'appelant).
var playerSlugGuards = []string{"RequirePlayerOwnership.", "RequirePlayerSlugParam."}

func TestPlayerSlugParamRoutes_CarrySlugFormGuard(t *testing.T) {
	r, ok := buildTestRouter(t).(chi.Router)
	if !ok {
		t.Fatal("le routeur n'est pas un chi.Router")
	}
	var unguarded, guarded []string
	err := chi.Walk(r, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		if !carriesPlayerSlugParam(route) {
			return nil
		}
		for _, mw := range mws {
			name := runtime.FuncForPC(reflect.ValueOf(mw).Pointer()).Name()
			for _, g := range playerSlugGuards {
				if strings.Contains(name, g) {
					guarded = append(guarded, method+" "+route)
					return nil
				}
			}
		}
		unguarded = append(unguarded, method+" "+route)
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk : %v", err)
	}
	sort.Strings(unguarded)
	for _, u := range unguarded {
		t.Errorf("%s : slug de joueur dans le chemin sans contrôle de forme "+
			"(RequirePlayerOwnership ou RequirePlayerSlugParam)", u)
	}
	if len(guarded) < 100 {
		t.Errorf("%d route(s) gardée(s) : le routeur de test ne porte pas la surface attendue", len(guarded))
	}
}

func carriesPlayerSlugParam(route string) bool {
	for _, p := range playerSlugParams {
		if strings.Contains(route, p) {
			return true
		}
	}
	return false
}
