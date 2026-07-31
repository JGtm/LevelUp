package main

import "fmt"

// centeredScan : teste le dequant CENTERED-QUANTUM (q-2^(w-1))*Q, centré sur 0, qui couvre
// ±2^(w-1)·Q. Q=0.01383. À w=13 -> ±56.6 ; w=12 -> ±28.3 ; w=11 -> ±14.2. La boîte oracle
// (X[-6,36] Y[-25,27] Z[-4,7]) est ~centrée. Balaye (Δ rel endBit, w, Q) -> 8 in-box distincts.
func centeredScan(pay []byte, endBits []int) {
	const Q = 0.01383
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
		half := float32(uint64(1) << uint(w-1))
		for i := 0; i < 3; i++ {
			wv := readBits(pay, q, w)
			q += w
			pos[i] = (float32(wv) - half) * Q
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
	fmt.Println("\n=== 7. CENTERED-QUANTUM (q-2^(w-1))*0.01383 : balayage (Δ rel endBit, w) ===")
	best := struct {
		d, w, nMap, nBox, dist int
	}{}
	for _, w := range []int{11, 12, 13, 14} {
		for d := -40; d <= 56; d++ {
			nMap, nBox := 0, 0
			seen := map[string]bool{}
			pts := [][3]float32{}
			for _, e := range endBits {
				pos, m := dec(e+d, w)
				pts = append(pts, pos)
				if m {
					nMap++
					if inB(pos) {
						nBox++
					}
				}
				seen[fmt.Sprintf("%.2f_%.2f_%.2f", pos[0], pos[1], pos[2])] = true
			}
			if nMap >= 7 && nBox >= 6 {
				fmt.Printf("  Δ=%+d w=%d inMap=%d/8 inBox=%d/8 distinct=%d\n", d, w, nMap, nBox, len(seen))
			}
			if nBox > best.nBox {
				best.d, best.w, best.nMap, best.nBox, best.dist = d, w, nMap, nBox, len(seen)
			}
		}
	}
	fmt.Printf("  meilleur (par inBox) : Δ=%+d w=%d inMap=%d/8 inBox=%d/8 distinct=%d\n",
		best.d, best.w, best.nMap, best.nBox, best.dist)
	// per-biped reachability : chaque biped a-t-il UN offset in-box (indép.) en centered w=13 ?
	fmt.Println("  per-biped (centered w=13, scan Δ rel endBit -40..+56) : #Δ in-map&in-box :")
	for bi, e := range endBits {
		n := 0
		var sample []int
		for d := -40; d <= 56; d++ {
			pos, m := dec(e+d, 13)
			if m && inB(pos) {
				n++
				if len(sample) < 6 {
					sample = append(sample, d)
				}
			}
		}
		fmt.Printf("    biped#%d endBit=%d : #%d Δ=%v\n", bi, e, n, sample)
	}
}
