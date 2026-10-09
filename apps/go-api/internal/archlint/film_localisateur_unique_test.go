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
//	essai de position   le bit qui precede une position `s` est teste (`BitAt(_, s-1)`, ou
//	                    `precedeDuTerminateur(_, s)`) et un delta est essaye a `s`
//	                    (`TryDeltaAt(_, s, …)`) dans le meme fichier : le pas du localisateur ;
//	slot de signature   un champ `Slot` compare au litteral 123 : la signature ecrite en dur.
//
// Le fichier hote doit porter l essai de position exactement deux fois (signature stricte et repli
// a largeur libre), sans quoi le garde-rail garde un fantome.
//
// LE BIT NUL DE TETE A UNE SEULE IMPLANTATION (lot VA, etape V2 ; regle des deux copies) : dans le
// paquet `grammar`, le test du bit qui precede une position (`BitAt(_, x-1)`) ne s ecrit que dans
// `precedeDuTerminateur` (`localisateur.go`), que les deux etages du localisateur appellent ; une
// troisieme copie du test ne peut pas revenir ailleurs.
//
// PERIMETRE : la production de `film/**`, hors `film/research/` et hors fichiers `//go:build
// research` (instruments de mesure, qui recopient le localisateur pour en essayer des variantes),
// tests exclus ([balayerLaProductionHorsResearch]).

import (
	"go/ast"
	"go/parser"
	"go/token"
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
	// plancherFichiersLocalisateur : plancher contre un balayage muet.
	plancherFichiersLocalisateur = 500
	// slotDeSignature : le slot de la signature du premier record.
	slotDeSignature = "123"
	// testDuTerminateur : la seule fonction de `grammar` qui teste le bit qui precede une position.
	testDuTerminateur = "precedeDuTerminateur"
	// paquetGrammar : le repertoire du paquet `grammar`, relatif a apps/go-api.
	paquetGrammar = "internal/games/halo_infinite/film/internal/grammar/"
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

// positionPrecedee : `BitAt(_, s-1)` ou `precedeDuTerminateur(_, s)`, le bit qui precede la
// position `s`.
func positionPrecedee(call *ast.CallExpr) (string, bool) {
	if nomAppele(call.Fun) == testDuTerminateur && len(call.Args) >= 2 {
		if id, ok := call.Args[1].(*ast.Ident); ok {
			return id.Name, true
		}
		return "", false
	}
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

// testsBrutsDuBitPrecedent compte, dans un source Go, les tests du bit qui precede une position
// (`BitAt(_, x-1)`, quelle que soit l expression `x`) ecrits hors de [testDuTerminateur].
func testsBrutsDuBitPrecedent(t *testing.T, nom string, src []byte) int {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), nom, src, 0)
	if err != nil {
		t.Fatalf("analyse de %s : %v", nom, err)
	}
	bruts := 0
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == testDuTerminateur {
			continue
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok && testeLeBitPrecedent(call) {
				bruts++
			}
			return true
		})
	}
	return bruts
}

// testeLeBitPrecedent : `BitAt(_, x-1)`.
func testeLeBitPrecedent(call *ast.CallExpr) bool {
	if nomAppele(call.Fun) != "BitAt" || len(call.Args) < 2 {
		return false
	}
	b, ok := call.Args[1].(*ast.BinaryExpr)
	return ok && b.Op == token.SUB && estLitteral(b.Y, "1")
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
	hote := -1
	fichiers := balayerLaProductionHorsResearch(t, []string{racineLocalisateur}, func(rel string, blob []byte) {
		if strings.HasPrefix(rel, paquetGrammar) {
			if n := testsBrutsDuBitPrecedent(t, rel, blob); n > 0 {
				t.Errorf("%s : %d test(s) du bit qui precede une position (`BitAt(_, x-1)`) hors de "+
					"%s — le bit nul de tete de la vue B se teste par grammar.%s (%s)", rel, n,
					testDuTerminateur, testDuTerminateur, hoteLocalisateur)
			}
		}
		essais, slots := formesDuSource(t, rel, blob)
		if rel == hoteLocalisateur {
			hote = essais
			return
		}
		if essais > 0 || slots > 0 {
			t.Errorf("%s : %d essai(s) de position et %d comparaison(s) de Slot au litteral %s — "+
				"copie du localisateur : appeler grammar.LocaliserBoucleDeRecords avec l ordre du "+
				"site (%s)", rel, essais, slots, slotDeSignature, hoteLocalisateur)
		}
	})
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
// un test de bit precedent sans essai de delta (`grammar/kills_rattrapes.go`) et un essai sur une
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

// TestGardeRailBitDeTeteVecteurs : le test du bit qui precede une position ne s ecrit que dans
// [testDuTerminateur] ; un appel a celui-ci, suivi d un essai de delta a la meme position, est un
// essai de position.
func TestGardeRailBitDeTeteVecteurs(t *testing.T) {
	for _, c := range []struct {
		nom, src      string
		bruts, essais int
	}{
		{"test brut dans une autre fonction", "package p\n\nfunc f() bool { return source.BitAt(pay, p-extra-1) == 0 }\n", 1, 0},
		{"implantation unique", "package p\n\nfunc " + testDuTerminateur +
			"(pay []byte, tete int) bool { return source.BitAt(pay, tete-1) == 0 }\n", 0, 0},
		{"essai par l implantation unique", "package p\n\nfunc f() {\n\tif " + testDuTerminateur +
			"(pay, s) {\n\t\t_, _, _ = TryDeltaAt(pay, s, w, cfg)\n\t}\n}\n", 0, 1},
	} {
		bruts := testsBrutsDuBitPrecedent(t, "vecteur.go", []byte(c.src))
		essais, _ := formesDuSource(t, "vecteur.go", []byte(c.src))
		if bruts != c.bruts || essais != c.essais {
			t.Errorf("%s : %d test(s) brut(s), %d essai(s), attendu %d et %d", c.nom, bruts, essais,
				c.bruts, c.essais)
		}
	}
}
