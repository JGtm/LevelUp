package main

// migrations_cli.go — application des migrations de schéma par la CLI.
//
// POURQUOI (C-C du plan robustesse 2026-09-16). `sync-full` / `sync-delta` n'appliquaient
// AUCUNE migration : seuls le boot du serveur et les commandes `backfill` le faisaient. Le
// 2026-09-16, l'élargissement `match_registry.team_{0,1}_score SMALLINT → INTEGER` était
// livré mais pas appliqué pendant 83 minutes de synchronisation : les deux matchs à gros
// score d'équipe ont été rejetés une SECONDE fois. Un match rejeté du registre est perdu
// pour tous les joueurs.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
)

// applyMigrationsOnDB ouvre une DB en RW et applique les migrations enregistrees
// pour la cible. Idempotent — DuckDB tolere une migration deja appliquee via
// schema_migrations.
//
// Un seul exemplaire pour toute la CLI (déplacé depuis cmd_backfill.go le 2026-09-16) :
// treize appelants s'en servent, la copie n'avait plus lieu d'être dans un fichier de
// backfill. `OpenReadWrite` passe par le cache `rw:` du paquet duckdb — ce n'est pas un
// « bare connect », le modèle mono-process (ADR 0013) est respecté.
func applyMigrationsOnDB(path string, target migration.TargetDB) error {
	_ = migration.All()
	db, err := duckdbpkg.OpenReadWrite(path)
	if err != nil {
		return fmt.Errorf("open rw %s: %w", path, err)
	}
	defer db.Close()
	return migration.RunForDB(db.SQLDb(), target)
}

// applySharedMigrationsForTitle applique les migrations SHARED du titre donné avant une
// passe de synchronisation, et journalise combien d'étapes ont réellement été posées.
//
// `RunForTitleDB` et non `RunForDB` : ce dernier force `DefaultSlug` et appliquerait le jeu
// de Halo Infinite à la base d'un autre titre.
func applySharedMigrationsForTitle(cfg *config.AppConfig, titleSlug string) error {
	if titleSlug == "" {
		titleSlug = titlePkg.DefaultSlug
	}
	path := titlePkg.NewPathResolver(cfg.RepoRoot).SharedDBPath(titleSlug)

	_ = migration.All()
	db, err := duckdbpkg.OpenReadWrite(path)
	if err != nil {
		return fmt.Errorf("open rw %s: %w", path, err)
	}
	defer db.Close()

	before := countAppliedMigrations(db.SQLDb())
	if err := migration.RunForTitleDB(db.SQLDb(), titleSlug, migration.TargetShared); err != nil {
		return fmt.Errorf("migrations shared %s: %w", titleSlug, err)
	}
	applied := countAppliedMigrations(db.SQLDb()) - before
	if applied < 0 {
		applied = 0
	}
	slog.InfoContext(context.Background(), "migrations shared appliquées",
		"title_slug", titleSlug, "appliquees", applied, "db", path)
	return nil
}

// countAppliedMigrations compte les lignes de `schema_migrations`. Rend 0 si la table
// n'existe pas encore (base neuve) ou si la lecture échoue : c'est un COMPTEUR DE JOURNAL,
// jamais une condition — il ne doit pas faire échouer une passe de sync.
func countAppliedMigrations(db *sql.DB) int {
	var n int
	if err := db.QueryRowContext(context.Background(), `SELECT count(*) FROM schema_migrations`).Scan(&n); err != nil {
		slog.DebugContext(context.Background(), "migrations: comptage indisponible", "err", err)
		return 0
	}
	return n
}
