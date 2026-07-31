// tmp_aimdiag — diagnostic THROWAWAY : où est réellement la direction dans le record
// biped ? Histogramme des index de composants du masque, forme du champ i1, et test de
// MAGNITUDE (indépendant du repère) : |pos[t+1]-pos[t]|/dt vs magnitude décodée.
package main

import (
	"fmt"
	"math"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const defaultFilm = `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\cache\film_chunks\000d5950`

func main() {
	dir := defaultFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	idxHist := map[int]int{}
	secondHist := map[int]int{}
	filmdec.SetRecordMaskHook(func(idx []int, _ []byte, _ int) {
		for _, k := range idx {
			idxHist[k]++
		}
		if len(idx) > 1 {
			secondHist[idx[1]]++
		}
	})
	opt := filmdec.DefaultScanFilmOptions()
	opt.CaptureDirs = true
	pos, err := filmdec.ScanFilmBipedPositions(dir, opt)
	if err != nil {
		fmt.Println("erreur:", err)
		os.Exit(1)
	}
	fmt.Println("index de composants présents dans les masques :")
	printHist(idxHist)
	fmt.Println("PREMIER composant après i0 :")
	printHist(secondHist)

	var faces [6]int
	var zAbs, mags []float64
	for _, p := range pos {
		if !p.HasVel {
			continue
		}
		v, ok := filmdec.DecodeAimVectorChecked(p.VelRaw, 19)
		if !ok {
			continue
		}
		faces[faceOf(p.VelRaw)]++
		zAbs = append(zAbs, math.Abs(float64(v[2])))
		vv, _ := p.VelocityVector()
		mags = append(mags, math.Sqrt(float64(vv[0]*vv[0]+vv[1]*vv[1]+vv[2]*vv[2])))
	}
	tot := len(zAbs)
	fmt.Printf("\ni1: n=%d faces(+X,+Y,+Z,-X,-Y,-Z) =", tot)
	for _, f := range faces {
		fmt.Printf(" %.1f%%", 100*float64(f)/float64(maxi(tot, 1)))
	}
	fmt.Println()
	sort.Float64s(zAbs)
	sort.Float64s(mags)
	if tot > 0 {
		fmt.Printf("i1 |z| p10=%.3f médian=%.3f p90=%.3f | magnitude p10=%.2f médiane=%.2f p90=%.2f u/s\n",
			zAbs[tot/10], zAbs[tot/2], zAbs[tot*9/10], mags[tot/10], mags[tot/2], mags[tot*9/10])
	}

	bySlot := map[uint32][]filmdec.BipedPosition{}
	for _, p := range pos {
		bySlot[p.Slot] = append(bySlot[p.Slot], p)
	}
	var ratios []float64
	for _, list := range bySlot {
		sort.Slice(list, func(i, j int) bool { return list[i].TimestampUS < list[j].TimestampUS })
		for i := 0; i+1 < len(list); i++ {
			a, b := list[i], list[i+1]
			dt := float64(b.TimestampUS-a.TimestampUS) / 1e6
			if dt <= 0 || dt > 0.2 || !a.HasVel {
				continue
			}
			d := math.Sqrt(float64((b.X-a.X)*(b.X-a.X) + (b.Y-a.Y)*(b.Y-a.Y) + (b.Z-a.Z)*(b.Z-a.Z)))
			sp := d / dt
			if sp < 1 {
				continue
			}
			vv, _ := a.VelocityVector()
			m := math.Sqrt(float64(vv[0]*vv[0] + vv[1]*vv[1] + vv[2]*vv[2]))
			if m > 0 {
				ratios = append(ratios, sp/m)
			}
		}
	}
	sort.Float64s(ratios)
	if len(ratios) > 0 {
		fmt.Printf("\nvitesse mesurée / magnitude i1 : n=%d p10=%.3f médiane=%.3f p90=%.3f (1.0 attendu)\n",
			len(ratios), ratios[len(ratios)/10], ratios[len(ratios)/2], ratios[len(ratios)*9/10])
	}
}

func printHist(h map[int]int) {
	keys := make([]int, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		fmt.Printf("  i%-3d %d\n", k, h[k])
	}
}

func faceOf(code uint32) int {
	fs := int(uint64(1)<<19) / 6
	f := int(code) / fs
	if f < 0 || f > 5 {
		return 0
	}
	return f
}

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}
