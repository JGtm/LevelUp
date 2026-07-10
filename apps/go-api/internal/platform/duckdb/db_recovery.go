// Package duckdb — db_recovery.go : classification des erreurs d invalidation/lock +
// reopen auto-repairant (Reopen / WithReopenOnInvalidated). Extrait de db.go (K3f god-file
// split, 2026-07-06), meme package.
package duckdb

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

// IsInvalidatedError détecte les erreurs qui marquent une connexion comme
// inutilisable et exigent un Reopen() pour récupérer. Trois classes :
//
//  1. FATAL DuckDB (ART corruption) :
//     "database has been invalidated because of a previous fatal error."
//     Cause racine : « Failed to delete all rows from index. Only deleted
//     N out of M rows. » sur un index ART avec valeurs NULL (duckdb#9277).
//
//  2. Handle fermée côté stdlib database/sql (corrigé 2026-05-25) :
//     "sql: database is closed" — retourné par toute opération sur un
//     *sql.DB après un Close(). Observé en prod 2026-05-25 11:20-11:23
//     en cascade massive (home + career + teammates + filters + explorer)
//     sur la player DB de JGtm. La DB physique reste saine, c'est juste
//     la handle Go périmée (close volontaire, swap RO↔RW, refcount cache).
//     Avant ce fix : WithReopenOnInvalidated retournait directement
//     l'erreur sans tenter Reopen → 500 cascade pendant plusieurs minutes.
//
//  3. Variantes driver-level "connection was closed".
//
// Exporté pour que les callers (repo, scheduler) puissent décider de :
//   - logger un incident métier (corruption d'index)
//   - tenter un Reopen() via WithReopenOnInvalidated
//   - propager au handler HTTP qui choisira la stratégie (503 + retry, etc.)
func IsInvalidatedError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "database has been invalidated") ||
		strings.Contains(s, "Failed to delete all rows from index") ||
		strings.Contains(s, "database must be restarted prior") ||
		strings.Contains(s, "sql: database is closed") ||
		strings.Contains(s, "connection was closed") ||
		strings.Contains(s, "database is closed")
}

// IsFileLockError détecte l'erreur DuckDB d'ouverture d'un fichier déjà
// verrouillé en écriture par un AUTRE process (mono-writer). Distinct de
// IsInvalidatedError (corruption/handle périmée) : ici la DB est saine, c'est
// une contention inter-process (CLI backfill concurrent, 2e instance serveur,
// hot-reload Air pas encore libéré). Permet aux callers d'émettre un message
// actionnable au lieu d'un "open rw" opaque (cf. spartan_cron Madina 2026-05-31).
func IsFileLockError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "Could not set lock on file") ||
		strings.Contains(s, "Conflicting lock is held") ||
		strings.Contains(s, "different configuration") ||
		// Windows/DuckDB : un autre détenteur (process distinct OU instance in-process
		// avec une config différente) tient déjà le fichier. DuckDB ajoute toujours ce
		// marqueur EN — "File is already open in <exe> (PID N)" — indépendamment de la
		// locale OS (le message Win FR "utilisé par un autre processus" varie, pas lui).
		// Sans ça, le boot Air laissant un tmp/server.exe résiduel produit un WARN
		// par-joueur dans spartan_cron au lieu de l'ERROR agrégée. Cf. 2026-06-27.
		strings.Contains(s, "File is already open in")
}

// Reopen ferme la connexion actuelle et en ouvre une nouvelle avec les
// mêmes paramètres (DSN, max conns, timezone). Permet de récupérer d'une
// invalidation fatale sans redémarrer le serveur.
//
// Thread-safe via le mutex du cache. Tout autre *DB existant qui pointait
// sur la MÊME instance partagée sera également mis à jour (le pointeur
// sqlDB est remplacé in-place et la cache entry pointe sur le même *DB).
//
// Limitations :
//   - Les requêtes/transactions en cours sur l'ancien sqlDB échoueront
//     (comportement attendu : l'invalidation a déjà cassé ces requêtes).
//   - Si le ping de la nouvelle connexion échoue (fichier corrompu côté
//     OS), retourne l'erreur sans toucher au sqlDB existant.
func (db *DB) Reopen() error {
	if db == nil {
		return errors.New("duckdb.Reopen: nil DB")
	}
	openDBsMu.Lock()
	defer openDBsMu.Unlock()

	// Construit le nouveau sqlDB avec la même config qu'à l'ouverture initiale.
	newSQLDB, err := openSQLDBFor(db.dsn, db.timezone, db.op, db.path)
	if err != nil {
		slog.Error("duckdb: Reopen a échoué (fichier inaccessible ?)",
			"path", db.path, "op", db.op, "err", err)
		return err
	}
	applyConnLimits(newSQLDB, db.maxOpenConns, db.maxIdleConns)

	// Ferme l'ancien sqlDB en best-effort (déjà invalidé côté DuckDB).
	if old := db.loadSQL(); old != nil {
		_ = old.Close()
	}
	db.sqlDB.Store(newSQLDB)
	db.closed.Store(false)

	// Restaure l'entrée cache pointant sur cette même *DB (refCount préservé
	// si déjà présent, sinon refCount=1).
	if db.cacheKey != "" {
		if cached, ok := openDBs[db.cacheKey]; ok {
			cached.db = db
		} else {
			openDBs[db.cacheKey] = &cachedDB{db: db, refCount: 1}
		}
	}

	slog.Info("duckdb: connexion ré-ouverte après invalidation",
		"path", db.path, "op", db.op)
	return nil
}

// WithReopenOnInvalidated exécute fn ; si fn renvoie une erreur
// d'invalidation détectée par IsInvalidatedError, fait un Reopen() et
// retry fn une fois. Sinon retourne l'erreur originale.
//
// Pattern à utiliser dans les repos qui font des opérations write sensibles
// (UPDATE/DELETE) sur des DB partagées process-level — ainsi un bug DuckDB
// transitoire n'invalide pas la DB pour toute la durée de vie du process.
//
// Le retry est borné à 1 : si la 2e tentative échoue aussi avec une
// invalidation, on remonte l'erreur (probablement une corruption durable
// du fichier qui nécessite intervention).
func (db *DB) WithReopenOnInvalidated(fn func() error) error {
	err := fn()
	if !IsInvalidatedError(err) {
		return err
	}
	slog.Warn("duckdb: connexion invalidée, tentative de reopen",
		"path", db.path, "err", err)
	if reopenErr := db.Reopen(); reopenErr != nil {
		return fmt.Errorf("reopen after invalidation: %w (original: %v)", reopenErr, err)
	}
	retryErr := fn()
	if retryErr != nil {
		// Retry échoué : incident à traiter par les ops (corruption persistante ?).
		slog.Error("duckdb: retry post-reopen a échoué",
			"path", db.path, "err", retryErr,
			"persistent_invalidation", IsInvalidatedError(retryErr))
	}
	return retryErr
}
