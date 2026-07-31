package main

import (
	"fmt"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

// traceOne dumps a full record traversal for inspection.
func traceOne(reg *filmdec.Registry, payload []byte, start, d int, rsp uint32) {
	filmdec.SetRecordStateParam(rsp)
	br := filmdec.NewBitReader(payload)
	br.Skip(start)
	t := filmdec.TraverseEntity(br, reg, d)
	fmt.Printf("\n=== TRACE start=%d d=%d rsp=%d : typeIndex=%d nComps=%d desyncAt=i%d endBit=%d ===\n",
		start, d, rsp, t.TypeIndex, len(t.Comps), t.DesyncAt, t.EndBit)
	for _, c := range t.Comps {
		extra := ""
		switch c.Name {
		case "weapon-state-type-info":
			g := bitsAt(payload, c.StartBit, 1)
			h := uint32(bitsAt(payload, c.StartBit+1, 32))
			v := uint32(bitsAt(payload, c.StartBit+33, 32))
			id := (uint64(h) << 32) | uint64(v)
			nm := analysis.WeaponIDToName[id]
			extra = fmt.Sprintf("  gate=%d handle=0x%08x variant=0x%08x id64=0x%016x arme=%q", g, h, v, id, nm)
		case "object-multiplayer-properties-component":
			lid, lidp, variant, vbit := objePrefix(payload, c.StartBit)
			extra = fmt.Sprintf("  LocalID(present=%v)=0x%08x VariantName@%d=0x%08x (compResult.Variant=0x%08x)", lidp, lid, vbit, variant, c.Variant)
		}
		mark := ""
		if !c.Ported {
			mark = "  <<< DESYNC"
		}
		fmt.Printf("  i%-2d %-46s @bit%d%s%s\n", c.Index, c.Name, c.StartBit, extra, mark)
	}
}
