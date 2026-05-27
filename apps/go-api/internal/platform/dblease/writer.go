package dblease

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ErrDBLocked est retourné par AcquireWriter / AcquireWriterCtx quand le lease
// n'a pas pu être acquis dans le délai imparti (timeout) ou avant l'annulation
// du contexte. Les handlers HTTP doivent mapper cette erreur en HTTP 503 avec
// le header Retry-After pour permettre un retry côté client.
//
// Usage attendu :
//
//	w, err := pdb.AcquirePlayerWriter(ctx)
//	if errors.Is(err, dblease.ErrDBLocked) {
//	    http.Error(rw, "DB busy", http.StatusServiceUnavailable)
//	    rw.Header().Set("Retry-After", "5")
//	    return
//	}
var ErrDBLocked = errors.New("dblease: database write lease unavailable")

// LeasedWriter représente un droit exclusif d'écriture sur une DB DuckDB
// donnée. Construit uniquement via AcquireWriter ou AcquireWriterCtx — c'est
// la garantie compile-time du refactor : un repo qui prend un *LeasedWriter
// ne peut être appelé que par un caller qui a explicitement acquis le lease.
//
// Le type implémente port.DBWriter et port.DBExecutor (cf. compile-time
// checks dans writer_test.go).
//
// release() est idempotent (sync.Once) — un double appel est sûr.
type LeasedWriter struct {
	db          *sql.DB
	path        string
	kind        Kind
	mu          *sync.Mutex // mutex partagé du package, déjà tenu à la construction
	acquiredAt  time.Time
	releaseOnce sync.Once
}

// Path retourne le chemin de la DB ciblée. Utile pour logging structuré et
// debug (jamais pour relâcher manuellement le lease).
func (w *LeasedWriter) Path() string { return w.path }

// Kind retourne le Kind de la DB ciblée.
func (w *LeasedWriter) Kind() Kind { return w.kind }

// ExecContext implémente port.DBExecutor.
func (w *LeasedWriter) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return w.db.ExecContext(ctx, query, args...)
}

// QueryContext implémente port.DBExecutor.
func (w *LeasedWriter) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return w.db.QueryContext(ctx, query, args...)
}

// QueryRowContext implémente port.DBExecutor.
func (w *LeasedWriter) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return w.db.QueryRowContext(ctx, query, args...)
}

// BeginTx implémente port.DBWriter — ouvre une transaction sur la DB sous-jacente.
// Le *sql.Tx retourné satisfait port.DBExecutor mais pas port.DBWriter (pas de
// sous-transaction possible avec database/sql).
func (w *LeasedWriter) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return w.db.BeginTx(ctx, opts)
}

// CommitWithCheckpoint commit la transaction puis exécute un CHECKPOINT
// immédiat sur la DB sous-jacente (ADR 0021 Phase 3.2 bis).
//
// À utiliser pour les opérations critiques sur shared_social (likes, favoris,
// associations média) où la fenêtre d'exposition au bug WAL orphelin doit
// être minimisée. Le CHECKPOINT est exécuté APRÈS Commit pour ne pas bloquer
// la TX elle-même.
//
// Différences vs `tx.Commit()` simple :
//   - garanti d'un CHECKPOINT immédiat (pas d'attente du scheduler 5min)
//   - erreur CHECKPOINT loguée mais NON propagée (les données sont déjà
//     commit, le CHECKPOINT peut retry au prochain tick scheduler)
//
// Le caller doit appeler Release() ou laisser le defer s'en charger : cette
// méthode NE LIBÈRE PAS le lease (séparation des préoccupations).
func (w *LeasedWriter) CommitWithCheckpoint(ctx context.Context, tx *sql.Tx) error {
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("dblease: tx.Commit: %w", err)
	}
	if _, err := w.db.ExecContext(ctx, "CHECKPOINT"); err != nil {
		// Non-fatal : commit déjà effectué, data persisted dans le WAL.
		// Le CHECKPOINT scheduler 5min fera fallback.
		slog.WarnContext(ctx, "dblease: CHECKPOINT post-commit échoué (non-fatal — scheduler 5min fallback)",
			"kind", string(w.kind), "path", w.path, "err", err)
	}
	return nil
}

// Release relâche le lease. Idempotent : un second appel est no-op (pas de
// panic, pas de double-decrement de métrique). Doit être appelé via defer
// immédiatement après acquisition réussie pour garantir le release sur panic
// ou ctx cancel.
func (w *LeasedWriter) Release() {
	w.releaseOnce.Do(func() {
		heldMs := time.Since(w.acquiredAt).Milliseconds()
		w.mu.Unlock()
		recordRelease(w.kind)
		slog.Debug("dblease released",
			"kind", string(w.kind),
			"path", w.path,
			"held_ms", heldMs,
		)
	})
}

// AcquireWriter tente d'acquérir un writer exclusif sur la DB ciblée par path.
// Bloque au plus `timeout` puis retourne ErrDBLocked. Préférer
// AcquireWriterCtx pour les flux pilotés par contexte.
//
// L'appelant DOIT capturer le résultat et appeler Release() (typiquement via
// defer w.Release()) — sinon le lease reste tenu indéfiniment.
func AcquireWriter(db *sql.DB, path string, kind Kind, timeout time.Duration) (*LeasedWriter, error) {
	mu := leaseMutex(path)
	deadline := time.Now().Add(timeout)
	start := time.Now()

	for {
		if mu.TryLock() {
			waitMs := time.Since(start).Milliseconds()
			recordAcquire(kind, waitMs)
			slog.Debug("dblease acquired",
				"kind", string(kind),
				"path", path,
				"wait_ms", waitMs,
			)
			return &LeasedWriter{
				db: db, path: path, kind: kind, mu: mu, acquiredAt: time.Now(),
			}, nil
		}
		if time.Now().After(deadline) {
			recordTimeout(kind)
			slog.Warn("dblease timeout",
				"kind", string(kind),
				"path", path,
				"timeout", timeout,
			)
			return nil, fmt.Errorf("dblease: %s lease timeout after %v on %s: %w",
				kind, timeout, path, ErrDBLocked)
		}
		time.Sleep(pollInterval)
	}
}

// AcquireWriterCtx acquiert un writer sur la DB ciblée. Bloque jusqu'à
// l'acquisition ou l'annulation du contexte. Adapté aux flux longs (sync
// engine) où le contexte porte le shutdown / cancellation.
//
// Si le contexte est annulé avant l'acquisition, retourne une erreur qui wrap
// ctx.Err() ET ErrDBLocked (les deux peuvent être matchés via errors.Is).
func AcquireWriterCtx(ctx context.Context, db *sql.DB, path string, kind Kind) (*LeasedWriter, error) {
	mu := leaseMutex(path)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	start := time.Now()

	for {
		if mu.TryLock() {
			waitMs := time.Since(start).Milliseconds()
			recordAcquire(kind, waitMs)
			slog.DebugContext(ctx, "dblease acquired",
				"kind", string(kind),
				"path", path,
				"wait_ms", waitMs,
			)
			return &LeasedWriter{
				db: db, path: path, kind: kind, mu: mu, acquiredAt: time.Now(),
			}, nil
		}
		select {
		case <-ctx.Done():
			recordTimeout(kind)
			slog.WarnContext(ctx, "dblease ctx cancelled",
				"kind", string(kind),
				"path", path,
				"err", ctx.Err(),
			)
			// errors.Join wrappe les DEUX sentinels — un caller peut faire
			// errors.Is(err, ErrDBLocked) ou errors.Is(err, context.Canceled).
			return nil, fmt.Errorf("dblease: %s lease cancelled on %s: %w",
				kind, path, errors.Join(ErrDBLocked, ctx.Err()))
		case <-ticker.C:
			// retry
		}
	}
}
