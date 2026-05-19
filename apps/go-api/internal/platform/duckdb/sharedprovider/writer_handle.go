package sharedprovider

import (
	"database/sql"
	"log/slog"
	"sync"

	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/dblease"
)

// WriterHandle représente un droit exclusif d'écriture sur shared_matches_v2
// pour un provider donné. Construit uniquement via Provider.AcquireWriter.
//
// L'API expose un *sql.DB direct (pas un wrapper) pour permettre l'usage
// natif des transactions, batches, et helpers `database/sql` standard.
//
// Release() DOIT être appelé via defer immédiatement après une acquisition
// réussie. Idempotent.
type WriterHandle struct {
	provider *providerImpl
	rwHandle *duckdbpkg.DB
	lease    *dblease.LeasedWriter

	releaseOnce sync.Once
}

// DB retourne la connexion RW sous-jacente. Reste valide jusqu'au Release().
//
// Le caller peut utiliser librement Exec/Query/BeginTx etc. — c'est un
// *sql.DB standard du package database/sql.
func (w *WriterHandle) DB() *sql.DB {
	return w.rwHandle.SQLDb()
}

// Release relâche le writer :
//  1. Ferme la conn RW.
//  2. Rouvre la conn RO (ou passe en StateError + retry async si échec).
//  3. Libère le lease dblease (autorise le prochain writer à acquérir).
//  4. Débloque les Get HTTP en attente.
//
// Recovery panic : si une panic survient pendant le release (ex: bug DuckDB
// fermeture), le lease est tout de même libéré pour ne pas bloquer
// indéfiniment le prochain writer. Le compteur shared_provider_swap_failures_total{panic}
// est incrémenté.
//
// Idempotent (sync.Once) — un second appel est no-op.
func (w *WriterHandle) Release() {
	w.releaseOnce.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				swapFailuresTotal.Add(failReasonPanic, 1)
				slog.Error("sharedprovider: panic during Release",
					"panic", r, "path", w.provider.path)
			}
			// IMPORTANT : libérer le mutex dblease même en cas de panic,
			// sinon plus aucun writer ne pourra acquérir.
			w.lease.Release()
		}()

		w.provider.releaseWriter(w.rwHandle)
	})
}
