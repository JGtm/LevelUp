// tmp_objsol7 — CONTROLE INTERNE de la dichotomie « vies ti=42 persistantes = vies sans
// libelle ». THROWAWAY.
//
// L EXPLICATION BANALE A ELIMINER : si les records de ces entites sont simplement plus
// COURTS que les autres, aucun identifiant de famille de 32 bits n a la place d y tenir, et
// la dichotomie n est qu un artefact de largeur. On mesure donc la largeur d emprise de
// chaque record ti=42 et on la ventile entre les deux populations.
package main

import (
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/weaponv3"
)

const defFilm = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	known := map[uint32]string{}
	for f, n := range weaponv3.KnownWeaponHigh32 {
		known[f] = n
	}
	// slots persistants mesures par cmd/tmp_objsol6
	persist := map[int]bool{1346: true, 1347: true, 1348: true, 1349: true, 1886: true, 2046: true, 2392: true}

	n := filmdec.CountFilmChunks(dir)
	var largP, largA []int
	nommeP, nommeA := 0, 0
	for c := 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range filmdec.WalkPackets(data) {
			if pk.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			recs := filmdec.WalkKeyframeWorld(pay)
			sort.Slice(recs, func(i, j int) bool { return recs[i].Bit < recs[j].Bit })
			starts := make([]int, len(recs))
			for i, r := range recs {
				starts[i] = r.Bit
			}
			fam := map[int]int{}
			total := len(pay) * 8
			var w uint32
			for b := 0; b < total; b++ {
				w = w<<1 | uint32(bitAt(pay, b))
				if b < 31 {
					continue
				}
				if _, ok := known[w]; !ok {
					continue
				}
				if ri := recordContaining(starts, b-31); ri >= 0 {
					fam[ri]++
				}
			}
			for ri, r := range recs {
				if r.TI != 42 {
					continue
				}
				end := total
				if ri+1 < len(recs) {
					end = recs[ri+1].Bit
				}
				lg := end - r.Bit
				if persist[r.Slot] {
					largP = append(largP, lg)
					if fam[ri] > 0 {
						nommeP++
					}
				} else {
					largA = append(largA, lg)
					if fam[ri] > 0 {
						nommeA++
					}
				}
			}
		}
	}
	stat := func(nom string, v []int, nomme int) {
		if len(v) == 0 {
			fmt.Printf("  %-28s aucune donnee\n", nom)
			return
		}
		s := append([]int{}, v...)
		sort.Ints(s)
		som := 0
		for _, x := range s {
			som += x
		}
		fmt.Printf("  %-28s n=%3d · largeur min %5d · mediane %5d · moyenne %6.0f · max %6d · portant une famille %d\n",
			nom, len(s), s[0], s[len(s)/2], float64(som)/float64(len(s)), s[len(s)-1], nomme)
	}
	fmt.Println("largeur d emprise (bits) des records ti=42, par population :")
	stat("PERSISTANTS (7 slots)", largP, nommeP)
	stat("AUTRES", largA, nommeA)
}

func bitAt(b []byte, p int) byte {
	if idx := p >> 3; idx < len(b) {
		return (b[idx] >> (7 - uint(p&7))) & 1
	}
	return 0
}

func recordContaining(starts []int, at int) int {
	lo, hi := 0, len(starts)
	for lo < hi {
		mid := (lo + hi) / 2
		if starts[mid] > at {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	return lo - 1
}
