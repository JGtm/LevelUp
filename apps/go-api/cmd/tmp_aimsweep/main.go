// tmp_aimsweep — recherche THROWAWAY du champ de DIRECTION dans le record biped delta.
//
// Principe : pour chaque record, on mémorise les N bits qui suivent i0 ; puis, pour chaque
// offset candidat, on décode 19 bits en direction cubemap et on mesure l'écart angulaire
// médian entre sa projection 2D et la direction de déplacement du même slot (différence
// finie des positions consécutives, pas >= 0,05 u). Le bon champ ressort par un minimum
// net ; les mauvais offsets restent à ~90° (hasard).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_aimsweep [filmDir] [maskFilter]
package main

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"

	"levelup/go-api/internal/analysis/filmdec"
)

const defaultFilm = `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\cache\film_chunks\000d5950`

// tailBits : fenêtre balayée après la fin d'i0.
const tailBits = 192

type rec struct {
	slot  uint32
	ts    uint64
	x, y  float32
	first int // index du 1er composant après i0 (-1 si aucun)
	tail  []byte
}

func main() {
	dir := defaultFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	firstFilter := 1
	if len(os.Args) > 2 {
		firstFilter, _ = strconv.Atoi(os.Args[2])
	}
	var tails []rec
	filmdec.SetRecordMaskHook(func(idx []int, pay []byte, at int) {
		r := rec{first: -1}
		if len(idx) > 1 {
			r.first = idx[1]
		}
		buf := make([]byte, tailBits/8)
		for i := 0; i < tailBits/8; i++ {
			if at+8*i+8 <= len(pay)*8 {
				buf[i] = byte(filmdec.ReadBitsAtForDiag(pay, at+8*i, 8))
			}
		}
		r.tail = buf
		tails = append(tails, r)
	})
	opt := filmdec.DefaultScanFilmOptions()
	opt.CaptureDirs = true
	opt.MaxSpeedMPS = 0    // pas de post-filtre : alignement 1:1 hook <-> positions
	opt.IsolationGapMS = 0 //
	pos, err := filmdec.ScanFilmBipedPositions(dir, opt)
	if err != nil {
		fmt.Println("erreur:", err)
		os.Exit(1)
	}
	if len(pos) != len(tails) {
		fmt.Printf("désalignement hook/positions : %d vs %d\n", len(pos), len(tails))
		os.Exit(1)
	}
	for i := range pos {
		tails[i].slot, tails[i].ts = pos[i].Slot, pos[i].TimestampUS
		tails[i].x, tails[i].y = pos[i].X, pos[i].Y
	}
	fmt.Printf("records=%d (filtre premier composant = i%d)\n", len(tails), firstFilter)

	// paires (record, cap de déplacement) du même slot, consécutives dans le temps
	type pair struct {
		r  *rec
		mv float64
	}
	bySlot := map[uint32][]*rec{}
	for i := range tails {
		bySlot[tails[i].slot] = append(bySlot[tails[i].slot], &tails[i])
	}
	var pairs []pair
	for _, l := range bySlot {
		sort.Slice(l, func(i, j int) bool { return l[i].ts < l[j].ts })
		for i := 0; i+1 < len(l); i++ {
			a, b := l[i], l[i+1]
			dx, dy := float64(b.x-a.x), float64(b.y-a.y)
			if math.Hypot(dx, dy) < 0.05 || a.first != firstFilter {
				continue
			}
			pairs = append(pairs, pair{a, math.Atan2(dy, dx)})
		}
	}
	fmt.Printf("paires exploitables=%d\n\n", len(pairs))
	if len(pairs) == 0 {
		return
	}

	type res struct {
		off, proj int
		med       float64
		n         int
	}
	var best []res
	projNames := []string{"xy", "xz", "yz", "yx", "zx", "zy"}
	for off := 0; off+19 <= tailBits; off++ {
		for proj := 0; proj < 6; proj++ {
			var degs []float64
			for _, p := range pairs {
				code := filmdec.ReadBitsAtForDiag(p.r.tail, off, 19)
				v, ok := filmdec.DecodeAimVectorChecked(code, 19)
				if !ok {
					continue
				}
				a, b := projAxes(v, proj)
				degs = append(degs, angDiff(math.Atan2(float64(b), float64(a)), p.mv))
			}
			if len(degs) < 100 {
				continue
			}
			sort.Float64s(degs)
			best = append(best, res{off, proj, degs[len(degs)/2], len(degs)})
		}
	}
	sort.Slice(best, func(i, j int) bool { return best[i].med < best[j].med })
	fmt.Println("meilleurs (offset, projection) par écart médian au cap de déplacement :")
	for i := 0; i < 15 && i < len(best); i++ {
		b := best[i]
		fmt.Printf("  offset %3d proj %-2s médiane=%6.1f° n=%d\n", b.off, projNames[b.proj], b.med, b.n)
	}
	fmt.Println("pires (contrôle, doivent avoisiner 90°) :")
	for i := len(best) - 3; i < len(best); i++ {
		if i < 0 {
			continue
		}
		b := best[i]
		fmt.Printf("  offset %3d proj %-2s médiane=%6.1f°\n", b.off, projNames[b.proj], b.med)
	}
}

func projAxes(v [3]float32, proj int) (float32, float32) {
	switch proj {
	case 0:
		return v[0], v[1]
	case 1:
		return v[0], v[2]
	case 2:
		return v[1], v[2]
	case 3:
		return v[1], v[0]
	case 4:
		return v[2], v[0]
	default:
		return v[2], v[1]
	}
}

func angDiff(a, b float64) float64 {
	d := math.Abs(a - b)
	for d > math.Pi {
		d = 2*math.Pi - d
	}
	return d * 180 / math.Pi
}
