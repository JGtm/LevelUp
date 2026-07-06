//go:build integration

// Tests d'intégration du loader TOML Prestige — placés dans le package duckdb
// pour éviter le cycle d'import (prestige importerait duckdb).

package prestige

import (
	"context"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/prestige"
)

func TestLoadTemplatesFromTOML_HaloInfinite(t *testing.T) {
	db := setupPrestigeDB(t, migration.TargetMetadata)
	repo := NewPrestigeTemplateRepo(db)
	ctx := context.Background()

	path := filepath.Join(repoRoot(t), "config", "titles", "halo_infinite", "challenges", "templates.toml")

	count, err := prestige.LoadTemplatesFromTOML(ctx, repo, path)
	if err != nil {
		t.Fatalf("LoadTemplatesFromTOML: %v", err)
	}
	if count < 10 {
		t.Errorf("expected >= 10 templates, got %d", count)
	}

	list, err := repo.ListByTitle(ctx, "halo_infinite")
	if err != nil {
		t.Fatalf("ListByTitle: %v", err)
	}
	for _, tpl := range list {
		if !(tpl.NormalTarget < tpl.HeroicTarget &&
			tpl.HeroicTarget < tpl.LegendaryTarget &&
			tpl.LegendaryTarget < tpl.MythicTarget) {
			t.Errorf("template %s: paliers non monotones", tpl.ID)
		}
	}
}

func TestLoadPresetArcsFromTOML_HaloInfinite(t *testing.T) {
	db := setupPrestigeDB(t, migration.TargetMetadata)
	repo := NewPrestigePresetArcRepo(db)
	ctx := context.Background()

	path := filepath.Join(repoRoot(t), "config", "titles", "halo_infinite", "arcs", "presets.toml")

	count, err := prestige.LoadPresetArcsFromTOML(ctx, repo, path)
	if err != nil {
		t.Fatalf("LoadPresetArcsFromTOML: %v", err)
	}
	if count < 4 {
		t.Errorf("expected >= 4 preset arcs, got %d", count)
	}

	arcs, err := repo.ListByTitle(ctx, "halo_infinite")
	if err != nil {
		t.Fatalf("ListByTitle: %v", err)
	}
	if len(arcs) < 4 {
		t.Fatalf("expected >= 4 arcs in DB, got %d", len(arcs))
	}
	for _, a := range arcs {
		steps, err := repo.GetSteps(ctx, a.ID)
		if err != nil {
			t.Fatalf("GetSteps(%s): %v", a.ID, err)
		}
		if len(steps) == 0 {
			t.Errorf("arc %s has no steps", a.ID)
		}
	}
}

// repoRoot remonte de apps/go-api/internal/platform/duckdb/prestige/ jusqu'à la
// racine du repo (6 niveaux — un de plus depuis l'extraction du sous-package K3a).
func repoRoot(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	return abs
}
