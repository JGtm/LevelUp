package filmprofile_test

// conformite_empreintes_test.go — LA TABLE DES EMPREINTES DU PROFIL EST LA SECTION
// `registryFingerprints` DU CATALOGUE, LIGNE POUR LIGNE (lot 3.1.1-b).
//
// # CE QUE CE TEST GARDE
//
// Le decodeur NE PEUT PAS lire le catalogue : ce serait un import d un paquet de catalogue
// depuis un calque, que le sens unique interdit (ADR 0034, D-1). La table du profil
// (`film/internal/profile/registre_empreintes.go`) le RECOPIE donc, et deux copies d une meme
// verite divergent des qu on cesse de les comparer. Ce test les tient egales : meme ordre,
// memes clefs, memes empreintes, memes blocs, memes slots, memes statuts, memes provenances,
// memes temoins, memes preuves, memes dates.
//
// Il est le PENDANT de [TestCatalogueConformeALaTableDuLot21], qui fait le meme travail sur la
// section `entries`. Un fichier a part parce que la section est a part — et elle l est pour une
// raison : une empreinte, un nombre de blocs et un nombre de slots sont des GRANDEURS, qui se
// comparent valeur contre valeur, la ou `entries.value` est une phrase ecrite pour un humain.
//
// # POURQUOI IL LIT LE FICHIER GO AU LIEU D IMPORTER `profile`
//
// Memes trois raisons que [TestCatalogueConformeALaTableDuLot21] (cf. son en-tete), et une
// quatrieme, decisive : depuis le lot 2.5.e la couche `profile` vit sous `film/internal/`, donc
// ce paquet NE PEUT PAS l importer, meme s il le voulait. La preuve porte sur ce qui est dans
// l arbre, pas sur ce qui compile.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/filmprofile"
	"levelup/go-api/internal/testutil"
)

// cheminTableDesEmpreintes : le fichier de la table des empreintes, relatif a la racine du depot.
var cheminTableDesEmpreintes = filepath.Join("apps", "go-api", "internal", "games",
	"halo_infinite", "film", "internal", "profile", "registre_empreintes.go")

// fonctionDeLaTableDesEmpreintes : la fonction qui CONCATENE les sous-tables.
const fonctionDeLaTableDesEmpreintes = "EmpreintesRegistre"

// sousTablesDesEmpreintes : les sous-tables, DANS L ORDRE ou [fonctionDeLaTableDesEmpreintes]
// les concatene, avec leur nombre de lignes mesure le 2026-09-17 — les sept builds du cache et
// les deux versions majeures des films sans section d identification.
//
// LA LISTE EST CONFRONTEE AU CORPS DE LA FONCTION par [verifierTablesConcatenees], la meme
// derivation que pour `TableProfil` : une sous-table ajoutee et non declaree ici laisserait ce
// test VERT sur une table qui a grossi.
//
// LE PLANCHER FERME L AUTRE DEFAUT CLASSIQUE : une analyse syntaxique qui rendrait ZERO ligne
// serait VERTE sans avoir rien compare.
var sousTablesDesEmpreintes = []struct {
	fonction string
	lignes   int
}{
	{"empreintesDesBuildsRecents", 4},
	{"empreintesDesBuildsAnciens", 3},
	{"empreintesSansSectionDIdentification", 2},
}

// lignesAttenduesDesEmpreintes : le total des trois, calcule une fois.
func lignesAttenduesDesEmpreintes() int {
	n := 0
	for _, t := range sousTablesDesEmpreintes {
		n += t.lignes
	}
	return n
}

// TestCatalogueConformeALaTableDesEmpreintes — LA GARDE.
//
// Mutation qui doit le faire rougir : changer une empreinte, un compte de blocs, un statut, un
// temoin ou une date d un cote sans toucher l autre.
func TestCatalogueConformeALaTableDesEmpreintes(t *testing.T) {
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	table := lireTableDesEmpreintes(t, filepath.Join(racine, cheminTableDesEmpreintes))
	if len(table) != lignesAttenduesDesEmpreintes() {
		t.Fatalf("%d ligne(s) lue(s) dans %s, %d attendue(s) — l analyse de la table s est "+
			"cassee, ou la table a bouge sans que le catalogue suive",
			len(table), cheminTableDesEmpreintes, lignesAttenduesDesEmpreintes())
	}

	chemin := title.NewPathResolver(racine).FilmProfilesPath(title.DefaultSlug)
	cat, err := filmprofile.Charger(chemin)
	if err != nil {
		t.Fatalf("chargement du catalogue commis : %v", err)
	}
	if len(cat.EmpreintesRegistre) != len(table) {
		t.Fatalf("le catalogue porte %d empreinte(s), la table du profil %d — une clef a ete "+
			"ajoutee d un cote sans l autre", len(cat.EmpreintesRegistre), len(table))
	}
	for i, attendue := range table {
		comparerEmpreinte(t, i, cat.EmpreintesRegistre[i], attendue)
	}
}

// comparerEmpreinte confronte UNE ligne, champ par champ, en nommant celui qui diverge : un
// `!=` sur la structure entiere laisserait chercher lequel des neuf champs a bouge.
func comparerEmpreinte(t *testing.T, i int, cat, table filmprofile.EmpreinteRegistre) {
	t.Helper()
	nom := func(champ string) string {
		return "registryFingerprints[" + strconv.Itoa(i) + "] (" + cat.Cle + ") " + champ
	}
	if cat.Cle != table.Cle {
		t.Errorf("%s : catalogue %q, table %q", nom("key"), cat.Cle, table.Cle)
		return
	}
	if cat.Empreinte != table.Empreinte {
		t.Errorf("%s : catalogue %q, table %q", nom("fingerprint"), cat.Empreinte, table.Empreinte)
	}
	if cat.Blocs != table.Blocs || cat.SlotsNommes != table.SlotsNommes {
		t.Errorf("%s : catalogue %d/%d, table %d/%d", nom("blocks/namedSlots"),
			cat.Blocs, cat.SlotsNommes, table.Blocs, table.SlotsNommes)
	}
	if cat.Statut != table.Statut {
		t.Errorf("%s : catalogue %q, table %q", nom("status"), cat.Statut, table.Statut)
	}
	if cat.Source != table.Source {
		t.Errorf("%s : catalogue %q, table %q", nom("provenance"), cat.Source, table.Source)
	}
	if len(cat.Temoins) != len(table.Temoins) {
		t.Errorf("%s : catalogue %v, table %v", nom("witnesses"), cat.Temoins, table.Temoins)
	} else {
		for j := range cat.Temoins {
			if cat.Temoins[j] != table.Temoins[j] {
				t.Errorf("%s[%d] : catalogue %q, table %q", nom("witnesses"), j,
					cat.Temoins[j], table.Temoins[j])
			}
		}
	}
	if cat.Preuve != table.Preuve {
		t.Errorf("%s diverge :\n  catalogue %q\n  table     %q", nom("proof"), cat.Preuve, table.Preuve)
	}
	if cat.Date != table.Date {
		t.Errorf("%s : catalogue %q, table %q", nom("date"), cat.Date, table.Date)
	}
}

// lireTableDesEmpreintes analyse `registre_empreintes.go` et rend ses lignes, dans l ordre.
//
// Les valeurs se lisent sous la forme du CATALOGUE — l empreinte en `0x%016x`, le statut et la
// provenance en chaines — pour que la comparaison soit directe. C est la conversion que le
// decodeur fait dans l autre sens (`fmt.Sprintf("0x%016x", ...)` dans `coverage_decoder.go`).
func lireTableDesEmpreintes(t *testing.T, chemin string) []filmprofile.EmpreinteRegistre {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, chemin, nil, 0)
	if err != nil {
		t.Fatalf("analyse de %s : %v", chemin, err)
	}
	constantes := constantesDuFichier(f)
	corps := map[string]*ast.FuncDecl{}
	for _, d := range f.Decls {
		if decl, ok := d.(*ast.FuncDecl); ok && decl.Recv == nil {
			corps[decl.Name.Name] = decl
		}
	}
	verifierSousTablesDesEmpreintes(t, chemin, corps)
	var out []filmprofile.EmpreinteRegistre
	for _, sous := range sousTablesDesEmpreintes {
		fn, ok := corps[sous.fonction]
		if !ok {
			t.Fatalf("%s : fonction %s introuvable — la table a ete restructuree, ce test doit "+
				"suivre", chemin, sous.fonction)
		}
		lits := litterauxDeStructure(t, fn)
		if len(lits) != sous.lignes {
			t.Fatalf("%s rend %d ligne(s), %d attendue(s)", sous.fonction, len(lits), sous.lignes)
		}
		for _, lit := range lits {
			out = append(out, empreinteDuLitteral(t, lit, constantes))
		}
	}
	return out
}

// verifierSousTablesDesEmpreintes : LA LISTE DES SOUS-TABLES DERIVE DU CODE, ELLE N EST PAS
// RECOPIEE. Meme derivation, meme raison et meme mecanique que `verifierTablesConcatenees` pour
// `TableProfil` (cf. conformite_table21_test.go) : sans elle, une quatrieme sous-table ajoutee a
// `EmpreintesRegistre()` laisserait ce test VERT sur des lignes que le catalogue ne porte pas.
func verifierSousTablesDesEmpreintes(t *testing.T, chemin string, corps map[string]*ast.FuncDecl) {
	t.Helper()
	fn, ok := corps[fonctionDeLaTableDesEmpreintes]
	if !ok {
		t.Fatalf("%s : fonction %s introuvable", chemin, fonctionDeLaTableDesEmpreintes)
	}
	concatenees := fonctionsConcatenees(fn)
	declarees := make([]string, 0, len(sousTablesDesEmpreintes))
	for _, sous := range sousTablesDesEmpreintes {
		declarees = append(declarees, sous.fonction)
	}
	if !slices.Equal(concatenees, declarees) {
		t.Fatalf("%s : %s concatene %v, sousTablesDesEmpreintes declare %v — l ordre fait celui "+
			"du catalogue, et une sous-table non declaree ne serait pas comparee",
			chemin, fonctionDeLaTableDesEmpreintes, concatenees, declarees)
	}
}

// litterauxDeStructure rend les litteraux de structure du `return []T{...}` de `fn`.
func litterauxDeStructure(t *testing.T, fn *ast.FuncDecl) []*ast.CompositeLit {
	t.Helper()
	var out []*ast.CompositeLit
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
			out = append(out, ligne)
		}
		return false
	})
	return out
}

// empreinteDuLitteral convertit UN `EmpreinteRegistre{...}` en la forme du catalogue.
func empreinteDuLitteral(t *testing.T, lit *ast.CompositeLit,
	constantes map[string]string) filmprofile.EmpreinteRegistre {
	t.Helper()
	out := filmprofile.EmpreinteRegistre{}
	for _, e := range lit.Elts {
		kv, ok := e.(*ast.KeyValueExpr)
		if !ok {
			t.Fatal("champ de ligne sans nom — la table doit rester nommee champ par champ")
		}
		nom, ok := kv.Key.(*ast.Ident)
		if !ok {
			t.Fatal("clef de champ illisible")
		}
		poserChampDEmpreinte(t, &out, nom.Name, kv.Value, constantes)
	}
	return out
}

// poserChampDEmpreinte pose UN champ lu dans la structure du catalogue. Un champ inconnu est un
// ECHEC et non un silence : la table gagnerait une colonne que ce test ne comparerait pas.
func poserChampDEmpreinte(t *testing.T, out *filmprofile.EmpreinteRegistre, nom string,
	val ast.Expr, constantes map[string]string) {
	t.Helper()
	switch nom {
	case "Cle":
		out.Cle = chaineDuChamp(t, nom, val, constantes)
	case "Empreinte":
		out.Empreinte = empreinteEnFormeDuCatalogue(t, val)
	case "Blocs":
		out.Blocs = entierDuChamp(t, nom, val)
	case "SlotsNommes":
		out.SlotsNommes = entierDuChamp(t, nom, val)
	case "Statut":
		out.Statut = filmprofile.StatutEmpreinte(identifiantEnValeur(t, nom, val, statutsDeLaTable))
	case "Source":
		out.Source = filmprofile.Provenance(identifiantEnValeur(t, nom, val, provenancesDeLaTable))
	case "Temoins":
		out.Temoins = temoinsDuChamp(t, val, constantes)
	case "Preuve":
		out.Preuve = chaineDuChamp(t, nom, val, constantes)
	case "Date":
		out.Date = chaineDuChamp(t, nom, val, constantes)
	default:
		t.Fatalf("champ %q inconnu de la table des empreintes — ce test doit le comparer", nom)
	}
}

// statutsDeLaTable / provenancesDeLaTable : les identifiants Go de la couche `profile` et la
// valeur de catalogue qu ils portent. Ils sont ECRITS ici parce que le test ne peut pas importer
// la couche (elle est sous `film/internal/`) ; un identifiant hors de ces tables fait ECHOUER le
// test plutot que de passer pour une chaine vide.
var statutsDeLaTable = map[string]string{
	"StatutCatalogueConnue":   "connue",
	"StatutCataloguePresumee": "presumee",
}

var provenancesDeLaTable = map[string]string{
	"ProvenanceRelue":    "relue",
	"ProvenanceMesuree":  "mesuree",
	"ProvenancePresumee": "presumee",
}

// identifiantEnValeur traduit un identifiant de constante en sa valeur de catalogue.
func identifiantEnValeur(t *testing.T, nom string, val ast.Expr, table map[string]string) string {
	t.Helper()
	id, ok := val.(*ast.Ident)
	if !ok {
		t.Fatalf("champ %s : une constante nommee est attendue", nom)
	}
	v, connu := table[id.Name]
	if !connu {
		t.Fatalf("champ %s : constante %q hors de la table de ce test — l ajouter ici AVANT de "+
			"l employer dans la table du profil", nom, id.Name)
	}
	return v
}

// empreinteEnFormeDuCatalogue relit un litteral `0x...` et le rend sous la forme du catalogue
// (`0x` + 16 chiffres minuscules) — c est ainsi que les deux se comparent sans conversion.
func empreinteEnFormeDuCatalogue(t *testing.T, val ast.Expr) string {
	t.Helper()
	lit, ok := val.(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		t.Fatal("champ Empreinte : un litteral entier hexadecimal est attendu")
	}
	n, err := strconv.ParseUint(lit.Value, 0, 64)
	if err != nil {
		t.Fatalf("champ Empreinte : %q illisible (%v)", lit.Value, err)
	}
	return "0x" + padHex(strconv.FormatUint(n, 16))
}

// padHex complete a SEIZE chiffres, zeros de tete compris — la forme que le catalogue valide.
func padHex(s string) string {
	for len(s) < 16 {
		s = "0" + s
	}
	return s
}

// entierDuChamp relit un litteral entier decimal.
func entierDuChamp(t *testing.T, nom string, val ast.Expr) int {
	t.Helper()
	lit, ok := val.(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		t.Fatalf("champ %s : un litteral entier est attendu", nom)
	}
	n, err := strconv.Atoi(lit.Value)
	if err != nil {
		t.Fatalf("champ %s : %q illisible (%v)", nom, lit.Value, err)
	}
	return n
}

// chaineDuChamp evalue une expression de chaine par le MEME evaluateur que la conformite des
// `entries` (litteral, constante de fichier, concatenation).
func chaineDuChamp(t *testing.T, nom string, val ast.Expr, constantes map[string]string) string {
	t.Helper()
	v, ok := evaluerChaine(val, constantes)
	if !ok {
		t.Fatalf("champ %s : valeur illisible (ni chaine, ni constante de fichier, ni "+
			"concatenation des deux)", nom)
	}
	return v
}

// temoinsDuChamp relit un `[]string{...}` de temoins.
func temoinsDuChamp(t *testing.T, val ast.Expr, constantes map[string]string) []string {
	t.Helper()
	lit, ok := val.(*ast.CompositeLit)
	if !ok {
		t.Fatal("champ Temoins : un litteral de tranche est attendu")
	}
	out := make([]string, 0, len(lit.Elts))
	for _, e := range lit.Elts {
		v, ok := evaluerChaine(e, constantes)
		if !ok {
			t.Fatal("champ Temoins : temoin illisible")
		}
		out = append(out, v)
	}
	return out
}
