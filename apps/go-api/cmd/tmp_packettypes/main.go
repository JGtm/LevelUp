// tmp_packettypes — THROWAWAY : énumère tous les types de paquets du film
// (Type u16 LE | b2 b3 | Size u32 | Ts u64 | payload) par chunk, pour localiser
// le flux où vit le DamageReport/kill-report [killer][weapon][victim].
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
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

type pkt struct {
	typ        uint16
	b2, b3     byte
	size       int
	ts         uint64
	payloadOff int
}

func listPackets(d []byte) []pkt {
	var out []pkt
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		b2, b3 := d[off+2], d[off+3]
		size := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if size <= 0 || off+16+size > len(d) {
			break
		}
		out = append(out, pkt{typ, b2, b3, size, ts, off + 16})
		off += 16 + size
	}
	return out
}

func main() {
	// agrégat global type -> (count, tailles min/max, b2b3 set)
	type agg struct {
		count        int
		minSz, maxSz int
		chunks       map[int]bool
	}
	global := map[uint16]*agg{}
	for n := 0; n <= 27; n++ {
		name := fmt.Sprintf("chunk_%02d.bin", n)
		d := inflate(cache + "/" + name)
		if len(d) == 0 {
			continue
		}
		for _, p := range listPackets(d) {
			a := global[p.typ]
			if a == nil {
				a = &agg{minSz: 1 << 30, chunks: map[int]bool{}}
				global[p.typ] = a
			}
			a.count++
			if p.size < a.minSz {
				a.minSz = p.size
			}
			if p.size > a.maxSz {
				a.maxSz = p.size
			}
			a.chunks[n] = true
		}
	}
	var types []int
	for t := range global {
		types = append(types, int(t))
	}
	sort.Ints(types)
	fmt.Println("=== Types de paquets (global) ===")
	for _, t := range types {
		a := global[uint16(t)]
		fmt.Printf("  type=%-3d count=%-6d size[%d..%d] chunks=%d\n", t, a.count, a.minSz, a.maxSz, len(a.chunks))
	}

	// Détail packets d'un chunk gameplay (chunk_05) : ordre + tailles
	fmt.Println("\n=== chunk_05 : séquence de paquets ===")
	d := inflate(cache + "/chunk_05.bin")
	pkts := listPackets(d)
	for i, p := range pkts {
		if i > 40 {
			fmt.Printf("  ... (%d paquets au total)\n", len(pkts))
			break
		}
		fmt.Printf("  [%2d] type=%-3d b2=%02x b3=%02x size=%-7d ts=%d\n", i, p.typ, p.b2, p.b3, p.size, p.ts)
	}
}
