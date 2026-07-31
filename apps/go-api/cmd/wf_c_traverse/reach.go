package main

// reach : pour chaque record, balaie (start, default-state) sur une large grille et
// rapporte si TraverseEntity atteint UN composant weapon-state-type-info (i43..46)
// dont le StartBit est proche de l'ancre. Si aucun start n'atteint i43 sans désync,
// la traversée ECS ne peut pas restituer l'arme dans la keyframe dense.

import (
	"fmt"

	"levelup/go-api/internal/analysis/filmdec"
)

func reachWeaponSlot(reg *filmdec.Registry, payload []byte) {
	fmt.Println("\n=== TraverseEntity : un slot i43.. décode-t-il l'ARME (variant) à l'ancre ? ===")
	for _, a := range chunk02Anchors {
		target := readU32At(payload, a.Bit) // famille d'arme attendue à l'ancre
		decodedHit := 0                     // traversées dont un slot wst décode EXACTEMENT target
		var hitStart, hitDef, hitSlotBit int = -1, -1, -1
		for start := a.Bit - 2900; start <= a.Bit-30; start++ {
			if start < 0 {
				continue
			}
			b0 := filmdec.NewBitReader(payload)
			b0.Skip(start)
			if uint32(b0.ReadBits(6)) != 35 {
				continue
			}
			for d := 50; d <= 130; d++ {
				b := filmdec.NewBitReader(payload)
				b.Skip(start)
				t := filmdec.TraverseEntity(b, reg, d)
				if t.TypeIndex != 35 {
					continue
				}
				for _, c := range t.Comps {
					if c.Name == "weapon-state-type-info" && c.Variant == target {
						decodedHit++
						if hitStart == -1 {
							hitStart, hitDef, hitSlotBit = start, d, c.StartBit
						}
					}
				}
			}
		}
		status := "AUCUN slot ne décode l'arme"
		if decodedHit > 0 {
			status = fmt.Sprintf("%d traversées décodent l'arme (1er: start=%d d=%d slot@%d)",
				decodedHit, hitStart, hitDef, hitSlotBit)
		}
		fmt.Printf("  %-32s target=0x%08x(%s) -> %s\n", a.Pair, target, famName(target), status)
	}
}
