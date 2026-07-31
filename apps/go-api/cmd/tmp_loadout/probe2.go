package main

import (
	"fmt"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

// probeSecondary investigates, for each weapon group, whether the 2nd literal
// (the "secondary" loadout slot) is a WST of the SAME biped record (in a slot
// reached after the first/held WST) or a separate entity. It calibrates the biped
// on the FIRST literal (held weapon, desync tolerated) and reports every WST slot
// reached, with its catalogued / non-catalogued id64.
func probeSecondary(reg *filmdec.Registry, payload []byte, groups [][]weaponLit) {
	fmt.Printf("\n================ PROBE slots WST par record (ancré 1er littéral) ================\n")
	for gi, g := range groups {
		anchor := g[0]
		gateBit := anchor.bit - 1
		// Best = deepest DesyncAt with the anchor WST hit.
		var bestT *filmdec.EntityTrace
		var bStart, bD int
		var bRSP uint32
		for start := gateBit - 2800; start <= gateBit-20; start++ {
			if uint32(bitsAt(payload, start, 6)) != 35 {
				continue
			}
			for d := 1; d <= 420; d++ {
				for r := uint32(0); r <= 3; r++ {
					filmdec.SetRecordStateParam(r)
					b := filmdec.NewBitReader(payload)
					b.Skip(start)
					t := filmdec.TraverseEntity(b, reg, d)
					if t.TypeIndex != 35 {
						continue
					}
					hit := false
					for _, c := range t.Comps {
						if c.Name == "weapon-state-type-info" && c.StartBit == gateBit {
							hh := uint32(bitsAt(payload, gateBit+1, 32))
							vv := uint32(bitsAt(payload, gateBit+33, 32))
							if (uint64(hh)<<32)|uint64(vv) == anchor.id64 {
								hit = true
							}
						}
					}
					if !hit {
						continue
					}
					better := bestT == nil
					if bestT != nil {
						// prefer DesyncAt==-1, else deeper desync.
						if (t.DesyncAt < 0) != (bestT.DesyncAt < 0) {
							better = t.DesyncAt < 0
						} else if (t.DesyncAt < 0) == (bestT.DesyncAt < 0) {
							better = t.DesyncAt > bestT.DesyncAt
						}
					}
					if better {
						tc := t
						bestT = &tc
						bStart, bD, bRSP = start, d, r
					}
				}
			}
		}
		if bestT == nil {
			fmt.Printf("\n  groupe %d (%s + %s) : ÉCHEC calibration\n", gi, g[0].name, secName(g))
			continue
		}
		fmt.Printf("\n  groupe %d : start=%d d=%d rsp=%d desyncAt=i%d nComps=%d ; littéraux attendus: %s\n",
			gi, bStart, bD, bRSP, bestT.DesyncAt, len(bestT.Comps), litNames(g))
		for _, c := range bestT.Comps {
			if c.Name != "weapon-state-type-info" {
				continue
			}
			h := uint32(bitsAt(payload, c.StartBit+1, 32))
			v := uint32(bitsAt(payload, c.StartBit+33, 32))
			id := (uint64(h) << 32) | uint64(v)
			nm, ok := analysis.WeaponIDToName[id]
			tag := "?non-catalogué"
			if ok {
				tag = nm
			}
			fmt.Printf("      WST i%-2d @bit%-7d handle=0x%08x variant=0x%08x id64=0x%016x => %s\n",
				c.Index, c.StartBit, h, v, id, tag)
		}
	}
}

func secName(g []weaponLit) string {
	if len(g) >= 2 {
		return g[1].name
	}
	return "(aucune)"
}

func litNames(g []weaponLit) string {
	s := ""
	for _, w := range g {
		s += fmt.Sprintf("%s@%d ", w.name, w.bit)
	}
	return s
}
