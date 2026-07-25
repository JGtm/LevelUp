//go:build cgo

// route_collision_test.go — garde-rail SYSTÉMIQUE anti-collision de routes.
//
// POURQUOI. HomeHandler et PrestigeHandler enregistraient tous deux la même route
// GET /players/{player_slug}/challenges sur le MÊME sous-routeur chi ; monté en
// dernier, prestige écrasait silencieusement home (aucune erreur, aucun panic). chi
// REMPLACE l'endpoint d'un couple (méthode, motif) déjà enregistré : un chi.Walk de
// l'arbre assemblé ne voit alors qu'UNE occurrence et ne peut donc PAS détecter
// l'écrasement (vérifié empiriquement, chi v5). Ce garde-rail capture les
// enregistrements AU MOMENT où ils se produisent (hook humacore.OnAPICreatedRouter)
// et échoue si un même (routeur, méthode, chemin) est enregistré >= 2 fois — c'est
// ce test qui aurait attrapé la collision, et qui attrapera les prochaines.
//
// DEUX niveaux de couverture, complémentaires :
//
//	1. TestNoDuplicateRouteRegistration — SYSTÉMIQUE : monte le VRAI routeur assemblé
//	   (buildTestRouter) et vérifie qu'aucun (méthode, chemin) n'est enregistré deux
//	   fois sur un même sous-routeur. Couvre home + tous les handlers effectivement
//	   montés. NB : en mode démo, le bundle Prestige échoue à s'initialiser (pas de
//	   DuckDB) → les routes prestige NE SONT PAS montées ici ; c'est le point (2) qui
//	   couvre la paire home/prestige du bug corrigé.
//	2. TestHomeAndPrestigeShareNoRoute — CIBLÉ : monte HomeHandler ET PrestigeHandler
//	   sur le MÊME sous-routeur (comme le groupe /players/{player_slug} en prod) via
//	   leurs vraies méthodes Mount (deps nil : Mount n'enregistre que des method
//	   values, aucun accès service/DB) et vérifie zéro collision. ROUGE avant le fix
//	   (les deux enregistraient GET /challenges), VERT après.
//
// LIMITE connue. La détection groupe par IDENTITÉ DE ROUTEUR (pointeur du *chi.Mux
// passé à NewAPI). Deux handlers montés via des wrappers r.With(...)/r.Group(...)
// DISTINCTS partagent le même arbre chi mais exposent des pointeurs différents : une
// collision entre ces deux-là ne serait pas vue. Le cas dominant — et celui du bug
// corrigé — est plusieurs .Mount(r) sur le même r de groupe, entièrement couvert.
// Les routes chi directes (r.Get/r.Post hors Huma) ne passent pas par le hook.

package api_test

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/humacore"
)

// routeRegKey identifie une opération : routeur (par pointeur) + méthode + chemin.
type routeRegKey struct {
	router uintptr
	method string
	path   string
}

// capturedAPI : paire (identité de routeur, huma.API) relevée par le hook au moment
// de NewAPI. Les Paths de l'api ne sont lus qu'APRÈS le montage complet (mutation en
// place par huma.Get/Post).
type capturedAPI struct {
	rid uintptr
	api huma.API
}

// routeRegistry accumule les enregistrements Huma pour détecter les doublons.
type routeRegistry struct {
	ops map[routeRegKey][]string // clé → operationIDs enregistrées (pour le diagnostic)
}

func newRouteRegistry() *routeRegistry {
	return &routeRegistry{ops: map[routeRegKey][]string{}}
}

// routerID : adresse stable du *chi.Mux sous-jacent = identité du routeur.
func routerID(r chi.Router) uintptr {
	v := reflect.ValueOf(r)
	if v.Kind() == reflect.Pointer {
		return v.Pointer()
	}
	return 0
}

// record enregistre toutes les opérations déclarées par une huma.API sur le routeur
// d'identité `rid`. À N'APPELER qu'APRÈS le montage complet : le hook capture la
// paire (routeur, api) AU MOMENT de NewAPI (avant tout huma.Get/Post), mais l'api est
// mutée en place par les enregistrements suivants — ses Paths ne sont peuplés qu'une
// fois tous les .Mount terminés (même invariant que openapi_schema_drift_test).
func (rr *routeRegistry) record(rid uintptr, a huma.API) {
	oapi := a.OpenAPI()
	if oapi == nil || oapi.Paths == nil {
		return
	}
	for path, pi := range oapi.Paths {
		if pi == nil {
			continue
		}
		for method, op := range pathItemOperations(pi) {
			opID := ""
			if op != nil {
				opID = op.OperationID
			}
			k := routeRegKey{router: rid, method: method, path: path}
			rr.ops[k] = append(rr.ops[k], opID)
		}
	}
}

// duplicates retourne la liste triée des (méthode, chemin) enregistrés >= 2 fois sur
// un même routeur (écrasement chi silencieux).
func (rr *routeRegistry) duplicates() []string {
	var dups []string
	for k, ids := range rr.ops {
		if len(ids) > 1 {
			dups = append(dups, fmt.Sprintf("%s %s enregistré %d fois sur le même sous-routeur (ops: %v)",
				k.method, k.path, len(ids), ids))
		}
	}
	sort.Strings(dups)
	return dups
}

// total : nombre d'opérations distinctes capturées (sanity du hook).
func (rr *routeRegistry) total() int { return len(rr.ops) }

// has : vrai si un (méthode, chemin) est enregistré sur au moins un routeur capturé.
func (rr *routeRegistry) has(method, path string) bool {
	for k := range rr.ops {
		if k.method == method && k.path == path {
			return true
		}
	}
	return false
}

// pathItemOperations mappe méthode HTTP → *Operation pour un PathItem Huma.
func pathItemOperations(pi *huma.PathItem) map[string]*huma.Operation {
	m := map[string]*huma.Operation{}
	if pi.Get != nil {
		m["GET"] = pi.Get
	}
	if pi.Post != nil {
		m["POST"] = pi.Post
	}
	if pi.Put != nil {
		m["PUT"] = pi.Put
	}
	if pi.Patch != nil {
		m["PATCH"] = pi.Patch
	}
	if pi.Delete != nil {
		m["DELETE"] = pi.Delete
	}
	if pi.Head != nil {
		m["HEAD"] = pi.Head
	}
	if pi.Options != nil {
		m["OPTIONS"] = pi.Options
	}
	if pi.Trace != nil {
		m["TRACE"] = pi.Trace
	}
	return m
}

// TestNoDuplicateRouteRegistration : sur le VRAI routeur assemblé, aucun couple
// (méthode, chemin) n'est enregistré deux fois sur le même sous-routeur chi. C'est
// le garde-rail qui aurait attrapé la collision home/prestige GET /challenges.
func TestNoDuplicateRouteRegistration(t *testing.T) {
	// Flags ON pour exposer le MAXIMUM de routes (prestige + multi-titre), comme
	// contract_test — la surface complète doit être vérifiée.
	t.Setenv("LEVELUP_DEMO_MODE", "true")
	t.Setenv("MULTI_TITLE_API_ENABLED", "true")
	t.Setenv("PRESTIGE_ENABLED", "true")

	// V72-01 / H1 : sous le document PARTAGÉ, huma écrase silencieusement une
	// collision (méthode, chemin) — elle n'est PLUS lisible dans le document final
	// (une seule opération survit). On observe donc CHAQUE enregistrement via
	// OnOperationRegistered (chemin ABSOLU) : deux enregistrements d'un même
	// (méthode, chemin absolu) = collision (écrasement chi silencieux). Détection
	// PLUS fine que l'ancien groupement par pointeur de routeur — elle couvre aussi
	// deux sous-groupes r.With/r.Group distincts d'un même arbre chi (l'ancienne
	// LIMITE documentée : préfixe absolu identique ⇒ même endpoint chi écrasé).
	type opKey struct{ method, path string }
	counts := map[opKey]int{}
	humacore.OnOperationRegistered = func(method, path string) {
		counts[opKey{method, path}]++
	}
	defer func() { humacore.OnOperationRegistered = nil }()

	_ = buildTestRouter(t) // déclenche tous les Mount → tous les NewAPI → observe

	total := len(counts)
	t.Logf("capture : %d opérations Huma distinctes (document partagé)", total)

	// Sanity : le hook a bien observé un grand nombre de routes. Sinon la verdeur
	// ci-dessous serait vide de sens.
	if total < 30 {
		t.Fatalf("capture anormalement faible : %d opérations (attendu >= 30) — hook/collecte cassés ?", total)
	}

	var dups []string
	for k, n := range counts {
		if n > 1 {
			dups = append(dups, fmt.Sprintf("%s %s enregistré %d fois (écrasement chi silencieux)", k.method, k.path, n))
		}
	}
	sort.Strings(dups)
	if len(dups) > 0 {
		t.Errorf("%d collision(s) de route détectée(s) — deux handlers enregistrent le même "+
			"(méthode, chemin absolu) sur le même arbre chi. Déplacer l'une des routes sous "+
			"un préfixe non ambigu :", len(dups))
		for _, d := range dups {
			t.Errorf("  - %s", d)
		}
	}
}

// TestHomeAndPrestigeShareNoRoute : garde-rail CIBLÉ de la régression corrigée. Monte
// HomeHandler ET PrestigeHandler sur le MÊME sous-routeur (le groupe
// /players/{player_slug} en prod) via leurs vraies méthodes Mount, et vérifie
// qu'aucun (méthode, chemin) n'est enregistré par les deux. C'est le seul test de
// routeur où les routes prestige sont effectivement montées (le bundle prestige est
// indisponible en mode démo, cf. TestNoDuplicateRouteRegistration). ROUGE avant le
// fix (les deux exposaient GET /challenges), VERT après (prestige → /prestige/...).
//
// deps nil : Mount n'enregistre que des method values (h.CreateChallenge, …) sans
// toucher au service ni à une DB ; la dérivation de schéma Huma ne lit que les types
// d'E/S.
func TestHomeAndPrestigeShareNoRoute(t *testing.T) {
	var caps []capturedAPI
	humacore.OnAPICreatedRouter = func(rt chi.Router, a huma.API) {
		caps = append(caps, capturedAPI{rid: routerID(rt), api: a})
	}
	defer func() { humacore.OnAPICreatedRouter = nil }()

	home := handlers.NewHomeHandler(nil, nil)
	prestige := handlers.NewPrestigeHandler(nil, nil)

	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(sr chi.Router) {
		home.Mount(sr)     // prod : server_apiv1.go (~638)
		prestige.Mount(sr) // prod : même sous-routeur (~821)
	})

	rr := newRouteRegistry()
	for _, c := range caps {
		rr.record(c.rid, c.api)
	}

	if dups := rr.duplicates(); len(dups) > 0 {
		t.Errorf("HomeHandler et PrestigeHandler enregistrent %d route(s) identique(s) sur le "+
			"groupe /players/{player_slug} (collision → écrasement chi silencieux) :", len(dups))
		for _, d := range dups {
			t.Errorf("  - %s", d)
		}
	}

	// Encode le fix : les défis prestige vivent désormais sous /prestige/challenges,
	// et plus aucune route bare /challenges ne subsiste (source de la collision avec
	// la home). Couvre l'intention de l'ex-smoke test prestige (routes enregistrées).
	if !rr.has("GET", "/prestige/challenges") || !rr.has("POST", "/prestige/challenges") {
		t.Error("routes défis prestige attendues sous /prestige/challenges (GET+POST) — absentes")
	}
	if rr.has("GET", "/challenges") {
		t.Error("route bare GET /challenges encore enregistrée — la collision home/prestige n'est pas éliminée")
	}
}

// TestRouteCollisionDetector_FiresOnDuplicate : contrôle POSITIF — prouve que le
// détecteur n'est pas vide et qu'il AURAIT échoué avant le fix. Il reproduit
// exactement le mécanisme du bug corrigé : deux huma.API sur le MÊME routeur chi
// enregistrant GET /challenges (home puis prestige). chi écrase silencieusement (une
// seule route survit à un chi.Walk), mais le détecteur, groupant par routeur, voit
// bien 2 enregistrements → collision.
func TestRouteCollisionDetector_FiresOnDuplicate(t *testing.T) {
	r := chi.NewRouter()

	// "home" : GET /challenges
	apiHome := humacore.NewAPI(r)
	huma.Get(apiHome, "/challenges", probeChallengesHandler)
	// "prestige" (monté après → écrase) : GET /challenges sur le MÊME routeur.
	apiPrestige := humacore.NewAPI(r)
	huma.Get(apiPrestige, "/challenges", probeChallengesHandler)

	rr := newRouteRegistry()
	rr.record(routerID(r), apiHome)
	rr.record(routerID(r), apiPrestige)

	dups := rr.duplicates()
	if len(dups) == 0 {
		t.Fatal("le détecteur n'a pas repéré la collision GET /challenges (mécanisme du bug home/prestige) — garde-rail vide, sans valeur")
	}
	found := false
	for _, d := range dups {
		if strings.Contains(d, "GET /challenges") {
			found = true
		}
	}
	if !found {
		t.Fatalf("collision attendue sur GET /challenges, obtenu : %v", dups)
	}
}

// probeChallengesInput/Output : E/S minimales pour enregistrer une route de sonde
// via huma.Get dans le contrôle positif (le chemin "/challenges" n'a aucun paramètre).
type probeChallengesInput struct{}

type probeChallengesOutput struct {
	Body struct {
		OK bool `json:"ok"`
	}
}

func probeChallengesHandler(_ context.Context, _ *probeChallengesInput) (*probeChallengesOutput, error) {
	return &probeChallengesOutput{}, nil
}
