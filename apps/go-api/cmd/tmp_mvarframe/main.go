// tmp_mvarframe — outil jetable : VERIFICATION DE REPERE.
//
// Confronte trois sources independantes de coordonnees pour la meme carte :
//
//	(a) les objets de la variante .mvar
//	(b) les bornes monde du BSP (data/.../reference/map_quant_bounds.json)
//	(c) les positions joueur decodees du film (.ai/V7.5/dumps/offline_trajectories*.csv)
//
// Si les trois se recouvrent sans facteur d'echelle ni translation, le repere est
// commun. Sinon il manque une transformation — et on le DIT, on ne la fabrique pas.
package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

type quantFile struct {
	Maps map[string]struct {
		Module string    `json:"module"`
		Min    []float64 `json:"min"`
		Max    []float64 `json:"max"`
	} `json:"maps"`
}

type box struct{ min, max [3]float64 }

func (b box) String() string {
	return fmt.Sprintf("x[%8.2f..%8.2f] y[%8.2f..%8.2f] z[%8.2f..%8.2f]",
		b.min[0], b.max[0], b.min[1], b.max[1], b.min[2], b.max[2])
}

func (b box) contains(p [3]float64, tol float64) bool {
	for i := 0; i < 3; i++ {
		if p[i] < b.min[i]-tol || p[i] > b.max[i]+tol {
			return false
		}
	}
	return true
}

func main() {
	if len(os.Args) < 5 {
		fmt.Println("usage: tmp_mvarframe <mvar> <map_quant_bounds.json> <cle_carte> <trajectoires.csv>")
		os.Exit(2)
	}
	buf, err := os.ReadFile(os.Args[1])
	must(err)
	v, err := mapvar.Parse(buf)
	must(err)

	// (a) objets .mvar
	pts := make([][3]float64, 0, len(v.Objects))
	for _, o := range v.Objects {
		pts = append(pts, [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z})
	}
	mvarBox := bounds(pts)
	fmt.Printf("(a) objets .mvar      n=%4d  %s\n", len(pts), mvarBox)

	// (b) bornes BSP
	raw, err := os.ReadFile(os.Args[2])
	must(err)
	var qf quantFile
	must(json.Unmarshal(raw, &qf))
	e, ok := qf.Maps[os.Args[3]]
	if !ok {
		fmt.Printf("carte %q absente de map_quant_bounds.json\n", os.Args[3])
		os.Exit(1)
	}
	bsp := box{min: [3]float64{e.Min[0], e.Min[1], e.Min[2]},
		max: [3]float64{e.Max[0], e.Max[1], e.Max[2]}}
	fmt.Printf("(b) BSP %-12s        %s\n", e.Module, bsp)

	// (c) positions joueur
	traj := readTraj(os.Args[4])
	trajBox := bounds(traj)
	fmt.Printf("(c) positions joueur  n=%4d  %s\n", len(traj), trajBox)

	fmt.Println("--- recouvrement ---")
	inBSP := 0
	for _, p := range pts {
		if bsp.contains(p, 0) {
			inBSP++
		}
	}
	fmt.Printf("objets .mvar dans les bornes BSP : %d/%d (%.1f %%)\n",
		inBSP, len(pts), 100*float64(inBSP)/float64(len(pts)))

	inT := 0
	for _, p := range traj {
		if bsp.contains(p, 0) {
			inT++
		}
	}
	fmt.Printf("positions joueur dans les bornes BSP : %d/%d (%.1f %%)\n",
		inT, len(traj), 100*float64(inT)/float64(len(traj)))

	// Distance de chaque position joueur a l'objet .mvar le plus proche : si les
	// reperes divergent d'une translation, cette distance explose uniformement.
	var d []float64
	for i, p := range traj {
		if i%37 != 0 {
			continue
		}
		best := math.Inf(1)
		for _, q := range pts {
			if dd := dist3(p, q); dd < best {
				best = dd
			}
		}
		d = append(d, best)
	}
	sort.Float64s(d)
	if len(d) > 0 {
		fmt.Printf("distance position joueur -> objet .mvar le plus proche (n=%d) : p05=%.2f m median=%.2f m p95=%.2f m\n",
			len(d), d[len(d)/20], d[len(d)/2], d[len(d)*19/20])
	}

	fmt.Println("--- objectifs de cette variante ---")
	for _, o := range v.Objectives() {
		fmt.Printf("  %-20s equipe=%2d type=%12d pos=(%8.3f,%8.3f,%8.3f) dansBSP=%v labels=%v\n",
			o.Role, o.TeamIndex, o.TypeID, o.Pos.X, o.Pos.Y, o.Pos.Z,
			bsp.contains([3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z}, 0), o.Labels)
	}
}

func readTraj(path string) [][3]float64 {
	f, err := os.Open(path)
	must(err)
	defer func() { _ = f.Close() }()
	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	must(err)
	if len(rows) < 2 {
		return nil
	}
	idx := map[string]int{}
	for i, h := range rows[0] {
		idx[h] = i
	}
	var out [][3]float64
	for _, row := range rows[1:] {
		x, e1 := strconv.ParseFloat(row[idx["x"]], 64)
		y, e2 := strconv.ParseFloat(row[idx["y"]], 64)
		z, e3 := strconv.ParseFloat(row[idx["z"]], 64)
		if e1 != nil || e2 != nil || e3 != nil {
			continue
		}
		out = append(out, [3]float64{x, y, z})
	}
	return out
}

func bounds(pts [][3]float64) box {
	if len(pts) == 0 {
		return box{}
	}
	b := box{min: pts[0], max: pts[0]}
	for _, p := range pts {
		for i := 0; i < 3; i++ {
			if p[i] < b.min[i] {
				b.min[i] = p[i]
			}
			if p[i] > b.max[i] {
				b.max[i] = p[i]
			}
		}
	}
	return b
}

func dist3(a, b [3]float64) float64 {
	dx, dy, dz := a[0]-b[0], a[1]-b[1], a[2]-b[2]
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func must(err error) {
	if err != nil {
		fmt.Println("erreur:", err)
		os.Exit(1)
	}
}
