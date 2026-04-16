// Package migrate implémente la migration des données legacy vers le namespace multi-titres.
//
// Sprint 44 WP3 : déplace data/warehouse/ et data/players/ vers
// data/titles/{title_slug}/warehouse/ et data/titles/{title_slug}/players/.
//
// Modes :
//   - dry-run : affiche le plan sans toucher au disque
//   - apply   : exécute les déplacements + écrit le manifest
//   - rollback: restaure les chemins legacy depuis le manifest
package migrate

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
)

// Manifest décrit les opérations de migration réalisées.
type Manifest struct {
	Version    string     `json:"version"`
	TitleSlug  string     `json:"title_slug"`
	MigratedAt time.Time  `json:"migrated_at"`
	Operations []MoveOp   `json:"operations"`
	RolledBack bool       `json:"rolled_back"`
	RollbackAt *time.Time `json:"rollback_at,omitempty"`
}

// MoveOp décrit un déplacement de fichier/dossier.
type MoveOp struct {
	Source string `json:"source"`
	Dest   string `json:"dest"`
	IsDir  bool   `json:"is_dir"`
}

// Plan représente le plan de migration (preview avant apply).
type Plan struct {
	TitleSlug  string   `json:"title_slug"`
	Operations []MoveOp `json:"operations"`
	Warnings   []string `json:"warnings"`
}

// BuildPlan construit le plan de migration pour un titre donné.
// Analyse les chemins legacy existants et détermine les déplacements nécessaires.
func BuildPlan(repoRoot, titleSlug string) (*Plan, error) {
	reg := titlePkg.NewRegistry()
	pr := titlePkg.NewPathResolver(repoRoot, reg)

	plan := &Plan{TitleSlug: titleSlug}

	// 1. data/warehouse/ → data/titles/{title}/warehouse/
	legacyWarehouse := pr.LegacyWarehouseDir()
	if dirExists(legacyWarehouse) {
		plan.Operations = append(plan.Operations, MoveOp{
			Source: legacyWarehouse,
			Dest:   pr.WarehouseDir(titleSlug),
			IsDir:  true,
		})
	} else {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("répertoire legacy warehouse absent : %s", legacyWarehouse))
	}

	// 2. data/players/*/ → data/titles/{title}/players/*/
	legacyPlayersDir := filepath.Join(repoRoot, "data", "players")
	entries, err := os.ReadDir(legacyPlayersDir)
	if err != nil {
		if os.IsNotExist(err) {
			plan.Warnings = append(plan.Warnings, "répertoire data/players/ absent")
		} else {
			return nil, fmt.Errorf("lecture data/players/: %w", err)
		}
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		gamertag := e.Name()
		plan.Operations = append(plan.Operations, MoveOp{
			Source: filepath.Join(legacyPlayersDir, gamertag),
			Dest:   pr.PlayerDir(titleSlug, gamertag),
			IsDir:  true,
		})
	}

	return plan, nil
}

// Apply exécute le plan de migration et écrit le manifest.
func Apply(repoRoot string, plan *Plan) (*Manifest, error) {
	manifest := &Manifest{
		Version:    "1.0",
		TitleSlug:  plan.TitleSlug,
		MigratedAt: time.Now(),
	}

	for _, op := range plan.Operations {
		// Créer le parent du dest.
		if err := os.MkdirAll(filepath.Dir(op.Dest), 0o755); err != nil {
			return nil, fmt.Errorf("mkdir %s: %w", filepath.Dir(op.Dest), err)
		}

		// Vérifier que la destination n'existe pas déjà.
		if _, err := os.Stat(op.Dest); err == nil {
			return nil, fmt.Errorf("destination existe déjà : %s", op.Dest)
		}

		slog.Info("migrate: déplacement", "source", op.Source, "dest", op.Dest)
		if err := os.Rename(op.Source, op.Dest); err != nil {
			return nil, fmt.Errorf("rename %s → %s: %w", op.Source, op.Dest, err)
		}
		manifest.Operations = append(manifest.Operations, op)
	}

	// Écrire le manifest.
	manifestPath := filepath.Join(repoRoot, "data", "titles", plan.TitleSlug, "migration_manifest.json")
	if err := writeManifest(manifestPath, manifest); err != nil {
		return nil, fmt.Errorf("écriture manifest: %w", err)
	}

	return manifest, nil
}

// Rollback annule la migration en utilisant le manifest.
func Rollback(repoRoot, titleSlug string) error {
	manifestPath := filepath.Join(repoRoot, "data", "titles", titleSlug, "migration_manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("lecture manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
	}

	if manifest.RolledBack {
		return fmt.Errorf("migration déjà annulée (rollback_at: %v)", manifest.RollbackAt)
	}

	// Rollback en ordre inverse.
	for i := len(manifest.Operations) - 1; i >= 0; i-- {
		op := manifest.Operations[i]
		slog.Info("rollback: restauration", "source", op.Dest, "dest", op.Source)

		// Recréer le parent source si nécessaire.
		if err := os.MkdirAll(filepath.Dir(op.Source), 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(op.Source), err)
		}

		if err := os.Rename(op.Dest, op.Source); err != nil {
			return fmt.Errorf("rollback rename %s → %s: %w", op.Dest, op.Source, err)
		}
	}

	// Marquer le manifest comme rollbacké.
	now := time.Now()
	manifest.RolledBack = true
	manifest.RollbackAt = &now
	if err := writeManifest(manifestPath, &manifest); err != nil {
		return fmt.Errorf("mise à jour manifest rollback: %w", err)
	}

	return nil
}

// --- helpers ---

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func writeManifest(path string, m *Manifest) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
