// Package lab charge le diagnostic d'instance (ex-Lab) depuis DuckDB et le
// filesystem.
//
// A3.5 (DC-9, 2026-07-10) : le Lab est retiré de l'app — ne reste que
// GetDiagnostics (rapport de parité + garde-fous médailles), consommé par le
// panneau Diagnostics de l'onglet admin Données. Les explorateurs Resources /
// Contracts / Waypoint sont partis avec leurs endpoints.
package lab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	metadata_guard "levelup/go-api/internal/metadata"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
)

var _ port.LabProvider = (*Provider)(nil)

// Provider charge le diagnostic d'instance.
type Provider struct {
	cfg *config.AppConfig
}

// NewProvider crée un provider du diagnostic d'instance.
func NewProvider(cfg *config.AppConfig) *Provider {
	return &Provider{cfg: cfg}
}

// GetDiagnostics charge le rapport de parité stocké et les garde-fous médailles.
func (p *Provider) GetDiagnostics(
	ctx context.Context,
	titleSlug string,
) (*domain.LabDiagnosticsResponse, error) {
	parityPath := filepath.Join(p.cfg.RepoRoot, "apps", "go-api", "tests", "fixtures", "parity_report.json")
	parityFile, err := buildFileStatus(parityPath)
	if err != nil {
		return nil, err
	}
	parityReport, err := loadParityReport(parityFile)
	if err != nil {
		return nil, err
	}
	medalGuards, err := p.loadMedalGuards(ctx, titleSlug)
	if err != nil {
		return nil, err
	}
	return &domain.LabDiagnosticsResponse{
		TitleSlug:        titleSlug,
		ParityReportFile: parityFile,
		ParityReport:     parityReport,
		MedalGuards:      medalGuards,
	}, nil
}

func (p *Provider) loadMedalGuards(
	ctx context.Context,
	titleSlug string,
) (*domain.LabMedalGuardsReport, error) {
	// OpenReadWriteShared : partage la même instance DuckDB que le pool joueur ;
	// OpenReadOnly créerait une 2e config sur le même fichier → erreur DuckDB.
	metaDB, err := duckdb.OpenReadWriteShared(p.metadataDBPath(titleSlug))
	if err != nil {
		return nil, nil
	}
	defer metaDB.Close()
	entries, err := listAllMedalEntries(ctx, metaDB, titleSlug)
	if err != nil {
		if isMissingRelationError(err) {
			return nil, nil
		}
		return nil, err
	}
	cardinality := metadata_guard.CheckCardinalityGuard(len(entries), len(entries), 10.0)
	required := metadata_guard.CheckRequiredFieldsGuard(entries)
	images := metadata_guard.CheckImageGuard(entries)
	overall := metadata_guard.RunAllGuards(entries, len(entries))
	return &domain.LabMedalGuardsReport{
		EntryCount:     len(entries),
		Cardinality:    toGuardResult(cardinality),
		RequiredFields: toGuardResult(required),
		Images:         toGuardResult(images),
		Overall:        toGuardResult(overall),
	}, nil
}

func (p *Provider) metadataDBPath(titleSlug string) string {
	if titleSlug == "" {
		titleSlug = titlePkg.DefaultSlug
	}
	if p.cfg.DemoMode {
		return filepath.Join(p.cfg.DemoFixturesDir, "metadata.duckdb")
	}
	registry := titlePkg.NewRegistry()
	resolver := titlePkg.NewPathResolver(p.cfg.RepoRoot, registry)
	return resolver.MetadataDBPath(titleSlug)
}

func buildFileStatus(path string) (domain.LabFileStatus, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return domain.LabFileStatus{Path: path, Exists: false}, nil
	}
	if err != nil {
		return domain.LabFileStatus{}, fmt.Errorf("stat %s: %w", path, err)
	}
	modifiedAt := info.ModTime().UTC()
	return domain.LabFileStatus{
		Path:       path,
		Exists:     true,
		SizeBytes:  info.Size(),
		ModifiedAt: &modifiedAt,
	}, nil
}

func loadParityReport(file domain.LabFileStatus) (*domain.LabParityReport, error) {
	if !file.Exists {
		return nil, nil
	}
	data, err := os.ReadFile(file.Path)
	if err != nil {
		return nil, fmt.Errorf("read parity report: %w", err)
	}
	var report domain.LabParityReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parse parity report: %w", err)
	}
	return &report, nil
}

// listAllMedalEntries charge les entrées waypoint_medals_raw pour les guards.
func listAllMedalEntries(
	ctx context.Context,
	metaDB *duckdb.DB,
	titleSlug string,
) ([]metadata_guard.MedalEntry, error) {
	rows, err := metaDB.Query(ctx, `
		SELECT title_id, medal_id, name_id, description_id, medal_type, difficulty, '', sprite_index, raw_json
		FROM waypoint_medals_raw
		WHERE title_id = ?
		ORDER BY medal_id ASC`, titleSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []metadata_guard.MedalEntry
	for rows.Next() {
		var entry metadata_guard.MedalEntry
		if err := rows.Scan(
			&entry.TitleID, &entry.MedalID, &entry.Label, &entry.Description,
			&entry.Category, &entry.Rarity, &entry.ImageURL,
			&entry.SpriteIdx, &entry.RawJSON,
		); err != nil {
			return nil, fmt.Errorf("scan medal guard entry: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func isMissingRelationError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "catalog error") && strings.Contains(msg, "does not exist")
}

func toGuardResult(result metadata_guard.GuardResult) domain.LabGuardResult {
	return domain.LabGuardResult{Passed: result.Passed, Reason: result.Reason, Details: result.Details}
}
