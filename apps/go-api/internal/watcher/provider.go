// Package watcher — provider.go : interface d'exposition de l'état du watcher vers l'API HTTP.
//
// Le WatcherStateProvider est le seul point de contact entre le watcher
// et les handlers HTTP. Aucun handler ne doit accéder directement
// à la FSM, au RTAClient, ou au SteamPoller.
package watcher

import (
	"time"
)

// PlayerPresenceStatus représente l'état de présence d'un joueur exposé à l'API.
type PlayerPresenceStatus struct {
	Gamertag       string `json:"gamertag"`
	XUID           string `json:"xuid"`
	State          string `json:"state"`           // "Idle", "Watching", "Syncing", "Cooling"
	InGame         bool   `json:"in_game"`         // présence active
	StateSince     string `json:"state_since"`     // ISO 8601
	StateDuration  string `json:"state_duration"`  // durée lisible
	CooldownLeft   string `json:"cooldown_left,omitempty"`
}

// WatcherStatus est le résumé global du watcher exposé à l'API.
type WatcherStatus struct {
	Running        bool                    `json:"running"`
	RTAConnected   bool                    `json:"rta_connected"`
	RTASubscribed  int                     `json:"rta_subscribed"`
	PlayersWatched int                     `json:"players_watched"`
	Players        []PlayerPresenceStatus  `json:"players"`
}

// WatcherStateProvider fournit l'état du watcher en lecture seule.
type WatcherStateProvider interface {
	GetStatus() WatcherStatus
}

// StateProvider implémente WatcherStateProvider à partir du WatcherDaemon.
type StateProvider struct {
	daemon *Daemon
}

// NewStateProvider crée un provider d'état.
func NewStateProvider(daemon *Daemon) *StateProvider {
	return &StateProvider{daemon: daemon}
}

// GetStatus retourne l'état courant du watcher.
func (p *StateProvider) GetStatus() WatcherStatus {
	if p.daemon == nil {
		return WatcherStatus{Running: false}
	}

	status := WatcherStatus{
		Running: p.daemon.IsRunning(),
	}

	if p.daemon.rtaClient != nil {
		status.RTAConnected = p.daemon.rtaClient.IsConnected()
		status.RTASubscribed = len(p.daemon.rtaClient.Subscriptions())
	}

	p.daemon.playersMu.RLock()
	defer p.daemon.playersMu.RUnlock()

	status.PlayersWatched = len(p.daemon.players)
	status.Players = make([]PlayerPresenceStatus, 0, len(p.daemon.players))

	for _, pw := range p.daemon.players {
		fsm := pw.FSM()
		ps := PlayerPresenceStatus{
			Gamertag:      pw.gamertag,
			XUID:          pw.xuid,
			State:         fsm.State().String(),
			StateSince:    fsm.stateEnteredAt.Format(time.RFC3339),
			StateDuration: fsm.StateDuration().Truncate(time.Second).String(),
		}

		pw.mu.Lock()
		ps.InGame = pw.inGame
		pw.mu.Unlock()

		if fsm.State() == StateCooling {
			ps.CooldownLeft = fsm.CooldownRemaining().Truncate(time.Second).String()
		}

		status.Players = append(status.Players, ps)
	}

	return status
}
