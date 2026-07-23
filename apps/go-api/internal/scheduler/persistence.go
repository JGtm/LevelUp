// Package scheduler — persistence.go : persistance JSON légère (HORS DuckDB) du
// snapshot post-sync, pour guérir l'amnésie du dashboard admin au reboot (C1,
// plan diag apparence admin).
//
// En fin de cycle, storeCycleResult persiste un snapshot léger (dernier cycle +
// détail par joueur + ring des durées post-sync). Au boot, RehydrateFromDisk le
// recharge dans l'état mémoire : le dashboard affiche alors le dernier cycle
// connu (daté, marqué « avant redémarrage » via SchedulerSnapshot.SinceBoot=false)
// au lieu d'une timeline/matrice vides. Un premier cycle depuis le boot bascule
// SinceBoot à true. AUCUNE écriture DuckDB.
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/platform/adminstate"
)

// WithSnapshotStore branche le store JSON de persistance du snapshot post-sync
// (C1). À appeler au boot AVANT RehydrateFromDisk. Nil → persistance off.
func (s *AutoSyncScheduler) WithSnapshotStore(store *adminstate.FileStore) *AutoSyncScheduler {
	s.snapshotStore = store
	return s
}

// WithActionJournal branche le journal des actions globales (C2) : le scheduler
// y enregistre chaque cycle de sync (action « sync_cycle », déclencheur
// tick/manual). Nil → non journalisé.
func (s *AutoSyncScheduler) WithActionJournal(j *adminstate.ActionJournal) *AutoSyncScheduler {
	s.actionJournal = j
	return s
}

// persistedSnapshotVersion versionne le format du fichier (tolérance d'évolution).
const persistedSnapshotVersion = 1

// persistedSnapshot est la forme sérialisée du snapshot post-sync. Types
// scheduler réutilisés tels quels (JSON-contractés) — le fichier reste lisible et
// léger (quelques joueurs).
type persistedSnapshot struct {
	Version         int                   `json:"version"`
	SavedAt         time.Time             `json:"saved_at"`
	LastCycleAt     time.Time             `json:"last_cycle_at"`
	LastCycleResult *RunOnceResult        `json:"last_cycle_result,omitempty"`
	Players         []PlayerOutcomeDetail `json:"players"`
	PostSyncHistory map[string][]int64    `json:"post_sync_history,omitempty"`
}

// RehydrateFromDisk recharge le dernier snapshot post-sync persisté dans l'état
// mémoire (à appeler UNE fois au boot, après WithSnapshotStore, avant Run).
//
// Dégradation propre : store non câblé → no-op ; fichier absent → premier boot
// (debug) ; fichier corrompu → ErrorContext loggé et démarrage sans historique
// (jamais fatal). cycleRanSinceBoot reste false : le snapshot réhydraté est
// marqué « cycle précédent » côté API tant qu'aucun cycle n'a tourné.
func (s *AutoSyncScheduler) RehydrateFromDisk(ctx context.Context) {
	if s.snapshotStore == nil {
		return
	}
	var snap persistedSnapshot
	found, err := s.snapshotStore.Load(&snap)
	if err != nil {
		slog.ErrorContext(ctx, "auto_sync: snapshot post-sync illisible — démarrage sans historique",
			"path", s.snapshotStore.Path(), "err", err)
		return
	}
	if !found {
		slog.DebugContext(ctx, "auto_sync: aucun snapshot post-sync persisté (premier boot)",
			"path", s.snapshotStore.Path())
		return
	}

	s.snapshotMu.Lock()
	defer s.snapshotMu.Unlock()
	s.lastCycleAt = snap.LastCycleAt
	s.lastCycleResult = snap.LastCycleResult
	if s.playerOutcomes == nil {
		s.playerOutcomes = make(map[string]PlayerOutcomeDetail, len(snap.Players))
	}
	for _, d := range snap.Players {
		// PostSyncHistoryMs est ré-inliné par Snapshot() depuis postSyncHistory :
		// ne pas le dupliquer dans la map d'outcomes (source unique = le ring).
		d.PostSyncHistoryMs = nil
		s.playerOutcomes[d.Gamertag] = d
	}
	if snap.PostSyncHistory != nil {
		s.postSyncHistory = snap.PostSyncHistory
	}
	s.cycleRanSinceBoot = false
	slog.InfoContext(ctx, "auto_sync: snapshot post-sync réhydraté (dashboard non amnésique au reboot)",
		"players", len(snap.Players), "last_cycle_at", snap.LastCycleAt, "saved_at", snap.SavedAt)
}

// persistSnapshot écrit le snapshot post-sync courant sur disque (best-effort).
// Appelé en fin de storeCycleResult, hors verrou snapshotMu. Un échec est LOGGÉ
// (jamais avalé) : l'API continue de servir l'état mémoire.
func (s *AutoSyncScheduler) persistSnapshot(ctx context.Context) {
	if s.snapshotStore == nil {
		return
	}
	snap := s.buildPersistedSnapshot()
	if err := s.snapshotStore.Save(snap); err != nil {
		slog.ErrorContext(ctx, "auto_sync: persistance du snapshot post-sync échouée (état mémoire servi)",
			"path", s.snapshotStore.Path(), "err", err)
		return
	}
	slog.DebugContext(ctx, "auto_sync: snapshot post-sync persisté",
		"players", len(snap.Players), "last_cycle_at", snap.LastCycleAt)
}

// buildPersistedSnapshot copie l'état mémoire courant vers la forme sérialisée
// (sous RLock — copies défensives).
func (s *AutoSyncScheduler) buildPersistedSnapshot() persistedSnapshot {
	s.snapshotMu.RLock()
	defer s.snapshotMu.RUnlock()
	players := make([]PlayerOutcomeDetail, 0, len(s.playerOutcomes))
	for _, d := range s.playerOutcomes {
		players = append(players, d)
	}
	var res *RunOnceResult
	if s.lastCycleResult != nil {
		r := *s.lastCycleResult
		res = &r
	}
	hist := make(map[string][]int64, len(s.postSyncHistory))
	for k, v := range s.postSyncHistory {
		hist[k] = append([]int64(nil), v...)
	}
	return persistedSnapshot{
		Version:         persistedSnapshotVersion,
		SavedAt:         time.Now().UTC(),
		LastCycleAt:     s.lastCycleAt,
		LastCycleResult: res,
		Players:         players,
		PostSyncHistory: hist,
	}
}
