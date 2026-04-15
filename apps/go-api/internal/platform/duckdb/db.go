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
	"fmt"
	"sync"

	_ "github.com/duckdb/duckdb-go/v2" // driver duckdb
)

// DB encapsule une connexion DuckDB ouverte.
type DB struct {
	sqlDB *sql.DB
	path  string
}

var (
	openDBsMu sync.Mutex
	openDBs   = map[string]*DB{}
)

// OpenReadOnly ouvre une base DuckDB en lecture seule.
// Le DSN est : "file.duckdb?access_mode=read_only".
// Une seule instance par chemin est maintenue (cache process-level).
func OpenReadOnly(path string) (*DB, error) {
	openDBsMu.Lock()
	defer openDBsMu.Unlock()

	key := "ro:" + path
	if db, ok := openDBs[key]; ok {
		return db, nil
	}

	dsn := path + "?access_mode=read_only"
	sqlDB, err := sql.Open("duckdb", dsn)
	if err != nil {
		return nil, fmt.Errorf("duckdb.OpenReadOnly(%s): %w", path, err)
	}
	// Vérification active de la connexion
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("duckdb.OpenReadOnly ping(%s): %w", path, err)
	}
	// Pool conservateur : DuckDB ne supporte pas bien le parallélisme multi-writer.
	// En read-only, plusieurs connexions sont possibles mais on reste raisonnable.
	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(2)

	db := &DB{sqlDB: sqlDB, path: path}
	openDBs[key] = db
	return db, nil
}

// OpenReadWrite ouvre une base DuckDB en lecture-écriture.
// Utilisé pour les migrations au démarrage. UNE seule connexion : pas de pool.
func OpenReadWrite(path string) (*DB, error) {
	openDBsMu.Lock()
	defer openDBsMu.Unlock()

	key := "rw:" + path
	if db, ok := openDBs[key]; ok {
		return db, nil
	}

	sqlDB, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("duckdb.OpenReadWrite(%s): %w", path, err)
	}
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("duckdb.OpenReadWrite ping(%s): %w", path, err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	db := &DB{sqlDB: sqlDB, path: path}
	openDBs[key] = db
	return db, nil
}

// Close ferme la connexion DuckDB. À appeler au shutdown.
func (db *DB) Close() error {
	openDBsMu.Lock()
	defer openDBsMu.Unlock()
	delete(openDBs, "ro:"+db.path)
	delete(openDBs, "rw:"+db.path)
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
