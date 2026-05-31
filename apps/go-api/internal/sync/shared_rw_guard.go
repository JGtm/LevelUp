// Package sync — shared_rw_guard.go : garde-fou fail-fast contre l'écriture sur
// une connexion shared_matches_v2 ouverte en read-only.
//
// Contexte (incident mai 2026, cf. .ai/HANDOFF_sync_combat_completion.md) : la
// complétion post-sync (events / killer_victim / skill) écrivait sur un handle
// shared qui pouvait être read-only (suite à une corruption ART → cascade FATAL
// `database invalidated`). Chaque INSERT échouait, mais l'échec était avalé et
// compté `no_film` → 31h de panne silencieuse. Ce garde-fou rend la condition
// VISIBLE et IMMÉDIATE : au lieu de boucler sur N matchs en tentant N écritures
// vouées à l'échec, le caller détecte l'état read-only une fois et remonte une
// erreur nommée (ErrSharedReadOnly).
package sync

import (
	"context"
	"database/sql"
	"errors"
)

// ErrSharedReadOnly signale qu'une base utilisateur (shared_matches_v2 typiquement)
// vue par la connexion est ouverte en read-only — toute écriture échouerait.
var ErrSharedReadOnly = errors.New("shared DB ouverte en read-only — écriture impossible (handle non-RW)")

// assertSharedWritable vérifie qu'aucune base utilisateur (hors `system`/`temp`,
// les catalogues internes DuckDB) vue par cette connexion n'est en read-only.
// Robuste au nom de fichier : on ne filtre PAS sur "shared_matches_v2" (qui
// dépend du basename), on inspecte toute base attachée non-interne.
//
// Retourne :
//   - ErrSharedReadOnly si au moins une base utilisateur est read-only ;
//   - nil si tout est writable OU si le probe lui-même échoue (best-effort :
//     un environnement de test minimal sans `duckdb_databases()` ne doit pas
//     bloquer le heal — l'écriture réelle tranchera et sera comptée `failed`).
func assertSharedWritable(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return nil
	}
	var anyReadOnly bool
	err := db.QueryRowContext(ctx, `
		SELECT COALESCE(bool_or(readonly), FALSE)
		FROM duckdb_databases()
		WHERE database_name NOT IN ('system', 'temp')
	`).Scan(&anyReadOnly)
	if err != nil {
		// Probe indisponible (DuckDB trop ancien, table système absente en test
		// minimal) — ne pas bloquer ; on dégrade vers le comportement legacy
		// (tenter l'écriture, comptée failed si elle échoue).
		return nil
	}
	if anyReadOnly {
		return ErrSharedReadOnly
	}
	return nil
}
