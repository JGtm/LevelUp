// Package analysis — highlight_event_parser.go : parsing binaire du chunk highlight events.
//
// Port Go de spnkr/film/highlight_events.py (acurtis166/SPNKr).
// Le chunk highlight events (ChunkType=3) est le dernier chunk du manifest film Halo.
// Il est zlib-compressé et contient un enregistrement binaire par événement de match
// (kills, deaths, medals, mode events).
//
// Le flux décompressé est packé au bit, pas à l'octet : les marqueurs `0xc0`,
// `0x2d|0x25` et l'end-marker `0x00 00 2e e0` peuvent commencer à n'importe
// quel offset de bit. Un scan byte-aligné rate ~88% des événements (les XUIDs
// sont distribués uniformément sur les 8 alignements possibles).
//
// Algorithme :
//  1. Décompresser zlib.
//  2. Scanner le flux bit par bit pour trouver les XUIDs : pattern
//     [...64 bits XUID LE...][8 bits 0x2d ou 0x25][8 bits 0xc0].
//  3. Pour chaque XUID candidat : chercher l'end-marker `0x00 00 2e e0` au
//     bit près dans une fenêtre de 20_000 bits qui suit, lire les 60 octets
//     d'event qui le précèdent (au bit près également).
//  4. Décoder l'event selon la version du film.
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

	// eventWindowBits est la taille de la fenêtre d'analyse autour d'un XUID.
	// Reproduit la valeur Python `selected = bits[start : start + 20_000]`.
	eventWindowBits = 20_000

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

	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("ParseHighlightEvents zlib: %w", err)
	}
	defer r.Close()
	decompressed, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("ParseHighlightEvents decompress: %w", err)
	}

	return scanEvents(decompressed, filmMajorVersion), nil
}

// scanEvents identifie chaque XUID dans le flux binaire (au bit près) et
// parse l'event associé. Retourne tous les events reconnus, ignore les non
// reconnus.
func scanEvents(data []byte, version int) []HighlightEvent {
	totalBits := len(data) * 8
	if totalBits < 80 {
		return nil
	}

	var events []HighlightEvent
	// Indices déjà traités (en bits) pour éviter les doublons.
	seenPositions := make(map[int]bool)

	// Scan bit par bit : chercher l'octet 0xc0 (8 bits = 11000000), puis valider :
	//   bits[markerStart-8 : markerStart] = 0x2d ou 0x25
	//   bits[markerStart-72 : markerStart-8] = uint64 LE dans [minXUID..maxXUID]
	for markerStart := 8; markerStart <= totalBits-8; markerStart++ {
		if readByteAtBit(data, markerStart) != 0xc0 {
			continue
		}
		xuidEnd := markerStart - 8
		if xuidEnd < 64 {
			continue
		}
		prefix := readByteAtBit(data, xuidEnd)
		if prefix != 0x2d && prefix != 0x25 {
			continue
		}
		xuidStart := xuidEnd - 64
		if seenPositions[xuidStart] {
			continue
		}

		xuid := readUint64LEAtBit(data, xuidStart)
		if xuid <= minXUID || xuid >= maxXUID {
			continue
		}

		ev, err := parseEventAtBit(data, xuidStart, xuid, version)
		if err != nil {
			// Event non reconnu (type_hint inconnu, end-marker absent dans la
			// fenêtre, etc.) → skip silencieusement.
			continue
		}
		events = append(events, ev)
		seenPositions[xuidStart] = true
	}
	return events
}

// parseEventAtBit parse l'event situé à xuidStartBit dans le flux binaire
// décompressé. Toutes les positions sont en bits.
//
// Le marqueur de fin `0x00 00 2e e0` ne fait que 32 bits ; sur des flux qui
// contiennent des zones de zéros, plusieurs faux positifs bit-shiftés peuvent
// apparaître avant le vrai end-marker. On itère donc sur toutes les positions
// du marqueur dans la fenêtre et on retourne le premier event décodable
// (type_hint reconnu). Le Python upstream prend le premier match aveuglément
// — équivalent en pratique sur de la vraie data, mais cette version est plus
// robuste sur des flux synthétiques ou bruités.
func parseEventAtBit(data []byte, xuidStartBit int, xuid uint64, version int) (HighlightEvent, error) {
	totalBits := len(data) * 8
	windowEndBit := xuidStartBit + eventWindowBits
	if windowEndBit > totalBits {
		windowEndBit = totalBits
	}

	searchFrom := xuidStartBit
	var lastErr error
	for {
		endPosBit := findBitMarker(data, searchFrom, windowEndBit, endMarker)
		if endPosBit < 0 {
			if lastErr != nil {
				return HighlightEvent{}, lastErr
			}
			return HighlightEvent{}, fmt.Errorf("end-marker absent dans la fenêtre")
		}

		eventBitsStart := endPosBit - eventDataBytes*8
		if eventBitsStart < xuidStartBit {
			// End-marker trop proche du XUID — chercher après.
			searchFrom = endPosBit + 1
			lastErr = fmt.Errorf("données event insuffisantes avant le end-marker")
			continue
		}

		eventBytes := readBytesAtBit(data, eventBitsStart, eventDataBytes)
		if eventBytes == nil {
			searchFrom = endPosBit + 1
			lastErr = fmt.Errorf("lecture event bytes hors limites")
			continue
		}
		ev, err := decodeEventBytes(eventBytes, xuid, version)
		if err == nil {
			return ev, nil
		}
		// type_hint inconnu / event corrompu : c'était un faux positif du
		// scanner d'end-marker. Continuer après cette position.
		lastErr = err
		searchFrom = endPosBit + 1
	}
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
		gamertag = decodeUTF16LE(b[0:32])
	} else {
		gamertag = decodeUTF16LE(b[12:44])
	}

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

// decodeUTF16LE décode une chaîne utf-16le et retire les nulls.
// La tranche doit contenir les octets bruts de la gamertag (32 octets = 16 chars max).
func decodeUTF16LE(b []byte) string {
	if len(b) < 2 {
		return ""
	}
	u16 := make([]uint16, len(b)/2)
	for i := range u16 {
		u16[i] = binary.LittleEndian.Uint16(b[i*2 : i*2+2])
	}
	for i, c := range u16 {
		if c == 0 {
			u16 = u16[:i]
			break
		}
	}
	return string(utf16.Decode(u16))
}

// ─────────────────────────────────────────────────────────────────────────────
// Bit-level reader — équivalent minimal de Python `bitstring.Bits`.
// Les positions s'expriment en bits ; la lecture d'un octet à un bit-offset
// arbitraire concatène les bits hauts de data[byteIdx] et les bits bas de
// data[byteIdx+1] (convention MSB-first, identique à bitstring).
// ─────────────────────────────────────────────────────────────────────────────

// readByteAtBit lit 8 bits consécutifs commençant à la position `bit` (MSB-first).
// Retourne 0 si la position est hors limites.
func readByteAtBit(data []byte, bit int) byte {
	if bit < 0 || bit+8 > len(data)*8 {
		return 0
	}
	byteIdx := bit / 8
	off := uint(bit % 8)
	if off == 0 {
		return data[byteIdx]
	}
	hi := data[byteIdx] << off
	lo := data[byteIdx+1] >> (8 - off)
	return hi | lo
}

// readBytesAtBit lit n octets consécutifs à partir du bit-offset `bit`.
// Retourne nil si la lecture déborde.
func readBytesAtBit(data []byte, bit, n int) []byte {
	if bit < 0 || bit+n*8 > len(data)*8 {
		return nil
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = readByteAtBit(data, bit+i*8)
	}
	return out
}

// readUint64LEAtBit lit 64 bits à partir de `bit` et les interprète comme un
// uint64 little-endian (équivalent Python `bitstring.Bits.uintle` sur 64 bits :
// les 8 octets sont lus dans l'ordre, le premier octet étant le LSB).
func readUint64LEAtBit(data []byte, bit int) uint64 {
	b := readBytesAtBit(data, bit, 8)
	if b == nil {
		return 0
	}
	var x uint64
	for i := 0; i < 8; i++ {
		x |= uint64(b[i]) << (uint(i) * 8)
	}
	return x
}

// findBitMarker cherche `pattern` à toutes les positions de bit dans
// data[startBit : endBit]. Retourne la position du début du pattern (en bits)
// ou -1 si absent.
func findBitMarker(data []byte, startBit, endBit int, pattern []byte) int {
	if startBit < 0 {
		startBit = 0
	}
	totalBits := len(data) * 8
	if endBit > totalBits {
		endBit = totalBits
	}
	patBits := len(pattern) * 8
	if patBits == 0 {
		return -1
	}
	for bit := startBit; bit <= endBit-patBits; bit++ {
		match := true
		for i := 0; i < len(pattern); i++ {
			if readByteAtBit(data, bit+i*8) != pattern[i] {
				match = false
				break
			}
		}
		if match {
			return bit
		}
	}
	return -1
}
