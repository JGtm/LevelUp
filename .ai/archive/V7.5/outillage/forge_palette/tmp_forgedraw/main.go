// tmp_forgedraw — jetable, session « palette Forge : nommer et mesurer ».
//
// Dessine les zones d'objectif d'une variante de carte PAR-DESSUS le sol reconstruit
// et les positions joueur, et MESURE : part des positions joueur dans la boite
// ORIENTEE contre la boite ALIGNEE, et contre un temoin negatif (la meme forme
// deplacee). La regle du chantier est explicite : toujours dessiner un resultat
// geometrique, jamais seulement le compter.
//
// Usage :
//
//	tmp_forgedraw <mvar> <sol.csv> <trajectoires.csv> <sortie.png>
package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

const fixedPointUnit = 65536.0 // pas etabli : 393216 = 6 x 65536 exactement

type rect struct{ minX, minY, maxX, maxY float64 }

type zone struct {
	role           string
	kind           int32 // 2 = cylindre presume, 3 = boite presumee
	a, b, top, bot float64
	pos            mapvar.Vec3
	fwd            mapvar.Vec3
	team           int
}

func main() {
	if len(os.Args) < 5 {
		fmt.Fprintln(os.Stderr, "usage: tmp_forgedraw <mvar> <sol.csv> <trajectoires.csv> <sortie.png>")
		os.Exit(2)
	}
	if os.Args[1] == "map" {
		cmdMap(os.Args[2], os.Args[3], os.Args[4:])
		return
	}
	zones := loadZones(os.Args[1])
	floor := loadFloor(os.Args[2])
	pts := loadPoints(os.Args[3])
	fmt.Printf("zones=%d  rectangles de sol=%d  positions joueur=%d\n", len(zones), len(floor), len(pts))
	measure(zones, pts)
	fitTest(zones, pts)
	drawPNG(os.Args[4], zones, floor, pts)
}

func loadZones(path string) []zone {
	buf, err := os.ReadFile(path)
	must(err)
	root, err := mapvar.DecodeRoot(buf)
	must(err)
	v, err := mapvar.Parse(buf)
	must(err)
	objs, _ := root.Field(3)
	var out []zone
	for _, ob := range v.Objectives() {
		raw := objs.Items[ob.ObjectIdx]
		kind, slots, ok := shapeOf(raw)
		if !ok {
			continue
		}
		o := v.Objects[ob.ObjectIdx]
		out = append(out, zone{
			role: string(ob.Role), kind: kind,
			a:   float64(slots[5]) / fixedPointUnit,
			b:   float64(slots[6]) / fixedPointUnit,
			top: float64(slots[7]) / fixedPointUnit,
			bot: float64(slots[8]) / fixedPointUnit,
			pos: o.Pos, fwd: o.Forward, team: o.TeamIndex,
		})
	}
	return out
}

func shapeOf(raw mapvar.Value) (int32, [9]int64, bool) {
	var slots [9]int64
	bag, ok := raw.Field(8)
	if !ok {
		return 0, slots, false
	}
	lst, ok := bag.Field(0)
	if !ok || len(lst.Items) == 0 {
		return 0, slots, false
	}
	inner, ok := lst.Items[0].Field(0)
	if !ok || len(inner.Items) == 0 {
		return 0, slots, false
	}
	sh := inner.Items[0]
	kind, ok := sh.Field(0)
	if !ok {
		return 0, slots, false
	}
	for i := uint16(1); i <= 8; i++ {
		if f, ok := sh.Field(i); ok {
			if v, ok2 := f.Field(0); ok2 {
				slots[i] = v.Int
			}
		}
	}
	return int32(kind.Int), slots, true
}

// insideOBB teste l'appartenance a la boite ORIENTEE par `fwd` autour de `pos`.
// `half` dit si (a,b) sont des demi-extents ; sinon ce sont des tailles pleines.
func (z zone) insideOBB(x, y float64, half bool) bool {
	ha, hb := z.a, z.b
	if !half {
		ha, hb = z.a/2, z.b/2
	}
	dx, dy := x-z.pos.X, y-z.pos.Y
	fx, fy := z.fwd.X, z.fwd.Y
	n := math.Hypot(fx, fy)
	if n < 1e-6 {
		fx, fy, n = 1, 0, 1
	}
	fx, fy = fx/n, fy/n
	u := dx*fx + dy*fy  // le long de forward
	w := -dx*fy + dy*fx // perpendiculaire
	if z.kind == 2 {
		return math.Hypot(u, w) <= ha
	}
	return math.Abs(u) <= ha && math.Abs(w) <= hb
}

// insideAABB teste la boite ALIGNEE sur les axes du monde, memes demi-extents.
func (z zone) insideAABB(x, y float64, half bool) bool {
	ha, hb := z.a, z.b
	if !half {
		ha, hb = z.a/2, z.b/2
	}
	if z.kind == 2 {
		return math.Abs(x-z.pos.X) <= ha && math.Abs(y-z.pos.Y) <= ha
	}
	return math.Abs(x-z.pos.X) <= ha && math.Abs(y-z.pos.Y) <= hb
}

// measure : part des positions joueur DANS la forme, pour les deux lectures de
// (a,b), plus un temoin negatif (forme translatee de 25 m).
func measure(zones []zone, pts [][2]float64) {
	fmt.Printf("\n%-18s %-4s %-16s %-9s %-9s %-9s %-9s\n",
		"role", "f", "dims (m)", "OBB demi", "OBB plein", "AABB demi", "temoin-")
	for _, z := range zones {
		var nHalf, nFull, nAabbHalf, nCtrl int
		for _, p := range pts {
			if z.insideOBB(p[0], p[1], true) {
				nHalf++
			}
			if z.insideOBB(p[0], p[1], false) {
				nFull++
			}
			if z.insideAABB(p[0], p[1], true) {
				nAabbHalf++
			}
			if z.insideOBB(p[0]-25, p[1]-25, true) {
				nCtrl++
			}
		}
		n := float64(len(pts))
		fmt.Printf("%-18s %-4d %6.2f x %-7.2f %8.3f%% %8.3f%% %8.3f%% %8.3f%%\n",
			z.role, z.kind, z.a, z.b,
			100*float64(nHalf)/n, 100*float64(nFull)/n,
			100*float64(nAabbHalf)/n, 100*float64(nCtrl)/n)
	}
}

// fitTest discrimine « demi-extents » et « tailles pleines ». Une zone de capture
// est posee sur une plate-forme jouable : son empreinte doit tomber sur du sol
// EFFECTIVEMENT FOULE. On maille le terrain a 0,5 m avec les positions joueur, puis
// on mesure la part de l'empreinte qui tombe sur une case foulee. Une lecture deux
// fois trop grande deborde dans le vide et le score chute.
//
// Temoin : la meme mesure pour une forme volontairement doublee ET pour une forme
// volontairement halvee — elles bornent ce que « trop grand » et « trop petit »
// donnent sur cette carte.
func fitTest(zones []zone, pts [][2]float64) {
	const cell = 0.5
	visited := map[[2]int]bool{}
	for _, p := range pts {
		visited[[2]int{int(math.Floor(p[0] / cell)), int(math.Floor(p[1] / cell))}] = true
	}
	frac := func(z zone, mul float64, half bool) float64 {
		zz := z
		zz.a, zz.b = z.a*mul, z.b*mul
		var in, tot int
		ext := math.Max(zz.a, zz.b) + 1
		for x := zz.pos.X - ext; x <= zz.pos.X+ext; x += cell {
			for y := zz.pos.Y - ext; y <= zz.pos.Y+ext; y += cell {
				if !zz.insideOBB(x, y, half) {
					continue
				}
				tot++
				if visited[[2]int{int(math.Floor(x / cell)), int(math.Floor(y / cell))}] {
					in++
				}
			}
		}
		if tot == 0 {
			return 0
		}
		return 100 * float64(in) / float64(tot)
	}
	fmt.Printf("\npart de l'empreinte sur du sol FOULE (maille 0,5 m)\n")
	fmt.Printf("%-18s %-16s %-11s %-11s %-11s %-11s\n",
		"role", "dims (m)", "lect. demi", "lect. plein", "demi x2", "demi /2")
	var sh, sf float64
	for _, z := range zones {
		h, f := frac(z, 1, true), frac(z, 1, false)
		sh, sf = sh+h, sf+f
		fmt.Printf("%-18s %6.2f x %-7.2f %9.1f%% %9.1f%% %9.1f%% %9.1f%%\n",
			z.role, z.a, z.b, h, f, frac(z, 2, true), frac(z, 0.5, true))
	}
	n := float64(len(zones))
	fmt.Printf("%-18s %-16s %9.1f%% %9.1f%%   <- moyenne\n", "", "", sh/n, sf/n)
}

func loadFloor(path string) []rect {
	var out []rect
	each(path, func(f []string) {
		if len(f) < 6 {
			return
		}
		out = append(out, rect{num(f[1]), num(f[2]), num(f[3]), num(f[4])})
	})
	return out
}

func loadPoints(path string) [][2]float64 {
	var out [][2]float64
	each(path, func(f []string) {
		if len(f) < 6 {
			return
		}
		out = append(out, [2]float64{num(f[4]), num(f[5])})
	})
	return out
}

func each(path string, fn func([]string)) {
	f, err := os.Open(path)
	must(err)
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for i := 0; sc.Scan(); i++ {
		if i == 0 {
			continue
		}
		fn(strings.Split(sc.Text(), ","))
	}
}

func num(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
