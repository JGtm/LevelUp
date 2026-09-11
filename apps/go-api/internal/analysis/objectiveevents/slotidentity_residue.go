package objectiveevents

// slotidentity_residue.go — L'IDENTITE D'UN SLOT D'ENTITE PAR RESIDU DE MANCHE.
//
// # LE TROU QUE CE FICHIER FERME (decouverte 1 du lot 6.2,
// `.ai/V7.5/RAPPORT_ODDBALL_FANTOMES_2026-09-10.md` §6)
//
// [RoundIdentity.CompletedByElimination] ne nomme que le cas d'UNICITE : exactement un slot
// muet, exactement un xuid libre. Des qu'une manche en laisse DEUX, elle s'abstient — et c'est
// le cas de `43716616` (Oddball, 2 manches), dont la manche 0 laisse QUATRE slots muets pour
// CINQ xuids libres. Le plus gros porteur du match (2533274978052136, 62,3 s a l'oracle API)
// tombait dans ce trou : son train de tics partait en `noBridge`, le calque le montrait a 0.
//
// [RoundIdentity.CompletedByLines] existait pour ce trou-la aussi, mais elle est gardee
// MONO-MANCHE : le triplet apparie des TOTAUX DE MATCH, et en multi-manche le slot est
// reattribue tandis que les compteurs repartent de zero a chaque manche.
//
// # LA REGLE, ET POURQUOI ELLE SE VERIFIE
//
// Ce que le triplet ne peut pas faire sur un total, le RESIDU le fait sur une manche : le
// total de la feuille MOINS la somme des manches deja nommees du meme xuid doit egaler le
// segment (frags, morts, assistances) du slot candidat DANS cette manche. C'est le meme
// controle que [residuConcorde], deja pose par l'elimination — ici il ne CONTROLE plus une
// deduction, il la PRODUIT.
//
// Mesure d'entree sur `43716616`, manche 0 (releve du 2026-09-10, quatre slots muets) :
//
//	slot 12 segment (3,1,3)   residu de 2535449672349 (3,1,3)   un seul candidat
//	slot 18 segment (4,2,2)   residu de 2533274966155425 (4,2,2)   un seul candidat
//	slot 20 segment (0,2,0)   residu de 2535434750421 (0,2,0)   un seul candidat
//	slot 22 segment (2,1,4)   residu de 2533274978052136 (2,1,4)   un seul candidat
//	                          residu de 2535457270266 (0,0,0)   aucun slot
//
// # LES QUATRE GARDES, AUCUNE NEGOCIABLE
//
//  1. MULTI-MANCHE SEULEMENT. Sur un film a une manche le residu VAUT le total : la voie du
//     triplet ([RoundIdentity.CompletedByLines]) y repond deja, avec sa propre marge. Cette
//     garde est ce qui rend le lot NEUTRE sur les films mono-manche du parc — 60 des 64.
//  2. UNICITE MUTUELLE. Un slot n'est nomme que si un SEUL xuid libre porte son residu ET que
//     ce xuid ne concorde qu'avec CE slot. Deux joueurs au meme bilan de manche ne sont pas
//     departageables : on se tait.
//  3. LE SEGMENT NUL NE PROUVE RIEN. Un slot a (0, 0, 0) et un joueur au residu (0, 0, 0)
//     s'apparieraient sur du vide — un joueur arrive en fin de match, un slot qui n'a rien
//     fait. C'est une devinette, pas un appariement force.
//  4. COMPLETER, JAMAIS CONTREDIRE, ET AUCUN XUID DEUX FOIS — les deux gardes des deux autres
//     voies.
//
// `lines` vide rend l'identite INCHANGEE : l'artefact reste publiable hors ligne, sans base.

// OriginRoundResidue : le residu de la feuille sur la manche, apparie a un seul slot.
const OriginRoundResidue = "residu_de_manche"

// CompletedByRoundResidue complete l'identite PAR MANCHE en appariant le segment d'un slot muet
// au RESIDU d'un xuid libre, quand l'appariement est unique des deux cotes.
func (ri RoundIdentity) CompletedByRoundResidue(recs []StatRecord, lines []PlayerLine) RoundIdentity {
	if len(lines) == 0 || len(ri.byRound) <= 1 {
		return ri
	}
	seg := segmentsParManche(recs)
	out := ri.copieProfonde()
	for _, round := range ri.Rounds() {
		for slot, xuid := range appariementsParResidu(seg, out.byRound, recs, lines, round) {
			out.byRound[round][slot] = xuid
			out.origins[round][slot] = OriginRoundResidue
		}
	}
	return out
}

// appariementsParResidu rend les couples (slot muet -> xuid libre) que le residu apparie de
// facon UNIQUE DES DEUX COTES dans une manche.
func appariementsParResidu(seg map[int]map[int]segmentKDA, byRound map[int]map[int]string,
	recs []StatRecord, lines []PlayerLine, round int) map[int]string {
	muets, libres := muetsEtLibresDeManche(byRound, recs, lines, round)
	if len(muets) == 0 || len(libres) == 0 {
		return nil
	}
	// candidats[slot] = les xuid libres dont le residu vaut le segment du slot.
	candidats := make(map[int][]string, len(muets))
	parXUID := make(map[string][]int, len(libres))
	for _, slot := range muets {
		s := seg[round][slot]
		if s == (segmentKDA{}) {
			continue // garde n° 3 : un segment nul ne prouve rien
		}
		for _, xuid := range libres {
			if residuDeManche(seg, byRound, lines, round, xuid) == s {
				candidats[slot] = append(candidats[slot], xuid)
				parXUID[xuid] = append(parXUID[xuid], slot)
			}
		}
	}
	out := map[int]string{}
	for slot, xuids := range candidats {
		if len(xuids) != 1 || len(parXUID[xuids[0]]) != 1 {
			continue // garde n° 2 : unicite mutuelle
		}
		out[slot] = xuids[0]
	}
	return out
}

// muetsEtLibresDeManche rend les slots emetteurs non nommes de la manche et les xuid de la
// feuille qu'aucun slot de cette manche ne porte.
func muetsEtLibresDeManche(byRound map[int]map[int]string, recs []StatRecord,
	lines []PlayerLine, round int) ([]int, []string) {
	nommes := byRound[round]
	var muets []int
	for _, slot := range EmittingPlayerSlots(recs, round) {
		if nommes[slot] == "" {
			muets = append(muets, slot)
		}
	}
	pris := make(map[string]bool, len(nommes))
	for _, x := range nommes {
		pris[x] = true
	}
	var libres []string
	for _, l := range lines {
		if !pris[l.XUID] {
			libres = append(libres, l.XUID)
		}
	}
	return muets, libres
}

// residuDeManche rend le total de la feuille d'un joueur MOINS la somme de ses segments dans
// les autres manches ou un slot porte deja son nom — ce qu'il lui reste a avoir fait ICI.
func residuDeManche(seg map[int]map[int]segmentKDA, byRound map[int]map[int]string,
	lines []PlayerLine, round int, xuid string) segmentKDA {
	var out segmentKDA
	for _, l := range lines {
		if l.XUID == xuid {
			out = segmentKDA{kills: l.Kills, deaths: l.Deaths, assists: l.Assists}
			break
		}
	}
	for r, m := range byRound {
		if r == round {
			continue
		}
		for s, x := range m {
			if x != xuid {
				continue
			}
			d := seg[r][s]
			out.kills -= d.kills
			out.deaths -= d.deaths
			out.assists -= d.assists
		}
	}
	return out
}
