package coordination

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"strings"
	"testing"
)

// TestAucunTauxNu — GARDE-RAIL. Aucune fonction exportee de ce paquet ne rend un float64 :
// un taux sort dans `domain.Couverture`, avec son compte brut, sa quantite par match et son
// drapeau d'echantillon faible.
//
// Sans ce test, la regle se perd au premier ajout (« juste un TauxDEchange() pour le KPI »),
// et l'appelant qui recoit un nombre seul n'a plus aucun moyen de savoir s'il tient une
// mesure ou un tirage a huit morts. Les helpers NON exportes restent libres : le risque est
// a la frontiere du paquet, pas dans son arithmetique interne.
func TestAucunTauxNu(t *testing.T) {
	fset := token.NewFileSet()
	paquets, err := parser.ParseDir(fset, ".", func(f fs.FileInfo) bool {
		return !strings.HasSuffix(f.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("analyse du paquet : %v", err)
	}
	if len(paquets) == 0 {
		t.Fatalf("aucun fichier source analyse : le garde-rail ne garde rien")
	}

	vues := 0
	for _, p := range paquets {
		for chemin, fichier := range p.Files {
			for _, decl := range fichier.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || !fn.Name.IsExported() {
					continue
				}
				vues++
				if fn.Type.Results == nil {
					continue
				}
				for _, res := range fn.Type.Results.List {
					ident, ok := res.Type.(*ast.Ident)
					if ok && ident.Name == "float64" {
						t.Errorf("%s : %s rend un float64 nu — un taux sort dans domain.Couverture "+
							"(taux + brut + par match + echantillon faible), jamais seul", chemin, fn.Name.Name)
					}
				}
			}
		}
	}
	if vues == 0 {
		t.Fatalf("aucune fonction exportee vue : le garde-rail ne garde rien")
	}
}
