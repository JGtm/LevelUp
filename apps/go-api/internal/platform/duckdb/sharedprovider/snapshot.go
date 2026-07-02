package sharedprovider

import (
	"expvar"

	"levelup/go-api/internal/observability"
)

// SwapSnapshot est une capture instantanée des compteurs de swap exposés sur
// /debug/vars. PUREMENT EN LECTURE : ne modifie aucun état du provider, ne
// touche aucune écriture, n'ouvre aucun handle. Sert le dashboard admin
// « Contention DB ».
type SwapSnapshot struct {
	State          string // ro | draining | rw | reopening | error | closed | unknown
	SwapsToRW      int64  // bascules RO→RW = nombre d'écritures shared
	SwapsToRO      int64  // bascules RW→RO (retour lecture)
	AcquireMsTotal int64  // somme des durées RO→RW (drain + close RO + open RW)
	ReleaseMsTotal int64  // somme des durées RW→RO (close RW + reopen RO)
	DrainMsTotal   int64  // somme des temps d'attente du drain des lecteurs en vol
	ReadsTimedOut  int64  // lectures rejetées (ErrSwapTimeout → HTTP 503)
	ReadersInUse   int64  // lecteurs en vol à l'instant T
	SwapFailures   int64  // échecs de swap (reopen_ro + acquire_writer + panic)
	// Fenêtre de blocage des lecteurs par swap (drain + maintien RW + reopen) —
	// la durée la plus représentative du stall ressenti. 0 tant qu'aucun swap
	// complet n'a eu lieu.
	BlockedWindowAvgMs int64
	BlockedWindowMaxMs int64
	// Phase 0 — stall lecteur réel (Get), fenêtre RW stricte, drain timeouts.
	ReaderStallNsTotal int64 // somme NS d'attente réelle des Get retardés
	ReadersDelayed     int64 // nb de Get retardés par une fenêtre RW
	RWWindowAvgMs      int64 // durée moyenne de la fenêtre RW stricte
	RWWindowMaxMs      int64 // durée max de la fenêtre RW stricte
	RWWindowCount      int64 // nb de fenêtres RW mesurées
	DrainTimeouts      int64 // échecs de drain (rollback), désambiguïsés d'acquire_writer
	// Étape 0 attribution — ventilation de la fenêtre RW PAR DÉTENTEUR (label
	// ctxkeys.DBWriterLabel), triée TotalMs décroissant, + watchdog (writer tenu
	// au-delà du seuil). C'est la donnée qui désigne les cibles du refactor
	// « writer non tenu pendant I/O ».
	RWWindowByHolder []HolderWindowStat
	WatchdogFired    int64
}

// Snapshot lit les compteurs expvar du package et retourne une capture typée.
// Process-wide (compteurs partagés entre tous les providers en multi-titre).
func Snapshot() SwapSnapshot {
	initMetrics()
	_, _, blockedAvg, blockedMax := observability.LoadDurationStats("shared_provider_blocked_window_ms")
	rwCount, _, rwAvg, rwMax := observability.LoadDurationStats("shared_provider_rw_window_ms")
	return SwapSnapshot{
		State:          currentStateLabel(),
		SwapsToRW:      mapInt(swapTotal, swapDirRoToRw),
		SwapsToRO:      mapInt(swapTotal, swapDirRwToRo),
		AcquireMsTotal: mapInt(swapDurationMsTotal, swapDirRoToRw),
		ReleaseMsTotal: mapInt(swapDurationMsTotal, swapDirRwToRo),
		DrainMsTotal:   intVal(getWaitMsTotal),
		ReadsTimedOut:  intVal(getTimeoutTotal),
		ReadersInUse:   intVal(readersInUse),
		SwapFailures: mapInt(swapFailuresTotal, failReasonReopenRO) +
			mapInt(swapFailuresTotal, failReasonAcquireWriter) +
			mapInt(swapFailuresTotal, failReasonDrainTimeout) +
			mapInt(swapFailuresTotal, failReasonPanic),
		BlockedWindowAvgMs: blockedAvg,
		BlockedWindowMaxMs: blockedMax,
		ReaderStallNsTotal: intVal(readerStallNsTotal),
		ReadersDelayed:     intVal(readerDelayedTotal),
		RWWindowAvgMs:      rwAvg,
		RWWindowMaxMs:      rwMax,
		RWWindowCount:      rwCount,
		DrainTimeouts:      mapInt(swapFailuresTotal, failReasonDrainTimeout),
		RWWindowByHolder:   holderSnapshot(),
		WatchdogFired:      intVal(watchdogFiredTotal),
	}
}

// currentStateLabel retourne l'état dont la gauge vaut >=1 (mono-titre : un
// seul état actif). "unknown" si aucun (avant le 1er enregistrement).
func currentStateLabel() string {
	for _, s := range allStates {
		if mapInt(stateGauge, s.String()) >= 1 {
			return s.String()
		}
	}
	return "unknown"
}

// mapInt lit une clé int d'un expvar.Map (les entrées créées via Map.Add sont
// des *expvar.Int). Retourne 0 si la map ou la clé est absente.
func mapInt(m *expvar.Map, key string) int64 {
	if m == nil {
		return 0
	}
	if v := m.Get(key); v != nil {
		if iv, ok := v.(*expvar.Int); ok {
			return iv.Value()
		}
	}
	return 0
}

func intVal(i *expvar.Int) int64 {
	if i == nil {
		return 0
	}
	return i.Value()
}
