// tmp_framedump — dump VERBATIM d'une frame + sonde après le stop du décodeur.
// Question : le paquet contient-il tous les joueurs (dev privé = certitude) et
// est-ce que je m'arrête tôt (faux recEnd / record qui over-read) ?
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_framedump [filmDir] [globalFrameIdx]
//
//	sans idx : balaie et imprime les frames où des biped-deltas propres restent
//	APRÈS mon stop (preuve d'arrêt précoce), + un résumé agrégé.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"levelup/go-api/internal/analysis/filmdec"
)

const defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}

// bipedSlots = tous les ti=35 du dump (joueurs+bots+remplaçants, ~99). Rempli par loadBindings.
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

type frameTS struct {
	ts  uint64
	pay []byte
}

var frameTSs []frameTS

func listFrames(d []byte) [][]byte {
	var out [][]byte
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, d[off+16:off+16+sz])
			frameTSs = append(frameTSs, frameTS{ts, d[off+16 : off+16+sz]})
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
			if ti == 35 {
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

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	filmdec.SetRecordStateParam(2)
	loadBindings(dir)
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	var frames [][]byte
	for idx := 2; idx <= 26; idx++ {
		frames = append(frames, listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx)))...)
	}
	fmt.Printf("total frames type-0 = %d\n", len(frames))

	// Mode KF2 : sweep du préambule du type-2 (keyframe) + décode NEW records.
	// Les type-2 sont PAR CHUNK (25 keyframes) et contiennent l'état complet.
	if len(os.Args) > 2 && os.Args[2] == "kf2" {
		filmdec.SetInferChain(true)
		raw := inflate(dir + "/chunk_02.bin")
		_, pay2 := func() (uint64, []byte) {
			off := 0
			for off+16 <= len(raw) {
				typ := binary.LittleEndian.Uint16(raw[off:])
				sz := int(binary.LittleEndian.Uint32(raw[off+4:]))
				ts := binary.LittleEndian.Uint64(raw[off+8:])
				if sz < 0 || off+16+sz > len(raw) {
					break
				}
				if typ == 2 {
					return ts, raw[off+16 : off+16+sz]
				}
				off += 16 + sz
			}
			return 0, nil
		}()
		fmt.Printf("type-2 payload = %d octets\n", len(pay2))
		for _, skip := range []int{0, 4, 8, 12, 16, 20, 32, 44, 48, 64, 96, 100} {
			w := freshWorld(reg)
			br := filmdec.NewBitReader(pay2)
			br.Skip(skip)
			recs, _ := filmdec.DecodeFrameRecords(br, w, calCfg)
			nNew, nBip, nClean := 0, 0, 0
			for _, r := range recs {
				if r.Type == 1 {
					nNew++
				}
				if bipedSlots[r.Slot] {
					nBip++
				}
				if r.DesyncAt == -1 {
					nClean++
				}
			}
			fmt.Printf("  skip=%-3d -> %d records (%d NEW, %d clean, %d biped)\n", skip, len(recs), nNew, nClean, nBip)
		}
		return
	}

	// Mode TYPES : histogramme des types de frames dans les chunks principaux
	// (les non-POV pourraient être dans un autre type que 0).
	if len(os.Args) > 2 && os.Args[2] == "types" {
		for idx := 2; idx <= 26; idx++ {
			raw := inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx))
			hist := map[uint16]int{}
			sizeByType := map[uint16]int{}
			off := 0
			for off+16 <= len(raw) {
				typ := binary.LittleEndian.Uint16(raw[off:])
				sz := int(binary.LittleEndian.Uint32(raw[off+4:]))
				if sz < 0 || off+16+sz > len(raw) {
					break
				}
				hist[typ]++
				sizeByType[typ] += sz
				off += 16 + sz
			}
			line := fmt.Sprintf("chunk_%02d (%d o) :", idx, len(raw))
			for t := uint16(0); t <= 14; t++ {
				if hist[t] > 0 {
					line += fmt.Sprintf(" type%d×%d(%do)", t, hist[t], sizeByType[t]/hist[t])
				}
			}
			fmt.Println(line)
			if idx >= 6 {
				break
			}
		}
		return
	}

	// Mode KF : décode le keyframe (chunk_02, type-2) = état complet initial.
	// But : binder TOUTES les entités + extraire l'archétype/position de chaque record.
	if len(os.Args) > 2 && os.Args[2] == "kf" {
		raw := inflate(dir + "/chunk_02.bin")
		fmt.Printf("chunk_02 inflaté = %d octets\n", len(raw))
		// liste TOUS les frames (tous types), pas seulement type-0.
		off := 0
		for off+16 <= len(raw) {
			typ := binary.LittleEndian.Uint16(raw[off:])
			sz := int(binary.LittleEndian.Uint32(raw[off+4:]))
			ts := binary.LittleEndian.Uint64(raw[off+8:])
			if sz < 0 || off+16+sz > len(raw) {
				fmt.Printf("  @%d: header invalide (typ=%d sz=%d) — arrêt\n", off, typ, sz)
				break
			}
			fmt.Printf("frame @%d type=%d size=%d ts=%d\n", off, typ, sz, ts)
			if typ == 2 || typ == 1 { // keyframe / full-state : préambule + sweep de configs
				pay := raw[off+16 : off+16+sz]
				n := 24
				if n > len(pay) {
					n = len(pay)
				}
				fmt.Printf("  préambule hex:")
				for i := 0; i < n; i++ {
					fmt.Printf(" %02x", pay[i])
				}
				fmt.Println()
				// Aucune de ces configs ne décode le keyframe (préambule non-record) :
				// le bootstrap keyframe demande un skip d'en-tête à trouver (Ghidra).
				for _, hef := range []bool{false, true} {
					for _, idlb := range []int{11, 12, 13} {
						cfg := filmdec.FrameConfig{HasExtraFields: hef, IDLowBits: idlb}
						w := freshWorld(reg)
						res := filmdec.DebugDecodeFrame(pay, w, cfg, bipedSlots)
						nNew := 0
						for _, r := range res.Recs {
							if r.Type == 1 {
								nNew++
							}
						}
						fmt.Printf("  cfg hef=%v idlb=%d -> %d records (%d NEW), stop=%s@%d\n",
							hef, idlb, len(res.Recs), nNew, res.StopReason, res.StopBit)
					}
				}
			}
			off += 16 + sz
		}
		return
	}

	// Mode MV : test décodage MULTI-VUE ([config bit][3 boucles]) découvert live.
	if len(os.Args) > 2 && os.Args[2] == "mv" {
		filmdec.SetInferChain(true)
		skip, views3 := 0, 3
		if v := os.Getenv("SKIP"); v != "" {
			fmt.Sscanf(v, "%d", &skip)
		}
		if v := os.Getenv("VIEWS"); v != "" {
			fmt.Sscanf(v, "%d", &views3)
		}
		nBiped := map[uint32]int{}
		framesReached3 := 0
		totalViews := 0
		totalBipedRecs := 0 // total records biped propres (densité)
		nF := 0
		for _, pay := range frames {
			nF++
			w := freshWorld(reg)
			recs, views := filmdec.DecodeFrameViews(pay, w, calCfg, views3, skip)
			totalViews += views
			if views >= 3 {
				framesReached3++
			}
			seen := map[uint32]bool{}
			for _, r := range recs {
				if bipedSlots[r.Slot] && r.DesyncAt == -1 {
					seen[r.Slot] = true
					totalBipedRecs++
				}
			}
			for s := range seen {
				nBiped[s]++
			}
		}
		fmt.Printf("MULTIVUE skip=%d vues=%d : frames=%d, vues moy=%.2f, frames atteignant %d vues=%d\n",
			skip, views3, nF, float64(totalViews)/float64(nF), views3, framesReached3)
		fmt.Printf("bipeds distincts avec position : %d ; TOTAL records biped propres : %d\n", len(nBiped), totalBipedRecs)
		// dump la 1re frame qui atteint les 3 vues (voir si vues 1/2 ont des bipeds)
		for fi, pay := range frames {
			w := freshWorld(reg)
			recs, views := filmdec.DecodeFrameViews(pay, w, calCfg, views3, skip)
			if views < views3 {
				continue
			}
			fmt.Printf("\nframe#%d (%d vues) : %d records\n", fi, views, len(recs))
			for i, r := range recs {
				if i < 30 {
					tag := ""
					if bipedSlots[r.Slot] {
						tag = " <BIPED>"
					}
					fmt.Printf("  #%d t=%d slot=%d ti=%d desync=%d%s\n", i, r.Type, r.Slot, r.TypeIndex, r.DesyncAt, tag)
				}
			}
			break
		}
		return
	}

	// Mode TS : timestamps d'une plage de frames (multi-paquet par tick ?).
	if len(os.Args) > 2 && os.Args[2] == "ts" {
		start := 5000
		if len(os.Args) > 3 {
			fmt.Sscanf(os.Args[3], "%d", &start)
		}
		for i := start; i < start+16 && i < len(frameTSs); i++ {
			w := freshWorld(reg)
			res := filmdec.DebugDecodeFrame(frameTSs[i].pay, w, calCfg, bipedSlots)
			var bs []uint32
			for _, r := range res.Recs {
				if bipedSlots[r.Slot] && r.DesyncAt == -1 {
					bs = append(bs, r.Slot)
				}
			}
			dt := int64(0)
			if i > start {
				dt = int64(frameTSs[i].ts) - int64(frameTSs[i-1].ts)
			}
			fmt.Printf("frame #%d ts=%d (dt=%+d us) bits=%d bipeds=%v\n", i, frameTSs[i].ts, dt, len(frameTSs[i].pay)*8, bs)
		}
		return
	}

	// Mode 1 : dump d'une frame précise.
	if len(os.Args) > 2 {
		var fi int
		fmt.Sscanf(os.Args[2], "%d", &fi)
		if fi < 0 || fi >= len(frames) {
			fmt.Println("index hors bornes")
			return
		}
		w := freshWorld(reg)
		res := filmdec.DebugDecodeFrame(frames[fi], w, calCfg, bipedSlots)
		fmt.Printf("=== frame #%d ===\n%s", fi, res.String())
		// Détail composant du 1er record biped (détection over-read).
		for _, d := range res.Recs {
			if d.IsBiped && len(d.Comps) > 0 {
				fmt.Printf("  détail biped slot%d mask=%#x (%d comps) :\n", d.Slot, d.Mask, len(d.Comps))
				for i, c := range d.Comps {
					w := 0
					if i+1 < len(d.Comps) {
						w = d.Comps[i+1].StartBit - c.StartBit
					} else {
						w = d.EndBit - c.StartBit
					}
					fmt.Printf("     i%-2d %-45s @%d w=%d ported=%v\n", c.Index, c.Name, c.StartBit, w, c.Ported)
				}
				break
			}
		}
		// Scan brut : où commencent réellement les deltas biped propres ?
		w2 := freshWorld(reg)
		scan := filmdec.ScanFrameBipeds(frames[fi], w2, calCfg, bipedSlots)
		fmt.Printf("  SCAN frame : %d deltas biped propres trouvés :\n", len(scan))
		for _, s := range scan {
			fmt.Printf("     @%-5d slot%d bits=%d..%d comps=%d\n", s.Bit, s.Slot, s.Bit, s.End, s.Comps)
		}
		return
	}

	// Mode 2 : agrégat sur toutes les frames — combien de biped-deltas propres restent
	// APRÈS mon stop (= arrêt précoce), et distribution.
	var sumReached, sumAfter, framesWithAfter int
	histAfter := map[int]int{}
	examples := 0
	for fi, pay := range frames {
		w := freshWorld(reg)
		res := filmdec.DebugDecodeFrame(pay, w, calCfg, bipedSlots)
		sumReached += res.BipedDeltas
		sumAfter += res.AfterBipeds
		histAfter[res.AfterBipeds]++
		if res.AfterBipeds > 0 {
			framesWithAfter++
			if examples < 3 {
				fmt.Printf("\n--- EXEMPLE frame #%d (biped-deltas APRÈS le stop = %d) ---\n%s", fi, res.AfterBipeds, res.String())
				examples++
			}
		}
	}
	fmt.Printf("\n=== AGRÉGAT ===\n")
	fmt.Printf("biped-deltas atteints par la boucle : total=%d (moy %.2f/frame)\n", sumReached, float64(sumReached)/float64(len(frames)))
	fmt.Printf("biped-deltas propres APRÈS le stop  : total=%d dans %d frames\n", sumAfter, framesWithAfter)
	fmt.Printf("histogramme (# biped-deltas propres après stop) :\n")
	for n := 0; n <= 8; n++ {
		if c := histAfter[n]; c > 0 {
			fmt.Printf("  %d après : %6d frames\n", n, c)
		}
	}
	fmt.Println("\nSi 'après le stop' est ÉLEVÉ = je m'arrête tôt et rate les joueurs (à corriger).")
	fmt.Println("Si ~0 = les autres joueurs ne sont PAS des deltas propres décodables là où je cherche.")
}
