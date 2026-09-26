package replaybuild

// observe_test.go — LES ETAPES DE BuildBytes, dans l'ordre du source (cf. BuildBytesSteps*).
//
// Meme garde que replay/observe_test.go, sur la source de BuildBytes : les litteraux
// `b.observe("...")` doivent etre BuildBytesStepsBefore, puis l'appel a replay.BuildFromFilm,
// puis BuildBytesStepsAfter.
//
// LE GARDE DESCEND DANS LES SOUS-FONCTIONS (2026-09-02). `BuildBytes` depassait 80 lignes ; les
// lectures de catalogue en sont sorties dans `collecterEntreesCatalogue`, avec HUIT des etapes
// observees. Un garde qui ne lirait que le corps de `BuildBytes` aurait alors declare huit
// etapes disparues — ou, pire, aurait ete « corrige » en retirant ces etapes de la liste, ce qui
// aurait rendu l'equivalence aveugle sur les zones, les socles et les frags. Il INLINE donc les
// sous-fonctions declarees ici, a l'endroit ou elles sont appelees : deplacer un balayage d'une
// fonction a l'autre ne change rien, en supprimer un ou en ajouter un sans etape casse toujours.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strconv"
	"testing"
)

// fichiersDeLaSuiteObservee : les fichiers ou vit la suite d etapes de `BuildBytes`.
//
// DEUX FICHIERS DEPUIS LE LOT 4.1.2 (2026-09-17), et c est une consequence mesuree du plafond de
// taille : `replaybuild.go` est gele a 577 lignes par `archlint/film_file_size_test.go`, donc la
// bascule « relire les faits ou decoder » et la seconde moitie commune (`serialiserDocument`) ont
// du naitre dans `filmfacts_cuisson.go`. Le garde lit LES DEUX : une suite d etapes qui se
// verifierait sur un seul fichier declarerait disparues celles de l autre.
var fichiersDeLaSuiteObservee = []string{"replaybuild.go", "filmfacts_cuisson.go"}

// sousFonctionsObservantes : les fonctions dans lesquelles le garde descend quand BuildBytes les
// appelle. Une sous-fonction ABSENTE de cette liste rendrait ses etapes invisibles au garde —
// c est pour cela que la liste est ecrite, et courte.
var sousFonctionsObservantes = []string{
	"collecterEntreesCatalogue", "documentDeLaCuisson", "serialiserDocument",
}

// etapesParIdentifiant : les etapes observees par CONSTANTE et non par litteral.
//
// Le garde lit le SOURCE : `b.observe(EtapeRejeuDepuisLesFaits, …)` ne porte pas de chaine, donc
// sans cette table l etape serait INVISIBLE au garde — et une etape invisible est exactement ce
// que ce test existe pour interdire. Une entree par constante, resolue ici a la compilation :
// renommer la constante casse le test, et c est le bon sens de la faute.
var etapesParIdentifiant = map[string]string{
	"EtapeRejeuDepuisLesFaits": EtapeRejeuDepuisLesFaits,
}

// marqueursDuDocument : les appels qui PRODUISENT le document, une branche chacun. Le garde en
// emet UN SEUL marqueur `<BuildFromFilm>` — les deux branches sont au MEME point de la suite, et
// exiger deux marqueurs consecutifs ne mesurerait que l ordre du `if`.
var marqueursDuDocument = []string{"BuildFromFilmAvecFaits", "BuildFromFacts"}

// profondeurInlineMax borne la descente : deux sous-fonctions qui s'appelleraient l'une l'autre
// feraient boucler le garde au lieu de le faire echouer.
const profondeurInlineMax = 4

func TestObserveEtapesBuildBytes(t *testing.T) {
	parNom := map[string]*ast.FuncDecl{}
	for _, nom := range fichiersDeLaSuiteObservee {
		f, err := parser.ParseFile(token.NewFileSet(), nom, nil, 0)
		if err != nil {
			t.Fatalf("%s illisible : %v", nom, err)
		}
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok {
				parNom[fn.Name.Name] = fn
			}
		}
	}
	fn := parNom["BuildBytes"]
	if fn == nil {
		t.Fatal("BuildBytes introuvable")
	}
	got := etapesObservees(t, fn.Body, parNom, 0)
	want := slices.Concat(BuildBytesStepsBefore, []string{"<BuildFromFilm>"}, BuildBytesStepsAfter)
	if !slices.Equal(got, want) {
		t.Fatalf("etapes de BuildBytes dans le source\n  source : %v\n  attendu : %v", got, want)
	}
}

// etapesObservees rend, DANS L'ORDRE DU SOURCE, les etapes observees par un corps de fonction —
// en inlinant les sous-fonctions observantes a leur point d'appel.
func etapesObservees(t *testing.T, body *ast.BlockStmt, parNom map[string]*ast.FuncDecl, profondeur int) []string {
	t.Helper()
	if profondeur > profondeurInlineMax {
		t.Fatalf("descente trop profonde dans les sous-fonctions observantes (%d) : cycle d'appels ?",
			profondeur)
	}
	var out []string
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		switch nom := sel.Sel.Name; {
		case nom == "observe":
			// La garde sur le nombre d'arguments precede l'indexation (comme chez le jumeau
			// `replay/observe_test.go`) : un `x.observe(...)` d'une autre signature ferait
			// paniquer le garde au lieu de le faire echouer proprement.
			if len(call.Args) != 2 {
				return true
			}
			switch a := call.Args[0].(type) {
			case *ast.BasicLit:
				if a.Kind == token.STRING {
					s, _ := strconv.Unquote(a.Value)
					out = append(out, s)
				}
			case *ast.Ident:
				etape, connue := etapesParIdentifiant[a.Name]
				if !connue {
					t.Fatalf("etape observee par l identifiant %q, absent de "+
						"`etapesParIdentifiant` : le garde ne la verrait pas.", a.Name)
				}
				out = append(out, etape)
			}
		case slices.Contains(marqueursDuDocument, nom):
			if len(out) == 0 || out[len(out)-1] != "<BuildFromFilm>" {
				out = append(out, "<BuildFromFilm>")
			}
		case slices.Contains(sousFonctionsObservantes, nom):
			sous := parNom[nom]
			if sous == nil {
				t.Fatalf("sous-fonction observante %q appelee mais introuvable dans %v", nom,
					fichiersDeLaSuiteObservee)
			}
			out = append(out, etapesObservees(t, sous.Body, parNom, profondeur+1)...)
		}
		return true
	})
	return out
}
