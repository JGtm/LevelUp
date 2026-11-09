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
// routes racine /auth/xbox/{login,callback} (mode "xbox", ou profil desktop sous la capacité
// LoginViaBrowser : cf. TestBareRoutesRatchet_DesktopEntryRoutes) sont publics par
// conception, hors surface, couverts par LOT_S_ROUTE_GUARD_TABLE.
//
// MAINTENANCE. Route publique légitime = 1 ligne d'allowlist AVEC justification.
// Route qui doit être gardée = la monter sous une garde (elle disparaît de l'échec).
// Nouvelle route nue non justifiée = ROUGE.

package api_test

import (
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/openapigen"
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

// costClass range une route publique selon ce qu'un anonyme peut lui faire faire.
type costClass string

const (
	// costCheap : lecture locale bornée, aucun appel sortant, aucune écriture non bornée.
	costCheap costClass = "cheap"
	// costOutbound : peut déclencher un appel vers un service tiers (Microsoft, API Halo...).
	costOutbound costClass = "outbound"
	// costDisk : peut écrire sur le disque ou en base à la demande d'un anonyme.
	costDisk costClass = "disk"
)

// publicRoute : une entrée de l'allowlist. Une route outbound ou disk NOMME le test qui
// prouve qu'un anonyme n'en tire ni appel sortant ni écriture non bornée ; ce test doit
// exister (publicRouteProblems).
type publicRoute struct {
	class  costClass
	reason string
	proof  string // nom d'un Test... existant ; vide pour cheap
}

func cheap(reason string) publicRoute { return publicRoute{costCheap, reason, ""} }
func outbound(reason, proof string) publicRoute {
	return publicRoute{costOutbound, reason, proof}
}
func disk(reason, proof string) publicRoute { return publicRoute{costDisk, reason, proof} }

// publicRoutesAllowlist — routes légitimement accessibles sans garde d'auth, chacune
// avec sa CLASSE DE COÛT (costClass) : le ratchet de présence d'un garde ne voit pas les
// effets de bord d'une route publique (appel sortant, écriture disque).
// Établie 2026-07-07 (relevé comportemental croisé + marquage). Clé = « METHOD /route »
// exact (chi.Walk rend les paramètres {name}). DÉCROISSANTE : n'ajouter qu'avec raison.
var publicRoutesAllowlist = map[string]publicRoute{
	// Liveness / readiness (sondes infra, aucune donnée).
	"GET /health":  cheap("sonde liveness"),
	"GET /healthz": cheap("sonde liveness"),
	"GET /readyz":  cheap("sonde readiness"),
	// Auth-bootstrap (nécessaire AVANT login).
	"GET /api/v1/auth/device-flow/{attempt_id}": cheap("polling device-flow (onboarding, pas de session)"),
	"POST /api/v1/auth/device-flow/start":       outbound("démarrage device-flow (onboarding) — POST protégé CSRF", "TestStartDeviceFlow_PerIPCap"),
	"POST /api/v1/auth/login":                   cheap("login local — POST protégé CSRF"),
	"POST /api/v1/auth/logout":                  cheap("logout — POST protégé CSRF"),
	"POST /api/v1/auth/register":                disk("register (invite) — POST protégé CSRF", "TestRegister_InviteIsSingleUseUnderConcurrency"),
	"POST /api/v1/auth/password":                cheap("changement mot de passe (session) — POST protégé CSRF"),
	"POST /api/v1/session/context":              disk("établit le contexte de session (avant login) — POST protégé CSRF", "TestSave_RefusesSessionOverSizeLimit"),
	// Shell applicatif + annuaire (filtrés par ownership IN-SERVICE, pas au routeur).
	"GET /api/v1/bootstrap":                  cheap("shell React ; available_players filtré par session dans BootstrapService"),
	"GET /api/v1/players":                    cheap("liste joueurs filtrée par ownership in-service (S4, BuildPlayersList)"),
	"GET /api/v1/directory/gamertags/search": outbound("annuaire gamertags (503 si shared DB absente) — référentiel", "TestGamertagSearch_LiveIgnoredWithoutAuthenticatedSession"),
	// Contenu public (markdown / version).
	"GET /api/v1/changelog":          cheap("notes de version (markdown public)"),
	"GET /api/v1/help/release-notes": cheap("contenu public (README)"),
	"GET /api/v1/media/feed-version": cheap("numéro de version de flux (polling public)"),
	// Référentiels d'images (assets — aucune identité).
	"GET /api/v1/assets/battlepass/{subdir}/*":                 outbound("référentiel images battlepass", "TestAssetRoutes_UnverifiableKindNeedsTrust"),
	"GET /api/v1/assets/challenge-badge/{title_id}/{badge_id}": outbound("référentiel images badges", "TestAssetRoutes_UnverifiableKindNeedsTrust"),
	"GET /api/v1/assets/maps/{title_id}/{map_id}/image":        outbound("référentiel images maps", "TestAssetRoutes_MapImage_NewVariantCallsOutOnlyForTrustedCallers"),
	"GET /api/v1/assets/medals/{title_id}/{medal_id}/image":    outbound("référentiel images médailles", "TestAssetRoutes_UnknownIDDoesNotCallOut"),
	"GET /api/v1/assets/spartan/{image_type}/{title_id}/*":     outbound("référentiel images spartan", "TestAssetRoutes_UnverifiableKindNeedsTrust"),
	"GET /api/v1/assets/{title_id}/maps":                       cheap("référentiel catalogue maps"),
	"GET /api/v1/assets/{title_id}/medals":                     cheap("référentiel catalogue médailles"),
	"GET /api/v1/assets/{title_id}/weapons":                    cheap("référentiel catalogue armes"),
	// Référentiels de titre (labels / capacités, montés sans condition). NOTE :
	// les catalog/{playlists,pairs,maps} sont dep-gated (câblés seulement si le
	// catalog seasons est branché) → absents du routeur de test démo, donc non
	// couverts ici ; référentiels GET public par conception (LOT_S table §1).
	"GET /api/v1/titles/{slug}/capabilities":   cheap("référentiel capacités titre"),
	"GET /api/v1/titles/{slug}/feature-matrix": cheap("référentiel matrice de features"),
	"GET /api/v1/titles/{slug}/field-mappings": cheap("référentiel labels de champs"),
	// Assets statiques servis par le backend (file-server chi → TOUTES les méthodes).
	"GET /static/*":                   cheap("assets statiques"),
	"HEAD /static/*":                  cheap("assets statiques (HEAD)"),
	"OPTIONS /static/*":               cheap("assets statiques (preflight)"),
	"POST /static/*":                  cheap("assets statiques (file-server chi, toutes méthodes)"),
	"PUT /static/*":                   cheap("assets statiques (file-server chi, toutes méthodes)"),
	"PATCH /static/*":                 cheap("assets statiques (file-server chi, toutes méthodes)"),
	"DELETE /static/*":                cheap("assets statiques (file-server chi, toutes méthodes)"),
	"CONNECT /static/*":               cheap("assets statiques (file-server chi, toutes méthodes)"),
	"TRACE /static/*":                 cheap("assets statiques (file-server chi, toutes méthodes)"),
	"QUERY /static/*":                 cheap("assets statiques (file-server chi, toutes méthodes) — méthode QUERY ajoutée par go-chi 5.3.1 (bump 2026-08-02)"),
	"GET /static/commendations/*":     cheap("assets statiques commendations"),
	"HEAD /static/commendations/*":    cheap("assets statiques commendations (HEAD)"),
	"OPTIONS /static/commendations/*": cheap("assets statiques commendations (preflight)"),
	"POST /static/commendations/*":    cheap("assets statiques commendations (file-server chi, toutes méthodes)"),
	"PUT /static/commendations/*":     cheap("assets statiques commendations (file-server chi, toutes méthodes)"),
	"PATCH /static/commendations/*":   cheap("assets statiques commendations (file-server chi, toutes méthodes)"),
	"DELETE /static/commendations/*":  cheap("assets statiques commendations (file-server chi, toutes méthodes)"),
	"CONNECT /static/commendations/*": cheap("assets statiques commendations (file-server chi, toutes méthodes)"),
	"TRACE /static/commendations/*":   cheap("assets statiques commendations (file-server chi, toutes méthodes)"),
	"QUERY /static/commendations/*":   cheap("assets statiques commendations (file-server chi, toutes méthodes) — méthode QUERY ajoutée par go-chi 5.3.1 (bump 2026-08-02)"),
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

// TestBareRoutesRatchet_DesktopShutdownRouteIsGuarded : la route d'arrêt propre n'existe que
// si un secret d'arrêt est configuré, et alors elle est gardée par LoopbackOnly (garde reconnue
// par le ratchet : aucune entrée d'allowlist, qui serait signalée INUTILE). Absente sans secret.
func TestBareRoutesRatchet_DesktopShutdownRouteIsGuarded(t *testing.T) {
	const key = "POST /api/v1/internal/shutdown"
	walk := func(t *testing.T, secret string) (guarded, found bool) {
		t.Helper()
		router, _, err := openapigen.BuildDemoRouter(t.Context(), openapigen.Options{
			Root:           t.TempDir(),
			GroupStorePath: filepath.Join(os.TempDir(), "levelup_contract_groups.json"),
			Prestige:       true,
			DesktopSecret:  secret,
		})
		if err != nil {
			t.Fatalf("BuildDemoRouter: %v", err)
		}
		r, ok := router.(chi.Router)
		if !ok {
			t.Fatalf("le routeur n'est pas un chi.Router (%T)", router)
		}
		err = chi.Walk(r, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
			if method+" "+route == key {
				found, guarded = true, routeIsGuarded(mws)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("chi.Walk: %v", err)
		}
		return guarded, found
	}

	if _, found := walk(t, ""); found {
		t.Errorf("%s montée sans secret d'arrêt : un serveur hébergé ne doit pas l'exposer", key)
	}
	guarded, found := walk(t, "secret-de-test") // pragma: allowlist secret
	if !found {
		t.Fatalf("%s absente avec un secret d'arrêt configuré", key)
	}
	if !guarded {
		t.Errorf("%s montée sans garde reconnue (LoopbackOnly attendue)", key)
	}
	if _, listed := publicRoutesAllowlist[key]; listed {
		t.Errorf("%s ne doit pas figurer dans l'allowlist : elle est gardée", key)
	}
}

// TestBareRoutesRatchet_DesktopEntryRoutes : sous l'entrée de la coquille desktop (ADR 0045 D-3),
//
//   - GET /desktop/enter n'existe que là, et y est gardée par LoopbackOnly (garde reconnue : aucune
//     entrée d'allowlist, qui serait signalée INUTILE) ;
//
//   - GET /api/v1/desktop/strings n'existe que là ;
//
//   - TOUTE route du routeur y est derrière le garde de la coquille (Host strict, cookie exigé),
//     monté sur la racine : /health reste ouvert par exception DU GARDE, pas par route nue.
//
//   - GET /auth/xbox/callback et GET /api/v1/auth/xbox/login (connexion par le navigateur, L21,
//     2026-11-02) n'existent que sous la capacité LoginViaBrowser, le retour derrière LoopbackOnly.
//
// Hors de cette capacité, aucune de ces routes n'est montée (le routeur du ratchet principal,
// en démo, ne les voit pas).
func TestBareRoutesRatchet_DesktopEntryRoutes(t *testing.T) {
	const enterKey, stringsKey = "GET /desktop/enter", "GET /api/v1/desktop/strings"
	const loginKey, callbackKey = "GET /api/v1/auth/xbox/login", "GET /auth/xbox/callback"
	walk := func(t *testing.T, desktop bool) (found map[string]bool, guarded map[string]bool, behindGate map[string]bool) {
		t.Helper()
		opts := openapigen.Options{
			Root:           t.TempDir(),
			GroupStorePath: filepath.Join(os.TempDir(), "levelup_contract_groups.json"),
			Prestige:       true,
		}
		if desktop {
			opts.DesktopSecret = "secret-du-lanceur" // pragma: allowlist secret
			opts.ShellEntryPort = 49555
			opts.ShellCookieKey = "cle-de-cookie-de-test-de-trente-deux-octets" // pragma: allowlist secret
		}
		router, _, err := openapigen.BuildDemoRouter(t.Context(), opts)
		if err != nil {
			t.Fatalf("BuildDemoRouter: %v", err)
		}
		r, ok := router.(chi.Router)
		if !ok {
			t.Fatalf("le routeur n'est pas un chi.Router (%T)", router)
		}
		found, guarded, behindGate = map[string]bool{}, map[string]bool{}, map[string]bool{}
		err = chi.Walk(r, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
			key := method + " " + route
			found[key], guarded[key] = true, routeIsGuarded(mws)
			behindGate[key] = false
			for _, mw := range mws {
				if strings.Contains(runtime.FuncForPC(reflect.ValueOf(mw).Pointer()).Name(), "DesktopGate") {
					behindGate[key] = true
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("chi.Walk: %v", err)
		}
		return found, guarded, behindGate
	}

	foundHosted, _, gateHosted := walk(t, false)
	if foundHosted[enterKey] || foundHosted[stringsKey] || foundHosted[loginKey] || foundHosted[callbackKey] {
		t.Errorf("%s, %s, %s ou %s montée hors du profil desktop", enterKey, stringsKey, loginKey, callbackKey)
	}
	for key, behind := range gateHosted {
		if behind {
			t.Errorf("%s est derrière le garde de la coquille hors du profil desktop", key)
		}
	}

	found, guarded, behindGate := walk(t, true)
	if !found[enterKey] || !found[stringsKey] {
		t.Fatalf("routes absentes sous l'entrée de la coquille : %s=%v %s=%v", enterKey, found[enterKey], stringsKey, found[stringsKey])
	}
	if !guarded[enterKey] {
		t.Errorf("%s montée sans garde reconnue (LoopbackOnly attendue)", enterKey)
	}
	for _, key := range []string{enterKey, stringsKey} {
		if _, listed := publicRoutesAllowlist[key]; listed {
			t.Errorf("%s ne doit pas figurer dans l'allowlist : elle n'existe que sous le garde de la coquille", key)
		}
	}
	for key, behind := range behindGate {
		if !behind {
			t.Errorf("%s n'est pas derrière le garde de la coquille (Host strict, cookie exigé)", key)
		}
	}
	// Connexion par le navigateur (L21, 2026-11-02) : le retour de Microsoft est la seule route racine
	// que le garde laisse passer sans cookie (AllowBrowserCallback, Host localhost ou 127.0.0.1 du
	// port lié) ; il est monté derrière LoopbackOnly et protégé par le state à usage unique de la
	// tentative. Le démarrage reste derrière le cookie. Ni l'un ni l'autre ne figure dans l'allowlist
	// des routes publiques : ils n'existent que sous le garde de la coquille.
	if !found[loginKey] || !found[callbackKey] {
		t.Fatalf("routes de connexion par le navigateur absentes : %s=%v %s=%v", loginKey, found[loginKey], callbackKey, found[callbackKey])
	}
	if !guarded[callbackKey] {
		t.Errorf("%s montée sans garde reconnue (LoopbackOnly attendue)", callbackKey)
	}
	for _, key := range []string{loginKey, callbackKey} {
		if _, listed := publicRoutesAllowlist[key]; listed {
			t.Errorf("%s ne doit pas figurer dans l'allowlist : elle n'existe que sous le garde de la coquille", key)
		}
	}
}
