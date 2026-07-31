package main

import (
	"fmt"

	"levelup/go-api/internal/analysis"
)

// buildFinalTable assembles the per-record loadout table directly from the WST
// literals (each proven gate=1/handle=high32/variant=low32). For each group it
// reads the primary (1st literal) and secondary (2nd literal) catalogued weapon,
// and the intermediate entity-slot handle between them (the 0x1792NNeb / 0x3f92NNeb
// counter that increments +2 per record — a candidate player-slot key).
func buildFinalTable(payload []byte, groups [][]weaponLit) {
	fmt.Printf("\n================ TABLE FINALE record -> {primaire, secondaire} ================\n")
	fmt.Printf("%-8s %-16s %-16s %-12s %-12s\n", "record", "PRIMAIRE", "SECONDAIRE", "slotHandle", "(NN)")
	for gi, g := range groups {
		prim := weaponAt(payload, g[0].bit)
		sec := ""
		var slotHandle uint32
		var nn uint32
		if len(g) >= 2 {
			sec = weaponAt(payload, g[1].bit)
			// The intermediate WST handle sits 32 bits before the 2nd literal's handle
			// (it is the variant field's predecessor). Empirically handle@(litBit2-32).
			slotHandle = uint32(bitsAt(payload, g[1].bit-32, 32))
			// Extract NN from 0xHHHH_NN_eb pattern.
			nn = (slotHandle >> 8) & 0xff
		}
		fmt.Printf("%-8d %-16s %-16s 0x%08x   0x%02x\n", gi, prim, sec, slotHandle, nn)
	}

	// Inspect the slot handles in order, and the primary-handle high bytes, to see
	// if any monotonic counter aligns 1:1 with player index 0..7.
	fmt.Printf("\n--- Handles d'entité (primaire) par record, pour déceler un index joueur ---\n")
	for gi, g := range groups {
		ph := uint32(bitsAt(payload, g[0].bit, 32)) // primary handle (= weapon family high32, NOT entity)
		// The entity/object handle is elsewhere; print the obje variant if we can.
		fmt.Printf("  record %d : primHandle(famille)=0x%08x  prim=%s\n", gi, ph, weaponAt(payload, g[0].bit))
	}
}

// weaponAt reads a WST literal at litBit (handle=high32@litBit, variant=low32@litBit+32)
// and returns the catalogued name (or a ?hex tag).
func weaponAt(payload []byte, litBit int) string {
	h := uint32(bitsAt(payload, litBit, 32))
	v := uint32(bitsAt(payload, litBit+32, 32))
	id := (uint64(h) << 32) | uint64(v)
	if nm, ok := analysis.WeaponIDToName[id]; ok {
		return nm
	}
	return fmt.Sprintf("?0x%016x", id)
}
