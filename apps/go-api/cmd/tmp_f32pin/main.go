// tmp_f32pin — TEST du cadrage utilisateur : la position biped keyframe est-elle un
// float32 LE (byteswap) BIT-PACKE a un offset CONSTANT du record ? Pour les 8 bipeds de
// chunk_02 (stateBit connus dans biped_record_offsets.txt), on balaye l'offset bit relatif
// au stateBit et on lit un triplet float32 LE. On cherche l'offset ou les 8 tombent dans la
// boite oracle x[-6,36] y[-25,27] z[-4,7] ET sont DISTINCTS. Compare BE vs LE.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
)

const film = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

// stateBits des 8 bipeds keyframe (biped_record_offsets.txt : hdrBit+64), relatifs au payload type-2.
var stateBits = map[int]int{
	512: 193467, 513: 196252, 514: 199068, 515: 201862,
	516: 204665, 517: 207460, 518: 210262, 519: 213057,
}
var hdrBits = map[int]int{
	512: 193403, 513: 196188, 514: 199004, 515: 201798,
	516: 204601, 517: 207396, 518: 210198, 519: 212993,
}

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

func bitAt(p []byte, i int) uint32 {
	if i < 0 || i>>3 >= len(p) {
		return 0
	}
	return uint32((p[i>>3] >> uint(7-(i&7))) & 1)
}

// read32 : 32 bits MSB-first a l'offset bit o.
func read32(p []byte, o int) uint32 {
	var v uint32
	for i := 0; i < 32; i++ {
		v = (v << 1) | bitAt(p, o+i)
	}
	return v
}

func bswap(v uint32) uint32 {
	return (v&0xff)<<24 | (v&0xff00)<<8 | (v&0xff0000)>>8 | (v&0xff000000)>>24
}

// readF32LE : lit float32 apres byteswap (comme positions.go readFloat32LE).
func readF32LE(p []byte, o int) float32 { return math.Float32frombits(bswap(read32(p, o))) }

// readF32BE : lit float32 direct MSB-first (comme readRawVec3).
func readF32BE(p []byte, o int) float32 { return math.Float32frombits(read32(p, o)) }

// boite oracle (marge legere).
func inBox(x, y, z float32) bool {
	return x >= -8 && x <= 38 && y >= -27 && y <= 29 && z >= -6 && z <= 9
}

// real3D : vrai vec3 non-degenere = 3 axes de magnitude reelle (>0.3), aucun near-zero,
// dans la boite. Elimine les faux positifs (x,0,0)/(0,y,0) triviaux.
func real3D(x, y, z float32) bool {
	if !inBox(x, y, z) {
		return false
	}
	for _, v := range []float32{x, y, z} {
		a := math.Abs(float64(v))
		if a < 0.3 { // axe quasi-nul = degenere
			return false
		}
	}
	return true
}
func finite(v float32) bool { return !math.IsNaN(float64(v)) && !math.IsInf(float64(v), 0) }

type sweepRes struct {
	off      int
	inbox    int
	distinct int
}

func main() {
	pay := framePayload(inflate(film+"/chunk_02.bin"), 2)
	if pay == nil {
		fmt.Println("pas de payload type-2")
		return
	}
	fmt.Printf("payload type-2 chunk_02 = %d octets (%d bits)\n\n", len(pay), len(pay)*8)

	slots := []int{512, 513, 514, 515, 516, 517, 518, 519}

	for _, endian := range []string{"LE", "BE"} {
		rd := readF32LE
		if endian == "BE" {
			rd = readF32BE
		}
		fmt.Printf("========== ENDIAN=%s ==========\n", endian)
		var best []sweepRes
		// on balaye l'offset relatif au stateBit ; on essaie aussi relatif au hdrBit.
		for _, anchor := range []string{"stateBit", "hdrBit"} {
			base := stateBits
			if anchor == "hdrBit" {
				base = hdrBits
			}
			fmt.Printf("--- ancre=%s : offsets avec inbox>=6 ---\n", anchor)
			hit := 0
			for off := -64; off <= 2400; off++ {
				real := 0
				seen := map[string]bool{}
				for _, s := range slots {
					o := base[s] + off
					x, y, z := rd(pay, o), rd(pay, o+32), rd(pay, o+64)
					if finite(x) && finite(y) && finite(z) && real3D(x, y, z) {
						real++
						seen[fmt.Sprintf("%.2f_%.2f_%.2f", x, y, z)] = true
					}
				}
				if real >= 3 {
					fmt.Printf("  ancre=%s off=%+d real3D=%d/8 distinct=%d\n", anchor, off, real, len(seen))
					hit++
					if endian == "LE" && anchor == "stateBit" {
						best = append(best, sweepRes{off, real, len(seen)})
					}
				}
			}
			if hit == 0 {
				fmt.Printf("  (aucun offset real3D>=3 pour ancre=%s)\n", anchor)
			}
		}
		// detail des meilleurs offsets LE stateBit
		if endian == "LE" {
			for _, b := range best {
				if b.inbox >= 5 {
					fmt.Printf("\n>>> PIN CANDIDAT off=%+d (real3D=%d distinct=%d) :\n", b.off, b.inbox, b.distinct)
					for _, s := range slots {
						o := stateBits[s] + b.off
						x, y, z := readF32LE(pay, o), readF32LE(pay, o+32), readF32LE(pay, o+64)
						fmt.Printf("    slot=%d (%.3f, %.3f, %.3f)\n", s, x, y, z)
					}
				}
			}
		}
		fmt.Println()
	}
}
