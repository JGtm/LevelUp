package sharedprovider

// holder_metrics.go — ventilation de la fenêtre RW par DÉTENTEUR du writer
// (étape 0 attribution contention). Le label est posé par le caller via
// ctxkeys.WithDBWriterLabel à l'entrée de son sous-système, capturé par
// AcquireWriter (swapToRW) et consommé au releaseWriter. C'est la donnée qui
// attribue directement la fenêtre RW moyenne (~13,5s mesurée) à ses vrais
// détenteurs — préalable au refactor « writer non tenu pendant I/O ».
//
// Cardinalité BORNÉE (ADR 0009) : les labels sont des constantes compile-time ;
// cap défensif maxHolderLabels avec éviction least-active si un label dynamique
// fuyait par erreur (même garde que observability/player_api_collector.go).

import (
	"expvar"
	"sort"
	"sync"
)

// maxHolderLabels borne le nombre de labels distincts suivis (garde ADR 0009).
const maxHolderLabels = 32

// holderStats accumule les fenêtres RW d'un détenteur.
type holderStats struct {
	count         int64
	sumMS         int64
	maxMS         int64
	watchdogFired int64
}

// holderCollector est le collecteur process-wide (partagé entre providers en
// multi-titre, comme les autres compteurs du package).
type holderCollector struct {
	mu      sync.Mutex
	entries map[string]*holderStats
}

var holders = &holderCollector{entries: make(map[string]*holderStats)}

// watchdogFiredTotal compte les détentions du writer au-delà du seuil watchdog
// (tous labels confondus). Publié par initHolderMetrics.
var watchdogFiredTotal *expvar.Int

// entryLocked retourne (en créant si besoin) l'entrée du label, sous c.mu.
// Au cap, évince l'entrée la moins active (count le plus bas) — un label
// dynamique fuité ne peut pas faire grossir la map indéfiniment.
func (c *holderCollector) entryLocked(label string) *holderStats {
	if e, ok := c.entries[label]; ok {
		return e
	}
	if len(c.entries) >= maxHolderLabels {
		var evict string
		var evictCount int64 = -1
		for l, e := range c.entries {
			if evictCount == -1 || e.count < evictCount {
				evict, evictCount = l, e.count
			}
		}
		delete(c.entries, evict)
	}
	e := &holderStats{}
	c.entries[label] = e
	return e
}

// recordHolderWindow accumule une fenêtre RW terminée pour un label.
func recordHolderWindow(label string, ms int64) {
	holders.mu.Lock()
	defer holders.mu.Unlock()
	e := holders.entryLocked(label)
	e.count++
	e.sumMS += ms
	if ms > e.maxMS {
		e.maxMS = ms
	}
}

// recordHolderWatchdog incrémente le compteur watchdog d'un label (writer tenu
// au-delà du seuil — le WARN associé est émis par le timer, pas ici).
func recordHolderWatchdog(label string) {
	holders.mu.Lock()
	defer holders.mu.Unlock()
	holders.entryLocked(label).watchdogFired++
}

// HolderWindowStat est la vue exportée d'un détenteur pour Snapshot() et la
// carte admin « Contention DB ». AvgMs dérivé à la lecture (comme LoadDurationStats).
type HolderWindowStat struct {
	Label         string
	Count         int64
	TotalMs       int64
	AvgMs         int64
	MaxMs         int64
	WatchdogFired int64
}

// holderSnapshot retourne la ventilation triée par TotalMs décroissant (le
// plus gros détenteur en premier — l'ordre de lecture naturel pour attribuer).
func holderSnapshot() []HolderWindowStat {
	holders.mu.Lock()
	defer holders.mu.Unlock()
	out := make([]HolderWindowStat, 0, len(holders.entries))
	for label, e := range holders.entries {
		avg := int64(0)
		if e.count > 0 {
			avg = e.sumMS / e.count
		}
		out = append(out, HolderWindowStat{
			Label: label, Count: e.count, TotalMs: e.sumMS,
			AvgMs: avg, MaxMs: e.maxMS, WatchdogFired: e.watchdogFired,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TotalMs > out[j].TotalMs })
	return out
}

// initHolderMetrics publie les expvar de la ventilation. Appelé UNE fois par
// initMetrics (sous metricsOnce) — expvar.Publish panique sur nom dupliqué.
func initHolderMetrics() {
	watchdogFiredTotal = expvar.NewInt("shared_provider_rw_watchdog_fired_total")
	expvar.Publish("shared_provider_rw_window_by_holder", expvar.Func(func() any {
		snap := holderSnapshot()
		out := make(map[string]map[string]int64, len(snap))
		for _, h := range snap {
			out[h.Label] = map[string]int64{
				"count":          h.Count,
				"total_ms":       h.TotalMs,
				"avg_ms":         h.AvgMs,
				"max_ms":         h.MaxMs,
				"watchdog_fired": h.WatchdogFired,
			}
		}
		return out
	}))
}
