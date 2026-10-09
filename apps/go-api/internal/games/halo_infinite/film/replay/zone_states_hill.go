package replay

// zone_states_hill.go — LE VOLET COLLINE : la zone ACTIVE quand le mode n'a aucun oracle nomme.
//
// DEUX VOIES, ET LA MESURE A DECIDE DE LEUR ORDRE (lot C-ter volet 1, 2026-08-19,
// `.ai/V7.5/replay2d/registre_film/LOTCTER_VOLET1.md`) :
//
//	le DESIGNATEUR   l objet de mode KOTH est un BLOC de proprietes `ti=13` nommees —
//	                 [tag 5 designateur][tag 4 proprietaire][tag 4 capteur][tag 3 jauge], le
//	                 designateur etant la cle du bloc (zone_states_owner_nom.go, hillDesignatorOf) — et
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
// CE QUE CE VOLET PUBLIE DEPUIS LE 2026-08-26 : le PROPRIETAIRE. Son canal est celui du bloc dont le
// designateur est la cle de nommage, designe par le NOM de propriete (hillOwnerSlotOf ; le slot
// voisin du designateur en repli, et c est le meme canal sur les films a colline du parc). Ce canal
// a ete confronte a trois oracles successifs ; deux se sont reveles inutilisables, le
// troisieme donne 88-89 % d'accord contre un temoin a 56 %. Sous le seuil de 90 % que le plan
// s'etait fixe — et publie quand meme, par DECISION UTILISATEUR datee. Le verdict complet, les
// trois campagnes et la reserve (l'erreur est concentree aux bascules) vivent en tete de
// `hillStatesOf` : c'est la qu'il faut lire avant de toucher a ce canal.
//
// CE QUE CE VOLET NE PUBLIE TOUJOURS PAS : un proprietaire sur le repli par les RAMPES. Sans
// designateur il n'y a pas d'objet de mode, donc pas de slot voisin ou lire le camp.

import (
	"cmp"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
)

// hillDesignatorMinOwnerSamples : SUR LA REGLE DE VOISINAGE (repli, cf. hillDesignatorOf), un slot de
// tag 5 n est un designateur que si le slot SUIVANT porte un canal de proprietaire qui parle (au
// moins deux emissions) — la structure de l objet de mode. Sans cette condition, le trio de fin
// de match (trois string-ids sur trois slots consecutifs, emis 15-21 ms apres la capture
// terminale) serait eligible.
const hillDesignatorMinOwnerSamples = 2

// hillDesignatorSpan : SUR LA REGLE DE VOISINAGE, la portee, en slots, dans laquelle on cherche les
// canaux voisins de l objet de mode (proprietaire, capteur, jauge) pour dater le premier contact.
const hillDesignatorSpan = 3

// buildHillStates rend les periodes de garde de la zone ACTIVE, appariees par la grappe.
//
// `teams` est l'ensemble des index d'equipe admis (roster). Il ne sert QUE sur la voie du
// designateur, qui seule publie un proprietaire : le repli par les rampes n'a pas d'objet de
// mode, donc pas de slot voisin ou lire le camp.
func buildHillStates(zones []Zone, ser zoneSeries, teams map[uint64]bool, c zoneCtx,
	cov *ZonesCoverage,
) []ZoneState {
	if d, ok := hillDesignatorOf(ser); ok {
		if d.parVoisinage {
			c.fb.Declenche(fallback.NomCollineDesignateurParVoisinage)
		}
		return buildDesignatedHills(zones, ser, hillCtx{d: d, teams: teams}, c, cov)
	}
	return buildRampHills(zones, ser, c, cov)
}

// hillCtx regroupe ce que la voie du designateur ajoute a la construction : le designateur elu
// et le referentiel d'equipes (regle des 5 parametres).
type hillCtx struct {
	d     hillDesignator
	teams map[uint64]bool
}

// hillDesignator est le slot qui DESIGNE la colline courante, et ses bascules.
type hillDesignator struct {
	slot uint32
	// changes : frames ou la designation change — chaque changement de valeur, la premiere
	// emission comprise (l'etat initial vit dans l'image-cle).
	changes []int
	// first : frame du premier contact avec l'objet de mode (borne haute de l'activation).
	first int
	// parVoisinage dit que le designateur et les canaux de l objet de mode viennent de la regle
	// de VOISINAGE (repli `repli_colline_designateur_par_voisinage`), faute de nom au vocabulaire.
	parVoisinage bool
}

// hillDesignatorOf elit le designateur : PAR LE NOM d abord — parmi les slots a serie de tag 5
// CHAINEE, ceux dont le nom est la CLE d un bloc du vocabulaire (zone_states_owner_nom.go) —, PAR
// LE VOISINAGE en repli — ceux dont le slot suivant porte un proprietaire qui parle. Dans chaque
// voie, celui qui bascule le plus (egalite : le plus petit slot). Faux quand aucune voie ne rend
// de designateur — le repli par les rampes prend la main.
//
// LE NOM EXCLUT DE LUI-MEME LE TRIO DE FIN DE MATCH (trois string-ids sur trois slots
// consecutifs, emis 15-21 ms apres la capture terminale) : ses noms ne sont la cle d aucun bloc.
// La regle de voisinage n avait que la condition sur le slot suivant pour l ecarter.
func hillDesignatorOf(ser zoneSeries) (hillDesignator, bool) {
	nomme := func(slot uint32) bool {
		_, ok := zoneBlocDuSlot(slot, ser.noms, func(b zoneBlocNomme) uint32 { return b.cle })
		return ok
	}
	if d, ok := hillDesignatorWhere(ser, nomme); ok {
		d.first = hillFirstContact(ser, d, hillModeObjectSlots(ser.noms, d.slot))
		return d, true
	}
	voisin := func(slot uint32) bool { return len(ser.owner[slot+1]) >= hillDesignatorMinOwnerSamples }
	d, ok := hillDesignatorWhere(ser, voisin)
	if !ok {
		return d, false
	}
	d.parVoisinage = true
	d.first = hillFirstContact(ser, d, hillNeighbourSlots(d.slot))
	return d, true
}

// hillDesignatorWhere rend, parmi les slots a serie de tag 5 chainee qui passent `eligible`, celui
// dont la designation bascule le plus (egalite : le plus petit slot).
func hillDesignatorWhere(ser zoneSeries, eligible func(uint32) bool) (hillDesignator, bool) {
	var best hillDesignator
	found := false
	for _, slot := range sortedZoneSlots(ser.desig) {
		if !eligible(slot) {
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
	return best, found
}

// hillModeObjectSlots rend les slots de l objet de mode que designe le nom : le proprietaire, le
// pousseur et la jauge du bloc dont le designateur est la cle. Un role dont le nom manque au film
// ne rend aucun slot.
func hillModeObjectSlots(noms zoneNoms, designateur uint32) []uint32 {
	b, ok := zoneBlocDuSlot(designateur, noms, func(b zoneBlocNomme) uint32 { return b.cle })
	if !ok {
		return nil
	}
	var out []uint32
	for _, nom := range []uint32{b.proprietaire, b.pousseur, b.jauge} {
		if s, ok := noms.parNom[nom]; ok {
			out = append(out, s)
		}
	}
	return out
}

// hillNeighbourSlots rend les slots VOISINS du designateur (`designateur+1` a `+hillDesignatorSpan`),
// ou la regle de voisinage cherche les canaux de l objet de mode.
func hillNeighbourSlots(designateur uint32) []uint32 {
	out := make([]uint32, 0, hillDesignatorSpan)
	for k := uint32(1); k <= hillDesignatorSpan; k++ {
		out = append(out, designateur+k)
	}
	return out
}

// hillFirstContact rend la frame de la premiere emission de l'objet de mode : son designateur, ou
// les canaux de `slots` (proprietaire, capteur, jauge).
func hillFirstContact(ser zoneSeries, d hillDesignator, slots []uint32) int {
	first := d.changes[0]
	for _, slot := range slots {
		for _, ss := range [][]zoneSample{ser.owner[slot], ser.gauge[slot]} {
			if len(ss) > 0 && ss[0].t < first {
				first = ss[0].t
			}
		}
	}
	return first
}

// buildDesignatedHills decoupe le match en periodes bornees par le designateur, place chaque
// periode sur la zone ou se tient le camp proprietaire (zone_states_hill_garde.go) — a defaut de
// garde lisible, par la grappe des positions pendant les montees de la jauge, puis pendant toute la
// periode — et publie les periodes localisees.
func buildDesignatedHills(zones []Zone, ser zoneSeries, h hillCtx, c zoneCtx,
	cov *ZonesCoverage,
) []ZoneState {
	cov.Method = ZoneMethodDesignator
	periods := hillDesignatedPeriods(h.d, c.frames)
	cov.HillPeriods = len(periods)
	// LE CANAL DE PROPRIETE EST CELUI DU BLOC DONT LE DESIGNATEUR EST LA CLE, designe par le nom
	// (le slot voisin du designateur en repli). Niveau de preuve accepte et reserve : cf.
	// hillStatesOf.
	owner := ser.owner[hillOwnerSlotOf(ser, h.d, cov, c.fb)]
	loc := hillLocator{zones: zones, ramps: zoneRampsOf(ser), pts: zonePointsByFrame(c.tracks),
		team: hillTeamPointsByFrame(c.tracks), owner: owner, fb: c.fb}
	kept := make([]hillPeriod, 0, len(periods))
	for _, p := range periods {
		if !loc.place(&p) {
			// UNE COLLINE DESIGNEE QUE NI LA GARDE NI LA GRAPPE NE LOCALISENT EST ECARTEE ET SE
			// COMPTE : elle a existe, on ne sait pas ou (cf. ZonesCoverage.Unpaired).
			cov.Unpaired++
			continue
		}
		kept = append(kept, p)
	}
	states := hillStatesOf(kept, owner, h.teams, cov, c.fb)
	cov.Paired = len(states)
	tallyZoneStates(states, cov)
	return states
}

// hillLocator porte ce que le placement d une periode lit (regle des 5 parametres).
type hillLocator struct {
	zones []Zone
	ramps []zoneRamp
	pts   map[int][]Point
	team  map[int][]hillTeamPoint
	owner []zoneSample
	fb    *fallback.Compteur
}

// place pose la zone de la periode : par la GARDE d abord ; une garde illisible revient aux votes
// de la grappe pendant les montees de la jauge, puis pendant toute la periode (deux replis
// nommes). Faux quand la periode reste sans zone.
func (l hillLocator) place(p *hillPeriod) bool {
	ref, issue := hillGardeOf(l.zones, l.team, l.owner, p.t0, p.t1).place()
	hillPeriodTop(l.ramps, p)
	switch issue {
	case hillGardePlacee:
		p.ref, p.hasRef = ref, true
		return true
	case hillGardeEcartee:
		return false
	}
	l.fb.Declenche(fallback.NomCollineVotesSansGarde)
	votes := hillVotesInRamps(l.zones, l.pts, l.ramps, p)
	if len(votes) == 0 {
		// REPLI NOMME ET COMPTE (D14) : aucune rampe de capture dans la periode, les votes
		// sont repris sur TOUTE la periode — donc sur des instants ou personne ne capture.
		l.fb.Declenche(fallback.NomCollineVotesPeriodeEntiere)
		votes = hillVotes(l.zones, l.pts, p.t0, p.t1)
	}
	p.ref, p.hasRef = clearModalZone(votes)
	return p.hasRef
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
			p.top, p.hasTop = r.top, true
		}
	}
	return votes
}

// hillPeriod est un intervalle pendant lequel une colline est gardee.
type hillPeriod struct {
	t0, t1 int
	ref    int
	hasRef bool
	// top est le sommet de la jauge atteint pendant la periode — la progression publiee, sur
	// l'echelle DU JEU (gaugeProgressOf, zone_states.go) — et hasTop dit si une rampe en a fourni
	// un : une colline gardee sans qu'aucune jauge n'y monte (colline VIDE) n'a pas de progression
	// a publier.
	top    uint64
	hasTop bool
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
	slices.SortStableFunc(ramps, func(a, b zoneRamp) int { return cmp.Compare(a.t0, b.t0) })
	if len(ramps) == 0 {
		return nil
	}
	pts := zonePointsByFrame(c.tracks)
	raw := make([]hillPeriod, 0, len(ramps))
	for _, r := range ramps {
		p := hillPeriod{t0: r.t0, t1: r.tPeak, top: r.top, hasTop: true}
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
	// AUCUNE JAUGE EN DIRECT SUR UNE COLLINE (lot C-ter, volets 1 et 3, 2026-08-19) : en KOTH le
	// tag 3 n'est PAS la progression de garde mais un COMPTEUR DE TRANSFERT d'environ une seconde
	// (9-10 pas fixes quelle que soit la duree de la garde, mesure du volet 1 sur les 4 films
	// KOTH) ; la progression de garde vit dans le canal par joueur (mode B tag 7), hors de ce
	// calque. Publier cette rampe comme jauge montrerait un arc qui se remplit en une seconde a
	// chaque prise — credible et faux. `ZoneState.Gauge` reste donc nil ici, et
	// `coverage.zones.gaugePoints` vaut 0 sur un film a colline.
	// AUCUN PROPRIETAIRE SUR CE REPLI : sans designateur, il n'y a pas d'objet de mode, donc pas
	// de slot voisin ou lire le camp. Une colline localisee par la seule grappe des positions
	// reste ACTIVE et sans camp — la deduire de la grappe serait une invention.
	states := hillStatesOf(periods, nil, nil, cov, c.fb)
	cov.Paired = len(states)
	tallyZoneStates(states, cov)
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

// hillOwnerSlotOf rend le canal de propriete de la colline : le PROPRIETAIRE du bloc dont le
// designateur est la cle de nommage, designe par le nom (zone_states_owner_nom.go). Faute de nom
// au vocabulaire, le slot VOISIN du designateur — la structure de l objet de mode que l election
// du designateur exige deja (repli compte). La ou les deux repondent, le voisin est le controle
// du nom : une discordance se compte dans `ownerVoteDisagreed`, le nom est retenu.
func hillOwnerSlotOf(ser zoneSeries, d hillDesignator, cov *ZonesCoverage, fb *fallback.Compteur) uint32 {
	voisin := d.slot + 1
	s, ok := zoneProprietaireDeCle(d.slot, ser.noms)
	if !ok {
		fb.Declenche(fallback.NomCollineProprietaireVoisinDuDesignateur)
		return voisin
	}
	cov.OwnerNamed++
	if s != voisin {
		cov.OwnerVoteDisagreed++
	}
	return s
}
