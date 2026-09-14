package replay

// observe_test.go — LA LISTE FERMEE DES ETAPES OBSERVEES, GARDEE SUR LA SOURCE.
//
// Le harnais d'equivalence hache la sortie de chaque etape de l'etage de balayage : une etape qui
// manque est un balayage que le refacto peut casser sans que rien ne le dise. La mini-bobine
// ne permet pas d'EXECUTER BuildFromFilm (aucune image-cle de bipede : ScanFilmBipedPositions
// refuse), et le corpus reel n'est pas en CI. Le garde porte donc sur la SOURCE, comme les
// ratchets d'archlint : il lit `scanFilmInputs` et les phases qu'elle appelle, et exige que
//
//  1. les litteraux `observe("...")`, dans l'ordre du source, soient EXACTEMENT
//     BuildFromFilmSteps ;
//  2. chaque appel de balayage (`Scan*`, `decodeFilm*`) ait son etape : autant d'appels de
//     balayage que d'etapes hors `.stats`. Un balayage ajoute sans `observe` casse ce compte.
//
// LE PREFIXE EST `Scan` DEPUIS LE LOT 1 (2026-09-02), ET C'EST UNE EXTENSION DELIBEREE DU GARDE.
// Les balayages prenaient un REPERTOIRE et s'appelaient `ScanFilmXxx(dir)` ; ils prennent
// desormais un `*filmsource.Film` deja charge et s'appellent `ScanXxx(film)` — les formes `dir`
// survivent en enveloppes hors production (D2), et AUCUNE n'est appelee ici. Garder `ScanFilm`
// aurait rendu le compte a 5 sur 22 : le garde aurait cesse de garder en silence.
//
// # IL DESCEND DANS LES PHASES DEPUIS LE LOT 1.0 (2026-09-14)
//
// La sequence ne tient plus dans UNE fonction : `BuildFromFilm` appelle `scanFilmInputs`, qui
// appelle CINQ phases (`film_scan.go`) ; QUATRE d'entre elles appellent une sous-phase, et il y a
// CINQ sous-phases en tout (`balayerPont` n'en appelle aucune). Un garde qui ne
// lirait que le corps de `scanFilmInputs` ne verrait plus AUCUNE etape — il rougirait le jour du
// decoupage puis ne garderait plus rien. Il descend donc, DANS L'ORDRE DES APPELS, dans les
// fonctions declarees par les fichiers de l'etage ([fichiersDuBalayage]), et s'arrete aux
// balayages eux-memes (`Scan*`, `decodeFilm*`), qui sont les feuilles : ce sont eux qu'on compte.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// fichiersDuBalayage : les fichiers ou vit l'etage de balayage. Le garde ne descend QUE dans les
// fonctions qu'ils declarent — descendre dans tout le paquet ramasserait des `observe` et des
// `Scan*` d'autres chemins (recherche, calques) et le compte cesserait de vouloir dire quelque
// chose.
var fichiersDuBalayage = []string{"build_from_film.go", "film_scan.go"}

// racineDuBalayage : la fonction par laquelle l'etage commence.
const racineDuBalayage = "scanFilmInputs"

// declarationsDuBalayage rend les fonctions et methodes declarees par les fichiers de l'etage,
// indexees par leur nom SIMPLE (sans recepteur) : c'est la forme qui se recoupe avec les noms lus
// dans les corps, ou `s.balayerMonde()` ne donne que `balayerMonde`.
func declarationsDuBalayage(t *testing.T) map[string]*ast.FuncDecl {
	t.Helper()
	out := map[string]*ast.FuncDecl{}
	for _, nom := range fichiersDuBalayage {
		f, err := parser.ParseFile(token.NewFileSet(), nom, nil, 0)
		if err != nil {
			t.Fatalf("%s illisible : %v", nom, err)
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			if _, deja := out[fn.Name.Name]; deja {
				t.Fatalf("deux declarations nommees %q dans %v : le garde ne saurait pas dans "+
					"laquelle descendre", fn.Name.Name, fichiersDuBalayage)
			}
			out[fn.Name.Name] = fn
		}
	}
	if _, ok := out[racineDuBalayage]; !ok {
		t.Fatalf("%s introuvable dans %v : ce garde-rail ne garde plus rien",
			racineDuBalayage, fichiersDuBalayage)
	}
	return out
}

// estBalayage dit si un nom de fonction est un BALAYAGE de film : la forme film (`ScanXxx`), la
// forme repertoire heritee (`ScanFilmXxx`, enveloppe D2 — aucune n'est appelee ici) ou un
// decodeur de calque (`decodeFilmXxx`).
func estBalayage(nom string) bool {
	return strings.HasPrefix(nom, "Scan") || strings.HasPrefix(nom, "decodeFilm")
}

// marcheurDEtapes accumule les etapes observees et les balayages rencontres, dans l'ordre du
// source, en descendant dans les fonctions de l'etage.
//
// IL DESCEND DANS CHAQUE APPEL, PAS DANS CHAQUE NOM (revue R1, constat R1-2). La premiere
// version memorisait les phases DEJA marchees et sautait les suivantes : une phase appelee DEUX
// fois n'etait comptee qu'une, donc la mutation « `s.balayerCalquesGardes()` appele deux fois »
// — trois etapes de plus emises en production — laissait les deux tests VERTS. `pile` ne sert
// plus qu'a refuser un CYCLE (une phase qui se rappellerait elle-meme, directement ou non) :
// sans lui la marche ne terminerait pas, et un cycle est de toute facon un defaut a nommer.
type marcheurDEtapes struct {
	decls map[string]*ast.FuncDecl
	// pile porte les phases EN COURS de marche, pas celles deja marchees.
	pile  map[string]bool
	cycle string
	steps []string
	scans int
}

// nomAppele rend le nom SIMPLE de la fonction appelee (`f()`, `p.f()`, `a.b.f()`) et sa presence.
func nomAppele(call *ast.CallExpr) (string, bool) {
	switch f := call.Fun.(type) {
	case *ast.Ident:
		return f.Name, true
	case *ast.SelectorExpr:
		return f.Sel.Name, true
	}
	return "", false
}

// descendre parcourt un corps DANS L'ORDRE DU SOURCE : il note les etapes observees, compte les
// balayages, et descend dans les fonctions de l'etage.
func (m *marcheurDEtapes) descendre(body *ast.BlockStmt) {
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		nom, ok := nomAppele(call)
		if !ok {
			return true
		}
		switch {
		case nom == "observe" && len(call.Args) == 2:
			if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				s, _ := strconv.Unquote(lit.Value)
				m.steps = append(m.steps, s)
			}
		case estBalayage(nom):
			m.scans++
			return false // un balayage est une feuille : on ne descend pas dedans
		default:
			fn, connue := m.decls[nom]
			if !connue {
				return true
			}
			if m.pile[nom] {
				m.cycle = nom
				return false
			}
			m.pile[nom] = true
			m.descendre(fn.Body)
			delete(m.pile, nom)
		}
		return true
	})
}

// marcher lance la marche depuis `racine` et rend le marcheur, cycle compris.
func marcher(decls map[string]*ast.FuncDecl, racine string) *marcheurDEtapes {
	m := &marcheurDEtapes{decls: decls, pile: map[string]bool{racine: true}}
	m.descendre(decls[racine].Body)
	return m
}

// etapesDuBalayage rend les etapes observees et le nombre de balayages, dans l'ordre du source.
func etapesDuBalayage(t *testing.T) (steps []string, scans int) {
	t.Helper()
	m := marcher(declarationsDuBalayage(t), racineDuBalayage)
	if m.cycle != "" {
		t.Fatalf("cycle d'appel dans l'etage de balayage sur %q : une phase qui se rappelle "+
			"elle-meme emettrait ses etapes sans fin", m.cycle)
	}
	return m.steps, m.scans
}

func TestObserveEtapesBuildFromFilm(t *testing.T) {
	steps, scans := etapesDuBalayage(t)
	if !slices.Equal(steps, BuildFromFilmSteps) {
		t.Fatalf("les etapes observees par l'etage de balayage ne sont pas BuildFromFilmSteps\n  source  : %v\n  liste   : %v", steps, BuildFromFilmSteps)
	}
	horsStats := 0
	for _, s := range BuildFromFilmSteps {
		if !strings.HasSuffix(s, ".stats") {
			horsStats++
		}
	}
	if scans != horsStats {
		t.Fatalf("%d appel(s) de balayage dans l'etage pour %d etape(s) observee(s) hors .stats — un balayage sans observe ?", scans, horsStats)
	}
}

// TestBuildFromFilmNeBalaiePlusLuiMeme : `BuildFromFilm` est le CHEMIN (verrou, largeurs,
// assemblage), pas la sequence. Un balayage ou une etape qui y reapparaitrait serait le debut
// d'une seconde sequence, celle-la meme que le lot 1.0 a supprimee.
func TestBuildFromFilmNeBalaiePlusLuiMeme(t *testing.T) {
	decls := declarationsDuBalayage(t)
	fn, ok := decls["BuildFromFilm"]
	if !ok {
		t.Fatalf("BuildFromFilm introuvable dans %v", fichiersDuBalayage)
	}
	m := &marcheurDEtapes{decls: map[string]*ast.FuncDecl{}, pile: map[string]bool{}}
	m.descendre(fn.Body)
	if len(m.steps) > 0 || m.scans > 0 {
		t.Fatalf("BuildFromFilm porte %d etape(s) observee(s) et %d balayage(s) : la sequence doit "+
			"vivre dans %s, et la seule", len(m.steps), m.scans, racineDuBalayage)
	}
}

// TestObserveNilNeCouteRien : sans observateur, observe est un no-op — ni panique, ni appel.
func TestObserveNilNeCouteRien(t *testing.T) {
	var o Options
	o.observe("positions", nil)
	if slices.Contains(BuildFromFilmSteps, "") {
		t.Fatal("BuildFromFilmSteps porte un nom vide")
	}
}

// ---------------------------------------------------------------------------
// LE MARCHEUR LUI-MEME, EPROUVE SUR UNE SOURCE FABRIQUEE
// ---------------------------------------------------------------------------

// declarationsDeSource rend les declarations d'une source Go fabriquee, indexees par nom simple.
// Elle existe pour eprouver [marcheurDEtapes] SANS muter le code de production : une garde de
// source qui n'est jamais mise en echec ne prouve rien.
func declarationsDeSource(t *testing.T, src string) map[string]*ast.FuncDecl {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "fabrique.go", src, 0)
	if err != nil {
		t.Fatalf("source fabriquee illisible : %v", err)
	}
	out := map[string]*ast.FuncDecl{}
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Body != nil {
			out[fn.Name.Name] = fn
		}
	}
	return out
}

// TestMarcheurDescendDansChaqueAppel : LE CONTRE-TEST DU CONSTAT R1-2.
//
// Une phase appelee DEUX fois emet ses etapes DEUX fois en production. Le marcheur doit les
// compter deux fois — la version qui memorisait les noms deja marches n'en voyait qu'une, et la
// mutation « appeler une phase une seconde fois » passait alors les deux gardes au vert.
func TestMarcheurDescendDansChaqueAppel(t *testing.T) {
	const src = `package p
func racine() { phase(); phase() }
func phase() { opt.observe("a", nil); ScanTruc() }
`
	m := marcher(declarationsDeSource(t, src), "racine")
	if m.cycle != "" {
		t.Fatalf("cycle signale a tort : %q", m.cycle)
	}
	if want := []string{"a", "a"}; !slices.Equal(m.steps, want) {
		t.Fatalf("etapes = %v, attendu %v — le marcheur ne descend pas dans CHAQUE appel", m.steps, want)
	}
	if m.scans != 2 {
		t.Fatalf("%d balayage(s) comptes, attendu 2 — un balayage d'une phase rappelee est perdu", m.scans)
	}
}

// TestMarcheurRefuseUnCycle : descendre dans chaque appel ne doit pas boucler. Une phase qui se
// rappelle (directement ou par une autre) est NOMMEE, pas marchee sans fin.
func TestMarcheurRefuseUnCycle(t *testing.T) {
	const src = `package p
func racine() { a() }
func a() { opt.observe("x", nil); b() }
func b() { a() }
`
	m := marcher(declarationsDeSource(t, src), "racine")
	if m.cycle != "a" {
		t.Fatalf("cycle detecte = %q, attendu \"a\" — la marche boucle ou ne nomme pas la phase", m.cycle)
	}
}
