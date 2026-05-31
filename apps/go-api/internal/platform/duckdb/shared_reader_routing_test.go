// Test anti-régression Phase 3 plan stabilisation 2026-05-22 :
//
// VOLONTAIREMENT PAS `//go:build integration` (retiré 2026-05-31) : ces deux
// tests sont du scan AST/statique PUR (aucune connexion DuckDB, aucune fixture,
// aucun CGO requis). Ils doivent tourner dans le gate par défaut `go test ./...`
// — sinon ils ne protègent rien. C'est précisément leur absence du gate par
// défaut qui a laissé shipper les bugs shared.highlight_events / shared.medals_earned
// (cf. thought_log 2026-05-31). Tout le reste du package est integration-tagged.
//
// **Test 1 — TestSharedReader_AllHomeRepoCallsRouted**
//
// Vérifie statiquement qu'aucune query référençant `shared.X` n'est exécutée
// via `pdb.ReadDB()` (player conn) — pattern interdit depuis ADR 0016
// (retrait de l'ATTACH shared sur la player conn).
//
// **Test 2 — TestSharedReader_NoSharedPrefixViaSharedReader** (ajouté 2026-05-26)
//
// Vérifie l'INVERSE : aucune query référençant `shared.X` ne doit non plus
// être exécutée via `pdb.SharedReadDB().Get(ctx)`. Cas révélé par le bug
// weapon_kills_repo (chart breakdown armes non affiché) : `SharedReader.Get`
// retourne une connexion directe à `shared_matches_v2.duckdb` où `shared`
// n'est ni schéma ni catalogue par défaut — les queries `shared.X` échouent
// silencieusement (capability not supported → nil → frontend masque le chart).
//
// Approche commune : scanner les fichiers Go du package duckdb pour identifier
// les const SQL contenant `shared.X` ET les SQL inline (string literals dans
// des function bodies via strings.Builder.WriteString), puis croiser avec
// les callers `.SharedReadDB().Get(`.
package duckdb

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// queryConstRe : matche `const Q<Name> = ` suivi d'une backtick-string
// (peut être multi-line). Utilisé par Test 1 (routing) qui ne cible que les
// const conventionnellement préfixées Q*.
var queryConstRe = regexp.MustCompile(`(?s)const\s+(Q\w+)\s*=\s*` + "`" + `([^` + "`" + `]+)` + "`")

// sqlConstRe : matche TOUTE const = backtick-string, quel que soit son nom
// (pas seulement Q*). Utilisé par Test 2 pour indexer aussi les const SQL non
// préfixées Q comme `highlightEventsBaseSelect` — angle mort historique qui a
// laissé passer le bug shared.highlight_events (cf. thought_log 2026-05-31).
var sqlConstRe = regexp.MustCompile(`(?s)const\s+(\w+)\s*=\s*` + "`" + `([^` + "`" + `]+)` + "`")

// readDBCallRe : matche `<receiver>.pdb.ReadDB().Query(...)` ou
// `<receiver>.pdb.ReadDB().QueryRow(...)`. Le contenu après "Query(" jusqu'à
// la virgule suivante donne l'identifiant de la query (ou expression sprintf).
var readDBCallRe = regexp.MustCompile(`\.pdb\.ReadDB\(\)\.(?:Query|QueryRow)\([^,)]*,\s*([A-Z]\w+)`)

// sprintfTemplateRe : matche `r.pdb.ReadDB().Query(ctx, query, args...)` après
// `query := fmt.Sprintf(Q*, ...)`. Trace l'identifiant de la const template.
var sprintfTemplateRe = regexp.MustCompile(`fmt\.Sprintf\((Q\w+Tpl|Q\w+Template)`)

// TestSharedReader_AllHomeRepoCallsRouted scanne le package duckdb et fait
// échouer le test si une const Q* contenant "shared." est utilisée via
// .ReadDB() (player conn).
func TestSharedReader_AllHomeRepoCallsRouted(t *testing.T) {
	// 1. Collecter toutes les const Q* du package + identifier celles qui
	//    réfèrent "shared." dans leur SQL.
	files, err := goFilesInPackage(t, ".")
	if err != nil {
		t.Fatalf("list files: %v", err)
	}

	sharedQueries := map[string]string{} // const name → snippet (premier 80 chars)
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		content := string(raw)
		for _, m := range queryConstRe.FindAllStringSubmatch(content, -1) {
			name := m[1]
			sql := m[2]
			if containsSharedRef(sql) {
				snippet := strings.TrimSpace(sql)
				if len(snippet) > 80 {
					snippet = snippet[:80] + "..."
				}
				sharedQueries[name] = snippet
			}
		}
	}
	if len(sharedQueries) == 0 {
		t.Skip("aucune query const contenant 'shared.' détectée — test sans objet")
	}
	t.Logf("queries 'shared.*' détectées (%d) :", len(sharedQueries))
	for name := range sharedQueries {
		t.Logf("  - %s", name)
	}

	// 2. Pour chaque fichier code (hors tests), trouver les appels
	//    .pdb.ReadDB().(Query|QueryRow)(..., QUERY_CONST, ...) qui
	//    référencent une const "shared.*".
	var violations []string
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		content := string(raw)

		// Direct calls : .ReadDB().Query(ctx, QUERY_CONST, ...)
		for _, m := range readDBCallRe.FindAllStringSubmatch(content, -1) {
			constName := m[1]
			if snippet, isShared := sharedQueries[constName]; isShared {
				violations = append(violations, formatRoutingViolation(path, constName, snippet))
			}
		}

		// Sprintf templates : query := fmt.Sprintf(Q*Tpl/Template, ...) puis ReadDB().Query(query, ...)
		// Heuristique : si le fichier contient un fmt.Sprintf(Q*Tpl) ET un .ReadDB().Query(query,
		// dans la même fonction, on flag.
		tplMatches := sprintfTemplateRe.FindAllStringSubmatch(content, -1)
		for _, tm := range tplMatches {
			tplName := tm[1]
			if snippet, isShared := sharedQueries[tplName]; isShared {
				// Check si le fichier appelle ReadDB().Query(query, après ce Sprintf
				// (heuristique simple : la même fonction réutilise `query` issu du Sprintf).
				if strings.Contains(content, ".pdb.ReadDB().Query(ctx, query") {
					violations = append(violations, formatRoutingViolation(path, tplName+" (via fmt.Sprintf+query)", snippet))
				}
			}
		}
	}

	if len(violations) > 0 {
		t.Errorf("régression Phase 3 ADR 0016 : %d query 'shared.*' exécutée(s) via pdb.ReadDB() (player conn) au lieu de SharedReader :", len(violations))
		for _, v := range violations {
			t.Errorf("  %s", v)
		}
		t.Errorf("\nFix : remplacer `r.pdb.ReadDB().Query(...)` par `sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx); defer release(); sharedDB.QueryContext(ctx, ...)`")
		t.Errorf("Et retirer le préfixe `shared.` de la const SQL (le conn pointe déjà sur shared_matches_v2).")
	}
}

// TestSharedReader_NoSharedPrefixViaSharedReader fait échouer le test si une
// fonction qui appelle `.SharedReadDB().Get(` contient ou référence du SQL
// avec `FROM shared.X` ou `JOIN shared.X`. Couvre 2 cas :
//
//  1. **Const Q*** : `db.QueryContext(ctx, Q*, ...)` où la const contient `shared.X`.
//  2. **SQL inline** : SQL construit via `strings.Builder.WriteString(\`...shared.X...\`)`
//     ou passé en backtick-string directement à QueryContext (cas weapon_kills_repo).
//
// Implémentation : parsing AST de chaque .go non-test, analyse fonction par
// fonction. Pour chaque fonction qui contient un appel à `.SharedReadDB().Get(`,
// on examine tous les `*ast.BasicLit` (string literals) ET toutes les références
// à des identifiants (consts) — si l'un d'eux match le pattern `shared.X`,
// c'est une violation.
//
// Pas de whitelist : si vous gardez du SQL `shared.X` en legacy/dead code,
// déplacez-le dans une fonction sans `SharedReader.Get`, ou supprimez-le.
func TestSharedReader_NoSharedPrefixViaSharedReader(t *testing.T) {
	files, err := goFilesInPackage(t, ".")
	if err != nil {
		t.Fatalf("list files: %v", err)
	}

	// 1. Indexer TOUTES les const = backtick-string du package : nom → SQL.
	//    (broadened : sqlConstRe, pas queryConstRe — sinon les const SQL non
	//    préfixées Q comme highlightEventsBaseSelect échappent à l'analyse.)
	allConsts := map[string]string{}
	fset := token.NewFileSet()
	var fileASTs []*ast.File
	// funcsByName : index package-wide nom_fonction → FuncDecl, pour suivre les
	// helpers (ex: buildHighlightEventsQuery) appelés depuis la fonction qui
	// fait SharedReadDB().Get() mais où le SQL `shared.` est réellement écrit.
	funcsByName := map[string]*ast.FuncDecl{}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, m := range sqlConstRe.FindAllStringSubmatch(string(raw), -1) {
			allConsts[m[1]] = m[2]
		}
		fileAST, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		fileASTs = append(fileASTs, fileAST)
		for _, decl := range fileAST.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Body != nil {
				funcsByName[fn.Name.Name] = fn
			}
		}
	}

	var violations []string

	// scanBody inspecte un corps de fonction : flag les BasicLit string et les
	// ident → const dont le SQL contient `shared.X`. `origin` identifie d'où
	// vient le SQL (fonction directe ou helper) pour le message d'erreur.
	scanBody := func(body ast.Node, path, fnName, origin string) {
		ast.Inspect(body, func(inner ast.Node) bool {
			switch x := inner.(type) {
			case *ast.BasicLit:
				if x.Kind == token.STRING && containsSharedRef(x.Value) {
					line := fset.Position(x.Pos()).Line
					snippet := strings.TrimSpace(stripQuotes(x.Value))
					if len(snippet) > 80 {
						snippet = snippet[:80] + "..."
					}
					violations = append(violations, formatSharedReaderViolation(
						path, line, fnName, origin+"inline SQL", snippet))
				}
			case *ast.Ident:
				sql, found := allConsts[x.Name]
				if !found || !containsSharedRef(sql) {
					return true
				}
				line := fset.Position(x.Pos()).Line
				snippet := strings.TrimSpace(sql)
				if len(snippet) > 80 {
					snippet = snippet[:80] + "..."
				}
				violations = append(violations, formatSharedReaderViolation(
					path, line, fnName, origin+x.Name, snippet))
			}
			return true
		})
	}

	for _, fileAST := range fileASTs {
		path := fset.Position(fileAST.Pos()).Filename
		ast.Inspect(fileAST, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				return true
			}
			if !funcCallsSharedReaderGet(fn) {
				return true
			}
			fnName := fn.Name.Name

			// a. Corps direct de la fonction qui fait SharedReadDB().Get().
			scanBody(fn.Body, path, fnName, "")

			// b. Helpers appelés depuis cette fonction (1 niveau de profondeur) :
			//    le SQL `shared.` est souvent écrit dans un build*Query() séparé,
			//    pas inline dans la fonction qui ouvre la conn.
			for callee := range collectCalleeNames(fn.Body) {
				if h, ok := funcsByName[callee]; ok && h != fn {
					scanBody(h.Body, path, fnName, "via "+callee+"() → ")
				}
			}
			return true
		})
	}

	if len(violations) == 0 {
		return
	}
	// Dédup (un même const référencé plusieurs fois dans une fonction).
	seen := map[string]struct{}{}
	unique := violations[:0]
	for _, v := range violations {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		unique = append(unique, v)
	}
	t.Errorf("%d query 'shared.*' exécutée(s) via pdb.SharedReadDB().Get() — interdit (la conn pointe directement sur shared_matches_v2.duckdb, `shared.` n'est ni schéma ni catalogue résolvable) :", len(unique))
	for _, v := range unique {
		t.Errorf("  %s", v)
	}
	t.Errorf("\nFix : retirer le préfixe `shared.` du SQL — les tables sont accessibles directement par leur nom (v_weapon_kills, xuid_aliases, match_participants, ...).")
}

// funcCallsSharedReaderGet retourne true si la fonction contient au moins un
// appel à `.SharedReadDB().Get(`.
func funcCallsSharedReaderGet(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Get" {
			return true
		}
		// sel.X doit être un appel à .SharedReadDB()
		inner, ok := sel.X.(*ast.CallExpr)
		if !ok {
			return true
		}
		innerSel, ok := inner.Fun.(*ast.SelectorExpr)
		if !ok || innerSel.Sel.Name != "SharedReadDB" {
			return true
		}
		found = true
		return false
	})
	return found
}

// collectCalleeNames retourne l'ensemble des noms de fonctions appelées
// directement (forme `foo(...)`) dans un corps. Sert à suivre les helpers
// build*Query() depuis la fonction qui ouvre la conn SharedReader.
func collectCalleeNames(body ast.Node) map[string]struct{} {
	out := map[string]struct{}{}
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if id, ok := call.Fun.(*ast.Ident); ok {
			out[id.Name] = struct{}{}
		}
		return true
	})
	return out
}

// stripQuotes retire les guillemets/backticks autour d'un BasicLit STRING.
func stripQuotes(s string) string {
	if len(s) >= 2 && (s[0] == '`' || s[0] == '"') {
		return s[1 : len(s)-1]
	}
	return s
}

// containsSharedRef retourne true si le SQL contient une référence cross-DB
// vers le schéma `shared.` (FROM ou JOIN). Insensible à la casse.
func containsSharedRef(sql string) bool {
	lower := strings.ToLower(sql)
	if strings.Contains(lower, "from shared.") {
		return true
	}
	if strings.Contains(lower, "join shared.") {
		return true
	}
	return false
}

// goFilesInPackage liste les .go du dossier (non récursif).
func goFilesInPackage(t *testing.T, dir string) ([]string, error) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		out = append(out, filepath.Join(dir, name))
	}
	return out, nil
}

func formatRoutingViolation(path, constName, snippet string) string {
	return path + " uses " + constName + " via .ReadDB() — SQL: " + snippet
}

func formatSharedReaderViolation(path string, line int, fnName, source, snippet string) string {
	return filepath.ToSlash(path) + ":" + itoa(line) + " in " + fnName + "() — " + source + ": " + snippet
}

// itoa minimal (évite import strconv pour rester aligné avec le style du file).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
