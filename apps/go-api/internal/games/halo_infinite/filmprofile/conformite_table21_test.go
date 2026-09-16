package filmprofile_test

// conformite_table21_test.go — LE CATALOGUE COMMIS EST LA TABLE DU LOT 2.1, LIGNE POUR LIGNE.
//
// # CE QUE CE TEST GARDE
//
// Le lot 3.1 sort le profil du code : la table `filmdec.TableProfil()` (lot 2.1) devient un
// fichier de reference, `data/titles/halo_infinite/reference/film_profiles.json`. Tant que les
// deux coexistent — le lot 3.1.1 supprime la copie en faisant LIRE le fichier a `filmdec` —
// une divergence entre elles serait un profil a deux verites, dont personne ne saurait laquelle
// le decodeur applique. Ce test les tient egales : meme ordre, memes clefs, memes valeurs,
// memes provenances, memes preuves, memes dates.
//
// # POURQUOI IL LIT LE FICHIER GO AU LIEU D IMPORTER `filmdec`
//
// Trois raisons, dans l ordre de leur poids :
//
//  1. LE PAQUET RESTE DEHORS. `filmprofile` ne doit rien devoir au film — c est sa raison
//     d etre : un consommateur qui n a pas le decodeur (un outil, une verification de
//     livraison, un futur titre) doit pouvoir lire ce que le depot sait d un build. Un import
//     dans le test ferait entrer `filmdec` dans la compilation du paquet de test, et la
//     frontiere ne serait plus verifiable d un coup d oeil sur les imports.
//  2. LE SENS DE LA DEPENDANCE S INVERSE AU LOT 3.1.1. Quand `filmdec` importera `filmprofile`,
//     un import en sens inverse dans le test de `filmprofile` serait un cycle passant par le
//     paquet de test externe : legal en Go, mais une dette posee sciemment a un lot d ici.
//  3. LA PREUVE PORTE SUR CE QUI EST DANS L ARBRE. Le test lit le FICHIER commis, pas le
//     binaire compile : il dit vrai meme si `filmdec` ne compile plus.
//
// Le prix est une petite analyse syntaxique (constantes de fichier, concatenations). Elle est
// bornee par [lignesAttenduesParTable] : une analyse qui rendrait moins de lignes que la table
// n en porte serait VERTE sans rien avoir compare — c est le defaut classique de ce genre de
// garde, et le plancher est ce qui le ferme.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/filmprofile"
	"levelup/go-api/internal/testutil"
)

// cheminTableProfil : le fichier de la table du lot 2.1, relatif a la racine du depot.
var cheminTableProfil = filepath.Join("apps", "go-api", "internal", "games", "halo_infinite",
	"film", "filmdec", "profile_table.go")

// tablesDuProfil : les quatre fonctions de la table, DANS L ORDRE ou `TableProfil()` les
// concatene, avec le nombre de lignes mesure le 2026-09-16. Le compte est ecrit pour que
// l ajout d une ligne a la table oblige a passer par ici — c est-a-dire par le catalogue.
var tablesDuProfil = []struct {
	fonction string
	lignes   int
}{
	{"tableProfilFormat", 2},
	{"tableProfilBuild", 7},
	{"tableProfilMajeure", 3},
	{"tableProfilInvariants", 11},
}

// lignesAttenduesParTable : le total des quatre, calcule une fois.
func lignesAttenduesParTable() int {
	n := 0
	for _, t := range tablesDuProfil {
		n += t.lignes
	}
	return n
}

// TestCatalogueConformeALaTableDuLot21 — LA GARDE.
//
// Mutation qui doit le faire rougir : changer une valeur, une provenance ou une date dans
// `profile_table.go` ou dans `film_profiles.json` sans toucher l autre.
func TestCatalogueConformeALaTableDuLot21(t *testing.T) {
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	table := lireTableProfil(t, filepath.Join(racine, cheminTableProfil))
	if len(table) != lignesAttenduesParTable() {
		t.Fatalf("%d ligne(s) lue(s) dans %s, %d attendue(s) — l analyse de la table s est "+
			"cassee, ou la table a bouge sans que le catalogue suive",
			len(table), cheminTableProfil, lignesAttenduesParTable())
	}

	chemin := title.NewPathResolver(racine).FilmProfilesPath(title.DefaultSlug)
	cat, err := filmprofile.Charger(chemin)
	if err != nil {
		t.Fatalf("chargement du catalogue commis : %v", err)
	}
	if len(cat.Entrees) != len(table) {
		t.Fatalf("le catalogue porte %d entree(s), la table %d — une ligne a ete ajoutee d un "+
			"cote sans l autre", len(cat.Entrees), len(table))
	}
	for i, attendue := range table {
		obtenue := cat.Entrees[i]
		if obtenue != attendue {
			t.Errorf("entries[%d] diverge de la table du lot 2.1 :\n  catalogue %#v\n  table     %#v",
				i, obtenue, attendue)
		}
	}
}

// lireTableProfil analyse `profile_table.go` et rend ses lignes, dans l ordre de la table.
func lireTableProfil(t *testing.T, chemin string) []filmprofile.Entree {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, chemin, nil, 0)
	if err != nil {
		t.Fatalf("analyse de %s : %v", chemin, err)
	}
	constantes := constantesDuFichier(f)
	corps := map[string]*ast.FuncDecl{}
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil {
			corps[fn.Name.Name] = fn
		}
	}
	var out []filmprofile.Entree
	for _, tbl := range tablesDuProfil {
		fn, ok := corps[tbl.fonction]
		if !ok {
			t.Fatalf("%s : fonction %s introuvable — la table du lot 2.1 a ete restructuree, "+
				"ce test doit suivre", chemin, tbl.fonction)
		}
		lignes := lignesDeLaFonction(t, fn, constantes)
		if len(lignes) != tbl.lignes {
			t.Fatalf("%s rend %d ligne(s), %d attendue(s)", tbl.fonction, len(lignes), tbl.lignes)
		}
		out = append(out, lignes...)
	}
	return out
}

// constantesDuFichier releve les constantes de chaine declarees au niveau du fichier — les
// libelles repetes de la table (`cleToutes`, `champSlotsPerso`, les dates, les provenances).
func constantesDuFichier(f *ast.File) map[string]string {
	out := map[string]string{}
	for _, d := range f.Decls {
		gen, ok := d.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, s := range gen.Specs {
			spec, ok := s.(*ast.ValueSpec)
			if !ok || len(spec.Names) != 1 || len(spec.Values) != 1 {
				continue
			}
			if v, ok := evaluerChaine(spec.Values[0], out); ok {
				out[spec.Names[0].Name] = v
			}
		}
	}
	return out
}

// lignesDeLaFonction rend les entrees du litteral `[]LigneProfil{...}` retourne par `fn`.
func lignesDeLaFonction(t *testing.T, fn *ast.FuncDecl, constantes map[string]string) []filmprofile.Entree {
	t.Helper()
	var out []filmprofile.Entree
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		ret, ok := n.(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 1 {
			return true
		}
		lit, ok := ret.Results[0].(*ast.CompositeLit)
		if !ok {
			return true
		}
		for _, elt := range lit.Elts {
			ligne, ok := elt.(*ast.CompositeLit)
			if !ok {
				t.Fatalf("%s : element de table qui n est pas un litteral de structure", fn.Name.Name)
			}
			out = append(out, entreeDuLitteral(t, fn.Name.Name, ligne, constantes))
		}
		return false
	})
	return out
}

// entreeDuLitteral convertit UN `LigneProfil{...}` en [filmprofile.Entree].
func entreeDuLitteral(t *testing.T, fonction string, lit *ast.CompositeLit,
	constantes map[string]string) filmprofile.Entree {
	t.Helper()
	champs := map[string]string{}
	for _, e := range lit.Elts {
		kv, ok := e.(*ast.KeyValueExpr)
		if !ok {
			t.Fatalf("%s : champ de ligne sans nom — la table doit rester nommee champ par champ",
				fonction)
		}
		nom, ok := kv.Key.(*ast.Ident)
		if !ok {
			t.Fatalf("%s : clef de champ illisible", fonction)
		}
		v, ok := evaluerChaine(kv.Value, constantes)
		if !ok {
			t.Fatalf("%s : valeur du champ %s illisible (ni chaine, ni constante de fichier, "+
				"ni concatenation des deux)", fonction, nom.Name)
		}
		champs[nom.Name] = v
	}
	return filmprofile.Entree{
		Cle:    champs["Cle"],
		Champ:  champs["Champ"],
		Valeur: champs["Valeur"],
		Source: filmprofile.Provenance(champs["Source"]),
		Preuve: champs["Preuve"],
		Date:   champs["Date"],
	}
}

// evaluerChaine evalue une expression de chaine : litteral, constante du fichier, ou
// concatenation des deux. Tout le reste est REFUSE (rendu `false`) plutot que devine.
func evaluerChaine(e ast.Expr, constantes map[string]string) (string, bool) {
	switch x := e.(type) {
	case *ast.BasicLit:
		if x.Kind != token.STRING {
			return "", false
		}
		v, err := strconv.Unquote(x.Value)
		if err != nil {
			return "", false
		}
		return v, true
	case *ast.Ident:
		v, ok := constantes[x.Name]
		return v, ok
	case *ast.BinaryExpr:
		if x.Op != token.ADD {
			return "", false
		}
		g, okG := evaluerChaine(x.X, constantes)
		d, okD := evaluerChaine(x.Y, constantes)
		return g + d, okG && okD
	case *ast.ParenExpr:
		return evaluerChaine(x.X, constantes)
	}
	return "", false
}
