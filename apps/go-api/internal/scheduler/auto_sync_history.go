// Package scheduler — auto_sync_history.go : historique en mémoire des
// derniers cycles auto-sync, pour le dashboard monitoring admin.
//
// Décision (plan dashboard monitoring, D1) : ring buffer mémoire de 48
// entrées, PAS de persistance fichier. Le dernier cycle était déjà mémorisé
// sous snapshotMu ; 48 cycles couvrent 12 h à 12 j selon l'intervalle, et
// l'historique long terme existe déjà dans logs/scheduler.log (consultable
// via le viewer de logs du dashboard). Perte au restart assumée — signalée
// par history_since_boot côté API.
package scheduler

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/adminstate"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/sync"
)

// cycleHistorySize borne l'historique mémoire des cycles (FIFO).
const cycleHistorySize = 48

// postSyncHistorySize borne le ring des durées post-sync par joueur (sparkline
// de tendance — 16 derniers post-syncs réussis du joueur).
const postSyncHistorySize = 16

// CycleRecord est l'entrée d'historique d'un cycle auto-sync.
//
// Les champs « charge » (BlockedMs, SwapCount, ReadsRejected, APIMs,
// PersistWriteMs) sont des DELTAS des cumuls process-wide capturés
// avant/après le cycle — ils corrèlent la durée du cycle avec la fenêtre
// d'INDISPONIBILITÉ des lectures shared (B-swap), les 503 servis aux
// lecteurs, le temps passé en appels API Halo et le temps d'écriture DB.
// Approximation assumée : un sync concurrent (watcher/HTTP) pendant le
// cycle est compté dans le cycle (compteurs process-wide).
type CycleRecord struct {
	At         time.Time `json:"at"`
	Trigger    string    `json:"trigger"` // "tick" (boucle périodique) | "manual" (HTTP/diag/job)
	Total      int       `json:"total"`
	Synced     int       `json:"synced"`
	Skipped    int       `json:"skipped"`
	Failed     int       `json:"failed"`
	DurationMs int64     `json:"duration_ms"`

	// BlockedMs : fenêtre cumulée pendant laquelle les lectures shared étaient
	// bloquées (drain + maintien RW + reopen) durant ce cycle.
	BlockedMs int64 `json:"blocked_ms"`
	// SwapCount : swaps RO→RW complets pendant ce cycle.
	SwapCount int64 `json:"swap_count"`
	// ReadsRejected : lectures rejetées en 503 (ErrSwapTimeout) pendant ce cycle.
	ReadsRejected int64 `json:"reads_rejected"`
	// APIMs : temps cumulé d'appels API Halo (toutes goroutines du cycle).
	APIMs int64 `json:"api_ms"`
	// PersistWriteMs : temps cumulé d'écriture persist (shared_write +
	// player_write du worker batch).
	PersistWriteMs int64 `json:"persist_write_ms"`
}

// cycleLoadSnapshot capture les cumuls process-wide servant aux deltas par
// cycle. Lecture pure (expvar + snapshot provider) — zéro effet de bord.
type cycleLoadSnapshot struct {
	blockedMs      int64
	swapCount      int64
	readsRejected  int64
	apiMs          int64
	persistWriteMs int64
}

// captureCycleLoad lit les cumuls courants.
func captureCycleLoad() cycleLoadSnapshot {
	blockedCount, blockedSum, _, _ := observability.LoadDurationStats("shared_provider_blocked_window_ms")
	var apiSum int64
	for _, call := range sync.HaloAPICallNames() {
		_, sum, _, _ := observability.LoadDurationStats("halo_api_ms_" + call)
		apiSum += sum
	}
	_, sharedWrite, _, _ := observability.LoadDurationStats("persist_shared_write_ms")
	_, playerWrite, _, _ := observability.LoadDurationStats("persist_player_write_ms")
	return cycleLoadSnapshot{
		blockedMs:      blockedSum,
		swapCount:      blockedCount,
		readsRejected:  sharedprovider.Snapshot().ReadsTimedOut,
		apiMs:          apiSum,
		persistWriteMs: sharedWrite + playerWrite,
	}
}

// deltaSince retourne les deltas (clampés à 0 — Reset des métriques en test).
func (after cycleLoadSnapshot) deltaSince(before cycleLoadSnapshot) cycleLoadSnapshot {
	clamp := func(v int64) int64 {
		if v < 0 {
			return 0
		}
		return v
	}
	return cycleLoadSnapshot{
		blockedMs:      clamp(after.blockedMs - before.blockedMs),
		swapCount:      clamp(after.swapCount - before.swapCount),
		readsRejected:  clamp(after.readsRejected - before.readsRejected),
		apiMs:          clamp(after.apiMs - before.apiMs),
		persistWriteMs: clamp(after.persistWriteMs - before.persistWriteMs),
	}
}

// storeCycleResult mémorise le résultat d'un cycle (dernier cycle + append
// historique borné) avec les deltas de charge attribués au cycle. Point de
// passage UNIQUE des deux paths V1/V2 de RunOnceTrigger — thread-safe via
// snapshotMu.
//
// En sortie (hors verrou) : journalise l'action « sync_cycle » (C2 — survit au
// reboot) et persiste le snapshot post-sync (C1). ctx sert la corrélation des
// logs (event_id du tick).
func (s *AutoSyncScheduler) storeCycleResult(ctx context.Context, res *RunOnceResult, trigger string, load cycleLoadSnapshot) {
	// Statut unifié des crons (A6/DC-5) : point de convergence des deux paths
	// (V2 orchestrator + filet syncPlayer). Échec = au moins un joueur failed.
	var cycleErr error
	if res.Failed > 0 {
		cycleErr = fmt.Errorf("%d/%d joueurs en échec", res.Failed, res.Total)
	}
	observability.ReportCronRun("auto_sync", time.Now().Add(-res.Duration), cycleErr, res.Duration.Milliseconds())

	s.snapshotMu.Lock()
	s.lastCycleAt = time.Now()
	copyRes := *res
	s.lastCycleResult = &copyRes
	s.cycleRanSinceBoot = true // un cycle a tourné : le snapshot n'est plus « réhydraté seul » (C1)
	s.cycleHistory = append(s.cycleHistory, CycleRecord{
		At:             s.lastCycleAt,
		Trigger:        trigger,
		Total:          res.Total,
		Synced:         res.Synced,
		Skipped:        res.Skipped,
		Failed:         res.Failed,
		DurationMs:     res.Duration.Milliseconds(),
		BlockedMs:      load.blockedMs,
		SwapCount:      load.swapCount,
		ReadsRejected:  load.readsRejected,
		APIMs:          load.apiMs,
		PersistWriteMs: load.persistWriteMs,
	})
	if len(s.cycleHistory) > cycleHistorySize {
		// Décale la fenêtre (copie pour ne pas retenir l'array sous-jacent).
		trimmed := make([]CycleRecord, cycleHistorySize)
		copy(trimmed, s.cycleHistory[len(s.cycleHistory)-cycleHistorySize:])
		s.cycleHistory = trimmed
	}
	s.snapshotMu.Unlock()

	// C2 : journal de l'action « cycle de sync » (dernière exécution / issue /
	// déclencheur) — survit au reboot. Trigger "tick"|"manual" tel quel.
	s.actionJournal.Record(ctx, adminstate.ActionSyncCycle, adminstate.Outcome(cycleErr), trigger)
	// C1 : snapshot post-sync persistant (timeline + matrice + horodatage).
	s.persistSnapshot(ctx)
}

// History retourne une copie de l'historique des cycles, le plus récent en
// premier. Vide si aucun cycle depuis le boot (l'historique n'est pas
// persisté — cf. commentaire de tête).
func (s *AutoSyncScheduler) History() []CycleRecord {
	s.snapshotMu.RLock()
	defer s.snapshotMu.RUnlock()
	out := make([]CycleRecord, len(s.cycleHistory))
	for i, rec := range s.cycleHistory {
		out[len(s.cycleHistory)-1-i] = rec
	}
	return out
}
