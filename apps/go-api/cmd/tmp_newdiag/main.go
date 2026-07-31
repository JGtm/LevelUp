// tmp_newdiag — DIAGNOSTIC : quels archétypes new-entity (NEW records) apparaissent
// et lesquels désync (default-state non porté). World frais par frame (world_dump_full).
// Pour chaque NEW record : (typeIndex, clean?). Pour chaque 1er desync de frame :
// (type, slot, typeIndex). But : scoper le porting des default-states non-bipèdes.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_newdiag [filmDir]
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

const defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

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

var bindings [][2]uint32

func loadBindings(dir string) {
	raw, _ := os.ReadFile(dir + "/world_dump_full.txt")
	for _, tok := range bytes.Fields(raw) {
		s := string(tok)
		if len(s) == 0 || s[0] == '#' {
			continue
		}
		var slot, ti uint32
		if _, e := fmt.Sscanf(s, "%d:%d", &slot, &ti); e == nil {
			bindings = append(bindings, [2]uint32{slot, ti})
		}
	}
}

func freshWorld(reg *filmdec.Registry) *filmdec.World {
	w := filmdec.NewWorld(reg)
	for _, b := range bindings {
		w.BindFull(b[0], b[1])
	}
	return w
}

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	filmdec.SetRecordStateParam(2)
	loadBindings(dir)
	fmt.Printf("world_dump_full : %d slots liés\n", len(bindings))
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	newClean := map[uint32]int{} // typeIndex -> NEW records décodés clean
	newTotal := map[uint32]int{} // typeIndex -> NEW records vus (clean inclus)
	desyncType := map[string]int{}
	desyncTypeIdx := map[uint32]int{} // typeIndex du record desyncer
	frames, framesDesync := 0, 0

	for idx := 2; idx <= 26; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx))) {
			frames++
			w := freshWorld(reg)
			br := filmdec.NewBitReader(fr)
			recs, e := filmdec.DecodeFrameRecords(br, w, calCfg)
			for _, r := range recs {
				if r.Type == 1 { // recNew
					newTotal[r.TypeIndex]++
					if r.DesyncAt == -1 {
						newClean[r.TypeIndex]++
					}
				}
			}
			if e != nil && len(recs) > 0 {
				framesDesync++
				last := recs[len(recs)-1]
				tn := map[int]string{1: "NEW", 2: "DEL", 3: "DELTA"}[last.Type]
				desyncType[tn]++
				desyncTypeIdx[last.TypeIndex]++
			}
		}
	}

	fmt.Printf("\nframes=%d desync=%d (%.0f%%)\n", frames, framesDesync, 100*float64(framesDesync)/float64(frames))
	fmt.Println("\n=== type du record qui DÉSYNC ===")
	for t, n := range desyncType {
		fmt.Printf("  %-6s : %d\n", t, n)
	}
	fmt.Println("\n=== typeIndex du record desyncer (top 15) ===")
	printTop(desyncTypeIdx, 15)
	fmt.Println("\n=== NEW records par typeIndex (clean/total, top 20) ===")
	type kv struct {
		ti         uint32
		clean, tot int
	}
	var arr []kv
	for ti, tot := range newTotal {
		arr = append(arr, kv{ti, newClean[ti], tot})
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].tot > arr[j].tot })
	for i := 0; i < 20 && i < len(arr); i++ {
		name := ""
		if a, ok := reg.Archetype(int(arr[i].ti)); ok && len(a.Components) > 0 {
			name = fmt.Sprintf("%dcomps [%s]", len(a.Components), a.Components[0])
		}
		fmt.Printf("  typeIdx=%-3d clean=%-5d/%-5d  %s\n", arr[i].ti, arr[i].clean, arr[i].tot, name)
	}
}

func printTop(m map[uint32]int, n int) {
	type kv struct {
		k uint32
		v int
	}
	var arr []kv
	for k, v := range m {
		arr = append(arr, kv{k, v})
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].v > arr[j].v })
	for i := 0; i < n && i < len(arr); i++ {
		fmt.Printf("  typeIdx=%-3d : %d\n", arr[i].k, arr[i].v)
	}
}
