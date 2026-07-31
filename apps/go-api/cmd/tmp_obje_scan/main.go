// tmp_obje_scan — THROWAWAY : cherche les weapon high-32 CONNUS (analysis.WeaponIDToName)
// à tous les alignements bit dans un chunk film, par type de paquet. But : prouver
// que l'arme ('obje' variant_name R(32)) est présente dans le keyframe type-2, et
// déterminer le namespace (weapon-definition high-32 vs object-variant distinct).
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"os"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tmp_obje_scan <chunk.bin>")
		os.Exit(1)
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	data := raw
	if len(raw) >= 2 && raw[0] == 0x78 {
		zr, _ := zlib.NewReader(bytes.NewReader(raw))
		data, _ = readAll(zr)
	}

	// Full 64-bit weapon-id -> name (depuis analysis.WeaponIDToName).
	fullSet := map[uint64]string{}
	for id, name := range analysis.WeaponIDToName {
		fullSet[id] = name
	}

	// Cherche le keyframe type-2 + son timestamp.
	off := 0
	for off+16 <= len(data) {
		typ := binary.LittleEndian.Uint16(data[off:])
		size := int(binary.LittleEndian.Uint32(data[off+4:]))
		ts := binary.LittleEndian.Uint64(data[off+8:])
		if size <= 0 || off+16+size > len(data) {
			break
		}
		if typ == 2 {
			payload := data[off+16 : off+16+size]
			counts := scanFullWeapons(payload, fullSet)
			fmt.Printf("KEYFRAME type-2 size=%d ts=%d : %d arme(s) distincte(s) (id 64-bit complet)\n",
				size, ts, len(counts))
			for name, c := range counts {
				fmt.Printf("    %-22s x%d\n", name, c)
			}
			return
		}
		off += 16 + size
	}
	fmt.Println("pas de keyframe type-2")
}

// scanFullWeapons cherche les weapon-id 64-bit complets (high-32 puis low-32
// consécutifs, MSB-first) à tous les alignements bit. Renvoie le compte par arme.
func scanFullWeapons(payload []byte, set map[uint64]string) map[string]int {
	counts := map[string]int{}
	total := len(payload) * 8
	for start := 0; start+64 <= total; start++ {
		br := filmdec.NewBitReader(payload)
		br.Skip(start)
		v := br.ReadBits(64)
		if name, ok := set[v]; ok {
			counts[name]++
		}
	}
	return counts
}

func readAll(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var out []byte
	buf := make([]byte, 1<<16)
	for {
		n, err := r.Read(buf)
		out = append(out, buf[:n]...)
		if err != nil {
			return out, nil
		}
	}
}
