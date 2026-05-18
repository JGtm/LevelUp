package sharedprovider

import (
	"expvar"
	"sync"
)

// Compteurs expvar exposés sur /debug/vars. Cardinalité bornée par State et
// direction du swap, jamais par chemin individuel — sinon explosion en
// multi-titre (cf. ADR-0009).
//
// Variables publiées :
//
//   - shared_provider_state{state}                       : gauge 0/1, 1 sur l'état courant
//   - shared_provider_swap_total{direction}              : compteur (ro_to_rw, rw_to_ro)
//   - shared_provider_swap_duration_ms_total{direction}  : somme des durées de swap (ms)
//   - shared_provider_swap_failures_total{reason}        : compteur échecs (reopen_ro, acquire_writer, panic)
//   - shared_provider_get_wait_ms_total                  : somme des attentes Get côté swap
//   - shared_provider_get_timeout_total                  : compteur ErrSwapTimeout
//   - shared_provider_readers_in_use                     : gauge readers en vol
//
// Pour la durée moyenne d'un swap : swap_duration_ms_total / swap_total.
//
// État du squelette (commit 2/9) : toutes les clés sont publiées et
// initialisées à zéro. Seul stateGauge est mis à jour activement (par
// recordState lors d'un changement d'état). Les autres compteurs deviennent
// vivants au commit 3 (swap RW) et au commit 4 (drain readers).
var (
	metricsOnce sync.Once

	stateGauge          *expvar.Map
	swapTotal           *expvar.Map
	swapDurationMsTotal *expvar.Map
	swapFailuresTotal   *expvar.Map
	getWaitMsTotal      *expvar.Int
	getTimeoutTotal     *expvar.Int
	readersInUse        *expvar.Int
)

const (
	swapDirRoToRw = "ro_to_rw"
	swapDirRwToRo = "rw_to_ro"

	failReasonReopenRO      = "reopen_ro"
	failReasonAcquireWriter = "acquire_writer"
	failReasonPanic         = "panic"
)

// init force la publication des clés dès l'import du package, pour que
// /debug/vars expose tous les états et directions de swap dès le boot,
// même ceux jamais atteints.
func init() {
	initMetrics()
}

func initMetrics() {
	metricsOnce.Do(func() {
		stateGauge = expvar.NewMap("shared_provider_state")
		swapTotal = expvar.NewMap("shared_provider_swap_total")
		swapDurationMsTotal = expvar.NewMap("shared_provider_swap_duration_ms_total")
		swapFailuresTotal = expvar.NewMap("shared_provider_swap_failures_total")
		getWaitMsTotal = expvar.NewInt("shared_provider_get_wait_ms_total")
		getTimeoutTotal = expvar.NewInt("shared_provider_get_timeout_total")
		readersInUse = expvar.NewInt("shared_provider_readers_in_use")

		for _, s := range allStates {
			stateGauge.Add(s.String(), 0)
		}
		for _, d := range []string{swapDirRoToRw, swapDirRwToRo} {
			swapTotal.Add(d, 0)
			swapDurationMsTotal.Add(d, 0)
		}
		for _, r := range []string{failReasonReopenRO, failReasonAcquireWriter, failReasonPanic} {
			swapFailuresTotal.Add(r, 0)
		}
	})
}

// recordStateTransition met à jour la gauge stateGauge : décrémente l'état
// précédent (s'il est différent) et incrémente le nouveau. La somme reste
// égale à 1 (un seul provider en mono-titre — au commit 5 quand le Manager
// crée plusieurs providers, la somme = nombre de providers actifs).
//
// Appelée par providerImpl à chaque changement d'état.
func recordStateTransition(prev, next State) {
	initMetrics()
	if prev != next {
		stateGauge.Add(prev.String(), -1)
	}
	stateGauge.Add(next.String(), 1)
}
