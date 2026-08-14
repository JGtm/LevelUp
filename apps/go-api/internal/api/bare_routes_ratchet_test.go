//go:build cgo

// bare_routes_ratchet_test.go — Ratchet « routes nues » (lot V3, DC-8).
//
// POURQUOI. La revue de sécurité du lot S (route → garde) était un grep MANUEL.
// Elle a laissé passer GET /jobs/{job_id} monté sans garde (finding VF-3) : aucun
// garde-rail automatisé n'attrapait une route montée sur `r` nu. Ce test est ce
// garde-rail permanent. Il aurait attrapé /jobs.
//
// COMMENT (marquage des middlewares par NOM de fonction, robuste). On construit le
// VRAI routeur en mode DÉMO (boot propre, aucune dépendance de service réelle : le
// mode enforcement, lui, wire des services nil → panics/os.Exit au boot, cf. la
// tentative comportementale abandonnée). En démo les gardes sont des NO-OP au
// RUNTIME (RequireAuth(demo=true) renvoie `next`), MAIS le closure du middleware
// reste présent dans la chaîne exposée par `chi.Walk`. On l'identifie donc par le
// nom runtime de sa fonction (`runtime.FuncForPC`), qui encode le constructeur
// (`...RequireAuth.1`, `...RequirePlayerOwnership.func18`, etc.) — stable, non
// dépendant de l'OS. Une route dont la chaîne contient un garde d'auth est GARDÉE.
// Une route SANS garde doit figurer dans l'allowlist datée ci-dessous, sinon échec.
//
// PÉRIMÈTRE. Routes montées sous le routeur assemblé (`/api/v1`, `/health*`,
// `/static/*`). Le catch-all SPA `/*` (handler NotFound, non walkable) et les
// routes racine /auth/xbox/{login,callback} (mode "xbox" only) sont publics par
// conception, hors surface, couverts par LOT_S_ROUTE_GUARD_TABLE.
//
// MAINTENANCE. Route publique légitime = 1 ligne d'allowlist AVEC justification.
// Route qui doit être gardée = la monter sous une garde (elle disparaît de l'échec).
// Nouvelle route nue non justifiée = ROUGE.

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

// guardMarkers — sous-chaînes de nom de fonction des middlewares qui constituent
// une garde d'accès (auth / admin / ownership / loopback). Le nom runtime d'un
// middleware construit par `middleware.RequireAuth(...)` contient « RequireAuth ».
var guardMarkers = []string{
	"RequireAuth",
	"RequireAdmin",
	"RequirePlayerOwnership",
	"LoopbackOnly",
	// RequireWorkerToken (2026-08-14, piste F) — garde des routes /internal/
	// build-queue/*. Ce n'est pas une garde de SESSION : un ouvrier n'a ni compte
	// ni cookie, il présente un jeton dédié (LEVELUP_BUILD_WORKER_TOKEN) qui
	// n'ouvre QUE ces quatre routes. Sans jeton configuré, elles répondent 503.
	// Reconnue comme garde plutôt qu'allowlistée : ces routes ne sont PAS
	// publiques, et les déclarer telles serait faux.
	"RequireWorkerToken",
}

// publicRoutesAllowlist — routes légitimement accessibles sans garde d'auth.
// Établie 2026-07-07 (relevé comportemental croisé + marquage). Clé = « METHOD /route »
// exact (chi.Walk rend les paramètres {name}). DÉCROISSANTE : n'ajouter qu'avec raison.
var publicRoutesAllowlist = map[string]string{
	// Liveness / readiness (sondes infra, aucune donnée).
	"GET /health":  "sonde liveness",
	"GET /healthz": "sonde liveness",
	"GET /readyz":  "sonde readiness",
	// Auth-bootstrap (nécessaire AVANT login).
	"GET /api/v1/auth/device-flow/{attempt_id}": "polling device-flow (onboarding, pas de session)",
	"POST /api/v1/auth/device-flow/start":       "démarrage device-flow (onboarding) — POST protégé CSRF",
	"POST /api/v1/auth/login":                   "login local — POST protégé CSRF",
	"POST /api/v1/auth/logout":                  "logout — POST protégé CSRF",
	"POST /api/v1/auth/register":                "register (invite) — POST protégé CSRF",
	"POST /api/v1/auth/password":                "changement mot de passe (session) — POST protégé CSRF",
	"POST /api/v1/session/context":              "établit le contexte de session (avant login) — POST protégé CSRF",
	// Shell applicatif + annuaire (filtrés par ownership IN-SERVICE, pas au routeur).
	"GET /api/v1/bootstrap":                  "shell React ; available_players filtré par session dans BootstrapService",
	"GET /api/v1/players":                    "liste joueurs filtrée par ownership in-service (S4, BuildPlayersList)",
	"GET /api/v1/directory/gamertags/search": "annuaire gamertags (503 si shared DB absente) — référentiel",
	// Contenu public (markdown / version).
	"GET /api/v1/changelog":          "notes de version (markdown public)",
	"GET /api/v1/help/release-notes": "contenu public (README)",
	"GET /api/v1/media/feed-version": "numéro de version de flux (polling public)",
	// Référentiels d'images (assets — aucune identité).
	"GET /api/v1/assets/battlepass/{subdir}/*":                 "référentiel images battlepass",
	"GET /api/v1/assets/challenge-badge/{title_id}/{badge_id}": "référentiel images badges",
	"GET /api/v1/assets/maps/{title_id}/{map_id}/image":        "référentiel images maps",
	"GET /api/v1/assets/medals/{title_id}/{medal_id}/image":    "référentiel images médailles",
	"GET /api/v1/assets/spartan/{image_type}/{title_id}/*":     "référentiel images spartan",
	"GET /api/v1/assets/{title_id}/maps":                       "référentiel catalogue maps",
	"GET /api/v1/assets/{title_id}/medals":                     "référentiel catalogue médailles",
	"GET /api/v1/assets/{title_id}/weapons":                    "référentiel catalogue armes",
	// Référentiels de titre (labels / capacités, montés sans condition). NOTE :
	// les catalog/{playlists,pairs,maps} sont dep-gated (câblés seulement si le
	// catalog seasons est branché) → absents du routeur de test démo, donc non
	// couverts ici ; référentiels GET public par conception (LOT_S table §1).
	"GET /api/v1/titles/{slug}/capabilities":   "référentiel capacités titre",
	"GET /api/v1/titles/{slug}/feature-matrix": "référentiel matrice de features",
	"GET /api/v1/titles/{slug}/field-mappings": "référentiel labels de champs",
	// Assets statiques servis par le backend (file-server chi → TOUTES les méthodes).
	"GET /static/*":                   "assets statiques",
	"HEAD /static/*":                  "assets statiques (HEAD)",
	"OPTIONS /static/*":               "assets statiques (preflight)",
	"POST /static/*":                  "assets statiques (file-server chi, toutes méthodes)",
	"PUT /static/*":                   "assets statiques (file-server chi, toutes méthodes)",
	"PATCH /static/*":                 "assets statiques (file-server chi, toutes méthodes)",
	"DELETE /static/*":                "assets statiques (file-server chi, toutes méthodes)",
	"CONNECT /static/*":               "assets statiques (file-server chi, toutes méthodes)",
	"TRACE /static/*":                 "assets statiques (file-server chi, toutes méthodes)",
	"QUERY /static/*":                 "assets statiques (file-server chi, toutes méthodes) — méthode QUERY ajoutée par go-chi 5.3.1 (bump 2026-08-02)",
	"GET /static/commendations/*":     "assets statiques commendations",
	"HEAD /static/commendations/*":    "assets statiques commendations (HEAD)",
	"OPTIONS /static/commendations/*": "assets statiques commendations (preflight)",
	"POST /static/commendations/*":    "assets statiques commendations (file-server chi, toutes méthodes)",
	"PUT /static/commendations/*":     "assets statiques commendations (file-server chi, toutes méthodes)",
	"PATCH /static/commendations/*":   "assets statiques commendations (file-server chi, toutes méthodes)",
	"DELETE /static/commendations/*":  "assets statiques commendations (file-server chi, toutes méthodes)",
	"CONNECT /static/commendations/*": "assets statiques commendations (file-server chi, toutes méthodes)",
	"TRACE /static/commendations/*":   "assets statiques commendations (file-server chi, toutes méthodes)",
	"QUERY /static/commendations/*":   "assets statiques commendations (file-server chi, toutes méthodes) — méthode QUERY ajoutée par go-chi 5.3.1 (bump 2026-08-02)",
}

// routeIsGuarded — vrai si la chaîne de middlewares de la route contient au moins
// un garde d'auth (identifié par le nom runtime de la fonction du closure).
func routeIsGuarded(mws []func(http.Handler) http.Handler) bool {
	for _, mw := range mws {
		name := runtime.FuncForPC(reflect.ValueOf(mw).Pointer()).Name()
		for _, marker := range guardMarkers {
			if strings.Contains(name, marker) {
				return true
			}
		}
	}
	return false
}

// TestBareRoutesRatchet_NoUnguardedRouteOutsideAllowlist : aucune route montée
// sans garde d'auth hors allowlist datée. Garde-rail permanent de VF-3.
func TestBareRoutesRatchet_NoUnguardedRouteOutsideAllowlist(t *testing.T) {
	// Flags ON pour exposer les routes conditionnelles (référentiels titre, prestige),
	// comme le contract_test — la surface publique doit être couverte au complet.
	t.Setenv("LEVELUP_DEMO_MODE", "true")
	t.Setenv("PRESTIGE_ENABLED", "true")

	router := buildTestRouter(t)
	r, ok := router.(chi.Router)
	if !ok {
		t.Fatalf("le routeur n'est pas un chi.Router (%T)", router)
	}

	seen := make(map[string]bool)     // routes walkées (self-check)
	hitAllow := make(map[string]bool) // entrées d'allowlist rencontrées
	var bare []string                 // routes publiques hors allowlist (violations)

	err := chi.Walk(r, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		key := method + " " + route
		seen[key] = true
		if routeIsGuarded(mws) {
			return nil
		}
		if _, allowed := publicRoutesAllowlist[key]; allowed {
			hitAllow[key] = true
			return nil
		}
		bare = append(bare, key)
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}

	if len(bare) > 0 {
		sort.Strings(bare)
		t.Errorf("%d route(s) NUE(S) hors allowlist (montées sans garde d'auth) — "+
			"les garder sous RequireAuth/RequireAdmin/ownership/LoopbackOnly OU les "+
			"ajouter à publicRoutesAllowlist AVEC justification :", len(bare))
		for _, b := range bare {
			t.Errorf("  - %s", b)
		}
	}

	// Self-check (leçon V4d) : aucune entrée d'allowlist morte. Une entrée pour une
	// route walkée mais gardée est INUTILE ; une entrée pour une route absente du
	// walk est ROT. Les deux → à retirer.
	for key := range publicRoutesAllowlist {
		if !seen[key] {
			t.Errorf("entrée d'allowlist MORTE (route absente du routeur) : %q — la retirer", key)
			continue
		}
		if !hitAllow[key] {
			t.Errorf("entrée d'allowlist INUTILE (route %q est en réalité gardée) — la retirer", key)
		}
	}
}
