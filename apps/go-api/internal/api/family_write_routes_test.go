//go:build cgo

// family_write_routes_test.go — Accès famille en LECTURE seule, sur TOUTES les routes
// du routeur (ADR 0029, décision utilisateur du 2026-10-15).
//
// COMMENT. Le routeur réel est assemblé en mode démo (cf. bare_routes_ratchet_test.go :
// le mode appliqué ne s'assemble pas sans services réels). chi.Walk en donne la liste
// des routes des groupes /players/{player_slug} et /profiles/{player_slug}/... ; chaque
// route doit porter la garde de propriété (closure nommée RequirePlayerOwnership).
// Chaque route est ensuite REJOUÉE, méthode et chemin réels, à travers la garde en
// mode appliqué : alice, co-membre de groupe de bob, reçoit 403 sur toute écriture du
// profil de bob et passe sur toute lecture. La classification lecture / écriture est
// celle de la garde (middleware.IsMutatingRequest), jamais une seconde liste.
package api_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
)

// ownershipScopedPrefixes : les groupes dont toutes les routes portent la garde.
var ownershipScopedPrefixes = []string{
	"/api/v1/players/{player_slug}",
	"/api/v1/profiles/{player_slug}/",
}

type familyUsers struct{}

func (familyUsers) Get(username string) (*domain.User, error) {
	if username == "alice" {
		return &domain.User{Username: "alice", Role: domain.RoleUser, XUID: "222"}, nil
	}
	return nil, errors.New("inconnu")
}

func (familyUsers) GetByXUID(string) (*domain.User, error) { return nil, errors.New("inconnu") }

// ownershipRoute : une route walkée et son chemin concret (concretePath, partagé avec
// le ratchet démo : tout paramètre vaut « x », et le profil « x » est celui de bob).
type ownershipRoute struct {
	method, pattern, path string
}

func walkOwnershipRoutes(t *testing.T) []ownershipRoute {
	t.Helper()
	t.Setenv("PRESTIGE_ENABLED", "true")
	r, ok := buildTestRouter(t).(chi.Router)
	if !ok {
		t.Fatal("le routeur n'est pas un chi.Router")
	}
	var routes []ownershipRoute
	var unguarded []string
	err := chi.Walk(r, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		scoped := false
		for _, p := range ownershipScopedPrefixes {
			scoped = scoped || strings.HasPrefix(route, p)
		}
		if !scoped {
			return nil
		}
		hasGuard := false
		for _, mw := range mws {
			name := runtime.FuncForPC(reflect.ValueOf(mw).Pointer()).Name()
			hasGuard = hasGuard || strings.Contains(name, "RequirePlayerOwnership")
		}
		if !hasGuard {
			unguarded = append(unguarded, method+" "+route)
		}
		routes = append(routes, ownershipRoute{method: method, pattern: route, path: concretePath(route)})
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk : %v", err)
	}
	if len(unguarded) > 0 {
		t.Fatalf("route(s) sans garde de propriété : %v", unguarded)
	}
	if len(routes) < 100 {
		t.Fatalf("%d route(s) walkée(s) : le routeur de test ne porte pas la surface attendue", len(routes))
	}
	sort.Slice(routes, func(i, j int) bool { return routes[i].pattern+routes[i].method < routes[j].pattern+routes[j].method })
	return routes
}

// enforcedOwnershipRouter : les mêmes motifs, derrière la garde en mode appliqué,
// jusqu'à un handler terminal qui répond 200.
func enforcedOwnershipRouter(routes []ownershipRoute) http.Handler {
	access := middleware.NewPlayerAccess(false, "password",
		func(_ context.Context, slug string) (string, bool) {
			x, ok := map[string]string{"alice": "222", "x": "999"}[slug]
			return x, ok
		},
		familyUsers{},
		func(context.Context) map[string]bool { return map[string]bool{"222": true, "999": true} })
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			user := "alice"
			sess := &domain.SessionData{SessionID: "s", Username: &user}
			next.ServeHTTP(w, req.WithContext(middleware.InjectSession(req.Context(), sess)))
		})
	})
	ok := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequirePlayerOwnership(access))
		for _, rt := range routes {
			r.MethodFunc(rt.method, rt.pattern, ok)
		}
	})
	return r
}

// TestFamilyMemberCannotWriteAnyPlayerRoute : co-membre × toutes les routes des
// groupes joueur → 403 sur chaque écriture, 200 sur chaque lecture.
func TestFamilyMemberCannotWriteAnyPlayerRoute(t *testing.T) {
	routes := walkOwnershipRoutes(t)
	router := enforcedOwnershipRouter(routes)
	writes := 0
	for _, rt := range routes {
		req := httptest.NewRequest(rt.method, rt.path, strings.NewReader("{}"))
		write := middleware.IsMutatingRequest(req)
		want := http.StatusOK
		if write {
			want = http.StatusForbidden
			writes++
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != want {
			t.Errorf("%s %s (écriture=%v) : statut %d, attendu %d", rt.method, rt.pattern, write, w.Code, want)
		}
		if write {
			t.Logf("écriture refusée au co-membre : %s %s", rt.method, rt.pattern)
		}
	}
	if writes < 40 {
		t.Fatalf("%d route(s) d'écriture seulement : la classification ne voit plus les écritures", writes)
	}
}
