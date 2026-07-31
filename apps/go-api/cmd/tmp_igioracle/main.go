// tmp_igioracle — ORACLE INTERNE de la transformation de placement des instances.
//
// PRINCIPE. Une instance porte position + base 3x3 (forward/left/up, échelle bakée) et
// son AABB MONDE (@0x7C). Si la transformation lue est correcte, l'AABB monde d'une
// boîte locale de demi-extensions h vaut, en convention vecteur-ligne :
//
//	H[a] = h0*|forward[a]| + h1*|left[a]| + h2*|up[a]|      (centre = position)
//
// h ne dépend QUE du mesh. On résout donc h sur UNE instance d'un mesh, puis on PRÉDIT
// l'AABB de TOUTES les autres instances du même mesh — qui ont d'autres rotations.
// C'est falsifiable et gratuit : aucune donnée externe, aucun paramètre ajusté.
//
// TÉMOIN : la même prédiction en appariant h à un mesh TIRÉ AU HASARD.
//
// Usage : CGO_ENABLED=1 go run ./cmd/tmp_igioracle [module...]
package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/himap"
)

var defaults = []string{
	`D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/pc/levels/multi/ridgeline/ridgeline-rtx-new.module`,
	`D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/pc/levels/multi/catalyst/catalyst-rtx-new.module`,
	`D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/pc/levels/multi/ctf_aquarius/ctf_aquarius-rtx-new.module`,
	`D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/pc/levels/multi/sgh_streets/sgh_streets-rtx-new.module`,
}

func main() {
	mods := defaults
	if len(os.Args) > 1 {
		mods = os.Args[1:]
	}
	for _, p := range mods {
		name := filepath.Base(filepath.Dir(p))
		bsps, err := himap.ReadModuleInstances(p)
		if err != nil {
			fmt.Printf("%-14s ERR %v\n", name, err)
			continue
		}
		main := pickMain(bsps)
		if main == nil {
			fmt.Printf("%-14s aucun BSP principal\n", name)
			continue
		}
		fmt.Printf("\n===== %s : %d instances (sbsp#%d) =====\n", name, len(main.Instances), main.FileIndex)
		scaleCheck(main.Instances)
		centerCheck(main.Instances)
		sphereSandwich(main.Instances)
		oracle(main.Instances)
	}
}

// pickMain : le BSP dont les bornes sont les plus PETITES parmi ceux qui portent des
// instances — le BSP de skybox a des bornes de plusieurs kilomètres.
func pickMain(bsps []himap.BSPInstances) *himap.BSPInstances {
	var best *himap.BSPInstances
	bestVol := math.Inf(1)
	for i := range bsps {
		b := &bsps[i]
		if len(b.Instances) == 0 || !b.Bounds.Valid() {
			continue
		}
		v := b.Bounds.Extent(0) * b.Bounds.Extent(1) * b.Bounds.Extent(2)
		if v < bestVol {
			bestVol, best = v, b
		}
	}
	return best
}

// centerCheck : écart entre le centre de l'AABB monde et la position (le centre local
// du mesh n'est PAS forcément son origine, donc un écart non nul est attendu).
func centerCheck(ins []himap.Instance) {
	var d []float64
	for _, in := range ins {
		s := 0.0
		for a := 0; a < 3; a++ {
			c := (in.AABBMin[a] + in.AABBMax[a]) / 2
			s += (c - in.Position[a]) * (c - in.Position[a])
		}
		d = append(d, math.Sqrt(s))
	}
	sort.Float64s(d)
	fmt.Printf("  centre(AABB) vs position : médian=%.6f p95=%.6f max=%.6f\n",
		d[len(d)/2], d[len(d)*95/100], d[len(d)-1])
}

// scaleCheck : normes de la base et contenu du champ scale @0x00.
func scaleCheck(ins []himap.Instance) {
	var nf, sc []float64
	nonUnitScale, leftHanded := 0, 0
	for _, in := range ins {
		nf = append(nf, norm3(in.Forward))
		for a := 0; a < 3; a++ {
			sc = append(sc, in.Scale[a])
		}
		if math.Abs(in.Scale[0]-1) > 1e-4 || math.Abs(in.Scale[1]-1) > 1e-4 || math.Abs(in.Scale[2]-1) > 1e-4 {
			nonUnitScale++
		}
		if det([3][3]float64{in.Forward, in.Left, in.Up}) < 0 {
			leftHanded++
		}
	}
	sort.Float64s(nf)
	sort.Float64s(sc)
	fmt.Printf("  |forward| min=%.4f médian=%.4f max=%.4f ; scale@0x00 min=%.4f médian=%.4f max=%.4f ; non-unitaire=%d ; base gauchère=%d\n",
		nf[0], nf[len(nf)/2], nf[len(nf)-1], sc[0], sc[len(sc)/2], sc[len(sc)-1], nonUnitScale, leftHanded)
}

func norm3(v [3]float64) float64 { return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2]) }

// sphereSandwich : ORACLE PAR INSTANCE, sans aucun appariement. La sphère englobante
// (centre @0x94, rayon @0xA0) et l'AABB (@0x7C) bornent le MÊME maillage, donc
//
//	max_a demi-extension[a]  <=  rayon  <=  demi-diagonale de l'AABB.
//
// TÉMOIN : le même encadrement avec les rayons permutés entre instances.
func sphereSandwich(ins []himap.Instance) {
	okReal, okShuf, n := 0, 0, 0
	tooSmall, tooBig := 0, 0
	var excess []float64
	var offCenter []float64
	radii := make([]float64, len(ins))
	for i, in := range ins {
		radii[i] = in.Radius
	}
	rng := rand.New(rand.NewSource(11))
	rng.Shuffle(len(radii), func(i, j int) { radii[i], radii[j] = radii[j], radii[i] })
	for i, in := range ins {
		var lo, diag, dc float64
		for a := 0; a < 3; a++ {
			h := (in.AABBMax[a] - in.AABBMin[a]) / 2
			if h > lo {
				lo = h
			}
			diag += h * h
			c := (in.AABBMin[a] + in.AABBMax[a]) / 2
			dc += (c - in.Sphere[a]) * (c - in.Sphere[a])
		}
		diag = math.Sqrt(diag)
		if diag <= 0 || math.IsNaN(diag) {
			continue
		}
		n++
		offCenter = append(offCenter, math.Sqrt(dc))
		const tol = 1e-3
		switch {
		case in.Radius < lo-tol:
			tooSmall++
		case in.Radius > diag+tol:
			tooBig++
			excess = append(excess, in.Radius/diag)
		default:
			okReal++
		}
		if radii[i] >= lo-tol && radii[i] <= diag+tol {
			okShuf++
		}
	}
	sort.Float64s(offCenter)
	sort.Float64s(excess)
	exMed := 0.0
	if len(excess) > 0 {
		exMed = excess[len(excess)/2]
	}
	fmt.Printf("  SANDWICH sphère/AABB : n=%d ; réel %.1f%% ; témoin rayons permutés %.1f%% ; rayon trop petit %.1f%% / trop grand %.1f%% (r/demi-diag médian=%.2f) ; |centre sphère - centre AABB| médian=%.5f p95=%.5f\n",
		n, 100*float64(okReal)/float64(n), 100*float64(okShuf)/float64(n),
		100*float64(tooSmall)/float64(n), 100*float64(tooBig)/float64(n), exMed,
		offCenter[len(offCenter)/2], offCenter[len(offCenter)*95/100])
}

type group struct {
	key string
	idx []int
}

func oracle(ins []himap.Instance) {
	byMesh := map[string][]int{}
	for i, in := range ins {
		byMesh[in.MeshKey()] = append(byMesh[in.MeshKey()], i)
	}
	var groups []group
	for k, v := range byMesh {
		if len(v) >= 2 {
			groups = append(groups, group{k, v})
		}
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].key < groups[j].key })
	nMulti := 0
	for _, g := range groups {
		nMulti += len(g.idx)
	}
	fmt.Printf("  %d meshes distincts ; %d groupes multi-instances couvrant %d instances\n",
		len(byMesh), len(groups), nMulti)
	if len(groups) == 0 {
		return
	}
	// Sur une base QUELCONQUE, l'AABB monde de la géométrie réelle est STRICTEMENT
	// incluse dans celle de la boîte locale pivotée : recomposer donne un majorant, pas
	// une égalité. L'égalité n'est exacte que si la base est une permutation signée
	// (rotation multiple de 90°). L'oracle se fait donc en deux volets.
	nAxis := 0
	for _, in := range ins {
		if axisAligned(in) {
			nAxis++
		}
	}
	fmt.Printf("  bases alignées sur les axes (permutation signée) : %d/%d (%.1f%%)\n",
		nAxis, len(ins), 100*float64(nAxis)/float64(len(ins)))

	var errs, wErrs, sameOri, trusted []float64
	tested, ok01, ok05 := 0, 0, 0
	models := make([]localBox, 0, len(groups))
	for _, g := range groups {
		var axis []int
		for _, i := range g.idx {
			if axisAligned(ins[i]) {
				axis = append(axis, i)
			}
		}
		if len(axis) < 2 {
			continue
		}
		src := axis[0]
		lb, ok := solveLocalBox(ins[src], false)
		if !ok {
			continue
		}
		models = append(models, lb)
		// CONFONDANT : (meshRef, meshIndex) n'identifie pas complètement la géométrie
		// (surcharges de style/matériau/LOD). On le MESURE sur les paires de MÊME
		// orientation — où la transformation ne joue aucun rôle — et on ne fait
		// confiance au groupe que si ces paires concordent.
		keyOK := true
		for _, j := range axis[1:] {
			if !sameOrientation(ins[src], ins[j]) {
				continue
			}
			e := predictErr(ins[j], lb, false)
			sameOri = append(sameOri, e)
			if e > 0.001 {
				keyOK = false
			}
		}
		for _, j := range axis[1:] {
			if sameOrientation(ins[src], ins[j]) {
				continue
			}
			e := predictErr(ins[j], lb, false)
			errs = append(errs, e)
			tested++
			if e <= 0.01 {
				ok01++
			}
			if e <= 0.05 {
				ok05++
			}
			if keyOK {
				trusted = append(trusted, e)
			}
		}
	}
	if len(sameOri) > 0 {
		sort.Float64s(sameOri)
		report("  CONFONDANT clé de mesh (MÊME orientation)  ", sameOri, len(sameOri),
			countLE(sameOri, 0.01), countLE(sameOri, 0.05))
	}
	report("  ORACLE tous groupes (orient. différentes)  ", errs, tested, ok01, ok05)
	if len(trusted) > 0 {
		sort.Float64s(trusted)
		report("  ORACLE groupes à clé vérifiée              ", trusted, len(trusted),
			countLE(trusted, 0.01), countLE(trusted, 0.05))
	}

	// TÉMOIN : boîte locale d'un mesh TIRÉ AU HASARD.
	rng := rand.New(rand.NewSource(7))
	wTested, wok01, wok05 := 0, 0, 0
	if len(models) > 0 {
		for _, g := range groups {
			for _, j := range g.idx {
				if !axisAligned(ins[j]) {
					continue
				}
				e := predictErr(ins[j], models[rng.Intn(len(models))], false)
				wErrs = append(wErrs, e)
				wTested++
				if e <= 0.01 {
					wok01++
				}
				if e <= 0.05 {
					wok05++
				}
			}
		}
	}
	report("  TÉMOIN (mesh tiré au hasard)           ", wErrs, wTested, wok01, wok05)

	containment(ins, groups)
}

// containment : sur base quelconque, l'AABB stockée doit être INCLUSE dans l'AABB
// recomposée depuis la boîte locale (majorant). Une violation = transformation fausse.
func containment(ins []himap.Instance, groups []group) {
	var viol []float64
	n, inside := 0, 0
	for _, g := range groups {
		src := pickSource(ins, g.idx)
		if src < 0 || !axisAligned(ins[src]) {
			continue
		}
		lb, ok := solveLocalBox(ins[src], false)
		if !ok {
			continue
		}
		for _, j := range g.idx {
			if j == src || axisAligned(ins[j]) {
				continue
			}
			v := overflow(ins[j], lb)
			viol = append(viol, v)
			n++
			if v <= 0.01 {
				inside++
			}
		}
	}
	if n == 0 {
		fmt.Println("  INCLUSION : aucune instance pivotée testable")
		return
	}
	sort.Float64s(viol)
	fmt.Printf("  INCLUSION (instances pivotées, n=%d) : dépassement médian=%.5f m p95=%.5f ; incluses à 1cm près %.1f%%\n",
		n, viol[len(viol)/2], viol[len(viol)*95/100], 100*float64(inside)/float64(n))
}

func countLE(sorted []float64, t float64) int {
	return sort.SearchFloat64s(sorted, t+1e-12)
}

// axisAligned : la base est une permutation signée (chaque vecteur porté par un axe).
func axisAligned(in himap.Instance) bool {
	for _, v := range [3][3]float64{in.Forward, in.Left, in.Up} {
		nz := 0
		for a := 0; a < 3; a++ {
			if math.Abs(v[a]) > 1e-4 {
				nz++
			}
		}
		if nz != 1 {
			return false
		}
	}
	return true
}

func sameOrientation(a, b himap.Instance) bool {
	for _, p := range [][2][3]float64{{a.Forward, b.Forward}, {a.Left, b.Left}, {a.Up, b.Up}} {
		for ax := 0; ax < 3; ax++ {
			if math.Abs(p[0][ax]-p[1][ax]) > 1e-4 {
				return false
			}
		}
	}
	return true
}

// overflow : de combien l'AABB stockée SORT de l'AABB recomposée (0 = incluse).
func overflow(in himap.Instance, lb localBox) float64 {
	a := absMat(in)
	b := [3][3]float64{in.Forward, in.Left, in.Up}
	worst := 0.0
	for ax := 0; ax < 3; ax++ {
		center := in.Position[ax] + lb.c[0]*b[0][ax] + lb.c[1]*b[1][ax] + lb.c[2]*b[2][ax]
		half := a[ax][0]*lb.h[0] + a[ax][1]*lb.h[1] + a[ax][2]*lb.h[2]
		for _, e := range []float64{
			(center - half) - in.AABBMin[ax],
			in.AABBMax[ax] - (center + half),
		} {
			if e > worst {
				worst = e
			}
		}
	}
	return worst
}

func report(tag string, errs []float64, tested, ok01, ok05 int) {
	if tested == 0 {
		fmt.Println(tag, "aucun test")
		return
	}
	sort.Float64s(errs)
	fmt.Printf("%s n=%-6d écart max sur les 6 faces : médian=%.5f m p90=%.5f p95=%.5f ; <=1cm %.1f%% ; <=5cm %.1f%%\n",
		tag, tested, errs[len(errs)/2], errs[len(errs)*90/100], errs[len(errs)*95/100],
		100*float64(ok01)/float64(tested), 100*float64(ok05)/float64(tested))
}
