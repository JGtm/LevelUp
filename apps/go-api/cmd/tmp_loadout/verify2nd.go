package main

import (
	"fmt"

	"levelup/go-api/internal/analysis"
)

// verifySecondLiterals checks, for each group's 2nd literal, whether it is a
// genuine WST (gate=1 at lit-1, handle=high32, variant=low32 forming a catalogued
// id64) — i.e. a real held weapon — or merely a field-overlap artefact (the high32
// happens to equal a catalogued weapon's high32, but the bits do not form a real
// WST slot with the canonical structure).
func verifySecondLiterals(payload []byte, groups [][]weaponLit) {
	fmt.Printf("\n================ VÉRIF 2e littéral de chaque groupe ================\n")
	for gi, g := range groups {
		if len(g) < 2 {
			fmt.Printf("  groupe %d : 1 seul littéral\n", gi)
			continue
		}
		l := g[1]
		litBit := l.bit
		// As a WST: StartBit (gate) would be at litBit-1, handle=R32@litBit, variant=R32@litBit+32.
		gate := bitsAt(payload, litBit-1, 1)
		handle := uint32(bitsAt(payload, litBit, 32))
		variant := uint32(bitsAt(payload, litBit+32, 32))
		id := (uint64(handle) << 32) | uint64(variant)
		nm, ok := analysis.WeaponIDToName[id]
		// Also: is litBit possibly the VARIANT field of a WST whose handle is at litBit-32?
		altHandle := uint32(bitsAt(payload, litBit-32, 32))
		altID := (uint64(altHandle) << 32) | uint64(handle)
		altNm, altOk := analysis.WeaponIDToName[altID]
		fmt.Printf("\n  groupe %d 2e littéral %s @bit%d (id64 scanné=0x%016x)\n", gi, l.name, litBit, l.id64)
		fmt.Printf("     comme WST: gate@%d=%d handle=0x%08x variant=0x%08x => id64=0x%016x cat=%v(%s)\n",
			litBit-1, gate, handle, variant, id, ok, nm)
		fmt.Printf("     comme VARIANT d'un WST handle@%d: 0x%08x|0x%08x => id64=0x%016x cat=%v(%s)\n",
			litBit-32, altHandle, handle, altID, altOk, altNm)
	}
}
