// tmp_xuidscan — THROWAWAY : cherche les 8 xuids du roster (motif acurtis : 8 octets
// LE relus BE, bit-level) dans les keyframes type-1 ET type-2, reporte position + pi
// (5 bits avant) + arme la plus proche. But : établir record_biped -> joueur.
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

// roster : xuid -> label (depuis tmp_matchinfo). JGtm marqué.
var roster = []struct {
	xuid  uint64
	label string
}{
	{2533274980284321, "p?_980284321(14/13 T0)"},
	{2533274815845110, "p?_815845110(12/10 T1)"},
	{2533274882097883, "p?_882097883(14/9 T1)"},
	{2535437947245250, "p?_947245250(14/13 T1)"},
	{2535467794760703, "p?_794760703(13/9 T0)"},
	{2535444178793711, "p?_178793711(10/11 T1)"},
	{2533274826120416, "p?_826120416(8/14 T0)"},
	{2533274823110022, "JGtm_110022(8/14 T0)"},
}

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

func extractPacket(data []byte, wantType uint16) []byte {
	off := 0
	for off+16 <= len(data) {
		typ := binary.LittleEndian.Uint16(data[off:])
		size := int(binary.LittleEndian.Uint32(data[off+4:]))
		if size <= 0 || off+16+size > len(data) {
			break
		}
		if typ == wantType {
			return data[off+16 : off+16+size]
		}
		off += 16 + size
	}
	return nil
}

func bitAt(data []byte, p int) uint64 {
	if p < 0 || p>>3 >= len(data) {
		return 0
	}
	return uint64((data[p>>3] >> uint(7-(p&7))) & 1)
}

func readBits(data []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		v = (v << 1) | bitAt(data, bp+i)
	}
	return v
}

func xuidTarget(xuid uint64) uint64 {
	le := make([]byte, 8)
	binary.LittleEndian.PutUint64(le, xuid)
	return binary.BigEndian.Uint64(le)
}

type wpnHit struct {
	bitPos int
	name   string
}

func scanWeapons(data []byte) []wpnHit {
	high2name := map[uint32]string{}
	for id, name := range analysis.WeaponIDToName {
		high2name[uint32(id>>32)] = name
	}
	var hits []wpnHit
	tot := len(data) * 8
	for bp := 0; bp+32 <= tot; bp++ {
		if name, ok := high2name[uint32(readBits(data, bp, 32))]; ok {
			hits = append(hits, wpnHit{bp, name})
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].bitPos < hits[j].bitPos })
	return hits
}

func nearestWeapon(hits []wpnHit, pos int) (string, int) {
	best := ""
	bestD := 1 << 30
	for _, h := range hits {
		d := h.bitPos - pos
		if d < 0 {
			d = -d
		}
		if d < bestD {
			bestD = d
			best = h.name
		}
	}
	return best, bestD
}

func main() {
	for _, chunkName := range []string{"chunk_02.bin", "chunk_19.bin"} {
		raw := inflate(cache + "/" + chunkName)
		for _, typ := range []uint16{1, 2} {
			payload := extractPacket(raw, typ)
			if payload == nil {
				continue
			}
			wpns := scanWeapons(payload)
			fmt.Printf("\n========== %s type-%d (%d octets, %d littéraux armes) ==========\n",
				chunkName, typ, len(payload), len(wpns))
			tot := len(payload) * 8
			type found struct {
				label  string
				bitPos int
				pi     uint64
				near   string
				nearD  int
			}
			var founds []found
			for _, r := range roster {
				target := xuidTarget(r.xuid)
				cnt := 0
				for bp := 0; bp+64 <= tot; bp++ {
					if readBits(payload, bp, 64) != target {
						continue
					}
					pi := readBits(payload, bp-5, 5)
					near, nd := nearestWeapon(wpns, bp)
					founds = append(founds, found{r.label, bp, pi, near, nd})
					cnt++
					if cnt >= 3 {
						break
					}
				}
				if cnt == 0 {
					fmt.Printf("  %-26s : ABSENT\n", r.label)
				}
			}
			sort.Slice(founds, func(i, j int) bool { return founds[i].bitPos < founds[j].bitPos })
			for _, f := range founds {
				fmt.Printf("  bit%-8d pi=%2d  %-26s  near=%s(+%d)\n", f.bitPos, f.pi, f.label, f.near, f.nearD)
			}
		}
	}
}
