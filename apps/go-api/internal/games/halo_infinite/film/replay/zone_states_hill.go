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
// CE QUE CE VOLET PUBLIE DEPUIS LE 2026-08-26 : le PROPRIETAIRE. Le tag 4 du slot voisin du
// designateur a ete confronte a trois oracles successifs ; deux se sont reveles inutilisables, le
// troisieme donne 88-89 % d'accord contre un temoin a 56 %. Sous le seuil de 90 % que le plan
// s'etait fixe — et publie quand meme, par DECISION UTILISATEUR datee. Le verdict complet, les
// trois campagnes et la reserve (l'erreur est concentree aux bascules) vivent en tete de
// `hillStatesOf` : c'est la qu'il faut lire avant de toucher a ce canal.
//
// CE QUE CE VOLET NE PUBLIE TOUJOURS PAS : un proprietaire sur le repli par les RAMPES. Sans
// designateur il n'y a pas d'objet de mode, donc pas de slot voisin ou lire le camp.

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
//
// `teams` est l'ensemble des index d'equipe admis (roster). Il ne sert QUE sur la voie du
// designateur, qui seule publie un proprietaire : le repli par les rampes n'a pas d'objet de
// mode, donc pas de slot voisin ou lire le camp.
func buildHillStates(zones []Zone, ser zoneSeries, teams map[uint64]bool, c zoneCtx,
	cov *ZonesCoverage,
) []ZoneState {
	if d, ok := hillDesignatorOf(ser); ok {
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
func buildDesignatedHills(zones []Zone, ser zoneSeries, h hillCtx, c zoneCtx,
	cov *ZonesCoverage,
) []ZoneState {
	cov.Method = ZoneMethodDesignator
	ramps := zoneRampsOf(ser)
	pts := zonePointsByFrame(c.tracks)
	periods := hillDesignatedPeriods(h.d, c.frames)
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
	// LE CANAL DE PROPRIETE EST LE SLOT VOISIN DU DESIGNATEUR — celui que l'election exige deja
	// (`hillDesignatorMinOwnerSamples`). Niveau de preuve accepte et reserve : cf. hillStatesOf.
	states := hillStatesOf(kept, ser.owner[h.d.slot+1], h.teams, cov)
	cov.Paired = len(states)
	tallyZoneStates(states, cov)
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
	sort.SliceStable(ramps, func(i, j int) bool { return ramps[i].t0 < ramps[j].t0 })
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
	states := hillStatesOf(periods, nil, nil, cov)
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

// hillStatesOf regroupe les periodes par zone et rend les intervalles ACTIFS, avec leur
// PROPRIETAIRE quand le canal le donne.
//
// # LE PROPRIETAIRE DE LA COLLINE, ET LE NIVEAU DE PREUVE QUI L'A AUTORISE (2026-08-26)
//
// Le canal est le tag 4 du slot VOISIN du designateur (`d.slot+1`) — celui-la meme que la
// condition d'election du designateur exige deja (`hillDesignatorMinOwnerSamples`). Trois
// campagnes de mesure l'ont confronte a trois oracles differents, et il faut lire leur verdict
// ensemble parce qu'il n'est PAS unanime :
//
//	D2      score de MODE : REFUTE COMME ORACLE — en KOTH il compte des collines GAGNEES
//	        (3-2, 4-2 : les scores de l'API), pas des secondes de garde, et deux films sur
//	        quatre n'en repliquent qu'UN camp. Aucun denominateur exploitable.
//	D2-bis  prises `th=10` : **88-89 % d'accord sur les deux films longs, contre un temoin de
//	        decalage a 56 %** — plus de trente points d'ecart, sur 64 et 92 confrontations.
//	        Sous le seuil de 90 % que le plan s'etait fixe. Sur les deux films COURTS (13 et 35
//	        emissions du canal) signal et temoins se confondent.
//	D2-ter  score PERSONNEL : REFUTE COMME ORACLE — delta dominant median de 150 points contre
//	        0-25 pour le camp domine, quand un frag vaut ~100 et un tic de colline quelques
//	        points. Il mesure qui a TUE, pas qui tient.
//
// **LE SEUIL DE 90 % N'A JAMAIS ETE ATTEINT NI REBAISSE.** Ce qui a change, c'est la DECISION :
// l'utilisateur a accepte ce niveau de preuve le 2026-08-26 pour ce calque, avec le precedent de
// la garde de l'ouvrier de rejeu (retenue a 88 %). Le canal n'a jamais ete refute — il a ete
// mesure sous le seuil, ce qui n'est pas la meme chose.
//
// **OU VIT L'ERREUR RESIDUELLE, ET C'EST CE QUI REND LE RISQUE ACCEPTABLE** : elle est
// concentree aux BASCULES. L'oracle `th=10` date ses prises au bloc de temps fort, pas a
// l'action — d'ou une fenetre d'appariement de +/- 20 s. Les 11 a 12 % de desaccord se lisent
// donc comme un flottement AUTOUR de l'instant du changement de main, pas comme une teinte
// fausse sur toute la duree d'une garde. Un lecteur qui trouverait une colline de la mauvaise
// couleur PENDANT une garde entiere tiendrait une regression, pas cette reserve.
//
// # CE QUE CE PRODUCTEUR NE PUBLIE PAS, ET POURQUOI
//
// `OwnerChecked` / `OwnerAgreed` restent a ZERO sur ce chemin. Ce sont les compteurs du CONTROLE
// INDEPENDANT de la methode par captures (la valeur du tag 4 confrontee a l'equipe du capteur) ;
// la colline n'a pas d'equivalent en production — son controle vit dans les instruments de D2-bis,
// sous garde `ZONE_FILM`. Publier des compteurs a zero comme s'ils avaient ete verifies serait
// pire que leur absence.
func hillStatesOf(periods []hillPeriod, owner []zoneSample, teams map[uint64]bool,
	cov *ZonesCoverage,
) []ZoneState {
	runs := hillOwnerRuns(owner, teams, cov)
	byRef := map[int][]ZoneSpan{}
	for _, p := range periods {
		if p.t1 < p.t0 {
			continue
		}
		// LE SOMMET DE JAUGE D'UNE PERIODE DE COLLINE SE CONVERTIT PAR LA MEME FONCTION QUE LES
		// ZONES SIMULTANEES (lot C-ter, fusion des volets 1 et 3, 2026-08-19) : l'echelle DU JEU
		// (gaugeProgressOf), pas l'ancienne echelle par excursion de match (zoneGaugeScales,
		// supprimee par le volet 3 — cf. zone_states.go). ABSENT quand aucune rampe n'a contribue
		// (colline gardee mais jamais approchee : hasTop faux), jamais une invention a zero.
		var prog *float32
		if p.hasTop {
			v := gaugeProgressOf(p.top)
			prog = &v
		}
		byRef[p.ref] = append(byRef[p.ref], hillSpansOf(p, runs, prog)...)
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

// hillOwnerRun est un intervalle de propriete CONSTANTE, bornes incluses. `team` vaut nil pour
// la valeur neutre du canal — « personne ne la tient » est une MESURE, pas une absence.
type hillOwnerRun struct {
	t0, t1 int
	team   *int
}

// hillOwnerRuns decoupe la serie du canal de propriete en intervalles de valeur constante.
//
// LA SEGMENTATION EST CELLE DE LA METHODE PAR CAPTURES (`mergeZoneRuns` puis « chaque groupe
// court jusqu'a la veille du suivant ») : une seconde ecriture de ce decoupage divergerait au
// premier correctif, et l'ecart serait invisible. Une valeur qui n'est ni le neutre ni un camp
// connu n'ouvre AUCUN intervalle et se compte (`UnknownOwner`) — publier un camp qu'aucun
// joueur n'occupe serait une invention, et la taire empecherait de la voir arriver.
//
// LA DERNIERE VALEUR COURT JUSQU'A LA FIN DE L'AXE, comme sur les zones simultanees : le canal
// est un ETAT, pas un evenement — il ne re-emet pas tant que rien ne change.
func hillOwnerRuns(owner []zoneSample, teams map[uint64]bool, cov *ZonesCoverage) []hillOwnerRun {
	groups := mergeZoneRuns(owner)
	out := make([]hillOwnerRun, 0, len(groups))
	for i, g := range groups {
		t1 := int(^uint(0) >> 1) // le dernier groupe court jusqu'a la fin : borne ouverte a droite
		if i+1 < len(groups) {
			t1 = groups[i+1].t - 1
		}
		if t1 < g.t {
			continue
		}
		team, known := zoneOwnerTeam(g.v, teams)
		if !known {
			cov.UnknownOwner++
			continue
		}
		out = append(out, hillOwnerRun{t0: g.t, t1: t1, team: team})
	}
	return out
}

// hillSpansOf decoupe UNE periode de colline par les changements de proprietaire qui la
// traversent. Rend un seul intervalle — sans proprietaire — quand le canal ne dit rien d'elle.
//
// # LE SOMMET DE JAUGE NE SE DUPLIQUE PAS SUR LES SOUS-INTERVALLES, ET C'EST DELIBERE
//
// `Progress` est le sommet atteint sur LA PERIODE. Quand la colline change de main en cours de
// periode, ce sommet n'est la propriete d'AUCUN des sous-intervalles : le recopier sur chacun
// affirmerait que chacun l'a atteint. Il n'est donc porte que par la periode qui sort d'un seul
// tenant. C'est une perte assumee, et elle ne coute rien a l'ecran : le client ne dessine plus
// `progress` depuis le schema 18 (le sommet statique se lisait comme une jauge en cours).
// # LA PERIODE EST COUVERTE ENTIEREMENT, MEME LA OU LE CANAL SE TAIT
//
// Un trou entre deux intervalles de propriete — ou avant la premiere emission du canal — ne
// FERME PAS la colline : elle est active, on ne sait simplement pas qui la tient. Ces morceaux
// sortent donc en intervalles SANS camp. Les omettre eteindrait la surbrillance au milieu d'une
// garde, ce qui se lirait comme « il ne se passe rien ici » — un contresens, et c'est le defaut
// que ce decoupage a eu avant d'etre corrige (le cas `606d9844` / `8076f97f`, ou le film ne
// replique qu'un camp et se tait sur tout le debut du match).
func hillSpansOf(p hillPeriod, runs []hillOwnerRun, prog *float32) []ZoneSpan {
	var out []ZoneSpan
	curseur := p.t0
	for _, r := range runs {
		t0, t1 := max(r.t0, p.t0), min(r.t1, p.t1)
		if t0 > t1 {
			continue
		}
		if t0 > curseur {
			out = append(out, ZoneSpan{T0: curseur, T1: t0 - 1, Active: true})
		}
		out = append(out, ZoneSpan{T0: t0, T1: t1, Owner: r.team, Active: true})
		curseur = t1 + 1
	}
	if curseur <= p.t1 {
		// Trou de fin — ou periode entiere quand le canal ne dit RIEN d'elle : la colline reste
		// ACTIVE et sans proprietaire, exactement ce que ce producteur publiait avant le
		// 2026-08-26. Une colline dont on ne lit pas le camp n'est pas une colline neutre.
		out = append(out, ZoneSpan{T0: curseur, T1: p.t1, Active: true})
	}
	if len(out) == 1 {
		out[0].Progress = prog
	}
	return out
}
