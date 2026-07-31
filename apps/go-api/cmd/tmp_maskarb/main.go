// tmp_maskarb — ARBITRAGE du desaccord d'en-tete entre les deux lecteurs.
// THROWAWAY : mesure, ne decode pas.
//
// Compare, sur le MEME corpus (film 000d5950, paquets type-0), les positions bit des
// records bipeds trouves par :
//
//	A) le WALK SEQUENTIEL (DebugDecodeFrame, cfg.IDLowBits=11) -> StartBit par record ;
//	B) le MATCHEUR OFFLINE valide (grammaire 21 bits : [1][13 slot][2 tag][2 zeros][3 mc]
//	   + mc x R(6) index croissants depuis 0), reimplemente ici a l'identique.
//
// et publie, pour chaque record clean ti=35 du walk, le PREAMBULE REELLEMENT CONSOMME
// (Comps[0].StartBit - StartBit) face aux deux formules candidates.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_maskarb [chunkLo chunkHi]
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

func bitAt(buf []byte, p int) int {
	if p < 0 || p>>3 >= len(buf) {
		return -1
	}
	return int(buf[p>>3]>>(7-uint(p&7))) & 1
}

func bitsAt(buf []byte, p, n int) int {
	v := 0
	for i := 0; i < n; i++ {
		b := bitAt(buf, p+i)
		if b < 0 {
			return -1
		}
		v = v<<1 | b
	}
	return v
}

func bitStr(buf []byte, p, n int) string {
	s := make([]byte, 0, n)
	for i := 0; i < n; i++ {
		b := bitAt(buf, p+i)
		if b < 0 {
			s = append(s, '?')
			continue
		}
		s = append(s, byte('0'+b))
	}
	return string(s)
}

// matcherOffline reimplemente matchBipedHeaderRaw (grammaire 21 bits) et rend les
// positions acceptees dans le payload.
func matcherOffline(pay []byte, lay filmdec.I0Layout, needTag1 bool) []int {
	total := len(pay) * 8
	i0Bits := lay.TotalBits()
	minRecord := 21 + 6*2 + i0Bits
	var out []int
	for p := 0; p+minRecord <= total; {
		if bitsAt(pay, p, 1) != 1 {
			p++
			continue
		}
		slot := bitsAt(pay, p+1, 13)
		if slot < 0 || !bipedSlots[uint32(slot)] {
			p++
			continue
		}
		if needTag1 && bitsAt(pay, p+14, 2) != 1 {
			p++
			continue
		}
		if bitsAt(pay, p+16, 2) != 0 {
			p++
			continue
		}
		mc := bitsAt(pay, p+18, 3)
		if mc < 2 || mc > 7 {
			p++
			continue
		}
		i0 := p + 21 + 6*mc
		if i0+i0Bits > total {
			p++
			continue
		}
		if !ascending(pay, p+21, mc) {
			p++
			continue
		}
		if bitsAt(pay, i0, lay.GateBits) != 0 {
			p++
			continue
		}
		out = append(out, p)
		p = i0 + i0Bits
	}
	return out
}

func ascending(pay []byte, at, count int) bool {
	prev := -1
	for k := 0; k < count; k++ {
		idx := bitsAt(pay, at+6*k, 6)
		if idx < 0 || (k == 0 && idx != 0) || idx <= prev {
			return false
		}
		prev = idx
	}
	return true
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

var allSlots = func() map[uint32]bool {
	m := make(map[uint32]bool, 8192)
	for i := uint32(0); i < 8192; i++ {
		m[i] = true
	}
	return m
}()

func popcount(v uint64) int {
	n := 0
	for v != 0 {
		n += int(v & 1)
		v >>= 1
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
	lay, _, err := filmdec.DetectI0Layout(cache)
	if err != nil {
		panic(err)
	}
	fmt.Printf("decoupage i0 detecte : %s (total=%d bits)\n", lay.String(), lay.TotalBits())

	filmdec.SetRecordStateParam(2)
	cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}
	w := filmdec.NewWorld(reg)
	for _, b := range binds {
		w.BindFull(uint32((b.Gen<<30)|b.Slot), uint32(b.TI))
	}

	preHist := map[int]int{} // (Comps[0].StartBit - StartBit) - 6*popcount(Mask)
	firstIdxHist := map[int]int{}
	seqBiped := 0
	offlineTotal := 0
	inter := 0
	// coincidence des CURSEURS i0 : offline (p+21+6mc) vs walk sequentiel.
	i0Match := map[int]int{} // decalage offlineI0 - plus proche i0 sequentiel (|d|<=6)
	i0Off, i0Seq, i0Exact := 0, 0, 0
	// echantillons de bits d'en-tete
	samples := 0
	for c := lo; c <= hi; c++ {
		data := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, c))
		if data == nil {
			continue
		}
		for _, pay := range listChunkFrames(data, 0) {
			off := matcherOffline(pay, lay, true)
			offlineTotal += len(off)
			offSet := map[int]bool{}
			for _, p := range off {
				offSet[p] = true
			}
			res := filmdec.DebugDecodeFrame(pay, w, cfg, allSlots)
			// curseurs i0 vus par le walk sequentiel (TOUS archetypes : i0 est le
			// composant de position, present sur tout objet mobile).
			seqI0 := map[int]bool{}
			for _, d := range res.Recs {
				for _, cr := range d.Comps {
					if cr.Index == 0 {
						seqI0[cr.StartBit] = true
						i0Seq++
					}
				}
			}
			for _, p := range off {
				mc := bitsAt(pay, p+18, 3)
				q := p + 21 + 6*mc
				i0Off++
				if seqI0[q] {
					i0Exact++
					i0Match[0]++
					continue
				}
				best := 99
				for d := -8; d <= 8; d++ {
					if seqI0[q+d] && abs(d) < abs(best) {
						best = d
					}
				}
				if best != 99 {
					i0Match[best]++
				} else {
					i0Match[99]++
				}
			}
			for _, d := range res.Recs {
				if !d.IsBiped || d.DesyncAt != -1 || d.TypeIndex != filmdec.BipedTypeIndex || len(d.Comps) == 0 {
					continue
				}
				seqBiped++
				pre := d.Comps[0].StartBit - d.StartBit
				preHist[pre-6*popcount(d.Mask)]++
				firstIdxHist[d.Comps[0].Index]++
				if offSet[d.StartBit] {
					inter++
				}
				if samples < 12 {
					samples++
					fmt.Printf("\n[ech %d] chunk%02d startBit=%d slot=%d mask=%016x pc=%d pre=%d\n",
						samples, c, d.StartBit, d.Slot, d.Mask, popcount(d.Mask), pre)
					fmt.Printf("   bits[p..p+32) = %s\n", bitStr(pay, d.StartBit, 32))
					fmt.Printf("   lecture SEQ (idLow=11) : type=%d idLow=%d tag=%d gate=%d cnt=%d -> idx@p+18\n",
						bitsAt(pay, d.StartBit, 1), bitsAt(pay, d.StartBit+1, 11), bitsAt(pay, d.StartBit+12, 2),
						bitAt(pay, d.StartBit+14), bitsAt(pay, d.StartBit+15, 3))
					fmt.Printf("   lecture OFF (21 bits)  : slot13=%d tag=%d z2=%d mc=%d -> idx@p+21 ; accepte=%v\n",
						bitsAt(pay, d.StartBit+1, 13), bitsAt(pay, d.StartBit+14, 2), bitsAt(pay, d.StartBit+16, 2),
						bitsAt(pay, d.StartBit+18, 3), offSet[d.StartBit])
					for k, cr := range d.Comps {
						if k >= 4 {
							break
						}
						fmt.Printf("     comp[%d] i%02d %-40s @%d (delta=%d)\n", k, cr.Index, cr.Name, cr.StartBit, cr.StartBit-d.StartBit)
					}
				}
			}
		}
	}
	fmt.Printf("\n=== BILAN (chunks %d..%d) ===\n", lo, hi)
	fmt.Printf("records ti=35 clean vus par le WALK SEQUENTIEL : %d\n", seqBiped)
	fmt.Printf("positions acceptees par le MATCHEUR OFFLINE 21b : %d\n", offlineTotal)
	fmt.Printf("intersection des positions de depart            : %d\n", inter)
	fmt.Println("\n-- preambule consomme par le walk, hors index (pre - 6*popcount) --")
	printHist(preHist)
	fmt.Println("\n-- index du 1er composant consomme (walk) --")
	printHist(firstIdxHist)
	fmt.Printf("\n=== COINCIDENCE DES CURSEURS i0 ===\n")
	fmt.Printf("curseurs i0 vus par le walk sequentiel : %d\n", i0Seq)
	fmt.Printf("curseurs i0 predits par le matcheur 21b: %d ; coincidence EXACTE : %d (%.2f %%)\n",
		i0Off, i0Exact, 100*float64(i0Exact)/float64(max1(i0Off)))
	fmt.Println("-- decalage offlineI0 - i0 sequentiel le plus proche (99 = aucun a +-8) --")
	printHist(i0Match)
}

func printHist(h map[int]int) {
	keys := make([]int, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		fmt.Printf("   %6d : %d\n", k, h[k])
	}
}

func max1(x int) int {
	if x < 1 {
		return 1
	}
	return x
}
