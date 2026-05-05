package dblease

import (
	"expvar"
	"sync"
	"sync/atomic"
)

// Compteurs expvar exposés sur /debug/vars. Cardinalité bornée par Kind
// (4 valeurs : player / shared_matches / shared_social / metadata) — pas par
// chemin individuel, sinon explosion en multi-user (cf. ADR-0009).
//
// Variables publiées :
//   - dblease_acquire_total{kind}        : nombre d'acquisitions réussies
//   - dblease_acquire_timeout_total{kind}: nombre de timeouts (ErrDBLocked)
//   - dblease_wait_duration_ms_total{kind}: somme des temps d'attente (ms)
//   - dblease_writers_in_use{kind}       : nombre de writers actuellement tenus
//
// Pour la latence moyenne d'attente : wait_duration_ms_total / acquire_total.
// Pas d'histogramme expvar disponible — la moyenne suffit en multi-user.

var (
	metricsOnce sync.Once

	acquireTotal        *expvar.Map
	acquireTimeoutTotal *expvar.Map
	waitDurationMsTotal *expvar.Map
	writersInUse        *expvar.Map

	// totalInUse compte les writers tenus tous Kind confondus. Utilisé par
	// AssertNoLeasedWriters pour vérifier l'absence de fuite dans les tests.
	totalInUse atomic.Int64
)

// init force l'initialisation des compteurs expvar dès l'import du package,
// pour que /debug/vars expose les 4 Kind dès le boot et que les tests puissent
// lire les compteurs avant toute acquisition.
func init() {
	initMetrics()
}

func initMetrics() {
	metricsOnce.Do(func() {
		acquireTotal = expvar.NewMap("dblease_acquire_total")
		acquireTimeoutTotal = expvar.NewMap("dblease_acquire_timeout_total")
		waitDurationMsTotal = expvar.NewMap("dblease_wait_duration_ms_total")
		writersInUse = expvar.NewMap("dblease_writers_in_use")
		// Pré-initialise les 4 Kind pour que /debug/vars expose toutes les clés
		// dès le boot, même celles jamais vues encore.
		for _, k := range allKinds {
			acquireTotal.Add(string(k), 0)
			acquireTimeoutTotal.Add(string(k), 0)
			waitDurationMsTotal.Add(string(k), 0)
			writersInUse.Add(string(k), 0)
		}
	})
}

func recordAcquire(k Kind, waitMs int64) {
	initMetrics()
	acquireTotal.Add(string(k), 1)
	waitDurationMsTotal.Add(string(k), waitMs)
	writersInUse.Add(string(k), 1)
	totalInUse.Add(1)
}

func recordRelease(k Kind) {
	initMetrics()
	writersInUse.Add(string(k), -1)
	totalInUse.Add(-1)
}

func recordTimeout(k Kind) {
	initMetrics()
	acquireTimeoutTotal.Add(string(k), 1)
}

// LeasedWritersInUse retourne le nombre total de writers actuellement tenus
// (tous Kind confondus). Exposé pour les helpers de test (AssertNoLeasedWriters)
// et pour observation runtime via reflection si besoin.
//
// Sous concurrence, la valeur peut être lue tandis qu'une autre goroutine est
// au milieu d'un acquire/release — c'est un point de mesure, pas un verrou.
func LeasedWritersInUse() int64 {
	return totalInUse.Load()
}
