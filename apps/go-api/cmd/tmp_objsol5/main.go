// tmp_objsol5 — LE TEST DECISIF. Pour chaque offset candidat d i0 dans un record de
// keyframe ti=35, on ne demande plus « la position tombe-t-elle dans la boite de jeu » mais
// « tombe-t-elle sur la position que le decodeur de PRODUCTION donne au meme bipede au meme
// instant ». C est un oracle interne, exact au metre. THROWAWAY.
//
// Si un offset atteint une concordance forte, le principe « i0 a un offset fixe pour une
// part des records » est etabli sur une population dont la reponse est connue, et il devient
// transferable a ti=42 / ti=37 avec son taux d erreur MESURE.
// S il n en existe aucun, la voie keyframe est refutee comme la voie delta l a ete, et c est
// ce qu il faut ecrire.
package main

import (
	"fmt"
	"math"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const defFilm = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

var worldRange = filmdec.Vec3Range{
	{Min: -41.102932, Max: 72.109375},
	{Min: -56.606728, Max: 57.211975},
	{Min: -84.37054, Max: 53.179653},
}

type kf struct {
	tsUS uint64
	pay  []byte
	recs []filmdec.KeyframeRec
}

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	lay, _, err := filmdec.DetectI0Layout(dir)
	if err != nil {
		fmt.Println("decoupage i0:", err)
		return
	}
	opt := filmdec.DefaultScanFilmOptions()
	opt.WorldRange = &worldRange
	pos, err := filmdec.ScanFilmBipedPositions(dir, opt)
	if err != nil {
		fmt.Println("trajectoires:", err)
		return
	}
	bySlot := map[uint32][]filmdec.BipedPosition{}
	for _, p := range pos {
		bySlot[p.Slot] = append(bySlot[p.Slot], p)
	}
	kfs := loadKeyframes(dir)
	fmt.Printf("decoupage %s · %d positions de production · %d keyframes\n\n", lay, len(pos), len(kfs))

	// population : records ti=35 pour lesquels l oracle repond a moins de 300 ms
	type cible struct {
		pay  []byte
		bit  int
		want [3]float32
	}
	var pop []cible
	for _, k := range kfs {
		for _, r := range k.recs {
			if r.TI != 35 {
				continue
			}
			e, ok := nearest(bySlot[uint32(r.Slot)], k.tsUS, 300_000)
			if !ok {
				continue
			}
			pop = append(pop, cible{k.pay, r.Bit, [3]float32{e.X, e.Y, e.Z}})
		}
	}
	fmt.Printf("population de controle : %d records ti=35 a oracle disponible\n\n", len(pop))

	const lo, hi = 64, 1400
	type row struct {
		off, box    int
		j2, j5, j15 int
		bestD       float64
	}
	rows := make([]row, 0, hi-lo+1)
	for off := lo; off <= hi; off++ {
		r := row{off: off, bestD: 1e9}
		for _, c := range pop {
			v, ok := decode(c.pay, c.bit+off, lay)
			if !ok {
				continue
			}
			r.box++
			d := math.Sqrt(dist2(v, c.want))
			if d < r.bestD {
				r.bestD = d
			}
			if d < 2 {
				r.j2++
			}
			if d < 5 {
				r.j5++
			}
			if d < 15 {
				r.j15++
			}
		}
		rows = append(rows, r)
	}
	t2, t5, t15 := 0, 0, 0
	for _, r := range rows {
		t2 += r.j2
		t5 += r.j5
		t15 += r.j15
	}
	fmt.Printf("CUMUL sur %d offsets : <2m %d · <5m %d · <15m %d (population %d par offset)\n",
		len(rows), t2, t5, t15, len(pop))
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].j15 != rows[j].j15 {
			return rows[i].j15 > rows[j].j15
		}
		return rows[i].box > rows[j].box
	})
	fmt.Println("meilleurs offsets, classes par concordance a moins de 15 m :")
	for i := 0; i < 12 && i < len(rows); i++ {
		r := rows[i]
		fmt.Printf("  offset %+5d : <2m %3d · <5m %3d · <15m %3d · lisibles %3d/%d · meilleur ecart %.2f m\n",
			r.off, r.j2, r.j5, r.j15, r.box, len(pop), r.bestD)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].box > rows[j].box })
	fmt.Println("offsets les plus LISIBLES (porte nulle + aucun axe sature) :")
	for i := 0; i < 10 && i < len(rows); i++ {
		r := rows[i]
		fmt.Printf("  offset %+5d : lisibles %3d/%d · <2m %3d · <15m %3d · meilleur ecart %.2f m\n",
			r.off, r.box, len(pop), r.j2, r.j15, r.bestD)
	}
}

func decode(pay []byte, i0 int, lay filmdec.I0Layout) ([3]float32, bool) {
	var v [3]float32
	if i0 < 0 || i0+lay.TotalBits() > len(pay)*8 {
		return v, false
	}
	if readBits(pay, i0, lay.GateBits) != 0 {
		return v, false
	}
	for ax := 0; ax < 3; ax++ {
		q := readBits(pay, i0+lay.AxisOffset(ax), int(lay.AxisW[ax]))
		if q == 0 || q == uint32(1)<<lay.AxisW[ax]-1 {
			return v, false
		}
		v[ax] = filmdec.DequantBipedAxis(q, ax, lay, worldRange)
	}
	return v, true
}

func nearest(v []filmdec.BipedPosition, ts uint64, tol uint64) (filmdec.BipedPosition, bool) {
	best, bestD, ok := filmdec.BipedPosition{}, uint64(1)<<62, false
	for _, p := range v {
		d := p.TimestampUS - ts
		if p.TimestampUS < ts {
			d = ts - p.TimestampUS
		}
		if d < bestD {
			best, bestD, ok = p, d, true
		}
	}
	return best, ok && bestD <= tol
}

func readBits(b []byte, pos, n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		p := pos + i
		var bit uint32
		if idx := p >> 3; idx < len(b) {
			bit = uint32(b[idx]>>(7-uint(p&7))) & 1
		}
		v = v<<1 | bit
	}
	return v
}

func dist2(a, b [3]float32) float64 {
	dx, dy, dz := float64(a[0]-b[0]), float64(a[1]-b[1]), float64(a[2]-b[2])
	return dx*dx + dy*dy + dz*dz
}

func loadKeyframes(dir string) []kf {
	n := filmdec.CountFilmChunks(dir)
	var out []kf
	for c := 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range filmdec.WalkPackets(data) {
			if pk.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			pay := append([]byte{}, pk.Payload(data)...)
			out = append(out, kf{tsUS: pk.TimestampUS, pay: pay, recs: filmdec.WalkKeyframeWorld(pay)})
		}
	}
	return out
}
