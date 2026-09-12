package replay

// pads_proximity_geometry_test.go — LA GEOMETRIE DE LA MESURE DE PROXIMITE : approcher un socle,
// couper une trajectoire a une fenetre de temps, et resumer une distribution.
//
// EXTRAIT DE `pads_proximity_research_test.go` LE 2026-08-27 (revue adversariale) : l'instrument
// depassait le seuil de taille du depot. La coupure n'est pas arbitraire — d'un cote le PROTOCOLE
// (corpus, denominateurs, rapport, verdict), de l'autre ces fonctions PURES qui ne connaissent ni
// artefact ni seuil et qu'on peut lire, verifier et corriger seules.
//
// TOUT EST EN 2D, EN COORDONNEES MONDE : l'altitude d'un socle ne dit rien de l'accessibilite —
// un joueur passe SOUS un ratelier mural sans que la distance 3D le raconte.

import (
	"math"
	"sort"
)

// padsProxWindow est CE QU'ON INTERROGE : un socle (cx, cy) et la fenetre d'incertitude d'une de
// ses occupations, en frames. Les quatre voyagent ensemble parce qu'aucune des fonctions de
// geometrie n'a de sens sans les quatre.
type padsProxWindow struct {
	cx, cy float64
	lo, hi float64
}

func padsProxWindowOf(pad WeaponPad, occ PadPresence) padsProxWindow {
	return padsProxWindow{
		cx: float64(pad.X), cy: float64(pad.Y),
		lo: float64(occ.TLow), hi: float64(occ.THigh),
	}
}

// padsProxFirstApproach rend l'instant (en frames, interpole) de la PREMIERE approche a moins de
// `r` du socle dans [tLow, tHigh], et `false` si aucun joueur n'y passe.
//
// LA RECHERCHE EST BORNEE PAR LA FENETRE, et c'est ce qui la rend tenable : les points d'une
// trace sont tries, une dichotomie donne le premier de la fenetre, et on remonte d'UN point pour
// attraper le segment qui y ENTRE.
func padsProxFirstApproach(tracks []Track, w padsProxWindow, r float64) (float64, bool) {
	best, found := math.Inf(1), false
	for _, tr := range tracks {
		pts := tr.Points
		if len(pts) == 0 {
			continue
		}
		if len(pts) == 1 {
			if t := float64(pts[0].T); t >= w.lo && t <= w.hi &&
				math.Hypot(float64(pts[0].X)-w.cx, float64(pts[0].Y)-w.cy) <= r {
				best, found = math.Min(best, t), true
			}
			continue
		}
		i := sort.Search(len(pts), func(k int) bool { return float64(pts[k].T) >= w.lo })
		if i > 0 {
			i-- // le segment qui ENTRE dans la fenetre part du point precedent
		}
		for ; i+1 < len(pts) && float64(pts[i].T) <= w.hi; i++ {
			if t, ok := padsProxSegmentTouch(pts[i], pts[i+1], w, r); ok && t < best {
				best, found = t, true
			}
		}
	}
	return best, found
}

// padsProxMinDistance rend la distance MINIMALE du socle aux trajectoires pendant la fenetre, le
// nombre d'ECHANTILLONS qu'elle contient, et si une trajectoire la COUVRE.
//
// LES DEUX DERNIERS NE SE CONFONDENT PAS : une fenetre etroite peut n'abriter aucun point tout en
// etant traversee par un segment. Elle est alors COUVERTE — sa distance est mesuree, pas devinee.
func padsProxMinDistance(tracks []Track, w padsProxWindow) (float64, int, bool) {
	best, n := math.Inf(1), 0
	for _, tr := range tracks {
		pts := tr.Points
		i := sort.Search(len(pts), func(k int) bool { return float64(pts[k].T) >= w.lo })
		for k := i; k < len(pts) && float64(pts[k].T) <= w.hi; k++ {
			n++
		}
		if i > 0 {
			i--
		}
		for ; i+1 < len(pts) && float64(pts[i].T) <= w.hi; i++ {
			if d, ok := padsProxSegmentDistance(pts[i], pts[i+1], w); ok && d < best {
				best = d
			}
		}
	}
	return best, n, !math.IsInf(best, 1)
}

// padsProxSegmentTouch rend le premier instant ou le segment [a, b], COUPE a la fenetre, passe a
// moins de `r` du socle.
//
// LE SEGMENT EST COUPE AVANT D'ETRE TESTE : un joueur qui frole le socle une seconde APRES la
// preuve d'absence ne dit rien de la disparition, et le compter avancerait la mesure.
func padsProxSegmentTouch(a, b Point, w padsProxWindow, r float64) (float64, bool) {
	a, b, d0, d1, ok := padsProxClip(a, b, w)
	if !ok {
		return 0, false
	}
	ax, ay := padsProxAt(a, b, d0)
	bx, by := padsProxAt(a, b, d1)
	u, touche := padsProxFirstWithin(ax-w.cx, ay-w.cy, bx-ax, by-ay, r)
	if !touche {
		return 0, false
	}
	return d0 + u*(d1-d0), true
}

// padsProxSegmentDistance rend la distance du socle au segment [a, b] COUPE a la fenetre.
func padsProxSegmentDistance(a, b Point, w padsProxWindow) (float64, bool) {
	a, b, d0, d1, ok := padsProxClip(a, b, w)
	if !ok {
		return 0, false
	}
	ax, ay := padsProxAt(a, b, d0)
	bx, by := padsProxAt(a, b, d1)
	dx, dy := bx-ax, by-ay
	u := 0.0
	if aa := dx*dx + dy*dy; aa > 0 {
		u = math.Min(1, math.Max(0, ((w.cx-ax)*dx+(w.cy-ay)*dy)/aa))
	}
	return math.Hypot(ax+u*dx-w.cx, ay+u*dy-w.cy), true
}

// padsProxClip remet le segment dans l'ordre du temps et rend les deux instants ou il croise la
// fenetre, ou `false` s'il n'y entre pas.
func padsProxClip(a, b Point, w padsProxWindow) (Point, Point, float64, float64, bool) {
	ta, tb := float64(a.T), float64(b.T)
	if tb < ta {
		ta, tb = tb, ta
		a, b = b, a
	}
	d0, d1 := math.Max(ta, w.lo), math.Min(tb, w.hi)
	return a, b, d0, d1, d1 >= d0
}

// padsProxAt interpole la position sur le segment [a, b] a l'instant `t` (frames). Un segment de
// duree nulle rend son point de depart.
func padsProxAt(a, b Point, t float64) (float64, float64) {
	ta, tb := float64(a.T), float64(b.T)
	if tb <= ta {
		return float64(a.X), float64(a.Y)
	}
	u := (t - ta) / (tb - ta)
	return float64(a.X) + u*(float64(b.X)-float64(a.X)), float64(a.Y) + u*(float64(b.Y)-float64(a.Y))
}

// padsProxFirstWithin rend le plus petit `u` de [0, 1] pour lequel le point `p + u*d` est a moins
// de `r` de l'origine — c'est-a-dire la premiere entree du segment dans le disque du socle.
//
// C'EST UNE EQUATION DU SECOND DEGRE, et il faut sa PREMIERE racine, pas la distance minimale :
// ce qu'on date, c'est l'ENTREE dans le rayon, pas le point le plus proche du passage.
func padsProxFirstWithin(px, py, dx, dy, r float64) (float64, bool) {
	aa := dx*dx + dy*dy
	cc := px*px + py*py - r*r
	if aa == 0 {
		return 0, cc <= 0
	}
	bb := 2 * (px*dx + py*dy)
	disc := bb*bb - 4*aa*cc
	if disc < 0 {
		return 0, false
	}
	root := math.Sqrt(disc)
	u0, u1 := (-bb-root)/(2*aa), (-bb+root)/(2*aa)
	if u1 < 0 || u0 > 1 {
		return 0, false
	}
	return math.Max(u0, 0), true
}

// padsProxCountWithin compte les passages a moins de `r`.
func padsProxCountWithin(dists []float64, r float64) int {
	n := 0
	for _, x := range dists {
		if x <= r {
			n++
		}
	}
	return n
}

// padsProxRatio rend une part, ou NaN quand le denominateur est vide (jamais 0 %, qui se lirait
// comme une mesure).
func padsProxRatio(num, den int) float64 {
	if den == 0 {
		return math.NaN()
	}
	return float64(num) / float64(den)
}

// padsProxQuantile rend le quantile d'un echantillon (NaN s'il est vide). Interpolation lineaire
// entre les deux rangs encadrants — la meme convention que les deciles du cycle.
func padsProxQuantile(xs []float64, q float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	if len(s) == 1 {
		return s[0]
	}
	pos := q * float64(len(s)-1)
	i := int(math.Floor(pos))
	if i >= len(s)-1 {
		return s[len(s)-1]
	}
	return s[i] + (pos-float64(i))*(s[i+1]-s[i])
}
