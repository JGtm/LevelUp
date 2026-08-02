package analysis

// weapon_scanner.go — Scan bas-niveau des films binaires Halo Infinite.
//
// Port de src/analysis/_weapon_scanners.py.
// Scanneurs Section 1 (Formula A) et Section 2 (Fire Events).

import (
	"bytes"
	"encoding/binary"
	"sort"
)

// ══════════════════════════════════════════════════════════════════════════════
//  Constantes de scan
// ══════════════════════════════════════════════════════════════════════════════

var (
	// FormulaAPattern est le marker Section 1 : [20 00 02].
	FormulaAPattern = []byte{0x20, 0x00, 0x02}

	// FrameMarker est le marker de position de frame : [A0 7B 42].
	FrameMarker = []byte{0xA0, 0x7B, 0x42}
)

const (
	weaponBitOffset  = 40 // bits depuis event_start → weapon_id (8 bytes, 64 bits)
	b5BitOffset      = 32 // bits depuis event_start → b5 (player_index<<4 | slot)
	b5DedupProximity = 2  // bytes — deux hits à ≤2 bytes = même event physique

	// pi5BitOffset / pi5Width — L'INDICE DE JOUEUR SUR SA LARGEUR RÉELLE : 5 bits à
	// event_start+31, soit le bit qui PRÉCÈDE l'octet b5. Le champ n'est pas ambigu, il est
	// TRONQUÉ par la lecture 4 bits (`b5 >> 4`), et la preuve est offline (949 films du cache,
	// population exacte de ScanFireEventsB5) :
	//
	//	                   roster <= 16 (823 films)     roster > 16 (116 films)
	//	bit emprunté à 1   1 sur 1 672 653 events       172 038 sur 542 992  (31,7 %)
	//	valeur max         (31,5)=20  (32,4)=7          (31,5)=26  (32,4)=15  ← saturation
	//	couverture roster  identique                    0,917 contre 0,615
	//
	// Sous 17 joueurs les deux lectures rendent la MÊME valeur (à un event près sur 1,6 M) ;
	// au-delà, la lecture 4 bits sature à 15 et FUSIONNE les joueurs d'indice >= 16 avec ceux
	// d'indice < 16. Corroboré par une autre structure du film : `weaponv3/pi_resolver.go`
	// déclare 5 bits pour le même espace d'indices.
	pi5BitOffset = 31
	pi5Width     = 5
)

// universalMarker = 0b10100100110 = 11 bits.
// Packed in 2 bytes : bits [0..10] = 101_00100110.
// We search bit-by-bit, so we store as uint16 and mask length.
const (
	universalMarkerBits = 0b10100100110
	universalMarkerLen  = 11
)

// ══════════════════════════════════════════════════════════════════════════════
//  Résultats
// ══════════════════════════════════════════════════════════════════════════════

// FormulaAResult est un snapshot arme Section 1.
type FormulaAResult struct {
	Offset      int
	PlayerIndex int
	WeaponBytes [8]byte
}

// FireEvent est un événement de tir scanné.
type FireEvent struct {
	TimestampMS float64
	// PlayerIndex : l'indice tel que le lit la CORRÉLATION D'ARMES DE PRODUCTION — 4 bits
	// (`b5 >> 4`). Il est conservé tel quel DÉLIBÉRÉMENT : le chemin qui le consomme
	// (`CorrelateKillsGlobal` via `backfill_weapons.go`) apparie l'indice du tueur avec
	// `getXuidToPI`, qui le dérive de l'ORDRE DE LA BASE — déclaré faux depuis la v3. Corriger
	// la largeur d'un champ NE RÉPARE PAS un appariement qui ne repose pas sur ce champ : mesuré
	// kill par kill, l'effet est STRICTEMENT NUL sous 17 joueurs (823 films, 72 077 kills, zéro
	// différence) et le gain au-delà exige de remplacer AUSSI `getXuidToPI`. Ce remplacement est
	// hors du périmètre de la session J4-2 ; il est consigné au plan.
	PlayerIndex int
	// PlayerIndex5 : LE MÊME indice sur sa largeur RÉELLE (5 bits, event_start+31). C'est celui
	// qu'écrit `shared.match_weapon_shots`, et il n'y a pas le choix : la lecture 4 bits SATURE
	// à 15 au-delà de 16 joueurs, donc elle FUSIONNERAIT les tirs de deux joueurs distincts sur
	// une même ligne. Une ligne fabriquée par troncature est pire qu'une ligne absente.
	PlayerIndex5 int
	Slot         int
	B5           int
	WeaponName   string
	WeaponBytes  [8]byte
	FireSeq      int
	FireCounter  int
	BytePos      int
	BurstEnd     bool
	HitLikely    *bool
	ChunkIdx     int // rempli par le parser
}

// ══════════════════════════════════════════════════════════════════════════════
//  Frame positions & timestamp estimation
// ══════════════════════════════════════════════════════════════════════════════

// FindFramePositions retourne toutes les positions du FrameMarker.
func FindFramePositions(data []byte) []int {
	var positions []int
	for pos := 0; ; {
		idx := bytes.Index(data[pos:], FrameMarker)
		if idx < 0 {
			break
		}
		positions = append(positions, pos+idx)
		pos += idx + 1
	}
	return positions
}

// TimestampEstimator retourne une fonction estimate(bytePos) → float64 ms.
func TimestampEstimator(data []byte, startMS, durationMS int) func(int) float64 {
	frames := FindFramePositions(data)
	nFrames := len(frames)
	var frameDurMS float64
	if nFrames > 0 {
		frameDurMS = float64(durationMS) / float64(nFrames)
	} else {
		frameDurMS = 16.67
	}
	return func(bytePos int) float64 {
		lo, hi := 0, nFrames-1
		frameIdx := 0
		for lo <= hi {
			mid := (lo + hi) / 2
			if frames[mid] <= bytePos {
				frameIdx = mid
				lo = mid + 1
			} else {
				hi = mid - 1
			}
		}
		return float64(startMS) + float64(frameIdx)*frameDurMS
	}
}

// ══════════════════════════════════════════════════════════════════════════════
//  Section 1 — Formula A (snapshot armes)
// ══════════════════════════════════════════════════════════════════════════════

// ScanFormulaA scanne les mises à jour d'état arme Section 1.
func ScanFormulaA(data []byte) []FormulaAResult {
	var results []FormulaAResult
	pos := 0
	for {
		idx := bytes.Index(data[pos:], FormulaAPattern)
		if idx < 0 || pos+idx+4 > len(data) {
			break
		}
		absPos := pos + idx
		pb := data[absPos+3]
		pi := int(pb >> 5)

		end := absPos + 68
		if end > len(data) {
			end = len(data)
		}

		bestSX := -1
		for suffix := range AllFormulaASuffixes {
			sxc := bytes.Index(data[absPos+4:end], suffix[:])
			if sxc < 0 {
				continue
			}
			sxc += absPos + 4 // absolute
			wsc := sxc - 4
			if wsc <= absPos+3 {
				continue
			}
			if suffix != CommonWeaponSuffix {
				var wb [8]byte
				copy(wb[:], data[wsc:wsc+8])
				if !WeaponBytesMap[wb] {
					continue
				}
			}
			if bestSX == -1 || sxc < bestSX {
				bestSX = sxc
			}
		}
		if bestSX >= 4 {
			ws := bestSX - 4
			if ws > absPos+3 {
				var wb [8]byte
				copy(wb[:], data[ws:ws+8])
				results = append(results, FormulaAResult{
					Offset:      absPos,
					PlayerIndex: pi,
					WeaponBytes: wb,
				})
			}
		}
		pos = absPos + 4
	}
	return results
}

// ScanFormulaANS scanne Section 1 dans la couche nibble-shiftée (TYPE IDs).
func ScanFormulaANS(data []byte) []FormulaAResult {
	if len(data) < 2 {
		return nil
	}
	// Construire la couche nibble-shiftée
	ns := make([]byte, len(data)-1)
	for i := 0; i < len(data)-1; i++ {
		ns[i] = (data[i] << 4) | (data[i+1] >> 4)
	}

	var results []FormulaAResult
	for wb := range WeaponBytesMap {
		if len(wb) != 8 {
			continue
		}
		pos := 0
		for {
			p := bytes.Index(ns[pos:], wb[:])
			if p < 0 {
				break
			}
			absP := pos + p
			if absP >= 5 && ns[absP-5] != 0x26 {
				pi := int(ns[absP-1] >> 5)
				results = append(results, FormulaAResult{
					Offset:      absP,
					PlayerIndex: pi,
					WeaponBytes: wb,
				})
			}
			pos = absP + 1
		}
	}
	// Trier par position
	sort.Slice(results, func(i, j int) bool { return results[i].Offset < results[j].Offset })
	return results
}

// ══════════════════════════════════════════════════════════════════════════════
//  Section 2 — Fire Events (marker universel b5>>4)
// ══════════════════════════════════════════════════════════════════════════════

// ScanFireEventsB5 scanne les fire events via le marker universel.
// Retourne les events triés par timestamp, dédupliqués par proximité byte.
func ScanFireEventsB5(data []byte, estimateTS func(int) float64) []FireEvent {
	totalBits := len(data) * 8
	var events []FireEvent

	// Scan bit par bit pour le marker universel (11 bits)
	for bitPos := 0; bitPos+universalMarkerLen+weaponBitOffset+64 <= totalBits; bitPos++ {
		if !matchMarkerAt(data, bitPos) {
			continue
		}

		eventStart := bitPos + 3 // après les 3 bits de préfixe "101"
		weaponStart := eventStart + weaponBitOffset
		b5Start := eventStart + b5BitOffset

		if weaponStart+64 > totalBits {
			continue
		}

		weaponInt := readBitsUint64(data, weaponStart, 64)
		var weaponBytes [8]byte
		binary.BigEndian.PutUint64(weaponBytes[:], weaponInt)

		if !WeaponIDs[weaponInt] && !hasSuffix(weaponBytes, CommonWeaponSuffix) {
			continue
		}

		b5Int := readBitsUint8(data, b5Start, 8)
		fireSeq := 0
		if eventStart+16 <= totalBits {
			fireSeq = int(readBitsUint8(data, eventStart+8, 8))
		}
		fireCounter := 0
		if eventStart+32 <= totalBits {
			fireCounter = int(readBitsUint8(data, eventStart+24, 8))
		}

		weaponName := WeaponIDToName[weaponInt]
		if weaponName == "" {
			weaponName = "INCONNU"
		}

		var burstEnd bool
		var hitLikely *bool
		postStart := weaponStart + 64
		if postStart+32 <= totalBits {
			postB1 := readBitsUint8(data, postStart+8, 8)
			postB2 := readBitsUint8(data, postStart+16, 8)
			burstEnd = (postB1 & 0x01) != 0
			hl := (postB2 & 0x01) == 0
			hitLikely = &hl
		}

		events = append(events, FireEvent{
			TimestampMS: estimateTS(bitPos / 8),
			PlayerIndex: int(b5Int >> 4),
			// Le champ 5 bits déborde d'un bit à GAUCHE de b5 : il se relit à sa position, pas
			// à partir de b5 (cf. pi5BitOffset).
			PlayerIndex5: int(readBitsUint8(data, eventStart+pi5BitOffset, pi5Width)),
			Slot:         int(b5Int & 0x03),
			B5:           int(b5Int),
			WeaponName:   weaponName,
			WeaponBytes:  weaponBytes,
			FireSeq:      fireSeq,
			FireCounter:  fireCounter,
			BytePos:      bitPos / 8,
			BurstEnd:     burstEnd,
			HitLikely:    hitLikely,
		})
	}

	// Dédupliquation par proximité byte_pos
	sort.Slice(events, func(i, j int) bool { return events[i].BytePos < events[j].BytePos })
	var deduped []FireEvent
	lastPos := -999
	for _, ev := range events {
		if ev.BytePos-lastPos > b5DedupProximity {
			deduped = append(deduped, ev)
			lastPos = ev.BytePos
		}
	}

	// Retrier par timestamp
	sort.Slice(deduped, func(i, j int) bool { return deduped[i].TimestampMS < deduped[j].TimestampMS })
	return deduped
}

// ══════════════════════════════════════════════════════════════════════════════
//  Helpers bit-level
// ══════════════════════════════════════════════════════════════════════════════

// matchMarkerAt vérifie si le marker universel 11 bits est à bitPos.
func matchMarkerAt(data []byte, bitPos int) bool {
	for i := 0; i < universalMarkerLen; i++ {
		byteIdx := (bitPos + i) / 8
		bitIdx := 7 - ((bitPos + i) % 8)
		bit := (data[byteIdx] >> uint(bitIdx)) & 1
		expected := (universalMarkerBits >> uint(universalMarkerLen-1-i)) & 1
		if int(bit) != expected {
			return false
		}
	}
	return true
}

// readBitsUint64 lit n bits depuis bitPos en big-endian.
func readBitsUint64(data []byte, bitPos, n int) uint64 {
	var result uint64
	for i := 0; i < n; i++ {
		byteIdx := (bitPos + i) / 8
		bitIdx := 7 - ((bitPos + i) % 8)
		bit := uint64((data[byteIdx] >> uint(bitIdx)) & 1)
		result = (result << 1) | bit
	}
	return result
}

// readBitsUint8 lit n bits (≤8) depuis bitPos en big-endian.
func readBitsUint8(data []byte, bitPos, n int) uint8 {
	var result uint8
	for i := 0; i < n; i++ {
		byteIdx := (bitPos + i) / 8
		bitIdx := 7 - ((bitPos + i) % 8)
		bit := (data[byteIdx] >> uint(bitIdx)) & 1
		result = (result << 1) | bit
	}
	return result
}

// hasSuffix vérifie si les 4 derniers bytes correspondent.
func hasSuffix(wb [8]byte, suffix [4]byte) bool {
	return wb[4] == suffix[0] && wb[5] == suffix[1] && wb[6] == suffix[2] && wb[7] == suffix[3]
}
