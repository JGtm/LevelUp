// Package himap — projection_triangle.go : projeter un triangle 3D sur une grille 2D.
//
// Les deux fonctions ci-dessous sont le socle PARTAGE de deux rasterisations distinctes :
// Rendu.triangleBorne (rendu.go, la carte de fond) et Rendu.poseReferenceTriangle
// (reference_navmesh.go, le tampon de reference du navmesh). Les deux bornent d'abord les
// indices de cellule touches par la boite englobante du triangle (borne), puis testent
// chaque centre de cellule par coordonnees barycentriques pour en tirer une altitude
// (altitudeAuPoint).
//
// HISTORIQUE (lot v2 G.2, 2026-09-05) : ce fichier portait aussi un troisieme
// consommateur, HeightField — un champ d'altitude par cellule pour la seule surface
// MARCHABLE (mesure du 2026-08-08 sur Cliffhanger). Approche abandonnee sans avoir jamais
// eu d'appelant hors de son propre test (handoff `.ai/V7.5/cartes/HANDOFF_PORT_TRIANGLES_
// 2026-08-08.md` §3 : le probleme qu'elle visait — le decor noyant l'arene — a ete resolu
// autrement, par l'ecretage des toits sur le rendu existant, cf. ecretage_toits.go). Les
// 175 lignes de HeightField/NewHeightField/AddMesh/faceMarchable/rasteriseTriangle/At/
// Cellule/Couverture/MinNormalZWalkable et leur test (97 lignes) sont supprimees ; git en
// garde l'historique. borne et altitudeAuPoint restent : ce sont eux que les deux
// consommateurs vivants appellent.
package himap

// borne ramene un indice de cellule dans [0, hi].
func borne(v, hi int) int {
	if v < 0 {
		return 0
	}
	if v > hi {
		return hi
	}
	return v
}

// altitudeAuPoint rend l'altitude du triangle au point (x, y) par coordonnees
// barycentriques, et dit si le point y tombe.
func altitudeAuPoint(a, b, c [3]float64, det, x, y float64) (float64, bool) {
	if det == 0 {
		return 0, false
	}
	l1 := ((b[1]-c[1])*(x-c[0]) + (c[0]-b[0])*(y-c[1])) / det
	l2 := ((c[1]-a[1])*(x-c[0]) + (a[0]-c[0])*(y-c[1])) / det
	l3 := 1 - l1 - l2
	const eps = -1e-9
	if l1 < eps || l2 < eps || l3 < eps {
		return 0, false
	}
	return l1*a[2] + l2*b[2] + l3*c[2], true
}
