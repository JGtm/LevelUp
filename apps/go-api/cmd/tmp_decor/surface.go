package main

import (
	"math"
	"sort"
)

// surface est la face supérieure de l'AABB monde d'une instance : le candidat
// « sol sur lequel on marche » à l'altitude z.
type surface struct {
	minX, minY, maxX, maxY float64
	z                      float64
}

func (s surface) contains(x, y float64) bool {
	return x >= s.minX && x <= s.maxX && y >= s.minY && y <= s.maxY
}

// grid indexe les surfaces par cellule XY pour la requête « quelle surface sous ce point ».
type grid struct {
	cell           float64
	minX, minY     float64
	nx, ny         int
	buckets        [][]int32
	surf           []surface
	maxX, maxY     float64
	emptyBucketNum int
}

const gridCell = 4.0

func newGrid(surf []surface) *grid {
	g := &grid{cell: gridCell, surf: surf,
		minX: math.Inf(1), minY: math.Inf(1), maxX: math.Inf(-1), maxY: math.Inf(-1)}
	for _, s := range surf {
		g.minX = math.Min(g.minX, s.minX)
		g.minY = math.Min(g.minY, s.minY)
		g.maxX = math.Max(g.maxX, s.maxX)
		g.maxY = math.Max(g.maxY, s.maxY)
	}
	g.nx = int((g.maxX-g.minX)/g.cell) + 2
	g.ny = int((g.maxY-g.minY)/g.cell) + 2
	g.buckets = make([][]int32, g.nx*g.ny)
	for i, s := range surf {
		x0, y0 := g.ix(s.minX), g.iy(s.minY)
		x1, y1 := g.ix(s.maxX), g.iy(s.maxY)
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				g.buckets[y*g.nx+x] = append(g.buckets[y*g.nx+x], int32(i))
			}
		}
	}
	return g
}

func (g *grid) ix(v float64) int { return clamp(int((v-g.minX)/g.cell), 0, g.nx-1) }
func (g *grid) iy(v float64) int { return clamp(int((v-g.minY)/g.cell), 0, g.ny-1) }

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// topBelow renvoie l'altitude de la surface la plus HAUTE sous (x, y, z+tol).
// ok=false si aucune surface ne couvre le point.
func (g *grid) topBelow(x, y, z, tol float64) (float64, bool) {
	if x < g.minX || x > g.maxX || y < g.minY || y > g.maxY {
		return 0, false
	}
	best, found := math.Inf(-1), false
	for _, i := range g.buckets[g.iy(y)*g.nx+g.ix(x)] {
		s := g.surf[i]
		if !s.contains(x, y) || s.z > z+tol {
			continue
		}
		if s.z > best {
			best, found = s.z, true
		}
	}
	return best, found
}

// dzStats résume la distribution des dz au-dessus de la surface la plus haute située
// sous le joueur. La discrimination se joue sur les bandes SERRÉES : avec des surfaces
// denses, « il existe une surface un peu en dessous » est presque tautologique — c'est
// « le joueur est POSÉ dessus au centimètre » qui ne l'est pas.
type dzStats struct {
	n        int
	covered  int
	within1  float64 // |dz| <= 1 cm
	within5  float64 // |dz| <= 5 cm
	within25 float64 // |dz| <= 25 cm
	median   float64
	mode     float64
}

const (
	binWidth = 0.02
	// searchTol : marge au-dessus du joueur pour tolérer une surface « au niveau des pieds ».
	searchTol = 0.02
)

func measure(g *grid, pts [][3]float64) dzStats {
	var dz []float64
	st := dzStats{n: len(pts)}
	for _, p := range pts {
		top, ok := g.topBelow(p[0], p[1], p[2], searchTol)
		if !ok {
			continue
		}
		st.covered++
		dz = append(dz, p[2]-top)
	}
	if st.covered == 0 {
		return st
	}
	hist := map[int]int{}
	for _, d := range dz {
		hist[int(math.Floor(d/binWidth))]++
	}
	bestBin, bestN := 0, -1
	for b, n := range hist {
		if n > bestN || (n == bestN && b < bestBin) {
			bestBin, bestN = b, n
		}
	}
	st.mode = (float64(bestBin) + 0.5) * binWidth
	var c1, c5, c25 int
	for _, d := range dz {
		a := math.Abs(d)
		if a <= 0.01 {
			c1++
		}
		if a <= 0.05 {
			c5++
		}
		if a <= 0.25 {
			c25++
		}
	}
	f := float64(len(dz))
	st.within1, st.within5, st.within25 = float64(c1)/f, float64(c5)/f, float64(c25)/f
	sort.Float64s(dz)
	st.median = dz[len(dz)/2]
	return st
}
