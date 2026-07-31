// tmp_grencause — THROWAWAY : test CAUSAL de la derive de chaine.
//
// L'echelle de continuite (tmp_grenladder) montre du signal jusqu'a i6 puis plus rien.
// Si un composant amont consomme une mauvaise largeur, alors les records ou CE composant
// est ABSENT du masque devraient garder du signal en aval. On stratifie donc la mesure de
// continuite d'un composant CIBLE (i22/i47/i48/i56) selon la presence d'un composant
// SUSPECT amont (i7, i9, i15, ...), et on compare a l'alea +37 dans chaque strate.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_grencause [chunkLo chunkHi]
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

type key struct {
	target  int
	suspect int
	present bool
}
type acc struct{ stable, total, aStable, aTotal int }

func main() {
	lo, hi := 1, 27
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[1], "%d", &lo)
		fmt.Sscanf(os.Args[2], "%d", &hi)
	}
	targets := []int{22, 47, 48, 56}
	suspects := []int{7, 9, 11, 15, 17, 19, 21}

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

	accs := map[key]*acc{}
	lastV := map[key]map[uint32]uint64{}
	lastA := map[key]map[uint32]uint64{}

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
				startOf := map[int]int{}
				for _, cr := range r.Trace.Comps {
					if cr.Ported {
						startOf[cr.Index] = cr.StartBit
					}
				}
				for _, t := range targets {
					sb, ok := startOf[t]
					if !ok {
						continue
					}
					v := bitsAt(pay, sb, 8)
					av := bitsAt(pay, sb+37, 8)
					for _, s := range suspects {
						k := key{t, s, r.Trace.Mask&(uint64(1)<<uint(s)) != 0}
						if accs[k] == nil {
							accs[k] = &acc{}
							lastV[k] = map[uint32]uint64{}
							lastA[k] = map[uint32]uint64{}
						}
						a := accs[k]
						if prev, ok := lastV[k][r.Slot]; ok {
							a.total++
							if prev == v {
								a.stable++
							}
						}
						if prev, ok := lastA[k][r.Slot]; ok {
							a.aTotal++
							if prev == av {
								a.aStable++
							}
						}
						lastV[k][r.Slot] = v
						lastA[k][r.Slot] = av
					}
				}
			}
		}
	}

	fmt.Printf("=== TEST CAUSAL : continuite d'un composant CIBLE selon la presence d'un SUSPECT amont ===\n")
	fmt.Printf("(chunks %d..%d ; ratio = %%stab / %%alea+37 ; ratio ~1 = bruit, >1.5 = signal)\n\n", lo, hi)
	fmt.Printf("%-6s %-9s %-9s %8s %8s %9s %7s\n", "cible", "suspect", "presence", "n", "%stab", "%alea+37", "ratio")
	var ks []key
	for k := range accs {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool {
		if ks[i].target != ks[j].target {
			return ks[i].target < ks[j].target
		}
		if ks[i].suspect != ks[j].suspect {
			return ks[i].suspect < ks[j].suspect
		}
		return !ks[i].present && ks[j].present
	})
	for _, k := range ks {
		a := accs[k]
		if a.total < 100 {
			continue
		}
		st := 100 * float64(a.stable) / float64(a.total)
		al := 100 * float64(a.aStable) / float64(maxi(a.aTotal, 1))
		pres := "ABSENT"
		if k.present {
			pres = "present"
		}
		fmt.Printf("i%-5d i%-8d %-9s %8d %7.2f%% %8.2f%% %7.2f\n", k.target, k.suspect, pres, a.total, st, al, st/maxf(al, 0.01))
	}
}

func maxi(a, b int) int {
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
