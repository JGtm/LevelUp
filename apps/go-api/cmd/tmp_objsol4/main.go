// tmp_objsol4 — position d i0 CHERCHEE DANS CHAQUE RECORD de keyframe, et non a un offset
// fixe. THROWAWAY.
//
// CE QUE LA MESURE PRECEDENTE A REFUTE (cmd/tmp_objsol3) : l offset d i0 depuis le debut du
// record n est PAS constant. Le meilleur offset fixe sur ti=35 ne place que 18 records sur
// 184 dans la boite de jeu (9,8 %). C est attendu une fois dit : le default-state a une
// largeur VARIABLE (c est une fonction du bitstream, cf. BipedDefaultStateEndBit) et le
// masque aussi (creux R(3)+6n contre dense R(64)).
//
// CE QU ON FAIT A LA PLACE : dans l emprise de chaque record, on cherche la PREMIERE
// position de bit ou (a) la porte d i0 vaut 0 sur ses 5 bits, (b) aucun axe n est sature,
// (c) le point tombe dans la boite de jeu. La boite ne fait que 1,9 % du volume de la carte
// et la porte coute 1 chance sur 32 : le motif est rare par construction.
//
// LE TAUX DE FAUX POSITIFS EST MESURE, PAS CALCULE, et il l est sur une population dont la
// reponse est connue : les records ti=35. A l instant du keyframe, la position de chaque
// bipede est deja donnee par le decodeur DE PRODUCTION (ScanFilmBipedPositions, la chaine
// qui alimente les trajectoires du POC). On compare donc la position trouvee a la position
// attendue, slot par slot. C est un oracle INTERNE : ni capture, ni releve humain.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_objsol4 [filmDir]
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
	chunk, pkt int
	tSec       float64
	tsUS       uint64
	pay        []byte
	recs       []filmdec.KeyframeRec
}

type hit struct {
	offset int
	v      [3]float32
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
	var xs, ys, zs []float32
	for _, p := range pos {
		xs, ys, zs = append(xs, p.X), append(ys, p.Y), append(zs, p.Z)
	}
	box := boxOf(xs, ys, zs, 0.005, 1.5)
	fmt.Printf("decoupage i0 %s · %d positions de bipede (production)\n", lay, len(pos))
	fmt.Printf("boite de jeu x[%.2f, %.2f] y[%.2f, %.2f] z[%.2f, %.2f] = %.3f %% du volume\n\n",
		box[0], box[1], box[2], box[3], box[4], box[5], 100*boxFraction(box))

	kfs := loadKeyframes(dir)
	// bornes de record : le record suivant dans le meme keyframe (par bit croissant)
	for i := range kfs {
		sort.Slice(kfs[i].recs, func(a, b int) bool { return kfs[i].recs[a].Bit < kfs[i].recs[b].Bit })
	}

	// ---------- ORACLE : position attendue par (keyframe, slot) ----------
	// ScanFilmBipedPositions rend des echantillons horodates ; on prend le plus proche du
	// keyframe, a 300 ms au plus.
	bySlot := map[uint32][]filmdec.BipedPosition{}
	for _, p := range pos {
		bySlot[p.Slot] = append(bySlot[p.Slot], p)
	}
	for s := range bySlot {
		v := bySlot[s]
		sort.Slice(v, func(a, b int) bool { return v[a].TimestampUS < v[b].TimestampUS })
		bySlot[s] = v
	}

	fmt.Println("=== TEMOIN POSITIF : records ti=35, premiere position plausible du record ===")
	var nRec, nHit, nOracle, nJuste int
	var ecarts []float64
	var offs []int
	for _, k := range kfs {
		for ri, r := range k.recs {
			if r.TI != 35 {
				continue
			}
			nRec++
			end := recEnd(k, ri)
			h, ok := firstHit(k.pay, r.Bit+64, end, lay, box)
			if !ok {
				continue
			}
			nHit++
			offs = append(offs, h.offset-r.Bit)
			exp, okE := nearest(bySlot[uint32(r.Slot)], k.tsUS, 300_000)
			if !okE {
				continue
			}
			nOracle++
			d := math.Sqrt(dist2(h.v, [3]float32{exp.X, exp.Y, exp.Z}))
			ecarts = append(ecarts, d)
			if d < 2.0 {
				nJuste++
			}
		}
	}
	fmt.Printf("  records ti=35 : %d · un candidat trouve : %d · oracle disponible : %d\n", nRec, nHit, nOracle)
	if nOracle > 0 {
		sort.Float64s(ecarts)
		fmt.Printf("  ECART A L ORACLE : mediane %.2f m · p90 %.2f m · max %.2f m\n",
			ecarts[len(ecarts)/2], ecarts[int(0.9*float64(len(ecarts)-1))], ecarts[len(ecarts)-1])
		fmt.Printf("  JUSTES a moins de 2 m : %d / %d = %.1f %%\n",
			nJuste, nOracle, 100*float64(nJuste)/float64(nOracle))
	}
	if len(offs) > 0 {
		sort.Ints(offs)
		fmt.Printf("  offset du candidat depuis le debut du record : min %d · mediane %d · max %d\n\n",
			offs[0], offs[len(offs)/2], offs[len(offs)-1])
	}

	// ---------- ti=42 / ti=37 ----------
	for _, ti := range []int{42, 37} {
		fmt.Printf("=== ti=%d ===\n", ti)
		n, found := 0, 0
		perSlot := map[int][][3]float32{}
		var o2 []int
		for _, k := range kfs {
			for ri, r := range k.recs {
				if r.TI != ti {
					continue
				}
				n++
				end := recEnd(k, ri)
				h, ok := firstHit(k.pay, r.Bit+64, end, lay, box)
				if !ok {
					continue
				}
				found++
				o2 = append(o2, h.offset-r.Bit)
				perSlot[r.Slot] = append(perSlot[r.Slot], h.v)
			}
		}
		fmt.Printf("  records %d · candidat trouve %d (%.1f %%) · slots touches %d\n",
			n, found, 100*float64(found)/float64(n), len(perSlot))
		if len(o2) > 0 {
			sort.Ints(o2)
			fmt.Printf("  offset depuis le debut du record : min %d · mediane %d · max %d\n",
				o2[0], o2[len(o2)/2], o2[len(o2)-1])
		}
		stable, pairs := 0, 0
		for _, vs := range perSlot {
			for i := 1; i < len(vs); i++ {
				pairs++
				if dist2(vs[i], vs[i-1]) < 0.25 {
					stable++
				}
			}
		}
		fmt.Printf("  STABILITE (meme slot, deux keyframes) : %d paires a moins de 0,5 m sur %d\n\n",
			stable, pairs)
	}
}

// recEnd : borne haute de l emprise d un record = debut du record suivant (ou fin du payload).
func recEnd(k kf, ri int) int {
	if ri+1 < len(k.recs) {
		return k.recs[ri+1].Bit
	}
	return len(k.pay) * 8
}

// firstHit cherche la premiere position de bit de [from, to) ou le motif d i0 absolu se
// referme dans la boite de jeu.
func firstHit(pay []byte, from, to int, lay filmdec.I0Layout, box [6]float32) (hit, bool) {
	need := lay.TotalBits()
	if to > len(pay)*8 {
		to = len(pay) * 8
	}
	for p := from; p+need <= to; p++ {
		if readBits(pay, p, lay.GateBits) != 0 {
			continue
		}
		var v [3]float32
		ok := true
		for ax := 0; ax < 3; ax++ {
			q := readBits(pay, p+lay.AxisOffset(ax), int(lay.AxisW[ax]))
			if q == 0 || q == uint32(1)<<lay.AxisW[ax]-1 {
				ok = false
				break
			}
			v[ax] = filmdec.DequantBipedAxis(q, ax, lay, worldRange)
		}
		if ok && inBox(v, box) {
			return hit{offset: p, v: v}, true
		}
	}
	return hit{}, false
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

func boxOf(xs, ys, zs []float32, trim float64, margin float32) [6]float32 {
	q := func(v []float32, p float64) float32 {
		s := append([]float32{}, v...)
		sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
		return s[int(p*float64(len(s)-1))]
	}
	return [6]float32{
		q(xs, trim) - margin, q(xs, 1-trim) + margin,
		q(ys, trim) - margin, q(ys, 1-trim) + margin,
		q(zs, trim) - margin, q(zs, 1-trim) + margin,
	}
}

func boxFraction(b [6]float32) float64 {
	vol := float64(b[1]-b[0]) * float64(b[3]-b[2]) * float64(b[5]-b[4])
	tot := 1.0
	for ax := 0; ax < 3; ax++ {
		tot *= float64(worldRange[ax].Max - worldRange[ax].Min)
	}
	return vol / tot
}

func inBox(v [3]float32, b [6]float32) bool {
	return v[0] >= b[0] && v[0] <= b[1] && v[1] >= b[2] && v[1] <= b[3] && v[2] >= b[4] && v[2] <= b[5]
}

func dist2(a, b [3]float32) float64 {
	dx, dy, dz := float64(a[0]-b[0]), float64(a[1]-b[1]), float64(a[2]-b[2])
	return dx*dx + dy*dy + dz*dz
}

func loadKeyframes(dir string) []kf {
	n := filmdec.CountFilmChunks(dir)
	var out []kf
	var t0 uint64
	first := true
	for c := 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range filmdec.WalkPackets(data) {
			if first {
				t0, first = pk.TimestampUS, false
			}
			if pk.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			pay := append([]byte{}, pk.Payload(data)...)
			out = append(out, kf{
				chunk: c, pkt: pk.Index, tSec: float64(pk.TimestampUS-t0) / 1e6,
				tsUS: pk.TimestampUS, pay: pay, recs: filmdec.WalkKeyframeWorld(pay),
			})
		}
	}
	return out
}
