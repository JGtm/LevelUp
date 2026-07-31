// tmp_fireevents — THROWAWAY : weapon events byte-alignés + TEMPS du paquet + tag.
// But : voir si ces events sont timés et corrèlent aux kills (focus marteau = kills IKE narrés).
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

var h32 = map[uint32]string{}

func buildCat() {
	for id, n := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = n
	}
}

// tsAt : ts (ms depuis t0) du paquet type-0/2 contenant l'offset abs `pos`.
func tsAt(d []byte, pos int) (int, bool) {
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

func hx(d []byte) string {
	s := ""
	for _, b := range d {
		s += fmt.Sprintf("%02x", b)
	}
	return s
}

type evt struct {
	tms int
	wpn string
	tag uint16 // 2 octets après le compteur
	ctx string
}

func main() {
	buildCat()
	var evts []evt
	tagCount := map[uint16]int{}
	wpnCount := map[string]int{}
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		for i := 0; i+8 <= len(d); i++ {
			hi := binary.BigEndian.Uint32(d[i:])
			name, ok := h32[hi]
			if !ok {
				continue
			}
			// suffixe attendu : low32 connu (42c9679f le plus souvent) — on accepte tout low32, weapon byte-aligné.
			tms, okt := tsAt(d, i)
			if !okt {
				continue
			}
			var tag uint16
			if i+12 <= len(d) {
				tag = binary.BigEndian.Uint16(d[i+10:]) // [id8][cnt2][TAG2]
			}
			lo := i - 8
			if lo < 0 {
				lo = 0
			}
			r := i + 20
			if r > len(d) {
				r = len(d)
			}
			evts = append(evts, evt{tms, name, tag, fmt.Sprintf("before=%s [%s] after=%s", hx(d[lo:i]), hx(d[i:i+8]), hx(d[i+8:r]))})
			tagCount[tag]++
			wpnCount[name]++
			i += 7
		}
	}
	sort.Slice(evts, func(a, b int) bool { return evts[a].tms < evts[b].tms })
	fmt.Printf("=== %d weapon events byte-alignés (tous chunks), 13.8→481s ===\n", len(evts))
	fmt.Printf("\n--- distribution armes ---\n")
	type kv struct {
		k string
		v int
	}
	var aw []kv
	for k, v := range wpnCount {
		aw = append(aw, kv{k, v})
	}
	sort.Slice(aw, func(a, b int) bool { return aw[a].v > aw[b].v })
	for _, e := range aw {
		fmt.Printf("  %-22s x%d\n", e.k, e.v)
	}
	fmt.Printf("\n--- distribution tag (2o après compteur) ---\n")
	var at []kv
	for k, v := range tagCount {
		at = append(at, kv{fmt.Sprintf("0x%04x", k), v})
	}
	_ = at
	for k, v := range tagCount {
		fmt.Printf("  0x%04x x%d\n", k, v)
	}

	// FOCUS MARTEAU (high32 0x841ac5e5) vs kills IKE narrés (115.5/292.5/355.7/375.1s)
	fmt.Printf("\n=== MARTEAU (Gravity/Rushdown Hammer) events timés ===\n")
	for _, e := range evts {
		if e.wpn == "Rushdown Hammer" || e.wpn == "Gravity Hammer" {
			fmt.Printf("  t=%7.1fs  %-16s tag=0x%04x  %s\n", float64(e.tms)/1000, e.wpn, e.tag, e.ctx)
		}
	}
	fmt.Printf("  (kills marteau IKE->JGtm narrés : 115.5 / 292.5 / 355.7 / 375.1 s)\n")
}
