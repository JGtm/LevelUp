package replay

// zone_states_hill.go — LE VOLET COLLINE : la zone ACTIVE quand le mode n'a aucun oracle nomme.
//
// DEUX VOIES, ET LA MESURE A DECIDE DE LEUR ORDRE (lot C-ter volet 1, 2026-08-19,
// `.ai/V7.5/replay2d/registre_film/LOTCTER_VOLET1.md`) :
//
//	le DESIGNATEUR   l'objet de mode KOTH tient sur quatre slots `ti=13` consecutifs —
//	                 [tag 5 designateur][tag 4 proprietaire][tag 4 capteur][tag 3 jauge] — et
//	                 le tag 5 y CHANGE de valeur 13 a 21 ms apres chaque capture non terminale
//	                 (13/13 changements sur 4 films, temoins decales 0 %, hasard 1,5-2,6 %) : il
//	                 DESIGNE la colline courante (vocabulaire ordinal identique sur 4 cartes).
//	                 Les periodes sont donc FERMEES A LA BASCULE, et la grappe des positions ne
//	                 sert plus qu'a apparier chaque periode a une forme. La colline VIDE (avant
//	                 que quelqu'un n'y entre) devient visible.
//	les RAMPES       l'ancienne lecture (lot C-bis phase 2a) : chaque montee de la jauge est une
//	                 session de garde, la zone se lit dans la grappe pendant la montee, les
//	                 voisines qui designent la meme zone se fondent. Elle reste le REPLI d'un
//	                 film sans designateur lisible (aucun des 4 films du corpus n'y retombe).
//
// CE QUE LE FILM NE DIT PAS : l'ACTIVATION DE LA PREMIERE COLLINE. La premiere designation vit
// dans l'image-cle, que le delta ne re-emet pas ; l'objet de mode est ABSENT des images-cles a 0
// et 20 s et PRESENT a 40 s sur les 4 films (cree entre les deux). La premiere periode s'ouvre
// donc au PREMIER CONTACT avec l'objet (premiere emission de sa jauge, de son proprietaire ou de
// son designateur) — une borne HAUTE de l'activation, jamais une invention.
//
// CE QUE CE VOLET NE PUBLIE PAS : le PROPRIETAIRE. Le tag 4 du slot voisin est un canal de
// propriete au sens de la phase 2a, mais il n'a pas ete confronte au roster sur les KOTH — on ne
// publie pas ce qu'on n'a pas mesure.

import "sort"

// hillDesignatorMinOwnerSamples : un slot de tag 5 n'est un designateur que si le slot SUIVANT
// porte un canal de proprietaire qui parle (au moins deux emissions) — la structure de l'objet
// de mode. Sans cette condition, le trio de fin de match (trois string-ids sur trois slots
// consecutifs, emis 15-21 ms apres la capture terminale sur les 4 films) serait eligible.
const hillDesignatorMinOwnerSamples = 2

// hillDesignatorSpan : la portee, en slots, dans laquelle on cherche les canaux voisins de
// l'objet de mode (proprietaire, capteur, jauge) pour dater le premier contact.
const hillDesignatorSpan = 3

// buildHillStates rend les periodes de garde de la zone ACTIVE, appariees par la grappe.
func buildHillStates(zones []Zone, ser zoneSeries, c zoneCtx,
	cov *ZonesCoverage,
) []ZoneState {
	if d, ok := hillDesignatorOf(ser); ok {
		return buildDesignatedHills(zones, ser, d, c, cov)
	}
	return buildRampHills(zones, ser, c, cov)
}

// hillDesignator est le slot qui DESIGNE la colline courante, et ses bascules.
type hillDesignator struct {
	slot uint32
	// changes : frames ou la designation change — chaque changement de valeur, la premiere
	// emission comprise (l'etat initial vit dans l'image-cle).
	changes []int
	// first : frame du premier contact avec l'objet de mode (borne haute de l'activation).
	first int
}

// hillDesignatorOf elit le designateur : parmi les slots a serie de tag 5 CHAINEE dont le slot
// suivant porte un proprietaire qui parle, celui qui bascule le plus (egalite : le plus petit
// slot). Faux quand aucun slot ne remplit la condition — le repli par les rampes prend la main.
func hillDesignatorOf(ser zoneSeries) (hillDesignator, bool) {
	var best hillDesignator
	found := false
	for _, slot := range sortedZoneSlots(ser.desig) {
		if len(ser.owner[slot+1]) < hillDesignatorMinOwnerSamples {
			continue
		}
		var changes []int
		for i, s := range ser.desig[slot] {
			if i == 0 || s.v != ser.desig[slot][i-1].v {
				changes = append(changes, s.t)
			}
		}
		if len(changes) == 0 || (found && len(changes) <= len(best.changes)) {
			continue
		}
		best, found = hillDesignator{slot: slot, changes: changes}, true
	}
	if !found {
		return best, false
	}
	best.first = hillFirstContact(ser, best)
	return best, true
}

// hillFirstContact rend la frame de la premiere emission de l'objet de mode : son designateur,
// ou les canaux des slots voisins (proprietaire, capteur, jauge).
func hillFirstContact(ser zoneSeries, d hillDesignator) int {
	first := d.changes[0]
	for k := uint32(1); k <= hillDesignatorSpan; k++ {
		for _, ss := range [][]zoneSample{ser.owner[d.slot+k], ser.gauge[d.slot+k]} {
			if len(ss) > 0 && ss[0].t < first {
				first = ss[0].t
			}
		}
	}
	return first
}

// buildDesignatedHills decoupe le match en periodes bornees par le designateur, apparie chaque
// periode par la grappe des positions pendant les montees de la jauge qu'elle contient (a defaut,
// pendant toute la periode), et publie les periodes localisees.
func buildDesignatedHills(zones []Zone, ser zoneSeries, d hillDesignator, c zoneCtx,
	cov *ZonesCoverage,
) []ZoneState {
	cov.Method = ZoneMethodDesignator
	ramps := zoneRampsOf(ser)
	pts := zonePointsByFrame(c.tracks)
	periods := hillDesignatedPeriods(d, c.frames)
	cov.HillPeriods = len(periods)
	kept := make([]hillPeriod, 0, len(periods))
	for _, p := range periods {
		votes := hillVotesInRamps(zones, pts, ramps, &p)
		if len(votes) == 0 {
			votes = hillVotes(zones, pts, p.t0, p.t1)
		}
		p.ref, p.hasRef = clearModalZone(votes)
		if !p.hasRef {
			// UNE COLLINE DESIGNEE QUE LA GRAPPE NE LOCALISE PAS EST ECARTEE ET SE COMPTE :
			// elle a existe, on ne sait pas ou (cf. ZonesCoverage.Unpaired).
			cov.Unpaired++
			continue
		}
		kept = append(kept, p)
	}
	states := hillStatesOf(kept, zoneGaugeScales(ser))
	cov.Paired = len(states)
	tallyZoneSpans(states, cov)
	return states
}

// hillDesignatedPeriods rend une periode par colline : [premier contact ; b1-1], [b1 ; b2-1],
// ..., [bn ; derniere frame]. Une periode vide (deux bascules dans la meme frame) est ecartee.
func hillDesignatedPeriods(d hillDesignator, frames int) []hillPeriod {
	bounds := append([]int{d.first}, d.changes...)
	out := make([]hillPeriod, 0, len(bounds))
	for i, t0 := range bounds {
		t1 := frames - 1
		if i+1 < len(bounds) {
			t1 = bounds[i+1] - 1
		}
		if t1 < t0 {
			continue
		}
		out = append(out, hillPeriod{t0: t0, t1: t1})
	}
	return out
}

// hillVotesInRamps compte les positions par zone pendant les montees de la jauge qui tombent
// dans la periode, et retient le sommet de jauge le plus haut (la progression publiee). Rend nil
// quand aucune montee ne tombe dans la periode.
func hillVotesInRamps(zones []Zone, pts map[int][]Point, ramps []zoneRamp, p *hillPeriod) map[int]int {
	var votes map[int]int
	for _, r := range ramps {
		if r.tPeak < p.t0 || r.t0 > p.t1 {
			continue
		}
		if votes == nil {
			votes = map[int]int{}
		}
		for ref, n := range hillVotes(zones, pts, max(r.t0, p.t0), min(r.tPeak, p.t1)) {
			votes[ref] += n
		}
		if !p.hasTop || r.top > p.top {
			p.top, p.gaugeSlot, p.hasTop = r.top, r.slot, true
		}
	}
	return votes
}

// hillPeriod est un intervalle pendant lequel une colline est gardee.
type hillPeriod struct {
	t0, t1 int
	ref    int
	hasRef bool
	// top est le sommet de la jauge atteint pendant la periode — la progression publiee — et
	// gaugeSlot le slot de jauge qui l'a atteint (son echelle).
	top       uint64
	hasTop    bool
	gaugeSlot uint32
}

// buildRampHills — LE REPLI : les periodes par RAMPE de la jauge (lot C-bis phase 2a).
//
// LA PERIODE EST UNE RAMPE, ET C'EST UNE MESURE QUI L'A DECIDE. La premiere definition essayee
// segmentait par SLOT ACTIF (une colline = un slot, comme une zone de Bastion = un slot) : elle
// a ete REFUTEE — un seul slot porte la jauge de tout un match KOTH, cette lecture rend UNE
// periode et n'apparie rien. Chaque montee est donc une session de garde, et les periodes
// voisines qui designent la meme zone se fondent.
//
// CE QUE CE REPLI VAUT, ET IL FAUT LE DIRE : la couverture temporelle est une clause FAIBLE (des
// que des rampes sont reparties sur le match, les periodes etendues couvrent presque tout). Ce
// qui porte le resultat est la NETTETE de chaque attribution, tres inegale d'un film a l'autre.
func buildRampHills(zones []Zone, ser zoneSeries, c zoneCtx, cov *ZonesCoverage) []ZoneState {
	cov.Method = ZoneMethodPositions
	ramps := zoneRampsOf(ser)
	sort.SliceStable(ramps, func(i, j int) bool { return ramps[i].t0 < ramps[j].t0 })
	if len(ramps) == 0 {
		return nil
	}
	pts := zonePointsByFrame(c.tracks)
	raw := make([]hillPeriod, 0, len(ramps))
	for _, r := range ramps {
		p := hillPeriod{t0: r.t0, t1: r.tPeak, top: r.top, gaugeSlot: r.slot, hasTop: true}
		p.ref, p.hasRef = clearModalZone(hillVotes(zones, pts, r.t0, r.tPeak))
		if !p.hasRef {
			// LA RAMPE QUE LA GRAPPE NE LOCALISE PAS EST ECARTEE, ET ELLE SE COMPTE (revue R1,
			// 2026-08-18) : une montee de jauge que personne n'entoure est une garde REELLE
			// dont on ne sait pas ou elle a lieu. La taire faisait passer un appariement
			// partiel pour un appariement complet — `unpaired` restait a zero quoi qu'il
			// arrive (cf. ZonesCoverage.Unpaired, semantique propre a cette methode).
			cov.Unpaired++
		}
		raw = append(raw, p)
	}
	periods := mergeHillPeriods(raw, c.frames)
	cov.HillPeriods = len(periods)
	states := hillStatesOf(periods, zoneGaugeScales(ser))
	cov.Paired = len(states)
	tallyZoneSpans(states, cov)
	return states
}

// zonePointsByFrame indexe toutes les positions publiees par frame : la grappe se lit par
// tranche de temps, pas par joueur.
func zonePointsByFrame(tracks []Track) map[int][]Point {
	out := map[int][]Point{}
	for _, tr := range tracks {
		for _, p := range tr.Points {
			out[p.T] = append(out[p.T], p)
		}
	}
	return out
}

// hillVotes compte, par zone, les positions qui tombent DANS une zone (et une seule) pendant
// [t0, t1].
func hillVotes(zones []Zone, pts map[int][]Point, t0, t1 int) map[int]int {
	votes := map[int]int{}
	for f := t0; f <= t1; f++ {
		for _, pt := range pts[f] {
			best, hits := nearestZones(zones, pt)
			if len(hits) != 1 || best > zoneCaptureDistanceM {
				continue
			}
			votes[hits[0].SpatialRank]++
		}
	}
	return votes
}

// clearModalZone rend la zone la plus votee, et si elle ressort.
//
// LA ZONE GAGNANTE DOIT DEVANCER LA DEUXIEME, sinon la periode reste NON APPARIEE : une grappe
// partagee entre deux zones ne designe pas une colline, elle dit que le trafic passe par les
// deux. Publier la premiere par defaut poserait une colline sur un couloir.
func clearModalZone(votes map[int]int) (int, bool) {
	ref, n := modalZone(votes)
	if n == 0 {
		return 0, false
	}
	second := 0
	for r, v := range votes {
		if r != ref && v > second {
			second = v
		}
	}
	return ref, n > second
}

// mergeHillPeriods fond les periodes voisines qui designent la MEME zone et etend chacune
// jusqu'au debut de la suivante : entre deux gardes de la meme colline, la colline n'a pas
// change. Les periodes non appariees sont ECARTEES — elles n'ont pas de zone ou se poser.
//
// UNE SEULE COLLINE EST ACTIVE A UN INSTANT, ET C'EST GARANTI PAR CONSTRUCTION : chaque garde
// FERME la precedente (cf. closeHillTail), au lieu de ne la fermer que lorsqu'un trou les
// separait.
func mergeHillPeriods(ps []hillPeriod, frames int) []hillPeriod {
	var out []hillPeriod
	for _, p := range ps {
		if !p.hasRef {
			continue
		}
		out = closeHillTail(out, p.t0)
		if n := len(out); n > 0 && out[n-1].ref == p.ref {
			out[n-1].t1 = max(out[n-1].t1, p.t1)
			out[n-1].top = max(out[n-1].top, p.top)
			continue
		}
		out = append(out, p)
	}
	if n := len(out); n > 0 && out[n-1].t1 < frames-1 {
		out[n-1].t1 = frames - 1
	}
	return out
}

// closeHillTail ferme les periodes deja retenues a `t0 - 1` : a l'instant ou une garde commence,
// la precedente s'arrete, QUEL QUE SOIT SON SLOT.
//
// C'EST LA GARANTIE « UNE SEULE COLLINE ACTIVE » (revue R1, 2026-08-18). L'ancienne ecriture ne
// fermait la periode precedente que si un TROU la separait de la suivante ; deux rampes de slots
// DIFFERENTS qui se RECOUVRENT laissaient donc deux zones marquees `active` au meme instant — ce
// que le mode ne permet pas, et ce que la phase 2a avait justement mesure (une seule jauge monte
// a la fois, 100,0 % du temps sur 60 rampes du film de reference).
//
// UNE PERIODE ENTIEREMENT RECOUVERTE DISPARAIT : fermee avant son propre debut, elle n'a plus
// d'instant a elle. La boucle remonte alors sur celle d'avant, qui redevient la derniere — et
// qui peut a son tour se fondre avec la garde entrante si elle designe la meme zone.
func closeHillTail(out []hillPeriod, t0 int) []hillPeriod {
	for len(out) > 0 {
		last := len(out) - 1
		out[last].t1 = t0 - 1
		if out[last].t1 >= out[last].t0 {
			break
		}
		out = out[:last]
	}
	return out
}

// hillStatesOf regroupe les periodes par zone et rend les intervalles ACTIFS.
//
// AUCUN PROPRIETAIRE N'EST PUBLIE ICI, et c'est la limite du mode : le canal de propriete ne
// parle que la ou il y a des captures nommees. Une colline gardee ne dit pas, dans le film, PAR
// QUI — l'affirmer d'apres la grappe serait une deduction, pas une lecture.
func hillStatesOf(periods []hillPeriod, scales map[uint32]zoneGauge) []ZoneState {
	byRef := map[int][]ZoneSpan{}
	for _, p := range periods {
		if p.t1 < p.t0 {
			continue
		}
		var prog *float32
		if p.hasTop {
			prog = scales[p.gaugeSlot].progressOf(p.top)
		}
		byRef[p.ref] = append(byRef[p.ref], ZoneSpan{T0: p.t0, T1: p.t1, Active: true, Progress: prog})
	}
	refs := make([]int, 0, len(byRef))
	for ref := range byRef {
		refs = append(refs, ref)
	}
	sort.Ints(refs)
	out := make([]ZoneState, 0, len(refs))
	for _, ref := range refs {
		spans := byRef[ref]
		sort.SliceStable(spans, func(i, j int) bool { return spans[i].T0 < spans[j].T0 })
		out = append(out, ZoneState{ZoneRef: ref, Spans: spans})
	}
	return out
}

// zoneGaugeScales releve l'excursion de chaque slot de jauge : c'est l'echelle de la
// progression publiee (cf. zoneGaugeOf).
func zoneGaugeScales(ser zoneSeries) map[uint32]zoneGauge {
	out := make(map[uint32]zoneGauge, len(ser.gauge))
	for slot, ss := range ser.gauge {
		out[slot] = zoneGaugeOf(ss)
	}
	return out
}
