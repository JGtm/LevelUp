// Package duckdb — catalog_repo.go : Phase F du plan PLAN_PLAYLISTS_CATALOG.md.
//
// Implémente port.CatalogRepo en lecture sur metadata.duckdb (catalogue
// Playlists/Pairs/Maps title-aware). Utilisé par FiltersService (Phase I) et
// les endpoints catalogue (Phase H).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/domain"
)

// CatalogRepo implémente port.CatalogRepo.
type CatalogRepo struct {
	metadataDB *sql.DB
	sharedDB   *sql.DB // pour le JOIN match_participants quand onlyPlayed = true
}

// NewCatalogRepo construit le repo avec les 2 connexions DuckDB.
// sharedDB peut être nil si on ne supporte pas onlyPlayed (le catalogue complet
// reste accessible).
func NewCatalogRepo(metadataDB, sharedDB *sql.DB) *CatalogRepo {
	return &CatalogRepo{metadataDB: metadataDB, sharedDB: sharedDB}
}

// PlaylistsByTitle retourne les playlists du catalogue pour un titre donné.
func (r *CatalogRepo) PlaylistsByTitle(ctx context.Context, titleSlug, xuid string, onlyPlayed bool) ([]domain.CatalogPlaylist, error) {
	if r.metadataDB == nil {
		return nil, fmt.Errorf("CatalogRepo: metadataDB nil")
	}
	if onlyPlayed && r.sharedDB != nil && xuid != "" {
		return r.playlistsPlayedByXUID(ctx, titleSlug, xuid)
	}

	rows, err := r.metadataDB.QueryContext(ctx, `
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
// Cross-DB JOIN entre metadata.playlists_catalog et shared.match_registry / match_participants.
func (r *CatalogRepo) playlistsPlayedByXUID(ctx context.Context, titleSlug, xuid string) ([]domain.CatalogPlaylist, error) {
	// La requête suppose que metadataDB et sharedDB sont sur la même
	// instance DuckDB ATTACHée (métadonnées en main, shared en attached).
	// En pratique, l'application monte les 2 schémas dans la même connexion.
	// Pour une isolation propre on pourrait faire 2 requêtes + JOIN en Go,
	// implémentation plus simple en attendant — TODO Phase I si besoin.
	rows, err := r.metadataDB.QueryContext(ctx, `
		SELECT pc.title_slug, pc.playlist_asset_id, COALESCE(pc.current_version_id, ''),
		       COALESCE(pc.name_canonical, ''), COALESCE(pc.experience, ''), COALESCE(pc.is_ranked, FALSE),
		       COUNT(DISTINCT mp.match_id) AS match_count
		FROM playlists_catalog pc
		JOIN shared.match_registry mr ON mr.playlist_id = pc.playlist_asset_id
		JOIN shared.match_participants mp ON mp.match_id = mr.match_id
		WHERE pc.title_slug = ? AND pc.is_active = TRUE AND mp.xuid = ?
		GROUP BY pc.title_slug, pc.playlist_asset_id, pc.current_version_id, pc.name_canonical, pc.experience, pc.is_ranked
		HAVING COUNT(DISTINCT mp.match_id) > 0
		ORDER BY pc.name_canonical
	`, titleSlug, xuid)
	if err != nil {
		// Fallback : si l'ATTACH shared n'est pas en place, retomber sur le catalogue complet.
		slog.WarnContext(ctx, "CatalogRepo.PlaylistsByTitle: cross-DB JOIN failed, falling back to full catalog",
			"err", err, "title_slug", titleSlug)
		return r.PlaylistsByTitle(ctx, titleSlug, "", false)
	}
	defer rows.Close()
	var out []domain.CatalogPlaylist
	for rows.Next() {
		var p domain.CatalogPlaylist
		if err := rows.Scan(&p.TitleSlug, &p.PlaylistAssetID, &p.CurrentVersionID, &p.Name, &p.Experience, &p.IsRanked, &p.MatchCount); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// PairsByPlaylist retourne les pairs liés à une playlist via playlist_pair_links.
func (r *CatalogRepo) PairsByPlaylist(ctx context.Context, titleSlug, playlistAssetID string) ([]domain.CatalogPair, error) {
	if r.metadataDB == nil {
		return nil, fmt.Errorf("CatalogRepo: metadataDB nil")
	}
	rows, err := r.metadataDB.QueryContext(ctx, `
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
	rows, err := r.metadataDB.QueryContext(ctx, `
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
	var n int
	err := r.metadataDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM playlists_catalog WHERE title_slug = ? AND is_active = TRUE`,
		titleSlug,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	return n, nil
}
