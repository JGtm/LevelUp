// tmp_oraclediff — THROWAWAY : apparie la capture CE dead-state (oracle) aux frames
// offline par TAILLE de buffer (szF8 = end-f8), puis compare la position bit où mon
// walk atteint i11 (CompResult.StartBit) à b2c (position bit d'i11 mesurée en live).
// L'écart constant/variable localise le composant i0-10 qui sur/sous-lit.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_oraclediff [maxChunk]
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
const wt = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce`

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

type frame struct {
	size    int
	payload []byte
}

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
			out = append(out, frame{sz, d[off+16 : off+16+sz]})
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

	// 1) capture CE : (szF8, b2c) des records résolus
	raw, err := os.ReadFile(wt + "/000d5950_deadstate.bin")
	if err != nil {
		fmt.Printf("pas de capture: %v\n", err)
		return
	}
	const REC = 0x60
	u := func(off int, b []byte) uint32 { return binary.LittleEndian.Uint32(b[off:]) }
	type oracle struct {
		b2c int
		fp  []byte // 16 octets au début du buffer (frame start si f8==base frame)
	}
	var oracles []oracle
	for i := 0; i+REC <= len(raw); i += REC {
		b := raw[i : i+REC]
		if u(0x0c, b) == 0xffffffff {
			continue // non résolu
		}
		fp := append([]byte(nil), b[0x40:0x50]...)
		allZero := true
		for _, x := range fp {
			if x != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			continue
		}
		oracles = append(oracles, oracle{int(u(0x1c, b)), fp})
	}
	fmt.Printf("=== %d oracles résolus (avec fingerprint) ===\n", len(oracles))

	// 2) offline : pour chaque frame, chercher le fingerprint de chaque oracle dans le payload ;
	//    si trouvé au byte P, i11 attendu = P*8 + b2c ; comparer aux StartBit du dead-state biped.
	matched, diffHist := 0, map[int]int{}
	newDesync := map[string]int{} // recNew game-object qui désync -> (archetype, composant)
	nFrames := 0
	w := freshWorld(reg)                   // PERSISTE en ordre (NEW/DEL accumulent ; rollback par frame)
	for idx := 1; idx <= maxChunk; idx++ { // DÉMARRE à chunk_01 (recNew initiaux des game-objects)
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			nFrames++
			var matchedFp []byte
			var matchedB2c, matchedP int
			for oi := range oracles {
				if p := bytes.Index(fr.payload, oracles[oi].fp); p >= 0 {
					// la frame doit etre assez longue pour contenir i11 (anti faux-match 16o)
					if p*8+oracles[oi].b2c > len(fr.payload)*8 {
						continue
					}
					matchedFp, matchedB2c, matchedP = oracles[oi].fp, oracles[oi].b2c, p
					break
				}
			}
			br := filmdec.NewBitReader(fr.payload)
			recs, e := filmdec.DecodeFrameRecords(br, w, calCfg)
			deathSlots := map[uint32]bool{1030: true, 1038: true, 1068: true, 1086: true, 1494: true}
			for _, r := range recs {
				if r.Type != 1 || !deathSlots[r.Slot] {
					continue
				}
				if r.DesyncAt < 0 {
					newDesync[fmt.Sprintf("slot=%d ti=%d recNew CLEAN (bound, mais rollback?)", r.Slot, r.TypeIndex)]++
				} else if arch, ok := reg.Archetype(int(r.TypeIndex)); ok && r.DesyncAt < len(arch.Components) {
					newDesync[fmt.Sprintf("slot=%d ti=%d DESYNC i%d %s", r.Slot, r.TypeIndex, r.DesyncAt, arch.Components[r.DesyncAt])]++
				}
			}
			if matchedFp == nil {
				continue // advance World only
			}
			{
				matched++
				expected := matchedP*8 + matchedB2c
				best, found := 1<<30, false
				for _, r := range recs {
					if !bipedSlots[r.Slot] {
						continue
					}
					for _, c := range r.Trace.Comps {
						if c.Name == "object-dead-state-component" {
							d := c.StartBit - expected
							if d < 0 {
								d = -d
							}
							if d < best {
								best = c.StartBit - expected
								found = true
							}
						}
					}
				}
				// détail : où le décodage s'arrête vs i11 attendu
				endbit := -1
				dcomp := "?"
				slot0 := uint32(0)
				if len(recs) > 0 {
					last := recs[len(recs)-1]
					endbit = last.Trace.EndBit
					slot0 = recs[0].Slot
					if last.DesyncAt >= 0 {
						if arch, ok := reg.Archetype(int(last.TypeIndex)); ok && last.DesyncAt < len(arch.Components) {
							dcomp = fmt.Sprintf("i%d %s (ti=%d)", last.DesyncAt, arch.Components[last.DesyncAt], last.TypeIndex)
						}
					}
				}
				_, bound0 := w.ArchetypeForSlot(slot0)
				errStr := ""
				if e != nil {
					errStr = e.Error()
				}
				fmt.Printf("  [match] i11attendu=%d nrecs=%d slot0=%d bound0=%v offlineEnd=%d desync=%s found=%v\n    err: %s\n",
					expected, len(recs), slot0, bound0, endbit, dcomp, found, errStr)
				if found {
					diffHist[best]++
				} else {
					diffHist[1<<29]++
				}
			}
		}
	}
	fmt.Printf("=== %d frames offline ; %d fingerprints trouvés ===\n\n", nFrames, matched)
	fmt.Println("histogramme des écarts (StartBit_offline - i11attendu) :")
	type kv struct{ d, n int }
	var kvs []kv
	for d, n := range diffHist {
		kvs = append(kvs, kv{d, n})
	}
	sort.Slice(kvs, func(i, j int) bool { return kvs[i].n > kvs[j].n })
	for i, k := range kvs {
		if i >= 20 {
			break
		}
		fmt.Printf("  écart=%+d : %d\n", k.d, k.n)
	}

	fmt.Println("\nrecNew game-object (slot 1000-1500) qui DÉSYNCENT -> (archetype, composant à porter) :")
	type sk struct {
		s string
		n int
	}
	var sks []sk
	for s, n := range newDesync {
		sks = append(sks, sk{s, n})
	}
	sort.Slice(sks, func(i, j int) bool { return sks[i].n > sks[j].n })
	for i, k := range sks {
		if i >= 15 {
			break
		}
		fmt.Printf("  %5d  %s\n", k.n, k.s)
	}
}
