// Package sync — lease.go : façade sur internal/platform/dblease.
//
// Ce fichier délègue entièrement au package dblease pour éviter
// la duplication de la map de mutex et les cycles d'import entre
// internal/sync et internal/platform/duckdb (PersistSink).
//
// Graphe d'import garanti sans cycle :
//
//	internal/sync → internal/platform/dblease
//	internal/platform/duckdb → internal/platform/dblease
//	internal/platform/dblease → (stdlib uniquement)
//
// Usage :
//
//	release, err := AcquireLeaseCtx(ctx, dbPath)
//	if err != nil { return err }
//	defer release()
//
// ─── Coordination avec les commits 2-6 db-concurrency ───
//
// Les fonctions AcquireLease / AcquireLeaseCtx ci-dessous partagent le même
// mutex sous-jacent (`dblease.leaseMutex(path)`) que les méthodes
// `pdb.AcquirePlayerWriterTimeout` / `pdb.AcquireSharedSocialWriterTimeout`
// utilisées par les handlers HTTP (Prestige, Notifications, Social, Media).
//
// Conséquence pratique : pendant qu'un sync engine tient son lease via
// AcquireLeaseCtx, un POST /challenges concurrent qui appelle
// AcquirePlayerWriterTimeout retournera ErrDBLocked et sera mappé en HTTP 503
// par le handler. Idem en sens inverse. **C'est le comportement attendu.**
//
// ⚠️ Invariant fragile : le sync engine appelle ensuite RunPostSyncHook qui
// invoque prestige.Service.EvaluateForUser. Cette dernière n'acquiert
// volontairement pas de lease (cf. commit 3 thought_log) — sinon deadlock
// puisque le sync engine tient déjà le mutex. Si un futur commit configure
// LazyPrestigeService pour qu'EvaluateForUser acquière un lease, il faudra
// soit (a) propager le writer du sync engine au hook (signature
// EvaluateForUserWithWriter), soit (b) déplacer le hook après le release du
// writer sync. Documenter dans le PR.
//
// ─── Dette : observabilité expvar ───
//
// Les compteurs `dblease_acquire_total{kind=player|shared_matches|...}`
// introduits au commit 1 ne sont **pas** alimentés par AcquireLeaseCtx
// ci-dessous (qui appelle l'API legacy `dblease.AcquireLeaseCtx`). Migration
// 1:1 vers `dblease.AcquireWriterCtx(ctx, nil, path, kind)` faisable mais
// touche 11 sites et risque de casser ~63 tests sync — différée en CI
// cgo-enabled. Cf. plan §commit 7 + §commit 8 lint analyzer.
package sync

import (
	"context"
	"time"

	"levelup/go-api/internal/platform/dblease"
)

// AcquireLease est une façade sur dblease.AcquireLease.
// Préférer AcquireLeaseCtx pour les flux de sync longs.
func AcquireLease(path string, timeout time.Duration) (func(), error) {
	return dblease.AcquireLease(path, timeout)
}

// AcquireLeaseCtx est une façade sur dblease.AcquireLeaseCtx.
// L'attente s'arrête si le contexte est annulé.
func AcquireLeaseCtx(ctx context.Context, path string) (func(), error) {
	return dblease.AcquireLeaseCtx(ctx, path)
}
