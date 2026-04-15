// Package analysis — algorithme de résolution killer→victim (algorithme 3).
//
// Port Go de src/analysis/killer_victim.py (compute_killer_victim_pairs).
//
// Stratégie :
//  - Séparer les événements en kills et deaths.
//  - Trier les deaths par timestamp.
//  - Pour chaque kill au temps T, chercher le death le plus proche
//    dans l'intervalle [T-toleranceMS, T+toleranceMS].
//  - Éviter de réutiliser le même death (bijection 1→1).
//
// Ce package est stateless (aucune DB, aucune dépendance platform/).
package analysis

import "sort"

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// RawEvent représente un highlight event brut.
type RawEvent struct {
	EventType string // "kill" ou "death"
	XUID      string // joueur qui a effectué l'action
	Gamertag  string // gamertag (peut être vide → fallback sur XUID)
	TimeMS    int64  // timestamp en millisecondes depuis le début du match
}

// KVPair est une paire killer→victim résolue.
type KVPair struct {
	KillerXUID string
	KillerGT   string
	VictimXUID string
	VictimGT   string
	TimeMS     int64
}

// ---------------------------------------------------------------------------
// Algorithme principal
// ---------------------------------------------------------------------------

// ComputeKillerVictimPairs construit les paires killer→victim à partir
// des highlight events bruts.
//
// toleranceMS : fenêtre de jointure en millisecondes (défaut recommandé : 5).
// Retourne les paires dans l'ordre chronologique des kills.
func ComputeKillerVictimPairs(events []RawEvent, toleranceMS int64) []KVPair {
	if toleranceMS < 0 {
		toleranceMS = 0
	}

	type timedEvent struct {
		t     int64
		event RawEvent
	}

	var kills, deaths []timedEvent
	for _, e := range events {
		switch e.EventType {
		case "kill":
			kills = append(kills, timedEvent{e.TimeMS, e})
		case "death":
			deaths = append(deaths, timedEvent{e.TimeMS, e})
		}
	}

	if len(kills) == 0 || len(deaths) == 0 {
		return nil
	}

	// Trier kills et deaths par temps pour la recherche binaire
	sort.Slice(kills, func(i, j int) bool { return kills[i].t < kills[j].t })
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].t < deaths[j].t })

	deathTimes := make([]int64, len(deaths))
	for i, d := range deaths {
		deathTimes[i] = d.t
	}

	usedDeathIdx := make(map[int]bool, len(deaths))
	out := make([]KVPair, 0, len(kills))

	for _, k := range kills {
		lo := lowerBound(deathTimes, k.t-toleranceMS)
		hi := upperBound(deathTimes, k.t+toleranceMS)
		if lo >= hi {
			continue
		}

		bestIdx := -1
		var bestDelta int64 = -1
		for idx := lo; idx < hi; idx++ {
			if usedDeathIdx[idx] {
				continue
			}
			delta := abs64(deathTimes[idx] - k.t)
			if bestIdx == -1 || delta < bestDelta {
				bestDelta = delta
				bestIdx = idx
			}
		}

		if bestIdx == -1 {
			continue
		}

		usedDeathIdx[bestIdx] = true
		victim := deaths[bestIdx].event

		killerGT := k.event.Gamertag
		if killerGT == "" {
			killerGT = k.event.XUID
		}
		victimGT := victim.Gamertag
		if victimGT == "" {
			victimGT = victim.XUID
		}

		out = append(out, KVPair{
			KillerXUID: k.event.XUID,
			KillerGT:   killerGT,
			VictimXUID: victim.XUID,
			VictimGT:   victimGT,
			TimeMS:     k.t,
		})
	}

	return out
}

// ComputeAntagonistCounts agrège les comptes kills/deaths par paire de joueurs.
// killerXUID→victimXUID. Retourne (killed_by_me, killed_me) maps.
func ComputeAntagonistCounts(pairs []KVPair, myXUID string) (killedByMe, killedMe map[string]int) {
	killedByMe = make(map[string]int)
	killedMe = make(map[string]int)
	for _, p := range pairs {
		if p.KillerXUID == myXUID {
			killedByMe[p.VictimXUID]++
		} else if p.VictimXUID == myXUID {
			killedMe[p.KillerXUID]++
		}
	}
	return killedByMe, killedMe
}

// ---------------------------------------------------------------------------
// Helpers bisect (equivalent Python bisect_left / bisect_right)
// ---------------------------------------------------------------------------

// lowerBound retourne l'indice du premier élément >= target.
func lowerBound(arr []int64, target int64) int {
	return sort.Search(len(arr), func(i int) bool { return arr[i] >= target })
}

// upperBound retourne l'indice du premier élément > target.
func upperBound(arr []int64, target int64) int {
	return sort.Search(len(arr), func(i int) bool { return arr[i] > target })
}

func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
