package main

import "fmt"

// comprehensiveSearch cherche exhaustivement un (Δ relatif au endBit calculé, width, range)
// donnant 8 spawns distincts NON-dégénérés in-box. range=CE (in-box = sous-boîte oracle) OU
// range=box (in-box trivial -> juge = étalement non-dégénéré + distinct). Gate in-map requis.
func comprehensiveSearch(pay []byte, endBits []int) {
	type rng3 [3][2]float32
	ce := rng3{{-41.10318, 72.10963}, {-56.60697, 57.212566}, {-84.37078, 53.18034}}
	box := rng3(oracleBox)

	dequant := func(p, w int, r rng3, lead bool) (pos [3]float32, inMap bool) {
		if lead {
			p++
		}
		if readBits(pay, p, 1) != 0 { // precHigh
			return pos, false
		}
		q := p + 1
		idx := -1
		if readBits(pay, q, 1) == 0 { // idxSel==0 -> lit index (indexW=1)
			idx = int(readBits(pay, q+1, 1))
			q += 2
		} else {
			q++
		}
		scale := float32(uint64(1) << uint(w))
		for i := 0; i < 3; i++ {
			wv := readBits(pay, q, w)
			q += w
			step := (r[i][1] - r[i][0]) / scale
			pos[i] = float32(wv)*step + r[i][0] + step*0.5
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
	// axisSpread : fraction de la boîte oracle couverte par l'étalement des 8 valeurs d'un axe,
	// + nb de valeurs distinctes par axe. Non-dégénéré = spread élevé sur les 3 axes.
	score := func(pts [][3]float32) (dist int, spreadFrac [3]float32, perAxisDistinct [3]int) {
		seen := map[string]bool{}
		var mn, mx [3]float32
		for i := 0; i < 3; i++ {
			mn[i], mx[i] = pts[0][i], pts[0][i]
		}
		axSeen := [3]map[float32]bool{{}, {}, {}}
		for _, v := range pts {
			seen[fmt.Sprintf("%.3f_%.3f_%.3f", v[0], v[1], v[2])] = true
			for i := 0; i < 3; i++ {
				if v[i] < mn[i] {
					mn[i] = v[i]
				}
				if v[i] > mx[i] {
					mx[i] = v[i]
				}
				axSeen[i][v[i]] = true
			}
		}
		for i := 0; i < 3; i++ {
			spreadFrac[i] = (mx[i] - mn[i]) / (oracleBox[i][1] - oracleBox[i][0])
			perAxisDistinct[i] = len(axSeen[i])
		}
		return len(seen), spreadFrac, perAxisDistinct
	}

	fmt.Println("\n=== 5. RECHERCHE EXHAUSTIVE (per-biped endBit+Δ, width, range, lead) ===")
	type hit struct {
		rlab       string
		lead       bool
		d, w, nMap int
		nBox, dist int
		sf         [3]float32
		pad        [3]int
	}
	var hits []hit
	for _, rc := range []struct {
		lab string
		r   rng3
	}{{"CE", ce}, {"BOX", box}} {
		for _, lead := range []bool{false, true} {
			for _, w := range []int{6, 10, 11, 12, 13, 14, 15, 16} {
				for d := -40; d <= 56; d++ {
					pts := make([][3]float32, 0, 8)
					nMap, nBox := 0, 0
					for _, e := range endBits {
						pos, m := dequant(e+d, w, rc.r, lead)
						pts = append(pts, pos)
						if m {
							nMap++
							if inB(pos) {
								nBox++
							}
						}
					}
					dist, sf, pad := score(pts)
					// non-dégénéré : 3 axes avec >=6 valeurs distinctes ET spread >= 0.30 de la boîte
					nonDeg := pad[0] >= 6 && pad[1] >= 6 && pad[2] >= 6 &&
						sf[0] >= 0.30 && sf[1] >= 0.30 && sf[2] >= 0.30
					pass := nMap >= 7 && dist >= 7 && nonDeg
					if rc.lab == "CE" {
						pass = pass && nBox >= 7
					}
					if pass {
						hits = append(hits, hit{rc.lab, lead, d, w, nMap, nBox, dist, sf, pad})
					}
				}
			}
		}
	}
	if len(hits) == 0 {
		fmt.Println("  AUCUN combo (Δ,width,range,lead) ne donne 8 distincts non-dégénérés in-box in-map.")
		return
	}
	for _, h := range hits {
		fmt.Printf("  * range=%s lead=%v Δ=%+d w=%d inMap=%d/8 inBox=%d/8 dist=%d spread=[%.2f,%.2f,%.2f] axDist=%v\n",
			h.rlab, h.lead, h.d, h.w, h.nMap, h.nBox, h.dist, h.sf[0], h.sf[1], h.sf[2], h.pad)
	}
}
