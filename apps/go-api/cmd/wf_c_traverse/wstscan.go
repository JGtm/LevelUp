package main

// wstscan : brute-force le bit de DÉBUT du composant weapon-state-type-info qui
// (a) décode le variant Hydra à l'ancre1, et (b) consomme exactement jusqu'à l'ancre2
// -33 (donc i44 enchaîne et son variant tombe sur l'ancre2). On balaie start dans une
// fenêtre [ancre1-260, ancre1] et on retient les start où le deser RÉEL produit le bon
// variant ET un coût cohérent avec le spacing 196.

import (
	"fmt"

	"levelup/go-api/internal/analysis/filmdec"
)

func wstStartScan(payload []byte) {
	fmt.Println("\n=== BRUTE-FORCE start du deser weapon-state-type-info (R0) ===")
	const a1, a2 = 195323, 195519
	target := uint32(readU32At(payload, a1)) // = Hydra family-tagged full id
	fmt.Printf("  cible variant @ancre1 = 0x%08x (%s)\n", target, famName(target))

	type cand struct {
		start, end int
		variant    uint32
	}
	var cands []cand
	for start := a1 - 260; start <= a1; start++ {
		variant, end := filmdec.ConsumeWeaponStateTypeInfoVariantAt(payload, start)
		if variant == target {
			cands = append(cands, cand{start, end, variant})
		}
	}
	fmt.Printf("  %d start(s) où le deser produit EXACTEMENT le variant Hydra :\n", len(cands))
	for _, c := range cands {
		// si end+? aligne l'ancre2, on a le coût réel
		v2, _ := filmdec.ConsumeWeaponStateTypeInfoVariantAt(payload, c.end)
		fmt.Printf("    start=%d end=%d coût=%d  -> i44.variant=0x%08x(%s) (ancre2=%d, i44.varPos=%d)\n",
			c.start, c.end, c.end-c.start, v2, famName(v2), a2, c.end+filmdec.VariantBitOffsetInWST)
	}
	if len(cands) == 0 {
		fmt.Println("  AUCUN start ne produit le variant via le deser RE'd FUN_1407f06bc.")
		fmt.Println("  => le variant à l'ancre n'est PAS encodé via gate(1)+handle(32)+variant(32) :")
		fmt.Println("     le littéral d'arme dans la keyframe est un u32 BRUT, hors structure wst.")
	}

	// Confirmation : la séquence des 32 bits AVANT l'ancre — y a-t-il un gate=1 + handle ?
	fmt.Println("\n  -- contexte bits autour de l'ancre1 (66 bits avant variant) --")
	for _, d := range []int{65, 33, 32, 1, 0} {
		raw := readBitsAt(payload, a1-d, min(d, 40))
		fmt.Printf("    %d bits avant variant : 0x%x\n", d, raw)
	}
}

func readBitsAt(payload []byte, bit, n int) uint64 {
	if n <= 0 {
		return 0
	}
	br := filmdec.NewBitReader(payload)
	br.Skip(bit)
	return br.ReadBits(uint(n))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
