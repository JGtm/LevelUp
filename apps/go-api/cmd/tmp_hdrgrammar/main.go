// tmp_hdrgrammar — balayage (amorce de paquet x IDLowBits) juge par le CHAINAGE COMPLET
// d'une frame, pas seulement par son premier record. THROWAWAY.
//
// POURQUOI ce juge et pas la forme du masque du 1er record : plusieurs paramétrages
// donnent EXACTEMENT le même masque au premier record (ex. pre=2/idLow=11 et
// pre=1/idLow=12 ne diffèrent que par le bit de poids fort de l'id : même gate, même
// count, mêmes index). Seul le CHAINAGE les sépare, car l'en-tête des records SUIVANTS
// ne porte plus l'amorce : sa largeur vaut 1+idLow+2+1+3, donc elle dépend d'idLow seul.
//
// CRITERE PRINCIPAL (dur, non tautologique) : la frame se termine-t-elle par un record
// de type END (DecodeFrameRecords rend err==nil) avec un reliquat de bits < 8 ? Terminer
// proprement un payload de plusieurs centaines d'octets par hasard est hors de portée.
//
// CRITERES SECONDAIRES : profondeur atteinte (records avant désync), part des masques
// 1..7 sur les DELTA de slots bindés (vérité terrain : 99,86 %).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_hdrgrammar [filmdir]
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

func loadBindings(dir string) ([]binding, map[uint32]uint32) {
	raw, _ := os.ReadFile(dir + "/world_dump_full.txt")
	var bs []binding
	slotTI := map[uint32]uint32{}
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
			slotTI[uint32(id)&0x3fffffff] = uint32(ti)
		}
	}
	return bs, slotTI
}

type res struct {
	frames   int
	clean    int // frame terminée par END avec reliquat < 8 bits
	endOK    int // frame terminée par END (err==nil), quel que soit le reliquat
	records  int
	deltaB   int // DELTA sur slot bindé, popcount >= 1
	sparse   int // parmi deltaB, popcount 1..7
	depthSum int
	consumed float64
	maskHist map[int]int
	deepest  int
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
	bs, slotTI := loadBindings(dir)
	var frames [][]byte
	for i := 3; i <= 26; i++ {
		frames = append(frames, listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, i)))...)
	}
	fmt.Printf("film=%s frames(type0)=%d bindings=%d slots=%d\n", dir, len(frames), len(bs), len(slotTI))
	fmt.Println("JUGE PRINCIPAL : part des frames terminées par un record END avec reliquat < 8 bits.")

	newWorld := func() *filmdec.World {
		w := filmdec.NewWorld(reg)
		for _, b := range bs {
			w.BindFull(b.id, b.ti)
		}
		return w
	}

	// minBytes écarte les frames minuscules : sur celles-là, « terminer proprement » est
	// trivial (le payload tient en un ou deux records) et le critère devient tautologique.
	minBytes := 40
	if v := os.Getenv("MINBYTES"); v != "" {
		minBytes, _ = strconv.Atoi(v)
	}
	nBig := 0
	for _, p := range frames {
		if len(p) >= minBytes {
			nBig++
		}
	}
	fmt.Printf("frames de payload >= %d octets : %d (%.1f %%)\n", minBytes, nBig, 100*float64(nBig)/float64(len(frames)))

	measure := func(pre, idlow int) res {
		r := res{maskHist: map[int]int{}}
		cfg := filmdec.FrameConfig{IDLowBits: idlow, PacketPreambleBits: pre}
		for _, pay := range frames {
			if len(pay) < minBytes {
				continue
			}
			w := newWorld()
			br := filmdec.NewBitReader(pay)
			recs, derr := filmdec.DecodeFrameRecords(br, w, cfg)
			r.frames++
			r.records += len(recs)
			r.depthSum += len(recs)
			cons := float64(br.BitPos()) / float64(len(pay)*8)
			if cons > 1 {
				cons = 1 // plafonné : au-delà, le décodeur lit dans le vide
			}
			r.consumed += cons
			if len(recs) > r.deepest {
				r.deepest = len(recs)
			}
			if derr == nil {
				r.endOK++
				// PIEGE CORRIGE : le BitReader lit au-delà de la fin (bits nuls) sans
				// broncher, donc BitPos peut DEPASSER le payload et le reste devenir
				// négatif. Exiger 0 <= reste < 8, sinon une config qui dérape est comptée
				// « clean ».
				if rest := len(pay)*8 - br.BitPos(); rest >= 0 && rest < 8 {
					r.clean++
				}
			}
			for _, rc := range recs {
				if rc.Type != 3 {
					continue
				}
				if _, ok := slotTI[rc.Slot]; !ok {
					continue
				}
				pc := bits.OnesCount64(rc.Trace.Mask)
				if pc == 0 {
					continue
				}
				r.deltaB++
				r.maskHist[pc]++
				if pc <= 7 {
					r.sparse++
				}
			}
		}
		return r
	}

	fmt.Println("\n=== BALAYAGE : part de frames CLEAN (END + reliquat<8) / profondeur moyenne ===")
	fmt.Printf("%-8s", "pre\\id")
	for idlow := 9; idlow <= 15; idlow++ {
		fmt.Printf(" %16d", idlow)
	}
	fmt.Println()
	best := [2]int{-1, -1}
	bestClean := -1
	for pre := 0; pre <= 4; pre++ {
		fmt.Printf("pre=%-4d", pre)
		for idlow := 9; idlow <= 15; idlow++ {
			r := measure(pre, idlow)
			cl := 100 * float64(r.clean) / float64(r.frames)
			fmt.Printf(" %7.2f%%/%-7.2f", cl, r.consumed/float64(r.frames))
			if r.clean > bestClean {
				bestClean, best = r.clean, [2]int{pre, idlow}
			}
		}
		fmt.Println()
	}
	fmt.Printf("\nMEILLEUR : pre=%d idlow=%d (%d frames clean)\n", best[0], best[1], bestClean)

	fmt.Println("\n=== profils détaillés ===")
	for _, c := range [][2]int{{0, 11}, {0, 13}, {1, 11}, {1, 12}, {2, 10}, {2, 11}, {2, 12}, {3, 11}, best} {
		r := measure(c[0], c[1])
		fmt.Printf("pre=%d idlow=%-2d : frames=%d cleanEND=%5.2f%% endOK=%5.2f%% payload=%5.1f%% prof=%6.1f max=%d deltaBindes=%-7d masques1..7=%5.2f%% | %s\n",
			c[0], c[1], r.frames,
			100*float64(r.clean)/float64(r.frames), 100*float64(r.endOK)/float64(r.frames),
			100*r.consumed/float64(r.frames),
			float64(r.depthSum)/float64(r.frames), r.deepest,
			r.deltaB, 100*float64(r.sparse)/float64(max(r.deltaB, 1)), top(r.maskHist, 8))
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
