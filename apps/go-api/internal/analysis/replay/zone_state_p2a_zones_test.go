package replay

// zone_state_p2a_zones_test.go — CB.2a.1 : LE SLOT ti=13 ET LA ZONE DU CATALOGUE.
//
// LA QUESTION, exactement. Le film dit qu'un slot ti=13 voit sa jauge (tag 3) monter puis
// culminer ; l'oracle dit qu'un joueur nomme a capture une zone a l'instant t ; le rejeu dit ou
// ce joueur se trouvait a t. Ces trois lectures ne partagent AUCUN identifiant : la seule chose
// qui puisse les relier est la COINCIDENCE d'un sommet de jauge avec une capture attribuee a une
// zone. Si cette coincidence rend une carte slot -> zone STABLE sur tout le match, alors le slot
// ti=13 EST la zone — sinon la coincidence n'etait qu'une coincidence, et c'est ce que les deux
// temoins cherchent a montrer.
//
// LES DEUX TEMOINS, et pourquoi il en faut deux :
//
//	PERMUTATION  les memes captures, les memes slots, mais reapparies par decalage cyclique.
//	             Il casse le lien en preservant les marges — c'est le niveau du HASARD de la
//	             carte, celui contre lequel 90 % doit etre lu.
//	DECALAGE     le slot dont la jauge culmine 20 s plus loin. Il repond a l'objection que le
//	             premier ne couvre pas : « n'importe quel instant marcherait ».

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/analysis/replay/mapvar"
)

// p2aRamp est une montee monotone du tag 3 sur un slot. Definition reprise de la phase 1.
type p2aRamp struct {
	slot         uint32
	t0, tMax     int
	qStart, qMax uint64
	samples      int
}

// p2aFindRamps decoupe une serie chronologique en montees monotones.
func p2aFindRamps(slot uint32, ss []p2aEch) []p2aRamp {
	var out []p2aRamp
	i := 0
	for i < len(ss) {
		j := i
		for j+1 < len(ss) && ss[j+1].pay >= ss[j].pay {
			j++
		}
		if n := j - i + 1; n >= p2aRampMinSamples && ss[j].pay-ss[i].pay >= p2aRampMinAmplitude {
			out = append(out, p2aRamp{slot: slot, t0: ss[i].tMS, tMax: ss[j].tMS,
				qStart: ss[i].pay, qMax: ss[j].pay, samples: n})
		}
		if j == i {
			i++
			continue
		}
		i = j + 1
	}
	return out
}

// p2aRampes rend toutes les rampes du tag 3 du film, triees par sommet.
func p2aRampes(sc *p2aScan) []p2aRamp {
	var out []p2aRamp
	series := p2aSeries(sc.scal, p2aTagQuant)
	for _, s := range p2aSlotsTries(series) {
		out = append(out, p2aFindRamps(s, series[s])...)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].tMax < out[j].tMax })
	return out
}

// p2aCapture est une capture d'oracle, appariee des deux cotes quand c'est possible.
type p2aCapture struct {
	tMS  int
	xuid string
	stat string
	// zone du catalogue ou le capteur se trouvait (rang spatial), si attribuee.
	rank    int
	inst    int32
	hasZone bool
	// slot ti=13 dont la jauge culmine dans la fenetre, si unique.
	slot    uint32
	hasSlot bool
	// ambigu : plusieurs slots culminent a la meme distance de t.
	ambigu bool
}

// p2aAppariement est ce que CB.2a.1 transmet a CB.2a.2 : la carte slot -> zone et les captures.
type p2aAppariement struct {
	zoneParSlot map[uint32]int
	instParSlot map[uint32]int32
	captures    []p2aCapture
	coherents   int
	juges       int
}

// p2aVoletAppariement mesure CB.2a.1 et rend la carte slot -> zone.
func p2aVoletAppariement(t *testing.T, sb *strings.Builder, e p2aEntree) p2aAppariement {
	t.Helper()
	caps := p2aCaptures(e.src, e.film)
	t.Logf("")
	t.Logf("=== CB.2a.1 APPARIEMENT slot ti=13 -> zone — %d captures identifiees par xuid", len(caps))
	if len(caps) == 0 {
		t.Logf("  aucun oracle nomme sur ce mode : volet CB.2a.1 SANS OBJET (attendu hors zones)")
		fmt.Fprintf(sb, "# a1 %s : sans objet (aucun oracle nomme)\n", e.short)
		return p2aAppariement{}
	}
	if len(e.zones) == 0 {
		t.Logf("  aucune zone au catalogue : volet CB.2a.1 NON MESURABLE")
		fmt.Fprintf(sb, "# a1 %s : non mesurable (aucune zone au catalogue)\n", e.short)
		return p2aAppariement{}
	}
	actions, poses := p2aActions(e.doc, caps)
	t.Logf("  %d captures posees sur l'axe du rejeu (%d hors fenetre ou sans origine)",
		len(actions), len(caps)-poses)
	p2aCourbeDistance(t, sb, e, actions)
	return p2aTableSlotZone(t, sb, e, actions)
}

// p2aActions pose les captures sur l'axe de frames du rejeu (origine retranchee).
func p2aActions(doc ReplayDocument, caps []objectiveevents.IdentifiedEvent) ([]ObjectiveAction, int) {
	out := make([]ObjectiveAction, 0, len(caps))
	n := 0
	for _, c := range caps {
		f, ok := p2aFrameOf(doc, c.TimeMS)
		if !ok {
			continue
		}
		n++
		out = append(out, ObjectiveAction{T: f, XUID: c.XUID, Stat: c.Stat, TimeMS: c.TimeMS})
	}
	return out, n
}

// p2aCourbeDistance publie le taux d'attribution a chaque tolerance, avec le temoin translate.
// Le contrat de `AttributeOptions.MaxDistanceM` l'exige : jamais un seuil seul.
func p2aCourbeDistance(t *testing.T, sb *strings.Builder, e p2aEntree, act []ObjectiveAction) {
	t.Helper()
	loin := TranslateZones(e.zones, mapvar.Vec3{X: p2aTemoinTranslationM, Y: p2aTemoinTranslationM})
	for _, d := range p2aDistancesM {
		_, cov := AttributeZones(act, e.doc.Tracks, e.zones, AttributeOptions{MaxDistanceM: d})
		_, temoin := AttributeZones(act, e.doc.Tracks, loin, AttributeOptions{MaxDistanceM: d})
		t.Logf("  tolerance %4.1f m : attribuees %d/%d = %5.1f %% (hors %d, sans position %d,"+
			" ambigues %d) · TEMOIN zones a %.0f m : %.1f %%", d, cov.Attributed, cov.Actions,
			100*p2aRate(cov.Attributed, cov.Actions), cov.Outside, cov.NoPosition, cov.Ambiguous,
			p2aTemoinTranslationM, 100*p2aRate(temoin.Attributed, temoin.Actions))
		fmt.Fprintf(sb, "a1_courbe\t%s\t%.1f\t%d\t%d\t%d\t%d\t%d\t%d\n", e.short, d,
			cov.Actions, cov.Attributed, cov.Outside, cov.NoPosition, cov.Ambiguous,
			temoin.Attributed)
	}
}

// p2aTableSlotZone apparie chaque capture a un slot, construit la carte modale, et la juge.
func p2aTableSlotZone(t *testing.T, sb *strings.Builder, e p2aEntree,
	act []ObjectiveAction,
) p2aAppariement {
	t.Helper()
	att, _ := AttributeZones(act, e.doc.Tracks, e.zones, AttributeOptions{MaxDistanceM: p2aVerdictDistanceM})
	ramps := p2aRampes(e.sc)
	rows := make([]p2aCapture, 0, len(att))
	for _, a := range att {
		c := p2aCapture{tMS: a.Action.TimeMS, xuid: a.Action.XUID, stat: a.Action.Stat,
			rank: a.SpatialRank, inst: a.InstanceID, hasZone: a.Attributed}
		c.slot, c.hasSlot, c.ambigu = p2aSlotAuSommet(ramps, c.tMS, 0)
		rows = append(rows, c)
	}
	table, votes := p2aCarteModale(rows)
	coherents, juges := p2aCoherence(rows, table)
	perm := p2aTemoinPermutation(rows)
	decal := p2aTemoinDecale(rows, ramps)
	p2aLogTable(t, sb, e, table, votes)
	t.Logf("  COHERENCE de la carte slot -> zone : %d/%d = %.1f %% (seuil %.0f %%, tolerance %.0f m)",
		coherents, juges, 100*p2aRate(coherents, juges), 100*p2aSeuilCoherence, p2aVerdictDistanceM)
	t.Logf("  TEMOINS : permutation cyclique %.1f %% · sommet decale de %d s %.1f %%",
		100*perm, p2aDecalageMS/1000, 100*decal)
	v := "NON TENU"
	if juges > 0 && p2aRate(coherents, juges) >= p2aSeuilCoherence {
		v = "TENU"
	}
	t.Logf("  VERDICT CB.2a.1 : %s", v)
	fmt.Fprintf(sb, "a1_verdict\t%s\t%d\t%d\t%.4f\t%.4f\t%.4f\t%s\n", e.short, coherents, juges,
		p2aRate(coherents, juges), perm, decal, v)
	return p2aAppariement{zoneParSlot: table, instParSlot: p2aInstances(rows, table),
		captures: rows, coherents: coherents, juges: juges}
}

// p2aSlotAuSommet rend le slot dont une rampe culmine au plus pres de `tMS + shift`, dans la
// fenetre. `ambigu` signale deux slots distincts a egale distance : on ne tranche pas.
func p2aSlotAuSommet(ramps []p2aRamp, tMS, shift int) (uint32, bool, bool) {
	cible := tMS + shift
	best, bestD, found, ambigu := uint32(0), p2aFenetreMS+1, false, false
	for _, r := range ramps {
		d := r.tMax - cible
		if d < 0 {
			d = -d
		}
		if d > p2aFenetreMS {
			continue
		}
		switch {
		case !found || d < bestD:
			best, bestD, found, ambigu = r.slot, d, true, false
		case d == bestD && r.slot != best:
			ambigu = true
		}
	}
	if ambigu {
		return 0, false, true
	}
	return best, found, false
}

// p2aCarteModale rend, pour chaque slot, la zone qu'il designe le plus souvent, et les votes.
func p2aCarteModale(rows []p2aCapture) (map[uint32]int, map[uint32]map[int]int) {
	votes := map[uint32]map[int]int{}
	for _, c := range rows {
		if !c.hasSlot || !c.hasZone {
			continue
		}
		if votes[c.slot] == nil {
			votes[c.slot] = map[int]int{}
		}
		votes[c.slot][c.rank]++
	}
	table := map[uint32]int{}
	for s, m := range votes {
		best, bestN := -1, -1
		for z, n := range m {
			if n > bestN || (n == bestN && z < best) {
				best, bestN = z, n
			}
		}
		table[s] = best
	}
	return table, votes
}

// p2aCoherence compte les captures dont la zone est celle que la carte modale attend.
func p2aCoherence(rows []p2aCapture, table map[uint32]int) (int, int) {
	ok, tot := 0, 0
	for _, c := range rows {
		if !c.hasSlot || !c.hasZone {
			continue
		}
		tot++
		if z, has := table[c.slot]; has && z == c.rank {
			ok++
		}
	}
	return ok, tot
}

// p2aTemoinPermutation rend la coherence obtenue en reappariant les slots par decalage cyclique.
// Deterministe : aucun tirage aleatoire, la mesure se rejoue a l'identique.
func p2aTemoinPermutation(rows []p2aCapture) float64 {
	var pairs []p2aCapture
	for _, c := range rows {
		if c.hasSlot && c.hasZone {
			pairs = append(pairs, c)
		}
	}
	if len(pairs) < 2 {
		return 0
	}
	d := len(pairs) / 2
	perm := make([]p2aCapture, len(pairs))
	for i := range pairs {
		perm[i] = pairs[i]
		perm[i].slot = pairs[(i+d)%len(pairs)].slot
	}
	table, _ := p2aCarteModale(perm)
	ok, tot := p2aCoherence(perm, table)
	return p2aRate(ok, tot)
}

// p2aTemoinDecale rend la coherence obtenue en cherchant le sommet 20 s plus loin.
func p2aTemoinDecale(rows []p2aCapture, ramps []p2aRamp) float64 {
	shifted := make([]p2aCapture, 0, len(rows))
	for _, c := range rows {
		if !c.hasZone {
			continue
		}
		c.slot, c.hasSlot, c.ambigu = p2aSlotAuSommet(ramps, c.tMS, p2aDecalageMS)
		shifted = append(shifted, c)
	}
	table, _ := p2aCarteModale(shifted)
	ok, tot := p2aCoherence(shifted, table)
	return p2aRate(ok, tot)
}

// p2aInstances rend l'InstanceID du jeu associe a la zone modale de chaque slot.
func p2aInstances(rows []p2aCapture, table map[uint32]int) map[uint32]int32 {
	out := map[uint32]int32{}
	for _, c := range rows {
		if !c.hasSlot || !c.hasZone {
			continue
		}
		if z, has := table[c.slot]; has && z == c.rank {
			out[c.slot] = c.inst
		}
	}
	return out
}

// p2aCles rend l'identifiant de chaine (tag 5) porte par chaque slot — LA CLE DE NOMMAGE dont la
// phase 1 a mesure qu'elle est identique sur les deux Strongholds.
func p2aCles(sc *p2aScan) map[uint32]uint64 {
	out := map[uint32]uint64{}
	for _, e := range sc.scal {
		if e.tag == p2aTagStringID && e.hasPay {
			out[e.slot] = e.pay
		}
	}
	return out
}

// p2aStructure publie, pour CHAQUE slot de la bande ti=13, le nom de propriete qu'il porte (i0),
// son tag dominant et son volume.
//
// POURQUOI CETTE TABLE EST UNE MESURE, ET LA PLUS STRUCTURANTE DE LA PHASE. Elle rend visible ce
// que l'archetype disait deja sans qu'on l'ait lu ainsi : UN SLOT ti=13 N'EST PAS UNE ZONE, C'EST
// UNE PROPRIETE RESEAU NOMMEE. Les slots qui portent la jauge (tag 3), ceux qui portent le canal
// enumerable (tag 4) et ceux qui portent l'identifiant de chaine (tag 5) sont DISJOINTS — la
// phase 1 avait mesure les trois separement sans pouvoir le dire, faute d'avoir publie i0.
func p2aStructure(t *testing.T, sb *strings.Builder, e p2aEntree) {
	t.Helper()
	parSlot := map[uint32]map[int]int{}
	for _, ech := range e.sc.scal {
		if parSlot[ech.slot] == nil {
			parSlot[ech.slot] = map[int]int{}
		}
		parSlot[ech.slot][ech.tag]++
	}
	t.Logf("")
	t.Logf("=== STRUCTURE de ti=13 — %d slots emettent une valeur scalaire", len(parSlot))
	for _, s := range p2aSlotsTries(parSlot) {
		tag, n, tot := p2aTagDominant(parSlot[s])
		t.Logf("    slot %-5d nom i0 %s · tag dominant %d (%d/%d)", s, p2aNomDe(e.sc, s), tag, n, tot)
		fmt.Fprintf(sb, "structure\t%s\t%d\t%s\t%d\t%d\t%d\n", e.short, s, p2aNomDe(e.sc, s),
			tag, n, tot)
	}
}

// p2aTagDominant rend le tag le plus emis d'un slot, son compte et le total.
func p2aTagDominant(m map[int]int) (int, int, int) {
	best, bestN, tot := -1, -1, 0
	for tag, n := range m {
		tot += n
		if n > bestN {
			best, bestN = tag, n
		}
	}
	return best, bestN, tot
}

// p2aNomDe rend le nom de propriete (i0) le plus emis par un slot, ou « (jamais emis) ».
func p2aNomDe(sc *p2aScan, s uint32) string {
	m := sc.noms[s]
	if len(m) == 0 {
		return "(jamais emis)"
	}
	best, bestN := uint64(0), -1
	for v, n := range m {
		if n > bestN {
			best, bestN = v, n
		}
	}
	return fmt.Sprintf("0x%08X x%d", best, bestN)
}

// p2aLogTable publie la table slot -> zone, avec la cle tag 5 : c'est elle qui porte la clause de
// STABILITE entre les deux Strongholds (memes identifiants -> memes zones).
func p2aLogTable(t *testing.T, sb *strings.Builder, e p2aEntree,
	table map[uint32]int, votes map[uint32]map[int]int,
) {
	t.Helper()
	cles := p2aCles(e.sc)
	t.Logf("  TABLE slot -> zone (cle tag 5, rang spatial, InstanceID) :")
	for _, s := range p2aSlotsTries(table) {
		tot, best := 0, 0
		for z, n := range votes[s] {
			tot += n
			if z == table[s] {
				best = n
			}
		}
		inst := int32(0)
		for _, z := range e.zones {
			if z.SpatialRank == table[s] {
				inst = z.InstanceID
			}
		}
		t.Logf("    slot %-5d cle 0x%08X -> zone rang %d (InstanceID %d) : %d/%d votes (%.0f %%),"+
			" %d zones vues", s, cles[s], table[s], inst, best, tot, 100*p2aRate(best, tot),
			len(votes[s]))
		fmt.Fprintf(sb, "a1_table\t%s\t%d\t0x%08X\t%d\t%d\t%d\t%d\n", e.short, s, cles[s],
			table[s], inst, best, tot)
	}
}
