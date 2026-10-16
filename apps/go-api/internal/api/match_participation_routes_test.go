//go:build cgo

// match_participation_routes_test.go — couche B de l'ADR 0029 sur TOUTES les routes d'un
// match : /players/{player_slug}/matches/{match_id}/... (registre sécurité C4-8 ; le
// rejeu est public en production).
//
// Le routeur réel (démo) est walké : chaque route dont le motif porte
// /matches/{match_id} doit porter RequireMatchParticipation. Chacune est ensuite
// rejouée derrière ce middleware avec une vérification qui répond « non joué » :
// 404 match_not_participant partout. La séquence d'une mort (POST
// /tactical/{map_id}/sequence, match_id dans le corps) a sa propre vérification
// d'ouvrabilité dans le service (TestSequence_Indisponible, cas « match d'un autre
// joueur ») : hors de ce motif.
package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/middleware"
)

const matchScopedMarker = "/api/v1/players/{player_slug}/matches/{match_id}"

func TestMatchRoutesRequireParticipation(t *testing.T) {
	t.Setenv("PRESTIGE_ENABLED", "true")
	r, ok := buildTestRouter(t).(chi.Router)
	if !ok {
		t.Fatal("le routeur n'est pas un chi.Router")
	}
	type route struct{ method, pattern string }
	var routes []route
	err := chi.Walk(r, func(method, pattern string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		if !strings.HasPrefix(pattern, matchScopedMarker) {
			return nil
		}
		routes = append(routes, route{method, pattern})
		for _, mw := range mws {
			if strings.Contains(runtime.FuncForPC(reflect.ValueOf(mw).Pointer()).Name(), "RequireMatchParticipation") {
				return nil
			}
		}
		t.Errorf("%s %s sans RequireMatchParticipation", method, pattern)
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk : %v", err)
	}
	if len(routes) < 10 {
		t.Fatalf("%d route(s) de match walkée(s) : surface attendue absente", len(routes))
	}

	notPlayed := middleware.RequireMatchParticipation(func(context.Context, string, string) (bool, error) {
		return false, nil
	})
	probe := chi.NewRouter()
	probe.Group(func(g chi.Router) {
		for _, rt := range routes {
			g.With(notPlayed).MethodFunc(rt.method, rt.pattern, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
		}
	})
	for _, rt := range routes {
		req := httptest.NewRequest(rt.method, concretePath(rt.pattern), strings.NewReader("{}"))
		w := httptest.NewRecorder()
		probe.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "match_not_participant") {
			t.Errorf("%s %s : statut %d (%s), attendu 404 match_not_participant", rt.method, rt.pattern, w.Code, w.Body.String())
		}
	}
}
