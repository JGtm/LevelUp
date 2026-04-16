// Package migration — registre et exécuteur des migrations DuckDB.
//
// Portage de src/data/migration/registry.py + runner.py.
// Chaque migration est idempotente et trackée dans schema_migrations.
package migration

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// TargetDB identifie la base cible d'une migration.
type TargetDB string

const (
	TargetPlayer    TargetDB = "player"
	TargetShared    TargetDB = "shared"
	TargetSharedPvE TargetDB = "shared_pve"
	TargetMetadata  TargetDB = "metadata"
)

// Migration décrit une migration DuckDB idempotente.
type Migration struct {
	Name          string
	TargetDB      TargetDB
	Description   string
	ApplySchema   func(db *sql.DB) error // DDL obligatoire
	ApplyBackfill func(db *sql.DB) error // Backfill optionnel
	RequiresAPI   bool                   // Si true, backfill ignoré sans API
}

// registry est le registre ordonné (ordre d'insertion = ordre d'exécution).
var registry []Migration

// Register ajoute une migration au registre global.
func Register(m Migration) {
	registry = append(registry, m)
}

// All retourne toutes les migrations enregistrées dans l'ordre.
func All() []Migration {
	return registry
}

// ForTarget filtre les migrations par target_db.
func ForTarget(target TargetDB) []Migration {
	var out []Migration
	for _, m := range registry {
		if m.TargetDB == target {
			out = append(out, m)
		}
	}
	return out
}

// migrationState est l'état d'une migration dans schema_migrations.
type migrationState struct {
	SchemaDone   bool
	BackfillDone bool
}

// ensureMigrationTable crée la table de tracking si absente.
func ensureMigrationTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name          VARCHAR PRIMARY KEY,
			description   VARCHAR,
			applied_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			schema_done   BOOLEAN DEFAULT FALSE,
			backfill_done BOOLEAN DEFAULT FALSE
		)
	`)
	return err
}

// getApplied charge l'état de toutes les migrations déjà appliquées.
func getApplied(db *sql.DB) (map[string]migrationState, error) {
	rows, err := db.Query("SELECT name, schema_done, backfill_done FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	applied := make(map[string]migrationState)
	for rows.Next() {
		var name string
		var s migrationState
		if err := rows.Scan(&name, &s.SchemaDone, &s.BackfillDone); err != nil {
			return nil, err
		}
		applied[name] = s
	}
	return applied, rows.Err()
}

// RunForDB applique toutes les migrations en attente pour une DB donnée.
//
// Le db fourni doit être ouvert en lecture/écriture.
// Pour target=shared, la DB metadata doit être ATTACHée en amont si des vues
// y font référence.
func RunForDB(db *sql.DB, target TargetDB) error {
	if err := ensureMigrationTable(db); err != nil {
		return fmt.Errorf("migration: ensure table: %w", err)
	}
	applied, err := getApplied(db)
	if err != nil {
		return fmt.Errorf("migration: get applied: %w", err)
	}

	migrations := ForTarget(target)
	for _, m := range migrations {
		state, exists := applied[m.Name]

		if !exists {
			// Nouvelle migration → appliquer le schéma
			slog.Info("migration: applying schema", "name", m.Name, "target", target)
			if err := m.ApplySchema(db); err != nil {
				return fmt.Errorf("migration %s schema: %w", m.Name, err)
			}
			backfillDone := m.ApplyBackfill == nil
			if _, err := db.Exec(
				`INSERT INTO schema_migrations (name, description, applied_at, schema_done, backfill_done)
				 VALUES (?, ?, ?, TRUE, ?)`,
				m.Name, m.Description, time.Now(), backfillDone,
			); err != nil {
				return fmt.Errorf("migration %s record: %w", m.Name, err)
			}
			state = migrationState{SchemaDone: true, BackfillDone: backfillDone}
		}

		// Backfill si schéma fait mais backfill manquant
		if state.SchemaDone && !state.BackfillDone && m.ApplyBackfill != nil {
			if m.RequiresAPI {
				slog.Debug("migration: skipping backfill (requires API)", "name", m.Name)
				continue
			}
			slog.Info("migration: applying backfill", "name", m.Name, "target", target)
			if err := m.ApplyBackfill(db); err != nil {
				slog.Warn("migration: backfill failed", "name", m.Name, "err", err)
				continue // ne bloque pas les suivantes
			}
			if _, err := db.Exec(
				"UPDATE schema_migrations SET backfill_done = TRUE WHERE name = ?",
				m.Name,
			); err != nil {
				slog.Warn("migration: update backfill_done failed", "name", m.Name, "err", err)
			}
		}
	}
	return nil
}
