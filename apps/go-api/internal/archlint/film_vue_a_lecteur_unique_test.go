package archlint

// film_vue_a_lecteur_unique_test.go — LE GARDE-RAIL DE LA LECTURE UNIQUE DE LA VUE A (lot VA de la
// campagne de grammaire ; regle des deux copies, CLAUDE.md n. 6).
//
// La vue A d une trame delta (`FUN_14076a1c4`) se lit en UN seul endroit :
// `grammar/vue_a_lecture.go` ([grammar.lireLaVueA]), que trois fonctions appellent — `rangerLaTete`,
// qui la lit une fois par trame dans la marche des trames et la range, `lireTrameParRangs`, qui la
// lit depuis la tete du paquet quand on ne la lui passe pas, et `DebutDeLaVueB`
// (`grammar/localisateur.go`), qui la lit pour les deux marches qui lisent les morts et ne rangent
// pas de structure (lot VA, etape V2). Ce test interdit, dans l arbre syntaxique de la production
// du decodeur :
//
//	un appel a `lireLaVueA` hors de ces trois fonctions ;
//	un appel a `chargeDuGenre` (le lecteur de la charge d un message) hors de `lireUnMessage`, le
//	corps de message de la lecture unique ;
//	toute fonction nommee `consumeVueA`, la lecture de la tete seule que le lot a remplacee
//	(ratchet anti-resurrection).
//
// PERIMETRE : la production de `film/**`, hors `film/research/` et hors fichiers `//go:build
// research`, tests exclus.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const (
	// plancherFichiersVueA : plancher contre un balayage muet.
	plancherFichiersVueA = 500
	// lectureRetiree : la lecture de la tete seule, remplacee par la lecture unique.
	lectureRetiree = "consumeVueA"
)

// appelantsPermis : pour chaque fonction de la lecture unique, les seules fonctions qui l appellent.
func appelantsPermis() map[string][]string {
	return map[string][]string{
		"lireLaVueA":    {"rangerLaTete", "lireTrameParRangs", "DebutDeLaVueB"},
		"chargeDuGenre": {"lireUnMessage"},
	}
}

// ecartsDeLaVueA rend, pour un source Go, les appels aux fonctions de la lecture unique faits hors de
// leurs appelants permis, et les declarations de la lecture retiree.
func ecartsDeLaVueA(t *testing.T, nom string, src []byte) []string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), nom, src, 0)
	if err != nil {
		t.Fatalf("analyse de %s : %v", nom, err)
	}
	permis := appelantsPermis()
	var ecarts []string
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Name.Name == lectureRetiree {
			ecarts = append(ecarts, "declaration de "+lectureRetiree)
		}
		ast.Inspect(fn, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			appele := nomAppele(call.Fun)
			if appelants, suivi := permis[appele]; suivi && !contientNom(appelants, fn.Name.Name) {
				ecarts = append(ecarts, fn.Name.Name+" appelle "+appele)
			}
			return true
		})
	}
	return ecarts
}

func contientNom(noms []string, nom string) bool {
	for _, n := range noms {
		if n == nom {
			return true
		}
	}
	return false
}

// TestLectureDeLaVueAUnique interdit les lectures de la vue A hors de la lecture unique.
func TestLectureDeLaVueAUnique(t *testing.T) {
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(ici)))
	base := filepath.Join(goAPIRoot, filepath.FromSlash(racineLocalisateur))
	fichiers := 0
	appels := map[string]int{}
	err := filepath.WalkDir(base, func(chemin string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(goAPIRoot, chemin)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if chemin != base && repertoireExcluDuTriTotal(d.Name(), rel) {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(chemin, ".go") || strings.HasSuffix(chemin, "_test.go") {
			return nil
		}
		blob, err := os.ReadFile(chemin) //nolint:gosec // chemin derive du perimetre
		if err != nil || estSousTagResearch(blob) {
			return err
		}
		fichiers++
		for _, e := range ecartsDeLaVueA(t, rel, blob) {
			t.Errorf("%s : %s — la vue A se lit par grammar.lireLaVueA (vue_a_lecture.go), depuis "+
				"rangerLaTete ou lireTrameParRangs seulement", rel, e)
		}
		for appele := range appelantsPermis() {
			appels[appele] += strings.Count(string(blob), appele+"(")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("balayage de %s : %v", base, err)
	}
	if fichiers < plancherFichiersVueA {
		t.Fatalf("balayage muet : %d fichiers de production vus, plancher %d", fichiers, plancherFichiersVueA)
	}
	for appele, n := range appels {
		if n == 0 {
			t.Errorf("%s n est plus appelee : deplacer le garde-rail avec la lecture unique", appele)
		}
	}
}

// lectureDeTeteRetiree : la lecture que `grammar/frame_vue_messages.go` portait jusqu au lot VA.
const lectureDeTeteRetiree = `package grammar

func consumeVueA(br *Lecteur, frameLen int) FluxVueA {
	out := FluxVueA{}
	if !br.ReadBit() {
		out.Vide, out.Porte = true, true
		return out
	}
	out.Genres = append(out.Genres, int(br.ReadBits(LargeurGenreVueA)))
	return out
}

func rangerLaTete(p *lecture.Paquet) {
	a := consumeVueA(br, len(p.Payload)*8)
}
`

// TestGardeRailVueAVecteurs : la lecture retiree rougit (declaration et appel), une seconde lecture
// de la vue A ou d une charge hors des appelants permis aussi ; les appelants permis passent.
func TestGardeRailVueAVecteurs(t *testing.T) {
	for _, c := range []struct {
		nom, src string
		ecarts   int
	}{
		{"lecture de la tete retiree", lectureDeTeteRetiree, 1},
		{"seconde lecture", "package p\n\nfunc localiser() { _ = lireLaVueA(pay, 1, bal) }\n", 1},
		{"charge hors du corps de message", "package p\n\nfunc f() { _ = chargeDuGenre(15) }\n", 1},
		{"appelants permis", "package p\n\nfunc rangerLaTete() { _ = lireLaVueA(p, 1, b) }\n\n" +
			"func lireUnMessage() { _ = chargeDuGenre(g) }\n", 0},
	} {
		if e := ecartsDeLaVueA(t, "vecteur.go", []byte(c.src)); len(e) != c.ecarts {
			t.Errorf("%s : ecarts %v, attendu %d", c.nom, e, c.ecarts)
		}
	}
}
