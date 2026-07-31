// tmp_after — THROWAWAY : regarde ce qui SUIT le death event (type-20) dans chunk_27.
// La référence d'arme (modèle user : le tir rattaché à la mort) est peut-être APRÈS le
// bloc/end-marker, pas dedans. On dump la zone post-event + scan weapon high-32 + le
// type de l'event suivant.
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
func byteAtBit(d []byte, bit int) byte {
	if bit < 0 || bit+8 > len(d)*8 {
		return 0
	}
	bi := bit / 8
	o := uint(bit % 8)
	if o == 0 {
		return d[bi]
	}
	return (d[bi] << o) | (d[bi+1] >> (8 - o))
}
func bitsU32(d []byte, bit int) uint32 {
	var v uint32
	for i := 0; i < 32; i++ {
		bp := bit + i
		v = (v << 1) | uint32((d[bp>>3]>>uint(7-(bp&7)))&1)
	}
	return v
}
func u64le(d []byte, bit int) uint64 {
	var x uint64
	for i := 0; i < 8; i++ {
		x |= uint64(byteAtBit(d, bit+i*8)) << (uint(i) * 8)
	}
	return x
}

type evt struct {
	xs, endBit, typ, tms int
}

func main() {
	h2n := map[uint32]string{}
	for id, n := range analysis.WeaponIDToName {
		h2n[uint32(id>>32)] = n
	}
	d := inflate(cache + "/chunk_27.bin")
	tot := len(d) * 8
	em := []byte{0x00, 0x00, 0x2e, 0xe0}
	findEnd := func(from int) int {
		for b := from; b <= tot-32; b++ {
			if byteAtBit(d, b) == em[0] && byteAtBit(d, b+8) == em[1] && byteAtBit(d, b+16) == em[2] && byteAtBit(d, b+24) == em[3] {
				return b
			}
		}
		return -1
	}

	var evs []evt
	seen := map[int]bool{}
	for ms := 8; ms <= tot-8; ms++ {
		if byteAtBit(d, ms) != 0xc0 {
			continue
		}
		xe := ms - 8
		if xe < 64 {
			continue
		}
		pre := byteAtBit(d, xe)
		if pre != 0x2d && pre != 0x25 {
			continue
		}
		xs := xe - 64
		if seen[xs] {
			continue
		}
		x := u64le(d, xs)
		if x <= 2e15 || x >= 3e15 {
			continue
		}
		seen[xs] = true
		endBit := findEnd(xs)
		if endBit < 0 || endBit-60*8 < xs {
			continue
		}
		th := int(byteAtBit(d, endBit-60*8+47*8))
		tms := int(binary.BigEndian.Uint32([]byte{
			byteAtBit(d, endBit-60*8+48*8), byteAtBit(d, endBit-60*8+49*8),
			byteAtBit(d, endBit-60*8+50*8), byteAtBit(d, endBit-60*8+51*8)}))
		evs = append(evs, evt{xs, endBit, th, tms})
	}
	sort.Slice(evs, func(i, j int) bool { return evs[i].xs < evs[j].xs })
	fmt.Printf("%d events (par position)\n", len(evs))

	// type de l'event qui SUIT chaque death (en position)
	fmt.Println("\n=== type de l'event suivant un death (type-20) ===")
	nextTypeCount := map[int]int{}
	for i, e := range evs {
		if e.typ != 20 {
			continue
		}
		if i+1 < len(evs) {
			nextTypeCount[evs[i+1].typ]++
		}
	}
	fmt.Printf("  %v\n", nextTypeCount)

	// pour chaque death : scan weapon high-32 dans [endBit, nextEvent.xs] (la zone APRÈS le bloc)
	fmt.Println("\n=== weapon high-32 dans la zone APRÈS le death (endBit -> prochain xuid) ===")
	deathWithWeapon := 0
	nDeath := 0
	for i, e := range evs {
		if e.typ != 20 {
			continue
		}
		nDeath++
		hi := tot
		if i+1 < len(evs) {
			hi = evs[i+1].xs
		}
		var found []string
		for bp := e.endBit; bp+32 <= hi; bp++ {
			if name, ok := h2n[bitsU32(d, bp)]; ok {
				found = append(found, fmt.Sprintf("%s@+%db", name, (bp-e.endBit)/8))
			}
		}
		if len(found) > 0 {
			deathWithWeapon++
			if deathWithWeapon <= 15 {
				fmt.Printf("  death t=%-7d zone=%dB : %v\n", e.tms, (hi-e.endBit)/8, found)
			}
		}
	}
	fmt.Printf("\n  -> %d/%d deaths ont >=1 weapon high-32 dans la zone après\n", deathWithWeapon, nDeath)

	// aussi : scan AVANT le xuid (la zone qui PRÉCÈDE le death) sur 200 octets
	fmt.Println("\n=== weapon high-32 dans la zone AVANT le death (xuid-1600b -> xuid) ===")
	dWB := 0
	for _, e := range evs {
		if e.typ != 20 {
			continue
		}
		lo := e.xs - 1600
		if lo < 0 {
			lo = 0
		}
		var found []string
		for bp := lo; bp+32 <= e.xs; bp++ {
			if name, ok := h2n[bitsU32(d, bp)]; ok {
				found = append(found, fmt.Sprintf("%s@-%db", name, (e.xs-bp)/8))
			}
		}
		if len(found) > 0 {
			dWB++
			if dWB <= 10 {
				fmt.Printf("  death t=%-7d : %v\n", e.tms, found)
			}
		}
	}
	fmt.Printf("  -> %d/%d deaths ont >=1 weapon high-32 dans les 200o avant\n", dWB, nDeath)
}
