package main

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"levelup/go-api/internal/analysis/filmdec"
)

const scratch = `C:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration/d6f109c3-2822-4e19-8fd4-cf65c50fec3d/scratchpad`

// emitDeliverables écrit keyframe_spawns.txt + un PNG scatter XY (positions décodées au
// endBit CALCULÉ, width 13, range CE) avec le rectangle de la boîte oracle superposé.
func emitDeliverables(pay []byte, bislot, stateBits, endBits []int) {
	type row struct {
		slot, off, gp, gi, idx int
		pos                    [3]float32
		inMap, box             bool
	}
	rows := make([]row, 0, 8)
	for i := range bislot {
		pos, inMap, _, gp, gi, idx := decodeI0(pay, endBits[i], 1, 13, false)
		rows = append(rows, row{bislot[i], endBits[i] - stateBits[i], gp, gi, idx, pos, inMap, inBox(pos)})
	}

	// candidat fixe rel. stateBit : off+303 w13 = 8/8 in-map avec meilleur etalement X/Y.
	dec303 := func(p int) (pos [3]float32, inMap bool) {
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
		rng := filmdec.WorldPositionRange
		scale := float32(uint64(1) << 13)
		for i := 0; i < 3; i++ {
			wv := readBits(pay, q, 13)
			q += 13
			step := (rng[i].Max - rng[i].Min) / scale
			pos[i] = float32(wv)*step + rng[i].Min + step*0.5
		}
		return pos, idx == 0
	}

	f, _ := os.Create(scratch + "/keyframe_spawns.txt")
	w := bufio.NewWriter(f)
	fmt.Fprintln(w, "# Spawns keyframe biped (film 000d5950, chunk_02, 8 joueurs slots 512-519).")
	fmt.Fprintln(w, "# GRAMMAIRE R1 FERMEE : FUN_1406d84b4 R(14) ajoute dans la branche G3 du bloc variant")
	fmt.Fprintln(w, "#   FUN_14080cfe8 (default_state.go). Le default-state complet est porte, data-driven :")
	fmt.Fprintln(w, "#   R1=152 bits (slot512) .. 184 bits (slot519) ; total default-state=224..256 bits.")
	fmt.Fprintln(w, "# Offset i0 = CALCULE via filmdec.BipedDefaultStateEndBit (fin de FUN_140f44c38).")
	fmt.Fprintln(w, "# Decode i0 = consumeAbsoluteWithGate (precHigh R1 ; idxSel R1 ; idx R1 ; 3xR(13)) + range CE + centre 0.5.")
	fmt.Fprintf(w, "# range CE = X[%.3f,%.3f] Y[%.3f,%.3f] Z[%.3f,%.3f]\n",
		filmdec.WorldPositionRange[0].Min, filmdec.WorldPositionRange[0].Max,
		filmdec.WorldPositionRange[1].Min, filmdec.WorldPositionRange[1].Max,
		filmdec.WorldPositionRange[2].Min, filmdec.WorldPositionRange[2].Max)
	fmt.Fprintln(w, "# boite oracle = X[-6.33,35.70] Y[-25.14,27.50] Z[-4.20,7.08]")
	fmt.Fprintln(w, "#")
	fmt.Fprintln(w, "# RESULTAT (HONNETE, success=false) :")
	fmt.Fprintln(w, "#  A) au endBit CALCULE, 5/8 bipeds lisent precHigh=1 (vecteur defaut 0,0,0), 2/8 idxSel=1 (off-map),")
	fmt.Fprintln(w, "#     seuls 2/8 in-map, AUCUN in-box. => endBit ne pointe PAS sur le gate i0 (grammaire non bit-exacte).")
	fmt.Fprintln(w, "#  B) recherche exhaustive (offset rel.stateBit 100..700 ET rel.endBit -60..+200 ; widths 6-16 ;")
	fmt.Fprintln(w, "#     dequant range CE + centered-quantum ; lead on/off ; per-biped) : DES offsets FIXES rel.stateBit")
	fmt.Fprintln(w, "#     (+272..+274, +303..+304) donnent 8/8 in-map avec etalement 3 axes (vraies positions candidates),")
	fmt.Fprintln(w, "#     MAIS l'axe Z ne rentre JAMAIS dans [-4.20,7.08] pour les 8 (Z decode ~ -84..+2, jamais la bande box).")
	fmt.Fprintln(w, "#     => Z est le bloqueur systematique : soit la range Z / l'encodage differe, soit la boite oracle Z")
	fmt.Fprintln(w, "#     ne correspond pas a ces spawns. Positions NON reconstructibles in-box avec (endBit calcule, w13, range CE).")
	fmt.Fprintln(w, "#")
	fmt.Fprintln(w, "# TABLE 1 : decode au endBit CALCULE (par tache).")
	fmt.Fprintln(w, "# colonnes : slot off(rel.stateBit) gate(prec,idxSel,idx) inMap x y z inBox")
	for _, r := range rows {
		fmt.Fprintf(w, "%d %d %d %d %d %v %.3f %.3f %.3f %v\n",
			r.slot, r.off, r.gp, r.gi, r.idx, r.inMap, r.pos[0], r.pos[1], r.pos[2], r.box)
	}
	fmt.Fprintln(w, "#")
	fmt.Fprintln(w, "# TABLE 2 : meilleur candidat FIXE off=+303 rel.stateBit w13 (8/8 in-map, etalement 3 axes) — DIAGNOSTIC, Z hors box.")
	fmt.Fprintln(w, "# colonnes : slot x y z inMap inBox")
	for i, sb := range stateBits {
		pos, m := dec303(sb + 303)
		fmt.Fprintf(w, "%d %.3f %.3f %.3f %v %v\n", bislot[i], pos[0], pos[1], pos[2], m, inBox(pos))
	}
	w.Flush()
	f.Close()
	fmt.Printf("\necrit %s/keyframe_spawns.txt\n", scratch)

	// PNG scatter XY : boite oracle (rect vert) + positions decodees (rouge=hors-box, bleu=in-box).
	const W, H = 640, 480
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			img.Set(x, y, color.RGBA{20, 22, 28, 255})
		}
	}
	// echelle : range CE X/Y -> pixels (marge 40)
	xr := filmdec.WorldPositionRange[0]
	yr := filmdec.WorldPositionRange[1]
	px := func(vx float32) int { return 40 + int(float32(W-80)*(vx-xr.Min)/(xr.Max-xr.Min)) }
	py := func(vy float32) int { return H - 40 - int(float32(H-80)*(vy-yr.Min)/(yr.Max-yr.Min)) }
	drawRect := func(x0, y0, x1, y1 int, c color.RGBA) {
		if x0 > x1 {
			x0, x1 = x1, x0
		}
		if y0 > y1 {
			y0, y1 = y1, y0
		}
		for x := x0; x <= x1; x++ {
			img.Set(x, y0, c)
			img.Set(x, y1, c)
		}
		for y := y0; y <= y1; y++ {
			img.Set(x0, y, c)
			img.Set(x1, y, c)
		}
	}
	// boite oracle (projection XY)
	drawRect(px(-6.33), py(-25.14), px(35.70), py(27.50), color.RGBA{60, 200, 90, 255})
	// points : candidat off+303 w13 (8/8 in-map, meilleur etalement) ; bleu = Z in-box, rouge = Z hors-box.
	for i, sb := range stateBits {
		pos, _ := dec303(sb + 303)
		c := color.RGBA{230, 70, 70, 255}
		if pos[2] >= oracleBox[2][0] && pos[2] <= oracleBox[2][1] {
			c = color.RGBA{80, 140, 240, 255}
		}
		cx, cy := px(pos[0]), py(pos[1])
		for dy := -3; dy <= 3; dy++ {
			for dx := -3; dx <= 3; dx++ {
				x, y := cx+dx, cy+dy
				if x >= 0 && x < W && y >= 0 && y < H {
					img.Set(x, y, c)
				}
			}
		}
		_ = i
	}
	pf, _ := os.Create(scratch + "/keyframe_spawns.png")
	png.Encode(pf, img)
	pf.Close()
	fmt.Printf("ecrit %s/keyframe_spawns.png (rect vert=boite oracle, rouge=hors-box)\n", scratch)
}
