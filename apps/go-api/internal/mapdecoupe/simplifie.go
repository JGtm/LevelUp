package mapdecoupe

// simplifie.go — RAMENER un contour d'escalier à ses sommets utiles.
//
// Une frontière re-vectorisée est un escalier : un segment par côté de cellule. Une zone de
// 200 m² en compte plusieurs milliers, alors que sa forme en demande quelques dizaines. Le
// catalogue est un fichier VERSIONNÉ servi au rejeu : la simplification n'est pas du confort,
// c'est ce qui garde l'asset transportable.
//
// LA TOLÉRANCE RESTE SOUS LA CELLULE (`SimplifieParDefaut`). Au-delà, on déplacerait la
// frontière de plus que ce que le masque sait résoudre — on inventerait de la précision.

import "math"

// simplifieAnneau réduit un anneau fermé (donné ouvert) sous une tolérance en mètres.
//
// L'anneau est d'abord tourné pour DÉMARRER sur son sommet lexicographiquement le plus
// petit : le résultat ne dépend alors plus de l'endroit où le chaînage a commencé, et deux
// exécutions rendent le même fichier.
func simplifieAnneau(pts [][2]float64, eps float64) [][2]float64 {
	if len(pts) < 4 || eps <= 0 {
		return pts
	}
	d := 0
	for k, p := range pts {
		if p[0] < pts[d][0] || (p[0] == pts[d][0] && p[1] < pts[d][1]) {
			d = k
		}
	}
	chaine := make([][2]float64, 0, len(pts)+1)
	chaine = append(chaine, pts[d:]...)
	chaine = append(chaine, pts[:d]...)
	chaine = append(chaine, pts[d])
	out := rdp(chaine, eps)
	return out[:len(out)-1]
}

// rdp : Ramer-Douglas-Peucker sur une polyligne ouverte.
//
// Sur un anneau, la corde de départ est dégénérée (mêmes extrémités) : le premier point
// retenu est alors le plus ÉLOIGNÉ du départ, ce qui donne les deux ancres opposées dont une
// courbe fermée a besoin. C'est voulu, pas un effet de bord.
func rdp(pts [][2]float64, eps float64) [][2]float64 {
	if len(pts) < 3 {
		return pts
	}
	a, b := pts[0], pts[len(pts)-1]
	imax, dmax := 0, 0.0
	for i := 1; i < len(pts)-1; i++ {
		if d := distanceSegment(pts[i], a, b); d > dmax {
			imax, dmax = i, d
		}
	}
	if dmax <= eps {
		return [][2]float64{a, b}
	}
	gauche := rdp(pts[:imax+1], eps)
	droite := rdp(pts[imax:], eps)
	return append(gauche[:len(gauche)-1:len(gauche)-1], droite...)
}

// distanceSegment rend la distance d'un point au segment [a, b] (à `a` si le segment est
// dégénéré).
func distanceSegment(p, a, b [2]float64) float64 {
	vx, vy := b[0]-a[0], b[1]-a[1]
	wx, wy := p[0]-a[0], p[1]-a[1]
	l2 := vx*vx + vy*vy
	if l2 == 0 {
		return math.Hypot(wx, wy)
	}
	t := math.Max(0, math.Min(1, (wx*vx+wy*vy)/l2))
	return math.Hypot(wx-t*vx, wy-t*vy)
}
