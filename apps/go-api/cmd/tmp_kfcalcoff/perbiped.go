package main

import "fmt"

// perBipedScan : pour chaque biped, liste les offsets absolus (depuis stateBit) dans
// [120,360] où i0 décode en position IN-MAP (gate 0,0,0) ET IN-BOX (boîte oracle), width 13,
// range CE. But : voir si un offset commun (relatif au endBit calculé) émerge, ou si chaque
// record a besoin d'un offset différent (=> grammaire à largeur variable non fermée).
func perBipedScan(pay []byte, bislot, stateBits, endBits []int) {
	ce := [3][2]float32{{-41.10318, 72.10963}, {-56.60697, 57.212566}, {-84.37078, 53.18034}}
	dec := func(p, w int) (pos [3]float32, inMap bool) {
		if readBits(pay, p, 1) != 0 {
			return pos, false
		}
		q := p + 1
		idx := -1
		if readBits(pay, q, 1) == 0 {
			idx = int(readBits(pay, q+1, 1))
			q += 2
		} else {
			q++
		}
		scale := float32(uint64(1) << uint(w))
		for i := 0; i < 3; i++ {
			wv := readBits(pay, q, w)
			q += w
			step := (ce[i][1] - ce[i][0]) / scale
			pos[i] = float32(wv)*step + ce[i][0] + step*0.5
		}
		return pos, idx == 0
	}
	inB := func(v [3]float32) bool {
		for i := 0; i < 3; i++ {
			if v[i] < oracleBox[i][0] || v[i] > oracleBox[i][1] {
				return false
			}
		}
		return true
	}
	fmt.Println("\n=== 6. PER-BIPED : offsets absolus (rel. stateBit) donnant in-map & in-box (w=13, CE) ===")
	// pour chaque biped, on collecte l'ensemble des Δ (rel. endBit) qui marchent
	perBipedDeltas := make([]map[int]bool, len(bislot))
	for i := range bislot {
		sb, eb := stateBits[i], endBits[i]
		relEnd := eb - sb
		hitsRel := []int{}
		deltas := map[int]bool{}
		for off := 120; off <= 360; off++ {
			pos, m := dec(sb+off, 13)
			if m && inB(pos) {
				hitsRel = append(hitsRel, off)
				deltas[off-relEnd] = true
			}
		}
		perBipedDeltas[i] = deltas
		fmt.Printf("  slot=%d relEnd=+%d  #hits=%d offs(rel.stateBit)=%v\n", bislot[i], relEnd, len(hitsRel), hitsRel)
	}
	// intersection des Δ (rel endBit) communs à tous les bipeds
	if len(perBipedDeltas) > 0 {
		common := map[int]bool{}
		for d := range perBipedDeltas[0] {
			common[d] = true
		}
		for _, m := range perBipedDeltas[1:] {
			for d := range common {
				if !m[d] {
					delete(common, d)
				}
			}
		}
		fmt.Printf("  Δ (rel. endBit) communs aux 8 bipeds pour in-map&in-box : %v\n", keysOf(common))
	}
}

func keysOf(m map[int]bool) []int {
	out := []int{}
	for k := range m {
		out = append(out, k)
	}
	return out
}
