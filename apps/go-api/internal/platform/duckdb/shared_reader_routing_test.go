//go:build integration

// Test anti-régression Phase 3 plan stabilisation 2026-05-22 :
// TestSharedReader_AllHomeRepoCallsRouted vérifie statiquement qu'aucune query
// référençant `shared.X` n'est exécutée via `pdb.ReadDB()` (player conn) — pattern
// interdit depuis ADR 0016 (retrait de l'ATTACH shared sur la player conn).
//
// Approche : scanner les fichiers Go du package duckdb pour identifier :
//  1. Les constantes SQL qui contiennent "FROM shared." ou "JOIN shared."
//  2. Les appels `pdb.ReadDB().Query(...)` ou `r.pdb.ReadDB().QueryRow(...)`
//
// Croisement : si une const SQL "shared.*" est utilisée via ReadDB → fail.
//
// Si quelqu'un réintroduit le pattern (par oubli ou par refactor sans audit),
// ce test échoue immédiatement — pas besoin d'attendre un crash runtime.
package duckdb

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// queryConstRe : matche `const Q<Name> = ` suivi d'une backtick-string
// (peut être multi-line).
var queryConstRe = regexp.MustCompile(`(?s)const\s+(Q\w+)\s*=\s*` + "`" + `([^` + "`" + `]+)` + "`")

// readDBCallRe : matche `<receiver>.pdb.ReadDB().Query(...)` ou
// `<receiver>.pdb.ReadDB().QueryRow(...)`. Le contenu après "Query(" jusqu'à
// la virgule suivante donne l'identifiant de la query (ou expression sprintf).
var readDBCallRe = regexp.MustCompile(`\.pdb\.ReadDB\(\)\.(?:Query|QueryRow)\([^,)]*,\s*([A-Z]\w+)`)

// sprintfReadDBRe : matche `r.pdb.ReadDB().Query(ctx, query, args...)` après
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
				violations = append(violations, formatViolation(path, constName, snippet))
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
					violations = append(violations, formatViolation(path, tplName+" (via fmt.Sprintf+query)", snippet))
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

func formatViolation(path, constName, snippet string) string {
	return path + " uses " + constName + " via .ReadDB() — SQL: " + snippet
}
