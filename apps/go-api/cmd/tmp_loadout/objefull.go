package main

import (
	"fmt"

	"levelup/go-api/internal/analysis/filmdec"
)

// objeForEachRecord calibrates each of the 8 biped records (anchored on EITHER the
// primary OR secondary literal of the group, whichever yields the deepest sane
// traversal that reaches i9 'obje'), then prints the obje VariantName + LocalID.
// This is the only player-identifying field present in the keyframe; the goal is to
// see whether the 8 obje variants form a coherent per-player key.
func objeForEachRecord(reg *filmdec.Registry, payload []byte, groups [][]weaponLit) {
	fmt.Printf("\n================ 'obje' i9 (VariantName/LocalID) par record ================\n")
	for gi, g := range groups {
		var bestObje = -1
		var bestLID uint32
		var bestLIDp bool
		var bestVar uint32
		var bStart, bD int
		var bRSP uint32
		var bDesync int
		// Try anchoring on each literal of the group.
		for _, anchor := range g {
			gateBit := anchor.bit - 1
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
						// must hit the anchor WST
						hit, objeStart := false, -1
						for _, c := range t.Comps {
							if c.Name == "weapon-state-type-info" && c.StartBit == gateBit {
								hh := uint32(bitsAt(payload, gateBit+1, 32))
								vv := uint32(bitsAt(payload, gateBit+33, 32))
								if (uint64(hh)<<32)|uint64(vv) == anchor.id64 {
									hit = true
								}
							}
							if c.Name == "object-multiplayer-properties-component" && objeStart < 0 {
								objeStart = c.StartBit
							}
						}
						if !hit || objeStart < 0 {
							continue
						}
						if objeStart > bestObje || (objeStart == bestObje && t.DesyncAt < 0) {
							lid, lidp, variant, _ := objePrefix(payload, objeStart)
							bestObje = objeStart
							bestLID, bestLIDp, bestVar = lid, lidp, variant
							bStart, bD, bRSP, bDesync = start, d, r, t.DesyncAt
						}
					}
				}
			}
			if bestObje >= 0 {
				break // got an obje for this group
			}
		}
		if bestObje < 0 {
			fmt.Printf("  record %d : pas de 'obje' atteint (prim=%s)\n", gi, g[0].name)
			continue
		}
		prim := weaponAt(payload, g[0].bit)
		sec := ""
		if len(g) >= 2 {
			sec = weaponAt(payload, g[1].bit)
		}
		fmt.Printf("  record %d : prim=%-15s sec=%-15s | objeStart=%d LocalID(present=%v)=0x%08x VariantName=0x%08x (start=%d d=%d rsp=%d desync=i%d)\n",
			gi, prim, sec, bestObje, bestLIDp, bestLID, bestVar, bStart, bD, bRSP, bDesync)
	}
}
