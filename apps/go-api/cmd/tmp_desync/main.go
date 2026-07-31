// tmp_desync — DIAGNOSTIC frame-decoder : histogramme des points de desync sur tous
// les frames d'un film. Quel (typeIndex, composant iN) casse le plus l'alignement →
// la cible à corriger pour décoder PLUS de records (donc plus de joueurs).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_desync [filmDir]
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

var worldFile = "/world_dump.txt"

func freshWorld(dir string, reg *filmdec.Registry) *filmdec.World {
	raw, _ := os.ReadFile(dir + worldFile)
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
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if len(os.Args) > 2 {
		worldFile = "/" + os.Args[2]
	}
	fmt.Printf("world=%s\n", worldFile)
	filmdec.SetRecordStateParam(2)
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	type key struct {
		typeIdx uint32
		comp    int
	}
	desyncHist := map[key]int{}
	unboundSlots := map[uint32]int{} // slots des desyncs typeIdx-0 (non liés)
	bipedDesync := map[key]int{}     // desyncs sur archetype biped (typeIdx 35)
	totalFrames, framesWithDesync := 0, 0
	recsBeforeDesync := 0
	bipedRecDecoded := map[uint32]int{} // slot -> nb records bipèdes décodés OK
	var firstBipedDesyncSlots []uint32

	persist := os.Getenv("PERSIST") == "1"
	w := freshWorld(dir, reg)
	for idx := 2; idx <= 26; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx))) {
			if !persist {
				w = freshWorld(dir, reg)
			}
			br := filmdec.NewBitReader(fr)
			recs, err := filmdec.DecodeFrameRecords(br, w, calCfg)
			totalFrames++
			for _, r := range recs {
				if r.DesyncAt == -1 {
					if bipedSlots[r.Slot] {
						bipedRecDecoded[r.Slot]++
					}
				}
			}
			if err != nil && len(recs) > 0 {
				framesWithDesync++
				last := recs[len(recs)-1]
				recsBeforeDesync += len(recs) - 1
				k := key{last.TypeIndex, last.DesyncAt}
				desyncHist[k]++
				if last.TypeIndex == 0 {
					unboundSlots[last.Slot]++
				}
				if last.TypeIndex == 35 {
					bipedDesync[k]++
					if len(firstBipedDesyncSlots) < 12 {
						firstBipedDesyncSlots = append(firstBipedDesyncSlots, last.Slot)
					}
				}
			}
		}
	}

	fmt.Printf("frames=%d, avec desync=%d (%.0f%%), records OK avant desync (moy)=%.1f\n",
		totalFrames, framesWithDesync, 100*float64(framesWithDesync)/float64(totalFrames),
		float64(recsBeforeDesync)/float64(max(1, framesWithDesync)))

	fmt.Println("\n=== records bipèdes décodés OK par slot ===")
	for s := uint32(512); s <= 519; s++ {
		fmt.Printf("  slot%d : %d\n", s, bipedRecDecoded[s])
	}

	// top slots non-liés (desync typeIdx-0) : peu nombreux = fixable.
	type sc struct {
		slot uint32
		n    int
	}
	var us []sc
	for s, n := range unboundSlots {
		us = append(us, sc{s, n})
	}
	sort.Slice(us, func(i, j int) bool { return us[i].n > us[j].n })
	fmt.Printf("\n=== slots NON-LIÉS qui desync (%d distincts) — top 15 ===\n", len(us))
	for i := 0; i < 15 && i < len(us); i++ {
		fmt.Printf("  slot %d : %d desyncs\n", us[i].slot, us[i].n)
	}

	fmt.Println("\n=== TOP desyncs (typeIndex, composant iN) ===")
	type kv struct {
		k key
		n int
	}
	var sorted []kv
	for k, n := range desyncHist {
		sorted = append(sorted, kv{k, n})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].n > sorted[j].n })
	for i := 0; i < 20 && i < len(sorted); i++ {
		mark := ""
		if sorted[i].k.typeIdx == 35 {
			mark = "  <<< BIPED"
		}
		fmt.Printf("  typeIdx=%-3d i%-2d : %d desyncs%s\n", sorted[i].k.typeIdx, sorted[i].k.comp, sorted[i].n, mark)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
