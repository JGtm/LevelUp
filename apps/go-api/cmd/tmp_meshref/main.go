// tmp_meshref — anatomie du champ `Runtime geo mesh reference` (0x1C o) et
// distribution de l'erreur de l'oracle selon la taille / le groupe.
//
// Usage : CGO_ENABLED=1 go run ./cmd/tmp_meshref [module]
package main

import (
	"fmt"
	"math"
	"os"
	"sort"

	"levelup/go-api/internal/himap"
)

const defMod = `D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/pc/levels/multi/ridgeline/ridgeline-rtx-new.module`

func main() {
	p := defMod
	if len(os.Args) > 1 {
		p = os.Args[1]
	}
	bsps, err := himap.ReadModuleInstances(p)
	if err != nil {
		fmt.Println(err)
		return
	}
	var main *himap.BSPInstances
	bestVol := math.Inf(1)
	for i := range bsps {
		b := &bsps[i]
		if len(b.Instances) == 0 || !b.Bounds.Valid() {
			continue
		}
		v := b.Bounds.Extent(0) * b.Bounds.Extent(1) * b.Bounds.Extent(2)
		if v < bestVol {
			bestVol, main = v, b
		}
	}
	ins := main.Instances
	fmt.Printf("%d instances\n", len(ins))

	// Quels octets de meshRef varient ?
	varies := map[int]map[byte]bool{}
	for i := 0; i < 28; i++ {
		varies[i] = map[byte]bool{}
	}
	for _, in := range ins {
		for i := 0; i < 28; i++ {
			varies[i][in.MeshRef[i]] = true
		}
	}
	fmt.Print("  valeurs distinctes par octet de meshRef : ")
	for i := 0; i < 28; i++ {
		fmt.Printf("%d ", len(varies[i]))
	}
	fmt.Println()

	refs := map[[28]byte]int{}
	pairs := map[string]int{}
	idxs := map[int]int{}
	for _, in := range ins {
		refs[in.MeshRef]++
		pairs[in.MeshKey()]++
		idxs[in.MeshIndex]++
	}
	fmt.Printf("  meshRef distincts=%d ; (meshRef,meshIndex) distincts=%d ; meshIndex distincts=%d (max=%d)\n",
		len(refs), len(pairs), len(idxs), maxKey(idxs))
	fmt.Println("  5 premiers meshRef (hex) :")
	shown := 0
	for r, n := range refs {
		fmt.Printf("    %x  x%d\n", r, n)
		if shown++; shown >= 5 {
			break
		}
	}

	// Erreur de l'oracle en fonction de la taille du groupe et de la taille de l'AABB.
	byMesh := map[string][]int{}
	for i, in := range ins {
		byMesh[in.MeshKey()] = append(byMesh[in.MeshKey()], i)
	}
	type row struct {
		err, size float64
		grp       int
	}
	var rows []row
	for _, g := range byMesh {
		if len(g) < 2 {
			continue
		}
		src := g[0]
		lb, ok := solveLocalBox(ins[src])
		if !ok {
			continue
		}
		for _, j := range g[1:] {
			d := 0.0
			for a := 0; a < 3; a++ {
				if e := ins[j].AABBMax[a] - ins[j].AABBMin[a]; e > d {
					d = e
				}
			}
			rows = append(rows, row{predictErr(ins[j], lb), d, len(g)})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].err < rows[j].err })
	fmt.Printf("  n=%d ; quantiles d'erreur : p50=%.4f p75=%.4f p90=%.4f p99=%.4f\n",
		len(rows), rows[len(rows)/2].err, rows[len(rows)*3/4].err, rows[len(rows)*9/10].err, rows[len(rows)*99/100].err)
	// L'erreur est-elle portée par les gros objets ?
	var small, big []float64
	for _, r := range rows {
		if r.size < 5 {
			small = append(small, r.err)
		} else {
			big = append(big, r.err)
		}
	}
	sort.Float64s(small)
	sort.Float64s(big)
	if len(small) > 0 {
		fmt.Printf("  AABB < 5 m  : n=%d médiane err=%.4f p90=%.4f\n", len(small), small[len(small)/2], small[len(small)*9/10])
	}
	if len(big) > 0 {
		fmt.Printf("  AABB >= 5 m : n=%d médiane err=%.4f p90=%.4f\n", len(big), big[len(big)/2], big[len(big)*9/10])
	}
}

func maxKey(m map[int]int) int {
	mx := 0
	for k := range m {
		if k > mx {
			mx = k
		}
	}
	return mx
}

type localBox struct{ c, h [3]float64 }

func absMat(in himap.Instance) [3][3]float64 {
	b := [3][3]float64{in.Forward, in.Left, in.Up}
	var a [3][3]float64
	for ax := 0; ax < 3; ax++ {
		for i := 0; i < 3; i++ {
			a[ax][i] = math.Abs(b[i][ax])
		}
	}
	return a
}

func solveLocalBox(in himap.Instance) (localBox, bool) {
	a := absMat(in)
	d := det(a)
	if math.Abs(d) < 1e-6 {
		return localBox{}, false
	}
	var H [3]float64
	for ax := 0; ax < 3; ax++ {
		H[ax] = (in.AABBMax[ax] - in.AABBMin[ax]) / 2
	}
	var lb localBox
	for i := 0; i < 3; i++ {
		m := a
		for ax := 0; ax < 3; ax++ {
			m[ax][i] = H[ax]
		}
		lb.h[i] = det(m) / d
	}
	b := [3][3]float64{in.Forward, in.Left, in.Up}
	var dc [3]float64
	for ax := 0; ax < 3; ax++ {
		dc[ax] = (in.AABBMin[ax]+in.AABBMax[ax])/2 - in.Position[ax]
	}
	for i := 0; i < 3; i++ {
		lb.c[i] = dc[0]*b[i][0] + dc[1]*b[i][1] + dc[2]*b[i][2]
	}
	return lb, true
}

func predictErr(in himap.Instance, lb localBox) float64 {
	a := absMat(in)
	b := [3][3]float64{in.Forward, in.Left, in.Up}
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
