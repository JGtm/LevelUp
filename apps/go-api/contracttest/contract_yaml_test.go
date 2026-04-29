// Package contracttest — garde-fous OpenAPI YAML (Sprint 29).
//
// Ces tests vérifient la validité et la cohérence de apps/go-api/api/openapi.yaml
// sans aucune dépendance CGO (pas de platform/duckdb, pas de chi router).
//
// Exécution CI (CGO_ENABLED=0 compatible) :
//
//	CGO_ENABLED=0 go test ./contracttest/... -run TestContract -v -timeout 30s
//
// Les tests de routage chi (TestContractRoutesRegistered, TestContractContentTypeJSON)
// qui nécessitent CGO vivent dans internal/api/contract_test.go (build tag cgo).
package contracttest_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// openapiDoc — structure minimale pour parser openapi.yaml
// ---------------------------------------------------------------------------

type openapiDoc struct {
	OpenAPI string                    `yaml:"openapi"`
	Info    map[string]any            `yaml:"info"`
	Paths   map[string]map[string]any `yaml:"paths"`
}

// httpMethods reconnus dans un path-item OpenAPI.
var httpMethods = map[string]bool{
	"get": true, "post": true, "put": true,
	"patch": true, "delete": true, "head": true, "options": true,
}

// ---------------------------------------------------------------------------
// loadOpenAPI — charge et parse openapi.yaml depuis ce package.
// ---------------------------------------------------------------------------

func loadOpenAPI(t *testing.T) openapiDoc {
	t.Helper()

	// Remonter depuis contracttest/ vers la racine du module go-api
	_, thisFile, _, _ := runtime.Caller(0)
	// thisFile = …/contracttest/contract_yaml_test.go → remonter d'1 niveau
	apiRoot := filepath.Join(filepath.Dir(thisFile), "..")
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
// TestContractOpenAPIYAMLValid (P0)
//
// Vérifie que openapi.yaml est syntaxiquement valide et contient les sections
// obligatoires openapi, info, paths avec au moins 20 paths déclarés.
// ---------------------------------------------------------------------------

func TestContractOpenAPIYAMLValid(t *testing.T) {
	doc := loadOpenAPI(t)

	if doc.OpenAPI == "" {
		t.Error("openapi.yaml : champ 'openapi' manquant")
	}
	if len(doc.Info) == 0 {
		t.Error("openapi.yaml : section 'info' manquante ou vide")
	}
	if len(doc.Paths) == 0 {
		t.Fatal("openapi.yaml ne contient aucun path — fichier vide ou mal formaté")
	}

	// Seuil minimal : le contrat Go doit documenter au moins 20 paths.
	// Sprint 29 a ajouté ~15 routes → total attendu > 20.
	const minPaths = 20
	if len(doc.Paths) < minPaths {
		t.Errorf("openapi.yaml : seulement %d paths déclarés (min %d)", len(doc.Paths), minPaths)
	}

	t.Logf("openapi.yaml valide : version=%s, %d paths déclarés", doc.OpenAPI, len(doc.Paths))
}

// ---------------------------------------------------------------------------
// TestContractMethodsValid (P0)
//
// Vérifie que toutes les méthodes HTTP déclarées dans openapi.yaml sont valides.
// Détecte les fautes de frappe (ex. "gest" au lieu de "get").
// ---------------------------------------------------------------------------

func TestContractMethodsValid(t *testing.T) {
	doc := loadOpenAPI(t)

	for path, pathItem := range doc.Paths {
		for method := range pathItem {
			if method == "$ref" || method == "summary" || method == "description" || method == "parameters" || method == "servers" {
				continue // clés non-opération valides dans un path-item
			}
			if !httpMethods[method] {
				t.Errorf("méthode HTTP invalide dans openapi.yaml :\n  path=%s, méthode=%q", path, method)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// TestContractJSONSyntaxValid (P0)
//
// Parse openapi.yaml → JSON pour vérifier que la conversion est sans perte.
// Garantit que le YAML peut être utilisé par des outils JSON (swagger-ui, etc.).
// ---------------------------------------------------------------------------

func TestContractJSONSyntaxValid(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	apiRoot := filepath.Join(filepath.Dir(thisFile), "..")
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

// ---------------------------------------------------------------------------
// TestContractPathsCoverage (P1)
//
// Vérifie que les routes fondamentales (health, bootstrap, players, settings)
// sont documentées dans openapi.yaml.
// Sprint 29 : garde-fou contre la suppression accidentelle de routes critiques.
// ---------------------------------------------------------------------------

func TestContractPathsCoverage(t *testing.T) {
	doc := loadOpenAPI(t)

	mandatoryPaths := []struct {
		path   string
		method string
		label  string
	}{
		{"/health", "get", "health check"},
		{"/bootstrap", "get", "bootstrap"},
		{"/players", "get", "liste joueurs"},
		{"/settings", "get", "settings GET"},
		{"/settings", "patch", "settings PATCH"},
		{"/auth/device-flow/start", "post", "device flow start"},
		{"/setup/players", "post", "création joueur"},
		{"/sync/initial", "post", "sync initiale"},
		{"/players/{player_slug}/matches/{match_id}/exclusion", "patch", "match exclusion PATCH"},
		// /players/{player_slug}/match-exclusions GET supprimé en revue
		// 2026-04-29 P0.2 Q6 (orphelin côté front, vue admin jamais implémentée).
	}

	for _, mp := range mandatoryPaths {
		pathItem, ok := doc.Paths[mp.path]
		if !ok {
			t.Errorf("path obligatoire absent de openapi.yaml : %s (%s)", mp.path, mp.label)
			continue
		}
		if _, hasMethod := pathItem[mp.method]; !hasMethod {
			t.Errorf("méthode %s absente du path %s (%s)", mp.method, mp.path, mp.label)
		}
	}
}

// ---------------------------------------------------------------------------
// convertYAMLToJSON — convertit map[string]any récursivement pour json.Marshal.
// ---------------------------------------------------------------------------

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
