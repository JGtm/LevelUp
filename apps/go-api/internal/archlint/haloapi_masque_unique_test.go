package archlint

// haloapi_masque_unique_test.go : le masquage d'un secret de la brique « API Halo » a UNE seule
// définition, `haloapi.Masque` (`libs/haloapi/masque.go`). Il était recopié dans quatre
// sous-paquets (CLAUDE.md règle 6) ; une copie oubliée au prochain changement de masquage laisserait
// fuir un jeton dans un journal. Aucun fichier Go de l'arbre, tests compris, hors `masque.go`, ne
// déclare une fonction nommée `masque` ou `Masque`.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestHaloapiMasqueUnique(t *testing.T) {
	racine := racineDeclaree(t, racineBriqueHaloapi)
	var violations []string
	vus := 0
	err := filepath.WalkDir(racine, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || d.Name() == "masque.go" {
			return walkErr
		}
		vus++
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Recv == nil && (fn.Name.Name == "masque" || fn.Name.Name == "Masque") {
				violations = append(violations, fset.Position(fn.Pos()).String()+" : func "+fn.Name.Name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if vus == 0 {
		t.Fatalf("aucun fichier Go sous %s hors masque.go : le garde-rail ne couvre rien", racineBriqueHaloapi)
	}
	if len(violations) > 0 {
		t.Errorf("le masquage d'un secret est redéclaré sous libs/haloapi (une seule définition : "+
			"haloapi.Masque, masque.go) :\n  %s", strings.Join(violations, "\n  "))
	}
}
