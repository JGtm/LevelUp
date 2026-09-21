//go:build research

package grammar

// mouvement_5_3_2d_couverture_research_test.go — LA MESURE QUI A REFUTE L AMORCAGE DU MONDE.
//
// # LA QUESTION, ET POURQUOI ELLE DECIDE
//
// Les rejets de generation peuvent avoir DEUX causes, et elles appellent deux corrections
// opposees : soit le monde ne connait pas le slot (il faut LIER PLUS), soit il le connait mais
// sa generation a avance (il faut RELACHER le test). Une mesure qui ne les separe pas corrige
// au hasard.
//
// LE DISCRIMINANT : un slot rejete qui apparait AUSSI dans un record sain est connu du monde —
// c est donc sa generation qui a bouge. Un slot rejete qui n apparait JAMAIS sain n est pas
// lie. La proportion tranche.
//
// # CE QU ELLE A RENDU SUR `bfecd02b`, ET CE QUE CELA A COUTE A L HYPOTHESE
//
// 4 568 slots distincts rejetes, dont 28 seulement (0,6 %) vus sains. Les plus rejetes — 137,
// 1041, 2616, 3933, 5268 — sont tous inconnus. Or 4 568 slots etales de 137 a 5 268, c est
// PLUS D ENTITES QU UN MATCH N EN PORTE : ce sont des identifiants lus dans du bruit, pas des
// liaisons manquantes. L amorcage du monde n est donc pas la cause, et `BindWildcard` l a
// confirme en ne deplacant aucun chiffre.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// m532dCouvertureMonde separe les deux causes possibles d un rejet de generation.
func m532dCouvertureMonde(t *testing.T, echecs []m532dEchec, sains []m532Ech) {
	t.Helper()
	rejetes := map[uint32]int{}
	for _, e := range echecs {
		if e.ti == 0 && e.fautif == 0 && e.composants == 0 {
			rejetes[e.slot]++
		}
	}
	if len(rejetes) == 0 {
		t.Logf("    COUVERTURE DU MONDE : aucun rejet de generation")
		return
	}
	vus := map[uint32]bool{}
	for _, e := range sains {
		vus[e.slot] = true
	}
	communs := 0
	for s := range rejetes {
		if vus[s] {
			communs++
		}
	}
	t.Logf("    COUVERTURE DU MONDE : %d slots DISTINCTS rejetes, dont %d vus AUSSI dans un "+
		"record sain (%.1f %%). Un slot rejete jamais vu sain n est pas LIE ; un slot vu dans "+
		"les deux est une GENERATION qui a avance.",
		len(rejetes), communs, m532Pct(communs, len(rejetes)))
	type paire struct {
		slot uint32
		n    int
	}
	top := make([]paire, 0, len(rejetes))
	for sl, n := range rejetes {
		top = append(top, paire{sl, n})
	}
	sort.Slice(top, func(a, b int) bool { return top[a].n > top[b].n })
	var parts []string
	for i, x := range top {
		if i >= 8 {
			break
		}
		etat := "inconnu"
		if vus[x.slot] {
			etat = "vu-sain"
		}
		parts = append(parts, fmt.Sprintf("%d:%d(%s)", x.slot, x.n, etat))
	}
	t.Logf("    slots les plus rejetes : %s", strings.Join(parts, " "))
}
