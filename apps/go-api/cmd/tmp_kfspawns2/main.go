// tmp_kfspawns2 — DIAGNOSTIC + validation du serializer MOVEMENT porté (r-c) et de
// l'offset i0 CALCULÉ (BipedDefaultStateEndBit + prologue has-components + masque
// FUN_1406d7610), contre l'oracle box. THROWAWAY.
//
// Phase A : structure. Pour chaque biped, endBit = BipedDefaultStateEndBit(stateBit) ;
//
//	inspecte les bits [endBit..endBit+80) : has-components, masque (gate/count/full),
//	puis les gates i0. Compare au +182 scanné par tmp_kfworldpos.
//
// Phase B : décode i0 au nouvel offset CALCULÉ (axisW=13, range CEBiped) -> (x,y,z).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_kfspawns2 [filmDir]
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
	"os"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

const defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

// stateBits des 8 bipeds (biped_record_offsets.txt : hdrBit+64).
var stateBits = []struct{ slot, stateBit int }{
	{512, 193467}, {513, 196252}, {514, 199068}, {515, 201862},
	{516, 204665}, {517, 207460}, {518, 210262}, {519, 213057},
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

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	pay := framePayload(inflate(dir+"/chunk_02.bin"), 2)
	if pay == nil {
		fmt.Println("type-2 introuvable")
		return
	}
	fmt.Printf("payload %d o (%d bits)\n\n", len(pay), len(pay)*8)

	// ===== Phase A : structure prologue + masque au endBit CALCULÉ =====
	fmt.Println("=== Phase A : endBit CALCULÉ (rep FUN_140f44c38) + prologue + masque ===")
	fmt.Println("slot | endOff(=endBit-stateBit) | endBit | +182scanné | bits[endBit..+40]")
	for _, b := range stateBits {
		endBit := filmdec.BipedDefaultStateEndBit(pay, b.stateBit)
		endOff := endBit - b.stateBit
		// bits bruts à partir de endBit
		hc := readBits(pay, endBit, 1)      // has-components gate [P2]
		mgate := readBits(pay, endBit+1, 1) // masque [M1] gate
		count := readBits(pay, endBit+2, 3) // si sparse : count
		fmt.Printf("  %d | +%d | %d | +182=>%d | hc=%d mgate=%d count(sparse)=%d\n",
			b.slot, endOff, endBit, b.stateBit+182, hc, mgate, count)
	}
	fmt.Println()

	// ===== Phase A2 : où est le VRAI i0 (stateBit+182) relativement à endBit ? =====
	fmt.Println("=== Phase A2 : delta (i0scanné=stateBit+182) - endBit = bits prologue+masque réels ===")
	for _, b := range stateBits {
		endBit := filmdec.BipedDefaultStateEndBit(pay, b.stateBit)
		i0scan := b.stateBit + 182
		fmt.Printf("  %d | i0scan=%d endBit=%d delta=%d\n", b.slot, i0scan, endBit, i0scan-endBit)
	}
	fmt.Println()

	// ===== Phase B : BALAYAGE offset ABSOLU (stateBit+off), axisW=13, range CEBiped, box oracle SERRÉE =====
	fmt.Println("=== Phase B : balayage off=stateBit+[100..320], axisW=13 CEBiped, box oracle serrée ===")
	rng := filmdec.QuantRangeCEBiped
	box := struct{ xlo, xhi, ylo, yhi, zlo, zhi float32 }{-6.33, 35.70, -25.14, 27.50, -4.20, 7.08}
	inBox := func(v [3]float32) bool {
		return v[0] >= box.xlo && v[0] <= box.xhi && v[1] >= box.ylo && v[1] <= box.yhi && v[2] >= box.zlo && v[2] <= box.zhi
	}
	scan := func(anchor func(b struct{ slot, stateBit int }) int, lo, hi, awv int, label string) {
		dq := func(q uint64, axis int) float32 {
			scale := float32(uint64(1) << uint(awv))
			step := (rng[axis].Max - rng[axis].Min) / scale
			return float32(q)*step + rng[axis].Min + step*0.5
		}
		dec := func(p int) ([3]float32, bool) {
			if readBits(pay, p, 1) != 0 || readBits(pay, p+1, 1) != 0 || readBits(pay, p+2, 1) != 0 {
				return [3]float32{}, false
			}
			q := p + 3
			var v [3]float32
			for i := 0; i < 3; i++ {
				v[i] = dq(readBits(pay, q, awv), i)
				q += awv
			}
			return v, true
		}
		bestOff, bestN := 0, -1
		for off := lo; off <= hi; off++ {
			nBox := 0
			seen := map[string]bool{}
			for _, b := range stateBits {
				if v, ok := dec(anchor(b) + off); ok && inBox(v) {
					nBox++
					seen[fmt.Sprintf("%.2f_%.2f_%.2f", v[0], v[1], v[2])] = true
				}
			}
			if nBox >= 6 {
				fmt.Printf("  [%s] off=%+d inBox=%d distinct=%d\n", label, off, nBox, len(seen))
			}
			if nBox > bestN {
				bestN, bestOff = nBox, off
			}
		}
		fmt.Printf("  [%s aw=%d] meilleur inBox off=%+d (%d/8)\n", label, awv, bestOff, bestN)
	}
	endAnchor := func(b struct{ slot, stateBit int }) int { return filmdec.BipedDefaultStateEndBit(pay, b.stateBit) }
	stAnchor := func(b struct{ slot, stateBit int }) int { return b.stateBit }
	fmt.Println("-- relatif à endBit (rep) --")
	scan(endAnchor, -120, 120, 13, "endBit")
	scan(endAnchor, -120, 120, 6, "endBit")
	fmt.Println("-- relatif à stateBit --")
	scan(stAnchor, 100, 320, 13, "stateBit")
	scan(stAnchor, 100, 320, 6, "stateBit")
	_ = stAnchor

	// ===== Phase C : positions concrètes aw13 aux offsets candidats =====
	fmt.Println("\n=== Phase C : positions concrètes (aw13 CEBiped) ===")
	dq13 := func(q uint64, axis int) float32 {
		scale := float32(uint64(1) << 13)
		step := (rng[axis].Max - rng[axis].Min) / scale
		return float32(q)*step + rng[axis].Min + step*0.5
	}
	decAt := func(p int) ([3]float32, uint64, uint64, uint64) {
		ph := readBits(pay, p, 1)
		isel := readBits(pay, p+1, 1)
		idx := readBits(pay, p+2, 1)
		q := p + 3
		var v [3]float32
		for i := 0; i < 3; i++ {
			v[i] = dq13(readBits(pay, q, 13), i)
			q += 13
		}
		return v, ph, isel, idx
	}
	// C1 : +182 scanné
	fmt.Println("-- C1 : offset scanné +182 --")
	for _, b := range stateBits {
		p := b.stateBit + 182
		v, ph, isel, idx := decAt(p)
		fmt.Printf("  %d bit=%d ph=%d isel=%d idx=%d pos=(%.2f, %.2f, %.2f) inBox=%v\n",
			b.slot, p, ph, isel, idx, v[0], v[1], v[2], inBox(v))
	}
	// C2 : offset i0 CALCULÉ via le serializer porté (BipedMovementI0Bit = rep + [P2] + masque)
	fmt.Println("-- C2 : offset i0 CALCULÉ (BipedMovementI0Bit : rep + has-comp + masque FUN_1406d7610) --")
	var rows []genRow
	for _, b := range stateBits {
		endBit := filmdec.BipedDefaultStateEndBit(pay, b.stateBit)
		i0, hc, mb := filmdec.BipedMovementI0Bit(pay, b.stateBit)
		v, ph, isel, idx := decAt(i0)
		r := genRow{b.slot, i0, hc, mb, endBit, ph, isel, idx, v, inBox(v)}
		rows = append(rows, r)
		fmt.Printf("  %d endBit=%d hc=%d maskBits=%d i0=%d ph=%d isel=%d idx=%d pos=(%.2f, %.2f, %.2f) inBox=%v\n",
			r.slot, r.endBit, r.hasComp, r.maskBits, r.i0Bit, r.ph, r.isel, r.idx, v[0], v[1], v[2], r.inBox)
	}

	// C3 : hypothèse résidu — rep sous-lit de 8 bits ; has-comp cohérent @endBit+8, masque full,
	// i0 @endBit+8+1+65 = endBit+74. (k=+8 donne has-comp=1 ET mask gate=1 sur 8/8.)
	fmt.Println("-- C3 : i0 = endBit+74 (rep+8 correctif ; has-comp@+8 full-mask) --")
	for _, b := range stateBits {
		endBit := filmdec.BipedDefaultStateEndBit(pay, b.stateBit)
		p := endBit + 74
		v, ph, isel, idx := decAt(p)
		fmt.Printf("  %d i0=%d ph=%d isel=%d idx=%d pos=(%.2f, %.2f, %.2f) inBox=%v\n",
			b.slot, p, ph, isel, idx, v[0], v[1], v[2], inBox(v))
	}
	// balaye finement autour de endBit+74 (aw13) pour capturer un in-box éventuel
	fmt.Println("-- C3b : balayage fin endBit+[66..90] aw13 --")
	for k := 66; k <= 90; k++ {
		nb, nGate := 0, 0
		for _, b := range stateBits {
			endBit := filmdec.BipedDefaultStateEndBit(pay, b.stateBit)
			v, ph, isel, idx := decAt(endBit + k)
			if ph == 0 && isel == 0 && idx == 0 {
				nGate++
				if inBox(v) {
					nb++
				}
			}
		}
		if nGate >= 4 || nb >= 3 {
			fmt.Printf("  k=%+d gate000=%d/8 inBox=%d/8\n", k, nGate, nb)
		}
	}

	// OLD offset (ancien port : R(1) gate à endBit puis precHigh) : compte precHigh=1
	oldPrecHigh1 := 0
	for _, b := range stateBits {
		endBit := filmdec.BipedDefaultStateEndBit(pay, b.stateBit)
		if readBits(pay, endBit+1, 1) == 1 { // ancien : R(1) gate @endBit, precHigh @endBit+1
			oldPrecHigh1++
		}
	}
	fmt.Printf("\nANCIEN offset (endBit) : precHigh=1 sur %d/8 bipeds (dégénéré)\n", oldPrecHigh1)

	nInBox := 0
	for _, r := range rows {
		if r.inBox {
			nInBox++
		}
	}

	// ===== Livrables =====
	scratch := `C:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration/d6f109c3-2822-4e19-8fd4-cf65c50fec3d/scratchpad`
	writeReport(scratch+"/keyframe_spawns2.txt", rows, oldPrecHigh1, nInBox)
	writePNG(scratch+"/keyframe_spawns2.png", rows, [6]float32{box.xlo, box.xhi, box.ylo, box.yhi, box.zlo, box.zhi})
	fmt.Printf("\nÉcrit : %s/keyframe_spawns2.txt et .png\n", scratch)
	fmt.Printf("BILAN : i0 CALCULÉ inBox=%d/8 (Z inclus)\n", nInBox)
}

type genRow struct {
	slot, i0Bit, hasComp, maskBits, endBit int
	ph, isel, idx                          uint64
	v                                      [3]float32
	inBox                                  bool
}

func writeReport(path string, rows []genRow, oldPh, nInBox int) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# keyframe_spawns2 — spawns i0 biped au offset CALCULÉ (serializer movement porté)\n")
	fmt.Fprintf(&sb, "# Film 000d5950/chunk_02, 8 bipeds slots 512-519.\n")
	fmt.Fprintf(&sb, "# Offset i0 = BipedMovementI0Bit = rep FUN_140f44c38 (endBit) + [P2] has-comp R(1) + masque FUN_1406d7610.\n")
	fmt.Fprintf(&sb, "# Déquant : axisW=13, range CEBiped X[-41.10,72.11] Y[-56.61,57.21] Z[-84.37,53.18], step=(Max-Min)/2^13, q*step+Min+step*0.5.\n")
	fmt.Fprintf(&sb, "# Box oracle joueur : x[-6.33,35.70] y[-25.14,27.50] z[-4.20,7.08].\n")
	fmt.Fprintf(&sb, "# ANCIEN offset (endBit brut) : precHigh=1 (vecteur défaut dégénéré) sur %d/8.\n#\n", oldPh)
	fmt.Fprintf(&sb, "# slot endBit hasComp maskBits i0Bit precHigh idxSel idx  x  y  z  inBox\n")
	for _, r := range rows {
		fmt.Fprintf(&sb, "%d %d %d %d %d %d %d %d  %.3f %.3f %.3f  %v\n",
			r.slot, r.endBit, r.hasComp, r.maskBits, r.i0Bit, r.ph, r.isel, r.idx, r.v[0], r.v[1], r.v[2], r.inBox)
	}
	fmt.Fprintf(&sb, "#\n# BILAN : inBox=%d/8 (Z inclus). success=%v\n", nInBox, nInBox == 8)
	_ = os.WriteFile(path, []byte(sb.String()), 0o644)
}

func writePNG(path string, rows []genRow, box [6]float32) {
	const W, H = 640, 480
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	for i := range img.Pix {
		img.Pix[i] = 0xff
	}
	rng := filmdec.QuantRangeCEBiped
	// mapping X->px, Y->py sur toute la range CEBiped X/Y (pour voir où tombent les points vs box)
	mapX := func(x float32) int { return int((x - rng[0].Min) / (rng[0].Max - rng[0].Min) * (W - 40)) }
	mapY := func(y float32) int { return H - 40 - int((y-rng[1].Min)/(rng[1].Max-rng[1].Min)*(H-60)) }
	put := func(px, py int, c color.RGBA) {
		px += 20
		if px < 0 || px >= W || py < 0 || py >= H {
			return
		}
		o := py*img.Stride + px*4
		img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = c.R, c.G, c.B, 0xff
	}
	// box oracle (rectangle bleu)
	blue := color.RGBA{40, 90, 220, 255}
	bx0, bx1, by0, by1 := mapX(box[0]), mapX(box[1]), mapY(box[3]), mapY(box[2])
	for x := bx0; x <= bx1; x++ {
		put(x, by0, blue)
		put(x, by1, blue)
	}
	for y := by0; y <= by1; y++ {
		put(bx0, y, blue)
		put(bx1, y, blue)
	}
	// points i0 (rouge, 3x3)
	red := color.RGBA{220, 30, 30, 255}
	for _, r := range rows {
		px, py := mapX(r.v[0]), mapY(r.v[1])
		for dx := -2; dx <= 2; dx++ {
			for dy := -2; dy <= 2; dy++ {
				put(px+dx, py+dy, red)
			}
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	_ = png.Encode(f, img)
}
