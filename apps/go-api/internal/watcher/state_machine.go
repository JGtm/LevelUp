// Package watcher — state_machine.go : FSM de monitoring d'un joueur.
//
// États :
//   - Idle       : joueur pas en jeu → pas de polling
//   - Watching   : joueur en jeu → poll API Halo pour détecter nouveaux matchs
//   - Syncing    : sync en cours sur ce joueur
//   - Cooling    : cooldown post-sync avant retour à Watching
//
// Transitions valides :
//   - Idle       → Watching  (présence détectée : joueur lance le jeu)
//   - Watching   → Idle      (présence perdue : joueur quitte le jeu)
//   - Watching   → Syncing   (nouveau match détecté)
//   - Syncing    → Cooling   (sync terminé)
//   - Cooling    → Watching  (cooldown expiré, joueur encore en jeu)
//   - Cooling    → Idle      (cooldown expiré, joueur a quitté)
package watcher

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// State représente l'état courant de la FSM.
type State int

const (
	StateIdle     State = iota // pas en jeu
	StateWatching              // en jeu, polling des matchs
	StateSyncing               // sync en cours
	StateCooling               // cooldown post-sync
)

// String retourne le nom lisible de l'état.
func (s State) String() string {
	switch s {
	case StateIdle:
		return "Idle"
	case StateWatching:
		return "Watching"
	case StateSyncing:
		return "Syncing"
	case StateCooling:
		return "Cooling"
	default:
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

// TransitionCallback est appelé à chaque changement d'état.
type TransitionCallback func(from, to State)

// FSM gère l'état d'un joueur pour le watcher.
type FSM struct {
	mu       sync.RWMutex
	state    State
	gamertag string

	// Timing
	stateEnteredAt time.Time
	cooldownEnd    time.Time

	// Callback optionnel
	onTransition TransitionCallback
}

// NewFSM crée une FSM en état Idle.
func NewFSM(gamertag string, onTransition TransitionCallback) *FSM {
	return &FSM{
		state:          StateIdle,
		gamertag:       gamertag,
		stateEnteredAt: time.Now(),
		onTransition:   onTransition,
	}
}

// State retourne l'état courant.
func (f *FSM) State() State {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state
}

// Gamertag retourne le gamertag associé.
func (f *FSM) Gamertag() string {
	return f.gamertag
}

// StateDuration retourne le temps passé dans l'état courant.
func (f *FSM) StateDuration() time.Duration {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return time.Since(f.stateEnteredAt)
}

// StateEnteredAt retourne l'instant d'entrée dans l'état courant (lecture
// synchronisée — ne jamais lire f.stateEnteredAt directement depuis l'extérieur,
// data race avec les transitions FSM ; revue 2026-06-02).
func (f *FSM) StateEnteredAt() time.Time {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.stateEnteredAt
}

// CooldownRemaining retourne le temps restant de cooldown (0 si pas en cooling).
func (f *FSM) CooldownRemaining() time.Duration {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.state != StateCooling {
		return 0
	}
	remaining := time.Until(f.cooldownEnd)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// --- Transitions ---

// GoWatching : Idle → Watching (joueur détecté en jeu).
func (f *FSM) GoWatching() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.state != StateIdle && f.state != StateCooling {
		return fmt.Errorf("fsm(%s): transition invalide %s → Watching", f.gamertag, f.state)
	}
	f.transition(StateWatching)
	return nil
}

// GoIdle : Watching|Cooling → Idle (joueur quitte le jeu).
func (f *FSM) GoIdle() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.state != StateWatching && f.state != StateCooling {
		return fmt.Errorf("fsm(%s): transition invalide %s → Idle", f.gamertag, f.state)
	}
	f.transition(StateIdle)
	return nil
}

// GoSyncing : Watching → Syncing (nouveau match détecté, sync lancé).
func (f *FSM) GoSyncing() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.state != StateWatching {
		return fmt.Errorf("fsm(%s): transition invalide %s → Syncing", f.gamertag, f.state)
	}
	f.transition(StateSyncing)
	return nil
}

// GoCooling : Syncing → Cooling (sync terminé, cooldown démarre).
func (f *FSM) GoCooling(cooldown time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.state != StateSyncing {
		return fmt.Errorf("fsm(%s): transition invalide %s → Cooling", f.gamertag, f.state)
	}
	f.cooldownEnd = time.Now().Add(cooldown)
	f.transition(StateCooling)
	return nil
}

// transition effectue le changement d'état (appelé sous lock).
func (f *FSM) transition(to State) {
	from := f.state
	f.state = to
	f.stateEnteredAt = time.Now()

	slog.Info("fsm: transition",
		"gamertag", f.gamertag,
		"from", from.String(),
		"to", to.String(),
	)

	if f.onTransition != nil {
		f.onTransition(from, to)
	}
}
