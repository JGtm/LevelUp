package main

import (
	"fmt"

	"levelup/go-api/internal/analysis/filmdec"
)

// traceGrammar : dump position bit après chaque champ du default-state pour un biped, pour
// exposer combien de bits chaque étape consomme (repérer un champ à mauvaise largeur).
func traceGrammar(pay []byte, stateBit int) {
	fmt.Printf("\n=== 8. TRACE grammaire default-state biped @ stateBit=%d ===\n", stateBit)
	prev := stateBit
	filmdec.SetRepTraceHook(func(label string, bitpos int) {
		fmt.Printf("  %-14s bit=%d (Δ=%d)\n", label, bitpos, bitpos-prev)
		prev = bitpos
	})
	end := filmdec.BipedDefaultStateEndBit(pay, stateBit)
	filmdec.SetRepTraceHook(nil)
	fmt.Printf("  version(uVar10)=%d ; endBit=%d total default-state=%d bits\n",
		filmdec.LastRepVersion(), end, end-stateBit)
}
