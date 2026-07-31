// tmp_bitgap — MESURE L'ERREUR EN BITS a la fin du record 1 de chaque frame. THROWAWAY.
//
// Le record 1 (curseur de depart = bit 2, amorce de paquet) decode avec un masque epars
// dans 99,93 % des cas, conforme a la verite terrain. Le record 2 ne l'est que dans 45,6 %.
// L'ecart est donc dans la LONGUEUR du record 1. On la mesure directement : on cherche, pour
// chaque frame, le decalage d tel que demarrer le record 2 a (fin_calculee + d) produise un
// record DELTA sur un slot BINDE avec un masque epars (popcount 1..7). L'histogramme de d
// donne la taille ET le signe de l'erreur ; d=0 = notre longueur est bonne.
//
// On correle ensuite d avec le CONTENU du record 1 (archetype + composants presents) : le
// composant dont la presence predit un d non nul est le deser fautif.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_bitgap [filmdir]
// Env   : PRE (def 2), IDLOW (def 11), SPAN (def 96)
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

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	pre, idlow, span := 2, 11, 96
	if v := os.Getenv("PRE"); v != "" {
		pre, _ = strconv.Atoi(v)
	}
	if v := os.Getenv("IDLOW"); v != "" {
		idlow, _ = strconv.Atoi(v)
	}
	if v := os.Getenv("SPAN"); v != "" {
		span, _ = strconv.Atoi(v)
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
	fmt.Printf("frames=%d PRE=%d IDLOW=%d SPAN=+-%d\n", len(frames), pre, idlow, span)

	cfg := filmdec.FrameConfig{IDLowBits: idlow}
	newWorld := func() *filmdec.World {
		w := filmdec.NewWorld(reg)
		for _, b := range bs {
			w.BindFull(b.id, b.ti)
		}
		return w
	}
	healthy := func(pay []byte, at int, w *filmdec.World) bool {
		rec, _, _ := filmdec.TryDeltaAt(pay, at, w, cfg)
		if rec.Type != 3 {
			return false
		}
		if _, ok := slotTI[rec.Slot]; !ok {
			return false
		}
		pc := bits.OnesCount64(rec.Trace.Mask)
		return pc >= 1 && pc <= 7
	}

	gapHist := map[int]int{}
	gapByTI := map[uint32]map[int]int{}
	gapByComp := map[[2]uint32]map[int]int{}
	usable, none := 0, 0
	for _, pay := range frames {
		w := newWorld()
		rec1, end1, _ := filmdec.TryDeltaAt(pay, pre, w, cfg)
		if rec1.Type != 3 {
			continue
		}
		if _, ok := slotTI[rec1.Slot]; !ok {
			continue
		}
		pc := bits.OnesCount64(rec1.Trace.Mask)
		if pc < 1 || pc > 7 {
			continue // record 1 lui-meme suspect : on n'en tire rien
		}
		// plus petit |d| qui rend le record 2 sain
		best, found := 0, false
		for r := 0; r <= span && !found; r++ {
			for _, d := range []int{r, -r} {
				if r == 0 && d < 0 {
					continue
				}
				at := end1 + d
				if at < 0 || at >= len(pay)*8-24 {
					continue
				}
				if healthy(pay, at, w) {
					best, found = d, true
					break
				}
			}
		}
		if !found {
			none++
			continue
		}
		usable++
		gapHist[best]++
		ti := rec1.TypeIndex
		if gapByTI[ti] == nil {
			gapByTI[ti] = map[int]int{}
		}
		gapByTI[ti][best]++
		for _, c := range rec1.Trace.Comps {
			k := [2]uint32{ti, uint32(c.Index)}
			if gapByComp[k] == nil {
				gapByComp[k] = map[int]int{}
			}
			gapByComp[k][best]++
		}
	}
	fmt.Printf("frames exploitables=%d  (aucun d trouve : %d)\n", usable, none)

	fmt.Println("\n=== HISTOGRAMME DE L'ERREUR d (bits) a la fin du record 1 ===")
	fmt.Println("(d=0 : notre longueur est exacte ; d>0 : nous SOUS-consommons de d bits)")
	type kv struct{ k, v int }
	var a []kv
	for k, v := range gapHist {
		a = append(a, kv{k, v})
	}
	sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
	for i := 0; i < len(a) && i < 16; i++ {
		fmt.Printf("   d=%+4d : %6d  (%.2f %%)\n", a[i].k, a[i].v, 100*float64(a[i].v)/float64(usable))
	}

	fmt.Println("\n=== erreur par ARCHETYPE du record 1 ===")
	type tk struct {
		ti uint32
		h  map[int]int
	}
	var tks []tk
	for ti, h := range gapByTI {
		tks = append(tks, tk{ti, h})
	}
	sort.Slice(tks, func(i, j int) bool { return sum(tks[i].h) > sum(tks[j].h) })
	for i, t := range tks {
		if i >= 10 {
			break
		}
		n := sum(t.h)
		fmt.Printf("  ti=%-3d n=%6d  d=0 : %5.1f %%   top d : %s\n",
			t.ti, n, 100*float64(t.h[0])/float64(n), top(t.h, 5))
	}

	// CONTROLE DE HASARD : le critere « sain » (DELTA + slot binde + popcount 1..7) est-il
	// assez selectif pour que le pic a d=0 signifie quelque chose ? On l'evalue a des
	// decalages ARBITRAIRES (end1+300+k), ou aucun record ne commence : le taux de succes
	// obtenu est le NIVEAU DU HASARD. Si le pic d=0 est du meme ordre, il ne prouve rien.
	{
		pass, tot := 0, 0
		for _, pay := range frames {
			w := newWorld()
			rec1, end1, _ := filmdec.TryDeltaAt(pay, pre, w, cfg)
			if rec1.Type != 3 {
				continue
			}
			for k := 0; k < 20; k++ {
				at := end1 + 300 + k*7
				if at < 0 || at >= len(pay)*8-24 {
					continue
				}
				tot++
				if healthy(pay, at, w) {
					pass++
				}
			}
		}
		fmt.Printf("\n=== CONTROLE DE HASARD du critere « sain » ===\n")
		fmt.Printf("  decalages arbitraires testes : %d ; acceptes : %d  => niveau du hasard = %.2f %%\n",
			tot, pass, 100*float64(pass)/float64(tot))
	}

	fmt.Println("\n=== BISECTION : records a UN SEUL composant (blame sans ambiguite) ===")
	fmt.Println("(le record 1 ne contient qu'un composant : tout ecart d lui est imputable)")
	solo := map[[2]uint32]map[int]int{}
	for _, pay := range frames {
		w := newWorld()
		rec1, end1, _ := filmdec.TryDeltaAt(pay, pre, w, cfg)
		if rec1.Type != 3 || len(rec1.Trace.Comps) != 1 {
			continue
		}
		if _, ok := slotTI[rec1.Slot]; !ok {
			continue
		}
		if bits.OnesCount64(rec1.Trace.Mask) != 1 {
			continue
		}
		best, found := 0, false
		for r := 0; r <= span && !found; r++ {
			for _, d := range []int{r, -r} {
				if r == 0 && d < 0 {
					continue
				}
				at := end1 + d
				if at < 0 || at >= len(pay)*8-24 {
					continue
				}
				if healthy(pay, at, w) {
					best, found = d, true
					break
				}
			}
		}
		if !found {
			continue
		}
		k := [2]uint32{rec1.TypeIndex, uint32(rec1.Trace.Comps[0].Index)}
		if solo[k] == nil {
			solo[k] = map[int]int{}
		}
		solo[k][best]++
	}
	type sk struct {
		k [2]uint32
		h map[int]int
	}
	var sks []sk
	for k, h := range solo {
		sks = append(sks, sk{k, h})
	}
	sort.Slice(sks, func(i, j int) bool { return sum(sks[i].h) > sum(sks[j].h) })
	fmt.Printf("%-5s %-4s %-46s %7s %8s   %s\n", "ti", "i", "composant", "n", "d=0", "top d")
	for i, s := range sks {
		if i >= 20 {
			break
		}
		n := sum(s.h)
		name := ""
		if arch, ok := reg.Archetype(int(s.k[0])); ok && int(s.k[1]) < len(arch.Components) {
			name = arch.Components[s.k[1]]
		}
		fmt.Printf("%-5d i%02d  %-46s %7d %7.1f %%   %s\n",
			s.k[0], s.k[1], name, n, 100*float64(s.h[0])/float64(n), top(s.h, 5))
	}

	fmt.Println("\n=== erreur par COMPOSANT present dans le record 1 (trie par effectif) ===")
	fmt.Printf("%-5s %-4s %-46s %7s %8s   %s\n", "ti", "i", "composant", "n", "d=0", "top d")
	type ck struct {
		k [2]uint32
		h map[int]int
	}
	var cks []ck
	for k, h := range gapByComp {
		cks = append(cks, ck{k, h})
	}
	sort.Slice(cks, func(i, j int) bool { return sum(cks[i].h) > sum(cks[j].h) })
	for i, c := range cks {
		if i >= 30 {
			break
		}
		n := sum(c.h)
		name := ""
		if arch, ok := reg.Archetype(int(c.k[0])); ok && int(c.k[1]) < len(arch.Components) {
			name = arch.Components[c.k[1]]
		}
		fmt.Printf("%-5d i%02d  %-46s %7d %7.1f %%   %s\n",
			c.k[0], c.k[1], name, n, 100*float64(c.h[0])/float64(n), top(c.h, 4))
	}
}

func sum(h map[int]int) int {
	t := 0
	for _, v := range h {
		t += v
	}
	return t
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
		sb = append(sb, fmt.Sprintf("%+d:%.0f%%", a[i].k, 100*float64(a[i].v)/float64(tot)))
	}
	return strings.Join(sb, " ")
}
