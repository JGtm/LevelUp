//go:build cgo

// bare_routes_ratchet_test.go — Ratchet « routes nues » (lot V3, DC-8).
//
// POURQUOI. La revue de sécurité du lot S (route → garde) était un grep MANUEL.
// Elle a laissé passer GET /jobs/{job_id} monté sans garde (finding VF-3) : aucun
// garde-rail automatisé n'attrapait une route montée sur `r` nu. Ce test est ce
// garde-rail permanent. Il aurait attrapé /jobs.
//
// COMMENT (approche COMPORTEMENTALE, choisie car le marquage des middlewares est
// fragile — chi ne fournit qu'un slice de closures non comparables). On construit
// le VRAI routeur en mode ENFORCEMENT (DemoMode=false, AuthMode="password" — sinon
// les gardes du lot S no-opent et tout paraîtrait public). On `chi.Walk` le routeur
// assemblé ; pour CHAQUE route on compose UNIQUEMENT sa chaîne de middlewares
// autour d'un handler bidon (jamais le vrai handler → aucune dépendance nil, aucun
// accès DuckDB) et on envoie une requête ANONYME. Une route qui répond 401/403 est
// gardée (RequireAuth / RequireAdmin / ownership / LoopbackOnly / CSRF). Une route
// qui répond autre chose (2xx…) est PUBLIQUE : elle DOIT figurer dans l'allowlist
// datée ci-dessous, sinon le test échoue.
//
// PÉRIMÈTRE. Mode "password" (auth déployée en self-hosted). Les routes racine
// /auth/xbox/{login,callback} (mode "xbox" uniquement) et le catch-all SPA `/*`
// (handler NotFound, non walkable) ne sont pas exercés ici — publics par
// conception, hors surface /api/v1, couverts par le tableau LOT_S_ROUTE_GUARD_TABLE.
//
// MAINTENANCE. Ajouter une route PUBLIQUE légitime = ajouter 1 ligne à l'allowlist
// AVEC sa justification. Ajouter une route qui doit être gardée = la monter sous
// une garde (elle passe alors en 401/403, le test reste vert). Une nouvelle route
// nue non justifiée = ROUGE.

package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/groupstore"
	"levelup/go-api/internal/service"
)

// publicRoutesAllowlist — routes légitimement accessibles sans garde d'auth.
// Établie 2026-07-07 par relevé comportemental (voir en-tête). Toute entrée =
// « METHOD /route » exact (paramètres sous forme {name}, comme chi.Walk les rend).
// DÉCROISSANTE par principe : on n'ajoute une entrée qu'avec une raison écrite.
var publicRoutesAllowlist = map[string]string{
	// Liveness / readiness (sondes infra, aucune donnée).
	"GET /health":  "sonde liveness",
	"GET /healthz": "sonde liveness",
	"GET /readyz":  "sonde readiness",
	// Auth-bootstrap (nécessaire AVANT login).
	"GET /api/v1/auth/device-flow/{attempt_id}": "polling device-flow (onboarding, pas de session)",
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
	// Référentiels de titre (labels / capacités / catalogues — MULTI_TITLE_API_ENABLED).
	"GET /api/v1/titles/{slug}/capabilities":      "référentiel capacités titre",
	"GET /api/v1/titles/{slug}/feature-matrix":    "référentiel matrice de features",
	"GET /api/v1/titles/{slug}/field-mappings":    "référentiel labels de champs",
	"GET /api/v1/titles/{slug}/catalog/maps":      "référentiel catalogue maps titre",
	"GET /api/v1/titles/{slug}/catalog/pairs":     "référentiel catalogue pairs titre",
	"GET /api/v1/titles/{slug}/catalog/playlists": "référentiel catalogue playlists titre",
	// Assets statiques servis par le backend (SPA + commendations).
	"GET /static/*":                   "assets statiques",
	"HEAD /static/*":                  "assets statiques (HEAD)",
	"OPTIONS /static/*":               "assets statiques (preflight)",
	"GET /static/commendations/*":     "assets statiques commendations",
	"HEAD /static/commendations/*":    "assets statiques commendations (HEAD)",
	"OPTIONS /static/commendations/*": "assets statiques commendations (preflight)",
}

// buildEnforcedRouter construit le routeur RÉEL en mode enforcement pour le
// ratchet. RepoRoot pointe le dépôt (TOML de mappings réels → validation boot OK).
func buildEnforcedRouter(t *testing.T) http.Handler {
	t.Helper()
	repoRoot := findRepoRoot(t)
	tmpDir := t.TempDir()
	appSettingsPath := filepath.Join(tmpDir, "app_settings.json")
	if err := os.WriteFile(appSettingsPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write app_settings: %v", err)
	}
	sessionDir := filepath.Join(tmpDir, "sessions")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}

	cfg := &config.AppConfig{
		RepoRoot:             repoRoot,
		DBProfilesPath:       filepath.Join(tmpDir, "db_profiles.json"), // absent → liste vide, pas de DuckDB
		AppSettingsPath:      appSettingsPath,
		SessionDir:           sessionDir,
		DemoMode:             false,      // CRITIQUE : gardes lot S actives
		AuthMode:             "password", // enforcement password (auth déployée)
		AuthDir:              filepath.Join(tmpDir, "auth"),
		DemoFixturesDir:      tmpDir,
		APIHost:              "127.0.0.1",
		APIPort:              8000,
		SessionSecret:        "CHANGE_ME_IN_PRODUCTION", // pragma: allowlist secret
		CORSOrigins:          []string{},
		Lang:                 "fr",
		RateLimitRPM:         1000000, // ne pas rate-limiter le balayage
		MultiTitleAPIEnabled: true,    // expose les référentiels de titre (routes publiques à couvrir)
		PrestigeEnabled:      true,    // expose les routes prestige (gardées → doivent l'être)
	}
	bootRepo := &mockBootstrapRepo{}
	bootSvc := service.NewBootstrapService(cfg, bootRepo)
	gs := groupstore.NewGroupStore(filepath.Join(tmpDir, "groups.json"))
	router, _ := api.NewRouter(context.Background(), cfg, bootRepo, bootSvc, nil, nil, nil, nil, gs)
	return router
}

// findRepoRoot remonte depuis le CWD du test jusqu'à trouver config/titles.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "config", "titles", "halo_infinite", "mappings", "fields.toml")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("repo root introuvable (config/titles/halo_infinite/mappings/fields.toml absent)")
	return ""
}

// probeAnonymous compose la chaîne de middlewares de la route autour d'un handler
// bidon (200) et renvoie le code d'une requête anonyme. 401/403 = gardée.
func probeAnonymous(method, route string, mws []func(http.Handler) http.Handler) int {
	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	paramRe := regexp.MustCompile(`\{[^/]+\}`)
	concrete := paramRe.ReplaceAllString(route, "x")
	req := httptest.NewRequest(method, concrete, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

// TestBareRoutesRatchet_NoUnguardedRouteOutsideAllowlist : aucune route montée
// sans garde d'auth hors allowlist datée. Garde-rail permanent de VF-3.
func TestBareRoutesRatchet_NoUnguardedRouteOutsideAllowlist(t *testing.T) {
	router := buildEnforcedRouter(t)
	r, ok := router.(chi.Router)
	if !ok {
		t.Fatalf("le routeur n'est pas un chi.Router (%T)", router)
	}

	seen := make(map[string]bool)     // routes walkées (pour le self-check)
	hitAllow := make(map[string]bool) // entrées d'allowlist effectivement rencontrées
	var bare []string                 // routes publiques hors allowlist (violations)

	err := chi.Walk(r, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		key := method + " " + route
		seen[key] = true
		code := probeAnonymous(method, route, mws)
		if code == http.StatusUnauthorized || code == http.StatusForbidden {
			return nil // gardée
		}
		if _, allowed := publicRoutesAllowlist[key]; allowed {
			hitAllow[key] = true
			return nil // publique légitime
		}
		bare = append(bare, key+" (code "+http.StatusText(code)+")")
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}

	if len(bare) > 0 {
		sort.Strings(bare)
		t.Errorf("%d route(s) NUE(S) hors allowlist (montées sans garde d'auth) — "+
			"les garder sous RequireAuth/RequireAdmin/ownership OU les ajouter à "+
			"publicRoutesAllowlist AVEC justification :", len(bare))
		for _, b := range bare {
			t.Errorf("  - %s", b)
		}
	}

	// Self-check (leçon V4d) : aucune entrée d'allowlist morte. Une entrée pour une
	// route walkée mais gardée (401/403) est INUTILE → la retirer. Une entrée pour
	// une route absente du walk (route supprimée/renommée) est ROT → la retirer.
	for key := range publicRoutesAllowlist {
		if !seen[key] {
			t.Errorf("entrée d'allowlist MORTE (route absente du routeur) : %q — la retirer", key)
			continue
		}
		if !hitAllow[key] {
			t.Errorf("entrée d'allowlist INUTILE (route %q est en réalité gardée 401/403) — la retirer", key)
		}
	}
}
