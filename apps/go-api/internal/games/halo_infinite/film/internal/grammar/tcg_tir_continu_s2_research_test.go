//go:build research

package grammar

// tcg_tir_continu_s2_research_test.go — S2 de la sonde P1 : LES COMPOSANTS du vehicule suivi, de
// son pilote et de tout objet attache au vehicule (`i10 object-parent-state`). Question : un
// composant bascule-t-il au debut d une rafale ? Mesure par TAUX (records portant le composant /
// records du role) dans trois populations disjointes : les 2 s avant un frag, le reste des
// episodes, hors des episodes ; et presence frag par frag contre le temoin decale de +/-60 s.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// tcgNomComposant rend le nom d un composant d archetype.
func tcgNomComposant(reg *Registry, ti uint32, i int) string {
	a, ok := reg.Archetype(int(ti))
	if !ok || i < 0 || i >= len(a.Components) {
		return fmt.Sprintf("i%d", i)
	}
	return fmt.Sprintf("i%d %s", i, strings.TrimSuffix(a.Components[i], "-component"))
}

// tcgIndices rend les indices poses dans un masque.
func tcgIndices(m uint64) []int {
	var out []int
	for i := 0; i < 64; i++ {
		if m&(1<<uint(i)) != 0 {
			out = append(out, i)
		}
	}
	return out
}

// tcgRole dit le role d un record : "vehicule", "pilote", "attache" ou "".
func tcgRole(c tcgCadre, r tcgRec) string {
	switch {
	case c.vehiculeSuivi(r.slot):
		return "vehicule"
	case c.suivi(r.slot):
		return "pilote"
	case r.aParent && c.vehiculeSuivi(r.parent):
		return "attache"
	}
	return ""
}

// tcgPopulation range un record : 0 = 2 s avant un frag, 1 = reste des episodes, 2 = hors.
func tcgPopulation(c tcgCadre, tr int) int {
	for _, f := range c.frags {
		if tr >= f-tcgAvantFrag && tr <= f {
			return 0
		}
	}
	if c.episodeDe(tr, 10) >= 0 {
		return 1
	}
	return 2
}

// tcgS2Rapport publie S2.
func tcgS2Rapport(t *testing.T, p *tcgPasse, reg *Registry) {
	t.Helper()
	tcgS2Taux(t, p, reg)
	tcgS2Gate(t, p, reg)
	tcgS2Attaches(t, p, reg)
	tcgS2Naissances(t, p, reg)
	tcgS2AutourDesFrags(t, p, reg)
}

// tcgS2Taux publie, par role et par composant, le TAUX de presence dans les trois populations,
// avec le desync et la part des trames fermees.
func tcgS2Taux(t *testing.T, p *tcgPasse, reg *Registry) {
	t.Helper()
	type cle struct {
		role string
		ti   uint32
	}
	n := map[cle]*[3]int{}
	parComp := map[cle]map[int]*[3]int{}
	desync, fermees := map[cle]map[int]int{}, map[cle]int{}
	for _, r := range p.recs {
		k := cle{tcgRole(p.cad, r), r.ti}
		if n[k] == nil {
			n[k], parComp[k], desync[k] = &[3]int{}, map[int]*[3]int{}, map[int]int{}
		}
		pop := tcgPopulation(p.cad, r.trame)
		n[k][pop]++
		desync[k][r.desync]++
		if r.fermee {
			fermees[k]++
		}
		for _, i := range tcgIndices(r.masque) {
			if parComp[k][i] == nil {
				parComp[k][i] = &[3]int{}
			}
			parComp[k][i][pop]++
		}
	}
	t.Logf("== S2.1 TAUX PAR COMPOSANT (records portant le composant / records du role) — " +
		"[2 s avant frag | reste des episodes | hors episodes]")
	for k, tot := range n {
		t.Logf("   ROLE %s ti=%d : records %v · trames fermees %d · desync (index:nombre) %s", k.role,
			k.ti, *tot, fermees[k], tcgCompteTri(desync[k], func(i int) string { return fmt.Sprint(i) }, 0))
		idx := make([]int, 0, len(parComp[k]))
		for i := range parComp[k] {
			idx = append(idx, i)
		}
		sort.Ints(idx)
		for _, i := range idx {
			c := parComp[k][i]
			t.Logf("      %-52s %5.1f %% (%d) | %5.1f %% (%d) | %5.1f %% (%d)",
				tcgNomComposant(reg, k.ti, i), tcgPct(c[0], tot[0]), c[0], tcgPct(c[1], tot[1]), c[1],
				tcgPct(c[2], tot[2]), c[2])
		}
	}
}

// tcgPct rend un pourcentage.
func tcgPct(a, b int) float64 { return 100 * float64(a) / float64(max(1, b)) }

// tcgS2Gate publie, pour chaque (role, composant), la presence frag par frag dans [f-2 s, f] contre
// le temoin decale, et le compte hors episodes.
func tcgS2Gate(t *testing.T, p *tcgPasse, reg *Registry) {
	t.Helper()
	type cle struct {
		role string
		ti   uint32
		comp int
	}
	vus := map[cle]bool{}
	for _, r := range p.recs {
		for _, i := range tcgIndices(r.masque) {
			vus[cle{tcgRole(p.cad, r), r.ti, i}] = true
		}
	}
	cles := make([]cle, 0, len(vus))
	for k := range vus {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool { return fmt.Sprint(cles[i]) < fmt.Sprint(cles[j]) })
	t.Logf("== S2.2 GATE PAR COMPOSANT : records portant le composant dans [f-2 s, f] pour chaque frag,"+
		" temoin -60 s / +60 s, hors episodes (%d frags)", len(p.cad.frags))
	for _, k := range cles {
		compte := func(f int) int {
			n := 0
			for _, r := range p.recs {
				if r.trame >= f-tcgAvantFrag && r.trame <= f && r.masque&(1<<uint(k.comp)) != 0 &&
					tcgRole(p.cad, r) == k.role && r.ti == k.ti {
					n++
				}
			}
			return n
		}
		var par, moins, plus []int
		for _, f := range p.cad.frags {
			par, moins, plus = append(par, compte(f)), append(moins, compte(f-tcgDecalage)),
				append(plus, compte(f+tcgDecalage))
		}
		t.Logf("   %-8s ti=%d %-48s frags=%v (%d/%d) · -60s=%v · +60s=%v", k.role, k.ti,
			tcgNomComposant(reg, k.ti, k.comp), par, tcgNonNuls(par), len(par), moins, plus)
	}
}

// tcgS2Attaches publie les objets dont `i10` designe un vehicule suivi.
func tcgS2Attaches(t *testing.T, p *tcgPasse, reg *Registry) {
	t.Helper()
	type cle struct{ slot, ti, parent uint32 }
	type span struct{ n, t0, t1 int }
	vus := map[cle]*span{}
	for _, r := range p.recs {
		if !r.aParent || !p.cad.vehiculeSuivi(r.parent) {
			continue
		}
		k := cle{r.slot, r.ti, r.parent}
		if vus[k] == nil {
			vus[k] = &span{t0: r.trame}
		}
		vus[k].n++
		vus[k].t1 = r.trame
	}
	t.Logf("== S2.3 OBJETS ATTACHES A UN VEHICULE SUIVI (i10 attache, parent = base 0x200 + valeur) : %d",
		len(vus))
	for k, s := range vus {
		a, _ := reg.Archetype(int(k.ti))
		nom := ""
		if len(a.Components) > 0 {
			nom = a.Components[len(a.Components)-1]
		}
		t.Logf("   slot %d ti=%d (dernier composant %s) -> parent %d : %d lectures, t %d..%d", k.slot,
			k.ti, nom, k.parent, s.n, s.t0, s.t1)
	}
}

// tcgS2Naissances publie les naissances voisines de celle de chaque vehicule suivi, et les entites
// dont la duree de vie aux images-cles est celle du vehicule.
func tcgS2Naissances(t *testing.T, p *tcgPasse, reg *Registry) {
	t.Helper()
	t.Logf("== S2.4 NAISSANCES (records NEW) a +/-3 s de celle d un vehicule suivi :")
	for _, v := range p.news {
		if !p.cad.vehiculeSuivi(v.slot) {
			continue
		}
		t.Logf("   vehicule %d ne a t=%d (ti=%d, gen %d) :", v.slot, v.trame, v.ti, v.gen)
		for _, x := range p.news {
			if x.trame >= v.trame-30 && x.trame <= v.trame+30 && x.ti != BipedTypeIndex {
				t.Logf("      NEW t=%d slot=%d gen=%d ti=%d masque=%v", x.trame, x.slot, x.gen, x.ti,
					tcgIndices(x.masque))
			}
		}
	}
	type ent struct{ slot, ti, gen int }
	prem, dern := map[ent]int{}, map[ent]int{}
	for i, ic := range p.ics {
		for _, r := range ic.ents {
			k := ent{r.Slot, r.TI, r.Gen}
			if _, ok := prem[k]; !ok {
				prem[k] = i
			}
			dern[k] = i
		}
	}
	for k := range prem {
		if !p.cad.vehiculeSuivi(uint32(k.slot)) { //nolint:gosec // slot du walker
			continue
		}
		var memes []string
		for j := range prem {
			if j != k && j.ti != BipedTypeIndex && prem[j] == prem[k] && dern[j] == dern[k] {
				memes = append(memes, fmt.Sprintf("%d/ti%d/g%d", j.slot, j.ti, j.gen))
			}
		}
		sort.Strings(memes)
		t.Logf("   vehicule %d (ti=%d gen %d) images-cles %d..%d (t %d..%d) ; entites de MEME duree : %v",
			k.slot, k.ti, k.gen, prem[k], dern[k], p.ics[prem[k]].trame, p.ics[dern[k]].trame, memes)
	}
}

// tcgS2AutourDesFrags liste les records du vehicule, du pilote et des objets attaches dans
// [f-2 s, f+0,2 s] de chaque frag.
func tcgS2AutourDesFrags(t *testing.T, p *tcgPasse, reg *Registry) {
	t.Helper()
	for _, f := range p.cad.frags {
		t.Logf("== S2.5 frag t=%d : records suivis dans [f-20, f+2]", f)
		for _, r := range p.recs {
			if r.trame < f-tcgAvantFrag || r.trame > f+2 {
				continue
			}
			var noms []string
			for _, i := range tcgIndices(r.masque) {
				noms = append(noms, strings.SplitN(tcgNomComposant(reg, r.ti, i), " ", 2)[0])
			}
			t.Logf("   t=%d %-8s slot=%d ti=%d typ=%d rang=%d fermee=%v desync=%d masque=%s", r.trame,
				tcgRole(p.cad, r), r.slot, r.ti, r.typ, r.rang, r.fermee, r.desync,
				strings.Join(noms, ","))
		}
	}
}
