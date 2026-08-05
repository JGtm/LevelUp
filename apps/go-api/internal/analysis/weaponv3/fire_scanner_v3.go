package weaponv3

// fire_scanner_v3.go — scanner de fire-events v3 à largeur de player_index
// CONFIGURABLE (débloquer le BTB, réf PLAN §8).
//
// PROBLÈME (plan §8) : la v2 (analysis.ScanFireEventsB5) lit le player_index du
// fire-event sur 4 BITS (le nibble HAUT de l'octet b5 = byte5>>4, 0-15). Or le
// résolveur xuid→pi v3 (pi_resolver.go) lit 5 BITS (0-31). Sur les matchs >16
// joueurs (BTB), les tueurs avec pi∈[16,31] ne matchent JAMAIS un fire-event pi
// plafonné à 15 dans analysis.CorrelateKillsGlobal (ev.PlayerIndex != killerPI)
// → leur tir n'est pas réclamé → fallback formula_a → NULL/low. C'est le levier
// NULL #1 sur le BTB (04023f8a : 65 % NULL, fire=21 vs formula_a=42 mesuré).
//
// CE SCANNER porte FIDÈLEMENT analysis.ScanFireEventsB5 (même marqueur 11-bit,
// même offset weapon-id à +40 bits, même dédup par proximité d'octet, mêmes
// post-bits burst/hit) et NE CHANGE QUE la lecture du player_index : il relit le
// champ pi avec une largeur/layout paramétrable (FirePiBits) pour que les tueurs
// pi>15 réclament enfin leur fire-event.
//
// C'est de la RE : le 5-bit pi est une HYPOTHÈSE. La largeur gagnante est
// déterminée par MESURE (chute du NULL BTB SANS régression Arena), pas par
// déduction théorique. Le défaut de l'orchestrateur (correlate.go) reste le
// comportement v2 (piBits=0 → ScanFireEventsB5) tant que la mesure ne valide pas
// un layout 5-bit.
//
// Réfs : .ai/PLAN_WEAPON_ATTRIBUTION_V3.md §8 (pi-fix), §9 (recall fire),
// .ai/RESEARCH_THEATER_RE.md §L (marqueur fire 0b101 0010 0110, weapon @+40).

import (
	"encoding/binary"
	"sort"

	"levelup/go-api/internal/analysis"
)

const (
	// Offsets BITS relatifs à eventStart (= bitPos du marqueur + 3 bits "101"),
	// alignés sur analysis.ScanFireEventsB5 (b5BitOffset=32, weaponBitOffset=40).
	fireB5BitOffset     = 32 // octet b5 (player_index<<4 | slot) — début du nibble pi
	fireWeaponBitOffset = 40 // weapon-id 64-bit
	fireDedupProximity  = 2  // octets — deux hits à ≤2 octets = même event physique

	// fireMarkerBits / fireMarkerLen — marqueur universel 11 bits (identique à
	// analysis.universalMarkerBits, non exporté → redéclaré ici).
	fireMarkerBits = 0b10100100110
	fireMarkerLen  = 11

	// fireMarker3Bits / fireMarker3Len — les 3 BITS de tête constants du marqueur
	// (§9 : « only the final 3 bits are constant for the event start »). En recall
	// relâché, on n'exige QUE ces 3 bits puis on valide a posteriori par canon
	// high-32 pour écarter les faux positifs.
	fireMarker3Bits = 0b101
	fireMarker3Len  = 3
)

// FirePiLayout — stratégie de lecture du player_index dans/autour de l'octet b5.
//
// b5 occupe les bits [eventStart+32, eventStart+40). Le weapon-id suit à +40. Les
// layouts testés (RE, plan §8) lisent le pi sur 4 ou 5 bits à différents endroits.
type FirePiLayout int

const (
	// FirePi4High — BASELINE v2 : nibble HAUT de b5 (byte5>>4), 4 bits, 0-15.
	FirePi4High FirePiLayout = iota
	// FirePi5HighInB5 — 5 bits HAUTS de b5 (byte5>>3 & 0x1f), bits [32,37).
	FirePi5HighInB5
	// FirePi5LowInB5 — 5 bits BAS de b5 (byte5 & 0x1f), bits [35,40).
	FirePi5LowInB5
	// FirePi5SpanBefore — 5 bits chevauchant le bit AVANT b5, bits [31,36) : le
	// nibble pi 4-bit historique PLUS le bit de poids fort emprunté à b5-1. C'est
	// l'extension naturelle « le pi déborde d'un bit vers le haut quand >16 joueurs ».
	FirePi5SpanBefore
)

// ScanFireEventsV3 scanne les fire-events comme analysis.ScanFireEventsB5 mais lit
// le player_index selon `layout` (4 ou 5 bits). `relax3` active le recall relâché
// (§9 : marqueur 3-bit au lieu de 11-bit + validation canon high-32 anti-FP).
//
// Renvoie des analysis.FireEvent (mêmes champs) pour alimenter directement
// analysis.CorrelateKillsGlobal. Le Slot reste byte5&0x03 (inchangé v2).
func ScanFireEventsV3(data []byte, estimateTS func(int) float64, layout FirePiLayout, relax3 bool) []analysis.FireEvent {
	totalBits := len(data) * 8
	var events []analysis.FireEvent

	markerLen := fireMarkerLen
	if relax3 {
		markerLen = fireMarker3Len
	}

	for bitPos := 0; bitPos+markerLen+fireWeaponBitOffset+64 <= totalBits; bitPos++ {
		if !matchFireMarkerAt(data, bitPos, relax3) {
			continue
		}
		// Le préfixe "101" occupe les 3 premiers bits du marqueur 11-bit ; en
		// recall relâché le marqueur EST ces 3 bits → eventStart = bitPos+3 dans
		// les deux cas (offsets weapon/b5 ancrés sur la même origine que la v2).
		eventStart := bitPos + fireMarker3Len
		weaponStart := eventStart + fireWeaponBitOffset
		if weaponStart+64 > totalBits {
			continue
		}

		weapon := fireWeapon{id: readFireBitsU64(data, weaponStart, 64)}
		binary.BigEndian.PutUint64(weapon.bytes[:], weapon.id)
		if !fireWeaponAccepted(weapon.id, weapon.bytes, relax3) {
			continue
		}

		site := fireSite{bitPos: bitPos, eventStart: eventStart, weaponStart: weaponStart, totalBits: totalBits}
		events = append(events, buildFireEvent(data, site, weapon, layout, estimateTS))
	}

	return dedupAndSortFires(events)
}

// fireSite regroupe les quatre positions de bits d'un fire-event candidat. Elles pointent
// toutes dans le MÊME flux et sont toutes des `int` : passées à la file, deux d'entre elles
// s'intervertissent sans que le compilateur bronche.
type fireSite struct {
	bitPos      int // début du marqueur (origine de l'horodatage)
	eventStart  int // bitPos + longueur du préfixe "101"
	weaponStart int // eventStart + fireWeaponBitOffset
	totalBits   int // borne du flux
}

// fireWeapon porte l'identifiant d'arme sous ses deux formes : l'entier lu et ses 8 octets
// big-endian (les deux sont attendus par analysis.FireEvent).
type fireWeapon struct {
	id    uint64
	bytes [8]byte
}

// buildFireEvent assemble un analysis.FireEvent à partir des positions de bits
// résolues. Seul le player_index dépend du layout ; le reste (fireSeq, counter,
// burst/hit) reprend exactement la v2.
func buildFireEvent(
	data []byte, site fireSite, weapon fireWeapon,
	layout FirePiLayout, estimateTS func(int) float64,
) analysis.FireEvent {
	eventStart, totalBits := site.eventStart, site.totalBits
	b5Int := readFireBitsU8(data, eventStart+fireB5BitOffset, 8)
	pi := readFirePI(data, eventStart, b5Int, layout)

	fireSeq := 0
	if eventStart+16 <= totalBits {
		fireSeq = int(readFireBitsU8(data, eventStart+8, 8))
	}
	fireCounter := 0
	if eventStart+32 <= totalBits {
		fireCounter = int(readFireBitsU8(data, eventStart+24, 8))
	}

	weaponName := analysis.WeaponIDToName[weapon.id]
	if weaponName == "" {
		weaponName = "INCONNU"
	}

	var burstEnd bool
	var hitLikely *bool
	postStart := site.weaponStart + 64
	if postStart+32 <= totalBits {
		postB1 := readFireBitsU8(data, postStart+8, 8)
		postB2 := readFireBitsU8(data, postStart+16, 8)
		burstEnd = (postB1 & 0x01) != 0
		hl := (postB2 & 0x01) == 0
		hitLikely = &hl
	}

	return analysis.FireEvent{
		TimestampMS: estimateTS(site.bitPos / 8),
		PlayerIndex: pi,
		Slot:        int(b5Int & 0x03),
		B5:          int(b5Int),
		WeaponName:  weaponName,
		WeaponBytes: weapon.bytes,
		FireSeq:     fireSeq,
		FireCounter: fireCounter,
		BytePos:     site.bitPos / 8,
		BurstEnd:    burstEnd,
		HitLikely:   hitLikely,
	}
}

// readFirePI lit le player_index selon le layout choisi. b5 = readFireBitsU8 à
// eventStart+32 (déjà lu par l'appelant pour Slot/B5). FirePi5SpanBefore lit 5 bits
// directement dans le flux (le bit AVANT b5 + les 4 bits hauts de b5).
func readFirePI(data []byte, eventStart int, b5 uint8, layout FirePiLayout) int {
	switch layout {
	case FirePi5HighInB5:
		return int(b5 >> 3) // bits [32,37)
	case FirePi5LowInB5:
		return int(b5 & 0x1f) // bits [35,40)
	case FirePi5SpanBefore:
		return int(readFireBitsU8(data, eventStart+fireB5BitOffset-1, 5)) // bits [31,36)
	case FirePi4High:
		fallthrough
	default:
		return int(b5 >> 4) // BASELINE v2 : bits [32,36)
	}
}

// fireWeaponAccepted reproduit le filtre d'arme v2 (WeaponIDs OU suffixe commun).
// En recall relâché (§9), on DURCIT : on exige un high-32 connu (CanonWeaponID) car
// le marqueur 3-bit sur-déclenche massivement et seul le canon écarte les FP.
func fireWeaponAccepted(weaponInt uint64, weaponBytes [8]byte, relax3 bool) bool {
	if relax3 {
		_, known := CanonWeaponID(weaponInt)
		return known
	}
	return analysis.WeaponIDs[weaponInt] || fireHasCommonSuffix(weaponBytes)
}

// fireHasCommonSuffix vérifie le suffixe commun 0x42c9679f (4 octets bas), comme
// le hasSuffix(…, CommonWeaponSuffix) non exporté de weapon_scanner.go.
func fireHasCommonSuffix(wb [8]byte) bool {
	return wb[4] == analysis.CommonWeaponSuffix[0] &&
		wb[5] == analysis.CommonWeaponSuffix[1] &&
		wb[6] == analysis.CommonWeaponSuffix[2] &&
		wb[7] == analysis.CommonWeaponSuffix[3]
}

// dedupAndSortFires reproduit la dédup-par-proximité-octet + le tri final par
// timestamp d'analysis.ScanFireEventsB5.
func dedupAndSortFires(events []analysis.FireEvent) []analysis.FireEvent {
	sort.Slice(events, func(i, j int) bool { return events[i].BytePos < events[j].BytePos })
	deduped := events[:0:0]
	lastPos := -999
	for _, ev := range events {
		if ev.BytePos-lastPos > fireDedupProximity {
			deduped = append(deduped, ev)
			lastPos = ev.BytePos
		}
	}
	sort.Slice(deduped, func(i, j int) bool { return deduped[i].TimestampMS < deduped[j].TimestampMS })
	return deduped
}

// matchFireMarkerAt vérifie le marqueur fire à bitPos (11 bits strict, ou 3 bits
// "101" en recall relâché). Réécrit la logique non exportée de weapon_scanner.go.
func matchFireMarkerAt(data []byte, bitPos int, relax3 bool) bool {
	bits, n := fireMarkerBits, fireMarkerLen
	if relax3 {
		bits, n = fireMarker3Bits, fireMarker3Len
	}
	for i := 0; i < n; i++ {
		byteIdx := (bitPos + i) / 8
		bitIdx := 7 - ((bitPos + i) % 8)
		bit := int((data[byteIdx] >> uint(bitIdx)) & 1)
		expected := (bits >> uint(n-1-i)) & 1
		if bit != expected {
			return false
		}
	}
	return true
}

// readFireBitsU64 lit n bits depuis bitPos en big-endian (port readBitsUint64).
func readFireBitsU64(data []byte, bitPos, n int) uint64 {
	var result uint64
	for i := 0; i < n; i++ {
		byteIdx := (bitPos + i) / 8
		bitIdx := 7 - ((bitPos + i) % 8)
		result = (result << 1) | uint64((data[byteIdx]>>uint(bitIdx))&1)
	}
	return result
}

// readFireBitsU8 lit n bits (≤8) depuis bitPos en big-endian (port readBitsUint8).
func readFireBitsU8(data []byte, bitPos, n int) uint8 {
	var result uint8
	for i := 0; i < n; i++ {
		byteIdx := (bitPos + i) / 8
		bitIdx := 7 - ((bitPos + i) % 8)
		result = (result << 1) | (data[byteIdx]>>uint(bitIdx))&1
	}
	return result
}
