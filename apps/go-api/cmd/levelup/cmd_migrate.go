// cmd_migrate.go — sous-commande migrate pour la migration vers namespace multi-titres.
//
//	levelup migrate --mode dry-run|apply|rollback [--title halo_infinite]
package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops/migrate"
)

func runMigrate(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	mode := fs.String("mode", "dry-run", "Mode: dry-run, apply, rollback")
	titleSlug := fs.String("title", titlePkg.DefaultSlug, "Slug du titre à migrer")
	jsonOutput := fs.Bool("json", false, "Sortie JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	switch *mode {
	case "dry-run":
		return runMigrateDryRun(cfg.RepoRoot, *titleSlug, *jsonOutput)
	case "apply":
		return runMigrateApply(cfg.RepoRoot, *titleSlug, *jsonOutput)
	case "rollback":
		return runMigrateRollback(cfg.RepoRoot, *titleSlug)
	default:
		return fmt.Errorf("mode inconnu : %q (attendu: dry-run, apply, rollback)", *mode)
	}
}

func runMigrateDryRun(repoRoot, titleSlug string, asJSON bool) error {
	plan, err := migrate.BuildPlan(repoRoot, titleSlug)
	if err != nil {
		return fmt.Errorf("plan de migration: %w", err)
	}

	if asJSON {
		data, err := json.MarshalIndent(plan, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("=== Plan de migration vers namespace %q ===\n\n", titleSlug)
	for _, op := range plan.Operations {
		kind := "fichier"
		if op.IsDir {
			kind = "dossier"
		}
		fmt.Printf("  [%s] %s → %s\n", kind, op.Source, op.Dest)
	}
	if len(plan.Warnings) > 0 {
		fmt.Println("\nAvertissements :")
		for _, w := range plan.Warnings {
			fmt.Printf("  ⚠ %s\n", w)
		}
	}
	fmt.Printf("\nTotal : %d opérations\n", len(plan.Operations))
	fmt.Println("Utiliser --mode apply pour exécuter.")
	return nil
}

func runMigrateApply(repoRoot, titleSlug string, asJSON bool) error {
	plan, err := migrate.BuildPlan(repoRoot, titleSlug)
	if err != nil {
		return fmt.Errorf("plan de migration: %w", err)
	}

	if len(plan.Operations) == 0 {
		fmt.Println("Aucune opération à effectuer.")
		return nil
	}

	manifest, err := migrate.Apply(repoRoot, plan)
	if err != nil {
		return fmt.Errorf("migration échouée: %w", err)
	}

	if asJSON {
		data, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Migration terminée : %d opérations\n", len(manifest.Operations))
	fmt.Printf("Manifest écrit : data/titles/%s/migration_manifest.json\n", titleSlug)
	return nil
}

func runMigrateRollback(repoRoot, titleSlug string) error {
	if err := migrate.Rollback(repoRoot, titleSlug); err != nil {
		return fmt.Errorf("rollback échoué: %w", err)
	}
	fmt.Println("Rollback terminé.")
	return nil
}
