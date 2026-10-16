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
// celle de la garde (middleware.IsMutatingRequest), jamais une seconde liste ; les
// seules écritures ouvertes au co-membre sont les actions du visiteur, nommées dans
// familyVisitorWrites.
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
	// ownerOnly : la route porte la garde de route RequirePlayerOwner (donnée privée du
	// profil, propriétaire ou admin seulement, lecture comprise).
	ownerOnly bool
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
		hasGuard, ownerOnly := false, false
		for _, mw := range mws {
			name := runtime.FuncForPC(reflect.ValueOf(mw).Pointer()).Name()
			hasGuard = hasGuard || strings.Contains(name, "RequirePlayerOwnership")
			ownerOnly = ownerOnly || strings.Contains(name, "RequirePlayerOwner.")
		}
		if !hasGuard {
			unguarded = append(unguarded, method+" "+route)
		}
		routes = append(routes, ownershipRoute{method: method, pattern: route, path: concretePath(route), ownerOnly: ownerOnly})
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

// enforcedOwnershipRouter : les mêmes motifs, derrière la garde en mode appliqué (et la
// garde du propriétaire sur les routes qui la portent), jusqu'à un handler terminal qui
// répond 200.
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
			if rt.ownerOnly {
				r.With(middleware.RequirePlayerOwner(access)).MethodFunc(rt.method, rt.pattern, ok)
				continue
			}
			r.MethodFunc(rt.method, rt.pattern, ok)
		}
	})
	return r
}

// familyVisitorWrites : les écritures ouvertes au co-membre, chacune une décision de
// l'utilisateur consignée dans l'ADR 0029. Liste exacte : une entrée qui ne désigne plus
// une route walkée fait échouer le test.
var familyVisitorWrites = map[string]bool{
	// Like d'un média depuis le profil d'un ami : action du visiteur, attribuée à son
	// propre profil (décision utilisateur du 2026-10-16).
	"PATCH /api/v1/players/{player_slug}/media/likes": true,
}

// TestFamilyMemberCannotWriteAnyPlayerRoute : co-membre × toutes les routes des
// groupes joueur → 403 sur chaque écriture hors familyVisitorWrites et sur toute route
// privée au propriétaire (lecture comprise), 200 sur chaque autre lecture et sur chaque
// action du visiteur.
func TestFamilyMemberCannotWriteAnyPlayerRoute(t *testing.T) {
	routes := walkOwnershipRoutes(t)
	router := enforcedOwnershipRouter(routes)
	writes := 0
	seenVisitor := map[string]bool{}
	for _, rt := range routes {
		req := httptest.NewRequest(rt.method, rt.path, strings.NewReader("{}"))
		write := middleware.IsMutatingRequest(req)
		want := http.StatusOK
		key := rt.method + " " + rt.pattern
		switch {
		case rt.ownerOnly:
			want = http.StatusForbidden
			if write {
				writes++
			}
		case write && familyVisitorWrites[key]:
			seenVisitor[key] = true
		case write:
			want = http.StatusForbidden
			writes++
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != want {
			t.Errorf("%s %s (écriture=%v) : statut %d, attendu %d", rt.method, rt.pattern, write, w.Code, want)
		}
		if want == http.StatusForbidden {
			t.Logf("refusé au co-membre : %s %s", rt.method, rt.pattern)
		}
	}
	for key := range familyVisitorWrites {
		if !seenVisitor[key] {
			t.Errorf("action du visiteur %q : aucune route d'écriture walkée ne la porte (liste périmée)", key)
		}
	}
	if writes < 40 {
		t.Fatalf("%d route(s) d'écriture seulement : la classification ne voit plus les écritures", writes)
	}
}

// ownerOnlyRouteMarkers : sous-chemins du groupe joueur dont les données sont privées au
// propriétaire du profil, lecture comprise (ADR 0029, décision utilisateur du
// 2026-10-16 pour les notifications). Toute route walkée sous l'un d'eux doit porter la
// garde de route RequirePlayerOwner.
var ownerOnlyRouteMarkers = []string{
	"/api/v1/players/{player_slug}/notifications",
}

// TestOwnerOnlyRoutesCarryOwnerGuard : chaque route des notifications porte la garde du
// propriétaire, quelle que soit sa méthode ; une route neuve qui l'oublierait échoue ici.
func TestOwnerOnlyRoutesCarryOwnerGuard(t *testing.T) {
	routes := walkOwnershipRoutes(t)
	seen := 0
	for _, rt := range routes {
		for _, marker := range ownerOnlyRouteMarkers {
			if rt.pattern != marker && !strings.HasPrefix(rt.pattern, marker+"/") {
				continue
			}
			seen++
			if !rt.ownerOnly {
				t.Errorf("%s %s : donnée privée du profil sans RequirePlayerOwner", rt.method, rt.pattern)
			}
		}
	}
	if seen < 9 {
		t.Fatalf("%d route(s) de notifications walkée(s), attendu au moins 9", seen)
	}
}
