// tmp_aimsweep2 — recherche THROWAWAY du champ de VISÉE (i21 unit-desired-aiming-vector)
// dans le record biped delta, à partir du composant i1 dont la longueur est maintenant
// connue exactement.
//
// Juge : la CONCENTRATION CIRCULAIRE R = |moyenne(exp(i·Δ))| de l'écart Δ entre l'angle
// candidat et le cap de déplacement du même slot. R est invariant par décalage constant
// de convention d'angle (contrairement à une médiane), donc il détecte un champ d'angle
// même si l'origine n'est pas la nôtre. R ~ 0 = hasard ; R > 0,3 = signal net.
//
// Interprétations testées à chaque offset : yaw 12 bits (direct/inversé), yaw 11 bits,
// paire (12,11) lue comme deux coordonnées, et direction cubemap 19 bits.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_aimsweep2 [filmDir]
package main

import (
	"fmt"
	"math"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const defaultFilm = `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\cache\film_chunks\000d5950`

const tailBits = 256

type rec struct {
	slot   uint32
	ts     uint64
	x, y   float32
	idx    []int
	tail   []byte
	i1Bits int
}

func main() {
	dir := defaultFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	var recs []rec
	filmdec.SetRecordMaskHook(func(idx []int, pay []byte, at int) {
		buf := make([]byte, tailBits/8)
		start := at + 2 // queue d'i0
		for i := 0; i < tailBits/8; i++ {
			if start+8*i+8 <= len(pay)*8 {
				buf[i] = byte(filmdec.ReadBitsAtForDiag(pay, start+8*i, 8))
			}
		}
		cp := make([]int, len(idx))
		copy(cp, idx)
		recs = append(recs, rec{idx: cp, tail: buf})
	})
	opt := filmdec.DefaultScanFilmOptions()
	opt.CaptureDirs = true
	opt.MaxSpeedMPS, opt.IsolationGapMS = 0, 0
	pos, err := filmdec.ScanFilmBipedPositions(dir, opt)
	if err != nil || len(pos) != len(recs) {
		fmt.Println("erreur:", err, len(pos), len(recs))
		os.Exit(1)
	}
	for i := range recs {
		recs[i].slot, recs[i].ts = pos[i].Slot, pos[i].TimestampUS
		recs[i].x, recs[i].y = pos[i].X, pos[i].Y
		recs[i].i1Bits = i1Len(recs[i])
	}

	// paires : record + cap de déplacement, restreintes aux masques [0,1,21,...]
	type pair struct {
		r  *rec
		mv float64
	}
	bySlot := map[uint32][]*rec{}
	for i := range recs {
		bySlot[recs[i].slot] = append(bySlot[recs[i].slot], &recs[i])
	}
	var pairs []pair
	for _, l := range bySlot {
		sort.Slice(l, func(i, j int) bool { return l[i].ts < l[j].ts })
		for i := 0; i+1 < len(l); i++ {
			a, b := l[i], l[i+1]
			dx, dy := float64(b.x-a.x), float64(b.y-a.y)
			if math.Hypot(dx, dy) < 0.05 {
				continue
			}
			if len(a.idx) < 3 || a.idx[1] != 1 || a.idx[2] != 21 || a.i1Bits < 0 {
				continue
			}
			pairs = append(pairs, pair{a, math.Atan2(dy, dx)})
		}
	}
	fmt.Printf("paires [0,1,21,...] exploitables = %d\n\n", len(pairs))
	if len(pairs) < 200 {
		fmt.Println("trop peu de paires : élargir le filtre")
		return
	}

	type res struct {
		off  int
		mode string
		r    float64
		n    int
	}
	var all []res
	modes := []string{"yaw12", "yaw12inv", "yaw11", "yaw11inv", "pair1211", "cube19"}
	for off := -4; off <= 60; off++ {
		for _, m := range modes {
			var sx, sy float64
			n := 0
			for _, p := range pairs {
				at := p.r.i1Bits + off
				if at < 0 || at+24 > tailBits {
					continue
				}
				ang, ok := angleAt(p.r.tail, at, m)
				if !ok {
					continue
				}
				d := ang - p.mv
				sx += math.Cos(d)
				sy += math.Sin(d)
				n++
			}
			if n < 200 {
				continue
			}
			all = append(all, res{off, m, math.Hypot(sx, sy) / float64(n), n})
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].r > all[j].r })
	fmt.Println("meilleures concentrations circulaires R (1 = parfait, 0 = hasard) :")
	for i := 0; i < 20 && i < len(all); i++ {
		a := all[i]
		fmt.Printf("  offset %3d %-9s R=%.4f n=%d\n", a.off, a.mode, a.r, a.n)
	}
	if len(all) > 0 {
		var med float64
		rs := make([]float64, len(all))
		for i, a := range all {
			rs[i] = a.r
		}
		sort.Float64s(rs)
		med = rs[len(rs)/2]
		fmt.Printf("R médian sur tous les candidats (niveau du bruit) = %.4f\n", med)
	}
}

// i1Len : longueur en bits du composant i1 tel qu'il commence au bit 0 du tail.
func i1Len(r rec) int {
	if len(r.idx) < 2 || r.idx[1] != 1 {
		return -1
	}
	if filmdec.ReadBitsAtForDiag(r.tail, 0, 1) == 1 {
		return 1 + 96
	}
	if filmdec.ReadBitsAtForDiag(r.tail, 1, 1) == 1 {
		return 2
	}
	return 2 + 19 + 10
}

func angleAt(tail []byte, at int, mode string) (float64, bool) {
	switch mode {
	case "yaw12":
		q := filmdec.ReadBitsAtForDiag(tail, at, 12)
		return 2 * math.Pi * (float64(q) + 0.5) / 4096, true
	case "yaw12inv":
		q := filmdec.ReadBitsAtForDiag(tail, at, 12)
		return -2 * math.Pi * (float64(q) + 0.5) / 4096, true
	case "yaw11":
		q := filmdec.ReadBitsAtForDiag(tail, at, 11)
		return 2 * math.Pi * (float64(q) + 0.5) / 2048, true
	case "yaw11inv":
		q := filmdec.ReadBitsAtForDiag(tail, at, 11)
		return -2 * math.Pi * (float64(q) + 0.5) / 2048, true
	case "pair1211":
		a := 2*(float64(filmdec.ReadBitsAtForDiag(tail, at, 12))+0.5)/4096 - 1
		b := 2*(float64(filmdec.ReadBitsAtForDiag(tail, at+12, 11))+0.5)/2048 - 1
		return math.Atan2(b, a), true
	default: // cube19
		v, ok := filmdec.DecodeAimVectorChecked(filmdec.ReadBitsAtForDiag(tail, at, 19), 19)
		if !ok {
			return 0, false
		}
		return math.Atan2(float64(v[1]), float64(v[0])), true
	}
}
