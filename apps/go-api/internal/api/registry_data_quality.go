// Package api — registry_data_quality.go : runners qualité données du
// dashboard monitoring (compteurs + listes d'inconnus).
//
// Handles : shared en RO via le cache duckdb (pattern openDBShared de
// data_health_check — clé "ro:path" partagée avec main.go, Close décrémente
// le ref-count) ; metadata via OpenReadWriteShared (handle process partagé
// avec les catalogs/prestige, jamais un second sql.Open concurrent).
package api

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/duckdb"
)

// dataQualityHandles ouvre shared (RO) + metadata (RW partagé). closeAll
// décrémente les ref-counts — à defer par le caller. metadata absente →
// metaSQL nil (les fonctions ops dégradent explicitement).
func (r *ServiceRegistry) dataQualityHandles(titleSlug string) (sharedSQL, metaSQL *sql.DB, closeAll func(), err error) {
	pr := titlePkg.NewPathResolver(r.cfg.RepoRoot)
	sharedPath := pr.SharedDBPath(titleSlug)
	if _, statErr := os.Stat(sharedPath); statErr != nil {
		return nil, nil, nil, fmt.Errorf("shared DB absente pour %s: %w", titleSlug, statErr)
	}
	shared, err := duckdb.OpenReadOnly(sharedPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open shared RO: %w", err)
	}

	var meta *duckdb.DB
	metaPath := pr.MetadataDBPath(titleSlug)
	if _, statErr := os.Stat(metaPath); statErr == nil {
		if m, openErr := duckdb.OpenReadWriteShared(metaPath); openErr == nil {
			meta = m
		} else {
			monitoringLog.WarnContext(context.Background(), "data_quality: metadata indisponible",
				"title", titleSlug, "err", openErr)
		}
	}

	closeAll = func() {
		_ = shared.Close() //nolint:errcheck // ref-count
		if meta != nil {
			_ = meta.Close() //nolint:errcheck // ref-count
		}
	}
	if meta != nil {
		metaSQL = meta.SQLDb()
	}
	return shared.SQLDb(), metaSQL, closeAll, nil
}

// DataQualityCounts calcule les compteurs d'inconnus (lectures seules).
func (r *ServiceRegistry) DataQualityCounts(ctx context.Context, titleSlug string) (domain.AdminDataQualityCounts, error) {
	resp := domain.AdminDataQualityCounts{
		TitleSlug:   titleSlug,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}
	sharedSQL, metaSQL, closeAll, err := r.dataQualityHandles(titleSlug)
	if err != nil {
		return resp, err
	}
	defer closeAll()

	counts, err := ops.CountDataQuality(ctx, sharedSQL, metaSQL, titleSlug)
	if err != nil {
		return resp, err
	}
	resp.RawUUIDPlaylists = counts.RawUUIDPlaylists
	resp.RawUUIDMaps = counts.RawUUIDMaps
	resp.RawUUIDPairs = counts.RawUUIDPairs
	resp.RawUUIDVariants = counts.RawUUIDVariants
	resp.RawUUIDTotal = counts.RawUUIDTotal()
	resp.UntranslatedModes = counts.UntranslatedModes
	resp.OrphanPlaylists = counts.OrphanPlaylists
	resp.OrphanXUIDs = counts.OrphanXUIDs
	resp.LyingBitsEvents = counts.LyingBitsEvents
	resp.LyingBitsWeapons = counts.LyingBitsWeapons

	// Gauges expvar (ADR 0009) : observables sans ouvrir le dashboard.
	observability.SetInt("data_quality_raw_uuids_last", int64(resp.RawUUIDTotal))
	observability.SetInt("data_quality_untranslated_modes_last", int64(resp.UntranslatedModes))
	observability.SetInt("data_quality_orphan_playlists_last", int64(resp.OrphanPlaylists))
	observability.SetInt("data_quality_orphan_xuids_last", int64(resp.OrphanXUIDs))
	return resp, nil
}

// DataQualityIssues liste les inconnus d'un kind donné.
func (r *ServiceRegistry) DataQualityIssues(ctx context.Context, titleSlug, kind string, limit int) (domain.AdminDataQualityIssues, error) {
	resp := domain.AdminDataQualityIssues{
		TitleSlug:   titleSlug,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Kind:        kind,
		Items:       []domain.AdminDataQualityIssue{},
	}
	sharedSQL, metaSQL, closeAll, err := r.dataQualityHandles(titleSlug)
	if err != nil {
		return resp, err
	}
	defer closeAll()

	items, err := ops.ListDataQualityIssues(ctx, sharedSQL, metaSQL, titleSlug, kind, limit)
	if err != nil {
		return resp, err
	}
	for _, it := range items {
		resp.Items = append(resp.Items, domain.AdminDataQualityIssue{
			Kind:        it.Kind,
			AssetKind:   it.AssetKind,
			ID:          it.ID,
			Label:       it.Label,
			Occurrences: it.Occurrences,
			LastSeen:    it.LastSeen,
		})
	}
	return resp, nil
}
