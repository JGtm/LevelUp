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

	// SupersededByAll — équivalence ledger d'un SQUASH (DM-5, chantier N4). Si non
	// vide, ce step est une BASELINE squashée : sur une DB EXISTANTE qui porte déjà
	// la SENTINELLE (dernier nom de la liste = dernier step squashé) dans
	// schema_migrations, la baseline est réputée DÉJÀ SATISFAITE — le runner
	// l'enregistre sans rejouer son DDL (aucun DDL flat rejoué sur une DB prod). Sur
	// une DB VIERGE (sentinelle absente), le DDL est appliqué normalement. La liste
	// (steps squashés, ordonnés) sert aussi de trace d'audit du périmètre du squash.
	SupersededByAll []string
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

// ensureMigrationTable crée la table de tracking si absente. La PK reste sur
// `name` : les DB sont physiquement per-titre (PathResolver), donc un `name` ne
// collisionne jamais entre titres — la colonne title_slug est INFORMATIVE (quel
// jeu de titre a appliqué ce step), pas une clé. ADD COLUMN IF NOT EXISTS rend
// l'ajout idempotent sur les DB Halo créées avant PMT-9 (rétro-rempli au défaut).
func ensureMigrationTable(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name          VARCHAR PRIMARY KEY,
			description   VARCHAR,
			applied_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			schema_done   BOOLEAN DEFAULT FALSE,
			backfill_done BOOLEAN DEFAULT FALSE,
			title_slug    VARCHAR DEFAULT '%s'
		)
	`, DefaultSlug)); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx, fmt.Sprintf(
		`ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS title_slug VARCHAR DEFAULT '%s'`,
		DefaultSlug))
	return err
}

// ensureTitleSchemaVersionTable crée le ledger de version de schéma PAR TITRE
// (PMT-9). 1 ligne par (title_slug, target) : la version courante du jeu de
// migrations de ce titre sur cette base. La PK est posée à la création (jamais
// d'ADD CONSTRAINT a posteriori) → ON CONFLICT fiable (cf. piège
// CREATE-IF-NOT-EXISTS+PK). Écriture boot-only, mono-process : pas de pression
// ART.
func ensureTitleSchemaVersionTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS title_schema_version (
			title_slug VARCHAR NOT NULL,
			target     VARCHAR NOT NULL,
			version    INTEGER NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (title_slug, target)
		)
	`)
	return err
}

// recordTitleSchemaVersion enregistre la version de schéma atteinte par un titre
// sur une target (= len de son ordre canonique, monotone et dérivé tant qu'on n'a
// pas de numérotation explicite). Best-effort : appelé en fin de cycle réussi.
func recordTitleSchemaVersion(ctx context.Context, db *sql.DB, slug string, target TargetDB, version int) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO title_schema_version (title_slug, target, version, applied_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (title_slug, target) DO UPDATE SET
			version = excluded.version, applied_at = excluded.applied_at
	`, slug, string(target), version, time.Now())
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

// RunForDB applique les migrations du titre PAR DÉFAUT (Halo) pour une DB/target.
// Conservé pour compat : délègue à RunForTitleDB(db, DefaultSlug, target). Le db
// fourni doit être ouvert en lecture/écriture ; pour target=shared, metadata doit
// être ATTACHée en amont si des vues y font référence.
func RunForDB(db *sql.DB, target TargetDB) error {
	return RunForTitleDB(db, DefaultSlug, target)
}

// RunForTitleDB applique les migrations d'un titre sur une DB/target (PMT-9).
// Route par slug via un lookup de map (data-driven, JAMAIS de comparaison de slug
// — archlint/no_slug_comparison) : si un TitleMigrationSet est enregistré pour ce
// slug (titre non-défaut), applique SON jeu + SON ordre canonique (isolation
// totale, zéro step Halo) ; sinon retombe sur le chemin legacy (registre global +
// titleStepsProvider, ordonné par canonicalOrder) — byte-identique au défaut Halo.
func RunForTitleDB(db *sql.DB, slug string, target TargetDB) error {
	if set, ok := migrationSetFor(slug); ok && set.ownsTarget(target) {
		return runSteps(db, slug, target, set.Steps(target), set.CanonicalOrder)
	}
	// Pas de set, OU le set n'isole PAS ce target (OwnsTarget==false) → fallback
	// complet (registre global Halo + titleStepsProvider), héritage uniforme.
	return runSteps(db, slug, target, stepsForTarget(target), canonicalOrder)
}

// RunSteps applique un ensemble EXPLICITE de migrations pour une target, tracé
// sous le titre par défaut et ordonné par canonicalOrder. Conservé pour compat.
func RunSteps(db *sql.DB, target TargetDB, steps []Migration) error {
	return runSteps(db, DefaultSlug, target, steps, canonicalOrder)
}

// runSteps applique steps sur db pour (slug, target), ordonnés par order. Trace
// chaque step dans schema_migrations (colonne title_slug) et la version atteinte
// dans title_schema_version (= len(order)). Primitive interne du runner.
func runSteps(db *sql.DB, slug string, target TargetDB, steps []Migration, order []string) error {
	// event_id par cycle de migrations sur une DB, pour tracer laquelle plante en
	// cas de schema mismatch. context.Background() car pas de scope HTTP au boot.
	ctx, evID := logging.WithEvent(context.Background(), fmt.Sprintf("migration.run:%s:%s", slug, target))
	cycleStart := time.Now()
	slog.InfoContext(ctx, "migration: cycle démarré", "title", slug, "target", target, "event", evID)

	if err := ensureMigrationTable(ctx, db); err != nil {
		return fmt.Errorf("migration: ensure table: %w", err)
	}
	if err := ensureTitleSchemaVersionTable(ctx, db); err != nil {
		return fmt.Errorf("migration: ensure title_schema_version: %w", err)
	}
	applied, err := getApplied(ctx, db)
	if err != nil {
		return fmt.Errorf("migration: get applied: %w", err)
	}

	// Ordre d'exécution EXPLICITE (order.go pour le défaut, set.CanonicalOrder
	// pour un titre) : indépendant de l'ordre des init()/fichiers/source. Copie
	// défensive pour ne pas muter la slice de l'appelant.
	steps = append([]Migration(nil), steps...)
	sortByOrder(order, steps)
	appliedCount := 0
	for i := range steps {
		state, exists := applied[steps[i].Name]
		if !exists {
			// DM-5 : une baseline squashée dont la sentinelle (dernier step squashé)
			// est déjà appliquée est réputée satisfaite → enregistrée sans DDL rejoué.
			if supersededBaselineSatisfied(applied, steps[i]) {
				if err := recordSupersededBaseline(ctx, db, slug, steps[i]); err != nil {
					return err
				}
				continue
			}
			newState, err := applyStepSchema(ctx, db, slug, target, steps[i])
			if err != nil {
				return err
			}
			state = newState
			appliedCount++
		}
		applyStepBackfill(ctx, db, target, steps[i], state)
	}
	checkpointPostMigration(ctx, db, target, appliedCount)

	if err := recordTitleSchemaVersion(ctx, db, slug, target, len(order)); err != nil {
		slog.WarnContext(ctx, "migration: title_schema_version non écrit (non-fatal)",
			"title", slug, "target", target, "err", err)
	}
	slog.InfoContext(ctx, "migration: cycle terminé",
		"title", slug, "target", target, "applied", appliedCount,
		"total", len(steps), "schema_version", len(order),
		"duration_ms", time.Since(cycleStart).Milliseconds())
	return nil
}

// supersededBaselineSatisfied indique si m est une baseline squashée dont la
// sentinelle (dernier step squashé) est déjà appliquée sur cette DB — auquel cas le
// schéma cumulé est déjà présent et rejouer le DDL flat de la baseline est inutile
// (DM-5). Les steps squashés étant séquentiels, la présence de la sentinelle implique
// celle de tous ses prédécesseurs.
func supersededBaselineSatisfied(applied map[string]migrationState, m Migration) bool {
	if len(m.SupersededByAll) == 0 {
		return false
	}
	sentinel := m.SupersededByAll[len(m.SupersededByAll)-1]
	_, ok := applied[sentinel]
	return ok
}

// recordSupersededBaseline enregistre une baseline squashée comme appliquée SANS
// rejouer son DDL (DM-5) : la DB porte déjà le schéma cumulé via les steps squashés
// historiques. schema_done=TRUE, backfill_done=TRUE (une baseline n'a pas de backfill).
func recordSupersededBaseline(ctx context.Context, db *sql.DB, slug string, m Migration) error {
	slog.InfoContext(ctx, "migration: baseline squashée réputée satisfaite (DM-5, DDL non rejoué)",
		"name", m.Name, "title", slug, "sentinel", m.SupersededByAll[len(m.SupersededByAll)-1])
	_, err := db.ExecContext(ctx,
		`INSERT INTO schema_migrations (title_slug, name, description, applied_at, schema_done, backfill_done)
		 VALUES (?, ?, ?, ?, TRUE, TRUE)`,
		slug, m.Name, m.Description, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("migration %s record (superseded baseline): %w", m.Name, err)
	}
	return nil
}

// applyStepSchema applique le DDL d'un step nouveau et l'enregistre dans
// schema_migrations (sous title_slug=slug). Retourne l'état résultant.
func applyStepSchema(ctx context.Context, db *sql.DB, slug string, target TargetDB, m Migration) (migrationState, error) {
	mStart := time.Now()
	slog.InfoContext(ctx, "migration: applying schema", "name", m.Name, "title", slug, "target", target)
	if err := m.ApplySchema(db); err != nil {
		slog.ErrorContext(ctx, "migration: schema apply échoué",
			"name", m.Name, "title", slug, "target", target,
			"duration_ms", time.Since(mStart).Milliseconds(), "err", err)
		return migrationState{}, fmt.Errorf("migration %s schema: %w", m.Name, err)
	}
	backfillDone := m.ApplyBackfill == nil
	if _, err := db.ExecContext(ctx,
		`INSERT INTO schema_migrations (title_slug, name, description, applied_at, schema_done, backfill_done)
		 VALUES (?, ?, ?, ?, TRUE, ?)`,
		slug, m.Name, m.Description, time.Now(), backfillDone,
	); err != nil {
		return migrationState{}, fmt.Errorf("migration %s record: %w", m.Name, err)
	}
	slog.DebugContext(ctx, "migration: schema OK", "name", m.Name, "duration_ms", time.Since(mStart).Milliseconds())
	return migrationState{SchemaDone: true, BackfillDone: backfillDone}, nil
}

// applyStepBackfill applique le backfill d'un step si nécessaire (best-effort, ne
// bloque jamais les suivants). No-op si déjà fait, absent, ou RequiresAPI.
func applyStepBackfill(ctx context.Context, db *sql.DB, target TargetDB, m Migration, state migrationState) {
	if !(state.SchemaDone && !state.BackfillDone && m.ApplyBackfill != nil) {
		return
	}
	if m.RequiresAPI {
		slog.DebugContext(ctx, "migration: skipping backfill (requires API)", "name", m.Name)
		return
	}
	bStart := time.Now()
	slog.InfoContext(ctx, "migration: applying backfill", "name", m.Name, "target", target)
	if err := m.ApplyBackfill(db); err != nil {
		slog.WarnContext(ctx, "migration: backfill failed",
			"name", m.Name, "duration_ms", time.Since(bStart).Milliseconds(), "err", err)
		return
	}
	if _, err := db.ExecContext(ctx,
		"UPDATE schema_migrations SET backfill_done = TRUE WHERE name = ?", m.Name,
	); err != nil {
		slog.WarnContext(ctx, "migration: update backfill_done failed", "name", m.Name, "err", err)
	}
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
