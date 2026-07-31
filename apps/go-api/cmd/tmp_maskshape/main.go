// tmp_maskshape — compare la FORME DU MASQUE de presence des records bipedes entre
// (a) la verite terrain CE (ce_capture_delta.csv) et (b) NOTRE walk sequentiel sur
// 000d5950. THROWAWAY.
//
// POURQUOI C'EST DECISIF : la grammaire du masque (FUN_1406d7610) ne depend NI de la
// carte NI du film — c'est R(1) gate + R(3) count + count x R(6) index, ou R(64) dense.
// Le nombre de composants presents par record et la frequence de presence de chaque
// composant sont donc DIRECTEMENT comparables entre les deux films, contrairement aux
// largeurs de quantification. Si notre walk voit des masques d'une forme radicalement
// differente, le curseur n'arrive deja plus au bon endroit a l'entree du masque.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_maskshape [filmdir]
// Env   : IDLOW (def 11)
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
	idlow := 11
	if v := os.Getenv("IDLOW"); v != "" {
		idlow, _ = strconv.Atoi(v)
	}
	pre := 0 // amorce de paquet (bits) sautee avant le premier record de la frame
	if v := os.Getenv("PRE"); v != "" {
		pre, _ = strconv.Atoi(v)
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	arch, _ := reg.Archetype(35)
	bs := loadBindings(dir)
	var frames []frame
	for i := 3; i <= 26; i++ {
		frames = append(frames, listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, i)))...)
	}
	fmt.Printf("frames=%d bindings=%d IDLOW=%d\n", len(frames), len(bs), idlow)

	cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: idlow}
	countHist := map[int]int{}
	presence := map[int]int{}
	bipedRecs := 0
	desync := 0
	for _, fr := range frames {
		w := filmdec.NewWorld(reg)
		for _, b := range bs {
			w.BindFull(b.id, b.ti)
		}
		br := filmdec.NewBitReader(fr.pay)
		br.Skip(pre)
		recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
		for _, r := range recs {
			if r.TypeIndex != 35 {
				continue
			}
			bipedRecs++
			if r.DesyncAt >= 0 {
				desync++
			}
			n := bits.OnesCount64(r.Trace.Mask)
			countHist[n]++
			for i := 0; i < 64; i++ {
				if r.Trace.Mask&(uint64(1)<<uint(i)) != 0 {
					presence[i]++
				}
			}
		}
	}
	fmt.Printf("records ti=35 vus par NOTRE walk : %d (dont desync %d)\n", bipedRecs, desync)

	fmt.Println("\n=== NOTRE walk : nb de composants presents par record ===")
	var ks []int
	for k := range countHist {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	for _, k := range ks {
		fmt.Printf("   %2d composants : %7d  (%.2f %%)\n", k, countHist[k], 100*float64(countHist[k])/float64(bipedRecs))
	}

	fmt.Println("\n=== NOTRE walk : presence par composant (bipede) ===")
	var cs []int
	for c := range presence {
		cs = append(cs, c)
	}
	sort.Ints(cs)
	for _, c := range cs {
		name := ""
		if c < len(arch.Components) {
			name = arch.Components[c]
		}
		fmt.Printf("i%02d  %-46s %7d  %6.2f %%\n", c, name, presence[c], 100*float64(presence[c])/float64(bipedRecs))
	}
}
