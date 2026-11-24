package tactical

// reapparition_evaluation.go — CE QUE VAUT UNE PREVISION DE REAPPARITION.
//
// Trois echelles, de la plus stricte a la plus lisible :
//
//   - le POINT EXACT : le candidat reel est-il le premier, ou parmi les trois premiers ?
//   - le RAYON : un des candidats retenus est-il a moins de RayonEvaluationM du point reel ?
//     (des points voisins sont une meme zone pour qui regarde la carte) ;
//   - la ZONE NOMMEE : la zone du point reel est-elle la premiere, ou parmi les trois
//     premieres, quand une zone recoit la SOMME des probabilites de ses candidats ?
//
// Le hasard de chaque echelle est l'esperance exacte d'un tirage uniforme parmi les candidats :
// pour le rayon, la chance que le premier tirage (HasardRayon1), ou qu'au moins un des trois
// tirages sans remise (HasardRayon3), tombe a moins de RayonEvaluationM du point reel.
// Les egalites de score se partagent sans jamais favoriser le reel : un rang fractionnaire pour
// le point exact et la zone, l'esperance d'un depart au hasard entre ex aequo pour le rayon.

import (
	"math"
	"sort"
)

// RayonEvaluationM est le rayon de l'echelle « rayon », en unites du jeu : 8 unites (24,4 m).
const RayonEvaluationM = 8.0

// Metriques est le resultat d'une evaluation, en pourcentages (0..100).
type Metriques struct {
	N             int
	HorsCandidats int

	Hasard1, Hasard3 float64
	Top1, Top3       float64
	RangMoyen        float64
	// HasardRayon1 : la part des candidats a moins de RayonEvaluationM du point reel (un tirage) ;
	// HasardRayon3 : la chance que l'un des trois premiers tirages sans remise en soit.
	HasardRayon1, HasardRayon3 float64
	Rayon1, Rayon3             float64
	ZonesMesurees              int
	HasardZone1, HasardZone3   float64
	Zone1, Zone3               float64
}

// Evaluer note des predictions (un tableau de scores par echantillon, dans l'ordre des
// candidats). Les echantillons dont la prediction est nil sont ignores.
func Evaluer(echs []Echantillon, scores [][]float64) Metriques {
	var m Metriques
	var h1, h3, t1, t3, rang, hr1, hr3, r1, r3 float64
	var hz1, hz3, z1, z3 float64
	for i, s := range echs {
		if scores[i] == nil {
			continue
		}
		m.N++
		k := float64(len(s.Feats))
		h1 += 1 / k
		h3 += math.Min(3, k) / k
		if s.Reel < 0 {
			m.HorsCandidats++
			rang += k
		} else {
			a, b, c := rangsExacts(scores[i], s.Reel)
			t1, t3, rang = t1+a, t3+b, rang+c
		}
		dedans, nCand := candidatsDansLeRayon(s), len(s.Pos)
		hr1 += partTouchee(nCand, dedans, 1)
		hr3 += partTouchee(nCand, dedans, 3)
		r1 += toucheRayon(s, scores[i], 1)
		r3 += toucheRayon(s, scores[i], 3)
		if s.Zones != nil && s.ZoneReelle >= 0 {
			m.ZonesMesurees++
			nz, a, b := rangsZone(s, scores[i])
			hz1 += 1 / float64(nz)
			hz3 += math.Min(3, float64(nz)) / float64(nz)
			z1, z3 = z1+a, z3+b
		}
	}
	if m.N == 0 {
		return m
	}
	n := float64(m.N)
	m.Hasard1, m.Hasard3 = 100*h1/n, 100*h3/n
	m.Top1, m.Top3, m.RangMoyen = 100*t1/n, 100*t3/n, rang/n
	m.HasardRayon1, m.HasardRayon3 = 100*hr1/n, 100*hr3/n
	m.Rayon1, m.Rayon3 = 100*r1/n, 100*r3/n
	if m.ZonesMesurees > 0 {
		nz := float64(m.ZonesMesurees)
		m.HasardZone1, m.HasardZone3 = 100*hz1/nz, 100*hz3/nz
		m.Zone1, m.Zone3 = 100*z1/nz, 100*z3/nz
	}
	return m
}

// rangsExacts rend (top1, top3, rang+1) du candidat reel : un rang 0 est le premier ; les
// egalites donnent un rang fractionnaire.
func rangsExacts(sc []float64, reel int) (top1, top3, rang float64) {
	a := sc[reel]
	au, egal := 0.0, 0.0
	for j, v := range sc {
		if j == reel {
			continue
		}
		if v > a {
			au++
		} else if v == a {
			egal++
		}
	}
	r := au + egal/2
	if r < 1 {
		top1 = 1 - r
	}
	if r < 3 {
		top3 = math.Min(1, 3-r)
	}
	return top1, top3, r + 1
}

// candidatsDansLeRayon compte les candidats a moins de RayonEvaluationM du point reel.
func candidatsDansLeRayon(s Echantillon) int {
	dedans := 0
	for _, p := range s.Pos {
		if p.Distance(s.Au) <= RayonEvaluationM {
			dedans++
		}
	}
	return dedans
}

// partTouchee est la chance qu'au moins un de `tirages` candidats pris au hasard, sans remise,
// parmi `n` dont `k` sont dans le rayon, y soit : 1 - C(n-k, m) / C(n, m), avec m = min(tirages, n).
// Zero sans candidat dans le rayon, 1 quand les tirages epuisent les candidats et que k >= 1.
func partTouchee(n, k, tirages int) float64 {
	if n <= 0 || k <= 0 {
		return 0
	}
	m := min(tirages, n)
	rate := 1.0
	for i := 0; i < m; i++ {
		if n-k-i <= 0 {
			return 1
		}
		rate *= float64(n-k-i) / float64(n-i)
	}
	return 1 - rate
}

// toucheRayon rend l'esperance que l'un des `n` premiers candidats soit dans le rayon du point
// reel. Les candidats de meme score forment un groupe : un groupe entierement retenu compte pour
// tous ses membres, le groupe a la frontiere se tire au hasard (partTouchee), de sorte que l'ordre
// des candidats ne favorise personne, et le reel pas plus qu'un autre.
func toucheRayon(s Echantillon, sc []float64, n int) float64 {
	idx := make([]int, len(sc))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return sc[idx[a]] > sc[idx[b]] })
	retenus := 0
	for debut := 0; debut < len(idx) && retenus < n; {
		fin, dedans := debut, 0
		for fin < len(idx) && sc[idx[fin]] == sc[idx[debut]] {
			if s.Pos[idx[fin]].Distance(s.Au) <= RayonEvaluationM {
				dedans++
			}
			fin++
		}
		groupe := fin - debut
		if retenus+groupe > n {
			return partTouchee(groupe, dedans, n-retenus)
		}
		if dedans > 0 {
			return 1
		}
		retenus += groupe
		debut = fin
	}
	return 0
}

// rangsZone somme les probabilites par zone, et rend le nombre de zones candidates puis
// (top1, top3) de la zone reelle. Un candidat sans zone (-1) ne compte dans aucune.
func rangsZone(s Echantillon, sc []float64) (nbZones int, top1, top3 float64) {
	p := Probabilites(sc)
	somme := map[int]float64{}
	for i, z := range s.Zones {
		if z >= 0 {
			somme[z] += p[i]
		}
	}
	nbZones = len(somme)
	if _, ok := somme[s.ZoneReelle]; !ok {
		// La zone reelle ne porte aucun candidat : elle ne peut pas etre predite.
		if nbZones == 0 {
			nbZones = 1
		}
		return nbZones, 0, 0
	}
	a := somme[s.ZoneReelle]
	au, egal := 0.0, 0.0
	for z, v := range somme {
		if z == s.ZoneReelle {
			continue
		}
		if v > a {
			au++
		} else if v == a {
			egal++
		}
	}
	r := au + egal/2
	if r < 1 {
		top1 = 1 - r
	}
	if r < 3 {
		top3 = math.Min(1, 3-r)
	}
	return nbZones, top1, top3
}
