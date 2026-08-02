// Package api_test — contract_test.go (Sprint 29)
//
// Vérifie la cohérence entre le routeur chi et l'OpenAPI déclaré dans api/openapi.yaml.
//
// Deux garde-fous :
//  1. Chaque path+method déclaré dans openapi.yaml a un handler chi correspondant.
//  2. Chaque handler chi répond avec Content-Type: application/json (pas text/plain).
//
// Nécessite CGO (dépendance transitives vers platform/duckdb) :
//
//	CGO_ENABLED=1 go test ./internal/api/ -run TestContract -v -timeout 30s
//
// Les tests YAML-only (sans CGO) vivent dans ../../contracttest/.

//go:build cgo

package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Helpers — chargement de l'OpenAPI
// ---------------------------------------------------------------------------

type openapiDoc struct {
	Paths map[string]map[string]any `yaml:"paths"`
}

// httpMethods reconnus dans un path-item OpenAPI.
var httpMethods = map[string]bool{
	"get":     true,
	"post":    true,
	"put":     true,
	"patch":   true,
	"delete":  true,
	"head":    true,
	"options": true,
}

func loadOpenAPI(t *testing.T) openapiDoc {
	t.Helper()
	// Remonter jusqu'à la racine du module go-api
	_, thisFile, _, _ := runtime.Caller(0)
	// thisFile = …/internal/api/contract_test.go → remonter de 2 niveaux
	apiRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	yamlPath := filepath.Join(apiRoot, "api", "openapi.yaml")

	data, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("impossible de lire openapi.yaml : %v\n(chemin : %s)", err, yamlPath)
	}

	var doc openapiDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("impossible de parser openapi.yaml : %v", err)
	}
	return doc
}

// ---------------------------------------------------------------------------
// Helpers — extraction des routes chi
// ---------------------------------------------------------------------------

// chiRoutes parcourt toutes les routes enregistrées dans un routeur chi
// et retourne un set "METHOD /path" normalisé (paramètres → {*}).
func chiRoutes(h http.Handler) map[string]bool {
	r, ok := h.(chi.Router)
	if !ok {
		return map[string]bool{}
	}
	result := make(map[string]bool)
	paramRe := regexp.MustCompile(`\{[^/]+\}`)

	walkFn := func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		normalizedRoute := paramRe.ReplaceAllString(route, "{*}")
		// P8.8 (revue 2026-04-29) : chi catchall trailing `*` aligné sur la
		// notation OpenAPI `{*}` (OpenAPI 3.x ne supporte pas les catchalls
		// nativement, on aligne donc côté test).
		if strings.HasSuffix(normalizedRoute, "/*") {
			normalizedRoute = strings.TrimSuffix(normalizedRoute, "/*") + "/{*}"
		}
		result[strings.ToUpper(method)+" "+normalizedRoute] = true
		return nil
	}
	_ = chi.Walk(r, walkFn)
	return result
}

// normalizePath convertit un path OpenAPI en chemin normalisé pour comparaison
// (paramètres {player_slug}, {match_id} → {*}).
func normalizePath(p string) string {
	re := regexp.MustCompile(`\{[^/]+\}`)
	return re.ReplaceAllString(p, "{*}")
}

// ---------------------------------------------------------------------------
// TestContractRoutesRegistered (R1 / P0-3)
// ---------------------------------------------------------------------------
// Vérifie que chaque path+method documenté dans openapi.yaml a un handler chi.
// Routes marquées "TODO Sprint 32" (501 Not Implemented) sont exemptées —
// elles sont enregistrées dans le YAML mais pas encore dans chi.
func TestContractRoutesRegistered(t *testing.T) {
	doc := loadOpenAPI(t)

	// Construire le routeur en mode démo (sans DuckDB)
	t.Setenv("LEVELUP_DEMO_MODE", "true")
	// P8.8 (revue 2026-04-29) : flag ON pour exposer les routes conditionnelles
	// (prestige challenges/arcs/...) que l'OpenAPI documente désormais. Les
	// routes multi-titres n'ont plus de flag depuis le 2026-08-02 (item 3.3).
	t.Setenv("PRESTIGE_ENABLED", "true")
	router := buildTestRouter(t)
	chiRouteSet := chiRoutes(router)

	// Endpoints intentionnellement absents ou divergents dans chi :
	// Sprint 49 : toutes les exemptions ont été résolues.
	// - citations/commendations/media/synthesis : OpenAPI aligné sur POST (chi)
	// - match-history/export : OpenAPI aligné sur GET (chi)
	// - gamertag search : route inconditionnelle avec 503 si DB absente
	// TODO : POST /players/{player_slug}/pages/last-match/resolve (pas encore implémenté)
	notYetImplemented := map[string]bool{
		"POST /api/v1/players/{*}/pages/last-match/resolve": true,
	}

	for path, pathItem := range doc.Paths {
		for method := range pathItem {
			if !httpMethods[method] {
				continue
			}
			normalizedPath := normalizePath(path)
			// L'OpenAPI utilise le préfixe /api/v1 dans ses paths (servers: [{url: /api/v1}])
			// mais les paths dans le fichier sont relatifs au server.
			// Dans chi, les routes sont enregistrées avec le préfixe complet.
			// On teste les deux formes pour être robuste.
			chiKey := strings.ToUpper(method) + " " + normalizedPath
			chiKeyPrefixed := strings.ToUpper(method) + " /api/v1" + normalizedPath

			if notYetImplemented[chiKeyPrefixed] {
				t.Logf("SKIP (not-yet-impl) : %s %s", strings.ToUpper(method), path)
				continue
			}

			found := chiRouteSet[chiKey] || chiRouteSet[chiKeyPrefixed]
			if !found {
				// Tolérance : routes avec préfixe /api/v1 dans le YAML peuvent être
				// enregistrées sans préfixe dans chi (ex. /health).
				t.Errorf(
					"route OpenAPI non enregistrée dans chi :\n  %s %s\n  (cherché : %q ou %q)\n  routes chi disponibles : %v",
					strings.ToUpper(method), path, chiKey, chiKeyPrefixed, sortedKeys(chiRouteSet),
				)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// TestContractContentTypeJSON (R1 / P0-3)
// ---------------------------------------------------------------------------
// TestContractRoutesDocumented (R2 / P3.3)
// ---------------------------------------------------------------------------
// Verification miroir de TestContractRoutesRegistered : chaque route chi doit
// etre documentee dans openapi.yaml. Plafonne le delta a une valeur sentinelle
// qui doit decroitre selon les jalons du PLAN_ACTION.md (revue 2026-04-29) :
//
//   - Au depart (revue) : 57 routes chi non documentees (engagement, squad-v2,
//     multi-title, asset-drawer, etc.). Plafond actuel = 65 (marge de 8 pour
//     les flags ON qui exposent des routes additionnelles).
//   - Apres P4 big-bang canonical : <= 30 (les services migres documentent
//     leurs DTOs canoniques).
//   - Apres P6 (capabilities + flags ON) : <= 10.
//   - En P8.8 (cloture OpenAPI) : 0.
//
// Le test ne FORCE pas la documentation immediate — il empeche que le delta
// grandisse en silence.
func TestContractRoutesDocumented(t *testing.T) {
	doc := loadOpenAPI(t)
	t.Setenv("LEVELUP_DEMO_MODE", "true")
	t.Setenv("PRESTIGE_ENABLED", "true")
	router := buildTestRouter(t)
	chiRouteSet := chiRoutes(router)

	// Construire le set des paths documentes (ignorer la methode pour ce check
	// — la couverture method-level est faite par TestContractRoutesRegistered
	// dans le sens inverse).
	openAPIPaths := make(map[string]bool, len(doc.Paths))
	for path := range doc.Paths {
		openAPIPaths[normalizePath(path)] = true
	}

	// Routes chi a ignorer : helpers Go natifs (CONNECT, TRACE, OPTIONS pour
	// /static/*) qui ne sont pas du contrat metier.
	ignoreMethods := map[string]bool{
		"CONNECT": true,
		"TRACE":   true,
		"OPTIONS": true,
		"HEAD":    true,
	}

	var undocumented []string
	for chiKey := range chiRouteSet {
		// chiKey est de la forme "METHOD /api/v1/path/{*}" ou "METHOD /path/{*}".
		parts := strings.SplitN(chiKey, " ", 2)
		if len(parts) != 2 {
			continue
		}
		method := parts[0]
		path := parts[1]

		if ignoreMethods[method] {
			continue
		}
		// Ignorer les routes static/assets non-API.
		if strings.HasPrefix(path, "/static") {
			continue
		}
		// P8.8 (revue 2026-04-29) : /debug/vars est l'expvar handler stdlib
		// (observability/admin-only). Pas un endpoint REST métier — ignoré
		// du contrat OpenAPI. Le routeur chi enregistre 5 méthodes (DELETE,
		// GET, PATCH, POST, PUT) sur le même path comme artefact du
		// HandleFunc générique d'expvar.
		if strings.HasPrefix(path, "/debug/vars") {
			continue
		}

		// Tester avec et sans le prefixe /api/v1 (les paths OpenAPI sont
		// relatifs au server, qui est /api/v1).
		pathStripped := strings.TrimPrefix(path, "/api/v1")
		if openAPIPaths[path] || openAPIPaths[pathStripped] {
			continue
		}
		undocumented = append(undocumented, chiKey)
	}

	// Plafond actuel — a decrocher progressivement selon les jalons P4/P6/P8.8.
	//
	// P7 (revue 2026-04-29) : abaissement progressif du plafond de 65 → 60.
	// Le jalon plan post-P6 ≤ 10 reste l'objectif final mais nécessite un
	// effort de documentation OpenAPI massif (40+ routes squad-v2,
	// engagement, multi-title, asset-drawer, prestige). On serre d'abord
	// le ratchet pour interdire toute régression silencieuse, puis on
	// fera le rattrapage en lots dédiés en P8.
	//
	// Suivi des jalons :
	//   - Baseline revue : 57 routes chi non documentees
	//   - Apres P4 big-bang canonical : <= 30 (services migres documentent leurs DTOs)
	//   - Apres P6 (capabilities + flags ON) : <= 10
	//   - En P8.8 (cloture OpenAPI) : 0
	const undocumentedThreshold = 0

	if len(undocumented) > undocumentedThreshold {
		// Tri des routes pour un diagnostic stable.
		sort.Strings(undocumented)
		t.Errorf(
			"P3.3 — %d routes chi non documentees dans openapi.yaml (plafond %d).\n"+
				"Routes manquantes :\n  %s\n\n"+
				"Jalons (PLAN_ACTION.md) :\n"+
				"  - apres P4 (canonical) : <= 30\n"+
				"  - apres P6 (capabilities) : <= 10\n"+
				"  - en P8.8 (cloture OpenAPI) : 0\n",
			len(undocumented), undocumentedThreshold,
			strings.Join(undocumented, "\n  "),
		)
	}
	t.Logf("P3.3 — %d routes chi non documentees (plafond %d)", len(undocumented), undocumentedThreshold)
}

// ---------------------------------------------------------------------------
// Vérifie que chaque handler déclaré retourne Content-Type: application/json
// pour les requêtes en mode démo, pas text/plain.
//
// Note : seules les routes qui retournent 200 (pas les 404/501) sont testées.
// Les routes "not yet implemented" (501) sont ignorées.
func TestContractContentTypeJSON(t *testing.T) {
	t.Setenv("LEVELUP_DEMO_MODE", "true")
	router := buildTestRouter(t)

	// Routes à tester (GET uniquement — les POST nécessitent un body valide).
	// Les POST sont couverts par les tests d'intégration.
	routesToTest := []struct {
		method string
		path   string
	}{
		{"GET", "/health"},
		{"GET", "/api/v1/bootstrap"},
		{"GET", "/api/v1/players"},
		{"GET", "/api/v1/settings"},
		{"GET", "/api/v1/jobs/nonexistent"},
	}

	for _, rt := range routesToTest {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), rt.method, rt.path, http.NoBody)
			req.Header.Set("Accept", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			ct := w.Header().Get("Content-Type")
			if !strings.Contains(ct, "application/json") {
				t.Errorf(
					"%s %s : Content-Type = %q, attendu application/json (status=%d)",
					rt.method, rt.path, ct, w.Code,
				)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestContractOpenAPIYAMLValid (P0-3)
// ---------------------------------------------------------------------------
// Vérifie que openapi.yaml est un YAML syntaxiquement valide et contient
// au minimum les clés openapi, info, paths.
func TestContractOpenAPIYAMLValid(t *testing.T) {
	doc := loadOpenAPI(t)

	if len(doc.Paths) == 0 {
		t.Fatal("openapi.yaml ne contient aucun path — fichier vide ou mal formaté")
	}
	t.Logf("openapi.yaml valide : %d paths déclarés", len(doc.Paths))
}

// ---------------------------------------------------------------------------
// TestContractJSONSyntaxValid (P0-3)
// ---------------------------------------------------------------------------
// Parse openapi.yaml → JSON pour vérifier que la conversion est sans perte.
// Utilisé comme smoke test du schéma.
func TestContractJSONSyntaxValid(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	apiRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	yamlPath := filepath.Join(apiRoot, "api", "openapi.yaml")

	data, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("lecture openapi.yaml : %v", err)
	}

	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("parse YAML : %v", err)
	}

	// Conversion YAML → JSON pour vérifier la sérialisabilité
	jsonBytes, err := json.Marshal(convertYAMLToJSON(raw))
	if err != nil {
		t.Fatalf("conversion YAML→JSON : %v", err)
	}
	if len(jsonBytes) == 0 {
		t.Fatal("conversion YAML→JSON a produit un JSON vide")
	}
	t.Logf("openapi.yaml → JSON OK (%d bytes)", len(jsonBytes))
}

// convertYAMLToJSON convertit les map[interface{}]interface{} (yaml.v3) en map[string]interface{}
// pour la sérialisation JSON.
func convertYAMLToJSON(v any) any {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, v2 := range val {
			result[k] = convertYAMLToJSON(v2)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, v2 := range val {
			result[i] = convertYAMLToJSON(v2)
		}
		return result
	default:
		return v
	}
}

// ---------------------------------------------------------------------------
// Helpers — tri
// ---------------------------------------------------------------------------

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Tri simple (pas de sort import pour minimiser les dépendances)
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}
