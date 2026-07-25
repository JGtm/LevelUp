//go:build cgo

// openapi_operation_metadata_test.go — garde-rail H2 (chantier V72-01).
//
// Prouve que CHAQUE opération du document OpenAPI PARTAGÉ (H1) porte des
// métadonnées stables — OperationID + Summary + au moins un Tag — et que, pour
// toute route COMMUNE avec le contrat manuel (api/openapi.yaml), l'operationID et
// les tags sont EN PARITÉ avec ce contrat. Ainsi la spec générée (H6) reproduit
// le groupage `tags:` et les operationId dont dépend le client TypeScript (H7).
//
// Deux familles échappent à la parité de chemin exact (donc requirement (a) seul) :
//   - routes Go-only jamais documentées dans le yaml (Prestige, multi-titre,
//     diagnostics auto-sync) ;
//   - routes dont le nom de PARAMÈTRE diffère du yaml (Go fait foi : {username}
//     vs {user_id}, {id} vs {notification_id}, {attempt_id} vs {provider}).
// Ces divergences sont documentées dans .ai/V7/PLAN_V72_HUMA_OPENAPI.md (H2).

package api_test

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

const apiV1DocPrefix = "/api/v1"

// yamlOpMeta = operationId + tags d'une opération du contrat manuel.
type yamlOpMeta struct {
	opID string
	tags []string
}

// loadManualYAMLOps charge api/openapi.yaml (kin-openapi, sans validation stricte
// — doc 3.1.0 à corps 3.0-style) et indexe operationId+tags par "METHOD path".
func loadManualYAMLOps(t *testing.T) map[string]yamlOpMeta {
	t.Helper()
	path := filepath.Join("..", "..", "api", "openapi.yaml")
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false
	doc, err := loader.LoadFromFile(path)
	if err != nil {
		t.Fatalf("chargement %s: %v", path, err)
	}
	if doc.Paths == nil {
		t.Fatalf("%s: aucun path", path)
	}
	ops := map[string]yamlOpMeta{}
	for p, item := range doc.Paths.Map() {
		for method, op := range item.Operations() {
			ops[method+" "+p] = yamlOpMeta{opID: op.OperationID, tags: append([]string(nil), op.Tags...)}
		}
	}
	return ops
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

// TestSharedOpenAPIDocOperationMetadata est le garde-rail central de H2.
func TestSharedOpenAPIDocOperationMetadata(t *testing.T) {
	// Mêmes flags que le garde-rail de fidélité H1 : surface Huma maximale.
	t.Setenv("LEVELUP_DEMO_MODE", "true")
	t.Setenv("MULTI_TITLE_API_ENABLED", "true")
	t.Setenv("PRESTIGE_ENABLED", "true")

	manual := loadManualYAMLOps(t)
	if len(manual) < 100 {
		t.Fatalf("contrat manuel : %d opérations chargées (attendu >= 100)", len(manual))
	}

	_, doc := buildTestRouterWithDoc(t)
	if doc == nil {
		t.Fatal("document OpenAPI partagé nil — NewRouter n'a pas retourné le doc")
	}

	var total, missingMeta, parityChecked, parityFail int
	for path, pi := range doc.Paths {
		relPath := path
		if relPath != apiV1DocPrefix {
			relPath = strings.TrimPrefix(relPath, apiV1DocPrefix)
		}
		for method, op := range pathItemOperations(pi) {
			total++

			// (a) Toute opération porte operationID + summary + >= 1 tag non vides.
			if strings.TrimSpace(op.OperationID) == "" {
				t.Errorf("op %s %s : OperationID vide", method, path)
				missingMeta++
			}
			if strings.TrimSpace(op.Summary) == "" {
				t.Errorf("op %s %s : Summary vide", method, path)
				missingMeta++
			}
			nonEmptyTags := 0
			for _, tag := range op.Tags {
				if strings.TrimSpace(tag) != "" {
					nonEmptyTags++
				}
			}
			if nonEmptyTags == 0 {
				t.Errorf("op %s %s : aucun Tag non vide", method, path)
				missingMeta++
			}

			// (b) Parité avec le contrat manuel pour les routes COMMUNES (même
			// chemin relatif exact + même méthode).
			ref, ok := manual[method+" "+relPath]
			if !ok {
				continue
			}
			parityChecked++
			if op.OperationID != ref.opID {
				t.Errorf("op %s %s : operationID=%q, yaml manuel=%q (parité rompue)",
					method, relPath, op.OperationID, ref.opID)
				parityFail++
			}
			if got, want := strings.Join(sortedCopy(op.Tags), ","), strings.Join(sortedCopy(ref.tags), ","); got != want {
				t.Errorf("op %s %s : tags=[%s], yaml manuel=[%s] (parité rompue)",
					method, relPath, got, want)
				parityFail++
			}
		}
	}

	if total < 100 {
		t.Fatalf("document : %d opérations (attendu >= 100) — partage du document cassé ?", total)
	}
	if parityChecked < 100 {
		t.Fatalf("parité vérifiée sur %d routes seulement (attendu >= 100) — mapping des chemins cassé ?", parityChecked)
	}
	t.Logf("H2 : %d ops (metadata OK=%d), parité vérifiée %d (échecs %d)",
		total, total-missingMeta, parityChecked, parityFail)
}
