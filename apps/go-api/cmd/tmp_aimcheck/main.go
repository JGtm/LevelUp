// tmp_aimcheck — harness de VALIDATION des directions décodées d'un film (THROWAWAY,
// comme tmp_kfworldpos ; aucun chemin de production ne l'appelle).
//
// Il mesure :
//   - normes et couverture de sphère des directions cubemap (i1 vélocité, i2 forward) ;
//   - l'écart angulaire 2D entre chaque direction et la direction de déplacement du MÊME
//     slot au même instant, contre deux CONTRÔLES (direction d'un autre échantillon tiré
//     au hasard ; direction du même slot 500 échantillons plus loin) ;
//   - le même écart avec la version communautaire du décodeur (2e coordonnée nulle) ;
//   - le cap de VISÉE (i21) : offset de convention et concentration circulaire ;
//   - le rapport vitesse mesurée / magnitude décodée (test indépendant du repère).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_aimcheck [filmDir]
package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const defaultFilm = `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\cache\film_chunks\000d5950`

// minStep : sous ce déplacement (unités monde) le cap de déplacement est du bruit de
// quantification (le quantum de grille vaut ~0,0138 u).
const minStep = 0.05

func main() {
	dir := defaultFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	opt := filmdec.DefaultScanFilmOptions()
	opt.CaptureDirs = true
	pos, err := filmdec.ScanFilmBipedPositions(dir, opt)
	if err != nil {
		fmt.Println("erreur:", err)
		os.Exit(1)
	}
	nVel, nAim, nYaw := 0, 0, 0
	perSlot := map[uint32][3]int{}
	for _, p := range pos {
		c := perSlot[p.Slot]
		if p.HasVel {
			nVel++
			c[0]++
		}
		if p.HasAim {
			nAim++
			c[1]++
		}
		if p.HasYaw {
			nYaw++
			c[2]++
		}
		perSlot[p.Slot] = c
	}
	fmt.Printf("records=%d | i1 vélocité=%d | i2 forward=%d | i21 visée=%d | slots=%d\n",
		len(pos), nVel, nAim, nYaw, len(perSlot))
	printPerSlot(perSlot)
	reportSphere(pos)
	reportHeading(pos)
	reportSpeed(pos)
}

func printPerSlot(m map[uint32][3]int) {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	fmt.Println("échantillons par slot (vel / fwd / visée), 12 premiers :")
	for i, k := range keys {
		if i >= 12 {
			fmt.Printf("  ... et %d autres slots\n", len(keys)-12)
			break
		}
		c := m[uint32(k)]
		fmt.Printf("  slot %-5d %6d / %4d / %6d\n", k, c[0], c[1], c[2])
	}
}

// reportSphere : normes unitaires + couverture des 6 faces + |z|.
func reportSphere(pos []filmdec.BipedPosition) {
	var faces [6]int
	minN, maxN, invalid := math.MaxFloat64, 0.0, 0
	var zAbs []float64
	fs := int(uint64(1)<<19) / 6
	for _, p := range pos {
		if !p.HasVel {
			continue
		}
		v, ok := filmdec.DecodeAimVectorChecked(p.VelRaw, 19)
		if !ok {
			invalid++
			continue
		}
		n := math.Sqrt(float64(v[0]*v[0] + v[1]*v[1] + v[2]*v[2]))
		minN, maxN = math.Min(minN, n), math.Max(maxN, n)
		if f := int(p.VelRaw) / fs; f >= 0 && f < 6 {
			faces[f]++
		}
		zAbs = append(zAbs, math.Abs(float64(v[2])))
	}
	fmt.Printf("\n--- vecteurs i1 : normes min=%.6f max=%.6f | face>=6 (désalignement) = %d\n",
		minN, maxN, invalid)
	tot := len(zAbs)
	fmt.Print("faces (+X,+Y,+Z,-X,-Y,-Z) :")
	for _, f := range faces {
		fmt.Printf(" %.1f%%", 100*float64(f)/float64(maxi(tot, 1)))
	}
	fmt.Println()
	sort.Float64s(zAbs)
	if tot > 0 {
		fmt.Printf("|z| p10=%.3f médian=%.3f p90=%.3f (déplacement d'un Spartan = quasi horizontal)\n",
			zAbs[tot/10], zAbs[tot/2], zAbs[tot*9/10])
	}
	// Couverture de la sphère : histogramme des indices de cellule iu/iv, pour détecter
	// un agrégat artificiel sur une couture ou une valeur interdite (stride faux).
	_, n, _ := 0, 294, 0 // gridSize(19) = 294 ; iu,iv doivent rester dans [0, N-2]
	maxU, maxV, bad := 0, 0, 0
	for _, p := range pos {
		if !p.HasVel {
			continue
		}
		rem := int(p.VelRaw) % fs
		iu, iv := rem/n, rem%n
		if iu > maxU {
			maxU = iu
		}
		if iv > maxV {
			maxV = iv
		}
		if iu > n-2 || iv > n-2 {
			bad++
		}
	}
	fmt.Printf("indices de cellule : max iu=%d max iv=%d (interdit : > %d) | violations=%d\n",
		maxU, maxV, n-2, bad)

	// Norme AVANT normalisation (sqrt(1+a²+b²), dans [1, sqrt(3)]) et écart 3D entre le
	// décodage complet et la version communautaire (2e coordonnée nulle).
	var raws, gaps []float64
	for _, p := range pos {
		if !p.HasVel {
			continue
		}
		rem := int(p.VelRaw) % fs
		iu, iv := rem/n, rem%n
		step := 2.0 / float64(n-1)
		a := float64(iu)*step - 1 + step/2
		b := float64(iv)*step - 1 + step/2
		raws = append(raws, math.Sqrt(1+a*a+b*b))
		full, _ := filmdec.DecodeAimVectorChecked(p.VelRaw, 19)
		flat, _ := filmdec.DecodeAimVectorFlat(p.VelRaw, 19)
		gaps = append(gaps, angDiff3(full, flat))
	}
	sort.Float64s(raws)
	sort.Float64s(gaps)
	if len(raws) > 0 {
		fmt.Printf("norme AVANT normalisation : min=%.4f médiane=%.4f max=%.4f (théorie [1 ; 1,7321])\n",
			raws[0], raws[len(raws)/2], raws[len(raws)-1])
		fmt.Printf("écart 3D complet vs 2e coord nulle : médiane=%.2f° p90=%.2f° max=%.2f°\n",
			gaps[len(gaps)/2], gaps[len(gaps)*9/10], gaps[len(gaps)-1])
	}
}

func angDiff3(a, b [3]float32) float64 {
	c := float64(a[0]*b[0] + a[1]*b[1] + a[2]*b[2])
	c = math.Max(-1, math.Min(1, c))
	return math.Acos(c) * 180 / math.Pi
}

type stat struct{ deg []float64 }

func (s *stat) add(d float64) { s.deg = append(s.deg, d) }
func (s *stat) report(label string) {
	if len(s.deg) == 0 {
		fmt.Printf("%-30s aucun échantillon\n", label)
		return
	}
	sort.Float64s(s.deg)
	var mean float64
	u30, u60 := 0, 0
	for _, d := range s.deg {
		mean += d
		if d < 30 {
			u30++
		}
		if d < 60 {
			u60++
		}
	}
	mean /= float64(len(s.deg))
	fmt.Printf("%-30s n=%6d médiane=%6.1f° moyenne=%6.1f° <30°=%5.1f%% <60°=%5.1f%%\n",
		label, len(s.deg), s.deg[len(s.deg)/2], mean, 100*float64(u30)/float64(len(s.deg)),
		100*float64(u60)/float64(len(s.deg)))
}

// reportHeading : le juge principal — cap décodé vs direction de déplacement, + contrôles.
func reportHeading(pos []filmdec.BipedPosition) {
	bySlot := map[uint32][]filmdec.BipedPosition{}
	for _, p := range pos {
		bySlot[p.Slot] = append(bySlot[p.Slot], p)
	}
	rnd := rand.New(rand.NewSource(42))
	var velFull, velFlat, ctrlRnd, ctrlShift, aimYaw, aimCtrl stat
	var pool []filmdec.BipedPosition
	for _, p := range pos {
		if p.HasVel {
			pool = append(pool, p)
		}
	}
	var yawPool []filmdec.BipedPosition
	for _, p := range pos {
		if p.HasYaw {
			yawPool = append(yawPool, p)
		}
	}
	var sx, sy float64 // moyenne circulaire de (yaw - cap mouvement) : offset de convention
	nYaw := 0
	slots := make([]int, 0, len(bySlot))
	for k := range bySlot {
		slots = append(slots, int(k))
	}
	sort.Ints(slots)
	for _, sl := range slots {
		l := bySlot[uint32(sl)]
		sort.Slice(l, func(i, j int) bool { return l[i].TimestampUS < l[j].TimestampUS })
		for i := 0; i+1 < len(l); i++ {
			a, b := l[i], l[i+1]
			dx, dy := float64(b.X-a.X), float64(b.Y-a.Y)
			if math.Hypot(dx, dy) < minStep {
				continue
			}
			mv := math.Atan2(dy, dx)
			if a.HasVel {
				if v, ok := filmdec.DecodeAimVectorChecked(a.VelRaw, 19); ok {
					velFull.add(angDiff(math.Atan2(float64(v[1]), float64(v[0])), mv))
				}
				if v, ok := filmdec.DecodeAimVectorFlat(a.VelRaw, 19); ok {
					velFlat.add(angDiff(math.Atan2(float64(v[1]), float64(v[0])), mv))
				}
				o := pool[rnd.Intn(len(pool))]
				if v, ok := filmdec.DecodeAimVectorChecked(o.VelRaw, 19); ok {
					ctrlRnd.add(angDiff(math.Atan2(float64(v[1]), float64(v[0])), mv))
				}
				j := (i + 500) % len(l)
				if l[j].HasVel {
					if v, ok := filmdec.DecodeAimVectorChecked(l[j].VelRaw, 19); ok {
						ctrlShift.add(angDiff(math.Atan2(float64(v[1]), float64(v[0])), mv))
					}
				}
			}
			if a.HasYaw {
				h, _ := a.AimHeadingDeg()
				rad := float64(h) * math.Pi / 180
				aimYaw.add(angDiff(rad, mv))
				d := rad - mv
				sx, sy, nYaw = sx+math.Cos(d), sy+math.Sin(d), nYaw+1
				o := yawPool[rnd.Intn(len(yawPool))]
				ho, _ := o.AimHeadingDeg()
				aimCtrl.add(angDiff(float64(ho)*math.Pi/180, mv))
			}
		}
	}
	fmt.Println("\n--- JUGE : direction décodée vs direction de déplacement (pas >= 0,05 u) ---")
	velFull.report("i1 vélocité (2e coord)")
	velFull2 := velFlat
	velFull2.report("i1 vélocité (2e coord NULLE)")
	aimYaw.report("i21 cap de VISÉE")
	fmt.Println("--- CONTRÔLES (doivent être plats : médiane ~90°, <30° ~33%) ---")
	ctrlRnd.report("vél. d'un autre échantillon")
	ctrlShift.report("vél. du même slot, +500")
	aimCtrl.report("visée d'un autre échantillon")
	if nYaw > 0 {
		off := math.Atan2(sy, sx) * 180 / math.Pi
		fmt.Printf("visée : offset de convention mesuré = %+.2f° | concentration R=%.4f (n=%d)\n",
			off, math.Hypot(sx, sy)/float64(nYaw), nYaw)
	}
}

// reportSpeed : test INDÉPENDANT DU REPÈRE — la magnitude log/exp décodée doit égaler la
// vitesse mesurée par différence finie des positions.
func reportSpeed(pos []filmdec.BipedPosition) {
	bySlot := map[uint32][]filmdec.BipedPosition{}
	for _, p := range pos {
		bySlot[p.Slot] = append(bySlot[p.Slot], p)
	}
	var ratios []float64
	for _, l := range bySlot {
		sort.Slice(l, func(i, j int) bool { return l[i].TimestampUS < l[j].TimestampUS })
		for i := 0; i+1 < len(l); i++ {
			a, b := l[i], l[i+1]
			dt := float64(b.TimestampUS-a.TimestampUS) / 1e6
			if dt <= 0 || dt > 0.2 || !a.HasVel {
				continue
			}
			d := math.Sqrt(float64((b.X-a.X)*(b.X-a.X) + (b.Y-a.Y)*(b.Y-a.Y) + (b.Z-a.Z)*(b.Z-a.Z)))
			if sp := d / dt; sp >= 1 {
				v, _ := a.VelocityVector()
				m := math.Sqrt(float64(v[0]*v[0] + v[1]*v[1] + v[2]*v[2]))
				if m > 0 {
					ratios = append(ratios, sp/m)
				}
			}
		}
	}
	sort.Float64s(ratios)
	if len(ratios) > 0 {
		fmt.Printf("\nvitesse mesurée / magnitude i1 décodée : n=%d p10=%.3f médiane=%.3f p90=%.3f (1,0 attendu)\n",
			len(ratios), ratios[len(ratios)/10], ratios[len(ratios)/2], ratios[len(ratios)*9/10])
	}
}

func angDiff(a, b float64) float64 {
	d := math.Abs(a - b)
	for d > math.Pi {
		d = 2*math.Pi - d
	}
	return d * 180 / math.Pi
}

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}
