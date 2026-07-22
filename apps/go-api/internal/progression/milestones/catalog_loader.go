package milestones

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// catalog_loader.go — chargement du catalogue depuis TOML.
//
// Source : config/titles/{slug}/milestones/catalog.toml
// Appelé au boot pour synchroniser le catalogue en metadata.duckdb. Le
// pattern est tout-ou-rien : si le TOML est invalide, le catalogue n'est
// pas mis à jour et un slog.Warn est émis.

// catalogTOML est la projection brute du fichier TOML.
type catalogTOML struct {
	Meta       catalogMetaTOML      `toml:"meta"`
	Milestones []milestoneEntryTOML `toml:"milestones"`
}

type catalogMetaTOML struct {
	SchemaVersion int    `toml:"schema_version"`
	TitleSlug     string `toml:"title_slug"`
}

type milestoneEntryTOML struct {
	ID          string  `toml:"id"`
	Metric      string  `toml:"metric"`
	Threshold   float64 `toml:"threshold"`
	TitleEN     string  `toml:"title_en"`
	TitleFR     string  `toml:"title_fr"`
	Icon        string  `toml:"icon"`
	Condition   string  `toml:"condition"`
	ConditionFR string  `toml:"condition_fr"`
	ConditionEN string  `toml:"condition_en"`
}

// LoadCatalogFromFile lit le TOML, valide la structure et retourne les entrées
// avec le title_slug du meta appliqué. Retourne une erreur si parsing ou
// validation échoue (catalogue laissé inchangé par le caller).
func LoadCatalogFromFile(path string) ([]CatalogEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("milestones: read %s: %w", path, err)
	}
	return parseCatalogBytes(data)
}

// parseCatalogBytes parse un TOML déjà lu (utile pour les tests avec fixtures
// inline).
func parseCatalogBytes(data []byte) ([]CatalogEntry, error) {
	var raw catalogTOML
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("milestones: TOML parse: %w", err)
	}
	if raw.Meta.TitleSlug == "" {
		return nil, fmt.Errorf("milestones: meta.title_slug required")
	}
	out := make([]CatalogEntry, 0, len(raw.Milestones))
	for i, m := range raw.Milestones {
		if m.ID == "" {
			return nil, fmt.Errorf("milestones: entry %d missing id", i)
		}
		if m.Metric == "" {
			return nil, fmt.Errorf("milestones: entry %s missing metric", m.ID)
		}
		if m.Threshold <= 0 {
			return nil, fmt.Errorf("milestones: entry %s threshold must be > 0, got %v", m.ID, m.Threshold)
		}
		if m.TitleEN == "" || m.TitleFR == "" {
			return nil, fmt.Errorf("milestones: entry %s missing title_en or title_fr", m.ID)
		}
		out = append(out, CatalogEntry{
			ID:          m.ID,
			TitleSlug:   raw.Meta.TitleSlug,
			Metric:      m.Metric,
			Threshold:   m.Threshold,
			TitleEN:     m.TitleEN,
			TitleFR:     m.TitleFR,
			Icon:        m.Icon,
			Condition:   m.Condition,
			ConditionFR: m.ConditionFR,
			ConditionEN: m.ConditionEN,
		})
	}
	return out, nil
}

// SyncCatalog charge le TOML et appelle CatalogRepo.Upsert pour chaque entrée.
// Si le TOML est invalide, log warn et retourne sans toucher au catalogue
// existant (graceful degradation).
func SyncCatalog(ctx context.Context, repo CatalogRepo, path string) error {
	entries, err := LoadCatalogFromFile(path)
	if err != nil {
		slog.WarnContext(ctx, "milestones: catalog load failed, keeping existing DB rows",
			"path", path, "err", err)
		return err
	}
	for _, e := range entries {
		if err := repo.Upsert(ctx, e); err != nil {
			return fmt.Errorf("milestones: upsert %s: %w", e.ID, err)
		}
	}
	slog.InfoContext(ctx, "milestones: catalog synced", "path", path, "count", len(entries))
	return nil
}
