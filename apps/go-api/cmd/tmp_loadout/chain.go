package main

import (
	"fmt"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

// countCatalogued returns how many WST slots in the trace reconstruct a catalogued
// weapon id64 (low32 == 0x42c9679f canonical suffix OR present in WeaponIDToName).
func countCatalogued(payload []byte, t filmdec.EntityTrace) int {
	n := 0
	for _, c := range t.Comps {
		if c.Name != "weapon-state-type-info" {
			continue
		}
		h := uint32(bitsAt(payload, c.StartBit+1, 32))
		v := uint32(bitsAt(payload, c.StartBit+33, 32))
		id := (uint64(h) << 32) | uint64(v)
		if _, ok := analysis.WeaponIDToName[id]; ok {
			n++
		}
	}
	return n
}

// chainRecords decodes biped records consecutively starting at firstStart, using
// each record's EndBit (+ optional separator sweep) as the next record's start.
// For each record it sweeps (defaultBits, rsp) to find the variant that traverses
// fully (DesyncAt == -1) and lands typeIndex==35; it then prints the held weapon
// (i45/i46 WST cataloguées) and the obje VariantName.
func chainRecords(reg *filmdec.Registry, payload []byte, firstStart int, maxRecords int) {
	fmt.Printf("\n================ CHAÎNAGE des records biped (depuis %d) ================\n", firstStart)
	pos := firstStart
	for rec := 0; rec < maxRecords; rec++ {
		// Sweep a separator window before the next R(6)==35, plus (d, rsp).
		// HARD CONSTRAINT: the chosen calibration MUST traverse fully (DesyncAt==-1)
		// AND contain >=1 CATALOGUED weapon (low32==0x42c9679f). This disambiguates
		// the rsp variants that all reach DesyncAt==-1 but mis-align the WST slots.
		best := -1
		var bstart, bd, brsp, bend, bdesync int
		// Records bipeds NON contigus : d'autres entités s'intercalent. Cherche le
		// PROCHAIN biped #35 décodable (DesyncAt=-1 + arme cataloguée) dans [pos, pos+3000].
		for sep := 0; sep <= 3000 && best < 0; sep++ {
			start := pos + sep
			if uint32(bitsAt(payload, start, 6)) != 35 {
				continue
			}
			for d := 1; d <= 420; d++ {
				for r := uint32(0); r <= 3; r++ {
					filmdec.SetRecordStateParam(r)
					b := filmdec.NewBitReader(payload)
					b.Skip(start)
					t := filmdec.TraverseEntity(b, reg, d)
					if t.TypeIndex != 35 || t.DesyncAt != -1 {
						continue
					}
					nCat := countCatalogued(payload, t)
					if nCat == 0 {
						continue
					}
					score := nCat*10000 + len(t.Comps)
					if score > best {
						best = score
						bstart, bd, brsp, bend, bdesync = start, d, int(r), t.EndBit, t.DesyncAt
					}
				}
			}
			if best > 0 {
				pos = bstart
			}
		}
		if best < 0 {
			fmt.Printf("  record %d : aucun biped #35 (DesyncAt=-1 + arme cataloguée) depuis bit %d\n", rec, pos)
			return
		}
		// Re-run the best to dump.
		filmdec.SetRecordStateParam(uint32(brsp))
		b := filmdec.NewBitReader(payload)
		b.Skip(pos)
		t := filmdec.TraverseEntity(b, reg, bd)
		var weapons []string
		var objeVariant string
		for _, c := range t.Comps {
			if c.Name == "weapon-state-type-info" {
				h := uint32(bitsAt(payload, c.StartBit+1, 32))
				v := uint32(bitsAt(payload, c.StartBit+33, 32))
				id := (uint64(h) << 32) | uint64(v)
				if nm, ok := analysis.WeaponIDToName[id]; ok {
					weapons = append(weapons, fmt.Sprintf("%s@%d", nm, c.StartBit+1))
				} else {
					weapons = append(weapons, fmt.Sprintf("?0x%016x@%d", id, c.StartBit+1))
				}
			}
			if c.Name == "object-multiplayer-properties-component" {
				_, _, variant, _ := objePrefix(payload, c.StartBit)
				objeVariant = fmt.Sprintf("0x%08x", variant)
			}
		}
		fmt.Printf("  record %d : start=%-7d d=%-3d rsp=%d desyncAt=i%-3d endBit=%-7d objeVar=%s armes=%v\n",
			rec, pos, bd, brsp, bdesync, bend, objeVariant, weapons)
		pos = bend
	}
}
