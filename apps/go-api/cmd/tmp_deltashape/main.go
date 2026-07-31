// tmp_deltashape — VALIDE que le chemin DELTA d'i0 (consumePredictedDelta : Delta8 /
// DeltaAxis) lit bien les deltas gameplay du film, en comparant la DISTRIBUTION des deltas
// offline (par slot biped, chunks 3..26) à celle de l'oracle CE (ce_pos_oracle.csv,
// différences consécutives par slot). Le juge est la distribution en QUANTA (delta/0.0138) :
// l'oracle est DOMINÉ par 1-4 quanta (moyenne ~0.04) ; si les deltas offline le sont aussi ->
// le décodeur delta lit au bon endroit. Puis ACCUMULE (seed 0/slot, reseed sur gros saut) pour
// produire des formes de trajectoires + PNG offline vs oracle.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_deltashape
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
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	cache     = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
	oracleCSV = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/.ai/re_dump/ce_pos_oracle.csv`
	scratch   = `c:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration/d6f109c3-2822-4e19-8fd4-cf65c50fec3d/scratchpad`
	idLowBits = 11
	quantum   = 0.01383
	chunkLo   = 3
	chunkHi   = 26
	bigJumpU  = 1.0 // |delta|>1u sur un axe = respawn/reseed (357 sauts oracle >12 quanta)
)

var bipedSlots = map[uint32]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}

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

type packet struct{ payload []byte }

func listType0(d []byte) []packet {
	var out []packet
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, packet{d[off+16 : off+16+sz]})
		}
		off += 16 + sz
	}
	return out
}

type binding struct{ full, ti uint32 }

func offlineBindings(pay []byte) []binding {
	recs := filmdec.WalkKeyframeWorld(pay)
	out := make([]binding, 0, len(recs))
	for _, r := range recs {
		out = append(out, binding{full: uint32((r.Gen << 30) | r.Slot), ti: uint32(r.TI)})
	}
	return out
}

func buildWorld(reg *filmdec.Registry, bs []binding) *filmdec.World {
	w := filmdec.NewWorld(reg)
	for _, b := range bs {
		w.BindFull(b.full, b.ti)
	}
	return w
}

// ---------- histogramme quanta ----------

type qhist struct {
	counts map[int]int // |quanta| -> nb
	nZero  int         // deltas nuls (0 quanta)
	nTot   int
	sumAbs float64 // somme des |delta| en unités monde (pour moyenne)
	nOn    int     // sur grille (|delta - round*q| < eps)
	nGrid  int
}

func newQhist() *qhist { return &qhist{counts: map[int]int{}} }

func (h *qhist) add(deltaWorld float64) {
	h.nTot++
	h.sumAbs += math.Abs(deltaWorld)
	q := deltaWorld / quantum
	rq := math.Round(q)
	// sur-grille : la valeur est-elle un multiple ~entier du quantum ?
	if math.Abs(q-rq) < 0.02 {
		h.nOn++
	}
	h.nGrid++
	aq := int(math.Abs(rq))
	if aq == 0 {
		h.nZero++
		return
	}
	h.counts[aq]++
}

func (h *qhist) report(label string) string {
	var b strings.Builder
	nz := h.nTot - h.nZero
	meanAll := 0.0
	if h.nTot > 0 {
		meanAll = h.sumAbs / float64(h.nTot)
	}
	fmt.Fprintf(&b, "%s : n=%d (dont %d nuls, %d non-nuls) moyenne|delta|=%.4f u ; sur-grille=%.1f%%\n",
		label, h.nTot, h.nZero, nz, meanAll, 100*float64(h.nOn)/math.Max(1, float64(h.nGrid)))
	// distribution 1..12+ quanta (sur les non-nuls)
	buckets := []int{1, 2, 3, 4}
	sml := 0
	for _, k := range buckets {
		sml += h.counts[k]
	}
	big := 0
	for k, c := range h.counts {
		if k > 12 {
			big += c
		}
	}
	fmt.Fprintf(&b, "   1q=%d 2q=%d 3q=%d 4q=%d  | 1-4q=%.1f%% des non-nuls | >12q=%d (%.2f%%)\n",
		h.counts[1], h.counts[2], h.counts[3], h.counts[4],
		100*float64(sml)/math.Max(1, float64(nz)), big, 100*float64(big)/math.Max(1, float64(nz)))
	return b.String()
}

// ---------- oracle ----------

type pt struct{ x, y, z float64 }

// readOracle retourne les trajectoires par slot (ordre du CSV) + l'histogramme des deltas consécutifs.
func readOracle() (map[int][]pt, *qhist) {
	raw, err := os.ReadFile(oracleCSV)
	if err != nil {
		return nil, nil
	}
	traj := map[int][]pt{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "eid") {
			continue
		}
		f := strings.Split(line, ",")
		if len(f) < 6 {
			continue
		}
		slot, _ := strconv.Atoi(f[1])
		x, _ := strconv.ParseFloat(f[3], 64)
		y, _ := strconv.ParseFloat(f[4], 64)
		z, _ := strconv.ParseFloat(f[5], 64)
		traj[slot] = append(traj[slot], pt{x, y, z})
	}
	h := newQhist()
	for _, ps := range traj {
		for i := 1; i < len(ps); i++ {
			h.add(ps[i].x - ps[i-1].x)
			h.add(ps[i].y - ps[i-1].y)
			h.add(ps[i].z - ps[i-1].z)
		}
	}
	return traj, h
}

// ---------- décodage offline ----------

// passRawDeltas : capture les deltas BRUTS (Delta8/DeltaAxis) des slots bipeds, SANS accumulation.
func passRawDeltas(reg *filmdec.Registry, offBs []binding, perSlot map[uint32]map[string]int) (*qhist, *qhist, map[string]int) {
	hD8 := newQhist()
	hDax := newQhist()
	kindCount := map[string]int{}
	filmdec.SetPositionAccumulator(nil)
	for c := chunkLo; c <= chunkHi; c++ {
		data := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, c))
		if data == nil {
			continue
		}
		for _, pk := range listType0(data) {
			w := buildWorld(reg, offBs)
			var curSlot uint32
			filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) {
				_ = curSlot
				if !bipedSlots[s.Slot] {
					return
				}
				kindCount[s.Kind.String()]++
				if perSlot[s.Slot] == nil {
					perSlot[s.Slot] = map[string]int{}
				}
				perSlot[s.Slot][s.Kind.String()]++
				switch s.Kind {
				case filmdec.PosKindDelta8:
					hD8.add(float64(s.Vec[0]))
					hD8.add(float64(s.Vec[1]))
					hD8.add(float64(s.Vec[2]))
				case filmdec.PosKindDeltaAxis:
					hDax.add(float64(s.Vec[0]))
					hDax.add(float64(s.Vec[1]))
					hDax.add(float64(s.Vec[2]))
				}
			})
			br := filmdec.NewBitReader(pk.payload)
			_, _ = filmdec.DecodeFrameRecords(br, w, filmdec.FrameConfig{HasExtraFields: false, IDLowBits: idLowBits})
		}
	}
	filmdec.SetPositionCaptureHook(nil)
	return hD8, hDax, kindCount
}

// passAccumulate : World PERSISTANT, accumulation par slot, reseed sur gros saut -> sous-trajectoires.
func passAccumulate(reg *filmdec.Registry, offBs []binding) map[uint32][][]pt {
	w := buildWorld(reg, offBs)
	filmdec.SetPositionAccumulator(w)
	defer filmdec.SetPositionAccumulator(nil)

	sub := map[uint32][][]pt{} // slot -> liste de sous-trajectoires
	last := map[uint32]pt{}    // dernière position émise par slot
	hasLast := map[uint32]bool{}

	push := func(slot uint32, p pt, reseed bool) {
		if _, ok := sub[slot]; !ok || reseed {
			sub[slot] = append(sub[slot], []pt{})
		}
		n := len(sub[slot])
		sub[slot][n-1] = append(sub[slot][n-1], p)
	}

	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) {
		if !bipedSlots[s.Slot] {
			return
		}
		p := pt{float64(s.Vec[0]), float64(s.Vec[1]), float64(s.Vec[2])}
		reseed := false
		if hasLast[s.Slot] {
			lp := last[s.Slot]
			if math.Abs(p.x-lp.x) > bigJumpU || math.Abs(p.y-lp.y) > bigJumpU || math.Abs(p.z-lp.z) > bigJumpU {
				reseed = true // respawn/teleport : nouvelle sous-trajectoire
			}
		}
		push(s.Slot, p, reseed)
		last[s.Slot] = p
		hasLast[s.Slot] = true
	})

	for c := chunkLo; c <= chunkHi; c++ {
		data := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, c))
		if data == nil {
			continue
		}
		for _, pk := range listType0(data) {
			br := filmdec.NewBitReader(pk.payload)
			_, _ = filmdec.DecodeFrameRecords(br, w, filmdec.FrameConfig{HasExtraFields: false, IDLowBits: idLowBits})
		}
	}
	filmdec.SetPositionCaptureHook(nil)
	return sub
}

// ---------- PNG ----------

var palette = []color.RGBA{
	{0xe6, 0x39, 0x46, 0xff}, {0x45, 0x7b, 0x9d, 0xff}, {0x2a, 0x9d, 0x8f, 0xff}, {0xe9, 0xc4, 0x6a, 0xff},
	{0xf4, 0xa2, 0x61, 0xff}, {0x8a, 0x5a, 0x33, 0xff}, {0x6a, 0x4c, 0x93, 0xff}, {0x9d, 0x35, 0x57, 0xff},
}

type polyline struct {
	pts []pt
	col color.RGBA
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx := int(math.Abs(float64(x1 - x0)))
	dy := -int(math.Abs(float64(y1 - y0)))
	sx, sy := 1, 1
	if x0 >= x1 {
		sx = -1
	}
	if y0 >= y1 {
		sy = -1
	}
	err := dx + dy
	for {
		if x0 >= 0 && x0 < img.Bounds().Dx() && y0 >= 0 && y0 < img.Bounds().Dy() {
			img.Set(x0, y0, c)
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

// renderPanel dessine des polylignes XZ (top-down) auto-échelonnées dans [x0,x0+w).
func renderPanel(img *image.RGBA, x0, w, h int, lines []polyline, title string) {
	// bornes
	minX, maxX, minZ, maxZ := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	any := false
	for _, l := range lines {
		for _, p := range l.pts {
			any = true
			minX, maxX = math.Min(minX, p.x), math.Max(maxX, p.x)
			minZ, maxZ = math.Min(minZ, p.z), math.Max(maxZ, p.z)
		}
	}
	if !any {
		return
	}
	spanX, spanZ := maxX-minX, maxZ-minZ
	if spanX < 1e-6 {
		spanX = 1
	}
	if spanZ < 1e-6 {
		spanZ = 1
	}
	pad := 20
	tx := func(x float64) int { return x0 + pad + int((x-minX)/spanX*float64(w-2*pad)) }
	tz := func(z float64) int { return pad + int((maxZ-z)/spanZ*float64(h-2*pad)) }
	for _, l := range lines {
		for i := 1; i < len(l.pts); i++ {
			drawLine(img, tx(l.pts[i-1].x), tz(l.pts[i-1].z), tx(l.pts[i].x), tz(l.pts[i].z), l.col)
		}
	}
	_ = title
}

func writePNG(path string, offLines, oraLines []polyline) error {
	w, h := 1600, 800
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = 0x12
	}
	for i := 3; i < len(img.Pix); i += 4 {
		img.Pix[i] = 0xff
	}
	renderPanel(img, 0, w/2, h, offLines, "offline")
	renderPanel(img, w/2, w/2, h, oraLines, "oracle")
	// séparateur
	for y := 0; y < h; y++ {
		img.Set(w/2, y, color.RGBA{0x55, 0x55, 0x55, 0xff})
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func main() {
	filmdec.SetRecordStateParam(2)
	filmdec.SetDeltaQuantum(quantum)

	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	pay02 := framePayload(inflate(cache+"/chunk_02.bin"), 2)
	if pay02 == nil {
		fmt.Println("chunk_02 frame type-2 introuvable")
		return
	}
	offBs := offlineBindings(pay02)
	nBiped := 0
	for _, b := range offBs {
		if (b.full&0x3fffffff) >= 512 && (b.full&0x3fffffff) <= 519 && b.ti == 35 {
			nBiped++
		}
	}

	// ---- oracle ----
	oraTraj, oraH := readOracle()

	// ---- pass A : deltas bruts ----
	perSlot := map[uint32]map[string]int{}
	hD8, hDax, kinds := passRawDeltas(reg, offBs, perSlot)

	// ---- pass B : accumulation ----
	sub := passAccumulate(reg, offBs)

	// ---- rapport ----
	var rep strings.Builder
	fmt.Fprintf(&rep, "=== tmp_deltashape — deltas i0 offline vs oracle CE ===\n")
	fmt.Fprintf(&rep, "keyframe bindings=%d ; bipeds 512-519 ti=35 bindés=%d/8 ; quantum=%.5f\n\n", len(offBs), nBiped, quantum)

	fmt.Fprintf(&rep, "--- ORACLE (ce_pos_oracle.csv, deltas consécutifs par slot) ---\n")
	if oraH != nil {
		rep.WriteString(oraH.report("oracle 3-axes"))
	}
	fmt.Fprintf(&rep, "\n--- OFFLINE (chunks %d..%d, slots bipeds 512-519) ---\n", chunkLo, chunkHi)
	fmt.Fprintf(&rep, "kinds i0 capturés bipeds : %v\n", kinds)
	fmt.Fprintf(&rep, "couverture par slot (samples i0 par chemin) :\n")
	for s := uint32(512); s <= 519; s++ {
		if perSlot[s] != nil {
			fmt.Fprintf(&rep, "   slot %d : %v\n", s, perSlot[s])
		} else {
			fmt.Fprintf(&rep, "   slot %d : (aucun sample)\n", s)
		}
	}
	rep.WriteString(hD8.report("offline Delta8 (signed8*q)"))
	rep.WriteString(hDax.report("offline DeltaAxis (6b centré)"))

	// combiné offline
	comb := newQhist()
	for k, v := range hD8.counts {
		comb.counts[k] += v
	}
	for k, v := range hDax.counts {
		comb.counts[k] += v
	}
	comb.nZero = hD8.nZero + hDax.nZero
	comb.nTot = hD8.nTot + hDax.nTot
	comb.sumAbs = hD8.sumAbs + hDax.sumAbs
	comb.nOn = hD8.nOn + hDax.nOn
	comb.nGrid = hD8.nGrid + hDax.nGrid
	rep.WriteString("\n")
	rep.WriteString(comb.report("offline COMBINÉ delta"))

	// ---- formes ----
	fmt.Fprintf(&rep, "\n--- FORMES accumulées (offline) ---\n")
	var offLines []polyline
	slots := []uint32{}
	for s := range sub {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	for _, s := range slots {
		tot, longest := 0, 0
		for _, t := range sub[s] {
			tot += len(t)
			if len(t) > longest {
				longest = len(t)
			}
			if len(t) >= 3 {
				offLines = append(offLines, polyline{pts: t, col: palette[int(s-512)%len(palette)]})
			}
		}
		fmt.Fprintf(&rep, "slot %d : %d points, %d sous-trajectoires (respawns/reseeds), plus longue=%d\n",
			s, tot, len(sub[s]), longest)
	}

	// oracle lines (par slot)
	var oraLines []polyline
	oslots := []int{}
	for s := range oraTraj {
		oslots = append(oslots, s)
	}
	sort.Ints(oslots)
	ci := 0
	for _, s := range oslots {
		if len(oraTraj[s]) >= 3 {
			oraLines = append(oraLines, polyline{pts: oraTraj[s], col: palette[ci%len(palette)]})
			ci++
		}
	}

	pngPath := scratch + "/delta_shapes.png"
	if err := writePNG(pngPath, offLines, oraLines); err != nil {
		fmt.Printf("PNG err: %v\n", err)
	}
	txtPath := scratch + "/delta_shapes.txt"
	_ = os.MkdirAll(scratch, 0o755)
	_ = os.WriteFile(txtPath, []byte(rep.String()), 0o644)

	fmt.Print(rep.String())
	fmt.Printf("\nPNG : %s\nTXT : %s\n", pngPath, txtPath)
}
