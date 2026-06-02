// Package lab charge les données du Lab interne depuis DuckDB et le filesystem.
package lab

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	metadata_guard "levelup/go-api/internal/metadata"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
)

var _ port.LabProvider = (*Provider)(nil)

var pathParamRE = regexp.MustCompile(`\{[^}]+\}`)

var openAPIMethods = map[string]bool{
	"get":     true,
	"post":    true,
	"put":     true,
	"patch":   true,
	"delete":  true,
	"head":    true,
	"options": true,
}

// Provider charge les données du Lab interne.
type Provider struct {
	cfg *config.AppConfig
}

// NewProvider crée un provider du Lab interne.
func NewProvider(cfg *config.AppConfig) *Provider {
	return &Provider{cfg: cfg}
}

// GetResources charge le panneau Explorateur interne.
func (p *Provider) GetResources(
	ctx context.Context,
	titleSlug string,
	query domain.LabResourcesQuery,
) (*domain.LabResourcesResponse, error) {
	metaPath := p.metadataDBPath(titleSlug)
	// OpenReadWriteShared : partage la même instance DuckDB que le pool joueur ;
	// OpenReadOnly créerait une 2e config sur le même fichier → erreur DuckDB.
	metaDB, err := duckdb.OpenReadWriteShared(metaPath)
	if err != nil {
		return nil, fmt.Errorf("lab resources open metadata: %w", err)
	}
	defer metaDB.Close()

	repo := duckdb.NewMetadataRepoFromDB(metaDB)
	currentSeason, err := repo.GetCurrentSeason(ctx, titleSlug)
	if err != nil && !isMissingRelationError(err) {
		return nil, err
	}
	seasons, err := listSeasonCalendars(ctx, metaDB, titleSlug)
	if err != nil {
		return nil, err
	}
	csrSeasons, err := repo.GetCSRSeasons(ctx, titleSlug)
	if err != nil && !isMissingRelationError(err) {
		return nil, err
	}
	snapshots, err := listSnapshots(ctx, metaDB, titleSlug, query.Limit)
	if err != nil {
		return nil, err
	}
	selectedSnapshot, err := loadSelectedSnapshot(ctx, repo, titleSlug, query.SnapshotKey)
	if err != nil {
		return nil, err
	}
	assets, err := loadAssets(ctx, metaDB, repo, titleSlug, query)
	if err != nil {
		return nil, err
	}
	medals, err := loadMedals(ctx, metaDB, titleSlug, query)
	if err != nil {
		return nil, err
	}

	return &domain.LabResourcesResponse{
		TitleSlug:        titleSlug,
		MetadataDBPath:   metaPath,
		CurrentSeason:    currentSeason,
		Seasons:          seasons,
		CSRSeasons:       orEmptyCSR(csrSeasons),
		Snapshots:        snapshots,
		SelectedSnapshot: selectedSnapshot,
		Assets:           assets,
		Medals:           medals,
	}, nil
}

// GetContracts charge le diff OpenAPI côté Go.
func (p *Provider) GetContracts(ctx context.Context) (*domain.LabContractsResponse, error) {
	goPath := filepath.Join(p.cfg.RepoRoot, "apps", "go-api", "api", "openapi.yaml")
	fastapiPath := filepath.Join(p.cfg.RepoRoot, "apps", "go-api", "api", "openapi_fastapi_reference.yaml")

	goFile, err := buildFileStatus(goPath)
	if err != nil {
		return nil, err
	}
	fastapiFile, err := buildFileStatus(fastapiPath)
	if err != nil {
		return nil, err
	}

	goRoutes, err := loadOpenAPIRoutes(goPath)
	if err != nil {
		return nil, err
	}
	fastapiRoutes, err := loadOpenAPIRoutes(fastapiPath)
	if err != nil {
		return nil, err
	}
	missing, extra, mismatches := compareOpenAPIRoutes(fastapiRoutes, goRoutes)

	_ = ctx
	return &domain.LabContractsResponse{
		GoOpenAPI:        goFile,
		FastAPIReference: fastapiFile,
		Summary: domain.LabOpenAPISummary{
			FastAPIRouteCount: len(fastapiRoutes),
			GoRouteCount:      len(goRoutes),
			MissingInGo:       len(missing),
			ExtraInGo:         len(extra),
			MethodMismatches:  len(mismatches),
			Status:            labContractStatus(missing, mismatches),
		},
		MissingInGo:      missing,
		ExtraInGo:        extra,
		MethodMismatches: mismatches,
	}, nil
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
	// OpenReadWriteShared : voir commentaire dans GetResources (config DuckDB unique par fichier).
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

func listSeasonCalendars(
	ctx context.Context,
	metaDB *duckdb.DB,
	titleSlug string,
) ([]domain.SeasonCalendar, error) {
	rows, err := metaDB.Query(ctx, `
		SELECT title_id, season_id, version, name, start_date, end_date, fetched_at, content_hash, etag, source_url
		FROM season_calendars
		WHERE title_id = ?
		ORDER BY start_date DESC`, titleSlug)
	if err != nil {
		if isMissingRelationError(err) {
			return []domain.SeasonCalendar{}, nil
		}
		return nil, fmt.Errorf("list season calendars: %w", err)
	}
	defer rows.Close()

	var seasons []domain.SeasonCalendar
	for rows.Next() {
		season, err := scanSeasonCalendar(rows)
		if err != nil {
			return nil, err
		}
		seasons = append(seasons, season)
	}
	return seasons, rows.Err()
}

func listSnapshots(
	ctx context.Context,
	metaDB *duckdb.DB,
	titleSlug string,
	limit int,
) ([]domain.LabSnapshotSummary, error) {
	rows, err := metaDB.Query(ctx, `
		SELECT title_id, resource_key, version, fetched_at, content_hash, etag, source_url, payload
		FROM waypoint_resource_snapshots
		WHERE title_id = ?
		ORDER BY fetched_at DESC
		LIMIT ?`, titleSlug, limit)
	if err != nil {
		if isMissingRelationError(err) {
			return []domain.LabSnapshotSummary{}, nil
		}
		return nil, fmt.Errorf("list snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []domain.LabSnapshotSummary
	for rows.Next() {
		var titleID string
		var item domain.LabSnapshotSummary
		var payload string
		if err := rows.Scan(
			&titleID, &item.ResourceKey, &item.Version, &item.FetchedAt,
			&item.ContentHash, &item.ETag, &item.SourceURL, &payload,
		); err != nil {
			return nil, fmt.Errorf("scan snapshot: %w", err)
		}
		item.PayloadSize = len(payload)
		snapshots = append(snapshots, item)
	}
	return snapshots, rows.Err()
}

func loadSelectedSnapshot(
	ctx context.Context,
	repo *duckdb.MetadataRepo,
	titleSlug string,
	resourceKey string,
) (*domain.LabSnapshotDetail, error) {
	if strings.TrimSpace(resourceKey) == "" {
		return nil, nil
	}
	snap, err := repo.GetSnapshot(ctx, titleSlug, resourceKey)
	if err != nil {
		if isMissingRelationError(err) {
			return nil, nil
		}
		return nil, err
	}
	if snap == nil {
		return nil, nil
	}
	return &domain.LabSnapshotDetail{
		ResourceKey: snap.ResourceKey,
		Version:     snap.Version,
		FetchedAt:   snap.FetchedAt,
		ContentHash: snap.ContentHash,
		ETag:        snap.ETag,
		SourceURL:   snap.SourceURL,
		Payload:     snap.Payload,
	}, nil
}
