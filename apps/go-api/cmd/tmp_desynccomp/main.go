// tmp_desynccomp — DIAGNOSTIC : quel COMPOSANT (archétype, nom) bloque le record-loop
// en PREMIER par frame. C'est le bloqueur à porter ensuite pour atteindre plus de
// bipèdes. World = world_dump.txt (250 entités validées, 512-519 bipèdes).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_desynccomp [filmDir]
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

// globalAug : bindings slot->typeIndex collectés par pré-passe (mode GLOBAL), ajoutés
// au world_dump pour lier les slots dynamiques. Nil = désactivé.
var globalAug map[uint32]uint32

func freshWorld(dir string, reg *filmdec.Registry) *filmdec.World {
	dumpFile := "/world_dump_full.txt"
	if v := os.Getenv("WDUMP"); v != "" {
		dumpFile = "/" + v
	}
	raw, _ := os.ReadFile(dir + dumpFile)
	if len(raw) == 0 {
		raw, _ = os.ReadFile(dir + "/world_dump.txt")
	}
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
	for slot, ti := range globalAug {
		w.BindFull(slot, ti) // augmente (le world_dump prime s'il a déjà le slot ? non, écrase)
	}
	return w
}

// buildGlobalBindings pré-passe le film : collecte les (slot, typeIndex) des records NEW
// à traversée PROPRE (DesyncAt==-1, avant tout desync du frame), et ne garde que les slots
// dont un typeIndex domine >=90% des occurrences (rejette les faux-cleans incohérents).
func buildGlobalBindings(dir string, reg *filmdec.Registry) map[uint32]uint32 {
	counts := map[uint32]map[uint32]int{}
	for idx := 2; idx <= 26; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx))) {
			w := freshWorld(dir, reg) // world_dump seul (globalAug encore nil)
			br := filmdec.NewBitReader(fr)
			recs, _ := filmdec.DecodeFrameRecords(br, w, calCfg)
			for _, r := range recs {
				if r.Type == 1 && r.DesyncAt == -1 { // recNew=1, traversée propre
					if counts[r.Slot] == nil {
						counts[r.Slot] = map[uint32]int{}
					}
					counts[r.Slot][r.TypeIndex]++
				}
			}
		}
	}
	out := map[uint32]uint32{}
	for slot, m := range counts {
		var bestTi uint32
		bestN, total := 0, 0
		for ti, n := range m {
			total += n
			if n > bestN {
				bestN, bestTi = n, ti
			}
		}
		if total >= 2 && float64(bestN)/float64(total) >= 0.9 {
			out[slot] = bestTi
		}
	}
	return out
}

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	filmdec.SetRecordStateParam(2)
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	blocker := map[string]int{}     // "typeIdx i<N> <comp>" -> count
	unboundSlot := map[uint32]int{} // slot non-lié (DesyncAt=0, pas de comps)
	recsBeforeBlock := map[int]int{}
	frames, desyncFrames := 0, 0

	// PERSIST=1 : World PERSISTANT à travers les frames (NEW lie, DEL délie), amorcé
	// UNE fois depuis world_dump. Rollback (snapshot/restore) sur les frames qui desync
	// pour ne pas propager des bindings partiels/faux. PERSIST=2 : idem SANS rollback
	// (garde les bindings du préfixe propre d'une frame désync). Vide = réamorçage/frame.
	persistMode := os.Getenv("PERSIST")
	// GLOBAL=1 : augmente le world_dump avec les bindings clean-NEW consistants (pré-passe).
	if os.Getenv("GLOBAL") != "" {
		globalAug = buildGlobalBindings(dir, reg)
		fmt.Printf("global bindings collectés : %d slots consistants (>=90%%)\n\n", len(globalAug))
	}
	var pw *filmdec.World
	if persistMode != "" {
		pw = freshWorld(dir, reg)
	}
	// INFER=1 : décode avec inférence d'archétype des slots transitoires non-liés.
	// INFERLAX=1 : inférence sur unicité seule (sans exiger un successeur lié) -> chaînes.
	inferMode := os.Getenv("INFER") != ""
	if os.Getenv("INFERLAX") != "" {
		filmdec.SetInferStrict(false)
	}
	totalInferred := 0

	for idx := 2; idx <= 26; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx))) {
			frames++
			var w *filmdec.World
			var snap filmdec.WorldSnapshot
			if pw != nil {
				w = pw
				snap = w.Snapshot()
			} else {
				w = freshWorld(dir, reg)
			}
			var recs []filmdec.FrameRecord
			var e error
			if inferMode {
				var inf int
				recs, inf = filmdec.DecodeFrameInfer(fr, w, calCfg)
				totalInferred += inf
				if len(recs) > 0 && recs[len(recs)-1].DesyncAt != -1 {
					e = fmt.Errorf("desync") // dernier record en erreur
				}
			} else {
				br := filmdec.NewBitReader(fr)
				recs, e = filmdec.DecodeFrameRecords(br, w, calCfg)
			}
			if e != nil && persistMode == "1" {
				w.Restore(snap) // rollback la frame désync (bindings partiels non fiables)
			}
			if e == nil || len(recs) == 0 {
				continue
			}
			desyncFrames++
			recsBeforeBlock[len(recs)-1]++
			last := recs[len(recs)-1]
			// composant non porté = le bloqueur
			comp := "(slot non-lié / delta sans archétype)"
			if len(last.Trace.Comps) > 0 {
				for i := len(last.Trace.Comps) - 1; i >= 0; i-- {
					if !last.Trace.Comps[i].Ported {
						comp = fmt.Sprintf("i%d %s", last.Trace.Comps[i].Index, last.Trace.Comps[i].Name)
						break
					}
				}
			} else {
				unboundSlot[last.Slot]++
			}
			key := fmt.Sprintf("typeIdx=%-3d %s", last.TypeIndex, comp)
			blocker[key]++
		}
	}

	fmt.Printf("frames=%d desync=%d (%.0f%%) transitoires_inférés=%d\n", frames, desyncFrames, 100*float64(desyncFrames)/float64(frames), totalInferred)
	fmt.Println("\n=== position du record bloqueur dans le loop (0=1er record) ===")
	type ri struct{ pos, n int }
	var ris []ri
	for p, n := range recsBeforeBlock {
		ris = append(ris, ri{p, n})
	}
	sort.Slice(ris, func(i, j int) bool { return ris[i].n > ris[j].n })
	for i := 0; i < 6 && i < len(ris); i++ {
		fmt.Printf("  %d records OK avant blocage : %d frames\n", ris[i].pos, ris[i].n)
	}
	fmt.Println("\n=== TOP composants/archétypes bloqueurs (à porter) ===")
	type kv struct {
		k string
		n int
	}
	var arr []kv
	for k, n := range blocker {
		arr = append(arr, kv{k, n})
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].n > arr[j].n })
	for i := 0; i < 25 && i < len(arr); i++ {
		fmt.Printf("  %6d  %s\n", arr[i].n, arr[i].k)
	}
	if len(unboundSlot) > 0 {
		fmt.Printf("\n=== slots non-liés (delta sans archétype) : %d distincts ===\n", len(unboundSlot))
		var us []kv
		for s, n := range unboundSlot {
			us = append(us, kv{fmt.Sprintf("slot %d", s), n})
		}
		sort.Slice(us, func(i, j int) bool { return us[i].n > us[j].n })
		for i := 0; i < 10 && i < len(us); i++ {
			fmt.Printf("  %6d  %s\n", us[i].n, us[i].k)
		}
	}
}
