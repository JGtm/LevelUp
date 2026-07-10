//go:build cgo

package api_test

// openapi_schema_drift_test.go — Lever B (drift-detection des schémas OpenAPI).
//
// Agrège les `components.schemas` AUTO-DÉRIVÉS par Huma de tous les outputs typés
// des handlers (ex `bootstrapOutput{Body *domain.BootstrapResponse}`) et les
// compare au bloc `components.schemas` du `api/openapi.yaml` MANUEL.
//
// But : détecter la dérive silencieuse struct Go ↔ contrat (la cause du
// BootstrapResponse sous-spécifié) et prouver que le contrat de types est
// largement auto-dérivable (pivot Lever B). Les paths NE sont PAS régénérés
// (déjà à parité 0 via TestContractRoutes*, et Huma n'a pas les ~20 routes
// non-JSON de media.go).
//
// Mode actuel : RAPPORT (t.Log) + sanity + pivot. Le gate strict (échec sur
// MISSING de réponse) viendra par ratchet une fois le drift résorbé, miroir de
// `undocumentedThreshold` pour les paths.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"gopkg.in/yaml.v3"

	"levelup/go-api/internal/api/humacore"
)

// collectHumaSchemas installe le hook OnAPICreated, construit le routeur de test
// (DemoMode → pas de DuckDB), et agrège les schémas de toutes les instances Huma.
func collectHumaSchemas(t *testing.T) map[string]*huma.Schema {
	t.Helper()
	var apis []huma.API
	humacore.OnAPICreated = func(a huma.API) { apis = append(apis, a) }
	defer func() { humacore.OnAPICreated = nil }()

	_ = buildTestRouter(t) // déclenche tous les Mount() → tous les NewAPI → collecte

	merged := map[string]*huma.Schema{}
	for _, a := range apis {
		oapi := a.OpenAPI()
		if oapi == nil || oapi.Components == nil || oapi.Components.Schemas == nil {
			continue
		}
		for name, sch := range oapi.Components.Schemas.Map() {
			if _, ok := merged[name]; !ok {
				merged[name] = sch
			}
		}
	}
	return merged
}

// loadManualSchemas lit le bloc components.schemas du openapi.yaml manuel.
func loadManualSchemas(t *testing.T) map[string]any {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	apiRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	data, err := os.ReadFile(filepath.Join(apiRoot, "api", "openapi.yaml"))
	if err != nil {
		t.Fatalf("lecture openapi.yaml : %v", err)
	}
	var doc struct {
		Components struct {
			Schemas map[string]any `yaml:"schemas"`
		} `yaml:"components"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse openapi.yaml : %v", err)
	}
	return doc.Components.Schemas
}

// TestOpenAPISchemaDrift_AggregatesAndReports prouve l'agrégation (pivot Lever B)
// et rapporte le drift schémas Huma ↔ manuel.
func TestOpenAPISchemaDrift_AggregatesAndReports(t *testing.T) {
	huSchemas := collectHumaSchemas(t)
	manual := loadManualSchemas(t)

	// Sanity : Huma auto-dérive un grand nombre de schémas (outputs typés).
	if len(huSchemas) < 50 {
		t.Fatalf("agrégation Huma anormalement faible : %d schémas (attendu >= 50) — hook/collecte cassés ?", len(huSchemas))
	}
	t.Logf("Huma : %d schémas auto-dérivés ; openapi.yaml manuel : %d schémas", len(huSchemas), len(manual))

	// Pivot : les outputs construits sur des structs domaine partagées DOIVENT
	// apparaître dans l'agrégation Huma (preuve d'auto-dérivation exacte) ET dans
	// le manuel (réconciliés).
	for _, pivot := range []string{"BootstrapResponse"} {
		if _, ok := huSchemas[pivot]; !ok {
			t.Errorf("schéma pivot %q absent de l'agrégation Huma (auto-dérivation cassée)", pivot)
		}
		if _, ok := manual[pivot]; !ok {
			t.Errorf("schéma pivot %q absent du openapi.yaml manuel", pivot)
		}
	}

	missing, divergent, extra := diffSchemas(huSchemas, manual)

	// GATE de complétude (ratchet, miroir de undocumentedThreshold pour les paths) :
	// tout schéma de réponse auto-dérivé par Huma DOIT être documenté dans
	// openapi.yaml (sinon generated.ts est incomplet → type front écrit à la main →
	// dérive). Baseline 2026-06-18 = 0 après merge des 328 schémas manquants.
	if len(missing) > 0 {
		t.Errorf("complétude contrat : %d schéma(s) de réponse Huma NON documenté(s) dans openapi.yaml.\n"+
			"Régénérer : `OPENAPI_EMIT_OUT=/tmp/m.yaml CGO_ENABLED=1 go test ./internal/api/ -run TestOpenAPISchemaDrift`\n"+
			"puis appendre /tmp/m.yaml sous components.schemas + `npm -w apps/web run generate-types` (Huma dérive le contrat serveur — plus de `make gen`).\nManquants : %v",
			len(missing), missing)
	}

	t.Logf("DRIFT — MISSING (schéma de réponse Huma NON documenté dans openapi.yaml) : %d\n  %v", len(missing), missing)
	t.Logf("DRIFT — DIVERGENT (même nom, structure normalisée différente) : %d\n  %v", len(divergent), divergent)
	t.Logf("DRIFT — EXTRA (manuel sans contrepartie Huma : media.go / inputs RawBody / legacy) : %d", len(extra))

	// Mode -emit (gated par env, hors CI) : écrit des schémas Huma auto-dérivés en
	// YAML indenté prêt à coller/remplacer sous components.schemas du openapi.yaml.
	//  - OPENAPI_EMIT_OUT          : émet les MISSING (à appendre).
	//  - OPENAPI_EMIT_DIVERGENT_OUT: émet les DIVERGENT (versions Huma fidèles aux
	//    structs Go, pour remplacer les schémas manuels périmés).
	//  - OPENAPI_EMIT_PREFIX       : filtre optionnel par préfixe de nom.
	if out := os.Getenv("OPENAPI_EMIT_OUT"); out != "" {
		emitSchemasYAML(t, huSchemas, missing, os.Getenv("OPENAPI_EMIT_PREFIX"), out)
	}
	if out := os.Getenv("OPENAPI_EMIT_DIVERGENT_OUT"); out != "" {
		emitSchemasYAML(t, huSchemas, divergent, os.Getenv("OPENAPI_EMIT_PREFIX"), out)
	}
}

// emitSchemasYAML écrit la liste `names` (filtrée par préfixe si fourni) en YAML
// indenté à 4 espaces (niveau components.schemas), trié, prêt à appendre/remplacer
// dans api/openapi.yaml. Chaque schéma vient de la sérialisation OpenAPI
// auto-dérivée par Huma du struct Go correspondant (source de vérité).
func emitSchemasYAML(t *testing.T, huSchemas map[string]*huma.Schema, names []string, prefix, outPath string) {
	t.Helper()
	var buf strings.Builder
	n := 0
	for _, name := range names { // déjà trié
		if prefix != "" && !strings.HasPrefix(name, prefix) {
			continue
		}
		sch := huSchemas[name]
		b, err := json.Marshal(sch)
		if err != nil {
			t.Fatalf("marshal schéma %q : %v", name, err)
		}
		var v any
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatalf("unmarshal schéma %q : %v", name, err)
		}
		y, err := yaml.Marshal(normalizeNullableTree(v))
		if err != nil {
			t.Fatalf("yaml schéma %q : %v", name, err)
		}
		buf.WriteString("    " + name + ":\n")
		for _, line := range strings.Split(strings.TrimRight(string(y), "\n"), "\n") {
			if line == "" {
				buf.WriteString("\n")
			} else {
				buf.WriteString("      " + line + "\n")
			}
		}
		n++
	}
	if err := os.WriteFile(outPath, []byte(buf.String()), 0o644); err != nil {
		t.Fatalf("écriture %q : %v", outPath, err)
	}
	t.Logf("EMIT : %d schémas (préfixe %q) → %s", n, prefix, outPath)
}

// diffSchemas classe les schémas en MISSING (Huma absent du manuel),
// DIVERGENT (présents des deux côtés mais structure normalisée ≠) et EXTRA
// (manuel sans contrepartie Huma). Comparaison normalisée (ignore le cosmétique).
func diffSchemas(huSchemas map[string]*huma.Schema, manual map[string]any) (missing, divergent, extra []string) {
	for name, sch := range huSchemas {
		man, ok := manual[name]
		if !ok {
			missing = append(missing, name)
			continue
		}
		if canonHuma(sch) != canonAny(stripCosmetic(normalizeNullableTree(man))) {
			divergent = append(divergent, name)
		}
	}
	for name := range manual {
		if _, ok := huSchemas[name]; !ok {
			extra = append(extra, name)
		}
	}
	sort.Strings(missing)
	sort.Strings(divergent)
	sort.Strings(extra)
	return
}

// canonHuma sérialise un *huma.Schema en JSON canonique (clés triées par
// json.Marshal), cosmétique retiré.
func canonHuma(s *huma.Schema) string {
	b, err := json.Marshal(s)
	if err != nil {
		return "ERR:" + err.Error()
	}
	var v any
	_ = json.Unmarshal(b, &v)
	return canonAny(stripCosmetic(normalizeNullableTree(v)))
}

// normalizeNullableTree convertit récursivement la nullable OpenAPI 3.1
// (`type: [X, "null"]`) en 3.0 (`type: X` + `nullable: true`). Sert (a) à émettre
// du 3.0 (oapi-codegen v2.6 ne gère pas le 3.1) et (b) à comparer Huma (émet 3.1)
// et le manuel (3.0) de façon rep-agnostique dans le diff.
func normalizeNullableTree(v any) any {
	switch t := v.(type) {
	case map[string]any:
		if arr, ok := t["type"].([]any); ok && len(arr) == 2 {
			var real any
			nulls := 0
			for _, e := range arr {
				if e == "null" {
					nulls++
				} else {
					real = e
				}
			}
			if nulls == 1 && real != nil {
				t["type"] = real
				t["nullable"] = true
			}
		}
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = normalizeNullableTree(val)
		}
		return out
	case []any:
		for i := range t {
			t[i] = normalizeNullableTree(t[i])
		}
		return t
	default:
		return v
	}
}

// canonAny : JSON déterministe (json.Marshal trie les clés des maps).
func canonAny(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// stripCosmetic retire récursivement description/example/title (bruit humain
// sans valeur de type).
func stripCosmetic(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			switch k {
			case "description", "example", "title":
				continue
			}
			out[k] = stripCosmetic(val)
		}
		return out
	case []any:
		for i := range t {
			t[i] = stripCosmetic(t[i])
		}
		return t
	default:
		return v
	}
}
