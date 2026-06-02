package duckdb

import "database/sql"

// newTestDB construit un *DB autour d'un *sql.DB déjà ouvert (tests :memory:).
// Le champ sqlDB étant atomique (atomic.Pointer, fix data race revue P0 2026-06-02),
// les tests ne peuvent plus l'initialiser via un littéral de struct — passer par ce
// helper qui fait le Store.
func newTestDB(sqlDB *sql.DB, path string) *DB {
	db := &DB{path: path}
	db.sqlDB.Store(sqlDB)
	return db
}
