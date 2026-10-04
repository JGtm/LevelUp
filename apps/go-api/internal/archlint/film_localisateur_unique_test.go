package archlint

// film_localisateur_unique_test.go — LE GARDE-RAIL DU LOCALISATEUR UNIQUE DE LA BOUCLE DE RECORDS
// (regle des deux copies, CLAUDE.md n. 6).
//
// Le debut de la boucle de records d un paquet a evenements se trouve en UN seul endroit :
// `grammar/localisateur.go` ([grammar.LocaliserBoucleDeRecords]), que la cuisson, la marche des
// morts d objet et la marche de killsource appellent chacune avec son ordre. Ce test interdit
// qu une copie revienne ailleurs dans la production du decodeur, sous deux formes lues dans l arbre
// syntaxique :
//
//	essai de position   le bit qui precede une position `s` est teste (`BitAt(_, s-1)`) et un delta
//	                    est essaye a `s` (`TryDeltaAt(_, s, …)`) dans le meme fichier : le pas du
//	                    localisateur ;
//	slot de signature   un champ `Slot` compare au litteral 123 : la signature ecrite en dur.
//
// Le fichier hote doit porter l essai de position exactement deux fois (signature stricte et repli
// a largeur libre), sans quoi le garde-rail garde un fantome.
//
// PERIMETRE : la production de `film/**`, hors `film/research/` et hors fichiers `//go:build
// research` (instruments de mesure, qui recopient le localisateur pour en essayer des variantes),
// tests exclus.

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
	// racineLocalisateur : le perimetre, relatif a apps/go-api.
	racineLocalisateur = "internal/games/halo_infinite/film"
	// hoteLocalisateur : le seul fichier qui porte le localisateur.
	hoteLocalisateur = "internal/games/halo_infinite/film/internal/grammar/localisateur.go"
	// essaisDeLHote : les deux etages du localisateur (signature stricte, largeur libre).
	essaisDeLHote = 2
	// plancherFichiersLocalisateur : contre un balayage muet (590 fichiers de production le
	// 2026-10-04).
	plancherFichiersLocalisateur = 500
	// slotDeSignature : le slot de la signature du premier record.
	slotDeSignature = "123"
)

// formesDuSource compte, dans un source Go, les essais de position (chaque `BitAt(_, s-1)` dont
// la position `s` est essayee par un `TryDeltaAt(_, s, …)` du meme fichier : le localisateur
// separe d ordinaire le test du bit et l essai en deux fonctions) et les comparaisons d un champ
// `Slot` au slot de signature.
func formesDuSource(t *testing.T, nom string, src []byte) (essais, slots int) {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), nom, src, 0)
	if err != nil {
		t.Fatalf("analyse de %s : %v", nom, err)
	}
	essaies := map[string]bool{}
	var precedes []string
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if pos, ok := positionEssayee(x); ok {
				essaies[pos] = true
			}
			if pos, ok := positionPrecedee(x); ok {
				precedes = append(precedes, pos)
			}
		case *ast.BinaryExpr:
			if comparaisonAuSlotDeSignature(x) {
				slots++
			}
		}
		return true
	})
	for _, pos := range precedes {
		if essaies[pos] {
			essais++
		}
	}
	return essais, slots
}

// positionEssayee : `TryDeltaAt(_, s, …)`, quel que soit le paquet qui qualifie l appel.
func positionEssayee(call *ast.CallExpr) (string, bool) {
	if nomAppele(call.Fun) != "TryDeltaAt" || len(call.Args) < 2 {
		return "", false
	}
	id, ok := call.Args[1].(*ast.Ident)
	if !ok {
		return "", false
	}
	return id.Name, true
}

// positionPrecedee : `BitAt(_, s-1)`, le bit qui precede la position `s`.
func positionPrecedee(call *ast.CallExpr) (string, bool) {
	if nomAppele(call.Fun) != "BitAt" || len(call.Args) < 2 {
		return "", false
	}
	b, ok := call.Args[1].(*ast.BinaryExpr)
	if !ok || b.Op != token.SUB || !estLitteral(b.Y, "1") {
		return "", false
	}
	id, ok := b.X.(*ast.Ident)
	if !ok {
		return "", false
	}
	return id.Name, true
}

// comparaisonAuSlotDeSignature : `x.Slot == 123`, `x.Slot != 123`, ou l inverse.
func comparaisonAuSlotDeSignature(b *ast.BinaryExpr) bool {
	if b.Op != token.EQL && b.Op != token.NEQ {
		return false
	}
	return (estChampSlot(b.X) && estLitteral(b.Y, slotDeSignature)) ||
		(estChampSlot(b.Y) && estLitteral(b.X, slotDeSignature))
}

func estChampSlot(e ast.Expr) bool {
	sel, ok := e.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "Slot"
}

func estLitteral(e ast.Expr, valeur string) bool {
	lit, ok := e.(*ast.BasicLit)
	return ok && lit.Kind == token.INT && lit.Value == valeur
}

// TestLocalisateurDeBoucleUnique interdit les copies du localisateur hors du fichier hote.
func TestLocalisateurDeBoucleUnique(t *testing.T) {
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(ici)))
	base := filepath.Join(goAPIRoot, filepath.FromSlash(racineLocalisateur))
	fichiers, hote := 0, -1
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
		essais, slots := formesDuSource(t, rel, blob)
		if rel == hoteLocalisateur {
			hote = essais
			return nil
		}
		if essais > 0 || slots > 0 {
			t.Errorf("%s : %d essai(s) de position et %d comparaison(s) de Slot au litteral %s — "+
				"copie du localisateur : appeler grammar.LocaliserBoucleDeRecords avec l ordre du "+
				"site (%s)", rel, essais, slots, slotDeSignature, hoteLocalisateur)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("balayage de %s : %v", base, err)
	}
	if fichiers < plancherFichiersLocalisateur {
		t.Fatalf("balayage muet : %d fichiers de production vus, plancher %d", fichiers,
			plancherFichiersLocalisateur)
	}
	if hote != essaisDeLHote {
		t.Errorf("%s porte %d essai(s) de position, %d attendus — deplacer le garde-rail avec le "+
			"localisateur", hoteLocalisateur, hote, essaisDeLHote)
	}
}

// copieRetireeDeKillsource : la copie que `killsource/walk.go` portait jusqu au lot LU (signature
// et test du bit precedent dans deux fonctions, repli a largeur libre).
const copieRetireeDeKillsource = `package killsource

func signature123(pl []byte, s int, w *grammar.World, cfg grammar.FrameConfig) bool {
	rec, end, ok := grammar.TryDeltaAt(pl, s, w, cfg)
	return ok && rec.Slot == 123 && end == s+35 && len(rec.Trace.Comps) == 1
}

func locateStrict(pl []byte, w *grammar.World, cfg grammar.FrameConfig) int {
	for s := 2; s+35 < len(pl)*8; s++ {
		if source.BitAt(pl, s-1) != 0 {
			continue
		}
		if signature123(pl, s, w, cfg) {
			return s
		}
	}
	return -1
}

func locateFallback(pl []byte, w *grammar.World, cfg grammar.FrameConfig) int {
	for s := 2; s+16 < len(pl)*8; s++ {
		if source.BitAt(pl, s-1) != 0 {
			continue
		}
		rec, _, ok := grammar.TryDeltaAt(pl, s, w, cfg)
		if !ok || rec.Slot != 123 {
			continue
		}
		return s
	}
	return -1
}
`

// TestGardeRailLocalisateurVecteurs : la copie retiree de killsource rougit sous ses deux formes ;
// un test de bit precedent sans essai de delta (`killsource/eventchain.go`) et un essai sur une
// autre position ne sont pas le localisateur.
func TestGardeRailLocalisateurVecteurs(t *testing.T) {
	for _, c := range []struct {
		nom, src      string
		essais, slots int
	}{
		{"copie retiree de killsource", copieRetireeDeKillsource, 2, 2},
		{"slot a droite", "package p\n\nfunc f() bool { return 123 != rec.Slot }\n", 0, 1},
		{"bit precedent seul", "package p\n\nfunc f() bool {\n\treturn source.BitAt(pl, x-1) == 1 && source.BitAt(pl, x) == 0\n}\n", 0, 0},
		{"autre position, autre slot", "package p\n\nfunc f() {\n\t_ = source.BitAt(pl, p-1)\n\t" +
			"_, _, _ = TryDeltaAt(pl, q, w, cfg)\n\t_ = rec.Slot == 124\n}\n", 0, 0},
	} {
		essais, slots := formesDuSource(t, "vecteur.go", []byte(c.src))
		if essais != c.essais || slots != c.slots {
			t.Errorf("%s : %d essai(s), %d slot(s), attendu %d et %d", c.nom, essais, slots,
				c.essais, c.slots)
		}
	}
}
