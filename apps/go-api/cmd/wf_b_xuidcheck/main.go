package main

// wf_b_xuidcheck — diagnostic : les xuids 64-bit sont-ils présents (bit-level)
// dans les chunks ? Et quel pi (5 bits avant) ressort par chunk ?

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

var xuids = []uint64{
	2533274980284321, 2533274815845110, 2533274882097883, 2535437947245250,
	2535467794760703, 2535444178793711, 2533274826120416, 2533274823110022,
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

func bitAt(d []byte, p int) uint64 {
	if p < 0 || p>>3 >= len(d) {
		return 0
	}
	return uint64((d[p>>3] >> uint(7-(p&7))) & 1)
}
func rb(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		v = (v << 1) | bitAt(d, bp+i)
	}
	return v
}

// target : xuid en 8 octets LE relu BE.
func target(x uint64) uint64 {
	le := make([]byte, 8)
	binary.LittleEndian.PutUint64(le, x)
	return binary.BigEndian.Uint64(le)
}

func main() {
	for i := 0; i <= 27; i++ {
		p := fmt.Sprintf("%s/chunk_%02d.bin", cache, i)
		d := inflate(p)
		if len(d) == 0 {
			continue
		}
		tot := len(d) * 8
		var found []string
		for _, x := range xuids {
			t := target(x)
			cnt := 0
			firstPI := -1
			for bp := 0; bp+64 <= tot; bp++ {
				if rb(d, bp, 64) == t {
					cnt++
					if firstPI < 0 {
						firstPI = int(rb(d, bp-5, 5))
					}
				}
			}
			if cnt > 0 {
				found = append(found, fmt.Sprintf("%d(x%d,pi=%d)", x%1000000, cnt, firstPI))
			}
		}
		if len(found) > 0 {
			fmt.Printf("chunk_%02d (%d bytes): %v\n", i, len(d), found)
		} else {
			fmt.Printf("chunk_%02d (%d bytes): aucun xuid BE-LE\n", i, len(d))
		}
	}

	// Aussi tester le xuid en BIG-ENDIAN brut (8 octets BE relus tels quels)
	// et en LE direct, au cas où l'encodage diffère.
	fmt.Println("\n=== variantes d'encodage xuid (recherche octet-alignée) sur chunk_02 ===")
	d := inflate(fmt.Sprintf("%s/chunk_02.bin", cache))
	for _, x := range xuids {
		le := make([]byte, 8)
		binary.LittleEndian.PutUint64(le, x)
		be := make([]byte, 8)
		binary.BigEndian.PutUint64(be, x)
		nLE := bytes.Count(d, le)
		nBE := bytes.Count(d, be)
		fmt.Printf("  xuid %d : LE-byte=%d BE-byte=%d\n", x%1000000, nLE, nBE)
	}
}
