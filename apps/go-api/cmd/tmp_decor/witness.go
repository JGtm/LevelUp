package main

import (
	"fmt"
	"math"
	"math/rand"

	"levelup/go-api/internal/himap"
)

// transform décrit une variante de la géométrie servant de témoin.
type transform struct {
	name string
	fn   func([]himap.Instance, himap.Bounds) []surface
}

func center(b himap.Bounds, a int) float64 { return (b.Min[a] + b.Max[a]) / 2 }

// rotXY tourne les emprises de k*90 degrés autour du centre des bornes monde.
func rotXY(k int) func([]himap.Instance, himap.Bounds) []surface {
	return func(ins []himap.Instance, b himap.Bounds) []surface {
		cx, cy := center(b, 0), center(b, 1)
		out := make([]surface, 0, len(ins))
		for _, in := range ins {
			xs := [2]float64{in.AABBMin[0] - cx, in.AABBMax[0] - cx}
			ys := [2]float64{in.AABBMin[1] - cy, in.AABBMax[1] - cy}
			var px, py [4]float64
			n := 0
			for _, x := range xs {
				for _, y := range ys {
					rx, ry := x, y
					for i := 0; i < k; i++ {
						rx, ry = -ry, rx
					}
					px[n], py[n] = rx+cx, ry+cy
					n++
				}
			}
			out = append(out, surface{
				minX: min4(px), maxX: max4(px), minY: min4(py), maxY: max4(py), z: in.AABBMax[2],
			})
		}
		return out
	}
}

func translate(dx, dy float64) func([]himap.Instance, himap.Bounds) []surface {
	return func(ins []himap.Instance, _ himap.Bounds) []surface {
		out := make([]surface, 0, len(ins))
		for _, in := range ins {
			out = append(out, surface{
				minX: in.AABBMin[0] + dx, maxX: in.AABBMax[0] + dx,
				minY: in.AABBMin[1] + dy, maxY: in.AABBMax[1] + dy, z: in.AABBMax[2],
			})
		}
		return out
	}
}

// permuteAxes : (x, y, z) -> (y, z, x). Le repère est conservé en volume mais pas en
// orientation : c'est le témoin « bon nuage, mauvais axes ».
func permuteAxes(ins []himap.Instance, _ himap.Bounds) []surface {
	out := make([]surface, 0, len(ins))
	for _, in := range ins {
		out = append(out, surface{
			minX: in.AABBMin[1], maxX: in.AABBMax[1],
			minY: in.AABBMin[2], maxY: in.AABBMax[2], z: in.AABBMax[0],
		})
	}
	return out
}

// shuffleZ conserve exactement les emprises 2D et la distribution des altitudes, mais
// casse l'appariement emprise <-> altitude. Témoin le plus sévère.
func shuffleZ(ins []himap.Instance, _ himap.Bounds) []surface {
	out := topSurfaces(ins)
	rng := rand.New(rand.NewSource(1234))
	zs := make([]float64, len(out))
	for i := range out {
		zs[i] = out[i].z
	}
	rng.Shuffle(len(zs), func(i, j int) { zs[i], zs[j] = zs[j], zs[i] })
	for i := range out {
		out[i].z = zs[i]
	}
	return out
}

func runWitnesses(bsp *himap.BSPInstances, kept []himap.Instance, all, grounded [][3]float64) {
	for _, set := range []struct {
		label string
		pts   [][3]float64
	}{{"toutes positions", all}, {"positions au sol (|vz| quasi nul)", grounded}} {
		if len(set.pts) == 0 {
			continue
		}
		witnessTable(bsp, kept, set.pts, set.label)
	}
}

func witnessTable(bsp *himap.BSPInstances, kept []himap.Instance, pts [][3]float64, label string) {
	variants := []transform{
		{"RÉEL (identité)", func(ins []himap.Instance, _ himap.Bounds) []surface { return topSurfaces(ins) }},
		{"témoin rotation 90 deg", rotXY(1)},
		{"témoin rotation 180 deg", rotXY(2)},
		{"témoin rotation 270 deg", rotXY(3)},
		{"témoin translation +8/+8 m", translate(8, 8)},
		{"témoin translation +3/-3 m", translate(3, -3)},
		{"témoin axes permutés x,y,z->y,z,x", permuteAxes},
		{"témoin altitudes permutées", shuffleZ},
	}
	// Deux jeux de surfaces : toutes, et seulement les grandes (>= minFloorArea m2),
	// qui sont les vrais sols/plateformes. Le second réduit la densité et rend le
	// témoin « altitudes permutées » nettement moins favorisé par la saturation.
	for _, sel := range []struct {
		label string
		min   float64
	}{{"toutes emprises", 0}, {"emprises >= 4 m2 (sols/plateformes)", 4}} {
		fmt.Printf("\n=== TÉMOIN DE SURFACE — %s — %s (n=%d) ===\n", label, sel.label, len(pts))
		fmt.Printf("%-36s %8s %8s %8s %8s %8s %10s\n",
			"variante", "couvert", "|dz|<1cm", "|dz|<5cm", "|dz|<25cm", "mode", "médiane dz")
		for _, v := range variants {
			g := newGrid(filterArea(v.fn(kept, bsp.Bounds), sel.min))
			printRow(v.name, measure(g, pts))
		}
		g := newGrid(filterArea(topSurfaces(kept), sel.min))
		printRow("témoin positions uniformes", measure(g, uniformPoints(pts, len(pts))))
		printRow("témoin positions x,y-z réappariés", measure(g, shufflePoints(pts)))
	}
}

// filterArea ne garde que les surfaces d'aire >= minArea.
func filterArea(s []surface, minArea float64) []surface {
	if minArea <= 0 {
		return s
	}
	out := make([]surface, 0, len(s))
	for _, x := range s {
		if (x.maxX-x.minX)*(x.maxY-x.minY) >= minArea {
			out = append(out, x)
		}
	}
	return out
}

func printRow(name string, st dzStats) {
	if st.covered == 0 {
		fmt.Printf("%-36s %8s\n", name, "0")
		return
	}
	fmt.Printf("%-36s %7.1f%% %7.1f%% %7.1f%% %7.1f%% %8.2f %10.3f\n",
		name, 100*float64(st.covered)/float64(st.n),
		100*st.within1, 100*st.within5, 100*st.within25, st.mode, st.median)
}

// uniformPoints tire des positions uniformes dans la boîte englobante des positions
// réelles : même volume d'exploration, aucune structure.
func uniformPoints(pts [][3]float64, n int) [][3]float64 {
	var lo, hi [3]float64
	for a := 0; a < 3; a++ {
		lo[a], hi[a] = math.Inf(1), math.Inf(-1)
	}
	for _, p := range pts {
		for a := 0; a < 3; a++ {
			lo[a] = math.Min(lo[a], p[a])
			hi[a] = math.Max(hi[a], p[a])
		}
	}
	rng := rand.New(rand.NewSource(99))
	out := make([][3]float64, n)
	for i := range out {
		for a := 0; a < 3; a++ {
			out[i][a] = lo[a] + rng.Float64()*(hi[a]-lo[a])
		}
	}
	return out
}

// shufflePoints garde les mêmes (x, y) et les mêmes z, mais les réapparie au hasard :
// la distribution marginale est identique, la corrélation xy<->z est détruite.
func shufflePoints(pts [][3]float64) [][3]float64 {
	out := make([][3]float64, len(pts))
	copy(out, pts)
	rng := rand.New(rand.NewSource(4321))
	zs := make([]float64, len(pts))
	for i, p := range pts {
		zs[i] = p[2]
	}
	rng.Shuffle(len(zs), func(i, j int) { zs[i], zs[j] = zs[j], zs[i] })
	for i := range out {
		out[i][2] = zs[i]
	}
	return out
}

func min4(v [4]float64) float64 {
	m := v[0]
	for _, x := range v[1:] {
		m = math.Min(m, x)
	}
	return m
}

func max4(v [4]float64) float64 {
	m := v[0]
	for _, x := range v[1:] {
		m = math.Max(m, x)
	}
	return m
}
