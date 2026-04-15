package analysis

// weapon_correlation.go — Corrélation globale fire events → kills (claim-and-remove).
//
// Port de src/analysis/_global_correlation.py.
// Chaque kill consomme le fire event non-claimé le plus proche dans [t-5s, t]
// pour le player_index du tueur.

import (
	"log/slog"
	"sort"
)

// Kill représente un kill à attribuer.
type Kill struct {
	MatchID   string
	XUID      string
	TimeMS    int
	IsMelee   bool
	IsGrenade bool
}

// CorrelateKillsGlobal effectue la corrélation claim-and-remove globale.
//
// Algorithme :
//  1. Sort kills par time_ms
//  2. Pour chaque kill :
//     - melee/grenade → sentinel
//     - sinon : chercher le fire event non-claimé le plus proche dans [t-5s, t]
//       dont player_index == killer_pi
//     - si trouvé → claim (retirer), path="fire_event"
//     - sinon → fallback Formula A
func CorrelateKillsGlobal(
	kills []Kill,
	fireEvents []FireEvent,
	xuidToPI map[string]int,
	timeline map[int]map[int][8]byte,
	swapPIs map[int]map[int]bool,
	timing []ChunkTiming,
	chunksSorted []int,
	matchID string,
	timelineNS map[int]map[int][8]byte,
) []KillAttribution {
	// Copier et trier les events pour claim-and-remove
	available := make([]FireEvent, len(fireEvents))
	copy(available, fireEvents)
	sort.Slice(available, func(i, j int) bool {
		return available[i].TimestampMS < available[j].TimestampMS
	})

	// Trier les kills par temps
	sortedKills := make([]Kill, len(kills))
	copy(sortedKills, kills)
	sort.Slice(sortedKills, func(i, j int) bool {
		return sortedKills[i].TimeMS < sortedKills[j].TimeMS
	})

	results := make([]KillAttribution, 0, len(kills))

	for _, kill := range sortedKills {
		var attr KillAttribution

		if kill.IsMelee {
			attr = makeSentinel(kill, matchID, MeleeWeaponID)
		} else if kill.IsGrenade {
			attr = makeSentinel(kill, matchID, GrenadeWeaponID)
		} else {
			tMS := kill.TimeMS
			killerPI, hasPI := xuidToPI[kill.XUID]

			bestIdx := -1
			var bestEv FireEvent
			for i, ev := range available {
				if ev.TimestampMS < float64(tMS-KillWindowMS) {
					continue
				}
				if ev.TimestampMS > float64(tMS) {
					break // sorted, no more candidates
				}
				if hasPI && ev.PlayerIndex != killerPI {
					continue
				}
				if bestIdx == -1 || ev.TimestampMS > bestEv.TimestampMS {
					bestIdx = i
					bestEv = ev
				}
			}

			if bestIdx >= 0 {
				// Claim : retirer de available
				available = append(available[:bestIdx], available[bestIdx+1:]...)
				attr = attributionFromEvent(kill, bestEv, matchID, timelineNS, timing, chunksSorted)
			} else {
				// Fallback Formula A
				pi := -1
				if hasPI {
					pi = killerPI
				}
				attr = fallbackFormulaA(kill, pi, timeline, swapPIs, timing, chunksSorted, matchID, timelineNS)
			}
		}

		results = append(results, attr)
	}

	return results
}

func makeSentinel(kill Kill, matchID string, weaponID uint64) KillAttribution {
	wid := weaponID
	return KillAttribution{
		MatchID:         matchID,
		XUID:            kill.XUID,
		TimeMS:          kill.TimeMS,
		WeaponID:        &wid,
		Confidence:      "none",
		AttributionPath: "none",
	}
}

func attributionFromEvent(
	kill Kill,
	ev FireEvent,
	matchID string,
	timelineNS map[int]map[int][8]byte,
	timing []ChunkTiming,
	chunksSorted []int,
) KillAttribution {
	widInt, ok := WeaponBytesToInt[ev.WeaponBytes]
	if !ok && timelineNS != nil && len(timing) > 0 {
		chunkIdx := FindChunkAtTime(chunksSorted, timing, int(ev.TimestampMS))
		if nsState, exists := timelineNS[chunkIdx]; exists {
			if nsWB, found := nsState[ev.PlayerIndex]; found {
				if nsID, nsOK := WeaponBytesToInt[nsWB]; nsOK {
					widInt = nsID
					ok = true
					slog.Debug("NS lookup hit", "match", matchID, "pi", ev.PlayerIndex, "chunk", chunkIdx)
				}
			}
		}
	}
	if !ok {
		// Utiliser le raw uint64
		widInt = hexToUint64(ev.WeaponBytes)
	}

	delta := kill.TimeMS - int(ev.TimestampMS)
	conf := ComputeConfidence(widInt, delta)
	t := GetTiming(widInt)
	pi := ev.PlayerIndex
	ci := ev.ChunkIdx

	return KillAttribution{
		MatchID:         matchID,
		XUID:            kill.XUID,
		TimeMS:          kill.TimeMS,
		WeaponID:        &widInt,
		DeltaMS:         &delta,
		Confidence:      conf,
		AttributionPath: "fire_event",
		SwapDetected:    delta >= t.SwapMS,
		DelayedDamage:   delta > t.TravelMax,
		PlayerIndex:     &pi,
		SourceChunkIdx:  &ci,
	}
}

func fallbackFormulaA(
	kill Kill,
	pi int,
	timeline map[int]map[int][8]byte,
	swapPIs map[int]map[int]bool,
	timing []ChunkTiming,
	chunksSorted []int,
	matchID string,
	timelineNS map[int]map[int][8]byte,
) KillAttribution {
	tMS := kill.TimeMS
	var widInt *uint64
	swap := false

	if pi >= 0 && len(chunksSorted) > 0 {
		chunkIdx := FindChunkAtTime(chunksSorted, timing, tMS)
		if swapSet, ok := swapPIs[chunkIdx]; ok {
			swap = swapSet[pi]
		}

		// 1. NS timeline (TYPE IDs canoniques)
		if timelineNS != nil {
			if nsState, exists := timelineNS[chunkIdx]; exists {
				if nsWB, found := nsState[pi]; found {
					if nsID, nsOK := WeaponBytesToInt[nsWB]; nsOK {
						widInt = &nsID
						slog.Debug("fallback_formula_a NS resolved", "match", matchID, "pi", pi, "chunk", chunkIdx)
					}
				}
			}
		}

		// 2. Raw FA timeline
		if widInt == nil {
			if chunkState, exists := timeline[chunkIdx]; exists {
				if widBytes, found := chunkState[pi]; found {
					if id, isKnown := WeaponBytesToInt[widBytes]; isKnown {
						widInt = &id
					} else {
						raw := hexToUint64(widBytes)
						widInt = &raw
					}
					slog.Debug("fallback_formula_a raw FA resolved", "match", matchID, "pi", pi, "chunk", chunkIdx)
				}
			}
		}
	}

	var conf string
	switch {
	case widInt != nil && !swap:
		conf = "medium"
	case widInt != nil:
		conf = "low"
	default:
		conf = "none"
	}

	piPtr := &pi
	if pi < 0 {
		piPtr = nil
	}

	return KillAttribution{
		MatchID:         matchID,
		XUID:            kill.XUID,
		TimeMS:          tMS,
		WeaponID:        widInt,
		Confidence:      conf,
		AttributionPath: "formula_a",
		SwapDetected:    swap,
		PlayerIndex:     piPtr,
	}
}
