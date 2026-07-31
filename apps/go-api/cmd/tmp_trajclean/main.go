// tmp_trajclean — audite la SATURATION des trajectoires décodées (points collés aux
// bornes de déquantification = faux positifs) et produit un CSV filtré.
//
// Bornes de déquantification connues (cf .ai) : X_min=-41.10318, Z_min=-84.37078.
// Un point collé à une borne n'est pas du mouvement : c'est un underflow du décodeur.
//
// Usage : go run ./cmd/tmp_trajclean <in.csv> <out_clean.csv>
package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

type pt struct {
	slot, chunk, pkt int
	ts               int64
	x, y, z          float64
}

// Bornes de déquantification (bucket 0 / bucket max). Tolérance = 3 millièmes.
const (
	xMinBound = -41.10318
	zMinBound = -84.37078
	tol       = 0.005
)

func quantile(v []float64, q float64) float64 {
	if len(v) == 0 {
		return math.NaN()
	}
	i := int(q * float64(len(v)-1))
	return v[i]
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: tmp_trajclean <in.csv> <out.csv>")
		return
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	sc.Scan() // header
	var pts []pt
	for sc.Scan() {
		c := strings.Split(sc.Text(), ",")
		if len(c) < 7 {
			continue
		}
		var p pt
		p.slot, _ = strconv.Atoi(c[0])
		p.chunk, _ = strconv.Atoi(c[1])
		p.pkt, _ = strconv.Atoi(c[2])
		p.ts, _ = strconv.ParseInt(c[3], 10, 64)
		p.x, _ = strconv.ParseFloat(c[4], 64)
		p.y, _ = strconv.ParseFloat(c[5], 64)
		p.z, _ = strconv.ParseFloat(c[6], 64)
		pts = append(pts, p)
	}
	fmt.Printf("%d points lus\n\n", len(pts))

	// --- enveloppe brute + maxima observés (pour détecter la saturation haute) ---
	var xs, ys, zs []float64
	for _, p := range pts {
		xs = append(xs, p.x)
		ys = append(ys, p.y)
		zs = append(zs, p.z)
	}
	sort.Float64s(xs)
	sort.Float64s(ys)
	sort.Float64s(zs)
	fmt.Printf("enveloppe BRUTE : X[%.4f,%.4f] Y[%.4f,%.4f] Z[%.4f,%.4f]\n",
		xs[0], xs[len(xs)-1], ys[0], ys[len(ys)-1], zs[0], zs[len(zs)-1])
	xMax, zMax := xs[len(xs)-1], zs[len(zs)-1]

	// combien de points EXACTEMENT sur chaque extremum (signature d'une borne dure) ?
	cntEq := func(v []float64, target float64) int {
		n := 0
		for _, x := range v {
			if math.Abs(x-target) <= tol {
				n++
			}
		}
		return n
	}
	fmt.Printf("points à <=%.3f d'un extremum : X_min=%d X_max=%d | Y_min=%d Y_max=%d | Z_min=%d Z_max=%d\n",
		tol, cntEq(xs, xs[0]), cntEq(xs, xMax), cntEq(ys, ys[0]), cntEq(ys, ys[len(ys)-1]),
		cntEq(zs, zs[0]), cntEq(zs, zMax))
	fmt.Printf("bornes de déquant. connues : X_min=%.5f (écart %.5f) Z_min=%.5f (écart %.5f)\n\n",
		xMinBound, xs[0]-xMinBound, zMinBound, zs[0]-zMinBound)

	// --- classification ---
	satX := func(p pt) bool { return p.x <= xMinBound+tol }
	satZ := func(p pt) bool { return p.z <= zMinBound+tol }
	satXhi := func(p pt) bool { return math.Abs(p.x-xMax) <= tol }
	satZhi := func(p pt) bool { return math.Abs(p.z-zMax) <= tol }

	type stat struct{ tot, sx, sz, sxh, szh, any int }
	byChunk := map[int]*stat{}
	bySlot := map[int]*stat{}
	get := func(m map[int]*stat, k int) *stat {
		if m[k] == nil {
			m[k] = &stat{}
		}
		return m[k]
	}
	var clean []pt
	for _, p := range pts {
		a, b, c, d := satX(p), satZ(p), satXhi(p), satZhi(p)
		bad := a || b || c || d
		for _, s := range []*stat{get(byChunk, p.chunk), get(bySlot, p.slot)} {
			s.tot++
			if a {
				s.sx++
			}
			if b {
				s.sz++
			}
			if c {
				s.sxh++
			}
			if d {
				s.szh++
			}
			if bad {
				s.any++
			}
		}
		if !bad {
			clean = append(clean, p)
		}
	}

	fmt.Println("SATURATION PAR CHUNK (chunks 01-08 = validés par rosetta_groundtruth)")
	fmt.Println("chunk  total   X_min   Z_min   X_max   Z_max   saturés    %")
	var ks []int
	for k := range byChunk {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	v18, s18, v926, s926 := 0, 0, 0, 0
	for _, k := range ks {
		s := byChunk[k]
		fmt.Printf("%5d %7d %7d %7d %7d %7d %8d %6.2f%%\n",
			k, s.tot, s.sx, s.sz, s.sxh, s.szh, s.any, 100*float64(s.any)/float64(s.tot))
		if k <= 8 {
			v18 += s.tot
			s18 += s.any
		} else {
			v926 += s.tot
			s926 += s.any
		}
	}
	fmt.Printf("\nchunks 01-08 (VALIDÉS) : %d pts, %d saturés (%.2f%%)\n", v18, s18, 100*float64(s18)/float64(max(v18, 1)))
	fmt.Printf("chunks 09-26 (NON validés) : %d pts, %d saturés (%.2f%%)\n\n", v926, s926, 100*float64(s926)/float64(max(v926, 1)))

	fmt.Println("SATURATION PAR SLOT")
	var kss []int
	for k := range bySlot {
		kss = append(kss, k)
	}
	sort.Ints(kss)
	for _, k := range kss {
		s := bySlot[k]
		fmt.Printf("slot %d : %6d pts, %6d saturés (%.2f%%)\n", k, s.tot, s.any, 100*float64(s.any)/float64(s.tot))
	}

	// --- enveloppe propre ---
	xs, ys, zs = nil, nil, nil
	for _, p := range clean {
		xs = append(xs, p.x)
		ys = append(ys, p.y)
		zs = append(zs, p.z)
	}
	sort.Float64s(xs)
	sort.Float64s(ys)
	sort.Float64s(zs)
	fmt.Printf("\n%d points PROPRES (%.2f%% conservés)\n", len(clean), 100*float64(len(clean))/float64(len(pts)))
	fmt.Printf("enveloppe PROPRE : X[%.3f,%.3f] Y[%.3f,%.3f] Z[%.3f,%.3f]\n",
		xs[0], xs[len(xs)-1], ys[0], ys[len(ys)-1], zs[0], zs[len(zs)-1])
	fmt.Printf("  (1-99 pct)      X[%.3f,%.3f] Y[%.3f,%.3f] Z[%.3f,%.3f]\n",
		quantile(xs, 0.01), quantile(xs, 0.99), quantile(ys, 0.01), quantile(ys, 0.99),
		quantile(zs, 0.01), quantile(zs, 0.99))

	out, err := os.Create(os.Args[2])
	if err != nil {
		fmt.Println(err)
		return
	}
	defer out.Close()
	w := bufio.NewWriter(out)
	defer w.Flush()
	fmt.Fprintln(w, "slot,chunk,packetIndex,ts,x,y,z")
	for _, p := range clean {
		fmt.Fprintf(w, "%d,%d,%d,%d,%.5f,%.5f,%.5f\n", p.slot, p.chunk, p.pkt, p.ts, p.x, p.y, p.z)
	}
	fmt.Printf("→ %s\n", os.Args[2])
}
