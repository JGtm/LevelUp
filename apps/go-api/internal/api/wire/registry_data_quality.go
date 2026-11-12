// Package api — registry_data_quality.go : runners qualité données du
// dashboard monitoring (compteurs + listes d'inconnus).
//
// Handles (K1e, B-swap-safe) : shared via le SharedProvider quand celui-ci tient CE
// fichier (titre par défaut — seul shared pris en RW en process) : le provider draine
// les fenêtres RW↔RO au lieu de forcer un OpenReadOnly qui échouerait en "different
// configuration" pendant un sync (5+ runners admin concurrents). Pour les autres titres
// (aucune fenêtre RW en process) OpenReadOnly reste correct. metadata via
// OpenReadWriteShared (handle process partagé avec les catalogs/prestige).
package wire

import (
	"context"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/duckdb"
)

// titleBases rend l'accès aux bases d'un titre (platform/duckdb), construit à l'appel.
func (r *ServiceRegistry) titleBases() duckdb.TitleBases {
	return duckdb.TitleBases{Paths: r.cfg.PathResolver(), SharedProvider: r.cfg.SharedProvider, Log: monitoringLog}
}

// dataQualityHandles ouvre shared (RO) + metadata (RW partagé). Close décrémente les ref-counts — à
// defer par le caller. metadata absente : Meta nil (les fonctions ops dégradent explicitement). Une tâche
// qui va sur le réseau prend les deux séparément (titleBases().ReadShared, .Metadata) et rend la base
// partagée avant : un lecteur en vol fait échouer toute prise d'écriture (ADR 0016).
func (r *ServiceRegistry) dataQualityHandles(ctx context.Context, titleSlug string) (duckdb.TitleHandles, error) {
	return r.titleBases().Open(ctx, titleSlug)
}

// DataQualityCounts calcule les compteurs d'inconnus (lectures seules). locale =
// langue cible du compteur untranslated_modes (échotée dans la réponse).
func (r *ServiceRegistry) DataQualityCounts(ctx context.Context, titleSlug, locale string) (domain.AdminDataQualityCounts, error) {
	resp := domain.AdminDataQualityCounts{
		TitleSlug:   titleSlug,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Locale:      locale,
	}
	h, err := r.dataQualityHandles(ctx, titleSlug)
	if err != nil {
		return resp, err
	}
	defer h.Close()

	counts, err := ops.CountDataQuality(ctx, h.Shared, h.Meta, titleSlug, locale)
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

// DataQualityIssues liste les inconnus d'un kind donné (fenêtre paginée
// [offset, offset+limit) + total avant fenêtrage).
func (r *ServiceRegistry) DataQualityIssues(ctx context.Context, titleSlug, kind, locale string, limit, offset int) (domain.AdminDataQualityIssues, error) {
	resp := domain.AdminDataQualityIssues{
		TitleSlug:   titleSlug,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Kind:        kind,
		Locale:      locale,
		Items:       []domain.AdminDataQualityIssue{},
	}
	h, err := r.dataQualityHandles(ctx, titleSlug)
	if err != nil {
		return resp, err
	}
	defer h.Close()

	items, total, err := ops.ListDataQualityIssues(ctx, h.Shared, h.Meta, titleSlug, kind, locale, limit, offset)
	if err != nil {
		return resp, err
	}
	resp.Total = total
	for _, it := range items {
		resp.Items = append(resp.Items, domain.AdminDataQualityIssue{
			Kind:            it.Kind,
			AssetKind:       it.AssetKind,
			ID:              it.ID,
			Label:           it.Label,
			Occurrences:     it.Occurrences,
			LastSeen:        it.LastSeen,
			ExampleMatchIDs: it.ExampleMatchIDs,
		})
	}
	return resp, nil
}
