package filmdec

// event_preamble_guard_test.go — LE PREAMBULE D'EVENEMENT ET LA TABLE DES DOMAINES N'ONT QU'UN
// SEUL LECTEUR DE PRODUCTION (lot E, item E.3 du PLAN_V2_REJEU_FILM, 2026-09-05).
//
// # CE QUE CE FICHIER EMPECHE, ET POURQUOI IL EXISTE
//
// La grammaire du preambule de 9 bits — `[config(1)][continuation(1)][R(7) type]` — etait ecrite
// SIX fois en ligne dans ce paquet, sous DEUX conventions (`Skip(1)` + `ReadBit()` d'un cote,
// `Skip(2)` de l'autre), et la table des largeurs de reference par domaine existait en TROIS
// exemplaires de production. Deux de ces copies portaient `3: 8` la ou la mesure du siege dit 7
// (`event_list.go`, oracle du 2026-09-02 : 5/6 d'accord a 7 bits contre 0/6 a 8 bits). Rien de
// faux n'etait servi — aucun chemin de production ne lisait le domaine 3 hors de `boardRefs`,
// qui portait deja la valeur mesuree — mais le prochain decodeur qui aurait eu besoin du
// domaine 3 avait deux chances sur trois de prendre la copie perimee et de decaler d'un bit tout
// le corps de l'evenement.
//
// CLAUDE.md regle 6 : « a la 3e copie, centraliser dans un helper ET ajouter un garde-rail (test
// grep) qui interdit l'ancien litteral — une factorisation sans garde-rail re-diverge ». C'est ce
// fichier. Il ne mesure pas un comportement : il mesure que la SOURCE reste unique.
//
// # CE QU'IL NE COUVRE PAS, ET C'EST DELIBERE
//
// LES SOURCES DE TEST SONT HORS PORTEE. Deux instruments de recherche portent encore leur propre
// table (`bpkDomWidths` dans biped_pickup_research_test.go, `r7DomWidth` dans
// r7_grammaire_research_test.go), et l'un d'eux LIT le domaine 3 (type 8, `{2,3,7}`) a 8 bits.
// Les migrer changerait une largeur DANS UN INSTRUMENT DE MESURE DATE, ce que le lot E-I
// (« comportement strictement identique ») s'interdit. Le fait est consigne au journal du lot ;
// la portee du garde-rail s'etendra quand ces deux instruments seront traites.
//
// RETRAIT : jamais tant que `filmdec` porte la grammaire de la liste d'evenements. Si le
// preambule devait un jour se lire autrement selon le type d'evenement, c'est `readPacketHead`
// qui prendrait le parametre — pas une septieme copie.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// TestRefDomWidthEstLaSeuleTable — la table des domaines rend les largeurs mesurees, dom3 = 7
// compris, et un domaine hors table rend 0 (meme semantique que les cartes qu'elle remplace :
// une cle absente y valait le zero du type, donc `ReadBits(0)`).
func TestRefDomWidthEstLaSeuleTable(t *testing.T) {
	attendu := map[int]uint{0: 13, 1: 13, 2: 8, 3: 7, 4: 9, 5: 8, 6: 9, 7: 13, 8: 13}
	for dom, want := range attendu {
		if got := refDomWidth(dom); got != want {
			t.Errorf("refDomWidth(%d) = %d, attendu %d", dom, got, want)
		}
	}
	for _, dom := range []int{-1, 9, 12, 255} {
		if got := refDomWidth(dom); got != 0 {
			t.Errorf("refDomWidth(%d) = %d : un domaine hors table doit rendre 0 (zero bit lu)", dom, got)
		}
	}
	// Le domaine 3 est LA valeur qui distingue la table canonique des deux copies supprimees.
	if refDomWidth(3) != dom3RefWidth || dom3RefWidth != 7 {
		t.Fatalf("le domaine 3 vaut %d : la mesure du siege dit 7 (event_list.go), la prose de "+
			"l'executable disait 8 — c'est la MESURE qui fait foi", refDomWidth(3))
	}
	// Les constantes nommees et la table ne peuvent pas diverger.
	if refDomWidth(7) != dom7RefWidth || refDomWidth(2) != dom2RefWidth || refDomWidth(4) != dom4RefWidth {
		t.Fatal("refDomWidth diverge des constantes nommees qu'elle compose")
	}
	// La largeur que les instruments de translocation calculent a la compilation suit la table.
	if uint(translocRefWidth) != refDomWidth(translocRefDomain) {
		t.Errorf("translocRefWidth = %d mais refDomWidth(%d) = %d",
			translocRefWidth, translocRefDomain, refDomWidth(translocRefDomain))
	}
}

// tableDomainesRecopiee reconnait une table domaine -> largeur ecrite en litteral : au moins
// quatre paires `<chiffre>: <nombre>` separees par des virgules. C'est la forme exacte des deux
// copies de production supprimees (`lot1RefDomWidths`, `zoomRefWidth`).
var tableDomainesRecopiee = regexp.MustCompile(`(?:\b\d\s*:\s*\d{1,2}\s*,\s*){3,}\d\s*:\s*\d{1,2}`)

// TestAucuneTableDeDomainesRecopiee — aucune source de PRODUCTION du paquet ne reecrit la table
// des domaines.
func TestAucuneTableDeDomainesRecopiee(t *testing.T) {
	for _, nom := range sourcesDeProductionDuPaquet(t) {
		for i, ligne := range lignesDeCode(t, nom) {
			if ligne != "" && tableDomainesRecopiee.MatchString(ligne) {
				t.Errorf("%s:%d recopie une table domaine -> largeur :\n\t%s\n"+
					"La seule table du paquet est `refDomWidth` (event_list.go). Une copie "+
					"re-diverge : les deux precedentes portaient `3: 8` contre la valeur "+
					"MESUREE 7.", nom, i+1, ligne)
			}
		}
	}
}

// --- LE PREAMBULE N'A QU'UN SEUL LECTEUR : DETECTION PAR AST -----------------------------------
//
// POURQUOI L'AST ET PLUS LE GREP (correction C3 de la revue E-R1, 2026-09-06). La premiere version
// de ce controle cherchait un `Skip(1)` ou `Skip(2)` puis un `ReadBits(7)` dans les TROIS lignes
// suivantes. Elle avait deux trous demontres par mutation :
//
//   - M1 — trois des six copies d'origine s'etalaient sur QUATRE OU CINQ lignes de code
//     (`biped_pickups.go:212-216`, `transloc_events.go:168-172`, `zoom_events.go:136-140` a la base
//     `a21fd77f4`) : elles tombaient hors de la fenetre de trois lignes. La prose qui affirmait
//     « les six copies tenaient toutes en trois lignes » etait fausse, et mesurable.
//   - M2 — la copie la PLUS PROBABLE, celle qu'on obtient en copiant-collant le corps du lecteur
//     unique (`br.ReadBit(); br.ReadBit(); br.ReadBits(7)`), n'utilise NI `Skip(1)` NI `Skip(2)` :
//     le motif ne la voyait pas du tout.
//
// La detection ci-dessous ne regarde plus des lignes : elle compte des BITS. Pour chaque lecteur,
// dans chaque SUITE DE STATEMENTS, elle ordonne les operations de bits (`Skip(n)`, `ReadBit()`,
// `ReadBits(n)`) et cherche « exactement deux bits consommes, puis une lecture de sept », sur des
// operations CONSECUTIVES. Les trois conventions du depot y tombent : `Skip(2)+R(7)`,
// `Skip(1)+ReadBit()+R(7)` et `ReadBit()+ReadBit()+R(7)`, quel que soit le nombre de lignes qui
// les separent — une lecture placee dans la CONDITION d'un `if` appartient a la suite qui porte ce
// `if`, exactement comme les trois copies d'origine l'ecrivaient.
//
// POURQUOI « MEME SUITE DE STATEMENTS » ET PAS « MEME FONCTION ». Mesure du 2026-09-06 : sans cette
// borne, le controle rend DEUX faux positifs, et ce ne sont pas des copies du preambule mais des
// grammaires de composant que le flot de controle separe —
// `consumeObjectLowFrequency` (`R(2)` de tete puis `R(7)` DANS le `if f < 2`, FUN_1407ef088) et
// `consumeByName` (un `ReadBit()` et un `ReadBits(7)` dans DEUX branches exclusives du meme
// `switch`). Un garde-rail qui exige une allowlist des le premier jour ne tient pas : la bonne
// borne est le flot, pas une liste.
//
// CE QU'IL NE VOIT PAS — LISTE OUVERTE, TENUE A JOUR, JAMAIS PRESENTEE COMME COMPLETE. La revue
// E-R2 a exhibe deux formes d'evasion que la version du 2026-09-06 ne listait pas (D-1) : la prose
// se donnait pour une enumeration alors qu'elle n'en etait pas une. Ce qui suit est l'etat CONNU
// des angles morts, et rien de plus — une forme qui n'y figure pas peut exister.
//
//  1. UNE COPIE REPARTIE ENTRE DEUX SUITES IMBRIQUEES (`br.Skip(2)` puis
//     `if x { br.ReadBits(7) }`). C'est le prix de la borne ci-dessus ; aucune des six copies
//     d'origine n'avait cette forme. NON FERME.
//  2. UNE COPIE PAR ARITHMETIQUE D'OFFSET, sans lecteur (`readBitsAt(pay, p+2, 7)`). La suivre
//     demanderait de suivre la valeur de `p`, et le controle mentirait plus souvent qu'il
//     n'attraperait. NON FERME.
//  3. UNE OPERATION DE LARGEUR NULLE INTERCALEE (`br.Skip(1); br.ReadBit(); br.Skip(0);
//     br.ReadBits(7)`) : elle ne consomme aucun bit mais rompait la consecutivite.
//     **FERME le 2026-09-06** — `opDeLAppel` ne retient plus les operations a zero bit.
//  4. LE PREAMBULE LU EN UN SEUL COUP PUIS MASQUE (`v := br.ReadBits(9)` puis `v&0x100`, `v&0x7F`).
//     L'ancre du motif est la lecture de SEPT bits ; une lecture de neuf n'en est pas une. La
//     fermer demanderait de traiter toute lecture de 9 bits comme suspecte, alors que la largeur
//     9 n'a rien de propre au preambule — le controle deviendrait bruyant sur des grammaires de
//     composant. NON FERME, et c'est un arbitrage, pas un oubli.
//
// Le jour ou l'une des formes non fermees apparait, c'est ici qu'elle se traite.

// lecteurUniqueDuPreambule est la SEULE fonction autorisee a lire le preambule d'evenement.
const lecteurUniqueDuPreambule = "readPacketHead"

// opBit est UNE operation de lecture de bits sur un lecteur, telle que l'AST la donne.
type opBit struct {
	bits   int // bits consommes ; -1 quand la largeur n'est pas un litteral connu
	offset int // position dans la source, pour ordonner
	ligne  int
	texte  string
}

// suiteDOps est la suite des operations d'UN lecteur dans UNE suite de statements.
type suiteDOps struct {
	lecteur string
	ops     []opBit
}

// bitsConsommes rend le nombre de bits qu'un appel consomme sur son lecteur, et s'il en est un.
func bitsConsommes(sel *ast.SelectorExpr, args []ast.Expr) (int, bool) {
	switch sel.Sel.Name {
	case "ReadBit":
		if len(args) == 0 {
			return 1, true
		}
	case "Skip", "ReadBits":
		if len(args) == 1 {
			return largeurLitterale(args[0]), true
		}
	}
	return 0, false
}

// largeurLitterale rend la largeur d'un argument quand elle est connue a la lecture (entier
// litteral ou la constante nommee du type d'evenement), -1 sinon.
func largeurLitterale(a ast.Expr) int {
	switch v := a.(type) {
	case *ast.BasicLit:
		if v.Kind != token.INT {
			return -1
		}
		n, err := strconv.Atoi(v.Value)
		if err != nil {
			return -1
		}
		return n
	case *ast.Ident:
		if v.Name == "eventTypeBits" {
			return eventTypeBits
		}
	}
	return -1
}

// nomDuReceveur rend le receveur d'un appel sous forme textuelle (`br`, `s.br`), ou "" si
// l'expression n'est pas une chaine d'identifiants — auquel cas deux appels ne sont pas
// comparables et on ne les rapproche pas.
func nomDuReceveur(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		if prefixe := nomDuReceveur(v.X); prefixe != "" {
			return prefixe + "." + v.Sel.Name
		}
	}
	return ""
}

// porteUneSuiteDeStatements dit si un noeud ouvre une suite de statements : un bloc, ou le corps
// d'une clause de `switch` / `select`, qui n'est pas un bloc mais en tient lieu.
func porteUneSuiteDeStatements(n ast.Node) bool {
	switch n.(type) {
	case *ast.BlockStmt, *ast.CaseClause, *ast.CommClause:
		return true
	}
	return false
}

// suitesDuCorps rend les suites d'operations de bits d'un corps, une par couple
// (suite de statements, lecteur), ordonnees par leur premiere operation.
func suitesDuCorps(fset *token.FileSet, src []byte, body *ast.BlockStmt) []suiteDOps {
	type cle struct {
		suite   ast.Node
		lecteur string
	}
	groupes := map[cle][]opBit{}
	var pile []ast.Node
	ast.Inspect(body, func(n ast.Node) bool {
		if n == nil {
			pile = pile[:len(pile)-1]
			return false
		}
		if op, lecteur, ok := opDeLAppel(fset, src, n); ok {
			k := cle{suite: suiteEnglobante(pile), lecteur: lecteur}
			groupes[k] = append(groupes[k], op)
		}
		pile = append(pile, n)
		return true
	})
	out := make([]suiteDOps, 0, len(groupes))
	for k, ops := range groupes {
		sort.Slice(ops, func(i, j int) bool { return ops[i].offset < ops[j].offset })
		out = append(out, suiteDOps{lecteur: k.lecteur, ops: ops})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ops[0].offset != out[j].ops[0].offset {
			return out[i].ops[0].offset < out[j].ops[0].offset
		}
		return out[i].lecteur < out[j].lecteur
	})
	return out
}

// suiteEnglobante rend la suite de statements la plus proche dans la pile d'ancetres.
func suiteEnglobante(pile []ast.Node) ast.Node {
	for i := len(pile) - 1; i >= 0; i-- {
		if porteUneSuiteDeStatements(pile[i]) {
			return pile[i]
		}
	}
	return nil
}

// opDeLAppel reconnait une operation de bits et rend son lecteur.
func opDeLAppel(fset *token.FileSet, src []byte, n ast.Node) (opBit, string, bool) {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return opBit{}, "", false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return opBit{}, "", false
	}
	bits, estLecture := bitsConsommes(sel, call.Args)
	lecteur := nomDuReceveur(sel.X)
	if !estLecture || lecteur == "" {
		return opBit{}, "", false
	}
	// UNE OPERATION DE LARGEUR NULLE NE ROMPT PAS LA SUITE, parce qu'elle ne consomme rien : un
	// `br.Skip(0)` glisse entre deux lectures laissait echapper la copie en cassant la
	// consecutivite (D-1 de la revue E-R2, 2026-09-06). On ne la retient pas du tout.
	if bits == 0 {
		return opBit{}, "", false
	}
	pos := fset.Position(call.Pos())
	return opBit{
		bits: bits, offset: pos.Offset, ligne: pos.Line, texte: texteSource(src, fset, call),
	}, lecteur, true
}

// texteSource rend le texte source d'un noeud, pour que le message d'echec montre la ligne fautive.
func texteSource(src []byte, fset *token.FileSet, n ast.Node) string {
	d, f := fset.Position(n.Pos()).Offset, fset.Position(n.End()).Offset
	if d < 0 || f > len(src) || d >= f {
		return ""
	}
	return strings.Join(strings.Fields(string(src[d:f])), " ")
}

// preambuleRecopie cherche, dans la suite d'operations d'UN lecteur, « deux bits consommes puis
// une lecture de sept » sur des operations CONSECUTIVES. Rend l'indice ou la sequence commence et
// celui de la lecture de sept bits.
func preambuleRecopie(ops []opBit) (debut, fin int, trouve bool) {
	for i, o := range ops {
		if o.bits != eventTypeBits {
			continue
		}
		switch {
		case i >= 1 && ops[i-1].bits == 2:
			return i - 1, i, true
		case i >= 2 && ops[i-1].bits == 1 && ops[i-2].bits == 1:
			return i - 2, i, true
		}
	}
	return 0, 0, false
}

// TestPreambuleNaQuUnSeulLecteur — la SEQUENCE du preambule (deux bits de tete puis R(7) de type)
// ne s'ecrit que dans `readPacketHead`. Les six copies en ligne d'avant le 2026-09-05 sont
// interdites de retour, et les trois conventions du depot le sont avec elles.
//
// L'EXEMPTION EST LA FONCTION, PAS LE FICHIER. La version d'avant exemptait `event_list.go` en
// entier ; ici seule la declaration de `readPacketHead` sort du controle, donc une septieme copie
// ecrite dans le fichier du lecteur unique serait vue.
func TestPreambuleNaQuUnSeulLecteur(t *testing.T) {
	fset := token.NewFileSet()
	for _, nom := range sourcesDeProductionDuPaquet(t) {
		src, err := os.ReadFile(nom)
		if err != nil {
			t.Fatalf("lecture de %s : %v", nom, err)
		}
		f, err := parser.ParseFile(fset, nom, src, 0)
		if err != nil {
			t.Fatalf("analyse de %s : %v", nom, err)
		}
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil || fd.Name.Name == lecteurUniqueDuPreambule {
				continue
			}
			for _, s := range suitesDuCorps(fset, src, fd.Body) {
				debut, fin, trouve := preambuleRecopie(s.ops)
				if !trouve {
					continue
				}
				t.Errorf("%s:%d-%d (%s) recopie le preambule d'evenement sur `%s` :\n\t%s\n\t%s\n"+
					"Le preambule de 9 bits se lit par `%s` (event_list.go). Six copies en ligne, "+
					"sous deux conventions, ont ete ramenees a ce lecteur unique le 2026-09-05 "+
					"(lot E, item E.3).",
					nom, s.ops[debut].ligne, s.ops[fin].ligne, fd.Name.Name, s.lecteur,
					s.ops[debut].texte, s.ops[fin].texte, lecteurUniqueDuPreambule)
			}
		}
	}
}

// sourcesDeProductionDuPaquet rend les sources Go non-test du paquet, relatives au repertoire
// courant (celui du paquet quand `go test` s'execute).
func sourcesDeProductionDuPaquet(t *testing.T) []string {
	t.Helper()
	noms, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("listage du paquet : %v", err)
	}
	var out []string
	for _, nom := range noms {
		if !strings.HasSuffix(nom, "_test.go") {
			out = append(out, nom)
		}
	}
	if len(out) == 0 {
		t.Fatal("aucune source de production trouvee : le garde-rail ne mesure plus rien")
	}
	return out
}

// lignesDeCode rend les lignes du fichier, celles de commentaire et les vides rendues comme la
// chaine vide : la prose a le droit de citer une table ou un preambule.
func lignesDeCode(t *testing.T, nom string) []string {
	t.Helper()
	src, err := os.ReadFile(nom)
	if err != nil {
		t.Fatalf("lecture de %s : %v", nom, err)
	}
	brutes := strings.Split(string(src), "\n")
	out := make([]string, len(brutes))
	for i, ligne := range brutes {
		nue := strings.TrimSpace(ligne)
		if nue == "" || strings.HasPrefix(nue, "//") {
			continue
		}
		out[i] = nue
	}
	return out
}
