// tmp_prebits — que valent VRAIMENT les premiers bits d'un payload de frame type-0 ?
// THROWAWAY.
//
// Le balayage rend (amorce=2, idLow=11) gagnant, mais le désassemblage ne montre qu'UN
// bit avant la boucle de records (FUN_142987460 : DAT_144706104 = FUN_1406cf008(reader)).
// Trois grammaires produisent le MÊME masque au premier record (en-tête total = 20 bits) :
//
//	A  amorce 1 + type court(1) + id 11 + tag 2 + gate 1 + count 3  -> pas testé, id=bits 2..12
//	B  amorce 2 + type court(1) + id 11 + tag 2 + gate 1 + count 3  -> le « gagnant » (id bits 3..13)
//	C  amorce 1 + type LONG(3, R(2)=3) + id 10 + tag 2 + gate 1 + count 3  (id bits 4..13)
//
// B et C ne diffèrent QUE par l'interprétation des bits 2 et 3. Si le bit 3 vaut 1 dans
// (quasi) tous les payloads, ce n'est pas un bit d'identifiant mais le second bit du code
// de type long, et C est la bonne lecture. On mesure donc directement les bits 0..5.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_prebits [filmdir]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const defFilm = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

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

func listFrames(d []byte) [][]byte {
	var out [][]byte
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, d[off+16:off+16+sz])
		}
		off += 16 + sz
	}
	return out
}

func bitAt(b []byte, i int) int {
	if i/8 >= len(b) {
		return -1
	}
	return int(b[i/8]>>(7-uint(i%8))) & 1
}

func bitsAt(b []byte, i, n int) uint32 {
	var v uint32
	for k := 0; k < n; k++ {
		v = v<<1 | uint32(bitAt(b, i+k))
	}
	return v
}

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	var frames [][]byte
	for i := 1; i <= 26; i++ {
		frames = append(frames, listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, i)))...)
	}
	fmt.Printf("frames type-0 : %d\n\n", len(frames))

	fmt.Println("=== part des payloads dont le bit i vaut 1 ===")
	for i := 0; i < 8; i++ {
		n := 0
		for _, p := range frames {
			if bitAt(p, i) == 1 {
				n++
			}
		}
		fmt.Printf("bit %d : %6.2f %%\n", i, 100*float64(n)/float64(len(frames)))
	}

	fmt.Println("\n=== histogramme des 4 premiers bits (valeur 0..15) ===")
	h := map[uint32]int{}
	for _, p := range frames {
		h[bitsAt(p, 0, 4)]++
	}
	for v := uint32(0); v < 16; v++ {
		if h[v] == 0 {
			continue
		}
		fmt.Printf("%04b : %6d (%5.2f %%)\n", v, h[v], 100*float64(h[v])/float64(len(frames)))
	}

	fmt.Println("\n=== identifiants candidats du 1er record, selon la grammaire ===")
	type g struct {
		name string
		off  int
		w    int
	}
	for _, c := range []g{{"B  amorce2+court : id = bits 3..13 (11 b)", 3, 11}, {"C  amorce1+long  : id = bits 4..13 (10 b)", 4, 10}, {"A  amorce1+court : id = bits 2..12 (11 b)", 2, 11}} {
		hist := map[uint32]int{}
		for _, p := range frames {
			hist[bitsAt(p, c.off, c.w)]++
		}
		// top 8
		type kv struct {
			k uint32
			v int
		}
		var a []kv
		for k, v := range hist {
			a = append(a, kv{k, v})
		}
		for i := 0; i < len(a); i++ {
			for j := i + 1; j < len(a); j++ {
				if a[j].v > a[i].v {
					a[i], a[j] = a[j], a[i]
				}
			}
		}
		fmt.Printf("%s : %d valeurs distinctes ; top :", c.name, len(hist))
		for i := 0; i < len(a) && i < 8; i++ {
			fmt.Printf(" %d(%.1f%%)", a[i].k, 100*float64(a[i].v)/float64(len(frames)))
		}
		fmt.Println()
	}
}
