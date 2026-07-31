// tmp_bipedfmt — répond à : les 8 bipeds sont-ils enregistrés dans le MÊME format,
// et le POV n'est-il « dense » que par PORTÉE (ordre) ou par FRÉQUENCE d'émission ?
//
// Sur chaque frame, on décode (chaîne + resync) et on distingue les records ATTEINTS
// AVANT tout desync. Par slot biped on compte, sur l'ensemble ET restreint aux frames
// CLEAN (décodées jusqu'à la fin, aucun desync) :
//   - appear   : # frames où le slot apparaît comme record décodé
//   - withPos  : # de celles portant un composant position i0
//   - abs/delta/raw : ventilation du type de position (raw = keep-baseline = non réémise)
//   - avgIdx   : rang moyen du record dans la frame (le POV est-il systématiquement 1er ?)
//
// Lecture : si en frames CLEAN les 8 ont un taux withPos/appear comparable → même
// format, sparseness = PORTÉE. Si le POV a withPos/appear bien plus haut → différence
// de FRÉQUENCE d'émission (même format, culling réseau). raw élevé = keep-baseline.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_bipedfmt [filmDir]
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

const defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}

// bipedSlots est construit depuis le world_dump (TOUTES les entités ti=35 : joueurs +
// bots + remplaçants), pas seulement 512-519.
var bipedSlots = map[uint32]bool{}

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

var cachedBindings [][2]uint32

func loadBindings(dir string) {
	raw, _ := os.ReadFile(dir + "/world_dump_full.txt")
	for _, tok := range bytes.Fields(raw) {
		s := string(tok)
		if len(s) == 0 || s[0] == '#' {
			continue
		}
		var slot, ti uint32
		if _, e := fmt.Sscanf(s, "%d:%d", &slot, &ti); e == nil {
			cachedBindings = append(cachedBindings, [2]uint32{slot, ti})
			if ti == 35 { // biped : joueur, bot ou remplaçant
				bipedSlots[slot&0x3fffffff] = true
			}
		}
	}
}

func freshWorld(reg *filmdec.Registry) *filmdec.World {
	w := filmdec.NewWorld(reg)
	for _, b := range cachedBindings {
		w.BindFull(b[0], b[1])
	}
	return w
}

type stat struct {
	appear, withPos       int
	abs, delta, raw       int
	appearClean, posClean int // restreint aux frames décodées clean jusqu'au bout
	idxSum, idxN          int
}

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	filmdec.SetRecordStateParam(2)
	filmdec.SetInferChain(true)
	loadBindings(dir)
	if os.Getenv("RESYNC") != "" {
		filmdec.SetInferResyncTargets(bipedSlots) // resync vers N'IMPORTE quel biped (99)
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	// Type de position par record : capture le PosKind au StartBit du composant i0.
	posByBit := map[int]filmdec.PosKind{}
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) { posByBit[s.BitPos] = s.Kind })

	stats := map[uint32]*stat{}
	for s := range bipedSlots {
		stats[s] = &stat{}
	}
	nClean, nTotal := 0, 0
	bipedsPerFrame := map[int]int{} // histogramme : # bipeds distincts décodés dans une frame
	recCleanSum, recDesyncSum, nDesyncFr := 0, 0, 0
	frameBitsSum, consumedSum := 0, 0
	ratioSum := 0.0
	for idx := 2; idx <= 26; idx++ {
		for _, pay := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx))) {
			nTotal++
			w := freshWorld(reg)
			for k := range posByBit {
				delete(posByBit, k)
			}
			recs, _ := filmdec.DecodeFrameInfer(pay, w, calCfg)
			clean := len(recs) == 0 || recs[len(recs)-1].DesyncAt == -1
			if clean {
				nClean++
				recCleanSum += len(recs)
			} else {
				recDesyncSum += len(recs)
				nDesyncFr++
			}
			seen := map[uint32]bool{}
			lastEnd := 0
			for _, r := range recs {
				if r.Trace.EndBit > lastEnd {
					lastEnd = r.Trace.EndBit
				}
				if bipedSlots[r.Slot] && r.DesyncAt == -1 {
					seen[r.Slot] = true
				}
			}
			bipedsPerFrame[len(seen)]++
			if clean {
				fl := len(pay) * 8
				if lastEnd > fl {
					lastEnd = fl // clamp (EndBit spurieux)
				}
				frameBitsSum += fl
				consumedSum += lastEnd
				if fl > 0 {
					ratioSum += float64(lastEnd) / float64(fl)
				}
			}
			for ri, r := range recs {
				if !bipedSlots[r.Slot] || r.DesyncAt != -1 {
					continue
				}
				st := stats[r.Slot]
				st.appear++
				st.idxSum += ri
				st.idxN++
				if clean {
					st.appearClean++
				}
				hasPos := false
				for _, c := range r.Trace.Comps {
					if c.Name != "object-position-dynamic-precision-component" {
						continue
					}
					hasPos = true
					switch posByBit[c.StartBit] {
					case filmdec.PosKindAbsolute, filmdec.PosKindAbsFallback:
						st.abs++
					case filmdec.PosKindDelta8, filmdec.PosKindDeltaAxis:
						st.delta++
					case filmdec.PosKindRaw:
						st.raw++
					}
					break
				}
				if hasPos {
					st.withPos++
					if clean {
						st.posClean++
					}
				}
			}
		}
	}
	filmdec.SetPositionCaptureHook(nil)

	fmt.Printf("frames total=%d clean=%d (%.1f%%)\n", nTotal, nClean, 100*float64(nClean)/float64(nTotal))
	fmt.Printf("%-5s %8s %8s %7s %7s %7s %10s %9s %7s\n",
		"slot", "appear", "withPos", "abs", "delta", "raw", "appClean", "posClean", "avgIdx")
	var slots []uint32
	for s := range stats {
		slots = append(slots, s)
	}
	// tri par withPos décroissant : les joueurs/bots ACTIFS ressortent en tête.
	sort.Slice(slots, func(i, j int) bool { return stats[slots[i]].withPos > stats[slots[j]].withPos })
	fmt.Printf("(%d slots biped au total ; top 30 par withPos)\n", len(slots))
	nActive := 0
	for i, s := range slots {
		st := stats[s]
		if st.withPos > 0 {
			nActive++
		}
		if i >= 30 {
			continue
		}
		avgIdx := 0.0
		if st.idxN > 0 {
			avgIdx = float64(st.idxSum) / float64(st.idxN)
		}
		fmt.Printf("%-5d %8d %8d %7d %7d %7d %10d %9d %7.1f\n",
			s, st.appear, st.withPos, st.abs, st.delta, st.raw, st.appearClean, st.posClean, avgIdx)
	}
	fmt.Printf("bipeds avec >=1 position décodée : %d / %d\n", nActive, len(slots))
	fmt.Printf("\nbipeds distincts décodés par frame (histogramme) :\n")
	for n := 0; n <= 8; n++ {
		if c := bipedsPerFrame[n]; c > 0 {
			fmt.Printf("  %d biped(s) : %6d frames\n", n, c)
		}
	}
	avgClean, avgDesync := 0.0, 0.0
	if nClean > 0 {
		avgClean = float64(recCleanSum) / float64(nClean)
	}
	if nDesyncFr > 0 {
		avgDesync = float64(recDesyncSum) / float64(nDesyncFr)
	}
	fmt.Printf("records/frame : clean=%.1f desync-prefix=%.1f (desync coupe tôt = préfixe court)\n", avgClean, avgDesync)
	if nClean > 0 {
		fmt.Printf("frames CLEAN : consommé moy=%.0f bits / paquet moy=%.0f bits ; ratio moy par frame=%.0f%%\n",
			float64(consumedSum)/float64(nClean), float64(frameBitsSum)/float64(nClean),
			100*ratioSum/float64(nClean))
		fmt.Println("  ratio <<100%% = recEnd précoce → paquet non lu jusqu'au bout (bug/portée, distants cachés)")
		fmt.Println("  ratio ~100%% = paquet lu en entier → contenu génuinement petit (FRÉQUENCE réseau)")
	}
	fmt.Println("\nLecture : si AUCUNE frame n'a >2 bipeds décodés → soit les distants sont")
	fmt.Println("génuinement rares (FRÉQUENCE réseau), soit tous cachés après desync (PORTÉE).")
	fmt.Println("raw = keep-baseline (position non réémise, prédite par le client = dead-reckoning).")
}
