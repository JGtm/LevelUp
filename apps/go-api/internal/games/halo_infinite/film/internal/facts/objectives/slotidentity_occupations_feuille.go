package objectives

import (
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// slotidentity_occupations_feuille.go — LA FEUILLE DE MATCH FACE AUX SIEGES RECYCLES (lot R1) : le
// triplet de la DERNIERE occupation, et la garde qui empeche le triplet de manche entiere de donner
// un occupant a un second slot. Le decoupage lui-meme vit dans slotidentity_occupations.go.

// occupantAilleurs dit qu'un xuid occupe un AUTRE siege recycle de la manche : la completion par
// le triplet ne le donne alors pas en plus a `slot` (aucun xuid deux fois).
func (ri RoundIdentity) occupantAilleurs(round, slot int, xuid string) bool {
	for s, occ := range ri.occupations[round] {
		if s == slot {
			continue
		}
		for _, o := range occ {
			if o.XUID == xuid {
				return true
			}
		}
	}
	return false
}

// withLastOccupantsByLines nomme, par le TRIPLET DE SON SEGMENT, la derniere occupation d'un siege
// recycle que les instants de mort n'ont pas nommee (un remplacant qui meurt moins de trois fois).
// Mono-manche seulement, comme le triplet de la manche entiere (le triplet apparie des totaux de
// match) ; la derniere occupation seulement, parce que les totaux de la feuille d'un joueur arrive
// en cours sont ceux de son segment, et qu'un occupant anterieur peut avoir rejoint un autre siege.
// Trois gardes : un triplet nul ne prouve rien, une seule ligne de la feuille doit le porter, et son
// xuid ne doit etre porte par aucun autre slot de la manche.
func (ri RoundIdentity) withLastOccupantsByLines(recs []types.StatRecord, lines []types.PlayerLine) RoundIdentity {
	if len(ri.occupations) == 0 || len(ri.byRound) != 1 {
		return ri
	}
	round := ri.Rounds()[0]
	bySlot := ri.occupations[round]
	if len(bySlot) == 0 {
		return ri
	}
	pris := ri.occupantsDeManche(round)
	for slot, xuid := range ri.byRound[round] {
		if _, recycle := bySlot[slot]; !recycle {
			pris[xuid] = true
		}
	}
	out := ri
	out.occupations = copieDesOccupations(ri.occupations)
	var series [3]map[int]map[int][]types.ScorePoint
	slots := make([]int, 0, len(bySlot))
	for slot := range bySlot {
		slots = append(slots, slot)
	}
	slices.Sort(slots)
	for _, slot := range slots {
		occ := out.occupations[round][slot]
		last := &occ[len(occ)-1]
		if last.XUID != "" {
			continue
		}
		for i, k := range coreKeys {
			if series[i] == nil {
				series[i] = rawSeriesByRound(recs, k, false, nil)
			}
		}
		xuid, ok := uniqueLineOf(occupationTriplet(series, round, slot, *last), lines)
		if !ok || pris[xuid] {
			continue
		}
		last.XUID, last.Origin = xuid, OriginSheetTriplet
		pris[xuid] = true
	}
	return out
}

// occupationTriplet rend (frags, morts, assistances) d'une occupation : la derniere valeur de chaque
// compteur sur le segment, sous les filtres de la serie publiee.
func occupationTriplet(series [3]map[int]map[int][]types.ScorePoint, round, slot int, o Occupation) [3]int64 {
	var out [3]int64
	for i := range coreKeys {
		var pts []types.ScorePoint
		for _, p := range series[i][slot][round] {
			if o.Covers(p.TimeMS) {
				pts = append(pts, p)
			}
		}
		if len(pts) == 0 {
			continue
		}
		parInstant(pts)
		kept := boundedSeries(longestRun(pts, false))
		out[i] = kept[len(kept)-1].Value
	}
	return out
}

// uniqueLineOf rend le xuid de l'UNIQUE ligne de la feuille qui porte ce triplet ; faux pour un
// triplet nul, sans ligne, ou porte par plusieurs lignes.
func uniqueLineOf(triplet [3]int64, lines []types.PlayerLine) (string, bool) {
	if triplet == [3]int64{} {
		return "", false
	}
	found, n := "", 0
	for _, l := range lines {
		if int64(l.Kills) == triplet[0] && int64(l.Deaths) == triplet[1] && int64(l.Assists) == triplet[2] {
			found, n = l.XUID, n+1
		}
	}
	return found, n == 1
}
