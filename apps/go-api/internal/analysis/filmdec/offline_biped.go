// Décodage 100 % OFFLINE des positions absolues des bipeds (joueurs) depuis les seuls
// chunks d'un film — zéro capture Cheat Engine.
//
// GRAMMAIRE DU RECORD BIPED (validée au quantum exact, 99,99 %, contre une table de
// vérité live ; cf. .ai/thought_log_replay.md) :
//
//	[1 préfixe=1][13 idLow = slot][2 tag][1 gate=0][1 maskSel=0][3 maskCount]
//	[6 bits x maskCount indices de composants, croissants, le premier = 0]
//	puis les composants ; offset(i0) = début + 21 + 6*maskCount
//	i0 = [5 bits de gate à 0][X sur 13 bits][Y sur 13 bits][Z sur 14 bits]
//
// Déquantification : v = min + step*(q + 0.5), step = (max-min)/2^w, bornes
// QuantRangeCEBiped (spécifiques à la map du film 000d5950).
package filmdec

import (
	"fmt"
	"sort"
)

// BipedTypeIndex est le typeIndex (ti) des entités biped (joueurs) dans le registre film.
const BipedTypeIndex = 35

// Largeurs de quantification des axes de la position absolue i0 d'un biped.
var bipedAxisWidths = [3]uint{13, 13, 14}

// Tailles (en bits) de la grammaire du record biped.
const (
	bipedHeaderBits = 21 // préfixe + slot + tag + gate + maskSel + maskCount
	bipedIndexBits  = 6  // un index de composant
	bipedI0GateBits = 5  // gate en tête de i0, toujours nul pour une absolue
	bipedI0Bits     = bipedI0GateBits + 13 + 13 + 14
	bipedMinMaskCnt = 2
	bipedMaxMaskCnt = 7
	bipedSlotBits   = 13
)

// BipedPosition est une position absolue de biped décodée offline.
type BipedPosition struct {
	// Slot est l'identifiant bas (13 bits) de l'entité. Il MIGRE aux respawns : un slot
	// correspond à UNE VIE, pas à un joueur (l'attribution slot -> joueur n'est pas faite).
	Slot uint32
	// Chunk / PacketIndex localisent l'échantillon dans le film.
	Chunk, PacketIndex int
	// TimestampUS est l'horodatage du paquet porteur, en microsecondes (horloge du film).
	TimestampUS uint64
	// X, Y, Z sont en unités monde ; Z est l'axe vertical (étages).
	X, Y, Z float32
	// Directions capturées dans le MÊME record (composants i1/i2 qui suivent i0), quand
	// ScanFilmOptions.CaptureDirs est actif. Cf. offline_aim.go.
	componentDirs
}

// ScanFilmOptions règle le balayage offline.
type ScanFilmOptions struct {
	// Chunks à balayer ; vide -> tous les chunks présents (chunk_01..chunk_NN).
	Chunks []int
	// RequireTag1 exige tag == 1 dans l'en-tête (filtre éprouvé sur 000d5950).
	RequireTag1 bool
	// DropSaturated rejette les positions dont un axe tombe dans le bucket extrême
	// (q == 0 ou q == 2^w-1) : ce sont des valeurs écrêtées, pas des positions réelles
	// (9 points sur 171 116 pour 000d5950, mais ils multipliaient le span Z par 8).
	DropSaturated bool
	// MaxSpeedMPS écarte les téléportations : une position dont la vitesse depuis la
	// précédente position ACCEPTÉE du même slot dépasse ce seuil est un faux positif du
	// balayage bit à bit (le motif d'en-tête peut coïncider avec du bruit). 0 = désactivé.
	MaxSpeedMPS float64
	// IsolationGapMS écarte les échantillons temporellement isolés dans leur slot (cf.
	// DropIsolated). 0 = désactivé.
	IsolationGapMS int
	// CaptureDirs décode en plus les composants i1 (vélocité) et i2 (forward/cap) qui
	// suivent i0 dans le même record. Ne change AUCUNE position émise (le curseur repart
	// toujours de la fin d'i0) : le décodage des directions est en lecture seule.
	CaptureDirs bool
}

// DefaultMaxSpeedMPS : borne haute très large (le déplacement le plus rapide d'un Spartan,
// grappin ou véhicule, reste sous ~35 m/s). Ne sert qu'à couper les aberrations, pas à
// lisser le mouvement.
const DefaultMaxSpeedMPS = 100

// maxRejectStreak : après ce nombre de rejets consécutifs sur un slot, on se réancre sur la
// position courante — sinon une ancre elle-même fausse condamnerait tout le reste du slot.
const maxRejectStreak = 3

// DefaultScanFilmOptions est le réglage validé sur 000d5950.
func DefaultScanFilmOptions() ScanFilmOptions {
	return ScanFilmOptions{
		RequireTag1:    true,
		DropSaturated:  true,
		MaxSpeedMPS:    DefaultMaxSpeedMPS,
		IsolationGapMS: DefaultIsolationGapMS,
	}
}

// ScanFilmBipedPositions décode toutes les positions absolues de bipeds des chunks du
// film situé dans dir. Les chunks illisibles sont ignorés (le film peut être partiel) ;
// une erreur n'est renvoyée que si AUCUN chunk n'a pu être lu.
func ScanFilmBipedPositions(dir string, opt ScanFilmOptions) ([]BipedPosition, error) {
	chunks := opt.Chunks
	if len(chunks) == 0 {
		n := CountFilmChunks(dir)
		for i := 1; i <= n; i++ {
			chunks = append(chunks, i)
		}
	}
	if len(chunks) == 0 {
		return nil, fmt.Errorf("aucun chunk film dans %s", dir)
	}
	slots := bipedSlotBand(dir, chunks)
	if len(slots) == 0 {
		return nil, fmt.Errorf("aucun slot biped (ti=%d) dans les keyframes de %s", BipedTypeIndex, dir)
	}
	var out []BipedPosition
	read := 0
	for _, c := range chunks {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		read++
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			for _, r := range ScanBipedRecords(pk.Payload(data), slots, opt) {
				r.Chunk, r.PacketIndex, r.TimestampUS = c, pk.Index, pk.TimestampUS
				out = append(out, r)
			}
		}
	}
	if read == 0 {
		return nil, fmt.Errorf("aucun chunk film lisible dans %s", dir)
	}
	return DropTeleports(DropIsolated(out, opt.IsolationGapMS), opt.MaxSpeedMPS), nil
}

// bipedSlotBand construit l'ensemble des slots biped plausibles : union des ti=35 des
// keyframes de tous les chunks balayés (+ le suivant, car un biped créé en cours de chunk
// n'apparaît que dans le keyframe d'après), trous comblés entre min et max — les slots
// biped sont alloués dans une bande contiguë, et un biped créé PUIS détruit à l'intérieur
// d'un chunk n'apparaît dans aucun keyframe.
func bipedSlotBand(dir string, chunks []int) map[uint32]bool {
	seen := map[uint32]bool{}
	scan := append(append([]int{}, chunks...), chunks[len(chunks)-1]+1)
	for _, c := range scan {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				if r.TI == BipedTypeIndex && r.Slot >= 0 {
					seen[uint32(r.Slot)] = true
				}
			}
			break
		}
	}
	return fillSlotBand(seen)
}

// fillSlotBand comble les trous entre le min et le max de l'ensemble (bande contiguë).
func fillSlotBand(s map[uint32]bool) map[uint32]bool {
	if len(s) == 0 {
		return s
	}
	keys := make([]uint32, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	out := make(map[uint32]bool, int(keys[len(keys)-1]-keys[0])+1)
	for k := keys[0]; k <= keys[len(keys)-1]; k++ {
		out[k] = true
	}
	return out
}

// ScanBipedRecords balaie un payload de paquet delta bit à bit et renvoie les positions
// absolues des records biped reconnus. PUR (aucune I/O) : c'est le cœur testable du
// décodeur. Les champs Chunk/PacketIndex/TimestampUS sont laissés à zéro (remplis par
// l'appelant).
func ScanBipedRecords(payload []byte, slots map[uint32]bool, opt ScanFilmOptions) []BipedPosition {
	total := len(payload) * 8
	minRecord := bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + bipedI0Bits
	var out []BipedPosition
	for p := 0; p+minRecord <= total; {
		i0, slot, idx, ok := matchBipedHeader(payload, p, total, slots, opt.RequireTag1)
		if !ok {
			p++
			continue
		}
		q := [3]uint32{
			readBitsAt(payload, i0+bipedI0GateBits, 13),
			readBitsAt(payload, i0+bipedI0GateBits+13, 13),
			readBitsAt(payload, i0+bipedI0GateBits+26, 14),
		}
		if opt.DropSaturated && saturatedQuantum(q) {
			p = i0 + bipedI0Bits
			continue
		}
		rec := BipedPosition{
			Slot: slot,
			X:    DequantBipedAxis(q[0], 0),
			Y:    DequantBipedAxis(q[1], 1),
			Z:    DequantBipedAxis(q[2], 2),
		}
		if opt.CaptureDirs {
			rec.componentDirs = scanRecordDirs(payload, i0+bipedI0Bits, total, idx)
			if recordMaskHook != nil {
				recordMaskHook(idx, payload, i0+bipedI0Bits)
			}
		}
		out = append(out, rec)
		p = i0 + bipedI0Bits // pas de re-scan chevauchant
	}
	return out
}

// matchBipedHeader teste la grammaire d'en-tête biped à la position bit p et renvoie
// l'offset bit de i0, le slot et la liste des index de composants du masque.
func matchBipedHeader(pay []byte, p, total int, slots map[uint32]bool, needTag1 bool) (int, uint32, []int, bool) {
	if readBitsAt(pay, p, 1) != 1 {
		return 0, 0, nil, false
	}
	slot := readBitsAt(pay, p+1, bipedSlotBits)
	if !slots[slot] {
		return 0, 0, nil, false
	}
	if needTag1 && readBitsAt(pay, p+14, 2) != 1 {
		return 0, 0, nil, false
	}
	if readBitsAt(pay, p+16, 2) != 0 { // gate + maskSel
		return 0, 0, nil, false
	}
	mc := int(readBitsAt(pay, p+18, 3))
	if mc < bipedMinMaskCnt || mc > bipedMaxMaskCnt {
		return 0, 0, nil, false
	}
	i0 := p + bipedHeaderBits + bipedIndexBits*mc
	if i0+bipedI0Bits > total {
		return 0, 0, nil, false
	}
	idx, ok := ascendingFromZero(pay, p+bipedHeaderBits, mc)
	if !ok {
		return 0, 0, nil, false
	}
	if readBitsAt(pay, i0, bipedI0GateBits) != 0 { // i0 absolu : gate nul
		return 0, 0, nil, false
	}
	return i0, slot, idx, true
}

// ascendingFromZero valide la liste d'indices de composants (premier = 0, strictement
// croissants — invariant fort du masque) et la renvoie.
func ascendingFromZero(pay []byte, at, count int) ([]int, bool) {
	out := make([]int, 0, count)
	prev := -1
	for k := 0; k < count; k++ {
		idx := int(readBitsAt(pay, at+bipedIndexBits*k, bipedIndexBits))
		if (k == 0 && idx != 0) || idx <= prev {
			return nil, false
		}
		prev = idx
		out = append(out, idx)
	}
	return out, true
}

// saturatedQuantum signale un axe dans son bucket extrême (valeur écrêtée).
func saturatedQuantum(q [3]uint32) bool {
	for i, w := range bipedAxisWidths {
		if q[i] == 0 || q[i] == uint32(1)<<w-1 {
			return true
		}
	}
	return false
}

// DequantBipedAxis déquantifie l'axe ax (0=X, 1=Y, 2=Z) d'une position absolue biped.
// Calcul en float64 pour ne pas décaler l'indice de quantum sur les arrondis float32.
func DequantBipedAxis(q uint32, ax int) float32 {
	rng := QuantRangeCEBiped[ax]
	step := (float64(rng.Max) - float64(rng.Min)) / float64(uint64(1)<<bipedAxisWidths[ax])
	return float32(float64(rng.Min) + step*(float64(q)+quantCenter))
}

// readBitsAt lit n bits MSB-first à la position bit pos (n <= 32).
func readBitsAt(b []byte, pos, n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		p := pos + i
		v = v<<1 | uint32(b[p>>3]>>(7-uint(p&7))&1)
	}
	return v
}
