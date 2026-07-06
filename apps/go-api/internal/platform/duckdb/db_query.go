// Package duckdb — db_query.go : methodes DB de requete/execution (Query/Exec/*Recovered +
// logDBError/queryExcerpt/Close/SQLDb/Path). Extrait de db.go (K3f god-file split,
// 2026-07-06), meme package.
package duckdb

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"
)

// Close ferme la connexion DuckDB. À appeler au shutdown.
func (db *DB) Close() error {
	if db == nil || db.loadSQL() == nil {
		return nil
	}

	openDBsMu.Lock()
	defer openDBsMu.Unlock()

	if db.closed.Load() {
		return nil
	}
	if db.cacheKey != "" {
		if cached, ok := openDBs[db.cacheKey]; ok {
			if cached.refCount > 1 {
				cached.refCount--
				return nil
			}
			delete(openDBs, db.cacheKey)
		}
	}
	db.closed.Store(true)
	return db.loadSQL().Close()
}

// QueryRow exécute une requête qui retourne exactement une ligne.
// L'erreur réelle de la requête est différée jusqu'à Scan ; on ne peut donc pas
// la logger ici. Les call sites critiques doivent capturer err sur Scan eux-mêmes.
func (db *DB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return db.loadSQL().QueryRowContext(ctx, query, args...)
}

// Query exécute une requête qui retourne plusieurs lignes.
func (db *DB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := db.loadSQL().QueryContext(ctx, query, args...)
	if err != nil {
		logDBError(ctx, "duckdb: query failed", db, query, err)
	}
	return rows, err
}

// Exec exécute une instruction sans valeur de retour.
func (db *DB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	res, err := db.loadSQL().ExecContext(ctx, query, args...)
	if err != nil {
		logDBError(ctx, "duckdb: exec failed", db, query, err)
	}
	return res, err
}

// ExecRecovered exécute un Exec sous WithReopenOnInvalidated : si la connexion
// est invalidée par un bug DuckDB transitoire, fait un Reopen + retry une fois.
//
// À utiliser par défaut dans les repos qui écrivent sur des bases partagées
// process-level (shared_social.duckdb, notamment) — un seul incident de
// connexion ne doit pas casser l'API jusqu'au prochain restart.
func (db *DB) ExecRecovered(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	var res sql.Result
	err := db.WithReopenOnInvalidated(func() error {
		var execErr error
		res, execErr = db.loadSQL().ExecContext(ctx, query, args...)
		return execErr
	})
	if err != nil {
		logDBError(ctx, "duckdb: exec failed (after recovery)", db, query, err)
	}
	return res, err
}

// QueryRecovered exécute une Query sous WithReopenOnInvalidated. Cf. ExecRecovered.
//
// ATTENTION : si l'invalidation se produit pendant l'itération des rows (donc
// après QueryRecovered a retourné), le wrapper ne peut plus reagir. Seule la
// requête initiale est protégée. Les rows.Next()/rows.Scan() restent
// vulnérables et doivent être audités dans les chemins critiques (rare).
func (db *DB) QueryRecovered(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	var rows *sql.Rows
	err := db.WithReopenOnInvalidated(func() error {
		var qerr error
		rows, qerr = db.loadSQL().QueryContext(ctx, query, args...)
		return qerr
	})
	if err != nil {
		logDBError(ctx, "duckdb: query failed (after recovery)", db, query, err)
	}
	return rows, err
}

// logDBError centralise le log d'erreur DuckDB avec contexte standard
// (path, op, query excerpt, err). Niveau Error — toute erreur DB est anormale
// par défaut ; les call sites qui tolèrent une erreur (ex. : fichier absent
// pour Stat) doivent éviter de passer par db.Query/Exec, ou wrapper l'err.
func logDBError(ctx context.Context, msg string, db *DB, query string, err error) {
	slog.ErrorContext(ctx, msg,
		"path", db.path,
		"op", db.op,
		"query_excerpt", queryExcerpt(query),
		"err", err,
	)
}

// queryExcerpt retourne les ~80 premiers caractères de la query, sur une seule
// ligne (newlines → espaces, espaces consécutifs collapsés). Évite de polluer
// la console avec des SQL multi-lignes tout en gardant assez d'info pour
// identifier la requête fautive.
func queryExcerpt(q string) string {
	const maxLen = 80
	// Collapse whitespace (newlines, tabs, multiples spaces) en un seul espace.
	collapsed := strings.Join(strings.Fields(q), " ")
	if len(collapsed) <= maxLen {
		return collapsed
	}
	// Tronquage avec ellipsis. Comptage byte = ok pour ASCII SQL ; runes pour
	// les cas avec accents dans les noms de tables/colonnes (rare).
	if cnt := 0; true {
		for i := range collapsed {
			if cnt == maxLen {
				return collapsed[:i] + "…"
			}
			cnt++
		}
	}
	return collapsed
}

// SQLDb retourne le *sql.DB sous-jacent (pour interop avec d'autres packages).
func (db *DB) SQLDb() *sql.DB { return db.loadSQL() }

// Path retourne le chemin de la base.
func (db *DB) Path() string { return db.path }
