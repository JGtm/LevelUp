package replay

// zone_states_owner.go — LE VOLET « QUI TIENT LA ZONE » : l'appariement des slots aux zones par
// les captures nommees, et les intervalles de propriete qui en sortent.
//
// POURQUOI DEUX FAMILLES DE SLOTS, ET POURQUOI ELLES SE CHERCHENT SEPAREMENT. La mesure de la
// phase 2a a etabli qu'un slot `ti=13` n'est pas une zone mais UNE PROPRIETE : sur une carte de
// Bastion, la JAUGE (tag 3) et le PROPRIETAIRE (tag 4) vivent sur des slots DISJOINTS — 10 slots
// portent le tag 4, 2 seulement portent aussi une jauge, et ces deux-la n'ont qu'une valeur.
// Chercher le proprietaire sur le slot de la jauge rendrait « non mesurable » sans rien dire.
//
//	la JAUGE          se rattache a sa zone par le SOMMET de sa rampe, contre une capture nommee
//	                  attribuee geometriquement. Sans circularite : la zone vient de la position
//	                  du capteur, pas du canal.
//	le PROPRIETAIRE   se rattache a sa zone par la jauge quand le meme slot porte les deux ;
//	                  sinon PAR VOTE — la zone des captures qui tombent dans la fenetre de ses
//	                  changements. Ce vote est partiellement circulaire, et c'est pourquoi le
//	                  CONTROLE publie (`ownerChecked` / `ownerAgreed`) porte sur autre chose :
//	                  la VALEUR contre l'equipe du capteur, que le vote n'a pas servi a choisir.

import "sort"

// zoneOwnerMinAgreements est le nombre MINIMAL de captures concordantes qu'un canal doit porter
// pour etre elu proprietaire d'une zone (revue R1, 2026-08-18).
//
// POURQUOI DEUX, ET PAS UN. L'election maximise l'accord avec le roster (cf. ownerScores) sur des
// canaux ENUMERABLES qui ne prennent que trois valeurs — `0`, `1` et le neutre. Une concordance
// UNIQUE est donc ce que le hasard produit tout seul : un canal quelconque qui vaut `0` au bon
// moment est elu, et il publie alors une teinte sur toute la duree du match. L'artefact ne dit
// plus « je ne sais pas », il dit une couleur — invisible et credible. Deux concordances ne sont
// pas une preuve ; elles ferment le cas ou le canal n'a jamais ete confirme qu'une seule fois.
const zoneOwnerMinAgreements = 2

// zoneOwnerStates construit les intervalles de propriete de chaque zone appariee.
func zoneOwnerStates(in ZoneInput, ser zoneSeries, pairs []zonePair, c zoneCtx,
	cov *ZonesCoverage,
) []ZoneState {
	win := zoneWindowFrames(c.intervalMS)
	ramps := zoneRampsOf(ser)
	gaugeSlot, unpaired := pairGaugeSlots(ramps, pairs, win)
	cov.Paired, cov.Unpaired = len(gaugeSlot), unpaired
	ownerSlot := pairOwnerSlots(ser, pairs, in.TeamByXUID, win)
	cov.Method = ZoneMethodCaptures
	refs := zoneRefsOf(gaugeSlot, ownerSlot)
	// Les zones dont la jauge est appariee mais dont AUCUN canal n'a ete elu : elles ne sont
	// pas publiees, et sans ce compteur leur silence serait indistinguable d'une carte qui ne
	// les declare pas (cf. ZonesCoverage.OwnerUnpaired).
	cov.OwnerUnpaired = len(gaugeSlot) - len(refs)
	out := make([]ZoneState, 0, len(refs))
	for _, ref := range refs {
		st := ZoneState{ZoneRef: ref, Key: ser.keys[gaugeSlot[ref]]}
		st.Spans = ownerSpansOf(ser.owner[ownerSlot[ref]], ser.gauge[gaugeSlot[ref]],
			zoneSpanCtx{frames: c.frames, teams: zoneTeamSet(in.TeamByXUID)}, cov)
		if len(st.Spans) == 0 {
			continue
		}
		out = append(out, st)
	}
	checkOwnerAgreement(ser, ownerSlot, pairs, in.TeamByXUID, win, cov)
	return out
}

// zoneSpanCtx porte ce qu'un intervalle doit connaitre (regle des 5 parametres).
type zoneSpanCtx struct {
	frames int
	// teams est l'ensemble des camps du roster. Vide : seuls 0 et 1 — les deux valeurs
	// MESUREES du canal — sont acceptes comme camps.
	teams map[uint64]bool
}

// zoneTeamSet rend les camps du roster, en valeurs de canal.
func zoneTeamSet(teams map[string]int) map[uint64]bool {
	out := map[uint64]bool{}
	for _, t := range teams {
		if t >= 0 {
			out[uint64(t)] = true
		}
	}
	return out
}

// zoneRefsOf rend les zones qui ont A LA FOIS une jauge et un proprietaire apparies, triees.
//
// LES DEUX SONT EXIGES, et c'est le sens du calque : une jauge sans proprietaire dirait qu'on
// capture sans dire POUR QUI, et un proprietaire sans jauge n'aurait pas d'ancrage a la zone
// hors du vote. La zone qui n'a pas les deux n'est pas publiee.
func zoneRefsOf(gauge, owner map[int]uint32) []int {
	out := make([]int, 0, len(gauge))
	for ref := range gauge {
		if _, ok := owner[ref]; ok {
			out = append(out, ref)
		}
	}
	sort.Ints(out)
	return out
}

// pairGaugeSlots apparie chaque slot de jauge a une zone par VOTE MODAL des captures dont le
// sommet de rampe tombe dans la fenetre. Rend la carte zone -> slot et le nombre de slots
// porteurs d'une rampe qu'aucune capture n'a rattaches.
//
// LA CARTE EST INVERSEE (zone -> slot) PARCE QU'UNE ZONE N'A QU'UNE JAUGE : deux slots qui
// voteraient pour la meme zone sont departages par le nombre de votes, et le perdant reste non
// apparie plutot que de publier deux fois la meme zone.
func pairGaugeSlots(ramps []zoneRamp, pairs []zonePair, win int) (map[int]uint32, int) {
	votes := map[uint32]map[int]int{}
	for _, p := range pairs {
		slot, ok := slotAtPeak(ramps, p.t, win)
		if !ok {
			continue
		}
		if votes[slot] == nil {
			votes[slot] = map[int]int{}
		}
		votes[slot][p.ref]++
	}
	best := map[int]uint32{}
	bestN := map[int]int{}
	for _, slot := range sortedZoneSlots(votes) {
		ref, n := modalZone(votes[slot])
		if n > bestN[ref] {
			best[ref], bestN[ref] = slot, n
		}
	}
	return best, countUnpairedGauges(ramps, best)
}

// countUnpairedGauges compte les slots porteurs d'au moins une rampe qui ne sont retenus par
// aucune zone. Ils ne sont PAS publies — le compteur existe pour que ce silence se voie.
func countUnpairedGauges(ramps []zoneRamp, kept map[int]uint32) int {
	held := map[uint32]bool{}
	for _, s := range kept {
		held[s] = true
	}
	seen := map[uint32]bool{}
	n := 0
	for _, r := range ramps {
		if seen[r.slot] || held[r.slot] {
			continue
		}
		seen[r.slot] = true
		n++
	}
	return n
}

// slotAtPeak rend le slot dont une rampe culmine au plus pres de `t`, dans la fenetre. Deux
// slots a egale distance ne tranchent pas : on refuse plutot que de tirer au sort.
func slotAtPeak(ramps []zoneRamp, t, win int) (uint32, bool) {
	best, bestD, found, ambiguous := uint32(0), win+1, false, false
	for _, r := range ramps {
		d := r.tPeak - t
		if d < 0 {
			d = -d
		}
		if d > win {
			continue
		}
		switch {
		case !found || d < bestD:
			best, bestD, found, ambiguous = r.slot, d, true, false
		case d == bestD && r.slot != best:
			ambiguous = true
		}
	}
	return best, found && !ambiguous
}

// modalZone rend la zone la plus votee d'un slot, et son compte.
func modalZone(m map[int]int) (int, int) {
	best, bestN := -1, 0
	for _, ref := range sortedZoneRefs(m) {
		if m[ref] > bestN {
			best, bestN = ref, m[ref]
		}
	}
	return best, bestN
}

// sortedZoneRefs rend les zones d'une table de votes, triees — determinisme du parcours.
func sortedZoneRefs(m map[int]int) []int {
	out := make([]int, 0, len(m))
	for ref := range m {
		out = append(out, ref)
	}
	sort.Ints(out)
	return out
}

// zoneOwnerCandidate est un couple (canal, zone) note par son ACCORD AVEC LE ROSTER.
type zoneOwnerCandidate struct {
	slot  uint32
	ref   int
	score int
}

// pairOwnerSlots elit le canal de propriete de chaque zone : les candidats sont notes par
// l'accord avec le roster, puis attribues par accord decroissant.
func pairOwnerSlots(ser zoneSeries, pairs []zonePair, teams map[string]int, win int) map[int]uint32 {
	return electZoneOwners(zoneOwnerCandidates(ser, pairs, teams, win))
}

// zoneOwnerCandidates note tout couple (canal, zone) qui ATTEINT LE SEUIL d'accord.
//
// UN SLOT SANS AUCUN CHANGEMENT EST ECARTE D'OFFICE : le corpus porte des canaux constamment
// neutres (`0xFFFFFFFF` sur tout le match, trois slots par carte mesuree). Ils ne disent rien
// d'un proprietaire, et les publier remplirait la carte d'intervalles « personne ».
//
// LE SEUIL EST LA SECONDE GARDE (cf. zoneOwnerMinAgreements) : sous deux concordances, le couple
// n'entre meme pas dans l'election — la zone reste sans proprietaire et se compte.
func zoneOwnerCandidates(ser zoneSeries, pairs []zonePair, teams map[string]int,
	win int,
) []zoneOwnerCandidate {
	var out []zoneOwnerCandidate
	for _, slot := range sortedZoneSlots(ser.owner) {
		ss := ser.owner[slot]
		if len(zoneChanges(ss)) == 0 {
			continue
		}
		scores := ownerScores(ss, pairs, teams, win)
		for _, ref := range sortedZoneRefs(scores) {
			if scores[ref] < zoneOwnerMinAgreements {
				continue
			}
			out = append(out, zoneOwnerCandidate{slot: slot, ref: ref, score: scores[ref]})
		}
	}
	return out
}

// electZoneOwners attribue AU PLUS un canal par zone et AU PLUS une zone par canal, par accord
// decroissant.
//
// L'UNICITE DU CANAL EST LA CORRECTION QUI COMPTE (revue R1, 2026-08-18). Sans elle, un canal
// bavard pouvait etre l'argmax de DEUX zones a la fois et publiait alors les MEMES intervalles
// sur les deux — deux zones qui basculent ensemble, ce que le film ne dit nulle part, et
// qu'aucun compteur ne signalait. En cas de double argmax, la zone au plus grand accord garde le
// canal ; l'autre reste sans proprietaire, n'est pas publiee, et entre dans `ownerUnpaired`.
//
// LES EGALITES SE TRANCHENT PAR L'ORDRE (zone croissante, puis slot croissant), jamais au
// hasard : deux cuissons du meme film doivent rendre le meme artefact.
func electZoneOwners(cands []zoneOwnerCandidate) map[int]uint32 {
	sort.SliceStable(cands, func(i, j int) bool {
		switch {
		case cands[i].score != cands[j].score:
			return cands[i].score > cands[j].score
		case cands[i].ref != cands[j].ref:
			return cands[i].ref < cands[j].ref
		default:
			return cands[i].slot < cands[j].slot
		}
	})
	out := map[int]uint32{}
	held := map[uint32]bool{}
	for _, c := range cands {
		if _, taken := out[c.ref]; taken || held[c.slot] {
			continue
		}
		out[c.ref], held[c.slot] = c.slot, true
	}
	return out
}

// ownerScores note un slot candidat, ZONE PAR ZONE : combien de captures de cette zone sont
// suivies, dans la fenetre, d'une valeur qui EST l'index d'equipe du capteur.
//
// POURQUOI CE CRITERE, ET PAS LE SIMPLE VOISINAGE (correction mesuree du 2026-08-18). Une carte
// de Bastion porte DEUX familles de canaux enumerables par zone : le proprietaire canonique (11 a
// 16 emissions, uniquement {0, 1}) et un canal bavard ou `0xFFFFFFFF` domine (32 a 39 emissions).
// Compter les changements VOISINS d'une capture elit le second — il en a simplement plus (20-21
// contre 14) — et le calque publie alors une zone qui bascule sans arret vers « personne » :
// controle a 45,8 %. Noter l'ACCORD AVEC LE ROSTER elit le canal canonique, celui que la phase 2a
// a mesure a 100 % de concordance.
//
// L'ORACLE EST EXTERIEUR AU FILM : l'equipe du capteur vient de la base. Le canal ne se juge donc
// pas sur lui-meme — mais l'election MAXIMISE cette quantite, et c'est pourquoi
// `coverage.zones.ownerAgreed` se lit comme la qualite du MEILLEUR candidat, pas comme une preuve
// independante (celle-la est ecrite au journal du lot, inventaire des canaux a l'appui).
//
// SANS ROSTER (CLI hors ligne), l'accord n'est pas calculable : le score retombe alors sur les
// changements qui SUIVENT une capture, et la degradation se lit dans `ownerChecked` a zero.
func ownerScores(ss []zoneSample, pairs []zonePair, teams map[string]int, win int) map[int]int {
	out := map[int]int{}
	for _, p := range pairs {
		v, ok := zoneValueAfter(ss, p.t, win)
		if !ok {
			continue
		}
		team, known := teams[p.xuid]
		switch {
		case known && v == uint64(team):
			out[p.ref]++
		case len(teams) == 0 && v != zoneNeutralOwner:
			out[p.ref]++
		}
	}
	return out
}

// zoneChanges rend les emissions dont la valeur DIFFERE de la precedente. La premiere n'en est
// pas une : elle n'a pas de precedent, la compter gonflerait le denominateur.
func zoneChanges(ss []zoneSample) []zoneSample {
	var out []zoneSample
	for i := 1; i < len(ss); i++ {
		if ss[i].v != ss[i-1].v {
			out = append(out, ss[i])
		}
	}
	return out
}

// ownerSpansOf construit les intervalles de propriete d'une zone : une valeur tenue jusqu'a la
// suivante, la derniere jusqu'a la fin de l'axe.
//
// L'INTERVALLE COMMENCE A LA PREMIERE EMISSION, PAS A LA FRAME 0 : avant elle, le film ne dit
// rien de cette zone. L'etendre jusqu'au debut affirmerait une neutralite qui n'est pas mesuree.
func ownerSpansOf(owner, gauge []zoneSample, c zoneSpanCtx, cov *ZonesCoverage) []ZoneSpan {
	groups := mergeZoneRuns(owner)
	// L echelle de la progression est celle de la jauge DE CETTE ZONE sur CE match (cf.
	// zoneGaugeOf) : la plage declaree du deser ne dit rien de l excursion reelle.
	scale := zoneGaugeOf(gauge)
	out := make([]ZoneSpan, 0, len(groups))
	for i, g := range groups {
		t1 := c.frames - 1
		if i+1 < len(groups) {
			t1 = groups[i+1].t - 1
		}
		if t1 < g.t {
			continue
		}
		team, known := zoneOwnerTeam(g.v, c.teams)
		if !known {
			cov.UnknownOwner++
			continue
		}
		span := ZoneSpan{T0: g.t, T1: t1, Owner: team}
		span.Progress = zonePeakProgress(gauge, scale, g.t, t1)
		out = append(out, span)
	}
	return out
}

// mergeZoneRuns fond les emissions consecutives de MEME valeur : le canal re-emet sans changer.
func mergeZoneRuns(ss []zoneSample) []zoneSample {
	out := make([]zoneSample, 0, len(ss))
	for i, s := range ss {
		if i > 0 && s.v == ss[i-1].v {
			continue
		}
		out = append(out, s)
	}
	return out
}

// zoneOwnerTeam traduit une valeur de canal en camp. Rend (nil, true) pour la valeur neutre, et
// (nil, false) pour une valeur qui n'est pas un camp connu — celle-la n'ouvre aucun intervalle.
func zoneOwnerTeam(v uint64, teams map[uint64]bool) (*int, bool) {
	if v == zoneNeutralOwner {
		return nil, true
	}
	switch {
	case len(teams) > 0 && teams[v]:
	case len(teams) == 0 && v <= 1:
	default:
		return nil, false
	}
	t := int(v)
	return &t, true
}

// zonePeakProgress rend le SOMMET de la jauge dans l'intervalle, ou nil quand aucune emission
// n'y tombe (la zone n'a pas de jauge appariee, ou personne ne l'a contestee).
func zonePeakProgress(gauge []zoneSample, scale zoneGauge, t0, t1 int) *float32 {
	top, found := uint64(0), false
	for _, s := range gauge {
		if s.t < t0 || s.t > t1 {
			continue
		}
		if !found || s.v > top {
			top, found = s.v, true
		}
	}
	if !found {
		return nil
	}
	return scale.progressOf(top)

}

// checkOwnerAgreement confronte la valeur du canal juste APRES chaque capture a l'equipe du
// capteur, et remplit le controle publie.
//
// LES EMISSIONS NEUTRES SONT HORS DENOMINATEUR, et c'est ce que la phase 2a a mesure : une
// capture ne change pas toujours le proprietaire (une zone deja tenue par l'equipe qui la
// securise ne fait pas bouger le canal), et le neutre est alors la valeur d'AVANT.
func checkOwnerAgreement(ser zoneSeries, owner map[int]uint32, pairs []zonePair,
	teams map[string]int, win int, cov *ZonesCoverage,
) {
	for _, p := range pairs {
		slot, ok := owner[p.ref]
		if !ok {
			continue
		}
		team, ok := teams[p.xuid]
		if !ok {
			continue
		}
		v, ok := zoneValueAfter(ser.owner[slot], p.t, win)
		if !ok || v == zoneNeutralOwner {
			continue
		}
		cov.OwnerChecked++
		if v == uint64(team) {
			cov.OwnerAgreed++
		}
	}
}

// zoneValueAfter rend la valeur du canal a la frame de la capture ou APRES, dans la fenetre.
func zoneValueAfter(ss []zoneSample, t, win int) (uint64, bool) {
	i := sort.Search(len(ss), func(k int) bool { return ss[k].t >= t })
	if i >= len(ss) || ss[i].t > t+win {
		return 0, false
	}
	return ss[i].v, true
}

// tallyZoneSpans compte les intervalles publies, toutes zones confondues.
func tallyZoneSpans(states []ZoneState, cov *ZonesCoverage) {
	for _, s := range states {
		cov.Spans += len(s.Spans)
	}
}
