package filmdec

// rtpc_ti10_scan_test.go — LE BALAYAGE DE L'ARCHETYPE DES OBJETS SCRIPTES DU MODE (ti=10), pour
// le canal qui y pilote un son : `managed-object-rtpc-component` (i26..i29).
//
// # CE QUE CE BALAYAGE FAIT SORTIR
//
// Un RTPC est un parametre temps reel Wwise : l'IDENTIFIANT de 32 bits nomme le canal (il est
// constant par composant, c'est `FNV-1(nom minuscule)` — la fabrique est lue et validee cinq
// fois), et les 22 bits qui suivent portent sa VALEUR courante, dequantifiee dans
// [-10000, +10000]. Le lecteur est `FUN_140796d38`, le composant est PORTE depuis le lot C, et
// `SetManagedObjectHook` le publie deja. Rien a porter : tout est a MESURER.
//
// # CLONE D'INSTRUMENT, ET LA COPIE EST ASSUMEE
//
// Squelette repris de `navpoint_ti12_scan_test.go`, lui-meme clone d'`objective_scan.go`. Le
// depot demande de centraliser des la troisieme copie d'un motif — mais l'ancrage de records
// d'objet du monde (`matchWorldObjectRecord` + `worldObjectHeaderAt`) est deja recopie dans
// VINGT-SIX fichiers du paquet, un par instrument de mesure. Centraliser ces vingt-six est une
// dette reelle et un chantier a part ; en centraliser trois serait une factorisation abandonnee,
// l anti-patron numero 8 du depot. Ce qui POUVAIT etre partage l'a ete : `bandeObserveeKeyframes`,
// `filmArchetype`, `ti12Horloge`, `ti12Quantile`, `ti12Bloquants`, l'oracle `ti12Explosions` et
// les deux seuils du critere (`ti12SensMaxMS`, `ti12DispersionMax`) sont REUTILISES, pas copies.
//
// # LES TROIS POINTS QUI FONT L'HONNETETE D'UN BALAYAGE, repris a l'identique
//
//	BANDE OBSERVEE      les slots REELLEMENT vus porter ti=10 aux images-cles, sans comblement ;
//	                    la bande comblee est publiee a cote, comme temoin de son cout.
//	DEUX VOIES          paquets DELTA (ancrage par bande de slots) ET IMAGES-CLES (ancrage
//	                    structurel par en-tete de 64 bits).
//	TEMOIN PAR LECTURE  `Chained` — la position de fin du record porte-t-elle un en-tete valide.
//
// LE DESERIALISEUR PUBLIE, JAMAIS UN SECOND LECTEUR : la marche appelle `consumeByName` (code de
// PRODUCTION) et recolte ce que `SetManagedObjectHook` publie. Zero copie de grammaire.
//
// # CE QUI DIFFERE DE ti=12, ET IL FAUT LE DIRE AVANT DE LIRE LES CHIFFRES
//
// ti=10 compte 30 composants et n'est porte qu'a SIX : i0, i1 et i26..i29. Les vingt-quatre du
// milieu (i2..i17 navpoints, i18..i21 proprietes reseau, i22 filtre d'interaction, i23 drapeaux,
// i24/i25 sons en boucle) arretent la marche des qu'un masque les cite. Consequence mecanique,
// la meme qu'en ti=12 :
//
//	VOIE IMAGE-CLE  quasi condamnee — un record d'etat complet cite i2 des le depart et la
//	                traversee desynchronise. Elle est balayee quand meme, pour que le chiffre
//	                soit PUBLIE et non suppose.
//	VOIE DELTA      la seule exploitable — un record qui ne bouge qu'un rtpc porte {26} seul.
//
// La distribution du PREMIER COMPOSANT BLOQUANT est donc publiee : elle dit quelle part du trafic
// ti=10 nous echappe, et c'est un resultat, pas un defaut d'instrument.
//
// # L'IDENTIFIANT EST LA VRAIE IDENTITE DU CANAL, PAS L'INDEX DE COMPOSANT
//
// `consumeByName` ne recoit pas l'index du composant (le jeu, lui, le lit dans son descripteur
// pour choisir sa case d'etat `etat + 0x17c + index*8`). Le BALAYEUR, en revanche, le connait :
// c'est lui qui deroule le masque. Les deux sont donc ranges dans la lecture — l'identifiant
// pour comparer les modes entre eux, l'index i26..i29 pour savoir quelle case parle.
//
// UN IDENTIFIANT NUL N'EST PAS UNE DONNEE MANQUANTE : c'est un emplacement LIBERE
// (`Object_ClearRtpcs` ecrit {id=0, valeur=0x00800000}, la meme sentinelle que le lecteur de
// film pose quand l'identifiant est nul). Ces lectures sont comptees a part et n'entrent dans
// aucune serie de valeurs.
//
// FICHIER DE TEST, PAS DE PRODUCTION : cet instrument ne modifie aucun chemin servi.
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requete.

import "fmt"

// ti10ArchIndex est l'archetype des objets scriptes du mode.
const ti10ArchIndex = 10

// ti10MaxLectures plafonne la recolte par film. UN BALAYAGE PAR ANCRAGE PEUT DIVERGER (bande
// large + bruit) et la machine a deja connu quatre sinistres memoire : le plafond est une garde,
// pas une optimisation. Il est ANNONCE quand il mord.
const ti10MaxLectures = 3_000_000

// ti10Read est UNE lecture de `managed-object-rtpc-component`, datee sur l'horloge du MANIFESTE
// (la meme que `objectiveevents.StatRecords`, donc que l'oracle des explosions).
type ti10Read struct {
	Slot uint32
	TMS  int32
	// Comp est l'index de composant qui a publie (26..29 sur la voie delta). VAUT -1 SUR LA
	// VOIE IMAGE-CLE : `TraverseEntity` y deroule la boucle de composants en interne, et
	// l'appelant ne peut pas s'intercaler entre deux composants pour le savoir.
	Comp int16
	// ID est l'identifiant Wwise du canal, HasQ dit si une valeur a suivi (identifiant non nul)
	// et Q la porte, en quantum de 22 bits.
	ID      uint32
	Q       uint32
	HasQ    bool
	Chained bool
}

// ti10Scan est ce qu'un balayage rend : les lectures, et de quoi juger.
type ti10Scan struct {
	Reads []ti10Read
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

// ti10ScanFilm balaye UN film et rend les lectures de rtpc.
//
// UN SEUL DECODAGE filmdec A LA FOIS PAR PROCESS : le balayage installe un hook global.
func ti10ScanFilm(dir string, clk zcClock) (*ti10Scan, error) {
	sc := &ti10Scan{Bloque: map[int]int{}}
	n := CountFilmChunks(dir)
	if n == 0 {
		return sc, fmt.Errorf("aucun chunk film dans %s", dir)
	}
	kf := ScanFilmWorldObjectKeyframes(dir, ti10ArchIndex)
	band := bandeObserveeKeyframes(kf)
	sc.KeyCensus, sc.SlotsBande, sc.SlotsObserves = len(kf.SeenUS), len(kf.Band), len(band)
	if len(band) == 0 {
		return sc, nil // pas une erreur : c'est le negatif du gate 0
	}
	arch, reg, err := filmArchetypeDir(dir, ti10ArchIndex)
	if err != nil {
		return sc, err
	}
	w := ti10Walk{arch: arch, reg: reg, sc: sc, band: band}
	defer w.install()()
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		w.scanChunk(data, clk.startMS, c)
	}
	return sc, nil
}

// ti10Walk porte ce que la marche d'un record doit connaitre, et l'etat que le hook y depose
// (regle des 5 parametres).
type ti10Walk struct {
	arch Archetype
	reg  *Registry
	band map[uint32]bool
	cur  ti10Read
	got  bool
	sc   *ti10Scan
	key  bool
}

// install pose le hook de ti=10 et rend sa restauration (defer).
//
// LE HOOK NE RANGE LUI-MEME QUE SUR LA VOIE IMAGE-CLE, ou la boucle de composants est interne a
// `TraverseEntity` ; sur la voie delta c'est la marche qui range, APRES avoir verifie que le
// composant a bien ete consomme — une lecture publiee par un composant qui deborde ensuite ne
// vaut rien.
func (w *ti10Walk) install() func() {
	prev := managedObjectHook
	SetManagedObjectHook(func(f ManagedObjectField, values []uint64) {
		if f != ManagedObjectRTPC || len(values) == 0 {
			return
		}
		w.cur.ID, w.cur.Q, w.cur.HasQ = uint32(values[0]), 0, false
		if len(values) > 1 {
			w.cur.Q, w.cur.HasQ = uint32(values[1]), true
		}
		w.got = true
		if w.key && w.sc != nil {
			w.sc.ajouter(w.cur)
		}
	})
	return func() { SetManagedObjectHook(prev) }
}

// ajouter range une lecture sous le plafond de recolte.
func (s *ti10Scan) ajouter(r ti10Read) {
	if len(s.Reads) >= ti10MaxLectures {
		s.Tronque = true
		return
	}
	s.Reads = append(s.Reads, r)
}

// scanChunk balaye les paquets d'UN chunk, delta et image-cle, sur l'horloge du manifeste.
//
// LA BASE DE L'HORLOGE EST LE PREMIER PAQUET DELTA DU CHUNK, exactement comme `zcScan` et comme
// l'instrument ti=12 : c'est ce qui met les lectures sur la MEME base que
// `objectiveevents.StatRecords`, donc que l'oracle des explosions.
func (w *ti10Walk) scanChunk(data []byte, startMS map[int]int, c int) {
	pks := WalkPackets(data)
	base, ok := navpointRadialBaseChunk(pks)
	start, hasStart := startMS[c]
	for _, pk := range pks {
		if !ok || !hasStart {
			w.sc.PaquetsSansHorloge++
			continue
		}
		ms := int32(start + int((int64(pk.TimestampUS)-int64(base))/1000))
		switch pk.Type {
		case PacketTypeDelta:
			w.scanPayload(pk.Payload(data), ms)
		case PacketTypeKeyframe:
			w.scanKeyframe(pk.Payload(data), ms)
		}
	}
}

// scanPayload ancre les records delta de la bande, marche leur masque, et compte le chainage.
func (w *ti10Walk) scanPayload(pay []byte, ms int32) {
	sc := w.sc
	limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits)
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, w.band)
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
func (w *ti10Walk) dansLeDomaine(idx []int) bool {
	for _, id := range idx {
		if id < 0 || id >= len(w.arch.Components) {
			return false
		}
	}
	return true
}

// walk marche les composants du masque avec les desers DE PRODUCTION et recolte ce que le hook
// publie. Rend la position de fin et l'aboutissement, et note le premier composant bloquant.
func (w *ti10Walk) walk(pay []byte, rec WorldObjectRecord, ms int32) (int, bool) {
	total := len(pay) * 8
	at := rec.After
	for _, id := range rec.Idx {
		name := w.arch.component(id)
		if name == "" || at > total {
			w.sc.Bloque[id]++
			return at, false
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		w.got, w.key = false, false
		_, _, ported := consumeByName(br, name, ti10ArchIndex, w.arch.Level(id))
		if !ported || br.BitPos() > total {
			w.sc.Bloque[id]++
			return at, false
		}
		at = br.BitPos()
		if w.got {
			w.cur.Slot, w.cur.TMS, w.cur.Comp = rec.Slot, ms, int16(id)
			w.sc.ajouter(w.cur)
		}
	}
	return at, true
}

// scanKeyframe balaye UN payload d'image-cle : les records y sont ancres par leur en-tete de
// 64 bits. Sur ti=10 la traversee desynchronise des le premier composant non porte (i2 des qu'un
// navpoint est present) — le chiffre est publie pour que le negatif soit MESURE, pas suppose.
func (w *ti10Walk) scanKeyframe(pay []byte, ms int32) {
	sc := w.sc
	total := len(pay) * 8
	w.key = true
	defer func() { w.key = false }()
	for _, r := range WalkKeyframeWorld(pay) {
		if r.TI != ti10ArchIndex {
			continue
		}
		sc.KeyRecords++
		first := len(sc.Reads)
		br := NewBitReader(pay)
		br.SetBitPos(r.Bit + keyframeRecordTIBit)
		w.cur.Slot, w.cur.TMS, w.cur.Comp = uint32(r.Slot), ms, -1
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
