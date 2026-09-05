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

// sousFonctionsObservantes : les fonctions de replaybuild.go dans lesquelles le garde descend
// quand BuildBytes les appelle. Une sous-fonction ABSENTE de cette liste rendrait ses etapes
// invisibles au garde — c'est pour cela que la liste est ecrite, et courte.
var sousFonctionsObservantes = []string{"collecterEntreesCatalogue"}

// profondeurInlineMax borne la descente : deux sous-fonctions qui s'appelleraient l'une l'autre
// feraient boucler le garde au lieu de le faire echouer.
const profondeurInlineMax = 4

func TestObserveEtapesBuildBytes(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "replaybuild.go", nil, 0)
	if err != nil {
		t.Fatalf("replaybuild.go illisible : %v", err)
	}
	parNom := map[string]*ast.FuncDecl{}
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			parNom[fn.Name.Name] = fn
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
			if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				s, _ := strconv.Unquote(lit.Value)
				out = append(out, s)
			}
		case nom == "BuildFromFilm":
			out = append(out, "<BuildFromFilm>")
		case slices.Contains(sousFonctionsObservantes, nom):
			sous := parNom[nom]
			if sous == nil {
				t.Fatalf("sous-fonction observante %q appelee mais introuvable dans replaybuild.go", nom)
			}
			out = append(out, etapesObservees(t, sous.Body, parNom, profondeur+1)...)
		}
		return true
	})
	return out
}
