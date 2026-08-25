// Package duckdb — read_recovery.go : lectures BEST-EFFORT sur une DB partagée,
// résilientes à l'invalidation CONCURRENTE du handle (RecoveringReader).
//
// # Le trou que ce fichier ferme
//
// `OpenReadForQuery(path)` retourne un *INSTANTANÉ* : le `*sql.DB` que le
// wrapper `*DB` porte AU MOMENT de l'appel. Quand l'appel a emprunté le handle
// du cache process (`LookupCachedDB`), l'emprunt est NON POSSÉDANT — aucun
// incrément de refCount — et le release rendu est un no-op. Le propriétaire peut
// donc fermer le `*sql.DB` sous les pieds de l'emprunteur :
//
//   - B-swap RO→RW (ADR 0016) : `PlayerDB.PrepareForSharedSwap` ferme `pdb.Shared`
//     pour que le Provider puisse ouvrir le fichier en RW ;
//   - fin d'un autre `OpenReadForQuery` PROPRIÉTAIRE (celui qui a eu le cache
//     miss et détient donc refCount=1) : son release supprime l'entrée du cache
//     et ferme le `*sql.DB` — que d'autres lecteurs concurrents ont pourtant
//     emprunté entre-temps ;
//   - `EvictAndCloseCached` (purge délibérée d'une DB).
//
// L'emprunteur voit alors « sql: database is closed » PENDANT sa requête ou son
// itération de rows. Les deux familles observées en prod (2026-08-25) sont
// exactement ce cas :
//   - `ops.IndexMedia` → `loadMatchTimeWindows` sur shared_matches_v2 (~2,7 ERROR/j) ;
//   - `sync.buildCitationContext` → `loadPveStats` sur shared_pve (372 WARN/mois) :
//     le pipeline post-sync tourne en parallèle SANS limite par joueur
//     (`RunPostSync`, PostSyncParallelism = 0), chaque joueur acquiert son handle
//     shared_pve ; le premier à finir ferme le fichier pour tous les autres.
//
// # Pourquoi PAS « rendre l'emprunt possédant »
//
// Faire incrémenter le refCount par `OpenReadForQuery` supprimerait la classe
// entière… en cassant le B-swap : `pdb.Shared.Close()` deviendrait un simple
// décrément, le fichier resterait tenu en RO, et l'`OpenReadWrite` du Provider
// échouerait ("different configuration"). Le modèle assumé est « le swap gagne,
// le lecteur se répare » — d'où la reprise ci-dessous plutôt qu'une possession.
//
// # Le contrat
//
// `RecoveringReader` garde le CHEMIN (pas seulement l'instantané) : si `fn`
// échoue sur une erreur d'invalidation (`IsInvalidatedError`), le handle est
// RE-RÉSOLU via le cache — qui rend forcément un `*sql.DB` vivant, l'entrée
// morte ayant été supprimée par le `Close()` du propriétaire — et `fn` est
// rejouée UNE seule fois. Récupération loguée en WARN (même registre que
// `WithReopenOnInvalidated`, db_recovery.go) ; échec du retry logué en ERROR.
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
)

// RecoveringReader porte un handle de LECTURE sur une DB DuckDB partagée et le
// ré-ouvre quand il est invalidé sous les pieds du lecteur (cf. en-tête).
//
// Usage : acquérir UNE fois (l'ouverture DuckDB est coûteuse — ne pas en faire
// une par ligne lue), puis passer chaque requête par Do. Le zéro-value n'est pas
// utilisable : passer par OpenRecoveringReader. Un `*RecoveringReader` nil est
// accepté par Close (no-op) pour que les callers en dégradation gracieuse
// (fichier absent) puissent différer Close sans test préalable.
//
// Sûr en usage concurrent : plusieurs goroutines peuvent partager un reader ;
// une seule ré-ouverture a lieu par invalidation (compteur de génération).
type RecoveringReader struct {
	path     string
	timezone []string

	mu      sync.Mutex
	db      *sql.DB
	release func()
	gen     uint64
	closed  bool
}

// ErrReaderClosed est renvoyée par Do sur un reader nil ou déjà fermé.
var ErrReaderClosed = errors.New("duckdb: RecoveringReader fermé ou nil")

// OpenRecoveringReader acquiert un handle de lecture sur path (via
// OpenReadForQuery : réutilise le handle process RW/RO s'il existe, ouvre en RO
// sinon) et l'enveloppe dans un reader auto-réparant. Le caller DOIT différer
// Close (qui rend l'emprunt / ferme le handle possédé, exactement comme le
// release d'OpenReadForQuery).
func OpenRecoveringReader(path string, timezone ...string) (*RecoveringReader, error) {
	r := &RecoveringReader{path: path, timezone: timezone}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, _, err := r.openLocked(); err != nil {
		return nil, err
	}
	return r, nil
}

// Path retourne le chemin de la base lue (diagnostic / logs des callers).
func (r *RecoveringReader) Path() string {
	if r == nil {
		return ""
	}
	return r.path
}

// Do exécute fn avec le handle de lecture courant. Si fn échoue sur une
// invalidation (IsInvalidatedError : « sql: database is closed », handle fermée
// par un B-swap ou par le release d'un emprunt concurrent), le handle est
// ré-ouvert et fn est rejouée UNE fois — jamais plus (un second échec signale
// une invalidation durable, à traiter par les ops).
//
// Contrat de fn : elle doit être REJOUABLE — toute accumulation (slice, map)
// doit être réinitialisée en entrée de fn, sinon le retry duplique les lignes.
// fn ne doit PAS conserver le *sql.DB au-delà de son retour.
//
// Les erreurs non-invalidation (y compris sql.ErrNoRows) sont retournées telles
// quelles, sans retry ni enveloppe — `errors.Is` reste utilisable côté caller.
func (r *RecoveringReader) Do(ctx context.Context, fn func(db *sql.DB) error) error {
	if r == nil {
		return ErrReaderClosed
	}
	db, gen, err := r.current()
	if err != nil {
		return err
	}
	if err = fn(db); !IsInvalidatedError(err) {
		return err
	}
	slog.WarnContext(ctx, "duckdb: handle de lecture invalidée pendant la requête — ré-ouverture et retry unique",
		"path", r.path, "err", err)

	fresh, refreshErr := r.refresh(gen)
	if refreshErr != nil {
		return fmt.Errorf("ré-ouverture lecture %s: %w (invalidation initiale: %v)", r.path, refreshErr, err)
	}
	retryErr := fn(fresh)
	if retryErr != nil {
		slog.ErrorContext(ctx, "duckdb: retry de lecture après ré-ouverture a échoué",
			"path", r.path, "err", retryErr,
			"persistent_invalidation", IsInvalidatedError(retryErr))
	}
	return retryErr
}

// Close rend le handle (no-op sur un emprunt du cache, fermeture sur un handle
// possédé). Idempotent, et sûr sur un reader nil.
func (r *RecoveringReader) Close() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.releaseLocked()
	r.closed = true
}

// current retourne le handle courant et sa génération, en le ré-ouvrant si une
// ré-ouverture précédente a échoué et laissé le reader vide.
func (r *RecoveringReader) current() (*sql.DB, uint64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, 0, ErrReaderClosed
	}
	if r.db != nil {
		return r.db, r.gen, nil
	}
	return r.openLocked()
}

// refresh rend le handle mort et en acquiert un frais. seenGen est la génération
// observée par le Do appelant : si elle a déjà changé, une autre goroutine a
// ré-ouvert entre-temps — on réutilise son handle au lieu d'en ouvrir un 2e.
func (r *RecoveringReader) refresh(seenGen uint64) (*sql.DB, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, ErrReaderClosed
	}
	if r.gen != seenGen && r.db != nil {
		return r.db, nil
	}
	// Rendre AVANT de ré-acquérir : si ce reader était le PROPRIÉTAIRE de
	// l'entrée de cache, son release la supprime — sans quoi OpenReadForQuery
	// re-emprunterait le même instantané mort.
	r.releaseLocked()
	db, _, err := r.openLocked()
	return db, err
}

// openLocked acquiert un handle frais. À appeler sous r.mu.
func (r *RecoveringReader) openLocked() (*sql.DB, uint64, error) {
	db, release, err := OpenReadForQuery(r.path, r.timezone...)
	if err != nil {
		return nil, r.gen, err
	}
	r.db, r.release, r.gen = db, release, r.gen+1
	return db, r.gen, nil
}

// releaseLocked rend le handle courant s'il y en a un. À appeler sous r.mu.
func (r *RecoveringReader) releaseLocked() {
	if r.release != nil {
		r.release()
	}
	r.db, r.release = nil, nil
}
