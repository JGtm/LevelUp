// tmp_mvarsym — outil jetable : TEMOIN de symetrie d'une carte.
//
// Protocole (aucune etape n'est tautologique) :
//  1. Le barycentre de TOUS les objets est un invariant de toute symetrie exacte.
//     On le calcule sans jamais regarder les objectifs.
//  2. On teste 3 transformations candidates autour de ce barycentre : miroir x,
//     miroir y, rotation 180 verticale. Pour chacune, on mesure le taux d'objets
//     dont l'image tombe a moins de 0.5 m d'un objet de MEME type_id.
//     -> etablit SI la carte est symetrique, et selon quelle transformation.
//  3. On mesure alors l'ecart a la symetrie des paires d'objectif d'equipe.
//  4. TEMOIN NEGATIF : meme mesure sur des paires d'objets tirees au hasard.
package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

type xform struct {
	name string
	fn   func(p mapvar.Vec3, c mapvar.Vec3) mapvar.Vec3
}

var xforms = []xform{
	{"miroir_x", func(p, c mapvar.Vec3) mapvar.Vec3 {
		return mapvar.Vec3{X: 2*c.X - p.X, Y: p.Y, Z: p.Z}
	}},
	{"miroir_y", func(p, c mapvar.Vec3) mapvar.Vec3 {
		return mapvar.Vec3{X: p.X, Y: 2*c.Y - p.Y, Z: p.Z}
	}},
	{"rot180_z", func(p, c mapvar.Vec3) mapvar.Vec3 {
		return mapvar.Vec3{X: 2*c.X - p.X, Y: 2*c.Y - p.Y, Z: p.Z}
	}},
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: tmp_mvarsym <fichier.mvar>")
		os.Exit(2)
	}
	buf, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println("erreur:", err)
		os.Exit(1)
	}
	v, err := mapvar.Parse(buf)
	if err != nil {
		fmt.Println("erreur:", err)
		os.Exit(1)
	}
	fmt.Printf("fichier=%s objets=%d\n", os.Args[1], len(v.Objects))

	c := barycentre(v.Objects)
	fmt.Printf("barycentre (tous objets, aucune connaissance des objectifs) = (%.3f, %.3f, %.3f)\n",
		c.X, c.Y, c.Z)

	byType := map[int32][]mapvar.Object{}
	for _, o := range v.Objects {
		byType[o.TypeID] = append(byType[o.TypeID], o)
	}

	fmt.Println("--- ETAPE 2 : la carte est-elle symetrique ? ---")
	best := ""
	bestRate := 0.0
	for _, x := range xforms {
		hit, med := matchRate(v.Objects, byType, c, x.fn)
		fmt.Printf("  %-10s appariement<0.5m = %5.1f %%   ecart median = %7.3f m\n",
			x.name, 100*hit, med)
		if hit > bestRate {
			bestRate, best = hit, x.name
		}
	}
	fmt.Printf("  => meilleure transformation : %s (%.1f %%)\n", best, 100*bestRate)

	fmt.Println("--- ETAPE 2bis : recherche du centre par vote (aucun centre suppose) ---")
	hough(v.Objects, byType)

	fmt.Println("--- ETAPE 3 : ecart a la symetrie des paires d'objectif d'equipe ---")
	objs := v.Objectives()
	pairs := teamPairs(objs)
	if len(pairs) == 0 {
		fmt.Println("  aucune paire d'objectif d'equipe sur cette carte")
	}
	for _, p := range pairs {
		fmt.Printf("  %-20s type=%11d\n", p.role, p.typeID)
		fmt.Printf("      equipe0 (%8.3f,%8.3f,%8.3f)  equipe1 (%8.3f,%8.3f,%8.3f)\n",
			p.a.X, p.a.Y, p.a.Z, p.b.X, p.b.Y, p.b.Z)
		for _, x := range xforms {
			img := x.fn(p.a, c)
			fmt.Printf("      %-10s ecart |img(eq0) - eq1| = %8.4f m  (dz=%7.4f)\n",
				x.name, dist(img, p.b), math.Abs(p.a.Z-p.b.Z))
		}
		fmt.Printf("      centre implicite de la paire = (%.3f, %.3f)\n",
			(p.a.X+p.b.X)/2, (p.a.Y+p.b.Y)/2)
	}

	fmt.Println("--- ETAPE 4 : TEMOIN NEGATIF (paires aleatoires, meme protocole) ---")
	rng := rand.New(rand.NewSource(20260726))
	var neg []float64
	for i := 0; i < 2000; i++ {
		a := v.Objects[rng.Intn(len(v.Objects))]
		b := v.Objects[rng.Intn(len(v.Objects))]
		neg = append(neg, dist(xforms[1].fn(a.Pos, c), b.Pos))
	}
	sort.Float64s(neg)
	fmt.Printf("  ecart miroir_y de 2000 paires aleatoires : p05=%.2f m  median=%.2f m  p95=%.2f m\n",
		neg[100], neg[1000], neg[1900])
}

func barycentre(objs []mapvar.Object) mapvar.Vec3 {
	var s mapvar.Vec3
	for _, o := range objs {
		s.X += o.Pos.X
		s.Y += o.Pos.Y
		s.Z += o.Pos.Z
	}
	n := float64(len(objs))
	return mapvar.Vec3{X: s.X / n, Y: s.Y / n, Z: s.Z / n}
}

func matchRate(objs []mapvar.Object, byType map[int32][]mapvar.Object,
	c mapvar.Vec3, f func(mapvar.Vec3, mapvar.Vec3) mapvar.Vec3) (float64, float64) {
	hits := 0
	var ds []float64
	for _, o := range objs {
		img := f(o.Pos, c)
		bestD := math.Inf(1)
		for _, cand := range byType[o.TypeID] {
			if d := dist(img, cand.Pos); d < bestD {
				bestD = d
			}
		}
		ds = append(ds, bestD)
		if bestD < 0.5 {
			hits++
		}
	}
	sort.Float64s(ds)
	return float64(hits) / float64(len(objs)), ds[len(ds)/2]
}

type pair struct {
	role   mapvar.Role
	typeID int32
	a, b   mapvar.Vec3
}

// teamPairs apparie, pour chaque (role, type_id), le barycentre des objets
// d'equipe 0 et celui des objets d'equipe 1.
func teamPairs(objs []mapvar.Objective) []pair {
	type key struct {
		r mapvar.Role
		t int32
	}
	acc := map[key][2][]mapvar.Vec3{}
	for _, o := range objs {
		if o.TeamIndex != 0 && o.TeamIndex != 1 {
			continue
		}
		k := key{o.Role, o.TypeID}
		cur := acc[k]
		cur[o.TeamIndex] = append(cur[o.TeamIndex], o.Pos)
		acc[k] = cur
	}
	var out []pair
	for k, v := range acc {
		if len(v[0]) == 0 || len(v[1]) == 0 {
			continue
		}
		out = append(out, pair{role: k.r, typeID: k.t, a: mean(v[0]), b: mean(v[1])})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].role < out[j].role })
	return out
}

func mean(v []mapvar.Vec3) mapvar.Vec3 {
	var s mapvar.Vec3
	for _, p := range v {
		s.X += p.X
		s.Y += p.Y
		s.Z += p.Z
	}
	n := float64(len(v))
	return mapvar.Vec3{X: s.X / n, Y: s.Y / n, Z: s.Z / n}
}

func dist(a, b mapvar.Vec3) float64 {
	dx, dy, dz := a.X-b.X, a.Y-b.Y, a.Z-b.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// hough — recherche du centre de symetrie SANS le supposer : chaque paire d'objets
// de meme type_id compatible avec la transformation vote pour le centre qu'elle
// implique. Un pic net = la carte est symetrique. Un histogramme plat = elle ne
// l'est pas, et aucun choix de centre ne peut la sauver.
func hough(objs []mapvar.Object, byType map[int32][]mapvar.Object) {
	const bin = 0.25
	voteY := map[int]int{} // miroir_y : centre cy
	voteX := map[int]int{} // miroir_x : centre cx
	type cell struct{ i, j int }
	voteR := map[cell]int{} // rot180 : centre (cx,cy)
	pairs := 0
	for _, a := range objs {
		for _, b := range byType[a.TypeID] {
			if b.Index <= a.Index {
				continue
			}
			pairs++
			if math.Abs(a.Pos.Z-b.Pos.Z) < 0.5 {
				if math.Abs(a.Pos.X-b.Pos.X) < 0.5 {
					voteY[int(math.Round((a.Pos.Y+b.Pos.Y)/2/bin))]++
				}
				if math.Abs(a.Pos.Y-b.Pos.Y) < 0.5 {
					voteX[int(math.Round((a.Pos.X+b.Pos.X)/2/bin))]++
				}
				voteR[cell{int(math.Round((a.Pos.X + b.Pos.X) / 2 / bin)),
					int(math.Round((a.Pos.Y + b.Pos.Y) / 2 / bin))}]++
			}
		}
	}
	fmt.Printf("  paires de meme type_id examinees : %d\n", pairs)
	report1D("miroir_y (centre cy)", voteY, bin)
	report1D("miroir_x (centre cx)", voteX, bin)
	bestN, total := 0, 0
	var bc cell
	for c, n := range voteR {
		total += n
		if n > bestN {
			bestN, bc = n, c
		}
	}
	fmt.Printf("  rot180_z  : pic %d votes en (%.2f, %.2f) sur %d votes repartis en %d cellules (fond moyen %.2f)\n",
		bestN, float64(bc.i)*bin, float64(bc.j)*bin, total, len(voteR),
		float64(total)/math.Max(1, float64(len(voteR))))
}

func report1D(label string, votes map[int]int, bin float64) {
	bestN, total := 0, 0
	best := 0
	for k, n := range votes {
		total += n
		if n > bestN {
			bestN, best = n, k
		}
	}
	fmt.Printf("  %-22s pic %d votes en %.2f m sur %d votes / %d cases (fond moyen %.2f)\n",
		label, bestN, float64(best)*bin, total, len(votes),
		float64(total)/math.Max(1, float64(len(votes))))
}
