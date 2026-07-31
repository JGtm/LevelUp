// tmp_d2hist — THROWAWAY : compare l'histogramme TOTAL des armes dans le flux 0xd2 OFFLINE
// vs le flux dégât LIVE (ground-truth dual-hook), + raisons de rejet du filtre 0xd2.
// But : tester si le biais BR75 vient d'un sous-comptage MA40 dans le décode 0xd2 (bug données)
// ou de l'attribution/warp. Usage : CGO_ENABLED=0 go run ./cmd/tmp_d2hist <matchID>
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

const root = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`
const liveDir = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce`
const sfx = uint32(0x42c9679f)

var h32 = map[uint32]string{}

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
func idxD(h uint32) int {
	if h < 0xEC500000 || h > 0xEC600000 {
		return -1
	}
	return int((h - 0xEC500000) / 0x10002)
}

func printHist(title string, h map[string]int) {
	type kv struct {
		k string
		v int
	}
	var s []kv
	tot := 0
	for k, v := range h {
		s = append(s, kv{k, v})
		tot += v
	}
	sort.Slice(s, func(i, j int) bool { return s[i].v > s[j].v })
	fmt.Printf("=== %s (total %d) ===\n", title, tot)
	for _, e := range s {
		fmt.Printf("  %-18s %d\n", e.k, e.v)
	}
}

func main() {
	m := "9b191a7f"
	if len(os.Args) > 1 {
		m = os.Args[1]
	}
	for id := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = analysis.WeaponIDToName[id]
	}
	cache := root + "/" + m

	// --- OFFLINE 0xd2 ---
	offHist := map[string]int{}
	rawD2, noFam, noSfx := 0, 0, 0
	famNoSfx := map[string]int{} // f reconnue mais suffixe KO (perte par arme)
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		off := 0
		for off+16 <= len(d) {
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			typ := binary.LittleEndian.Uint16(d[off:])
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := d[off+16 : off+16+sz]
			off += 16 + sz
			if typ != 0 || len(pl) == 0 || pl[0] != 0xd2 {
				continue
			}
			rawD2++
			bp := 41
			if bitsAt(pl, bp, 1) == 1 {
				bp++
			} else {
				bp += 3
			}
			f := uint32(bitsAt(pl, bp, 32))
			nm, ok := h32[f]
			sfxOK := uint32(bitsAt(pl, bp+32, 32)) == sfx
			switch {
			case ok && sfxOK:
				offHist[nm]++
			case ok && !sfxOK:
				noSfx++
				famNoSfx[nm]++
			default:
				noFam++
			}
		}
	}
	fmt.Printf("match %s : %d records 0xd2 bruts ; gardés=%d ; rejet noFam=%d noSfx=%d\n\n",
		m, rawD2, offHistTotal(offHist), noFam, noSfx)
	printHist("OFFLINE 0xd2 (filtre actuel)", offHist)
	if len(famNoSfx) > 0 {
		fmt.Println()
		printHist("0xd2 famille RECONNUE mais suffixe KO (pertes)", famNoSfx)
	}

	// --- LIVE damage ground-truth ---
	dd, e1 := os.ReadFile(liveDir + "/" + m + "_dmg.bin")
	if e1 != nil {
		fmt.Printf("\npas de capture live %s_dmg.bin\n", m)
		return
	}
	liveHist := map[string]int{}
	for o := 0; o+32 <= len(dd); o += 32 {
		if idxD(binary.LittleEndian.Uint32(dd[o:])) < 0 {
			continue
		}
		liveHist[h32[binary.LittleEndian.Uint32(dd[o+8:])]]++
	}
	fmt.Println()
	printHist("LIVE damage (dual-hook ground-truth)", liveHist)
}

// offHistTotal renvoie le total (helper trivial pour le print de tête).
func offHistTotal(h map[string]int) int {
	t := 0
	for _, v := range h {
		t += v
	}
	return t
}
