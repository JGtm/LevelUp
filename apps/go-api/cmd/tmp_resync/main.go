// tmp_resync — casse le desync séquentiel SANS porter les default-states : World
// PERSISTANT + calibration de la largeur default-state par record NEW (brute-force d,
// validé par lookahead 1-niveau). Les entités se lient au fil de l'eau → les DELTA
// décodent → on atteint tous les bipèdes. Capture i0 par slot.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_resync [filmDir] [chunkMax]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)
const idLowBits = 11
const maxD = 420 // borne sup de la largeur default-state (biped ~380)

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

// rdHeader lit le header de record à partir du bit pos : (type, slot, bodyStart).
// type: 0=end,1=new,2=del,3=delta. -1 si hors borne.
func rdHeader(pay []byte, pos int) (typ int, slot uint32, body int) {
	if pos+3 > len(pay)*8 {
		return -1, 0, pos
	}
	br := filmdec.NewBitReader(pay)
	br.Skip(pos)
	if br.ReadBit() {
		typ = 3
	} else {
		typ = int(br.ReadBits(2))
	}
	if typ == 0 {
		return 0, 0, br.BitPos()
	}
	low := uint32(br.ReadBits(idLowBits))
	br.ReadBits(2) // tag
	slot = low & 0x3fffffff
	return typ, slot, br.BitPos()
}

type sample struct {
	t    int
	kind filmdec.PosKind
	vec  [3]float32
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
	w := filmdec.NewWorld(reg)
	for _, b := range bindings {
		w.BindFull(b[0], b[1])
	}

	var captured []filmdec.PositionSample
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) { captured = append(captured, s) })

	posBySlot := map[uint32][]sample{}
	frames, fullFrames := 0, 0
	recsDecoded, recsDesync := 0, 0
	newCalibrated, newFailed := 0, 0

	for idx := 2; idx <= chunkMax; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx))) {
			frames++
			tms := int((fr.ts - t0Us) / 1000)
			pos := 0
			clean := true
		recordLoop:
			for {
				typ, slot, body := rdHeader(fr.pay, pos)
				switch typ {
				case -1, 0: // hors borne / END
					break recordLoop
				case 2: // DEL : R(32) inconditionnel
					pos = body + 32
					w.Unbind(slot)
				case 1: // NEW : calibrer d
					d, end, ti, ok := calibrateNew(fr.pay, body, reg, w)
					if !ok {
						newFailed++
						clean = false
						break recordLoop
					}
					newCalibrated++
					w.BindFull(slot, ti) // slot = id (tag ignoré ; suffisant ici)
					captureIfBiped(fr.pay, body, slot, tms, posBySlot, &captured, func() (filmdec.EntityTrace, int) {
						b := filmdec.NewBitReader(fr.pay)
						b.Skip(body)
						t := filmdec.TraverseEntity(b, reg, d)
						return t, b.BitPos()
					})
					pos = end
					recsDecoded++
				case 3: // DELTA
					captured = captured[:0]
					t, end := filmdec.DecodeDeltaRecordAt(fr.pay, body, w, slot)
					if t.DesyncAt != -1 {
						recsDesync++
						clean = false
						break recordLoop
					}
					recsDecoded++
					if bipedSlots[slot] {
						recordBipedPos(t, slot, tms, captured, posBySlot)
					}
					pos = end
				}
			}
			if clean {
				fullFrames++
			}
		}
	}
	filmdec.SetPositionCaptureHook(nil)

	fmt.Printf("film=%s frames=%d\n", dir, frames)
	fmt.Printf("  frames décodées ENTIÈREMENT : %d (%.1f%%)\n", fullFrames, 100*float64(fullFrames)/float64(frames))
	fmt.Printf("  records OK=%d desync=%d | NEW calibrés=%d échoués=%d\n", recsDecoded, recsDesync, newCalibrated, newFailed)
	fmt.Println("  trajectoire par slot biped (baseline absolu + accumulation deltas) :")
	for s := uint32(512); s <= 519; s++ {
		raw := posBySlot[s]
		sort.Slice(raw, func(i, j int) bool { return raw[i].t < raw[j].t })
		nAbs, nD := 0, 0
		traj := buildTraj(raw, &nAbs, &nD)
		var steps []float64
		for i := 1; i < len(traj); i++ {
			steps = append(steps, dist(traj[i-1].vec, traj[i].vec))
		}
		sort.Float64s(steps)
		med := 0.0
		if len(steps) > 0 {
			med = steps[len(steps)/2]
		}
		fmt.Printf("    slot%d : %d samples (abs=%d delta=%d) → traj %d pts, pas médian=%.2f\n",
			s, len(raw), nAbs, nD, len(traj), med)
	}
}

// buildTraj : 1er absolu = spawn, puis accumulation des deltas (les Raw/keep-baseline
// = position inchangée → on répète la dernière).
func buildTraj(raw []sample, nAbs, nD *int) []sample {
	var cur [3]float32
	have := false
	var out []sample
	for _, p := range raw {
		switch p.kind {
		case filmdec.PosKindAbsolute, filmdec.PosKindAbsFallback:
			cur = p.vec
			have = true
			*nAbs++
			out = append(out, sample{p.t, p.kind, cur})
		case filmdec.PosKindDelta8, filmdec.PosKindDeltaAxis:
			if !have {
				continue
			}
			cur[0] += p.vec[0]
			cur[1] += p.vec[1]
			cur[2] += p.vec[2]
			*nD++
			out = append(out, sample{p.t, p.kind, cur})
		}
	}
	return out
}

// calibrateNew brute-force la largeur default-state d d'un record NEW à bodyStart.
// Accepte le 1er d où TraverseEntity est clean (DesyncAt==-1) ET le record suivant
// décode aussi clean (lookahead 1-niveau). Retourne (d, endBit, typeIndex, ok).
func calibrateNew(pay []byte, body int, reg *filmdec.Registry, w *filmdec.World) (int, int, uint32, bool) {
	for d := 0; d <= maxD; d++ {
		b := filmdec.NewBitReader(pay)
		b.Skip(body)
		t := filmdec.TraverseEntity(b, reg, d)
		if t.DesyncAt != -1 {
			continue
		}
		end := b.BitPos()
		if end <= body || end > len(pay)*8 {
			continue
		}
		if lookaheadOK(pay, end, reg, w) {
			return d, end, t.TypeIndex, true
		}
	}
	return 0, 0, 0, false
}

// lookaheadOK : le record à `pos` décode-t-il clean (ou END/borne) ? Lève l'ambiguïté
// sur d. Ne mute pas le World (clone des deltas via décodage en lecture seule).
func lookaheadOK(pay []byte, pos int, reg *filmdec.Registry, w *filmdec.World) bool {
	typ, slot, body := rdHeader(pay, pos)
	switch typ {
	case -1, 0:
		return true // fin de frame plausible
	case 2:
		return body+32 <= len(pay)*8 // DEL : place pour R(32)
	case 3:
		if _, ok := w.ArchetypeForSlot(slot); !ok {
			return false // delta sur slot non-lié = mauvais d (très probablement)
		}
		t, _ := filmdec.DecodeDeltaRecordAt(pay, body, w, slot)
		return t.DesyncAt == -1
	case 1:
		// NEW suivant : on vérifie juste qu'un d existe (clean) sans récursion profonde.
		for d := 0; d <= maxD; d++ {
			b := filmdec.NewBitReader(pay)
			b.Skip(body)
			if filmdec.TraverseEntity(b, reg, d).DesyncAt == -1 {
				return true
			}
		}
		return false
	}
	return false
}

func captureIfBiped(pay []byte, body int, slot uint32, tms int, posBySlot map[uint32][]sample, captured *[]filmdec.PositionSample, decode func() (filmdec.EntityTrace, int)) {
	if !bipedSlots[slot] {
		return
	}
	*captured = (*captured)[:0]
	t, _ := decode()
	recordBipedPos(t, slot, tms, *captured, posBySlot)
}

func recordBipedPos(t filmdec.EntityTrace, slot uint32, tms int, captured []filmdec.PositionSample, posBySlot map[uint32][]sample) {
	for _, c := range t.Comps {
		if c.Name != "object-position-dynamic-precision-component" {
			continue
		}
		for _, s := range captured {
			if s.BitPos == c.StartBit {
				posBySlot[slot] = append(posBySlot[slot], sample{tms, s.Kind, s.Vec})
			}
		}
		break
	}
}

func dist(a, b [3]float32) float64 {
	dx, dy, dz := float64(a[0]-b[0]), float64(a[1]-b[1]), float64(a[2]-b[2])
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}
