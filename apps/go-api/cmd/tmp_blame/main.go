// tmp_blame — attribue la RUPTURE DE CHAINAGE a un composant. THROWAWAY.
//
// Une fois l'amorce de paquet de 2 bits retablie (PRE=2), le PREMIER record de chaque frame
// decode avec un masque epars dans 99,9 % des cas — conforme a la verite terrain
// (ce_capture_delta.csv : 99,86 % des records portent <= 7 composants). La chaine casse
// ensuite : un record dont le successeur presente un masque a >= 8 composants signale que
// la LONGUEUR du record courant est fausse — donc qu'un de ses desers de composant
// sur- ou sous-consomme.
//
// SCORE DE BLAME : pour chaque composant c, on compte les records qui le contiennent et
// dont le successeur est CASSE (popcount >= 8) contre ceux dont le successeur est SAIN
// (popcount 1..7). Un composant a fort taux de casse ET presence significative est le
// suspect. C'est une correlation, pas une preuve : on publie les deux effectifs pour que
// le lecteur juge (un composant vu 3 fois a 100 % de casse ne prouve rien).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_blame [filmdir]
// Env   : PRE (def 2), IDLOW (def 11)
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

type stat struct{ ok, bad int }

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	pre, idlow := 2, 11
	if v := os.Getenv("PRE"); v != "" {
		pre, _ = strconv.Atoi(v)
	}
	if v := os.Getenv("IDLOW"); v != "" {
		idlow, _ = strconv.Atoi(v)
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
	fmt.Printf("frames=%d PRE=%d IDLOW=%d\n", len(frames), pre, idlow)

	cfg := filmdec.FrameConfig{IDLowBits: idlow}
	rankOK := map[int]*stat{}
	blame := map[[2]uint32]*stat{} // (ti, compIndex) du record COURANT -> etat du successeur
	soloTI := map[uint32]*stat{}   // archetype du record courant
	for _, pay := range frames {
		w := filmdec.NewWorld(reg)
		for _, b := range bs {
			w.BindFull(b.id, b.ti)
		}
		br := filmdec.NewBitReader(pay)
		br.Skip(pre)
		recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
		for i := 0; i < len(recs); i++ {
			pc := bits.OnesCount64(recs[i].Trace.Mask)
			if recs[i].Type != 3 {
				continue
			}
			rk := i + 1
			if rk > 8 {
				rk = 8
			}
			if rankOK[rk] == nil {
				rankOK[rk] = &stat{}
			}
			if pc >= 8 {
				rankOK[rk].bad++
			} else if pc >= 1 {
				rankOK[rk].ok++
			}
			// blame : etat du SUCCESSEUR
			if i+1 >= len(recs) || recs[i+1].Type != 3 {
				continue
			}
			npc := bits.OnesCount64(recs[i+1].Trace.Mask)
			if npc == 0 {
				continue // non informatif
			}
			bad := npc >= 8
			ti := recs[i].TypeIndex
			if soloTI[ti] == nil {
				soloTI[ti] = &stat{}
			}
			if bad {
				soloTI[ti].bad++
			} else {
				soloTI[ti].ok++
			}
			for _, c := range recs[i].Trace.Comps {
				k := [2]uint32{ti, uint32(c.Index)}
				if blame[k] == nil {
					blame[k] = &stat{}
				}
				if bad {
					blame[k].bad++
				} else {
					blame[k].ok++
				}
			}
		}
	}

	fmt.Println("\n=== sante du masque par RANG du record dans la frame (tous archetypes) ===")
	fmt.Println("(sain = popcount 1..7 ; casse = popcount >= 8 ; verite : 99,86 % sain)")
	var rks []int
	for k := range rankOK {
		rks = append(rks, k)
	}
	sort.Ints(rks)
	for _, k := range rks {
		s := rankOK[k]
		tot := s.ok + s.bad
		lbl := fmt.Sprintf("rang %d", k)
		if k == 8 {
			lbl = "rang >=8"
		}
		fmt.Printf("  %-9s n=%6d   sain %6.2f %%\n", lbl, tot, 100*float64(s.ok)/float64(tot))
	}

	fmt.Println("\n=== taux de casse du SUCCESSEUR, par archetype du record courant ===")
	type tk struct {
		ti uint32
		s  *stat
	}
	var tks []tk
	for ti, s := range soloTI {
		tks = append(tks, tk{ti, s})
	}
	sort.Slice(tks, func(i, j int) bool { return tks[i].s.ok+tks[i].s.bad > tks[j].s.ok+tks[j].s.bad })
	for i, t := range tks {
		if i >= 12 {
			break
		}
		tot := t.s.ok + t.s.bad
		fmt.Printf("  ti=%-3d n=%6d  casse %6.2f %%\n", t.ti, tot, 100*float64(t.s.bad)/float64(tot))
	}

	fmt.Println("\n=== BLAME par composant (record courant) — trie par effectif ===")
	fmt.Printf("%-5s %-4s %-46s %7s %8s\n", "ti", "i", "composant", "n", "casse")
	type bk struct {
		k [2]uint32
		s *stat
	}
	var bks []bk
	for k, s := range blame {
		bks = append(bks, bk{k, s})
	}
	sort.Slice(bks, func(i, j int) bool {
		ni, nj := bks[i].s.ok+bks[i].s.bad, bks[j].s.ok+bks[j].s.bad
		return ni > nj
	})
	for i, b := range bks {
		if i >= 40 {
			break
		}
		tot := b.s.ok + b.s.bad
		name := ""
		if a, ok := reg.Archetype(int(b.k[0])); ok && int(b.k[1]) < len(a.Components) {
			name = a.Components[b.k[1]]
		}
		fmt.Printf("%-5d i%02d  %-46s %7d %7.2f %%\n", b.k[0], b.k[1], name, tot, 100*float64(b.s.bad)/float64(tot))
	}
}
