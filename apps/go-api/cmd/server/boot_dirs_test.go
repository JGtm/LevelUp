// Package main — boot_dirs_test.go : simulation clone frais (data/ absent).
//
// Valide le fix onboarding #1 sans dépendre d'un login Microsoft SSO :
//   - sans le dossier warehouse, runMigrations échoue (OpenReadWrite ne crée
//     PAS le dossier parent) — c'est la cause du crash boot sur clone frais ;
//   - ensureWarehouseDir crée le dossier → les migrations créent les .duckdb
//     et le serveur peut démarrer.
//
// Build tag cgo : runMigrations dépend de duckdb.OpenReadWrite (cgo).
//
//go:build cgo

package main

import (
	"os"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
)

func TestEnsureWarehouseDir_FreshCloneBoots(t *testing.T) {
	slug := titlePkg.DefaultSlug

	// runMigrationsFor exécute les migrations pour un repoRoot donné, sans seed
	// prestige/milestones (file-dependent → hors scope d'un boot clone frais).
	runMigrationsFor := func(pr *titlePkg.PathResolver) error {
		return runMigrations(
			pr.MetadataDBPath(slug),
			pr.SharedDBPath(slug),
			pr.SharedSocialDBPath(slug),
			pr.SharedPVEDBPath(slug),
			"",
		)
	}

	// --- Cas négatif : dossier warehouse absent → runMigrations échoue ---
	// Prouve que le fix est nécessaire (et que OpenReadWrite ne crée pas le dir).
	t.Run("missing_warehouse_dir_fails", func(t *testing.T) {
		pr := titlePkg.NewPathResolver(t.TempDir())
		if err := runMigrationsFor(pr); err == nil {
			t.Fatal("runMigrations devrait échouer quand le dossier warehouse n'existe pas")
		}
	})

	// --- Cas positif : ensureWarehouseDir crée le dossier → migrations OK ---
	t.Run("ensure_then_migrate_ok", func(t *testing.T) {
		pr := titlePkg.NewPathResolver(t.TempDir())

		if _, err := os.Stat(pr.WarehouseDir(slug)); !os.IsNotExist(err) {
			t.Fatalf("warehouse devrait être absent au départ (stat err = %v)", err)
		}

		if err := ensureWarehouseDir(pr, slug); err != nil {
			t.Fatalf("ensureWarehouseDir: %v", err)
		}
		if _, err := os.Stat(pr.WarehouseDir(slug)); err != nil {
			t.Fatalf("warehouse devrait exister après ensureWarehouseDir: %v", err)
		}

		if err := runMigrationsFor(pr); err != nil {
			t.Fatalf("runMigrations sur clone frais devrait réussir: %v", err)
		}

		// Les bases warehouse doivent avoir été créées par les migrations.
		for _, p := range []string{
			pr.MetadataDBPath(slug),
			pr.SharedDBPath(slug),
			pr.SharedSocialDBPath(slug),
		} {
			if _, err := os.Stat(p); err != nil {
				t.Errorf("DB attendue absente après migrations: %s (%v)", p, err)
			}
		}
	})
}
