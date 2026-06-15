// Package service — title_diagnostic_service.go : rapport de santé d'un titre
// (PMT-14 volet A). Compose la présence des mappings TOML (config) et la réalité
// DB (présence des bases + lignes des tables cœur) via un port.TableInspector
// read-only. Aucune écriture.
package service

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

// TitleDiagnosticService produit le rapport de diagnostic d'un titre.
type TitleDiagnosticService struct {
	repoRoot  string
	inspector port.TableInspector
}

// NewTitleDiagnosticService crée le service.
func NewTitleDiagnosticService(repoRoot string, inspector port.TableInspector) *TitleDiagnosticService {
	return &TitleDiagnosticService{repoRoot: repoRoot, inspector: inspector}
}

type diagConfigFile struct {
	name     string
	required bool
}

// diagnosticConfigFiles = mappings TOML attendus pour un titre. fields +
// capabilities sont requis (sans eux le titre n'est pas exploitable).
var diagnosticConfigFiles = []diagConfigFile{
	{"fields.toml", true},
	{"capabilities.toml", true},
	{"outcomes.toml", false},
	{"assets.toml", false},
	{"constants.toml", false},
}

// Diagnose produit le rapport pour titleSlug. Best-effort : une DB absente ou
// illisible n'interrompt pas le rapport (statut renseigné par DB/table).
func (s *TitleDiagnosticService) Diagnose(ctx context.Context, titleSlug string) (*domain.TitleDiagnostic, error) {
	pr := titlePkg.NewPathResolver(s.repoRoot)
	report := &domain.TitleDiagnostic{TitleSlug: titleSlug}

	mappingsDir := filepath.Join(s.repoRoot, "config", "titles", titleSlug, "mappings")
	for _, cf := range diagnosticConfigFiles {
		_, err := os.Stat(filepath.Join(mappingsDir, cf.name))
		report.ConfigFiles = append(report.ConfigFiles, domain.ConfigFileStatus{
			Name: cf.name, Present: err == nil, Required: cf.required,
		})
	}

	probes := []struct {
		name   string
		path   string
		tables []string
	}{
		{"metadata.duckdb", pr.MetadataDBPath(titleSlug), []string{"season_calendars", "career_ranks", "asset_translations"}},
		{"shared_matches_v2.duckdb", pr.SharedDBPath(titleSlug), []string{"match_registry", "match_participants"}},
		{"shared_pve.duckdb", pr.SharedPVEDBPath(titleSlug), []string{"pve_match_stats"}},
	}
	for _, p := range probes {
		ds := domain.DatabaseStatus{Name: p.name}
		if _, err := os.Stat(p.path); err != nil {
			report.Databases = append(report.Databases, ds) // Exists=false
			continue
		}
		ds.Exists = true
		for _, tbl := range p.tables {
			n, exists, err := s.inspector.CountRows(ctx, p.path, tbl)
			if err != nil {
				if ds.Error == "" {
					ds.Error = err.Error()
				}
				continue
			}
			ds.Tables = append(ds.Tables, domain.TableStatus{Name: tbl, Exists: exists, Rows: n})
		}
		report.Databases = append(report.Databases, ds)
	}

	slog.InfoContext(ctx, "title_diagnostic_report",
		"title", titleSlug,
		"config_files", len(report.ConfigFiles),
		"databases", len(report.Databases),
	)
	return report, nil
}
