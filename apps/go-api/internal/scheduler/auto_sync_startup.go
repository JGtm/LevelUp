// Package scheduler — auto_sync_startup.go : le premier cycle d'une boucle qui démarre avec une dette.
//
// La boucle de Run ne garde rien de sa cadence d'un démarrage à l'autre : un serveur redémarré armait son
// premier tick à l'intervalle réglé (6 h par défaut) même quand un joueur avait son historique incomplet, et
// le rattrapage restait à l'arrêt des heures (sur le bureau : à chaque fermeture et réouverture de
// l'application). Au démarrage, la dette se relit comme après un cycle (joueur à l'historique incomplet,
// lecture en échec comprise ; arriéré de films quand son lecteur est branché) et le premier cycle vient
// après catchUpCycleInterval au plus, tout de suite si le dernier cycle connu est plus ancien.
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
)

// startupCatchUpMinDelay est le délai le plus court du premier cycle d'un démarrage avec une dette : le temps
// que le serveur finisse de s'installer, jamais un tick immédiat dans le démarrage lui-même.
const startupCatchUpMinDelay = 5 * time.Second

// firstCycleDelay rend le délai avant le premier cycle de la boucle : l'intervalle réglé, sauf quand
// l'auto-sync est activée et qu'une dette reste à rattraper, auquel cas catchUpCycleInterval au plus, réduit
// au temps qui manque pour atteindre catchUpCycleInterval depuis le dernier cycle connu (réhydraté du disque).
// Un intervalle réglé qui n'est pas plus long que catchUpCycleInterval garde la main, comme après un cycle.
func (s *AutoSyncScheduler) firstCycleDelay(ctx context.Context, interval time.Duration) time.Duration {
	if catchUpCycleInterval >= interval {
		return interval
	}
	cfg, err := s.settings.Load()
	if err != nil || !cfg.SpnkrAutoSyncEnabled {
		return interval
	}
	players, err := s.cfg.LoadPlayers()
	if err != nil {
		slog.WarnContext(ctx, "auto_sync: chargement des joueurs échoué au démarrage — premier cycle à l'intervalle réglé", "err", err)
		return interval
	}
	players = domain.SyncablePlayers(players)
	debt := cycleDebt{history: s.pendingHistoryOf(ctx, players), films: s.filmBacklogOf(players)}
	if !debt.pending() {
		return interval
	}
	delay := catchUpCycleInterval
	s.snapshotMu.RLock()
	last := s.lastCycleAt
	s.snapshotMu.RUnlock()
	if !last.IsZero() {
		delay = min(delay, max(startupCatchUpMinDelay, last.Add(catchUpCycleInterval).Sub(s.clock())))
	}
	slog.InfoContext(ctx, "auto_sync: dette au démarrage — premier cycle avancé",
		"joueurs_en_rattrapage", len(debt.history), "gamertags", debt.history,
		"films_en_retard", debt.films.Total(), "dernier_cycle", last,
		"delai", delay, "intervalle_regle", interval)
	return delay
}
