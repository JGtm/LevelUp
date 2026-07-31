// tmp_wpnscan — THROWAWAY : balaie TOUS les chunks / toutes les keyframes (type-1 et type-2)
// pour le high-32 de chaque arme cataloguée (ignore le suffixe variant). Reporte par
// keyframe : timestamp du paquet + multiset d'armes (high-32) trouvées + positions bit.
//
// But : situer la keyframe de SPAWN (multiset == armes de spawn observées in-game) et
// celle vers 5:55 (BR75 = Frag Parfait de JGtm), puis tester l'hypothèse "ordre des
// littéraux arme == ordre des joueurs".
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

func inflate(path string) []byte {
	raw, _ := os.ReadFile(path)
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

type packet struct {
	chunk string
	typ   uint16
	ts    uint64
	off   int
	size  int
	data  []byte
}

// listPackets découpe un blob inflaté en paquets [Type u16][b2 b3][Size u32][Ts u64][payload].
func listPackets(chunk string, data []byte) []packet {
	var out []packet
	off := 0
	for off+16 <= len(data) {
		typ := binary.LittleEndian.Uint16(data[off:])
		size := int(binary.LittleEndian.Uint32(data[off+4:]))
		ts := binary.LittleEndian.Uint64(data[off+8:])
		if size <= 0 || off+16+size > len(data) {
			break
		}
		out = append(out, packet{chunk, typ, ts, off, size, data[off+16 : off+16+size]})
		off += 16 + size
	}
	return out
}

// readBits lit n bits MSB-first à partir du bit bitPos (big-endian bit order).
func readBitsAt(data []byte, bitPos, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		bp := bitPos + i
		byteIdx := bp >> 3
		if byteIdx >= len(data) {
			return v
		}
		bit := (data[byteIdx] >> uint(7-(bp&7))) & 1
		v = (v << 1) | uint64(bit)
	}
	return v
}

type hit struct {
	bitPos int
	name   string
	high32 uint32
}

func main() {
	// high-32 -> nom (famille), dédupliqué
	high2name := map[uint32]string{}
	for id, name := range analysis.WeaponIDToName {
		high2name[uint32(id>>32)] = name
	}
	// armes de spawn observées (pour le tag "GROUND TRUTH")
	gt := map[uint32]string{
		0x71ab0a2c: "M41", 0x230447b1: "Cindershot", 0x9387a8b9: "ShockRifle", 0x1a22fee6: "ShockRifleR",
		0x841ac5e5: "GravHammer", 0x767db96d: "Hydra", 0xb619d84a: "Bulldog",
		0xc30d87c7: "Ravager", 0x0d20c469: "Skewer", 0x2b1824d5: "BR75",
	}

	files, _ := filepath.Glob(cache + "/chunk_*.bin")
	sort.Strings(files)

	var allPackets []packet
	for _, f := range files {
		name := filepath.Base(f)
		pkts := listPackets(name, inflate(f))
		allPackets = append(allPackets, pkts...)
	}

	// Résumé des paquets type-1 / type-2 (keyframes) triés par ts
	var kf []packet
	for _, p := range allPackets {
		if p.typ == 1 || p.typ == 2 {
			kf = append(kf, p)
		}
	}
	sort.Slice(kf, func(i, j int) bool { return kf[i].ts < kf[j].ts })

	fmt.Printf("=== %d keyframes (type-1/2) sur %d paquets ===\n", len(kf), len(allPackets))
	for _, p := range kf {
		// scan high-32 sur tous les bits
		hits := scan(p.data, high2name)
		// multiset
		counts := map[string]int{}
		gtCount := 0
		for _, h := range hits {
			counts[h.name]++
			if _, ok := gt[h.high32]; ok {
				gtCount++
			}
		}
		fmt.Printf("\n%-12s type=%d ts=%d size=%d  -> %d hits armes (%d ground-truth)\n",
			p.chunk, p.typ, p.ts, p.size, len(hits), gtCount)
		// tri par nom
		var names []string
		for n := range counts {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			fmt.Printf("    %-26s x%d\n", n, counts[n])
		}
	}

	// Table ts -> secondes (réf = 1er type-2 avec armes = spawn). Ratio supposé 1e6 ts/s.
	fmt.Println("\n=== type-2 keyframes : ts -> ~secondes (réf spawn) ===")
	var t2 []packet
	for _, p := range kf {
		if p.typ == 2 {
			t2 = append(t2, p)
		}
	}
	var ref uint64
	for _, p := range t2 {
		if len(scan(p.data, high2name)) > 0 {
			ref = p.ts
			break
		}
	}
	for _, p := range t2 {
		secs := float64(int64(p.ts)-int64(ref)) / 1e6
		fmt.Printf("  %-12s ts=%d  ~%6.1fs  (%d armes)\n", p.chunk, p.ts, secs, len(scan(p.data, high2name)))
	}

	// Dump ordonné + gaps pour chunk_02 (spawn-like) et la keyframe ~355s (Frag Parfait).
	for _, target := range []string{"chunk_02.bin", "chunk_20.bin", "chunk_19.bin"} {
		for _, p := range t2 {
			if p.chunk != target {
				continue
			}
			fmt.Printf("\n=== %s ts=%d : 16 littéraux ordonnés + gaps (pairing) ===\n", p.chunk, p.ts)
			hits := scan(p.data, high2name)
			sort.Slice(hits, func(i, j int) bool { return hits[i].bitPos < hits[j].bitPos })
			prev := 0
			for i, h := range hits {
				gap := h.bitPos - prev
				prev = h.bitPos
				tag := ""
				if g, ok := gt[h.high32]; ok {
					tag = "  <GT:" + g + ">"
				}
				fmt.Printf("    [%2d] bit%-8d (+%-6d) %-26s%s\n", i, h.bitPos, gap, h.name, tag)
			}
		}
	}
}

func scan(data []byte, high2name map[uint32]string) []hit {
	var hits []hit
	totalBits := len(data) * 8
	for bp := 0; bp+32 <= totalBits; bp++ {
		v := uint32(readBitsAt(data, bp, 32))
		if name, ok := high2name[v]; ok {
			hits = append(hits, hit{bp, name, v})
		}
	}
	return hits
}
