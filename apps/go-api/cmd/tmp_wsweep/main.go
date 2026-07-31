// tmp_wsweep — SWEEP de la LARGEUR DE BASE d'axe i0 (TraversalPrecision.AxisW=[W,W,W]) sur
// le chemin DOMINANT (DELTAAXIS: 3xR(AxisW) dans consumePredictedDelta, ET absolu via
// absAxisW). Contrairement aux sweeps precedents (SetAbsoluteAxisW = seul l'absolu rare
// bougeait), ici on branche la largeur sur pd.AxisW -> le chemin delta biped bouge.
//
// Pour chaque W in {6,13,14,15,16} on mesure :
//
//	(A) MUR : records bipeds clean vs desync (anchor-scan DecodeDeltaRecordAt sur slots
//	    512-519, chunks 2..26). Le mur recule-t-il = plus de clean ? histo DesyncAt.
//	(B) POSITIONS : accumulation offline (ScanFrameTargets + World persistant) avec dequant
//	    CENTRE (q-2^(W-1))*quantum. box oracle %, spans X/Y/Z, sigma_z, densite/slot.
//	(C) FIT DIRECT (meilleur W) : q brut de i0 par slot, reconstruction, comparaison oracle.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_wsweep
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
	cache   = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
	oracleP = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/.ai/re_dump/ce_pos_oracle.csv`
	scratch = `c:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration/d6f109c3-2822-4e19-8fd4-cf65c50fec3d/scratchpad`
	idLow   = 11
	t0Us    = uint64(4537898226)
)

var bipedSlots = []uint32{512, 513, 514, 515, 516, 517, 518, 519}
var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: idLow}

// oracle box
var boxLo = [3]float64{-6, -25, -4}
var boxHi = [3]float64{36, 27, 7}

var okabeIto = []color.RGBA{
	{0xE6, 0x9F, 0x00, 255}, {0x56, 0xB4, 0xE9, 255}, {0x00, 0x9E, 0x73, 255},
	{0xF0, 0xE4, 0x42, 255}, {0x00, 0x72, 0xB2, 255}, {0xD5, 0x5E, 0x00, 255},
	{0xCC, 0x79, 0xA7, 255}, {0xBB, 0xBB, 0xBB, 255},
}

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

type frame struct {
	ts  uint64
	pay []byte
}

func framePayload(d []byte, want uint16) []byte {
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		if typ == want {
			return d[off+16 : off+16+sz]
		}
		off += 16 + sz
	}
	return nil
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

// bitsAt lit `n` bits MSB-first a partir du bit b (n<=32).
func bitsAt(d []byte, b, n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		bit := b + i
		by := bit >> 3
		if by >= len(d) {
			return v << (n - i)
		}
		bitv := (d[by] >> (7 - uint(bit&7))) & 1
		v = (v << 1) | uint32(bitv)
	}
	return v
}

var fullBindsCache [][2]uint32

func fullBinds() [][2]uint32 {
	if fullBindsCache != nil {
		return fullBindsCache
	}
	raw, _ := os.ReadFile(cache + "/world_dump_full.txt")
	for _, tok := range bytes.Fields(raw) {
		s := string(tok)
		if len(s) == 0 || s[0] == '#' {
			continue
		}
		var slot, ti uint32
		if _, e := fmt.Sscanf(s, "%d:%d", &slot, &ti); e == nil {
			fullBindsCache = append(fullBindsCache, [2]uint32{slot, ti})
		}
	}
	return fullBindsCache
}

func fullSeedWorld(reg *filmdec.Registry) *filmdec.World {
	w := filmdec.NewWorld(reg)
	for _, b := range fullBinds() {
		w.BindFull(b[0], b[1])
	}
	return w
}

// ---- oracle ----

type orec struct {
	slot    int
	x, y, z float64
}

func loadOracle() []orec {
	f, err := os.Open(oracleP)
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []orec
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		t := strings.TrimSpace(sc.Text())
		if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "eid") {
			continue
		}
		f := strings.Split(t, ",")
		if len(f) < 6 {
			continue
		}
		slot, e1 := strconv.Atoi(strings.TrimSpace(f[1]))
		x, e2 := strconv.ParseFloat(strings.TrimSpace(f[3]), 64)
		y, e3 := strconv.ParseFloat(strings.TrimSpace(f[4]), 64)
		z, e4 := strconv.ParseFloat(strings.TrimSpace(f[5]), 64)
		if e1 == nil && e2 == nil && e3 == nil && e4 == nil {
			out = append(out, orec{slot, x, y, z})
		}
	}
	return out
}

func oracleGridQuantum(rs []orec) float64 {
	m := map[int][][3]float64{}
	for _, r := range rs {
		m[r.slot] = append(m[r.slot], [3]float64{r.x, r.y, r.z})
	}
	axisMin := func(axis int) float64 {
		best := math.MaxFloat64
		for _, s := range m {
			for i := 1; i < len(s); i++ {
				d := math.Abs(s[i][axis] - s[i-1][axis])
				if d > 0.005 && d < best {
					best = d
				}
			}
		}
		return best
	}
	q := []float64{axisMin(0), axisMin(1), axisMin(2)}
	sort.Float64s(q)
	return q[1]
}

// ---- (A) MUR : anchor-scan biped clean/desync ----

type wallStat struct {
	clean, desync int
	desyncAt      map[int]int // component index -> count
	reachI63      int         // clean records whose comps reach index>=63 (frontier)
	endAtFrontier int         // clean records ending near frame end
}

func surveyWall(reg *filmdec.Registry, frames []frame) wallStat {
	ws := wallStat{desyncAt: map[int]int{}}
	for _, fr := range frames {
		frameBits := len(fr.pay) * 8
		w := fullSeedWorld(reg)
		// dedup par (slot, position P) : anchor-scan brut (candidats).
		for P := 0; P+14 < frameBits; P++ {
			if bitsAt(fr.pay, P, 1) != 1 { // DELTA prefix
				continue
			}
			slot := bitsAt(fr.pay, P+1, 11) // R(11) low, idBase=0
			if slot < 512 || slot > 519 {
				continue
			}
			// tag R(2) a P+12 ; header total 14 bits -> mask a P+14
			t, end := filmdec.DecodeDeltaRecordAt(fr.pay, P+14, w, slot)
			body := end - (P + 14)
			if t.DesyncAt != -1 {
				// desync candidate : ne compter que ceux "plausiblement alignes"
				// (au moins qq comps decodes avant le mur, pour rejeter le garbage total).
				if len(t.Comps) >= 4 {
					ws.desync++
					ws.desyncAt[t.DesyncAt]++
				}
				continue
			}
			if body < 30 {
				continue // trop court = faux-clean fortuit
			}
			ws.clean++
			maxIdx := -1
			for _, c := range t.Comps {
				if c.Index > maxIdx {
					maxIdx = c.Index
				}
			}
			if maxIdx >= 63 {
				ws.reachI63++
			}
			if end >= frameBits-64 {
				ws.endAtFrontier++
			}
		}
	}
	return ws
}

// ---- (B) POSITIONS : accumulation offline ----

type binding struct {
	full uint32
	ti   uint32
}

func offlineBindings(pay []byte) []binding {
	recs := filmdec.WalkKeyframeWorld(pay)
	out := make([]binding, 0, len(recs))
	for _, r := range recs {
		out = append(out, binding{full: uint32((r.Gen << 30) | r.Slot), ti: uint32(r.TI)})
	}
	return out
}

// posResult : positions accumulees par slot (dequant centre).
func offlinePositions(reg *filmdec.Registry, offBs []binding, quantum float64) map[int][][3]float64 {
	w := filmdec.NewWorld(reg)
	bipedSet := map[uint32]bool{}
	for _, s := range bipedSlots {
		bipedSet[s] = true
	}
	for _, b := range offBs {
		w.BindFull(b.full, b.ti)
	}
	traj := map[int][][3]float64{}
	filmdec.SetPositionAccumulator(w)
	filmdec.SetAbsDequantMode(filmdec.AbsDequantCenteredQuantum)
	filmdec.SetDeltaQuantum(float32(quantum))
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) {
		if bipedSet[s.Slot] {
			traj[int(s.Slot)] = append(traj[int(s.Slot)], [3]float64{float64(s.Vec[0]), float64(s.Vec[1]), float64(s.Vec[2])})
		}
	})
	filmdec.SetRecordStateParam(2)
	for c := 2; c <= 26; c++ {
		data := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, c))
		if data == nil {
			continue
		}
		for _, fr := range listFrames(data) {
			filmdec.ScanFrameTargets(fr.pay, w, calCfg, bipedSet, filmdec.HarvestNextBound)
		}
	}
	filmdec.SetPositionAccumulator(nil)
	filmdec.SetPositionCaptureHook(nil)
	filmdec.SetAbsDequantMode(filmdec.AbsDequantRange)
	return traj
}

type posStat struct {
	nPts     int
	inBox    int
	spanX    float64
	spanY    float64
	spanZ    float64
	sigmaZ   float64
	perSlotN map[int]int
}

func analysePositions(traj map[int][][3]float64) posStat {
	ps := posStat{perSlotN: map[int]int{}}
	var mn, mx [3]float64
	first := true
	var szsum float64
	var szn int
	for _, s := range bipedSlots {
		seq := traj[int(s)]
		ps.perSlotN[int(s)] = len(seq)
		ps.nPts += len(seq)
		if len(seq) >= 2 {
			var m, m2 float64
			for _, p := range seq {
				m += p[2]
			}
			m /= float64(len(seq))
			for _, p := range seq {
				m2 += (p[2] - m) * (p[2] - m)
			}
			szsum += math.Sqrt(m2 / float64(len(seq)))
			szn++
		}
		for _, p := range seq {
			inb := true
			for a := 0; a < 3; a++ {
				if p[a] < boxLo[a] || p[a] > boxHi[a] {
					inb = false
				}
				if first {
					mn[a], mx[a] = p[a], p[a]
				} else {
					if p[a] < mn[a] {
						mn[a] = p[a]
					}
					if p[a] > mx[a] {
						mx[a] = p[a]
					}
				}
			}
			first = false
			if inb {
				ps.inBox++
			}
		}
	}
	if ps.nPts > 0 {
		ps.spanX = mx[0] - mn[0]
		ps.spanY = mx[1] - mn[1]
		ps.spanZ = mx[2] - mn[2]
	}
	if szn > 0 {
		ps.sigmaZ = szsum / float64(szn)
	}
	return ps
}

// ---- (C) FIT DIRECT : q brut de i0 (delta) par slot ----
// On capture le delta BRUT (accumWorld=nil) -> q = round(delta/quantum)+2^(W-1).
// On reconstruit la trajectoire par accumulation des q-crans, span vs oracle.

// absQ capture le q BRUT des chemins ABSOLUS i0 (chemin dominant biped), par slot/axe.
// Sous AbsDequantCenteredQuantum : Vec[a]=(q-half)*quantum -> q = round(Vec/quantum)+half.
func absQTrajectory(reg *filmdec.Registry, offBs []binding, W uint, quantum float64) map[int][3][]int {
	bipedSet := map[uint32]bool{}
	for _, s := range bipedSlots {
		bipedSet[s] = true
	}
	half := float64(int(1) << (W - 1))
	qseq := map[int][3][]int{} // slot -> [axis]q list (absolute only)
	filmdec.SetPositionAccumulator(nil)
	filmdec.SetAbsDequantMode(filmdec.AbsDequantCenteredQuantum)
	filmdec.SetDeltaQuantum(float32(quantum))
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) {
		if !bipedSet[s.Slot] {
			return
		}
		if s.Kind != filmdec.PosKindAbsolute && s.Kind != filmdec.PosKindAbsFallback {
			return
		}
		cur := qseq[int(s.Slot)]
		for a := 0; a < 3; a++ {
			q := int(math.Round(float64(s.Vec[a])/quantum + half))
			cur[a] = append(cur[a], q)
		}
		qseq[int(s.Slot)] = cur
	})
	filmdec.SetRecordStateParam(2)
	w := filmdec.NewWorld(reg)
	for _, b := range offBs {
		w.BindFull(b.full, b.ti)
	}
	for c := 2; c <= 26; c++ {
		data := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, c))
		if data == nil {
			continue
		}
		for _, fr := range listFrames(data) {
			filmdec.ScanFrameTargets(fr.pay, w, calCfg, bipedSet, filmdec.HarvestNextBound)
		}
	}
	filmdec.SetPositionCaptureHook(nil)
	filmdec.SetAbsDequantMode(filmdec.AbsDequantRange)
	return qseq
}

// ---- PNG ----

func renderPNG(traj map[int][][3]float64, out string) (int, error) {
	type track struct {
		slot int
		pts  [][3]float64
	}
	var tracks []track
	for _, s := range bipedSlots {
		if len(traj[int(s)]) >= 15 {
			tracks = append(tracks, track{int(s), traj[int(s)]})
		}
	}
	if len(tracks) == 0 {
		return 0, fmt.Errorf("aucune trajectoire >=15 points")
	}
	var mn, mx [3]float64
	first := true
	for _, t := range tracks {
		for _, e := range t.pts {
			for a := 0; a < 3; a++ {
				if first {
					mn[a], mx[a] = e[a], e[a]
				}
				if e[a] < mn[a] {
					mn[a] = e[a]
				}
				if e[a] > mx[a] {
					mx[a] = e[a]
				}
			}
			first = false
		}
	}
	// axes = X (0) et Y (1) : plan de sol
	ax, ay := 0, 1
	const S, pad = 820, 40
	span := float64(S - 2*pad)
	img := image.NewRGBA(image.Rect(0, 0, S, S))
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			img.Set(x, y, color.RGBA{14, 16, 22, 255})
		}
	}
	sx, sy := mx[ax]-mn[ax], mx[ay]-mn[ay]
	if sx == 0 {
		sx = 1
	}
	if sy == 0 {
		sy = 1
	}
	px := func(v [3]float64) (int, int) {
		return pad + int((v[ax]-mn[ax])/sx*span), S - pad - int((v[ay]-mn[ay])/sy*span)
	}
	for i, t := range tracks {
		col := okabeIto[i%len(okabeIto)]
		var lx, ly int
		for j, e := range t.pts {
			x, y := px(e)
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
	f, err := os.Create(out)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return len(tracks), png.Encode(f, img)
}

func main() {
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	filmdec.SetRecordStateParam(2)

	// PNGONLY=<W> : rend uniquement les trajectoires du W donne (skip le survey lourd)
	if v := os.Getenv("PNGONLY"); v != "" {
		var wv uint
		fmt.Sscanf(v, "%d", &wv)
		oracle := loadOracle()
		quantum := oracleGridQuantum(oracle)
		offBs := offlineBindings(framePayload(inflate(cache+"/chunk_02.bin"), 2))
		filmdec.TraversalPrecision = filmdec.PrecisionDescriptor{IndexW: 1, AxisW: [3]uint{wv, wv, wv}}
		traj := offlinePositions(reg, offBs, quantum)
		p := fmt.Sprintf("%s/trajectoires_w%d.png", scratch, wv)
		nt, e := renderPNG(traj, p)
		fmt.Printf("PNGONLY W=%d -> %s (%d slots, err=%v)\n", wv, p, nt, e)
		return
	}

	// pre-charge frames (chunks 2..26) une fois
	var frames []frame
	for c := 2; c <= 26; c++ {
		data := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, c))
		if data == nil {
			continue
		}
		frames = append(frames, listFrames(data)...)
	}
	offBs := offlineBindings(framePayload(inflate(cache+"/chunk_02.bin"), 2))

	// quantum oracle
	oracle := loadOracle()
	quantum := oracleGridQuantum(oracle)
	oBox := map[int][][3]float64{}
	for _, r := range oracle {
		oBox[r.slot] = append(oBox[r.slot], [3]float64{r.x, r.y, r.z})
	}

	origPrec := filmdec.TraversalPrecision

	var sb strings.Builder
	W := func(f string, a ...any) { fmt.Fprintf(&sb, f, a...) }
	W("========== SWEEP LARGEUR DE BASE i0 (TraversalPrecision.AxisW) ==========\n")
	W("frames(chunks 2..26)=%d  quantum_oracle=%.5f  offBindings=%d\n", len(frames), quantum, len(offBs))
	W("oracle box X[%.0f,%.0f] Y[%.0f,%.0f] Z[%.0f,%.0f]  spans 42/53/11\n\n", boxLo[0], boxHi[0], boxLo[1], boxHi[1], boxLo[2], boxHi[2])

	Ws := []uint{6, 13, 14, 15, 16}
	type row struct {
		w  uint
		ws wallStat
		ps posStat
	}
	var rows []row
	W("W  | clean | desync | reachI63 | endFront | i0pts | inBox%% | spanX | spanY | spanZ | sigmaZ\n")
	W("---+-------+--------+----------+----------+-------+--------+-------+-------+-------+-------\n")
	for _, wv := range Ws {
		filmdec.TraversalPrecision = filmdec.PrecisionDescriptor{IndexW: 1, AxisW: [3]uint{wv, wv, wv}}
		ws := surveyWall(reg, frames)
		traj := offlinePositions(reg, offBs, quantum)
		ps := analysePositions(traj)
		rows = append(rows, row{wv, ws, ps})
		boxPct := 0.0
		if ps.nPts > 0 {
			boxPct = 100 * float64(ps.inBox) / float64(ps.nPts)
		}
		W("%-2d | %-5d | %-6d | %-8d | %-8d | %-5d | %6.1f | %5.1f | %5.1f | %5.1f | %.3f\n",
			wv, ws.clean, ws.desync, ws.reachI63, ws.endAtFrontier, ps.nPts, boxPct, ps.spanX, ps.spanY, ps.spanZ, ps.sigmaZ)
	}

	// detail DesyncAt histo pour chaque W
	W("\n---- histogramme DesyncAt (composant ou le record biped desync) ----\n")
	for _, r := range rows {
		type kv struct{ idx, n int }
		var arr []kv
		for k, v := range r.ws.desyncAt {
			arr = append(arr, kv{k, v})
		}
		sort.Slice(arr, func(i, j int) bool { return arr[i].n > arr[j].n })
		W("W=%d desync-total=%d : ", r.w, r.ws.desync)
		for i := 0; i < 6 && i < len(arr); i++ {
			W("i%d=%d ", arr[i].idx, arr[i].n)
		}
		W("\n")
	}

	// choix du meilleur W : max clean records, puis max inBox%
	best := rows[0]
	for _, r := range rows[1:] {
		bp := func(x row) float64 {
			if x.ps.nPts == 0 {
				return 0
			}
			return float64(x.ps.inBox) / float64(x.ps.nPts)
		}
		if r.ws.clean > best.ws.clean || (r.ws.clean == best.ws.clean && bp(r) > bp(best)) {
			best = r
		}
	}
	W("\n>>> meilleur W (max clean, tie=inBox) = %d\n", best.w)

	// ---- FIT DIRECT (chemin ABSOLU dominant) ----
	// oracle spans par axe (global)
	var oMn, oMx [3]float64
	ofirst := true
	for _, r := range oracle {
		v := [3]float64{r.x, r.y, r.z}
		for a := 0; a < 3; a++ {
			if ofirst {
				oMn[a], oMx[a] = v[a], v[a]
			}
			if v[a] < oMn[a] {
				oMn[a] = v[a]
			}
			if v[a] > oMx[a] {
				oMx[a] = v[a]
			}
		}
		ofirst = false
	}
	oSpan := [3]float64{oMx[0] - oMn[0], oMx[1] - oMn[1], oMx[2] - oMn[2]}
	W("\n---- FIT DIRECT (q BRUT du chemin ABSOLU i0, dominant) ----\n")
	W("oracle spans : X=%.1f Y=%.1f Z=%.1f (box %.0f/%.0f/%.0f)\n", oSpan[0], oSpan[1], oSpan[2], oSpan[0], oSpan[1], oSpan[2])
	W("pour chaque W : q-span brut (bits utiles) + step_implique = oracleSpan/q-span (doit ~= quantum %.5f si W correct)\n", quantum)
	for _, wv := range []uint{13, 14, 15, 16} {
		filmdec.TraversalPrecision = filmdec.PrecisionDescriptor{IndexW: 1, AxisW: [3]uint{wv, wv, wv}}
		aq := absQTrajectory(reg, offBs, wv, quantum)
		// slot le mieux echantillonne
		bestSlot, bestN := -1, 0
		for s, ax := range aq {
			if len(ax[0]) > bestN {
				bestN, bestSlot = len(ax[0]), s
			}
		}
		if bestSlot < 0 {
			W("  W=%d : aucun abs capture\n", wv)
			continue
		}
		ax := aq[bestSlot]
		W("  W=%d slot%d (n=%d abs) :\n", wv, bestSlot, bestN)
		for a := 0; a < 3; a++ {
			qmin, qmax := 1<<30, -(1 << 30)
			for _, q := range ax[a] {
				if q < qmin {
					qmin = q
				}
				if q > qmax {
					qmax = q
				}
			}
			qspan := float64(qmax - qmin)
			step := 0.0
			if qspan > 0 {
				step = oSpan[a] / qspan
			}
			bits := 0
			if qmax > 0 {
				bits = int(math.Ceil(math.Log2(float64(qmax) + 1)))
			}
			axn := []string{"X", "Y", "Z"}[a]
			W("    %s: q[min=%d max=%d span=%.0f bits~%d] step_implique=%.5f %s\n",
				axn, qmin, qmax, qspan, bits, step,
				map[bool]string{true: "(~quantum: PLAUSIBLE)", false: "(!= quantum)"}[math.Abs(step-quantum) < 0.4*quantum])
		}
	}

	// ---- PNG best W ----
	filmdec.TraversalPrecision = filmdec.PrecisionDescriptor{IndexW: 1, AxisW: [3]uint{best.w, best.w, best.w}}
	trajBest := offlinePositions(reg, offBs, quantum)
	pngPath := fmt.Sprintf("%s/trajectoires_w%d.png", scratch, best.w)
	nt, perr := renderPNG(trajBest, pngPath)
	if perr != nil {
		W("\nPNG: erreur %v\n", perr)
	} else {
		W("\nPNG: %s (%d slots)\n", pngPath, nt)
	}

	filmdec.TraversalPrecision = origPrec // remise du defaut

	_ = os.MkdirAll(scratch, 0o755)
	_ = os.WriteFile(scratch+"/w_sweep.txt", []byte(sb.String()), 0o644)
	fmt.Print(sb.String())
	fmt.Printf("\n-> %s/w_sweep.txt\n", scratch)
}
