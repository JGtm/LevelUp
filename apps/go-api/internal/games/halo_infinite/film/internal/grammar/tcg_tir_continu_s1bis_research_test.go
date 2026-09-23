//go:build research

package grammar

// tcg_tir_continu_s1bis_research_test.go — S1 BIS de la sonde P1 : trois lectures de la liste qui
// ne sont pas le tir mais qui le BORNENT.
//
//	(a) le COMPTEUR du record 36 : le champ de 8 bits lu par `FUN_141fcf670` (R(7) puis R(1))
//	    avance de 2 a chaque tir d un meme tireur. Si le Ghost tirait par des records 36 que la
//	    marche ne voit pas, le compteur du pilote SAUTERAIT a travers l episode ;
//	(b) les DEGATS du pilote (`damage_aftermath`, ref0 = blesse, ref1 = responsable, source = tag
//	    de 32 bits) : le tir qui TOUCHE est-il dans le film ?
//	(c) les evenements de PROJECTILE (5 detonation, 6 impact, 7 impact sur objet) et leur tag.

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// tcgPilotes rend les slots des pilotes des episodes.
func tcgPilotes(c tcgCadre) map[uint32]bool {
	out := map[uint32]bool{}
	for _, e := range c.episodes {
		out[e.pilote] = true
	}
	return out
}

// tcgS1Compteur publie les tirs du pilote (index 5 bits) dans l ordre du temps, avec le compteur.
func tcgS1Compteur(t *testing.T, p *tcgPasse) {
	t.Helper()
	var lignes []string
	var cur []string
	for _, e := range p.evs {
		if e.tir == nil || e.tir.idx5 != p.cad.index {
			continue
		}
		ep := ""
		if p.cad.episodeDe(e.trame, 0) >= 0 {
			ep = "*EPISODE*"
		}
		cur = append(cur, fmt.Sprintf("t%d:c%d(p%d,%v,%08X)%s", e.trame, e.tir.attaquant, e.pos,
			e.refs[0], e.tir.haut, ep))
		if len(cur) == 8 {
			lignes = append(lignes, strings.Join(cur, " "))
			cur = nil
		}
	}
	if len(cur) > 0 {
		lignes = append(lignes, strings.Join(cur, " "))
	}
	t.Logf("== S1.4 COMPTEUR DU RECORD 36 POUR L INDEX %d (t:compteur(position, ref0, arme haute)) :",
		p.cad.index)
	for _, l := range lignes {
		t.Logf("   %s", l)
	}
}

// tcgSourceGate rend la source de degat du gate : TCG_SOURCE si fournie, sinon la plus frequente
// des degats du pilote dans ses episodes.
func tcgSourceGate(parSource map[uint64]*[3]int) uint64 {
	if v, err := strconv.ParseUint(os.Getenv("TCG_SOURCE"), 16, 64); err == nil {
		return v
	}
	var meilleure uint64
	n := -1
	for s, c := range parSource {
		if c[0]+c[1] > n {
			meilleure, n = s, c[0]+c[1]
		}
	}
	return meilleure
}

// tcgS1Degats publie les degats dont le pilote est le RESPONSABLE (ref1), par source et par
// population, puis le gate sur la source du Ghost.
func tcgS1Degats(t *testing.T, p *tcgPasse) {
	t.Helper()
	pilotes := tcgPilotes(p.cad)
	parSource := map[uint64]*[3]int{}
	for _, e := range p.evs {
		if e.degat == nil || !e.refs[1].present || !pilotes[e.refs[1].slot()] {
			continue
		}
		if parSource[e.degat.sourceID] == nil {
			parSource[e.degat.sourceID] = &[3]int{}
		}
		parSource[e.degat.sourceID][tcgPopulation(p.cad, e.trame)]++
	}
	t.Logf("== S1.5 DEGATS DONT LE PILOTE EST RESPONSABLE (ref1), par source [2 s avant frag | reste "+
		"des episodes | hors episodes] : %d sources", len(parSource))
	for s, c := range parSource {
		t.Logf("   source %08X : %v", s, *c)
	}
	src := tcgSourceGate(parSource)
	cat := func(e tcgEv) bool {
		return e.degat != nil && e.degat.sourceID == src && e.refs[1].present && pilotes[e.refs[1].slot()]
	}
	var par, moins, plus []int
	for _, f := range p.cad.frags {
		par = append(par, tcgFenetre(p, f, cat))
		moins = append(moins, tcgFenetre(p, f-tcgDecalage, cat))
		plus = append(plus, tcgFenetre(p, f+tcgDecalage, cat))
	}
	hors, dans, instants := 0, 0, map[int]bool{}
	for _, e := range p.evs {
		if !cat(e) {
			continue
		}
		if p.cad.episodeDe(e.trame, 10) < 0 {
			hors++
			continue
		}
		dans++
		instants[e.trame] = true
	}
	t.Logf("   GATE source %08X : frags=%v (%d/%d) · -60s=%v · +60s=%v · hors episodes=%d · dans les "+
		"episodes %d evenements sur %d instants distincts (pas de 100 ms)", src, par, tcgNonNuls(par),
		len(par), moins, plus, hors, dans, len(instants))
	tcgVictimes(t, p, cat)
}

// tcgVictimes publie, pour la categorie de degats du gate, les victimes (ref0) et les intervalles
// entre instants successifs (la cadence observable des touches).
func tcgVictimes(t *testing.T, p *tcgPasse, cat func(tcgEv) bool) {
	t.Helper()
	vict := map[int]int{}
	var ts []int
	for _, e := range p.evs {
		if cat(e) && p.cad.episodeDe(e.trame, 10) >= 0 {
			vict[int(e.refs[0].slot())]++
			ts = append(ts, e.trame)
		}
	}
	sort.Ints(ts)
	ecarts := map[int]int{}
	for i := 1; i < len(ts); i++ {
		d := ts[i] - ts[i-1]
		if d > 10 {
			d = 10 // tout ecart > 1 s est range a 1 s
		}
		ecarts[d]++
	}
	nom := func(k int) string { return strconv.Itoa(k) }
	t.Logf("   victimes (ref0 -> slot) : %s", tcgCompteTri(vict, nom, 0))
	t.Logf("   ecarts entre touches successives (pas de 100 ms, 10 = >= 1 s) : %s",
		tcgCompteTri(ecarts, nom, 0))
	tcgCadence(t, p, cat)
}

// tcgCadence mesure la CADENCE des touches a l horloge des paquets (microsecondes) : ecarts entre
// touches successives de moins de 400 ms (une rafale), ranges par tick de 1/60 s.
func tcgCadence(t *testing.T, p *tcgPasse, cat func(tcgEv) bool) {
	t.Helper()
	var ts []uint64
	for _, e := range p.evs {
		if cat(e) && p.cad.episodeDe(e.trame, 10) >= 0 {
			ts = append(ts, e.ts)
		}
	}
	sort.Slice(ts, func(i, j int) bool { return ts[i] < ts[j] })
	parTick := map[int]int{}
	var somme uint64
	n := 0
	for i := 1; i < len(ts); i++ {
		d := ts[i] - ts[i-1]
		if d == 0 || d >= 400_000 {
			continue
		}
		parTick[int((d+8_333)/16_667)]++ //nolint:gosec // < 400 ms
		somme += d
		n++
	}
	if n == 0 {
		t.Logf("   cadence : aucun ecart < 400 ms")
		return
	}
	t.Logf("   cadence des touches en rafale (ecarts < 400 ms, en ticks de 16,7 ms) : %s · moyenne %.0f ms "+
		"(%.1f touches/s) sur %d ecarts", tcgCompteTri(parTick, strconv.Itoa, 0),
		float64(somme)/float64(n)/1000, 1e6*float64(n)/float64(somme), n)
}

// tcgS1Projectiles publie les evenements de projectile (5, 6, 7) : par tag, dans et hors des
// episodes, et ceux des 2 s avant chaque frag.
func tcgS1Projectiles(t *testing.T, p *tcgPasse) {
	t.Helper()
	type cle struct {
		typ int
		tag uint64
	}
	parCle := map[cle]*[3]int{}
	for _, e := range p.evs {
		if e.typ != 5 && e.typ != 6 && e.typ != 7 {
			continue
		}
		k := cle{e.typ, e.tag}
		if parCle[k] == nil {
			parCle[k] = &[3]int{}
		}
		parCle[k][tcgPopulation(p.cad, e.trame)]++
	}
	t.Logf("== S1.6 EVENEMENTS DE PROJECTILE par (type, tag) [2 s avant frag | reste des episodes | " +
		"hors episodes] :")
	cles := make([]cle, 0, len(parCle))
	for k := range parCle {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool {
		a, b := parCle[cles[i]], parCle[cles[j]]
		return a[0]+a[1]+a[2] > b[0]+b[1]+b[2]
	})
	for i, k := range cles {
		if i >= 30 {
			break
		}
		t.Logf("   %-34s tag %08X : %v", tcgNomType(k.typ), k.tag, *parCle[k])
	}
	for _, f := range p.cad.frags {
		for _, e := range p.evs {
			if (e.typ == 5 || e.typ == 6 || e.typ == 7) && e.trame >= f-tcgAvantFrag && e.trame <= f {
				t.Logf("   frag t=%d : %s", f, tcgLigne(e))
			}
		}
	}
}
