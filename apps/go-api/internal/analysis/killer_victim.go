// Package analysis — algorithme de résolution killer→victim (algorithme 3).
//
// Stratégie :
//   - Séparer les événements en kills et deaths.
//   - Trier les deaths par timestamp.
//   - Apparier kills et deaths distants d au plus toleranceMS, de l écart le
//     plus petit au plus grand (instant exact d abord), jamais un joueur avec
//     lui-même.
//   - Éviter de réutiliser le même kill ou le même death (bijection 1→1).
//
// Ce package est stateless (aucune DB, aucune dépendance platform/).
package analysis

import (
	"sort"

	"levelup/go-api/internal/domain/highlightevent"
)

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

// timedEvent est un événement daté, pour le tri et la recherche binaire.
type timedEvent struct {
	t     int64
	event RawEvent
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

// KillDeathPairToleranceMS : LA fenêtre d appariement kill <-> mort, en millisecondes. Les deux
// événements sont deux lectures du même kill-feed de l API, horodatées à quelques millisecondes
// l une de l autre. Toute paire kill <-> mort du dépôt (couples `killer_victim_pairs`, base crédit,
// timeline canonique, adossement des morts servies à un événement death) emploie cette valeur :
// deux fenêtres différentes rendraient deux jeux de couples incomparables (garde-rail
// `archlint/kill_death_tolerance_test.go`).
const KillDeathPairToleranceMS int64 = 5

// ComputeKillerVictimPairs construit les paires killer→victim à partir
// des highlight events bruts.
//
// AFFECTATION PAR ÉCART CROISSANT : toutes les paires (kill, mort) distantes d au plus
// `toleranceMS` sont candidates, et elles se décident de la plus proche à la plus lointaine —
// l instant exact d abord. Un kill et une mort ne servent qu une fois, et un joueur ne se tue pas
// lui-même (une paire dont le tueur est la victime n est jamais candidate : une mort de soi n a
// pas de kill dans le kill-feed). L ordre de parcours des kills ne décide donc rien : un kill sur
// un bot (aucune mort dans le kill-feed, humain seul) ne prend plus la mort d un humain tombée
// 2 ms plus loin et revendiquée par un autre kill à l instant exact. À écart égal, l ordre total
// des événements ([timedEventLess]) départage, kill puis mort.
//
// toleranceMS : fenêtre de jointure en millisecondes (défaut recommandé : 5).
// Retourne les paires dans l'ordre chronologique des kills.
func ComputeKillerVictimPairs(events []RawEvent, toleranceMS int64) []KVPair {
	if toleranceMS < 0 {
		toleranceMS = 0
	}

	var kills, deaths []timedEvent
	for _, e := range events {
		switch e.EventType {
		case highlightevent.EventTypeKill:
			kills = append(kills, timedEvent{e.TimeMS, e})
		case highlightevent.EventTypeDeath:
			deaths = append(deaths, timedEvent{e.TimeMS, e})
		}
	}

	if len(kills) == 0 || len(deaths) == 0 {
		return nil
	}

	// Trier kills et deaths par temps pour la recherche binaire
	sortTimedEvents(kills)
	sortTimedEvents(deaths)

	victimeDuKill := affecterParEcartCroissant(kills, deaths, toleranceMS)
	out := make([]KVPair, 0, len(kills))
	for ki, k := range kills {
		di, ok := victimeDuKill[ki]
		if !ok {
			continue
		}
		out = append(out, nouvellePaire(k.event, deaths[di].event))
	}
	return out
}

// candidatPaire : un kill et une mort distants d au plus la tolérance, par leurs rangs dans les
// listes triées.
type candidatPaire struct {
	kill, mort int
	ecart      int64
}

// affecterParEcartCroissant rend, pour chaque kill apparié, le rang de sa mort (cf.
// [ComputeKillerVictimPairs]). Les rangs suivent l ordre total des deux listes triées : trier les
// candidats par (écart, rang du kill, rang de la mort) rend l affectation indépendante de l ordre
// des événements en entrée.
func affecterParEcartCroissant(kills, deaths []timedEvent, toleranceMS int64) map[int]int {
	deathTimes := make([]int64, len(deaths))
	for i, d := range deaths {
		deathTimes[i] = d.t
	}
	var cands []candidatPaire
	for ki, k := range kills {
		lo := lowerBound(deathTimes, k.t-toleranceMS)
		hi := upperBound(deathTimes, k.t+toleranceMS)
		for di := lo; di < hi; di++ {
			if deaths[di].event.XUID == k.event.XUID {
				continue // un joueur ne se tue pas lui-même
			}
			cands = append(cands, candidatPaire{kill: ki, mort: di, ecart: abs64(deathTimes[di] - k.t)})
		}
	}
	sort.Slice(cands, func(i, j int) bool {
		a, b := cands[i], cands[j]
		if a.ecart != b.ecart {
			return a.ecart < b.ecart
		}
		if a.kill != b.kill {
			return a.kill < b.kill
		}
		return a.mort < b.mort
	})
	victimeDuKill := make(map[int]int, len(kills))
	mortPrise := make([]bool, len(deaths))
	for _, c := range cands {
		if _, pris := victimeDuKill[c.kill]; pris || mortPrise[c.mort] {
			continue
		}
		victimeDuKill[c.kill] = c.mort
		mortPrise[c.mort] = true
	}
	return victimeDuKill
}

// nouvellePaire : la paire d un kill et de sa mort ; un gamertag vide retombe sur le xuid.
func nouvellePaire(kill, victim RawEvent) KVPair {
	killerGT := kill.Gamertag
	if killerGT == "" {
		killerGT = kill.XUID
	}
	victimGT := victim.Gamertag
	if victimGT == "" {
		victimGT = victim.XUID
	}
	return KVPair{
		KillerXUID: kill.XUID,
		KillerGT:   killerGT,
		VictimXUID: victim.XUID,
		VictimGT:   victimGT,
		TimeMS:     kill.TimeMS,
	}
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

// timedEventLess est l ordre TOTAL des événements : temps, puis XUID, gamertag et type. Deux
// événements de même instant se départagent donc par leur contenu et non par l ordre d arrivée
// ou l instabilité du tri : la paire servie ne dépend plus de l ordre des événements en entrée.
func timedEventLess(a, b timedEvent) bool {
	if a.t != b.t {
		return a.t < b.t
	}
	if a.event.XUID != b.event.XUID {
		return a.event.XUID < b.event.XUID
	}
	if a.event.Gamertag != b.event.Gamertag {
		return a.event.Gamertag < b.event.Gamertag
	}
	return a.event.EventType < b.event.EventType
}

func sortTimedEvents(evs []timedEvent) {
	sort.SliceStable(evs, func(i, j int) bool { return timedEventLess(evs[i], evs[j]) })
}
