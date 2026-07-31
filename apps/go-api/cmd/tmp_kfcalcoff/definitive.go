package main

import "fmt"

// definitiveScan : cherche N'IMPORTE OU (offset rel. stateBit 100..700, OU Δ rel. endBit
// -60..+200) un offset ou les 8 bipeds ont gate in-map 8/8 ET un etalement reel sur les 3
// axes (>=6 valeurs distinctes/axe). C'est la signature de 8 vraies positions joueur. width 13,
// range CE. Si rien -> la position n'est pas reconstructible ainsi (honnetete).
func definitiveScan(pay []byte, stateBits, endBits []int) {
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
	eval := func(bases []int, off, w int) (nMap int, axDist [3]int) {
		axSeen := [3]map[float32]bool{{}, {}, {}}
		for _, b := range bases {
			pos, m := dec(b+off, w)
			if m {
				nMap++
				for i := 0; i < 3; i++ {
					axSeen[i][pos[i]] = true
				}
			}
		}
		for i := 0; i < 3; i++ {
			axDist[i] = len(axSeen[i])
		}
		return
	}
	fmt.Println("\n=== 9. SCAN DÉFINITIF : offset avec in-map 8/8 ET 3 axes >=6 distincts (vraies positions) ===")
	found := 0
	// relatif au stateBit
	for off := 100; off <= 700; off++ {
		for _, w := range []int{6, 12, 13, 14} {
			nMap, ad := eval(stateBits, off, w)
			if nMap == 8 && ad[0] >= 6 && ad[1] >= 6 && ad[2] >= 6 {
				fmt.Printf("  [rel stateBit] off=+%d w=%d inMap=8/8 axDist=%v\n", off, w, ad)
				found++
			}
		}
	}
	// relatif au endBit calculé
	for d := -60; d <= 200; d++ {
		for _, w := range []int{6, 12, 13, 14} {
			nMap, ad := eval(endBits, d, w)
			if nMap == 8 && ad[0] >= 6 && ad[1] >= 6 && ad[2] >= 6 {
				fmt.Printf("  [rel endBit] Δ=%+d w=%d inMap=8/8 axDist=%v\n", d, w, ad)
				found++
			}
		}
	}
	if found == 0 {
		fmt.Println("  AUCUN offset (rel stateBit 100..700 ni rel endBit -60..+200, w∈{6,12,13,14})")
		fmt.Println("  ne donne 8/8 in-map avec 3 axes etales. => positions non reconstructibles ainsi.")
	}
	// détail in-box aux candidats fixes rel. stateBit
	inB := func(v [3]float32) bool {
		for i := 0; i < 3; i++ {
			if v[i] < oracleBox[i][0] || v[i] > oracleBox[i][1] {
				return false
			}
		}
		return true
	}
	for _, tc := range []struct{ off, w int }{{272, 13}, {274, 13}, {303, 13}, {303, 12}, {274, 6}} {
		fmt.Printf("\n  -- DÉTAIL off=+%d(rel stateBit) w=%d --\n", tc.off, tc.w)
		nBox := 0
		for i, sb := range stateBits {
			pos, m := dec(sb+tc.off, tc.w)
			b := inB(pos)
			if b {
				nBox++
			}
			fmt.Printf("    biped#%d pos=(%7.2f,%7.2f,%7.2f) inMap=%-5v inBox=%v\n", i, pos[0], pos[1], pos[2], m, b)
		}
		fmt.Printf("    -> inBox=%d/8\n", nBox)
	}
}
