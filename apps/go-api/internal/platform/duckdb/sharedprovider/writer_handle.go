package sharedprovider

import (
	"database/sql"
	"sync"
)

// WriterHandle représente un droit exclusif d'écriture sur shared_matches_v2
// pour un Provider donné. Construit uniquement via Provider.AcquireWriter.
//
// Refacto commit 8b : la struct ne référence plus un *providerImpl ; toute
// la logique de release est capturée dans la closure releaseFn. Cela permet
// à différentes implémentations de Provider (providerImpl, inMemoryProvider…)
// de construire des WriterHandle avec leur propre stratégie de release —
// sans cycle d'import ni interface intermédiaire.
//
// L'API publique (DB(), Release()) est inchangée. Les callers existants
// continuent de fonctionner sans modification.
type WriterHandle struct {
	db          *sql.DB
	releaseFn   func()
	releaseOnce sync.Once
}

// DB retourne la connexion RW sous-jacente. Reste valide jusqu'au Release().
//
// Le caller peut utiliser librement Exec/Query/BeginTx etc. — c'est un
// *sql.DB standard du package database/sql.
func (w *WriterHandle) DB() *sql.DB {
	return w.db
}

// Release relâche le writer en invoquant la closure stockée par le Provider
// qui a créé le WriterHandle. Idempotent (sync.Once) — un second appel est
// no-op.
//
// La closure de release contient typiquement :
//   - Pour providerImpl : recover() + close RW + reopen RO + lease.Release()
//   - Pour inMemoryProvider : no-op (le db est partagé en mémoire, aucun swap)
func (w *WriterHandle) Release() {
	w.releaseOnce.Do(w.releaseFn)
}
