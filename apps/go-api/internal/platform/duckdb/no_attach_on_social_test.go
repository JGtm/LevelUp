// Package duckdb — test sentinel anti-régression : aucun ATTACH ne doit être
// exécuté sur la connexion shared_social.duckdb (socialDB) dans tout le projet.
//
// Contexte : ATTACH sur une conn RW écrit dans le WAL une entrée non-rejouable
// au reboot (bug DuckDB upstream #7659). Cf. commit fix + thought_log 2026-05-25.
//
// Ce test parse tous les fichiers .go du projet (sauf vendor + tests) et
// vérifie qu'aucune occurrence ne :
//
//  1. Appelle `attachGlobalXuidAliases(*, socialDB, *)` ou `attachShared*(*, socialDB, *)`
//  2. Exécute un SQL contenant `ATTACH ` directement sur une variable nommée
//     `socialDB`, `sharedSocialDB`, ou similaire
//
// Limite : ne détecte pas les ATTACH via abstractions (interfaces, méthodes
// nommées différemment). Pragmatique : couvre les sites historiques connus.

package duckdb

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoATTACHOnSocialDB scanne tout le projet pour les appels à
// attachGlobalXuidAliases ou méthodes ATTACH-like ciblant socialDB. Échoue si
// l'un est trouvé — empêche la régression du bug WAL corruption.
func TestNoATTACHOnSocialDB(t *testing.T) {
	// Remonter à la racine apps/go-api/ pour scanner tous les packages.
	root := findGoAPIRoot(t)

	var violations []string

	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == "vendor" || name == "tmp" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil // skip tests (peuvent contenir ATTACH pour reproduire le bug en test)
		}

		f, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if parseErr != nil {
			return nil // skip fichiers non-parsables (CGO, build tags, etc.)
		}

		ast.Inspect(f, func(n ast.Node) bool {
			// Cas 1 : appel à attachGlobalXuidAliases(*, socialDB, *) ou variations
			if call, ok := n.(*ast.CallExpr); ok {
				if ident, ok := call.Fun.(*ast.Ident); ok && isATTACHFuncName(ident.Name) {
					for _, arg := range call.Args {
						if argIdent, ok := arg.(*ast.Ident); ok && isSocialDBVarName(argIdent.Name) {
							pos := fset.Position(call.Pos())
							violations = append(violations, formatViolation(pos, ident.Name+"("+argIdent.Name+")"))
						}
					}
				}
			}
			// Cas 2 : string literal SQL "ATTACH ..." dans un ExecContext ciblant socialDB
			// Heuristique simple : ExecContext/Exec sur receiver nommé socialDB-like.
			if call, ok := n.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					methodName := sel.Sel.Name
					if methodName != "Exec" && methodName != "ExecContext" && methodName != "Query" && methodName != "QueryContext" {
						return true
					}
					recvIdent, ok := sel.X.(*ast.Ident)
					if !ok || !isSocialDBVarName(recvIdent.Name) {
						return true
					}
					for _, arg := range call.Args {
						if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
							if containsATTACHKeyword(lit.Value) {
								pos := fset.Position(call.Pos())
								violations = append(violations, formatViolation(pos, recvIdent.Name+"."+methodName+"(ATTACH ...)"))
							}
						}
					}
				}
			}
			return true
		})
		return nil
	})

	if err != nil {
		t.Fatalf("WalkDir: %v", err)
	}

	if len(violations) > 0 {
		t.Fatalf("ATTACH detecté sur socialDB — bug DuckDB #7659 ré-introduit :\n%s\n"+
			"Fix : faire la jointure cross-DB en Go (cf. ops/media_associate.go) au lieu de ATTACH sur shared_social RW.",
			strings.Join(violations, "\n"))
	}
}

func isATTACHFuncName(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "attach") &&
		(strings.Contains(lower, "global") || strings.Contains(lower, "shared") || strings.Contains(lower, "xuid"))
}

func isSocialDBVarName(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "social")
}

func containsATTACHKeyword(litValue string) bool {
	// litValue est inclus avec ses délimiteurs (" ou `).
	upper := strings.ToUpper(litValue)
	return bytes.Contains([]byte(upper), []byte("ATTACH "))
}

func formatViolation(pos token.Position, what string) string {
	return "  " + pos.Filename + ":" + intToStr(pos.Line) + " — " + what
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// findGoAPIRoot remonte depuis le test courant jusqu'à apps/go-api/.
func findGoAPIRoot(t *testing.T) string {
	t.Helper()
	// Le test tourne depuis internal/platform/duckdb/ → remonter 3 niveaux.
	wd, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	// On veut le répertoire qui contient internal/ et cmd/.
	root := wd
	for i := 0; i < 5; i++ {
		if _, err := filepath.Glob(filepath.Join(root, "internal")); err == nil {
			if matches, _ := filepath.Glob(filepath.Join(root, "internal")); len(matches) > 0 {
				if matches2, _ := filepath.Glob(filepath.Join(root, "cmd")); len(matches2) > 0 {
					return root
				}
			}
		}
		root = filepath.Dir(root)
	}
	t.Fatalf("apps/go-api/ root introuvable depuis %s", wd)
	return ""
}
