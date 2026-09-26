//go:build research

package grammar

// tcg_tir_continu_compteur_research_test.go — S1 TER de la sonde P1 : LE COMPTEUR DE TIR DU
// RECORD 36, POUR TOUS LES TIREURS.
//
// Le champ de 8 bits que `FUN_141fcf670` lit en tete de charge (R(7) puis R(1)) avance d un pas a
// chaque tir d un meme joueur, a travers ses morts et ses corps : c est un NUMERO DE TIR par joueur.
// Sa lecture : n mod 256 = (v >> 1) | ((v & 1) << 7) — les 7 bits de poids faible en tete, le
// bit 7 en queue (mesure : ...c252 c254 c1 c3... quand n franchit 128). On le deplie par
// monotonie. Un SAUT de n entre deux records successifs d un joueur compte des tirs que le jeu a
// numerotes et dont le film ne porte aucun record 36 lisible — et la ou il tombe dit ou ils sont
// passes. Temoin : les petits sauts (1 a 3) sont les records hors tete que la marche ne valide pas.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// tcgNumero rend n mod 256 depuis la valeur brute du champ.
func tcgNumero(v uint64) int { return int((v >> 1) | ((v & 1) << 7)) } //nolint:gosec // 8 bits

// tcgSautMin est le plus petit saut publie : au-dela de 3 tirs manques.
const tcgSautMin = 4

// tcgS1Compteurs publie, par tireur, les sauts du numero de tir et les episodes qu ils recouvrent.
func tcgS1Compteurs(t *testing.T, p *tcgPasse) {
	t.Helper()
	par := map[int][]tcgEv{}
	for _, e := range p.evs {
		// Cadrage CERTAIN seulement : la tete de liste, ou une liste validee par l oracle.
		if e.tir != nil && e.tir.idx5 >= 0 && (e.pos == 1 || e.valide) {
			par[e.tir.idx5] = append(par[e.tir.idx5], e)
		}
	}
	t.Logf("== S1.7 NUMERO DE TIR PAR TIREUR (record 36) : premier numero, sauts >= %d (tirs numerotes "+
		"sans record lisible) et episodes recouverts", tcgSautMin)
	idx := make([]int, 0, len(par))
	for i := range par {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	for _, ix := range idx {
		es := par[ix]
		sort.SliceStable(es, func(i, j int) bool { return es[i].ts < es[j].ts })
		n, prev := tcgNumero(es[0].tir.attaquant), tcgNumero(es[0].tir.attaquant)
		var sauts []string
		total := 0
		for k := 1; k < len(es); k++ {
			v := tcgNumero(es[k].tir.attaquant)
			d := (v - prev + 256) % 256
			prev = v
			n += d
			if d >= tcgSautMin {
				total += d - 1
				sauts = append(sauts, fmt.Sprintf("t%d-%d:+%d%s", es[k-1].trame, es[k].trame, d-1,
					tcgRecouvre(p.cad, es[k-1].trame, es[k].trame)))
			}
		}
		t.Logf("   index %2d : %d records, premier n=%d a t%d, dernier n=%d a t%d, %d tirs dans les sauts : %s",
			ix, len(es), tcgNumero(es[0].tir.attaquant), es[0].trame, n, es[len(es)-1].trame, total,
			strings.Join(sauts, " "))
	}
}

// tcgRecouvre rend les episodes (index du pilote) que l intervalle [a, b] recouvre.
func tcgRecouvre(c tcgCadre, a, b int) string {
	var out []string
	for _, e := range c.episodes {
		if e.t0 <= b && e.t1 >= a {
			out = append(out, fmt.Sprintf("ep%d-%d/i%d", e.t0, e.t1, e.index))
		}
	}
	if len(out) == 0 {
		return ""
	}
	return "[" + strings.Join(out, ",") + "]"
}

// tcgS1TypesDePaquet publie le recensement des types de paquet du film, dans et hors des episodes.
func tcgS1TypesDePaquet(t *testing.T, tc t516Temoin, cad tcgCadre) {
	t.Helper()
	dans, hors := map[int]int{}, map[int]int{}
	for _, c := range tc.fc.ChunkNumbers() {
		_, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if cad.episodeDe(cad.trame(pk.TimestampUS), 0) >= 0 {
				dans[int(pk.Type)]++
			} else {
				hors[int(pk.Type)]++
			}
		}
	}
	nom := func(k int) string { return fmt.Sprintf("type%d", k) }
	t.Logf("== S1.8 TYPES DE PAQUET : dans les episodes %s · hors %s", tcgCompteTri(dans, nom, 0),
		tcgCompteTri(hors, nom, 0))
}
