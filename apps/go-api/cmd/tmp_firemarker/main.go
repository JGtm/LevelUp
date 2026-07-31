// tmp_firemarker — THROWAWAY (A.1) : densifier la capture FIRE.
//  1. Combien d'occurrences high32∈catalogue en paquets type-0, par low32 (suffixe variant) ?
//     -> si beaucoup de low32 != 0x42c9679f, l'ancre stricte rate des fire events.
//  2. Carte des bits CONSTANTS autour des ancres fire strictes (high32∈cat && low32==0x42c9679f)
//     -> révèle un marqueur de fire event court (co-localisation, comme grenade 0x4c0c00 / melee 0x534).
//  3. Le champ bp-4 (player_index) est-il borné 0-7 sur l'ancre élargie ?
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
const maxMatchMs = 600000
const variantSuffix = uint32(0x42c9679f)

var h32 = map[uint32]string{}

func build() {
	for id, n := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = n
	}
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
func bitsAt(p []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		q := bp + i
		if q>>3 >= len(p) || q < 0 {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((p[q>>3]>>uint(7-(q&7)))&1)
	}
	return v
}

type pkt struct {
	typ     uint16
	ts      uint64
	payload []byte
}

func type0Packets() []pkt {
	var out []pkt
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			size := int(binary.LittleEndian.Uint32(d[off+4:]))
			ts := binary.LittleEndian.Uint64(d[off+8:])
			if size <= 0 || off+16+size > len(d) {
				break
			}
			if typ == 0 && ts >= t0Us {
				tms := int((ts - t0Us) / 1000)
				if tms >= 0 && tms <= maxMatchMs {
					out = append(out, pkt{typ, ts, d[off+16 : off+16+size]})
				}
			}
			off += 16 + size
		}
	}
	return out
}

func main() {
	build()
	pkts := type0Packets()
	fmt.Printf("=== %d paquets type-0 (ts-valides) ===\n", len(pkts))

	// 1) low32 distribution pour high32∈catalogue
	lowDist := map[uint32]int{}
	strictAnchors := 0
	type anchor struct {
		pl []byte
		bp int
	}
	var strict []anchor
	for _, p := range pkts {
		pl := p.payload
		total := len(pl) * 8
		for bp := 0; bp+64 <= total; bp++ {
			hi := uint32(bitsAt(pl, bp, 32))
			if _, ok := h32[hi]; !ok {
				continue
			}
			low := uint32(bitsAt(pl, bp+32, 32))
			lowDist[low]++
			if low == variantSuffix {
				strictAnchors++
				if len(strict) < 4000 {
					strict = append(strict, anchor{pl, bp})
				}
			}
		}
	}
	fmt.Printf("\n=== (1) low32 (suffixe variant) après high32∈catalogue [type-0] ===\n")
	type lc struct {
		l uint32
		c int
	}
	var lcs []lc
	for l, c := range lowDist {
		lcs = append(lcs, lc{l, c})
	}
	sort.Slice(lcs, func(i, j int) bool { return lcs[i].c > lcs[j].c })
	tot := 0
	for _, x := range lcs {
		tot += x.c
	}
	fmt.Printf("  total high32∈cat = %d ; dont low32==0x42c9679f = %d (%.0f%%)\n", tot, strictAnchors, 100*float64(strictAnchors)/float64(tot))
	for i, x := range lcs {
		if i >= 12 {
			fmt.Printf("  ... (%d valeurs low32 distinctes)\n", len(lcs))
			break
		}
		fmt.Printf("  low32=0x%08x x%d\n", x.l, x.c)
	}

	// 2) carte des bits constants autour des ancres strictes : pour chaque offset o dans [-48,+160],
	//    fraction de bits == 1 ; un offset constant (≈0 ou ≈1) = structure/marqueur.
	fmt.Printf("\n=== (2) bits constants autour de l'ancre fire stricte (n=%d) ===\n", len(strict))
	const lo, hi = -48, 160
	ones := make([]int, hi-lo)
	for _, a := range strict {
		for i := lo; i < hi; i++ {
			if bitsAt(a.pl, a.bp+i, 1) == 1 {
				ones[i-lo]++
			}
		}
	}
	n := float64(len(strict))
	// afficher les runs constants (>95% identiques) et marquer la zone weapon_id [0,64)
	fmt.Printf("  offset: frac1  (| = ancre weapon_id 0..63)\n")
	runStart, runVal := -999, -1
	flush := func(end int) {
		if runStart > -999 && end-runStart >= 3 {
			fmt.Printf("  CONSTANT [%+d..%+d] = %d (len %d)\n", runStart, end-1, runVal, end-runStart)
		}
		runStart, runVal = -999, -1
	}
	for i := lo; i < hi; i++ {
		f := float64(ones[i-lo]) / n
		cv := -1
		if f >= 0.97 {
			cv = 1
		} else if f <= 0.03 {
			cv = 0
		}
		if i >= 0 && i < 64 {
			cv = -2 // zone weapon_id, ignore
		}
		if cv != runVal {
			flush(i)
			if cv == 0 || cv == 1 {
				runStart, runVal = i, cv
			}
		}
	}
	flush(hi)

	// 3) champ bp-4 (player_index) borné 0-7 ? distribution sur ancres strictes
	fmt.Printf("\n=== (3) champ bp-4 (5b) sur ancres strictes : distribution ===\n")
	pidxDist := map[int]int{}
	for _, a := range strict {
		pidxDist[int(bitsAt(a.pl, a.bp-4, 5))]++
	}
	for v := 0; v < 32; v++ {
		if pidxDist[v] > 0 {
			tag := ""
			if v <= 7 {
				tag = " (slot valide)"
			}
			fmt.Printf("  v=%2d x%d%s\n", v, pidxDist[v], tag)
		}
	}

	// 3b) bp-4 sur l'ancre ÉLARGIE (tout low32) : les 4 joueurs manquants (0,1,4,5) apparaissent-ils ?
	fmt.Printf("\n=== (3b) bp-4 (5b) sur ancre ÉLARGIE (high32∈cat, tout low32) ===\n")
	wideP := map[int]int{}
	for _, p := range pkts {
		pl := p.payload
		total := len(pl) * 8
		for bp := 4; bp+64 <= total; bp++ {
			hi := uint32(bitsAt(pl, bp, 32))
			if _, ok := h32[hi]; !ok {
				continue
			}
			wideP[int(bitsAt(pl, bp-4, 5))]++
		}
	}
	for v := 0; v < 8; v++ {
		fmt.Printf("  v=%d x%d\n", v, wideP[v])
	}

	// 4) dump brut autour de 6 ancres strictes consécutives (hex des bits -48..+160) pour inspection
	fmt.Printf("\n=== (4) dump bits -16..+96 de 8 ancres strictes (lecture marqueur) ===\n")
	for i := 0; i < len(strict) && i < 8; i++ {
		a := strict[i]
		pre := bitsAt(a.pl, a.bp-16, 16)
		wid := uint32(bitsAt(a.pl, a.bp, 32))
		post := uint32(bitsAt(a.pl, a.bp+64, 32))
		fmt.Printf("  pre16=0x%04x | wid32=0x%08x(%s) | post32=0x%08x\n", pre, wid, h32[wid], post)
	}
}
