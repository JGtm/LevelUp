package positions

import "sort"

// minClusterSize : nb minimal de positions par groupe pour qu'un team-split soit
// crédible (évite de couper du bruit). Spawn = 4 joueurs / équipe (§N).
const minClusterSize = 3

// minGapRatio : le vide séparant les 2 clusters sur l'axe X doit valoir au moins
// ce ratio de l'étendue totale pour qu'on ose un split (sépare 2 spawns nets).
const minGapRatio = 0.35

// assignTeamsBestEffort tente un team-split spatial sur l'axe X (signature spawn
// §N : 2 clusters nets, high-ground vs low-ground). Mute les positions en place :
// Team 0/1 si un gap franc sépare 2 groupes suffisants, sinon laisse TeamUnknown.
//
// Honnête : aucune attribution forcée. Sans clustering net (mêlée générale,
// trop peu de points), tout reste TeamUnknown.
func assignTeamsBestEffort(ps []PlayerPosition) {
	if len(ps) < 2*minClusterSize {
		return
	}
	idx := make([]int, len(ps))
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(a, b int) bool { return ps[idx[a]].X < ps[idx[b]].X })

	xmin, xmax := ps[idx[0]].X, ps[idx[len(idx)-1]].X
	span := float64(xmax - xmin)
	if span <= 0 {
		return
	}

	splitPos, ok := largestGapSplit(ps, idx, span)
	if !ok {
		return
	}
	for rank, i := range idx {
		if rank < splitPos {
			ps[i].Team = 0
		} else {
			ps[i].Team = 1
		}
	}
}

// largestGapSplit cherche, parmi les positions triées par X (via idx), le plus
// grand vide entre deux X consécutifs. Renvoie le rang de coupe (taille du
// groupe bas) et true si le vide est assez franc ET que les deux groupes
// atteignent minClusterSize.
func largestGapSplit(ps []PlayerPosition, idx []int, span float64) (int, bool) {
	bestGap := 0.0
	bestRank := -1
	for rank := 1; rank < len(idx); rank++ {
		gap := float64(ps[idx[rank]].X - ps[idx[rank-1]].X)
		if gap > bestGap {
			bestGap = gap
			bestRank = rank
		}
	}
	if bestRank < 0 {
		return 0, false
	}
	lowSize := bestRank
	highSize := len(idx) - bestRank
	if lowSize < minClusterSize || highSize < minClusterSize {
		return 0, false
	}
	if bestGap < minGapRatio*span {
		return 0, false
	}
	return bestRank, true
}
