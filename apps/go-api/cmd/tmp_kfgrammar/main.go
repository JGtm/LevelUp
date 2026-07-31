// tmp_kfgrammar — décodeur i0 des bipeds keyframe par PORT BIT-EXACT de la grammaire
// FUN_140f44c38 (default-state biped). THROWAWAY / harness de validation.
//
// DÉMARCHE (≠ tmp_kfworldpos qui localisait i0 par CONSENSUS de gate) :
//  1. Pour chaque biped keyframe (stateBit connu), on RUN la grammaire portée
//     filmdec.BipedDefaultStateEndBit(pay, stateBit) qui avance le curseur champ par
//     champ (version, name, entityRef, bloc multiplayer-props FUN_14080cfe8, gates,
//     R(19), ...). Le curseur final = l'offset CALCULÉ où i0 commence (par-biped, il
//     varie selon les bits de gate name/ref du record — attendu).
//  2. À cet offset on décode i0 = consumeAbsoluteWithGate :
//     precHigh R(1) + idxSel R(1) + idx R(1) + 3×R(13), déquant QuantRangeCEBiped.
//     width=13 car span_X(113.2)/2^13 = 0.0138 = le quantum oracle exact.
//  3. VALIDATION : 8 positions distinctes, non-dégénérées, DANS la boîte oracle
//     x[-6.33,35.70] y[-25.14,27.50] z[-4.20,7.08] (Z inclus).
//
// Le port de la grammaire vit dans filmdec (consumeBipedDefaultState) ; ici on ne fait
// que l'invoquer + déquantifier + valider.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_kfgrammar [filmDir]
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

const (
	defFilm    = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
	axisW      = 13 // span_X/2^13 = 0.0138 = quantum oracle
	scratchpad = `C:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration/d6f109c3-2822-4e19-8fd4-cf65c50fec3d/scratchpad`
)

// stateBits des 8 bipeds keyframe (biped_record_offsets.txt : stateBit = hdrBit+64).
var bipeds = []struct {
	Slot, StateBit int
}{
	{512, 193467}, {513, 196252}, {514, 199068}, {515, 201862},
	{516, 204665}, {517, 207460}, {518, 210262}, {519, 213057},
}

type row struct {
	slot, stateBit, endBit int
}

// boîte oracle (globale du dump CE ce_pos_oracle.csv).
var oracleBox = [3][2]float32{{-6.33, 35.70}, {-25.14, 27.50}, {-4.20, 7.08}}

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

func readBits(buf []byte, pos, n int) uint64 {
	var r uint64
	for i := 0; i < n; i++ {
		p := pos + i
		var bit uint64
		if idx := p >> 3; idx < len(buf) {
			bit = uint64(buf[idx]>>(7-uint(p&7))) & 1
		}
		r = r<<1 | bit
	}
	return r
}

func finite(v float32) bool { return !math.IsNaN(float64(v)) && !math.IsInf(float64(v), 0) }

// decodeI0 décode consumeAbsoluteWithGate à la position bit p avec QuantRangeCEBiped
// et width axisW. Retourne (pos, gateOK) : gateOK = precHigh(0)+idxSel(0)+idx(1b==0)
// (chemin in-map DAT_14462cbe0[0]). Réplique dequantWorldAxis (min+step*(q+0.5)).
func decodeI0(pay []byte, p int) (v [3]float32, gateOK bool) {
	if readBits(pay, p, 1) != 0 { // precHigh
		return
	}
	if readBits(pay, p+1, 1) != 0 { // idxSel (0 -> lit l'index)
		return
	}
	if readBits(pay, p+2, 1) != 0 { // idx (IndexW=1 ; 0 -> in-map)
		return
	}
	rng := filmdec.QuantRangeCEBiped
	q := p + 3
	for i := 0; i < 3; i++ {
		w := readBits(pay, q, axisW)
		q += axisW
		scale := float32(uint64(1) << uint(axisW))
		step := (rng[i].Max - rng[i].Min) / scale
		v[i] = float32(w)*step + rng[i].Min + step*0.5
	}
	gateOK = true
	return
}

// axisMinSpread retourne le plus petit étalement (max-min) parmi les 3 axes du nuage :
// un décodage dégénéré (au moins un axe constant) donne 0.
func axisMinSpread(pts map[int][3]float32) float32 {
	if len(pts) == 0 {
		return 0
	}
	var mn, mx [3]float32
	first := true
	for _, v := range pts {
		for i := 0; i < 3; i++ {
			if first || v[i] < mn[i] {
				mn[i] = v[i]
			}
			if first || v[i] > mx[i] {
				mx[i] = v[i]
			}
		}
		first = false
	}
	ms := mx[0] - mn[0]
	for i := 1; i < 3; i++ {
		if s := mx[i] - mn[i]; s < ms {
			ms = s
		}
	}
	return ms
}

func inBox(v [3]float32) bool {
	for i := 0; i < 3; i++ {
		if !finite(v[i]) || v[i] < oracleBox[i][0] || v[i] > oracleBox[i][1] {
			return false
		}
	}
	return true
}

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	pay := framePayload(inflate(dir+"/chunk_02.bin"), 2)
	if pay == nil {
		fmt.Println("type-2 introuvable dans chunk_02")
		return
	}
	fmt.Printf("chunk_02 : payload %d o (%d bits) ; axisW=%d range=%v\n\n", len(pay), len(pay)*8, axisW, filmdec.QuantRangeCEBiped)

	// ===== 0. TRACE grammaire pour slot 512 (diagnostic alignement) =====
	fmt.Println("=== TRACE grammaire slot 512 (champ -> bitpos, offset depuis stateBit) ===")
	sb0 := bipeds[0].StateBit
	filmdec.SetRepTraceHook(func(label string, bp int) {
		fmt.Printf("  %-14s bit=%d (+%d)\n", label, bp, bp-sb0)
	})
	filmdec.BipedDefaultStateEndBit(pay, sb0)
	filmdec.SetRepTraceHook(nil)
	fmt.Println()

	// ===== 1. offset i0 CALCULÉ par la grammaire, par biped =====
	fmt.Println("=== offset i0 CALCULÉ (grammaire FUN_140f44c38) ===")
	fmt.Printf("%-6s %-9s %-9s %-8s %-8s\n", "slot", "stateBit", "endBit", "endBit-sb", "version")
	rows := make([]row, 0, 8)
	for _, b := range bipeds {
		end := filmdec.BipedDefaultStateEndBit(pay, b.StateBit)
		ver := filmdec.LastRepVersion()
		rows = append(rows, row{b.Slot, b.StateBit, end})
		fmt.Printf("%-6d %-9d %-9d %-8d %-8d\n", b.Slot, b.StateBit, end, end-b.StateBit, ver)
	}

	// ===== 2. décodage i0 à l'offset calculé (± lead FUN_14076e420) =====
	// Le port se termine juste avant i0. i0 = consumeAbsoluteWithGate. Un possible
	// R(1) range-table-select (FUN_14076e420) précède peut-être : on essaie lead∈{0,1}.
	fmt.Println("\n=== décodage i0 à l'offset calculé ===")
	best := struct {
		lead, gate, box, distinct int
		pts                       map[int][3]float32
	}{lead: -1}
	for _, lead := range []int{0, 1} {
		pts := map[int][3]float32{}
		gate, box := 0, 0
		for _, r := range rows {
			v, ok := decodeI0(pay, r.endBit+lead)
			if ok {
				gate++
				pts[r.slot] = v
				if inBox(v) {
					box++
				}
			}
		}
		seen := map[string]bool{}
		for _, v := range pts {
			seen[fmt.Sprintf("%.2f_%.2f_%.2f", v[0], v[1], v[2])] = true
		}
		fmt.Printf("  lead=%d : gate=%d/8 inBox=%d/8 distinct=%d\n", lead, gate, box, len(seen))
		if gate > best.gate || (gate == best.gate && box > best.box) {
			best.lead, best.gate, best.box, best.distinct, best.pts = lead, gate, box, len(seen), pts
		}
	}

	// ===== 3. sweep de contrôle autour de l'offset (honnêteté : boundary) =====
	// WIDE sweep offset-from-stateBit (axisW=13, oracle box).
	fmt.Println("\n=== WIDE sweep (offset depuis stateBit, axisW=13) : gate>=6 ===")
	wideBestOff, wideBestBox, wideBestDist := -1, -1, -1
	var wideBestPts map[int][3]float32
	for off := 100; off <= 340; off++ {
		pts := map[int][3]float32{}
		gate, box := 0, 0
		for _, b := range bipeds {
			if v, ok := decodeI0(pay, b.StateBit+off); ok {
				gate++
				pts[b.Slot] = v
				if inBox(v) {
					box++
				}
			}
		}
		if gate < 6 {
			continue
		}
		seen := map[string]bool{}
		xs := map[float64]bool{}
		for _, v := range pts {
			seen[fmt.Sprintf("%.2f_%.2f_%.2f", v[0], v[1], v[2])] = true
			xs[math.Round(float64(v[0])*100)/100] = true
		}
		mark := ""
		if box == 8 && len(seen) == 8 {
			mark = "  <== 8/8 IN-BOX DISTINCT"
		}
		// score = box prioritaire, puis étalement minimal par axe (anti-dégénérescence :
		// un gradient X monotone Y/Z constants a minSpread=0 et perd).
		minSpread := int(axisMinSpread(pts))
		if gate == 8 && (box > wideBestBox || (box == wideBestBox && minSpread > wideBestDist)) {
			wideBestOff, wideBestBox, wideBestDist, wideBestPts = off, box, minSpread, pts
		}
		fmt.Printf("  +%-3d gate=%d inBox=%d distinct=%d xbuckets=%d%s\n", off, gate, box, len(seen), len(xs), mark)
	}
	// Pour les artefacts : si l'offset grammaire (endBit) ne gate pas, on rapporte le
	// MEILLEUR offset du wide-sweep (max inBox puis distinct) — clairement étiqueté.
	if best.gate < 8 && wideBestOff >= 0 {
		best.lead, best.gate, best.box, best.distinct, best.pts = wideBestOff, 8, wideBestBox, wideBestDist, wideBestPts
		fmt.Printf("\n(offset grammaire ne gate pas -> artefacts sur meilleur wide-sweep +%d : inBox=%d distinct=%d)\n", wideBestOff, wideBestBox, wideBestDist)
	}

	// DUMP positions aux offsets forts gate-8-distinct-8 (caractérisation vs box).
	fmt.Println("\n=== DUMP positions (width13+CEBiped) aux offsets gate-8 ===")
	for _, off := range []int{113, 156, 182, 272, 303} {
		fmt.Printf("--- offset +%d ---\n", off)
		for _, b := range bipeds {
			if v, ok := decodeI0(pay, b.StateBit+off); ok {
				fmt.Printf("  slot=%d (%.2f, %.2f, %.2f) inBox=%v\n", b.Slot, v[0], v[1], v[2], inBox(v))
			}
		}
	}

	fmt.Println("\n=== sweep de contrôle (endBit + delta) : gate/inBox/distinct ===")
	for delta := -6; delta <= 10; delta++ {
		pts := map[int][3]float32{}
		gate, box := 0, 0
		for _, r := range rows {
			if v, ok := decodeI0(pay, r.endBit+delta); ok {
				gate++
				pts[r.slot] = v
				if inBox(v) {
					box++
				}
			}
		}
		seen := map[string]bool{}
		for _, v := range pts {
			seen[fmt.Sprintf("%.2f_%.2f_%.2f", v[0], v[1], v[2])] = true
		}
		mark := ""
		if box == 8 && len(seen) == 8 {
			mark = "  <== 8/8 in-box distinct"
		}
		fmt.Printf("  +%-3d gate=%d inBox=%d distinct=%d%s\n", delta, gate, box, len(seen), mark)
	}

	// ===== 4. positions finales (meilleur lead) + validation =====
	fmt.Printf("\n=== POSITIONS i0 (lead=%d, gate=%d/8 inBox=%d/8 distinct=%d) ===\n", best.lead, best.gate, best.box, best.distinct)
	slots := make([]int, 0, len(best.pts))
	for s := range best.pts {
		slots = append(slots, s)
	}
	sort.Ints(slots)
	// non-dégénérescence : étalement par axe
	var spread [3]float32
	if len(best.pts) > 0 {
		var mn, mx [3]float32
		first := true
		for _, s := range slots {
			v := best.pts[s]
			for i := 0; i < 3; i++ {
				if first || v[i] < mn[i] {
					mn[i] = v[i]
				}
				if first || v[i] > mx[i] {
					mx[i] = v[i]
				}
			}
			first = false
		}
		for i := 0; i < 3; i++ {
			spread[i] = mx[i] - mn[i]
		}
	}
	for _, s := range slots {
		v := best.pts[s]
		fmt.Printf("  slot=%d  (%.3f, %.3f, %.3f)  inBox=%v\n", s, v[0], v[1], v[2], inBox(v))
	}
	fmt.Printf("étalement X=%.2f Y=%.2f Z=%.2f\n", spread[0], spread[1], spread[2])

	writeOutputs(best.pts, slots, rows, best.lead, best.gate, best.box, best.distinct, spread)
}

func writeOutputs(pts map[int][3]float32, slots []int, rows []row, lead, gate, box, distinct int, spread [3]float32) {
	// keyframe_pos.txt
	var sb bytes.Buffer
	fmt.Fprintf(&sb, "# i0 keyframe positions — port grammaire FUN_140f44c38 (offset CALCULÉ)\n")
	fmt.Fprintf(&sb, "# axisW=%d range=QuantRangeCEBiped lead=%d gate=%d/8 inBox=%d/8 distinct=%d\n", axisW, lead, gate, box, distinct)
	fmt.Fprintf(&sb, "# spread X=%.2f Y=%.2f Z=%.2f ; box x[-6.33,35.70] y[-25.14,27.50] z[-4.20,7.08]\n", spread[0], spread[1], spread[2])
	fmt.Fprintf(&sb, "# slot stateBit endBit i0Bit x y z inBox\n")
	endBy := map[int]int{}
	for _, r := range rows {
		endBy[r.slot] = r.endBit
	}
	for _, s := range slots {
		v := pts[s]
		fmt.Fprintf(&sb, "%d %d %d %d %.4f %.4f %.4f %v\n", s, 0, endBy[s], endBy[s]+lead, v[0], v[1], v[2], inBox(v))
	}
	_ = os.WriteFile(scratchpad+"/keyframe_pos.txt", sb.Bytes(), 0644)
	fmt.Printf("\nécrit %s/keyframe_pos.txt\n", scratchpad)

	// PNG : XY scatter des spawns + boîte oracle.
	const W, H = 600, 500
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	for i := range img.Pix {
		img.Pix[i] = 0x18
	}
	// axes monde -> pixels (X box -10..40, Y box -30..30)
	xm0, xm1 := float32(-12.0), float32(40.0)
	ym0, ym1 := float32(-30.0), float32(30.0)
	toPx := func(x, y float32) (int, int) {
		px := int((x - xm0) / (xm1 - xm0) * float32(W))
		py := int((1 - (y-ym0)/(ym1-ym0)) * float32(H))
		return px, py
	}
	// boîte oracle en cadre
	bx0, by0 := toPx(oracleBox[0][0], oracleBox[1][0])
	bx1, by1 := toPx(oracleBox[0][1], oracleBox[1][1])
	drawRect(img, bx0, by1, bx1, by0, color.RGBA{60, 90, 60, 255})
	pal := []color.RGBA{
		{240, 80, 80, 255}, {80, 200, 240, 255}, {240, 200, 60, 255}, {120, 240, 120, 255},
		{220, 120, 240, 255}, {240, 150, 60, 255}, {120, 160, 255, 255}, {255, 255, 255, 255},
	}
	for i, s := range slots {
		v := pts[s]
		px, py := toPx(v[0], v[1])
		drawDot(img, px, py, 5, pal[i%len(pal)])
	}
	f, err := os.Create(scratchpad + "/keyframe_spawns.png")
	if err == nil {
		_ = png.Encode(f, img)
		f.Close()
		fmt.Printf("écrit %s/keyframe_spawns.png\n", scratchpad)
	}
}

func drawDot(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				img.Set(cx+x, cy+y, c)
			}
		}
	}
}

func drawRect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	for x := x0; x <= x1; x++ {
		img.Set(x, y0, c)
		img.Set(x, y1, c)
	}
	for y := y0; y <= y1; y++ {
		img.Set(x0, y, c)
		img.Set(x1, y, c)
	}
}
