package mapcatalog

// callouts_classement.go — le classement GRANDES / FINES des zones nommées, par la mesure.
//
// CE QUE LE POC A ÉTABLI (Ridgeline, rendu de référence) : les GRANDES zones pavent la
// carte — aplat pair-impair et frontières — quand les zones FINES sont des étages
// imbriqués (toit, sous-sol, couloir intérieur) qui se superposent en 2D aux grandes et
// entre elles : les remplir rendrait la carte illisible, elles ne portent qu'un contour
// pointillé.
//
// LA RÈGLE DÉRIVE DU PAVAGE, elle n'est pas déclarée à la main : une zone dont l'emprise
// 2D est majoritairement RECOUVERTE par les autres zones dessinées n'appartient pas au
// pavage — c'est un étage posé par-dessus. Le recouvrement se mesure au raster
// (appartenance pair-impair au polygone), et le seuil est étalonné sur Ridgeline contre le
// classement du POC (11 grandes / 5 fines) — cf. callouts_classement_test.go, qui rejoue
// l'étalonnage sur le dump versionné.
//
// UN SEUL EXEMPLAIRE pour les deux chaînes qui produisent des zones : la passe native de
// `cmd/mapcallouts-build` et l'entrée Forge (callouts_entree.go), que la CLI et le rattrapage
// au fetch de film partagent. Deux classements divergeraient au premier réglage, et la même
// carte se dessinerait différemment selon le chemin qui l'a mise au catalogue.

import "math"

// PasDeClassementNatif : pas du raster de recouvrement, en mètres. Assez fin pour des zones de
// quelques dizaines de m², assez gros pour balayer 50 zones sans coûter.
const PasDeClassementNatif = 0.25

// recouvrementFineMin : au-delà de cette fraction recouverte par les AUTRES zones, une zone
// est FINE (« majoritairement recouverte »). Mesuré sur Ridgeline (polygones bruts, cellule
// 0,25 m) : les 11 grandes du POC sont recouvertes à 0,00 — un pavage ne se recouvre pas — et
// les 5 fines à 0,56 (Horseshoe), 0,59 (Hex Roof), 1,00 (Hex Basement, Red Hallway, Lower
// Horseshoe). Le seuil de majorité tombe dans la marge et porte le sens de la règle.
const recouvrementFineMin = 0.5

// cellulesDeClassementMax : nombre de cellules visé sur le plus grand côté d'une zone lors du
// classement d'une carte Forge. Le pas natif ferait exploser le raster sur un canevas de 500 m ;
// à 200 cellules la mesure de recouvrement garde le même sens (elle compare des SURFACES, pas
// des contours) pour un coût borné.
const cellulesDeClassementMax = 200

// FormeDeZone est une zone candidate au classement : son indice et son contour.
type FormeDeZone struct {
	Index   int
	Contour [][2]float64
}

// ClasserGrandes rend, par indice, `true` pour les zones du pavage (grandes), au pas natif.
//
// Les zones sans forme ne sont pas classées (elles ne se dessinent pas) ; une carte à zone
// unique rend cette zone grande — rien ne la recouvre.
func ClasserGrandes(formes []FormeDeZone) map[int]bool {
	return classerGrandesAuPas(formes, PasDeClassementNatif)
}

// classerGrandesAuPas est le même classement à PAS CHOISI : le raster coûte (emprise / pas)²
// par zone, et une carte Forge se dessine sur un canevas de plusieurs centaines de mètres
// (cf. `pasDeClassementForge`).
func classerGrandesAuPas(formes []FormeDeZone, pas float64) map[int]bool {
	r := classementRaster{formes: formes, cell: pas, boites: make([][4]float64, len(formes))}
	for i, f := range formes {
		r.boites[i] = boiteDe(f.Contour)
	}
	out := make(map[int]bool, len(formes))
	for i, f := range formes {
		out[f.Index] = r.couverture(i) <= recouvrementFineMin
	}
	return out
}

// pasDeClassementForge choisit le pas du raster d'une carte Forge : le pas natif tant qu'il
// tient dans `cellulesDeClassementMax` cellules sur le plus grand côté, desserré au-delà.
func pasDeClassementForge(formes []FormeDeZone) float64 {
	cote := 0.0
	for _, f := range formes {
		b := boiteDe(f.Contour)
		cote = math.Max(cote, math.Max(b[2]-b[0], b[3]-b[1]))
	}
	if pas := cote / cellulesDeClassementMax; pas > PasDeClassementNatif {
		return pas
	}
	return PasDeClassementNatif
}

// classementRaster porte le corpus de zones et le pas du raster, le temps d'un classement.
type classementRaster struct {
	formes []FormeDeZone
	boites [][4]float64
	cell   float64
}

// couverture mesure la fraction de l'emprise de la zone `self` recouverte par l'UNION des
// autres zones.
func (r classementRaster) couverture(self int) float64 {
	z, box := r.formes[self], r.boites[self]
	dedans, couvertes := 0, 0
	for y := box[1] + r.cell/2; y < box[3]; y += r.cell {
		for x := box[0] + r.cell/2; x < box[2]; x += r.cell {
			if !pointDansPolygone(z.Contour, x, y) {
				continue
			}
			dedans++
			if r.couvertParUneAutre(self, x, y) {
				couvertes++
			}
		}
	}
	if dedans == 0 {
		return 0
	}
	return float64(couvertes) / float64(dedans)
}

// couvertParUneAutre dit si le point (x, y) tombe dans une zone autre que `self`.
func (r classementRaster) couvertParUneAutre(self int, x, y float64) bool {
	for j, o := range r.formes {
		if j == self || !boiteContient(r.boites[j], x, y) {
			continue
		}
		if pointDansPolygone(o.Contour, x, y) {
			return true
		}
	}
	return false
}

// boiteDe rend la boîte englobante (minX, minY, maxX, maxY) d'un contour non vide.
func boiteDe(poly [][2]float64) [4]float64 {
	b := [4]float64{poly[0][0], poly[0][1], poly[0][0], poly[0][1]}
	for _, p := range poly[1:] {
		b[0] = math.Min(b[0], p[0])
		b[1] = math.Min(b[1], p[1])
		b[2] = math.Max(b[2], p[0])
		b[3] = math.Max(b[3], p[1])
	}
	return b
}

func boiteContient(b [4]float64, x, y float64) bool {
	return x >= b[0] && x <= b[2] && y >= b[1] && y <= b[3]
}

// pointDansPolygone : appartenance pair-impair par croisement de rayon — la même règle que le
// remplissage `evenodd` du rendu.
func pointDansPolygone(poly [][2]float64, x, y float64) bool {
	in := false
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		xi, yi := poly[i][0], poly[i][1]
		xj, yj := poly[j][0], poly[j][1]
		if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
			in = !in
		}
	}
	return in
}
