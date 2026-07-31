// tmp_objsol3 — LOCALISATION d i0 (object-position-component) dans les records de KEYFRAME,
// par archetype, au balayage d offset. THROWAWAY.
//
// POURQUOI LES KEYFRAMES ET PAS LES DELTAS. Mesure de cmd/tmp_objsol2 : le detecteur de
// records delta rend 5 echantillons sur les 178 slots ti=42 declares, contre 1 006 sur un
// jeu de slots FANTOME de meme cardinalite. Le signal est SOUS le bruit — la voie delta est
// refusee, sur piece. Explication coherente avec la grammaire : un objet pose ne bouge pas,
// il n emet donc aucun delta d i0 ; porte par un joueur il est parente (i10), sa position
// n est plus repliquee.
//
// LA METHODE. Un record de keyframe vaut [id:32][field:26][ti:6] puis un default-state de
// largeur PROPRE A L ARCHETYPE (vtable[0x60], connue seulement pour ti=35), puis la porte,
// le masque, puis les composants — i0 en tete. La largeur du default-state de ti=37 et
// ti=42 n est pas portee : on la mesure en BALAYANT l offset et en gardant celui qui fait
// tomber les positions dans la BOITE DE JEU.
//
// LA BOITE DE JEU N EST PAS DEVINEE : c est l enveloppe (centiles 0,5 / 99,5) des positions
// de bipede rendues par le decodeur DE PRODUCTION, ScanFilmBipedPositions — la meme chaine
// qui alimente les trajectoires du POC, validee au quantum exact.
//
// TEMOIN POSITIF, ET IL DOIT PASSER LE PREMIER : le meme balayage sur ti=35 doit trouver UN
// offset qui fait tomber les huit bipedes de chaque keyframe dans cette boite, nettement
// au-dessus du bruit. Si le temoin echoue, aucun resultat sur ti=42 n est publiable.
//
// TAUX DE FAUX POSITIFS : MESURE, jamais calcule — le balayage publie la moyenne et le
// 99e centile du score sur les 837 offsets essayes.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_objsol3 [filmDir]
package main

import (
	"fmt"
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
	pay        []byte
	recs       []filmdec.KeyframeRec
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
	fmt.Printf("decoupage i0 lu dans le film : %s\n", lay)

	// ---------- BOITE DE JEU : le decodeur de PRODUCTION ----------
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
	fmt.Printf("boite de jeu = enveloppe des %d positions de bipede du decodeur de production\n", len(pos))
	fmt.Printf("  x[%.2f, %.2f] y[%.2f, %.2f] z[%.2f, %.2f] — %.3f %% du volume de la carte\n\n",
		box[0], box[1], box[2], box[3], box[4], box[5], 100*boxFraction(box))

	kfs := loadKeyframes(dir)
	fmt.Printf("%d keyframes charges\n\n", len(kfs))

	for _, ti := range []int{35, 42, 37, 38, 41, 43} {
		sweep(kfs, ti, lay, box)
	}
}

func countTI(kfs []kf, ti int) int {
	n := 0
	for _, k := range kfs {
		for _, r := range k.recs {
			if r.TI == ti {
				n++
			}
		}
	}
	return n
}

type scoreRow struct {
	off             int
	nGate0, nInBox  int
	nStable, nPairs int
	total           int
}

func sweep(kfs []kf, ti int, lay filmdec.I0Layout, box [6]float32) {
	const lo, hi = 64, 900
	total := countTI(kfs, ti)
	if total == 0 {
		fmt.Printf("=== ti=%d : aucun record, balayage saute ===\n\n", ti)
		return
	}
	rows := make([]scoreRow, 0, hi-lo+1)
	for off := lo; off <= hi; off++ {
		row := scoreRow{off: off, total: total}
		bySlot := map[int][][3]float32{}
		for _, k := range kfs {
			for _, r := range k.recs {
				if r.TI != ti {
					continue
				}
				g, v, ok := decode(k.pay, r.Bit+off, lay)
				if !ok || g != 0 {
					continue
				}
				row.nGate0++
				if inBox(v, box) {
					row.nInBox++
					bySlot[r.Slot] = append(bySlot[r.Slot], v)
				}
			}
		}
		for _, vs := range bySlot {
			for i := 1; i < len(vs); i++ {
				row.nPairs++
				if dist2(vs[i], vs[i-1]) < 0.25 { // 0,5 m au carre
					row.nStable++
				}
			}
		}
		rows = append(rows, row)
	}
	scores := make([]int, len(rows))
	sum := 0
	for i, r := range rows {
		scores[i] = r.nInBox
		sum += r.nInBox
	}
	sort.Ints(scores)
	moy := float64(sum) / float64(len(rows))
	p99 := scores[int(0.99*float64(len(scores)-1))]
	med := scores[len(scores)/2]

	best := append([]scoreRow{}, rows...)
	sort.Slice(best, func(i, j int) bool {
		if best[i].nInBox != best[j].nInBox {
			return best[i].nInBox > best[j].nInBox
		}
		if best[i].nStable != best[j].nStable {
			return best[i].nStable > best[j].nStable
		}
		return best[i].off < best[j].off
	})
	fmt.Printf("=== ti=%d : %d records cumules · balayage offset %d..%d (%d offsets) ===\n",
		ti, total, lo, hi, len(rows))
	fmt.Printf("  BRUIT MESURE sur les %d offsets : mediane %d · moyenne %.1f · 99e centile %d (sur %d records)\n",
		len(rows), med, moy, p99, total)
	fmt.Println("  offset | porte0 | dansBoite | part du total | stables/paires")
	for i := 0; i < 10 && i < len(best); i++ {
		r := best[i]
		fmt.Printf("  %+6d | %6d | %9d | %5.1f | %d/%d\n",
			r.off, r.nGate0, r.nInBox, 100*float64(r.nInBox)/float64(total), r.nStable, r.nPairs)
	}
	fmt.Println()
}

func decode(pay []byte, i0 int, lay filmdec.I0Layout) (uint32, [3]float32, bool) {
	if i0 < 0 || i0+lay.TotalBits() > len(pay)*8 {
		return 0, [3]float32{}, false
	}
	g := readBits(pay, i0, lay.GateBits)
	var v [3]float32
	for ax := 0; ax < 3; ax++ {
		q := readBits(pay, i0+lay.AxisOffset(ax), int(lay.AxisW[ax]))
		if q == 0 || q == uint32(1)<<lay.AxisW[ax]-1 {
			return g, v, false // quantum sature : valeur ecretee, pas une position
		}
		v[ax] = filmdec.DequantBipedAxis(q, ax, lay, worldRange)
	}
	return g, v, true
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
				pay: pay, recs: filmdec.WalkKeyframeWorld(pay),
			})
		}
	}
	return out
}
