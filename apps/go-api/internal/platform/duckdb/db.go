// Package duckdb fournit l'accès aux bases DuckDB de LevelUp.
// CGO requis : compilé avec github.com/duckdb/duckdb-go/v2.
//
// Stratégie de connexion (Sprint 0) :
//   - Une connexion par base, avec sql.DB (pool natif Go).
//   - Les bases read-only utilisent "?access_mode=read_only".
//   - ATTACH est exécuté via une connexion dédiée (sql.Conn pinée).
//   - Les types critiques : UBIGINT→uint64, TIMESTAMP WITH TIME ZONE→time.Time,
//     VARCHAR→string, BOOLEAN→bool.
package duckdb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log/slog"
	"sync"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

// DB encapsule une connexion DuckDB ouverte.
type DB struct {
	sqlDB    *sql.DB
	path     string
	cacheKey string
	closed   bool
}

type cachedDB struct {
	db       *DB
	refCount int
}

var (
	openDBsMu sync.Mutex
	openDBs   = map[string]*cachedDB{}
)

// sanitizeTimezone valide un nom de timezone IANA pour éviter les injections SQL.
// Retourne "" si la valeur contient des caractères non autorisés.
func sanitizeTimezone(tz string) string {
	for _, c := range tz {
		switch {
		case c >= 'A' && c <= 'Z':
		case c >= 'a' && c <= 'z':
		case c >= '0' && c <= '9':
		case c == '/' || c == '_' || c == '-' || c == '+':
		default:
			return ""
		}
	}
	return tz
}

// OpenReadOnly ouvre une base DuckDB en lecture seule.
// Le DSN est : "file.duckdb?access_mode=read_only".
// Une seule instance par chemin est maintenue (cache process-level).
// timezone (optionnel) : nom IANA (ex: "Europe/Paris") — appliqué via SET TimeZone sur chaque connexion.
func OpenReadOnly(path string, timezone ...string) (*DB, error) {
	tz := ""
	if len(timezone) > 0 {
		raw := timezone[0]
		tz = sanitizeTimezone(raw)
		if tz == "" && raw != "" {
			slog.Warn("duckdb: timezone invalide ignorée", "input", raw, "path", path)
		}
	}
	return openCachedDB(
		"ro:"+path,
		path,
		path+"?access_mode=read_only",
		4,
		2,
		"OpenReadOnly",
		tz,
	)
}

// OpenReadWriteShared ouvre une base DuckDB en lecture-écriture avec un pool de connexions
// (4 conns max, comme OpenReadOnly). À utiliser pour les bases partagées (ex: metadata.duckdb)
// qui reçoivent des lectures concurrentes ET des écritures occasionnelles.
// Partage la même clé de cache que OpenReadWrite : si l'un est déjà ouvert, l'autre le réutilise.
func OpenReadWriteShared(path string, timezone ...string) (*DB, error) {
	tz := ""
	if len(timezone) > 0 {
		raw := timezone[0]
		tz = sanitizeTimezone(raw)
		if tz == "" && raw != "" {
			slog.Warn("duckdb: timezone invalide ignorée", "input", raw, "path", path)
		}
	}
	return openCachedDB("rw:"+path, path, path, 4, 2, "OpenReadWriteShared", tz)
}

// OpenReadWrite ouvre une base DuckDB en lecture-écriture.
// Utilisé pour les migrations au démarrage. UNE seule connexion : pas de pool.
// timezone (optionnel) : nom IANA (ex: "Europe/Paris") — appliqué via SET TimeZone sur chaque connexion.
func OpenReadWrite(path string, timezone ...string) (*DB, error) {
	tz := ""
	if len(timezone) > 0 {
		raw := timezone[0]
		tz = sanitizeTimezone(raw)
		if tz == "" && raw != "" {
			slog.Warn("duckdb: timezone invalide ignorée", "input", raw, "path", path)
		}
	}
	return openCachedDB("rw:"+path, path, path, 1, 1, "OpenReadWrite", tz)
}

func openCachedDB(
	key, path, dsn string,
	maxOpenConns, maxIdleConns int,
	op string,
	timezone string,
) (*DB, error) {
	openDBsMu.Lock()
	defer openDBsMu.Unlock()

	if cached, ok := openDBs[key]; ok {
		if err := cached.db.sqlDB.Ping(); err == nil {
			cached.refCount++
			return cached.db, nil
		}
		_ = cached.db.sqlDB.Close()
		delete(openDBs, key)
	}

	var sqlDB *sql.DB
	if timezone != "" {
		tz := timezone
		connector, err := duckdb.NewConnector(dsn, func(execer driver.ExecerContext) error {
			_, initErr := execer.ExecContext(context.Background(), "SET TimeZone='"+tz+"'", nil)
			return initErr
		})
		if err != nil {
			return nil, fmt.Errorf("duckdb.%s connector(%s): %w", op, path, err)
		}
		slog.Debug("duckdb: timezone appliquée", "timezone", timezone, "path", path)
		sqlDB = sql.OpenDB(connector)
	} else {
		var err error
		sqlDB, err = sql.Open("duckdb", dsn)
		if err != nil {
			return nil, fmt.Errorf("duckdb.%s(%s): %w", op, path, err)
		}
	}
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("duckdb.%s ping(%s): %w", op, path, err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	if maxOpenConns > 1 {
		sqlDB.SetMaxOpenConns(maxOpenConns)
	}
	if maxIdleConns > 1 {
		sqlDB.SetMaxIdleConns(maxIdleConns)
	}

	db := &DB{sqlDB: sqlDB, path: path, cacheKey: key}
	openDBs[key] = &cachedDB{db: db, refCount: 1}
	return db, nil
}

// Close ferme la connexion DuckDB. À appeler au shutdown.
func (db *DB) Close() error {
	if db == nil || db.sqlDB == nil {
		return nil
	}

	openDBsMu.Lock()
	defer openDBsMu.Unlock()

	if db.closed {
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
	db.closed = true
	return db.sqlDB.Close()
}

// QueryRow exécute une requête qui retourne exactement une ligne.
func (db *DB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return db.sqlDB.QueryRowContext(ctx, query, args...)
}

// Query exécute une requête qui retourne plusieurs lignes.
func (db *DB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return db.sqlDB.QueryContext(ctx, query, args...)
}

// Exec exécute une instruction sans valeur de retour.
func (db *DB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return db.sqlDB.ExecContext(ctx, query, args...)
}

// SQLDb retourne le *sql.DB sous-jacent (pour interop avec d'autres packages).
func (db *DB) SQLDb() *sql.DB { return db.sqlDB }

// Path retourne le chemin de la base.
func (db *DB) Path() string { return db.path }
