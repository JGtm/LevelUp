package coordination

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"strings"
	"testing"
)

// TestAucunTauxNu — GARDE-RAIL. Aucune fonction exportee de ce paquet ne rend un float64, sous
// AUCUN emballage : ni `float64`, ni `[]float64`, ni `map[string]float64`, ni `*float64`, ni un
// type nomme du paquet dont le sous-jacent est un float64. Un taux sort dans
// `domain.Couverture`, avec son compte brut, sa quantite par match et son drapeau
// d'echantillon faible.
//
// Sans ce test, la regle se perd au premier ajout (« juste un TauxDEchange() pour le KPI »), et
// l'appelant qui recoit un nombre seul n'a plus aucun moyen de savoir s'il tient une mesure ou
// un tirage a huit morts. La premiere version n'inspectait que `*ast.Ident` : elle laissait
// passer les quatre emballages ci-dessus, c'est-a-dire toutes les facons realistes de servir une
// serie de taux (revue ronde 1, 2026-09-06).
//
// Les helpers NON exportes restent libres : le risque est a la frontiere du paquet, pas dans son
// arithmetique interne.
//
// SECOND VOLET : le paquet ne declare AUCUN type struct exporte. Les types de resultat vivent
// dans `domain/` (arch-rules) — c'est ce qui garantit qu'un taux emballe dans une struct maison
// ne puisse pas contourner la premiere regle, et que les couches hautes n'aient qu'un seul
// vocabulaire.
func TestAucunTauxNu(t *testing.T) {
	fichiers := sourcesDuPaquet(t)
	declarations := typesDeclares(fichiers)

	for chemin, fichier := range fichiers {
		for _, decl := range fichier.Decls {
			verifieTypeExporte(t, chemin, decl)
			verifieFonctionExportee(t, chemin, decl, declarations)
		}
	}
}

// sourcesDuPaquet parse les fichiers non-test du paquet courant.
func sourcesDuPaquet(t *testing.T) map[string]*ast.File {
	t.Helper()
	fset := token.NewFileSet()
	paquets, err := parser.ParseDir(fset, ".", func(f fs.FileInfo) bool {
		return !strings.HasSuffix(f.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("analyse du paquet : %v", err)
	}
	fichiers := map[string]*ast.File{}
	for _, p := range paquets {
		for chemin, f := range p.Files {
			fichiers[chemin] = f
		}
	}
	if len(fichiers) == 0 {
		t.Fatalf("aucun fichier source analyse : le garde-rail ne garde rien")
	}
	return fichiers
}

// typesDeclares releve les types du paquet et leur sous-jacent, pour pouvoir suivre un alias
// jusqu'au float64 qu'il cache.
func typesDeclares(fichiers map[string]*ast.File) map[string]ast.Expr {
	declarations := map[string]ast.Expr{}
	for _, fichier := range fichiers {
		for _, decl := range fichier.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					declarations[ts.Name.Name] = ts.Type
				}
			}
		}
	}
	return declarations
}

// verifieTypeExporte : les types de resultat vivent dans domain/, pas ici.
func verifieTypeExporte(t *testing.T, chemin string, decl ast.Decl) {
	t.Helper()
	gen, ok := decl.(*ast.GenDecl)
	if !ok || gen.Tok != token.TYPE {
		return
	}
	for _, spec := range gen.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok || !ts.Name.IsExported() {
			continue
		}
		if _, struc := ts.Type.(*ast.StructType); struc {
			t.Errorf("%s : le type struct exporte %s ne doit pas vivre ici — les types de resultat "+
				"vivent dans internal/domain (arch-rules), sans quoi un taux emballe dans une struct "+
				"maison contournerait la regle du taux nu", chemin, ts.Name.Name)
		}
	}
}

// verifieFonctionExportee : aucun retour ne porte un float64, quel que soit son emballage.
func verifieFonctionExportee(t *testing.T, chemin string, decl ast.Decl, declarations map[string]ast.Expr) {
	t.Helper()
	fn, ok := decl.(*ast.FuncDecl)
	if !ok || !fn.Name.IsExported() || fn.Type.Results == nil {
		return
	}
	for _, res := range fn.Type.Results.List {
		if porteUnFloat64(res.Type, declarations, map[string]bool{}) {
			t.Errorf("%s : %s rend un float64 (nu ou emballe) — un taux sort dans domain.Couverture "+
				"(taux + brut + par match + echantillon faible), jamais seul", chemin, fn.Name.Name)
		}
	}
}

// porteUnFloat64 deballe un type de retour jusqu'a savoir s'il transporte un float64.
//
// Un `*ast.SelectorExpr` (domain.Couverture, time.Duration, ...) est TERMINAL et autorise : les
// types de resultat d'un autre paquet sont hors du perimetre de ce garde-rail, et c'est
// precisement la sortie que la regle demande.
func porteUnFloat64(expr ast.Expr, declarations map[string]ast.Expr, vus map[string]bool) bool {
	switch t := expr.(type) {
	case *ast.Ident:
		if t.Name == "float64" {
			return true
		}
		sousJacent, connu := declarations[t.Name]
		if !connu || vus[t.Name] {
			return false
		}
		vus[t.Name] = true
		return porteUnFloat64(sousJacent, declarations, vus)
	case *ast.ArrayType:
		return porteUnFloat64(t.Elt, declarations, vus)
	case *ast.Ellipsis:
		return porteUnFloat64(t.Elt, declarations, vus)
	case *ast.MapType:
		return porteUnFloat64(t.Value, declarations, vus)
	case *ast.StarExpr:
		return porteUnFloat64(t.X, declarations, vus)
	case *ast.ChanType:
		return porteUnFloat64(t.Value, declarations, vus)
	case *ast.StructType:
		return champsPortentUnFloat64(t.Fields, declarations, vus)
	case *ast.FuncType:
		return champsPortentUnFloat64(t.Results, declarations, vus)
	}
	return false
}

// champsPortentUnFloat64 : un des champs (ou des resultats) transporte-t-il un float64 ?
func champsPortentUnFloat64(champs *ast.FieldList, declarations map[string]ast.Expr, vus map[string]bool) bool {
	if champs == nil {
		return false
	}
	for _, c := range champs.List {
		if porteUnFloat64(c.Type, declarations, vus) {
			return true
		}
	}
	return false
}
