//go:build cgo

// demo_read_only_ratchet_test.go — ratchet de la garde « démo en lecture seule »
// (middleware/demo_read_only.go) sur le VRAI routeur assemblé en démo.
//
// DEUX PROPRIÉTÉS, et chacune ferme un trou différent :
//
//  1. CÂBLAGE : toute route du routeur porte la garde dans sa chaîne de middlewares. Une
//     route montée hors de la racine (un second routeur, un http.Handle nu) échapperait au
//     refus sans que rien ne le signale.
//  2. LISTE NOMMÉE : l'ensemble des routes d'écriture (POST/PUT/PATCH/DELETE) que la garde
//     LAISSE PASSER en démo est égal, à la route près, à demoLetThroughRoutes. Une route
//     d'écriture NOUVELLE est refusée par défaut (rien à faire) ; une route qui tomberait
//     dans une famille de lecture (`/pages/`, `/filters/`, `/tactical/`…) fait échouer ce
//     test tant qu'elle n'est pas relue et ajoutée ici.
//
// La décision est prise par la garde elle-même (middleware.DemoReadOnly appliquée à un
// handler témoin), jamais par une seconde classification écrite dans ce test.
package api_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/middleware"
)

// demoWorkerPrefix : le préfixe du protocole ouvrier, exempté comme du CSRF
// (apiV1InternalBasePath dans server_apiv1.go).
const demoWorkerPrefix = "/api/v1/internal"

// demoLetThroughRoutes : les SEULES écritures que la démo laisse passer, chacune avec sa
// raison (lecture produit en POST, état de session du visiteur, protocole ouvrier à jeton).
var demoLetThroughRoutes = []string{
	// Protocole ouvrier : jeton Bearer dédié, aucun cookie ; 503 en démo (aucun jeton).
	"POST /api/v1/internal/build-queue/artifact",
	"POST /api/v1/internal/build-queue/claim",
	"POST /api/v1/internal/build-queue/complete",
	"POST /api/v1/internal/build-queue/heartbeat",
	// Lectures produit en POST (filtres en corps) — middleware.IsMutatingRequest.
	"POST /api/v1/players/{player_slug}/engagement/timeseries",
	"POST /api/v1/players/{player_slug}/filters/match-ids",
	"POST /api/v1/players/{player_slug}/filters/resolve",
	"POST /api/v1/players/{player_slug}/pages/citations",
	"POST /api/v1/players/{player_slug}/pages/commendations",
	"POST /api/v1/players/{player_slug}/pages/compare",
	"POST /api/v1/players/{player_slug}/pages/explorer/matches-query",
	"POST /api/v1/players/{player_slug}/pages/explorer/player-query",
	"POST /api/v1/players/{player_slug}/pages/match-history/query",
	"POST /api/v1/players/{player_slug}/pages/medals",
	"POST /api/v1/players/{player_slug}/pages/media",
	"POST /api/v1/players/{player_slug}/pages/palmares/relations",
	"POST /api/v1/players/{player_slug}/pages/palmares/relations/moments",
	"POST /api/v1/players/{player_slug}/pages/sessions/detail",
	"POST /api/v1/players/{player_slug}/pages/stats/query",
	"POST /api/v1/players/{player_slug}/pages/synthesis",
	"POST /api/v1/players/{player_slug}/pages/teammates",
	"POST /api/v1/players/{player_slug}/pages/timeseries",
	"POST /api/v1/players/{player_slug}/pages/trends",
	"POST /api/v1/players/{player_slug}/tactical/maps",
	"POST /api/v1/players/{player_slug}/tactical/{map_id}/cellule",
	"POST /api/v1/players/{player_slug}/tactical/{map_id}/raster",
	// État de navigation du visiteur, écrit sous <démo>/runtime/sessions.
	"POST /api/v1/session/context",
}

// demoRefusedWitnesses : écritures relevées au backlog comme ouvertes en démo avant la garde
// générale. Elles doivent exister ET être refusées.
var demoRefusedWitnesses = []string{
	"PATCH /api/v1/profiles/{player_slug}/titles/{slug}/sync",
	"POST /api/v1/watcher/auth/start",
	"POST /api/v1/admin/actions/replay-build/run",
	"DELETE /api/v1/admin/users/{username}",
	"POST /api/v1/settings/backup/run",
}

var demoWriteMethods = map[string]bool{
	http.MethodPost: true, http.MethodPut: true, http.MethodPatch: true, http.MethodDelete: true,
}

// concretePath remplace les paramètres de motif par une valeur : la garde raisonne sur le
// chemin demandé, pas sur le motif.
func concretePath(route string) string {
	parts := strings.Split(route, "/")
	for i, p := range parts {
		if strings.HasPrefix(p, "{") || p == "*" {
			parts[i] = "x"
		}
	}
	return strings.Join(parts, "/")
}

// demoGuardLetsThrough applique la VRAIE garde à un handler témoin.
func demoGuardLetsThrough(method, route string) bool {
	reached := false
	h := middleware.DemoReadOnly(true, demoWorkerPrefix)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		reached = true
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(method, concretePath(route), nil))
	return reached
}

func routeHasDemoGuard(mws []func(http.Handler) http.Handler) bool {
	for _, mw := range mws {
		if strings.Contains(runtime.FuncForPC(reflect.ValueOf(mw).Pointer()).Name(), "middleware.DemoReadOnly") {
			return true
		}
	}
	return false
}

func TestDemoReadOnlyRatchet(t *testing.T) {
	t.Setenv("LEVELUP_DEMO_MODE", "true")
	t.Setenv("PRESTIGE_ENABLED", "true")
	r, ok := buildTestRouter(t).(chi.Router)
	if !ok {
		t.Fatal("le routeur n'est pas un chi.Router")
	}

	var unguarded, letThrough []string
	seen := map[string]bool{}
	writes := 0
	err := chi.Walk(r, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		key := method + " " + route
		seen[key] = true
		if !routeHasDemoGuard(mws) {
			unguarded = append(unguarded, key)
		}
		if !demoWriteMethods[method] {
			return nil
		}
		writes++
		if demoGuardLetsThrough(method, route) {
			letThrough = append(letThrough, key)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}
	if writes < 50 {
		t.Fatalf("%d route(s) d'écriture walkées : le routeur assemblé ne ressemble pas à celui de production", writes)
	}
	if len(unguarded) > 0 {
		sort.Strings(unguarded)
		t.Errorf("%d route(s) SANS la garde démo dans leur chaîne :\n  %s", len(unguarded), strings.Join(unguarded, "\n  "))
	}

	sort.Strings(letThrough)
	want := append([]string(nil), demoLetThroughRoutes...)
	sort.Strings(want)
	if !reflect.DeepEqual(letThrough, want) {
		t.Errorf("écritures laissées passer en démo ≠ liste nommée.\n  obtenu  : %s\n  attendu : %s",
			strings.Join(letThrough, "\n            "), strings.Join(want, "\n            "))
	}

	for _, w := range demoRefusedWitnesses {
		parts := strings.SplitN(w, " ", 2)
		if !seen[w] {
			t.Errorf("témoin %q absent du routeur : la liste des témoins est périmée", w)
			continue
		}
		if demoGuardLetsThrough(parts[0], parts[1]) {
			t.Errorf("témoin %q laissé passer en démo", w)
		}
	}
}
