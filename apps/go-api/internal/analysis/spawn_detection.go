// Package analysis — spawn_detection.go : détection du début de match depuis les films.
//
// Portage de src/analysis/spawn_detection.py (Python, ~562 lignes).
//
// Algorithme 7 : estime le timestamp (ms) de début de match dans un film binaire
// via détection du premier mouvement des joueurs (changement de signature).
//
// Performance cible (corpus 198 matchs) :
//   - 55% à ±5s
//   - 60% à ±10s
//   - 91% à ±30s
package analysis

import (
	"math"
	"sort"
)

// ─────────────────────────────────────────────────────────────────────────────
// Constantes filmshell
// ─────────────────────────────────────────────────────────────────────────────

const (
	// Marqueur de frame replication (3 octets en-tête).
	frameMarkerB0 = 0xA0
	frameMarkerB1 = 0x7B
	frameMarkerB2 = 0x42

	// Byte5/Byte9 pour les frames humains (posture debout active).
	byteHumanB5 = 0x40
	byteHumanB9 = 0x56

	// Paramètres algorithme.
	spawnClusterWindowMS = 2000.0
	afkThresholdMS       = 10_000.0
	apiCapBufferMS       = 5_000.0

	// Frame replication header minimum : 3 (marker) + 12 minimum.
	minFrameLen = 15
)

// validBaseTypes est l'ensemble des base_type acceptés dans les frames.
var validBaseTypes = map[byte]bool{
	0x08: true, 0x09: true, 0x0A: true, 0x0B: true,
	0x28: true, 0x29: true,
}

// ─────────────────────────────────────────────────────────────────────────────
// Structures de données
// ─────────────────────────────────────────────────────────────────────────────

// ChunkRef décrit un chunk de film (position + timestamps).
// Same layout as in weapon_parser.go (ChunkData) but minimal for spawn.
type SpawnChunk struct {
	Index    int
	Data     []byte
	StartMS  float64
	EndMS    float64
}

// FirstMovement décrit le premier mouvement détecté pour un joueur.
type FirstMovement struct {
	TimestampMS float64
	ChunkIndex  int
	PlayerIdx   int
	B5          byte
	B9          byte
	YRaw        int // -1 si non décodé
	XRaw        int
	DelayMS     float64 // rempli par pickSpawnReferences
}

// ─────────────────────────────────────────────────────────────────────────────
// API publique
// ─────────────────────────────────────────────────────────────────────────────

// EstimateFilmMatchStartMS estime le timestamp (ms) du début du match dans le film.
//
// Paramètres :
//   - chunks : map[chunkIndex]SpawnChunk (données binaires + timestamps)
//   - minPlayers : nombre minimum de joueurs actifs pour le pic (défaut 3)
//   - apiFirstEventMS : si non nil, plafond = api_first_event - buffer 5s
//
// Retourne -1 si impossible à estimer.
// Portage de estimate_film_match_start_ms() Python.
func EstimateFilmMatchStartMS(
	chunks []SpawnChunk,
	minPlayers int,
	apiFirstEventMS float64, // 0 = non fourni
) float64 {
	if minPlayers == 0 {
		minPlayers = 3
	}

	movements := ScanFirstMovements(chunks)
	if len(movements) < minPlayers {
		return -1
	}

	peakTS := FindPeakActivityWindow(chunks, movements, spawnClusterWindowMS)
	if peakTS < 0 {
		return -1
	}

	// Contrainte dure API : si notre estimate dépasse l'événement API, reculer
	if apiFirstEventMS > 0 && peakTS > apiFirstEventMS {
		peakTS = apiFirstEventMS - apiCapBufferMS
	}
	if peakTS < 0 {
		peakTS = 0
	}
	return peakTS
}

// ─────────────────────────────────────────────────────────────────────────────
// Scan premier mouvement
// ─────────────────────────────────────────────────────────────────────────────

// ScanFirstMovements cherche le premier changement de signature dans les frames
// de chaque joueur (player_index 0-7).
// Portage de scan_first_movements() Python.
func ScanFirstMovements(chunks []SpawnChunk) []FirstMovement {
	// Pour chaque joueur : signature précédente et premier mouvement
	type state struct {
		prevSig  [7]byte
		movement *FirstMovement
	}
	players := make(map[int]*state, 8)

	for _, chunk := range chunks {
		data := chunk.Data
		pos := 0
		for pos+minFrameLen < len(data) {
			// Chercher FRAME_MARKER
			if data[pos] != frameMarkerB0 || data[pos+1] != frameMarkerB1 || data[pos+2] != frameMarkerB2 {
				pos++
				continue
			}
			pi, sig, y, x, ok := decodePositionFrame(data, pos)
			if !ok {
				pos++
				continue
			}

			st, exists := players[pi]
			if !exists {
				st = &state{}
				copy(st.prevSig[:], sig[:])
				players[pi] = st
				pos++
				continue
			}

			// Détecter le changement de signature
			if st.movement == nil && sig != st.prevSig {
				ts := estimateFrameTimestamp(chunk, pos)
				mv := &FirstMovement{
					TimestampMS: ts,
					ChunkIndex:  chunk.Index,
					PlayerIdx:   pi,
					B5:          data[pos+5],
					B9:          data[pos+9],
					YRaw:        y,
					XRaw:        x,
				}
				st.movement = mv
			}
			copy(st.prevSig[:], sig[:])
			pos++
		}
	}

	// Collecter les mouvements trouvés
	var result []FirstMovement
	for _, st := range players {
		if st.movement != nil {
			result = append(result, *st.movement)
		}
	}
	return result
}

// decodePositionFrame tente de décoder une frame de réplication à la position pos.
// Retourne (player_index, signature[7], y_raw, x_raw, ok).
func decodePositionFrame(data []byte, pos int) (pi int, sig [7]byte, y, x int, ok bool) {
	if pos+16 >= len(data) {
		return
	}
	// Byte 3 : base_type
	baseType := data[pos+3]
	if !validBaseTypes[baseType] {
		return
	}
	// Byte 5 : player_index (format humain : 0x40)
	b5 := data[pos+5]
	if b5 != byteHumanB5 {
		return
	}
	// Byte 9 doit être 0x56 pour les frames de position humaine
	if pos+9 >= len(data) || data[pos+9] != byteHumanB9 {
		return
	}
	// player_index encodé dans les 4 bits hauts du byte 4
	if pos+4 >= len(data) {
		return
	}
	pi = int(data[pos+4] >> 4)
	if pi < 0 || pi > 7 {
		return
	}
	// Signature = bytes 9..15 (7 octets de coordonnées/état)
	if pos+16 >= len(data) {
		return
	}
	copy(sig[:], data[pos+9:pos+16])
	// Coordonnées brutes (optionnel, pour le debug)
	y = int(sig[0])<<8 | int(sig[1])
	x = int(sig[2])<<8 | int(sig[3])
	ok = true
	return
}

// estimateFrameTimestamp estime le timestamp en ms d'une frame dans un chunk.
func estimateFrameTimestamp(chunk SpawnChunk, framePos int) float64 {
	if chunk.EndMS <= chunk.StartMS || len(chunk.Data) == 0 {
		return chunk.StartMS
	}
	ratio := float64(framePos) / float64(len(chunk.Data))
	return chunk.StartMS + ratio*(chunk.EndMS-chunk.StartMS)
}

// ─────────────────────────────────────────────────────────────────────────────
// Fenêtre glissante — pic d'activité
// ─────────────────────────────────────────────────────────────────────────────

// FindPeakActivityWindow cherche la fenêtre [t, t+windowMS] avec le plus de joueurs actifs.
// Portage de find_peak_activity_window() Python.
func FindPeakActivityWindow(chunks []SpawnChunk, movements []FirstMovement, windowMS float64) float64 {
	if len(movements) == 0 {
		return -1
	}

	// Collecter tous les timestamps de mouvement
	sort.Slice(movements, func(i, j int) bool {
		return movements[i].TimestampMS < movements[j].TimestampMS
	})

	// Fenêtre glissante : pour chaque timestamp de début, compter les joueurs actifs
	bestCount := 0
	bestTS := movements[0].TimestampMS

	for i, m := range movements {
		windowEnd := m.TimestampMS + windowMS
		active := 0
		seenPI := make(map[int]bool)
		for j := i; j < len(movements); j++ {
			if movements[j].TimestampMS > windowEnd {
				break
			}
			if !seenPI[movements[j].PlayerIdx] {
				seenPI[movements[j].PlayerIdx] = true
				active++
			}
		}
		if active > bestCount {
			bestCount = active
			bestTS = m.TimestampMS
		}
	}
	return bestTS
}

// ─────────────────────────────────────────────────────────────────────────────
// Filtrage des AFK / références de spawn
// ─────────────────────────────────────────────────────────────────────────────

// PickSpawnReferences filtre les premiers mouvements, retire les AFK et
// calcule la distance au cluster pour chaque joueur.
// Portage de pick_spawn_references() + find_densest_spawn_cluster() Python.
func PickSpawnReferences(movements []FirstMovement, n int) []FirstMovement {
	if len(movements) == 0 {
		return nil
	}
	sort.Slice(movements, func(i, j int) bool {
		return movements[i].TimestampMS < movements[j].TimestampMS
	})

	// Médiane
	median := movements[len(movements)/2].TimestampMS

	// Filtrer les AFK (mouvement > 10s après médiane)
	var filtered []FirstMovement
	for _, m := range movements {
		delay := m.TimestampMS - median
		if delay < afkThresholdMS {
			mv := m
			mv.DelayMS = math.Abs(delay)
			filtered = append(filtered, mv)
		}
	}

	if len(filtered) == 0 {
		filtered = movements
	}

	// Limiter à n
	if n > 0 && len(filtered) > n {
		filtered = filtered[:n]
	}
	return filtered
}
