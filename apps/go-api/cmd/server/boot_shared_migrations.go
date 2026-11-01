// boot_shared_migrations.go — migrations des bases partagées au démarrage : reprise bornée du
// verrou, garde de version, copie de sécurité, puis montée de schéma.
//
// Ordre, sur chaque base partagée déjà présente :
//  1. lecture SEULE de la version de schéma (migration.InspectSchema) : une base plus récente
//     que le binaire arrête tout (ErrSchemaNewerThanBinary), sans qu'aucune base soit ouverte
//     en écriture ;
//  2. si des étapes sont en attente, copie de la base FERMÉE (migration.BackupBeforeUpgrade) ;
//  3. seulement alors, runMigrations ouvre en écriture et monte le schéma.
//
// Un verrou tenu par un autre processus (hot-reload qui libère ses poignées, instance qui
// s'arrête) est repris pendant lockRetryAttempts essais : la même reprise sert l'ensemble
// garde + migrations et l'ouverture de metadata qui suit (retryOnFileLock). Toute autre erreur
// est définitive : main refuse de démarrer (boot_failure.go).
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/games/titleseams"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/diskfree"
	"levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"
)

const (
	// lockRetryAttempts × lockRetryDelay : fenêtre de reprise d'un verrou transitoire. Sur
	// Windows, après la mort d'air, l'OS peut mettre plusieurs secondes à libérer les poignées
	// DuckDB orphelines ; 6 × 500 ms ne suffisait pas dans certains cas observés.
	lockRetryAttempts = 12
	lockRetryDelay    = 500 * time.Millisecond
)

// bootMigrationTarget : une base partagée et la target de ses migrations.
type bootMigrationTarget struct {
	path   string
	target migration.TargetDB
}

// bootMigrationDeps réunit ce que migrateSharedAtBoot lit et appelle (injectable en test).
type bootMigrationDeps struct {
	slug    string
	targets []bootMigrationTarget
	// backupsDir : dossier des copies avant montée ; "" = pas de copie (démo : fixtures
	// régénérables).
	backupsDir string
	freeBytes  func(path string) (uint64, error)
	now        func() time.Time
	openRead   func(path string) (*sql.DB, func(), error)
	// lockAttempts × lockDelay : fenêtre de reprise d'un verrou transitoire.
	lockAttempts int
	lockDelay    time.Duration
	// prepare enregistre les étapes title-owned avant la garde (prepareMigrationRegistry).
	prepare func() error
	// run monte le schéma (runMigrations en production).
	run func() error
}

// sharedMigrationTargets liste les bases partagées dans l'ordre de runMigrations.
func sharedMigrationTargets(dbPaths bootDBPaths) []bootMigrationTarget {
	return []bootMigrationTarget{
		{dbPaths.metadata, migration.TargetMetadata},
		{dbPaths.shared, migration.TargetShared},
		{dbPaths.pve, migration.TargetSharedPvE},
		{dbPaths.sharedSocial, migration.TargetSharedSocial},
	}
}

// newBootMigrationDeps câble la production : lecture via le fournisseur DuckDB, espace libre
// du volume, runMigrations.
func newBootMigrationDeps(slug, backupsDir, prestigeConfigDir string, targets []bootMigrationTarget, run func() error) bootMigrationDeps {
	return bootMigrationDeps{
		slug:       slug,
		targets:    targets,
		backupsDir: backupsDir,
		prepare:    func() error { return prepareMigrationRegistry(prestigeConfigDir) },
		freeBytes: func(path string) (uint64, error) {
			free, _, err := diskfree.Free(path)
			return free, err
		},
		now:          time.Now,
		openRead:     func(path string) (*sql.DB, func(), error) { return duckdb.OpenReadForQuery(path) },
		lockAttempts: lockRetryAttempts,
		lockDelay:    lockRetryDelay,
		run:          run,
	}
}

// migrateSharedAtBoot exécute garde, copie et migrations, avec reprise du verrou. Rend l'état
// d'échec et l'erreur ; (“”, nil) si tout est monté.
func migrateSharedAtBoot(ctx context.Context, d bootMigrationDeps) (bootFailureState, error) {
	if err := d.prepare(); err != nil {
		return bootFailureMigrationFailed, err
	}
	err := retryOnFileLock(ctx, "migrations", d.lockAttempts, d.lockDelay, func() error {
		states, err := d.inspectAll(ctx)
		if err != nil {
			return err
		}
		if err := d.backupPending(ctx, states); err != nil {
			return err
		}
		return d.run()
	})
	if err != nil {
		return classifyBootFailure(err), err
	}
	return "", nil
}

// inspectedBase : une base présente et son état de schéma.
type inspectedBase struct {
	bootMigrationTarget
	state migration.SchemaState
}

// inspectAll lit l'état de schéma de chaque base présente (lecture seule, poignée fermée aussitôt)
// et refuse la première dont la version dépasse celle du binaire.
func (d bootMigrationDeps) inspectAll(ctx context.Context) ([]inspectedBase, error) {
	var out []inspectedBase
	for _, t := range d.targets {
		if _, err := os.Stat(t.path); errors.Is(err, os.ErrNotExist) {
			continue // base à créer : rien à garder ni à comparer
		} else if err != nil {
			return nil, fmt.Errorf("migration: base %s illisible: %w", t.path, err)
		}
		st, err := d.inspectOne(ctx, t)
		if err != nil {
			return nil, err
		}
		if err := st.CheckNotNewer(); err != nil {
			return nil, fmt.Errorf("%s: %w", t.path, err)
		}
		slog.DebugContext(ctx, "migration: état avant montée",
			"path", t.path, "target", t.target, "version", st.Version,
			"binary_version", st.BinaryVersion, "pending", st.Pending)
		out = append(out, inspectedBase{t, st})
	}
	return out, nil
}

// inspectOne ouvre la base en lecture seule, lit son état, la referme.
func (d bootMigrationDeps) inspectOne(ctx context.Context, t bootMigrationTarget) (migration.SchemaState, error) {
	db, release, err := d.openRead(t.path)
	if err != nil {
		return migration.SchemaState{}, fmt.Errorf("migration: ouverture en lecture de %s: %w", t.path, err)
	}
	defer release()
	st, err := migration.InspectSchema(ctx, db, d.slug, t.target)
	if err != nil {
		return st, fmt.Errorf("%s: %w", t.path, err)
	}
	return st, nil
}

// backupPending copie chaque base qui a des étapes en attente (fermée : les poignées de lecture
// ont été relâchées par inspectOne). Un refus (espace, E/S) échoue le démarrage.
func (d bootMigrationDeps) backupPending(ctx context.Context, bases []inspectedBase) error {
	if d.backupsDir == "" {
		return nil
	}
	for _, b := range bases {
		if b.state.Pending == 0 {
			continue
		}
		if _, err := migration.BackupBeforeUpgrade(ctx, migration.BackupParams{
			DBPath:    b.path,
			DestDir:   d.backupsDir,
			Version:   b.state.Version,
			Now:       d.now(),
			FreeBytes: d.freeBytes,
		}); err != nil {
			return err
		}
	}
	return nil
}

// retryOnFileLock exécute fn et la rejoue, jusqu'à attempts essais séparés de delay, tant
// qu'elle échoue sur un verrou de fichier tenu par un autre processus. Toute autre erreur, ou
// la dernière erreur de verrou, est rendue telle quelle. what nomme l'opération dans le
// journal (DEBUG par essai ; l'échec définitif est journalisé par l'appelant).
func retryOnFileLock(ctx context.Context, what string, attempts int, delay time.Duration, fn func() error) error {
	for attempt := 1; ; attempt++ {
		err := fn()
		if err == nil || !isBootFileLock(err) || attempt >= attempts {
			return err
		}
		// Cause connue et transitoire (hot-reload, ancien process pas encore parti) : DEBUG +
		// compteur expvar ; l'échec définitif remonte à l'appelant.
		observability.IncCounter("boot_metadata_lock_retry_total")
		slog.DebugContext(ctx, "base verrouillée, nouvelle tentative...",
			"what", what, "attempt", attempt, "max", attempts, "err", err)
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return err
		}
	}
}

// prepareMigrationRegistry enregistre les étapes de migration title-owned (seeds Prestige et
// jalons, seams de titre) et vérifie les classifiers LUSR. Idempotente : appelée avant la garde
// de version (qui compte les étapes enregistrées) puis par runMigrations.
func prepareMigrationRegistry(prestigeConfigDir string) error {
	// Ensure all step init() have been registered (side-effect imports).
	_ = migration.All()

	// Seed Prestige catalogue via migration backfill (une seule fois, idempotent).
	if prestigeConfigDir != "" {
		halomigrations.RegisterPrestigeSeedMigration(prestigeConfigDir)
	}

	// Seed Milestones catalogue (Phase 4 plan stabilisation 2026-05-22) via
	// migration backfill multi-titres. Pattern identique à Prestige.
	// configTitlesRoot = parent du dossier titre (ex: config/titles/) — itère
	// sur tous les `<slug>/milestones/catalog.toml` présents.
	// Cf. AUDIT_ASCENSION_PIPELINE_DISCONNECTED_2026-05-21 §4 cause A.
	if prestigeConfigDir != "" {
		configTitlesRoot := filepath.Dir(prestigeConfigDir)
		halomigrations.RegisterMilestonesSeedMigration(configTitlesRoot)
	}

	// Seams title-owned (provider d'étapes, racine des jalons h5, traductions de
	// rangs, classifiers LUSR + famille objectif) : câblage UNIQUE, partagé avec
	// toutes les CLI depuis le 2026-09-16 (cf. internal/games/titleseams — les
	// commentaires MT-07 / MT-15 / D-A y ont déménagé). Sans ce câblage, un sync
	// hors serveur panique à l'étape LUSR du post-sync.
	titleseams.RegisterAll(prestigeConfigDir)
	// Lot B (audit robustesse) : fail-fast au boot si le classifier LUSR par
	// défaut n'a pas été posé, au lieu du panic tardif au 1er match live.
	if err := syncpkg.ValidateLUSRChainClassifierWired(); err != nil {
		return fmt.Errorf("boot: %w", err)
	}
	// Idem pour la famille objectif : sans elle, tout le classé serait noté en
	// ranked_slayer (le fallback existe pour les binaires hors chemin de scoring,
	// pas pour le serveur).
	if err := syncpkg.ValidateObjectiveFamilyClassifierWired(); err != nil {
		return fmt.Errorf("boot: %w", err)
	}
	return nil
}
