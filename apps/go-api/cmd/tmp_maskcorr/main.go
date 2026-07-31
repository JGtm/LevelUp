package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"levelup/go-api/internal/analysis/filmdec"
	"os"
	"sort"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}

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
func listFrames(d []byte) [][]byte {
	var out [][]byte
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, d[off+16:off+16+sz])
		}
		off += 16 + sz
	}
	return out
}
func freshWorld(reg *filmdec.Registry) *filmdec.World {
	raw, _ := os.ReadFile(cache + "/world_dump.txt")
	w := filmdec.NewWorld(reg)
	for _, tok := range bytes.Fields(raw) {
		s := string(tok)
		if len(s) == 0 || s[0] == '#' {
			continue
		}
		var slot, ti uint32
		if _, e := fmt.Sscanf(s, "%d:%d", &slot, &ti); e == nil {
			w.BindFull(slot, ti)
		}
	}
	return w
}
func main() {
	filmdec.SetRecordStateParam(2)
	filmdec.PositionCalibratedSkip = true
	for n, w := range map[string]int{"object-forward-and-up-component": 9, "object-angular-velocity-component": 1, "object-shield-vitality-component": 29, "object-region-state-component": 358, "object-multiplayer-properties-component": 334} {
		filmdec.SetCalibratedWidth(n, w)
	}
	reg, _ := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	arch, _ := reg.Archetype(35)
	i63 := -1
	for i, c := range arch.Components {
		if c == "biped-action-component" || c == "biped-action" {
			i63 = i
		}
	}
	// presence par composant dans {desync@i63, clean} parmi records AYANT i63
	presDes := map[int]int{}
	presCln := map[int]int{}
	nDes, nCln := 0, 0
	for idx := 2; idx <= 20; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			w := freshWorld(reg)
			recs, _ := filmdec.DecodeFrameRecords(filmdec.NewBitReader(fr), w, calCfg)
			for _, r := range recs {
				if r.TypeIndex != 35 {
					continue
				}
				if r.Trace.Mask&(uint64(1)<<uint(i63&63)) == 0 {
					continue
				} // doit avoir i63
				des := r.DesyncAt != -1
				if des {
					nDes++
				} else {
					nCln++
				}
				for i := 0; i < i63; i++ {
					if r.Trace.Mask&(uint64(1)<<uint(i&63)) != 0 {
						if des {
							presDes[i]++
						} else {
							presCln[i]++
						}
					}
				}
			}
		}
	}
	fmt.Printf("=== records biped AVEC i63 : %d désync, %d clean ===\n", nDes, nCln)
	type rc struct {
		i      int
		dr, cr float64
		name   string
	}
	var rs []rc
	for i := 0; i < i63; i++ {
		if i >= len(arch.Components) {
			break
		}
		dr := 0.0
		cr := 0.0
		if nDes > 0 {
			dr = float64(presDes[i]) / float64(nDes)
		}
		if nCln > 0 {
			cr = float64(presCln[i]) / float64(nCln)
		}
		rs = append(rs, rc{i, dr, cr, arch.Components[i]})
	}
	sort.Slice(rs, func(a, b int) bool { return (rs[a].dr - rs[a].cr) > (rs[b].dr - rs[b].cr) })
	fmt.Println("=== composants i<63 corrélés au désync (présence désync - présence clean, top) ===")
	for k, e := range rs {
		if k >= 12 {
			break
		}
		fmt.Printf("  i%-2d %-46s désync=%.2f clean=%.2f Δ=%+.2f\n", e.i, e.name, e.dr, e.cr, e.dr-e.cr)
	}
}
