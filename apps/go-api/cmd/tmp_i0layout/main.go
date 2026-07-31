// tmp_i0layout — affiche le decoupage i0 LU dans un film par filmdec.DetectI0Layout,
// puis les statistiques de pas (quanta bruts) du decodage complet.
package main

import (
	"flag"
	"fmt"
	"math"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const base = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`

func main() {
	film := flag.String("film", "000d5950", "id du film")
	stats := flag.Bool("stats", false, "decoder et mesurer les pas")
	flag.Parse()
	dir := filepath.Join(base, *film)

	lay, rep, err := filmdec.DetectI0Layout(dir)
	if err != nil {
		fmt.Printf("%s : ERREUR %v (rep: %d records, %d paires, frontieres %v)\n",
			*film, err, rep.Records, rep.Pairs, rep.Boundaries)
		return
	}
	fmt.Printf("%s : %s | frontieres=%v | %d records, %d paires, bit d'index a 1 sur %d records\n",
		*film, lay, rep.Boundaries, rep.Records, rep.Pairs, rep.IndexBitOnes)
	if !*stats {
		return
	}

	opt := filmdec.DefaultScanFilmOptions()
	// Outil de MESURE du découpage : il travaille sur les quanta bruts, pas sur des
	// coordonnées monde (celles-ci exigent les bornes de la carte, cf. map_bounds.go).
	opt.QuantaOnly = true
	opt.MaxSpeedMPS = 0    // pas de filtre teleport : on veut LES MESURER
	opt.IsolationGapMS = 0 // idem
	opt.DropSaturated = true
	pos, err := filmdec.ScanFilmBipedPositions(dir, opt)
	if err != nil {
		fmt.Println("scan:", err)
		return
	}
	fmt.Printf("positions decodees : %d\n", len(pos))
	report(pos, lay)
}

func report(pos []filmdec.BipedPosition, lay filmdec.I0Layout) {
	bySlot := map[uint32][]filmdec.BipedPosition{}
	for _, p := range pos {
		bySlot[p.Slot] = append(bySlot[p.Slot], p)
	}
	var dq [3][]float64
	var d3 []float64
	var dtms []float64
	tele := 0
	pairs := 0
	for _, ss := range bySlot {
		sort.Slice(ss, func(i, j int) bool { return ss[i].TimestampUS < ss[j].TimestampUS })
		for i := 1; i < len(ss); i++ {
			a, b := ss[i-1], ss[i]
			dt := float64(b.TimestampUS-a.TimestampUS) / 1000
			if dt <= 0 || dt > 100 {
				continue
			}
			pairs++
			dtms = append(dtms, dt)
			var sum float64
			for ax := 0; ax < 3; ax++ {
				d := math.Abs(float64(b.Q[ax]) - float64(a.Q[ax]))
				dq[ax] = append(dq[ax], d)
				sum += d * d
			}
			n := math.Sqrt(sum)
			d3 = append(d3, n)
			// teleport : > 1/8 de la boite sur un axe en un pas
			for ax := 0; ax < 3; ax++ {
				if math.Abs(float64(b.Q[ax])-float64(a.Q[ax])) > float64(uint64(1)<<lay.AxisW[ax])/8 {
					tele++
					break
				}
			}
		}
	}
	// quantum : W = ceilLog2(ceil(60*extent)) => extent/2^W dans ]1/120, 1/60].
	// borne mediane geometrique 1/sqrt(120*60) pour convertir en unites monde.
	const qLo, qHi = 1.0 / 120, 1.0 / 60
	qMid := math.Sqrt(qLo * qHi)
	fmt.Printf("paires consecutives : %d ; dt median = %.1f ms ; teleports (>1/8 boite) = %d (%.4f%%)\n",
		pairs, med(dtms), tele, 100*float64(tele)/float64(pairs))
	for ax, name := range []string{"X", "Y", "Z"} {
		m := med(dq[ax])
		fmt.Printf("  %s (w=%2d) : |dq| median = %6.1f quanta  -> pas median in [%.4f ; %.4f] wu (mid %.4f)\n",
			name, lay.AxisW[ax], m, m*qLo, m*qHi, m*qMid)
	}
	m3 := med(d3)
	fmt.Printf("  norme 3D : |dq| median = %6.1f quanta -> pas median in [%.4f ; %.4f] wu (mid %.4f)\n",
		m3, m3*qLo, m3*qHi, m3*qMid)
}

func med(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	c := append([]float64(nil), v...)
	sort.Float64s(c)
	return c[len(c)/2]
}
