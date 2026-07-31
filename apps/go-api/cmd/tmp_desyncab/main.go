// tmp_desyncab — mesure le DESYNC des records biped gameplay (TypeIndex 35) et A/B le flag
// simStateComplete (i60). Reporte : nb records biped, clean-rate (DesyncAt==-1), et l'histo
// du composant de desync. Objectif : activer simStateComplete recule-t-il le desync (i60 -> i63) ?
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

const film = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

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

type frame struct{ pay []byte }

func listFrames(d []byte) []frame {
	var out []frame
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, frame{d[off+16 : off+16+sz]})
		}
		off += 16 + sz
	}
	return out
}

func freshWorld(reg *filmdec.Registry) *filmdec.World {
	raw, _ := os.ReadFile(film + "/world_dump_full.txt")
	w := filmdec.NewWorld(reg)
	for _, tok := range bytes.Fields(raw) {
		var slot, ti uint32
		if _, e := fmt.Sscanf(string(tok), "%d:%d", &slot, &ti); e == nil {
			w.BindFull(slot, ti)
		}
	}
	return w
}

func run(reg *filmdec.Registry, targets map[uint32]bool, label string) {
	arch, _ := reg.Archetype(35)
	total, clean := 0, 0
	desyncComp := map[string]int{}
	maxComps := map[uint32]int{}
	for idx := 3; idx <= 26; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", film, idx))) {
			w := freshWorld(reg)
			recs := filmdec.DecodeFrameResync(fr.pay, w, calCfg, targets, nil)
			for _, r := range recs {
				if r.TypeIndex != 35 {
					continue
				}
				total++
				if r.DesyncAt == -1 {
					clean++
				} else {
					name := fmt.Sprintf("idx%d", r.DesyncAt)
					if r.DesyncAt >= 0 && r.DesyncAt < len(arch.Components) {
						name = fmt.Sprintf("i%d:%s", r.DesyncAt, arch.Components[r.DesyncAt])
					}
					desyncComp[name]++
				}
				if n := len(r.Trace.Comps); n > maxComps[r.Slot] {
					maxComps[r.Slot] = n
				}
			}
		}
	}
	fmt.Printf("[%s] biped records=%d clean(DesyncAt=-1)=%d (%.1f%%)\n", label, total, clean, 100*float64(clean)/float64(max1(total)))
	type kv struct {
		k string
		n int
	}
	var arr []kv
	for k, n := range desyncComp {
		arr = append(arr, kv{k, n})
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].n > arr[j].n })
	for i := 0; i < len(arr) && i < 12; i++ {
		fmt.Printf("    desync @ %s : %d\n", arr[i].k, arr[i].n)
	}
}

func max1(n int) int {
	if n == 0 {
		return 1
	}
	return n
}

func main() {
	filmdec.SetRecordStateParam(2)
	reg, err := filmdec.ParseRegistryChunk(inflate(film + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	// cible large : bipeds gameplay (slots ~512..640).
	targets := map[uint32]bool{}
	for s := uint32(500); s <= 660; s++ {
		targets[s] = true
	}
	arch, _ := reg.Archetype(35)
	fmt.Printf("archétype #35 = %d composants\n", len(arch.Components))
	// localiser i60 simulation-state
	for i, c := range arch.Components {
		if c == "simulation-state" || c == "simulation-state-component" {
			fmt.Printf("simulation-state @ index i%d\n", i)
		}
	}
	fmt.Println()

	filmdec.SetSimStateComplete(false)
	run(reg, targets, "simStateComplete=false (défaut)")
	fmt.Println()
	filmdec.SetSimStateComplete(true)
	run(reg, targets, "simStateComplete=true")
	fmt.Println()
	// sweep simStateExtra avec complete=true
	for _, ex := range []int{8, 16, 32} {
		filmdec.SetSimStateComplete(true)
		filmdec.SetSimStateExtra(ex)
		run(reg, targets, fmt.Sprintf("simStateComplete=true simStateExtra=%d", ex))
		fmt.Println()
	}
}
