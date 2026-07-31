// tmp_worldcheck — contrôle PHYSIQUE des coordonnées monde produites avec les bornes
// par carte du catalogue : nuage de points contenu dans la boîte du BSP, pas médian en
// unités monde, vitesse médiane, téléportations.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_worldcheck <carte> <filmId> [<carte> <filmId> ...]
package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	repoRoot  = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`
	filmCache = repoRoot + `/data/cache/film_chunks`
	catalog   = repoRoot + `/data/titles/halo_infinite/reference/map_quant_bounds.json`
)

func main() {
	if len(os.Args) < 3 || len(os.Args)%2 != 1 {
		fmt.Println("usage: tmp_worldcheck <carte> <filmId> [...]")
		os.Exit(2)
	}
	cat, err := filmdec.LoadMapQuantCatalog(catalog)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	for i := 1; i+1 < len(os.Args); i += 2 {
		check(cat, os.Args[i], os.Args[i+1])
	}
}

func check(cat *filmdec.MapQuantCatalog, mapName, filmID string) {
	dir := filepath.Join(filmCache, filmID)
	entry, err := cat.Lookup(mapName)
	if err != nil {
		fmt.Printf("%-14s %s : %v\n", mapName, filmID, err)
		return
	}
	lay, _, err := filmdec.DetectI0Layout(dir)
	if err != nil {
		fmt.Printf("%-14s %s : découpage illisible : %v\n", mapName, filmID, err)
		return
	}
	rng := entry.Range()
	opt := filmdec.DefaultScanFilmOptions()
	opt.WorldRange = &rng
	pos, err := filmdec.ScanFilmBipedPositions(dir, opt)
	if err != nil {
		fmt.Printf("%-14s %s : %v\n", mapName, filmID, err)
		return
	}
	// contrôle croisé découpage film <-> largeur déduite des bornes
	agree := lay.AxisW == entry.AxisWidths

	var lo, hi [3]float64
	for ax := 0; ax < 3; ax++ {
		lo[ax], hi[ax] = math.Inf(1), math.Inf(-1)
	}
	bySlot := map[uint32][]filmdec.BipedPosition{}
	for _, p := range pos {
		v := [3]float64{float64(p.X), float64(p.Y), float64(p.Z)}
		for ax := 0; ax < 3; ax++ {
			lo[ax] = math.Min(lo[ax], v[ax])
			hi[ax] = math.Max(hi[ax], v[ax])
		}
		bySlot[p.Slot] = append(bySlot[p.Slot], p)
	}
	var steps, speeds []float64
	for _, ss := range bySlot {
		sort.Slice(ss, func(i, j int) bool { return ss[i].TimestampUS < ss[j].TimestampUS })
		for i := 1; i < len(ss); i++ {
			a, b := ss[i-1], ss[i]
			dt := float64(b.TimestampUS-a.TimestampUS) / 1e6
			if dt <= 0 || dt > 0.5 {
				continue
			}
			d := math.Sqrt(sq(float64(b.X-a.X)) + sq(float64(b.Y-a.Y)) + sq(float64(b.Z-a.Z)))
			steps = append(steps, d)
			speeds = append(speeds, d/dt)
		}
	}
	// occupation : fraction de la boîte BSP réellement parcourue
	var occ [3]float64
	for ax := 0; ax < 3; ax++ {
		occ[ax] = (hi[ax] - lo[ax]) / (float64(entry.Max[ax]) - float64(entry.Min[ax]))
	}
	fmt.Printf("%-14s %s  positions=%6d  découpage film=%d/%d/%d catalogue=%d/%d/%d %s\n",
		mapName, filmID, len(pos), lay.AxisW[0], lay.AxisW[1], lay.AxisW[2],
		entry.AxisWidths[0], entry.AxisWidths[1], entry.AxisWidths[2], okStr(agree))
	fmt.Printf("   boîte BSP   x[%9.3f,%9.3f] y[%9.3f,%9.3f] z[%9.3f,%9.3f]\n",
		entry.Min[0], entry.Max[0], entry.Min[1], entry.Max[1], entry.Min[2], entry.Max[2])
	fmt.Printf("   nuage joueur x[%9.3f,%9.3f] y[%9.3f,%9.3f] z[%9.3f,%9.3f]  occupation %.0f%%/%.0f%%/%.0f%%\n",
		lo[0], hi[0], lo[1], hi[1], lo[2], hi[2], 100*occ[0], 100*occ[1], 100*occ[2])
	fmt.Printf("   quantum=%.4f/%.4f/%.4f u  pas médian=%.4f u  vitesse médiane=%.2f u/s  p99=%.2f u/s  max=%.2f u/s\n",
		quantum(entry, lay, 0), quantum(entry, lay, 1), quantum(entry, lay, 2),
		pct(steps, 0.5), pct(speeds, 0.5), pct(speeds, 0.99), pct(speeds, 1.0))
}

func quantum(e filmdec.MapQuantEntry, lay filmdec.I0Layout, ax int) float64 {
	return (float64(e.Max[ax]) - float64(e.Min[ax])) / math.Exp2(float64(lay.AxisW[ax]))
}

func okStr(b bool) string {
	if b {
		return "CONCORDE"
	}
	return "DIVERGE"
}

func sq(v float64) float64 { return v * v }

func pct(v []float64, p float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	i := int(p * float64(len(s)-1))
	return s[i]
}
