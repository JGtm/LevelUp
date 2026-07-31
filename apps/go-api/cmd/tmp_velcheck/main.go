// tmp_velcheck — DIAGNOSTIC : la vélocité i1 (object-translational-velocity) des
// joueurs non-POV varie-t-elle dans le temps (ils bougent, dead-reckoning) ou est-elle
// ~nulle (stationnaires) ? Capture les valeurs brutes (dir packée 19b + scale 10b) par
// slot via le hook dynPrec, attribuées au composant i1 du record.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_velcheck [filmDir] [chunkMax]
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
const t0Us = uint64(4537898226)

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}
var bipedSlots = map[uint32]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}

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

type frame struct {
	ts  uint64
	pay []byte
}

func listFrames(d []byte) []frame {
	var out []frame
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, frame{ts, d[off+16 : off+16+sz]})
		}
		off += 16 + sz
	}
	return out
}

var bindings [][2]uint32

func loadBindings(dir string) {
	raw, _ := os.ReadFile(dir + "/world_dump_full.txt")
	for _, tok := range bytes.Fields(raw) {
		s := string(tok)
		if len(s) == 0 || s[0] == '#' {
			continue
		}
		var slot, ti uint32
		if _, e := fmt.Sscanf(s, "%d:%d", &slot, &ti); e == nil {
			bindings = append(bindings, [2]uint32{slot, ti})
		}
	}
}

func freshWorld(reg *filmdec.Registry) *filmdec.World {
	w := filmdec.NewWorld(reg)
	for _, b := range bindings {
		w.BindFull(b[0], b[1])
	}
	return w
}

type vel struct {
	t       int
	present bool
	dir     uint64
	scale   uint64
}

func main() {
	dir, chunkMax := defFilm, 26
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if len(os.Args) > 2 {
		fmt.Sscanf(os.Args[2], "%d", &chunkMax)
	}
	filmdec.SetRecordStateParam(2)
	loadBindings(dir)
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	type emit struct {
		present bool
		dir, sc uint64
	}
	byBit := map[int]emit{}
	filmdec.SetDynPrecHook(func(present bool, packedDir, scale uint64, bitpos int) {
		byBit[bitpos] = emit{present, packedDir, scale}
	})

	velBySlot := map[uint32][]vel{}
	for idx := 2; idx <= chunkMax; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx))) {
			w := freshWorld(reg)
			for k := range byBit {
				delete(byBit, k)
			}
			var recs []filmdec.FrameRecord
			if os.Getenv("RESYNC") != "" {
				recs = filmdec.DecodeFrameResync(fr.pay, w, calCfg, bipedSlots, nil)
			} else {
				br := filmdec.NewBitReader(fr.pay)
				recs, _ = filmdec.DecodeFrameRecords(br, w, calCfg)
			}
			tms := int((fr.ts - t0Us) / 1000)
			for _, r := range recs {
				if !bipedSlots[r.Slot] {
					continue
				}
				for _, c := range r.Trace.Comps {
					if c.Name != "object-translational-velocity-dynamic-precision-component" {
						continue
					}
					e, ok := byBit[c.StartBit+1]
					if !ok {
						e, ok = byBit[c.StartBit]
					}
					if ok {
						velBySlot[r.Slot] = append(velBySlot[r.Slot], vel{tms, e.present, e.dir, e.sc})
					}
					break
				}
			}
		}
	}
	filmdec.SetDynPrecHook(nil)

	fmt.Println("vélocité i1 par slot biped (present = vélocité émise) + validation cubemap (gridSize 19) :")
	for s := uint32(512); s <= 519; s++ {
		vs := velBySlot[s]
		sort.Slice(vs, func(i, j int) bool { return vs[i].t < vs[j].t })
		present, distinct := 0, map[uint64]bool{}
		var prevDir [3]float32
		havePrev := false
		var cosSum float64
		var cosN int
		for _, v := range vs {
			if !v.present {
				continue
			}
			present++
			distinct[v.dir<<10|v.scale] = true
			d := filmdec.DecodeDynPrecDir(v.dir, 19) // cubemap unit direction
			if havePrev {
				cos := float64(d[0]*prevDir[0] + d[1]*prevDir[1] + d[2]*prevDir[2])
				cosSum += cos
				cosN++
			}
			prevDir, havePrev = d, true
		}
		meanCos := 0.0
		if cosN > 0 {
			meanCos = cosSum / float64(cosN)
		}
		fmt.Printf("  slot%d : %d éch i1, %d présents, %d distinctes | cosinus moyen vél-consécutives = %.3f (proche 1 = lisse = cubemap OK)\n",
			s, len(vs), present, len(distinct), meanCos)
		// montrer les 5 premières vélocités décodées
		shown := 0
		for _, v := range vs {
			if v.present && shown < 5 {
				d := filmdec.DecodeDynPrecDir(v.dir, 19)
				fmt.Printf("      @%6.1fs dir=%-7d scale=%-4d → unit=(%.3f,%.3f,%.3f)\n", float64(v.t)/1000, v.dir, v.scale, d[0], d[1], d[2])
				shown++
			}
		}
	}
}
