package filmdec

// navpoint_radial_scan.go — LE BALAYAGE DE LA PROGRESSION RADIALE DES POINTS DE NAVIGATION
// (ti=12 i14, `managed-navpoint-radial-progress`), en production.
//
// # CE QUE CE BALAYAGE FAIT SORTIR, ET CE QUE C'EST
//
// L'anneau qui se remplit autour d'un marqueur d'objectif a l'ecran. Sur les films d'ASSAUT,
// cet anneau EST LA JAUGE D'ARMEMENT DE LA BOMBE — ce n'est pas une hypothese, c'est le
// resultat du protocole du 2026-09-01 (`navpoint_ti12_plancher_test.go`, fixe AVANT la mesure) :
//
//	A (reel)      la fin de la derniere montee CONTIGUE de l'anneau precede CHACUNE des 13
//	              explosions des 5 films Neutral Bomb de 4,93 s median, CV 0,016 ;
//	B (nulle)     0 des 1 000 tirages uniformes (graine fixe) ne fait aussi bien ;
//	C (decalages) +45 s / -45 s / +120 s echouent tous les trois.
//
// Husky Raid confirme (4/4, mediane 5,1 s, CV 0,016, 0/1000) : la meche est du MOTEUR, le hold
// est un reglage de mode. One Bomb NE TIENT PAS (CV 0,725, 87/1000) : ne pas generaliser a
// cette variante — la garde est chez l'appelant (`analysis/replay`).
//
// # LA MEME REGLE QUE `objective_scan.go`
//
// LE DESERIALISEUR PUBLIE, JAMAIS UN SECOND LECTEUR : ce fichier ancre les records de
// l'archetype, marche leur masque avec les deserialiseurs DE PRODUCTION (`consumeByName`) et
// recolte ce que `SetNavpointHook` publie. Zero copie de grammaire. Bande d'ancrage = les slots
// OBSERVES aux images-cles (meme regle qu'`observedSlotBand`), jamais la bande comblee.
//
// # DEUX VOIES, ET UNE SEULE EST EXPLOITABLE — dit avant les chiffres
//
// ti=12 n'est porte qu'a DEUX composants sur 24 (`i0 sub-type` et `i14 radial-progress`).
// Toute marche dont le masque cite un autre composant s'arrete au premier non porte :
//
//	VOIE DELTA       la seule exploitable — un record delta ne cite que ce qui a CHANGE, donc
//	                 un record qui ne bouge que la progression radiale porte le masque {14}
//	                 seul. C'est elle qui a passe le plancher 0/1000.
//	VOIE IMAGE-CLE   quasi condamnee — un record d'image-cle porte l'etat COMPLET, donc cite
//	                 i1 des le depart ; la traversee desynchronise. Elle est balayee quand
//	                 meme, pour que le chiffre soit publie et non suppose (champ Blocked).
//
// # L'HORLOGE EST CELLE DU MANIFESTE, ET C'EST CE QUI REND LA LECTURE CONFRONTABLE
//
// Les lectures sont datees sur `start_ms` par chunk (le manifeste du cache film), base = le
// PREMIER PAQUET DELTA du chunk — la MEME base que `objectiveevents.StatRecords`, donc que les
// explosions du statborg. Un chunk absent du manifeste rend ses paquets inconfrontables : ils
// sont comptes (`PacketsNoClock`), jamais tus.
//
// # CE QU'IL NE DIT PAS
//
// QUI arme : le navpoint est un marqueur d'ecran, pas un acteur. Et les navpoints vont PAR
// PAIRES (+12 d'ecart de slot, un par camp) qui repliquent le MEME anneau : la deduplication
// est le travail de l'interpretation (`analysis/replay`), pas du decodeur.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requete.

import (
	"fmt"

	"levelup/go-api/internal/analysis/filmsource"
)

// navpointRadialArchIndex est l'archetype des points de navigation geres.
const navpointRadialArchIndex = 12

// navpointRadialMaxReads plafonne la recolte par film. UN BALAYAGE PAR ANCRAGE PEUT DIVERGER
// (bande large + bruit) et la machine a deja connu quatre sinistres memoire : le plafond est
// une garde, pas une optimisation. Il est ANNONCE quand il mord (champ Truncated).
const navpointRadialMaxReads = 3_000_000

// NavpointRadialRead est UNE lecture de `managed-navpoint-radial-progress` (ti=12 i14), datee
// sur l'horloge du MANIFESTE (la meme que `objectiveevents.StatRecords`, donc que les
// explosions du statborg).
type NavpointRadialRead struct {
	// Slot identifie le point de navigation. Les navpoints vont par paires (+12, un par camp).
	Slot uint32
	// TMS est l'instant en millisecondes de MATCH (horloge du manifeste).
	TMS int32
	// Q est le quantum de progression, plage R(8) : 0..255.
	Q uint8
	// Chained dit que le record porteur se termine sur un en-tete de record valide — le seul
	// temoin de fiabilite PAR LECTURE que le balayage possede.
	Chained bool
}

// NavpointRadialScan est ce qu'un balayage rend : les lectures, et de quoi juger.
type NavpointRadialScan struct {
	Reads []NavpointRadialRead
	// SlotsObserved / SlotsBand : la bande utilisee (slots observes), et celle qu'un
	// comblement aurait rendue (temoin de ce que le comblement aurait coute).
	SlotsObserved, SlotsBand int
	// KeyCensus est le nombre de vies (slot, gen) recensees aux images-cles.
	KeyCensus int
	// Records / Walked / Broken / Chained : voie DELTA.
	Records, Walked, Broken, Chained int
	// KeyRecords / KeyWalked / KeyBroken / KeyChained : voie IMAGE-CLE.
	KeyRecords, KeyWalked, KeyBroken, KeyChained int
	// Blocked[i] compte les marches arretees au composant i (premier non porte rencontre).
	Blocked map[int]int
	// PacketsNoClock compte les paquets d'un chunk absent du manifeste : sans horloge, la
	// lecture ne serait pas confrontable. Dit, jamais tu.
	PacketsNoClock int
	// Truncated dit que le plafond de lectures a mordu.
	Truncated bool
}

// ScanFilmNavpointRadial balaye UN film et rend les lectures de la progression radiale, datees
// sur l'horloge du manifeste (chunkStartMS : start_ms par index de chunk, lu par l'appelant au
// manifeste du cache film — ce paquet ne connait pas le cache du titre).
//
// Une bande vide n'est pas une erreur : c'est le negatif « aucun navpoint dans ce film », et le
// scan rendu le dit (SlotsObserved == 0).
//
// UN SEUL DECODAGE filmdec A LA FOIS PAR PROCESS : le balayage installe un hook global de
// paquet. L'appelant detient `LockProcessDecode` (BuildFromFilm le fait).
//
// ScanFilmNavpointRadial est l'ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle
// [ScanNavpointRadial].
func ScanFilmNavpointRadial(dir string, chunkStartMS map[int]int) (*NavpointRadialScan, error) {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return &NavpointRadialScan{Blocked: map[int]int{}}, err
	}
	return ScanNavpointRadial(film, chunkStartMS)
}

// ScanNavpointRadial balaye l'anneau ti=12 d'un film DEJA CHARGE.
func ScanNavpointRadial(film *filmsource.Film, chunkStartMS map[int]int) (*NavpointRadialScan, error) {
	sc := &NavpointRadialScan{Blocked: map[int]int{}}
	nums := FilmChunkNumbers(film)
	if len(nums) == 0 {
		return sc, ErrNoFilmChunk
	}
	kf := ScanWorldObjectKeyframes(film, navpointRadialArchIndex)
	band := bandeObserveeKeyframes(kf)
	sc.KeyCensus, sc.SlotsBand, sc.SlotsObserved = len(kf.SeenUS), len(kf.Band), len(band)
	if len(band) == 0 {
		return sc, nil
	}
	arch, reg, err := filmArchetype(film, navpointRadialArchIndex)
	if err != nil {
		return sc, err
	}
	w := navpointRadialWalk{arch: arch, reg: reg, sc: sc}
	defer w.install()()
	for _, c := range nums {
		data, pks, ok := FilmChunkAt(film, c)
		if !ok {
			continue
		}
		w.scanChunk(data, pks, band, chunkStartMS, c)
	}
	return sc, nil
}

// bandeObserveeKeyframes rend les slots REELLEMENT vus porter l'archetype recense, prives de
// ceux qu'un autre archetype revendique. PARTAGEE : ti=12 (production) et l'instrument ti=10
// s'en servent.
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

// filmArchetype charge UN archetype du registre du film. PARTAGEE : ti=12 (production) et
// l'instrument ti=10 s'en servent (le chemin ti=11 garde `objectiveArchetype`, qui ne rend pas
// les memes erreurs).
func filmArchetype(film *filmsource.Film, ti int) (Archetype, *Registry, error) {
	reg, err := filmRegistry(film)
	if err != nil {
		return Archetype{}, nil, err
	}
	arch, ok := reg.Archetype(ti)
	if !ok {
		return Archetype{}, nil, fmt.Errorf("archetype ti=%d absent du registre", ti)
	}
	return arch, reg, nil
}

// navpointRadialWalk porte ce que la marche d'un record doit connaitre, et l'etat que le hook
// y depose (regle des 5 parametres).
type navpointRadialWalk struct {
	arch Archetype
	reg  *Registry
	cur  NavpointRadialRead
	got  bool
	sc   *NavpointRadialScan
	key  bool
}

// install pose le hook de ti=12 et rend sa restauration (defer).
func (w *navpointRadialWalk) install() func() {
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
func (s *NavpointRadialScan) ajouter(r NavpointRadialRead) {
	if len(s.Reads) >= navpointRadialMaxReads {
		s.Truncated = true
		return
	}
	s.Reads = append(s.Reads, r)
}

// scanChunk balaye les paquets d'UN chunk, delta et image-cle, sur l'horloge du manifeste.
//
// LA BASE DE L'HORLOGE EST LE PREMIER PAQUET DELTA DU CHUNK, exactement comme la mesure du
// lot C (`zsScan`) : c'est ce qui met les lectures sur la MEME base que
// `objectiveevents.StatRecords`, donc que les explosions du statborg.
func (w *navpointRadialWalk) scanChunk(data []byte, pks []FilmPacket, band map[uint32]bool,
	startMS map[int]int, c int,
) {
	base, ok := navpointRadialBaseChunk(pks)
	start, hasStart := startMS[c]
	for _, pk := range pks {
		if !ok || !hasStart {
			w.sc.PacketsNoClock++
			continue
		}
		ms := int32(start + int((int64(pk.TimestampUS)-int64(base))/1000))
		switch pk.Type {
		case PacketTypeDelta:
			w.scanPayload(pk.Payload(data), band, ms)
		case PacketTypeKeyframe:
			w.scanKeyframe(pk.Payload(data), ms)
		}
	}
}

// navpointRadialBaseChunk rend l'horodatage moteur du PREMIER paquet delta du chunk.
func navpointRadialBaseChunk(pks []FilmPacket) (uint64, bool) {
	for _, pk := range pks {
		if pk.Type == PacketTypeDelta {
			return pk.TimestampUS, true
		}
	}
	return 0, false
}

// scanPayload ancre les records delta de la bande, marche leur masque, et compte le chainage.
func (w *navpointRadialWalk) scanPayload(pay []byte, band map[uint32]bool, ms int32) {
	sc := w.sc
	limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits)
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, band)
		if !ok || !w.dansLeDomaine(rec.Idx) {
			continue
		}
		sc.Records++
		first := len(sc.Reads)
		end, done := w.walk(pay, rec, ms)
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
func (w *navpointRadialWalk) dansLeDomaine(idx []int) bool {
	for _, id := range idx {
		if id < 0 || id >= len(w.arch.Components) {
			return false
		}
	}
	return true
}

// walk marche les composants du masque avec les desers DE PRODUCTION et recolte ce que le hook
// publie. Rend la position de fin et l'aboutissement, et note le premier composant bloquant.
func (w *navpointRadialWalk) walk(pay []byte, rec WorldObjectRecord, ms int32) (int, bool) {
	sc := w.sc
	total := len(pay) * 8
	at := rec.After
	for _, id := range rec.Idx {
		name := w.arch.component(id)
		if name == "" || at > total {
			sc.Blocked[id]++
			return at, false
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		w.got, w.key = false, false
		_, _, ported := consumeByName(br, name, navpointRadialArchIndex, w.arch.Level(id))
		if !ported || br.BitPos() > total {
			sc.Blocked[id]++
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
func (w *navpointRadialWalk) scanKeyframe(pay []byte, ms int32) {
	sc := w.sc
	total := len(pay) * 8
	w.key = true
	defer func() { w.key = false }()
	for _, r := range WalkKeyframeWorld(pay) {
		if r.TI != navpointRadialArchIndex {
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
			sc.Blocked[tr.DesyncAt]++
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
