package title

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// capabilities_parity_scan_test.go — EXTRACTEURS du garde-rail de parité des
// capabilities title-level (les tests eux-mêmes vivent dans
// capabilities_parity_test.go, qui porte la doc d'ensemble).
//
// Principe commun : toutes les listes surveillées sont LUES dans les sources
// (go/ast pour le Go, scan borné pour le TypeScript) — jamais recopiées ici. Un
// garde-rail qui duplique la liste qu'il surveille re-diverge avec elle.

// capabilityRepoRoot remonte à la racine du dépôt depuis ce fichier de test
// (apps/go-api/internal/domain/title → 5 niveaux).
func capabilityRepoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")
}

func capabilityRegistryPath(root string) string {
	return filepath.Join(root, "apps", "go-api", "internal", "domain", "title", "registry.go")
}

// extractKnownCapabilityKeys parse config_loader.go et retourne les NOMS des
// constantes utilisées comme clés du littéral composite `knownCapabilities`.
// (Les clés sont des identifiants Go : un renommage de constante est déjà attrapé
// par le compilateur ; ce qui échappe au compilateur, c'est l'OUBLI d'une entrée.)
func extractKnownCapabilityKeys(t *testing.T, path string) map[string]bool {
	t.Helper()

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parser.ParseFile(%s): %v", path, err)
	}

	out := make(map[string]bool)
	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok || len(valueSpec.Names) != 1 || valueSpec.Names[0].Name != "knownCapabilities" {
				continue
			}
			collectCompositeKeys(valueSpec.Values, out)
		}
	}
	return out
}

func collectCompositeKeys(values []ast.Expr, out map[string]bool) {
	for _, v := range values {
		lit, ok := v.(*ast.CompositeLit)
		if !ok {
			continue
		}
		for _, elt := range lit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			if ident, ok := kv.Key.(*ast.Ident); ok {
				out[ident.Name] = true
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Scan front : littéraux de gating de apps/web
// ---------------------------------------------------------------------------

// gatingCallees — appels dont les arguments littéraux sont des capabilities title-level.
var gatingCallees = []string{"useCapability(", "useCapabilityStrict(", "hasCapabilityIn("}

// capabilityPropRe couvre les formes déclaratives : prop JSX `capability="media"`
// et champ d'objet `capability: 'career'` (navL1Sections, FeatureGate, RouteCapabilityGate).
var capabilityPropRe = regexp.MustCompile(`\bcapability\s*[:=]\s*['"]([^'"\n]+)['"]`)

// capabilityShapeRe borne les littéraux retenus dans les arguments d'appel à la
// FORME d'une capability (évite d'imputer un littéral quelconque à un gate).
var capabilityShapeRe = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*$`)

var quotedLiteralRe = regexp.MustCompile(`['"]([^'"\n]*)['"]`)

// Les deux scans d'arborescence sont MÉMOÏSÉS : trois tests les consomment, et un
// walk complet de apps/web/src + apps/go-api coûte ~19 s sur disque Windows. Les
// tests du package ne sont pas parallèles ; sync.Once suffit et garde le résultat
// identique d'un appel à l'autre (déterminisme).
var (
	frontLiteralsOnce  sync.Once
	frontLiteralsCache map[string][]string
	goRefsOnce         sync.Once
	goRefsCache        map[string]string
)

// frontCapabilityLiterals scanne apps/web/src et retourne {capability -> occurrences}.
// Les fichiers de test sont EXCLUS : ils contiennent volontairement des capabilities
// factices (cf. spartan-customizer-strict.guard.test.ts, 'weapon_analysis').
func frontCapabilityLiterals(t *testing.T, webSrc string) map[string][]string {
	t.Helper()
	frontLiteralsOnce.Do(func() { frontLiteralsCache = scanFrontCapabilityLiterals(t, webSrc) })
	return frontLiteralsCache
}

func scanFrontCapabilityLiterals(t *testing.T, webSrc string) map[string][]string {
	t.Helper()

	out := make(map[string][]string)
	err := filepath.WalkDir(webSrc, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".ts") && !strings.HasSuffix(name, ".tsx") {
			return nil
		}
		if strings.HasSuffix(name, ".d.ts") || strings.Contains(name, ".test.") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, _ := filepath.Rel(filepath.Dir(webSrc), path)
		collectCapabilityLiterals(string(data), filepath.ToSlash(rel), out)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", webSrc, err)
	}
	return out
}

func collectCapabilityLiterals(code, rel string, out map[string][]string) {
	record := func(value string, at int) {
		out[value] = append(out[value], rel+":"+strconv.Itoa(1+strings.Count(code[:at], "\n")))
	}
	for _, callee := range gatingCallees {
		for _, span := range callArgSpans(code, callee) {
			args := code[span[0]:span[1]]
			for _, m := range quotedLiteralRe.FindAllStringSubmatchIndex(args, -1) {
				value := args[m[2]:m[3]]
				if capabilityShapeRe.MatchString(value) {
					record(value, span[0]+m[2])
				}
			}
		}
	}
	for _, m := range capabilityPropRe.FindAllStringSubmatchIndex(code, -1) {
		record(code[m[2]:m[3]], m[2])
	}
}

// callArgSpans retourne les intervalles [début, fin) des ARGUMENTS de chaque appel
// à `callee`, en suivant la profondeur de parenthèses.
//
// Pourquoi pas une regex (leçon reprise de spartan-customizer-strict.guard.test.ts) :
// un motif `\([^)]*'x'` s'arrête à la première parenthèse fermante et rate donc
// `hasCapabilityIn(useTitleCapabilities(), 'x')` — la forme la plus naturelle.
func callArgSpans(code, callee string) [][2]int {
	var spans [][2]int
	from := 0
	for {
		idx := strings.Index(code[from:], callee)
		if idx < 0 {
			return spans
		}
		at := from + idx
		from = at + len(callee)
		// Rejette les noms dont `callee` n'est qu'un suffixe (ex. myUseCapability().
		if at > 0 && isIdentByte(code[at-1]) {
			continue
		}
		depth, i := 1, from
		for i < len(code) && depth > 0 {
			switch code[i] {
			case '(':
				depth++
			case ')':
				depth--
			}
			i++
		}
		end := i
		if depth == 0 {
			end = i - 1
		}
		spans = append(spans, [2]int{from, end})
	}
}

func isIdentByte(b byte) bool {
	return b == '_' || b == '$' || b == '.' ||
		(b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// ---------------------------------------------------------------------------
// Scan Go : consommateurs des constantes Cap*
// ---------------------------------------------------------------------------

// goCapabilityReferences scanne les sources Go NON-test de apps/go-api (hors package
// domain/title, qui DÉCLARE les constantes) et retourne {nom de constante -> 1re
// occurrence "fichier:ligne"}.
//
// Les références qualifiées par `games.` sont IGNORÉES : le package games déclare
// des constantes HOMONYMES d'un autre système (games.CapWeaponAccuracy /
// games.CapEngagement sont des CapabilityKey DATA-level, cf. diag §4.4). Les
// compter reviendrait à créditer une capability title-level d'un consommateur
// qui ne la lit pas.
func goCapabilityReferences(t *testing.T, goAPIRoot string) map[string]string {
	t.Helper()
	goRefsOnce.Do(func() { goRefsCache = scanGoCapabilityReferences(t, goAPIRoot) })
	return goRefsCache
}

var goCapRefRe = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\.(Cap[A-Za-z0-9_]*)\b`)

func scanGoCapabilityReferences(t *testing.T, goAPIRoot string) map[string]string {
	t.Helper()

	titlePkgDir := filepath.Join(goAPIRoot, "internal", "domain", "title")
	out := make(map[string]string)
	for _, sub := range []string{"internal", "cmd", "scripts"} {
		root := filepath.Join(goAPIRoot, sub)
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" || path == titlePkgDir {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			collectGoCapRefs(string(data), filepath.ToSlash(rel), out)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
	return out
}

func collectGoCapRefs(code, rel string, out map[string]string) {
	for _, m := range goCapRefRe.FindAllStringSubmatchIndex(code, -1) {
		if code[m[2]:m[3]] == "games" {
			continue
		}
		name := code[m[4]:m[5]]
		if _, seen := out[name]; seen {
			continue
		}
		out[name] = rel + ":" + strconv.Itoa(1+strings.Count(code[:m[4]], "\n"))
	}
}

// sortedKeys rend les messages d'échec déterministes (ordre d'itération d'une map
// Go = aléatoire : sans tri, deux exécutions donnent deux diffs différents).
func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
