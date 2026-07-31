package main

import (
	"fmt"

	"levelup/go-api/internal/analysis"
)

// dumpPairAnatomy walks the bits between a group's primary and secondary literal
// to reveal the exact slot structure (3 consecutive WST: primary, an intermediate
// entity-handle slot, secondary). It prints each candidate WST (gate=1 + 64 bits)
// in the window [prim-2, sec+64].
func dumpPairAnatomy(payload []byte, groups [][]weaponLit) {
	fmt.Printf("\n================ ANATOMIE des paires (slots WST consécutifs) ================\n")
	for gi, g := range groups {
		if len(g) < 2 {
			continue
		}
		lo := g[0].bit - 2
		hi := g[1].bit + 2
		fmt.Printf("\n  record %d : prim %s @%d ... sec %s @%d (fenêtre %d..%d)\n",
			gi, g[0].name, g[0].bit, g[1].name, g[1].bit, lo, hi)
		// Scan for gate=1 positions whose following 64 bits form (handle,variant).
		for bp := lo; bp <= hi; bp++ {
			if bitsAt(payload, bp, 1) != 1 {
				continue
			}
			h := uint32(bitsAt(payload, bp+1, 32))
			v := uint32(bitsAt(payload, bp+33, 32))
			id := (uint64(h) << 32) | uint64(v)
			nm, ok := analysis.WeaponIDToName[id]
			tag := ""
			if ok {
				tag = "  <== ARME " + nm
			} else if v == 0x42c9679f {
				tag = "  (suffixe canonique mais id inconnu)"
			}
			// Only print "interesting" gate hits: catalogued, or canonical suffix, or
			// the slot-counter handle pattern 0xNNNN NNeb.
			if ok || v == 0x42c9679f || (h&0xff) == 0xeb {
				fmt.Printf("    gate@%-7d handle=0x%08x variant=0x%08x id64=0x%016x%s\n", bp, h, v, id, tag)
			}
		}
	}
}
