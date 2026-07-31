// tmp_maskrank — LOCALISE la desynchronisation : forme du masque bipede par RANG du record
// dans la frame, plus balayage (preambule de paquet x IDLowBits). THROWAWAY.
//
// CRITERE ABSOLU (verite terrain ce_capture_delta.csv, 138 390 records bipedes) :
//
//	99,86 % des records DELTA bipedes portent <= 7 composants (branche EPARSE du masque).
//	Repartition : 3 comps 32,5 % / 4 comps 43,6 % / 5 comps 11,5 %.
//
// La grammaire du masque ne depend NI de la carte NI du film : ce critere est directement
// transposable a 000d5950. Un walk qui produit des masques a 15-25 composants lit le gate
// du masque au mauvais bit.
//
// Rang 1 = premier record de la frame : son curseur de depart est CONNU (bit 0 + preambule),
// aucun chainage en amont. Si le rang 1 est deja faux, la faute est dans l'amorce de paquet
// ou l'en-tete de record ; si le rang 1 est bon et que ca degrade au rang 2, la faute est
// dans la LONGUEUR du record precedent (un deser de composant).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_maskrank [filmdir]
package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"math/bits"
	"os"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

const defFilm = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

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

type binding struct{ id, ti uint32 }

func loadBindings(dir string) []binding {
	raw, _ := os.ReadFile(dir + "/world_dump_full.txt")
	var bs []binding
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		for _, tok := range strings.Fields(line) {
			kv := strings.SplitN(tok, ":", 2)
			if len(kv) != 2 {
				continue
			}
			id, e1 := strconv.ParseUint(kv[0], 10, 64)
			ti, e2 := strconv.Atoi(kv[1])
			if e1 != nil || e2 != nil {
				continue
			}
			bs = append(bs, binding{uint32(id), uint32(ti)})
		}
	}
	return bs
}

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	bs := loadBindings(dir)
	var frames [][]byte
	for i := 3; i <= 26; i++ {
		frames = append(frames, listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, i)))...)
	}
	fmt.Printf("frames=%d bindings=%d\n\n", len(frames), len(bs))

	newWorld := func() *filmdec.World {
		w := filmdec.NewWorld(reg)
		for _, b := range bs {
			w.BindFull(b.id, b.ti)
		}
		return w
	}

	// ---- (A) forme du masque par RANG, config courante (IDLow=11, preambule 0) ----
	fmt.Println("=== (A) forme du masque bipede par RANG du record (IDLow=11, preambule 0) ===")
	fmt.Println("critere verite : <=7 composants dans 99,86 % des records ; mode = 4 comps")
	rankSparse := map[int]int{}
	rankTot := map[int]int{}
	rankCnt := map[int]map[int]int{}
	for _, pay := range frames {
		w := newWorld()
		br := filmdec.NewBitReader(pay)
		recs, _ := filmdec.DecodeFrameRecords(br, w, filmdec.FrameConfig{IDLowBits: 11})
		rank := 0
		for _, r := range recs {
			rank++
			if r.TypeIndex != 35 {
				continue
			}
			rk := rank
			if rk > 6 {
				rk = 6 // 6 = "rang >= 6"
			}
			n := bits.OnesCount64(r.Trace.Mask)
			rankTot[rk]++
			if n <= 7 {
				rankSparse[rk]++
			}
			if rankCnt[rk] == nil {
				rankCnt[rk] = map[int]int{}
			}
			rankCnt[rk][n]++
		}
	}
	var rks []int
	for k := range rankTot {
		rks = append(rks, k)
	}
	sort.Ints(rks)
	for _, k := range rks {
		lbl := fmt.Sprintf("rang %d", k)
		if k == 6 {
			lbl = "rang >=6"
		}
		fmt.Printf("  %-9s n=%6d   <=7 comps : %6.2f %%   top popcount : %s\n",
			lbl, rankTot[k], 100*float64(rankSparse[k])/float64(rankTot[k]), top(rankCnt[k], 4))
	}

	// ---- (B) balayage preambule de paquet x IDLowBits, sur le PREMIER record seulement ----
	// Le premier record d'une frame ne depend d'aucun chainage : sa seule inconnue est
	// l'amorce de paquet et la largeur d'id. On mesure la part de records (tous archetypes
	// confondus dont le slot est bindé) dont le masque est epars.
	fmt.Println("\n=== (B) PREMIER record de chaque frame : part de masques epars (<=7 comps) ===")
	fmt.Printf("%-6s", "pre\\id")
	ids := []int{9, 10, 11, 12, 13, 14, 15}
	for _, id := range ids {
		fmt.Printf(" %8d", id)
	}
	fmt.Println()
	for pre := 0; pre <= 4; pre++ {
		fmt.Printf("%-6d", pre)
		for _, idlow := range ids {
			sparse, tot := 0, 0
			for _, pay := range frames {
				w := newWorld()
				br := filmdec.NewBitReader(pay)
				br.Skip(pre)
				recs, _ := filmdec.DecodeFrameRecords(br, w, filmdec.FrameConfig{IDLowBits: idlow})
				if len(recs) == 0 {
					continue
				}
				r := recs[0]
				if r.Type != 3 || r.TypeIndex == 0 && r.Trace.Mask == 0 {
					// on ne garde que les DELTA dont l'archetype a ete resolu
				}
				if r.Type != 3 {
					continue
				}
				tot++
				if bits.OnesCount64(r.Trace.Mask) <= 7 {
					sparse++
				}
			}
			if tot == 0 {
				fmt.Printf("      n/a")
				continue
			}
			fmt.Printf(" %7.1f%%", 100*float64(sparse)/float64(tot))
		}
		fmt.Println()
	}

	// ---- (C) idem mais sur TOUS les records, pour voir l'effet global ----
	fmt.Println("\n=== (C) TOUS les records DELTA bipede : part de masques epars (<=7 comps) ===")
	fmt.Printf("%-6s", "pre\\id")
	for _, id := range ids {
		fmt.Printf(" %8d", id)
	}
	fmt.Println()
	for pre := 0; pre <= 4; pre++ {
		fmt.Printf("%-6d", pre)
		for _, idlow := range ids {
			sparse, tot := 0, 0
			for _, pay := range frames {
				w := newWorld()
				br := filmdec.NewBitReader(pay)
				br.Skip(pre)
				recs, _ := filmdec.DecodeFrameRecords(br, w, filmdec.FrameConfig{IDLowBits: idlow})
				for _, r := range recs {
					if r.TypeIndex != 35 {
						continue
					}
					tot++
					if bits.OnesCount64(r.Trace.Mask) <= 7 {
						sparse++
					}
				}
			}
			if tot == 0 {
				fmt.Printf("      n/a")
				continue
			}
			fmt.Printf(" %6.1f%%(%d)", 100*float64(sparse)/float64(tot), tot)
		}
		fmt.Println()
	}
}

func top(h map[int]int, k int) string {
	type kv struct{ k, v int }
	var a []kv
	tot := 0
	for x, y := range h {
		a = append(a, kv{x, y})
		tot += y
	}
	sort.Slice(a, func(i, j int) bool {
		if a[i].v != a[j].v {
			return a[i].v > a[j].v
		}
		return a[i].k < a[j].k
	})
	var sb []string
	for i := 0; i < len(a) && i < k; i++ {
		sb = append(sb, fmt.Sprintf("%d:%.0f%%", a[i].k, 100*float64(a[i].v)/float64(tot)))
	}
	return strings.Join(sb, " ")
}
