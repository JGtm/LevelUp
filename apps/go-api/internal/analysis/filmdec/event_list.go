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
	// OccupantSonde : la sonde de la réf 0 (domaine 1). 1 = index bipède-relatif (9 bits),
	// 0 = slot absolu (13 bits).
	OccupantSonde int
	// OccupantSlot est le slot de l'unité : base bipède + index si sonde=1, index brut sinon.
	OccupantSlot uint32
	// OccupantInBand : le slot tombe dans la bande de slots bipèdes du film (contrôle).
	OccupantInBand bool

	// Seat est le siège (R(6)) lu en fin de charge. SeatValid=false si le payload est trop court.
	Seat      uint32
	SeatValid bool
}

// decodeVehicleEvent décode l'événement de tête board/exit d'un payload. `base` est le début de
// la plage de slots bipèdes du film (min de la bande) ; `inBand` teste l'appartenance d'un slot
// à la bande. Rend ok=false si le type de tête n'est pas board/exit.
//
// GRAMMAIRE (mesurée sur corpus + dispatcher Ghidra) :
//   - ref0 = l'unité (domaine 1, sonde). Pour la SORTIE, sonde=1 sur 237/237 : l'occupant est
//     un bipède, slot = base + index(9). Validé : 95,6 % des occupants tombent dans la bande.
//   - SORTIE : ref1 (domaine 1) + ref2 (domaine 7) puis R(6) siège. Siège = 0 sur 224/237
//     (94,5 %) — « conducteur qui descend », conforme au « siège dominant 0 » mesuré.
//   - EMBARQUEMENT : ref1 + ref2 (domaine 7) puis R(6) siège. Siège = 16 dominant (mesuré ;
//     échantillon corpus réduit, board = 374 vs exit = 5 600). RÉF0 de l'embarquement porte une
//     sonde variable (souvent 0) : son occupant est moins net que celui de la sortie.
func decodeVehicleEvent(pay []byte, base uint32, inBand map[uint32]bool) (VehicleEvent, bool) {
	typ, present := PacketHeadEventType(pay)
	if !present || (typ != EventBipedBoardVehicle && typ != EventUnitExitVehicle) {
		return VehicleEvent{}, false
	}
	ev := VehicleEvent{Kind: typ}
	ref0 := readDom1Ref(pay, eventPayloadStartBit)
	if ref0.Present {
		ev.OccupantPresent = true
		ev.OccupantSonde = ref0.Sonde
		if ref0.Sonde == 1 {
			ev.OccupantSlot = base + ref0.Index
		} else {
			ev.OccupantSlot = ref0.Index
		}
		ev.OccupantInBand = inBand[ev.OccupantSlot]
	}
	// Position du siège : après ref0, deux références gardées puis R(6). La SORTIE lit ref1 en
	// domaine 1 (sonde), l'EMBARQUEMENT en domaine 7 — départage mesuré par la valeur du siège
	// (sortie -> 0, embarquement -> 16).
	var seatBit int
	switch typ {
	case EventUnitExitVehicle:
		r1 := readDom1Ref(pay, ref0.EndBit)
		r2 := readPlainRef(pay, r1.EndBit, dom7RefWidth)
		seatBit = r2.EndBit
	default: // EventBipedBoardVehicle
		r1 := readPlainRef(pay, ref0.EndBit, dom7RefWidth)
		r2 := readPlainRef(pay, r1.EndBit, dom7RefWidth)
		seatBit = r2.EndBit
	}
	if seatBit+vehicleSeatBits <= len(pay)*8 {
		ev.Seat = readBitsAt(pay, seatBit, vehicleSeatBits)
		ev.SeatValid = true
	}
	return ev, true
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
