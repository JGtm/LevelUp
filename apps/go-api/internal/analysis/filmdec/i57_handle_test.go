package filmdec

// i57_handle_test.go — INSTRUMENT DE MESURE de la PHASE C du plan
// .ai/V7.5/replay2d/PLAN_ETAT_ACTIF_EQUIPEMENT.md : les DÉPLOYABLES (mur rang 19,
// capteur rang 22 — relevé Theater du 27/07) — trois signaux datés à croiser.
//
// C.1 — publier la branche v==1 d'i57 par hook (spartanAbilityHook) : tag R(2), R(2)
// interne et R(24), par slot et horodatage — la SEULE branche qui paie 24 bits, jamais
// publiée. 75 occurrences attendues sur 000d5950 (14/08 : 0:693 · 1:75 · 2:613 · 3:33).
//
// C.2 — HYPOTHÈSE FALSIFIABLE, ÉNONCÉE AVANT LA MESURE : le R(24) RÉFÉRENCE quelque
// chose — candidat : l'entité ti=37 posée (handle slot 13 bits + génération 2 bits,
// la paire de l'en-tête des records d'objet du monde). Contrôle : croiser ses valeurs
// avec les vies ti=37 VIVANTES au même instant. S'il n'en croise aucune, l'hypothèse
// TOMBE et on le dit — les décompositions essayées et leurs taux sont tous publiés.
//
// C.3 — croiser TROIS horloges : naissances d'entités ti=37 (premier record delta d'une
// vie), transitions d'`equipment-activated` (balayage de production
// ScanFilmEquipmentState), lectures v==1 d'i57 des porteurs de rangs 19/22. Canal
// CLAIRSEMÉ : l'écart au plus proche voisin prime, les fenêtres (fines puis larges) sont
// jugées par témoins décalés ±5 s (décision n°2 du plan).
// C.4 — la FIN d'une vie ti=37 est-elle lisible ? Compter : durées, composants du DERNIER
// record (at-rest / dead-state résolus PAR NOM), vies finissant avant la fin du film.
//
// LECTURE SEULE, gardé par I57H_FILM, sauté partout ailleurs (CI comprise). UN SEUL
// décodage filmdec par process : le verrou est pris pour tout le test.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 I57H_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestI57HandleAndDeployables$' -timeout 60m -v

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

const i57hFilmEnv = "I57H_FILM"

// i57hSample est une lecture d'i57 par le déser de production, datée et attribuée.
type i57hSample struct {
	slot   uint32
	tsUS   uint64
	tag    uint32
	sub    uint32
	ref    uint32
	hasRef bool
}

// i57hCapture retient la DERNIÈRE publication du hook pendant la marche d'un record.
type i57hCapture struct {
	tag, sub, ref uint64
	hasRef, got   bool
}

func TestI57HandleAndDeployables(t *testing.T) {
	dir := os.Getenv(i57hFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", i57hFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	s := eaSetupBiped(t, dir)
	idx57 := s.arch.indicesOfFirst("biped-spartan-ability-component")
	if idx57 < 0 {
		t.Fatalf("biped-spartan-ability-component absent de l'archétype — composants : %v",
			s.arch.Components)
	}
	t.Logf("composant %d du registre biped : %q", idx57, s.arch.component(idx57))
	slotRanks := eaSlotRanks(t, dir)

	samples, st := i57hScan(t, s, idx57)
	t.Logf("C.1 — RECORDS %d (creux %d · dense %d) · masque∋i57 %d · marchés %d · cassés %d",
		st.sparse+st.dense, st.sparse, st.dense, st.with57, st.walked, st.broken)
	i57hLogValues(t, samples)

	lives := eaScanTi37Lives(t, dir)
	i57hControlC2(t, samples, lives, s)
	acts := i57hActivatedTransitions(t, dir)
	i57hCrossClocks(t, samples, lives, acts, slotRanks)
	i57hLogLifeEnds(t, dir, lives)
}

// i57hStats porte les dénominateurs du balayage.
type i57hStats struct {
	sparse, dense, with57, walked, broken int
}

// i57hScan balaye les records biped (masques creux ET denses — réutilise i57MatchDense de
// l'instrument du 14/08) et fait publier i57 par le déser de production.
func i57hScan(t *testing.T, s eaFilmSetup, idx57 int) ([]i57hSample, i57hStats) {
	t.Helper()
	var (
		capt    i57hCapture
		samples []i57hSample
		st      i57hStats
	)
	prev := spartanAbilityHook
	SetSpartanAbilityHook(func(tag, sub, ref uint64, hasRef bool) {
		capt.tag, capt.sub, capt.ref, capt.hasRef, capt.got = tag, sub, ref, hasRef, true
	})
	defer SetSpartanAbilityHook(prev)

	minRecord := bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + s.lay.TotalBits()
	for _, c := range s.chunks {
		data, err := ReadFilmChunk(s.dir, c)
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
				if i0, slot, idx, ok := matchBipedHeader(pay, p, total, s.slots, true, s.lay); ok {
					st.sparse++
					i57hAccount(pay, i0, total, idx, s, idx57, &capt, &st, &samples, slot, pk.TimestampUS)
					p = i0 + s.lay.TotalBits()
					continue
				}
				if i0, slot, idx, ok := i57MatchDense(pay, p, total, s.slots, s.lay, s.arch); ok {
					st.dense++
					i57hAccount(pay, i0, total, idx, s, idx57, &capt, &st, &samples, slot, pk.TimestampUS)
					p = i0 + s.lay.TotalBits()
					continue
				}
				p++
			}
		}
	}
	return samples, st
}

// i57hAccount marche le record jusqu'à i57 inclus (le déser publie via le hook) et impute.
// La validation du masque dense a pu déclencher le hook : la capture est réarmée AVANT la
// marche de mesure.
func i57hAccount(
	pay []byte, i0, total int, idx []int, s eaFilmSetup, idx57 int,
	capt *i57hCapture, st *i57hStats, samples *[]i57hSample, slot uint32, tsUS uint64,
) {
	if !eaMaskHas(idx, idx57) {
		return
	}
	st.with57++
	capt.got = false
	if eaWalkThrough(pay, i0, total, idx, s, idx57) && capt.got {
		st.walked++
		*samples = append(*samples, i57hSample{
			slot: slot, tsUS: tsUS, tag: uint32(capt.tag),
			sub: uint32(capt.sub), ref: uint32(capt.ref), hasRef: capt.hasRef,
		})
	} else {
		st.broken++
	}
}

// i57hLogValues publie la distribution des tags et, pour v==1, du R(2) interne et du R(24).
func i57hLogValues(t *testing.T, samples []i57hSample) {
	tags := map[uint32]int{}
	subs := map[uint32]int{}
	refs := map[uint32]int{}
	for _, sm := range samples {
		tags[sm.tag]++
		if sm.hasRef {
			subs[sm.sub]++
			refs[sm.ref]++
		}
	}
	t.Logf("  tags R(2) : %s", i48RenderU32(tags))
	t.Logf("  v==1 : R(2) interne : %s", i48RenderU32(subs))
	minV, maxV := ^uint32(0), uint32(0)
	for v := range refs {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}
	if len(refs) > 0 {
		maxMult, repeated := 0, 0
		for _, n := range refs {
			if n > maxMult {
				maxMult = n
			}
			if n > 1 {
				repeated++
			}
		}
		t.Logf("  v==1 : R(24) — %d valeurs distinctes · min %d (0x%06X) · max %d (0x%06X) · "+
			"0xFFFFFF : %d · 0 : %d", len(refs), minV, minV, maxV, maxV,
			refs[0xFFFFFF], refs[0])
		t.Logf("  v==1 : R(24) multiplicité — %d valeurs répétées · multiplicité max %d "+
			"(un handle re-référencé se répète ; des valeurs toutes uniques ressemblent à un "+
			"compteur/horodatage, pas à un handle)", repeated, maxMult)
		hist := map[int]int{}
		for v, n := range refs {
			hist[int(v)] += n
		}
		t.Logf("  v==1 : R(24) valeurs : %s", i28RenderCapped(hist, 40))
	}
}

// i57hControlC2 joue l'hypothèse C.2 sur chaque décomposition candidate du R(24), et
// publie leurs taux — y compris nuls.
func i57hControlC2(t *testing.T, samples []i57hSample, lives map[eaLifeKey]*eaLife, s eaFilmSetup) {
	n := CountFilmChunks(s.dir)
	ti37band := worldObjectSlotBandDir(s.dir, n, EquipmentTypeIndex)
	var withRef []i57hSample
	for _, sm := range samples {
		if sm.hasRef && sm.ref != 0xFFFFFF {
			withRef = append(withRef, sm)
		}
	}
	t.Logf("== CONTRÔLE C.2 — le R(24) référence-t-il l'entité ti=37 posée ? "+
		"(%d lectures v==1 hors sentinelle) ==", len(withRef))
	if len(withRef) == 0 {
		t.Log("  aucune lecture exploitable : hypothèse non testable sur ce film")
		return
	}
	type decomp struct {
		name string
		f    func(ref uint32) (slot, gen uint32)
	}
	decomps := []decomp{
		{"low13+gen2 (slot=ref&0x1FFF, gen=(ref>>13)&3)", func(r uint32) (uint32, uint32) {
			return r & 0x1FFF, (r >> 13) & 3
		}},
		{"gen2+slot13 (gen=ref&3, slot=(ref>>2)&0x1FFF)", func(r uint32) (uint32, uint32) {
			return (r >> 2) & 0x1FFF, r & 3
		}},
	}
	for _, d := range decomps {
		inBand, lifeExists, aliveAt := 0, 0, 0
		for _, sm := range withRef {
			slot, gen := d.f(sm.ref)
			if !ti37band[slot] {
				continue
			}
			inBand++
			l, ok := lives[eaLifeKey{slot, gen}]
			if !ok {
				continue
			}
			lifeExists++
			if sm.tsUS+2_000_000 >= l.firstUS && sm.tsUS <= l.lastUS+2_000_000 {
				aliveAt++
			}
		}
		t.Logf("  %s : slot∈bande ti37 %d/%d · vie (slot,gen) existe %d · vivante à ±2 s %d",
			d.name, inBand, len(withRef), lifeExists, aliveAt)
	}
	// Repli descriptif : le champ tombe-t-il dans la bande de slots BIPED (référence à un
	// autre joueur plutôt qu'à un objet) ?
	inBiped := 0
	for _, sm := range withRef {
		if s.slots[sm.ref&0x1FFF] {
			inBiped++
		}
	}
	t.Logf("  repli : low13 ∈ bande de slots BIPED %d/%d", inBiped, len(withRef))
	t.Log("RAPPEL C.2 : si aucune décomposition ne croise les vies ti=37 vivantes, " +
		"l'hypothèse « R(24) = handle ti=37 » TOMBE")
}

// i57hActEvent est une transition de valeur d'equipment-activated, datée.
type i57hActEvent struct {
	life     eaLifeKey
	tsUS     uint64
	from, to uint64
}

// i57hActivatedTransitions rejoue le balayage de PRODUCTION de ti=37 et rend les
// transitions d'`equipment-activated` par vie d'objet (le protocole du 15/08).
func i57hActivatedTransitions(t *testing.T, dir string) []i57hActEvent {
	t.Helper()
	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("découpage i0 illisible : %v", err)
	}
	prevPrec := WorldObjectPrecision
	t.Cleanup(func() { WorldObjectPrecision = prevPrec })
	SetWorldObjectPrecisionFromLayout(lay)
	samples, st, err := ScanFilmEquipmentState(dir)
	if err != nil {
		t.Fatalf("balayage equipment-state impossible : %v", err)
	}
	t.Logf("equipment-state (production) : records ti=37 %d · marchés %d · activated lu %d "+
		"(porte fermée %d)", st.Records, st.Walked, st.Read[EquipActivated], st.Gated[EquipActivated])
	series := map[eaLifeKey][]EquipmentStateSample{}
	for _, sm := range samples {
		if sm.Present[EquipActivated] {
			k := eaLifeKey{sm.Slot, sm.Gen}
			series[k] = append(series[k], sm)
		}
	}
	var out []i57hActEvent
	for k, ss := range series {
		sort.Slice(ss, func(a, b int) bool { return ss[a].TimestampUS < ss[b].TimestampUS })
		for i := 1; i < len(ss); i++ {
			if ss[i].Val[EquipActivated] != ss[i-1].Val[EquipActivated] {
				out = append(out, i57hActEvent{
					life: k, tsUS: ss[i].TimestampUS,
					from: ss[i-1].Val[EquipActivated], to: ss[i].Val[EquipActivated],
				})
			}
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].tsUS < out[b].tsUS })
	t.Logf("equipment-activated : %d transitions de valeur", len(out))
	return out
}

// i57hCrossClocks croise les trois horloges (item C.3) : écarts au plus proche voisin
// d'abord (canaux clairsemés), fenêtres larges et témoins décalés ensuite.
func i57hCrossClocks(
	t *testing.T, samples []i57hSample, lives map[eaLifeKey]*eaLife,
	acts []i57hActEvent, slotRanks map[uint32][]int,
) {
	var births, filmStart, filmEnd = []uint64{}, ^uint64(0), uint64(0)
	for _, l := range lives {
		births = append(births, l.firstUS)
		if l.firstUS < filmStart {
			filmStart = l.firstUS
		}
		if l.lastUS > filmEnd {
			filmEnd = l.lastUS
		}
	}
	births = eaSortedU64(births)
	// Les naissances ÉPHÉMÈRES : nées en cours de film (> 10 s après la première) ET de vie
	// courte (<= 90 s). Les objets de la carte naissent au début du film et vivent longtemps ;
	// un déployable naît au geste du joueur et expire. Sans ce filtre, la densité des
	// naissances (objets ramassables compris) rend toute co-datation vacueuse — c'est le
	// résultat de la première passe sur 000d5950, témoins décalés à 100 % eux aussi.
	var ephem []uint64
	for _, l := range lives {
		if l.firstUS > filmStart+10_000_000 && l.lastUS-l.firstUS <= 90_000_000 {
			ephem = append(ephem, l.firstUS)
		}
	}
	ephem = eaSortedU64(ephem)
	var actTs []uint64
	for _, a := range acts {
		actTs = append(actTs, a.tsUS)
	}
	actTs = eaSortedU64(actTs)

	var v1All, v1Deploy []uint64
	for _, sm := range samples {
		if sm.tag != 1 {
			continue
		}
		v1All = append(v1All, sm.tsUS)
		ranks := slotRanks[sm.slot]
		if eaHasRank(ranks, 19) || eaHasRank(ranks, 22) {
			v1Deploy = append(v1Deploy, sm.tsUS)
		}
	}
	v1All, v1Deploy = eaSortedU64(v1All), eaSortedU64(v1Deploy)

	durS := float64(filmEnd-filmStart) / 1e6
	rate := 0.0
	if durS > 0 {
		rate = float64(len(births)) / durS
	}
	t.Logf("== C.3 — TROIS HORLOGES : naissances ti=37 %d (%.2f/s — à ce débit le plus "+
		"proche voisin d'un instant QUELCONQUE est à ~%.0f ms : une fenêtre ne discrimine "+
		"rien, seuls les témoins décalés jugent) · dont ÉPHÉMÈRES %d · transitions "+
		"activated %d · i57 v==1 %d (dont porteurs 19/22 : %d) ==",
		len(births), rate, 1000*0.693/(2*maxF(rate, 0.001)), len(ephem), len(acts),
		len(v1All), len(v1Deploy))

	// Transition -> naissance de la MÊME vie : la jointure exacte, sans fenêtre.
	late, atBirth := 0, 0
	for _, a := range acts {
		l := lives[a.life]
		if l == nil {
			continue
		}
		if a.tsUS > l.firstUS {
			late++
		} else {
			atBirth++
		}
	}
	t.Logf("  transitions activated : %d APRÈS la naissance de leur vie · %d au premier record",
		late, atBirth)

	i57hPairReport(t, "i57 v==1 (porteurs 19/22) -> naissance ti=37 la plus proche", v1Deploy, births)
	i57hPairReport(t, "i57 v==1 (tous porteurs)  -> naissance ti=37 la plus proche", v1All, births)
	i57hPairReport(t, "i57 v==1 (porteurs 19/22) -> naissance ÉPHÉMÈRE la plus proche", v1Deploy, ephem)
	i57hPairReport(t, "i57 v==1 (tous porteurs)  -> naissance ÉPHÉMÈRE la plus proche", v1All, ephem)
	i57hPairReport(t, "i57 v==1 (porteurs 19/22) -> transition activated la plus proche", v1Deploy, actTs)
	i57hPairReport(t, "transitions activated      -> i57 v==1 le plus proche", actTs, v1All)
}

// maxF évite une division par zéro dans le journal du débit.
func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// i57hPairReport publie la distribution des écarts au plus proche voisin, puis les
// fenêtres larges avec témoins décalés ±5 s.
func i57hPairReport(t *testing.T, label string, events, sorted []uint64) {
	if len(events) == 0 || len(sorted) == 0 {
		t.Logf("  %s : non calculable (%d événements, %d cibles)", label, len(events), len(sorted))
		return
	}
	var deltas []int64
	for _, e := range events {
		if d, ok := eaNearestDelta(e, sorted); ok {
			deltas = append(deltas, d)
		}
	}
	sort.Slice(deltas, func(a, b int) bool { return abs64(deltas[a]) < abs64(deltas[b]) })
	med := deltas[len(deltas)/2]
	shown := deltas
	if len(shown) > 12 {
		shown = shown[:12]
	}
	human := make([]string, 0, len(shown))
	for _, d := range shown {
		human = append(human, fmt.Sprintf("%+.2fs", float64(d)/1e6))
	}
	t.Logf("  %s : n=%d · |écart| médian %.2f s · plus proches : %s",
		label, len(deltas), float64(abs64(med))/1e6, strings.Join(human, " "))
	// Fenêtres FINES d'abord : à ~5 cibles/s, la discrimination vit à l'échelle du paquet
	// (20-100 ms) — les fenêtres de l'ordre de la seconde saturent, témoins compris.
	for _, win := range []uint64{20_000, 50_000, 100_000, 500_000, 1_000_000, 5_000_000} {
		real := eaCountWithin(events, sorted, win, 0)
		plus := eaCountWithin(events, sorted, win, 5_000_000)
		minus := eaCountWithin(events, sorted, win, -5_000_000)
		t.Logf("    fenêtre ±%.2f s : réel %d/%d (%.1f %%) · témoin +5 s %d · témoin -5 s %d",
			float64(win)/1e6, real, len(events), 100*float64(real)/float64(len(events)), plus, minus)
	}
}

// i57hLogLifeEnds répond à l'item C.4 : les vies ti=37 ont-elles une FIN lisible ?
func i57hLogLifeEnds(t *testing.T, dir string, lives map[eaLifeKey]*eaLife) {
	arch, err := EquipmentArchetype(dir)
	if err != nil {
		t.Fatalf("archétype ti=37 illisible : %v", err)
	}
	endIdx := map[int]string{}
	for i, name := range arch.Components {
		if strings.Contains(name, "at-rest") || strings.Contains(name, "dead-state") {
			endIdx[i] = name
		}
	}
	t.Logf("== C.4 — FINS DE VIE ti=37 (%d vies) · composants de fin candidats : %v ==",
		len(lives), endIdx)
	if len(lives) == 0 {
		return
	}
	var filmEnd uint64
	var durs []float64
	for _, l := range lives {
		if l.lastUS > filmEnd {
			filmEnd = l.lastUS
		}
	}
	endsBefore, endsAtFilmEnd := 0, 0
	lastHasEndComp := map[string]int{}
	for _, l := range lives {
		durs = append(durs, float64(l.lastUS-l.firstUS)/1e6)
		if l.lastUS+5_000_000 < filmEnd {
			endsBefore++
		} else {
			endsAtFilmEnd++
		}
		for _, id := range l.lastIdx {
			if name, ok := endIdx[id]; ok {
				lastHasEndComp[name]++
			}
		}
	}
	sort.Float64s(durs)
	t.Logf("  durées de vie : médiane %.1f s · p90 %.1f s · max %.1f s",
		durs[len(durs)/2], durs[len(durs)*9/10], durs[len(durs)-1])
	t.Logf("  %d vies finissent > 5 s avant la fin du film (fin RÉELLE) · %d tiennent "+
		"jusqu'à la fin", endsBefore, endsAtFilmEnd)
	if len(lastHasEndComp) == 0 {
		t.Log("  AUCUN dernier record ne porte un composant at-rest/dead-state")
	}
	for name, n := range lastHasEndComp {
		t.Logf("  dernier record porteur de %q : %d vies", name, n)
	}
	t.Log("RAPPEL C.4 : sans composant de fin, la « fin » est la DISPARITION du masque — " +
		"datable par le dernier record, sans cause lisible")
}
