package lab

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"levelup/go-api/internal/domain"
	metadata_guard "levelup/go-api/internal/metadata"
	"levelup/go-api/internal/platform/duckdb"
	"strings"
)

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
		if err := rows.Scan(
			&titleID, &item.AssetID, &item.AssetType, &item.VersionID,
			&name, &description, &rawJSON, &item.FetchedAt, &contentHash,
		); err != nil {
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
		if err := rows.Scan(
			&titleID, &item.MedalID, &item.NameID, &item.DescriptionID,
			&item.SpriteIndex, &item.Difficulty, &item.MedalType,
			&personalScore, &rawJSON, &item.FetchedAt, &contentHash,
		); err != nil {
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
	if err := row.Scan(
		&titleID, &item.MedalID, &item.NameID, &item.DescriptionID,
		&item.SpriteIndex, &item.Difficulty, &item.MedalType,
		&item.PersonalScore, &item.RawJSON, &item.FetchedAt, &item.ContentHash,
	); err != nil {
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
