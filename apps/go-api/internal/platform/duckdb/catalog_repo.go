// Package duckdb — catalog_repo.go : Phase F du plan PLAN_PLAYLISTS_CATALOG.md.
//
// Implémente port.CatalogRepo en lecture sur metadata.duckdb (catalogue
// Playlists/Pairs/Maps title-aware). Utilisé par FiltersService (Phase I) et
// les endpoints catalogue (Phase H).
package duckdb

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/domain"
)

// CatalogRepo implémente port.CatalogRepo.
//
// sharedDB *sql.DB remplacé par SharedReader pour
// permettre la coordination avec le SharedDBProvider (cycle RO↔RW).
type CatalogRepo struct {
	// metadataDB est le wrapper *DB (et NON un *sql.DB nu) pour permettre la
	// recovery FATAL via QueryRecovered : sur invalidation du handle metadata
	// (incident ON CONFLICT), Reopen() swap le sqlDB interne et les lectures
	// catalogue se reconnectent au lieu de casser jusqu'au restart (revue
	// 2026-06-01 META-4).
	metadataDB   *DB
	sharedReader SharedReader // pour le JOIN quand onlyPlayed = true ; peut être nil
}

// NewCatalogRepo construit le repo avec metadataDB (wrapper *DB) et un SharedReader
// optionnel. sharedReader peut être nil si on ne supporte pas onlyPlayed (le
// catalogue complet reste accessible).
func NewCatalogRepo(metadataDB *DB, sharedReader SharedReader) *CatalogRepo {
	return &CatalogRepo{metadataDB: metadataDB, sharedReader: sharedReader}
}

// PlaylistsByTitle retourne les playlists du catalogue pour un titre donné.
func (r *CatalogRepo) PlaylistsByTitle(ctx context.Context, titleSlug, xuid string, onlyPlayed bool) ([]domain.CatalogPlaylist, error) {
	if r.metadataDB == nil {
		return nil, fmt.Errorf("CatalogRepo: metadataDB nil")
	}
	if onlyPlayed && r.sharedReader != nil && xuid != "" {
		return r.playlistsPlayedByXUID(ctx, titleSlug, xuid)
	}

	rows, err := r.metadataDB.QueryRecovered(ctx, `
		SELECT title_slug, playlist_asset_id, COALESCE(current_version_id, ''),
		       COALESCE(name_canonical, ''), COALESCE(experience, ''), COALESCE(is_ranked, FALSE)
		FROM playlists_catalog
		WHERE title_slug = ? AND is_active = TRUE
		ORDER BY name_canonical
	`, titleSlug)
	if err != nil {
		slog.ErrorContext(ctx, "CatalogRepo.PlaylistsByTitle: query failed", "err", err, "title_slug", titleSlug)
		return nil, fmt.Errorf("query playlists: %w", err)
	}
	defer rows.Close()
	var out []domain.CatalogPlaylist
	for rows.Next() {
		var p domain.CatalogPlaylist
		if err := rows.Scan(&p.TitleSlug, &p.PlaylistAssetID, &p.CurrentVersionID, &p.Name, &p.Experience, &p.IsRanked); err != nil {
			return nil, fmt.Errorf("scan playlist: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// playlistsPlayedByXUID retourne uniquement les playlists ayant ≥ 1 match joué par xuid.
//
// split+merge cross-DB.
//
//	Étape 1 (SharedReader) : SELECT DISTINCT playlist_id, COUNT(DISTINCT match_id)
//	  FROM match_registry mr JOIN match_participants mp ON mp.match_id = mr.match_id
//	  WHERE mp.xuid = ? GROUP BY playlist_id → liste (playlist_id, match_count).
//	Étape 2 (metadataDB) : SELECT FROM playlists_catalog WHERE title_slug AND
//	  playlist_asset_id IN (...).
//	Étape 3 (Go) : merge — hydrate match_count + ordre name_canonical.
func (r *CatalogRepo) playlistsPlayedByXUID(ctx context.Context, titleSlug, xuid string) ([]domain.CatalogPlaylist, error) {
	// Étape 1 : playlists jouées + count depuis shared.
	sharedDB, release, err := r.sharedReader.Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "CatalogRepo.playlistsPlayedByXUID: shared reader indisponible, fallback catalogue complet",
			"err", err, "title_slug", titleSlug)
		return r.PlaylistsByTitle(ctx, titleSlug, "", false)
	}
	defer release()

	sharedRows, err := sharedDB.QueryContext(ctx, `
		SELECT mr.playlist_id, COUNT(DISTINCT mr.match_id)
		FROM match_registry mr
		JOIN match_participants mp ON mp.match_id = mr.match_id
		WHERE mp.xuid = ?`+excludeCampaignClause(titleSlug, "mr")+` AND mr.playlist_id IS NOT NULL
		GROUP BY mr.playlist_id
	`, xuid)
	if err != nil {
		slog.WarnContext(ctx, "CatalogRepo.playlistsPlayedByXUID: shared query failed, fallback catalogue complet",
			"err", err, "title_slug", titleSlug)
		return r.PlaylistsByTitle(ctx, titleSlug, "", false)
	}
	counts := make(map[string]int)
	for sharedRows.Next() {
		var pid string
		var n int
		if err := sharedRows.Scan(&pid, &n); err != nil {
			sharedRows.Close()
			return nil, fmt.Errorf("scan shared playlist: %w", err)
		}
		counts[pid] = n
	}
	sharedRows.Close()
	if len(counts) == 0 {
		return nil, nil
	}

	// Étape 2 : metadata playlists_catalog pour les playlist_ids retournés.
	ids := make([]string, 0, len(counts))
	for id := range counts {
		ids = append(ids, id)
	}
	q := fmt.Sprintf(`
		SELECT title_slug, playlist_asset_id, COALESCE(current_version_id, ''),
		       COALESCE(name_canonical, ''), COALESCE(experience, ''), COALESCE(is_ranked, FALSE)
		FROM playlists_catalog
		WHERE title_slug = ? AND is_active = TRUE AND playlist_asset_id IN (%s)
		ORDER BY name_canonical
	`, Placeholders(len(ids)))
	args := make([]any, 0, 1+len(ids))
	args = append(args, titleSlug)
	args = append(args, ToAnySlice(ids)...)
	rows, err := r.metadataDB.QueryRecovered(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query metadata playlists_catalog: %w", err)
	}
	defer rows.Close()
	var out []domain.CatalogPlaylist
	for rows.Next() {
		var p domain.CatalogPlaylist
		if err := rows.Scan(&p.TitleSlug, &p.PlaylistAssetID, &p.CurrentVersionID, &p.Name, &p.Experience, &p.IsRanked); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		p.MatchCount = counts[p.PlaylistAssetID]
		out = append(out, p)
	}
	return out, rows.Err()
}

// PairsByPlaylist retourne les pairs liés à une playlist via playlist_pair_links.
func (r *CatalogRepo) PairsByPlaylist(ctx context.Context, titleSlug, playlistAssetID string) ([]domain.CatalogPair, error) {
	if r.metadataDB == nil {
		return nil, fmt.Errorf("CatalogRepo: metadataDB nil")
	}
	rows, err := r.metadataDB.QueryRecovered(ctx, `
		SELECT mmp.title_slug, mmp.pair_asset_id, COALESCE(mmp.name_canonical, ''),
		       COALESCE(mmp.map_asset_id, ''), COALESCE(mmp.game_variant_asset_id, ''),
		       COALESCE(mmp.mode_category, ''), COALESCE(ppl.weight, 0.0)
		FROM map_mode_pair_definitions mmp
		LEFT JOIN playlist_pair_links ppl
		  ON ppl.title_slug = mmp.title_slug
		 AND ppl.pair_asset_id = mmp.pair_asset_id
		 AND ppl.playlist_asset_id = ?
		WHERE mmp.title_slug = ?
		  AND (ppl.playlist_asset_id = ? OR ? = '')
		ORDER BY mmp.name_canonical
	`, playlistAssetID, titleSlug, playlistAssetID, playlistAssetID)
	if err != nil {
		return nil, fmt.Errorf("query pairs: %w", err)
	}
	defer rows.Close()
	var out []domain.CatalogPair
	for rows.Next() {
		var p domain.CatalogPair
		if err := rows.Scan(&p.TitleSlug, &p.PairAssetID, &p.Name, &p.MapAssetID, &p.GameVariantAssetID, &p.ModeCategory, &p.Weight); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// MapsByTitle retourne les maps du catalogue.
func (r *CatalogRepo) MapsByTitle(ctx context.Context, titleSlug, xuid string, onlyPlayed bool) ([]domain.CatalogMap, error) {
	if r.metadataDB == nil {
		return nil, fmt.Errorf("CatalogRepo: metadataDB nil")
	}
	// Variante onlyPlayed = JOIN match_registry sur map_id, similaire à playlistsPlayedByXUID.
	// Pour ce sprint, version simple : retourne le catalogue complet ; le filtre onlyPlayed
	// peut être ajouté en Phase I quand FiltersService consommera ce repo.
	rows, err := r.metadataDB.QueryRecovered(ctx, `
		SELECT title_slug, map_asset_id, COALESCE(name_canonical, ''), COALESCE(image_url, '')
		FROM maps_catalog
		WHERE title_slug = ?
		ORDER BY name_canonical
	`, titleSlug)
	if err != nil {
		return nil, fmt.Errorf("query maps: %w", err)
	}
	defer rows.Close()
	var out []domain.CatalogMap
	for rows.Next() {
		var m domain.CatalogMap
		if err := rows.Scan(&m.TitleSlug, &m.MapAssetID, &m.Name, &m.ImageURL); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// CountCatalogEntries retourne le nombre de playlists actives.
// Utilisé comme guard de fallback en Phase I.
func (r *CatalogRepo) CountCatalogEntries(ctx context.Context, titleSlug string) (int, error) {
	if r.metadataDB == nil {
		return 0, nil
	}
	// QueryRecovered (et non QueryRowContext) pour bénéficier de la recovery FATAL ;
	// pas de QueryRowRecovered → scan manuel de la première row.
	rows, err := r.metadataDB.QueryRecovered(ctx,
		`SELECT COUNT(*) FROM playlists_catalog WHERE title_slug = ? AND is_active = TRUE`,
		titleSlug,
	)
	if err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	defer rows.Close()
	var n int
	if rows.Next() {
		if err := rows.Scan(&n); err != nil {
			return 0, fmt.Errorf("count scan: %w", err)
		}
	}
	return n, rows.Err()
}
