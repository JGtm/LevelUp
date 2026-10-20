package tactical

// geometrie.go — la forme au sol d'une zone nommée : appartenance d'un point et distance au bord,
// en mètres monde, vue du dessus.
//
// LA FORME EST CELLE QUE LE RENDU DESSINE : le contour principal, les parties détachées et les
// trous, remplis en règle pair-impair. Un point appartient à la forme quand il est dans un nombre
// IMPAIR de ses anneaux — dans le contour et dans un trou, il n'y est pas ; dans une partie, il y
// est. La distance au bord est la plus courte distance à une arête de l'un des anneaux : depuis un
// trou, le bord le plus proche est celui du trou.

import "math"

// anneauxDe rend les anneaux d'une zone : contour, parties, trous. Un anneau de moins de trois
// sommets n'est pas une surface et n'est pas rendu.
func anneauxDe(z ZoneNommee) [][][2]float64 {
	out := make([][][2]float64, 0, 1+len(z.Parties)+len(z.Trous))
	for _, a := range append(append([][][2]float64{z.Polygone}, z.Parties...), z.Trous...) {
		if len(a) >= 3 {
			out = append(out, a)
		}
	}
	return out
}

// aUneForme dit si la zone porte au moins un contour ou une partie exploitable.
func aUneForme(z ZoneNommee) bool {
	if len(z.Polygone) >= 3 {
		return true
	}
	for _, p := range z.Parties {
		if len(p) >= 3 {
			return true
		}
	}
	return false
}

// dansAnneau : le point est-il à l'intérieur de l'anneau (test du rayon, pair-impair) ?
func dansAnneau(x, y float64, anneau [][2]float64) bool {
	dedans := false
	for i, k := 0, len(anneau)-1; i < len(anneau); k, i = i, i+1 {
		xi, yi := anneau[i][0], anneau[i][1]
		xk, yk := anneau[k][0], anneau[k][1]
		if (yi > y) != (yk > y) && x < (xk-xi)*(y-yi)/(yk-yi)+xi {
			dedans = !dedans
		}
	}
	return dedans
}

// dansForme : le point appartient-il à la forme de la zone (règle pair-impair sur ses anneaux) ?
func dansForme(x, y float64, z ZoneNommee) bool {
	dedans := false
	for _, a := range anneauxDe(z) {
		if dansAnneau(x, y, a) {
			dedans = !dedans
		}
	}
	return dedans
}

// distanceAuBord rend la plus courte distance du point à une arête de la forme (+Inf sans forme).
func distanceAuBord(x, y float64, z ZoneNommee) float64 {
	best := math.Inf(1)
	for _, a := range anneauxDe(z) {
		for i, k := 0, len(a)-1; i < len(a); k, i = i, i+1 {
			best = math.Min(best, distanceAuSegment(x, y, a[k], a[i]))
		}
	}
	return best
}

// distanceAuSegment : distance du point au segment [a, b].
func distanceAuSegment(x, y float64, a, b [2]float64) float64 {
	dx, dy := b[0]-a[0], b[1]-a[1]
	l2 := dx*dx + dy*dy
	t := 0.0
	if l2 > 0 {
		t = math.Max(0, math.Min(1, ((x-a[0])*dx+(y-a[1])*dy)/l2))
	}
	return math.Hypot(x-(a[0]+t*dx), y-(a[1]+t*dy))
}

// distanceAZone : 0 quand le point est dans la forme, sa distance au bord sinon.
func distanceAZone(x, y float64, z ZoneNommee) float64 {
	if dansForme(x, y, z) {
		return 0
	}
	return distanceAuBord(x, y, z)
}
