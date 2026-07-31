// tmp_tightclean — THROWAWAY. Oracle de cohérence OFFLINE (pas de CE) : une frame est
// "tight-clean" si son décodage atteint recEnd (err==nil) EN CONSOMMANT ~tout le payload
// (reliquat < seuil). Un recEnd atteint loin de la fin du payload = faux-propre (motif 000
// fortuit). Ce filet distingue les frames RÉELLEMENT bit-exactes des faux-propres, sans
// oracle runtime, et mesure le vrai plafond : combien de bipèdes distincts ont une position
// réelle (abs/delta, pas keep-baseline) dans les frames tight-clean.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_tightclean
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

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}

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

func main() {
	filmdec.SetRecordStateParam(2)
	reg, _ := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))

	// world_dump seed (250 slots).
	wd, _ := os.ReadFile(cache + "/world_dump.txt")
	seed := func(w *filmdec.World) {
		for _, tok := range bytes.Fields(wd) {
			s := string(tok)
			if len(s) == 0 || s[0] == '#' {
				continue
			}
			var slot, ti uint32
			if _, e := fmt.Sscanf(s, "%d:%d", &slot, &ti); e == nil {
				w.BindFull(slot, ti)
			}
		}
	}

	// capture des positions bipèdes par frame, avec le kind (keep-baseline vs réel).
	var frameSamples []filmdec.PositionSample
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) { frameSamples = append(frameSamples, s) })
	defer filmdec.SetPositionCaptureHook(nil)

	// bandes de reliquat (bits non consommés à recEnd) pour les frames err==nil.
	bands := []struct {
		lo, hi int
		n      int
	}{{0, 8, 0}, {8, 32, 0}, {32, 128, 0}, {128, 512, 0}, {512, 1 << 30, 0}}
	nFrames, nCleanErr := 0, 0
	realPosSlots := map[uint32]int{} // slot -> #frames tight-clean avec position réelle
	tightCleanFrames := 0

	w := filmdec.NewWorld(reg)
	seed(w)
	for idx := 1; idx <= 26; idx++ {
		for _, pay := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			nFrames++
			payloadBits := len(pay) * 8
			br := filmdec.NewBitReader(pay)
			frameSamples = frameSamples[:0]
			recs, derr := filmdec.DecodeFrameRecords(br, w, calCfg)
			if derr != nil {
				continue
			}
			nCleanErr++
			residue := payloadBits - br.BitPos()
			for i := range bands {
				if residue >= bands[i].lo && residue < bands[i].hi {
					bands[i].n++
					break
				}
			}
			// tight-clean = reliquat dans [0,32) : consomme ~tout le payload SANS déborder.
			// Un reliquat négatif (lecture PAST la fin) = faux-propre silencieux, exclu.
			if residue < 0 || residue >= 32 {
				continue
			}
			tightCleanFrames++
			// positions réelles (abs/absfb/d8/dax) attribuées à un slot biped via les records.
			byBit := map[int]filmdec.PositionSample{}
			for _, s := range frameSamples {
				byBit[s.BitPos] = s
			}
			for _, r := range recs {
				ti, ok := w.ArchetypeForSlot(r.Slot)
				if !ok || ti != 35 {
					continue
				}
				for _, c := range r.Trace.Comps {
					if c.Name != "object-position-dynamic-precision-component" {
						continue
					}
					if s, ok := byBit[c.StartBit]; ok && s.Kind != filmdec.PosKindNone && s.Kind != filmdec.PosKindRaw {
						realPosSlots[r.Slot]++
					}
					break
				}
			}
		}
	}

	filmdec.SetPositionCaptureHook(nil)
	fmt.Printf("=== Oracle de cohérence OFFLINE (tight-clean) ===\n")
	fmt.Printf("frames=%d ; err==nil (recEnd atteint)=%d\n", nFrames, nCleanErr)
	fmt.Printf("répartition du reliquat (bits non consommés) sur les frames err==nil :\n")
	labels := []string{"[0,8)", "[8,32)", "[32,128)", "[128,512)", "[512,+)"}
	for i, b := range bands {
		fmt.Printf("  reliquat %-10s : %d frames\n", labels[i], b.n)
	}
	fmt.Printf("\nfor TIGHT-CLEAN (reliquat<32) = %d frames :\n", tightCleanFrames)
	fmt.Printf("bipèdes (arch#35) avec position RÉELLE (non keep-baseline) :\n")
	var slots []uint32
	for s := range realPosSlots {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	for _, s := range slots {
		fmt.Printf("  slot%d : %d frames tight-clean avec position réelle\n", s, realPosSlots[s])
	}
	fmt.Printf("=> %d bipèdes distincts avec position réelle en frames tight-clean\n", len(slots))
}
