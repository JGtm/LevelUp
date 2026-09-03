package filmdec

// transloc_events.go — LA TÉLÉPORTATION DU TRANSLOCATEUR, LUE DANS LES ÉVÉNEMENTS DU FILM.
//
// CE QUE C'EST. Chaque usage du translocateur émet un événement type 117
// `EquipmentTranslocatorTeleportEffects` (nom sourcé de l'exécutable, lecteur 0x140f04fb8)
// dans la liste d'événements en tête d'un paquet delta — le même canal que `unit_zoom`
// (zoom_events.go), qui est le patron de ce fichier. L'événement porte la référence du
// bipède qui saute et précède la discontinuité de position de 5 à 80 ms.
//
// LA MESURE QUI FONDE LE FICHIER (rapport R1 §4, 2026-09-03, 5 films) : précision 18/18
// (chaque tête 117 correspond à une discontinuité vérifiée de 3,24 à 25,36 m du slot
// désigné), rappel 8/8 sur les téléportations connues — y compris un saut de 3,24 m
// invisible à tout seuil spatial. AUCUNE heuristique : l'événement EST la mesure.
//
// LA FORME DU RECORD (mesurée sur pièces, R1 §4.2) : premier octet 0xFA = bit de
// configuration (1) + bit de présence (1) + les 6 bits hauts du type ; le bit de poids
// faible du type est le premier bit du 2e octet (117 = 1110101). Puis ref0 = l'unité qui se
// téléporte : porte 1 bit, index 8 bits (domaine 2), génération 2 bits — l'index plus 512
// donne le SLOT du bipède, la même fermeture que le zoom (base mesurée, 18/18 slots
// plausibles). Les refs 1 et 2 sont absentes (portes à 0) sur toutes les observations, et ne
// sont pas lues : le slot suffit.
//
// RÉSERVE HONNÊTE, la même que zoom_events.go : seuls les événements EN TÊTE de paquet sont
// lus. Trois `spent` du corpus n'ont aucune tête 117 de leur slot (expiration sans usage, ou
// événement hors tête de liste — non départagé, R1 §4.3). Lire la liste entière est le
// chantier PLAN_PERCER_TRAME_FILM.

import "sort"

const (
	// translocFamilyByte : le premier octet des paquets dont l'événement de tête est le
	// type 117 — config(1) + présence(1) + les 6 bits hauts de 1110101.
	translocFamilyByte = 0xFA
	// translocEventType : le numéro de type de `EquipmentTranslocatorTeleportEffects` sous
	// la numérotation du dispatcher.
	translocEventType = 117
	// translocRefWidth : largeur de l'index de ref0 (domaine 2 de la table des domaines).
	translocRefWidth = 8
	// translocSlotBase : la base du domaine — même fermeture mesurée que zoomSlotBase.
	translocSlotBase = 512
)

// TranslocatorTeleport est UNE téléportation exécutée : quand, et par quel bipède.
type TranslocatorTeleport struct {
	// TimestampUS : l'horodatage du paquet porteur — MÊME horloge que
	// BipedPosition.TimestampUS (l'horloge MOTEUR des paquets, cf. le piège documenté au
	// rapport R1 §0 : elle n'est PAS la timeline de l'artefact).
	TimestampUS uint64
	// Slot : le slot du bipède qui se téléporte, directement comparable à
	// BipedPosition.Slot.
	Slot uint32
}

// ScanFilmTranslocatorTeleports lit les téléportations du translocateur d'un film, triées
// par instant.
//
// Lecture seule, sans état global de décodage : comme le zoom, ce scanner n'appelle aucun
// déserialiseur de trame et n'a pas besoin du verrou de paquet. Les chunks illisibles sont
// une couverture moindre, pas une erreur.
func ScanFilmTranslocatorTeleports(dir string) []TranslocatorTeleport {
	var out []TranslocatorTeleport
	for c := 1; c <= CountFilmChunks(dir); c++ {
		chunk, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(chunk) {
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			pay := pk.Payload(chunk)
			if pay[0] != translocFamilyByte {
				continue
			}
			if ev, ok := decodeTranslocHead(pay, pk.TimestampUS); ok {
				out = append(out, ev)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TimestampUS < out[j].TimestampUS })
	return out
}

// decodeTranslocHead lit l'événement de tête d'un paquet de la famille et rend la
// téléportation. ok=false si la tête n'est pas un type 117 ou si l'unité n'est pas désignée
// (porte de ref0 à 0 — jamais observé, mais un slot non transmis ne se devine pas).
func decodeTranslocHead(pay []byte, tsUS uint64) (TranslocatorTeleport, bool) {
	br := NewBitReader(pay)
	br.Skip(1) // bit de configuration
	if !br.ReadBit() {
		return TranslocatorTeleport{}, false // liste vide : pas d'événement en tête
	}
	if int(br.ReadBits(7)) != translocEventType {
		return TranslocatorTeleport{}, false
	}
	if !br.ReadBit() {
		return TranslocatorTeleport{}, false // ref0 absente : pas d'unité désignée
	}
	idx := br.ReadBits(translocRefWidth)
	br.Skip(2) // génération
	return TranslocatorTeleport{
		TimestampUS: tsUS,
		Slot:        uint32(idx + translocSlotBase),
	}, true
}
