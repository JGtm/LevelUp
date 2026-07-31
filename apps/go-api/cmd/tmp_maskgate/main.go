// tmp_maskgate — balayage (amorce de paquet x IDLowBits) juge par la FORME DU MASQUE du
// PREMIER record de chaque frame. THROWAWAY.
//
// ANTI-TAUTOLOGIE (piege rencontre a la version precedente de ce balayage) : un IDLowBits
// faux resout un slot INEXISTANT -> decodeDelta rend Mask=0 -> popcount 0 -> compte a tort
// comme « masque epars ». On EXIGE donc : record de type DELTA + slot present dans le
// world dump + popcount >= 1. Et on publie n, pour qu'une configuration qui « gagne » en
// ne decodant plus rien se voie immediatement.
//
// JUGE (verite terrain ce_capture_delta.csv, 138 390 records bipedes) :
//
//	popcount 1..7 : 99,86 %   |   mode 4 (43,6 %)   |   3 : 32,5 %   |   5 : 11,5 %
//
// La grammaire du masque (R(1) gate + R(3) count + count x R(6)) ne depend NI de la carte
// NI du film : ce profil est transposable tel quel a 000d5950.
//
// Le PREMIER record d'une frame ne depend d'AUCUN chainage : son curseur de depart est le
// bit 0 du payload (+ amorce). S'il est deja faux, aucun deser de composant n'est en cause.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_maskgate [filmdir]
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
	n, sparse int
	hist      map[int]int
	tiHist    map[uint32]int
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
	fmt.Printf("frames=%d bindings=%d slotsUniques=%d\n", len(frames), len(bs), len(slotTI))
	fmt.Println("JUGE (verite) : popcount 1..7 = 99,86 % ; mode 4 (43,6 %) ; 3 = 32,5 % ; 5 = 11,5 %")

	newWorld := func() *filmdec.World {
		w := filmdec.NewWorld(reg)
		for _, b := range bs {
			w.BindFull(b.id, b.ti)
		}
		return w
	}

	// on decode UNIQUEMENT le premier record de chaque frame (TryDeltaAt sur bit=pre),
	// pas la frame entiere : aucun chainage ne peut contaminer la mesure.
	measure := func(pre, idlow int, onlyBiped bool) res {
		r := res{hist: map[int]int{}, tiHist: map[uint32]int{}}
		cfg := filmdec.FrameConfig{IDLowBits: idlow}
		for _, pay := range frames {
			w := newWorld()
			rec, _, _ := filmdec.TryDeltaAt(pay, pre, w, cfg)
			ti, bound := slotTI[rec.Slot]
			if !bound {
				continue // slot inexistant : la config n'a pas resolu d'entite, on n'en tire rien
			}
			if onlyBiped && ti != 35 {
				continue
			}
			pc := bits.OnesCount64(rec.Trace.Mask)
			if pc == 0 {
				continue // masque vide : non informatif (et non observe dans la verite)
			}
			r.n++
			r.hist[pc]++
			r.tiHist[ti]++
			if pc <= 7 {
				r.sparse++
			}
		}
		return r
	}

	for _, onlyBiped := range []bool{true, false} {
		lbl := "BIPEDE (ti=35) seulement"
		if !onlyBiped {
			lbl = "TOUS archetypes bindes"
		}
		fmt.Printf("\n=== PREMIER record de frame — %s ===\n", lbl)
		fmt.Printf("%-8s %s\n", "pre\\id", "  (part popcount 1..7 / n)")
		for pre := 0; pre <= 6; pre++ {
			fmt.Printf("pre=%-4d", pre)
			for idlow := 9; idlow <= 16; idlow++ {
				r := measure(pre, idlow, onlyBiped)
				if r.n == 0 {
					fmt.Printf(" %14s", "--/0")
					continue
				}
				fmt.Printf(" %8.1f%%/%-4d", 100*float64(r.sparse)/float64(r.n), r.n)
			}
			fmt.Println()
		}
	}

	// detail du meilleur candidat + du candidat courant
	fmt.Println("\n=== profils detailles (bipede) ===")
	for _, c := range [][2]int{{0, 11}, {0, 13}, {2, 11}, {2, 13}, {2, 14}, {3, 11}, {1, 13}} {
		r := measure(c[0], c[1], true)
		if r.n == 0 {
			fmt.Printf("pre=%d idlow=%-2d : aucun record\n", c[0], c[1])
			continue
		}
		fmt.Printf("pre=%d idlow=%-2d : n=%-6d 1..7=%5.1f%%  popcount %s\n",
			c[0], c[1], r.n, 100*float64(r.sparse)/float64(r.n), top(r.hist, 6))
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
