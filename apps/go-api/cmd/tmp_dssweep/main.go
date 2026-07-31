// tmp_dssweep — THROWAWAY : sweep d'un pré-skip constant avant le dead-state (i11)
// pour trouver une misalignment d'amont CONSTANTE. Une mort a TOUJOURS une victime,
// donc au bon offset le taux d'EnumA=-1 (gate "absent") chute. Si aucun N ne le fait
// chuter -> l'erreur est data-dépendante (un count/loop d'un composant i0-i10).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_dssweep [maxChunk]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}
var bipedSlots = map[uint32]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}

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

type packet struct{ payload []byte }

func listFrames(d []byte) []packet {
	var out []packet
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, packet{d[off+16 : off+16+sz]})
		}
		off += 16 + sz
	}
	return out
}

func freshWorld(reg *filmdec.Registry) *filmdec.World {
	raw, _ := os.ReadFile(cache + "/world_dump.txt")
	w := filmdec.NewWorld(reg)
	for _, tok := range bytes.Fields(raw) {
		s := string(tok)
		if len(s) == 0 || s[0] == '#' {
			continue
		}
		var slot, ti uint32
		if _, e := fmt.Sscanf(s, "%d:%d", &slot, &ti); e == nil {
			w.BindFull(slot, ti)
		}
	}
	return w
}

func main() {
	maxChunk := 26
	if len(os.Args) >= 2 {
		fmt.Sscanf(os.Args[1], "%d", &maxChunk)
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	frames := map[int][]packet{}
	for idx := 2; idx <= maxChunk; idx++ {
		frames[idx] = listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx)))
	}

	fmt.Println("N   total  A=-1%   B=-1%   bothValid%(A,B in 0-7 distincts)")
	for n := -8; n <= 8; n++ {
		filmdec.SetDeadStatePreSkip(n)
		total, aNeg, bNeg, both := 0, 0, 0, 0
		for idx := 2; idx <= maxChunk; idx++ {
			for _, fr := range frames[idx] {
				w := freshWorld(reg)
				br := filmdec.NewBitReader(fr.payload)
				recs, _ := filmdec.DecodeFrameRecords(br, w, calCfg)
				for _, r := range recs {
					d := r.Trace.Dead
					if !bipedSlots[r.Slot] || d == nil || !d.Mort {
						continue
					}
					total++
					if d.EnumA == -1 {
						aNeg++
					}
					if d.EnumB == -1 {
						bNeg++
					}
					if d.EnumA >= 0 && d.EnumA <= 7 && d.EnumB >= 0 && d.EnumB <= 7 && d.EnumA != d.EnumB {
						both++
					}
				}
			}
		}
		if total == 0 {
			total = 1
		}
		fmt.Printf("%+d  %5d  %5.1f   %5.1f   %5.1f\n", n, total,
			100*float64(aNeg)/float64(total), 100*float64(bNeg)/float64(total), 100*float64(both)/float64(total))
	}
	filmdec.SetDeadStatePreSkip(0)
}
