// tmp_widthsweep — THROWAWAY. Utilise l'ORACLE offline (frames tight-clean = consomment
// ~tout leur payload) comme CHERCHEUR DE RECETTE : balaie la largeur d'axe du composant
// position i0 (TraversalPrecision.AxisW) et retient celle qui MAXIMISE les frames
// bit-exactes. Aucune calibration : la vraie largeur est celle qui rend le décodeur
// cohérent (recadrage user : la recette existe, on la trouve par cohérence, pas par devinette).
//
// World FRAIS par frame (seed = world_dump_full.txt) pour isoler l'effet de la largeur du
// binding accumulé. On rapporte aussi les bipèdes avec position réelle par largeur.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_widthsweep
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

func loadBindings() [][2]uint32 {
	var b [][2]uint32
	for _, name := range []string{"/world_dump_full.txt", "/world_dump.txt"} {
		raw, _ := os.ReadFile(cache + name)
		if len(raw) == 0 {
			continue
		}
		for _, tok := range bytes.Fields(raw) {
			s := string(tok)
			if len(s) == 0 || s[0] == '#' {
				continue
			}
			var slot, ti uint32
			if _, e := fmt.Sscanf(s, "%d:%d", &slot, &ti); e == nil {
				b = append(b, [2]uint32{slot, ti})
			}
		}
		if len(b) > 0 {
			fmt.Printf("seed = %s (%d slots)\n", name, len(b))
			return b
		}
	}
	return b
}

func main() {
	filmdec.SetRecordStateParam(2)
	reg, _ := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	binds := loadBindings()

	// pré-charge les payloads une fois.
	var frames [][]byte
	for idx := 1; idx <= 26; idx++ {
		frames = append(frames, listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx)))...)
	}
	fmt.Printf("frames=%d ; balayage largeur d'axe position i0 :\n\n", len(frames))

	fresh := func() *filmdec.World {
		w := filmdec.NewWorld(reg)
		for _, b := range binds {
			w.BindFull(b[0], b[1])
		}
		return w
	}

	fmt.Printf("  axisW | tight-clean | err==nil | bipèdes pos-réelle\n")
	fmt.Printf("  ------+-------------+----------+-------------------\n")
	for w := uint(4); w <= 20; w++ {
		filmdec.TraversalPrecision = filmdec.PrecisionDescriptor{IndexW: 1, AxisW: [3]uint{w, w, w}}
		var realSlots map[uint32]bool = map[uint32]bool{}
		var frameSamples []filmdec.PositionSample
		filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) { frameSamples = append(frameSamples, s) })
		tight, cleanErr := 0, 0
		for _, pay := range frames {
			payloadBits := len(pay) * 8
			br := filmdec.NewBitReader(pay)
			frameSamples = frameSamples[:0]
			recs, derr := filmdec.DecodeFrameRecords(br, fresh(), calCfg)
			if derr != nil {
				continue
			}
			cleanErr++
			residue := payloadBits - br.BitPos()
			if residue < 0 || residue >= 32 {
				continue
			}
			tight++
			byBit := map[int]filmdec.PositionSample{}
			for _, s := range frameSamples {
				byBit[s.BitPos] = s
			}
			for _, r := range recs {
				for _, c := range r.Trace.Comps {
					if c.Name != "object-position-dynamic-precision-component" {
						continue
					}
					if s, ok := byBit[c.StartBit]; ok && s.Kind != filmdec.PosKindNone && s.Kind != filmdec.PosKindRaw {
						realSlots[r.Slot] = true
					}
					break
				}
			}
		}
		filmdec.SetPositionCaptureHook(nil)
		fmt.Printf("  %-5d | %-11d | %-8d | %d\n", w, tight, cleanErr, len(realSlots))
	}
}
