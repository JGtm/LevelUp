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
//
// split+merge cross-DB. La query historique unique
// (shared.v_match_full ⨝ shared.match_participants ⨝ player_match_enrichment)
// est découpée en 2 :
//  1. Partie shared (Q4SharedMatchesForFilters ou Q4MVSharedMatchesForFilters)
//     via SharedReader.Get → liste de matchs avec metadata.
//  2. Partie player (Q4PlayerEnrichmentForMatchesTpl) via pdb.Player → enrichments
//     pour les match_ids retournés en étape 1.
//  3. Merge en Go (LEFT JOIN semantics : enrichment manquant → defaults).
func (r *FiltersRepo) LoadMatchesForFilters(ctx context.Context) ([]domain.FilterMatchRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	hasMV := r.hasMVPlayerMatches(ctx)
	var sharedQuery string
	if hasMV {
		sharedQuery = Q4MVSharedMatchesForFilters
	} else {
		sharedQuery = Q4SharedMatchesForFilters
	}

	results, err := r.loadSharedFilterRows(ctx, sharedQuery)
	if err != nil {
		return nil, fmt.Errorf("FiltersRepo.LoadMatchesForFilters: %w", err)
	}
	if len(results) == 0 {
		return results, nil
	}

	if err := r.mergePlayerEnrichments(ctx, results); err != nil {
		return nil, fmt.Errorf("FiltersRepo.LoadMatchesForFilters: %w", err)
	}

	r.applyModeFRTranslations(ctx, results)
	r.applyMapFRTranslations(ctx, results)
	r.applyPlaylistFRTranslations(ctx, results)
	return results, nil
}

// loadSharedFilterRows exécute l'étape 1 du split LoadMatchesForFilters via
// SharedReader. Renvoie les rows sans enrichment (SessionID/Label/IsWithFriends
// non remplis).
func (r *FiltersRepo) loadSharedFilterRows(ctx context.Context, query string) ([]domain.FilterMatchRow, error) {
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, query, r.pdb.XUID)
	if err != nil {
		return nil, fmt.Errorf("shared query: %w", err)
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
			&m.PairID,
			&m.PlaylistName,
			&m.IsFirefight,
			&m.IsRanked,
			&m.PlaylistNameEN,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// mergePlayerEnrichments exécute l'étape 2 du split (player_match_enrichment)
// et applique la sémantique LEFT JOIN en Go : enrichment manquant pour un
// match_id → SessionID/Label/IsWithFriends restent à leurs valeurs zero.
func (r *FiltersRepo) mergePlayerEnrichments(ctx context.Context, rows []domain.FilterMatchRow) error {
	matchIDs := make([]string, 0, len(rows))
	for _, m := range rows {
		matchIDs = append(matchIDs, m.MatchID)
	}
	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	enrichments, err := LoadPlayerMatchEnrichments(ctx2, r.pdb.Player, matchIDs)
	if err != nil {
		return err
	}

	for i := range rows {
		e, ok := enrichments[rows[i].MatchID]
		if !ok {
			continue
		}
		if e.SessionID.Valid {
			s := e.SessionID.String
			rows[i].SessionID = &s
		}
		if e.SessionLabel.Valid {
			s := e.SessionLabel.String
			rows[i].SessionLabel = &s
		}
		rows[i].IsWithFriends = e.IsWithFriends
	}
	return nil
}

// GetMatchCount retourne le nombre total de matchs dans shared_matches_v2.
func (r *FiltersRepo) GetMatchCount(ctx context.Context) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return 0, fmt.Errorf("FiltersRepo.GetMatchCount: %w", err)
	}
	defer release()

	var count int
	q := resolveCampaignExclusion(Q1MatchCount, r.pdb.TitleSlug, "mr")
	err = db.QueryRowContext(ctx, q).Scan(&count)
	return count, err
}

// GetPlayerMatchCount retourne le nombre de matchs du joueur dans shared.
func (r *FiltersRepo) GetPlayerMatchCount(ctx context.Context) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return 0, fmt.Errorf("FiltersRepo.GetPlayerMatchCount: %w", err)
	}
	defer release()

	var count int
	q := `SELECT COUNT(*) FROM match_participants WHERE xuid = ?` +
		excludeCampaignByMatchID(r.pdb.TitleSlug, "match_id")
	err = db.QueryRowContext(ctx, q, r.pdb.XUID).Scan(&count)
	return count, err
}

// hasMVPlayerMatches vérifie si la vue matérialisée shared.mv_player_matches existe.
func (r *FiltersRepo) hasMVPlayerMatches(ctx context.Context) bool {
	ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx2)
	if err != nil {
		return false
	}
	defer release()

	rows, err := db.QueryContext(ctx2, "SELECT 1 FROM mv_player_matches LIMIT 0")
	if err != nil {
		return false
	}
	rows.Close()
	return true
}

// applyModeFRTranslations enrichit PairNameFR dans les rows via la cascade
// unifiée analysis.ResolvePairNameFR (mode_name_tr puis re-lookup via
// asset_translations puis raw fallback).
//
// Source unique de vérité partagée avec home_repo et match_history. Sans le
// re-lookup asset_translations, certains pair_id corrompus (asset_translations
// retournant l'EN raw "Arena:CTF on X" pour toutes les langues) feraient
// cohabiter "CTF" + "Capture du drapeau" dans available_modes (cf. thought_log
// 2026-05-09 root cause P2).
//
// Best-effort : les erreurs sont silencieuses pour ne pas bloquer la
// résolution des filtres.
func (r *FiltersRepo) applyModeFRTranslations(ctx context.Context, rows []domain.FilterMatchRow) {
	if r.pdb.Metadata == nil || len(rows) == 0 {
		return
	}

	// Étape 1 : collecter les pair_id distincts ET les mode_en normalisés
	// (depuis pair_name brut ET depuis l'asset name si déjà connu).
	pairIDs := collectDistinctPairIDsForFilters(rows)
	pairAssetNames := loadPairAssetNamesFR(ctx, r.pdb.Metadata, pairIDs)

	uniqueEN := make(map[string]struct{}, 32)
	for _, row := range rows {
		if en := analysis.NormalizeModeLabel(derefString(row.PairName)); en != "" {
			uniqueEN[en] = struct{}{}
		}
		if row.PairID != nil {
			if assetName := strings.TrimSpace(pairAssetNames[*row.PairID]); assetName != "" {
				if en := analysis.NormalizeModeLabel(assetName); en != "" {
					uniqueEN[en] = struct{}{}
				}
			}
		}
	}
	if len(uniqueEN) == 0 {
		return
	}

	enNames := make([]string, 0, len(uniqueEN))
	for en := range uniqueEN {
		enNames = append(enNames, en)
	}
	modeFR := loadModeNamesFRForKeys(ctx, r.pdb.Metadata, enNames)
	if len(modeFR) == 0 && len(pairAssetNames) == 0 {
		return
	}

	// Étape 2 : appliquer le helper canonique sur chaque row.
	for i := range rows {
		var assetName string
		if rows[i].PairID != nil {
			assetName = pairAssetNames[*rows[i].PairID]
		}
		if fr := analysis.ResolvePairNameFR(
			derefString(rows[i].PairName),
			derefString(rows[i].PairNameFR),
			assetName,
			modeFR,
		); fr != "" {
			rows[i].PairNameFR = &fr
		}
	}
}

// collectDistinctPairIDsForFilters extrait les pair_id uniques non-vides.
func collectDistinctPairIDsForFilters(rows []domain.FilterMatchRow) []string {
	seen := make(map[string]struct{}, len(rows))
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		if r.PairID == nil {
			continue
		}
		id := strings.TrimSpace(*r.PairID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// applyMapFRTranslations enrichit MapNameFR quand map_name_fr est absent de match_registry
// (MapNameFR == MapName = COALESCE fallback EN). Interroge match_registry pour les map_id
// puis asset_translations pour les noms FR. Best-effort : erreurs silencieuses.
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
// un set de noms EN. Helper générique pour FR translations (map, playlist).
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
		return nil, false
	}
	defer release()

	idRows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
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
	return idToFR
}

// applyPlaylistFRTranslations enrichit PlaylistName quand playlist_name_fr est absent de
// match_registry (PlaylistName == PlaylistNameEN = COALESCE fallback EN).
// Best-effort : erreurs silencieuses.
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
	FROM match_registry r
	JOIN match_participants p ON r.match_id = p.match_id
	WHERE p.xuid = ?` + excludeCampaignClause(r.pdb.TitleSlug, "r") + `
	  AND r.playlist_name IS NOT NULL
	  AND r.playlist_name != ''
	ORDER BY label ASC`

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetAvailablePlaylists: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, q, r.pdb.XUID)
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
	FROM match_registry r
	JOIN match_participants p ON r.match_id = p.match_id
	WHERE p.xuid = ?` + excludeCampaignClause(r.pdb.TitleSlug, "r") + `
	  AND r.map_name IS NOT NULL
	ORDER BY label ASC`

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetAvailableMaps: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, q, r.pdb.XUID)
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
