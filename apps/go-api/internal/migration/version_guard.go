// version_guard.go — lecture SEULE de l'état de schéma d'une base, avant toute migration.
//
// Le runner (runSteps) ouvre la base en écriture et enregistre la version atteinte dans
// title_schema_version (= longueur de l'ordre canonique du titre). Un binaire plus ancien
// que sa base (retour arrière après une montée) ne doit pas la rouvrir en écriture : il
// ignorerait des étapes qu'il ne connaît pas et pourrait y écrire à contresens. La garde
// lit donc la version sur une connexion que l'APPELANT ouvre en lecture seule (ce paquet
// n'importe pas platform/duckdb, qui l'importe), la compare à celle du binaire, et compte
// les étapes en attente pour décider d'une copie de sécurité (backup_before_upgrade.go).
package migration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ErrSchemaNewerThanBinary : la base porte une version de schéma supérieure à celle que ce
// binaire sait produire. Aucune migration ne doit être tentée ; le démarrage est refusé.
var ErrSchemaNewerThanBinary = errors.New("migration: schéma de la base plus récent que ce binaire")

// SchemaState : ce que la lecture seule apprend d'une base pour (titre, target).
type SchemaState struct {
	// Version : version enregistrée en base (0 si le ledger ou la ligne est absent).
	Version int
	// BinaryVersion : version que ce binaire atteint (longueur de l'ordre canonique).
	BinaryVersion int
	// Pending : nombre d'étapes que le runner appliquerait (schéma seulement).
	Pending int
}

// CheckNotNewer rend ErrSchemaNewerThanBinary (enveloppée des deux versions) si la base
// est plus récente que le binaire, nil sinon.
func (s SchemaState) CheckNotNewer() error {
	if s.Version > s.BinaryVersion {
		return fmt.Errorf("%w : base v%d, binaire v%d", ErrSchemaNewerThanBinary, s.Version, s.BinaryVersion)
	}
	return nil
}

// InspectSchema lit l'état de schéma de db pour (slug, target). db peut être une connexion
// en lecture seule : aucune écriture n'est émise. Doit être appelée une fois les étapes
// title-owned enregistrées (titleseams.RegisterAll), comme pour le runner.
func InspectSchema(ctx context.Context, db *sql.DB, slug string, target TargetDB) (SchemaState, error) {
	steps, order := stepsAndOrderFor(slug, target)
	state := SchemaState{BinaryVersion: len(order)}

	hasVersion, err := tableExists(db, "title_schema_version")
	if err != nil {
		return state, fmt.Errorf("migration: inspection du schéma (title_schema_version): %w", err)
	}
	if hasVersion {
		err := db.QueryRowContext(ctx,
			`SELECT version FROM title_schema_version WHERE title_slug = ? AND target = ?`,
			slug, string(target)).Scan(&state.Version)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return state, fmt.Errorf("migration: lecture de title_schema_version: %w", err)
		}
	}

	applied, err := appliedNames(ctx, db)
	if err != nil {
		return state, err
	}
	for i := range steps {
		if !steps[i].appliesToTitle(slug) {
			continue
		}
		if _, done := applied[steps[i].Name]; done {
			continue
		}
		if supersededBaselineSatisfied(applied, steps[i]) {
			continue
		}
		state.Pending++
	}
	return state, nil
}

// appliedNames lit les noms déjà enregistrés dans schema_migrations, sans créer la table
// (lecture seule) : table absente = aucune étape appliquée.
func appliedNames(ctx context.Context, db *sql.DB) (map[string]migrationState, error) {
	has, err := tableExists(db, "schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("migration: inspection du schéma (schema_migrations): %w", err)
	}
	if !has {
		return map[string]migrationState{}, nil
	}
	applied, err := getApplied(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("migration: lecture de schema_migrations: %w", err)
	}
	return applied, nil
}
