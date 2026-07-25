//go:build cgo

// openapi_golden_test.go — GOLDEN du contrat généré (chantier V72-01 / H6).
//
// api/openapi.yaml n'est plus écrit à la main : il est produit par
// `make openapi-gen` (document OpenAPI PARTAGÉ de Huma + fragment manuel
// versionné). Ce test régénère le contrat EN MÉMOIRE — même fonction que la
// commande, aucun chemin de code parallèle — et le compare byte-à-byte au
// fichier commité. Toute évolution d'un handler ou d'un DTO qui change le
// contrat sans régénération échoue ici.
//
// Il remplace `openapi_schema_drift_test.go` (gate MISSING=0 des schémas Huma
// non documentés) : sous génération, un schéma dérivé ne PEUT plus manquer.
//
// Second garde-rail : les chemins que Huma ne connaît pas (routes chi-brut du
// fragment) doivent SURVIVRE à la régénération — une fusion qui les perdrait
// casserait la parité chi <-> yaml vérifiée par contract_test.

package api_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"levelup/go-api/internal/api/openapigen"
)

// apiDir retourne le répertoire api/ du module go-api.
func apiDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "api")
}

// generateContract régénère le contrat en mémoire (harnais partagé avec
// cmd/openapi-gen).
func generateContract(t *testing.T) []byte {
	t.Helper()
	got, err := openapigen.Generate(t.Context(), t.TempDir(),
		filepath.Join(apiDir(t), "openapi_manual_fragment.yaml"))
	if err != nil {
		t.Fatalf("génération du contrat : %v", err)
	}
	return got
}

// TestOpenAPIYAMLIsUpToDate — le golden.
func TestOpenAPIYAMLIsUpToDate(t *testing.T) {
	got := generateContract(t)
	path := filepath.Join(apiDir(t), "openapi.yaml")
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("lecture %s : %v", path, err)
	}
	if string(got) == string(want) {
		t.Logf("contrat à jour (%d octets)", len(want))
		return
	}
	t.Errorf("api/openapi.yaml N'EST PAS à jour (%d octets générés, %d commités) — relancer `make openapi-gen`.\n"+
		"Première divergence : %s", len(got), len(want), firstDiff(string(want), string(got)))
}

// firstDiff localise la première ligne divergente (diagnostic lisible).
func firstDiff(want, got string) string {
	wl, gl := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < len(wl) && i < len(gl); i++ {
		if wl[i] != gl[i] {
			return "ligne " + strconv.Itoa(i+1) + "\n  commité : " + wl[i] + "\n  généré  : " + gl[i]
		}
	}
	return "longueurs différentes (" + strconv.Itoa(len(wl)) + " vs " + strconv.Itoa(len(gl)) + " lignes)"
}

// TestManualFragmentPathsSurviveGeneration : les chemins du fragment (routes
// chi-brut, hors document Huma) figurent bien dans le contrat régénéré.
func TestManualFragmentPathsSurviveGeneration(t *testing.T) {
	fragmentPath := filepath.Join(apiDir(t), "openapi_manual_fragment.yaml")
	raw, err := os.ReadFile(fragmentPath)
	if err != nil {
		t.Fatalf("lecture %s : %v", fragmentPath, err)
	}
	var fragment struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(raw, &fragment); err != nil {
		t.Fatalf("parse fragment : %v", err)
	}
	var generated struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(generateContract(t), &generated); err != nil {
		t.Fatalf("parse contrat généré : %v", err)
	}

	// Chemins chi-brut = ceux que le fragment déclare AVEC leurs opérations
	// complètes (operationId présent) : Huma ne les connaît pas.
	var chiBrut, missing []string
	for path, item := range fragment.Paths {
		if !declaresFullOperation(item) {
			continue // simple enrichissement d'une route dérivée
		}
		chiBrut = append(chiBrut, path)
		if _, ok := generated.Paths[path]; !ok {
			missing = append(missing, path)
		}
	}
	if len(chiBrut) < 8 {
		t.Fatalf("fragment : %d chemin(s) chi-brut détecté(s) (attendu >= 8) — détection cassée ?", len(chiBrut))
	}
	if len(missing) > 0 {
		t.Errorf("%d chemin(s) du fragment ABSENT(s) du contrat régénéré (fusion cassée) : %v", len(missing), missing)
	}
	t.Logf("%d chemins chi-brut du fragment présents dans le contrat régénéré", len(chiBrut))
}

// declaresFullOperation : le path-item déclare au moins une opération COMPLÈTE
// (operationId), signe d'une route absente du document Huma.
func declaresFullOperation(item map[string]any) bool {
	for _, opAny := range item {
		op, ok := opAny.(map[string]any)
		if !ok {
			continue
		}
		if _, has := op["operationId"]; has {
			return true
		}
	}
	return false
}
