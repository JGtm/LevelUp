package main

// cmd_backfill_killsource_selection_unique_test.go — GARDE-RAIL : `backfill-killsource` N A QU UNE
// SELECTION, pour le plan (`--dry-run`) comme pour la passe (cmd_backfill_killsource_carte.go).
//
// Les deux lectures du registre ([filmsACollecter] hors ligne, [matchsSansPasseDeFilm] en ligne) ne
// s appellent, hors des tests, que depuis la fonction de selection de leur passe — celle qui retire
// les matchs sans carte avant `--limit`. Un plan qui les appellerait directement annoncerait des
// matchs que la passe retire. MUTATION QUI LE FAIT ROUGIR : un `--dry-run` qui rend la main sur
// `filmsACollecter(…)` avant la construction du collecteur.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

func TestSelectionDeLaPasse_UnSeulAppelantParLecture(t *testing.T) {
	appelantUnique := map[string]string{
		"filmsACollecter":       "candidatsDeLaPasse",
		"matchsSansPasseDeFilm": "idsDeLaPasseEnLigne",
	}
	entrees, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	fset := token.NewFileSet()
	vus := map[string]int{}
	for _, e := range entrees {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, nom, nil, 0)
		if err != nil {
			t.Fatalf("%s : %v", nom, err)
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				appel, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				id, ok := appel.Fun.(*ast.Ident)
				if !ok {
					return true
				}
				attendu, garde := appelantUnique[id.Name]
				if !garde {
					return true
				}
				vus[id.Name]++
				if fn.Name.Name != attendu {
					t.Errorf("%s : %s appelle %s — seule %s le fait, pour que le plan et la passe aient la "+
						"meme selection", fset.Position(appel.Pos()), fn.Name.Name, id.Name, attendu)
				}
				return true
			})
		}
	}
	for lecture := range appelantUnique {
		if vus[lecture] != 1 {
			t.Errorf("%s appele %d fois hors des tests, attendu 1 (depuis %s)", lecture, vus[lecture],
				appelantUnique[lecture])
		}
	}
}
