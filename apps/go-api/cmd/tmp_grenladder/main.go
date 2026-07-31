// tmp_grenladder — THROWAWAY : ECHELLE DE CONTINUITE composant par composant.
//
// Objet : i4/i5 (vie/bouclier) portent un VRAI signal temporel, i22/i47/i48/i56 non.
// Si la chaine de composants derive, la derive commence quelque part entre i5 et i22.
// Cet outil mesure, pour CHAQUE composant present de l'archetype biped, la stabilite
// echantillon-a-echantillon de ses 8 premiers bits (par slot joueur), et la compare a
// un ALEA pris au MEME endroit decale de +37 bits. Le rapport stabilite/alea est le
// signal : ~1 = indiscernable du bruit, >>1 = vraie donnee.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_grenladder [chunkLo chunkHi]
package main

import (
	"bytes"
	"compress/zlib"
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

func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p < 0 || p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

type acc struct {
	name             string
	stable, total    int
	aStable, aTotal  int
	widthSum, widthN int
	distinct         map[uint64]int
}

func main() {
	lo, hi := 1, 27
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[1], "%d", &lo)
		fmt.Sscanf(os.Args[2], "%d", &hi)
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	kfChunk := inflate(cache + "/chunk_02.bin")
	var kf []byte
	for _, p := range filmdec.WalkPackets(kfChunk) {
		if p.Type == filmdec.PacketTypeKeyframe {
			kf = p.Payload(kfChunk)
			break
		}
	}
	binds := filmdec.WalkKeyframeWorld(kf)
	filmdec.SetRecordStateParam(2)
	cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: idLow}
	w := filmdec.NewWorld(reg)
	for _, b := range binds {
		w.BindFull(uint32((b.Gen<<30)|b.Slot), uint32(b.TI))
	}

	accs := map[int]*acc{}
	lastV := map[int]map[uint32]uint64{}
	lastA := map[int]map[uint32]uint64{}

	for c := lo; c <= hi; c++ {
		chunk := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, c))
		if chunk == nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta {
				continue
			}
			pay := p.Payload(chunk)
			br := filmdec.NewBitReader(pay)
			recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
			for _, r := range recs {
				if r.TypeIndex != filmdec.BipedTypeIndex || r.Slot < 512 || r.Slot > 519 {
					continue
				}
				comps := r.Trace.Comps
				for ci, cr := range comps {
					if !cr.Ported {
						continue
					}
					a := accs[cr.Index]
					if a == nil {
						a = &acc{name: cr.Name, distinct: map[uint64]int{}}
						accs[cr.Index] = a
						lastV[cr.Index] = map[uint32]uint64{}
						lastA[cr.Index] = map[uint32]uint64{}
					}
					// largeur reelle consommee (jusqu'au composant suivant)
					if ci+1 < len(comps) {
						a.widthSum += comps[ci+1].StartBit - cr.StartBit
						a.widthN++
					}
					v := bitsAt(pay, cr.StartBit, 8)
					av := bitsAt(pay, cr.StartBit+37, 8)
					a.distinct[v]++
					if prev, ok := lastV[cr.Index][r.Slot]; ok {
						a.total++
						if prev == v {
							a.stable++
						}
					}
					if prev, ok := lastA[cr.Index][r.Slot]; ok {
						a.aTotal++
						if prev == av {
							a.aStable++
						}
					}
					lastV[cr.Index][r.Slot] = v
					lastA[cr.Index][r.Slot] = av
				}
			}
		}
	}

	var idx []int
	for i := range accs {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	fmt.Printf("=== ECHELLE DE CONTINUITE, archetype biped, slots joueur (chunks %d..%d) ===\n", lo, hi)
	fmt.Printf("%-4s %-44s %7s %8s %8s %7s %8s %6s\n",
		"i", "composant", "n", "%stab", "%alea+37", "ratio", "largeur", "modale%")
	for _, i := range idx {
		a := accs[i]
		if a.total < 50 {
			continue
		}
		st := 100 * float64(a.stable) / float64(a.total)
		al := 100 * float64(a.aStable) / float64(max(a.aTotal, 1))
		ratio := st / maxf(al, 0.01)
		wavg := float64(a.widthSum) / float64(max(a.widthN, 1))
		var tot, best int
		for _, n := range a.distinct {
			tot += n
			if n > best {
				best = n
			}
		}
		fmt.Printf("i%-3d %-44s %7d %7.2f%% %7.2f%% %7.2f %7.1f %5.1f%%\n",
			i, a.name, a.total, st, al, ratio, wavg, 100*float64(best)/float64(max(tot, 1)))
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
