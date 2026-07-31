// tmp_l0witness — TEMOIN des correctifs L0 (polarite weapon-state-ammo, i56 energie).
// THROWAWAY : sert a mesurer AVANT/APRES, pas a decoder.
//
// Reproduit a l'identique le compteur de reference de cmd/tmp_absscore (replay()) :
// decodage SEQUENTIEL des paquets type-0, World seede par les bindings OFFLINE du
// keyframe (chunk_02), comptage des records bipeds clean (DesyncAt==-1) / desync +
// histogramme du composant de desync. C'est le couple 86019/66106 du journal.
//
// Il ajoute la mesure d'ATTEIGNABILITE, sans laquelle un temoin immobile n'est pas
// interpretable : combien de fois les composants VISES par les correctifs sont-ils
// effectivement atteints (presents dans le masque ET avant le desync) ?
//   - weapon-state-ammo (i30/33/36/39) : occurrences atteintes + histogramme des portes
//     gate1/gate2 relues DIRECTEMENT dans le buffer a CompResult.StartBit (donc
//     independant de la polarite du deser).
//   - biped-spartan-ability-energy (i56) : occurrences atteintes + histogramme du
//     masque R(3) (popcount 0 => le correctif ne change aucun bit sur ce site).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_l0witness [chunkLo chunkHi]
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

const (
	cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
	idLow = 11
)

var bipedSlots = map[uint32]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}

func inflate(p string) []byte {
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); (e2 == nil || e2 == io.ErrUnexpectedEOF) && len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

func listChunkFrames(d []byte, want uint16) [][]byte {
	var out [][]byte
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == want {
			out = append(out, d[off+16:off+16+sz])
		}
		off += 16 + sz
	}
	return out
}

func bit(buf []byte, p int) int {
	if p < 0 || p>>3 >= len(buf) {
		return -1
	}
	return int(buf[p>>3]>>(7-uint(p&7))) & 1
}

func bits3(buf []byte, p int) int {
	v := 0
	for i := 0; i < 3; i++ {
		b := bit(buf, p+i)
		if b < 0 {
			return -1
		}
		v = v<<1 | b
	}
	return v
}

func popcount3(v int) int {
	n := 0
	for i := 0; i < 3; i++ {
		n += (v >> uint(i)) & 1
	}
	return n
}

func main() {
	lo, hi := 3, 26
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[1], "%d", &lo)
		fmt.Sscanf(os.Args[2], "%d", &hi)
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	kf := listChunkFrames(inflate(cache+"/chunk_02.bin"), 2)
	if len(kf) == 0 {
		panic("keyframe introuvable")
	}
	binds := filmdec.WalkKeyframeWorld(kf[0])

	filmdec.SetRecordStateParam(2)
	cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: idLow}

	w := filmdec.NewWorld(reg)
	for _, b := range binds {
		w.BindFull(uint32((b.Gen<<30)|b.Slot), uint32(b.TI))
	}

	clean, desync := 0, 0
	cleanTi35, desyncTi35 := 0, 0
	desyncAt := map[string]int{}
	// atteignabilite des sites vises
	ammoHits := 0
	ammoGate := map[string]int{} // "g1=x g2=y"
	i56Hits := 0
	i56Mask := map[int]int{}
	// temoin de NON-REGRESSION : les morts (i11 object-dead-state) lues dans les
	// records bipeds. Les correctifs sont EN AVAL d'i11 dans le record, mais ils
	// deplacent le debut des records SUIVANTS dans la trame — donc l'invariance des
	// morts se MESURE, elle ne se deduit pas.
	deadRecords, deadMort := 0, 0
	deadSig := map[string]int{}
	compReached := map[string]int{} // nom -> nb de fois consomme proprement

	for c := lo; c <= hi; c++ {
		data := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, c))
		if data == nil {
			continue
		}
		for _, pay := range listChunkFrames(data, 0) {
			br := filmdec.NewBitReader(pay)
			recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
			for _, r := range recs {
				if r.TypeIndex == filmdec.BipedTypeIndex {
					if r.DesyncAt < 0 {
						cleanTi35++
					} else {
						desyncTi35++
					}
				}
				if !bipedSlots[r.Slot] {
					continue
				}
				if r.DesyncAt < 0 {
					clean++
				} else {
					desync++
					if arch, ok := reg.Archetype(int(r.TypeIndex)); ok && r.DesyncAt < len(arch.Components) {
						desyncAt[fmt.Sprintf("i%02d %s", r.DesyncAt, arch.Components[r.DesyncAt])]++
					}
				}
				if d := r.Trace.Dead; d != nil {
					deadRecords++
					if d.Mort {
						deadMort++
						deadSig[fmt.Sprintf("c%02d slot%d gid=%08x", c, r.Slot, d.GlobalID)]++
					}
				}
				for _, cr := range r.Trace.Comps {
					if !cr.Ported {
						continue
					}
					compReached[cr.Name]++
					switch cr.Name {
					case "weapon-state-ammo":
						ammoHits++
						g1 := bit(pay, cr.StartBit)
						g2p := cr.StartBit + 1
						if g1 == 0 {
							g2p += 8
						}
						ammoGate[fmt.Sprintf("g1=%d g2=%d", g1, bit(pay, g2p))]++
					case "biped-spartan-ability-energy", "biped-spartan-ability-energy-component":
						i56Hits++
						i56Mask[bits3(pay, cr.StartBit)]++
					}
				}
			}
		}
	}

	fmt.Printf("=== TEMOIN L0 (chunks %d..%d, film 000d5950) ===\n", lo, hi)
	fmt.Printf("records slots bipeds 512..519 : clean=%d  desync=%d  (total=%d)\n", clean, desync, clean+desync)
	fmt.Printf("records archetype ti=35       : clean=%d  desync=%d  (total=%d)\n", cleanTi35, desyncTi35, cleanTi35+desyncTi35)

	fmt.Println("\n-- histogramme du composant de desync (top 12) --")
	type kv struct {
		k string
		n int
	}
	var hs []kv
	for k, n := range desyncAt {
		hs = append(hs, kv{k, n})
	}
	sort.Slice(hs, func(i, j int) bool { return hs[i].n > hs[j].n })
	for i := 0; i < 12 && i < len(hs); i++ {
		fmt.Printf("  %-56s %d\n", hs[i].k, hs[i].n)
	}

	fmt.Printf("\n-- ATTEIGNABILITE des sites vises par les correctifs --\n")
	fmt.Printf("  weapon-state-ammo atteint            : %d fois\n", ammoHits)
	var ag []kv
	for k, n := range ammoGate {
		ag = append(ag, kv{k, n})
	}
	sort.Slice(ag, func(i, j int) bool { return ag[i].n > ag[j].n })
	for _, e := range ag {
		fmt.Printf("      %-12s %d\n", e.k, e.n)
	}
	fmt.Printf("  biped-spartan-ability-energy atteint : %d fois\n", i56Hits)
	var ms []int
	for m := range i56Mask {
		ms = append(ms, m)
	}
	sort.Ints(ms)
	for _, m := range ms {
		fmt.Printf("      masque=%d (popcount=%d, bits sup. attendus=%d) : %d\n", m, popcount3(m), 7*popcount3(m), i56Mask[m])
	}

	fmt.Printf("\n-- NON-REGRESSION morts (i11 object-dead-state) --\n")
	fmt.Printf("  records portant un dead-state : %d ; dont Mort=true : %d ; signatures distinctes : %d\n",
		deadRecords, deadMort, len(deadSig))
	if out := os.Getenv("DEADSIG_OUT"); out != "" {
		var sigs []string
		for k := range deadSig {
			sigs = append(sigs, k)
		}
		sort.Strings(sigs)
		f, err := os.Create(out)
		if err != nil {
			panic(err)
		}
		for _, s := range sigs {
			fmt.Fprintln(f, s)
		}
		f.Close()
		fmt.Printf("  signatures ecrites -> %s\n", out)
	}

	fmt.Println("\n-- composants bipeds les plus consommes (top 12) --")
	var cs []kv
	for k, n := range compReached {
		cs = append(cs, kv{k, n})
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].n > cs[j].n })
	for i := 0; i < 12 && i < len(cs); i++ {
		fmt.Printf("  %-56s %d\n", cs[i].k, cs[i].n)
	}
}
