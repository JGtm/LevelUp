package killsource

// chaines_evenements_test.go — L ORACLE DE LA CHAINE D EVENEMENTS, FIGE AVANT L ABSORPTION
// (lot 2.4.1, item 2.4.1 du PLAN_DECODEUR_FILM_2026-09-13).
//
// # CE QUE CE GOLDEN EST, ET QUAND IL A ETE PRODUIT
//
// `testdata/chaines_evenements.golden` a ete produit par le code de la BASE `88f1a1115`, AVANT
// que `evReader` ne soit absorbe par le lecteur canonique de la couche source — par une COPIE
// JETABLE du paquet entier (un `git show` de ses 23 fichiers de production sous un autre nom de
// paquet), supprimee aussitot le fichier ecrit. Ce n est donc PAS une reference que le lot a
// figee sur son propre resultat : c est la sortie de l ANCIEN lecteur, et ce test exige que le
// nouveau rende la meme, a l octet.
//
// # CE QU IL FIGE
//
// Pour chaque bobine versionnee (les huit `replay/testdata/minifilm_*` et les deux
// `testdata/minibobine_*`), pour chaque paquet type-0 A EVENTS, dans l ordre :
//
//	chaine=   les triplets (code, bit de debut, bit de fin) de la liste d evenements qui
//	          commence au bit 2, et la maniere dont elle se termine (FIN sur un bit de
//	          continuation a 0, STOP sur un corps non modelise ou un debordement).
//	ancres=   pour chaque position candidate du generateur de kill-events — c est la marche qui
//	          visite le PLUS de positions reelles du paquet : TOUS les bits sont essayes — la
//	          fin des champs obligatoires, les six champs lus, la longueur de chaine mesuree,
//	          et les triplets de cette chaine.
//
// Ce sont les positions REELLES de la chaine, pas un balayage aveugle : 1 528 ancres et
// 1 121 paquets sur les dix bobines.
//
// # IL N A PAS DE PORTE `-update`, ET C EST VOULU
//
// Un golden qui fige un AVANT ne se regenere pas : le regenerer reviendrait a declarer que la
// consommation de bits a le droit de bouger. S il rougit, c est que la grammaire a change — et
// c est alors `GrammarRev` et `facts.Rev` qu il faut rouvrir, pas ce fichier.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/source"
)

// cheminGoldenChaines : l oracle, relatif au paquet.
const cheminGoldenChaines = "testdata/chaines_evenements.golden"

// bobinesVersionnees : les dix bobines du depot. Aucune n exige de fixture ni de cache — ce test
// tourne en CI.
func bobinesVersionnees(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, motif := range []string{"../../replay/testdata/minifilm_*", "testdata/minibobine_*"} {
		m, err := filepath.Glob(motif)
		if err != nil {
			t.Fatalf("glob %s : %v", motif, err)
		}
		out = append(out, m...)
	}
	sort.Strings(out)
	if len(out) < 10 {
		t.Fatalf("%d bobine(s) trouvee(s), 10 attendues — les fixtures ont bouge, ce test ne "+
			"garderait plus rien", len(out))
	}
	return out
}

// TestChainesDEvenementsIdentiquesAuGolden : la consommation de bits de la chaine d evenements
// est IDENTIQUE a celle d avant l absorption.
func TestChainesDEvenementsIdentiquesAuGolden(t *testing.T) {
	var lignes []string
	for _, d := range bobinesVersionnees(t) {
		lignes = append(lignes, lignesDeBobine(t, d)...)
	}
	sort.Strings(lignes)
	obtenu := strings.Join(lignes, "\n") + "\n"
	blob, err := os.ReadFile(cheminGoldenChaines)
	if err != nil {
		t.Fatalf("golden %s illisible : %v — il est VERSIONNE, son absence est une erreur",
			cheminGoldenChaines, err)
	}
	attendu := strings.ReplaceAll(string(blob), "\r\n", "\n")
	if obtenu == attendu {
		return
	}
	t.Errorf("LA CHAINE D EVENEMENTS NE CONSOMME PLUS LES MEMES BITS QU AVANT L ABSORPTION.\n%s\n"+
		"Ce golden fige l ANCIEN lecteur (base 88f1a1115) : il n a pas de porte -update. "+
		"Rouvrir GrammarRev et facts.Rev avant toute autre chose.",
		premiereDifference(attendu, obtenu))
}

// premiereDifference rend la premiere ligne qui differe, avec son rang.
func premiereDifference(attendu, obtenu string) string {
	a, b := strings.Split(attendu, "\n"), strings.Split(obtenu, "\n")
	for i := 0; i < len(a) || i < len(b); i++ {
		var la, lb string
		if i < len(a) {
			la = a[i]
		}
		if i < len(b) {
			lb = b[i]
		}
		if la != lb {
			return fmt.Sprintf("  ligne %d\n  attendu : %s\n  obtenu  : %s\n  (%d lignes "+
				"attendues, %d obtenues)", i+1, la, lb, len(a), len(b))
		}
	}
	return "  (contenu identique ligne a ligne : difference de fin de fichier)"
}

// lignesDeBobine : une ligne par paquet type-0 a events.
func lignesDeBobine(t *testing.T, dir string) []string {
	t.Helper()
	src, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("%s : %v", dir, err)
	}
	nom := filepath.Base(dir)
	f, err := loadFilm(src)
	if err != nil {
		return []string{fmt.Sprintf("%s\tSANS-PAQUET-TYPE-0\t%v", nom, err)}
	}
	g15 := pickGate15(f)
	var out []string
	for i := range f.t0 {
		p := &f.t0[i]
		if !hasEvents(p) {
			continue
		}
		out = append(out, fmt.Sprintf("%s\t%d\t%d\tg15=%v\tchaine=%s\tancres=%s",
			nom, p.chunk, p.idx, g15, chaineDepuis(p.payload, 2, g15), ancresDuPaquet(p.payload, g15)))
	}
	return out
}

// chaineDepuis : les triplets (code, bit de debut, bit de fin) de la chaine qui commence au bit
// `depart`, plus la facon dont elle se termine.
func chaineDepuis(pl []byte, depart int, g15 bool) string {
	r := nouveauCurseurEv(pl, depart)
	var b strings.Builder
	for n := 0; n < 4096; n++ {
		debut := r.pos()
		code := int(source.BitsAt(pl, debut+1, 7))
		fin, ok := evStep(r, g15)
		if fin {
			fmt.Fprintf(&b, "|FIN@%d", debut)
			return b.String()
		}
		if !ok {
			fmt.Fprintf(&b, "|STOP@%d,over=%v", debut, r.over)
			return b.String()
		}
		fmt.Fprintf(&b, "|%d:%d-%d", code, debut, r.pos())
	}
	return b.String() + "|BORNE"
}

// ancresDuPaquet : pour chaque position candidate du generateur de kill-events, les six champs
// lus, la longueur de chaine et les triplets de cette chaine. MEME parcours que [killEventsIn],
// deroule ici pour publier ce qu il lit.
func ancresDuPaquet(pl []byte, g15 bool) string {
	var b strings.Builder
	for _, x := range positionsCandidates(pl) {
		r := nouveauCurseurEv(pl, x+7)
		if !evPresence(r, killEventCode) {
			continue
		}
		k := readKillEvent(pl, r.pos())
		if !killEventPlausible(k) {
			continue
		}
		fmt.Fprintf(&b, "|%d>%d,%d,%d,%d,%d,%d,%d:%d%s", x, k.end, k.victim, k.killer,
			k.assist, k.killerPct, k.assistPct, k.flag,
			evChainLen(pl, k.end, g15, maxChainProbe), chaineDepuis(pl, k.end, g15))
	}
	return b.String()
}

// positionsCandidates : les positions que le generateur de kill-events retient (bit de
// continuation a 1 suivi du code 85). MEME condition que [killEventsIn].
func positionsCandidates(pl []byte) []int {
	var out []int
	nb := len(pl) * 8
	for x := 1; x+8 <= nb; x++ {
		if !estAncreDeKillEvent(pl, x) {
			continue
		}
		out = append(out, x)
	}
	return out
}
