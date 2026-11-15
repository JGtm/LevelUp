package tactical

// ligne_de_vue.go — UNE LIGNE DE VUE APPROCHEE, a partir des boites de la structure de la carte.
//
// # CE QUE C'EST, ET CE QUE CE N'EST PAS
//
// La structure d'un artefact de rejeu decrit la carte par des BOITES : une emprise au sol
// (x0, y0, x1, y1) et deux faces, haute et basse. Un ennemi « voit » un point quand le segment
// qui va de ses yeux au point ne traverse aucune boite. C'est une APPROXIMATION : une boite
// alignee sur les axes deborde d'une piece tournee, une rampe n'est qu'une boite, et le vide
// d'une arche n'existe pas. Elle sert a MESURER si la visibilite ajoute quelque chose a la
// prevision, pas a dire ce qui se voit.
//
// Les boites qui CONTIENNENT une extremite du segment sont ignorees : un segment commence et
// finit dans l'air libre par construction, et une boite qui l'enferme est un artefact de
// l'emprise alignee (un mur tourne dont la boite englobe toute la piece).
//
// PUR : aucune I/O.

import "math"

// Hauteurs de la ligne de vue, en metres au-dessus de la position (le pied).
const (
	// HauteurYeuxM : les yeux de l'ennemi.
	HauteurYeuxM = 1.6
	// HauteurCibleM : le torse du joueur qui reapparait.
	HauteurCibleM = 1.0
	// CelluleObstaclesM : la maille de l'index des boites.
	CelluleObstaclesM = 8.0
)

// Boite est une boite de la structure : emprise au sol et altitudes des faces.
type Boite struct {
	X0, Y0, X1, Y1 float64
	ZHaut, ZBas    float64
}

// Obstacles indexe les boites d'une carte pour repondre vite a « ce segment est-il libre ».
// Il est en lecture seule apres construction, donc partageable entre goroutines.
type Obstacles struct {
	boites []Boite
	grille map[[2]int][]int32
}

// NouveauxObstacles indexe les boites sur une grille de CelluleObstaclesM. Une boite degeneree
// (non finie, ou emprise vide) est ecartee.
func NouveauxObstacles(boites []Boite) *Obstacles {
	o := &Obstacles{grille: map[[2]int][]int32{}}
	for _, b := range boites {
		if !finie(b) || b.X1 <= b.X0 || b.Y1 <= b.Y0 {
			continue
		}
		if b.ZBas > b.ZHaut {
			b.ZBas, b.ZHaut = b.ZHaut, b.ZBas
		}
		id := int32(len(o.boites))
		o.boites = append(o.boites, b)
		cx0, cy0 := celluleObstacle(b.X0), celluleObstacle(b.Y0)
		cx1, cy1 := celluleObstacle(b.X1), celluleObstacle(b.Y1)
		for cx := cx0; cx <= cx1; cx++ {
			for cy := cy0; cy <= cy1; cy++ {
				k := [2]int{cx, cy}
				o.grille[k] = append(o.grille[k], id)
			}
		}
	}
	return o
}

func finie(b Boite) bool {
	for _, v := range []float64{b.X0, b.Y0, b.X1, b.Y1, b.ZHaut, b.ZBas} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}

func celluleObstacle(v float64) int { return int(math.Floor(v / CelluleObstaclesM)) }

// NbBoites rend le nombre de boites retenues.
func (o *Obstacles) NbBoites() int { return len(o.boites) }

// Voit dit si l'ennemi (position du pied) voit la cible (position du pied) : le segment des yeux
// de l'ennemi au torse de la cible ne traverse aucune boite.
func (o *Obstacles) Voit(ennemi, cible Point3) bool {
	a := Point3{ennemi.X, ennemi.Y, ennemi.Z + HauteurYeuxM}
	b := Point3{cible.X, cible.Y, cible.Z + HauteurCibleM}
	cx0, cx1 := minmaxCellule(a.X, b.X)
	cy0, cy1 := minmaxCellule(a.Y, b.Y)
	for cx := cx0; cx <= cx1; cx++ {
		for cy := cy0; cy <= cy1; cy++ {
			for _, id := range o.grille[[2]int{cx, cy}] {
				bt := &o.boites[id]
				if contient(bt, a) || contient(bt, b) {
					continue
				}
				if segmentTraverse(bt, a, b) {
					return false
				}
			}
		}
	}
	return true
}

func minmaxCellule(u, v float64) (int, int) {
	lo, hi := celluleObstacle(u), celluleObstacle(v)
	if lo > hi {
		lo, hi = hi, lo
	}
	return lo, hi
}

func contient(b *Boite, p Point3) bool {
	return p.X >= b.X0 && p.X <= b.X1 && p.Y >= b.Y0 && p.Y <= b.Y1 && p.Z >= b.ZBas && p.Z <= b.ZHaut
}

// segmentTraverse : test segment / boite par intervalles (methode des plaques).
func segmentTraverse(b *Boite, a, e Point3) bool {
	t0, t1 := 0.0, 1.0
	axes := [3][2]float64{{a.X, e.X - a.X}, {a.Y, e.Y - a.Y}, {a.Z, e.Z - a.Z}}
	bornes := [3][2]float64{{b.X0, b.X1}, {b.Y0, b.Y1}, {b.ZBas, b.ZHaut}}
	for i := 0; i < 3; i++ {
		o, d := axes[i][0], axes[i][1]
		if d == 0 {
			if o < bornes[i][0] || o > bornes[i][1] {
				return false
			}
			continue
		}
		ta, tb := (bornes[i][0]-o)/d, (bornes[i][1]-o)/d
		if ta > tb {
			ta, tb = tb, ta
		}
		t0, t1 = math.Max(t0, ta), math.Min(t1, tb)
		if t0 > t1 {
			return false
		}
	}
	return true
}
