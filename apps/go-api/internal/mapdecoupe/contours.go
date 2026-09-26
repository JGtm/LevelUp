package mapdecoupe

// contours.go — RE-VECTORISER une région binaire : de cellules allumées à des anneaux.
//
// LA MÉTHODE, et pourquoi celle-là. On ne suit pas les cellules (traçage de Moore, qui perd
// les trous et doit être relancé par composante) mais les ARÊTES entre une cellule pleine et
// une cellule vide. Chaque arête est orientée intérieur-à-droite ; les arêtes se chaînent
// alors en boucles fermées, une par frontière — extérieure ou trou, sans distinction de
// traitement. Le SIGNE de l'aire les sépare à la fin.
//
// LE POINT DÉLICAT — LE PINCEMENT. Deux cellules pleines qui ne se touchent que par un coin
// partagent un sommet du treillis : deux arêtes en sortent, et le choix décide de la
// topologie. On prend systématiquement le virage de MÊME SENS que le parcours d'une cellule
// isolée : la boucle reste alors collée à la cellule d'où l'on vient, et les deux cellules
// rendent deux anneaux simples au lieu d'un anneau en huit. C'est la lecture 4-connexe, la
// même que celle du masque (`Comble` raisonne en distance, pas en diagonale).

import "sort"

// contours extrait les boucles frontière d'une région binaire.
//
// Les boucles sont en coordonnées du TREILLIS local (coins de cellules, (nx+1) x (ny+1)),
// ouvertes (le dernier sommet ne répète pas le premier). Aire signée positive = frontière
// extérieure, négative = trou.
func contours(dedans []bool, nx, ny int) [][][2]int {
	nv := int32(nx + 1)
	plein := func(i, j int) bool {
		if i < 0 || i >= nx || j < 0 || j >= ny {
			return false
		}
		return dedans[j*nx+i]
	}
	sortantes := make(map[int32][]int32)
	ajoute := func(ai, aj, bi, bj int) {
		a := int32(aj)*nv + int32(ai)
		sortantes[a] = append(sortantes[a], int32(bj)*nv+int32(bi))
	}
	for j := 0; j < ny; j++ {
		for i := 0; i < nx; i++ {
			if !plein(i, j) {
				continue
			}
			if !plein(i, j-1) {
				ajoute(i, j, i+1, j)
			}
			if !plein(i+1, j) {
				ajoute(i+1, j, i+1, j+1)
			}
			if !plein(i, j+1) {
				ajoute(i+1, j+1, i, j+1)
			}
			if !plein(i-1, j) {
				ajoute(i, j+1, i, j)
			}
		}
	}
	return chaine(sortantes, nv)
}

// chaine consomme les arêtes sortantes et rend les boucles, dans un ordre déterministe.
func chaine(sortantes map[int32][]int32, nv int32) [][][2]int {
	departs := make([]int32, 0, len(sortantes))
	for v := range sortantes {
		departs = append(departs, v)
	}
	sort.Slice(departs, func(a, b int) bool { return departs[a] < departs[b] })

	var boucles [][][2]int
	for _, d := range departs {
		for len(sortantes[d]) > 0 {
			if b := suit(sortantes, d, nv); len(b) >= 4 {
				boucles = append(boucles, b)
			}
		}
	}
	return boucles
}

// suit parcourt une boucle depuis un sommet, en consommant les arêtes empruntées.
func suit(sortantes map[int32][]int32, depart, nv int32) [][2]int {
	var pts [][2]int
	cur := depart
	dir := [2]int{}
	for {
		suivant, ok := prend(sortantes, cur, dir, nv)
		if !ok {
			return pts
		}
		pts = append(pts, coord(cur, nv))
		dir = [2]int{coord(suivant, nv)[0] - coord(cur, nv)[0], coord(suivant, nv)[1] - coord(cur, nv)[1]}
		cur = suivant
		if cur == depart {
			return pts
		}
	}
}

// prend retire du sommet l'arête sortante la mieux classée et rend son arrivée.
func prend(sortantes map[int32][]int32, v int32, dir [2]int, nv int32) (int32, bool) {
	cands := sortantes[v]
	if len(cands) == 0 {
		return 0, false
	}
	best, bestRang := 0, 4
	for k, c := range cands {
		d := [2]int{coord(c, nv)[0] - coord(v, nv)[0], coord(c, nv)[1] - coord(v, nv)[1]}
		if r := rang(dir, d); r < bestRang {
			best, bestRang = k, r
		}
	}
	choisi := cands[best]
	reste := append(cands[:best:best], cands[best+1:]...)
	if len(reste) == 0 {
		delete(sortantes, v)
	} else {
		sortantes[v] = reste
	}
	return choisi, true
}

// rang classe un virage : 0 = même sens que le parcours d'une cellule isolée, 1 = tout
// droit, 2 = sens inverse, 3 = demi-tour. Plus petit est préféré.
//
// Le sens de référence est celui des arêtes émises par `contours` : sur une cellule seule,
// le parcours enchaîne (+1,0) puis (0,+1), dont le produit vectoriel vaut +1.
func rang(dir, d [2]int) int {
	if dir == ([2]int{}) {
		return 0
	}
	switch cr := dir[0]*d[1] - dir[1]*d[0]; {
	case cr > 0:
		return 0
	case cr == 0 && dir == d:
		return 1
	case cr < 0:
		return 2
	default:
		return 3
	}
}

// coord éclate un identifiant de sommet du treillis.
func coord(v, nv int32) [2]int { return [2]int{int(v % nv), int(v / nv)} }

// aireSignee rend DEUX FOIS l'aire algébrique d'une boucle du treillis (lacet de Gauss).
// Positive pour une frontière extérieure, négative pour un trou.
func aireSignee(pts [][2]int) int {
	s := 0
	for k := range pts {
		a, b := pts[k], pts[(k+1)%len(pts)]
		s += a[0]*b[1] - b[0]*a[1]
	}
	return s
}
