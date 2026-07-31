// tmp_registry — THROWAWAY : inflate chunk_00 (registre ECS) et tranche la question
// A-vs-C : chunk_00 contient-il des BLOCS d'archetype ordonnes (Front A : BIPED #33
// @0x08e300, 84 composants weapon-state-type-info/object-position en ordre) ou juste
// une liste globale plate de 264 composants (Front C) ? On dumpe les slots 260o autour
// des offsets cles + on cherche les noms de composants attendus.
package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"strings"
)

func inflate(path string) []byte {
	raw, _ := os.ReadFile(path)
	if len(raw) >= 2 && raw[0] == 0x78 {
		zr, e := zlib.NewReader(bytes.NewReader(raw))
		if e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				fmt.Printf("zlib %d -> %d octets\n", len(raw), len(d))
				return d
			}
		}
	}
	fmt.Printf("brut %d octets\n", len(raw))
	return raw
}

// dumpSlot lit un slot 260o : [u32 A][u32 B][nom ASCII NUL-pad 252].
func dumpSlot(data []byte, off int) (a, b uint32, name string) {
	if off+260 > len(data) {
		return 0, 0, ""
	}
	a = uint32(data[off]) | uint32(data[off+1])<<8 | uint32(data[off+2])<<16 | uint32(data[off+3])<<24
	b = uint32(data[off+4]) | uint32(data[off+5])<<8 | uint32(data[off+6])<<16 | uint32(data[off+7])<<24
	raw := data[off+8 : off+260]
	if z := bytes.IndexByte(raw, 0); z >= 0 {
		raw = raw[:z]
	}
	name = string(raw)
	return a, b, name
}

func main() {
	path := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950/chunk_00.bin`
	if len(os.Args) >= 2 {
		path = os.Args[1]
	}
	data := inflate(path)

	// Mode dump-range : <chunk> <startSlot> <count>
	if len(os.Args) >= 4 {
		var start, cnt int
		fmt.Sscanf(os.Args[2], "%d", &start)
		fmt.Sscanf(os.Args[3], "%d", &cnt)
		fmt.Printf("\n=== slots [%d..%d] (a, b, prefix-change marque '<<') ===\n", start, start+cnt)
		prev := ""
		for i := start; i < start+cnt; i++ {
			a, b, name := dumpSlot(data, i*260)
			pre := name
			if d := strings.IndexByte(name, '-'); d >= 0 {
				if d2 := strings.IndexByte(name[d+1:], '-'); d2 >= 0 {
					pre = name[:d+1+d2]
				}
			}
			mark := ""
			if pre != prev {
				mark = "  << " + pre
			}
			prev = pre
			fmt.Printf("  slot[%5d] @0x%07x a=%-3d b=%-3d : %q%s\n", i, i*260, a, b, name, mark)
		}
		return
	}

	// 1) Liste depuis offset 0 (world archetype #0 : game-engine-*).
	fmt.Println("\n=== slots depuis offset 0 (archetype #0 world) ===")
	for i := 0; i < 14; i++ {
		off := i * 260
		a, b, name := dumpSlot(data, off)
		if name == "" {
			break
		}
		fmt.Printf("  slot[%2d] @0x%06x a=%d b=%d : %q\n", i, off, a, b, name)
	}

	// 2) Bloc BIPED #33 (Front A @0x08e300) : 84 composants ordonnes ?
	fmt.Println("\n=== slots @0x08e300 (archetype BIPED #33) ===")
	base := 0x08e300
	for i := 0; i < 20; i++ {
		off := base + i*260
		a, b, name := dumpSlot(data, off)
		if name == "" {
			fmt.Printf("  [%2d] @0x%06x : (vide)\n", i, off)
			continue
		}
		fmt.Printf("  [%2d] @0x%06x a=%d b=%d : %q\n", i, off, a, b, name)
	}

	// 3) Recherche directe des noms de composants cles -> structure reelle.
	fmt.Println("\n=== positions des noms de composants cles ===")
	for _, key := range []string{"weapon-state-type-info", "object-body-vitality", "object-dead-state", "object-position-component", "biped"} {
		idx := 0
		count := 0
		for {
			p := bytes.Index(data[idx:], []byte(key))
			if p < 0 {
				break
			}
			abs := idx + p
			if count < 6 {
				// contexte : le slot commence 8o avant le nom si c'est un slot
				fmt.Printf("  %q @0x%06x (slot? a=%d)\n", key, abs, le32(data, abs-8))
			}
			count++
			idx = abs + len(key)
		}
		fmt.Printf("  -> %q : %d occurrence(s)\n", key, count)
	}

	// 4) Combien de slots non-vides au total (estimation du nb de composants/archetypes) ?
	nonEmpty := 0
	for off := 0; off+260 <= len(data); off += 260 {
		if _, _, n := dumpSlot(data, off); n != "" && isASCIIName(n) {
			nonEmpty++
		}
	}
	fmt.Printf("\n=== %d slots 260o avec nom ASCII plausible (sur %d octets) ===\n", nonEmpty, len(data))
}

func le32(d []byte, off int) uint32 {
	if off < 0 || off+4 > len(d) {
		return 0
	}
	return uint32(d[off]) | uint32(d[off+1])<<8 | uint32(d[off+2])<<16 | uint32(d[off+3])<<24
}

func isASCIIName(s string) bool {
	if len(s) < 3 {
		return false
	}
	for _, r := range s {
		if r < 0x20 || r > 0x7e {
			return false
		}
	}
	return strings.ContainsAny(s, "-abcdefghijklmnopqrstuvwxyz")
}
