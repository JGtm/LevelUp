package grammar

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

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
// événement, l'octet 0 du payload vaut `0xC0 | (type >> 1)`. En lecture MSB-first (Lecteur) :
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

// packetHead est le PRÉAMBULE DE 9 BITS d'un paquet delta, décodé : [config(1)][continuation(1)]
// [R(7) type]. C'est la seule forme du préambule dans ce paquet — elle était écrite SIX fois en
// ligne, sous deux conventions différentes, avant le 2026-09-05 (lot E, item E.3).
type packetHead struct {
	// Config : bit 0, le drapeau de configuration (1 sur 100 % du corpus).
	Config bool
	// More : bit 1, la continuation. Faux = liste d'événements VIDE (le paquet est une trame
	// de records pure) ; le champ Type n'a alors aucun sens grammatical.
	More bool
	// Type : les 7 bits de type de l'événement de tête. LU DANS TOUS LES CAS, y compris quand
	// More est faux — voir readPacketHead pour la raison, qui est un invariant de compatibilité.
	Type int
}

// readPacketHead lit le préambule de 9 bits sur `br` et le laisse positionné au PREMIER BIT DU
// CORPS de l'événement (soit `eventPayloadStartBit` quand `br` partait du bit 0).
//
// IL LIT TOUJOURS LES NEUF BITS, y compris quand la continuation est nulle. C'est ce qui rend
// cette factorisation BIT-EXACTE pour les six appelants d'origine, qui se répartissaient en deux
// conventions :
//
//   - TROIS testaient la continuation et sortaient sans lire le type (`decodeZoomHead`,
//     `decodeTranslocHead`, `decodeBipedPickup`). Ils abandonnent leur lecteur au même instant,
//     donc les 7 bits lus en trop ne sont observés par personne.
//   - TROIS la SAUTAIENT et lisaient le type quoi qu'il arrive (`modalPostCountsBit`,
//     et les deux balayages de `weapon_hits.go`). Chacun est précédé d'un filtre sur l'OCTET DE
//     TÊTE du payload — 0xD2 pour le type 36, 0xC0 pour le type 0 — dont le bit 1 vaut 1 : la
//     continuation y est donc posée par construction, et ne pas la tester ne leur coûte rien.
//
// DEPUIS LE LOT M4b (2026-09-24), LE RECORD DE TIR TESTE LA CONTINUATION : sa tête est lue par
// [lireEnteteTir36], qui exige une liste d'événements (`More`) et le type 36 ENTIER — l'octet 0xD2
// laissait passer le type 37. Le harnais `writeModalHeader` écrit désormais le préfixe du film
// (configuration et continuation à 1). Le balayage des dégâts de `weapon_hits.go` garde son filtre
// sur l'octet de tête.
func readPacketHead(br *Lecteur) packetHead {
	var h packetHead
	h.Config = br.ReadBit()
	h.More = br.ReadBit()
	h.Type = int(br.ReadBits(eventTypeBits))
	return h
}

// PacketHeadEventType lit le type de l'ÉVÉNEMENT DE TÊTE d'un payload de paquet delta. Si la
// continuation est nulle, la liste est vide (le paquet est une trame de records pure) et present
// vaut false. Aucune charge n'est décodée : O(1), sans allocation au-delà du reader.
//
// C'est le CADRAGE MINIMAL de la liste : il suffit à compter les familles par type de tête
// (validation des comptes corpus) sans porter la grammaire de charge de chaque type.
func PacketHeadEventType(pay []byte) (typ int, present bool) {
	if len(pay) < 1 {
		return 0, false
	}
	h := readPacketHead(LecteurSur(pay))
	if !h.More {
		return 0, false
	}
	return h.Type, true
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

// refDomWidth rend la largeur (bits) de l'index d'une référence gardée du DOMAINE `dom`.
//
// C'EST LA SEULE TABLE DES DOMAINES DU PAQUET, et elle l'est depuis le 2026-09-05 (lot E, item
// E.3). Elle en remplace trois qui coexistaient — `dom7RefWidth`/`dom2RefWidth`/`dom3RefWidth`
// ici, `lot1RefDomWidths` (weapon_hits_decode.go) et `zoomRefWidth` (zoom_events.go) —, et deux
// d'entre elles CONTREDISAIENT la valeur mesurée du domaine 3 en portant `3: 8`. Un garde-rail
// (`event_preamble_guard_test.go`) interdit qu'une quatrième renaisse.
//
// LES LARGEURS SONT RUNTIME EN TOUTE RIGUEUR — `FUN_1406d310c(count)` rend ceil(log2(count)) sur
// une table initialisée à l'exécution, illisible dans l'image statique. Les valeurs ci-dessous
// sont celles de la build de référence, et les deux domaines de l'EMBARQUEMENT ont chacun leur
// oracle propre (balayage de largeurs, event_list_board_test.go, 8 films / 22 embarquements) :
//
//   - domaine 2 = 8 : l'occupant tombe dans la bande bipède 22/22 = 100 % et ouvre un trou du flux
//     de position à l'instant de l'événement dans 77,3 % des cas (témoin décalé : 0 %). À 7 bits le
//     recoupement s'effondre à 4,5 %, à 9 bits la bande n'est plus tenue (72,7 %), à 13 bits 9,1 %.
//   - domaine 3 = 7 : départagé par le SIÈGE, qui se lit après les trois réfs. À 7 bits le siège de
//     l'embarquement égale celui de la sortie appariée dans 5/6 = 83,3 % des cas et vaut 0 sur 21/22
//     (même « siège dominant 0 » que la sortie) ; à 8 bits l'accord tombe à 0/6, à 13 bits 4/6.
//     LA PROSE DE L'EXÉCUTABLE DIT 8 ; la mesure dit 7, et c'est la mesure qui fait foi. Les deux
//     tables recopiées portaient 8 : aucune ne servait le domaine 3 en production, sans quoi le
//     décalage d'un bit aurait emporté tout le corps de l'événement.
//
// Le domaine 1 est le seul à porter une SONDE (1 bit) qui abaisse sa largeur de 13 à 9 :
// `FUN_1406d3140` ne la lit que pour lui (`if (param_3 == 1 && ReadBit())`). Elle n'est PAS dans
// cette table, qui ne rend que la largeur de l'index ; ses lecteurs sont `readDom1Ref` et
// `lot1RefDom1`.
//
// UN DOMAINE HORS TABLE REND 0, exactement comme les cartes qu'elle remplace : une clé absente y
// valait le zéro du type, donc `ReadBits(0)` — zéro bit consommé.
func refDomWidth(dom int) uint {
	switch dom {
	case 0, 1, 7, 8:
		return dom7RefWidth
	case 2, 5:
		return dom2RefWidth
	case 3:
		return dom3RefWidth
	case 4, 6:
		return dom4RefWidth
	}
	return 0
}

// Les largeurs elles-mêmes, une déclaration par valeur. `refDomWidth` est le SEUL endroit qui les
// associe à un numéro de domaine ; ces noms restent parce que la grammaire d'un événement se lit
// mieux « domaine 2 » que « 8 », et parce que les instruments de mesure les nomment.
const (
	// dom7RefWidth : domaines 0, 1, 7 et 8. Runtime en toute rigueur (cf. `FrameConfig.IDLowBits`,
	// 11..14 selon le film), mais 13 vaut sur les films de référence.
	dom7RefWidth = 13
	// dom2RefWidth : domaines 2 et 5.
	dom2RefWidth = 8
	// dom3RefWidth : domaine 3. MESURÉE 7 (cf. l'oracle du siège ci-dessus), contre 8 dans la
	// prose de l'exécutable.
	dom3RefWidth = 7
	// dom4RefWidth : domaines 4 et 6.
	dom4RefWidth = 9
)

// guardedRef est une référence d'entité gardée décodée.
type guardedRef struct {
	Present bool
	Sonde   int    // domaine 1 uniquement : 1 => index sur 9 bits, 0 => 13 bits
	Index   uint32 // index brut (à additionner à la base du domaine pour obtenir le slot)
	Gen     uint32 // génération du handle (2 bits)
	EndBit  int    // bit juste après la référence
	// Tronquee : la référence déborde du payload (lot J2.9, constats GA1-1/GB-2, 2026-09-26).
	// Rien n'est lu au-delà du dernier bit, Present reste faux, EndBit pointe APRÈS la fin du
	// payload — les lectures qui suivent sont donc tronquées à leur tour — et l'événement qui la
	// porte est refusé par son décodeur.
	Tronquee bool
}

// readDom1Ref lit une référence gardée de DOMAINE 1 (bipède/unité) à partir du bit `at` :
// garde(1) ; si 1 : sonde(1) ; R(sonde?9:13) index ; R(2) génération.
//
// BORNÉE (lot J2.9, 2026-09-26) : une référence qui déborde du payload est rendue TRONQUÉE
// ([refTronquee]) au lieu de faire paniquer `readBitsAt`. Une référence qui tient est lue
// exactement comme avant.
func readDom1Ref(pay []byte, at int) guardedRef {
	fin := len(pay) * 8
	if at < 0 || at >= fin {
		return refTronquee(pay)
	}
	r := guardedRef{EndBit: at + 1}
	if readBitsAt(pay, at, 1) == 0 {
		return r
	}
	b := at + 1
	if b >= fin {
		return refTronquee(pay)
	}
	sonde := int(readBitsAt(pay, b, 1))
	b++
	w := 13
	if sonde == 1 {
		w = 9
	}
	if b+w+2 > fin {
		return refTronquee(pay)
	}
	r.Present, r.Sonde = true, sonde
	r.Index = readBitsAt(pay, b, w)
	b += w
	r.Gen = readBitsAt(pay, b, 2)
	r.EndBit = b + 2
	return r
}

// readPlainRef lit une référence gardée SANS sonde (domaines 2..8) de largeur w : garde(1) ;
// si 1 : R(w) index ; R(2) génération. Bornée comme [readDom1Ref].
func readPlainRef(pay []byte, at, w int) guardedRef {
	fin := len(pay) * 8
	if at < 0 || at >= fin {
		return refTronquee(pay)
	}
	r := guardedRef{EndBit: at + 1}
	if readBitsAt(pay, at, 1) == 0 {
		return r
	}
	b := at + 1
	if b+w+2 > fin {
		return refTronquee(pay)
	}
	r.Present = true
	r.Index = readBitsAt(pay, b, w)
	b += w
	r.Gen = readBitsAt(pay, b, 2)
	r.EndBit = b + 2
	return r
}

// refTronquee rend la référence d'un payload trop court : absente, marquée tronquée, et finie
// APRÈS le dernier bit — la référence suivante, lue à partir de là, l'est donc aussi, et le
// décodeur de l'événement n'a qu'à regarder la DERNIÈRE pour savoir si une seule a débordé.
func refTronquee(pay []byte) guardedRef {
	return guardedRef{Tronquee: true, EndBit: len(pay)*8 + 1}
}

// --- Événements véhicule (embarquement / sortie) ----------------------------------------------

// vehicleSeatBits est la largeur du champ SIÈGE dans la charge d'un événement véhicule : R(6).
const vehicleSeatBits = 6

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
func decodeVehicleEvent(pay []byte, base uint32, inBand SlotBand) (types.VehicleEvent, bool) {
	typ, present := PacketHeadEventType(pay)
	if !present || (typ != EventBipedBoardVehicle && typ != EventUnitExitVehicle) {
		return types.VehicleEvent{}, false
	}
	ev := types.VehicleEvent{Kind: typ}
	var derniere guardedRef
	if typ == EventUnitExitVehicle {
		derniere = decodeExitRefs(pay, base, inBand, &ev)
	} else {
		derniere = decodeBoardRefs(pay, base, inBand, &ev)
	}
	if derniere.Tronquee {
		// Une référence a débordé du payload (lot J2.9) : l'événement est REFUSÉ, jamais publié
		// avec un occupant ou un véhicule absent qui serait un fait plausible et faux.
		return types.VehicleEvent{}, false
	}
	seatBit := derniere.EndBit
	if seatBit+vehicleSeatBits <= len(pay)*8 {
		ev.Seat = readBitsAt(pay, seatBit, vehicleSeatBits)
		ev.SeatValid = true
	}
	return ev, true
}

// decodeExitRefs lit les trois références gardées d'une SORTIE et rend la DERNIÈRE : le siège
// se lit à son `EndBit`, et elle est tronquée dès qu'une des trois l'est ([refTronquee]).
// Domaines lus dans l'exécutable (vtable+0x58 du descripteur `unit_exit_vehicle`, 0x14080a018) :
// réf 0 -> domaine 1, réf 1 -> domaine 1, réf 2 -> domaine 7. C'est exactement la grammaire
// validée par la mesure (occupant en bande 95,5 %, siège 0 sur 93,8 %).
//
// LES DEUX RÉFÉRENCES DE DOMAINE 1 SONT LES DEUX UNITÉS DE LA SCÈNE, et elles ne se confondent
// pas : la réf 0 est l'OCCUPANT (100 % en bande bipède), la réf 1 est le VÉHICULE (105 / 105 en
// bande `ti=40`, zéro bipède — V7 § 7). La réf 1 était lue et JETÉE jusqu'au lot V8 ; elle est
// désormais publiée, et c'est elle qui nomme le véhicule d'un épisode d'occupation.
func decodeExitRefs(pay []byte, base uint32, inBand SlotBand, ev *types.VehicleEvent) guardedRef {
	r0 := readDom1Ref(pay, eventPayloadStartBit)
	if r0.Present {
		ev.OccupantPresent = true
		ev.OccupantSonde = r0.Sonde
		if r0.Sonde == 1 {
			ev.OccupantSlot = base + r0.Index
		} else {
			ev.OccupantSlot = r0.Index
		}
		ev.OccupantInBand = inBand.Has(ev.OccupantSlot)
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
	return r2
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

// decodeBoardRefs remplit l'occupant d'un EMBARQUEMENT et rend sa DERNIÈRE référence (cf.
// [decodeExitRefs]). L'occupant est la réf 0 (domaine 2), rapportée à la base de la bande
// bipède comme pour la sortie.
func decodeBoardRefs(pay []byte, base uint32, inBand SlotBand, ev *types.VehicleEvent) guardedRef {
	r0, _, r2 := boardRefs(pay)
	if r0.Present {
		ev.OccupantPresent = true
		ev.OccupantSlot = base + r0.Index
		ev.OccupantInBand = inBand.Has(ev.OccupantSlot)
	}
	return r2
}

// ScanFilmVehicleEvents est l'ENVELOPPE D2, HORS PRODUCTION : elle charge le film puis appelle
// [ScanVehicleEvents]. La cuisson passe un contexte deja ouvert.
func ScanFilmVehicleEvents(dir string) ([]types.VehicleEvent, error) {
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		return nil, err
	}
	return ScanVehicleEvents(contexteDeBobine(film))
}

// ScanVehicleEvents décode tous les événements d'embarquement / sortie de véhicule des chunks
// d'un film DEJA CHARGE. La bande de slots bipèdes (keyframes ti=35) qui résout l'occupant est
// celle du CONTEXTE — relevée une seule fois pour tous les balayages (lot 2 de
// PLAN_CUISSON_PERF) ; le balayage parcourt ensuite les paquets delta dont l'événement de tête
// est board/exit. Les chunks illisibles sont ignorés (film partiel) ; erreur seulement si aucun
// chunk lisible.
func ScanVehicleEvents(fc *FilmContext) ([]types.VehicleEvent, error) {
	nums := fc.ChunkNumbers()
	band := fc.BipedSlots()
	base := uint32(0)
	if slots := band.Slots(); len(slots) > 0 {
		base = slots[0] // la bande est rendue en ordre croissant : le premier EST le minimum
	}
	var out []types.VehicleEvent
	read := 0
	for _, c := range nums {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		read++
		for _, p := range pks {
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
		return nil, ErrNoReadableFilmChunk
	}
	return out, nil
}
