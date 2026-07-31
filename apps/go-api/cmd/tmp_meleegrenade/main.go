// tmp_meleegrenade — THROWAWAY : décode MELEE + GRENADE events avec PLAYER INDEX
// (recherche communautaire). Marqueurs bit-packés :
//
//	MELEE   : 0b10100110010 (11 bits) ; anchor=+3b ; player idx @+20 (5b) ; type @+76 (8b,
//	          valide 0x42/0x47/0x60) ; weapon @+86 (0x47 Hammer) / +88 (0x42) / +101/103 (0x60).
//	GRENADE : 0x4c0c00 (24 bits) ; grenade id 32b ; +47b ; player idx 5b. ids 32-bit:
//	          FRAG 0xB0171062, PLASMA 0xC0E34C44, SHOCK 0x3B2567D4, SPIKE 0x9212E428.
//
// But : tireur+arme+temps -> attribuer par tueur (chunk_27). Focus marteau IKE(pi4)@115.5s.
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

var h32 = map[uint32]string{}
var grenades = map[uint32]string{0xB0171062: "Frag", 0xC0E34C44: "Plasma", 0x3B2567D4: "Shock", 0x9212E428: "Spike"}
var pi = map[int]string{0: "whiteknight2519", 1: "JAVIERLOLITO540", 2: "JGtm", 3: "LORD PEINX13", 4: "IKE ILYA", 5: "Akatsuki fire17", 6: "aldusbroncus", 7: "VitaminA1688"}

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
func buildCat() {
	for id, n := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = n
	}
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

// tsAtBit : ts ms du paquet contenant le bit bp.
func tsAtBit(d []byte, bp int) (int, bool) {
	pos := bp >> 3
	off := 0
	for off+16 <= len(d) {
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if pos >= off+16 && pos < off+16+sz {
			return int((ts - t0Us) / 1000), true
		}
		off += 16 + sz
	}
	return -1, false
}

type ev struct {
	tms  int
	kind string
	wpn  string
	pidx int
}

func main() {
	buildCat()
	var evs []ev
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		total := len(d) * 8
		// MELEE : marqueur 11-bit 0x532
		for bp := 0; bp+120 < total; bp++ {
			if bitsAt(d, bp, 11) != 0x532 {
				continue
			}
			anchor := bp + 3
			typ := uint8(bitsAt(d, anchor+76, 8))
			woff := -1
			switch typ {
			case 0x47:
				woff = anchor + 86
			case 0x42:
				woff = anchor + 88
			case 0x60:
				woff = anchor + 101
			default:
				continue
			}
			hi := uint32(bitsAt(d, woff, 32))
			name, ok := h32[hi]
			if !ok {
				if typ == 0x60 {
					hi = uint32(bitsAt(d, anchor+103, 32))
					name, ok = h32[hi]
				}
				if !ok {
					continue
				}
			}
			pidx := int(bitsAt(d, anchor+20, 5))
			tms, okt := tsAtBit(d, bp)
			if !okt {
				continue
			}
			evs = append(evs, ev{tms, "melee", name, pidx})
		}
		// GRENADE : marqueur 24-bit 0x4c0c00
		for bp := 0; bp+110 < total; bp++ {
			if bitsAt(d, bp, 24) != 0x4c0c00 {
				continue
			}
			gid := uint32(bitsAt(d, bp+24, 32))
			gname, ok := grenades[gid]
			if !ok {
				continue
			}
			pidx := int(bitsAt(d, bp+24+32+47, 5))
			tms, okt := tsAtBit(d, bp)
			if !okt {
				continue
			}
			evs = append(evs, ev{tms, "grenade", gname + " Grenade", pidx})
		}
	}
	sort.Slice(evs, func(a, b int) bool { return evs[a].tms < evs[b].tms })
	mc, gc := 0, 0
	pidxCount := map[int]int{}
	for _, e := range evs {
		if e.kind == "melee" {
			mc++
		} else {
			gc++
		}
		pidxCount[e.pidx]++
	}
	fmt.Printf("=== MELEE=%d  GRENADE=%d events décodés (avec player index) ===\n", mc, gc)
	fmt.Printf("\n--- distribution player index (doit être 0-7 si valide) ---\n")
	for i := 0; i < 32; i++ {
		if pidxCount[i] > 0 {
			fmt.Printf("  pidx=%2d (%s) x%d\n", i, pi[i], pidxCount[i])
		}
	}
	fmt.Printf("\n=== MÊLÉE events timés (focus marteau ; IKE=pi4 ; kills narrés 115.5/292.5/355.7/375.1s) ===\n")
	for _, e := range evs {
		if e.kind == "melee" {
			fmt.Printf("  t=%7.1fs  pidx=%d(%-16s) %s\n", float64(e.tms)/1000, e.pidx, pi[e.pidx], e.wpn)
		}
	}
	fmt.Printf("\n=== GRENADE events timés ===\n")
	gn := 0
	for _, e := range evs {
		if e.kind == "grenade" {
			fmt.Printf("  t=%7.1fs  pidx=%d(%-16s) %s\n", float64(e.tms)/1000, e.pidx, pi[e.pidx], e.wpn)
			gn++
			if gn > 25 {
				break
			}
		}
	}
}
