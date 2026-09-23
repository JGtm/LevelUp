//go:build research

package grammar

// p2_marche_imagecle_variantes_research_test.go — SONDE P2, variantes d'ELECTION du balayeur
// (prototypes de MESURE, jamais de production). Le balayeur de production elit, faute de voisin
// immediat, le candidat de plus basse generation PUIS DE PLUS BAS SLOT dans une fenetre de
// 120 000 bits : une fausse ancre de slot bas situee loin devant bat la vraie ancre proche. Les
// variantes gardent tout le reste (voisin immediat, saut de largeur, fenetre) et ne changent que
// cette regle :
//
//	V2  generation 1 d'abord, puis le PLUS PROCHE (bit le plus bas) ;
//	V4  V2 SANS fenetre (le jeu n en a pas) ;
//	V3  le plus proche des candidats FORTEMENT CREDIBLES (`n1` a +108 egal au `n1` modal NON NUL
//	    de son archetype, vu au moins trois fois dans le payload) ; a defaut, la regle de
//	    production.

import (
	"fmt"
	"testing"
)

// p2CandN est un candidat avec son archetype et son n1.
type p2CandN struct {
	bit, slot, gen, ti int
	n1                 uint64
}

// p2Election choisit parmi les candidats (ordre des bits) ; -1 = aucun.
type p2Election func(c []p2CandN) int

// p2Candidats rend le voisin immediat eventuel (retour anticipe comme en production) et sinon
// tous les candidats valides de la fenetre.
func p2Candidats(pay []byte, from, prev, total, maxWin int) (voisin int, c []p2CandN) {
	end := from + maxWin
	if end > total {
		end = total
	}
	sent := 0
	for q := from; q+64 <= end; q++ {
		id := kfReadBits(pay, q, 32)
		if id == kfSent {
			if sent++; sent >= 2048 {
				break
			}
			continue
		}
		sent = 0
		s, ti, g, ok := kfAnchorFromID(pay, q, id, prev, total)
		if !ok {
			continue
		}
		if s == prev+1 && g == 1 {
			return q, nil
		}
		c = append(c, p2CandN{bit: q, slot: s, gen: g, ti: ti, n1: kfReadBits(pay, q+108, 32)})
	}
	return -1, c
}

// p2ElectionProd est la regle de production (`betterThan`).
func p2ElectionProd(prev int) p2Election {
	return func(c []p2CandN) int {
		at, best := -1, kfCand{}
		for _, x := range c {
			k := kfCand{gen: x.gen, slot: x.slot, bit: x.bit}
			if x.slot == prev+1 {
				k.consecutive = 1
			}
			if at < 0 || k.betterThan(best) {
				at, best = x.bit, k
			}
		}
		return at
	}
}

// p2ElectionV2 : generation 1 d'abord, puis le plus proche.
func p2ElectionV2(prev int) p2Election {
	return func(c []p2CandN) int {
		for _, x := range c {
			if x.gen == 1 {
				return x.bit
			}
		}
		return p2ElectionProd(prev)(c)
	}
}

// p2ElectionV3 : le plus proche des candidats fortement credibles, sinon la production.
func p2ElectionV3(prev int, modal map[int]uint64) p2Election {
	return func(c []p2CandN) int {
		for _, x := range c {
			if m := modal[x.ti]; m != 0 && x.n1 == m {
				return x.bit
			}
		}
		return p2ElectionProd(prev)(c)
	}
}

// p2ModalN1 rend, par archetype, le n1 modal NON NUL vu au moins trois fois (ombres d'un bit
// ecartees).
func p2ModalN1(pay []byte) map[int]uint64 {
	total := len(pay) * 8
	par := map[int]map[uint64]int{}
	vu := map[int]bool{}
	for q := 0; q+172 <= total; q++ {
		arch := kfReadBits(pay, q+32, 32)
		id := kfReadBits(pay, q, 32)
		if arch >= kfArchMax || id>>30 == 0 || int(id&0x3FFFFFFF) >= kfTableCap || vu[q-1] {
			continue
		}
		vu[q] = true
		ti := int(arch)
		if par[ti] == nil {
			par[ti] = map[uint64]int{}
		}
		par[ti][kfReadBits(pay, q+108, 32)]++
	}
	out := map[int]uint64{}
	for ti, m := range par {
		var best uint64
		for v, n := range m {
			if v != 0 && n >= 3 && n > m[best] {
				best = v
			}
		}
		out[ti] = best
	}
	return out
}

// p2WalkVariante est le balayeur de production avec une regle d'election injectee.
func p2WalkVariante(pay []byte, elire func(prev int) p2Election, sansFenetre bool) (recs []KeyframeRec, elections int) {
	total, maxWin := len(pay)*8, kfScanFenetreBits
	if sansFenetre {
		maxWin = total
	}
	width, seen := map[int]int{}, map[int]int{}
	next := func(from, prev int) int {
		v, c := p2Candidats(pay, from, prev, total, maxWin)
		if v >= 0 {
			return v
		}
		if len(c) > 0 {
			elections++
		}
		return elire(prev)(c)
	}
	pos, prev := 1, -1
	if _, _, _, ok := kfValidAnchor(pay, pos, prev, total); !ok {
		pos = next(pos, prev)
	}
	for pos >= 0 {
		slot, ti, gen, ok := kfValidAnchor(pay, pos, prev, total)
		if !ok {
			break
		}
		st := pos + 64
		var nat int
		w, has := width[ti]
		if _, _, jg, vok := kfValidAnchor(pay, st+w, slot, total); has && vok && jg == 1 {
			nat = st + w
		} else {
			nat = next(st, slot)
		}
		recs = append(recs, KeyframeRec{Slot: slot, TI: ti, Gen: gen, Bit: pos})
		prev = slot
		if nat < 0 {
			break
		}
		nw := nat - st
		if pw, s := seen[ti]; s {
			if pw == nw {
				width[ti] = nw
			} else {
				delete(width, ti)
			}
		} else {
			seen[ti] = nw
		}
		pos = nat
	}
	return recs, elections
}

// p2Variantes publie, pour chaque regle, ancres, bipedes atteints et ancres de production
// perdues / gagnees. La regle de production rejouee doit egaler `WalkKeyframeWorld`
// (etalonnage).
func p2Variantes(t *testing.T, pay []byte, bip []p2Entete, prod []KeyframeRec) {
	t.Helper()
	modal := p2ModalN1(pay)
	regles := []struct {
		nom         string
		f           func(prev int) p2Election
		sansFenetre bool
	}{
		{"PROD", p2ElectionProd, false},
		{"V2", p2ElectionV2, false},
		{"V3", func(prev int) p2Election { return p2ElectionV3(prev, modal) }, false},
		{"V4", p2ElectionV2, true},
	}
	enProd := map[int]bool{}
	for _, r := range prod {
		enProd[r.Bit] = true
	}
	for _, r := range regles {
		recs, el := p2WalkVariante(pay, r.f, r.sansFenetre)
		if r.nom == "PROD" && fmt.Sprint(recs) != fmt.Sprint(prod) {
			t.Fatalf("ETALONNAGE variantes : PROD rejouee (%d) != WalkKeyframeWorld (%d)", len(recs), len(prod))
		}
		at := map[int]bool{}
		var gagnees, bipTI int
		for _, x := range recs {
			at[x.Bit] = true
			if !enProd[x.Bit] {
				gagnees++
			}
			if x.TI == p2BipedeTI {
				bipTI++
			}
		}
		var perdues, nb int
		var lp []string
		for _, x := range prod {
			if !at[x.Bit] {
				perdues++
				if len(lp) < 12 {
					lp = append(lp, fmt.Sprintf("%d/ti%d@%d", x.Slot, x.TI, x.Bit))
				}
			}
		}
		if len(lp) > 0 {
			t.Logf("V. %s perd : %v", r.nom, lp)
		}
		for _, e := range bip {
			if at[e.bit] {
				nb++
			}
		}
		t.Logf("V. %-4s : %4d ancres (%d elections), ancres de production perdues %d, gagnees %d ; bipedes exacts atteints %d / %d (ancres ti35 %d)",
			r.nom, len(recs), el, perdues, gagnees, nb, len(bip), bipTI)
	}
	t.Logf("V. n1 modal du bipede (ti 35) : %d", modal[p2BipedeTI])
}
