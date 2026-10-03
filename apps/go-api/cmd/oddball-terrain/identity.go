package main

// identity.go — PONT slot -> xuid, PAR MANCHE.
//
// DEUX pieges du multi-manche, tous deux mesures sur d9781168 (3 manches) :
//
//  1. Le compteur de morts (comp 2 B) REPART DE ZERO a chaque manche (indexe `manche*4` dans le
//     getter natif). Un deroulage monotone sur tout le match ne voit que la manche 0.
//  2. PLUS PROFOND : le SLOT D'ENTITE est REATTRIBUE a un AUTRE joueur d'une manche a l'autre.
//     Mesure : les instants de mort du slot 22 valent scuderiasven en manche 0
//     (120925,139494,159148,174913 <-> 120926,139497,159150,174916) puis LadyJezz en manches 1-2
//     (14 instants <-> 245200,303775,...,698184). Un slot n'est donc PAS une identite stable ;
//     l'identite se resout PAR MANCHE.
//
// La resolution : par manche, on prend les instants de mort du slot (increments de comp 2 B de
// cette manche) et on les apparie au fil des morts du film (complet, horloge du match) par un
// appariement BIJECTIF GLOBAL (meilleure paire d'abord). Le fil coincide a ~3 ms avec le compteur.

import "sort"

const (
	identTolMS    = 150 // fenetre d'appariement (bruit d'horloge)
	identMinMatch = 3   // minimum de coincidences pour retenir une paire
)

// roundIdentities resout slot -> xuid pour CHAQUE manche reelle du film.
func roundIdentities(d *filmDump) map[int]map[int]string {
	out := map[int]map[int]string{}
	for _, round := range d.rounds {
		out[round] = resolveIdentityRound(d, round)
	}
	return out
}

// resolveIdentityRound resout slot -> xuid pour UNE manche, par appariement bijectif des instants
// de mort de cette manche au fil des morts du film.
func resolveIdentityRound(d *filmDump, round int) map[int]string {
	type pair struct {
		slot int
		xuid uint64
		co   int
	}
	slotDeaths := deathInstantsForRound(d, round)
	var pairs []pair
	for slot, inst := range slotDeaths {
		for xuid, fil := range d.deaths {
			if co := coincidences(inst, fil); co >= identMinMatch {
				pairs = append(pairs, pair{slot: slot, xuid: xuid, co: co})
			}
		}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].co > pairs[j].co })
	usedSlot, usedXUID := map[int]bool{}, map[uint64]bool{}
	out := map[int]string{}
	for _, p := range pairs {
		if usedSlot[p.slot] || usedXUID[p.xuid] {
			continue
		}
		out[p.slot], usedSlot[p.slot], usedXUID[p.xuid] = u64s(p.xuid), true, true
	}
	return out
}

// deathInstantsForRound rend, par slot, les instants de mort d'UNE manche (increments de comp 2 B
// de cette manche ; l'instant est absolu sur l'horloge du match, seule la valeur repart de zero).
func deathInstantsForRound(d *filmDump, round int) map[int][]int {
	key := emplKey{comp: 2, side: "B"}
	out := map[int][]int{}
	for slot, byRound := range d.series {
		if inst := incrementInstants(byRound[round][key]); len(inst) > 0 {
			out[slot] = inst
		}
	}
	return out
}

// identityDiagRound rend, par slot, les deux meilleurs candidats xuid par coincidences de mort de
// la manche `round` : de quoi juger l'appariement sur pieces.
func identityDiagRound(d *filmDump, round int) string {
	slotDeaths := deathInstantsForRound(d, round)
	slots := make([]int, 0, len(slotDeaths))
	for s := range slotDeaths {
		slots = append(slots, s)
	}
	sort.Ints(slots)
	out := "DIAGNOSTIC identite manche " + u64s(uint64(round)) +
		" (coincidences de mort par slot ; deux meilleurs) :\n"
	for _, slot := range slots {
		best, bestX, second, secX := 0, uint64(0), 0, uint64(0)
		for xuid, fil := range d.deaths {
			co := coincidences(slotDeaths[slot], fil)
			if co > best {
				second, secX, best, bestX = best, bestX, co, xuid
			} else if co > second {
				second, secX = co, xuid
			}
		}
		out += "  slot " + u64s(uint64(slot)) + " : " + gamertagOf(d, u64s(bestX)) + "=" +
			u64s(uint64(best)) + "  (2e " + gamertagOf(d, u64s(secX)) + "=" + u64s(uint64(second)) + ")\n"
	}
	return out
}

// coincidences compte les instants appariables entre deux series (appariement glouton croissant,
// chaque instant du fil ne servant qu'une fois).
func coincidences(a, b []int) int {
	used := make([]bool, len(b))
	n := 0
	for _, t := range a {
		best, bd := -1, identTolMS+1
		for i, u := range b {
			if used[i] {
				continue
			}
			if dd := abs(u - t); dd < bd {
				bd, best = dd, i
			}
		}
		if best >= 0 {
			used[best] = true
			n++
		}
	}
	return n
}

func u64s(v uint64) string {
	const digits = "0123456789"
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = digits[v%10]
		v /= 10
	}
	return string(buf[i:])
}
