// tmp_bipedresync — teste la RÉCUPÉRATION des bipèdes tardifs via DecodeFrameResync :
// après un desync (slot non-lié / composant non-porté), scan forward pour le prochain
// delta biped propre et resync. Compare le nombre d'échantillons i0 par slot vs le
// décodage séquentiel normal, avec la répartition par type (abs/delta = exploitable).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_bipedresync [filmDir]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"levelup/go-api/internal/analysis/filmdec"
)

const defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}
var bipedSlots = map[uint32]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}

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

func freshWorld(dir string, reg *filmdec.Registry) *filmdec.World {
	raw, _ := os.ReadFile(dir + "/world_dump_full.txt")
	w := filmdec.NewWorld(reg)
	for _, tok := range bytes.Fields(raw) {
		var slot, ti uint32
		if _, e := fmt.Sscanf(string(tok), "%d:%d", &slot, &ti); e == nil {
			w.BindFull(slot, ti)
		}
	}
	return w
}

type counts struct{ tot, abs, d8, dax, raw, absfb int }

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	resync := os.Getenv("SEQ") == "" // par défaut resync ; SEQ=1 -> séquentiel (baseline)
	filmdec.SetRecordStateParam(2)
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	var samples []filmdec.PositionSample
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) { samples = append(samples, s) })

	perSlot := map[uint32]*counts{}
	for s := uint32(512); s <= 519; s++ {
		perSlot[s] = &counts{}
	}
	for idx := 2; idx <= 26; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx))) {
			w := freshWorld(dir, reg)
			samples = samples[:0]
			byBit := map[int]filmdec.PositionSample{}
			var recs []filmdec.FrameRecord
			if resync {
				recs = filmdec.DecodeFrameResync(fr, w, calCfg, bipedSlots, nil)
			} else {
				br := filmdec.NewBitReader(fr)
				recs, _ = filmdec.DecodeFrameRecords(br, w, calCfg)
			}
			for _, s := range samples {
				byBit[s.BitPos] = s
			}
			for _, r := range recs {
				if !bipedSlots[r.Slot] {
					continue
				}
				for _, c := range r.Trace.Comps {
					if c.Name != "object-position-dynamic-precision-component" {
						continue
					}
					if s, ok := byBit[c.StartBit]; ok {
						cc := perSlot[r.Slot]
						cc.tot++
						switch s.Kind {
						case filmdec.PosKindAbsolute:
							cc.abs++
						case filmdec.PosKindAbsFallback:
							cc.absfb++
						case filmdec.PosKindDelta8:
							cc.d8++
						case filmdec.PosKindDeltaAxis:
							cc.dax++
						case filmdec.PosKindRaw:
							cc.raw++
						}
					}
					break
				}
			}
		}
	}
	filmdec.SetPositionCaptureHook(nil)

	mode := "RESYNC"
	if !resync {
		mode = "SEQUENTIEL"
	}
	fmt.Printf("mode=%s — échantillons i0 par slot biped (abs+delta = exploitables) :\n", mode)
	for s := uint32(512); s <= 519; s++ {
		c := perSlot[s]
		usable := c.abs + c.absfb + c.d8 + c.dax
		fmt.Printf("  slot%d : tot=%-5d exploitables=%-4d (abs=%d absfb=%d d8=%d dax=%d) raw=%d\n",
			s, c.tot, usable, c.abs, c.absfb, c.d8, c.dax, c.raw)
	}
}
