// Package duckdb — enrichissement FR best-effort des MatchHistoryRawRow.
//
// Reproduit le mécanisme de filters_repo.go (applyXxxFRTranslations) adapté au
// type MatchHistoryRawRow. Pour les matchs où match_registry n'a pas de
// *_name_fr peuplé, on enrichit à la volée depuis metadata.duckdb
// (mode_name_tr + asset_translations).
//
// Best-effort : erreurs silencieuses pour ne pas bloquer le rendu UI.
package duckdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// applyMatchHistoryFRTranslations enrichit les 3 dimensions (mode, map, playlist)
// pour les MatchHistoryRawRow. Appelé depuis MatchHistoryRepo.LoadAll après le scan.
func applyMatchHistoryFRTranslations(ctx context.Context, pdb *PlayerDB, rows []domain.MatchHistoryRawRow) {
	applyMatchHistoryModeFR(ctx, pdb, rows)
	applyMatchHistoryMapFR(ctx, pdb, rows)
	applyMatchHistoryPlaylistFR(ctx, pdb, rows)
}

// applyMatchHistoryModeFR : mêmes règles que FiltersRepo.applyModeFRTranslations
// — enrichit PairNameFR depuis mode_name_tr (metadata.duckdb).
func applyMatchHistoryModeFR(ctx context.Context, pdb *PlayerDB, rows []domain.MatchHistoryRawRow) {
	if pdb.Metadata == nil {
		return
	}
	uniqueEN := make(map[string]struct{}, 32)
	for _, row := range rows {
		if en := analysis.NormalizeModeLabel(derefString(row.PairName)); en != "" {
			uniqueEN[en] = struct{}{}
		}
	}
	if len(uniqueEN) == 0 {
		return
	}
	enNames := make([]string, 0, len(uniqueEN))
	for en := range uniqueEN {
		enNames = append(enNames, en)
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(enNames)), ",")
	q := fmt.Sprintf(`SELECT mode_en, name FROM mode_name_tr WHERE lang = 'fr' AND mode_en IN (%s)`, placeholders)
	args := make([]any, len(enNames))
	for i, n := range enNames {
		args[i] = n
	}
	ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	metaRows, err := pdb.Metadata.Query(ctx2, q, args...)
	if err != nil {
		return
	}
	defer metaRows.Close()

	tr := make(map[string]string, len(enNames))
	for metaRows.Next() {
		var modeEN, nameFR string
		if metaRows.Scan(&modeEN, &nameFR) == nil {
			tr[modeEN] = nameFR
		}
	}
	if len(tr) == 0 {
		return
	}
	for i := range rows {
		en := analysis.NormalizeModeLabel(derefString(rows[i].PairName))
		if fr, ok := tr[en]; ok {
			rows[i].PairNameFR = &fr
		}
	}
}

// applyMatchHistoryMapFR : enrichit MapNameFR via map_id (match_registry) +
// asset_translations (metadata.duckdb), quand map_name_fr est absent.
func applyMatchHistoryMapFR(ctx context.Context, pdb *PlayerDB, rows []domain.MatchHistoryRawRow) {
	if pdb.Metadata == nil {
		return
	}
	// Détection des rows à enrichir : MapNameFR == MapName (= COALESCE fallback).
	uniqueEN := make(map[string]struct{}, 16)
	for _, row := range rows {
		en := derefString(row.MapName)
		if en != "" && en == derefString(row.MapNameFR) {
			uniqueEN[en] = struct{}{}
		}
	}
	if len(uniqueEN) == 0 {
		return
	}
	mapNames := make([]string, 0, len(uniqueEN))
	for n := range uniqueEN {
		mapNames = append(mapNames, n)
	}
	ph := strings.TrimRight(strings.Repeat("?,", len(mapNames)), ",")
	q1 := fmt.Sprintf(`SELECT DISTINCT map_name, map_id FROM match_registry WHERE map_name IN (%s) AND map_id IS NOT NULL`, ph)
	args1 := make([]any, len(mapNames))
	for i, n := range mapNames {
		args1[i] = n
	}
	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	idRows, err := pdb.Shared.Query(ctx2, q1, args1...)
	if err != nil {
		return
	}
	defer idRows.Close()

	nameToID := make(map[string]string, len(mapNames))
	for idRows.Next() {
		var name, id string
		if idRows.Scan(&name, &id) == nil && name != "" && id != "" {
			nameToID[name] = id
		}
	}
	idRows.Close()
	if len(nameToID) == 0 {
		return
	}
	mapIDs := make([]string, 0, len(nameToID))
	for _, id := range nameToID {
		mapIDs = append(mapIDs, id)
	}
	ph2 := strings.TrimRight(strings.Repeat("?,", len(mapIDs)), ",")
	q2 := fmt.Sprintf(`SELECT asset_id, name FROM asset_translations WHERE asset_type = 'map' AND lang IN ('fr-FR', 'fr') AND asset_id IN (%s) ORDER BY asset_id, CASE WHEN lang = 'fr-FR' THEN 0 ELSE 1 END`, ph2)
	args2 := make([]any, len(mapIDs))
	for i, id := range mapIDs {
		args2[i] = id
	}
	trRows, err := pdb.Metadata.Query(ctx2, q2, args2...)
	if err != nil {
		return
	}
	defer trRows.Close()

	idToFR := make(map[string]string, len(mapIDs))
	for trRows.Next() {
		var assetID, name string
		if trRows.Scan(&assetID, &name) == nil {
			if _, exists := idToFR[assetID]; !exists {
				idToFR[assetID] = name
			}
		}
	}
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

// applyMatchHistoryPlaylistFR : enrichit PlaylistName via playlist_id +
// asset_translations (metadata.duckdb), quand playlist_name_fr est absent.
// Détection : PlaylistName == PlaylistNameEN.
func applyMatchHistoryPlaylistFR(ctx context.Context, pdb *PlayerDB, rows []domain.MatchHistoryRawRow) {
	if pdb.Metadata == nil {
		return
	}
	uniqueEN := make(map[string]struct{}, 8)
	for _, row := range rows {
		en := derefString(row.PlaylistNameEN)
		if en != "" && en == derefString(row.PlaylistName) {
			uniqueEN[en] = struct{}{}
		}
	}
	if len(uniqueEN) == 0 {
		return
	}
	names := make([]string, 0, len(uniqueEN))
	for n := range uniqueEN {
		names = append(names, n)
	}
	ph := strings.TrimRight(strings.Repeat("?,", len(names)), ",")
	q1 := fmt.Sprintf(`SELECT DISTINCT playlist_name, playlist_id FROM match_registry WHERE playlist_name IN (%s) AND playlist_id IS NOT NULL`, ph)
	args1 := make([]any, len(names))
	for i, n := range names {
		args1[i] = n
	}
	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	idRows, err := pdb.Shared.Query(ctx2, q1, args1...)
	if err != nil {
		return
	}
	defer idRows.Close()

	nameToID := make(map[string]string, len(names))
	for idRows.Next() {
		var name, id string
		if idRows.Scan(&name, &id) == nil && name != "" && id != "" {
			nameToID[name] = id
		}
	}
	idRows.Close()
	if len(nameToID) == 0 {
		return
	}
	plIDs := make([]string, 0, len(nameToID))
	for _, id := range nameToID {
		plIDs = append(plIDs, id)
	}
	ph2 := strings.TrimRight(strings.Repeat("?,", len(plIDs)), ",")
	q2 := fmt.Sprintf(`SELECT asset_id, name FROM asset_translations WHERE asset_type = 'playlist' AND lang IN ('fr-FR', 'fr') AND asset_id IN (%s) ORDER BY asset_id, CASE WHEN lang = 'fr-FR' THEN 0 ELSE 1 END`, ph2)
	args2 := make([]any, len(plIDs))
	for i, id := range plIDs {
		args2[i] = id
	}
	trRows, err := pdb.Metadata.Query(ctx2, q2, args2...)
	if err != nil {
		return
	}
	defer trRows.Close()

	idToFR := make(map[string]string, len(plIDs))
	for trRows.Next() {
		var assetID, name string
		if trRows.Scan(&assetID, &name) == nil {
			if _, exists := idToFR[assetID]; !exists {
				idToFR[assetID] = name
			}
		}
	}
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
