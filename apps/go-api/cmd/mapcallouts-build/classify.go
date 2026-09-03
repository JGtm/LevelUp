package main

// classify.go — le classement GRANDES / FINES des zones nommées, par la mesure.
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
// (cellules de 0,25 m, appartenance pair-impair au polygone), et le seuil est étalonné
// sur Ridgeline contre le classement du POC (11 grandes / 5 fines) — cf.
// classify_test.go, qui rejoue l'étalonnage sur le dump versionné.

// classifyCell : pas du raster de recouvrement, en mètres. Assez fin pour des zones de
// quelques dizaines de m², assez gros pour balayer 50 zones sans coûter.
const classifyCell = 0.25

// classifyCoveredMax : au-delà de cette fraction recouverte par les AUTRES zones, une
// zone est FINE (« majoritairement recouverte »). Mesuré sur Ridgeline (polygones bruts,
// cellule 0,25 m) : les 11 grandes du POC sont recouvertes à 0,00 — un pavage ne se
// recouvre pas — et les 5 fines à 0,56 (Horseshoe), 0,59 (Hex Roof), 1,00 (Hex Basement,
// Red Hallway, Lower Horseshoe). Le seuil de majorité tombe dans la marge et porte le
// sens de la règle.
const classifyCoveredMax = 0.5

// shapedPoly est une zone candidate au classement : son indice de volume et son contour.
type shapedPoly struct {
	vi   int
	poly [][2]float64
}

// classifyBig rend, par indice de volume, `true` pour les zones du pavage (grandes).
//
// Les zones sans forme ne sont pas classées (elles ne se dessinent pas) ; une carte à
// zone unique rend cette zone grande — rien ne la recouvre.
func classifyBig(zones []shapedPoly) map[int]bool {
	return classifyBigAvecPas(zones, classifyCell)
}

// classifyBigAvecPas est le même classement à PAS CHOISI.
//
// POURQUOI LE PAS EST PARAMÉTRABLE. Le raster coûte (emprise / pas)² par zone, et les
// zones d'une carte FORGE se comptent en dizaines sur un canevas de 500 m de côté : à
// 0,25 m une seule zone de 250 m rendrait un million de cellules, chacune confrontée à
// toutes les autres. Le pas natif reste celui étalonné sur Ridgeline (`classifyBig`) ;
// l'appelant Forge le desserre à emprise constante (cf. `pasDeClassement`).
func classifyBigAvecPas(zones []shapedPoly, pas float64) map[int]bool {
	r := classementRaster{zones: zones, cell: pas, boxes: make([][4]float64, len(zones))}
	for i, z := range zones {
		r.boxes[i] = bbox(z.poly)
	}
	out := make(map[int]bool, len(zones))
	for i, z := range zones {
		out[z.vi] = r.couverture(i) <= classifyCoveredMax
	}
	return out
}

// classementRaster porte le corpus de zones et le pas du raster, le temps d'un classement.
type classementRaster struct {
	zones []shapedPoly
	boxes [][4]float64
	cell  float64
}

// couverture mesure la fraction de l'emprise de la zone `self` recouverte par l'UNION des
// autres zones.
func (r classementRaster) couverture(self int) float64 {
	z, box := r.zones[self], r.boxes[self]
	inside, covered := 0, 0
	for y := box[1] + r.cell/2; y < box[3]; y += r.cell {
		for x := box[0] + r.cell/2; x < box[2]; x += r.cell {
			if !pointInPoly(z.poly, x, y) {
				continue
			}
			inside++
			for j, o := range r.zones {
				if j == self || !boxContains(r.boxes[j], x, y) {
					continue
				}
				if pointInPoly(o.poly, x, y) {
					covered++
					break
				}
			}
		}
	}
	if inside == 0 {
		return 0
	}
	return float64(covered) / float64(inside)
}

func bbox(poly [][2]float64) [4]float64 {
	b := [4]float64{poly[0][0], poly[0][1], poly[0][0], poly[0][1]}
	for _, p := range poly[1:] {
		if p[0] < b[0] {
			b[0] = p[0]
		}
		if p[1] < b[1] {
			b[1] = p[1]
		}
		if p[0] > b[2] {
			b[2] = p[0]
		}
		if p[1] > b[3] {
			b[3] = p[1]
		}
	}
	return b
}

func boxContains(b [4]float64, x, y float64) bool {
	return x >= b[0] && x <= b[2] && y >= b[1] && y <= b[3]
}

// pointInPoly : appartenance pair-impair par croisement de rayon — la même règle que le
// remplissage `evenodd` du rendu.
func pointInPoly(poly [][2]float64, x, y float64) bool {
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
