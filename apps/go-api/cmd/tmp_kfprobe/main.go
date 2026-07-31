// tmp_kfprobe — THROWAWAY : pourquoi le comb-anchor ne rend que ~3 positions par
// keyframe alors qu'il y a 8 joueurs ? Dump des hits comb bruts (avant filtres) +
// un scan "triplet float32 plausible" (3 float consécutifs en range carte) pour
// voir combien de positions sont réellement présentes dans UN keyframe.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_kfprobe [chunk.bin]
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
)

const defChunk = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950/chunk_02.bin`

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

func bitAt(p []byte, i int) int {
	if i < 0 || i>>3 >= len(p) {
		return 0
	}
	return int((p[i>>3] >> uint(7-(i&7))) & 1)
}

func readF32LE(p []byte, o int) float32 {
	var v uint32
	for i := 0; i < 32; i++ {
		v = (v << 1) | uint32(bitAt(p, o+i))
	}
	sw := (v&0xff)<<24 | (v&0xff00)<<8 | (v&0xff0000)>>8 | (v&0xff000000)>>24
	return math.Float32frombits(sw)
}

// combAt : motif 1⁸0¹⁶ ×4 (le comb du package positions).
func combAt(p []byte, bp int) bool {
	if bp < 0 || (bp+96)>>3 >= len(p) {
		return false
	}
	for rep := 0; rep < 4; rep++ {
		base := bp + rep*24
		for i := 0; i < 8; i++ {
			if bitAt(p, base+i) != 1 {
				return false
			}
		}
		for i := 8; i < 24; i++ {
			if bitAt(p, base+i) != 0 {
				return false
			}
		}
	}
	return true
}

// combRelaxed : variantes ×3, ×2 du motif (combien de hits si on relâche ?).
func combReps(p []byte, bp, reps int) bool {
	if bp < 0 || (bp+reps*24)>>3 >= len(p) {
		return false
	}
	for rep := 0; rep < reps; rep++ {
		base := bp + rep*24
		for i := 0; i < 8; i++ {
			if bitAt(p, base+i) != 1 {
				return false
			}
		}
		for i := 8; i < 24; i++ {
			if bitAt(p, base+i) != 0 {
				return false
			}
		}
	}
	return true
}

func plausible(x float32) bool {
	a := float64(x)
	return !math.IsNaN(a) && !math.IsInf(a, 0) && a > -60 && a < 60
}

func main() {
	chunk := defChunk
	if len(os.Args) > 1 {
		chunk = os.Args[1]
	}
	payload := extractType2(inflate(chunk))
	if payload == nil {
		fmt.Println("pas de type-2")
		return
	}
	maxBit := len(payload) * 8
	fmt.Printf("keyframe type-2 : %d octets (%d bits)\n\n", len(payload), maxBit)

	// (A) comb ×4 (strict) : combien de hits + triplet décodé.
	fmt.Println("=== (A) comb ×4 STRICT (positions package) ===")
	nStrict := 0
	for bp := 0; bp+96 <= maxBit; bp++ {
		if combAt(payload, bp) {
			o := bp - 273
			x, y, z := readF32LE(payload, o), readF32LE(payload, o+32), readF32LE(payload, o+64)
			fmt.Printf("  comb@%-8d triplet@%-8d (%.2f, %.2f, %.2f)\n", bp, o, x, y, z)
			nStrict++
			bp += 95
		}
	}
	fmt.Printf("  => %d hits ×4\n\n", nStrict)

	// (B) comb ×3 / ×2 : combien de hits supplémentaires ?
	for _, reps := range []int{3, 2} {
		n := 0
		for bp := 0; bp+reps*24 <= maxBit; bp++ {
			if combReps(payload, bp, reps) {
				n++
				bp += reps*24 - 1
			}
		}
		fmt.Printf("=== (B) comb ×%d : %d hits ===\n", reps, n)
	}
	fmt.Println()

	// (C) scan "triplet float32 LE plausible" aligné octet : 3 floats consécutifs
	// dans [-60,60], magnitude > 1, pas tous quasi-nuls. Combien de candidats ?
	fmt.Println("=== (C) triplets float32-LE plausibles (aligné octet) ===")
	type trip struct {
		off     int
		x, y, z float32
	}
	var trips []trip
	for o := 0; o+12 <= len(payload); o++ {
		x := math.Float32frombits(binary.LittleEndian.Uint32(payload[o:]))
		y := math.Float32frombits(binary.LittleEndian.Uint32(payload[o+4:]))
		z := math.Float32frombits(binary.LittleEndian.Uint32(payload[o+8:]))
		if !(plausible(x) && plausible(y) && plausible(z)) {
			continue
		}
		mag := math.Abs(float64(x)) + math.Abs(float64(y)) + math.Abs(float64(z))
		if mag < 2 || mag > 120 {
			continue
		}
		// rejette les triplets à composantes trop "rondes" (souvent des constantes)
		trips = append(trips, trip{o, x, y, z})
	}
	fmt.Printf("  %d triplets candidats (aligné octet)\n", len(trips))
	// clusterise grossièrement par proximité pour estimer le nb d'entités distinctes
	sort.Slice(trips, func(i, j int) bool { return trips[i].off < trips[j].off })
	shown := 0
	for _, t := range trips {
		if shown < 40 {
			fmt.Printf("    @%-7d (%.2f, %.2f, %.2f)\n", t.off, t.x, t.y, t.z)
			shown++
		}
	}
}
