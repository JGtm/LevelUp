// Package observability — player_api_collector.go : agrégat des appels API Halo
// PAR JOUEUR depuis le boot, pour les appels qui portent un identifiant
// (match_history, career_rank, player_csrs, playlist_csr). Complète les
// agrégats globaux par type d'appel (RecordDurationMS) : permet de repérer un
// joueur dont les tokens/réseau échouent systématiquement.
//
// Les appels match-level (match_stats, film, film_chunk, match_skill) ne sont
// pas attribuables à un joueur unique → restent globaux uniquement.
//
// Ring borné (defaultPlayerAPICap clés (call,player), éviction de la moins
// active) : aucune fuite mémoire même si un xuid inattendu apparaît.
package observability

import (
	"sort"
	"sync"
)

// defaultPlayerAPICap borne les couples (call, player) suivis (4 appels ×
// joueurs trackés — large marge).
const defaultPlayerAPICap = 64

// PlayerAPIStat agrège un couple (call, player). AvgMs est dérivé à la lecture.
type PlayerAPIStat struct {
	Player string
	Call   string
	Count  int64
	SumMs  int64
	MaxMs  int64
	AvgMs  int64
	Errors int64
}

type playerAPIEntry struct {
	count, sum, max, errors int64
}

type playerAPICollector struct {
	mu      sync.Mutex
	entries map[string]*playerAPIEntry // clé = call + "\x00" + player
	cap     int
}

func newPlayerAPICollector(capacity int) *playerAPICollector {
	return &playerAPICollector{entries: make(map[string]*playerAPIEntry), cap: capacity}
}

func (c *playerAPICollector) record(call, player string, ms int64, isErr bool) {
	if call == "" || player == "" {
		return
	}
	key := call + "\x00" + player
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.entries[key]
	if !ok {
		if len(c.entries) >= c.cap {
			c.evictLeastActive()
		}
		e = &playerAPIEntry{}
		c.entries[key] = e
	}
	e.count++
	e.sum += ms
	if ms > e.max {
		e.max = ms
	}
	if isErr {
		e.errors++
	}
}

// evictLeastActive retire l'entrée au plus faible count (appelé sous lock).
func (c *playerAPICollector) evictLeastActive() {
	var minKey string
	var minCount int64
	for k, e := range c.entries {
		if minKey == "" || e.count < minCount {
			minKey, minCount = k, e.count
		}
	}
	if minKey != "" {
		delete(c.entries, minKey)
	}
}

func (c *playerAPICollector) snapshot() []PlayerAPIStat {
	c.mu.Lock()
	out := make([]PlayerAPIStat, 0, len(c.entries))
	for key, e := range c.entries {
		call, player := splitPlayerKey(key)
		avg := int64(0)
		if e.count > 0 {
			avg = e.sum / e.count
		}
		out = append(out, PlayerAPIStat{
			Player: player, Call: call, Count: e.count,
			SumMs: e.sum, MaxMs: e.max, AvgMs: avg, Errors: e.errors,
		})
	}
	c.mu.Unlock()

	// Tri : erreurs desc (joueurs problématiques d'abord), puis count desc.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Errors != out[j].Errors {
			return out[i].Errors > out[j].Errors
		}
		return out[i].Count > out[j].Count
	})
	return out
}

func (c *playerAPICollector) reset() {
	c.mu.Lock()
	c.entries = make(map[string]*playerAPIEntry)
	c.mu.Unlock()
}

func splitPlayerKey(key string) (call, player string) {
	for i := 0; i < len(key); i++ {
		if key[i] == '\x00' {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
}

// ─── Singleton + API publique ───────────────────────────────────────────────

var defaultPlayerAPIColl = newPlayerAPICollector(defaultPlayerAPICap)

// RecordPlayerAPICall agrège un appel API Halo attribué à un joueur. No-op si
// call ou player est vide (appels match-level non attribuables).
func RecordPlayerAPICall(call, player string, ms int64, isErr bool) {
	defaultPlayerAPIColl.record(call, player, ms, isErr)
}

// PlayerAPIStats retourne l'agrégat par (call, player), trié erreurs desc.
func PlayerAPIStats() []PlayerAPIStat { return defaultPlayerAPIColl.snapshot() }

// ResetPlayerAPIStats vide le collecteur (tests).
func ResetPlayerAPIStats() { defaultPlayerAPIColl.reset() }
