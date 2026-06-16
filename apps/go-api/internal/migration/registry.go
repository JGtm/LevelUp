// Package migration — registre et exécuteur des migrations DuckDB.
//
// Portage de src/data/migration/registry.py + runner.py.
// Chaque migration est idempotente et trackée dans schema_migrations.
package migration

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/observability/logging"
)

// TargetDB identifie la base cible d'une migration.
type TargetDB string

const (
	TargetPlayer       TargetDB = "player"
	TargetShared       TargetDB = "shared"
	TargetSharedPvE    TargetDB = "shared_pve"
	TargetSharedSocial TargetDB = "shared_social"
	TargetMetadata     TargetDB = "metadata"
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

// IsRegistered indique si une migration de ce Name est déjà enregistrée. Utilisé
// par les enregistrements dynamiques title-owned (seeds TOML) pour rester idempotents
// sans accéder au registre privé (Phase 1.5 voie B).
func IsRegistered(name string) bool {
	for _, m := range registry {
		if m.Name == name {
			return true
		}
	}
	return false
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
func ensureMigrationTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
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
func getApplied(ctx context.Context, db *sql.DB) (map[string]migrationState, error) {
	rows, err := db.QueryContext(ctx, "SELECT name, schema_done, backfill_done FROM schema_migrations")
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

// RunForDB applique toutes les migrations enregistrées (registre global) pour
// une DB/target donnée. Conservé pour compat ; délègue à RunSteps. Le chemin
// title-agnostic (Phase 1.5.1, ADR 0025) appellera RunSteps avec les steps
// fournis par l'adapter du titre.
//
// Le db fourni doit être ouvert en lecture/écriture. Pour target=shared, la DB
// metadata doit être ATTACHée en amont si des vues y font référence.
func RunForDB(db *sql.DB, target TargetDB) error {
	return RunSteps(db, target, stepsForTarget(target))
}

// RunSteps applique un ensemble EXPLICITE de migrations pour une target —
// primitive title-agnostic (Phase 1.5.1 B). Les steps peuvent venir du registre
// global (legacy) et/ou de l'adapter d'un titre ; l'ordre d'exécution est imposé
// par canonicalOrder (order.go), indépendant de la source/ordre d'enregistrement.
func RunSteps(db *sql.DB, target TargetDB, steps []Migration) error {
	// Sprint B1 commit 19 : event_id par cycle de migrations sur une DB, pour
	// tracer laquelle plante en cas de schema mismatch. context.Background() car
	// pas de scope HTTP/RPC au boot (callers nombreux).
	ctx, evID := logging.WithEvent(context.Background(), "migration.run:"+string(target))
	cycleStart := time.Now()
	slog.InfoContext(ctx, "migration: cycle démarré", "target", target, "event", evID)

	if err := ensureMigrationTable(ctx, db); err != nil {
		return fmt.Errorf("migration: ensure table: %w", err)
	}
	applied, err := getApplied(ctx, db)
	if err != nil {
		return fmt.Errorf("migration: get applied: %w", err)
	}

	// Ordre d'exécution EXPLICITE (Phase 1.5.0, voir order.go) : indépendant de
	// l'ordre des init()/fichiers/source. Copie défensive pour ne pas muter la
	// slice de l'appelant (ex. steps mémorisés par un adapter).
	steps = append([]Migration(nil), steps...)
	sortByCanonicalOrder(steps)
	appliedCount := 0
	for _, m := range steps {
		state, exists := applied[m.Name]

		if !exists {
			// Nouvelle migration → appliquer le schéma
			mStart := time.Now()
			slog.InfoContext(ctx, "migration: applying schema", "name", m.Name, "target", target)
			if err := m.ApplySchema(db); err != nil {
				slog.ErrorContext(ctx, "migration: schema apply échoué",
					"name", m.Name, "target", target,
					"duration_ms", time.Since(mStart).Milliseconds(), "err", err)
				return fmt.Errorf("migration %s schema: %w", m.Name, err)
			}
			backfillDone := m.ApplyBackfill == nil
			if _, err := db.ExecContext(ctx,
				`INSERT INTO schema_migrations (name, description, applied_at, schema_done, backfill_done)
				 VALUES (?, ?, ?, TRUE, ?)`,
				m.Name, m.Description, time.Now(), backfillDone,
			); err != nil {
				return fmt.Errorf("migration %s record: %w", m.Name, err)
			}
			slog.DebugContext(ctx, "migration: schema OK",
				"name", m.Name, "duration_ms", time.Since(mStart).Milliseconds())
			state = migrationState{SchemaDone: true, BackfillDone: backfillDone}
			appliedCount++
		}

		// Backfill si schéma fait mais backfill manquant
		if state.SchemaDone && !state.BackfillDone && m.ApplyBackfill != nil {
			if m.RequiresAPI {
				slog.DebugContext(ctx, "migration: skipping backfill (requires API)", "name", m.Name)
				continue
			}
			bStart := time.Now()
			slog.InfoContext(ctx, "migration: applying backfill", "name", m.Name, "target", target)
			if err := m.ApplyBackfill(db); err != nil {
				slog.WarnContext(ctx, "migration: backfill failed",
					"name", m.Name, "duration_ms", time.Since(bStart).Milliseconds(), "err", err)
				continue // ne bloque pas les suivantes
			}
			if _, err := db.ExecContext(ctx,
				"UPDATE schema_migrations SET backfill_done = TRUE WHERE name = ?",
				m.Name,
			); err != nil {
				slog.WarnContext(ctx, "migration: update backfill_done failed", "name", m.Name, "err", err)
			}
		}
	}
	checkpointPostMigration(ctx, db, target, appliedCount)

	slog.InfoContext(ctx, "migration: cycle terminé",
		"target", target,
		"applied", appliedCount,
		"total", len(steps),
		"duration_ms", time.Since(cycleStart).Milliseconds(),
	)
	return nil
}

// shouldCheckpointAfterMigration retourne true pour les DB sensibles au bug
// WAL DuckDB #7659. shared_social + shared (matches) + player sont les
// targets concernés par les ATTACH/DDL historiques. metadata est plus
// pure (lecture seule en runtime, écriture rare) — pas critique mais on
// CHECKPOINT quand même pour défense en profondeur.
func shouldCheckpointAfterMigration(target TargetDB) bool {
	switch target {
	case TargetSharedSocial, TargetShared, TargetPlayer, TargetMetadata:
		return true
	default:
		return false
	}
}

// checkpointPostMigration exécute un CHECKPOINT post-cycle sur les targets
// sensibles au bug WAL DuckDB #7659 (Bonus 14 ADR 0021). No-op si
// `appliedCount == 0` ou si la target n'est pas listée comme sensible.
//
// Best-effort : si le CHECKPOINT échoue (lock contention rare en boot),
// log WARN et on continue — les DDL sont déjà commit, le scheduler 5min
// du serveur fait fallback.
//
// Extrait de RunForDB pour respecter la règle 80 lignes/fonction.
func checkpointPostMigration(ctx context.Context, db *sql.DB, target TargetDB, appliedCount int) {
	if appliedCount <= 0 || !shouldCheckpointAfterMigration(target) {
		return
	}
	ckStart := time.Now()
	if _, err := db.ExecContext(ctx, "CHECKPOINT"); err != nil {
		slog.WarnContext(ctx, "migration: CHECKPOINT post-cycle échoué (non-fatal)",
			"target", target, "err", err)
		return
	}
	slog.DebugContext(ctx, "migration: CHECKPOINT post-cycle OK",
		"target", target, "duration_ms", time.Since(ckStart).Milliseconds())
}
