// tmp_chunk00 — THROWAWAY. Examine la STRUCTURE COMPLÈTE de chunk_00 (registre) :
// header éventuel, nombre de blocs d'archétype (260×64=16640), et surtout la QUEUE
// après le dernier bloc (octets ignorés par parseRegistry). Hypothèse (recadrage user) :
// la "recette" des largeurs de précision runtime est auto-décrite dans le film — ici,
// potentiellement dans cette queue. On cherche une structure (tables de largeurs/ranges).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_chunk00
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"

	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

const blockSize = 260 * 64 // 16640

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

func hexdump(d []byte, base int) {
	for off := 0; off < len(d); off += 16 {
		end := off + 16
		if end > len(d) {
			end = len(d)
		}
		fmt.Printf("  %08x  ", base+off)
		for i := off; i < off+16; i++ {
			if i < len(d) {
				fmt.Printf("%02x ", d[i])
			} else {
				fmt.Printf("   ")
			}
		}
		fmt.Printf(" |")
		for i := off; i < end; i++ {
			c := d[i]
			if c >= 0x20 && c < 0x7f {
				fmt.Printf("%c", c)
			} else {
				fmt.Printf(".")
			}
		}
		fmt.Printf("|\n")
	}
}

func main() {
	d := inflate(cache + "/chunk_00.bin")
	nBlocks := len(d) / blockSize
	rem := len(d) % blockSize
	fmt.Printf("chunk_00 inflaté = %d octets\n", len(d))
	fmt.Printf("blocs d'archétype (16640) = %d  → %d octets ; QUEUE = %d octets\n\n", nBlocks, nBlocks*blockSize, rem)

	// Le tout premier bloc : est-ce un header ou déjà l'archétype #0 ?
	fmt.Printf("--- 48 premiers octets de chunk_00 (header ? archétype #0 ?) ---\n")
	hexdump(d[:48], 0)

	if rem == 0 {
		fmt.Printf("\n(pas de queue : len divisible par 16640)\n")
		return
	}
	tail := d[nBlocks*blockSize:]
	fmt.Printf("\n--- QUEUE (%d octets après le bloc %d) : 512 premiers octets ---\n", rem, nBlocks)
	n := len(tail)
	if n > 512 {
		n = 512
	}
	hexdump(tail[:n], nBlocks*blockSize)

	// Cherche des motifs : la queue est-elle des u32 petits (largeurs 0..26) ou des floats
	// (ranges ±100) ? Compte les u32 dans [0,26] (candidats largeurs) et les floats plausibles.
	nSmallU32, nRangeF32 := 0, 0
	for i := 0; i+4 <= len(tail); i += 4 {
		u := binary.LittleEndian.Uint32(tail[i:])
		if u <= 26 {
			nSmallU32++
		}
		f := math.Float32frombits(u)
		if f > -1e4 && f < 1e4 && (f > 1e-3 || f < -1e-3) {
			nRangeF32++
		}
	}
	fmt.Printf("\nqueue = %d u32 ; dont ≤26 (candidats largeurs) : %d ; floats plausibles [±1e4] : %d\n", len(tail)/4, nSmallU32, nRangeF32)
	nonzero := 0
	for _, b := range tail {
		if b != 0 {
			nonzero++
		}
	}
	fmt.Printf("queue : %d/%d octets non nuls (%.1f%%)\n", nonzero, len(tail), 100*float64(nonzero)/float64(len(tail)))

	// Barème par-composant du biped #35 : le champ slot+0 (kind) et slot+4 (flags).
	// On veut voir si flags forme un "level de précision" cohérent (PISTE 1) par composant.
	reg, _ := filmdec.ParseRegistryChunk(d)
	if a, ok := reg.Archetype(35); ok {
		fmt.Printf("\n--- archétype #35 (biped) : kind/flags par composant (les 24 premiers) ---\n")
		base := 35 * blockSize
		for i := 0; i < len(a.Components) && i < 24; i++ {
			off := base + i*260
			kind := binary.LittleEndian.Uint32(d[off:])
			flags := binary.LittleEndian.Uint32(d[off+4:])
			fmt.Printf("  i%-2d kind=%-4d flags=%-4d (level? 6+flags=%d)  %s\n", i, kind, flags, 6+flags, a.Components[i])
		}
	}
}
