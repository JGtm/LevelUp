// Package persist — worker.go : consommateur d'une queue BatchQueue qui
// persiste chaque batch via un BatchPersister puis ACK le WAL.
//
// Architecture :
//
//	Submit ─► WAL (durable) ─► chan *MatchBatch ─► Worker.Run() ──► Persister.Persist()
//	                                                          └─► ACK = delete WAL
//
// Un Worker == 1 goroutine consommant le channel UNIQUE de la queue. En prod il
// y a UN seul worker, dont le persister (CombinedPersister) écrit shared ET player
// par batch — il n'y a PAS de routage par DBTarget (le batch porte tous ses
// sous-batches). Le champ `target` est un simple label d'observabilité (logs).

package persist

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

// Retry+backoff (PLAN_PERSIST_ROBUSTNESS [A]) : valeurs par défaut. Le retry
// ne se déclenche QUE sur erreurs transitoires (lock/busy/IO) — jamais sur
// parse/contrainte (sinon boucle poison). Complément de la recovery périodique
// (Phase 1) : réduit le churn en re-tentant tout de suite un échec passager au
// lieu d'attendre le prochain tick de recovery.
const (
	defaultMaxPersistAttempts = 3
	defaultRetryBaseDelay     = 1 * time.Second
)

// transientErrorMarkers : sous-chaînes (lower-case) identifiant une erreur
// Persist TRANSITOIRE qui mérite un retry. Allowlist volontaire : par défaut
// une erreur est considérée PERMANENTE (pas de retry) — seul un marqueur connu
// déclenche le backoff. Évite de re-tenter à l'infini un batch poison.
var transientErrorMarkers = []string{
	"database is locked",
	"could not set lock",
	"conflicting lock",
	"lock on file",
	"database is busy",
	"resource busy",
	"i/o error",
	"input/output error",
	"disk i/o",
	"timeout",
	"deadline exceeded",
	"connection reset",
}

// isTransientPersistError retourne true si err correspond à un marqueur connu
// d'erreur transitoire (cf. transientErrorMarkers). nil → false.
func isTransientPersistError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, marker := range transientErrorMarkers {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

// BatchPersister : abstraction du composant qui persiste un batch dans une
// DB cible. Une implémentation par target :
//
//   - SharedPersister   (shared_matches_v2.duckdb)
//   - PlayerPersister   (stats.duckdb)
//   - PVEPersister      (shared_pve.duckdb)
//   - MetadataPersister (metadata.duckdb)
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

	// Retry+backoff sur erreurs transitoires (cf. isTransientPersistError).
	// Défauts posés par NewWorker. Champs non-exportés : surchargés par les
	// tests du package (delay court) pour ne pas dormir des secondes.
	maxPersistAttempts int
	retryBaseDelay     time.Duration

	// Hooks (optionnels) — branchés par le caller pour métriques externes
	// (expvar) sans coupler le package persist à observability.
	OnPersistOK    func()
	OnPersistError func(err error)
}

// NewWorker construit un Worker prêt à être démarré via Run.
//
//   - name      : identifiant logique pour les logs (ex. "shared", "player").
//   - queue     : queue d'où lire les batches.
//   - target    : label d'observabilité (logs/métriques) — le worker NE filtre
//     PAS par target (channel unique partagé, cf. queue.Channel).
//   - persister : composant qui écrit dans la/les DB cible(s).
func NewWorker(name string, queue *BatchQueue, target DBTarget, persister BatchPersister) *Worker {
	return &Worker{
		name:               name,
		queue:              queue,
		target:             target,
		persister:          persister,
		maxPersistAttempts: defaultMaxPersistAttempts,
		retryBaseDelay:     defaultRetryBaseDelay,
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
	ch := w.queue.Channel()
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

// handle traite un batch unique : Persist (avec retry transitoire) puis ACK
// si OK. Efface TOUJOURS l'entrée inFlight en sortie (succès OU échec) : un
// batch échoué redevient éligible à la recovery périodique (Phase 1).
//
// Phase 6 PLAN_FIX_SYNC_RELIABILITY_2026-05-24 : appelle systematiquement
// queue.RecordPersistResult() apres Persist pour alimenter le circuit-breaker
// sur Drain. Les hooks OnPersistOK/OnPersistError restent pour l'observabilite
// externe (expvar) — ils sont independants du circuit-breaker.
func (w *Worker) handle(ctx context.Context, batch *MatchBatch) {
	defer w.queue.clearInFlight(batch.BatchID)

	if err := w.persistWithRetry(ctx, batch); err != nil {
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

// persistWithRetry appelle Persist en re-tentant UNIQUEMENT sur erreur
// transitoire (lock/busy/IO), avec backoff exponentiel (base, 2×base, …).
// Une erreur permanente (parse/contrainte) retourne immédiatement (pas de
// boucle poison). En cas d'épuisement des tentatives ou de ctx annulé, retourne
// la dernière erreur — le batch reste en WAL (pas d'ACK) → repris par la
// recovery périodique. Cf. PLAN_PERSIST_ROBUSTNESS [A].
func (w *Worker) persistWithRetry(ctx context.Context, batch *MatchBatch) error {
	attempts := w.maxPersistAttempts
	if attempts < 1 {
		attempts = 1
	}
	var err error
	for attempt := 1; attempt <= attempts; attempt++ {
		err = w.persister.Persist(ctx, batch)
		if err == nil {
			return nil
		}
		if !isTransientPersistError(err) {
			return err // permanente → pas de retry
		}
		if attempt == attempts {
			return err // dernière tentative épuisée
		}
		delay := w.retryBaseDelay * time.Duration(1<<(attempt-1)) // 1×, 2×, 4×…
		slog.WarnContext(ctx, "persist worker: retry erreur transitoire",
			"name", w.name, "batch_id", batch.BatchID,
			"attempt", attempt, "max", attempts, "delay", delay, "err", err)
		select {
		case <-ctx.Done():
			return ctx.Err() // shutdown → laisse en WAL
		case <-time.After(delay):
		}
	}
	return err
}
