package filmdec

// ti11_delta_test.go — INSTRUMENT DU LOT ti11-DELTA (cf. protocole
// .ai/V7.5/replay2d/registre_film/TI11_DELTA_PROTOCOLE.md).
//
// LA QUESTION, UNE SEULE : la grammaire ti=11 34-feuilles VALIDEE en keyframe (atterrissage
// 90,32 % sous le cadre C5) fait-elle PARSER les records ti=11 du flux DELTA, et les champs
// VIVANTS (i3 object-reference, i1 couleur, i12/i13 progression, i16-31 sous-entites) y sont-ils
// PEUPLES (non-sentinelle) et EVOLUENT-ils dans le temps ? Le keyframe ne portait que l'ETAT PAR
// DEFAUT (T2 du lot precedent : i3=null, i12/i13=0, i1=gris, i16-31=null).
//
// LE CADRE DELTA N'EST PAS C5. C5 est le cadre de l'IMAGE-CLE (etat complet, tous les composants
// itere dans l'ordre). Le DELTA est le record d'OBJET DU MONDE masque : matchWorldObjectRecord
// (prefixe(1)+slot(13)+gen(2)+porte(2==0)+nb(3)+indices croissants(6b)) puis marche des SEULS
// composants du masque via consumeByName. Ce qui est PARTAGE et valide en T1, c'est la GRAMMAIRE
// (les desers de consumeByName). Ce chemin est DEJA en production pour le frere ti=13
// (zone_state_scan.go) : cet instrument le transpose a ti=11, en MESURE. Le chainage de ti=13
// (TestTI11DeltaSiblingRef) sert d'ETALON du cadre sur le MEME corpus (haut LA OU le mode
// replique les zones : 61 % KOTH ; au bruit ~4 % sur Oddball/CTF, modes sans zones).
//
// LECTURE SEULE, garde TI11_ROOT : saute partout ailleurs (CI comprise). UN SEUL decodage filmdec
// par process (bascules globales) : verrou pour tout le test. Un film charge a la fois.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 TI11_ROOT=<repo>/data/cache/film_chunks \
//	  go test ./internal/analysis/filmdec/ -run '^TestTI11Delta' -timeout 30m -v

import (
	"testing"
)

// Sentinelles de l'ETAT PAR DEFAUT (mesurees en keyframe, T2 du lot cadre) : ce qui EST peuple
// dans le delta et DIFFERE de ces valeurs = le vivant.
const (
	ti11SentinelRef   = 0xFFFFFFFF // i3 / i15 / i16-31 : GlobalID null
	ti11SentinelColor = 128        // i1 : gris (R=G=B=128) en etat par defaut
)

// ti11DeltaStat agrege un balayage delta d'un film (ou d'une bande de controle).
type ti11DeltaStat struct {
	records, walked, broken, chained, outOfGrammar int
	slots                                          map[uint32]bool
}

func newTI11DeltaStat() ti11DeltaStat { return ti11DeltaStat{slots: map[uint32]bool{}} }

// ti11RecFields porte les champs moissonnes par le hook pour LE record en cours de marche.
type ti11RecFields struct {
	i3       uint64
	i3Set    bool
	color    [4]uint64
	colorSet bool
	prog     uint64
	progSet  bool
	req      uint64
	reqSet   bool
	state    uint64
	stateSet bool
	typ      uint64
	typSet   bool
	subs     []uint64
}

// ti11Living accumule, par (Gen,Slot) et par champ, la SUITE TEMPORELLE des valeurs vivantes.
// C'est ce qui repond a « le champ evolue-t-il ? ».
type ti11Living struct {
	i3     map[[2]uint32][]uint64 // suite des i3 dans le temps
	color  map[[2]uint32][]uint64 // suite des R (canal 0) dans le temps (proxy de camp)
	prog   map[[2]uint32][]uint64 // suite des i12 dans le temps
	subs   map[[2]uint32][]uint64 // union des sous-entites vues (ensemble, pas suite)
	obsRef map[uint64]bool        // ensemble des i3/subs non-sentinelle (temoin aleatoire)
}

func newTI11Living() ti11Living {
	return ti11Living{
		i3: map[[2]uint32][]uint64{}, color: map[[2]uint32][]uint64{},
		prog: map[[2]uint32][]uint64{}, subs: map[[2]uint32][]uint64{},
		obsRef: map[uint64]bool{},
	}
}

// ti11InstallDeltaHook installe le hook nomme de ti=11 qui deverse dans `cur`, et rend sa
// restauration. cur.subs est remis a zero par l'appelant a chaque record.
func ti11InstallDeltaHook(cur *ti11RecFields) func() {
	SetManagedObjectiveHook(func(f ManagedObjectiveField, values []uint64) {
		if len(values) == 0 {
			return
		}
		switch f {
		case ManagedObjectiveObjectRef:
			cur.i3, cur.i3Set = values[0], true
		case ManagedObjectiveColor:
			if len(values) >= 4 {
				cur.color = [4]uint64{values[0], values[1], values[2], values[3]}
				cur.colorSet = true
			}
		case ManagedObjectiveProgress:
			cur.prog, cur.progSet = values[0], true
		case ManagedObjectiveRequired:
			cur.req, cur.reqSet = values[0], true
		case ManagedObjectiveState:
			cur.state, cur.stateSet = values[0], true
		case ManagedObjectiveType:
			cur.typ, cur.typSet = values[0], true
		case ManagedObjectiveSubEntity:
			cur.subs = append(cur.subs, values[0])
		}
	})
	return func() { SetManagedObjectiveHook(nil) }
}

// ti11DeltaWalkRecord marche le masque d'un record delta avec les desers DE PRODUCTION
// (consumeByName, typeIndex=11). Rend la position de fin et l'aboutissement. Meme forme que
// zone_state_scan.go:walk (le scanner ti=13 en prod).
func ti11DeltaWalkRecord(pay []byte, rec WorldObjectRecord, arch Archetype) (int, bool) {
	total := len(pay) * 8
	at := rec.After
	for _, id := range rec.Idx {
		name := arch.component(id)
		if name == "" || at > total { // indice hors grammaire (>33) ou debordement
			return at, false
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		_, _, ported := consumeByName(br, name, uint32(ti11TypeIndex), arch.Level(id))
		if !ported || br.BitPos() > total {
			return at, false
		}
		at = br.BitPos()
	}
	return at, true
}

// ti11ScanDelta balaie tous les paquets delta du film `dir` sur la bande `band`, marche les
// records ti=11, moissonne les champs vivants (dans `lv`, si non nil), et rend les statistiques.
// HORS LIGNE (I/O disque sur tout le film). L'appelant detient LockProcessDecode.
func ti11ScanDelta(dir string, n int, band map[uint32]bool, arch Archetype, lv *ti11Living) ti11DeltaStat {
	s := newTI11DeltaStat()
	if len(band) == 0 {
		return s
	}
	var cur ti11RecFields
	restore := ti11InstallDeltaHook(&cur)
	defer restore()

	for ch := 1; ch <= n; ch++ {
		data, err := ReadFilmChunk(dir, ch)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits)
			for p := 0; p <= limit; p++ {
				rec, ok := matchWorldObjectRecord(pay, p, band)
				if !ok {
					continue
				}
				s.records++
				s.slots[rec.Slot] = true
				for _, i := range rec.Idx {
					if i > ti11MaxComponentIndex {
						s.outOfGrammar++
						break
					}
				}
				cur = ti11RecFields{}
				end, done := ti11DeltaWalkRecord(pay, rec, arch)
				if !done {
					s.broken++
					p = rec.After
					continue
				}
				s.walked++
				if worldObjectHeaderAt(pay, end) {
					s.chained++
				}
				if lv != nil {
					ti11Harvest(lv, rec, &cur)
				}
				p = rec.After // meme convention que zone_state_scan.go (comparable a R4)
			}
		}
	}
	return s
}

// ti11Harvest deverse les champs du record courant dans les suites temporelles par (Gen,Slot).
func ti11Harvest(lv *ti11Living, rec WorldObjectRecord, cur *ti11RecFields) {
	key := [2]uint32{rec.Gen, rec.Slot}
	if cur.i3Set {
		lv.i3[key] = append(lv.i3[key], cur.i3)
		if cur.i3 != ti11SentinelRef {
			lv.obsRef[cur.i3] = true
		}
	}
	if cur.colorSet {
		lv.color[key] = append(lv.color[key], cur.color[0]) // canal R comme proxy de camp
	}
	if cur.progSet {
		lv.prog[key] = append(lv.prog[key], cur.prog)
	}
	for _, sv := range cur.subs {
		lv.subs[key] = append(lv.subs[key], sv)
		if sv != ti11SentinelRef {
			lv.obsRef[sv] = true
		}
	}
}

// ti11EvolveStat resume un champ vivant : records porteurs, valeurs distinctes, distinctes
// NON-sentinelle, et nombre de (Gen,Slot) dont la SUITE porte >= 2 valeurs distinctes (evolue).
type ti11EvolveStat struct {
	present     int
	distinct    map[uint64]int
	nonSentinel int
	evolving    int // (Gen,Slot) avec >= 2 valeurs distinctes dans le temps
	slots       int
	maxSeq      int
}

// ti11EvalField calcule l'evolution d'un champ. `isSentinel` distingue le vivant du defaut.
func ti11EvalField(series map[[2]uint32][]uint64, isSentinel func(uint64) bool) ti11EvolveStat {
	es := ti11EvolveStat{distinct: map[uint64]int{}, slots: len(series)}
	for _, seq := range series {
		if len(seq) > es.maxSeq {
			es.maxSeq = len(seq)
		}
		local := map[uint64]bool{}
		for _, v := range seq {
			es.present++
			es.distinct[v]++
			local[v] = true
			if !isSentinel(v) {
				es.nonSentinel++
			}
		}
		if len(local) >= 2 {
			es.evolving++
		}
	}
	return es
}

// TestTI11DeltaParse — GATE PARSE (D1, volet 1). Applique la grammaire 34-feuilles aux records
// delta ti=11 : taux de marche (walked) et de chainage (chained, temoin de largeur comme ti=13),
// part hors-grammaire (baseline R4 45,9 %), et TEMOIN bande fantome (doit s'effondrer).
func TestTI11DeltaParse(t *testing.T) {
	root := ti11Root(t)
	release := LockProcessDecode()
	defer release()

	var cumRec, cumWalk, cumChain, cumOOG int
	for _, name := range ti11FilmNames() {
		dir := root + "/" + name
		n := CountFilmChunks(dir)
		if n == 0 {
			t.Errorf("film %s : aucun chunk", name)
			continue
		}
		raw, err := ReadFilmChunk(dir, 0)
		if err != nil {
			t.Errorf("film %s : registre illisible : %v", name, err)
			continue
		}
		reg, err := ParseRegistryChunk(raw)
		if err != nil {
			t.Errorf("film %s : registre : %v", name, err)
			continue
		}
		arch, ok := reg.Archetype(ti11TypeIndex)
		if !ok {
			t.Errorf("film %s : archetype ti=11 absent", name)
			continue
		}
		census := ti11KeyframeCensus(dir)
		observed := map[uint32]bool{}
		for s := range census.slotsTI[ti11TypeIndex] {
			observed[s] = true
		}
		if len(observed) == 0 {
			t.Logf("  [%s] AUCUN slot ti=11 observe en keyframe — pas de bande delta.", name)
			continue
		}
		real := ti11ScanDelta(dir, n, observed, arch, nil)
		ghost := ti11GhostBand(census, len(observed))
		fake := ti11ScanDelta(dir, n, ghost, arch, nil)

		pct := func(a, b int) float64 {
			if b == 0 {
				return 0
			}
			return 100 * float64(a) / float64(b)
		}
		t.Logf("  [%s] bande OBSERVEE %d slots · records %d · walked %d (%.1f %%) · chained %d (%.1f %% des walked) · broken %d · hors-grammaire %.1f %%",
			name, len(observed), real.records, real.walked, pct(real.walked, real.records),
			real.chained, pct(real.chained, real.walked), real.broken, pct(real.outOfGrammar, real.records))
		t.Logf("      TEMOIN fantome %d slots · records %d · walked %d (%.1f %%) · chained %d",
			len(ghost), fake.records, fake.walked, pct(fake.walked, fake.records), fake.chained)
		cumRec += real.records
		cumWalk += real.walked
		cumChain += real.chained
		cumOOG += real.outOfGrammar
	}
	pct := func(a, b int) float64 {
		if b == 0 {
			return 0
		}
		return 100 * float64(a) / float64(b)
	}
	t.Logf("======== CUMUL PARSE (%d films) ========", len(ti11FilmNames()))
	t.Logf("  records %d · walked %d (%.1f %%) · chained %d (%.1f %% des records, %.1f %% des walked) · hors-grammaire %.1f %%",
		cumRec, cumWalk, pct(cumWalk, cumRec), cumChain, pct(cumChain, cumRec), pct(cumChain, cumWalk), pct(cumOOG, cumRec))
	t.Logf("  GATE D1 volet parse : >= 80 %% des records parsent proprement (chained est le temoin de largeur)")
}

// TestTI11DeltaLiving — GATE PARSE (D1, volet 2). LES CHAMPS VIVENT-ILS ? Pour chaque champ
// (i3 porteur, i1 couleur, i12 progression, i16-31 sous-entites), publie : records porteurs,
// valeurs distinctes, distinctes NON-sentinelle, et nombre de (Gen,Slot) dont la valeur EVOLUE
// dans le temps. TEMOIN aleatoire : 4096 GlobalID au hasard ne tombent pas dans l'observe.
func TestTI11DeltaLiving(t *testing.T) {
	root := ti11Root(t)
	release := LockProcessDecode()
	defer release()

	lv := newTI11Living()
	ghostLv := newTI11Living()
	var cumWalk, cumGhostWalk int
	for _, name := range ti11FilmNames() {
		dir := root + "/" + name
		n := CountFilmChunks(dir)
		if n == 0 {
			continue
		}
		raw, err := ReadFilmChunk(dir, 0)
		if err != nil {
			continue
		}
		reg, err := ParseRegistryChunk(raw)
		if err != nil {
			continue
		}
		arch, ok := reg.Archetype(ti11TypeIndex)
		if !ok {
			continue
		}
		census := ti11KeyframeCensus(dir)
		observed := map[uint32]bool{}
		for s := range census.slotsTI[ti11TypeIndex] {
			observed[s] = true
		}
		if len(observed) == 0 {
			continue
		}
		filmLv := newTI11Living()
		st := ti11ScanDelta(dir, n, observed, arch, &filmLv)
		cumWalk += st.walked
		// CONTROLE : la MEME moisson sur une bande FANTOME (slots jamais ti=11). Si le fantome
		// parait aussi « vivant », le vivant observe est un artefact de lecture de bits arbitraires.
		ghost := ti11GhostBand(census, len(observed))
		filmGhost := newTI11Living()
		gs := ti11ScanDelta(dir, n, ghost, arch, &filmGhost)
		cumGhostWalk += gs.walked
		// Bilan par film (les suites du film, puis fusion dans le cumul).
		ti11LogLiving(t, name, st.walked, filmLv)
		ti11MergeLiving(&lv, filmLv)
		ti11MergeLiving(&ghostLv, filmGhost)
	}

	t.Logf("======== CUMUL VIVANT bande OBSERVEE (%d films, %d records marches) ========", len(ti11FilmNames()), cumWalk)
	ti11LogLiving(t, "OBSERVEE", cumWalk, lv)
	t.Logf("======== CONTROLE bande FANTOME (%d records marches) — doit paraitre AUSSI vivant si c'est du bruit ========", cumGhostWalk)
	ti11LogLiving(t, "FANTOME", cumGhostWalk, ghostLv)

	// TEMOIN aleatoire : les i3/subs non-sentinelle observes forment-ils un ensemble creux ?
	rng := newTI11Rand(0x71110e17)
	const draws = 4096
	hits := 0
	for i := 0; i < draws; i++ {
		if lv.obsRef[uint64(rng())] {
			hits++
		}
	}
	t.Logf("  TEMOIN aleatoire : %d GlobalID 32b au hasard, %d dans l'ensemble non-sentinelle observe (|obs|=%d) | %.3f %%",
		draws, hits, len(lv.obsRef), 100*float64(hits)/float64(draws))
	t.Logf("  GATE D1 volet vivant : >= 1 champ non-sentinelle qui EVOLUE dans le temps")
}

// TestTI11DeltaSiblingRef — REFERENCE. Le frere ti=13 (managed-property, les zones) parcourt le
// MEME cadre delta (matchWorldObjectRecord + consumeByName + worldObjectHeaderAt) et est EN
// PRODUCTION (zone_state_scan.go). Cette mesure donne son taux de chainage sur LE MEME corpus :
// c'est l'etalon auquel le parse de ti=11 doit se comparer. Si ti=13 chaine haut et ti=11 ~1 %,
// c'est la preuve que ti=11 ne porte PAS de record delta dans ce cadre (et non que le cadre est
// faux — il marche pour ti=13).
func TestTI11DeltaSiblingRef(t *testing.T) {
	root := ti11Root(t)
	release := LockProcessDecode()
	defer release()
	pct := func(a, b int) float64 {
		if b == 0 {
			return 0
		}
		return 100 * float64(a) / float64(b)
	}
	var cumWalk, cumChain int
	for _, name := range ti11FilmNames() {
		sc, err := ScanFilmManagedProperties(root + "/" + name)
		if err != nil {
			t.Logf("  [%s] ti=13 (frere) : %v", name, err)
			continue
		}
		t.Logf("  [%s] ti=13 (frere) : slots %d · records %d · walked %d · chained %d (%.1f %% des walked)",
			name, sc.Slots, sc.Records, sc.Walked, sc.Chained, pct(sc.Chained, sc.Walked))
		cumWalk += sc.Walked
		cumChain += sc.Chained
	}
	t.Logf("======== CUMUL REFERENCE ti=13 : walked %d · chained %d (%.1f %% des walked) ========",
		cumWalk, cumChain, pct(cumChain, cumWalk))
	t.Logf("  A COMPARER au chainage ti=11 de TestTI11DeltaParse (meme cadre, meme corpus).")
}

// ti11LogLiving publie le bilan des champs vivants pour un ensemble de suites.
func ti11LogLiving(t *testing.T, name string, walked int, lv ti11Living) {
	notRef := func(v uint64) bool { return v == ti11SentinelRef }
	notGray := func(v uint64) bool { return v == ti11SentinelColor }
	i3 := ti11EvalField(lv.i3, notRef)
	col := ti11EvalField(lv.color, notGray)
	prog := ti11EvalField(lv.prog, func(v uint64) bool { return v == 0 })
	subs := ti11EvalField(lv.subs, notRef)
	t.Logf("  [%s] (records marches %d)", name, walked)
	ti11LogField(t, "i3  object-ref (PORTEUR)", i3)
	ti11LogField(t, "i1  couleur R (camp)     ", col)
	ti11LogField(t, "i12 progression          ", prog)
	ti11LogField(t, "i16-31 sous-entites      ", subs)
}

// ti11LogField publie une ligne de bilan de champ vivant.
func ti11LogField(t *testing.T, label string, es ti11EvolveStat) {
	t.Logf("      %s : present %5d · distinctes %3d · NON-sentinelle %5d · (Gen,Slot) %3d dont EVOLUENT %3d · seq max %d · %s",
		label, es.present, len(es.distinct), es.nonSentinel, es.slots, es.evolving, es.maxSeq,
		ti11SampleVals(es.distinct, 6))
}

// ti11MergeLiving fusionne les suites d'un film dans le cumul (concatenation par cle).
func ti11MergeLiving(dst *ti11Living, src ti11Living) {
	mergeSeq := func(d, s map[[2]uint32][]uint64) {
		for k, v := range s {
			d[k] = append(d[k], v...)
		}
	}
	mergeSeq(dst.i3, src.i3)
	mergeSeq(dst.color, src.color)
	mergeSeq(dst.prog, src.prog)
	mergeSeq(dst.subs, src.subs)
	for r := range src.obsRef {
		dst.obsRef[r] = true
	}
}
