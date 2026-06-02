package migration

import (
	"context"
	"database/sql"
)

// BootCtx retourne le contexte racine non-cancellable utilisé par les migrations
// DDL boot-time (cf. doc de bootCtx). Exposé pour les backfills title-owned qui
// font db.ExecContext.
func BootCtx() context.Context { return bootCtx() }

// helpers_export.go — API publique des helpers DDL idempotents (Phase 1.5
// title-agnostic, ADR 0025). Ces wrappers exposent les helpers internes pour
// que les migrations *appartenant à un titre* (futur
// internal/games/{slug}/migrations/...) puissent les utiliser sans dupliquer la
// logique ni vivre dans le package migration. Le package migration reste
// l'infrastructure (registry, runner, ordre) ; les DDL Halo-specific en
// sortiront en s'appuyant sur ces helpers.
//
// Sémantique inchangée : ces wrappers délèguent 1:1 aux helpers internes.

// TableExists indique si une table (non-temp) existe.
func TableExists(db *sql.DB, table string) (bool, error) { return tableExists(db, table) }

// ColumnExists indique si une colonne existe dans une table.
func ColumnExists(db *sql.DB, table, column string) (bool, error) {
	return columnExists(db, table, column)
}

// HasPrimaryKey indique si une table a une clé primaire déclarée.
func HasPrimaryKey(db *sql.DB, table string) (bool, error) { return hasPrimaryKey(db, table) }

// AddColumnIfMissing ajoute une colonne si absente (ADD COLUMN idempotent).
func AddColumnIfMissing(db *sql.DB, table, column, colType string) error {
	return addColumnIfMissing(db, table, column, colType)
}

// CreateIndexSafe exécute un CREATE INDEX en tolérant l'existence préalable.
func CreateIndexSafe(db *sql.DB, ddl string) error { return createIndexSafe(db, ddl) }

// ExecScript exécute un script multi-statements (split sur `;`).
func ExecScript(db *sql.DB, script string) error { return execScript(db, script) }

// SplitSQL découpe un script en statements individuels.
func SplitSQL(script string) []string { return splitSQL(script) }
