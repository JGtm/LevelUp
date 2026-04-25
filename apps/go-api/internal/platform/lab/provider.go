// Package lab charge les données du Lab interne depuis DuckDB et le filesystem.
package lab

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

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
		if err := rows.Scan(&titleID, &item.ResourceKey, &item.Version, &item.FetchedAt, &item.ContentHash, &item.ETag, &item.SourceURL, &payload); err != nil {
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

func loadAssets(
	ctx context.Context,
	metaDB *duckdb.DB,
	repo *duckdb.MetadataRepo,
	titleSlug string,
	query domain.LabResourcesQuery,
) (domain.LabAssetExplorer, error) {
	total, err := countAssets(ctx, metaDB, titleSlug, query.AssetSearch)
	if err != nil {
		return domain.LabAssetExplorer{}, err
	}
	items, err := listAssets(ctx, metaDB, titleSlug, query.AssetSearch, query.Limit)
	if err != nil {
		return domain.LabAssetExplorer{}, err
	}
	selected, err := loadSelectedAsset(ctx, repo, titleSlug, query.AssetID)
	if err != nil {
		return domain.LabAssetExplorer{}, err
	}
	return domain.LabAssetExplorer{Total: total, Search: query.AssetSearch, Items: items, Selected: selected}, nil
}

func countAssets(ctx context.Context, metaDB *duckdb.DB, titleSlug, search string) (int, error) {
	row := metaDB.QueryRow(ctx, `
		SELECT COUNT(DISTINCT asset_id)
		FROM waypoint_assets_raw
		WHERE title_id = ?
		  AND (? = '' OR LOWER(asset_id) LIKE ? OR LOWER(asset_type) LIKE ? OR LOWER(name) LIKE ?)`,
		titleSlug, search, likeQuery(search), likeQuery(search), likeQuery(search))
	var total int
	if err := row.Scan(&total); err != nil {
		if isMissingRelationError(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("count assets: %w", err)
	}
	return total, nil
}

func listAssets(
	ctx context.Context,
	metaDB *duckdb.DB,
	titleSlug string,
	search string,
	limit int,
) ([]domain.LabAssetSummary, error) {
	rows, err := metaDB.Query(ctx, `
		SELECT * FROM (
			SELECT DISTINCT ON (asset_id)
				title_id, asset_id, asset_type, version_id, name, description, raw_json, fetched_at, content_hash
			FROM waypoint_assets_raw
			WHERE title_id = ?
			ORDER BY asset_id, fetched_at DESC
		) latest
		WHERE (? = '' OR LOWER(asset_id) LIKE ? OR LOWER(asset_type) LIKE ? OR LOWER(name) LIKE ?)
		ORDER BY fetched_at DESC, asset_id ASC
		LIMIT ?`, titleSlug, search, likeQuery(search), likeQuery(search), likeQuery(search), limit)
	if err != nil {
		if isMissingRelationError(err) {
			return []domain.LabAssetSummary{}, nil
		}
		return nil, fmt.Errorf("list assets: %w", err)
	}
	defer rows.Close()

	var items []domain.LabAssetSummary
	for rows.Next() {
		var titleID string
		var name string
		var description string
		var rawJSON string
		var contentHash string
		var item domain.LabAssetSummary
		if err := rows.Scan(&titleID, &item.AssetID, &item.AssetType, &item.VersionID, &name, &description, &rawJSON, &item.FetchedAt, &contentHash); err != nil {
			return nil, fmt.Errorf("scan asset: %w", err)
		}
		item.Name = name
		items = append(items, item)
	}
	return items, rows.Err()
}

func loadSelectedAsset(
	ctx context.Context,
	repo *duckdb.MetadataRepo,
	titleSlug string,
	assetID string,
) (*domain.LabAssetDetail, error) {
	if strings.TrimSpace(assetID) == "" {
		return nil, nil
	}
	asset, err := repo.GetAssetByID(ctx, titleSlug, assetID)
	if err != nil {
		if isMissingRelationError(err) {
			return nil, nil
		}
		return nil, err
	}
	if asset == nil {
		return nil, nil
	}
	return &domain.LabAssetDetail{
		AssetID:     asset.AssetID,
		AssetType:   asset.AssetType,
		VersionID:   asset.VersionID,
		Name:        asset.Name,
		Description: asset.Description,
		FetchedAt:   asset.FetchedAt,
		ContentHash: asset.ContentHash,
		RawJSON:     asset.RawJSON,
	}, nil
}

func loadMedals(
	ctx context.Context,
	metaDB *duckdb.DB,
	titleSlug string,
	query domain.LabResourcesQuery,
) (domain.LabMedalExplorer, error) {
	total, err := countMedals(ctx, metaDB, titleSlug, query.MedalSearch)
	if err != nil {
		return domain.LabMedalExplorer{}, err
	}
	items, err := listMedals(ctx, metaDB, titleSlug, query.MedalSearch, query.Limit)
	if err != nil {
		return domain.LabMedalExplorer{}, err
	}
	selected, err := loadSelectedMedal(ctx, metaDB, titleSlug, query.MedalID)
	if err != nil {
		return domain.LabMedalExplorer{}, err
	}
	return domain.LabMedalExplorer{Total: total, Search: query.MedalSearch, Items: items, Selected: selected}, nil
}

func countMedals(ctx context.Context, metaDB *duckdb.DB, titleSlug, search string) (int, error) {
	row := metaDB.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM waypoint_medals_raw
		WHERE title_id = ?
		  AND (? = '' OR LOWER(name_id) LIKE ? OR LOWER(description_id) LIKE ? OR LOWER(medal_type) LIKE ?)`,
		titleSlug, search, likeQuery(search), likeQuery(search), likeQuery(search))
	var total int
	if err := row.Scan(&total); err != nil {
		if isMissingRelationError(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("count medals: %w", err)
	}
	return total, nil
}

func listMedals(
	ctx context.Context,
	metaDB *duckdb.DB,
	titleSlug string,
	search string,
	limit int,
) ([]domain.LabMedalSummary, error) {
	rows, err := metaDB.Query(ctx, `
		SELECT title_id, medal_id, name_id, description_id, sprite_index, difficulty, medal_type, personal_score, raw_json, fetched_at, content_hash
		FROM waypoint_medals_raw
		WHERE title_id = ?
		  AND (? = '' OR LOWER(name_id) LIKE ? OR LOWER(description_id) LIKE ? OR LOWER(medal_type) LIKE ?)
		ORDER BY fetched_at DESC, medal_id ASC
		LIMIT ?`, titleSlug, search, likeQuery(search), likeQuery(search), likeQuery(search), limit)
	if err != nil {
		if isMissingRelationError(err) {
			return []domain.LabMedalSummary{}, nil
		}
		return nil, fmt.Errorf("list medals: %w", err)
	}
	defer rows.Close()

	var items []domain.LabMedalSummary
	for rows.Next() {
		var titleID string
		var personalScore int
		var rawJSON string
		var contentHash string
		var item domain.LabMedalSummary
		if err := rows.Scan(&titleID, &item.MedalID, &item.NameID, &item.DescriptionID, &item.SpriteIndex, &item.Difficulty, &item.MedalType, &personalScore, &rawJSON, &item.FetchedAt, &contentHash); err != nil {
			return nil, fmt.Errorf("scan medal: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func loadSelectedMedal(
	ctx context.Context,
	metaDB *duckdb.DB,
	titleSlug string,
	medalID int64,
) (*domain.LabMedalDetail, error) {
	if medalID == 0 {
		return nil, nil
	}
	row := metaDB.QueryRow(ctx, `
		SELECT title_id, medal_id, name_id, description_id, sprite_index, difficulty, medal_type, personal_score, raw_json, fetched_at, content_hash
		FROM waypoint_medals_raw
		WHERE title_id = ? AND medal_id = ?`, titleSlug, medalID)
	var titleID string
	var item domain.LabMedalDetail
	if err := row.Scan(&titleID, &item.MedalID, &item.NameID, &item.DescriptionID, &item.SpriteIndex, &item.Difficulty, &item.MedalType, &item.PersonalScore, &item.RawJSON, &item.FetchedAt, &item.ContentHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) || isMissingRelationError(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("load medal detail: %w", err)
	}
	return &item, nil
}

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
		if err := rows.Scan(&entry.TitleID, &entry.MedalID, &entry.Label, &entry.Description, &entry.Category, &entry.Rarity, &entry.ImageURL, &entry.SpriteIdx, &entry.RawJSON); err != nil {
			return nil, fmt.Errorf("scan medal guard entry: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
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

func loadOpenAPIRoutes(path string) (map[string][]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read openapi %s: %w", path, err)
	}
	var doc struct {
		Paths map[string]map[string]interface{} `yaml:"paths"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse openapi %s: %w", path, err)
	}
	routes := make(map[string][]string, len(doc.Paths))
	for path, methods := range doc.Paths {
		var declared []string
		for method := range methods {
			if openAPIMethods[strings.ToLower(method)] {
				declared = append(declared, strings.ToLower(method))
			}
		}
		if len(declared) == 0 {
			continue
		}
		sort.Strings(declared)
		routes[path] = declared
	}
	return routes, nil
}

func compareOpenAPIRoutes(
	fastapiRoutes map[string][]string,
	goRoutes map[string][]string,
) ([]domain.LabRouteMethods, []domain.LabRouteMethods, []domain.LabMethodMismatch) {
	faNorm := normalizeRoutes(fastapiRoutes)
	goNorm := normalizeRoutes(goRoutes)

	var missing []domain.LabRouteMethods
	var extra []domain.LabRouteMethods
	var mismatches []domain.LabMethodMismatch

	for normalized, route := range faNorm {
		if goRoute, ok := goNorm[normalized]; ok {
			if !sameMethods(route.Methods, goRoute.Methods) {
				mismatches = append(mismatches, domain.LabMethodMismatch{
					FastAPIPath:    route.Path,
					GoPath:         goRoute.Path,
					FastAPIMethods: route.Methods,
					GoMethods:      goRoute.Methods,
					MissingMethods: diffMethods(route.Methods, goRoute.Methods),
					ExtraMethods:   diffMethods(goRoute.Methods, route.Methods),
				})
			}
			continue
		}
		missing = append(missing, domain.LabRouteMethods{Path: route.Path, Methods: route.Methods})
	}
	for normalized, route := range goNorm {
		if _, ok := faNorm[normalized]; ok {
			continue
		}
		extra = append(extra, domain.LabRouteMethods{Path: route.Path, Methods: route.Methods})
	}

	sortRoutes(missing)
	sortRoutes(extra)
	sortMethodMismatches(mismatches)
	return missing, extra, mismatches
}

func normalizeRoutes(routes map[string][]string) map[string]domain.LabRouteMethods {
	normalized := make(map[string]domain.LabRouteMethods, len(routes))
	for path, methods := range routes {
		normalized[pathParamRE.ReplaceAllString(path, "{*}")] = domain.LabRouteMethods{
			Path:    path,
			Methods: methods,
		}
	}
	return normalized
}

func sortRoutes(routes []domain.LabRouteMethods) {
	sort.Slice(routes, func(i, j int) bool { return routes[i].Path < routes[j].Path })
}

func sortMethodMismatches(items []domain.LabMethodMismatch) {
	sort.Slice(items, func(i, j int) bool { return items[i].FastAPIPath < items[j].FastAPIPath })
}

func scanSeasonCalendar(rows *sql.Rows) (domain.SeasonCalendar, error) {
	var season domain.SeasonCalendar
	var endDate sql.NullTime
	if err := rows.Scan(&season.TitleID, &season.SeasonID, &season.Version, &season.Name, &season.StartDate, &endDate, &season.FetchedAt, &season.ContentHash, &season.ETag, &season.SourceURL); err != nil {
		return domain.SeasonCalendar{}, fmt.Errorf("scan season calendar: %w", err)
	}
	if endDate.Valid {
		season.EndDate = &endDate.Time
	}
	return season, nil
}

func orEmptyCSR(items []domain.CSRSeasonCalendar) []domain.CSRSeasonCalendar {
	if items == nil {
		return []domain.CSRSeasonCalendar{}
	}
	return items
}

func likeQuery(search string) string {
	trimmed := strings.TrimSpace(strings.ToLower(search))
	if trimmed == "" {
		return ""
	}
	return "%" + trimmed + "%"
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

func sameMethods(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func diffMethods(left, right []string) []string {
	lookup := make(map[string]bool, len(right))
	for _, method := range right {
		lookup[method] = true
	}
	var diff []string
	for _, method := range left {
		if !lookup[method] {
			diff = append(diff, method)
		}
	}
	return diff
}

func labContractStatus(missing []domain.LabRouteMethods, mismatches []domain.LabMethodMismatch) string {
	if len(missing) == 0 && len(mismatches) == 0 {
		return "OK"
	}
	return "DIVERGENCES"
}
