// tmp_rawpos — TEST hypothèse OpenSpartan : les vec3 BRUTS (96 bits = 3 float32) lus
// par le chemin keep-baseline de la position (PosKindRaw) sont-ils DIRECTEMENT les
// coordonnées monde du joueur ? Capture PosKindRaw par slot biped (via resync),
// imprime plages/échantillons, et rend les trajectoires. Si sain -> positions denses
// directes, sans dequant/dead-reckoning.
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

func finite(v [3]float32) bool {
	for _, x := range v {
		if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
			return false
		}
	}
	return true
}

func bits32At(d []byte, b int) uint32 {
	bi := b >> 3
	off := uint(b & 7)
	var v uint64
	for i := 0; i < 5; i++ {
		var by uint64
		if bi+i < len(d) {
			by = uint64(d[bi+i])
		}
		v = v<<8 | by
	}
	return uint32((v >> (8 - off)) & 0xffffffff)
}

func f32(u uint32) float32 { return math.Float32frombits(u) }

func bswap(u uint32) uint32 {
	return (u&0xff)<<24 | (u&0xff00)<<8 | (u&0xff0000)>>8 | (u&0xff000000)>>24
}

// leMode : lecture little-endian (byteswap) validee par le match oracle (45 valeurs, 100% LE).
var leMode = os.Getenv("BE") == ""

func rd(d []byte, b int) float32 {
	u := bits32At(d, b)
	if leMode {
		u = bswap(u)
	}
	return f32(u)
}

// fixedOffset : si FIXOFF=<n> est défini, lit la position à un offset FIXE relatif à i0
// (au lieu de scanner la plage). Test de structure : un offset constant = position propre.
func fixedOffset() (int, bool) {
	v := os.Getenv("FIXOFF")
	if v == "" {
		return 0, false
	}
	var n int
	fmt.Sscanf(v, "%d", &n)
	return n, true
}

func posLike(x, y, z float32) bool {
	for _, v := range []float32{x, y, z} {
		a := math.Abs(float64(v))
		if math.IsNaN(a) || math.IsInf(a, 0) || (a != 0 && a < 1e-10) {
			return false
		}
	}
	// Repere ORACLE (petit) : x[-6..36] y[-25..27] z[-4..7], marge legere. Sans |x|>30.
	return x >= -8 && x <= 38 && y >= -27 && y <= 29 && z >= -5 && z <= 8
}

func dist2D(a, b [3]float32) float64 {
	dx, dy := float64(a[0]-b[0]), float64(a[1]-b[1])
	return math.Sqrt(dx*dx + dy*dy)
}

// scanRecord cherche dans [startBit,endBit] de pay le triplet float32 coord-like le plus
// proche de `last` (continuité) ; sans ancre, le plus précoce. Retourne (pos, trouvé).
func scanRecord(pay []byte, startBit, endBit int, last [3]float32, hasLast bool) ([3]float32, bool) {
	nb := len(pay) * 8
	if startBit < 0 {
		startBit = 0
	}
	if endBit > nb-96 {
		endBit = nb - 96
	}
	var best [3]float32
	bestScore, found := math.MaxFloat64, false
	for b := startBit; b <= endBit; b++ {
		x, y, z := rd(pay, b), rd(pay, b+32), rd(pay, b+64)
		if !posLike(x, y, z) {
			continue
		}
		v := [3]float32{x, y, z}
		score := float64(b - startBit)
		if hasLast {
			score = dist2D(last, v)
		}
		if score < bestScore {
			bestScore, best, found = score, v, true
		}
	}
	return best, found
}

func main() {
	dir, out := defFilm, "trajectoires_raw.png"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if len(os.Args) > 2 {
		out = os.Args[2]
	}
	filmdec.SetRecordStateParam(2)
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	// Attribution ECS + valeur scannée : le resync donne les records biped (SLOT correct) ;
	// on scanne la PLAGE du record pour le float32 brut de position (offset binaire raté
	// par le décodage quantifié), choisi par continuité par-slot.
	type pt struct {
		t int
		v [3]float32
	}
	bySlot := map[uint32][]pt{}
	lastPos := map[uint32][3]float32{}
	hasLast := map[uint32]bool{}
	histo := map[int]int{} // offset-dans-record des triplets coord-like (slot519), mode HISTO
	doHisto := os.Getenv("HISTO") != ""
	for idx := 2; idx <= 26; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx))) {
			w := freshWorld(dir, reg)
			recs := filmdec.DecodeFrameResync(fr.pay, w, calCfg, bipedSlots, nil)
			tms := int((fr.ts - t0Us) / 1000)
			for _, r := range recs {
				if !bipedSlots[r.Slot] || len(r.Trace.Comps) == 0 {
					continue
				}
				c0 := r.Trace.Comps[0].StartBit
				start := c0 - 200
				end := r.Trace.EndBit + 100
				if doHisto && r.Slot == 519 && r.DesyncAt == -1 {
					nb := len(fr.pay) * 8
					for b := start; b <= end && b+96 <= nb; b++ {
						if b < 0 {
							continue
						}
						x, y, z := rd(fr.pay, b), rd(fr.pay, b+32), rd(fr.pay, b+64)
						if posLike(x, y, z) {
							histo[b-c0]++
						}
					}
				}
				if fixOff, okf := fixedOffset(); okf {
					b := c0 + fixOff
					nb := len(fr.pay) * 8
					if b >= 0 && b+96 <= nb {
						x, y, z := rd(fr.pay, b), rd(fr.pay, b+32), rd(fr.pay, b+64)
						if posLike(x, y, z) {
							bySlot[r.Slot] = append(bySlot[r.Slot], pt{tms, [3]float32{x, y, z}})
						}
					}
				} else if v, ok := scanRecord(fr.pay, start, end, lastPos[r.Slot], hasLast[r.Slot]); ok {
					bySlot[r.Slot] = append(bySlot[r.Slot], pt{tms, v})
					lastPos[r.Slot], hasLast[r.Slot] = v, true
				}
			}
		}
	}
	if doHisto {
		type kv struct{ off, n int }
		var arr []kv
		for o, n := range histo {
			arr = append(arr, kv{o, n})
		}
		sort.Slice(arr, func(i, j int) bool { return arr[i].n > arr[j].n })
		fmt.Println("HISTO offset-dans-record (slot519) — top pics (offset relatif à i0 : n) :")
		for i := 0; i < 20 && i < len(arr); i++ {
			fmt.Printf("  offset %+d : %d\n", arr[i].off, arr[i].n)
		}
		return
	}

	type track struct {
		slot uint32
		pts  []pt
	}
	var tracks []track
	for s := uint32(512); s <= 519; s++ {
		p := bySlot[s]
		sort.Slice(p, func(i, j int) bool { return p[i].t < p[j].t })
		if len(p) == 0 {
			fmt.Printf("  slot%d : 0 raw\n", s)
			continue
		}
		var mn, mx [3]float32 = p[0].v, p[0].v
		for _, e := range p {
			for a := 0; a < 3; a++ {
				if e.v[a] < mn[a] {
					mn[a] = e.v[a]
				}
				if e.v[a] > mx[a] {
					mx[a] = e.v[a]
				}
			}
		}
		fmt.Printf("  slot%d : %d raw | X[%.1f,%.1f] Y[%.1f,%.1f] Z[%.1f,%.1f] | ex: (%.2f,%.2f,%.2f)\n",
			s, len(p), mn[0], mx[0], mn[1], mx[1], mn[2], mx[2], p[0].v[0], p[0].v[1], p[0].v[2])
		if len(p) >= 15 {
			tracks = append(tracks, track{s, p})
		}
	}
	if len(tracks) == 0 {
		fmt.Println("aucune trajectoire raw.")
		return
	}

	// Stats de lissage (pas 2D entre points consecutifs par slot) vs oracle (0.043/0.042/2.83).
	var steps []float64
	teleports := 0
	for _, t := range tracks {
		for j := 1; j < len(t.pts); j++ {
			d := dist2D(t.pts[j-1].v, t.pts[j].v)
			steps = append(steps, d)
			if d > 5 {
				teleports++
			}
		}
	}
	if len(steps) > 0 {
		sort.Float64s(steps)
		sum := 0.0
		for _, s := range steps {
			sum += s
		}
		fmt.Printf("LISSAGE pas 2D : mean=%.3f p50=%.3f max=%.3f | teleports(>5)=%d / %d pas | (oracle: 0.043/0.042/2.83)\n",
			sum/float64(len(steps)), steps[len(steps)/2], steps[len(steps)-1], teleports, len(steps))
	}

	var mn, mx [3]float32
	first := true
	for _, t := range tracks {
		for _, e := range t.pts {
			if first {
				mn, mx, first = e.v, e.v, false
			}
			for a := 0; a < 3; a++ {
				if e.v[a] < mn[a] {
					mn[a] = e.v[a]
				}
				if e.v[a] > mx[a] {
					mx[a] = e.v[a]
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
		return pad + int((float64(v[ax])-float64(mn[ax]))/sx*span), S - pad - int((float64(v[ay])-float64(mn[ay]))/sy*span)
	}
	for i, t := range tracks {
		col := okabeIto[i%len(okabeIto)]
		var lx, ly int
		for j, e := range t.pts {
			x, y := px(e.v)
			if j > 0 {
				dx, dy := x-lx, y-ly
				n := int(math.Max(math.Abs(float64(dx)), math.Abs(float64(dy)))) + 1
				for k := 0; k <= n; k++ {
					xx, yy := lx+dx*k/n, ly+dy*k/n
					if xx >= 0 && xx < S && yy >= 0 && yy < S {
						img.Set(xx, yy, col)
					}
				}
			}
			lx, ly = x, y
		}
	}
	f, _ := os.Create(out)
	defer f.Close()
	png.Encode(f, img)
	fmt.Printf("→ %s (%d slots)\n", out, len(tracks))
}
