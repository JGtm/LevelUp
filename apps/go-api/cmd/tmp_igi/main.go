// tmp_igi — sonde le bloc `instanced geometry instances` des tags sbsp d'un module
// et exécute l'ORACLE INTERNE : AABB recomposée depuis la base 3x3 vs AABB stockée.
//
// Usage : CGO_ENABLED=1 go run ./cmd/tmp_igi [module...]
package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/himap"
)

var defaults = []string{
	`D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/ds/levels/multi/ridgeline/ridgeline-rtx-new.module`,
	`D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/ds/levels/multi/catalyst/catalyst-rtx-new.module`,
	`D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/ds/levels/multi/ctf_aquarius/ctf_aquarius-rtx-new.module`,
	`D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/ds/levels/multi/ctf_bazaar/ctf_bazaar-rtx-new.module`,
	`D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/any/levels/multi/ridgeline/ridgeline-rtx-new.module`,
}

func main() {
	if s, err := himap.PluginInstanceStride(); err != nil {
		fmt.Println("stride plugin:", err)
	} else {
		fmt.Printf("stride enregistrement recalculé depuis le plugin = %#x (attendu 0x140)\n\n", s)
	}
	mods := defaults
	if len(os.Args) > 1 {
		mods = os.Args[1:]
	}
	for _, p := range mods {
		name := filepath.Base(filepath.Dir(p))
		bsps, err := himap.ReadModuleInstances(p)
		if err != nil {
			fmt.Printf("%-16s ERR %v\n", name, err)
			continue
		}
		for i, b := range bsps {
			fmt.Printf("%-16s sbsp#%d (%d o) : %d instances ; bounds x[%.1f,%.1f] y[%.1f,%.1f] z[%.1f,%.1f]\n",
				name, b.FileIndex, b.UncompSize, len(b.Instances),
				b.Bounds.Min[0], b.Bounds.Max[0], b.Bounds.Min[1], b.Bounds.Max[1],
				b.Bounds.Min[2], b.Bounds.Max[2])
			if i == 0 && len(b.Instances) > 0 {
				dumpSamples(b)
			}
		}
	}
}

func dumpSamples(b himap.BSPInstances) {
	fmt.Println("  --- 3 premières instances ---")
	for i := 0; i < 3 && i < len(b.Instances); i++ {
		in := b.Instances[i]
		fmt.Printf("  #%d pos=(%.2f %.2f %.2f) fwd=(%.3f %.3f %.3f) left=(%.3f %.3f %.3f) up=(%.3f %.3f %.3f)\n",
			in.Index, in.Position[0], in.Position[1], in.Position[2],
			in.Forward[0], in.Forward[1], in.Forward[2],
			in.Left[0], in.Left[1], in.Left[2], in.Up[0], in.Up[1], in.Up[2])
		fmt.Printf("      aabb=[%.2f,%.2f]x[%.2f,%.2f]x[%.2f,%.2f] sphere=(%.2f %.2f %.2f) r=%.2f mesh=%d io=%d fl=%#x fl2=%#x\n",
			in.AABBMin[0], in.AABBMax[0], in.AABBMin[1], in.AABBMax[1], in.AABBMin[2], in.AABBMax[2],
			in.Sphere[0], in.Sphere[1], in.Sphere[2], in.Radius, in.MeshIndex, in.UniqueIO, in.Flags, in.Flags2)
	}
	// contrôles de cohérence rapides
	orth, nQD, nOut := 0, 0, 0
	var scales []float64
	for _, in := range b.Instances {
		if isOrthoBasis(in) {
			orth++
		}
		if in.QuickDeleted() {
			nQD++
		}
		if !insideBounds(in, b) {
			nOut++
		}
		scales = append(scales, norm(in.Forward))
	}
	sort.Float64s(scales)
	fmt.Printf("  base orthogonale (à 1e-3) : %d/%d ; quick-deleted : %d ; hors bornes : %d ; |forward| médian=%.4f min=%.4f max=%.4f\n",
		orth, len(b.Instances), nQD, nOut, scales[len(scales)/2], scales[0], scales[len(scales)-1])
	// sphère : centre ≈ centre de l'AABB ? rayon ≈ demi-diagonale ?
	var dc, dr []float64
	for _, in := range b.Instances {
		var c [3]float64
		d := 0.0
		for a := 0; a < 3; a++ {
			c[a] = (in.AABBMin[a] + in.AABBMax[a]) / 2
			h := (in.AABBMax[a] - in.AABBMin[a]) / 2
			d += h * h
		}
		dc = append(dc, dist(c, in.Sphere))
		dr = append(dr, math.Abs(math.Sqrt(d)-in.Radius))
	}
	sort.Float64s(dc)
	sort.Float64s(dr)
	fmt.Printf("  sphère vs AABB : |centre-centre| médian=%.4f p95=%.4f ; |r - demi-diag| médian=%.4f p95=%.4f\n",
		dc[len(dc)/2], dc[len(dc)*95/100], dr[len(dr)/2], dr[len(dr)*95/100])
}

func isOrthoBasis(in himap.Instance) bool {
	d := func(a, b [3]float64) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
	nf, nl, nu := norm(in.Forward), norm(in.Left), norm(in.Up)
	if nf == 0 || nl == 0 || nu == 0 {
		return false
	}
	return math.Abs(d(in.Forward, in.Left))/(nf*nl) < 1e-3 &&
		math.Abs(d(in.Forward, in.Up))/(nf*nu) < 1e-3 &&
		math.Abs(d(in.Left, in.Up))/(nl*nu) < 1e-3
}

func norm(v [3]float64) float64 { return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2]) }

func dist(a, b [3]float64) float64 {
	s := 0.0
	for i := 0; i < 3; i++ {
		s += (a[i] - b[i]) * (a[i] - b[i])
	}
	return math.Sqrt(s)
}

func insideBounds(in himap.Instance, b himap.BSPInstances) bool {
	for a := 0; a < 3; a++ {
		if in.Position[a] < b.Bounds.Min[a] || in.Position[a] > b.Bounds.Max[a] {
			return false
		}
	}
	return true
}
