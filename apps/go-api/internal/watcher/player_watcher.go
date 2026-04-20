// Package watcher — player_watcher.go : goroutine de surveillance par joueur.
//
// Un PlayerWatcher orchestre :
//   - La FSM (state_machine.go) pour l'état du joueur
//   - Le MatchPoller (match_poller.go) pour détecter les nouveaux matchs
//   - Les callbacks de présence (RTA Xbox / Steam) pour piloter la FSM
//
// Cycle de vie :
//   1. Présence détectée (RTA/Steam) → FSM Idle→Watching + démarrage MatchPoller
//   2. Nouveau match détecté → FSM Watching→Syncing + envoi match_ids au sync
//   3. Sync terminé → FSM Syncing→Cooling (cooldown)
//   4. Cooldown expiré → FSM Cooling→Watching (si encore en jeu) ou →Idle
//   5. Présence perdue → FSM →Idle + arrêt MatchPoller
package watcher

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	// defaultCooldown après un sync réussi.
	defaultCooldown = 90 * time.Second
)

// SyncTrigger déclenche un sync pour les match_ids détectés.
type SyncTrigger interface {
	TriggerSync(ctx context.Context, gamertag, xuid string, matchIDs []string) error
}

// PlayerWatcher surveille un joueur et orchestre présence + match polling + sync.
type PlayerWatcher struct {
	gamertag string
	xuid     string

	fsm         *FSM
	fetcher     MatchFetcher
	syncTrigger SyncTrigger
	liveRefresh LiveRefreshTrigger // nil si non configuré
	cooldown    time.Duration

	pollerCancel context.CancelFunc
	pollerMu     sync.Mutex

	// inGame track si la présence dit "en jeu" (RTA ou Steam)
	inGame bool
	mu     sync.Mutex
}

// NewPlayerWatcher crée un watcher pour un joueur.
func NewPlayerWatcher(gamertag, xuid string, fetcher MatchFetcher, syncTrigger SyncTrigger) *PlayerWatcher {
	pw := &PlayerWatcher{
		gamertag:    gamertag,
		xuid:        xuid,
		fetcher:     fetcher,
		syncTrigger: syncTrigger,
		cooldown:    defaultCooldown,
	}
	pw.fsm = NewFSM(gamertag, pw.onTransition)
	return pw
}

// FSM retourne la FSM du watcher (pour lecture d'état).
func (pw *PlayerWatcher) FSM() *FSM {
	return pw.fsm
}

// WithLiveRefresh configure le refresher live BP/Challenges.
// Retourne le watcher pour permettre le chaînage.
func (pw *PlayerWatcher) WithLiveRefresh(r LiveRefreshTrigger) *PlayerWatcher {
	pw.liveRefresh = r
	return pw
}

// OnPresenceActive est appelé quand le joueur est détecté en jeu (RTA ou Steam).
func (pw *PlayerWatcher) OnPresenceActive(ctx context.Context) {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	pw.inGame = true

	state := pw.fsm.State()
	if state == StateIdle {
		if err := pw.fsm.GoWatching(); err != nil {
			slog.WarnContext(ctx, "player_watcher: erreur transition →Watching",
				"gamertag", pw.gamertag, "err", err)
			return
		}
		pw.startPoller(ctx)
	}

	if pw.liveRefresh != nil {
		pw.liveRefresh.OnPresenceActive(ctx)
	}
}

// OnPresenceInactive est appelé quand le joueur quitte le jeu.
func (pw *PlayerWatcher) OnPresenceInactive(ctx context.Context) {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	pw.inGame = false

	state := pw.fsm.State()
	if state == StateWatching || state == StateCooling {
		pw.stopPoller()
		if err := pw.fsm.GoIdle(); err != nil {
			slog.WarnContext(ctx, "player_watcher: erreur transition →Idle",
				"gamertag", pw.gamertag, "err", err)
		}
	}

	if pw.liveRefresh != nil {
		pw.liveRefresh.OnPresenceInactive(ctx)
	}
}

// OnNewMatches est le callback appelé par le MatchPoller quand de nouveaux matchs sont trouvés.
func (pw *PlayerWatcher) OnNewMatches(ctx context.Context, matchIDs []string) {
	if len(matchIDs) == 0 {
		return
	}

	state := pw.fsm.State()
	if state != StateWatching {
		slog.DebugContext(ctx, "player_watcher: matchs ignorés (état != Watching)",
			"gamertag", pw.gamertag,
			"state", state.String(),
			"count", len(matchIDs),
		)
		return
	}

	if err := pw.fsm.GoSyncing(); err != nil {
		slog.WarnContext(ctx, "player_watcher: erreur transition →Syncing",
			"gamertag", pw.gamertag, "err", err)
		return
	}

	// Lancer le sync en goroutine pour ne pas bloquer le poller
	go pw.runSync(ctx, matchIDs)
}

// OnSyncComplete est appelé quand le sync est terminé.
func (pw *PlayerWatcher) OnSyncComplete(ctx context.Context) {
	if err := pw.fsm.GoCooling(pw.cooldown); err != nil {
		slog.WarnContext(ctx, "player_watcher: erreur transition →Cooling",
			"gamertag", pw.gamertag, "err", err)
		return
	}

	// Lancer le cooldown timer en goroutine
	go pw.waitCooldown(ctx)
}

// runSync exécute le sync et gère la transition post-sync.
func (pw *PlayerWatcher) runSync(ctx context.Context, matchIDs []string) {
	slog.InfoContext(ctx, "player_watcher: démarrage sync",
		"gamertag", pw.gamertag,
		"match_count", len(matchIDs),
	)

	err := pw.syncTrigger.TriggerSync(ctx, pw.gamertag, pw.xuid, matchIDs)
	if err != nil {
		slog.ErrorContext(ctx, "player_watcher: sync échoué",
			"gamertag", pw.gamertag,
			"err", err,
		)
	} else {
		slog.InfoContext(ctx, "player_watcher: sync terminé",
			"gamertag", pw.gamertag,
		)
	}

	pw.OnSyncComplete(ctx)
}

// waitCooldown attend la fin du cooldown puis transite vers Watching ou Idle.
func (pw *PlayerWatcher) waitCooldown(ctx context.Context) {
	remaining := pw.fsm.CooldownRemaining()
	if remaining <= 0 {
		remaining = pw.cooldown
	}

	slog.DebugContext(ctx, "player_watcher: cooldown démarré",
		"gamertag", pw.gamertag,
		"duration", remaining,
	)

	select {
	case <-ctx.Done():
		return
	case <-time.After(remaining):
	}

	pw.mu.Lock()
	inGame := pw.inGame
	pw.mu.Unlock()

	if inGame {
		if err := pw.fsm.GoWatching(); err != nil {
			slog.WarnContext(ctx, "player_watcher: erreur transition Cooling→Watching",
				"gamertag", pw.gamertag, "err", err)
		}
	} else {
		if err := pw.fsm.GoIdle(); err != nil {
			slog.WarnContext(ctx, "player_watcher: erreur transition Cooling→Idle",
				"gamertag", pw.gamertag, "err", err)
		}
		pw.stopPoller()
	}
}

// startPoller démarre le MatchPoller dans une goroutine.
func (pw *PlayerWatcher) startPoller(ctx context.Context) {
	pw.pollerMu.Lock()
	defer pw.pollerMu.Unlock()

	// Arrêter un éventuel poller existant
	if pw.pollerCancel != nil {
		pw.pollerCancel()
	}

	pollerCtx, cancel := context.WithCancel(ctx)
	pw.pollerCancel = cancel

	poller := NewMatchPoller(pw.xuid, pw.gamertag, pw.fetcher, func(matchIDs []string) {
		pw.OnNewMatches(pollerCtx, matchIDs)
	})

	go poller.Run(pollerCtx)

	slog.InfoContext(ctx, "player_watcher: match poller démarré",
		"gamertag", pw.gamertag,
	)
}

// stopPoller arrête le MatchPoller.
func (pw *PlayerWatcher) stopPoller() {
	pw.pollerMu.Lock()
	defer pw.pollerMu.Unlock()

	if pw.pollerCancel != nil {
		pw.pollerCancel()
		pw.pollerCancel = nil
		slog.Info("player_watcher: match poller arrêté",
			"gamertag", pw.gamertag,
		)
	}
}

// onTransition callback de la FSM.
func (pw *PlayerWatcher) onTransition(from, to State) {
	slog.Info("player_watcher: FSM transition",
		"gamertag", pw.gamertag,
		"from", from.String(),
		"to", to.String(),
	)
}
