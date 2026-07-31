// tmp_rawalign — THROWAWAY : pinpointer l'alignement du vec3 keep-baseline (i0
// bUsePred=1). Pour les 1ers records keep-baseline de slot512, dumpe float32
// (byteswap LE) à StartBit+offset pour offset 0..24, afin de trouver l'offset où 3
// coords consécutives sont cohérentes (toutes en [-200,200], pas tiny/huge).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_rawalign [filmDir]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"math"
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

type pkt struct {
	ts  uint64
	pay []byte
}

func listFrames(d []byte) []pkt {
	var out []pkt
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, pkt{ts, d[off+16 : off+16+sz]})
		}
		off += 16 + sz
	}
	return out
}

func freshWorld(dir string, reg *filmdec.Registry) *filmdec.World {
	raw, _ := os.ReadFile(dir + "/world_dump.txt")
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

func bitAt(p []byte, i int) uint32 {
	if i < 0 || i>>3 >= len(p) {
		return 0
	}
	return uint32((p[i>>3] >> uint(7-(i&7))) & 1)
}

// f32 lit 32 bits MSB-first à o, byteswap LE, float32.
func f32(p []byte, o int) float32 {
	var v uint32
	for i := 0; i < 32; i++ {
		v = (v << 1) | bitAt(p, o+i)
	}
	v = (v&0xff)<<24 | (v&0xff00)<<8 | (v&0xff0000)>>8 | (v&0xff000000)>>24
	return math.Float32frombits(v)
}

func sane(x float32) bool {
	a := math.Abs(float64(x))
	return !math.IsNaN(float64(x)) && !math.IsInf(float64(x), 0) && (a == 0 || (a > 0.01 && a < 200))
}

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	filmdec.SetRecordStateParam(2)
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	var samples []filmdec.PositionSample
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) { samples = append(samples, s) })

	shown := 0
	for idx := 2; idx <= 26 && shown < 10; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx))) {
			if shown >= 10 {
				break
			}
			w := freshWorld(dir, reg)
			br := filmdec.NewBitReader(fr.pay)
			samples = samples[:0]
			recs, _ := filmdec.DecodeFrameRecords(br, w, calCfg)
			byBit := map[int]filmdec.PositionSample{}
			for _, s := range samples {
				byBit[s.BitPos] = s
			}
			for _, r := range recs {
				if r.Slot != 512 || len(r.Trace.Comps) == 0 {
					continue
				}
				c0 := r.Trace.Comps[0]
				if c0.Name != "object-position-dynamic-precision-component" {
					continue
				}
				s, ok := byBit[c0.StartBit]
				if !ok || s.Kind != filmdec.PosKindRaw {
					continue
				}
				// i0 header = bUsePred(1)+bDelta(1)+bHandle(1) = 3 bits, vec3 @ StartBit+3.
				fmt.Printf("slot512 keep-baseline i0 @bit%d (capture=(%.3g,%.3g,%.3g)) — scan offsets:\n",
					c0.StartBit, s.Vec[0], s.Vec[1], s.Vec[2])
				for off := 0; off <= 16; off++ {
					o := c0.StartBit + off
					x, y, z := f32(fr.pay, o), f32(fr.pay, o+32), f32(fr.pay, o+64)
					mark := ""
					if sane(x) && sane(y) && sane(z) {
						mark = "  <<< 3 coords saines"
					}
					fmt.Printf("   +%-2d (%12.4g, %12.4g, %12.4g)%s\n", off, x, y, z, mark)
				}
				shown++
				if shown >= 10 {
					break
				}
			}
		}
	}
	filmdec.SetPositionCaptureHook(nil)
}
