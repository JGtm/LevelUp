package filmdec

// navpoint_ti12_scan_test.go — LE BALAYAGE DE L'ARCHETYPE DES POINTS DE NAVIGATION (ti=12).
//
// CLONE D'INSTRUMENT de `objective_scan.go` (ti=11), a l'identique sur les trois points qui font
// l'honnetete d'un balayage :
//
//	BANDE OBSERVEE      les slots REELLEMENT vus porter ti=12 aux images-cles, sans comblement
//	                    (meme regle qu'`observedSlotBand`) ; la bande comblee est publiee a cote
//	                    comme temoin de ce que le comblement aurait coute ;
//	DEUX VOIES          les paquets DELTA (ancrage par bande de slots) ET les IMAGES-CLES
//	                    (ancrage structurel par en-tete de 64 bits) ;
//	TEMOIN PAR LECTURE  `Chained` — la position de fin du record porte-t-elle un en-tete valide.
//
// LE DESERIALISEUR PUBLIE, JAMAIS UN SECOND LECTEUR : la marche appelle `consumeByName` (code de
// PRODUCTION) et recolte ce que `SetNavpointHook` publie. Zero copie de grammaire.
//
// CE QUI DIFFERE DE ti=11, ET IL FAUT LE DIRE AVANT DE LIRE LES CHIFFRES. ti=11 est porte a
// 33 composants sur 34 ; ti=12 ne l'est qu'a DEUX sur 24 (`i0 sub-type` et `i14
// radial-progress`). Toute marche dont le masque cite un autre composant s'arrete au premier non
// porte. Consequence mecanique :
//
//	VOIE IMAGE-CLE   quasi condamnee — un record d'image-cle porte l'etat COMPLET, donc cite i1
//	                 des le depart ; la traversee desynchronise. Elle est balayee quand meme,
//	                 pour que le chiffre soit publie et non suppose.
//	VOIE DELTA       la seule exploitable — un record delta ne cite que ce qui a CHANGE, donc un
//	                 record qui ne bouge que la progression radiale porte le masque {14} seul.
//
// La distribution du PREMIER COMPOSANT BLOQUANT est donc publiee : elle dit quelle part du trafic
// ti=12 nous est inaccessible, et c'est un resultat, pas un defaut d'instrument.
//
// FICHIER DE TEST, PAS DE PRODUCTION : cet instrument ne modifie aucun chemin servi.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requete.

import "fmt"

// ti12ArchIndex est l'archetype des points de navigation geres.
const ti12ArchIndex = 12

// ti12MaxLectures plafonne la recolte par film. UN BALAYAGE PAR ANCRAGE PEUT DIVERGER (bande
// large + bruit) et la machine a deja connu quatre sinistres memoire : le plafond est une garde,
// pas une optimisation. Il est ANNONCE quand il mord.
const ti12MaxLectures = 3_000_000

// ti12Read est UNE lecture de `managed-navpoint-radial-progress` (ti=12 i14), datee sur
// l'horloge du MANIFESTE (la meme que `objectiveevents.StatRecords`, donc que l'oracle des
// explosions).
type ti12Read struct {
	Slot    uint32
	TMS     int32
	Q       uint8
	Chained bool
}

// ti12Scan est ce qu'un balayage rend : les lectures, et de quoi juger.
type ti12Scan struct {
	Reads []ti12Read
	// SlotsObserves / SlotsBande : la bande utilisee, et celle qu'un comblement aurait rendue.
	SlotsObserves, SlotsBande int
	// KeyCensus est le nombre de vies (slot, gen) recensees aux images-cles.
	KeyCensus int
	// Records / Walked / Broken / Chained : voie DELTA.
	Records, Walked, Broken, Chained int
	// KeyRecords / KeyWalked / KeyBroken / KeyChained : voie IMAGE-CLE.
	KeyRecords, KeyWalked, KeyBroken, KeyChained int
	// Bloque[i] compte les marches arretees au composant i (premier non porte rencontre).
	Bloque map[int]int
	// PaquetsSansHorloge compte les paquets d'un chunk absent du manifeste : sans horloge, la
	// lecture ne serait pas confrontable. Dit, jamais tu.
	PaquetsSansHorloge int
	// Tronque dit que le plafond de lectures a mordu.
	Tronque bool
}

// ti12ScanFilm balaye UN film et rend les lectures de la progression radiale.
//
// UN SEUL DECODAGE filmdec A LA FOIS PAR PROCESS : le balayage installe un hook global.
func ti12ScanFilm(dir string, clk zcClock) (*ti12Scan, error) {
	sc := &ti12Scan{Bloque: map[int]int{}}
	n := CountFilmChunks(dir)
	if n == 0 {
		return sc, fmt.Errorf("aucun chunk film dans %s", dir)
	}
	kf := ScanFilmWorldObjectKeyframes(dir, ti12ArchIndex)
	band := bandeObserveeKeyframes(kf)
	sc.KeyCensus, sc.SlotsBande, sc.SlotsObserves = len(kf.SeenUS), len(kf.Band), len(band)
	if len(band) == 0 {
		return sc, nil // pas une erreur : c'est le negatif du gate 0
	}
	arch, reg, err := filmArchetype(dir, ti12ArchIndex)
	if err != nil {
		return sc, err
	}
	w := ti12Walk{arch: arch, reg: reg, sc: sc}
	defer w.install()()
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		w.scanChunk(data, band, clk.startMS, c, sc)
	}
	return sc, nil
}

// bandeObserveeKeyframes rend les slots REELLEMENT vus porter l archetype recense, prives de ceux
// qu un autre archetype revendique. PARTAGEE : ti=12 et ti=10 s en servent (une copie de moins).
//
// POURQUOI L'INTERSECTION AVEC LA BANDE COMBLEE SUFFIT. `ScanFilmWorldObjectKeyframes` rend
// `SeenUS` (les vies de l'archetype, donc les slots OBSERVES) et `Band` = comblement(observes)
// prive des slots vus porter autre chose. L'intersection des deux vaut donc exactement
// « observes moins exclus » — la definition d'`observedSlotBand`, sans en recopier la marche.
func bandeObserveeKeyframes(kf WorldObjectKeyframes) map[uint32]bool {
	out := map[uint32]bool{}
	for k := range kf.SeenUS {
		if kf.Band[k.Slot] {
			out[k.Slot] = true
		}
	}
	return out
}

// filmArchetype charge UN archetype du registre du film. PARTAGEE : ti=12 et ti=10 s en servent
// (une copie de moins ; le chemin de PRODUCTION garde `objectiveArchetype`, qui ne rend pas les
// memes erreurs).
func filmArchetype(dir string, ti int) (Archetype, *Registry, error) {
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		return Archetype{}, nil, fmt.Errorf("chunk_00 (registre) illisible dans %s : %w", dir, err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		return Archetype{}, nil, fmt.Errorf("registre illisible dans %s : %w", dir, err)
	}
	arch, ok := reg.Archetype(ti)
	if !ok {
		return Archetype{}, nil, fmt.Errorf("archetype ti=%d absent du registre de %s", ti, dir)
	}
	return arch, reg, nil
}

// ti12Walk porte ce que la marche d'un record doit connaitre, et l'etat que le hook y depose
// (regle des 5 parametres).
type ti12Walk struct {
	arch Archetype
	reg  *Registry
	cur  ti12Read
	got  bool
	sc   *ti12Scan
	key  bool
}

// install pose le hook de ti=12 et rend sa restauration (defer).
func (w *ti12Walk) install() func() {
	prev := navpointHook
	SetNavpointHook(func(f NavpointField, values []uint64) {
		if f != NavpointRadialProgress || len(values) == 0 {
			return
		}
		w.cur.Q, w.got = uint8(values[0]), true
		if w.key && w.sc != nil {
			w.sc.ajouter(w.cur)
		}
	})
	return func() { SetNavpointHook(prev) }
}

// ajouter range une lecture sous le plafond de recolte.
func (s *ti12Scan) ajouter(r ti12Read) {
	if len(s.Reads) >= ti12MaxLectures {
		s.Tronque = true
		return
	}
	s.Reads = append(s.Reads, r)
}

// scanChunk balaye les paquets d'UN chunk, delta et image-cle, sur l'horloge du manifeste.
//
// LA BASE DE L'HORLOGE EST LE PREMIER PAQUET DELTA DU CHUNK, exactement comme la mesure du lot C
// (`zsScan`) : c'est ce qui met les lectures sur la MEME base que `objectiveevents.StatRecords`,
// donc que l'oracle des explosions.
func (w *ti12Walk) scanChunk(data []byte, band map[uint32]bool, startMS map[int]int, c int,
	sc *ti12Scan,
) {
	pks := WalkPackets(data)
	base, ok := ti12BaseChunk(pks)
	start, hasStart := startMS[c]
	for _, pk := range pks {
		if !ok || !hasStart {
			sc.PaquetsSansHorloge++
			continue
		}
		ms := int32(start + int((int64(pk.TimestampUS)-int64(base))/1000))
		switch pk.Type {
		case PacketTypeDelta:
			w.scanPayload(pk.Payload(data), band, ms, sc)
		case PacketTypeKeyframe:
			w.scanKeyframe(pk.Payload(data), ms, sc)
		}
	}
}

// ti12BaseChunk rend l'horodatage moteur du PREMIER paquet delta du chunk.
func ti12BaseChunk(pks []FilmPacket) (uint64, bool) {
	for _, pk := range pks {
		if pk.Type == PacketTypeDelta {
			return pk.TimestampUS, true
		}
	}
	return 0, false
}

// scanPayload ancre les records delta de la bande, marche leur masque, et compte le chainage.
func (w *ti12Walk) scanPayload(pay []byte, band map[uint32]bool, ms int32, sc *ti12Scan) {
	limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits)
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, band)
		if !ok || !w.dansLeDomaine(rec.Idx) {
			continue
		}
		sc.Records++
		first := len(sc.Reads)
		end, done := w.walk(pay, rec, ms, sc)
		if !done {
			sc.Broken++
		} else {
			sc.Walked++
			if worldObjectHeaderAt(pay, end) {
				sc.Chained++
				for k := first; k < len(sc.Reads); k++ {
					sc.Reads[k].Chained = true
				}
			}
		}
		p = rec.After
	}
}

// dansLeDomaine rejette les records dont le masque cite un composant absent de l'archetype.
func (w *ti12Walk) dansLeDomaine(idx []int) bool {
	for _, id := range idx {
		if id < 0 || id >= len(w.arch.Components) {
			return false
		}
	}
	return true
}

// walk marche les composants du masque avec les desers DE PRODUCTION et recolte ce que le hook
// publie. Rend la position de fin et l'aboutissement, et note le premier composant bloquant.
func (w *ti12Walk) walk(pay []byte, rec WorldObjectRecord, ms int32, sc *ti12Scan) (int, bool) {
	total := len(pay) * 8
	at := rec.After
	for _, id := range rec.Idx {
		name := w.arch.component(id)
		if name == "" || at > total {
			sc.Bloque[id]++
			return at, false
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		w.got, w.key = false, false
		_, _, ported := consumeByName(br, name, ti12ArchIndex, w.arch.Level(id))
		if !ported || br.BitPos() > total {
			sc.Bloque[id]++
			return at, false
		}
		at = br.BitPos()
		if w.got {
			w.cur.Slot, w.cur.TMS = rec.Slot, ms
			sc.ajouter(w.cur)
		}
	}
	return at, true
}

// scanKeyframe balaye UN payload d'image-cle : les records y sont ancres par leur en-tete de
// 64 bits. Sur ti=12 la traversee desynchronise des le premier composant non porte — le chiffre
// est publie pour que le negatif soit mesure et non suppose.
func (w *ti12Walk) scanKeyframe(pay []byte, ms int32, sc *ti12Scan) {
	total := len(pay) * 8
	w.key = true
	defer func() { w.key = false }()
	for _, r := range WalkKeyframeWorld(pay) {
		if r.TI != ti12ArchIndex {
			continue
		}
		sc.KeyRecords++
		first := len(sc.Reads)
		br := NewBitReader(pay)
		br.SetBitPos(r.Bit + keyframeRecordTIBit)
		w.cur.Slot, w.cur.TMS = uint32(r.Slot), ms
		tr := TraverseEntity(br, w.reg, 0)
		if tr.DesyncAt >= 0 || tr.EndBit > total {
			sc.KeyBroken++
			sc.Bloque[tr.DesyncAt]++
			sc.Reads = sc.Reads[:first]
			continue
		}
		sc.KeyWalked++
		if _, ok := readKeyframeHeader(pay, tr.EndBit, total); ok {
			sc.KeyChained++
			for k := first; k < len(sc.Reads); k++ {
				sc.Reads[k].Chained = true
			}
		}
	}
}
