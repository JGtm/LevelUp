package filmdec

// zone_state_measure_test.go — LA MESURE D'ETAT du lot C (items C.0.2 et C.1b.3), contre le gate
// G-C1 ecrit AVANT dans `LOTC_ARBITRAGE_PHASE0.md`.
//
// CE QUI CHANGE PAR RAPPORT A LA PHASE 0. La phase 0 ne pouvait compter que des ANNONCES : les
// composants n'etaient pas portes, donc leurs valeurs etaient hors d'atteinte. Ils le sont
// maintenant (item C.1b.1), et cet instrument lit les VALEURS.
//
// COMMENT IL LES LIT, ET POURQUOI PAS PAR LA TRAVERSEE CANONIQUE. La traversee sequentielle
// s'arrete au premier composant non porte du record ; ti=10 en compte encore 26. On garde donc
// l'ancrage par bande de slots de la phase 0 (`matchWorldObjectRecord`), puis on rejoue sur la
// charge utile la BOUCLE DE PRODUCTION composant par composant (`consumeByName`), dans l'ordre du
// masque, en s'arretant au premier non porte. Les valeurs viennent ainsi du code de production
// porte, pas d'une relecture parallele : ce que cet instrument mesure est exactement ce que la
// phase 2 publierait.
//
// LES TEMOINS QUE LE REGISTRE EXIGE DESORMAIS sont publies a chaque passe : rapport reel/fantome
// de la bande, et purete de `ti=4` (un slot, un composant declare : toute annonce hors i0 est une
// faute d'ancrage).
//
// USAGE (depuis apps/go-api) :
//
//	$env:CGO_ENABLED=0
//	$env:ZONE_FILM="C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/7344d24f"
//	$env:ZONE_OBJTYPE="zone"
//	go test -count=1 -run TestZoneStateLotC1b -v ./internal/analysis/filmdec/

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// zsSample est une valeur publiee par un hook, datee et attribuee a son slot.
type zsSample struct {
	tMS  int
	slot uint32
	vals []uint64
}

// zsCollect porte tout ce qu'une passe a recolte, par champ.
type zsCollect struct {
	radial  []zsSample // ti=12 i14
	color   []zsSample // ti=10 i1
	rtpc    []zsSample // ti=10 i26..i29
	visib   []zsSample // ti=10 i0
	records int
	walked  int // records dont TOUS les composants annonces ont ete consommes
	// temoins d'ancrage
	ghostRecords, ti4Records, ti4OutOfGrammar int
}

// TestZoneStateLotC1b joue les mesures C.0.2 et C.1b.3 sur UN film.
func TestZoneStateLotC1b(t *testing.T) {
	dir := zcDir(t)
	out := zcOutDir(t)
	short := filepath.Base(dir)
	release := LockProcessDecode()
	defer release()

	gram, reg := zcLoadGrammar(t, dir)
	c := zcKeyframeCensus(dir)
	bands := zcBuildBands(c)
	oracle := zcLoadOracle(t, dir)
	clk := zcLoadClock(t, dir)

	col := zsScan(t, c, bands, reg, clk)
	t.Logf("FILM %s — %d records ancres, %d entierement consommes · ORACLE %q : %d evenements"+
		" d'objectif", short, col.records, col.walked, oracle.family, len(oracle.times))
	t.Logf("  TEMOINS D'ANCRAGE : bande fantome (vide) %d records · purete ti=4 %d records dont"+
		" %.2f %% hors grammaire", col.ghostRecords, col.ti4Records,
		100*zcRate(col.ti4OutOfGrammar, col.ti4Records))
	t.Logf("  VALEURS RECOLTEES : radial-progress %d · boundary-color %d · rtpc %d ·"+
		" boundary-visibility %d", len(col.radial), len(col.color), len(col.rtpc), len(col.visib))

	var sb strings.Builder
	zsReportVisibility(t, &sb, col, oracle) // C.0.2
	zsReportRadial(t, &sb, col, oracle)     // C.1b.3 (a)
	zsReportColor(t, &sb, col, oracle)      // C.1b.3 (b)
	zsReportRTPC(t, &sb, col)               // C.1b.3 (c)
	_ = gram
	zcWriteFile(t, filepath.Join(out, short+"_etat_zones.tsv"), sb.String())
}

// zsScan ancre les records puis rejoue la boucle de production sur leur charge utile.
func zsScan(t *testing.T, c zcCensus, b zcBands, reg *Registry, clk zcClock) *zsCollect {
	t.Helper()
	col := &zsCollect{}
	var curT int
	var curSlot uint32
	prevM, prevN := managedObjectHook, navpointHook
	SetManagedObjectHook(func(f ManagedObjectField, values []uint64) {
		s := zsSample{tMS: curT, slot: curSlot, vals: append([]uint64(nil), values...)}
		switch f {
		case ManagedObjectBoundaryVisibility:
			col.visib = append(col.visib, s)
		case ManagedObjectBoundaryColor:
			col.color = append(col.color, s)
		case ManagedObjectRTPC:
			col.rtpc = append(col.rtpc, s)
		}
	})
	SetNavpointHook(func(f NavpointField, values []uint64) {
		if f == NavpointRadialProgress {
			col.radial = append(col.radial, zsSample{tMS: curT, slot: curSlot,
				vals: append([]uint64(nil), values...)})
		}
	})
	defer func() { SetManagedObjectHook(prevM); SetNavpointHook(prevN) }()

	for ch := 1; ch <= c.chunks; ch++ {
		data, err := ReadFilmChunk(c.dir, ch)
		if err != nil {
			continue
		}
		startMS, hasStart := clk.startMS[ch]
		var base uint64
		haveBase := false
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			if !haveBase {
				base, haveBase = pk.TimestampUS, true
			}
			if !hasStart {
				continue
			}
			curT = startMS + int((pk.TimestampUS-base)/1000)
			zsScanPayload(pk.Payload(data), b, reg, col, &curSlot)
		}
	}
	return col
}

// zsScanPayload ancre les records d'un payload et rejoue leurs composants annonces.
func zsScanPayload(pay []byte, b zcBands, reg *Registry, col *zsCollect, curSlot *uint32) {
	limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits)
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, b.all)
		if !ok {
			continue
		}
		p = rec.After
		cl := b.class[rec.Slot]
		switch {
		case cl == zcClassVide:
			col.ghostRecords++
			continue
		case cl < 0:
			continue
		case cl == 4:
			col.ti4Records++
			for _, i := range rec.Idx {
				if i != 0 {
					col.ti4OutOfGrammar++
					break
				}
			}
			continue
		case cl != 10 && cl != 12:
			continue
		}
		arch, ok := reg.Archetype(cl)
		if !ok {
			continue
		}
		col.records++
		*curSlot = rec.Slot
		if zsReplay(pay, rec, arch, uint32(cl)) {
			col.walked++
		}
	}
}

// zsReplay rejoue les composants annonces d'un record par la boucle de PRODUCTION, dans l'ordre
// du masque, et rend vrai si tous ont ete consommes. Les hooks publient au passage.
func zsReplay(pay []byte, rec WorldObjectRecord, arch Archetype, ti uint32) bool {
	br := NewBitReader(pay)
	br.SetBitPos(rec.After)
	for _, i := range rec.Idx {
		if i < 0 || i >= len(arch.Components) {
			return false
		}
		if _, _, ported := consumeByName(br, arch.Components[i], ti, arch.Level(i)); !ported {
			return false
		}
	}
	return true
}

// -----------------------------------------------------------------------------------------
// C.0.2 — les 32 drapeaux de ti=10 i0
// -----------------------------------------------------------------------------------------

// zsReportVisibility publie la distribution des 32 drapeaux et leurs transitions datees.
func zsReportVisibility(t *testing.T, sb *strings.Builder, col *zsCollect, o zcOracle) {
	t.Helper()
	if len(col.visib) == 0 {
		t.Logf("C.0.2 — aucun record de boundary-visibility recolte")
		return
	}
	distinct := map[uint64]int{}
	perBit := [32]int{}
	for _, s := range col.visib {
		v := s.vals[0]
		distinct[v]++
		for k := 0; k < 32; k++ {
			if v&(1<<uint(k)) != 0 {
				perBit[k]++
			}
		}
	}
	t.Logf("C.0.2 — boundary-visibility (ti=10 i0) : %d records, %d valeurs distinctes",
		len(col.visib), len(distinct))
	var bits strings.Builder
	used := 0
	for k := 0; k < 32; k++ {
		if perBit[k] == 0 {
			continue
		}
		used++
		fmt.Fprintf(&bits, "b%d:%.1f%% ", k, 100*float64(perBit[k])/float64(len(col.visib)))
	}
	t.Logf("  bits JAMAIS leves : %d/32 · bits utilises : %d — %s", 32-used, used, bits.String())
	t.Logf("  les 8 valeurs les plus frequentes : %s", zsTopValues(distinct, len(col.visib), 8))
	// Hypotheses a departager : 8 bits utiles = par joueur ; 2 = par equipe ; peu de valeurs
	// distinctes = par etat.
	switch {
	case used <= 4:
		t.Logf("  LECTURE : %d bits utiles seulement — compatible avec un ETAT (ou une equipe),"+
			" PAS avec un drapeau par joueur", used)
	case used <= 8:
		t.Logf("  LECTURE : %d bits utiles — compatible avec UN DRAPEAU PAR JOUEUR (8 joueurs)", used)
	default:
		t.Logf("  LECTURE : %d bits utiles — ni par joueur (8) ni par equipe (2) ; a expliquer", used)
	}
	trans := zsTransitions(col.visib)
	t.Logf("  transitions (changement de valeur sur un meme slot) : %d", len(trans))
	t.Logf("  %s", zsNearEvents("transitions de visibilite", trans, o.times))
	fmt.Fprintf(sb, "# C.0.2 boundary-visibility : %d records, %d valeurs distinctes, %d bits utiles, %d transitions\n",
		len(col.visib), len(distinct), used, len(trans))
}

// -----------------------------------------------------------------------------------------
// C.1b.3 (a) — radial-progress : les rampes
// -----------------------------------------------------------------------------------------

// zsRamp est une rampe monotone croissante detectee sur un slot : ses bornes et son sommet.
type zsRamp struct {
	slot         uint32
	t0, tMax     int
	qStart, qMax uint64
	samples      int
}

// zsReportRadial detecte les rampes et confronte leurs sommets aux evenements d'objectif.
func zsReportRadial(t *testing.T, sb *strings.Builder, col *zsCollect, o zcOracle) {
	t.Helper()
	if len(col.radial) == 0 {
		t.Logf("C.1b.3 (a) — aucun record de radial-progress recolte")
		return
	}
	bySlot := map[uint32][]zsSample{}
	for _, s := range col.radial {
		bySlot[s.slot] = append(bySlot[s.slot], s)
	}
	var ramps []zsRamp
	for slot, ss := range bySlot {
		sort.SliceStable(ss, func(i, j int) bool { return ss[i].tMS < ss[j].tMS })
		ramps = append(ramps, zsFindRamps(slot, ss)...)
	}
	sort.SliceStable(ramps, func(i, j int) bool { return ramps[i].tMax < ramps[j].tMax })
	t.Logf("C.1b.3 (a) — radial-progress : %d valeurs sur %d slots · %d rampes monotones"+
		" (>= %d echantillons, amplitude >= %d quanta)",
		len(col.radial), len(bySlot), len(ramps), zsRampMinSamples, zsRampMinAmplitude)
	if len(ramps) == 0 {
		return
	}
	tops := make([]int, 0, len(ramps))
	for _, r := range ramps {
		tops = append(tops, r.tMax)
	}
	t.Logf("  amplitude des rampes : %s", zsRampStats(ramps))
	t.Logf("  %s", zsNearEvents("sommets de rampe", tops, o.times))
	// Le sens du gate : chaque CAPTURE doit etre precedee d'une rampe qui atteint son max dans
	// [t-2 s ; t+2 s]. On mesure donc dans l'autre sens aussi : part des captures couvertes.
	cov, den := zsCoverage(o.times, tops, zsGateWindowMS)
	shift := make([]int, len(o.times))
	for i, tt := range o.times {
		shift[i] = tt + zsWitnessShiftMS
	}
	covW, _ := zsCoverage(shift, tops, zsGateWindowMS)
	t.Logf("  GATE G-C1 (a) : %d/%d captures (%.1f %%) ont un sommet de rampe dans +/- %d ms"+
		" — seuil 80 %% ; TEMOIN (memes captures decalees de +%d ms) : %.1f %% — seuil <= 20 %%",
		cov, den, 100*zcRate(cov, den), zsGateWindowMS, zsWitnessShiftMS, 100*zcRate(covW, den))
	// NIVEAU DU HASARD, publie parce que sans lui le temoin ne se juge pas : avec N rampes et
	// une fenetre de +/- w sur une duree T, une capture tombe pres d un sommet par pur hasard
	// avec une probabilite d environ N x 2w / T.
	span := zsSpan(tops)
	hasard := 0.0
	if span > 0 {
		hasard = float64(len(tops)) * float64(2*zsGateWindowMS) / float64(span)
		if hasard > 1 {
			hasard = 1
		}
	}
	t.Logf("  NIVEAU DU HASARD pour ce canal : %.1f %% (%d sommets, fenetre +/- %d ms, duree %d ms)",
		100*hasard, len(tops), zsGateWindowMS, span)
	uniq, tot := zsSoloRampTime(ramps)
	t.Logf("  CONCURRENCE (clause KOTH) : %.1f %% du temps couvert par une rampe n en porte QU UNE"+
		" seule (%d ms sur %d) — seuil 90 %%", 100*zcRate(uniq, tot), uniq, tot)
	verdict := "NON TENU"
	if den > 0 && zcRate(cov, den) >= 0.8 && zcRate(covW, den) <= 0.2 {
		verdict = "TENU"
	}
	t.Logf("  VERDICT G-C1 (a) : %s", verdict)
	fmt.Fprintf(sb, "# C.1b.3a radial-progress : %d valeurs, %d slots, %d rampes, captures couvertes %d/%d (%.1f %%), temoin %.1f %%, verdict %s\n",
		len(col.radial), len(bySlot), len(ramps), cov, den, 100*zcRate(cov, den), 100*zcRate(covW, den), verdict)
	for _, r := range ramps {
		fmt.Fprintf(sb, "rampe\t%d\t%d\t%d\t%d\t%d\t%d\n", r.slot, r.t0, r.tMax, r.qStart, r.qMax, r.samples)
	}
}

// zsRampMinSamples / zsRampMinAmplitude bornent ce qui compte comme une rampe. Ecrits avant la
// mesure : trois echantillons croissants et 16 quanta d'amplitude (6 % de la plage) ecartent le
// bruit d'un canal qui oscille d'un quantum.
const (
	zsRampMinSamples   = 3
	zsRampMinAmplitude = 16
	// zsGateWindowMS est la fenetre du gate G-C1 (+/- 2 s), zsWitnessShiftMS le decalage du
	// temoin (+20 s). Les deux viennent de l'arbitrage, ils ne sont pas ajustables ici.
	zsGateWindowMS   = 2000
	zsWitnessShiftMS = 20000
)

// zsFindRamps decoupe une serie chronologique en montees monotones.
func zsFindRamps(slot uint32, ss []zsSample) []zsRamp {
	var out []zsRamp
	i := 0
	for i < len(ss) {
		j := i
		for j+1 < len(ss) && ss[j+1].vals[0] >= ss[j].vals[0] {
			j++
		}
		n := j - i + 1
		amp := ss[j].vals[0] - ss[i].vals[0]
		if n >= zsRampMinSamples && amp >= zsRampMinAmplitude {
			out = append(out, zsRamp{slot: slot, t0: ss[i].tMS, tMax: ss[j].tMS,
				qStart: ss[i].vals[0], qMax: ss[j].vals[0], samples: n})
		}
		if j == i {
			i++
			continue
		}
		i = j + 1
	}
	return out
}

// zsRampStats resume les amplitudes.
func zsRampStats(rs []zsRamp) string {
	amps := make([]int, 0, len(rs))
	for _, r := range rs {
		amps = append(amps, int(r.qMax-r.qStart))
	}
	sort.Ints(amps)
	return fmt.Sprintf("min %d · mediane %d · max %d (quanta sur 256)",
		amps[0], amps[len(amps)/2], amps[len(amps)-1])
}
