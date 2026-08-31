package filmdec

// objective_scan.go — LE BALAYAGE DE L'ARCHETYPE DES OBJECTIFS GERES (ti=11), en production.
//
// # CE QUE CE BALAYAGE FAIT SORTIR
//
// Les six champs publies de `components_managed_objective.go`, dates sur l'HORLOGE MOTEUR du
// film — la meme que les positions de bipede, donc directement posable sur la grille de frames du
// rejeu. Au premier rang : la JAUGE (`i12`) et son SEUIL (`i13`), qui ensemble donnent la
// fraction de capture d'un objectif image par image.
//
// # LA MEME REGLE QUE `zone_state_scan.go`
//
// LE DESERIALISEUR PUBLIE, JAMAIS UN SECOND LECTEUR. Ce fichier ancre les records de
// l'archetype, marche leur masque avec les deserialiseurs DE PRODUCTION (`consumeByName`) et
// recolte ce que le hook publie. Zero copie de grammaire.
//
// # DEUX VOIES, ET AUCUNE N'EST ENCORE VALIDEE (mesure du 2026-09-01 — A LIRE AVANT USAGE)
//
// `ScanFilmManagedProperties` (ti=13) n'ouvre que les paquets DELTA, parce que sur cet
// archetype-la l'ancrage par bande de slots chaine a 87-99 %. Sur ti=11 la meme voie chaine a
// 2,7-26 % et sort des valeurs uniformement reparties sur 32 bits : la bande compte jusqu'a
// 1 704 slots, elle comble les trous et attrape du bruit. Ce fichier ouvre donc AUSSI les
// IMAGES-CLES, ou l'ancrage est structurel (en-tete de 64 bits).
//
// LA VOIE IMAGE-CLE N'EST PAS BONNE NON PLUS, ET IL FAUT LE DIRE. Elle marche 98,4 % des records
// jusqu'au bout — mais « marcher jusqu'au bout » ne teste QUE la COUVERTURE du dispatch, jamais
// la JUSTESSE des largeurs. Trois faits mesures disent qu'une derive subsiste :
//
//	1. la valeur de `i12` est CONSTANTE sur toute la duree d'un slot (35 lectures sur 720 s) ;
//	2. les MEMES valeurs apparaissent a l'identique dans TROIS MATCHS DIFFERENTS
//	   (3 997 696, 255 852 575, 268 435 471, 2 097 152, 16 777 216...) ;
//	3. des slots consecutifs rendent des valeurs decalees d'UN BIT : 0x04000003, 0x08000007,
//	   0x1000000F — la signature d'une fenetre qui glisse sur une zone constante.
//
// Une jauge de partie ne peut pas etre octet-pour-octet identique dans trois matchs. Ce que ces
// lectures montrent n'est donc PAS la progression : c'est une fenetre mal posee, soit parce
// qu'une largeur reste fausse, soit parce que `WalkKeyframeWorld` ancre des faux records (14,9 %
// des masques tombent deja hors du domaine de l'archetype).
//
// CE BALAYAGE EST DONC UN INSTRUMENT DE MESURE, PAS UNE SOURCE DE PRODUCTION. Prochaine etape :
// ancrer par `WalkKeyframeRecords` (marche deterministe) plutot que par le balayeur, et se donner
// un oracle de largeur — un record dont la fin tombe exactement sur l'en-tete suivant.
//
// # LE TEMOIN DE FIABILITE EST PUBLIE, PAS GARDE POUR LES JOURNAUX
//
// `Chained` compte les marches dont la position de fin porte un en-tete de record valide, par
// voie. Une grammaire fausse le fait s'effondrer. C'est a l'appelant de le comparer a `Walked`
// et d'ecarter la contamination d'ancrage.
//
// # CE QU'IL NE DIT PAS
//
// La SEMANTIQUE des valeurs. Les 32 bits de la jauge sortent BRUTS ; le type d'objectif sort en
// enumere brut ; l'etat sort en trois bits bruts. Interpreter est le travail de `analysis/replay`,
// pas celui du decodeur.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requete.

import "fmt"

// ObjectiveRead est UNE valeur de champ d'objectif lue dans un paquet.
type ObjectiveRead struct {
	// Slot identifie l'objectif gere. C'est la cle de regroupement : un slot = un objectif.
	Slot uint32
	// TimestampUS est l'horodatage MOTEUR du paquet.
	TimestampUS uint64
	// Field dit quel composant a publie.
	Field ObjectiveField
	// Value est la premiere valeur publiee. ValueB la seconde, et HasB dit si elle existe :
	// seul `i0 timers` en publie deux (le second minuteur). Un tableau fixe plutot qu'une
	// tranche — un balayage de corpus fait des centaines de milliers de lectures, et allouer
	// pour deux entiers coute plus que de les nommer.
	Value, ValueB uint64
	HasB          bool
	// Chained dit que le RECORD qui porte cette lecture est CHAINE : sa position de fin porte un
	// en-tete de record valide. C'est le seul temoin de fiabilite PAR LECTURE que le balayage
	// possede. Faux pour le dernier record d'un paquet (rien ne peut le suivre).
	Chained bool
	// FromKeyframe dit que la lecture vient d'un record d'IMAGE-CLE et non d'un paquet delta.
	//
	// LA DISTINCTION N'EST PAS COSMETIQUE, elle est le resultat d'une mesure (2026-09-01). Les
	// deux voies n'ont pas du tout la meme fiabilite sur cet archetype :
	//
	//	IMAGE-CLE   les records sont ancres par leur en-tete de 64 bits, la marche aboutit sur
	//	            98,4 % d'entre eux, et les masques tombent dans le domaine de l'archetype.
	//	DELTA       l'ancrage repose sur une BANDE DE SLOTS qui, pour ti=11, compte 43 a 1 704
	//	            slots — elle comble les trous et attrape du bruit. Le chainage mesure vaut
	//	            2,7 a 26 % (contre 87 a 99 % sur ti=13 correctement ancre), et les valeurs
	//	            sorties sont uniformement reparties sur les 32 bits : ce n'est pas une jauge,
	//	            c'est du bruit. L'APPELANT DOIT FILTRER.
	FromKeyframe bool
}

// ObjectiveScan est ce qu'un balayage rend : les lectures, et de quoi juger.
type ObjectiveScan struct {
	Reads []ObjectiveRead
	// Slots est la taille de la bande d'ancrage : le denominateur de tout ce qui suit.
	Slots int
	// Records / Walked / Broken : records DELTA ancres, records dont la marche a abouti, et ceux
	// dont elle s'est arretee (composant non porte — en pratique `i4 interaction-filter` —
	// ou debordement du payload).
	Records, Walked, Broken int
	// Chained compte les marches DELTA abouties dont la position de fin porte un EN-TETE DE
	// RECORD valide. L'appelant le compare a `Walked` — sur ti=11 ce rapport est bas (cf.
	// `ObjectiveRead.FromKeyframe`), et c'est la mesure qui dit de ne pas faire confiance a
	// cette voie.
	Chained int
	// KeyRecords / KeyWalked / KeyBroken / KeyChained : les memes comptes pour la voie
	// IMAGE-CLE, ou l'ancrage est structurel (en-tete de 64 bits) et non par bande de slots.
	KeyRecords, KeyWalked, KeyBroken, KeyChained int
}

// ScanFilmObjectives balaye les paquets DELTA et les IMAGES-CLES du film de dir, et rend les
// champs publies de ti=11. Chaque lecture porte sa voie (`FromKeyframe`) : lire la doc du champ
// avant d'exploiter la voie delta.
//
// UN SEUL DECODAGE filmdec A LA FOIS PAR PROCESS : ce balayage installe un hook global de
// paquet. Il est restaure a la sortie, y compris en cas d'erreur.
func ScanFilmObjectives(dir string) (ObjectiveScan, error) {
	sc := ObjectiveScan{}
	n := CountFilmChunks(dir)
	if n == 0 {
		return sc, fmt.Errorf("aucun chunk film dans %s", dir)
	}
	band := worldObjectSlotBand(dir, n, ObjectiveTypeIndex)
	if len(band) == 0 {
		return sc, fmt.Errorf("aucun slot d'archetype ti=%d dans les keyframes de %s",
			ObjectiveTypeIndex, dir)
	}
	sc.Slots = len(band)
	arch, reg, err := objectiveArchetype(dir)
	if err != nil {
		return sc, err
	}
	w := objectiveWalk{arch: arch, reg: reg, sc: &sc}
	defer w.install()()
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			switch pk.Type {
			case PacketTypeDelta:
				w.scanPayload(pk.Payload(data), band, pk.TimestampUS, &sc)
			case PacketTypeKeyframe:
				w.scanKeyframe(pk.Payload(data), pk.TimestampUS, &sc)
			}
		}
	}
	return sc, nil
}

// objectiveArchetype charge l'archetype des objectifs geres (ti=11) du registre.
//
// LE DECOUPAGE DU REGISTRE CHANGE AVEC LE BUILD (mesure du lot 0) : les noms sont lus du film,
// jamais supposes aux index attendus — c'est `consumeByName` qui route, et un archetype dont les
// noms ne sont pas ceux de ti=11 rend simplement zero lecture.
func objectiveArchetype(dir string) (Archetype, *Registry, error) {
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		return Archetype{}, nil, fmt.Errorf("chunk_00 (registre) illisible dans %s : %w", dir, err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		return Archetype{}, nil, fmt.Errorf("registre illisible dans %s : %w", dir, err)
	}
	arch, ok := reg.Archetype(ObjectiveTypeIndex)
	if !ok {
		return Archetype{}, nil, fmt.Errorf("archetype ti=%d absent du registre de %s",
			ObjectiveTypeIndex, dir)
	}
	return arch, reg, nil
}

// objectiveWalk porte ce que la marche d'un record doit connaitre, et l'etat que le hook y
// depose (regle des 5 parametres).
type objectiveWalk struct {
	arch Archetype
	// reg est le registre du film : la voie IMAGE-CLE rejoue `TraverseEntity`, qui le demande.
	reg *Registry
	// cur est la lecture en cours : le hook n'a pas le contexte du record, l'appelant si.
	cur ObjectiveRead
	// got dit que le hook a publie pour le composant courant.
	got bool
	// fromKeyframe dit dans quelle voie la marche courante se trouve, et sc est le balayage a
	// alimenter. LA VOIE IMAGE-CLE EN A BESOIN : elle rejoue `TraverseEntity`, qui deroule la
	// boucle de composants EN INTERNE — l'appelant ne peut pas s'intercaler entre deux
	// composants, donc c'est le hook qui range la lecture.
	fromKeyframe bool
	sc           *ObjectiveScan
}

// install pose le hook des champs d'objectif et rend sa restauration (defer).
func (w *objectiveWalk) install() func() {
	prev := objectiveHook
	SetObjectiveHook(func(f ObjectiveField, values []uint64) {
		if len(values) == 0 {
			return
		}
		w.cur.Field, w.cur.Value = f, values[0]
		w.cur.ValueB, w.cur.HasB = 0, false
		if len(values) > 1 {
			w.cur.ValueB, w.cur.HasB = values[1], true
		}
		w.got = true
		w.cur.FromKeyframe = w.fromKeyframe
		if w.fromKeyframe && w.sc != nil {
			w.sc.Reads = append(w.sc.Reads, w.cur)
		}
	})
	return func() { SetObjectiveHook(prev) }
}

// scanPayload balaye UN payload delta : ancre les records de la bande, marche leur masque, et
// compte le chainage.
func (w *objectiveWalk) scanPayload(pay []byte, band map[uint32]bool, ts uint64,
	sc *ObjectiveScan,
) {
	limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits)
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, band)
		if !ok {
			continue
		}
		sc.Records++
		first := len(sc.Reads) // les lectures de CE record commencent ici
		end, done := w.walk(pay, rec, ts, sc)
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
		p = rec.After // un record reconnu n'est pas re-balaye
	}
}

// walk marche les composants du masque avec les desers DE PRODUCTION et recolte ce que le hook
// publie. Rend la position de fin et l'aboutissement.
//
// ELLE S'ARRETE DES QU'UN COMPOSANT N'EST PAS PORTE ou que la marche deborde : au-dela, la
// position du curseur ne serait plus digne de confiance, et lire du bruit vaut moins que ne rien
// lire. Sur ti=11 le seul composant qui l'arrete est `i4 interaction-filter`.
func (w *objectiveWalk) walk(pay []byte, rec WorldObjectRecord, ts uint64,
	sc *ObjectiveScan,
) (int, bool) {
	total := len(pay) * 8
	at := rec.After
	for _, id := range rec.Idx {
		name := w.arch.component(id)
		if name == "" || at > total {
			return at, false
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		w.got = false
		_, _, ported := consumeByName(br, name, ObjectiveTypeIndex, w.arch.Level(id))
		if !ported || br.BitPos() > total {
			return at, false
		}
		at = br.BitPos()
		if w.got {
			w.cur.Slot, w.cur.TimestampUS = rec.Slot, ts
			sc.Reads = append(sc.Reads, w.cur)
		}
	}
	return at, true
}

// scanKeyframe balaie UN payload d'image-cle : les records de l'archetype y sont ancres par leur
// EN-TETE DE 64 BITS, sans bande de slots ni fenetre de balayage — c'est la voie fiable.
//
// POURQUOI ELLE VAUT MIEUX QUE LA VOIE DELTA SUR CET ARCHETYPE (mesure du 2026-09-01) : le
// recensement des masques a montre 2 211 records marches jusqu'au bout sur 2 248 dans le domaine
// (98,4 %), avec des masques structures ; la voie delta, elle, chaine a 2,7-26 % et sort des
// valeurs uniformement reparties sur 32 bits. L'ancrage delta de ti=11 est un chantier a part.
func (w *objectiveWalk) scanKeyframe(pay []byte, ts uint64, sc *ObjectiveScan) {
	total := len(pay) * 8
	w.fromKeyframe = true
	defer func() { w.fromKeyframe = false }()
	for _, r := range WalkKeyframeWorld(pay) {
		if r.TI != ObjectiveTypeIndex {
			continue
		}
		sc.KeyRecords++
		first := len(sc.Reads)
		br := NewBitReader(pay)
		br.SetBitPos(r.Bit + keyframeRecordTIBit)
		w.cur.Slot, w.cur.TimestampUS = uint32(r.Slot), ts
		tr := TraverseEntity(br, w.reg, 0)
		if tr.DesyncAt >= 0 || tr.EndBit > total {
			sc.KeyBroken++
			sc.Reads = sc.Reads[:first] // une marche cassee ne laisse aucune lecture
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
