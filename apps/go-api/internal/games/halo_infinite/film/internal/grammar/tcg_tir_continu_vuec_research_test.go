//go:build research

package grammar

// tcg_tir_continu_vuec_research_test.go — S2 BIS de la sonde P1 : LA VUE DE CONTROLE (vue C,
// `replication_control_view.cpp`). La note 5.22 a lu chez l ecrivain que la gachette est une
// ACTION D UNITE (`primary_trigger` = bit 4, `secondary_trigger` = bit 5 d un mot par unite, table
// thread-locale) et non un champ d objet : aucun composant `ti=35` / `ti=40` ne peut la porter. Le
// seul canal d ENTREE du film est la vue C (`FUN_1406d0388` : index de controle R(5), couple
// analogique, bloc d action `FUN_1406d025c`). Cette lecture la rejoue EN RENDANT SES VALEURS, au
// bit pres de la production (`consumeControleVueC`, `consumeEntreeControle`, `consume1406d025c`),
// sur les seules trames dont la vue B a clos sa liste (le debut de la vue C est alors certain).

import (
	"fmt"
	"sort"
	"testing"
)

// tcgCtl est UNE entree de controle lue, datee.
type tcgCtl struct {
	ts                     uint64
	trame, index, idx7     int
	fermee, lu             bool // lu : l entree a ete lue jusqu au bout (aucune branche non portee)
	bloc, second           bool
	a, b                   int  // le couple analogique (codes 6 bits, 0x1f = zero exact)
	actions                bool // la garde du bloc d action
	m0, m2, m4, m5, r3     uint64
	cBloc, secondBlocOuvre bool
}

// tcgLireActions rejoue `consume1406d025c` en rendant ses champs (memes lectures, meme ordre).
func tcgLireActions(br *Lecteur, e *tcgCtl) {
	if e.actions = br.ReadBit(); !e.actions {
		return
	}
	if br.ReadBit() {
		e.m0, e.m2 = br.ReadBits(3), br.ReadBits(3)
	}
	if br.ReadBit() {
		e.m4, e.m5 = br.ReadBits(2), br.ReadBits(2)
	}
	if e.cBloc = br.ReadBit(); e.cBloc {
		br.ReadBits(2)
		consumeOpt1431a0bbc(br)
		consumeOpt1431a0abc(br)
		consumeQuatBlock1431a0cbc(br)
	}
	e.r3 = br.ReadBits(3)
	if (e.m0&0b111) != 0 || (e.m4&0b11) != 0 {
		consumeID2(br)
	}
	if (e.m2&0b111) != 0 || (e.m5&0b11) != 0 {
		consumeID2(br)
	}
	consume142f26740(br)
}

// tcgLireControle rejoue `consumeControleVueC` + `consumeEntreeControle` en rendant les valeurs.
// Rend false quand une branche non portee s ouvre (le curseur ne peut plus avancer).
func tcgLireControle(br *Lecteur, frameLen int, e *tcgCtl) bool {
	e.idx7 = -1
	if !placeDisponible(br, frameLen, 1+largeurIndexCdc04+largeurIndexControle+1) {
		return false
	}
	if br.ReadBit() {
		e.idx7 = int(br.ReadBits(largeurIndexCdc04)) //nolint:gosec // 7 bits
	}
	e.index = int(br.ReadBits(largeurIndexControle)) //nolint:gosec // 5 bits
	if e.bloc = br.ReadBit(); e.bloc {
		if !placeDisponible(br, frameLen, 1+largeurCourteControle+2*LargeurScalaireAnalogique+3) {
			return false
		}
		if e.second = br.ReadBit(); e.second {
			br.Skip(largeurCourteControle)
		}
		e.a = int(br.ReadBits(LargeurScalaireAnalogique)) //nolint:gosec // 6 bits
		e.b = int(br.ReadBits(LargeurScalaireAnalogique)) //nolint:gosec // 6 bits
		if br.ReadBit() || br.ReadBit() || br.ReadBit() {
			return false // troisieme champ, champ de pile, ou branche R(5) : non portes
		}
		tcgLireActions(br, e)
		if br.BitPos() > frameLen {
			return false
		}
	}
	if !placeDisponible(br, frameLen, 1) {
		return false
	}
	e.secondBlocOuvre = br.ReadBit()
	return !e.secondBlocOuvre
}

// tcgLireVueC lit la vue C d un paquet a partir de la fin de la vue B, en rendant ses entrees.
func tcgLireVueC(pay []byte, finB, tr int, ts uint64, fermee bool) []tcgCtl {
	frameLen := len(pay) * 8
	br := LecteurSur(pay)
	br.Skip(finB)
	var out []tcgCtl
	for tour := 0; tour < plafondToursVueC; tour++ {
		if !placeDisponible(br, frameLen, 1+LargeurKindVueC) || !br.ReadBit() {
			return out
		}
		k := int(br.ReadBits(LargeurKindVueC)) //nolint:gosec // 2 bits
		if k == kindVueCNeant {
			continue
		}
		if k != kindVueCControle {
			return out
		}
		e := tcgCtl{ts: ts, trame: tr, fermee: fermee}
		e.lu = tcgLireControle(br, frameLen, &e)
		out = append(out, e)
		if !e.lu {
			return out
		}
	}
	return out
}

// tcgS2VueC publie la vue C : entrees par index, dans et hors des episodes, ouvertures du bloc
// d action, et le gate frag par frag pour l index du pilote.
func tcgS2VueC(t *testing.T, p *tcgPasse) {
	t.Helper()
	parIndex := map[int]*[3]int{}
	actions := map[int]*[3]int{}
	var fermees, luesFermees int
	for _, e := range p.ctls {
		pop := tcgPopulation(p.cad, e.trame)
		if parIndex[e.index] == nil {
			parIndex[e.index], actions[e.index] = &[3]int{}, &[3]int{}
		}
		parIndex[e.index][pop]++
		if e.actions {
			actions[e.index][pop]++
		}
		if e.fermee {
			fermees++
			if e.lu {
				luesFermees++
			}
		}
	}
	t.Logf("== S2.6 VUE C (controle) : %d entrees, dont %d en trame fermee (%d lues jusqu au bout) — "+
		"par index de controle [2 s avant frag | reste des episodes | hors episodes] :", len(p.ctls),
		fermees, luesFermees)
	idx := make([]int, 0, len(parIndex))
	for i := range parIndex {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	for _, i := range idx {
		t.Logf("   index %2d : entrees %v · bloc d action ouvert %v", i, *parIndex[i], *actions[i])
	}
	cats := []struct {
		nom string
		f   func(tcgCtl) bool
	}{
		{"entree du pilote", func(e tcgCtl) bool { return e.index == p.cad.indexDe(e.trame) }},
		{"bloc d action du pilote", func(e tcgCtl) bool {
			return e.index == p.cad.indexDe(e.trame) && e.actions
		}},
		{"m0|m2|m4|m5 non nul (pilote)", func(e tcgCtl) bool {
			return e.index == p.cad.indexDe(e.trame) && (e.m0|e.m2|e.m4|e.m5) != 0
		}},
	}
	for _, k := range cats {
		var par, moins, plus []int
		for _, f := range p.cad.frags {
			par = append(par, tcgCompteCtl(p, f, k.f))
			moins = append(moins, tcgCompteCtl(p, f-tcgDecalage, k.f))
			plus = append(plus, tcgCompteCtl(p, f+tcgDecalage, k.f))
		}
		t.Logf("   %-30s frags=%v (%d/%d) · -60s=%v · +60s=%v", k.nom, par, tcgNonNuls(par), len(par),
			moins, plus)
	}
	n := 0
	for _, e := range p.ctls {
		if e.index == p.cad.indexDe(e.trame) && p.cad.episodeDe(e.trame, 10) >= 0 && n < 30 {
			n++
			t.Logf("     t=%d fermee=%v lu=%v idx7=%d bloc=%v a=%d b=%d actions=%v m0=%d m2=%d m4=%d m5=%d "+
				"r3=%d c=%v", e.trame, e.fermee, e.lu, e.idx7, e.bloc, e.a, e.b, e.actions, e.m0, e.m2,
				e.m4, e.m5, e.r3, e.cBloc)
		}
	}
}

// tcgCompteCtl compte les entrees d une categorie dans [f-20, f].
func tcgCompteCtl(p *tcgPasse, f int, cat func(tcgCtl) bool) int {
	n := 0
	for _, e := range p.ctls {
		if e.trame >= f-tcgAvantFrag && e.trame <= f && cat(e) {
			n++
		}
	}
	return n
}

// tcgS2NaissancesParType publie les naissances (records NEW) par archetype, par seconde, dans et
// hors des episodes, et les projectiles nes dans les 2 s avant un frag.
func tcgS2NaissancesParType(t *testing.T, p *tcgPasse) {
	t.Helper()
	dans, hors := map[int]int{}, map[int]int{}
	dureeDans := 0
	for _, e := range p.cad.episodes {
		dureeDans += e.t1 - e.t0
	}
	for _, x := range p.news {
		if p.cad.episodeDe(x.trame, 0) >= 0 {
			dans[int(x.ti)]++
		} else {
			hors[int(x.ti)]++
		}
	}
	nom := func(k int) string { return fmt.Sprintf("ti%d", k) }
	t.Logf("== S2.7 NAISSANCES par archetype : dans les episodes (%.1f s) %s", float64(dureeDans)/10,
		tcgCompteTri(dans, nom, 0))
	t.Logf("   hors episodes : %s", tcgCompteTri(hors, nom, 0))
	for _, f := range p.cad.frags {
		var ids []string
		for _, x := range p.news {
			if x.trame >= f-tcgAvantFrag && x.trame <= f && x.ti == ProjectileTypeIndex {
				ids = append(ids, fmt.Sprintf("%d@%d", x.slot, x.trame))
			}
		}
		t.Logf("   frag t=%d : projectiles nes dans [f-20, f] : %v", f, ids)
	}
}
