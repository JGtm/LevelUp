package archlint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// goSource est un fichier Go non-test du module, déjà analysé.
type goSource struct {
	Rel  string // chemin relatif à apps/go-api, séparateurs "/"
	Src  []byte
	Fset *token.FileSet
	File *ast.File
}

// apiModuleRoot rend la racine du module apps/go-api.
func apiModuleRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
}

// goSourcesUnder analyse les fichiers Go non-test sous les racines données (relatives à
// apps/go-api, séparateurs "/"). Échoue si une racine n'existe pas ou ne contient rien :
// un garde-rail qui ne voit aucun fichier ne garde rien.
func goSourcesUnder(t *testing.T, roots ...string) []goSource {
	t.Helper()
	base := apiModuleRoot(t)
	var out []goSource
	for _, root := range roots {
		dir := filepath.Join(base, filepath.FromSlash(root))
		before := len(out)
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(base, path)
			out = append(out, goSource{Rel: filepath.ToSlash(rel), Src: src, Fset: fset, File: file})
			return nil
		})
		if err != nil {
			t.Fatalf("parcours de %s : %v", root, err)
		}
		if len(out) == before {
			t.Fatalf("aucun fichier Go sous %s : le garde-rail ne couvre rien", root)
		}
	}
	return out
}

// parseGoSnippet analyse une source de test (auto-preuve des garde-rails).
func parseGoSnippet(t *testing.T, rel, src string) goSource {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, rel, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("source de test invalide : %v", err)
	}
	return goSource{Rel: rel, Src: []byte(src), Fset: fset, File: file}
}
