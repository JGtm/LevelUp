package filmdec

// zone_state_report_test.go — suite de `zone_state_measure_test.go` (seuil de 500 lignes par
// fichier) : les volets (b) `boundary-color` et (c) `rtpc` du gate G-C1, et l outillage commun
// des mesures d etat (couverture, transitions, niveau du hasard, palmares de valeurs).

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// -----------------------------------------------------------------------------------------
// C.1b.3 (b) — boundary-color : les quadruplets
// -----------------------------------------------------------------------------------------

// zsReportColor publie les quadruplets distincts et leurs changements datés.
func zsReportColor(t *testing.T, sb *strings.Builder, col *zsCollect, o zcOracle) {
	t.Helper()
	if len(col.color) == 0 {
		t.Logf("C.1b.3 (b) — aucun record de boundary-color recolte")
		return
	}
	quad := map[[4]uint64]int{}
	firstByte := map[uint64]int{}
	for _, s := range col.color {
		if len(s.vals) < 4 {
			continue
		}
		quad[[4]uint64{s.vals[0], s.vals[1], s.vals[2], s.vals[3]}]++
		firstByte[s.vals[0]]++
	}
	t.Logf("C.1b.3 (b) — boundary-color : %d records · %d QUADRUPLETS distincts (seuil G-C1 : <= 8)",
		len(col.color), len(quad))
	type qc struct {
		q [4]uint64
		n int
	}
	list := make([]qc, 0, len(quad))
	for q, n := range quad {
		list = append(list, qc{q, n})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].n > list[j].n })
	for i, e := range list {
		if i >= 10 {
			break
		}
		t.Logf("    quadruplet %3d/%3d/%3d/%3d : %6d records (%5.1f %%) -> rgba %.3f/%.3f/%.3f/%.3f",
			e.q[0], e.q[1], e.q[2], e.q[3], e.n, 100*float64(e.n)/float64(len(col.color)),
			ManagedObjectBoundaryColorValue(e.q[0]), ManagedObjectBoundaryColorValue(e.q[1]),
			ManagedObjectBoundaryColorValue(e.q[2]), ManagedObjectBoundaryColorValue(e.q[3]))
	}
	t.Logf("  PREMIER OCTET (les 4 niveaux 55/119/183/247 de la phase 1a) : %s",
		zsTopValues(firstByte, len(col.color), 8))
	trans := zsTransitions(col.color)
	t.Logf("  changements de couleur sur un meme slot : %d", len(trans))
	cov, den := zsCoverage(o.times, trans, zsGateWindowMS)
	t.Logf("  GATE G-C1 (b) : %d/%d captures (%.1f %%) ont un changement de couleur dans"+
		" +/- %d ms — seuil 80 %% ; quadruplets distincts %d — seuil <= 8",
		cov, den, 100*zcRate(cov, den), zsGateWindowMS, len(quad))
	verdict := "NON TENU"
	if den > 0 && zcRate(cov, den) >= 0.8 && len(quad) <= 8 {
		verdict = "TENU"
	}
	t.Logf("  VERDICT G-C1 (b) : %s", verdict)
	fmt.Fprintf(sb, "# C.1b.3b boundary-color : %d records, %d quadruplets, %d changements, captures couvertes %d/%d (%.1f %%), verdict %s\n",
		len(col.color), len(quad), len(trans), cov, den, 100*zcRate(cov, den), verdict)
}

// -----------------------------------------------------------------------------------------
// C.1b.3 (c) — rtpc
// -----------------------------------------------------------------------------------------

// zsReportRTPC publie les identifiants et le comportement de la valeur.
func zsReportRTPC(t *testing.T, sb *strings.Builder, col *zsCollect) {
	t.Helper()
	if len(col.rtpc) == 0 {
		t.Logf("C.1b.3 (c) — aucun record de rtpc recolte")
		return
	}
	byID := map[uint64]int{}
	withVal := 0
	for _, s := range col.rtpc {
		byID[s.vals[0]]++
		if len(s.vals) > 1 {
			withVal++
		}
	}
	t.Logf("C.1b.3 (c) — rtpc : %d records · %d identifiants distincts · %d portent une valeur"+
		" (%.1f %%)", len(col.rtpc), len(byID), withVal, 100*zcRate(withVal, len(col.rtpc)))
	t.Logf("    identifiants : %s", zsTopValuesHex(byID, len(col.rtpc), 8))
	// Rampes de la valeur, par (slot, identifiant) : si le canal suit la progression, ses
	// rampes doivent avoir la meme forme que celles de radial-progress.
	bySlot := map[uint32][]zsSample{}
	for _, s := range col.rtpc {
		if len(s.vals) > 1 {
			bySlot[s.slot] = append(bySlot[s.slot], zsSample{tMS: s.tMS, slot: s.slot,
				vals: []uint64{s.vals[1]}})
		}
	}
	nRamps := 0
	for slot, ss := range bySlot {
		sort.SliceStable(ss, func(i, j int) bool { return ss[i].tMS < ss[j].tMS })
		nRamps += len(zsFindRampsRTPC(slot, ss))
	}
	t.Logf("  rampes monotones de la VALEUR (22 bits) : %d sur %d slots", nRamps, len(bySlot))
	fmt.Fprintf(sb, "# C.1b.3c rtpc : %d records, %d identifiants, %d avec valeur, %d rampes\n",
		len(col.rtpc), len(byID), withVal, nRamps)
}

// zsFindRampsRTPC reprend zsFindRamps avec une amplitude adaptee a 22 bits (le seuil de 16
// quanta sur 256 devient 16/256 de 2^22).
func zsFindRampsRTPC(slot uint32, ss []zsSample) []zsRamp {
	const minAmp = uint64(1) << 18 // meme part de la plage que 16/256 sur 8 bits
	var out []zsRamp
	i := 0
	for i < len(ss) {
		j := i
		for j+1 < len(ss) && ss[j+1].vals[0] >= ss[j].vals[0] {
			j++
		}
		if j-i+1 >= zsRampMinSamples && ss[j].vals[0]-ss[i].vals[0] >= minAmp {
			out = append(out, zsRamp{slot: slot, t0: ss[i].tMS, tMax: ss[j].tMS,
				qStart: ss[i].vals[0], qMax: ss[j].vals[0], samples: j - i + 1})
		}
		if j == i {
			i++
			continue
		}
		i = j + 1
	}
	return out
}

// -----------------------------------------------------------------------------------------
// outillage commun
// -----------------------------------------------------------------------------------------

// zsTransitions rend les instants ou la valeur publiee CHANGE sur un meme slot.
func zsTransitions(ss []zsSample) []int {
	bySlot := map[uint32][]zsSample{}
	for _, s := range ss {
		bySlot[s.slot] = append(bySlot[s.slot], s)
	}
	var out []int
	for _, list := range bySlot {
		sort.SliceStable(list, func(i, j int) bool { return list[i].tMS < list[j].tMS })
		for i := 1; i < len(list); i++ {
			if !zsSameVals(list[i-1].vals, list[i].vals) {
				out = append(out, list[i].tMS)
			}
		}
	}
	sort.Ints(out)
	return out
}

func zsSameVals(a, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// zsCoverage compte combien d'instants de `events` ont au moins un instant de `marks` a moins de
// `win` ms. Le denominateur est le nombre d'evenements.
func zsCoverage(events, marks []int, win int) (int, int) {
	if len(events) == 0 || len(marks) == 0 {
		return 0, len(events)
	}
	sorted := append([]int(nil), marks...)
	sort.Ints(sorted)
	n := 0
	for _, e := range events {
		i := sort.SearchInts(sorted, e)
		if (i < len(sorted) && sorted[i]-e <= win) || (i > 0 && e-sorted[i-1] <= win) {
			n++
		}
	}
	return n, len(events)
}

// zsNearEvents rend la part de `marks` qui tombe pres d'un evenement — l'autre sens de la
// mesure, publie pour qu'on voie les deux.
func zsNearEvents(nom string, marks, events []int) string {
	if len(marks) == 0 {
		return fmt.Sprintf("%s : aucun", nom)
	}
	n, _ := zsCoverage(marks, events, zsGateWindowMS)
	return fmt.Sprintf("%s : %d, dont %d (%.1f %%) a moins de %d ms d'un evenement d'objectif",
		nom, len(marks), n, 100*zcRate(n, len(marks)), zsGateWindowMS)
}

// zsTopValues rend les n valeurs les plus frequentes, en decimal.
func zsTopValues(m map[uint64]int, total, n int) string {
	return zsTop(m, total, n, "%d:%.1f%% ")
}

// zsTopValuesHex rend les n valeurs les plus frequentes, en hexadecimal.
func zsTopValuesHex(m map[uint64]int, total, n int) string {
	return zsTop(m, total, n, "0x%X:%.1f%% ")
}

func zsTop(m map[uint64]int, total, n int, format string) string {
	keys := make([]uint64, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return m[keys[i]] > m[keys[j]] })
	var sb strings.Builder
	for i, k := range keys {
		if i >= n {
			break
		}
		fmt.Fprintf(&sb, format, k, 100*float64(m[k])/float64(total))
	}
	return sb.String()
}

// zsSpan rend l'etendue temporelle d'une suite d'instants.
func zsSpan(ts []int) int {
	if len(ts) < 2 {
		return 0
	}
	lo, hi := ts[0], ts[0]
	for _, t := range ts {
		if t < lo {
			lo = t
		}
		if t > hi {
			hi = t
		}
	}
	return hi - lo
}

// zsSoloRampTime rend le temps (ms) ou EXACTEMENT une rampe est active, et le temps total ou au
// moins une l'est. C'est la clause KOTH du gate G-C1 : sur une colline, une seule zone se remplit
// a la fois.
func zsSoloRampTime(rs []zsRamp) (solo, total int) {
	type ev struct {
		t, d int
	}
	evs := make([]ev, 0, 2*len(rs))
	for _, r := range rs {
		if r.tMax <= r.t0 {
			continue
		}
		evs = append(evs, ev{r.t0, +1}, ev{r.tMax, -1})
	}
	sort.Slice(evs, func(i, j int) bool {
		if evs[i].t != evs[j].t {
			return evs[i].t < evs[j].t
		}
		return evs[i].d > evs[j].d
	})
	depth, prev := 0, 0
	for _, e := range evs {
		if depth >= 1 {
			total += e.t - prev
			if depth == 1 {
				solo += e.t - prev
			}
		}
		depth += e.d
		prev = e.t
	}
	return solo, total
}
