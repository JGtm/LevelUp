// tmp_formulaa — THROWAWAY : applique les scanners v2 EXISTANTS (ScanFormulaA /
// ScanFormulaANS, weapon_scanner.go) à la keyframe + au chunk complet, pour voir s'ils
// donnent (player_index, arme) — au lieu de re-décoder à la main.
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

func wname(wb [8]byte) string {
	id := binary.BigEndian.Uint64(wb[:])
	if n, ok := analysis.WeaponIDToName[id]; ok {
		return n
	}
	// essai high-32 famille
	for fid, n := range analysis.WeaponIDToName {
		if uint32(fid>>32) == uint32(id>>32) {
			return n + "?"
		}
	}
	return fmt.Sprintf("?%016x", id)
}

func report(label string, data []byte) {
	fa := analysis.ScanFormulaA(data)
	ns := analysis.ScanFormulaANS(data)
	fmt.Printf("\n===== %s (%d octets) : FormulaA=%d  FormulaANS=%d =====\n", label, len(data), len(fa), len(ns))
	// agrège par (pi, arme)
	type key struct {
		pi int
		w  string
	}
	agg := map[key]int{}
	for _, r := range fa {
		agg[key{r.PlayerIndex, wname(r.WeaponBytes)}]++
	}
	var keys []key
	for k := range agg {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].pi != keys[j].pi {
			return keys[i].pi < keys[j].pi
		}
		return keys[i].w < keys[j].w
	})
	fmt.Println("  -- FormulaA (pi, arme) x count --")
	for _, k := range keys {
		fmt.Printf("    pi=%d  %-22s x%d\n", k.pi, k.w, agg[k])
	}
	// NS
	aggn := map[key]int{}
	for _, r := range ns {
		aggn[key{r.PlayerIndex, wname(r.WeaponBytes)}]++
	}
	keys = keys[:0]
	for k := range aggn {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].pi != keys[j].pi {
			return keys[i].pi < keys[j].pi
		}
		return keys[i].w < keys[j].w
	})
	fmt.Println("  -- FormulaANS (pi, arme) x count --")
	for _, k := range keys {
		fmt.Printf("    pi=%d  %-22s x%d\n", k.pi, k.w, aggn[k])
	}
}

func main() {
	full := inflate(cache + "/chunk_02.bin")
	report("chunk_02 FULL inflated", full)
	report("chunk_02 type-2 payload", extractType2(full))
}
