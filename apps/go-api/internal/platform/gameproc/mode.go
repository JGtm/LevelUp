package gameproc

import (
	"context"
	"sync/atomic"
	"time"

	"levelup/go-api/internal/observability"
)

// Compteurs expvar du mode jeu.
const (
	// MetricRunning : 1 tant que le jeu tourne (jauge), 0 sinon.
	MetricRunning = "game_mode_running"
	// MetricEnter et MetricLeave : passages « jeu lancé » et « jeu fermé » observés.
	MetricEnter = "game_mode_enter_total"
	MetricLeave = "game_mode_leave_total"
	// MetricSyncTriggered : synchronisations demandées à la fermeture du jeu.
	MetricSyncTriggered = "game_mode_sync_triggered_total"
	// MetricSyncDebounced : fermetures dont la synchronisation a été absorbée par l'anti-rebond.
	MetricSyncDebounced = "game_mode_sync_debounced_total"
	// MetricPollErrors : énumérations de processus échouées.
	MetricPollErrors = "game_mode_poll_errors_total"
	// MetricHeavyDeferred : lancements de travail lourd différés parce que le jeu tournait.
	MetricHeavyDeferred = "game_mode_heavy_deferred_total"
)

// HeavyRetryEvery : l'intervalle auquel [Mode.WaitHeavyAllowed] relit l'état du jeu.
const HeavyRetryEvery = 30 * time.Second

// Mode porte l'état « le jeu tourne » et la porte du travail lourd. Un *Mode nil est un mode
// jamais actif : tout est permis (hôte sans détection, tests).
type Mode struct {
	running atomic.Bool
	// retryEvery : intervalle de [Mode.WaitHeavyAllowed] ; zéro = [HeavyRetryEvery].
	retryEvery time.Duration
}

// NewMode crée un mode inactif.
func NewMode() *Mode { return &Mode{} }

// NewModeRetrying crée un mode inactif dont l'attente de [Mode.WaitHeavyAllowed] relit l'état toutes
// les every (tests : un rythme court à la place de [HeavyRetryEvery]).
func NewModeRetrying(every time.Duration) *Mode { return &Mode{retryEvery: every} }

// Running dit si le jeu tourne.
func (m *Mode) Running() bool { return m != nil && m.running.Load() }

// HeavyWorkAllowed dit si une tâche lourde (cuisson de rejeu, recalcul, balayage) peut démarrer
// maintenant : faux tant que le jeu tourne. Elle se consulte AVANT chaque lancement ; une tâche
// déjà lancée n'est jamais interrompue.
func (m *Mode) HeavyWorkAllowed() bool { return !m.Running() }

// set pose l'état et la jauge ; rend vrai si l'état a changé.
func (m *Mode) set(running bool) bool {
	changed := m.running.Swap(running) != running
	if running {
		observability.SetInt(MetricRunning, 1)
	} else {
		observability.SetInt(MetricRunning, 0)
	}
	return changed
}

// WaitHeavyAllowed rend la main dès que le travail lourd est permis (immédiatement hors partie),
// ou ctx.Err() si ctx est annulé avant. Compte chaque différé dans [MetricHeavyDeferred] ; l'appelant
// journalise son propre nom de tâche via onDefer, appelé une fois quand l'attente commence.
func (m *Mode) WaitHeavyAllowed(ctx context.Context, onDefer func()) error {
	if m.HeavyWorkAllowed() {
		return nil
	}
	observability.IncCounter(MetricHeavyDeferred)
	if onDefer != nil {
		onDefer()
	}
	every := HeavyRetryEvery
	if m != nil && m.retryEvery > 0 {
		every = m.retryEvery
	}
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if m.HeavyWorkAllowed() {
				return nil
			}
		}
	}
}

// processMode est le mode jeu du processus : « le jeu tourne sur ce poste » est un fait de la
// machine, pas d'une instance de tâche, et ses consommateurs (cuisson des rejeux, crons lourds)
// sont construits en des endroits qui n'ont aucune raison de se le passer.
var processMode atomic.Pointer[Mode]

func init() { processMode.Store(NewMode()) }

// Default rend le mode jeu du processus, jamais nil.
func Default() *Mode { return processMode.Load() }

// SetDefault remplace le mode jeu du processus (tests) et rend l'ancien.
func SetDefault(m *Mode) *Mode {
	if m == nil {
		m = NewMode()
	}
	return processMode.Swap(m)
}
