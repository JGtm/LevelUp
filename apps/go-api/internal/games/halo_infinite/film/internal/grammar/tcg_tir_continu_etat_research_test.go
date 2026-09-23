//go:build research

package grammar

// tcg_tir_continu_etat_research_test.go — S2 QUATER de la sonde P1 : le bloc d action de la vue C
// lu comme un ETAT tenu (sample-and-hold : la derniere entree lue d un index vaut jusqu a la
// suivante), confronte a deux temoins INDEPENDANTS de la vue C :
//
//	(a) les tirs de vehicule que la production decode deja (records 36 de classe vehicule, armes a
//	    COUP : mortier du Wraith, canon du Scorpion, Wasp, Gungoose) — si le bloc est la gachette,
//	    il doit etre OUVERT a l instant de ces tirs, pour l index du tireur ;
//	(b) les touches du pilote (`damage_aftermath` dont ref1 est son corps) — le bloc doit etre
//	    ouvert (ou venir de se fermer : un projectile vole) a l instant de la touche.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// tcgEtat rend l etat tenu de l index `ix` a l instant `ts` : "O" (bloc ouvert), "-" (ferme) ou
// "?" (aucune entree depuis plus de `tcgEtatMaxUS`), et l age de la derniere entree en ms.
func tcgEtat(par map[int][]tcgCtl, ix int, ts uint64) (string, int) {
	es := par[ix]
	i := sort.Search(len(es), func(k int) bool { return es[k].ts > ts }) - 1
	if i < 0 || ts-es[i].ts > tcgEtatMaxUS {
		return "?", -1
	}
	age := int((ts - es[i].ts) / 1000) //nolint:gosec // borne par tcgEtatMaxUS
	if es[i].actions {
		return "O", age
	}
	return "-", age
}

// tcgEtatMaxUS borne la tenue d un etat : au-dela de 30 s sans entree, l etat est inconnu.
const tcgEtatMaxUS = 30_000_000

// tcgParIndex range les entrees de controle par index, triees par instant.
func tcgParIndex(p *tcgPasse) map[int][]tcgCtl {
	par := map[int][]tcgCtl{}
	for _, e := range p.ctls {
		par[e.index] = append(par[e.index], e)
	}
	for ix := range par {
		es := par[ix]
		sort.SliceStable(es, func(i, j int) bool { return es[i].ts < es[j].ts })
	}
	return par
}

// tcgS2TirsVehicule publie (a) : pour chaque tir decode, l etat tenu du bloc de son tireur.
func tcgS2TirsVehicule(t *testing.T, p *tcgPasse) {
	t.Helper()
	par := tcgParIndex(p)
	type cle struct {
		classe, etat string
		arme         uint64
	}
	n := map[cle]int{}
	var lignes []string
	for _, e := range p.evs {
		if e.tir == nil || e.tir.idx5 < 0 {
			continue
		}
		etat, age := tcgEtat(par, e.tir.idx5, e.ts)
		arme := uint64(0)
		if e.tir.classe() == "vehicule" {
			arme = e.tir.haut
			if len(lignes) < 60 {
				lignes = append(lignes, fmt.Sprintf("t=%d idx=%d %08X etat=%s(%d ms)", e.trame, e.tir.idx5,
					e.tir.haut, etat, age))
			}
		}
		n[cle{e.tir.classe(), etat, arme}]++
	}
	t.Logf("== S2.13 TIRS DECODES (record 36) contre l ETAT TENU du bloc d action de LEUR tireur :")
	cles := make([]cle, 0, len(n))
	for k := range n {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool { return fmt.Sprint(cles[i]) < fmt.Sprint(cles[j]) })
	for _, k := range cles {
		t.Logf("   classe=%-8s arme=%08X etat=%s : %d", k.classe, k.arme, k.etat, n[k])
	}
	for _, l := range lignes {
		t.Logf("     %s", l)
	}
}

// tcgS2Episodes publie (b) : par episode, la part du temps en etat ouvert, les rafales, et l etat
// tenu aux touches du pilote.
func tcgS2Episodes(t *testing.T, p *tcgPasse) {
	t.Helper()
	par := tcgParIndex(p)
	t.Logf("== S2.14 PAR EPISODE : temps en etat O / - / ? (pas de 100 ms), et etat tenu aux touches " +
		"du pilote (damage_aftermath, ref1 = son corps)")
	for _, ep := range p.cad.episodes {
		cpt := map[string]int{}
		for tr := ep.t0; tr <= ep.t1; tr++ {
			ts := p.cad.origineUS + uint64(tr)*tcgPasUS //nolint:gosec // trame >= 0 dans un episode
			e, _ := tcgEtat(par, ep.index, ts)
			cpt[e]++
		}
		touches := map[string]int{}
		var ages []string
		for _, e := range p.evs {
			if e.degat == nil || !e.refs[1].present || e.refs[1].slot() != ep.pilote ||
				e.trame < ep.t0-10 || e.trame > ep.t1+10 {
				continue
			}
			etat, age := tcgEtat(par, ep.index, e.ts)
			touches[etat]++
			if len(ages) < 40 {
				ages = append(ages, fmt.Sprintf("%d:%s(%dms)%08X>%d", e.trame, etat, age, e.degat.sourceID,
					e.refs[0].slot()))
			}
		}
		t.Logf("   episode t %d..%d pilote %d vehicule %d index %d : etats %v · touches du pilote par etat "+
			"%v", ep.t0, ep.t1, ep.pilote, ep.vehicule, ep.index, cpt, touches)
		if len(ages) > 0 {
			t.Logf("      touches (t:etat source) : %s", strings.Join(ages, " "))
		}
	}
}

// tcgS2Silences publie, pour les index de controle 0 a 15, les SILENCES de plus de 3 s entre deux
// entrees (bornes sur l axe du document) et la premiere / derniere entree : c est ce qui permet de
// ponter un index de controle a un joueur (un mort ne controle rien), sans rien supposer.
func tcgS2Silences(t *testing.T, p *tcgPasse) {
	t.Helper()
	par := tcgParIndex(p)
	t.Logf("== S2.15 SILENCES > 3 s des entrees de controle, par index (t0-t1 sur l axe du document)")
	for ix := 0; ix < 16; ix++ {
		es := par[ix]
		if len(es) < 20 {
			continue
		}
		var s []string
		for i := 1; i < len(es); i++ {
			if es[i].ts-es[i-1].ts > 3_000_000 {
				s = append(s, fmt.Sprintf("%d-%d", es[i-1].trame, es[i].trame))
			}
		}
		t.Logf("   index %2d : %d entrees, t %d..%d, silences %s", ix, len(es), es[0].trame,
			es[len(es)-1].trame, strings.Join(s, " "))
	}
}
