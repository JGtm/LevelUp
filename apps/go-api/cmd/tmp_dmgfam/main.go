// tmp_dmgfam — investigue TOUS les records 0xd2 (pas seulement ceux avec famille h32 +
// suffixe 0x42c9679f que le warp garde). But : tester l'hypothèse "mécanisme uniforme" —
// mêlée/grenade/splatter font des dégâts, donc devraient être des 0xd2 avec une famille
// NON-firearm que le warp jette silencieusement. Si oui, la cause est là, décodable.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_dmgfam [filmID]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"levelup/go-api/internal/analysis"
	"os"
	"sort"
)

const root = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`
const sfx = uint32(0x42c9679f)

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

func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

func main() {
	m := "000d5950"
	if len(os.Args) > 1 {
		m = os.Args[1]
	}
	cache := root + "/" + m
	h32 := map[uint32]string{}
	for id := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = analysis.WeaponIDToName[id]
	}

	nTot, nFirearm, nSfxNoName, nNoSfx := 0, 0, 0, 0
	famNamed := map[string]int{}  // familles reconnues (h32) avec sfx
	famRawSfx := map[uint32]int{} // 32-bit famille avec sfx mais PAS dans h32
	sfxAlt := map[uint32]int{}    // "suffixe" des 0xd2 sans le 0x42c9679f (autre marqueur ?)
	atkNoSfx := map[int]int{}     // attaquant supposé (bit 36) des 0xd2 non-firearm
	var samplesNoSfx []string
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := d[off+16 : off+16+sz]
			off += 16 + sz
			if typ != 0 || len(pl) == 0 || pl[0] != 0xd2 {
				continue
			}
			nTot++
			bp := 41
			if bitsAt(pl, bp, 1) == 1 {
				bp++
			} else {
				bp += 3
			}
			f := uint32(bitsAt(pl, bp, 32))
			suf := uint32(bitsAt(pl, bp+32, 32))
			switch {
			case suf == sfx && h32[f] != "":
				nFirearm++
				famNamed[h32[f]]++
			case suf == sfx:
				nSfxNoName++
				famRawSfx[f]++
			default:
				nNoSfx++
				sfxAlt[suf]++
				atkNoSfx[int(bitsAt(pl, 36, 5))>>1]++ // attaquant supposé (bit 36) — valide (0-16) ?
				if len(samplesNoSfx) < 6 {
					n := 40
					if n > len(pl) {
						n = len(pl)
					}
					samplesNoSfx = append(samplesNoSfx, fmt.Sprintf("%x", pl[:n]))
				}
			}
		}
	}
	fmt.Printf("=== film %s : %d records 0xd2 au total ===\n", m, nTot)
	fmt.Printf("  firearm (sfx + famille h32)      : %d\n", nFirearm)
	fmt.Printf("  sfx mais famille INCONNUE (h32)  : %d  (candidats mêlée/grenade/véhicule !)\n", nSfxNoName)
	fmt.Printf("  SANS le suffixe 0x42c9679f       : %d  (structure différente ?)\n", nNoSfx)

	fmt.Print("\nfamilles nommées: ")
	{
		type kv struct {
			k string
			v int
		}
		var a []kv
		for k, v := range famNamed {
			a = append(a, kv{k, v})
		}
		sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
		for _, x := range a {
			fmt.Printf("%s=%d ", x.k, x.v)
		}
		fmt.Println()
	}
	if nSfxNoName > 0 {
		fmt.Print("familles INCONNUES (id 32-bit, sfx OK) : ")
		type kv struct {
			k uint32
			v int
		}
		var a []kv
		for k, v := range famRawSfx {
			a = append(a, kv{k, v})
		}
		sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
		for i, x := range a {
			if i >= 15 {
				break
			}
			fmt.Printf("%08X=%d ", x.k, x.v)
		}
		fmt.Println()
	}
	if nNoSfx > 0 {
		fmt.Print("\"suffixes\" alternatifs (0xd2 sans 0x42c9679f) : ")
		type kv struct {
			k uint32
			v int
		}
		var a []kv
		for k, v := range sfxAlt {
			a = append(a, kv{k, v})
		}
		sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
		for i, x := range a {
			if i >= 12 {
				break
			}
			fmt.Printf("%08X=%d ", x.k, x.v)
		}
		fmt.Println()
		fmt.Print("attaquant supposé (bit 36 >>1) des non-firearm : ")
		{
			var ks []int
			for k := range atkNoSfx {
				ks = append(ks, k)
			}
			sort.Ints(ks)
			for _, k := range ks {
				fmt.Printf("%d=%d ", k, atkNoSfx[k])
			}
			fmt.Println()
		}
		fmt.Println("échantillons 0xd2 sans sfx (40o):")
		for _, s := range samplesNoSfx {
			fmt.Printf("  %s\n", s)
		}
	}
}
