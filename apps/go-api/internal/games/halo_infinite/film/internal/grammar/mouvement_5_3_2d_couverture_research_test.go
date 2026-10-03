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

// m532dEch relit les champs suivis aux positions que la trame publie.
func m532dEch(pay []byte, r FrameRecord, ts uint64) m532Ech {
	e := m532Ech{slot: r.Slot, tUS: ts, masque: r.Trace.Mask}
	total := len(pay) * 8
	e.marcheComplete = r.Trace.DesyncAt == -1
	for i, comp := range r.Trace.Comps {
		fin := r.Trace.EndBit
		if i+1 < len(r.Trace.Comps) {
			fin = r.Trace.Comps[i+1].StartBit
		}
		if comp.StartBit < 0 || fin > total || fin < comp.StartBit {
			continue
		}
		m532Champ(pay, comp, fin, &e)
	}
	return e
}

// m532dHistoEchecs — L HISTOGRAMME DES ECHECS, ET LA COMPARAISON QUI DECIDE.
//
// LA QUESTION : un delta ne porte `i29` QUE quand l accroupi CHANGE. Ces records sont donc
// rares. S ils sont AUSSI ceux sur lesquels la marche desynchronise, alors l accroupi par
// instant est bien dans la trame, derriere une largeur fausse — et le composant fautif se
// nomme. Sinon, il n y est pas.
//
// LA COMPARAISON EST UN RAPPORT, PAS UN COMPTE : la part des masques portant `i29` parmi les
// records EN ECHEC, contre cette meme part parmi les records SAINS. Un compte seul ne dirait
// rien, les deux populations n ayant pas la meme taille.
func m532dHistoEchecs(t *testing.T, sains []m532Ech, echecs []m532dEchec, reg *Registry) {
	t.Helper()
	suivis := []struct {
		nom string
		idx uint
	}{
		{"i18 unit-control", 18},
		{"i29 unit-crouch", 29},
		{"i54 biped-mobility-action", 54},
		{"i55 biped-posture-physics", 55},
		{"i62 biped-slide", 62},
	}
	var ech35 []m532dEchec
	parTI := map[uint32]int{}
	rejets := 0
	var reels []m532dEchec
	for _, e := range echecs {
		// LA DISCRIMINATION QUI MANQUAIT : un rejet de generation n est pas un composant fautif.
		if e.ti == 0 && e.fautif == 0 && e.composants == 0 {
			rejets++
			continue
		}
		reels = append(reels, e)
		parTI[e.ti]++
		if e.ti == BipedTypeIndex {
			ech35 = append(ech35, e)
		}
	}
	t.Logf("ECHECS, VENTILES PAR NATURE : %d rejets de GENERATION (monde incomplet, aucun "+
		"composant lu) · %d desynchronisations REELLES de grammaire", rejets, len(reels))
	m532dCouvertureMonde(t, echecs, sains)
	echecs = reels
	t.Logf("ECHECS : %d records desynchronises, dont %d de ti=35 (%.1f %%)",
		len(echecs), len(ech35), m532Pct(len(ech35), len(echecs)))
	tis := make([]int, 0, len(parTI))
	for k := range parTI {
		tis = append(tis, int(k)) //nolint:gosec // index d archetype
	}
	sort.Slice(tis, func(a, b int) bool { return parTI[uint32(tis[a])] > parTI[uint32(tis[b])] }) //nolint:gosec // clefs
	var parts []string
	for i, ti := range tis {
		if i >= 6 {
			break
		}
		parts = append(parts, fmt.Sprintf("ti=%d:%d", ti, parTI[uint32(ti)])) //nolint:gosec // clef
	}
	t.Logf("    par archetype : %s", strings.Join(parts, " "))
	// LE COMPOSANT FAUTIF DES TROIS ARCHETYPES QUI ECHOUENT LE PLUS : quand la trame casse sur
	// un archetype, LE RESTE DU PAQUET EST PERDU — donc les records bipedes riches, qui viennent
	// apres, ne sont jamais atteints. Nommer ces composants, c est nommer le verrou.
	for i, ti := range tis {
		if i >= 3 {
			break
		}
		f := map[int]int{}
		for _, e := range echecs {
			if int(e.ti) == ti { //nolint:gosec // index d archetype
				f[e.fautif]++
			}
		}
		ks := make([]int, 0, len(f))
		for k := range f {
			ks = append(ks, k)
		}
		sort.Slice(ks, func(a, b int) bool { return f[ks[a]] > f[ks[b]] })
		var pp []string
		for j, k := range ks {
			if j >= 5 {
				break
			}
			pp = append(pp, fmt.Sprintf("%s:%d", nomComposantBloquant(reg, ti, k), f[k]))
		}
		t.Logf("      ti=%-2d fautifs : %s", ti, strings.Join(pp, " "))
	}
	if len(ech35) == 0 {
		t.Logf("    AUCUN echec sur ti=35 : la desynchronisation ne vient pas du bipede")
		return
	}
	fautifs := map[int]int{}
	for _, e := range ech35 {
		fautifs[e.fautif]++
	}
	cles := make([]int, 0, len(fautifs))
	for k := range fautifs {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(a, b int) bool { return fautifs[cles[a]] > fautifs[cles[b]] })
	parts = parts[:0]
	for i, k := range cles {
		if i >= 8 {
			break
		}
		parts = append(parts, fmt.Sprintf("%s:%d", nomComposantBloquant(reg, BipedTypeIndex, k), fautifs[k]))
	}
	t.Logf("    COMPOSANT FAUTIF sur ti=35 (les 8 premiers) : %s", strings.Join(parts, " "))
	t.Logf("    SUR-REPRESENTATION DES MASQUES (part parmi les ECHECS ti=35 contre part parmi les SAINS) :")
	for _, s := range suivis {
		var e, sa int
		for _, x := range ech35 {
			if x.masque&(uint64(1)<<s.idx) != 0 {
				e++
			}
		}
		for _, x := range sains {
			if x.masque&(uint64(1)<<s.idx) != 0 {
				sa++
			}
		}
		pe, ps := m532Pct(e, len(ech35)), m532Pct(sa, len(sains))
		verdict := ""
		switch {
		case e == 0 && sa == 0:
			verdict = "  ABSENT DES DEUX POPULATIONS"
		case ps > 0 && pe > ps*3 && e >= 20:
			verdict = "  <-- SUR-REPRESENTE : piste de grammaire"
		case e == 0:
			verdict = "  jamais dans un masque en echec"
		}
		t.Logf("      %-30s echecs %5d (%5.2f %%) · sains %6d (%5.2f %%)%s",
			s.nom, e, pe, sa, ps, verdict)
	}
}
