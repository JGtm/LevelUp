package assets

import (
	"context"
	"log/slog"
	"math"
	"time"
)

const (
	writeQueueCap   = 256
	writeMaxRetry   = 5
	writeBackoffMin = 50 * time.Millisecond
	writeBackoffMax = 2 * time.Second
)

// writeJob est un travail d'écriture dans l'index DuckDB.
type writeJob struct {
	ref   Ref
	entry IndexEntry
}

// WriteQueue sérialise toutes les écritures DuckDB pour un IndexStore.
// Une seule goroutine writer consomme le channel ; elle retry en cas de lock.
// Garantit que DuckDB ne reçoit jamais d'écritures concurrentes intra-process.
type WriteQueue struct {
	ch       chan writeJob
	idxStore IndexStore
	metrics  Metrics
	done     chan struct{}
}

// NewWriteQueue crée et démarre une WriteQueue.
// Le caller doit appeler Shutdown pour arrêter proprement la goroutine.
func NewWriteQueue(idxStore IndexStore, m Metrics) *WriteQueue {
	q := &WriteQueue{
		ch:       make(chan writeJob, writeQueueCap),
		idxStore: idxStore,
		metrics:  m,
		done:     make(chan struct{}),
	}
	go q.run()
	return q
}

// Enqueue soumet un job d'écriture de façon non bloquante.
// Si la queue est pleine, le job est dropé (log Warn + métrique).
func (q *WriteQueue) Enqueue(ref Ref, entry IndexEntry) {
	select {
	case q.ch <- writeJob{ref: ref, entry: entry}:
		slog.Debug("assets: persist_index_enqueued", ref.LogAttrs()...)
	default:
		slog.Warn("assets: persist_index_overflow — job dropped", ref.LogAttrs()...)
		if q.metrics != nil {
			q.metrics.IncIndexWriteOverflow()
		}
	}
}

// Shutdown attend que la queue soit vidée (ou timeout) et arrête la goroutine.
func (q *WriteQueue) Shutdown(ctx context.Context) {
	close(q.ch)
	select {
	case <-q.done:
	case <-ctx.Done():
		slog.Warn("assets: WriteQueue shutdown timed out")
	}
}

// run est la goroutine writer — une seule instance par WriteQueue.
func (q *WriteQueue) run() {
	defer close(q.done)
	for job := range q.ch {
		q.processJob(context.Background(), job)
	}
}

// processJob tente d'écrire le job avec backoff exponentiel.
func (q *WriteQueue) processJob(ctx context.Context, job writeJob) {
	for attempt := range writeMaxRetry {
		err := q.idxStore.PersistIndex(ctx, job.ref, job.entry)
		if err == nil {
			return
		}
		if !isLockError(err) || attempt == writeMaxRetry-1 {
			slog.Warn("assets: persist_index_dropped",
				append(job.ref.LogAttrs(), "attempt", attempt+1, "err", err)...)
			if q.metrics != nil {
				q.metrics.IncIndexWriteDropped(job.ref.Kind)
			}
			return
		}
		wait := backoffDuration(attempt)
		slog.Warn("assets: persist_index_retry",
			append(job.ref.LogAttrs(), "attempt", attempt+1, "wait_ms", wait.Milliseconds(), "err", err)...)
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

// backoffDuration calcule la durée de retry avec jitter exponentiel.
// attempt 0→50ms, 1→100ms, 2→200ms, 3→400ms, 4→800ms (capped à 2s).
func backoffDuration(attempt int) time.Duration {
	d := time.Duration(float64(writeBackoffMin) * math.Pow(2, float64(attempt)))
	if d > writeBackoffMax {
		d = writeBackoffMax
	}
	return d
}
