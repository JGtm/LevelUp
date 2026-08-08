// diagnose.go — POURQUOI une action n'est pas attribuee.
//
// Un taux d'attribution faible a deux causes opposees, et les confondre coute une
// journee :
//
//	le joueur n'etait PAS dans la zone — fait de jeu, ou semantique de la statistique
//	                                     (une recompense d'equipe n'exige pas d'y etre) ;
//	les deux reperes ne coincident PAS  — les positions du film et les positions de la
//	                                     variante de carte ne parlent pas du meme monde.
//
// Deux mesures les separent. La DISTANCE au volume : quelques metres, c'est de la
// semantique ; des centaines, c'est un repere. Et l'ECART VERTICAL au centre de la zone la
// plus proche : un decalage de repere se voit surtout sur l'axe que le rejeu 2D n'affiche
// pas et que personne ne regarde.
package main

import (
	"fmt"
	"math"
	"sort"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/analysis/replay/mapvar"
)

// distanceBuckets : bornes de la distribution imprimee, en metres. La premiere est de
// l'ordre d'une zone (rayon max mesure 5,0 m), la derniere de l'ordre d'une carte.
var distanceBuckets = []float64{0, 2, 5, 10, 20, 50, 100}

type distanceStats struct {
	distance []float64
	vertical []float64
	byStat   map[string][2]int // stat -> {attribuees, total}
}

func newDistanceStats() *distanceStats {
	return &distanceStats{byStat: map[string][2]int{}}
}

// add cumule une action : sa distance a la zone la plus proche, et son sort par stat.
func (d *distanceStats) add(a replay.ZoneAttribution, zones []replay.Zone) {
	c := d.byStat[a.Action.Stat]
	c[1]++
	if a.Attributed {
		c[0]++
	}
	d.byStat[a.Action.Stat] = c
	if !a.HasSample {
		return
	}
	d.distance = append(d.distance, a.DistanceM)
	if v, ok := verticalOffsetToNearestCenter(zones, a.Sample); ok {
		d.vertical = append(d.vertical, v)
	}
}

// verticalOffsetToNearestCenter rend l'ecart vertical signe au centre de la zone
// horizontalement la plus proche.
func verticalOffsetToNearestCenter(zones []replay.Zone, p replay.Point) (float64, bool) {
	best, out, ok := math.Inf(1), 0.0, false
	for _, z := range zones {
		h := math.Hypot(float64(p.X)-z.Center.X, float64(p.Y)-z.Center.Y)
		if h < best {
			best, out, ok = h, float64(p.Z)-z.Center.Z, true
		}
	}
	return out, ok
}

func (d *distanceStats) print() {
	fmt.Println("DIAGNOSTIC — ou sont les joueurs par rapport a la zone la plus proche")
	if len(d.distance) == 0 {
		fmt.Println("  aucune position exploitable.")
		return
	}
	dist := sortedCopy(d.distance)
	vert := sortedCopy(d.vertical)
	fmt.Printf("  distance au volume (m) : p25 %.1f | mediane %.1f | p75 %.1f | max %.1f\n",
		quantile(dist, 0.25), quantile(dist, 0.5), quantile(dist, 0.75), dist[len(dist)-1])
	if len(vert) > 0 {
		// Une mediane proche de zero dit que les deux reperes se superposent : c'est le
		// controle qui elimine l'hypothese « bug de repere » sans autre mesure.
		fmt.Printf("  ecart vertical     (m) : p25 %+.1f | mediane %+.1f | p75 %+.1f\n",
			quantile(vert, 0.25), quantile(vert, 0.5), quantile(vert, 0.75))
	}
	for _, b := range distanceBuckets {
		fmt.Printf("  a %3.0f m ou moins : %5.1f %%\n", b, 100*fractionAtMost(dist, b))
	}
	fmt.Println()
	fmt.Println("  par statistique (seuil strict) :")
	stats := make([]string, 0, len(d.byStat))
	for s := range d.byStat {
		stats = append(stats, s)
	}
	sort.Strings(stats)
	for _, s := range stats {
		c := d.byStat[s]
		fmt.Printf("    %-16s %4d / %4d dedans (%.1f %%)\n",
			s, c[0], c[1], 100*float64(c[0])/float64(c[1]))
	}
}

// mapvarOffset construit le decalage du temoin spatial (x ET y).
func mapvarOffset(m float64) mapvar.Vec3 { return mapvar.Vec3{X: m, Y: m} }

// quantile sur une tranche DEJA triee.
func quantile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	return sorted[int(q*float64(len(sorted)-1))]
}

// fractionAtMost rend la part des valeurs <= bound (bornes INCLUSIVES, comme le test
// d'appartenance : a zero, c'est exactement la part des joueurs DANS une zone).
func fractionAtMost(sorted []float64, bound float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	n := sort.SearchFloat64s(sorted, math.Nextafter(bound, math.Inf(1)))
	return float64(n) / float64(len(sorted))
}
