// tmp_qsplit — teste si les champs Y/Z de la position i0 sont mal DÉCOUPÉS sur une map
// autre que celle de calibration, sans re-décoder le film.
//
// Le décodeur lit i0 = [5 gate][13 X][13 Y][14 Z]. Les 27 bits Y+Z sont contigus, donc
// entièrement reconstructibles depuis les quanta exportés (bits27 = qy<<14 | qz). On
// re-découpe ces 27 bits selon (skip, largeurY, largeurZ) et on retient le découpage qui
// minimise le pas médian entre échantillons consécutifs d'un même slot — signature d'une
// trajectoire continue, par opposition au bruit.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_qsplit <positions.csv>
package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
)

type sample struct {
	slot uint32
	ts   uint64
	qx   uint32
	b27  uint32 // les 27 bits contigus Y+Z tels que lus aujourd'hui
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: tmp_qsplit <positions.csv>")
		os.Exit(2)
	}
	rows, err := load(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("%d échantillons\n\n", len(rows))

	// Référence : l'axe X, non contesté (pas médian attendu ~1-3 quanta).
	fmt.Printf("axe X (13 bits, découpage actuel) : |dq| median %.0f p90 %.0f\n\n",
		medStep(rows, func(s sample) uint32 { return s.qx }), pctStep(rows, func(s sample) uint32 { return s.qx }, 90))

	type cand struct {
		skip, wy, wz int
		my, mz       float64
	}
	var cands []cand
	for skip := 0; skip <= 6; skip++ {
		for wy := 11; wy <= 15; wy++ {
			for wz := 11; wz <= 16; wz++ {
				if skip+wy+wz > 27 {
					continue
				}
				shY := uint(27 - skip - wy)
				shZ := uint(27 - skip - wy - wz)
				my := medStep(rows, func(s sample) uint32 { return s.b27 >> shY & (1<<uint(wy) - 1) })
				mz := medStep(rows, func(s sample) uint32 { return s.b27 >> shZ & (1<<uint(wz) - 1) })
				cands = append(cands, cand{skip, wy, wz, my, mz})
			}
		}
	}
	sort.Slice(cands, func(i, j int) bool {
		return cands[i].my+cands[i].mz < cands[j].my+cands[j].mz
	})
	fmt.Println("meilleurs découpages des 27 bits Y+Z (pas médian en quanta, plus petit = trajectoire) :")
	fmt.Println("skip  wY  wZ   |dY| med   |dZ| med")
	for i, c := range cands {
		if i >= 12 {
			break
		}
		fmt.Printf("%4d %3d %3d   %8.0f   %8.0f\n", c.skip, c.wy, c.wz, c.my, c.mz)
	}
	// Rappel du découpage actuel pour comparaison directe.
	for _, c := range cands {
		if c.skip == 0 && c.wy == 13 && c.wz == 14 {
			fmt.Printf("\nDÉCOUPAGE ACTUEL (skip=0, wY=13, wZ=14) : |dY| med %.0f  |dZ| med %.0f\n", c.my, c.mz)
		}
	}
}

func load(path string) ([]sample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	r := csv.NewReader(f)
	recs, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	out := make([]sample, 0, len(recs))
	for i, rec := range recs {
		if i == 0 || len(rec) < 7 {
			continue
		}
		n := func(k int) uint64 { v, _ := strconv.ParseUint(rec[k], 10, 64); return v }
		qy, qz := uint32(n(5)), uint32(n(6))
		out = append(out, sample{slot: uint32(n(0)), ts: n(3), qx: uint32(n(4)), b27: qy<<14 | qz})
	}
	return out, nil
}

// steps calcule les |delta| du champ extrait par get, entre échantillons consécutifs d'un
// même slot séparés de moins de 0,5 s.
func steps(rows []sample, get func(sample) uint32) []float64 {
	bySlot := map[uint32][]sample{}
	for _, r := range rows {
		bySlot[r.slot] = append(bySlot[r.slot], r)
	}
	var out []float64
	for _, s := range bySlot {
		for i := 1; i < len(s); i++ {
			dt := s[i].ts - s[i-1].ts
			if dt == 0 || dt > 500000 {
				continue
			}
			a, b := int64(get(s[i])), int64(get(s[i-1]))
			d := a - b
			if d < 0 {
				d = -d
			}
			out = append(out, float64(d))
		}
	}
	sort.Float64s(out)
	return out
}

func medStep(rows []sample, get func(sample) uint32) float64 {
	s := steps(rows, get)
	if len(s) == 0 {
		return -1
	}
	return s[len(s)/2]
}

func pctStep(rows []sample, get func(sample) uint32, p int) float64 {
	s := steps(rows, get)
	if len(s) == 0 {
		return -1
	}
	i := len(s) * p / 100
	if i >= len(s) {
		i = len(s) - 1
	}
	return s[i]
}
