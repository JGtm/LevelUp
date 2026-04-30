// Package duckdb — FiltersRepo : résolution du contexte de filtres.
package duckdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// FiltersRepo implémente port.FiltersRepository.
type FiltersRepo struct {
	pdb *PlayerDB
}

// NewFiltersRepo crée un FiltersRepo depuis un PlayerDB ouvert.
func NewFiltersRepo(pdb *PlayerDB) *FiltersRepo {
	return &FiltersRepo{pdb: pdb}
}

// LoadMatchesForFilters charge tous les matchs du joueur pour la résolution cascade.
// Utilise mv_player_matches si disponible, sinon fallback sur match_registry.
func (r *FiltersRepo) LoadMatchesForFilters(ctx context.Context) ([]domain.FilterMatchRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	hasMV := r.hasMVPlayerMatches(ctx)
	var query string
	if hasMV {
		query = Q4MVMatchesForFilters
	} else {
		query = Q4MatchesForFilters
	}

	rows, err := r.pdb.ReadDB().Query(ctx, query, r.pdb.XUID)
	if err != nil {
		return nil, fmt.Errorf("FiltersRepo.LoadMatchesForFilters: %w", err)
	}
	defer rows.Close()

	var results []domain.FilterMatchRow
	for rows.Next() {
		var m domain.FilterMatchRow
		if err := rows.Scan(
			&m.MatchID,
			&m.StartTime,
			&m.MapName,
			&m.MapNameFR,
			&m.PairName,
			&m.PairNameFR,
			&m.PlaylistName,
			&m.IsFirefight,
			&m.IsRanked,
			&m.SessionID,
			&m.SessionLabel,
			&m.IsWithFriends,
			&m.PlaylistNameEN,
		); err != nil {
			return nil, fmt.Errorf("FiltersRepo.LoadMatchesForFilters scan: %w", err)
		}
		results = append(results, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	r.applyModeFRTranslations(ctx, results)
	r.applyMapFRTranslations(ctx, results)
	r.applyPlaylistFRTranslations(ctx, results)
	return results, nil
}

// GetMatchCount retourne le nombre total de matchs dans shared_matches_v2.
func (r *FiltersRepo) GetMatchCount(ctx context.Context) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var count int
	err := r.pdb.Shared.QueryRow(ctx, Q1MatchCount).Scan(&count)
	return count, err
}

// GetPlayerMatchCount retourne le nombre de matchs du joueur dans shared.
func (r *FiltersRepo) GetPlayerMatchCount(ctx context.Context) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var count int
	q := `SELECT COUNT(*) FROM shared.match_participants WHERE xuid = ?`
	err := r.pdb.Shared.QueryRow(ctx, q, r.pdb.XUID).Scan(&count)
	return count, err
}

// hasMVPlayerMatches vérifie si la vue matérialisée shared.mv_player_matches existe.
func (r *FiltersRepo) hasMVPlayerMatches(ctx context.Context) bool {
	ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rows, err := r.pdb.ReadDB().Query(ctx2, "SELECT 1 FROM shared.mv_player_matches LIMIT 0")
	if err != nil {
		return false
	}
	rows.Close()
	return true
}

// applyModeFRTranslations enrichit PairNameFR dans les rows en interrogeant
// mode_name_tr depuis metadata.duckdb (même logique que home_repo).
// Opération best-effort : les erreurs sont silencieuses pour ne pas bloquer
// la résolution des filtres.
func (r *FiltersRepo) applyModeFRTranslations(ctx context.Context, rows []domain.FilterMatchRow) {
	uniqueEN := make(map[string]struct{}, 32)
	for _, row := range rows {
		if en := analysis.NormalizeModeLabel(derefString(row.PairName)); en != "" {
			uniqueEN[en] = struct{}{}
		}
	}
	if len(uniqueEN) == 0 || r.pdb.Metadata == nil {
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
	metaRows, err := r.pdb.Metadata.Query(ctx2, q, args...)
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

// applyMapFRTranslations enrichit MapNameFR quand map_name_fr est absent de match_registry
// (MapNameFR == MapName = COALESCE fallback EN). Interroge match_registry pour les map_id
// puis asset_translations pour les noms FR. Best-effort : erreurs silencieuses.
func (r *FiltersRepo) applyMapFRTranslations(ctx context.Context, rows []domain.FilterMatchRow) {
	if r.pdb.Metadata == nil {
		return
	}
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

	idRows, err := r.pdb.Shared.Query(ctx2, q1, args1...)
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

	trRows, err := r.pdb.Metadata.Query(ctx2, q2, args2...)
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

// applyPlaylistFRTranslations enrichit PlaylistName quand playlist_name_fr est absent de
// match_registry (PlaylistName == PlaylistNameEN = COALESCE fallback EN).
// Best-effort : erreurs silencieuses.
func (r *FiltersRepo) applyPlaylistFRTranslations(ctx context.Context, rows []domain.FilterMatchRow) {
	if r.pdb.Metadata == nil {
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

	idRows, err := r.pdb.Shared.Query(ctx2, q1, args1...)
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

	trRows, err := r.pdb.Metadata.Query(ctx2, q2, args2...)
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

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// GetAvailablePlaylists retourne les playlists uniques du joueur.
func (r *FiltersRepo) GetAvailablePlaylists(ctx context.Context) ([]domain.LabelValue, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	q := `
	SELECT DISTINCT
	    COALESCE(r.playlist_name_fr, r.playlist_name, '') AS label,
	    COALESCE(r.playlist_name, '')                     AS value
	FROM shared.match_registry r
	JOIN shared.match_participants p ON r.match_id = p.match_id
	WHERE p.xuid = ?
	  AND r.playlist_name IS NOT NULL
	  AND r.playlist_name != ''
	ORDER BY label ASC`

	rows, err := r.pdb.Shared.Query(ctx, q, r.pdb.XUID)
	if err != nil {
		return nil, fmt.Errorf("GetAvailablePlaylists: %w", err)
	}
	defer rows.Close()

	var results []domain.LabelValue
	for rows.Next() {
		var lv domain.LabelValue
		if err := rows.Scan(&lv.Label, &lv.Value); err != nil {
			return nil, err
		}
		results = append(results, lv)
	}
	return results, rows.Err()
}

// GetAvailableMaps retourne les cartes uniques jouées.
func (r *FiltersRepo) GetAvailableMaps(ctx context.Context) ([]domain.LabelValue, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	q := `
	SELECT DISTINCT
	    COALESCE(r.map_name_fr, r.map_name, '') AS label,
	    COALESCE(r.map_name, '')                AS value
	FROM shared.match_registry r
	JOIN shared.match_participants p ON r.match_id = p.match_id
	WHERE p.xuid = ?
	  AND r.map_name IS NOT NULL
	ORDER BY label ASC`

	rows, err := r.pdb.Shared.Query(ctx, q, r.pdb.XUID)
	if err != nil {
		return nil, fmt.Errorf("GetAvailableMaps: %w", err)
	}
	defer rows.Close()

	var results []domain.LabelValue
	for rows.Next() {
		var lv domain.LabelValue
		if err := rows.Scan(&lv.Label, &lv.Value); err != nil {
			return nil, err
		}
		results = append(results, lv)
	}
	return results, rows.Err()
}
