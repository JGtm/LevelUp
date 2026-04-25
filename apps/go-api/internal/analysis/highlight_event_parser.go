// Package analysis — highlight_event_parser.go : parsing binaire du chunk highlight events.
//
// Port Go de spnkr/film/highlight_events.py (acurtis166/SPNKr).
// Le chunk highlight events (ChunkType=3) est le dernier chunk du manifest film Halo.
// Il est gzip-compressé et contient un enregistrement binaire par événement de match
// (kills, deaths, medals, mode events).
//
// Algorithme de scan :
//  1. Décompresser gzip
//  2. Scanner les XUIDs Xbox : chercher le marqueur 0xc0 précédé de 0x2d ou 0x25,
//     puis valider les 8 octets précédents comme uint64 LE dans [2e15..3e15].
//  3. Pour chaque XUID trouvé : lire une fenêtre de 2500 octets, trouver le marqueur
//     de fin 0x00002ee0, lire les 60 octets d'event qui précèdent.
//  4. Décoder l'event selon la version du film (gestion du décalage gamertag).
package analysis

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"unicode/utf16"
)

// Constantes du parseur binaire film Halo.
const (
	// minXUID et maxXUID délimitent la plage valide des XUIDs Xbox Live.
	minXUID = uint64(2e15)
	maxXUID = uint64(3e15)

	// eventWindowBytes est la taille de la fenêtre d'analyse autour d'un XUID.
	eventWindowBytes = 2500

	// eventDataBytes est la taille fixe du bloc d'event data précédant le marqueur de fin.
	eventDataBytes = 60

	// typeHintKill est la valeur de type_hint pour les kills.
	typeHintKill = 50
	// typeHintDeath est la valeur de type_hint pour les deaths.
	typeHintDeath = 20
	// typeHintMode est la valeur de type_hint pour les events de mode.
	typeHintMode = 10
)

// medalSortingWeights contient les valeurs de type_hint qui désignent des médailles.
// Source : spnkr/film/highlight_events.py::_MEDAL_SORTING_WEIGHTS.
var medalSortingWeights = map[int]bool{
	50: true, 51: true, 52: true, 100: true, 101: true,
	150: true, 200: true, 205: true, 210: true, 220: true,
	225: true, 230: true, 235: true, 240: true, 245: true, 250: true,
}

// endMarker est le marqueur de fin d'un bloc event dans le flux binaire.
var endMarker = []byte{0x00, 0x00, 0x2e, 0xe0}

// EventType* sont les valeurs possibles du champ HighlightEvent.EventType.
const (
	EventTypeKill  = "kill"
	EventTypeDeath = "death"
	EventTypeMedal = "medal"
	EventTypeMode  = "mode"
)

// HighlightEvent représente un événement parsé depuis le chunk highlight events.
type HighlightEvent struct {
	XUID      uint64
	Gamertag  string
	EventType string // EventTypeKill | EventTypeDeath | EventTypeMedal | EventTypeMode
	TypeHint  int
	IsMedal   bool
	TimeMS    int
	MedalType int
}

// ParseHighlightEvents décompresse et parse le chunk highlight events binaire Halo.
// filmMajorVersion provient de CustomData.FilmMajorVersion dans le manifest.
// Retourne les événements parsés ; les events non reconnus sont silencieusement ignorés.
func ParseHighlightEvents(data []byte, filmMajorVersion int) ([]HighlightEvent, error) {
	if len(data) == 0 {
		return nil, nil
	}

	// 1. Décompresser le gzip (zlib en Python = compress/zlib en Go).
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("ParseHighlightEvents zlib: %w", err)
	}
	defer r.Close()
	decompressed, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("ParseHighlightEvents decompress: %w", err)
	}

	// 2. Scanner pour trouver les XUIDs et parser les events.
	return scanEvents(decompressed, filmMajorVersion), nil
}

// scanEvents identifie chaque XUID dans le flux binaire et parse l'event associé.
// Retourne tous les events reconnus, ignore les non reconnus.
func scanEvents(data []byte, version int) []HighlightEvent {
	var events []HighlightEvent
	// Indices déjà traités pour éviter les doublons (même XUID à positions proches).
	seenPositions := make(map[int]bool)

	// Scan : chercher l'octet 0xc0, puis valider le contexte.
	// Structure attendue : [...8 bytes XUID...][0x2d ou 0x25][0xc0]
	for i := 9; i < len(data); i++ {
		if data[i] != 0xc0 {
			continue
		}
		marker := data[i-1]
		if marker != 0x2d && marker != 0x25 {
			continue
		}
		xuidStart := i - 9
		if seenPositions[xuidStart] {
			continue
		}

		xuid := binary.LittleEndian.Uint64(data[xuidStart : xuidStart+8])
		if xuid <= minXUID || xuid >= maxXUID {
			continue
		}

		ev, err := parseEventAt(data, xuidStart, xuid, version)
		if err != nil {
			// Event non reconnu (type_hint inconnu) → skip silencieusement.
			continue
		}
		events = append(events, ev)
		seenPositions[xuidStart] = true
	}
	return events
}

// parseEventAt parse l'event situé à xuidStart dans le flux binaire décompressé.
func parseEventAt(data []byte, xuidStart int, xuid uint64, version int) (HighlightEvent, error) {
	windowEnd := xuidStart + eventWindowBytes
	if windowEnd > len(data) {
		windowEnd = len(data)
	}
	window := data[xuidStart:windowEnd]

	// Trouver le marqueur de fin 0x00002ee0.
	endPos := bytes.Index(window, endMarker)
	if endPos < eventDataBytes {
		return HighlightEvent{}, fmt.Errorf("marqueur de fin absent ou données insuffisantes")
	}

	eventBytes := window[endPos-eventDataBytes : endPos]
	return decodeEventBytes(eventBytes, xuid, version)
}

// decodeEventBytes décode les 60 octets d'event selon la version du film.
// Deux layouts possibles selon la version :
//   - version <= 38 ou >= 41 : gamertag[0:32] | pad[32:47] | type_hint[47] | time_ms[48:52] | pad | is_medal[55] | pad | medal_type[59]
//   - version 39–40          : pad[0:12] | gamertag[12:44] | pad[44:47] | type_hint[47] | time_ms[48:52] | pad | is_medal[55] | pad | medal_type[59]
func decodeEventBytes(b []byte, xuid uint64, version int) (HighlightEvent, error) {
	if len(b) < eventDataBytes {
		return HighlightEvent{}, fmt.Errorf("bloc event trop court: %d < 60", len(b))
	}

	var gamertag string
	if version <= 38 || version >= 41 {
		// Layout A : gamertag en début de bloc (offset 0).
		gamertag = decodeUTF16LE(b[0:32])
	} else {
		// Layout B (version 39–40) : gamertag décalé de 12 octets.
		gamertag = decodeUTF16LE(b[12:44])
	}

	// Offsets communs aux deux layouts à partir de l'octet 47.
	typeHint := int(b[47])
	// time_ms est un uint32 big-endian (bitstring.Bits.unpack("uint:32") = big-endian).
	timeMS := int(binary.BigEndian.Uint32(b[48:52]))
	isMedal := b[55] == 1
	medalType := int(b[59])

	eventType, err := inferEventType(typeHint, isMedal)
	if err != nil {
		return HighlightEvent{}, err
	}

	return HighlightEvent{
		XUID:      xuid,
		Gamertag:  gamertag,
		EventType: eventType,
		TypeHint:  typeHint,
		IsMedal:   isMedal,
		TimeMS:    timeMS,
		MedalType: medalType,
	}, nil
}

// inferEventType déduit le type d'event depuis type_hint et isMedal.
// Priorité : medal > mode > death > kill.
// Retourne une erreur si la combinaison est inconnue.
func inferEventType(typeHint int, isMedal bool) (string, error) {
	if isMedal && medalSortingWeights[typeHint] {
		return EventTypeMedal, nil
	}
	switch typeHint {
	case typeHintMode:
		return EventTypeMode, nil
	case typeHintDeath:
		return EventTypeDeath, nil
	case typeHintKill:
		return EventTypeKill, nil
	}
	return "", fmt.Errorf("type_hint=%d isMedal=%v non reconnu", typeHint, isMedal)
}

// decodeUTF16LE décode une chaîne utf-16le (16 octets max par caractère) et retire les nulls.
// La tranche doit contenir les octets bruts de la gamertag (32 octets = 16 chars max).
func decodeUTF16LE(b []byte) string {
	if len(b) < 2 {
		return ""
	}
	u16 := make([]uint16, len(b)/2)
	for i := range u16 {
		u16[i] = binary.LittleEndian.Uint16(b[i*2 : i*2+2])
	}
	// Tronquer sur le premier null character.
	for i, c := range u16 {
		if c == 0 {
			u16 = u16[:i]
			break
		}
	}
	return string(utf16.Decode(u16))
}
