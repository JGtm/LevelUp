// tmp_bipedframepos — extraction OFFLINE des positions DENSES des bipèdes GAMEPLAY
// (frame-space, slots 528+), reproduction de l'oracle ce_pos_oracle.csv.
//
// CORRECTION vs tmp_deadreckon : cible les BONS slots. En gameplay (frame type-0) les
// bipeds vivent aux slots 528-610 (ti=35), PAS 512-519 (keyframe/world-space). Prouvé par
// ce_capture_delta.csv (ti=35 min slot=528) + ce_pos_oracle.csv (positions 528-562).
//
// Méthode : World persistant seedé depuis world_dump_full.txt (100 slots ti=35, clés
// eid&0x3fffffff), accumulateur de position installé (deltas s'accumulent sur les seeds
// absolus), scan id-anchor parallèle (ScanFrameTargets) sur chaque frame des chunks 3..26.
// Calibration i0 : range CEBiped + largeur d'axe (ABSW) + DeltaQuantum, la boîte oracle
// x[-6..36] y[-25..27] z[-4..7] est le juge.
//
// Anti-faux-positif : n'accepte un sample que si in-box ET continu (saut <= seuil) avec le
// dernier sample du même slot.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_bipedframepos [filmDir] [out.png]
// Env : IDLOW (def 11), ABSW (def 13, 0=pd.AxisW), METHOD (scan|resync, def scan),
//
//	QUANT (def 0.0138), NOBOX=1 (désactive filtre boîte pour diag).
package main

import (
	"bufio"
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
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
	scratch = `c:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration/d6f109c3-2822-4e19-8fd4-cf65c50fec3d/scratchpad`
	oracleP = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/.ai/re_dump/ce_pos_oracle.csv`
)

// boîte oracle (marge 5%).
var boxMin = [3]float32{-6.33, -25.14, -4.20}
var boxMax = [3]float32{35.70, 27.50, 7.08}

func envI(k string, d int) int {
	if v := os.Getenv(k); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			return n
		}
	}
	return d
}
func envF(k string, d float64) float64 {
	if v := os.Getenv(k); v != "" {
		if n, e := strconv.ParseFloat(v, 64); e == nil {
			return n
		}
	}
	return d
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

// seedWorld : bind toutes les entités du full dump (clés eid&0x3fffffff via BindFull).
func seedWorld(dir string, reg *filmdec.Registry) (*filmdec.World, map[uint32]bool) {
	raw, _ := os.ReadFile(dir + "/world_dump_full.txt")
	w := filmdec.NewWorld(reg)
	ti35 := map[uint32]bool{}
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		for _, tok := range strings.Fields(line) {
			kv := strings.SplitN(tok, ":", 2)
			if len(kv) != 2 {
				continue
			}
			id, e1 := strconv.ParseUint(kv[0], 10, 64)
			ti, e2 := strconv.Atoi(kv[1])
			if e1 != nil || e2 != nil {
				continue
			}
			w.BindFull(uint32(id), uint32(ti))
			slot := uint32(id) & 0x3fffffff
			// frame-space bipeds = ti35 ET slot >= 528 (528+ prouvé par delta capture).
			if ti == 35 && slot >= 528 {
				ti35[slot] = true
			}
		}
	}
	return w, ti35
}

func loadOracle() map[uint32][][3]float32 {
	f, _ := os.Open(oracleP)
	if f == nil {
		return nil
	}
	defer f.Close()
	m := map[uint32][][3]float32{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "eid") {
			continue
		}
		p := strings.Split(line, ",")
		if len(p) < 6 {
			continue
		}
		slot, _ := strconv.Atoi(p[1])
		x, _ := strconv.ParseFloat(p[3], 32)
		y, _ := strconv.ParseFloat(p[4], 32)
		z, _ := strconv.ParseFloat(p[5], 32)
		m[uint32(slot)] = append(m[uint32(slot)], [3]float32{float32(x), float32(y), float32(z)})
	}
	return m
}

func inBox(v [3]float32) bool {
	for i := 0; i < 3; i++ {
		sp := boxMax[i] - boxMin[i]
		if v[i] < boxMin[i]-0.05*sp || v[i] > boxMax[i]+0.05*sp {
			return false
		}
	}
	return true
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

func dist(a, b [3]float32) float64 {
	dx, dy, dz := float64(a[0]-b[0]), float64(a[1]-b[1]), float64(a[2]-b[2])
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

var okabeIto = []color.RGBA{
	{0xE6, 0x9F, 0x00, 255}, {0x56, 0xB4, 0xE9, 255}, {0x00, 0x9E, 0x73, 255},
	{0xF0, 0xE4, 0x42, 255}, {0x00, 0x72, 0xB2, 255}, {0xD5, 0x5E, 0x00, 255},
	{0xCC, 0x79, 0xA7, 255}, {0xBB, 0xBB, 0xBB, 255},
}

func main() {
	dir, out := defFilm, scratch+"/traj_frame_offline.png"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if len(os.Args) > 2 {
		out = os.Args[2]
	}
	idlow := envI("IDLOW", 11)
	absw := envI("ABSW", 13)
	quant := float32(envF("QUANT", 0.0138))
	method := os.Getenv("METHOD")
	if method == "" {
		method = "scan"
	}
	noBox := os.Getenv("NOBOX") != ""

	filmdec.SetRecordStateParam(2)
	filmdec.WorldPositionRange = filmdec.QuantRangeCEBiped
	filmdec.SetAbsoluteAxisW(uint(absw))
	filmdec.SetDeltaQuantum(quant)

	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	world, targets := seedWorld(dir, reg)
	cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: idlow}
	fmt.Printf("cfg IDLow=%d ABSW=%d QUANT=%.5f METHOD=%s targets(ti35>=528)=%d\n", idlow, absw, quant, method, len(targets))

	// accumulateur persistant : les deltas s'accumulent sur les seeds absolus par slot.
	filmdec.SetPositionAccumulator(world)

	// capture : chaque sample émis (résolu en absolu par l'accumulateur).
	type sample struct {
		slot uint32
		v    [3]float32
		kind filmdec.PosKind
	}
	var pending []sample
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) {
		pending = append(pending, sample{s.Slot, s.Vec, s.Kind})
	})

	bySlot := map[uint32][][3]float32{}
	byKind := map[uint32]map[filmdec.PosKind][][3]float32{} // slot -> kind -> pts (pour overlap par kind)
	last := map[uint32][3]float32{}
	kindCount := map[filmdec.PosKind]int{}
	var accepted, rejBox, rejJump int
	const maxJump = 6.0 // u entre 2 samples consécutifs d'un slot (continuité)

	for idx := 3; idx <= 26; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx))) {
			pending = pending[:0]
			switch method {
			case "resync":
				filmdec.DecodeFrameResync(fr.pay, world, cfg, targets, nil)
			case "seq":
				br := filmdec.NewBitReader(fr.pay)
				filmdec.DecodeFrameRecords(br, world, cfg)
			case "chainwalk":
				filmdec.ScanFrameTargets(fr.pay, world, cfg, targets, filmdec.HarvestChainWalk)
			default:
				filmdec.ScanFrameTargets(fr.pay, world, cfg, targets, filmdec.HarvestNextBound)
			}
			for _, s := range pending {
				if !targets[s.slot] {
					continue
				}
				kindCount[s.kind]++
				if !noBox && !inBox(s.v) {
					rejBox++
					continue
				}
				if lp, ok := last[s.slot]; ok && dist(lp, s.v) > maxJump {
					rejJump++
					continue
				}
				bySlot[s.slot] = append(bySlot[s.slot], s.v)
				if byKind[s.slot] == nil {
					byKind[s.slot] = map[filmdec.PosKind][][3]float32{}
				}
				byKind[s.slot][s.kind] = append(byKind[s.slot][s.kind], s.v)
				last[s.slot] = s.v
				accepted++
			}
		}
	}
	filmdec.SetPositionCaptureHook(nil)
	filmdec.SetPositionAccumulator(nil)

	// stats
	oracle := loadOracle()
	fmt.Printf("\naccepted=%d rejBox=%d rejJump=%d ; kinds=%v\n", accepted, rejBox, rejJump, kindCount)
	slots := make([]uint32, 0, len(bySlot))
	for s := range bySlot {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	// boîte oracle par slot (pour métrique overlap : fraction des points offline DANS la
	// boîte oracle du MÊME slot -> discrimine track réel vs faux positif in-global-box).
	oraBox := map[uint32][2][3]float32{}
	for s, ps := range oracle {
		var mn, mx [3]float32
		for i, v := range ps {
			if i == 0 {
				mn, mx = v, v
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
		oraBox[s] = [2][3]float32{mn, mx}
	}
	inOraBox := func(s uint32, v [3]float32) bool {
		b, ok := oraBox[s]
		if !ok {
			return false
		}
		for a := 0; a < 3; a++ {
			sp := b[1][a] - b[0][a]
			if sp < 1 {
				sp = 1
			}
			if v[a] < b[0][a]-0.1*sp || v[a] > b[1][a]+0.1*sp {
				return false
			}
		}
		return true
	}

	var gmin, gmax [3]float32
	first := true
	totOff, totOra, totOverlap, totWithOra := 0, 0, 0, 0
	fmt.Printf("\n%-6s %8s %8s %8s %s\n", "slot", "off", "oracle", "inOraBx", "off box (x,y,z)")
	for _, s := range slots {
		pts := bySlot[s]
		totOff += len(pts)
		ov := 0
		for _, v := range pts {
			if inOraBox(s, v) {
				ov++
			}
		}
		if _, has := oracle[s]; has {
			totOverlap += ov
			totWithOra += len(pts)
		}
		var mn, mx [3]float32
		f2 := true
		for _, v := range pts {
			if f2 {
				mn, mx, f2 = v, v, false
			}
			for a := 0; a < 3; a++ {
				if v[a] < mn[a] {
					mn[a] = v[a]
				}
				if v[a] > mx[a] {
					mx[a] = v[a]
				}
				if first {
					gmin, gmax, first = v, v, false
				}
				if v[a] < gmin[a] {
					gmin[a] = v[a]
				}
				if v[a] > gmax[a] {
					gmax[a] = v[a]
				}
			}
		}
		no := len(oracle[s])
		totOra += no
		if len(pts) >= 5 {
			fmt.Printf("%-6d %8d %8d %7d%% x[%.1f,%.1f] y[%.1f,%.1f] z[%.1f,%.1f]\n",
				s, len(pts), no, ov*100/max1(len(pts)), mn[0], mx[0], mn[1], mx[1], mn[2], mx[2])
		}
	}
	overPct := 0
	if totWithOra > 0 {
		overPct = totOverlap * 100 / totWithOra
	}
	fmt.Printf("\nOVERLAP par-slot (points offline DANS boite oracle du MEME slot) = %d/%d = %d%%\n", totOverlap, totWithOra, overPct)
	// overlap ventilé par kind (discrimine le signal delta vrai des faux positifs abs)
	kindOv := map[filmdec.PosKind][2]int{}
	for s, km := range byKind {
		if _, has := oracle[s]; !has {
			continue
		}
		for k, ps := range km {
			c := kindOv[k]
			for _, v := range ps {
				c[1]++
				if inOraBox(s, v) {
					c[0]++
				}
			}
			kindOv[k] = c
		}
	}
	fmt.Printf("OVERLAP par kind : ")
	for _, k := range []filmdec.PosKind{filmdec.PosKindAbsolute, filmdec.PosKindAbsFallback, filmdec.PosKindDelta8, filmdec.PosKindDeltaAxis, filmdec.PosKindRaw} {
		c := kindOv[k]
		p := 0
		if c[1] > 0 {
			p = c[0] * 100 / c[1]
		}
		fmt.Printf("%s=%d/%d(%d%%) ", k, c[0], c[1], p)
	}
	fmt.Println()
	fmt.Printf("TOTAL offline=%d (sur %d slots) ; oracle=%d (sur %d slots)\n", totOff, len(slots), totOra, len(oracle))
	fmt.Printf("box offline: x[%.1f,%.1f] y[%.1f,%.1f] z[%.1f,%.1f]\n", gmin[0], gmax[0], gmin[1], gmax[1], gmin[2], gmax[2])
	fmt.Printf("box oracle : x[%.1f,%.1f] y[%.1f,%.1f] z[%.1f,%.1f]\n", boxMin[0], boxMax[0], boxMin[1], boxMax[1], boxMin[2], boxMax[2])

	writeTraj(out, bySlot, oracle)
	writeTxt(scratch+"/correctslots_pos.txt", slots, bySlot, oracle, idlow, absw, quant, method)
}

func writeTxt(p string, slots []uint32, bySlot, oracle map[uint32][][3]float32, idlow, absw int, quant float32, method string) {
	var b strings.Builder
	fmt.Fprintf(&b, "# tmp_bipedframepos — positions bipeds GAMEPLAY frame-space (slots 528+)\n")
	fmt.Fprintf(&b, "# calib IDLow=%d ABSW=%d QUANT=%.5f METHOD=%s range=CEBiped\n", idlow, absw, quant, method)
	fmt.Fprintf(&b, "# slot offlineRecords oracleRecords\n")
	for _, s := range slots {
		fmt.Fprintf(&b, "%d %d %d\n", s, len(bySlot[s]), len(oracle[s]))
	}
	os.WriteFile(p, []byte(b.String()), 0644)
}

func writeTraj(out string, bySlot, oracle map[uint32][][3]float32) {
	const S, pad = 900, 40
	span := float64(S/2 - 2*pad)
	img := image.NewRGBA(image.Rect(0, 0, S, S/2))
	for y := 0; y < S/2; y++ {
		for x := 0; x < S; x++ {
			img.Set(x, y, color.RGBA{14, 16, 22, 255})
		}
	}
	// bornes communes = boîte oracle (axes X horizontal, Y vertical).
	mnx, mxx, mny, mxy := float64(boxMin[0]), float64(boxMax[0]), float64(boxMin[1]), float64(boxMax[1])
	sx, sy := mxx-mnx, mxy-mny
	pt := func(v [3]float32, xoff int) (int, int) {
		u := (float64(v[0]) - mnx) / sx
		w := (float64(v[1]) - mny) / sy
		return xoff + pad + int(u*span), S/2 - pad - int(w*span)
	}
	plot := func(m map[uint32][][3]float32, xoff int) {
		i := 0
		ss := make([]uint32, 0, len(m))
		for s := range m {
			ss = append(ss, s)
		}
		sort.Slice(ss, func(a, b int) bool { return ss[a] < ss[b] })
		for _, s := range ss {
			col := okabeIto[i%len(okabeIto)]
			i++
			for _, v := range m[s] {
				x, y := pt(v, xoff)
				if x >= 0 && x < S && y >= 0 && y < S/2 {
					img.Set(x, y, col)
				}
			}
		}
	}
	plot(oracle, 0)   // gauche = oracle
	plot(bySlot, S/2) // droite = offline
	f, _ := os.Create(out)
	if f != nil {
		defer f.Close()
		png.Encode(f, img)
	}
	fmt.Printf("\n-> PNG %s (gauche=oracle, droite=offline)\n", out)
}
