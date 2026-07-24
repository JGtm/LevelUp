package title

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"testing"
)

// capabilities_ts_mirror_test.go — garde-rail V72-12 : parité entre les
// constantes `Cap*` (type Capability) déclarées dans registry.go et leur miroir
// manuel TS `TITLE_CAPABILITIES` (apps/web/src/lib/capabilities/capabilities.ts).
//
// Le TS n'est PAS généré depuis le Go (pas de codegen ici) : ce test compare les
// DEUX sources de vérité et échoue si elles divergent, dans les deux sens :
//   - une constante Go absente du TS (drift silencieux → capability jamais
//     gatée côté UI) ;
//   - un littéral TS qui ne correspond à AUCUNE constante Go (typo, ou
//     capability supprimée côté Go sans nettoyage du TS).
//
// Exception : `backendOnlyAllowlist` documente les capabilities Go qui n'ont
// délibérément AUCUNE surface de gating UI (prédicats internes au backend).
// Chaque entrée est datée et justifiée — cf. règle CLAUDE.md n°11 (kill-switch
// commenté daté).
func TestCapabilitiesGoTSMirror(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")

	registryPath := filepath.Join(repoRoot, "apps", "go-api", "internal", "domain", "title", "registry.go")
	tsPath := filepath.Join(repoRoot, "apps", "web", "src", "lib", "capabilities", "capabilities.ts")

	goCaps := extractGoCapabilities(t, registryPath)
	tsCaps := extractTSCapabilities(t, tsPath)

	if len(goCaps) == 0 {
		t.Fatalf("aucune constante Cap* de type Capability extraite de %s — le parseur go/ast a probablement cassé (registry.go restructuré ?)", registryPath)
	}
	if len(tsCaps) == 0 {
		t.Fatalf("aucun littéral extrait du tableau TITLE_CAPABILITIES dans %s — la regex bornée n'a pas matché (fichier renommé/restructuré ?)", tsPath)
	}

	// Sens 1 (Go -> TS) : toute constante Go doit être mirroirée côté TS, sauf
	// exception documentée dans backendOnlyAllowlist.
	for value, constName := range goCaps {
		if tsCaps[value] {
			continue
		}
		if _, allowed := backendOnlyAllowlist[value]; allowed {
			continue
		}
		t.Errorf(
			"constante Go %s (%q, registry.go) absente de TITLE_CAPABILITIES — "+
				"ajouter %q côté TS (apps/web/src/lib/capabilities/capabilities.ts) ou "+
				"documenter une exception datée dans backendOnlyAllowlist",
			constName, value, value,
		)
	}

	// Sens 2 (TS -> Go) : tout littéral TS doit correspondre à une constante Go
	// réelle (sinon typo, ou capability Go supprimée sans nettoyage du TS).
	for value := range tsCaps {
		if _, ok := goCaps[value]; !ok {
			t.Errorf(
				"TITLE_CAPABILITIES (TS) contient %q qui ne correspond à AUCUNE constante Cap* "+
					"de registry.go — typo, ou capability Go renommée/supprimée sans mise à jour du TS",
				value,
			)
		}
	}

	// Hygiène de l'allowlist elle-même : chaque entrée doit correspondre à une
	// constante Go existante et rester réellement absente du TS (sinon
	// l'exception est périmée et doit être retirée).
	for value, reason := range backendOnlyAllowlist {
		constName, isGoCap := goCaps[value]
		if !isGoCap {
			t.Errorf("backendOnlyAllowlist contient %q (%s) mais aucune constante Cap* de registry.go ne porte cette valeur — allowlist périmée, la retirer", value, reason)
			continue
		}
		if tsCaps[value] {
			t.Errorf(
				"backendOnlyAllowlist contient %q (constante Go %s, %s) mais elle est désormais présente dans "+
					"TITLE_CAPABILITIES (TS) — retirer l'entrée de l'allowlist, elle n'est plus backend-only",
				value, constName, reason,
			)
		}
	}
}

// backendOnlyAllowlist recense les capabilities Go (valeur string) volontairement
// ABSENTES du miroir TS TITLE_CAPABILITIES : prédicats internes au backend, sans
// aucune surface de gating UI. Chaque entrée DOIT être datée + justifiée avant
// d'être ajoutée — vérifier par grep qu'aucun code front (apps/web/src) ne
// référence le littéral avant de l'allowlister ici.
var backendOnlyAllowlist = map[string]string{
	// CapWeaponKills (registry.go) : prédicat de complétude de snapshot backend
	// (pipeline film-decoder — un titre sans cette cap n'exige pas les
	// weapon-kills pour qu'un match soit "complet"). Aucun gating UI : le front
	// ne branche jamais de feature sur cette capability (grep 2026-07-24 : le
	// seul littéral "weapon_kills" côté apps/web désigne l'étape du pipeline de
	// convergence admin — features/admin/convergence/timeline.ts — un concept
	// homonyme distinct, pas la capability). Daté 2026-07-24.
	"weapon_kills": "2026-07-24, prédicat de complétude snapshot backend, pas de gating UI",
}

// extractGoCapabilities parse registry.go via go/ast et retourne l'ensemble des
// constantes de type `Capability` sous la forme {valeur string -> nom de la
// constante}. Aucune liste dupliquée à la main : la source de vérité reste le
// fichier Go lui-même.
func extractGoCapabilities(t *testing.T, path string) map[string]string {
	t.Helper()

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parser.ParseFile(%s): %v", path, err)
	}

	out := make(map[string]string)
	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			typeIdent, ok := valueSpec.Type.(*ast.Ident)
			if !ok || typeIdent.Name != "Capability" {
				continue
			}
			for i, name := range valueSpec.Names {
				if i >= len(valueSpec.Values) {
					continue
				}
				lit, ok := valueSpec.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				val, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatalf("strconv.Unquote(%s) pour la constante %s: %v", lit.Value, name.Name, err)
				}
				out[val] = name.Name
			}
		}
	}
	return out
}

// titleCapabilitiesArrayRe borne l'extraction au tableau TITLE_CAPABILITIES
// (déclaration `export const TITLE_CAPABILITIES = [ ... ] as const`) — ne
// capture RIEN d'autre dans le fichier (hooks, types, autres exports).
var titleCapabilitiesArrayRe = regexp.MustCompile(`(?s)TITLE_CAPABILITIES\s*=\s*\[(.*?)\]\s*as\s+const`)

// capabilityStringLiteralRe extrait les littéraux string simples ('xxx') à
// l'intérieur du bloc capturé — tolérant aux virgules finales, retours à la
// ligne et commentaires entre éléments.
var capabilityStringLiteralRe = regexp.MustCompile(`'([^'\n]*)'`)

// extractTSCapabilities lit capabilities.ts et retourne l'ensemble des
// littéraux déclarés dans le tableau TITLE_CAPABILITIES.
func extractTSCapabilities(t *testing.T, path string) map[string]bool {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%s): %v", path, err)
	}

	block := titleCapabilitiesArrayRe.FindStringSubmatch(string(data))
	if block == nil {
		t.Fatalf("tableau TITLE_CAPABILITIES introuvable dans %s (regex bornée sans match — fichier renommé/restructuré ?)", path)
	}

	matches := capabilityStringLiteralRe.FindAllStringSubmatch(block[1], -1)
	out := make(map[string]bool, len(matches))
	for _, m := range matches {
		out[m[1]] = true
	}
	return out
}
