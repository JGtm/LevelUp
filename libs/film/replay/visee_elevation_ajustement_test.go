package replay

// visee_elevation_ajustement_test.go — LA STATISTIQUE de l'item E.0.1 : l'ajustement du quantum
// angulaire, et la regression qui le controle.
//
// ELLE VIT A PART parce qu'elle ne connait NI le film, NI les kills, NI le pont slot -> xuid :
// elle ne prend que des couples (pas de quantum, geometrie observee) et rend un quantum. La
// separer rend visible qu'aucune piece du decodage n'entre dans l'estimation, et garde le
// fichier de l'oracle sous le seuil de 500 lignes du depot.

import "math"

// aimQuantumCandidats : les deux conventions candidates, en degres par pas.
//
//	180/2048 = le quantum du CAP (360/4096) : plage +/- 90 deg sur TOUT le champ R(11).
//	360/2048 = deux fois ce quantum : le champ couvre +/- 180 deg et le tangage, borne a
//	           +/- 90 deg par le jeu, n'en occupe que la MOITIE (1024 pas utiles).
var aimQuantumCandidats = []struct {
	nom  string
	degs float64
}{
	{"180/2048 (quantum du cap)", 180.0 / 2048},
	{"360/2048 (deux fois le quantum du cap)", 360.0 / 2048},
}

// aimAjuste estime le quantum angulaire EN CORRIGEANT le biais de hauteur de visee.
//
// POURQUOI UN AJUSTEMENT A DEUX PARAMETRES, ET PAS LA SIMPLE PENTE. `elevDeg` est calcule entre
// deux ORIGINES de bipede, alors que le tir part de l'OEIL du tueur et arrive sur le CORPS de la
// victime : il manque une hauteur h, constante, que la geometrie transforme en une erreur
// d'angle INVERSEMENT proportionnelle a la distance. C'est elle qui ecrase le rapport
// angle/pas aux courtes portees et qui faisait varier la pente d'un facteur trois selon la
// tranche d'amplitude. Le modele l'absorbe :
//
//	dz = dxy * tan(c * pas) - h
//
// c (degres par pas) et h (metres) sortent ensemble. h est un CONTROLE PHYSIQUE : s'il tombe
// autour d'un demi-metre, le modele decrit bien la geometrie du tir ; s'il tombe a dix metres,
// c'est le modele qui est faux, et c alors ne vaut rien.
func aimAjuste(pts []aimPoint) (c, h, r2 float64) {
	meilleur := math.Inf(1)
	for pas := 0; pas <= 3000; pas++ {
		cand := 0.02 + float64(pas)*0.0001
		hh, sse := aimResidu(pts, cand)
		if sse < meilleur {
			meilleur, c, h = sse, cand, hh
		}
	}
	var moy float64
	for _, p := range pts {
		moy += p.dz
	}
	moy /= float64(len(pts))
	var sst float64
	for _, p := range pts {
		sst += (p.dz - moy) * (p.dz - moy)
	}
	if sst > 0 {
		r2 = 1 - meilleur/sst
	}
	return c, h, r2
}

// aimResidu rend le decalage de hauteur optimal pour un quantum donne, et la somme des carres.
func aimResidu(pts []aimPoint, c float64) (h, sse float64) {
	res := make([]float64, len(pts))
	for i, p := range pts {
		res[i] = p.dz - p.dxy*math.Tan(c*float64(p.pas)*math.Pi/180)
	}
	for _, v := range res {
		h += v
	}
	h /= float64(len(res))
	for _, v := range res {
		sse += (v - h) * (v - h)
	}
	return h, sse
}

// aimRegression rend la pente, l'ordonnee a l'origine et la correlation de elevDeg sur pas.
func aimRegression(pts []aimPoint) (pente, ord, r float64) {
	n := float64(len(pts))
	var sx, sy, sxx, syy, sxy float64
	for _, p := range pts {
		x, y := float64(p.pas), p.elevDeg
		sx, sy, sxx, syy, sxy = sx+x, sy+y, sxx+x*x, syy+y*y, sxy+x*y
	}
	varX, varY := n*sxx-sx*sx, n*syy-sy*sy
	cov := n*sxy - sx*sy
	if varX == 0 || varY == 0 {
		return 0, 0, 0
	}
	pente = cov / varX
	ord = (sy - pente*sx) / n
	r = cov / math.Sqrt(varX*varY)
	return pente, ord, r
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
