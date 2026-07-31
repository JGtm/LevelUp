// tmp_framedesync — THROWAWAY : trouve le PREMIER record qui désync dans chaque frame (= ce qui
// stoppe la traversée de toute la frame). Histogramme par (typeIndex, composant). Le blocage
// dominant = le prochain deser à porter. Inclut les archétypes NON-biped (le vrai reste).
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

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}

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
func listFrames(d []byte) [][]byte {
	var out [][]byte
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, d[off+16:off+16+sz])
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
	filmdec.SetRecordStateParam(2)
	filmdec.PositionCalibratedSkip = true
	// calibration des composants broken-constant biped (mêmes noms partagés par d'autres archétypes).
	for n, w := range map[string]int{
		"object-forward-and-up-component": 9, "object-angular-velocity-component": 1,
		"object-shield-vitality-component": 29, "object-region-state-component": 358,
		"object-multiplayer-properties-component": 334,
		// broken-constants supplémentaires (mêmes que tmp_autocal) :
		"object-dissolver-component": 4, "unit-desired-aiming-vector-component": 25,
		"unit-grenade-counts-component": 35, "unit-malleable-property-component": 19,
		"unit-command-tick-component": 10,
	} {
		filmdec.SetCalibratedWidth(n, w)
	}
	reg, _ := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))

	// histogramme du 1er record désync par frame : (typeIndex, compName)
	type key struct {
		ti   uint32
		comp string
	}
	hist := map[key]int{}
	frames, blocked := 0, 0
	recsBeforeDesync := map[int]int{} // nb de records décodés avant le 1er désync
	for idx := 2; idx <= 26; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			frames++
			w := freshWorld(reg)
			recs, _ := filmdec.DecodeFrameRecords(filmdec.NewBitReader(fr), w, calCfg)
			firstBad := -1
			for i, r := range recs {
				if r.DesyncAt != -1 {
					firstBad = i
					break
				}
			}
			if firstBad < 0 {
				continue // frame entièrement propre
			}
			blocked++
			n := firstBad
			if n > 20 {
				n = 20
			}
			recsBeforeDesync[n]++
			r := recs[firstBad]
			comp := fmt.Sprintf("i%d", r.DesyncAt)
			if arch, ok := reg.Archetype(int(r.TypeIndex)); ok && r.DesyncAt >= 0 && r.DesyncAt < len(arch.Components) {
				comp = fmt.Sprintf("i%d %s", r.DesyncAt, arch.Components[r.DesyncAt])
			}
			hist[key{r.TypeIndex, comp}]++
		}
	}
	fmt.Printf("=== %d frames ; %d avec désync (%.0f%%) ===\n", frames, blocked, 100*float64(blocked)/float64(frames))

	fmt.Println("\n=== distribution : nb de records décodés AVANT le 1er désync ===")
	var ns []int
	for n := range recsBeforeDesync {
		ns = append(ns, n)
	}
	sort.Ints(ns)
	for _, n := range ns {
		fmt.Printf("  %2d records propres puis désync : %d frames\n", n, recsBeforeDesync[n])
	}

	fmt.Println("\n=== 1er record désync par frame : (typeIndex, composant) -> count ===")
	type kv struct {
		k key
		c int
	}
	var kvs []kv
	for k, c := range hist {
		kvs = append(kvs, kv{k, c})
	}
	sort.Slice(kvs, func(i, j int) bool { return kvs[i].c > kvs[j].c })
	for i, e := range kvs {
		if i >= 20 {
			fmt.Printf("  ... (%d combinaisons distinctes)\n", len(kvs))
			break
		}
		fmt.Printf("  typeIdx=%-3d %-44s : %d frames\n", e.k.ti, e.k.comp, e.c)
	}
}
