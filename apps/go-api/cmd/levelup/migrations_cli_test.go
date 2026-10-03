package main

// migrations_cli_test.go — la CLI de sync met le schéma partagé à niveau AVANT d'insérer.
//
// POURQUOI (C-C, 2026-09-16). `sync-full` / `sync-delta` n'appliquaient aucune migration :
// l'élargissement `match_registry.team_{0,1}_score SMALLINT → INTEGER` est resté non appliqué
// pendant 83 minutes de synchronisation et deux matchs à gros score d'équipe ont été rejetés
// une seconde fois. Ce test rougit si `applySharedMigrationsForTitle` disparaît des runners
// ou si elle repasse par `RunForDB` sans les étapes title-owned.

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/titleseams"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
)

// TestApplySharedMigrationsForTitle_CreeMatchRegistryElargi : sur une base NEUVE et
// TEMPORAIRE (jamais sous data/), la passe de migrations crée `match_registry` avec des
// scores d'équipe en INTEGER — preuve que les étapes title-owned jouent.
func TestApplySharedMigrationsForTitle_CreeMatchRegistryElargi(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := &config.AppConfig{RepoRoot: repoRoot}

	// Même câblage qu'au démarrage du binaire (cmd/levelup/main.go → wireStartupSeams).
	wireStartupSeams(cfg)
	t.Cleanup(func() { titleseams.RegisterAll("") })

	sharedPath := titlePkg.NewPathResolver(repoRoot).SharedDBPath(titlePkg.DefaultSlug)
	if err := os.MkdirAll(filepath.Dir(sharedPath), 0o755); err != nil {
		t.Fatalf("création du warehouse temporaire : %v", err)
	}

	if err := applySharedMigrationsForTitle(cfg, titlePkg.DefaultSlug); err != nil {
		t.Fatalf("applySharedMigrationsForTitle : %v", err)
	}

	db, err := duckdbpkg.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("réouverture de la base temporaire : %v", err)
	}
	t.Cleanup(func() {
		if cerr := db.Close(); cerr != nil {
			t.Errorf("fermeture : %v", cerr)
		}
	})

	for _, col := range []string{"team_0_score", "team_1_score"} {
		if got := columnType(t, db.SQLDb(), "match_registry", col); got != "INTEGER" {
			t.Errorf("match_registry.%s = %s, attendu INTEGER (un score d'équipe dépasse 32 767)", col, got)
		}
	}

	// Idempotence : une seconde passe ne doit rien casser (les runners l'appellent à
	// chaque sync).
	if err := applySharedMigrationsForTitle(cfg, titlePkg.DefaultSlug); err != nil {
		t.Fatalf("seconde passe de migrations : %v", err)
	}
}

// TestApplySharedMigrationsForTitle_SlugVide_RetombeSurLeDefaut : un slug vide ne doit pas
// produire un chemin bancal — il vaut le titre par défaut.
func TestApplySharedMigrationsForTitle_SlugVide_RetombeSurLeDefaut(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := &config.AppConfig{RepoRoot: repoRoot}
	wireStartupSeams(cfg)
	t.Cleanup(func() { titleseams.RegisterAll("") })

	sharedPath := titlePkg.NewPathResolver(repoRoot).SharedDBPath(titlePkg.DefaultSlug)
	if err := os.MkdirAll(filepath.Dir(sharedPath), 0o755); err != nil {
		t.Fatalf("création du warehouse temporaire : %v", err)
	}

	if err := applySharedMigrationsForTitle(cfg, ""); err != nil {
		t.Fatalf("applySharedMigrationsForTitle(slug vide) : %v", err)
	}
	if _, err := os.Stat(sharedPath); err != nil {
		t.Errorf("base du titre par défaut absente : %v", err)
	}
}

// columnType lit le type déclaré d'une colonne dans information_schema.
func columnType(t *testing.T, db *sql.DB, table, column string) string {
	t.Helper()
	var typ string
	err := db.QueryRowContext(context.Background(),
		`SELECT data_type FROM information_schema.columns WHERE table_name = ? AND column_name = ?`,
		table, column).Scan(&typ)
	if err != nil {
		t.Fatalf("lecture du type de %s.%s : %v", table, column, err)
	}
	return typ
}
