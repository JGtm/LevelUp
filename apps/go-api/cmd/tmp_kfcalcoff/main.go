// tmp_kfcalcoff — THROWAWAY. Décode la position de spawn i0 des 8 bipeds au keyframe
// en localisant le début d'i0 par CALCUL (filmdec.BipedDefaultStateEndBit = fin de la
// grammaire FUN_140f44c38 portée, R1 fermé) et NON par scan de consensus. Décode ensuite
// la position movement i0 = consumeAbsoluteWithGate + range CE + width 13, puis valide les
// 8 spawns contre la boîte oracle joueur x[-6..36] y[-25..27] z[-4..7].
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

// stateBits des 8 bipeds (chunk_02, hdrBit+64), depuis biped_record_offsets.txt.
var bipeds = []struct{ slot, stateBit int }{
	{512, 193467}, {513, 196252}, {514, 199068}, {515, 201862},
	{516, 204665}, {517, 207460}, {518, 210262}, {519, 213057},
}

// Boîte oracle joueur (spawns réels attendus).
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

// decodeI0 : à partir du bit `p` (début i0 movement), consomme optionnellement un
// rangeSel R(1) (FUN_14076e420) puis consumeAbsoluteWithGate : precHigh R(1) ;
// si 0 idxSel R(1) ; si 0 idx R(indexW) ; 3×R(axisW). Déquant range CE + centre 0.5.
// Retourne pos, inMap (gates 0,0,0), endBit, et les bits de gate bruts.
func decodeI0(pay []byte, p, indexW, axisW int, lead bool) (pos [3]float32, inMap bool, endBit, gPrec, gIdxSel, idx int) {
	if lead {
		p++ // rangeSel R(1)
	}
	gPrec = int(readBits(pay, p, 1))
	if gPrec != 0 { // precHigh=1 -> vecteur défaut, 0 bit
		return pos, false, p + 1, gPrec, -1, -1
	}
	gIdxSel = int(readBits(pay, p+1, 1))
	q := p + 2
	if gIdxSel == 0 { // lit l'index
		idx = int(readBits(pay, q, indexW))
		q += indexW
	} else {
		idx = -1
	}
	rng := filmdec.WorldPositionRange
	scale := float32(uint64(1) << uint(axisW))
	for i := 0; i < 3; i++ {
		w := readBits(pay, q, axisW)
		q += axisW
		step := (rng[i].Max - rng[i].Min) / scale
		pos[i] = float32(w)*step + rng[i].Min + step*0.5
	}
	inMap = gIdxSel == 0 && idx == 0
	return pos, inMap, q, gPrec, gIdxSel, idx
}

func inBox(v [3]float32) bool {
	for i := 0; i < 3; i++ {
		if v[i] < oracleBox[i][0] || v[i] > oracleBox[i][1] {
			return false
		}
	}
	return true
}

func finite(v float32) bool { return !math.IsNaN(float64(v)) && !math.IsInf(float64(v), 0) }

func spread(vals []float32) (min, max float32) {
	min, max = vals[0], vals[0]
	for _, v := range vals {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return
}

func main() {
	pay := framePayload(inflate(defFilm+"/chunk_02.bin"), 2)
	if pay == nil {
		fmt.Println("chunk_02 type-2 introuvable")
		return
	}
	fmt.Printf("payload=%d o (%d bits) ; range CE=%v\n", len(pay), len(pay)*8, filmdec.WorldPositionRange)
	fmt.Printf("boîte oracle x[%.2f,%.2f] y[%.2f,%.2f] z[%.2f,%.2f]\n\n",
		oracleBox[0][0], oracleBox[0][1], oracleBox[1][0], oracleBox[1][1], oracleBox[2][0], oracleBox[2][1])

	// ===== 1. Offset i0 CALCULÉ via la grammaire portée =====
	fmt.Println("=== 1. OFFSET i0 CALCULÉ (BipedDefaultStateEndBit) ===")
	type bi struct {
		slot, stateBit, endBit, off int
	}
	bis := make([]bi, 0, 8)
	for _, b := range bipeds {
		e := filmdec.BipedDefaultStateEndBit(pay, b.stateBit)
		bis = append(bis, bi{b.slot, b.stateBit, e, e - b.stateBit})
		fmt.Printf("  slot=%d stateBit=%d endBit=%d off=+%d\n", b.slot, b.stateBit, e, e-b.stateBit)
	}

	// ===== 2. Décodage i0 aux deux variantes (lead rangeSel off/on), width 13 =====
	for _, lead := range []bool{false, true} {
		fmt.Printf("\n=== 2. DÉCODE i0 @ endBit  (leadRangeSel=%v, indexW=1, axisW=13, range CE) ===\n", lead)
		pts := map[int][3]float32{}
		var xs, ys, zs []float32
		nInMap, nInBox := 0, 0
		for _, b := range bis {
			pos, inMap, endB, gp, gi, idx := decodeI0(pay, b.endBit, 1, 13, lead)
			pts[b.slot] = pos
			box := inBox(pos)
			if inMap {
				nInMap++
			}
			if box {
				nInBox++
			}
			xs, ys, zs = append(xs, pos[0]), append(ys, pos[1]), append(zs, pos[2])
			fmt.Printf("  slot=%d off=+%-4d gate(prec=%d idxSel=%d idx=%d) inMap=%-5v pos=(%7.2f,%7.2f,%7.2f) inBox=%-5v endB=%d\n",
				b.slot, b.off, gp, gi, idx, inMap, pos[0], pos[1], pos[2], box, endB)
		}
		// distinct
		seen := map[string]bool{}
		for _, v := range pts {
			seen[fmt.Sprintf("%.3f_%.3f_%.3f", v[0], v[1], v[2])] = true
		}
		xmn, xmx := spread(xs)
		ymn, ymx := spread(ys)
		zmn, zmx := spread(zs)
		fmt.Printf("  --> distinct=%d/8 inMap=%d/8 inBox=%d/8 ; étalement X[%.1f,%.1f] Y[%.1f,%.1f] Z[%.1f,%.1f]\n",
			len(seen), nInMap, nInBox, xmn, xmx, ymn, ymx, zmn, zmx)
	}

	// ===== 3. Balayage 2D (Δ autour de endBit, width) : gate in-map + boîte oracle =====
	fmt.Println("\n=== 3. BALAYAGE 2D (Δ, width) : combos avec inMap>=7 ET inBox>=6 (lead off) ===")
	type combo struct {
		d, w, nMap, nBox, dist int
	}
	best := []combo{}
	for _, w := range []int{6, 12, 13, 14, 15, 16} {
		for d := -32; d <= 40; d++ {
			nBox, nMap := 0, 0
			seen := map[string]bool{}
			for _, b := range bis {
				pos, inMap, _, _, _, _ := decodeI0(pay, b.endBit+d, 1, w, false)
				if inMap {
					nMap++
				}
				if inMap && inBox(pos) { // in-box ne compte QUE si le gate est in-map (pas les 0,0,0)
					nBox++
				}
				seen[fmt.Sprintf("%.2f_%.2f_%.2f", pos[0], pos[1], pos[2])] = true
			}
			if nMap >= 7 && nBox >= 6 {
				best = append(best, combo{d, w, nMap, nBox, len(seen)})
			}
		}
	}
	if len(best) == 0 {
		fmt.Println("  aucun combo inMap>=7 & inBox>=6. Élargissement : inMap>=6 & inBox>=5 :")
		for _, w := range []int{6, 12, 13, 14, 15, 16} {
			for d := -32; d <= 40; d++ {
				nBox, nMap := 0, 0
				seen := map[string]bool{}
				for _, b := range bis {
					pos, inMap, _, _, _, _ := decodeI0(pay, b.endBit+d, 1, w, false)
					if inMap {
						nMap++
						if inBox(pos) {
							nBox++
						}
					}
					seen[fmt.Sprintf("%.2f_%.2f_%.2f", pos[0], pos[1], pos[2])] = true
				}
				if nMap >= 6 && nBox >= 5 {
					fmt.Printf("  Δ=%+d w=%d inMap=%d/8 inBox=%d/8 distinct=%d\n", d, w, nMap, nBox, len(seen))
				}
			}
		}
	}
	for _, c := range best {
		fmt.Printf("  * Δ=%+d w=%d inMap=%d/8 inBox=%d/8 distinct=%d\n", c.d, c.w, c.nMap, c.nBox, c.dist)
	}

	// ===== 4. positions détaillées au meilleur combo in-map (Δ=+16 candidat) pour width 13 & 14 =====
	for _, tc := range []struct {
		d, w int
	}{{16, 13}, {16, 14}, {17, 14}} {
		fmt.Printf("\n=== 4. POSITIONS @ Δ=%+d width=%d ===\n", tc.d, tc.w)
		for _, b := range bis {
			pos, inMap, _, gp, gi, idx := decodeI0(pay, b.endBit+tc.d, 1, tc.w, false)
			fmt.Printf("  slot=%d gate(prec=%d idxSel=%d idx=%d) inMap=%-5v pos=(%7.2f,%7.2f,%7.2f) inBox=%v\n",
				b.slot, gp, gi, idx, inMap, pos[0], pos[1], pos[2], inBox(pos))
		}
	}
	endBits := make([]int, 0, 8)
	for _, b := range bis {
		endBits = append(endBits, b.endBit)
	}
	comprehensiveSearch(pay, endBits)
	slots := make([]int, 0, 8)
	sbs := make([]int, 0, 8)
	for _, b := range bis {
		slots = append(slots, b.slot)
		sbs = append(sbs, b.stateBit)
	}
	perBipedScan(pay, slots, sbs, endBits)
	centeredScan(pay, endBits)
	traceGrammar(pay, 193467)
	traceGrammar(pay, 213057)
	definitiveScan(pay, sbs, endBits)
	emitDeliverables(pay, slots, sbs, endBits)
	_ = sort.Ints
	_ = finite
}
