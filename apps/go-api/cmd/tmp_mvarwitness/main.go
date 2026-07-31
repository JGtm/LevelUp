// tmp_mvarwitness — TEMOIN chiffre de l'identification des objectifs.
//
// La symetrie miroir attendue par le brief s'est revelee INAPPLICABLE (aucune des
// cartes disponibles n'a un placement d'objets symetrique — mesure par vote). Le
// temoin de remplacement, de force equivalente, est la CO-LOCALISATION : deux
// labels independants (flag_spawn, flag_delivery) portes par deux type_id
// DIFFERENTS doivent tomber au meme point pour une meme equipe. Rien dans le
// decodage ne force ce resultat : les positions sont lues champ par champ, les
// labels viennent d'une table de hash construite separement.
//
// Temoin negatif : meme mesure sur des paires d'objets tirees au hasard.
package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

func main() {
	for _, path := range os.Args[1:] {
		buf, err := os.ReadFile(path)
		if err != nil {
			fmt.Println("erreur:", err)
			continue
		}
		v, err := mapvar.Parse(buf)
		if err != nil {
			fmt.Println("erreur:", err)
			continue
		}
		fmt.Printf("===== %s (%d objets) =====\n", path, len(v.Objects))
		objs := v.Objectives()

		spawns := map[int][]mapvar.Objective{}
		delivs := map[int][]mapvar.Objective{}
		for _, o := range objs {
			switch o.Role {
			case mapvar.RoleFlagSpawn:
				spawns[o.TeamIndex] = append(spawns[o.TeamIndex], o)
			case mapvar.RoleFlagDelivery:
				delivs[o.TeamIndex] = append(delivs[o.TeamIndex], o)
			}
		}
		teams := []int{-1, 0, 1}
		for _, tm := range teams {
			for _, d := range delivs[tm] {
				best := math.Inf(1)
				var bt int32
				for _, s := range spawns[tm] {
					if dd := dist(s.Pos, d.Pos); dd < best {
						best, bt = dd, s.TypeID
					}
				}
				if math.IsInf(best, 1) {
					fmt.Printf("  equipe %2d : livraison sans apparition appariee\n", tm)
					continue
				}
				fmt.Printf("  equipe %2d : |flag_delivery(type %d) - flag_spawn(type %d)| = %.4f m\n",
					tm, d.TypeID, bt, best)
			}
		}

		// Temoin negatif : distances entre objets pris au hasard.
		rng := rand.New(rand.NewSource(20260726))
		var neg []float64
		for i := 0; i < 5000; i++ {
			a := v.Objects[rng.Intn(len(v.Objects))]
			b := v.Objects[rng.Intn(len(v.Objects))]
			neg = append(neg, dist(a.Pos, b.Pos))
		}
		sort.Float64s(neg)
		fmt.Printf("  TEMOIN NEGATIF : distance entre 2 objets au hasard (n=5000) p01=%.2f m median=%.2f m\n",
			neg[50], neg[2500])

		// Temoin negatif 2 : la distance minimale entre deux objets QUELCONQUES du
		// fichier, pour montrer que 0.005 m n'est pas simplement la densite locale.
		minAll := math.Inf(1)
		for i := range v.Objects {
			for j := i + 1; j < len(v.Objects); j++ {
				if d := dist(v.Objects[i].Pos, v.Objects[j].Pos); d < minAll {
					minAll = d
				}
			}
		}
		fmt.Printf("  TEMOIN NEGATIF 2 : distance minimale entre 2 objets quelconques = %.4f m\n", minAll)
		n, tot := countUnder(v.Objects, 0.02)
		fmt.Printf("  TEMOIN NEGATIF 3 : paires d'objets sous 0.02 m = %d sur %d (%.4f %%)\n",
			n, tot, 100*float64(n)/float64(tot))
	}
}

func dist(a, b mapvar.Vec3) float64 {
	dx, dy, dz := a.X-b.X, a.Y-b.Y, a.Z-b.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// countUnder compte les paires d'objets sous un seuil, pour situer l'ecart mesure
// dans la distribution reelle du fichier (anti-tautologie).
func countUnder(objs []mapvar.Object, seuil float64) (int, int) {
	n, total := 0, 0
	for i := range objs {
		for j := i + 1; j < len(objs); j++ {
			total++
			if dist(objs[i].Pos, objs[j].Pos) < seuil {
				n++
			}
		}
	}
	return n, total
}
