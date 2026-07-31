// tmp_i48trace — THROWAWAY : trace focalisée d'un biped keyframe APRÈS (a) le fix de
// polarité FUN_1406d1024 dans consume1407f0550 (i43) et (b) le portage d'i48
// (biped-desired-ability-set). Question : la traversée chaîne-t-elle au-delà d'i48 ?
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

func inflate(p string) []byte {
	raw, _ := os.ReadFile(p)
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

func extractType2(d []byte) []byte {
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		if typ == 2 {
			return d[off+16 : off+16+sz]
		}
		off += 16 + sz
	}
	return nil
}

func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

func knownHigh32(v uint32) (string, bool) {
	for id, n := range analysis.WeaponIDToName {
		if uint32(id>>32) == v {
			return n, true
		}
	}
	return "", false
}

func traceOne(payload []byte, reg *filmdec.Registry, start, d int, rsp uint32) {
	filmdec.SetRecordStateParam(rsp)
	br := filmdec.NewBitReader(payload)
	br.Skip(start)
	t := filmdec.TraverseEntity(br, reg, d)
	fmt.Printf("== start=%d d=%d rsp=%d : typeIndex=%d, %d comps, DesyncAt=i%d, endBit=%d\n",
		start, d, rsp, t.TypeIndex, len(t.Comps), t.DesyncAt, t.EndBit)
	for _, c := range t.Comps {
		extra := ""
		if c.Name == "weapon-state-type-info" {
			handle := uint32(bitsAt(payload, c.StartBit+1, 32))
			variant := uint32(bitsAt(payload, c.StartBit+33, 32))
			hn, hok := knownHigh32(handle)
			vn, vok := knownHigh32(variant)
			extra = fmt.Sprintf("  handle=0x%08x(%s) variant=0x%08x(%s)", handle, pick(hok, hn), variant, pick(vok, vn))
		}
		mark := ""
		if !c.Ported {
			mark = "  <<< DESYNC (non porté)"
		}
		fmt.Printf("  i%-2d %-44s @bit%d ported=%v%s%s\n", c.Index, c.Name, c.StartBit, c.Ported, extra, mark)
	}
}

func main() {
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	payload := extractType2(inflate(cache + "/chunk_02.bin"))
	fmt.Printf("registre %d archétypes ; keyframe %d octets\n\n", len(reg.Archetypes), len(payload))
	// Mauvaise calibration (Piste A) : start=193552, d=88, rsp=0 -> i43 = bruit.
	traceOne(payload, reg, 193552, 88, 0)
	fmt.Println()
	// BONNE calibration (K2) : start=194126, defaultBits=380, rsp=2 -> i45 WST = Hydra.
	traceOne(payload, reg, 194126, 380, 2)
}

func pick(ok bool, s string) string {
	if ok {
		return s
	}
	return "?"
}
