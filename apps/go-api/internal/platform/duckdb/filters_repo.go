// Package duckdb — FiltersRepo : résolution du contexte de filtres.
package duckdb

import (
	"context"
	"fmt"
	"time"

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

	// Masquage READ-SIDE des matchs Campagne (Halo 5) sur le chargement cascade
	// (options de filtres, comptes, scope sessions). No-op Infinite. L'alias de
	// résolution diffère selon la source : "r" pour v_match_full, le nom de la vue
	// pour mv_player_matches (colonnes non aliasées). Item backlog H1.
	hasMV := r.hasMVPlayerMatches(ctx)
	var sharedQuery string
	if hasMV {
		sharedQuery = resolveCampaignExclusion(Q4MVSharedMatchesForFilters, r.pdb.TitleSlug, "mv_player_matches")
	} else {
		sharedQuery = resolveCampaignExclusion(Q4SharedMatchesForFilters, r.pdb.TitleSlug, "r")
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

	// Résolution read-side ID->nom depuis metadata pour les titres sans noms
	// registry (Halo 5 : noms 100% NULL, ids remplis). No-op strict sur Infinite
	// (collecte vide -> zéro requête metadata). Précède les cascades FR, qui
	// deviennent des raffinements idempotents sur les noms nouvellement remplis.
	r.applyAssetNamesFromMetadata(ctx, results)

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
			&m.MapID,
			&m.PlaylistID,
			&m.GameVariantID,
			&m.GameVariantName,
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
