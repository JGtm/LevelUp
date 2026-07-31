// tmp_l0arb — ARBITRAGE mesure : balayage de la largeur du champ id (IDLowBits) du record
// DELTA, qui EST le desaccord "20 vs 21 bits" (preambule = 1 + idLow + 2 + 1 + 3).
// THROWAWAY. Meme moteur que cmd/tmp_l0witness (decodage SEQUENTIEL DecodeFrameRecords,
// World seme par les bindings offline du keyframe), avec en plus :
//   - le taux de trames terminees sur un marqueur de FIN DE RECORDS propre (le seul
//     critere objectif global : une trame mal cadree ne retombe pas sur son terminateur) ;
//   - le detail clean/desync PAR ARCHETYPE (ti=35 bipede, 5 joueur, 11 objectif,
//     37 equipement, 42 arme au sol, 41).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_l0arb [chunkLo chunkHi] [idLowList]
// ex    : CGO_ENABLED=0 go run ./cmd/tmp_l0arb 3 26 11,12,13,14,15
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

const worldDump = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/.ai/re_dump/ce_world_dump.txt`

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

// archetypes suivis (mission) : bipede, joueur, objectif, equipement, arme au sol, 41.
var watched = []uint32{35, 5, 11, 37, 42, 41}

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

type stat struct{ clean, desync int }

func main() {
	lo, hi := 3, 26
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[1], "%d", &lo)
		fmt.Sscanf(os.Args[2], "%d", &hi)
	}
	idLows := []int{11, 12, 13, 14, 15}
	if len(os.Args) >= 4 {
		idLows = nil
		for _, s := range strings.Split(os.Args[3], ",") {
			if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
				idLows = append(idLows, v)
			}
		}
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
	// SEED=dump : bindings slot->archetype issus de la capture LIVE (.ai/V7.5/dumps/
	// ce_world_dump.txt, 1754 slots reels). Indispensable pour balayer idLow sans
	// confondre "mauvais cadrage" et "slot absent du World" : les slots du keyframe ne
	// vivent pas dans le meme espace d'identifiants que ceux des trames.
	dump := map[uint32]uint32{}
	if b, e := os.ReadFile(worldDump); e == nil {
		for _, ln := range strings.Split(string(b), "\n") {
			ln = strings.TrimSpace(ln)
			if ln == "" || strings.HasPrefix(ln, "#") {
				continue
			}
			p := strings.SplitN(ln, ":", 2)
			if len(p) != 2 {
				continue
			}
			s, e1 := strconv.Atoi(p[0])
			t, e2 := strconv.Atoi(p[1])
			if e1 == nil && e2 == nil {
				dump[uint32(s)] = uint32(t)
			}
		}
	}
	seed := os.Getenv("SEED")
	fmt.Printf("bindings keyframe=%d ; bindings dump live=%d ; SEED=%q\n", len(binds), len(dump), seed)

	fmt.Printf("=== ARBITRAGE idLow (chunks %d..%d, film 000d5950) ===\n", lo, hi)
	fmt.Printf("preambule du record DELTA = 1 (prefixe type) + idLow + 2 (tag) + 1 (gate) + 3 (count)\n")
	fmt.Printf("  idLow=11 -> 18 bits | idLow=13 -> 20 bits (lecture du walk) | idLow=14 -> 21 bits (lecture offline)\n\n")

	leads := []int{0}
	if v := os.Getenv("LEADS"); v != "" {
		leads = nil
		for _, s := range strings.Split(v, ",") {
			if n, e := strconv.Atoi(strings.TrimSpace(s)); e == nil {
				leads = append(leads, n)
			}
		}
	}
	for _, lead := range leads {
		for _, idLow := range idLows {
			runOne(reg, dump, binds, seed, lo, hi, lead, idLow)
		}
	}
}

func runOne(reg *filmdec.Registry, dump map[uint32]uint32, binds []filmdec.KeyframeRec, seed string, lo, hi, lead, idLow int) {
	{
		cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: idLow}
		w := filmdec.NewWorld(reg)
		switch seed {
		case "dump":
			for s, t := range dump {
				w.BindFull(s, t)
			}
		case "both":
			for _, b := range binds {
				w.BindFull(uint32((b.Gen<<30)|b.Slot), uint32(b.TI))
			}
			for s, t := range dump {
				w.BindFull(s, t)
			}
		default:
			for _, b := range binds {
				w.BindFull(uint32((b.Gen<<30)|b.Slot), uint32(b.TI))
			}
		}
		per := map[uint32]*stat{}
		frames, framesClean := 0, 0
		totClean, totDesync := 0, 0
		recsTotal := 0
		for c := lo; c <= hi; c++ {
			data := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, c))
			if data == nil {
				continue
			}
			for _, pay := range listChunkFrames(data, 0) {
				frames++
				br := filmdec.NewBitReader(pay)
				br.Skip(lead)
				recs, err := filmdec.DecodeFrameRecords(br, w, cfg)
				if err == nil {
					framesClean++
				}
				recsTotal += len(recs)
				for _, r := range recs {
					if per[r.TypeIndex] == nil {
						per[r.TypeIndex] = &stat{}
					}
					if r.DesyncAt < 0 {
						per[r.TypeIndex].clean++
						totClean++
					} else {
						per[r.TypeIndex].desync++
						totDesync++
					}
				}
			}
		}
		fmt.Printf("--- idLow=%d  (preambule %d bits) ---\n", idLow, 7+idLow)
		fmt.Printf("  trames terminees sur un marqueur de fin PROPRE : %d / %d  (%.2f %%)\n",
			framesClean, frames, 100*float64(framesClean)/float64(max1(frames)))
		fmt.Printf("  records decodes : %d  (clean=%d desync=%d, %.2f %% clean)\n",
			recsTotal, totClean, totDesync, 100*float64(totClean)/float64(max1(totClean+totDesync)))
		fmt.Printf("  %-8s %10s %10s %8s\n", "ti", "clean", "desync", "%clean")
		for _, ti := range watched {
			s := per[ti]
			if s == nil {
				s = &stat{}
			}
			fmt.Printf("  ti=%-5d %10d %10d %7.2f%%\n", ti, s.clean, s.desync,
				100*float64(s.clean)/float64(max1(s.clean+s.desync)))
		}
		// top archetypes hors liste, pour ne rien cacher
		type kv struct {
			ti uint32
			s  *stat
		}
		var a []kv
		for ti, s := range per {
			a = append(a, kv{ti, s})
		}
		sort.Slice(a, func(i, j int) bool { return a[i].s.clean+a[i].s.desync > a[j].s.clean+a[j].s.desync })
		fmt.Printf("  (top archetypes tous confondus :")
		for i := 0; i < len(a) && i < 6; i++ {
			fmt.Printf(" ti%d=%d/%d", a[i].ti, a[i].s.clean, a[i].s.clean+a[i].s.desync)
		}
		fmt.Printf(")\n\n")
	}
}

func max1(x int) int {
	if x < 1 {
		return 1
	}
	return x
}
