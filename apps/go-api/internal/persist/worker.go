// Package persist — worker.go : consommateur d'une queue BatchQueue qui
// persiste chaque batch via un BatchPersister puis ACK le WAL.
//
// Architecture :
//
//	Submit ─► WAL (durable) ─► chan *MatchBatch ─► Worker.Run() ──► Persister.Persist()
//	                                                          └─► ACK = delete WAL
//
// Un Worker == 1 goroutine. Un Worker == 1 DBTarget. Pour scaler, instancier
// N workers sur la même target (mais shared.duckdb ayant un lease exclusif,
// 1 seul worker actif à la fois sur cette target).

package persist

import (
	"context"
	"log/slog"
)

// BatchPersister : abstraction du composant qui persiste un batch dans une
// DB cible. Une implémentation par target :
//
//   - SharedPersister  (shared_matches_v2.duckdb)
//   - PlayerPersister  (stats.duckdb)        — TODO Phase 1.5+
//   - PVEPersister     (shared_pve.duckdb)   — TODO Phase 1.5+
//   - MetadataPersister (metadata.duckdb)    — TODO Phase 1.5+
//
// L'interface garantit que le worker reste générique (pas de dépendance
// directe sur SharedPersister).
type BatchPersister interface {
	Persist(ctx context.Context, batch *MatchBatch) error
}

// Worker consomme une queue et persiste chaque batch reçu.
type Worker struct {
	name      string
	queue     *BatchQueue
	target    DBTarget
	persister BatchPersister

	// Hooks (optionnels) — branchés par le caller pour métriques externes
	// (expvar) sans coupler le package persist à observability.
	OnPersistOK    func()
	OnPersistError func(err error)
}

// NewWorker construit un Worker prêt à être démarré via Run.
//
//   - name      : identifiant logique pour les logs (ex. "shared", "player").
//   - queue     : queue d'où lire les batches.
//   - target    : DBTarget filtrée par ce worker (cf. queue.Channel(target)).
//   - persister : composant qui écrit dans la DB cible.
func NewWorker(name string, queue *BatchQueue, target DBTarget, persister BatchPersister) *Worker {
	return &Worker{
		name:      name,
		queue:     queue,
		target:    target,
		persister: persister,
	}
}

// Run boucle sur le channel de la queue et persiste chaque batch.
//
// Terminaison :
//   - ctx.Done()      → retourne ctx.Err() (shutdown gracieux).
//   - channel fermé   → retourne nil (queue.Close() = signal de fin).
//
// Sur erreur de Persist : log + pas d'ACK → le WAL reste sur disque pour
// retry au prochain boot via queue.RecoverPending(). Le worker continue
// à traiter les batches suivants (pas de blocage sur 1 batch problématique).
func (w *Worker) Run(ctx context.Context) error {
	ch := w.queue.Channel(w.target)
	slog.InfoContext(ctx, "persist worker started",
		"name", w.name, "target", string(w.target))

	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "persist worker shutdown",
				"name", w.name, "reason", "ctx.Done")
			return ctx.Err()
		case batch, ok := <-ch:
			if !ok {
				slog.InfoContext(ctx, "persist worker shutdown",
					"name", w.name, "reason", "channel closed")
				return nil
			}
			w.handle(ctx, batch)
		}
	}
}

// handle traite un batch unique : Persist puis ACK si OK.
//
// Phase 6 PLAN_FIX_SYNC_RELIABILITY_2026-05-24 : appelle systematiquement
// queue.RecordPersistResult() apres Persist pour alimenter le circuit-breaker
// sur Drain. Les hooks OnPersistOK/OnPersistError restent pour l'observabilite
// externe (expvar) — ils sont independants du circuit-breaker.
func (w *Worker) handle(ctx context.Context, batch *MatchBatch) {
	if err := w.persister.Persist(ctx, batch); err != nil {
		slog.ErrorContext(ctx, "persist worker: Persist failed",
			"name", w.name,
			"batch_id", batch.BatchID,
			"source", batch.Source,
			"err", err,
		)
		w.queue.RecordPersistResult(false) // alimente circuit-breaker
		if w.OnPersistError != nil {
			w.OnPersistError(err)
		}
		return
	}

	if err := w.queue.ACK(batch.BatchID); err != nil {
		// ACK failure = WAL file removal failed. Pas critique (le batch a
		// été persisté). Le WAL résiduel sera re-tenté au prochain boot et
		// idempotence côté SharedPersister (match_id exists check) skip.
		slog.ErrorContext(ctx, "persist worker: ACK failed",
			"name", w.name,
			"batch_id", batch.BatchID,
			"err", err,
		)
	}

	w.queue.RecordPersistResult(true) // reset circuit-breaker
	if w.OnPersistOK != nil {
		w.OnPersistOK()
	}
}
