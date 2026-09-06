package coordination

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"strings"
	"testing"
)

// LISTE BLANCHE DES TYPES DE RETOUR — 2026-09-06, revue ronde 2.
//
// Ce paquet ne rend que ces types-la, et les conteneurs dont CHAQUE composant (cle comprise)
// est lui-meme de cette liste. Tout le reste est refuse : `float64` nu, tout type nomme hors
// liste, `map[float64]T`, une struct anonyme, un type d'un autre paquet non cite ici.
//
// POURQUOI UNE LISTE BLANCHE ET NON UNE LISTE NOIRE. La version precedente cherchait des
// `float64` a travers les emballages et exemptait tout type d'un AUTRE paquet. Elle laissait
// donc passer la mutation la plus probable de toutes : `type TauxEchange float64` dans
// `domain`, puis `func TauxDEchange(...) domain.TauxEchange` ici. Un taux aurait retrouve sa
// forme de nombre seul par le chemin le plus court, sans qu'aucun `float64` n'apparaisse dans
// ce paquet. Une liste noire ne peut pas gagner cette course : il y a une infinite de facons
// d'emballer un nombre, et une seule liste de ce que ce paquet a le droit de rendre.
//
// AJOUTER UN TYPE A CETTE LISTE EXIGE UNE JUSTIFICATION DATEE, ICI MEME. Un type qui porte un
// taux le porte SOUS `domain.Couverture` (taux + brut + par match + N + echantillon faible) :
// c'est la seule forme sous laquelle un taux quitte ce paquet.
var (
	// identsAutorises : les types de base sans danger. Aucun ne peut porter un taux.
	identsAutorises = map[string]bool{
		"error":  true,
		"bool":   true,
		"int":    true,
		"int64":  true,
		"string": true,
	}

	// typesQualifiesAutorises : les types de resultat rendus AUJOURD'HUI par ce paquet,
	// verifies sur pieces le 2026-09-06 — `Mesurer` rend domain.Couverture, `Echanges` rend
	// domain.BilanEchanges. `domain.PaireEchange` n'y figure PAS : il voyage a l'interieur
	// du bilan, aucune fonction exportee ne le rend directement.
	//
	// `domain.MortSuivie` AJOUTE LE 2026-09-06 (phase 3, histogramme du delai d'echange de
	// la page Escouade). JUSTIFICATION : `Ripostes` le rend pour que l'appelant puisse
	// BINNER des DELAIS (des millisecondes, ADR 0010 : pre-binning serveur). Le type ne
	// porte AUCUN quotient — MatchID, deux xuids, un instant, trois booleens et un delai en
	// int64 : il n'y a rien dedans qu'un lecteur puisse prendre pour un taux, et le seul
	// taux de ce paquet reste celui de Mesurer.
	typesQualifiesAutorises = map[string]bool{
		"domain.Couverture":    true,
		"domain.BilanEchanges": true,
		"domain.MortSuivie":    true,
	}
)

// TestAucunTauxNu — GARDE-RAIL, deux volets.
//
//  1. Aucune fonction exportee ne rend un type hors de la liste blanche ci-dessus. Sans lui, la
//     regle se perd au premier ajout (« juste un TauxDEchange() pour le KPI »), et l'appelant
//     qui recoit un nombre seul n'a plus aucun moyen de savoir s'il tient une mesure ou un
//     tirage a huit morts.
//  2. Le paquet ne declare AUCUN type struct exporte : les types de resultat vivent dans
//     `domain/` (arch-rules). C'est ce qui empeche de contourner le premier volet en emballant
//     un taux dans une struct maison.
//
// Les helpers NON exportes restent libres : le risque est a la frontiere du paquet, pas dans
// son arithmetique interne.
func TestAucunTauxNu(t *testing.T) {
	fichiers := sourcesDuPaquet(t)

	inspectees := 0
	for chemin, fichier := range fichiers {
		for _, decl := range fichier.Decls {
			verifieTypeExporte(t, chemin, decl)
			if verifieFonctionExportee(t, chemin, decl) {
				inspectees++
			}
		}
	}

	// Sentinelle anti-vacuite : un garde-rail qui n'inspecte plus rien passe au vert en
	// silence, et c'est exactement ce qu'un refactor du parcours ci-dessus produirait.
	if inspectees == 0 {
		t.Fatalf("aucune fonction exportee inspectee : le garde-rail ne garde rien")
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
				"maison contournerait la liste blanche", chemin, ts.Name.Name)
		}
	}
}

// verifieFonctionExportee controle les retours d'une fonction exportee ; rend vrai si elle a
// ete inspectee (c'est ce compte qui alimente la sentinelle anti-vacuite).
func verifieFonctionExportee(t *testing.T, chemin string, decl ast.Decl) bool {
	t.Helper()
	fn, ok := decl.(*ast.FuncDecl)
	if !ok || !fn.Name.IsExported() || fn.Type.Results == nil {
		return false
	}
	for _, res := range fn.Type.Results.List {
		if !typeAutorise(res.Type) {
			t.Errorf("%s : %s rend %s, hors de la liste blanche des types de retour — un taux sort "+
				"dans domain.Couverture (taux + brut + par match + echantillon faible), jamais seul "+
				"ni sous un type nomme qui le deguiserait. Ajouter un type a la liste exige une "+
				"justification datee dans no_naked_rate_test.go", chemin, fn.Name.Name, types.ExprString(res.Type))
		}
	}
	return true
}

// typeAutorise : le type est-il dans la liste blanche, ou un conteneur dont TOUS les
// composants y sont ? Tout ce qui n'est pas reconnu est refuse — c'est le sens d'une liste
// blanche.
func typeAutorise(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.Ident:
		return identsAutorises[t.Name]
	case *ast.SelectorExpr:
		pkg, ok := t.X.(*ast.Ident)
		return ok && typesQualifiesAutorises[pkg.Name+"."+t.Sel.Name]
	case *ast.ArrayType:
		return typeAutorise(t.Elt)
	case *ast.Ellipsis:
		return typeAutorise(t.Elt)
	case *ast.MapType:
		// La CLE compte autant que la valeur : `map[float64]int` est une serie de taux.
		return typeAutorise(t.Key) && typeAutorise(t.Value)
	case *ast.StarExpr:
		return typeAutorise(t.X)
	case *ast.ChanType:
		return typeAutorise(t.Value)
	case *ast.FuncType:
		return resultatsAutorises(t.Results)
	}
	return false
}

// resultatsAutorises : tous les resultats d'une signature sont-ils autorises ?
func resultatsAutorises(champs *ast.FieldList) bool {
	if champs == nil {
		return true
	}
	for _, c := range champs.List {
		if !typeAutorise(c.Type) {
			return false
		}
	}
	return true
}
