// Package grammar — highlight_events.go : lecture binaire du chunk des temps forts.
//
// # D OU IL VIENT (lot 2.5.e, 2026-09-16, decision V15 (4))
//
// Ce lecteur vivait dans `internal/analysis/highlight_event_parser.go`, c est-a-dire dans le
// paquet title-agnostic : de la grammaire de film posee la ou aucun octet de film n a sa place.
// L ADR 0034 veut une seule porte aux octets et une seule maison pour la grammaire ; il descend
// donc ici SANS QU AUCUN APPELANT NE CHANGE DE TYPE — le lot 2.5.h avait deja fait remonter
// `HighlightEvent` en `domain/highlightevent`, ce qui etait son prerequis mesure (§4 D4).
//
// Ce qu il ne fait plus lui-meme : DECOMPRESSER et LIRE DES OCTETS. Le zlib passe par
// [source.Decompresser], l octet a un offset de bit par [source.OctetAuBit], les entiers par
// [source.U16LE] et [source.U32BE]. Il ne restait que ce fichier et son voisin `weaponscan` a
// porter leur propre copie.
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
package grammar

import (
	"errors"
	"fmt"
	"unicode/utf16"

	"github.com/JGtm/LevelUp/libs/film/highlightevent"
	"github.com/JGtm/LevelUp/libs/film/internal/source"
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
// filmMajorVersion a DEUX sources, et elles portent la même valeur : l'u32 little-endian en tête
// du registre du film (`chunk_00`, cf. `grammar.FilmMajorVersion`) et le `CustomData.FilmMajorVersion`
// du manifeste de spectate de l'API. Un appelant qui tient le film lit la première, un appelant
// du chemin en direct la seconde ; aucun ne la devine.
//
// 0 = INCONNUE (film sans registre, manifeste sans le champ) : le découpage historique « gamertag
// en tête » s'applique, et l'appelant a la charge de consigner la dégradation — sur un film de
// version 39-40 les gamertags rendus seraient du rembourrage.
//
// Retourne les événements parsés ; les events non reconnus sont silencieusement ignorés.
func ParseHighlightEvents(data []byte, filmMajorVersion int) ([]highlightevent.HighlightEvent, error) {
	if len(data) == 0 {
		return nil, nil
	}

	// [source.Decompresser] distingue les deux échecs par une sentinelle, et c'est exactement
	// la distinction que ce lecteur faisait lui-même : un EN-TÊTE qui n'est pas du zlib veut
	// dire « data est déjà en clair » et se traverse ; un flux zlib valide qui casse EN COURS
	// est une erreur à remonter.
	payload := data
	if inflated, err := source.Decompresser(data); err == nil {
		payload = inflated
	} else if !errors.Is(err, source.ErrEnTeteZlib) {
		return nil, fmt.Errorf("ParseHighlightEvents decompress: %w", err)
	}

	return scanHighlightEvents(payload, filmMajorVersion), nil
}

// scanHighlightEvents identifie chaque XUID dans le flux binaire (au bit près) et
// parse l'event associé. Retourne tous les events reconnus, ignore les non
// reconnus.
//
// `version` EST LUE, JAMAIS DEVINÉE (décision utilisateur du 2026-09-12). Le découpage du
// gamertag est le seul champ du bloc d'event qui en dépende, et l'indicateur qui le commande
// existe : `FilmMajorVersion`, l'u32 little-endian en tête du registre du film
// (`grammar.FilmMajorVersionFromHeader`), que l'API publie aussi dans son manifeste de
// spectate. Les appelants le lisent et le passent ; ce parseur ne mesure rien.
//
// `version` = 0 signifie « le film ne porte pas son registre » : le découpage historique
// « gamertag en tête » s'applique, et l'appelant a consigné la dégradation.
func scanHighlightEvents(data []byte, version int) []highlightevent.HighlightEvent {
	totalBits := len(data) * 8
	if totalBits < 80 {
		return nil
	}

	var events []highlightevent.HighlightEvent
	// Indices déjà traités (en bits) pour éviter les doublons.
	seenPositions := make(map[int]bool)
	// UN marqueur de fin = UN événement : pour chaque marqueur déjà consommé, l index dans
	// `events` et la position du XUID retenu. Voir [retenirLePlusProche].
	parMarqueur := make(map[int]candidatMarqueur)

	// Scan bit par bit : chercher l'octet 0xc0 (8 bits = 11000000), puis valider :
	//   bits[markerStart-8 : markerStart] = 0x2d ou 0x25
	//   bits[markerStart-72 : markerStart-8] = uint64 LE dans [minXUID..maxXUID]
	for markerStart := 8; markerStart <= totalBits-8; markerStart++ {
		if source.OctetAuBit(data, markerStart) != 0xc0 {
			continue
		}
		xuidEnd := markerStart - 8
		if xuidEnd < 64 {
			continue
		}
		prefix := source.OctetAuBit(data, xuidEnd)
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

		ev, endPos, err := parseEventAtBit(data, xuidStart, xuid, version)
		if err != nil {
			// Event non reconnu (type_hint inconnu, end-marker absent dans la
			// fenêtre, etc.) → skip silencieusement.
			continue
		}
		seenPositions[xuidStart] = true
		events = retenirLePlusProche(events, parMarqueur, ev, endPos, xuidStart)
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
func parseEventAtBit(data []byte, xuidStartBit int, xuid uint64, version int) (highlightevent.HighlightEvent, int, error) {
	totalBits := len(data) * 8
	windowEndBit := min(xuidStartBit+eventWindowBits, totalBits)

	searchFrom := xuidStartBit
	var lastErr error
	for {
		endPosBit := findBitMarker(data, searchFrom, windowEndBit, endMarker)
		if endPosBit < 0 {
			if lastErr != nil {
				return highlightevent.HighlightEvent{}, -1, lastErr
			}
			return highlightevent.HighlightEvent{}, -1, fmt.Errorf("end-marker absent dans la fenêtre")
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
			return ev, endPosBit, nil
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
func decodeEventBytes(b []byte, xuid uint64, version int) (highlightevent.HighlightEvent, error) {
	if len(b) < eventDataBytes {
		return highlightevent.HighlightEvent{}, fmt.Errorf("bloc event trop court: %d < 60", len(b))
	}

	var gamertag string
	if version <= 38 || version >= 41 {
		gamertag = decodeUTF16LE(b[0:32])
	} else {
		gamertag = decodeUTF16LE(b[12:44])
	}

	typeHint := int(b[47])
	// time_ms est un uint32 big-endian (bitstring.Bits.unpack("uint:32") = big-endian).
	timeMS := int(source.U32BE(b, 48))
	isMedal := b[55] == 1
	medalType := int(b[59])

	eventType, err := inferEventType(typeHint, isMedal)
	if err != nil {
		return highlightevent.HighlightEvent{}, err
	}

	return highlightevent.HighlightEvent{
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
		return highlightevent.EventTypeMedal, nil
	}
	switch typeHint {
	case typeHintMode:
		return highlightevent.EventTypeMode, nil
	case typeHintDeath:
		return highlightevent.EventTypeDeath, nil
	case typeHintKill:
		return highlightevent.EventTypeKill, nil
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
		u16[i] = source.U16LE(b, i*2)
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
// Lectures au bit — équivalent minimal de Python `bitstring.Bits`.
// Les positions s'expriment en bits ; l'octet à un bit-offset arbitraire est
// [source.OctetAuBit], qui porte EXACTEMENT la convention d'ici (MSB-first,
// ZÉRO quand l'octet ne tient pas entièrement dans le tampon). Ce fichier en
// portait sa propre copie jusqu'au lot 2.5.e.
// ─────────────────────────────────────────────────────────────────────────────

// readBytesAtBit lit n octets consécutifs à partir du bit-offset `bit`.
// Retourne nil si la lecture déborde.
func readBytesAtBit(data []byte, bit, n int) []byte {
	if bit < 0 || bit+n*8 > len(data)*8 {
		return nil
	}
	out := make([]byte, n)
	for i := range n {
		out[i] = source.OctetAuBit(data, bit+i*8)
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
	for i := range 8 {
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
		for i := range pattern {
			if source.OctetAuBit(data, bit+i*8) != pattern[i] {
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

// candidatMarqueur est le XUID retenu pour un marqueur de fin : l index de son événement dans
// la liste en cours et la position (en bits) du XUID.
type candidatMarqueur struct {
	index    int
	xuidBits int
}

// retenirLePlusProche applique « un marqueur de fin = un événement ». Deux motifs de XUID
// valides peuvent précéder le même marqueur dans la fenêtre (un faux motif dans les octets du
// bloc d une fiche, plusieurs milliers de bits avant le vrai) ; les deux décodent alors le même
// bloc de 60 octets et l événement sortirait deux fois, sous deux XUID. Le vrai XUID est celui
// qui précède le bloc : on garde le candidat le plus proche du marqueur (position de XUID la
// plus grande) et on écarte l autre.
func retenirLePlusProche(events []highlightevent.HighlightEvent, parMarqueur map[int]candidatMarqueur,
	ev highlightevent.HighlightEvent, endPos, xuidBits int) []highlightevent.HighlightEvent {
	if prev, vu := parMarqueur[endPos]; vu {
		if xuidBits > prev.xuidBits {
			events[prev.index] = ev
			parMarqueur[endPos] = candidatMarqueur{index: prev.index, xuidBits: xuidBits}
		}
		return events
	}
	parMarqueur[endPos] = candidatMarqueur{index: len(events), xuidBits: xuidBits}
	return append(events, ev)
}
