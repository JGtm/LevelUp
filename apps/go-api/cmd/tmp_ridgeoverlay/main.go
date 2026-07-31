// tmp_ridgeoverlay — superpose les trajectoires PROPRES (film) à l'empreinte 2D de
// ridgeline reconstruite depuis le .module, et mesure le recouvrement.
//
// Deux hypothèses testées, dans cet ordre :
//  1. ABSOLU  — repère film == repère monde du module (identité, ZÉRO paramètre ajusté).
//  2. NORMALISÉ — chaque nuage ramené à [0,1]^2 + meilleure des 8 orientations
//     (méthode tmp_mapoverlay/catalyst). C'est un PLACAGE : le score n'a pas la même
//     valeur de preuve que l'identité.
//
// Usage : go run ./cmd/tmp_ridgeoverlay <meshes.csv> <traj_clean.csv> <out.png>
package main

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

type box struct{ lo, hi [3]float64 }
type pt struct {
	slot    int
	x, y, z float64
}

func readMeshes(path string) []box {
	f, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	sc.Scan()
	var out []box
	for sc.Scan() {
		c := strings.Split(sc.Text(), ",")
		if len(c) < 11 {
			continue
		}
		var b box
		for a := 0; a < 3; a++ {
			b.lo[a], _ = strconv.ParseFloat(c[5+a], 64)
			b.hi[a], _ = strconv.ParseFloat(c[8+a], 64)
		}
		out = append(out, b)
	}
	return out
}

func readTraj(path string) []pt {
	f, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	sc.Scan()
	var out []pt
	for sc.Scan() {
		c := strings.Split(sc.Text(), ",")
		if len(c) < 7 {
			continue
		}
		var p pt
		p.slot, _ = strconv.Atoi(c[0])
		p.x, _ = strconv.ParseFloat(c[4], 64)
		p.y, _ = strconv.ParseFloat(c[5], 64)
		p.z, _ = strconv.ParseFloat(c[6], 64)
		out = append(out, p)
	}
	return out
}

func q(v []float64, p float64) float64 {
	if len(v) == 0 {
		return math.NaN()
	}
	i := int(p * float64(len(v)-1))
	return v[i]
}

// ---------------- grille d'occupation monde ----------------

// occGrid rasterise l'empreinte XY (Z = hauteur) des bbox, en coordonnées MONDE.
type occGrid struct {
	x0, y0, cell float64
	w, h         int
	c            []int
}

func newOcc(x0, y0, x1, y1, cell float64) *occGrid {
	w := int((x1-x0)/cell) + 2
	h := int((y1-y0)/cell) + 2
	return &occGrid{x0: x0, y0: y0, cell: cell, w: w, h: h, c: make([]int, w*h)}
}

func (g *occGrid) fill(b box) {
	i0 := int((b.lo[0] - g.x0) / g.cell)
	i1 := int((b.hi[0] - g.x0) / g.cell)
	j0 := int((b.lo[1] - g.y0) / g.cell)
	j1 := int((b.hi[1] - g.y0) / g.cell)
	for j := j0; j <= j1; j++ {
		for i := i0; i <= i1; i++ {
			if i >= 0 && i < g.w && j >= 0 && j < g.h {
				g.c[j*g.w+i]++
			}
		}
	}
}

func (g *occGrid) at(x, y float64) int {
	i := int((x - g.x0) / g.cell)
	j := int((y - g.y0) / g.cell)
	if i < 0 || i >= g.w || j < 0 || j >= g.h {
		return 0
	}
	return g.c[j*g.w+i]
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("usage: tmp_ridgeoverlay <meshes.csv> <traj_clean.csv> <out.png>")
		return
	}
	meshes := readMeshes(os.Args[1])
	traj := readTraj(os.Args[2])
	fmt.Printf("%d meshes, %d points de trajectoire\n\n", len(meshes), len(traj))
	if len(meshes) == 0 || len(traj) == 0 {
		return
	}

	// --- 1. stats meshes : distribution des tailles pour choisir le filtre skybox ---
	var ext []float64
	for _, m := range meshes {
		e := 0.0
		for a := 0; a < 3; a++ {
			e = math.Max(e, m.hi[a]-m.lo[a])
		}
		ext = append(ext, e)
	}
	sort.Float64s(ext)
	fmt.Printf("extension max des meshes (wu) : p10=%.1f p50=%.1f p75=%.1f p90=%.1f p95=%.1f p99=%.1f max=%.1f\n",
		q(ext, .1), q(ext, .5), q(ext, .75), q(ext, .9), q(ext, .95), q(ext, .99), ext[len(ext)-1])

	// filtre : on garde les meshes de taille "jouable" (< 60 wu) → exclut skybox/décor lointain
	const maxExt = 60.0
	var play []box
	for _, m := range meshes {
		e := 0.0
		for a := 0; a < 3; a++ {
			e = math.Max(e, m.hi[a]-m.lo[a])
		}
		if e < maxExt {
			play = append(play, m)
		}
	}
	bounds := func(bs []box) (lo, hi [3]float64) {
		for a := 0; a < 3; a++ {
			lo[a], hi[a] = math.Inf(1), math.Inf(-1)
		}
		for _, b := range bs {
			for a := 0; a < 3; a++ {
				lo[a] = math.Min(lo[a], b.lo[a])
				hi[a] = math.Max(hi[a], b.hi[a])
			}
		}
		return
	}
	alo, ahi := bounds(meshes)
	plo, phi := bounds(play)
	fmt.Printf("bbox TOUS meshes    : X[%.1f,%.1f] Y[%.1f,%.1f] Z[%.1f,%.1f]\n", alo[0], ahi[0], alo[1], ahi[1], alo[2], ahi[2])
	fmt.Printf("bbox meshes <%.0fwu   : X[%.1f,%.1f] Y[%.1f,%.1f] Z[%.1f,%.1f]  (%d meshes)\n\n",
		maxExt, plo[0], phi[0], plo[1], phi[1], plo[2], phi[2], len(play))

	// --- 2. stats trajectoires ---
	var tx, ty, tz []float64
	for _, p := range traj {
		tx = append(tx, p.x)
		ty = append(ty, p.y)
		tz = append(tz, p.z)
	}
	sort.Float64s(tx)
	sort.Float64s(ty)
	sort.Float64s(tz)
	fmt.Printf("traj    : X[%.2f,%.2f] Y[%.2f,%.2f] Z[%.2f,%.2f]\n", tx[0], tx[len(tx)-1], ty[0], ty[len(ty)-1], tz[0], tz[len(tz)-1])
	fmt.Printf("  1-99%% : X[%.2f,%.2f] Y[%.2f,%.2f] Z[%.2f,%.2f]\n\n",
		q(tx, .01), q(tx, .99), q(ty, .01), q(ty, .99), q(tz, .01), q(tz, .99))

	// --- 3. HYPOTHÈSE ABSOLUE : identité (aucun paramètre ajusté) ---
	// Empreinte XY à 1 wu de résolution, sur l'union des deux bbox.
	gx0 := math.Min(plo[0], tx[0]) - 5
	gy0 := math.Min(plo[1], ty[0]) - 5
	gx1 := math.Max(phi[0], tx[len(tx)-1]) + 5
	gy1 := math.Max(phi[1], ty[len(ty)-1]) + 5
	for _, cell := range []float64{0.5, 1.0, 2.0} {
		g := newOcc(gx0, gy0, gx1, gy1, cell)
		for _, b := range play {
			g.fill(b)
		}
		hit := 0
		for _, p := range traj {
			if g.at(p.x, p.y) > 0 {
				hit++
			}
		}
		fmt.Printf("ABSOLU (identité, cellule %.1f wu) : %d/%d points sur la géométrie = %.1f%%\n",
			cell, hit, len(traj), 100*float64(hit)/float64(len(traj)))
	}

	// contrôle négatif : nuage uniforme sur la même bbox que la trajectoire
	{
		g := newOcc(gx0, gy0, gx1, gy1, 1.0)
		for _, b := range play {
			g.fill(b)
		}
		hit, n := 0, 0
		for i := 0; i < 200; i++ {
			for j := 0; j < 200; j++ {
				x := tx[0] + (tx[len(tx)-1]-tx[0])*float64(i)/199
				y := ty[0] + (ty[len(ty)-1]-ty[0])*float64(j)/199
				n++
				if g.at(x, y) > 0 {
					hit++
				}
			}
		}
		fmt.Printf("  contrôle négatif (grille uniforme sur la même bbox) : %.1f%% — c'est le taux \"par hasard\"\n\n",
			100*float64(hit)/float64(n))
	}

	// --- 3bis. TEST DÉCISIF : balayage de translation. Si (0,0) est l'argmax, alors
	// l'alignement n'est PAS ajusté — c'est le repère natif qui coïncide.
	sweep(play, traj, gx0, gy0, gx1, gy1)

	// --- 3ter. TEST DES SOLS : le joueur doit être POSÉ sur une surface (Z du point
	// juste au-dessus du sommet d'un mesh sous lui). Un alignement fortuit en XY ne
	// produit PAS une distribution serrée de la hauteur au-dessus du sol.
	floorTest(play, traj)

	// --- 4. HYPOTHÈSE NORMALISÉE (placage, pour comparaison avec catalyst 84%) ---
	normScore(play, traj)

	// --- 5. rendu PNG en coordonnées MONDE (hypothèse absolue) ---
	renderPNG(os.Args[3], play, traj, gx0, gy0, gx1, gy1)
	fmt.Printf("→ %s\n", os.Args[3])
}

// sweep balaie des translations (dx,dy) et reporte le score de recouvrement.
// L'alignement est PROUVÉ (et non ajusté) si (0,0) est le maximum global.
func sweep(play []box, traj []pt, gx0, gy0, gx1, gy1 float64) {
	g := newOcc(gx0-60, gy0-60, gx1+60, gy1+60, 1.0)
	for _, b := range play {
		g.fill(b)
	}
	score := func(dx, dy float64) float64 {
		hit := 0
		for _, p := range traj {
			if g.at(p.x+dx, p.y+dy) > 0 {
				hit++
			}
		}
		return 100 * float64(hit) / float64(len(traj))
	}
	fmt.Println("BALAYAGE DE TRANSLATION (recouvrement %, dx en colonnes, dy en lignes) :")
	steps := []float64{-40, -30, -20, -10, -5, 0, 5, 10, 20, 30, 40}
	fmt.Printf("  dy\\dx ")
	for _, dx := range steps {
		fmt.Printf("%7.0f", dx)
	}
	fmt.Println()
	best, bx, by := -1.0, 0.0, 0.0
	for _, dy := range steps {
		fmt.Printf("%7.0f", dy)
		for _, dx := range steps {
			s := score(dx, dy)
			if s > best {
				best, bx, by = s, dx, dy
			}
			fmt.Printf("%7.1f", s)
		}
		fmt.Println()
	}
	fmt.Printf("  → max = %.1f%% à (dx=%.0f, dy=%.0f) ; identité (0,0) = %.1f%%\n",
		best, bx, by, score(0, 0))
	// balayage fin autour de 0
	bf, bfx, bfy := -1.0, 0.0, 0.0
	for dy := -4.0; dy <= 4.0; dy += 0.5 {
		for dx := -4.0; dx <= 4.0; dx += 0.5 {
			if s := score(dx, dy); s > bf {
				bf, bfx, bfy = s, dx, dy
			}
		}
	}
	fmt.Printf("  → balayage fin ±4wu : max %.1f%% à (%.1f, %.1f)\n\n", bf, bfx, bfy)
}

// meshIndex : index spatial grossier (cellules de 4 wu) pour requêtes en colonne XY.
type meshIndex struct {
	play []box
	idx  map[[2]int][]int
}

const idxCell = 4.0

func newMeshIndex(play []box) *meshIndex {
	m := &meshIndex{play: play, idx: map[[2]int][]int{}}
	for n, b := range play {
		for j := int(math.Floor(b.lo[1] / idxCell)); j <= int(math.Floor(b.hi[1]/idxCell)); j++ {
			for i := int(math.Floor(b.lo[0] / idxCell)); i <= int(math.Floor(b.hi[0]/idxCell)); i++ {
				k := [2]int{i, j}
				m.idx[k] = append(m.idx[k], n)
			}
		}
	}
	return m
}

// surfDist renvoie la distance VERTICALE signée minimale entre le point et le sommet
// (hi[2]) d'un mesh dont la colonne XY contient le point. NaN si aucun mesh.
func (m *meshIndex) surfDist(x, y, z float64) float64 {
	best, ok := math.Inf(1), false
	for _, n := range m.idx[[2]int{int(math.Floor(x / idxCell)), int(math.Floor(y / idxCell))}] {
		b := m.play[n]
		if x < b.lo[0] || x > b.hi[0] || y < b.lo[1] || y > b.hi[1] {
			continue
		}
		d := z - b.hi[2]
		if math.Abs(d) < math.Abs(best) {
			best, ok = d, true
		}
	}
	if !ok {
		return math.NaN()
	}
	return best
}

// floorScore : % de points (sur TOUS les points, un point sans surface = raté) posés
// à moins de tol wu d'une surface de la géométrie. C'est le critère discriminant.
func (m *meshIndex) floorScore(traj []pt, dx, dy, tol float64, step int) float64 {
	hit, n := 0, 0
	for i := 0; i < len(traj); i += step {
		n++
		d := m.surfDist(traj[i].x+dx, traj[i].y+dy, traj[i].z)
		if !math.IsNaN(d) && math.Abs(d) <= tol {
			hit++
		}
	}
	return 100 * float64(hit) / float64(n)
}

// floorTest mesure la hauteur au-dessus de la surface la plus proche, puis balaie les
// translations AVEC CE CRITÈRE. Si (0,0) est l'argmax, l'alignement est natif.
func floorTest(play []box, traj []pt) {
	mi := newMeshIndex(play)
	step := 1
	if len(traj) > 60000 {
		step = len(traj) / 60000
	}
	var d []float64
	noSurf := 0
	for n := 0; n < len(traj); n += step {
		p := traj[n]
		v := mi.surfDist(p.x, p.y, p.z)
		if math.IsNaN(v) {
			noSurf++
			continue
		}
		d = append(d, v)
	}
	if len(d) == 0 {
		fmt.Println("TEST DES SOLS : aucune surface sous les points")
		return
	}
	sort.Float64s(d)
	fmt.Printf("TEST DES SOLS (identité, %d pts avec surface dans leur colonne, %d sans) :\n", len(d), noSurf)
	fmt.Printf("  écart vertical signé au sommet de mesh le plus proche :\n")
	fmt.Printf("    p05=%.2f p25=%.2f p50=%.2f p75=%.2f p95=%.2f wu\n",
		q(d, .05), q(d, .25), q(d, .5), q(d, .75), q(d, .95))

	fmt.Println("\nBALAYAGE DE TRANSLATION — critère \"POSÉ SUR UNE SURFACE\" (|dz|<=1wu, % sur TOUS les points) :")
	steps := []float64{-20, -15, -10, -5, -2, 0, 2, 5, 10, 15, 20}
	fmt.Printf("  dy\\dx ")
	for _, dx := range steps {
		fmt.Printf("%7.0f", dx)
	}
	fmt.Println()
	best, bx, by := -1.0, 0.0, 0.0
	for _, dy := range steps {
		fmt.Printf("%7.0f", dy)
		for _, dx := range steps {
			s := mi.floorScore(traj, dx, dy, 1.0, step)
			if s > best {
				best, bx, by = s, dx, dy
			}
			fmt.Printf("%7.1f", s)
		}
		fmt.Println()
	}
	fmt.Printf("  → max = %.1f%% à (dx=%.0f, dy=%.0f) ; identité (0,0) = %.1f%%\n",
		best, bx, by, mi.floorScore(traj, 0, 0, 1.0, step))
	bf, bfx, bfy := -1.0, 0.0, 0.0
	for dy := -3.0; dy <= 3.0; dy += 0.25 {
		for dx := -3.0; dx <= 3.0; dx += 0.25 {
			if s := mi.floorScore(traj, dx, dy, 1.0, step*3); s > bf {
				bf, bfx, bfy = s, dx, dy
			}
		}
	}
	fmt.Printf("  → balayage fin ±3wu (pas 0.25) : max %.1f%% à (%.2f, %.2f)\n", bf, bfx, bfy)
	// balayage vertical : y a-t-il un offset Z ?
	fmt.Printf("  balayage Z : ")
	bz, bzs := 0.0, -1.0
	for dz := -3.0; dz <= 3.0; dz += 0.25 {
		hit, n := 0, 0
		for i := 0; i < len(traj); i += step {
			n++
			v := mi.surfDist(traj[i].x, traj[i].y, traj[i].z+dz)
			if !math.IsNaN(v) && math.Abs(v) <= 1.0 {
				hit++
			}
		}
		s := 100 * float64(hit) / float64(n)
		if s > bzs {
			bzs, bz = s, dz
		}
	}
	fmt.Printf("meilleur dz = %.2f wu (%.1f%%)\n\n", bz, bzs)

	// --- recherche 3D LARGE : existe-t-il UNE translation rigide qui pose les
	// trajectoires sur la géométrie ? (si le meilleur reste faible → la surface
	// jouée n'est pas dans ce module, ex. géométrie Forge.)
	fmt.Println("RECHERCHE 3D LARGE (dx,dy ∈ [-30,30] pas 5 ; dz ∈ [-30,30] pas 2.5) :")
	type res struct{ s, dx, dy, dz float64 }
	bestR := res{-1, 0, 0, 0}
	st := step * 4
	for dz := -30.0; dz <= 30.0; dz += 2.5 {
		for dy := -30.0; dy <= 30.0; dy += 5 {
			for dx := -30.0; dx <= 30.0; dx += 5 {
				hit, n := 0, 0
				for i := 0; i < len(traj); i += st {
					n++
					v := mi.surfDist(traj[i].x+dx, traj[i].y+dy, traj[i].z+dz)
					if !math.IsNaN(v) && math.Abs(v) <= 1.0 {
						hit++
					}
				}
				if s := 100 * float64(hit) / float64(n); s > bestR.s {
					bestR = res{s, dx, dy, dz}
				}
			}
		}
	}
	fmt.Printf("  → meilleure translation rigide : %.1f%% à (dx=%.1f, dy=%.1f, dz=%.1f)\n", bestR.s, bestR.dx, bestR.dy, bestR.dz)
	fmt.Printf("  → identité : %.1f%%\n\n", mi.floorScore(traj, 0, 0, 1.0, st))

	// distribution des Z : trajectoires vs sommets de mesh dans les mêmes colonnes
	var mz []float64
	for i := 0; i < len(traj); i += step * 2 {
		for _, n := range mi.idx[[2]int{int(math.Floor(traj[i].x / idxCell)), int(math.Floor(traj[i].y / idxCell))}] {
			b := mi.play[n]
			if traj[i].x >= b.lo[0] && traj[i].x <= b.hi[0] && traj[i].y >= b.lo[1] && traj[i].y <= b.hi[1] {
				mz = append(mz, b.hi[2])
			}
		}
	}
	var tz []float64
	for _, p := range traj {
		tz = append(tz, p.z)
	}
	sort.Float64s(mz)
	sort.Float64s(tz)
	if len(mz) > 0 {
		fmt.Printf("Z des sommets de mesh dans les colonnes visitées : p05=%.1f p25=%.1f p50=%.1f p75=%.1f p95=%.1f\n",
			q(mz, .05), q(mz, .25), q(mz, .5), q(mz, .75), q(mz, .95))
	}
	fmt.Printf("Z des trajectoires                               : p05=%.1f p25=%.1f p50=%.1f p75=%.1f p95=%.1f\n\n",
		q(tz, .05), q(tz, .25), q(tz, .5), q(tz, .75), q(tz, .95))

	// --- similitude complète (rotation autour de la verticale + translation) : si même
	// ÇA ne dépasse pas ~60%, la surface jouée n'existe pas dans ce module.
	fmt.Println("RECHERCHE SIMILITUDE (rotation Z 0-345° pas 15° + translation) :")
	type r4 struct{ s, a, dx, dy, dz float64 }
	bb := r4{-1, 0, 0, 0, 0}
	st2 := step * 8
	var sx, sy, sz []float64
	for i := 0; i < len(traj); i += st2 {
		sx = append(sx, traj[i].x)
		sy = append(sy, traj[i].y)
		sz = append(sz, traj[i].z)
	}
	cxm, cym := (tx0(sx)+tx1(sx))/2, (tx0(sy)+tx1(sy))/2
	for ai := 0; ai < 24; ai++ {
		a := float64(ai) * 15 * math.Pi / 180
		ca, sa := math.Cos(a), math.Sin(a)
		rx := make([]float64, len(sx))
		ry := make([]float64, len(sx))
		for i := range sx {
			ux, uy := sx[i]-cxm, sy[i]-cym
			rx[i], ry[i] = cxm+ux*ca-uy*sa, cym+ux*sa+uy*ca
		}
		for dz := -5.0; dz <= 25.0; dz += 5 {
			for dy := -30.0; dy <= 30.0; dy += 5 {
				for dx := -30.0; dx <= 30.0; dx += 5 {
					hit := 0
					for i := range rx {
						v := mi.surfDist(rx[i]+dx, ry[i]+dy, sz[i]+dz)
						if !math.IsNaN(v) && math.Abs(v) <= 1.0 {
							hit++
						}
					}
					if s := 100 * float64(hit) / float64(len(rx)); s > bb.s {
						bb = r4{s, float64(ai) * 15, dx, dy, dz}
					}
				}
			}
		}
	}
	fmt.Printf("  → meilleure similitude : %.1f%% à angle=%.0f° (dx=%.0f, dy=%.0f, dz=%.0f) — %d params ajustés\n\n",
		bb.s, bb.a, bb.dx, bb.dy, bb.dz, 4)
}

func tx0(v []float64) float64 {
	m := math.Inf(1)
	for _, x := range v {
		m = math.Min(m, x)
	}
	return m
}
func tx1(v []float64) float64 {
	m := math.Inf(-1)
	for _, x := range v {
		m = math.Max(m, x)
	}
	return m
}

// normScore reproduit la méthode catalyst : normalisation [0,1]^2 des deux nuages
// (bornes 1-99 pct) + meilleure des 8 orientations. Score de PLACAGE.
func normScore(play []box, traj []pt) {
	cx, cy := make([]float64, len(play)), make([]float64, len(play))
	for i, b := range play {
		cx[i], cy[i] = (b.lo[0]+b.hi[0])/2, (b.lo[1]+b.hi[1])/2
	}
	rb := func(v []float64) (float64, float64) {
		s := append([]float64(nil), v...)
		sort.Float64s(s)
		return q(s, .01), q(s, .99)
	}
	mlx, mhx := rb(cx)
	mly, mhy := rb(cy)
	var px, py []float64
	for _, p := range traj {
		px = append(px, p.x)
		py = append(py, p.y)
	}
	flx, fhx := rb(px)
	fly, fhy := rb(py)
	cl := func(v float64) float64 { return math.Max(0, math.Min(1, v)) }

	const G = 120
	foot := make([]int, G*G)
	for _, b := range play {
		x0 := int(cl((b.lo[0]-mlx)/(mhx-mlx)) * (G - 1))
		x1 := int(cl((b.hi[0]-mlx)/(mhx-mlx)) * (G - 1))
		y0 := int(cl((b.lo[1]-mly)/(mhy-mly)) * (G - 1))
		y1 := int(cl((b.hi[1]-mly)/(mhy-mly)) * (G - 1))
		for j := y0; j <= y1; j++ {
			for i := x0; i <= x1; i++ {
				foot[j*G+i]++
			}
		}
	}
	orient := func(u, v float64, k int) (float64, float64) {
		switch k {
		case 0:
			return u, v
		case 1:
			return 1 - u, v
		case 2:
			return u, 1 - v
		case 3:
			return 1 - u, 1 - v
		case 4:
			return v, u
		case 5:
			return 1 - v, u
		case 6:
			return v, 1 - u
		}
		return 1 - v, 1 - u
	}
	bestK, bestS := 0, -1
	for k := 0; k < 8; k++ {
		s := 0
		for _, p := range traj {
			u := cl((p.x - flx) / (fhx - flx))
			v := cl((p.y - fly) / (fhy - fly))
			a, b := orient(u, v, k)
			if foot[int(b*(G-1))*G+int(a*(G-1))] > 0 {
				s++
			}
		}
		if s > bestS {
			bestS, bestK = s, k
		}
	}
	fmt.Printf("NORMALISÉ (placage, meilleure des 8 orientations) : #%d → %d/%d = %.1f%% (réf catalyst : 84%%)\n\n",
		bestK, bestS, len(traj), 100*float64(bestS)/float64(len(traj)))
}

func renderPNG(path string, play []box, traj []pt, x0, y0, x1, y1 float64) {
	const S, pad = 1000, 40
	span := float64(S - 2*pad)
	sc := span / math.Max(x1-x0, y1-y0)
	img := image.NewRGBA(image.Rect(0, 0, S, S))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			img.SetRGBA(x, y, color.RGBA{14, 16, 22, 255})
		}
	}
	px := func(wx, wy float64) (int, int) {
		return pad + int((wx-x0)*sc), S - pad - int((wy-y0)*sc)
	}
	// géométrie : remplissage sombre pour les grosses dalles, contour clair pour les
	// meshes de taille "structure" (<15 wu) → révèle le détail de la géométrie canvas.
	acc := make([]int, S*S)
	maxc := 1
	for _, b := range play {
		ax, ay := px(b.lo[0], b.lo[1])
		bx, by := px(b.hi[0], b.hi[1])
		if ay < by {
			ay, by = by, ay
		}
		for y := by; y <= ay; y++ {
			for x := ax; x <= bx; x++ {
				if x >= 0 && x < S && y >= 0 && y < S {
					acc[y*S+x]++
					if acc[y*S+x] > maxc {
						maxc = acc[y*S+x]
					}
				}
			}
		}
	}
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			c := acc[y*S+x]
			if c == 0 {
				continue
			}
			l := uint8(38 + 120*math.Min(1, float64(c)/float64(maxc)*4))
			img.SetRGBA(x, y, color.RGBA{l, l, uint8(math.Min(255, float64(l)*1.2)), 255})
		}
	}
	line := func(x0, y0, x1, y1 int, col color.RGBA) {
		for x := x0; x <= x1; x++ {
			for _, y := range []int{y0, y1} {
				if x >= 0 && x < S && y >= 0 && y < S {
					img.SetRGBA(x, y, col)
				}
			}
		}
		for y := y0; y <= y1; y++ {
			for _, x := range []int{x0, x1} {
				if x >= 0 && x < S && y >= 0 && y < S {
					img.SetRGBA(x, y, col)
				}
			}
		}
	}
	for _, b := range play {
		e := 0.0
		for a := 0; a < 3; a++ {
			e = math.Max(e, b.hi[a]-b.lo[a])
		}
		if e >= 15 {
			continue
		}
		ax, ay := px(b.lo[0], b.lo[1])
		bx, by := px(b.hi[0], b.hi[1])
		if ay < by {
			ay, by = by, ay
		}
		line(ax, by, bx, ay, color.RGBA{130, 140, 165, 255})
	}
	// trajectoires : 1 couleur par slot (Okabe-Ito), points 1px + halo léger
	pal := []color.RGBA{
		{230, 159, 0, 255}, {86, 180, 233, 255}, {0, 158, 115, 255}, {240, 228, 66, 255},
		{0, 114, 178, 255}, {213, 94, 0, 255}, {204, 121, 167, 255}, {255, 255, 255, 255},
	}
	for _, p := range traj {
		cx, cy := px(p.x, p.y)
		col := pal[p.slot%len(pal)]
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				x, y := cx+dx, cy+dy
				if x >= 0 && x < S && y >= 0 && y < S {
					r, g, b, _ := img.At(x, y).RGBA()
					img.SetRGBA(x, y, color.RGBA{
						uint8(math.Min(255, float64(r>>8)*0.35+float64(col.R)*0.75)),
						uint8(math.Min(255, float64(g>>8)*0.35+float64(col.G)*0.75)),
						uint8(math.Min(255, float64(b>>8)*0.35+float64(col.B)*0.75)), 255})
				}
			}
		}
	}
	f, err := os.Create(path)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	png.Encode(f, img)
}
