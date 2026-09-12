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

// ParseHighlightEvents parse le chunk highlight events binaire Halo.
//
// Le chunk peut arriver dans 2 états selon la source :
//   - **Fresh download via halo_client.downloadBlob** : déjà décompressé (le
//     CDN renvoie du zlib brut que downloadBlob inflate depuis fix 9cc4c2bb
//     du 8 mai 2026). data est en clair.
//   - **Cached via LocalFilmCache.LoadChunk** : encore en zlib (le cache
//     hérité du projet Python stocke les chunks compressés tels que reçus
//     du CDN avant le fix). data commence par 0x78 0x.. (header zlib).
//
// On gère les 2 formats : on tente une décompression zlib, si elle échoue
// avec un header invalide on suppose que data est déjà en clair. Cette
// double-tolérance résout l'incident 2026-05-22 (41/41 matchs JGtm failed
// avec "zlib: invalid header" sur les fresh downloads, alors que les
// matchs anciens cachés parsaient OK).
//
// filmMajorVersion provient de CustomData.FilmMajorVersion dans le manifest.
// Retourne les événements parsés ; les events non reconnus sont silencieusement ignorés.
func ParseHighlightEvents(data []byte, filmMajorVersion int) ([]HighlightEvent, error) {
	if len(data) == 0 {
		return nil, nil
	}

	payload := data
	if r, err := zlib.NewReader(bytes.NewReader(data)); err == nil {
		// Path cache : data était zlib-compressé, on décompresse.
		defer r.Close()
		inflated, inflErr := io.ReadAll(r)
		if inflErr != nil {
			return nil, fmt.Errorf("ParseHighlightEvents decompress: %w", inflErr)
		}
		payload = inflated
	}
	// Sinon (err != nil sur zlib.NewReader = pas un header zlib) : data est
	// déjà en clair (path fresh download post-fix 9cc4c2bb). Pas d'erreur.

	return scanEvents(payload, filmMajorVersion), nil
}

// Versions d'implantation du bloc d'event. Elles ne servent qu'à NOMMER les deux découpages
// possibles du gamertag — voir decodeEventBytes pour le découpage lui-même.
const (
	// versionUnknown est la valeur que porte un film dont le manifeste ne dit pas la version.
	// `halo_client_film.go` la pose explicitement (« legacy cache n'a pas la version »), et
	// trois appelants la passent en dur faute de manifeste (killsource, replay/deaths_source,
	// ops/medal_feed_backfill). Elle ne DÉSIGNE aucune implantation : elle déclenche la
	// résolution par mesure (voir gamertagLayoutAlternatif).
	versionUnknown = 0
	// versionGamertagEnTete est une version dont le gamertag vit à b[0:32] (<= 38 ou >= 41).
	versionGamertagEnTete = 41
	// versionGamertagDecale est une version dont le gamertag vit à b[12:44] (39-40).
	versionGamertagDecale = 39
)

// scanEvents identifie chaque XUID dans le flux binaire (au bit près) et
// parse l'event associé. Retourne tous les events reconnus, ignore les non
// reconnus.
//
// VERSION INCONNUE = RÉSOLUTION PAR MESURE, et c'est le correctif du 2026-09-12. Le découpage
// du gamertag est le SEUL champ du bloc d'event qui dépende de la version ; l'acceptation d'un
// event (type_hint, end-marker) n'en dépend pas. Un appelant sans manifeste passait donc 0,
// c'est-à-dire l'implantation « en tête », sur TOUS les films — y compris ceux des versions
// 39-40, où le gamertag est décalé de 12 octets. Le symptôme mesuré sur quatre films Big Team
// Battle de mars à octobre 2025 : 2 gamertags distincts au lieu de 24 à 27, donc un roster
// humain effondré en aval. On décode donc les deux découpages et on retient celui qui rend le
// plus de gamertags distincts.
func scanEvents(data []byte, version int) []HighlightEvent {
	totalBits := len(data) * 8
	if totalBits < 80 {
		return nil
	}

	auto := version == versionUnknown
	if auto {
		version = versionGamertagEnTete
	}

	var events []HighlightEvent
	// alternatif[i] : le gamertag de events[i] sous l'AUTRE découpage. Rempli seulement quand
	// la version est inconnue — sinon le manifeste tranche et il n'y a rien à mesurer.
	var alternatif []string
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

		ev, brut, err := parseEventAtBit(data, xuidStart, xuid, version)
		if err != nil {
			// Event non reconnu (type_hint inconnu, end-marker absent dans la
			// fenêtre, etc.) → skip silencieusement.
			continue
		}
		events = append(events, ev)
		if auto {
			alternatif = append(alternatif, gamertagPourVersion(brut, versionGamertagDecale))
		}
		seenPositions[xuidStart] = true
	}
	if auto && gamertagLayoutAlternatif(events, alternatif) {
		for i := range events {
			events[i].Gamertag = alternatif[i]
		}
	}
	return events
}

// gamertagLayoutAlternatif : l'AUTRE découpage rend-il un meilleur roster que celui retenu ?
//
// CRITÈRE : le nombre de gamertags DISTINCTS non vides. Un découpage juste rend un nom par
// joueur (24 à 27 sur un Big Team Battle) ; un découpage faux lit du rembourrage, qui est le
// MÊME pour tous les events et ne rend qu'une poignée de valeurs. Le contraste mesuré le
// 2026-09-12 sur sept films est de 2 contre 24-27 — il n'y a pas de zone grise à arbitrer.
// Départage, si les deux découpages rendent autant de noms distincts : le nombre d'events
// nommés, puis le découpage déjà retenu (l'alternatif ne gagne jamais une égalité parfaite).
func gamertagLayoutAlternatif(events []HighlightEvent, alternatif []string) bool {
	if len(alternatif) != len(events) || len(events) == 0 {
		return false
	}
	distinctsRetenu, nommesRetenu := map[string]bool{}, 0
	distinctsAlt, nommesAlt := map[string]bool{}, 0
	for i, ev := range events {
		if ev.Gamertag != "" {
			distinctsRetenu[ev.Gamertag] = true
			nommesRetenu++
		}
		if alternatif[i] != "" {
			distinctsAlt[alternatif[i]] = true
			nommesAlt++
		}
	}
	if len(distinctsAlt) != len(distinctsRetenu) {
		return len(distinctsAlt) > len(distinctsRetenu)
	}
	return nommesAlt > nommesRetenu
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
//
// Elle rend AUSSI les 60 octets retenus : la résolution du découpage du gamertag (version
// inconnue) les relit sous l'autre implantation, et les re-scanner coûterait une seconde passe
// complète sur le chunk.
func parseEventAtBit(data []byte, xuidStartBit int, xuid uint64, version int) (HighlightEvent, []byte, error) {
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
				return HighlightEvent{}, nil, lastErr
			}
			return HighlightEvent{}, nil, fmt.Errorf("end-marker absent dans la fenêtre")
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
			return ev, eventBytes, nil
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

	gamertag := gamertagPourVersion(b, version)

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

// gamertagPourVersion : le gamertag des 60 octets d'event, sous le découpage de `version`.
// Les deux découpages sont ceux documentés par decodeEventBytes ; ils sont isolés ici parce que
// la résolution d'une version inconnue doit pouvoir relire le MÊME bloc sous l'autre.
func gamertagPourVersion(b []byte, version int) string {
	if len(b) < eventDataBytes {
		return ""
	}
	if version <= 38 || version >= 41 {
		return decodeUTF16LE(b[0:32])
	}
	return decodeUTF16LE(b[12:44])
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
