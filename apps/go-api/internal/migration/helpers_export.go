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

// LoadTableColumns retourne la liste ordonnée des colonnes d'une table. Exposé
// pour les rebuilds/append-only title-owned qui construisent un CTAS préservant
// les colonnes existantes (Phase 1.5 b13). Les 7 appelants in-package gardent la
// forme privée loadTableColumns.
func LoadTableColumns(ctx context.Context, db *sql.DB, tableName string) ([]string, error) {
	return loadTableColumns(ctx, db, tableName)
}

// FirstWords retourne les n premiers mots de s (séparés par des espaces). Exposé
// pour les migrations title-owned qui en ont besoin (purge/rebuilds, Phase 1.5
// b13). Les appelants in-package gardent la forme privée firstWords.
func FirstWords(s string, n int) string { return firstWords(s, n) }

// ApplyAppendOnlyWeaponKills applique la migration append-only weapon_kills
// (generation_id + written_at + vue v_weapon_kills dernière génération). La
// migration globale shared_append_only_weapon_kills_v1 délègue à elle. Exposé pour
// le test de cardinalité/idempotence relocalisé dans le package title-owned (la
// table weapon_kills est créée title-owned, donc le test vit là où le provider
// StepsFor est câblable). No-op si la table weapon_kills est absente.
func ApplyAppendOnlyWeaponKills(db *sql.DB) error { return applyAppendOnlyWeaponKills(db) }

// AppendOnlyRebuild décrit la conversion append-only d'une table d'état par swap
// CTAS TRANSACTIONNEL (cf. append_only_rebuild.go) : PK technique `id` + horloge,
// garde anti-perte rebuilt==before AVANT le DROP, recoverOrphan, idempotence.
// Alias exporté pour que les migrations title-owned (internal/games/{slug}/migrations)
// convertissent LEURS tables sans réimplémenter — ni dégrader — la mécanique de swap.
type AppendOnlyRebuild = appendOnlyRebuild

// SynthWrittenAt — expression de colonne synthétique written_at (mécanisme
// « dernière version par written_at »), commune aux conversions simples.
const SynthWrittenAt = synthWrittenAt

// ApplyAppendOnlyRebuild : point d'entrée idempotent d'une conversion append-only
// (recoverOrphan → no-op si table absente → no-op si déjà migrée → swap CTAS
// transactionnel + vue _latest). Exposé pour les migrations title-owned. Délègue 1:1.
func ApplyAppendOnlyRebuild(db *sql.DB, spec AppendOnlyRebuild) error {
	return applyAppendOnlyRebuild(db, spec)
}
