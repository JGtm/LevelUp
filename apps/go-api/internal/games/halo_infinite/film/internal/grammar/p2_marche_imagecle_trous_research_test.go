//go:build research

package grammar

// p2_marche_imagecle_trous_research_test.go — SONDE P2, suite : les trous enjambes par le balayeur
// (E), un prototype de balayeur (F) et le recensement des en-tetes credibles (G). Voir
// p2_marche_imagecle_research_test.go.

import (
	"fmt"
	"sort"
	"testing"
)

// ---------------------------------------------------------------------------------------------
// E. LES TROUS ENJAMBES SONT-ILS DES SUITES D'ENTREES SANS ARCHETYPE ?
//
// Une entree sans archetype fait 108 bits (l'ecrivain saute tailles et corps). Si les trous du
// balayeur sont remplis de telles entrees, on doit y lire des en-tetes exacts
// `[(gen<<30)|slot][0xFFFFFFFF]` enchaines tous les 108 bits, slot +1 a chaque pas, et la suite
// doit finir EXACTEMENT sur l'ancre vraie suivante.

// p2SuiteSA est une suite d'entrees sans archetype enchainees tous les 108 bits.
type p2SuiteSA struct {
	bit, fin, slot0, slotN, n int
	gens                      map[int]int
	sautsSlot                 int // pas ou le slot ne vaut pas le precedent + 1
}

// p2SuitesSA rend les suites SA maximales qui commencent dans [de, a).
func p2SuitesSA(pay []byte, de, a, pas int) []p2SuiteSA {
	total := len(pay) * 8
	var out []p2SuiteSA
	for q := de; q < a && q+64 <= total; q++ {
		h, ok := readKeyframeHeader(pay, q, total)
		if !ok || !h.SansArchetype() {
			continue
		}
		s := p2SuiteSA{bit: q, slot0: h.Slot, gens: map[int]int{}}
		prev, pos := h.Slot-1, q
		for ok && h.SansArchetype() && h.Slot > prev {
			if h.Slot != prev+1 {
				s.sautsSlot++
			}
			s.n++
			s.gens[h.Gen]++
			s.slotN, prev, pos = h.Slot, h.Slot, pos+pas
			h, ok = readKeyframeHeader(pay, pos, total)
		}
		s.fin = pos
		if s.n >= 2 {
			out = append(out, s)
			q = pos - 1
		}
	}
	return out
}

// p2Trous publie, pour le depart et chaque election, les suites SA du trou enjambe et si la
// derniere finit sur une ancre (en-tete a archetype valide dont le slot suit).
func p2Trous(t *testing.T, fc *FilmContext, pay []byte, pas []p2Pas) {
	t.Helper()
	sa := keyframeFullHeaderBits(fc.ContexteDeLecture())
	total := len(pay) * 8
	var nSA int
	for i, p := range pas {
		if p.scan == nil || p.scan.rapide || p.nat < 0 {
			continue
		}
		for _, s := range p2SuitesSA(pay, p.pos+64, p.nat, sa) {
			nSA += s.n
			h, ok := readKeyframeHeader(pay, s.fin, total)
			t.Logf("E. pas %d (slot %d) : suite SA bit %d-%d slots %d..%d (%d entrees, gens %v, sauts de slot %d) -> apres : ok=%v slot %d arch %#x",
				i, p.slot, s.bit, s.fin, s.slot0, s.slotN, s.n, s.gens, s.sautsSlot, ok, h.Slot, h.Archetype)
		}
	}
	t.Logf("E. entrees sans archetype dans les trous des elections : %d", nSA)
}

// p2WalkSA est un PROTOTYPE (non production) : le balayeur de production, plus deux regles de
// l'ecrivain — une entree sans archetype de slot prev+1 est un voisin immediat, et elle
// s'enchaine a +108 bits sans corps. Rend les ancres a archetype, le nombre d'entrees SA
// enchainees et le nombre d'elections restantes.
func p2WalkSA(pay []byte, pasSA int) (recs []KeyframeRec, nSA, elections int) {
	total, maxWin := len(pay)*8, kfScanFenetreBits
	pos, prev := 1, -1
	for pos >= 0 && pos+64 <= total {
		h, ok := readKeyframeHeader(pay, pos, total)
		if !ok || h.Slot <= prev {
			// Apres une suite SA, l'en-tete attendu est illisible : UN rebalayage, compte.
			nat, _ := p2ScanSA(pay, pos, prev, total, maxWin)
			if nat <= pos {
				break
			}
			elections++
			pos = nat
			continue
		}
		if h.SansArchetype() {
			nSA++
			prev, pos = h.Slot, pos+pasSA
			continue
		}
		recs = append(recs, KeyframeRec{Slot: h.Slot, TI: h.TI, Gen: h.Gen, Bit: pos})
		prev = h.Slot
		nat, elu := p2ScanSA(pay, pos+64, prev, total, maxWin)
		if elu {
			elections++
		}
		pos = nat
	}
	return recs, nSA, elections
}

// p2ScanSA : `kfScanNext` ou un en-tete SA de slot prev+1 compte comme voisin immediat.
func p2ScanSA(pay []byte, from, prev, total, maxWin int) (int, bool) {
	for q := from; q+64 <= total && q < from+maxWin; q++ {
		h, ok := readKeyframeHeader(pay, q, total)
		if ok && h.SansArchetype() && h.Slot == prev+1 {
			return q, false
		}
		if ok && !h.SansArchetype() && h.Slot == prev+1 && h.Gen == 1 && kfReadBits(pay, q+32, 32) < kfArchMax {
			return q, false
		}
	}
	s := p2ScanNext(pay, from, prev, total, maxWin)
	return s.at, s.at >= 0
}

// p2Prototype publie ce que le prototype SA atteint, contre la production.
func p2Prototype(t *testing.T, fc *FilmContext, pay []byte, bip []p2Entete, prod []KeyframeRec) {
	t.Helper()
	recs, nSA, el := p2WalkSA(pay, keyframeFullHeaderBits(fc.ContexteDeLecture()))
	at := map[int]bool{}
	for _, r := range recs {
		at[r.Bit] = true
	}
	var nb int
	for _, e := range bip {
		if at[e.bit] {
			nb++
		}
	}
	var perdus int
	for _, r := range prod {
		if !at[r.Bit] {
			perdus++
		}
	}
	t.Logf("F. PROTOTYPE SA : %d ancres (production %d, dont %d absentes du prototype), %d entrees SA enchainees, %d elections, bipedes %d / %d",
		len(recs), len(prod), perdus, nSA, el, nb, len(bip))
}

// ---------------------------------------------------------------------------------------------
// G. L'ORDRE REEL DE LA TABLE : tous les en-tetes CREDIBLES, sans filtre de slot precedent.
//
// Credible = `[(gen<<30)|slot][ti<50]` dont le `n1` (taille de tampon a +108, constante par
// archetype et par build) vaut le `n1` MODAL non nul de son archetype, ce modal etant vu au
// moins trois fois. Aucune hypothese d'ordre : c'est ce qui permet de voir si la table est a
// slots croissants.

type p2Credible struct {
	bit, slot, gen, ti int
	ancre              bool
}

func p2Credibles(pay []byte, anc map[int]bool) []p2Credible {
	total := len(pay) * 8
	type brut struct {
		p2Credible
		n1 uint64
	}
	var tous []brut
	parTI := map[int]map[uint64]int{}
	for q := 0; q+172 <= total; q++ {
		arch := kfReadBits(pay, q+32, 32)
		if arch >= kfArchMax {
			continue
		}
		id := kfReadBits(pay, q, 32)
		gen, slot := int(id>>30), int(id&0x3FFFFFFF)
		if gen == 0 || slot >= kfTableCap {
			continue
		}
		n1 := kfReadBits(pay, q+108, 32)
		b := brut{p2Credible{bit: q, slot: slot, gen: gen, ti: int(arch), ancre: anc[q]}, n1}
		tous = append(tous, b)
		if parTI[b.ti] == nil {
			parTI[b.ti] = map[uint64]int{}
		}
		parTI[b.ti][n1]++
	}
	modal := map[int]uint64{}
	for ti, m := range parTI {
		var best uint64
		for v, n := range m {
			if v != 0 && n >= 3 && n > m[best] {
				best = v
			}
		}
		modal[ti] = best
	}
	var out []p2Credible
	vrai := map[int]bool{}
	for _, b := range tous {
		if b.n1 != 0 && b.n1 == modal[b.ti] && !vrai[b.bit-1] {
			vrai[b.bit] = true
			out = append(out, b.p2Credible)
		}
	}
	return out
}

// p2Ordre publie les en-tetes credibles, les inversions de slot et ceux que le balayeur manque.
func p2Ordre(t *testing.T, pay []byte, recs []KeyframeRec, de, a int) {
	t.Helper()
	anc := map[int]bool{}
	for _, r := range recs {
		anc[r.Bit] = true
	}
	cr := p2Credibles(pay, anc)
	sort.Slice(cr, func(i, j int) bool { return cr[i].bit < cr[j].bit })
	var nAnc, inv int
	var manques []string
	for i, c := range cr {
		if c.ancre {
			nAnc++
		} else if len(manques) < 80 {
			manques = append(manques, fmt.Sprintf("%d/ti%d@%d", c.slot, c.ti, c.bit))
		}
		if i > 0 && c.slot <= cr[i-1].slot {
			inv++
			if inv <= 20 {
				t.Logf("G. INVERSION : slot %d (ti %d, bit %d) apres slot %d (ti %d, bit %d)",
					c.slot, c.ti, c.bit, cr[i-1].slot, cr[i-1].ti, cr[i-1].bit)
			}
		}
		if c.bit >= de && c.bit < a {
			t.Logf("G.   bit %8d slot %4d gen %d ti %2d ancre=%v", c.bit, c.slot, c.gen, c.ti, c.ancre)
		}
	}
	t.Logf("G. %d en-tetes credibles, %d ancres du balayeur, %d inversions de slot ; manques (%d premiers) : %v",
		len(cr), nAnc, inv, len(manques), manques)
}

// p2Voisinage publie les pas du balayeur d'index [de, a) : ancre, largeur jusqu'a l'ancre
// suivante, n1 et n2 presumes, et le profil de composants de l'archetype.
func p2Voisinage(t *testing.T, reg *Registry, pay []byte, pas []p2Pas, de, a int) {
	t.Helper()
	for i := de; i < a && i < len(pas); i++ {
		p := pas[i]
		nc, prem := 0, ""
		if arch, ok := reg.Archetype(p.ti); ok {
			nc = len(arch.Components)
			if nc > 0 {
				prem = arch.Components[0]
			}
		}
		t.Logf("H. pas %3d bit %7d slot %4d gen %d ti %2d (%d composants, 1er %q) -> %d (+%d bits, %s) n1 %d",
			i, p.pos, p.slot, p.gen, p.ti, nc, prem, p.nat, p.nat-p.pos, p.methode, kfReadBits(pay, p.pos+108, 32))
	}
}

// p2Fermeture publie, pour les ancres des pas [de, a), ou l'etat complet ferme : EXACTEMENT sur
// l'ancre suivante (preuve forte que les deux sont vraies), ailleurs, ou sur un composant non
// porte. Puis les en-tetes BRUTS (sans filtre de n1 ni de slot precedent, ombres d'un bit
// ecartees) de la fenetre [bruitDe, bruitA).
func p2Fermeture(t *testing.T, ctx ContexteDeLecture, reg *Registry, pay []byte, pas []p2Pas,
	de, a, bruitDe, bruitA int) {
	t.Helper()
	total := len(pay) * 8
	for i := de; i < a && i < len(pas); i++ {
		p := pas[i]
		tr := WalkKeyframeFullState(pay, p.pos, reg, ctx)
		h, ok := readKeyframeHeader(pay, tr.EndBit, total)
		t.Logf("J. pas %3d slot %4d ti %2d bit %7d : fin %7d (ancre suivante %7d, exacte=%v) desync %d ; en-tete a la fin ok=%v slot %d arch %#x",
			i, p.slot, p.ti, p.pos, tr.EndBit, p.nat, tr.EndBit == p.nat, tr.DesyncAt, ok, h.Slot, h.Archetype)
	}
	prec := -2
	for q := bruitDe; q < bruitA && q+64 <= total; q++ {
		h, ok := readKeyframeHeader(pay, q, total)
		if !ok {
			continue
		}
		if q != prec+1 {
			t.Logf("K. en-tete brut bit %7d slot %4d gen %d arch %#x n1 %d", q, h.Slot, h.Gen, h.Archetype,
				kfReadBits(pay, q+108, 32))
		}
		prec = q
	}
}
