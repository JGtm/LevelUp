//go:build research

package grammar

// p1s3_vuec_publication_research_test.go — PUBLICATION de la sonde P1-S3 (en-tete et lecture :
// `p1s3_vuec_tir_continu_research_test.go`) : entrees par index, gate frag par frag, rafales,
// entrees autour des frags, cadence. Mesure seule.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

func s3B(b bool) int {
	if b {
		return 1
	}
	return 0
}

// s3PublierIndex publie, par index de controle, les entrees et le bloc d action.
func s3PublierIndex(t *testing.T, ents []s3Entree, cad s3Cadre) {
	t.Helper()
	type agg struct{ n, val, act, tir, tirVal, tirDans, tirHors, drap int }
	par := map[int]*agg{}
	vals := map[string]int{}
	for _, e := range ents {
		a := par[e.index]
		if a == nil {
			a = &agg{}
			par[e.index] = a
		}
		a.n, a.val, a.act = a.n+1, a.val+s3B(e.valide), a.act+s3B(e.actions)
		a.drap += s3B(e.drapeaux != 0)
		if e.tir() {
			a.tir, a.tirVal = a.tir+1, a.tirVal+s3B(e.valide)
			if cad.dansEpisode(e.trame) {
				a.tirDans++
			} else {
				a.tirHors++
			}
			vals[fmt.Sprintf("i%d m0=%d m2=%d m4=%d m5=%d r3=%d arme=%d/%d genre=%d valide=%v", e.index, e.m0,
				e.m2, e.m4, e.m5, e.r3, e.arme0, e.arme1, e.genre, e.valide)]++
		}
	}
	idx := make([]int, 0, len(par))
	for i := range par {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	t.Logf("== ENTREES par index (%d) : n · validees · garde d action · TIR [valide] [dans | hors "+
		"episodes] · drapeaux R(5) non nuls", len(ents))
	for _, i := range idx {
		a := par[i]
		t.Logf("   index %2d : %6d · %6d · %5d · %5d [%5d] [%5d | %5d] · %5d", i, a.n, a.val, a.act,
			a.tir, a.tirVal, a.tirDans, a.tirHors, a.drap)
	}
	cles := make([]string, 0, len(vals))
	for k := range vals {
		cles = append(cles, k)
	}
	sort.Strings(cles)
	for _, k := range cles {
		t.Logf("   valeur %-52s x %d", k, vals[k])
	}
}

// s3Compte compte les entrees du pilote qui TIRENT dans [f-20, f], et les entrees lues.
func s3Compte(ents []s3Entree, idx, f int) (tir, lues int) {
	for _, e := range ents {
		if e.index == idx && e.trame >= f-20 && e.trame <= f {
			lues++
			tir += s3B(e.tir())
		}
	}
	return tir, lues
}

// s3Couverture compte, dans [f-20, f], les paquets par statut et les paquets clos.
func s3Couverture(paqs []s3Paquet, f int) string {
	var st [3]int
	clos := 0
	for _, b := range paqs {
		if b.trame >= f-20 && b.trame <= f {
			st[b.statut]++
			clos += s3B(b.fermeMoi)
		}
	}
	return fmt.Sprintf("%d/%d/%d clos %d", st[0], st[1], st[2], clos)
}

// s3PublierGate publie le gate de la sonde : chaque frag, temoins a +/-60 s, hors episodes.
func s3PublierGate(t *testing.T, ents []s3Entree, paqs []s3Paquet, cad s3Cadre, seulValide bool) {
	t.Helper()
	var par, moins, plus, couv []string
	nFrag := 0
	for _, f := range cad.frags {
		a, la := s3Compte(ents, cad.index, f)
		m, lm := s3Compte(ents, cad.index, f-600)
		p, lp := s3Compte(ents, cad.index, f+600)
		nFrag += s3B(a > 0)
		par = append(par, fmt.Sprintf("%d/%d", a, la))
		moins = append(moins, fmt.Sprintf("%d/%d(%v)", m, lm, cad.dansEpisode(f-600)))
		plus = append(plus, fmt.Sprintf("%d/%d(%v)", p, lp, cad.dansEpisode(f+600)))
		couv = append(couv, s3Couverture(paqs, f))
	}
	var hors, horsLues int
	for _, e := range ents {
		if e.index == cad.index && !cad.dansEpisode(e.trame) {
			horsLues++
			hors += s3B(e.tir())
		}
	}
	t.Logf("== GATE index %d, entrees %s (qui tirent / lues dans [f-2 s, f]) :", cad.index,
		map[bool]string{false: "TOUTES", true: "VALIDEES"}[seulValide])
	t.Logf("   frags %v : %v -> %d/%d", cad.frags, par, nFrag, len(cad.frags))
	t.Logf("   couverture [f-2 s, f] (non localises/vue B ouverte/vue B close, clos) : %v", couv)
	t.Logf("   temoin -60 s (dans un episode ?) : %v", moins)
	t.Logf("   temoin +60 s (dans un episode ?) : %v", plus)
	t.Logf("   hors episodes : %d entrees qui tirent sur %d lues", hors, horsLues)
}

// s3PublierRafales publie les rafales du pilote : suites d entrees lues qui tirent, bornees par
// une entree lue qui ne tire pas (les paquets non lus sont des TROUS, pas des arrets).
func s3PublierRafales(t *testing.T, ents []s3Entree, cad s3Cadre, seulValide bool) {
	t.Helper()
	var lignes []string
	debut, dern, n, avant := -1, -1, 0, -1
	var tsDebut, tsDern uint64
	fermer := func(apres int) {
		if debut >= 0 {
			lignes = append(lignes, fmt.Sprintf("t %d..%d (%d entrees, %.2f s) · derniere lue sans tir "+
				"avant %d, apres %d", debut, dern, n, float64(tsDern-tsDebut)/1e6, avant, apres))
		}
		debut, n = -1, 0
	}
	for _, e := range ents {
		if e.index != cad.index {
			continue
		}
		if !e.tir() {
			fermer(e.trame)
			avant = e.trame
			continue
		}
		if debut < 0 {
			debut, tsDebut = e.trame, e.ts
		}
		dern, tsDern = e.trame, e.ts
		n++
	}
	fermer(-1)
	t.Logf("== RAFALES de l index %d, entrees %s : %d", cad.index,
		map[bool]string{false: "TOUTES", true: "VALIDEES"}[seulValide], len(lignes))
	for _, l := range lignes {
		t.Logf("   %s", l)
	}
}

// s3PublierAutourDesFrags liste les entrees du pilote dans [f-25, f+3].
func s3PublierAutourDesFrags(t *testing.T, ents []s3Entree, cad s3Cadre) {
	t.Helper()
	for _, f := range cad.frags {
		var sb strings.Builder
		avant, apres := -1, -1
		for _, e := range ents {
			if e.index != cad.index || !e.valide {
				continue
			}
			if e.trame <= f {
				avant = e.trame
			} else if apres < 0 {
				apres = e.trame
			}
			if e.trame >= f-25 && e.trame <= f+3 {
				fmt.Fprintf(&sb, "%d%s ", e.trame, map[bool]string{true: "T", false: "-"}[e.tir()])
			}
		}
		t.Logf("   frag %d : derniere entree validee avant %d, premiere apres %d · [f-2,5 s, f+0,3 s] %s",
			f, avant, apres, sb.String())
	}
	s3PublierCouverture(t, ents, cad)
}

// s3PublierCouverture publie, par episode, la part des trames (100 ms) ou une entree validee du
// pilote est lue, et la part de celles-ci ou il tire.
func s3PublierCouverture(t *testing.T, ents []s3Entree, cad s3Cadre) {
	t.Helper()
	lues, tir := map[int]bool{}, map[int]bool{}
	for _, e := range ents {
		if e.index == cad.index && e.valide {
			lues[e.trame] = true
			if e.tir() {
				tir[e.trame] = true
			}
		}
	}
	for _, ep := range cad.episodes {
		var nl, nt int
		for tr := ep[0]; tr <= ep[1]; tr++ {
			nl += s3B(lues[tr])
			nt += s3B(tir[tr])
		}
		t.Logf("   episode %d-%d : trames lues %d/%d (%.0f %%) · trames qui tirent %d (%.0f %% des lues)",
			ep[0], ep[1], nl, ep[1]-ep[0]+1, 100*float64(nl)/float64(ep[1]-ep[0]+1), nt,
			100*float64(nt)/float64(max(nl, 1)))
	}
}

// s3PublierCadence publie les ecarts d horloge entre entrees consecutives du pilote, et idx7.
func s3PublierCadence(t *testing.T, ents []s3Entree, cad s3Cadre) {
	t.Helper()
	ecarts, idx7 := map[uint64]int{}, map[int]int{}
	var prec uint64
	for _, e := range ents {
		if e.index != cad.index || !cad.dansEpisode(e.trame) {
			continue
		}
		idx7[e.idx7]++
		if prec != 0 && e.ts > prec {
			ecarts[(e.ts-prec+500)/1000]++
		}
		prec = e.ts
	}
	cles := make([]uint64, 0, len(ecarts))
	for k := range ecarts {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool { return ecarts[cles[i]] > ecarts[cles[j]] })
	var sb strings.Builder
	for i, k := range cles {
		if i == 12 {
			break
		}
		fmt.Fprintf(&sb, "%d ms x%d  ", k, ecarts[k])
	}
	t.Logf("== CADENCE des entrees lues de l index %d dans ses episodes : %s", cad.index, sb.String())
	t.Logf("   idx7 (FUN_1406cdc04, -1 = absent) : %v", idx7)
}
