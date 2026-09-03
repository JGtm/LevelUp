package filmdec

import "fmt"

// Décodage de la LISTE D'ÉVÉNEMENTS en tête d'un paquet delta de film Theater.
//
// LE MODÈLE DE PAQUET (percée trame du 2026-08-30, chantier mené HORS de ce dépôt ; notes
// Notion « Percer la trame du film » + « Format du film Theater — le bit de continuation ») :
//
//	[1 bit config] [liste d'événements : ( 1 [R(7) type] [3 réfs gardées] [charge] )* 0]
//	[trame de records ECS]
//
// Le décodeur de records de ce dépôt (frame_records.go / DecodeFrameRecords) SAUTE cette liste :
// il consomme DefaultPacketPreambleBits (= 2) puis lit les records. Ces 2 bits sont exactement
// [config][continuation=0] du cas « liste vide » — c'est pourquoi il fonctionne sur les paquets
// à liste vide (octets de tête 0x80..0xBF) et rate ceux qui portent un événement (0xC0..0xFF).
//
// CE FICHIER LIT LA LISTE, EN AMONT DE LA TRAME. Il est STRICTEMENT ADDITIF : il n'appelle ni ne
// modifie le décodeur de records. La trame ECS et l'arme-du-kill restent décodées à l'identique
// (garde-fou : event_list_test.go rejoue le compte fire_events et l'histogramme de tête).
//
// ARITHMÉTIQUE DE TÊTE, prouvée sur tout le corpus par la percée : quand la liste porte un
// événement, l'octet 0 du payload vaut `0xC0 | (type >> 1)`. En lecture MSB-first (BitReader) :
// bit 0 = config, bit 1 = continuation, bits 2..8 = R(7) type. Le bit de poids faible du type
// est donc le bit de poids fort de l'octet 1. Un octet de tête dans 0x80..0xBF a son bit 1 à 0
// (liste vide = trame de records pure, cf. témoin 0xA0).

// Types d'événements de la liste (numérotation TRAME, établie le 2026-08-30 ; toute
// numérotation ANTÉRIEURE à cette date est sans valeur — piège documenté). Seuls les types
// utiles au décodage véhicule sont nommés ici ; le catalogue complet vit dans la note Notion.
const (
	// EventBipedBoardVehicle : embarquement (une unité monte dans un véhicule). Mesuré 374
	// sur 154 films (corpus 1 367). Octet de tête 0xC4.
	EventBipedBoardVehicle = 8
	// EventUnitExitVehicle : sortie (une unité descend d'un véhicule). Mesuré 5 600 sur 279
	// films. Octet de tête 0xCB.
	EventUnitExitVehicle = 22
	// EventUnitEnterVehicle : entrée (type 53) — mesuré 0 en arène (absent des films arène).
	EventUnitEnterVehicle = 53
)

// eventTypeBits est la largeur du champ de type d'un événement : R(7), cf. modèle de paquet.
const eventTypeBits = 7

// eventPayloadStartBit est le bit du PREMIER bit du CORPS du 1er événement (les 3 réfs gardées
// puis la charge) : après [config(1)][continuation(1)][R(7) type].
const eventPayloadStartBit = 2 + eventTypeBits // 9

// PacketHeadEventType lit le type de l'ÉVÉNEMENT DE TÊTE d'un payload de paquet delta : il lit
// le bit de configuration, puis le bit de continuation. Si la continuation est nulle, la liste
// est vide (le paquet est une trame de records pure) et present vaut false. Sinon il lit le
// R(7) du type de tête. Aucune charge n'est décodée : O(1), sans allocation au-delà du reader.
//
// C'est le CADRAGE MINIMAL de la liste : il suffit à compter les familles par type de tête
// (validation des comptes corpus) sans porter la grammaire de charge de chaque type.
func PacketHeadEventType(pay []byte) (typ int, present bool) {
	if len(pay) < 1 {
		return 0, false
	}
	br := NewBitReader(pay)
	_ = br.ReadBit() // bit 0 : drapeau de configuration (1 sur 100 % du corpus)
	if !br.ReadBit() {
		return 0, false // bit 1 : continuation = 0 -> liste vide
	}
	return int(br.ReadBits(eventTypeBits)), true
}

// --- Référence gardée -------------------------------------------------------------------------
//
// Le dispatcher d'événements (FUN_14080AADE, décompilé) lit EXACTEMENT 3 références gardées après
// le type, dans une boucle : pour chaque réf, un bit de garde (FUN_1406cf008) ; si posé, la réf
// est lue par un lecteur propre au domaine (vtable+0x58 + FUN_1406d3140, l'id-reader partagé avec
// readRecordID). La largeur dépend du DOMAINE de la réf (table runtime `0x1451f98d0`) : domaines
// 0/1/7/8 = 13 bits, 2/3/5 = 8, 4/6 = 9. Le domaine 1 porte une SONDE (1 bit) : si posée, la
// largeur tombe à 9. Ces largeurs de référence sont celles de la build de référence ; l'id-reader
// du domaine 7 est le même RUNTIME que `FrameConfig.IDLowBits` (11..14 selon le film).

// dom7RefWidth est la largeur (bits) d'un index de référence de domaine 7/8/0 sur la build de
// référence. Runtime en toute rigueur (cf. IDLowBits), mais 13 vaut sur les films de référence.
const dom7RefWidth = 13

// dom2RefWidth / dom3RefWidth : largeurs des domaines 2 et 3, les DOMAINES DES RÉFÉRENCES DE
// L'EMBARQUEMENT (lus dans l'exécutable le 2026-09-02, cf. boardRefs). Ni l'un ni l'autre ne
// porte de sonde : `FUN_1406d3140` ne la lit que pour le domaine 1.
//
// LA LARGEUR EST RUNTIME EN TOUTE RIGUEUR — `FUN_1406d310c(count)` rend ceil(log2(count)) sur une
// table initialisée à l'exécution, illisible dans l'image statique. Les deux valeurs ci-dessous
// sont donc MESURÉES, chacune par son propre oracle (balayage de largeurs, event_list_board_test.go,
// 8 films / 22 embarquements) :
//   - domaine 2 = 8 : l'occupant tombe dans la bande bipède 22/22 = 100 % et ouvre un trou du flux
//     de position à l'instant de l'événement dans 77,3 % des cas (témoin décalé : 0 %). À 7 bits le
//     recoupement s'effondre à 4,5 %, à 9 bits la bande n'est plus tenue (72,7 %), à 13 bits 9,1 %.
//   - domaine 3 = 7 : départagé par le SIÈGE, qui se lit après les trois réfs. À 7 bits le siège de
//     l'embarquement égale celui de la sortie appariée dans 5/6 = 83,3 % des cas et vaut 0 sur 21/22
//     (même « siège dominant 0 » que la sortie) ; à 8 bits l'accord tombe à 0/6, à 13 bits 4/6.
const (
	dom2RefWidth = 8
	dom3RefWidth = 7
)

// guardedRef est une référence d'entité gardée décodée.
type guardedRef struct {
	Present bool
	Sonde   int    // domaine 1 uniquement : 1 => index sur 9 bits, 0 => 13 bits
	Index   uint32 // index brut (à additionner à la base du domaine pour obtenir le slot)
	Gen     uint32 // génération du handle (2 bits)
	EndBit  int    // bit juste après la référence
}

// readDom1Ref lit une référence gardée de DOMAINE 1 (bipède/unité) à partir du bit `at` :
// garde(1) ; si 1 : sonde(1) ; R(sonde?9:13) index ; R(2) génération.
func readDom1Ref(pay []byte, at int) guardedRef {
	r := guardedRef{EndBit: at + 1}
	if readBitsAt(pay, at, 1) == 0 {
		return r
	}
	r.Present = true
	b := at + 1
	r.Sonde = int(readBitsAt(pay, b, 1))
	b++
	w := 13
	if r.Sonde == 1 {
		w = 9
	}
	r.Index = readBitsAt(pay, b, w)
	b += w
	r.Gen = readBitsAt(pay, b, 2)
	r.EndBit = b + 2
	return r
}

// readPlainRef lit une référence gardée SANS sonde (domaines 2..8) de largeur w : garde(1) ;
// si 1 : R(w) index ; R(2) génération.
func readPlainRef(pay []byte, at, w int) guardedRef {
	r := guardedRef{EndBit: at + 1}
	if readBitsAt(pay, at, 1) == 0 {
		return r
	}
	r.Present = true
	b := at + 1
	r.Index = readBitsAt(pay, b, w)
	b += w
	r.Gen = readBitsAt(pay, b, 2)
	r.EndBit = b + 2
	return r
}

// --- Événements véhicule (embarquement / sortie) ----------------------------------------------

// vehicleSeatBits est la largeur du champ SIÈGE dans la charge d'un événement véhicule : R(6).
const vehicleSeatBits = 6

// VehicleEvent est un embarquement (board) ou une sortie (exit) décodé depuis la liste
// d'événements d'un paquet delta.
type VehicleEvent struct {
	// Kind vaut EventBipedBoardVehicle ou EventUnitExitVehicle.
	Kind int
	// Chunk / PacketIndex localisent l'événement dans le film.
	Chunk, PacketIndex int
	// TimestampUS est l'INSTANT de l'événement, en microsecondes (horloge du film) — même
	// horloge que BipedPosition/FireEvent, donc directement croisable.
	TimestampUS uint64

	// OccupantPresent : la référence 0 (l'unité) est présente.
	OccupantPresent bool
	// OccupantSonde : la sonde de la réf 0 de la SORTIE (domaine 1). 1 = index bipède-relatif
	// (9 bits), 0 = slot absolu (13 bits). Toujours 0 pour un EMBARQUEMENT : ses réfs sont en
	// domaines 2/3/7, et `FUN_1406d3140` ne lit la sonde que pour le domaine 1.
	OccupantSonde int
	// OccupantSlot est le slot de l'unité : base bipède + index si sonde=1, index brut sinon.
	OccupantSlot uint32
	// OccupantInBand : le slot tombe dans la bande de slots bipèdes du film (contrôle).
	OccupantInBand bool

	// VehicleSlot est le slot du VÉHICULE que l'événement NOMME : la RÉFÉRENCE 1 de la SORTIE,
	// lue en domaine 1 exactement comme l'occupant.
	//
	// POURQUOI LE MÊME DOMAINE POUR DEUX CHOSES DIFFÉRENTES — c'est l'acquis du lot V7 (§ 6 du
	// rapport `V7_DESTRUCTION_EVENEMENT_2026-09-02.md`) : LE DOMAINE 1 EST CELUI DES UNITÉS, et
	// dans la taxonomie Halo le BIPÈDE et le VÉHICULE sont deux spécialisations d'UNITÉ. La base
	// est la même (le minimum de la bande bipède) et l'index de 9 bits porte au-delà, jusqu'à la
	// bande `ti=40`. Mesure : sur les références de domaine 1 dont l'index SORT de la bande
	// bipède, 99,6 à 100,0 % tombent dans la bande `ti=40` (types 0/1/7/36, 12 films) là où le
	// hasard en mettrait 3 à 16 %. Le lot V6 avait cherché le véhicule en domaine 7 et l'y avait
	// réfuté à raison : il est en domaine 1.
	//
	// POUR LA SORTIE, LA MESURE EST SANS RESTE : 105 / 105 sorties de 12 films, 100,0 % en bande
	// `ti=40`, zéro bipède, zéro hors bande (V7 § 7). C'est cette référence-là que le calque de
	// rejeu emploie pour résoudre le véhicule d'un épisode d'occupation, la géométrie n'étant plus
	// que le repli.
	VehicleSlot uint32
	// VehicleSlotValid : la référence du véhicule était PRÉSENTE (bit de garde posé). Faux pour un
	// EMBARQUEMENT : ses trois références sont en domaines 2/3/7 et AUCUNE ne résout un slot
	// `ti=40` (mesure au § 2 du rapport V8).
	VehicleSlotValid bool
	// VehicleGen est la GÉNÉRATION (2 bits) du handle du véhicule, lue dans la même référence.
	// Elle n'est PAS la clé de vie employée par le calque — celle-ci se résout par la fenêtre
	// temporelle du recensement —, mais elle en est le contrôle indépendant (V8 § 2).
	VehicleGen uint32

	// Seat est le siège (R(6)) lu en fin de charge. SeatValid=false si le payload est trop court.
	Seat      uint32
	SeatValid bool
}

// decodeVehicleEvent décode l'événement de tête board/exit d'un payload. `base` est le début de
// la plage de slots bipèdes du film (min de la bande) ; `inBand` teste l'appartenance d'un slot
// à la bande. Rend ok=false si le type de tête n'est pas board/exit.
//
// GRAMMAIRE — les DOMAINES des trois réfs sont lus dans l'exécutable, un descripteur par type
// d'événement (cf. decodeExitRefs et boardRefs, qui portent chacun l'adresse de leur vtable) :
//   - SORTIE (`unit_exit_vehicle`) : réfs en domaines 1, 1, 7 puis R(6) siège. La réf 0 est
//     l'occupant : sonde=1 sur 237/237, slot = base + index(9), 95,5 % en bande ; siège = 0 sur
//     93,8 % — « conducteur qui descend ».
//   - EMBARQUEMENT (`biped_board_vehicle`) : réfs en domaines 2, 3, 7 puis R(6) siège. AUCUNE
//     sonde (elle n'existe que pour le domaine 1). La réf 0 est l'occupant, slot = base +
//     index(8).
func decodeVehicleEvent(pay []byte, base uint32, inBand map[uint32]bool) (VehicleEvent, bool) {
	typ, present := PacketHeadEventType(pay)
	if !present || (typ != EventBipedBoardVehicle && typ != EventUnitExitVehicle) {
		return VehicleEvent{}, false
	}
	ev := VehicleEvent{Kind: typ}
	var seatBit int
	if typ == EventUnitExitVehicle {
		seatBit = decodeExitRefs(pay, base, inBand, &ev)
	} else {
		seatBit = decodeBoardRefs(pay, base, inBand, &ev)
	}
	if seatBit+vehicleSeatBits <= len(pay)*8 {
		ev.Seat = readBitsAt(pay, seatBit, vehicleSeatBits)
		ev.SeatValid = true
	}
	return ev, true
}

// decodeExitRefs lit les trois références gardées d'une SORTIE et rend le bit du siège.
// Domaines lus dans l'exécutable (vtable+0x58 du descripteur `unit_exit_vehicle`, 0x14080a018) :
// réf 0 -> domaine 1, réf 1 -> domaine 1, réf 2 -> domaine 7. C'est exactement la grammaire
// validée par la mesure (occupant en bande 95,5 %, siège 0 sur 93,8 %).
//
// LES DEUX RÉFÉRENCES DE DOMAINE 1 SONT LES DEUX UNITÉS DE LA SCÈNE, et elles ne se confondent
// pas : la réf 0 est l'OCCUPANT (100 % en bande bipède), la réf 1 est le VÉHICULE (105 / 105 en
// bande `ti=40`, zéro bipède — V7 § 7). La réf 1 était lue et JETÉE jusqu'au lot V8 ; elle est
// désormais publiée, et c'est elle qui nomme le véhicule d'un épisode d'occupation.
func decodeExitRefs(pay []byte, base uint32, inBand map[uint32]bool, ev *VehicleEvent) int {
	r0 := readDom1Ref(pay, eventPayloadStartBit)
	if r0.Present {
		ev.OccupantPresent = true
		ev.OccupantSonde = r0.Sonde
		if r0.Sonde == 1 {
			ev.OccupantSlot = base + r0.Index
		} else {
			ev.OccupantSlot = r0.Index
		}
		ev.OccupantInBand = inBand[ev.OccupantSlot]
	}
	r1 := readDom1Ref(pay, r0.EndBit)
	if r1.Present {
		ev.VehicleSlotValid = true
		ev.VehicleGen = r1.Gen
		if r1.Sonde == 1 {
			ev.VehicleSlot = base + r1.Index
		} else {
			ev.VehicleSlot = r1.Index
		}
	}
	r2 := readPlainRef(pay, r1.EndBit, dom7RefWidth)
	return r2.EndBit
}

// boardRefs lit les TROIS références gardées d'un EMBARQUEMENT (`biped_board_vehicle`).
//
// LES DOMAINES VIENNENT DE L'EXÉCUTABLE, PAS D'UNE DEVINETTE (lecture Ghidra du 2026-09-02).
// Le dispatcher (FUN_14080AADE) lit, pour chaque réf i de 0 à 2, un bit de garde puis appelle
// `vtable+0x58 (this, i)` qui rend le DOMAINE de cette réf, puis l'id-reader `FUN_1406d3140`.
// Le descripteur de l'embarquement est la vtable 0x143d0d330 (son thunk de nom, vtable+0x08 =
// 0x14119e9b0, pointe la chaîne « biped_board_vehicle » en 0x143c97f80) ; son vtable+0x58 vaut
// 0x142f1556c, dont le code est un simple aiguillage sur l'index de réf :
//
//	test edx,edx ; je  -> mov eax,2 ; ret      réf 0 -> DOMAINE 2
//	sub  edx,1   ; je  -> mov eax,3 ; ret      réf 1 -> DOMAINE 3
//	                     mov eax,7 ; ret       réf 2 -> DOMAINE 7
//
// C'était l'inconnue du portage du 2026-09-01, qui lisait la réf 0 en domaine 1 (avec sonde) et
// les deux suivantes en domaine 7 : trois domaines faux sur trois. `FUN_1406d3140` ne lit la
// SONDE que pour le domaine 1 (`if (param_3 == 1 && ReadBit())`), donc AUCUNE des trois réfs de
// l'embarquement n'en porte — d'où la « sonde variable » observée à la mesure, qui n'était que
// le premier bit de l'index.
func boardRefs(pay []byte) (r0, r1, r2 guardedRef) {
	r0 = readPlainRef(pay, eventPayloadStartBit, dom2RefWidth)
	r1 = readPlainRef(pay, r0.EndBit, dom3RefWidth)
	r2 = readPlainRef(pay, r1.EndBit, dom7RefWidth)
	return r0, r1, r2
}

// decodeBoardRefs remplit l'occupant d'un EMBARQUEMENT et rend le bit du siège. L'occupant est la
// réf 0 (domaine 2), rapportée à la base de la bande bipède comme pour la sortie.
func decodeBoardRefs(pay []byte, base uint32, inBand map[uint32]bool, ev *VehicleEvent) int {
	r0, _, r2 := boardRefs(pay)
	if r0.Present {
		ev.OccupantPresent = true
		ev.OccupantSlot = base + r0.Index
		ev.OccupantInBand = inBand[ev.OccupantSlot]
	}
	return r2.EndBit
}

// ScanFilmVehicleEvents décode tous les événements d'embarquement / sortie de véhicule des
// chunks du film de dir. Il relève d'abord la bande de slots bipèdes (keyframes ti=35) pour
// résoudre l'occupant, puis balaie les paquets delta dont l'événement de tête est board/exit.
// Les chunks illisibles sont ignorés (film partiel) ; erreur seulement si aucun chunk lisible.
func ScanFilmVehicleEvents(dir string) ([]VehicleEvent, error) {
	n := CountFilmChunks(dir)
	chunks := make([]int, 0, n)
	for i := 1; i <= n; i++ {
		chunks = append(chunks, i)
	}
	band := map[uint32]bool{}
	base := uint32(0)
	if len(chunks) > 0 {
		band = bipedSlotBand(dir, chunks)
		base = ^uint32(0)
		for s := range band {
			if s < base {
				base = s
			}
		}
		if len(band) == 0 {
			base = 0
		}
	}
	var out []VehicleEvent
	read := 0
	for _, c := range chunks {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		read++
		for _, p := range WalkPackets(data) {
			if p.Type != PacketTypeDelta || p.Size < 1 {
				continue
			}
			ev, ok := decodeVehicleEvent(p.Payload(data), base, band)
			if !ok {
				continue
			}
			ev.Chunk, ev.PacketIndex, ev.TimestampUS = c, p.Index, p.TimestampUS
			out = append(out, ev)
		}
	}
	if read == 0 {
		return nil, fmt.Errorf("aucun chunk film lisible dans %s", dir)
	}
	return out, nil
}
