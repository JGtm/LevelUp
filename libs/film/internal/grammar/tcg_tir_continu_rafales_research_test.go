//go:build research

package grammar

// tcg_tir_continu_rafales_research_test.go — S2 TER de la sonde P1 : LE BLOC D ACTION DE LA VUE C
// comme candidat du tir continu. Trois questions, dans l ordre ou elles jugent le candidat :
//
//	(a) QUI : pour chaque index de controle, les valeurs du bloc (m0, m2, m4, m5, r3, c) quand sa
//	    garde est ouverte — un pilote de vehicule a tir continu doit etre le seul a l ouvrir ;
//	(b) LA COUVERTURE, frag par frag : paquets de la fenetre, paquets dont la vue C est LUE, entrees
//	    du pilote, entrees a bloc ouvert, et l ecart a l ouverture la plus proche AVANT le frag —
//	    un zero ne vaut negatif que si la vue C a ete lue dans la fenetre ;
//	(c) LES RAFALES : suites d entrees a bloc ouvert (ecart < 500 ms), durees, et la cadence des
//	    entrees pendant une rafale.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// tcgRafaleEcartUS separe deux rafales : 500 ms sans entree a bloc ouvert.
const tcgRafaleEcartUS = 500_000

// tcgS2Blocs publie (a) : par index, la distribution des valeurs du bloc d action ouvert.
func tcgS2Blocs(t *testing.T, p *tcgPasse) {
	t.Helper()
	parIndex := map[int]map[string]int{}
	for _, e := range p.ctls {
		if !e.actions {
			continue
		}
		if parIndex[e.index] == nil {
			parIndex[e.index] = map[string]int{}
		}
		k := fmt.Sprintf("m0=%d m2=%d m4=%d m5=%d r3=%d c=%v", e.m0, e.m2, e.m4, e.m5, e.r3, e.cBloc)
		parIndex[e.index][k]++
	}
	t.Logf("== S2.8 BLOC D ACTION OUVERT, valeurs par index de controle :")
	for i, m := range parIndex {
		var parts []string
		for k, n := range m {
			parts = append(parts, fmt.Sprintf("[%s]:%d", k, n))
		}
		sort.Strings(parts)
		t.Logf("   index %d : %s", i, strings.Join(parts, " "))
	}
}

// tcgS2Couverture publie (b) : la couverture de la vue C autour de chaque frag.
func tcgS2Couverture(t *testing.T, p *tcgPasse) {
	t.Helper()
	t.Logf("== S2.9 COUVERTURE DE LA VUE C autour des frags (index %d) : paquets / vue C lue / "+
		"entrees du pilote / blocs ouverts dans [f-20, f] ; ouverture la plus proche avant et apres",
		p.cad.index)
	for _, f := range p.cad.frags {
		ix := p.cad.indexDe(f)
		paquets, lues := 0, 0
		for tr := f - tcgAvantFrag; tr <= f; tr++ {
			paquets += p.paquetsParTrame[tr]
			lues += p.vueCLue[tr]
		}
		entrees, ouverts := 0, 0
		avant, apres := -1, -1
		for _, e := range p.ctls {
			if e.index != ix {
				continue
			}
			if e.trame >= f-tcgAvantFrag && e.trame <= f {
				entrees++
				if e.actions {
					ouverts++
				}
			}
			if e.actions && e.trame <= f && (avant < 0 || f-e.trame < avant) {
				avant = f - e.trame
			}
			if e.actions && e.trame > f && (apres < 0 || e.trame-f < apres) {
				apres = e.trame - f
			}
		}
		t.Logf("   frag t=%d : paquets %d · vue C lue %d · entrees %d · blocs ouverts %d · ouverture la "+
			"plus proche AVANT : %d pas · APRES : %d pas", f, paquets, lues, entrees, ouverts, avant, apres)
	}
}

// tcgRafale est une suite d entrees a bloc ouvert d un meme index.
type tcgRafale struct {
	index          int
	debut, fin     uint64
	entrees        int
	lectures       int // entrees du meme index (ouvertes ou non) entre debut et fin
	t0, t1         int
	valeurs, suite string
}

// tcgRafales regroupe les entrees a bloc ouvert en rafales, par index.
func tcgRafales(p *tcgPasse) []tcgRafale {
	par := map[int][]tcgCtl{}
	for _, e := range p.ctls {
		par[e.index] = append(par[e.index], e)
	}
	var out []tcgRafale
	for idx, es := range par {
		sort.SliceStable(es, func(i, j int) bool { return es[i].ts < es[j].ts })
		var cur *tcgRafale
		for _, e := range es {
			switch {
			case e.actions && (cur == nil || e.ts-cur.fin > tcgRafaleEcartUS):
				if cur != nil {
					out = append(out, *cur)
				}
				cur = &tcgRafale{index: idx, debut: e.ts, fin: e.ts, t0: e.trame, t1: e.trame}
				cur.entrees, cur.lectures = 1, 1
				cur.suite = fmt.Sprintf("%d%d", e.m0, e.r3)
			case e.actions:
				cur.fin, cur.t1 = e.ts, e.trame
				cur.entrees++
				cur.lectures++
				if len(cur.suite) < 60 {
					cur.suite += fmt.Sprintf(" %d%d", e.m0, e.r3)
				}
			case cur != nil && e.ts-cur.fin <= tcgRafaleEcartUS:
				cur.lectures++
			}
		}
		if cur != nil {
			out = append(out, *cur)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].debut < out[j].debut })
	return out
}

// tcgS2Rafales publie (c) : les rafales, leurs durees, la cadence des entrees.
func tcgS2Rafales(t *testing.T, p *tcgPasse) {
	t.Helper()
	rs := tcgRafales(p)
	t.Logf("== S2.10 RAFALES (entrees a bloc ouvert, ecart < 500 ms) : %d", len(rs))
	var dureeTot float64
	var entreesTot int
	for _, r := range rs {
		d := float64(r.fin-r.debut) / 1e6
		dureeTot += d
		entreesTot += r.entrees
		ep := p.cad.episodeDe(r.t0, 10)
		t.Logf("   index %d t=%d..%d (%.2f s) episode=%d : %d entrees ouvertes / %d lues · m0r3=%s",
			r.index, r.t0, r.t1, d, ep, r.entrees, r.lectures, r.suite)
	}
	if dureeTot > 0 {
		t.Logf("   total : %.1f s de rafale, %d entrees ouvertes, soit %.1f entrees/s", dureeTot,
			entreesTot, float64(entreesTot)/dureeTot)
	}
}

// tcgS2Chronologie publie, seconde par seconde dans les episodes, la couverture de la vue C et les
// entrees du pilote : paquets, vue C lue, entrees, entrees a bloc ouvert, et l ETAT du bloc a la
// derniere entree lue (une entree emise au CHANGEMENT tient son etat jusqu a la suivante).
func tcgS2Chronologie(t *testing.T, p *tcgPasse) {
	t.Helper()
	t.Logf("== S2.11 CHRONOLOGIE par seconde (index %d) : s=[paquets/vueC lue/entrees/ouvertes] "+
		"etat=bloc de la derniere entree connue ; F = frag dans la seconde", p.cad.index)
	for _, ep := range p.cad.episodes {
		var es []tcgCtl
		for _, e := range p.ctls {
			if e.index == ep.index {
				es = append(es, e)
			}
		}
		sort.SliceStable(es, func(i, j int) bool { return es[i].ts < es[j].ts })
		k := 0
		etat := "?"
		t.Logf("   -- episode t %d..%d, index %d", ep.t0, ep.t1, ep.index)
		var ligne []string
		for s := ep.t0 / 10; s <= ep.t1/10; s++ {
			paq, lue, ent, ouv := 0, 0, 0, 0
			for tr := s * 10; tr < s*10+10; tr++ {
				paq += p.paquetsParTrame[tr]
				lue += p.vueCLue[tr]
			}
			for k < len(es) && es[k].trame < s*10+10 {
				if es[k].trame >= s*10 {
					ent++
					if es[k].actions {
						ouv++
					}
				}
				etat = map[bool]string{true: "O", false: "-"}[es[k].actions]
				k++
			}
			f := ""
			for _, fr := range p.cad.frags {
				if fr/10 == s {
					f = "F"
				}
			}
			ligne = append(ligne, fmt.Sprintf("%d=[%d/%d/%d/%d]%s%s", s, paq, lue, ent, ouv, etat, f))
			if len(ligne) == 8 {
				t.Logf("   %s", strings.Join(ligne, " "))
				ligne = nil
			}
		}
		if len(ligne) > 0 {
			t.Logf("   %s", strings.Join(ligne, " "))
		}
	}
}

// tcgS2Transitions publie les entrees du pilote de part et d autre de chaque CHANGEMENT d etat du
// bloc d action (ouvert <-> ferme) et de chaque silence de plus de 1 s : la forme d emission
// (continue, ou au changement) se lit la.
func tcgS2Transitions(t *testing.T, p *tcgPasse) {
	t.Helper()
	var es []tcgCtl
	for _, e := range p.ctls {
		if i := p.cad.episodeDe(e.trame, 10); i >= 0 && e.index == p.cad.episodes[i].index {
			es = append(es, e)
		}
	}
	sort.SliceStable(es, func(i, j int) bool { return es[i].ts < es[j].ts })
	t.Logf("== S2.12 TRANSITIONS du bloc d action (index %d) et silences > 1 s : ms(film) etat a/b",
		p.cad.index)
	for i := 1; i < len(es); i++ {
		a, b := es[i-1], es[i]
		change := a.actions != b.actions
		silence := b.ts-a.ts > 1_000_000
		if !change && !silence {
			continue
		}
		var avant, apres []string
		for j := max(0, i-3); j < i; j++ {
			avant = append(avant, tcgFmtCtl(es[j]))
		}
		for j := i; j < min(len(es), i+3); j++ {
			apres = append(apres, tcgFmtCtl(es[j]))
		}
		genre := "CHANGEMENT"
		if !change {
			genre = fmt.Sprintf("SILENCE %.1f s", float64(b.ts-a.ts)/1e6)
		}
		t.Logf("   %s t=%d->%d : %s || %s", genre, a.trame, b.trame, strings.Join(avant, " "),
			strings.Join(apres, " "))
	}
}

// tcgFmtCtl rend une entree de controle en une courte chaine.
func tcgFmtCtl(e tcgCtl) string {
	etat := "-"
	if e.actions {
		etat = "O"
	}
	return fmt.Sprintf("%d%s(%d,%d)", e.ts/1000, etat, e.a, e.b)
}
