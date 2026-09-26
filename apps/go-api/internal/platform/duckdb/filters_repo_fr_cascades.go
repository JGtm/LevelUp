// Package duckdb — filters_repo_fr_cascades.go : cascades FR historiques des noms de
// carte et de sélection du pipeline de filtres (applyMapFRTranslations,
// applyPlaylistFRTranslations) et leurs deux lectures (match_registry puis
// asset_translations).
//
// Extrait de filters_repo_asset_names.go sans changement de comportement (lot perf L9-go,
// 2026-09-23) : les échecs de ces lectures, jusque-là avalés en silence, sont désormais
// journalisés et consignés (bestEffortFailed) — les lignes d'une résolution dégradée ne
// sont jamais mises en cache (player_read_cache.go) — et le fichier d'origine aurait
// dépassé 500 lignes.
package duckdb

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
)

// applyMapFRTranslations enrichit MapNameFR quand map_name_fr est absent de match_registry
// (MapNameFR == MapName = COALESCE fallback EN). Interroge match_registry pour les map_id
// puis asset_translations pour les noms FR. Best-effort : échec via bestEffortFailed.
func (r *FiltersRepo) applyMapFRTranslations(ctx context.Context, rows []domain.FilterMatchRow) {
	if r.pdb.Metadata == nil {
		return
	}
	uniqueEN := r.collectMapENNeedingFR(rows)
	if len(uniqueEN) == 0 {
		return
	}

	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	nameToID, ok := r.resolveSharedAssetNameToID(ctx2, uniqueEN, "map_name", "map_id")
	if !ok || len(nameToID) == 0 {
		return
	}

	idToFR := r.loadAssetFRTranslations(ctx2, nameToID, "map")
	if len(idToFR) == 0 {
		return
	}

	for i := range rows {
		en := derefString(rows[i].MapName)
		if en == "" || en != derefString(rows[i].MapNameFR) {
			continue
		}
		if mapID, ok := nameToID[en]; ok {
			if fr, ok2 := idToFR[mapID]; ok2 && fr != "" {
				rows[i].MapNameFR = &fr
			}
		}
	}
}

// collectMapENNeedingFR retourne les noms EN distincts pour lesquels la traduction FR
// est manquante (MapNameFR == MapName : fallback COALESCE).
func (r *FiltersRepo) collectMapENNeedingFR(rows []domain.FilterMatchRow) map[string]struct{} {
	uniqueEN := make(map[string]struct{}, 16)
	for _, row := range rows {
		en := derefString(row.MapName)
		if en != "" && en == derefString(row.MapNameFR) {
			uniqueEN[en] = struct{}{}
		}
	}
	return uniqueEN
}

// resolveSharedAssetNameToID résout en map (name → id) depuis shared.match_registry pour
// un set de noms EN (FR translations map, playlist). Échec : bestEffortFailed, ok=false.
func (r *FiltersRepo) resolveSharedAssetNameToID(
	ctx context.Context,
	uniqueEN map[string]struct{},
	nameCol, idCol string,
) (map[string]string, bool) {
	names := make([]string, 0, len(uniqueEN))
	for n := range uniqueEN {
		names = append(names, n)
	}
	ph := Placeholders(len(names))
	q := fmt.Sprintf(
		`SELECT DISTINCT %s, %s FROM match_registry WHERE %s IN (%s) AND %s IS NOT NULL`,
		nameCol, idCol, nameCol, ph, idCol,
	)
	args := make([]any, len(names))
	for i, n := range names {
		args[i] = n
	}

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		bestEffortFailed(ctx, "asset_name_ids", err)
		return nil, false
	}
	defer release()

	idRows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		bestEffortFailed(ctx, "asset_name_ids", err)
		return nil, false
	}
	defer idRows.Close()

	nameToID := make(map[string]string, len(names))
	for idRows.Next() {
		var name, id string
		if idRows.Scan(&name, &id) == nil && name != "" && id != "" {
			nameToID[name] = id
		}
	}
	if err := idRows.Err(); err != nil {
		bestEffortFailed(ctx, "asset_name_ids", err)
		return nil, false
	}
	return nameToID, true
}

// loadAssetFRTranslations charge les traductions FR (fr-FR > fr) depuis metadata.asset_translations
// pour un set d'IDs résolus, en priorisant fr-FR > fr.
func (r *FiltersRepo) loadAssetFRTranslations(
	ctx context.Context,
	nameToID map[string]string,
	assetType string,
) map[string]string {
	ids := make([]string, 0, len(nameToID))
	for _, id := range nameToID {
		ids = append(ids, id)
	}
	ph := Placeholders(len(ids))
	q := fmt.Sprintf(
		`SELECT asset_id, name FROM asset_translations WHERE asset_type = ? AND lang IN ('fr-FR', 'fr') AND asset_id IN (%s) ORDER BY asset_id, CASE WHEN lang = 'fr-FR' THEN 0 ELSE 1 END`,
		ph,
	)
	args := make([]any, 0, len(ids)+1)
	args = append(args, assetType)
	for _, id := range ids {
		args = append(args, id)
	}

	// QueryRecovered : auto-réparation si handle metadata FATAL-invalidated (bug ART).
	trRows, err := r.pdb.Metadata.QueryRecovered(ctx, q, args...)
	if err != nil {
		bestEffortFailed(ctx, "asset_fr_translations", err)
		return nil
	}
	defer trRows.Close()

	idToFR := make(map[string]string, len(ids))
	for trRows.Next() {
		var assetID, name string
		if trRows.Scan(&assetID, &name) == nil {
			if _, exists := idToFR[assetID]; !exists {
				idToFR[assetID] = name
			}
		}
	}
	if err := trRows.Err(); err != nil {
		bestEffortFailed(ctx, "asset_fr_translations", err)
	}
	return idToFR
}

// applyPlaylistFRTranslations enrichit PlaylistName quand playlist_name_fr est absent de
// match_registry (PlaylistName == PlaylistNameEN = COALESCE fallback EN).
// Best-effort : échec via bestEffortFailed.
func (r *FiltersRepo) applyPlaylistFRTranslations(ctx context.Context, rows []domain.FilterMatchRow) {
	if r.pdb.Metadata == nil {
		return
	}
	uniqueEN := r.collectPlaylistENNeedingFR(rows)
	if len(uniqueEN) == 0 {
		return
	}

	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	nameToID, ok := r.resolveSharedAssetNameToID(ctx2, uniqueEN, "playlist_name", "playlist_id")
	if !ok || len(nameToID) == 0 {
		return
	}

	idToFR := r.loadAssetFRTranslations(ctx2, nameToID, "playlist")
	if len(idToFR) == 0 {
		return
	}

	for i := range rows {
		en := derefString(rows[i].PlaylistNameEN)
		if en == "" || en != derefString(rows[i].PlaylistName) {
			continue
		}
		if plID, ok := nameToID[en]; ok {
			if fr, ok2 := idToFR[plID]; ok2 && fr != "" {
				rows[i].PlaylistName = &fr
			}
		}
	}
}

// collectPlaylistENNeedingFR retourne les playlist names EN distincts dont la traduction FR
// est absente de match_registry.
func (r *FiltersRepo) collectPlaylistENNeedingFR(rows []domain.FilterMatchRow) map[string]struct{} {
	uniqueEN := make(map[string]struct{}, 8)
	for _, row := range rows {
		en := derefString(row.PlaylistNameEN)
		if en != "" && en == derefString(row.PlaylistName) {
			uniqueEN[en] = struct{}{}
		}
	}
	return uniqueEN
}
