package analysis

// weapon_parser.go — Orchestration haut-niveau du weapon parser.
//
// Port de src/analysis/weapon_parser.py.
// Fonctions de scan, timeline, confidence et corrélation.

import "sort"

// KillWindowMS est la fenêtre de recherche fire events avant un kill.
const KillWindowMS = 5000

// ChunkTiming représente l'intervalle temporel d'un chunk.
type ChunkTiming struct {
	StartMS int
	EndMS   int
}

// ChunkData encapsule les données binaires d'un chunk.
type ChunkData struct {
	Data       []byte
	StartMS    int
	DurationMS int
}

// ══════════════════════════════════════════════════════════════════════════════
//  Scan haut-niveau
// ══════════════════════════════════════════════════════════════════════════════

// ScanFireEventsAll scanne les fire events de TOUS les joueurs d'un chunk.
func ScanFireEventsAll(data []byte, startMS, durationMS int) []FireEvent {
	estimateTS := TimestampEstimator(data, startMS, durationMS)
	return ScanFireEventsB5(data, estimateTS)
}

// ScanFireEvents scanne les fire events d'un seul player_index.
func ScanFireEvents(data []byte, playerIndex, startMS, durationMS int) []FireEvent {
	all := ScanFireEventsAll(data, startMS, durationMS)
	var filtered []FireEvent
	for _, ev := range all {
		if ev.PlayerIndex == playerIndex {
			filtered = append(filtered, ev)
		}
	}
	return filtered
}

// ══════════════════════════════════════════════════════════════════════════════
//  Timeline & chunk lookup
// ══════════════════════════════════════════════════════════════════════════════

// WeaponTimelines contient les données de timeline raw + NS.
type WeaponTimelines struct {
	// Timeline raw : chunk_idx → player_index → weapon_bytes
	Timeline map[int]map[int][8]byte
	// TimelineNS : nibble-shifted TYPE IDs
	TimelineNS map[int]map[int][8]byte
	// SwapPIs : chunk_idx → set de player_indices ayant changé d'arme
	SwapPIs map[int]map[int]bool
	// Timing : intervalles temporels ordonnés par chunk
	Timing []ChunkTiming
}

// BuildWeaponTimelines construit timeline raw + NS en une seule passe.
func BuildWeaponTimelines(chunks map[int]ChunkData) (WeaponTimelines, []int) {
	// Trier les idx
	indices := make([]int, 0, len(chunks))
	for idx := range chunks {
		indices = append(indices, idx)
	}
	sort.Ints(indices)

	wt := WeaponTimelines{
		Timeline:   make(map[int]map[int][8]byte, len(chunks)),
		TimelineNS: make(map[int]map[int][8]byte, len(chunks)),
		SwapPIs:    make(map[int]map[int]bool, len(chunks)),
		Timing:     make([]ChunkTiming, 0, len(chunks)),
	}

	for _, idx := range indices {
		chunk := chunks[idx]

		// Raw timeline (Formula A, Section 1) + swap detection
		rawEvents := ScanFormulaA(chunk.Data)
		chunkRaw := make(map[int][8]byte)
		piSeenWIDs := make(map[int]map[[8]byte]bool)
		for _, ev := range rawEvents {
			chunkRaw[ev.PlayerIndex] = ev.WeaponBytes
			if piSeenWIDs[ev.PlayerIndex] == nil {
				piSeenWIDs[ev.PlayerIndex] = make(map[[8]byte]bool)
			}
			piSeenWIDs[ev.PlayerIndex][ev.WeaponBytes] = true
		}
		wt.Timeline[idx] = chunkRaw
		swapSet := make(map[int]bool)
		for pi, wids := range piSeenWIDs {
			if len(wids) > 1 {
				swapSet[pi] = true
			}
		}
		wt.SwapPIs[idx] = swapSet
		wt.Timing = append(wt.Timing, ChunkTiming{
			StartMS: chunk.StartMS,
			EndMS:   chunk.StartMS + chunk.DurationMS,
		})

		// NS timeline (TYPE IDs) — même chunk, pas de second sort
		nsEvents := ScanFormulaANS(chunk.Data)
		chunkNS := make(map[int][8]byte)
		for _, ev := range nsEvents {
			chunkNS[ev.PlayerIndex] = ev.WeaponBytes
		}
		wt.TimelineNS[idx] = chunkNS
	}

	return wt, indices
}

// FindChunkAtTime retourne l'index de chunk couvrant tMS.
func FindChunkAtTime(chunksSorted []int, timing []ChunkTiming, tMS int) int {
	for i, idx := range chunksSorted {
		if i < len(timing) && timing[i].StartMS <= tMS && tMS < timing[i].EndMS {
			return idx
		}
	}
	if len(chunksSorted) > 0 {
		return chunksSorted[len(chunksSorted)-1]
	}
	return 0
}

// ══════════════════════════════════════════════════════════════════════════════
//  Confidence
// ══════════════════════════════════════════════════════════════════════════════

// ComputeConfidence calcule la confidence d'une attribution.
func ComputeConfidence(weaponID uint64, deltaMS int) string {
	t := GetTiming(weaponID)
	if deltaMS < t.SwapMS {
		return confidenceHigh
	}
	if deltaMS <= t.TravelMax {
		return confidenceMedium
	}
	return confidenceLow
}

// ══════════════════════════════════════════════════════════════════════════════
//  Agrégation
// ══════════════════════════════════════════════════════════════════════════════

// CountKillsByWeapon agrège les kills corrélés en map[weapon_id]count.
func CountKillsByWeapon(attributions []KillAttribution) map[uint64]int {
	counts := make(map[uint64]int)
	for _, a := range attributions {
		eid := a.EffectiveWeaponID()
		if eid != nil {
			counts[*eid]++
		}
	}
	return counts
}
