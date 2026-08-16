package filmdec

// i28_camo_test.go — INSTRUMENT DE MESURE de la PHASE A du plan
// .ai/V7.5/replay2d/PLAN_ETAT_ACTIF_EQUIPEMENT.md : le composant i28
// `unit-active-camo-state` — LE composant NOMMÉ camouflage — est-il l'ÉTAT ACTIF ?
//
// CE QU'IL PUBLIE : chaque lecture d'i28 par le déser de PRODUCTION (camoStateHook,
// cf. ability_state_hooks.go) — le R(3), les deux portes, le quantum R(12) principal et
// les six champs optionnels de queue — par slot et horodatage, avec les dénominateurs
// (records / masque∋i28 / lectures / illisibles).
//
// LE CONTRÔLE NON CIRCULAIRE, ÉNONCÉ AVANT LA MESURE (item A.2). Sur `084a804d` (i48 :
// rang 8 x10, rang 9 x8) et `06dfe6d9` (8:2, 9:2, 11:3), les TRANSITIONS de la fraction
// doivent se CONCENTRER sur les vies dont i48 transmet le rang 8 (camouflage, famille A).
// Les vies aux autres rangs sont le témoin interne : elles doivent rester à la valeur par
// défaut. Si les transitions sont réparties uniformément, i28 n'est pas l'état du
// camouflage ÉQUIPEMENT (camouflage de véhicule/bonus à compter à part si observé).
//
// LECTURE SEULE, gardé par I28_FILM, sauté partout ailleurs (CI comprise). UN SEUL
// décodage filmdec par process : le verrou est pris pour tout le test.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 I28_FILM=<repo>/data/cache/film_chunks/084a804d \
//	  go test ./internal/analysis/filmdec/ -run '^TestI28CamoActiveState$' -timeout 30m -v

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

const i28FilmEnv = "I28_FILM"

// arch28Name est l'étiquette de registre d'i28 — celle par laquelle consumeByName route
// vers consumeUnitActiveCamoState.
const arch28Name = "unit-active-camo-state-component"

// i28Sample est une lecture d'i28 localisée dans le film.
type i28Sample struct {
	slot uint32
	tsUS uint64
	st   CamoState
}

func TestI28CamoActiveState(t *testing.T) {
	dir := os.Getenv(i28FilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", i28FilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	s := eaSetupBiped(t, dir)
	i28idx := s.arch.indicesOfFirst(arch28Name)
	if i28idx < 0 {
		t.Fatalf("composant %q absent de l'archétype biped — composants : %v",
			arch28Name, s.arch.Components)
	}
	t.Logf("composant %d du registre biped : %q", i28idx, s.arch.component(i28idx))
	slotRanks := eaSlotRanks(t, dir)

	var (
		capt struct {
			st  CamoState
			got bool
		}
		records, with28, read, unread int
		samples                       []i28Sample
	)
	prev := camoStateHook
	SetCamoStateHook(func(st CamoState) { capt.st, capt.got = st, true })
	defer SetCamoStateHook(prev)

	minRecord := bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + s.lay.TotalBits()
	for _, c := range s.chunks {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			total := len(pay) * 8
			for p := 0; p+minRecord <= total; {
				i0, slot, idx, ok := matchBipedHeader(pay, p, total, s.slots, true, s.lay)
				if !ok {
					p++
					continue
				}
				records++
				if eaMaskHas(idx, i28idx) {
					with28++
					capt.got = false
					if eaWalkThrough(pay, i0, total, idx, s, i28idx) && capt.got {
						read++
						samples = append(samples, i28Sample{slot: slot, tsUS: pk.TimestampUS, st: capt.st})
					} else {
						unread++
					}
				}
				p = i0 + s.lay.TotalBits()
			}
		}
	}
	if records == 0 {
		t.Fatal("aucun record delta biped reconnu : rien à mesurer")
	}
	t.Logf("RECORDS delta biped %d · masque∋i28 %d (%.3f %%) · LUS %d · illisibles %d",
		records, with28, 100*float64(with28)/float64(records), read, unread)
	if len(samples) == 0 {
		t.Log("VERDICT A : aucune lecture d'i28 exploitable — l'état camo ne se lit pas par ce canal")
		return
	}
	i28LogShape(t, samples)
	perSlot := i28BySlot(samples)
	for ch := 0; ch < i28ChannelCount; ch++ {
		i28LogConcentration(t, ch, perSlot, slotRanks)
	}
	i28LogMinority(t, samples, slotRanks)
	i28LogCurve(t, perSlot, slotRanks)
}

// i28ChannelCount : la voie « fraction principale » (0) puis les six champs de queue
// (1..6). La mesure ne présume PAS laquelle porte l'état : elle les traite toutes.
const i28ChannelCount = 7

// i28ChannelName nomme une voie pour les journaux.
func i28ChannelName(ch int) string {
	if ch == 0 {
		return "fraction principale (R12 sous flag0/flag1)"
	}
	return fmt.Sprintf("queue[%d]", ch-1)
}

// i28ChannelVal rend (valeur, présente) de la voie ch pour une lecture.
func i28ChannelVal(st CamoState, ch int) (int, bool) {
	if ch == 0 {
		return int(st.FracQ), st.HasFrac
	}
	return int(st.SubQ[ch-1]), st.SubPresent[ch-1]
}

// i28LogShape publie la forme du composant : portes, valeurs, champs de queue.
func i28LogShape(t *testing.T, samples []i28Sample) {
	c3 := map[int]int{}
	fracHist := map[int]int{}
	var flag0On, flag1On, hasFrac int
	subPresent := [6]int{}
	subVals := [6]map[int]int{}
	for i := range subVals {
		subVals[i] = map[int]int{}
	}
	for _, sm := range samples {
		c3[int(sm.st.C3)]++
		if sm.st.Flag0 {
			flag0On++
		}
		if sm.st.Flag1Read && sm.st.Flag1 {
			flag1On++
		}
		if sm.st.HasFrac {
			hasFrac++
			fracHist[int(sm.st.FracQ)]++
		}
		for i := 0; i < 6; i++ {
			if sm.st.SubPresent[i] {
				subPresent[i]++
				subVals[i][int(sm.st.SubQ[i])]++
			}
		}
	}
	n := len(samples)
	t.Logf("== FORME d'i28 sur %d lectures ==", n)
	t.Logf("  flag0==1 (aucune fraction) %d · flag0==0&flag1==1 (aucune fraction) %d · "+
		"FRACTION transmise %d", flag0On, flag1On, hasFrac)
	t.Logf("  R(3) de tête : %s", i48RenderInt(c3))
	t.Logf("  quantum R(12) principal (%d valeurs distinctes) : %s",
		len(fracHist), i28RenderCapped(fracHist, 48))
	for i := 0; i < 6; i++ {
		t.Logf("  queue[%d] : présent %d · %s", i, subPresent[i], i28RenderCapped(subVals[i], 12))
	}
}

// i28BySlot regroupe et trie les lectures par slot.
func i28BySlot(samples []i28Sample) map[uint32][]i28Sample {
	out := map[uint32][]i28Sample{}
	for _, sm := range samples {
		out[sm.slot] = append(out[sm.slot], sm)
	}
	for _, list := range out {
		sort.Slice(list, func(a, b int) bool { return list[a].tsUS < list[b].tsUS })
	}
	return out
}

// i28ChanTransitions compte les CHANGEMENTS de valeur de la voie ch entre lectures
// consécutives à valeur transmise d'un même slot — c'est la transition qui date un
// événement, pas la valeur.
func i28ChanTransitions(list []i28Sample, ch int) (trans, pairs int) {
	last, has := 0, false
	for _, sm := range list {
		v, present := i28ChannelVal(sm.st, ch)
		if !present {
			continue
		}
		if has {
			pairs++
			if v != last {
				trans++
			}
		}
		last, has = v, true
	}
	return trans, pairs
}

// i28LogConcentration joue le contrôle A.2 sur UNE voie : transitions par GROUPE DE RANG
// i48. Une voie sans aucune transition est dite en une ligne.
func i28LogConcentration(t *testing.T, ch int, perSlot map[uint32][]i28Sample, slotRanks map[uint32][]int) {
	type group struct{ slots, reads, present, trans, pairs int }
	var g8, gOther, gNoID group
	slots := make([]uint32, 0, len(perSlot))
	for sl := range perSlot {
		slots = append(slots, sl)
	}
	sort.Slice(slots, func(a, b int) bool { return slots[a] < slots[b] })
	totalTrans := 0
	var rows []string
	for _, sl := range slots {
		list := perSlot[sl]
		trans, pairs := i28ChanTransitions(list, ch)
		totalTrans += trans
		present := 0
		for _, sm := range list {
			if _, p := i28ChannelVal(sm.st, ch); p {
				present++
			}
		}
		ranks, hasID := slotRanks[sl]
		var g *group
		switch {
		case eaHasRank(ranks, 8):
			g = &g8
		case hasID:
			g = &gOther
		default:
			g = &gNoID
		}
		g.slots++
		g.reads += len(list)
		g.present += present
		g.trans += trans
		g.pairs += pairs
		if trans > 0 {
			rows = append(rows, fmt.Sprintf("  slot %-5d rangs i48 %v : %d lectures · %d valeurs · %d transitions (%d paires)",
				sl, eaRankSet(ranks), len(list), present, trans, pairs))
		}
	}
	if totalTrans == 0 {
		t.Logf("== CONTRÔLE A.2 — %s : AUCUNE transition sur tout le film ==", i28ChannelName(ch))
		return
	}
	t.Logf("== CONTRÔLE A.2 — %s : transitions par vie, avec le rang i48 de la vie ==",
		i28ChannelName(ch))
	for _, r := range rows {
		t.Log(r)
	}
	rate := func(g group) string {
		if g.pairs == 0 {
			return "aucune paire"
		}
		return fmt.Sprintf("%d/%d (%.1f %%)", g.trans, g.pairs, 100*float64(g.trans)/float64(g.pairs))
	}
	t.Logf("  GROUPE rang 8 (camo)      : %d vies · %d lectures · %d valeurs · transitions %s",
		g8.slots, g8.reads, g8.present, rate(g8))
	t.Logf("  GROUPE autres rangs       : %d vies · %d lectures · %d valeurs · transitions %s",
		gOther.slots, gOther.reads, gOther.present, rate(gOther))
	t.Logf("  GROUPE sans identité i48  : %d vies · %d lectures · %d valeurs · transitions %s",
		gNoID.slots, gNoID.reads, gNoID.present, rate(gNoID))
	t.Log("RAPPEL du critère A.2 : les transitions doivent se CONCENTRER sur le groupe " +
		"rang 8 ; réparties uniformément = la voie n'est pas l'état du camouflage équipement")
}

// i28LogMinority liste, pour chaque voie quasi binaire (2 valeurs distinctes ou moins,
// minorité ≤ 40 lectures), les lectures MINORITAIRES avec slot, rang i48 et instant : si la
// minorité tombe toute sur des vies rang 8, la voie est l'interrupteur du camouflage.
func i28LogMinority(t *testing.T, samples []i28Sample, slotRanks map[uint32][]int) {
	for ch := 0; ch < i28ChannelCount; ch++ {
		hist := map[int]int{}
		for _, sm := range samples {
			if v, p := i28ChannelVal(sm.st, ch); p {
				hist[v]++
			}
		}
		if len(hist) < 2 {
			continue
		}
		major, majorN := 0, -1
		total := 0
		for v, n := range hist {
			total += n
			if n > majorN {
				major, majorN = v, n
			}
		}
		if total-majorN > 40 {
			continue
		}
		t.Logf("== LECTURES MINORITAIRES de %s (majorité %d x%d) ==",
			i28ChannelName(ch), major, majorN)
		for _, sm := range samples {
			v, p := i28ChannelVal(sm.st, ch)
			if !p || v == major {
				continue
			}
			t.Logf("  slot %-5d rangs i48 %v · t=%.2f s · valeur %d",
				sm.slot, eaRankSet(slotRanks[sm.slot]), float64(sm.tsUS)/1e6, v)
		}
	}
}

// i28LogCurve publie la COURBE de la vie rang 8 la plus riche en transitions (item A.3),
// sur la voie qui en porte le plus : c'est elle qui date activation ET désactivation si la
// concentration tient. Sans vie rang 8 à transitions, la vie la plus riche TOUTES voies
// confondues est publiée à la place — étiquetée comme telle.
func i28LogCurve(t *testing.T, perSlot map[uint32][]i28Sample, slotRanks map[uint32][]int) {
	// La voie de la courbe est celle qui a PASSÉ le contrôle A.2 : des transitions sur les
	// vies rang 8 et AUCUNE ailleurs. À défaut (aucune voie exclusive), la voie la plus
	// transitionnelle d'une vie rang 8, puis de n'importe quelle vie — étiquetée comme telle.
	exclusive := -1
	for ch := 0; ch < i28ChannelCount; ch++ {
		on8, off8 := 0, 0
		for sl, list := range perSlot {
			trans, _ := i28ChanTransitions(list, ch)
			if eaHasRank(slotRanks[sl], 8) {
				on8 += trans
			} else {
				off8 += trans
			}
		}
		if on8 > 0 && off8 == 0 {
			exclusive = ch
			break
		}
	}
	bestCh, bestSlot, bestTrans, rank8 := -1, uint32(0), 0, true
	pick := func(needRank8 bool) {
		for sl, list := range perSlot {
			if needRank8 && !eaHasRank(slotRanks[sl], 8) {
				continue
			}
			for ch := 0; ch < i28ChannelCount; ch++ {
				if exclusive >= 0 && ch != exclusive {
					continue
				}
				if trans, _ := i28ChanTransitions(list, ch); trans > bestTrans {
					bestCh, bestSlot, bestTrans = ch, sl, trans
				}
			}
		}
	}
	pick(true)
	if bestCh < 0 {
		rank8 = false
		pick(false)
	}
	if bestCh < 0 {
		t.Log("COURBE A.3 : aucune vie avec transition sur aucune voie — pas de courbe à publier")
		return
	}
	list := perSlot[bestSlot]
	t0 := list[0].tsUS
	t.Logf("== COURBE A.3 — slot %d (rangs i48 %v, rang8=%v) · voie %s · %d transitions · "+
		"t0 = première lecture i28 ==", bestSlot, eaRankSet(slotRanks[bestSlot]), rank8,
		i28ChannelName(bestCh), bestTrans)
	shown, last, has := 0, -1, false
	for _, sm := range list {
		v, p := i28ChannelVal(sm.st, bestCh)
		if !p {
			continue
		}
		if has && v == last && shown > 0 {
			continue // même valeur : seule la PREMIÈRE lecture du palier est montrée
		}
		last, has = v, true
		if shown == 60 {
			t.Log("  ... tronqué")
			break
		}
		t.Logf("  t=%8.2f s  q=%4d  q/4095=%.3f", float64(sm.tsUS-t0)/1e6, v, float64(v)/4095)
		shown++
	}
}

// i28RenderCapped rend un histogramme borné à limit entrées (les plus fréquentes d'abord).
func i28RenderCapped(h map[int]int, limit int) string {
	type kv struct{ k, v int }
	rows := make([]kv, 0, len(h))
	for k, v := range h {
		rows = append(rows, kv{k, v})
	}
	sort.Slice(rows, func(a, b int) bool {
		if rows[a].v != rows[b].v {
			return rows[a].v > rows[b].v
		}
		return rows[a].k < rows[b].k
	})
	out := ""
	for i, r := range rows {
		if i == limit {
			out += fmt.Sprintf(" … tronqué, %d valeurs distinctes", len(rows))
			break
		}
		if out != "" {
			out += " "
		}
		out += fmt.Sprintf("%d:%d", r.k, r.v)
	}
	if out == "" {
		return "(vide)"
	}
	return out
}
