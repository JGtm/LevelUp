// tmp_grenreach — THROWAWAY : mesure l'ATTEIGNABILITE des composants grenades/capacites
// (i22 unit-grenade-counts, i47 biped-desired-grenade-set, i48 biped-desired-ability-set,
// i56 biped-spartan-ability-energy, i57) sur le walk SEQUENTIEL, meme harnais que
// cmd/tmp_l0witness (World seme par les bindings offline du keyframe).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_grenreach [chunkLo chunkHi]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
	idLow = 11
)

var bipedSlots = map[uint32]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}

func inflate(p string) []byte {
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); (e2 == nil || e2 == io.ErrUnexpectedEOF) && len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

func listChunkFrames(d []byte, want uint16) [][]byte {
	var out [][]byte
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == want {
			out = append(out, d[off+16:off+16+sz])
		}
		off += 16 + sz
	}
	return out
}

func main() {
	lo, hi := 3, 26
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[1], "%d", &lo)
		fmt.Sscanf(os.Args[2], "%d", &hi)
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	if arch, ok := reg.Archetype(int(filmdec.BipedTypeIndex)); ok {
		fmt.Printf("=== archetype ti=%d : %d composants ===\n", filmdec.BipedTypeIndex, len(arch.Components))
		for i, n := range arch.Components {
			switch i {
			case 22, 47, 48, 56, 57:
				fmt.Printf("   i%02d %s\n", i, n)
			}
		}
	}
	kf := listChunkFrames(inflate(cache+"/chunk_02.bin"), 2)
	if len(kf) == 0 {
		panic("keyframe introuvable")
	}
	binds := filmdec.WalkKeyframeWorld(kf[0])

	filmdec.SetRecordStateParam(2)
	cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: idLow}

	w := filmdec.NewWorld(reg)
	for _, b := range binds {
		w.BindFull(uint32((b.Gen<<30)|b.Slot), uint32(b.TI))
	}

	targets := map[int]bool{22: true, 47: true, 48: true, 56: true, 57: true}
	type stat struct {
		name       string
		present    int // present dans le masque
		reached    int // consomme proprement (Ported)
		reachedBip int // idem, slot joueur 512..519
	}
	st := map[int]*stat{}
	recTotal, recClean := 0, 0
	bipTotal, bipClean := 0, 0

	for c := lo; c <= hi; c++ {
		data := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, c))
		if data == nil {
			continue
		}
		for _, pay := range listChunkFrames(data, 0) {
			br := filmdec.NewBitReader(pay)
			recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
			for _, r := range recs {
				if r.TypeIndex != filmdec.BipedTypeIndex {
					continue
				}
				recTotal++
				if r.DesyncAt < 0 {
					recClean++
				}
				isBip := bipedSlots[r.Slot]
				if isBip {
					bipTotal++
					if r.DesyncAt < 0 {
						bipClean++
					}
				}
				arch, _ := reg.Archetype(int(r.TypeIndex))
				for i := range targets {
					if i < len(arch.Components) && r.Trace.Mask&(uint64(1)<<uint(i)) != 0 {
						if st[i] == nil {
							st[i] = &stat{name: arch.Components[i]}
						}
						st[i].present++
					}
				}
				for _, cr := range r.Trace.Comps {
					if !targets[cr.Index] || !cr.Ported {
						continue
					}
					if st[cr.Index] == nil {
						st[cr.Index] = &stat{name: cr.Name}
					}
					st[cr.Index].reached++
					if isBip {
						st[cr.Index].reachedBip++
					}
				}
			}
		}
	}

	fmt.Printf("\n=== ATTEIGNABILITE (chunks %d..%d, film 000d5950) ===\n", lo, hi)
	fmt.Printf("records ti=35 : total=%d clean=%d (%.2f%%)\n", recTotal, recClean, 100*float64(recClean)/float64(max(recTotal, 1)))
	fmt.Printf("dont slots joueur 512..519 : total=%d clean=%d\n\n", bipTotal, bipClean)
	var idx []int
	for i := range st {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	for _, i := range idx {
		s := st[i]
		fmt.Printf("  i%02d %-46s present=%-7d atteint=%-7d (%.2f%%)  dont slots joueur=%d\n",
			i, s.name, s.present, s.reached, 100*float64(s.reached)/float64(max(s.present, 1)), s.reachedBip)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
