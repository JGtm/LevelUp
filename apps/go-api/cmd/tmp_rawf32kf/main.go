// tmp_rawf32kf — teste si la POSITION i0 keyframe des 8 bipeds est stockée en float32 BRUT
// (96 bits, 3×IEEE-754, MSB-first) plutôt qu'en quantifié. On balaie l'offset (bit) depuis le
// stateBit de chaque biped et on cherche un offset où les 8 bipeds donnent 3 float32 finis,
// DISTINCTS et DANS la boîte oracle x[-6.33,35.70] y[-25.14,27.50] z[-4.20,7.08] (ou une boîte
// élargie ±2). THROWAWAY.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_rawf32kf
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const film = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

var box = [3][2]float64{{-6.33, 35.70}, {-25.14, 27.50}, {-4.20, 7.08}}

func inflate(p string) []byte {
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); (e2 == nil || e2 == io.ErrUnexpectedEOF) && len(d) > 0 {
				return d
			}
		}
	}
	return raw
}
func framePayload(d []byte, want uint16) []byte {
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		if typ == want {
			return d[off+16 : off+16+sz]
		}
		off += 16 + sz
	}
	return nil
}
func readBits(buf []byte, pos, n int) uint64 {
	var r uint64
	for i := 0; i < n; i++ {
		p := pos + i
		var bit uint64
		if idx := p >> 3; idx < len(buf) {
			bit = uint64(buf[idx]>>(7-uint(p&7))) & 1
		}
		r = r<<1 | bit
	}
	return r
}

// f32MSB lit un float32 MSB-first à partir du bit p.
func f32MSB(buf []byte, p int) float32 {
	return math.Float32frombits(uint32(readBits(buf, p, 32)))
}

// f32LSB : lit 32 bits MSB-first puis byte-swap (little-endian dans un flux MSB).
func f32LSB(buf []byte, p int) float32 {
	u := uint32(readBits(buf, p, 32))
	u = (u>>24)&0xff | (u>>8)&0xff00 | (u<<8)&0xff0000 | (u<<24)&0xff000000
	return math.Float32frombits(u)
}

func finite(v float32) bool {
	return !math.IsNaN(float64(v)) && !math.IsInf(float64(v), 0)
}
func inBox(v [3]float64, pad float64) bool {
	for i := 0; i < 3; i++ {
		if v[i] < box[i][0]-pad || v[i] > box[i][1]+pad {
			return false
		}
	}
	return true
}

func main() {
	pay := framePayload(inflate(film+"/chunk_02.bin"), 2)
	recs := filmdec.WalkKeyframeWorld(pay)
	hdr := map[int]int{}
	for _, r := range recs {
		if r.TI == 35 && r.Slot >= 512 && r.Slot <= 519 {
			hdr[r.Slot] = r.Bit
		}
	}
	slots := []int{}
	for s := range hdr {
		slots = append(slots, s)
	}
	sort.Ints(slots)
	total := len(pay) * 8
	// pour chaque slot, on cherche entre stateBit et stateBit+400
	type hit struct {
		off, nIn, nFin, nDist int
		bo                    bool // byteswap
	}
	var best []hit
	for _, bo := range []bool{false, true} {
		for off := 0; off <= 400; off++ {
			nFin, nIn := 0, 0
			seen := map[[3]int]bool{}
			for _, s := range slots {
				p := hdr[s] + 64 + off
				if p+96 > total {
					continue
				}
				var v [3]float64
				ok := true
				for i := 0; i < 3; i++ {
					var f float32
					if bo {
						f = f32LSB(pay, p+i*32)
					} else {
						f = f32MSB(pay, p+i*32)
					}
					if !finite(f) || math.Abs(float64(f)) > 1e4 {
						ok = false
					}
					v[i] = float64(f)
				}
				if ok {
					nFin++
					seen[[3]int{int(v[0]), int(v[1]), int(v[2])}] = true
					if inBox(v, 3) {
						nIn++
					}
				}
			}
			if nIn >= 5 {
				best = append(best, hit{off, nIn, nFin, len(seen), bo})
			}
		}
	}
	sort.Slice(best, func(i, j int) bool { return best[i].nIn > best[j].nIn })
	fmt.Printf("bipeds=%d slots=%v ; boîte oracle (pad 3) x%v y%v z%v\n", len(slots), slots, box[0], box[1], box[2])
	fmt.Printf("=== offsets où >=5 bipeds ont un float32 raw in-box (pad 3) : %d ===\n", len(best))
	for i, h := range best {
		if i >= 25 {
			break
		}
		fmt.Printf("  off=+%-3d byteswap=%v inBox=%d/8 finite=%d distinct=%d\n", h.off, h.bo, h.nIn, h.nFin, h.nDist)
	}
	if len(best) > 0 {
		h := best[0]
		fmt.Printf("\n=== MEILLEUR off=+%d byteswap=%v ===\n", h.off, h.bo)
		for _, s := range slots {
			p := hdr[s] + 64 + h.off
			var v [3]float64
			for i := 0; i < 3; i++ {
				if h.bo {
					v[i] = float64(f32LSB(pay, p+i*32))
				} else {
					v[i] = float64(f32MSB(pay, p+i*32))
				}
			}
			fmt.Printf("  slot=%d (%.3f, %.3f, %.3f)\n", s, v[0], v[1], v[2])
		}
	} else {
		fmt.Println("  AUCUN offset float32-raw in-box. La position keyframe n'est pas un float32 brut à offset fixe.")
	}
}
