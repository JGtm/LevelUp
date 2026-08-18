package replay

// zone_states_hill.go — LE VOLET COLLINE : la zone ACTIVE quand le mode n'a aucun oracle nomme.
//
// CE QUE LA MESURE A ETABLI, ET CE QU'ELLE N'A PAS PU. En KOTH, une seule zone `ti=13` voit sa
// jauge monter a la fois — 100,0 % du temps sur 60 rampes du film de reference. Mais QUELLE zone
// ne se lit nulle part : le catalogue de formes ne connait AUCUN role de colline (mesure du
// 2026-08-18 sur 6 cartes). L'appariement se fait donc sur la GRAPPE DES POSITIONS — la ou les
// joueurs s'agglutinent pendant que la jauge monte — contre les zones que la carte declare sous
// d'autres roles.
//
// LA PERIODE EST UNE RAMPE, ET C'EST UNE MESURE QUI L'A DECIDE. La premiere definition essayee
// segmentait par SLOT ACTIF (une colline = un slot, comme une zone de Bastion = un slot) : elle
// a ete REFUTEE — un seul slot porte la jauge de tout un match KOTH, cette lecture rend UNE
// periode et n'apparie rien. Chaque montee est donc une session de garde, et les periodes
// voisines qui designent la meme zone se fondent.
//
// CE QUE CE VOLET VAUT, ET IL FAUT LE DIRE : la couverture temporelle est une clause FAIBLE (des
// que des rampes sont reparties sur le match, les periodes etendues couvrent presque tout). Ce
// qui porte le resultat est la NETTETE de chaque attribution, et elle est tres inegale —
// excellente sur un film du corpus, moyenne sur un deuxieme, NULLE sur un troisieme. C'est
// pourquoi `coverage.zones.method` nomme cette methode a part : elle ne vaut pas celle des
// captures.

import "sort"

// buildHillStates rend les periodes de garde de la zone ACTIVE, apparieees par la grappe.
func buildHillStates(zones []Zone, ser zoneSeries, c zoneCtx,
	cov *ZonesCoverage,
) []ZoneState {
	cov.Method = ZoneMethodPositions
	ramps := zoneRampsOf(ser)
	sort.SliceStable(ramps, func(i, j int) bool { return ramps[i].t0 < ramps[j].t0 })
	if len(ramps) == 0 {
		return nil
	}
	pts := zonePointsByFrame(c.tracks)
	raw := make([]hillPeriod, 0, len(ramps))
	for _, r := range ramps {
		p := hillPeriod{slot: r.slot, t0: r.t0, t1: r.tPeak, top: r.top}
		p.ref, p.hasRef = clusterZoneOf(zones, pts, r.t0, r.tPeak)
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

// hillPeriod est un intervalle pendant lequel une colline est gardee.
type hillPeriod struct {
	slot   uint32
	t0, t1 int
	ref    int
	hasRef bool
	// top est le sommet de la jauge de la rampe — la progression de la garde.
	top uint64
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

// clusterZoneOf rend la zone la plus peuplee pendant [t0, t1], et si elle ressort.
//
// LA ZONE GAGNANTE DOIT DEVANCER LA DEUXIEME, sinon la periode reste NON APPARIEE : une grappe
// partagee entre deux zones ne designe pas une colline, elle dit que le trafic passe par les
// deux. Publier la premiere par defaut poserait une colline sur un couloir.
func clusterZoneOf(zones []Zone, pts map[int][]Point, t0, t1 int) (int, bool) {
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
func mergeHillPeriods(ps []hillPeriod, frames int) []hillPeriod {
	var out []hillPeriod
	for _, p := range ps {
		if !p.hasRef {
			continue
		}
		if n := len(out); n > 0 && out[n-1].ref == p.ref {
			out[n-1].t1 = p.t1
			if p.top > out[n-1].top {
				out[n-1].top = p.top
			}
			continue
		}
		if n := len(out); n > 0 && out[n-1].t1 < p.t0-1 {
			out[n-1].t1 = p.t0 - 1 // la colline reste la meme jusqu'a la garde suivante
		}
		out = append(out, p)
	}
	if n := len(out); n > 0 && out[n-1].t1 < frames-1 {
		out[n-1].t1 = frames - 1
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
		prog := scales[p.slot].progressOf(p.top)
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
