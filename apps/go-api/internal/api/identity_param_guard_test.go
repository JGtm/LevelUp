//go:build cgo

// identity_param_guard_test.go — ratchet « paramètre d'identité joueur hors du groupe
// joueur » (registre sécurité C4-7, 2026-10-15).
//
// POURQUOI. /healthz/home?player= construisait la home d'un joueur quelconque derrière
// RequireAuth seul : un compte ordinaire lisait le profil d'autrui. Le groupe
// /players/{player_slug} porte la garde de propriété (family_write_routes_test.go) ;
// hors de lui, une route qui désigne un joueur par un paramètre doit porter elle-même
// la propriété (RequirePlayerOwnership) ou l'admin (RequireAdmin).
//
// RÈGLE. Toute opération du document OpenAPI partagé, hors /api/v1/players/{player_slug},
// qui reçoit un paramètre (chemin ou requête) nommé player, player_slug, xuid ou
// gamertag porte RequirePlayerOwnership ou RequireAdmin dans sa chaîne chi. Une
// exception porte sa date et sa raison dans identityParamAllowlist.
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

// identityParamNames : noms de paramètre qui désignent un joueur.
var identityParamNames = map[string]bool{"player": true, "player_slug": true, "xuid": true, "gamertag": true}

// identityParamAllowlist : "MÉTHODE chemin" -> date et justification.
var identityParamAllowlist = map[string]string{
	// 2026-10-15 : {xuid} désigne le membre à RETIRER d'un groupe ; la décision est
	// « propriétaire du groupe » (GroupsHandler.requireOwnedGroup), pas la propriété
	// d'un profil joueur. RequireAuth + identité Halo liée exigés par le handler.
	"DELETE /api/v1/groups/{id}/members/{xuid}": "propriétaire du groupe seulement (handler)",
	// 2026-10-15 : ?xuid= n'agit qu'avec only_played ET un lecteur partagé ; le
	// CatalogRepo est câblé sans lecteur partagé (server_apiv1.go, NewCatalogRepo(..., nil)) :
	// filtre inactif, catalogue du titre seul (registre sécurité, constat écarté).
	"GET /api/v1/titles/{slug}/catalog/maps":      "filtre xuid inactif (lecteur partagé non câblé)",
	"GET /api/v1/titles/{slug}/catalog/playlists": "filtre xuid inactif (lecteur partagé non câblé)",
}

func TestIdentityParamsRequireOwnershipOrAdmin(t *testing.T) {
	t.Setenv("PRESTIGE_ENABLED", "true")
	router, doc := buildTestRouterWithDoc(t)
	r, ok := router.(chi.Router)
	if !ok {
		t.Fatal("le routeur n'est pas un chi.Router")
	}
	chains := map[string]string{}
	err := chi.Walk(r, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		var names []string
		for _, mw := range mws {
			names = append(names, runtime.FuncForPC(reflect.ValueOf(mw).Pointer()).Name())
		}
		chains[method+" "+route] = strings.Join(names, " ")
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk : %v", err)
	}

	var violations []string
	checked := 0
	for path, item := range doc.Paths {
		if strings.HasPrefix(path, playerScopedPrefix) {
			continue
		}
		for method, op := range pathItemOperations(item) {
			designates := false
			for _, p := range op.Parameters {
				designates = designates || identityParamNames[p.Name]
			}
			if !designates {
				continue
			}
			checked++
			key := method + " " + path
			if _, allowed := identityParamAllowlist[key]; allowed {
				continue
			}
			chain, walked := chains[key]
			if !walked {
				violations = append(violations, key+" (absente de chi.Walk)")
				continue
			}
			if !strings.Contains(chain, "RequirePlayerOwnership") && !strings.Contains(chain, "RequireAdmin") {
				violations = append(violations, key)
			}
		}
	}
	if checked == 0 {
		t.Fatal("aucune opération à paramètre d'identité vue : l'analyse ne lit plus le document")
	}
	sort.Strings(violations)
	for _, v := range violations {
		t.Errorf("%s : désigne un joueur sans RequirePlayerOwnership ni RequireAdmin", v)
	}
	for key := range identityParamAllowlist {
		if _, walked := chains[key]; !walked {
			t.Errorf("allowlist périmée : %s n'existe plus", key)
		}
	}
}
