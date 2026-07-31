// tmp_kf_timeline — Option A (propre) : timeline d'arme par joueur via les 26 keyframes
// type-2 (un /20s). Chaque keyframe porte l'arme de chaque joueur en littéral id64 complet
// (comme les 8 loadouts). On scanne chaque keyframe, on groupe les littéraux en records
// (1 record = 1 joueur), et on sort l'arme PRIMAIRE par record et par temps.
//
// Usage : tmp_kf_timeline
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)

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

func extractType2(d []byte) ([]byte, uint64) {
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		if typ == 2 {
			return d[off+16 : off+16+sz], ts
		}
		off += 16 + sz
	}
	return nil, 0
}

func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

var h32set = map[uint32]bool{}
var id64name = map[uint64]string{}

func buildSets() {
	for id, n := range analysis.WeaponIDToName {
		h32set[uint32(id>>32)] = true
		id64name[id] = n
	}
}

type lit struct {
	bit  int
	name string
}

// findLits : tous les littéraux d'arme COMPLETS (id64) du payload.
func findLits(payload []byte) []lit {
	var out []lit
	total := len(payload) * 8
	for bp := 0; bp+64 <= total; bp++ {
		h := uint32(bitsAt(payload, bp, 32))
		if !h32set[h] {
			continue
		}
		low := uint32(bitsAt(payload, bp+32, 32))
		if n, ok := id64name[(uint64(h)<<32)|uint64(low)]; ok {
			out = append(out, lit{bp, n})
			bp += 63
		}
	}
	return out
}

// group : nouveau record si gap > 1000 bits (1 record = 1 joueur ; 1ère arme = primaire).
func group(lits []lit) [][]lit {
	var g [][]lit
	for _, l := range lits {
		if len(g) == 0 || l.bit-g[len(g)-1][len(g[len(g)-1])-1].bit > 1000 {
			g = append(g, []lit{l})
		} else {
			g[len(g)-1] = append(g[len(g)-1], l)
		}
	}
	return g
}

func main() {
	buildSets()
	fmt.Printf("=== Timeline d'arme primaire par record (8 joueurs), 26 keyframes /20s ===\n")
	fmt.Printf("%-8s %-4s %s\n", "t(s)", "n", "armes primaires par record (ordre bit)")
	for n := 1; n <= 26; n++ {
		kf, ts := extractType2(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n)))
		if kf == nil {
			continue
		}
		groups := group(findLits(kf))
		var prims []string
		for _, gr := range groups {
			prims = append(prims, gr[0].name)
		}
		fmt.Printf("%-8.1f %-4d %v\n", float64(int64(ts)-int64(t0Us))/1e6, len(groups), prims)
	}

	// vue par record-index : l'arme de chaque record au fil du temps (si l'ordre est stable)
	fmt.Printf("\n=== Vue par record-index (si l'ordre des records est stable = par joueur) ===\n")
	perRec := map[int][]string{}
	var times []float64
	for n := 1; n <= 26; n++ {
		kf, ts := extractType2(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n)))
		if kf == nil {
			continue
		}
		times = append(times, float64(int64(ts)-int64(t0Us))/1e6)
		groups := group(findLits(kf))
		for i, gr := range groups {
			perRec[i] = append(perRec[i], gr[0].name)
		}
	}
	var ks []int
	for k := range perRec {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	for _, k := range ks {
		fmt.Printf("  record#%d : %v\n", k, perRec[k])
	}
}
