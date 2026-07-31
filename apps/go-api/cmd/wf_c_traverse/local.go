package main

// Validation LOCALE du composant weapon-state-type-info (i43) : on connaît l'ancre
// arme à 195323 (Hydra) et la 2e à 195519 (Diminisher of Hope = Hammer), +196 bits.
// Le deser lit variant à offset fixe (gate R1 + handle R32 = +33 bits) donc i43
// démarre à 195323-33 = 195290. On déroule le VRAI deser à partir de là et on vérifie
// qu'il enchaîne i44 dont le variant tombe exactement sur la 2e ancre. C'est une
// preuve INDÉPENDANTE de la traversée globale (qui, elle, exige i0..i42 bit-exacts).

import (
	"fmt"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

func famName(v uint32) string {
	for id, n := range analysis.WeaponIDToName {
		if uint32(id>>32) == v {
			return n
		}
	}
	return fmt.Sprintf("0x%08x?", v)
}

func readU32At(payload []byte, bit int) uint32 {
	br := filmdec.NewBitReader(payload)
	br.Skip(bit)
	return uint32(br.ReadBits(32))
}

func validateLocalWST(payload []byte) {
	fmt.Println("\n=== VALIDATION LOCALE weapon-state-type-info (i43->i46) sur R0 ===")
	anchors := []int{195323, 195519} // Hydra, Diminisher of Hope (Hammer)
	off := filmdec.VariantBitOffsetInWST

	start := anchors[0] - off
	fmt.Printf("  i43.start = ancre1(%d) - offset(%d) = %d\n", anchors[0], off, start)

	// Déroule i43, i44, i45, i46 en chaîne et vérifie chaque variant vs ancre.
	pos := start
	for slot := 0; slot < 4; slot++ {
		variant, end := filmdec.ConsumeWeaponStateTypeInfoVariantAt(payload, pos)
		cost := end - pos
		varPos := pos + off
		raw := readU32At(payload, varPos)
		match := ""
		if slot < len(anchors) {
			if varPos == anchors[slot] {
				match = fmt.Sprintf("  == ANCRE%d ✓", slot+1)
			} else {
				match = fmt.Sprintf("  ancre%d=%d delta=%d", slot+1, anchors[slot], varPos-anchors[slot])
			}
		}
		fmt.Printf("  i4%d start=%d end=%d coût=%d variant=0x%08x(%s) varPos=%d raw=0x%08x(%s)%s\n",
			3+slot, pos, end, cost, variant, famName(variant), varPos, raw, famName(raw), match)
		pos = end
	}

	fmt.Printf("\n  vérité terrain : ancre2-ancre1 = %d bits (coût observé d'un wst plein)\n",
		anchors[1]-anchors[0])
	fmt.Println("  => si le deser i43 consomme exactement ce nombre de bits, i44.variant tombe sur ancre2.")
}

// validateAllRecordsLocal applique la validation locale à chaque record (les 8 ancres
// de chunk_02) : i43.start = ancre - offset, déroule, le 2e variant doit matcher la 2e
// arme du record (ancre+196). Donne le coût wst par record (constance = bit-exactness).
func validateAllRecordsLocal(payload []byte) {
	fmt.Println("\n=== COÛT wst PAR RECORD (les 8 records de chunk_02) ===")
	for _, a := range chunk02Anchors {
		off := filmdec.VariantBitOffsetInWST
		start := a.Bit - off
		v1, end1 := filmdec.ConsumeWeaponStateTypeInfoVariantAt(payload, start)
		v2, _ := filmdec.ConsumeWeaponStateTypeInfoVariantAt(payload, end1)
		v2pos := end1 + off
		secondAnchor := a.Bit + 196
		ok := "✗"
		if v2pos == secondAnchor {
			ok = "✓"
		}
		fmt.Printf("  %-32s coût_i43=%d  v1=%s  v2=%s  v2pos=%d/attendu=%d %s\n",
			a.Pair, end1-start, famName(v1), famName(v2), v2pos, secondAnchor, ok)
	}
	_ = analysis.WeaponIDToName
}
