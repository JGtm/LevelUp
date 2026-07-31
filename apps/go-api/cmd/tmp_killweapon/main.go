// tmp_killweapon — THROWAWAY : voie KNOWN-PLAINTEXT (REPRISE_KILLWEAPON).
// Ancre sur les kill events de chunk_27 (time_ms + killer xuid), calcule le ts gameplay
// correspondant, et scanne les paquets du flux gameplay à cet instant pour un 'obje'
// arme (high-32 famille). But : l'arme du kill feed apparaît-elle près du kill, et est-ce
// un id GLOBAL (matche WeaponIDToName) ou un handle per-match ?
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
const jgtmXUID = uint64(2533274823110022)

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
func readByteAtBit(d []byte, bit int) byte {
	if bit < 0 || bit+8 > len(d)*8 {
		return 0
	}
	bi := bit / 8
	off := uint(bit % 8)
	if off == 0 {
		return d[bi]
	}
	return (d[bi] << off) | (d[bi+1] >> (8 - off))
}
func readBitsU32(d []byte, bit int) uint32 {
	var v uint32
	for i := 0; i < 32; i++ {
		bp := bit + i
		v = (v << 1) | uint32((d[bp>>3]>>uint(7-(bp&7)))&1)
	}
	return v
}
func readU64LEAtBit(d []byte, bit int) uint64 {
	var x uint64
	for i := 0; i < 8; i++ {
		x |= uint64(readByteAtBit(d, bit+i*8)) << (uint(i) * 8)
	}
	return x
}

type pkt struct {
	chunk string
	typ   uint16
	ts    uint64
	data  []byte
}

func listPackets(name string, d []byte) []pkt {
	var out []pkt
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		out = append(out, pkt{name, typ, ts, d[off+16 : off+16+sz]})
		off += 16 + sz
	}
	return out
}

// kill events JGtm depuis chunk_27
func jgtmKillTimes(d []byte) []int {
	tot := len(d) * 8
	em := []byte{0x00, 0x00, 0x2e, 0xe0}
	findEnd := func(from int) int {
		for b := from; b <= tot-32; b++ {
			if readByteAtBit(d, b) == em[0] && readByteAtBit(d, b+8) == em[1] && readByteAtBit(d, b+16) == em[2] && readByteAtBit(d, b+24) == em[3] {
				return b
			}
		}
		return -1
	}
	var times []int
	seen := map[int]bool{}
	for ms := 8; ms <= tot-8; ms++ {
		if readByteAtBit(d, ms) != 0xc0 {
			continue
		}
		xe := ms - 8
		if xe < 64 {
			continue
		}
		pre := readByteAtBit(d, xe)
		if pre != 0x2d && pre != 0x25 {
			continue
		}
		xs := xe - 64
		if seen[xs] {
			continue
		}
		x := readU64LEAtBit(d, xs)
		if x != jgtmXUID {
			continue
		}
		seen[xs] = true
		endBit := findEnd(xs)
		if endBit < 0 || endBit-60*8 < xs {
			continue
		}
		blk := make([]byte, 60)
		for i := 0; i < 60; i++ {
			blk[i] = readByteAtBit(d, endBit-60*8+i*8)
		}
		if int(blk[47]) == 50 { // kill
			times = append(times, int(binary.BigEndian.Uint32(blk[48:52])))
		}
	}
	sort.Ints(times)
	return times
}

func main() {
	h2n := map[uint32]string{}
	for id, n := range analysis.WeaponIDToName {
		h2n[uint32(id>>32)] = n
	}
	// tous les paquets + ref ts (chunk_02 type-2 = t0)
	var all []pkt
	var t0 uint64
	for n := 0; n <= 27; n++ {
		name := fmt.Sprintf("chunk_%02d.bin", n)
		d := inflate(cache + "/" + name)
		ps := listPackets(name, d)
		all = append(all, ps...)
		if n == 2 {
			for _, p := range ps {
				if p.typ == 2 {
					t0 = p.ts
					break
				}
			}
		}
	}
	fmt.Printf("t0 (chunk_02 type-2 ts) = %d ; %d paquets\n", t0, len(all))

	kt := jgtmKillTimes(inflate(cache + "/chunk_27.bin"))
	fmt.Printf("JGtm kills (ms) : %v\n", kt)

	// pour chaque kill : ts cible = t0 + ms*1000 ; paquets dans +-400ms ; scan weapons
	for _, ms := range kt {
		target := int64(t0) + int64(ms)*1000
		fmt.Printf("\n=== kill t=%dms (%.1fs)  target_ts=%d ===\n", ms, float64(ms)/1000, target)
		type wh struct {
			name string
			typ  uint16
			dms  int64
		}
		var hits []wh
		seen := map[string]bool{}
		for _, p := range all {
			d := int64(p.ts) - target
			if d < -400000 || d > 400000 {
				continue
			}
			tot := len(p.data) * 8
			for bp := 0; bp+32 <= tot; bp++ {
				if name, ok := h2n[readBitsU32(p.data, bp)]; ok {
					key := fmt.Sprintf("%s|%d|%d", name, p.typ, d/1000)
					if !seen[key] {
						seen[key] = true
						hits = append(hits, wh{name, p.typ, d / 1000})
					}
				}
			}
		}
		sort.Slice(hits, func(i, j int) bool {
			if hits[i].dms != hits[j].dms {
				return hits[i].dms < hits[j].dms
			}
			return hits[i].name < hits[j].name
		})
		if len(hits) == 0 {
			fmt.Println("  (aucun 'obje' arme global dans +-400ms)")
		}
		for _, h := range hits {
			fmt.Printf("  %+4dms  type-%-2d  %s\n", h.dms, h.typ, h.name)
		}
	}
}
