package objectives

import (
	"cmp"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// slotidentity_occupations.go — LE SIEGE RECYCLE : un slot d'entite statborg qui CHANGE D'OCCUPANT
// au cours d'une manche (lot R1 du plan de suite d audit, constat C1 du rapport
// `.ai/V7.5/film_re/G_CORPUS_J11_2026-09-28.md`).
//
// # LE DEFAUT, MESURE SUR `bcb6d393` (CTF, une manche)
//
// Un joueur qui quitte la partie libere son siege, et le remplacant le reprend : le MEME slot
// statborg porte alors deux joueurs l'un apres l'autre. Le film le dit sans ambiguite — les trois
// compteurs de base du slot (frags, morts, assistances) RETOMBENT A ZERO au changement d'occupant :
//
//	slot 12  70 706 k1 d0 ... 179 114 k2 d5 | 191 979 k0 d0 | 231 170 k0 d1 ... 342 765 k1 d5
//	         2535460750735339 (K2 D5 A0)                      2535468064146356 (K1 D5 A0)
//	slot 14  48 600 d1 ... 159 730 k4 d3 | 170 639 0/0/0 | 223 679 d1 | 232 705 0/0 | 274 479 k1 ...
//	         2533274876732804 (K4 D3 A1)                    2535418713587213         2533274811363842
//
// Chaque segment reproduit la ligne de match de son occupant, et les morts de chaque segment
// coincident avec le fil des morts de cet occupant, et de lui seul. Le pont par instants de mort,
// lui, deroulait la serie PUBLIEE du slot (plus longue sous-suite non decroissante, lot J8.5) : elle
// garde UN des segments — celui du remplacant sur le slot 12 —, et le lien resolu valait pour TOUTE
// la manche. L'action `kills` de 70 706 ms (le frag du premier occupant, mort 16 ms plus tard) etait
// publiee au nom du remplacant, entre dans la partie deux minutes et demie apres.
//
// # LA REGLE : UN LIEN PAR OCCUPATION PROUVEE, ET RIEN SUR CE QUI NE L'EST PAS
//
// Les bornes d'une occupation sont LUES dans le film : une emission dont tous les compteurs de base
// presents valent zero alors que l'un d'eux valait plus (la RETOMBEE), confirmee par la reprise du
// comptage d'un nouvel occupant (la premiere emission qui compte ensuite ne prolonge pas l'ancien
// comptage). Une retombee sans reprise (fin de match, depart sans remplacant) ne coupe rien. Chaque
// occupation est ensuite nommee par la MEME regle que le pont plat ([bestDeathClaim] : trois
// coincidences au moins, marge sur le suivant), sur la serie de SON segment, filtree comme la serie
// publiee (manche confrontee au temps, plus longue sous-suite non decroissante, borne par pas). Une
// occupation que rien ne nomme reste VIDE : le pont s'abstient sur son intervalle, et l'abstention se
// compte : `coverage.objectives.noSlot` pour les actions d'objectif, une ligne `non_resolu` bornee
// dans `identity.statborgSlots`, et le journal de la cuisson (`siegeNonProuve`, toutes familles).
// Ce n'est pas un repli (D14 (b)) : rien ne decide a la place de la lecture, le pont se tait.
//
// # POURQUOI LES FAITS DU MATCH NE BORNENT PAS LE LIEN
//
// La feuille porte `first_joined_time` / `last_leave_time`, mais pas sur l'horloge du film : leur
// origine est le debut du match au registre, et l'ecart avec le film varie d'un match a l'autre
// (mesure du 2026-09-28 sur les 19 temoins du gate : sur `a521164d` les joueurs du coup d'envoi
// « arrivent » a 82,9 s quand la premiere mort du film tombe a 71,0 s ; sur `bcb6d393` le depart
// declare du premier occupant du slot 12 est a 238,6 s, la retombee du siege dans le film a
// 192,0 s, et six joueurs arrives en cours meurent avant leur arrivee declaree, jusqu'a 24,5 s
// avant). Borner par la feuille abstiendrait des actions justes et en laisserait passer de fausses ;
// le film, lui, date le changement de siege a l'emission pres.
//
// # NEUTRALITE : UN SLOT SANS RETOMBEE NE CHANGE PAS
//
// Seuls les couples (manche, slot) qui portent au moins une retombee confirmee recoivent des
// occupations. Tous les autres gardent le lien de la manche entiere, octet pour octet — et
// [RoundIdentity.AtRound], qui repond pour une manche sans instant, reste inchange partout.

// Occupation est l'occupant d'un slot de joueur sur un INTERVALLE d'une manche, sur l'horloge des
// enregistrements : [FromMS, ToMS[. OpenFrom / OpenTo : l'intervalle n'a pas de borne de ce cote
// (premier ou dernier occupant de la manche). XUID vide : l'occupation n'a pas ete nommee — le pont
// s'abstient sur son intervalle.
type Occupation struct {
	FromMS, ToMS     int
	OpenFrom, OpenTo bool
	XUID             string
	// Origin est la voie qui a nomme l'occupant ([OriginDeathInstants], [OriginSheetTriplet]),
	// vide quand l'occupation n'est pas nommee.
	Origin string
}

// Covers dit que l'instant tombe dans l'occupation.
func (o Occupation) Covers(timeMS int) bool {
	return (o.OpenFrom || timeMS >= o.FromMS) && (o.OpenTo || timeMS < o.ToMS)
}

// coreKeys : les trois compteurs de base, dans l'ordre (frags, morts, assistances).
var coreKeys = [3]statSlotKey{{coreKillsComp, sideA}, {coreKillsComp, sideB}, {coreAssistsComp, sideA}}

// coreEmission : les compteurs de base PRESENTS dans une emission d'un slot de joueur.
type coreEmission struct {
	timeMS int
	vals   [3]int64
	has    [3]bool
}

// coreEmissionsBySeat rend, par manche REELLE et par slot de joueur, les emissions des compteurs de
// base en ordre chronologique — sous les MEMES filtres que les series publiees : manche confrontee
// au temps ([RoundBounds.Excludes]), manches fantomes ecartees ([RealRounds]), emission hors domaine
// jetee ([emissionHorsDomaine]).
func coreEmissionsBySeat(recs []types.StatRecord) map[int]map[int][]coreEmission {
	bornes := ResolveRoundBounds(recs)
	real := RealRounds(recs)
	out := map[int]map[int][]coreEmission{}
	for _, r := range recs {
		if IsTeamSlot(r.Slot) || !real[r.Round] || bornes.Excludes(r) {
			continue
		}
		e, ok := coreEmissionOf(r)
		if !ok {
			continue
		}
		if out[r.Round] == nil {
			out[r.Round] = map[int][]coreEmission{}
		}
		out[r.Round][r.Slot] = append(out[r.Round][r.Slot], e)
	}
	for _, bySlot := range out {
		for _, ems := range bySlot {
			// STABLE sur l ordre du film (DT-9) : a instant egal, le rang de balayage departage.
			slices.SortStableFunc(ems, func(a, b coreEmission) int { return cmp.Compare(a.timeMS, b.timeMS) })
		}
	}
	return out
}

// coreEmissionOf lit les compteurs de base d'un enregistrement ; faux quand il n'en porte aucun.
func coreEmissionOf(r types.StatRecord) (coreEmission, bool) {
	e := coreEmission{timeMS: r.TimeMS}
	lu := false
	for i, k := range coreKeys {
		v, ok := r.Comps[k.Comp]
		if !ok {
			continue
		}
		val := v.A
		if k.Side == sideB {
			val = v.B
		}
		if emissionHorsDomaine(k, v, val) {
			continue
		}
		e.vals[i], e.has[i], lu = val, true, true
	}
	return e, lu
}

// seatChanges rend les instants ou le siege CHANGE D'OCCUPANT : une retombee (cf. l'en-tete)
// confirmee par la premiere emission qui compte ensuite. Une retombee que rien ne suit, ou que
// l'emission suivante prolonge (tous les compteurs presents au moins egaux a ceux d'avant la
// retombee, dont un non nul), ne coupe rien.
func seatChanges(ems []coreEmission) []int {
	var out []int
	var last, avant [3]int64
	enAttente, instant := false, 0
	for _, e := range ems {
		if retombe(e, last) {
			if !enAttente {
				enAttente, instant, avant = true, e.timeMS, last
			}
			last = [3]int64{}
			continue
		}
		if enAttente && compte(e) {
			if prolonge(e, avant) {
				last = avant
			} else {
				out = append(out, instant)
			}
			enAttente = false
		}
		for i := range e.vals {
			if e.has[i] {
				last[i] = e.vals[i]
			}
		}
	}
	return out
}

// retombe : tous les compteurs presents valent zero, et l'un d'eux valait plus.
func retombe(e coreEmission, last [3]int64) bool {
	chute := false
	for i := range e.vals {
		if !e.has[i] {
			continue
		}
		if e.vals[i] != 0 {
			return false
		}
		if last[i] > 0 {
			chute = true
		}
	}
	return chute
}

// compte : l'emission porte au moins un compteur non nul.
func compte(e coreEmission) bool {
	for i := range e.vals {
		if e.has[i] && e.vals[i] > 0 {
			return true
		}
	}
	return false
}

// prolonge : l'emission continue le comptage d'avant la retombee — chaque compteur present au moins
// egal a sa valeur d'avant, et l'un d'eux reprenant une valeur non nulle.
func prolonge(e coreEmission, avant [3]int64) bool {
	reprend := false
	for i := range e.vals {
		if !e.has[i] {
			continue
		}
		if e.vals[i] < avant[i] {
			return false
		}
		if avant[i] > 0 {
			reprend = true
		}
	}
	return reprend
}

// occupationsBetween decoupe une manche aux changements de siege : n changements, n+1 occupations.
func occupationsBetween(changes []int) []Occupation {
	out := make([]Occupation, 0, len(changes)+1)
	prev, open := 0, true
	for _, c := range changes {
		out = append(out, Occupation{FromMS: prev, OpenFrom: open, ToMS: c})
		prev, open = c, false
	}
	return append(out, Occupation{FromMS: prev, OpenFrom: open, OpenTo: true})
}

// seatOccupations rend, par manche et par slot RECYCLE, les occupations nommees par les instants de
// mort de leur propre segment. `whole` est l'identite par manche deja resolue : un occupant qu'un
// AUTRE slot porte deja dans la manche n'est pas nomme (aucun xuid deux fois, meme regle que
// [withoutContestedXUID]). Les slots sans changement de siege n'y figurent pas.
func seatOccupations(recs []types.StatRecord, thread map[string][]int,
	whole map[int]map[int]string) map[int]map[int][]Occupation {
	var deaths map[int]map[int][]types.ScorePoint // deroule a la demande : la plupart des films n'ont aucun siege recycle
	out := map[int]map[int][]Occupation{}
	for round, bySlot := range coreEmissionsBySeat(recs) {
		for slot, ems := range bySlot {
			changes := seatChanges(ems)
			if len(changes) == 0 {
				continue
			}
			if deaths == nil {
				deaths = rawSeriesByRound(recs, DeathsComponent.key(), false, nil)
			}
			occ := occupationsBetween(changes)
			for i := range occ {
				if xuid, ok := bestDeathClaim(occupationDeaths(deaths[slot][round], occ[i]), thread); ok {
					occ[i].XUID, occ[i].Origin = xuid, OriginDeathInstants
				}
			}
			if out[round] == nil {
				out[round] = map[int][]Occupation{}
			}
			out[round][slot] = occ
		}
	}
	for round, bySlot := range out {
		withoutContestedOccupants(bySlot, whole[round])
	}
	return out
}

// occupationDeaths deroule le compteur de morts d'UNE occupation en un instant par mort, sous les
// filtres de la serie publiee (plus longue sous-suite non decroissante, borne par pas) appliques au
// seul segment : le compteur y repart de zero, comme celui du jeu.
func occupationDeaths(raw []types.ScorePoint, o Occupation) []int {
	var pts []types.ScorePoint
	for _, p := range raw {
		if o.Covers(p.TimeMS) {
			pts = append(pts, p)
		}
	}
	if len(pts) == 0 {
		return nil
	}
	parInstant(pts)
	kept := boundedSeries(longestRun(pts, DeathsComponent.Strict))
	return instantsDesMorts(map[int][]types.ScorePoint{0: kept})[0]
}

// withoutContestedOccupants vide les occupations dont le xuid est porte par un AUTRE slot de la
// manche — par son lien de manche entiere (slots non recycles) ou par une de ses occupations. Deux
// occupations du MEME slot au meme xuid (un joueur qui retrouve son siege) ne se contestent pas.
func withoutContestedOccupants(bySlot map[int][]Occupation, whole map[int]string) {
	slotsOf := map[string]map[int]bool{}
	marquer := func(xuid string, slot int) {
		if xuid == "" {
			return
		}
		if slotsOf[xuid] == nil {
			slotsOf[xuid] = map[int]bool{}
		}
		slotsOf[xuid][slot] = true
	}
	for slot, xuid := range whole {
		if _, recycle := bySlot[slot]; !recycle {
			marquer(xuid, slot)
		}
	}
	for slot, occ := range bySlot {
		for _, o := range occ {
			marquer(o.XUID, slot)
		}
	}
	for _, occ := range bySlot {
		for i := range occ {
			if len(slotsOf[occ[i].XUID]) > 1 {
				occ[i].XUID, occ[i].Origin = "", ""
			}
		}
	}
}

// occupantAt rend l'occupant de l'occupation qui couvre l'instant — vide si aucune ne le nomme.
func occupantAt(occ []Occupation, timeMS int) string {
	for _, o := range occ {
		if o.Covers(timeMS) {
			return o.XUID
		}
	}
	return ""
}

// Occupations rend les occupations d'un slot RECYCLE dans une manche (une copie), nil pour un slot
// dont le siege n'a pas change d'occupant : son lien vaut alors pour toute la manche
// ([RoundIdentity.AtRound]).
func (ri RoundIdentity) Occupations(round, slot int) []Occupation {
	occ := ri.occupations[round][slot]
	if occ == nil {
		return nil
	}
	return append([]Occupation(nil), occ...)
}

// UnprovenOccupantAt dit que l'instant tombe sur un siege RECYCLE dans une occupation que rien n'a
// nommee : le pont s'y abstient. C'est le compte de l'abstention, distinct d'un slot jamais nomme.
func (ri RoundIdentity) UnprovenOccupantAt(slot, timeMS int) bool {
	round, ok := ri.roundOfInstant(timeMS)
	if !ok {
		return false
	}
	occ := ri.occupations[round][slot]
	return occ != nil && occupantAt(occ, timeMS) == ""
}

// occupantsDeManche rend les xuid que les occupations d'une manche nomment.
func (ri RoundIdentity) occupantsDeManche(round int) map[string]bool {
	out := map[string]bool{}
	for _, occ := range ri.occupations[round] {
		for _, o := range occ {
			if o.XUID != "" {
				out[o.XUID] = true
			}
		}
	}
	return out
}

// copieDesOccupations duplique les occupations : une completion ne modifie jamais l'identite que son
// appelant lui a passee.
func copieDesOccupations(src map[int]map[int][]Occupation) map[int]map[int][]Occupation {
	if src == nil {
		return nil
	}
	out := make(map[int]map[int][]Occupation, len(src))
	for round, bySlot := range src {
		m := make(map[int][]Occupation, len(bySlot))
		for slot, occ := range bySlot {
			m[slot] = append([]Occupation(nil), occ...)
		}
		out[round] = m
	}
	return out
}

// parInstant ordonne des points par instant, STABLE sur l ordre du film (DT-9) : a instant egal, le
// rang de balayage departage, comme dans la serie publiee.
func parInstant(pts []types.ScorePoint) {
	slices.SortStableFunc(pts, func(a, b types.ScorePoint) int { return cmp.Compare(a.TimeMS, b.TimeMS) })
}
