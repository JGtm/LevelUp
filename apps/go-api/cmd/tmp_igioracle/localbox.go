// Algèbre de la boîte locale : inversion et recomposition de l'AABB monde d'une
// instance à partir de son placement (position + base 3x3).
package main

import (
	"math"

	"levelup/go-api/internal/himap"
)

// pickSource choisit l'instance du groupe dont la matrice |B| est la mieux conditionnée.
func pickSource(ins []himap.Instance, idx []int) int {
	best, bestD := -1, 0.0
	for _, i := range idx {
		d := math.Abs(det(absMat(ins[i])))
		if d > bestD {
			bestD, best = d, i
		}
	}
	if bestD < 1e-3 {
		return -1
	}
	return best
}

// absMatOf : A[a][i] = |B_i[a]| — la matrice qui envoie les demi-extensions locales sur
// les demi-extensions monde.
func absMatOf(b [3][3]float64) [3][3]float64 {
	var a [3][3]float64
	for ax := 0; ax < 3; ax++ {
		for i := 0; i < 3; i++ {
			a[ax][i] = math.Abs(b[i][ax])
		}
	}
	return a
}

func absMat(in himap.Instance) [3][3]float64 {
	return absMatOf([3][3]float64{in.Forward, in.Left, in.Up})
}

func halfExtents(in himap.Instance) [3]float64 {
	var h [3]float64
	for a := 0; a < 3; a++ {
		h[a] = (in.AABBMax[a] - in.AABBMin[a]) / 2
	}
	return h
}

// localBox : boîte du mesh dans son repère local (centre + demi-extensions).
type localBox struct{ c, h [3]float64 }

// solveLocalBox inverse la transformation sur UNE instance :
//
//	c = (centre_monde - position) · B^T   (B orthonormée : l'inverse est la transposée)
//	h = |B|^-T · H_monde
//
// basis renvoie la base de l'instance ; transposed teste la convention COLONNE
// (contrôle : elle doit donner de moins bons résultats que la convention ligne).
func basis(in himap.Instance, transposed bool) [3][3]float64 {
	b := [3][3]float64{in.Forward, in.Left, in.Up}
	if !transposed {
		return b
	}
	var t [3][3]float64
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			t[i][j] = b[j][i]
		}
	}
	return t
}

func solveLocalBox(in himap.Instance, transposed bool) (localBox, bool) {
	a := absMatOf(basis(in, transposed))
	d := det(a)
	if math.Abs(d) < 1e-6 {
		return localBox{}, false
	}
	H := halfExtents(in)
	var lb localBox
	for i := 0; i < 3; i++ {
		m := a
		for ax := 0; ax < 3; ax++ {
			m[ax][i] = H[ax]
		}
		lb.h[i] = det(m) / d
	}
	b := basis(in, transposed)
	var dc [3]float64
	for ax := 0; ax < 3; ax++ {
		dc[ax] = (in.AABBMin[ax]+in.AABBMax[ax])/2 - in.Position[ax]
	}
	for i := 0; i < 3; i++ {
		lb.c[i] = dc[0]*b[i][0] + dc[1]*b[i][1] + dc[2]*b[i][2]
	}
	return lb, true
}

// predictErr : plus grand écart, sur les 6 faces, entre l'AABB monde RECOMPOSÉE depuis
// la boîte locale + le placement de l'instance, et l'AABB monde STOCKÉE @0x7C.
func predictErr(in himap.Instance, lb localBox, transposed bool) float64 {
	b := basis(in, transposed)
	a := absMatOf(b)
	worst := 0.0
	for ax := 0; ax < 3; ax++ {
		center := in.Position[ax] + lb.c[0]*b[0][ax] + lb.c[1]*b[1][ax] + lb.c[2]*b[2][ax]
		half := a[ax][0]*lb.h[0] + a[ax][1]*lb.h[1] + a[ax][2]*lb.h[2]
		for _, e := range []float64{
			math.Abs(center - half - in.AABBMin[ax]),
			math.Abs(center + half - in.AABBMax[ax]),
		} {
			if e > worst {
				worst = e
			}
		}
	}
	return worst
}

func det(m [3][3]float64) float64 {
	return m[0][0]*(m[1][1]*m[2][2]-m[1][2]*m[2][1]) -
		m[0][1]*(m[1][0]*m[2][2]-m[1][2]*m[2][0]) +
		m[0][2]*(m[1][0]*m[2][1]-m[1][1]*m[2][0])
}
