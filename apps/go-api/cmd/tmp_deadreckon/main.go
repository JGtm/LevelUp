// tmp_deadreckon — trajectoires DENSES des 8 bipèdes par DEAD-RECKONING : on ancre
// sur les positions absolues (repère map réel) et on INTÈGRE la vélocité décodée
// (direction cubemap × magnitude log/exp) entre les échantillons, comme le jeu prédit
// les joueurs distants (keep-baseline). Décodage via DecodeFrameResync (mur desync
// cassé) + capture position (posHook) et vélocité (dynPrecHook) par slot.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_deadreckon [filmDir] [out.png]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}
var bipedSlots = map[uint32]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}

var okabeIto = []color.RGBA{
	{0xE6, 0x9F, 0x00, 255}, {0x56, 0xB4, 0xE9, 255}, {0x00, 0x9E, 0x73, 255},
	{0xF0, 0xE4, 0x42, 255}, {0x00, 0x72, 0xB2, 255}, {0xD5, 0x5E, 0x00, 255},
	{0xCC, 0x79, 0xA7, 255}, {0xBB, 0xBB, 0xBB, 255},
}

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

func freshWorld(dir string, reg *filmdec.Registry) *filmdec.World {
	raw, _ := os.ReadFile(dir + "/world_dump_full.txt")
	w := filmdec.NewWorld(reg)
	for _, tok := range bytes.Fields(raw) {
		var slot, ti uint32
		if _, e := fmt.Sscanf(string(tok), "%d:%d", &slot, &ti); e == nil {
			w.BindFull(slot, ti)
		}
	}
	return w
}

// event = un échantillon d'un slot à un instant : position absolue (hasAbs) et/ou vélocité (hasVel).
type event struct {
	t      float64 // secondes
	abs    [3]float32
	hasAbs bool
	vel    [3]float32
	hasVel bool
}

func dist(a, b [3]float32) float64 {
	dx, dy, dz := float64(a[0]-b[0]), float64(a[1]-b[1]), float64(a[2]-b[2])
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// deadReckon intègre : ancre au 1er absolu, puis pos += vel*dt entre événements, recale
// sur chaque absolu proche (rejette un absolu qui téléporte = faux-positif). Retourne la
// trajectoire dense.
func deadReckon(evs []event) [][3]float32 {
	sort.Slice(evs, func(i, j int) bool { return evs[i].t < evs[j].t })
	start := -1
	for i, e := range evs {
		if e.hasAbs {
			start = i
			break
		}
	}
	if start < 0 {
		return nil
	}
	cur := evs[start].abs
	var curVel [3]float32
	haveVel := false
	lastT := evs[start].t
	out := [][3]float32{cur}
	const maxSpeed = 350.0 // borne magnitude vélocité (unités/s)
	const maxHold = 0.3    // s : ne pas extrapoler une vélocité STALE au-delà de 0.3s (sinon spikes)
	for i := start; i < len(evs); i++ {
		e := evs[i]
		dt := e.t - lastT
		if dt > 0 && haveVel && dt <= maxHold {
			cur[0] += curVel[0] * float32(dt)
			cur[1] += curVel[1] * float32(dt)
			cur[2] += curVel[2] * float32(dt)
		}
		if e.hasAbs {
			// recale seulement si l'absolu est cohérent avec la prédiction (sinon faux-positif)
			if dt <= 0.001 || dist(cur, e.abs)/dt <= maxSpeed*1.5 {
				cur = e.abs
			}
		}
		if e.hasVel {
			curVel, haveVel = e.vel, true
		}
		out = append(out, cur)
		lastT = e.t
	}
	return out
}

func main() {
	dir, out := defFilm, "trajectoires_dr.png"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if len(os.Args) > 2 {
		out = os.Args[2]
	}
	filmdec.SetRecordStateParam(2)
	useChain := os.Getenv("CHAIN") != ""
	if useChain {
		filmdec.SetInferChain(true)
		filmdec.SetInferResyncTargets(bipedSlots) // reach tail bipeds, structurally confirmed
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	posByBit := map[int]filmdec.PositionSample{}
	velByBit := map[int][3]float32{}
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) { posByBit[s.BitPos] = s })
	filmdec.SetDynPrecHook(func(present bool, packedDir, scale uint64, bitpos int) {
		if present {
			velByBit[bitpos] = filmdec.DecodeVelocity(packedDir, scale)
		}
	})

	// Rejet des faux-positifs du resync par continuité de position. Seuil GÉNÉREUX
	// (525 u/s = 1.5× la magnitude max) pour laisser passer les canons humains / chutes /
	// véhicules (lancements rapides légitimes), tout en coupant les téléports garbage.
	lastPos := map[uint32][3]float32{}
	lastTime := map[uint32]float64{}
	curT := 0.0
	const acceptMaxSpeed = 525.0
	accept := func(slot uint32, pos [3]float32, hasPos bool) bool {
		if !hasPos {
			return true
		}
		if lt, ok := lastTime[slot]; ok {
			dt := curT - lt
			if dt < 0.001 {
				dt = 0.001
			}
			if dist(lastPos[slot], pos)/dt > acceptMaxSpeed {
				return false
			}
		}
		return true
	}

	bySlot := map[uint32][]event{}
	for idx := 2; idx <= 26; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx))) {
			w := freshWorld(dir, reg)
			for k := range posByBit {
				delete(posByBit, k)
			}
			for k := range velByBit {
				delete(velByBit, k)
			}
			curT = float64(fr.ts-t0Us) / 1e6
			var recs []filmdec.FrameRecord
			if useChain {
				recs, _ = filmdec.DecodeFrameInfer(fr.pay, w, calCfg)
			} else {
				recs = filmdec.DecodeFrameResync(fr.pay, w, calCfg, bipedSlots, accept)
			}
			t := curT // secondes
			for _, r := range recs {
				if !bipedSlots[r.Slot] {
					continue
				}
				var ev event
				ev.t = t
				for _, c := range r.Trace.Comps {
					switch c.Name {
					case "object-position-dynamic-precision-component":
						if s, ok := posByBit[c.StartBit]; ok && (s.Kind == filmdec.PosKindAbsolute || s.Kind == filmdec.PosKindAbsFallback) {
							ev.abs, ev.hasAbs = s.Vec, true
						}
					case "object-translational-velocity-dynamic-precision-component":
						// la vélocité présente est capturée à StartBit ou StartBit+1 (gate R(1))
						if v, ok := velByBit[c.StartBit+1]; ok {
							ev.vel, ev.hasVel = v, true
						} else if v, ok := velByBit[c.StartBit]; ok {
							ev.vel, ev.hasVel = v, true
						}
					}
				}
				if ev.hasAbs || ev.hasVel {
					bySlot[r.Slot] = append(bySlot[r.Slot], ev)
				}
				if ev.hasAbs {
					lastPos[r.Slot], lastTime[r.Slot] = ev.abs, t // ancre continuité pour l'accept
				}
			}
		}
	}
	filmdec.SetPositionCaptureHook(nil)
	filmdec.SetDynPrecHook(nil)

	type track struct {
		slot uint32
		pts  [][3]float32
	}
	var tracks []track
	for s := uint32(512); s <= 519; s++ {
		evs := bySlot[s]
		na, nv := 0, 0
		for _, e := range evs {
			if e.hasAbs {
				na++
			}
			if e.hasVel {
				nv++
			}
		}
		pts := deadReckon(evs)
		fmt.Printf("  slot%d : %d events (abs=%d vel=%d) → traj %d pts\n", s, len(evs), na, nv, len(pts))
		if len(pts) >= 15 {
			tracks = append(tracks, track{s, pts})
		}
	}
	fmt.Printf("film=%s : %d trajectoires dead-reckonées\n", dir, len(tracks))
	if len(tracks) == 0 {
		return
	}

	// bornes communes + rendu (2 axes de plus grand span).
	var mn, mx [3]float32
	first := true
	for _, t := range tracks {
		for _, v := range t.pts {
			if first {
				mn, mx, first = v, v, false
			}
			for a := 0; a < 3; a++ {
				if v[a] < mn[a] {
					mn[a] = v[a]
				}
				if v[a] > mx[a] {
					mx[a] = v[a]
				}
			}
		}
	}
	sp := []struct {
		a int
		s float32
	}{{0, mx[0] - mn[0]}, {1, mx[1] - mn[1]}, {2, mx[2] - mn[2]}}
	sort.Slice(sp, func(i, j int) bool { return sp[i].s > sp[j].s })
	ax, ay := sp[0].a, sp[1].a

	const S, pad = 820, 40
	span := float64(S - 2*pad)
	img := image.NewRGBA(image.Rect(0, 0, S, S))
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			img.Set(x, y, color.RGBA{14, 16, 22, 255})
		}
	}
	sx, sy := float64(mx[ax]-mn[ax]), float64(mx[ay]-mn[ay])
	if sx == 0 {
		sx = 1
	}
	if sy == 0 {
		sy = 1
	}
	px := func(v [3]float32) (int, int) {
		u := (float64(v[ax]) - float64(mn[ax])) / sx
		w := (float64(v[ay]) - float64(mn[ay])) / sy
		return pad + int(u*span), S - pad - int(w*span)
	}
	line := func(x0, y0, x1, y1 int, col color.RGBA) {
		dx, dy := x1-x0, y1-y0
		n := int(math.Max(math.Abs(float64(dx)), math.Abs(float64(dy)))) + 1
		for i := 0; i <= n; i++ {
			x, y := x0+dx*i/n, y0+dy*i/n
			for oy := -1; oy <= 1; oy++ {
				for ox := -1; ox <= 1; ox++ {
					if x+ox >= 0 && x+ox < S && y+oy >= 0 && y+oy < S {
						img.Set(x+ox, y+oy, col)
					}
				}
			}
		}
	}
	for i, t := range tracks {
		col := okabeIto[i%len(okabeIto)]
		var lx, ly int
		for j, v := range t.pts {
			x, y := px(v)
			if j > 0 {
				line(lx, ly, x, y, col)
			}
			lx, ly = x, y
		}
	}
	f, _ := os.Create(out)
	defer f.Close()
	png.Encode(f, img)
	fmt.Printf("→ %s\n", out)
}
