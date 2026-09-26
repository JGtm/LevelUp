package himap

// distance_masque.go — LA DISTANCE EXACTE PLUTOT QU'UNE DILATATION EN ESCALIER.
//
// LE DEFAUT, signale par l'utilisateur le 2026-08-30 sur Dredge : « y a du crenelage ». Il ne
// vient pas du reglage mais de la FORME de l'operation. `dilate` grossit un masque en repetant un
// voisinage carre : chaque cellule semee devient un carre, et l'union de carres poses tous les
// demi-metres — le pas du corpus, soit une douzaine de pixels du fond — dessine un escalier.
//
// LA SORTIE : mesurer la vraie distance au point seme le plus proche, puis seuiller. Le bord
// devient l'union de DISQUES, c'est-a-dire une courbe, et il ne depend plus de l'orientation des
// axes. C'est aussi plus juste : « a moins de quatre metres d'un endroit parcouru » est une
// phrase sur des distances, pas sur des carres.
//
// LE CALCUL est une transformee de distance par chanfrein en deux balayages (Borgefors) : un
// passage avant, un passage arriere, avec les poids 5 / 7 / 11 pour les voisins orthogonaux,
// diagonaux et « cavalier ». L'erreur reste sous 2 % de la distance vraie, contre 8 % pour le
// chanfrein 3-4 classique, pour un cout identique — deux balayages lineaires, sans file ni tri.

import "math"

// poids du chanfrein 5-7-11, normalises par `chanfreinUnite`.
const (
	chanfreinDroit = 5
	chanfreinDiag  = 7
	chanfreinSaut  = 11
	chanfreinUnite = 5.0
)

// distanceAuxSemis rend, pour chaque cellule, la distance approchee en CELLULES jusqu'au semis le
// plus proche. Une grille sans aucun semis rend des distances infinies.
func distanceAuxSemis(sem []bool, nx, ny int) []float64 {
	const inf = math.MaxInt32
	d := make([]int, len(sem))
	for k := range d {
		if sem[k] {
			d[k] = 0
		} else {
			d[k] = inf
		}
	}
	// Voisins du passage AVANT, en (di, dj, poids) ; le passage arriere prend les opposes.
	avant := [][3]int{
		{-1, 0, chanfreinDroit}, {0, -1, chanfreinDroit},
		{-1, -1, chanfreinDiag}, {1, -1, chanfreinDiag},
		{-2, -1, chanfreinSaut}, {-1, -2, chanfreinSaut},
		{1, -2, chanfreinSaut}, {2, -1, chanfreinSaut},
	}
	balaye := func(ordreInverse bool) {
		for t := 0; t < nx*ny; t++ {
			k := t
			if ordreInverse {
				k = nx*ny - 1 - t
			}
			if d[k] == 0 {
				continue
			}
			i, j := k%nx, k/nx
			for _, v := range avant {
				di, dj := v[0], v[1]
				if ordreInverse {
					di, dj = -di, -dj
				}
				ii, jj := i+di, j+dj
				if ii < 0 || jj < 0 || ii >= nx || jj >= ny {
					continue
				}
				if s := d[jj*nx+ii] + v[2]; s < d[k] {
					d[k] = s
				}
			}
		}
	}
	balaye(false)
	balaye(true)
	out := make([]float64, len(d))
	for k, v := range d {
		if v >= inf {
			out[k] = math.Inf(1)
			continue
		}
		out[k] = float64(v) / chanfreinUnite
	}
	return out
}

// masqueAutourDesSemis rend les cellules a moins de `rayon` cellules d'un semis.
func masqueAutourDesSemis(sem []bool, nx, ny int, rayon float64) []bool {
	d := distanceAuxSemis(sem, nx, ny)
	out := make([]bool, len(sem))
	for k, v := range d {
		out[k] = v <= rayon
	}
	return out
}
