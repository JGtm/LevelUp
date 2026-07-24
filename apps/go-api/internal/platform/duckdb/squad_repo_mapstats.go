package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

func (r *SquadRepo) LoadMapStatsForSquad(ctx context.Context, mainXUID string, squadXUIDs, excludeXUIDs []string) (map[string]domain.MapSquadStats, error) {
	if mainXUID == "" || len(squadXUIDs) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Étape 1 : shared per-match rows.
	matchRows, err := r.loadMapStatsSquadShared(ctx, mainXUID, squadXUIDs, excludeXUIDs)
	if err != nil {
		return nil, fmt.Errorf("LoadMapStatsForSquad: %w", err)
	}
	if len(matchRows) == 0 {
		return map[string]domain.MapSquadStats{}, nil
	}

	// Étape 2 : performance_score depuis player_match_enrichment.
	matchIDs := make([]string, 0, len(matchRows))
	for _, m := range matchRows {
		matchIDs = append(matchIDs, m.matchID)
	}
	perfMap, err := r.loadMatchPerformanceScores(ctx, matchIDs)
	if err != nil {
		return nil, fmt.Errorf("LoadMapStatsForSquad: %w", err)
	}

	// Étape 3 : aggregation par map_id en Go.
	type mapAgg struct {
		total, wins int
		perfSum     float64
		perfCount   int
	}
	aggs := make(map[string]*mapAgg)
	for _, m := range matchRows {
		if m.mapID == "" {
			continue
		}
		a, ok := aggs[m.mapID]
		if !ok {
			a = &mapAgg{}
			aggs[m.mapID] = a
		}
		a.total++
		if m.outcome == domain.OutcomeWin {
			a.wins++
		}
		if perf, ok := perfMap[m.matchID]; ok {
			a.perfSum += perf
			a.perfCount++
		}
	}

	result := make(map[string]domain.MapSquadStats, len(aggs))
	for mapID, a := range aggs {
		s := domain.MapSquadStats{Wins: a.wins, Total: a.total}
		if a.perfCount > 0 {
			v := a.perfSum / float64(a.perfCount)
			s.PerfAvg = &v
		}
		result[mapID] = s
	}
	return result, nil
}

// mapStatsSquadMatchRow porte une row per-match retournée par l'étape 1 du
// split LoadMapStatsForSquad.
type mapStatsSquadMatchRow struct {
	matchID string
	mapID   string
	outcome int
}

// loadMapStatsSquadShared exécute l'étape 1 du split LoadMapStatsForSquad.
// excludeXUIDs non vide -> anti-join composition exacte (Q42MapStatsSquadExtraExclusionFrag).
func (r *SquadRepo) loadMapStatsSquadShared(ctx context.Context, mainXUID string, squadXUIDs, excludeXUIDs []string) ([]mapStatsSquadMatchRow, error) {
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()

	// Masquage Campagne (Halo 5) : registre aliasé "r", résolu AVANT Sprintf
	// (sans placeholder). No-op Infinite. Item backlog H1.
	tpl := resolveCampaignExclusion(Q42MapStatsForSquadSharedTpl, pdbTitleSlug(r.pdb), "r")
	q := fmt.Sprintf(tpl, Placeholders(len(squadXUIDs)), len(uniqueXUIDs(squadXUIDs)))
	args := make([]any, 0, len(squadXUIDs)+1+len(excludeXUIDs))
	for _, x := range squadXUIDs {
		args = append(args, x)
	}
	args = append(args, mainXUID)
	// Anti-join composition exacte (appended APRÈS le mainXUID : ordre positionnel
	// des placeholders). Omis si aucun coéquipier connu hors sélection.
	if len(excludeXUIDs) > 0 {
		q += fmt.Sprintf(Q42MapStatsSquadExtraExclusionFrag, Placeholders(len(excludeXUIDs)))
		for _, x := range excludeXUIDs {
			args = append(args, x)
		}
	}

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("shared query: %w", err)
	}
	defer rows.Close()

	var out []mapStatsSquadMatchRow
	for rows.Next() {
		var m mapStatsSquadMatchRow
		var outcome sql.NullInt64
		if err := rows.Scan(&m.matchID, &m.mapID, &outcome); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		if outcome.Valid {
			m.outcome = int(outcome.Int64)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// loadMatchPerformanceScores retourne le performance_score par match_id depuis
// player_match_enrichment (étape 2 du split LoadMapStatsForSquad).
func (r *SquadRepo) loadMatchPerformanceScores(ctx context.Context, matchIDs []string) (map[string]float64, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	query := fmt.Sprintf(`
		SELECT match_id, performance_score
		FROM player_match_enrichment_latest
		WHERE match_id IN (%s) AND performance_score IS NOT NULL`,
		Placeholders(len(matchIDs)))
	rows, err := r.pdb.Player.QueryRecovered(ctx, query, ToAnySlice(matchIDs)...)
	if err != nil {
		return nil, fmt.Errorf("performance scores query: %w", err)
	}
	defer rows.Close()
	out := make(map[string]float64, len(matchIDs))
	for rows.Next() {
		var mid string
		var perf float64
		if err := rows.Scan(&mid, &perf); err != nil {
			return nil, fmt.Errorf("performance scores scan: %w", err)
		}
		out[mid] = perf
	}
	return out, rows.Err()
}

// uniqueXUIDs déduplique sans modifier l'ordre — utile uniquement pour le
// HAVING COUNT(DISTINCT) du SQL squad_matches (la cardinalité ne doit pas
// compter les doublons applicatifs).
func uniqueXUIDs(xs []string) []string {
	seen := make(map[string]struct{}, len(xs))
	out := xs[:0:0]
	for _, x := range xs {
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

// LoadAssetTranslationsFR retourne les traductions FR depuis metadata.asset_translations.
// Wrapper mince autour du résolveur unifié `MetadataRepo.ResolveAssetNamesBulk`
// (cf. metadata_repo_assets.go). Cohérent avec home_repo.resolveAssetNames.
// assetType : "map" | "playlist" | "game_variant" | "pair".
func (r *SquadRepo) LoadAssetTranslationsFR(ctx context.Context, assetType string, assetIDs []string) (map[string]string, error) {
	if len(assetIDs) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return nil, nil
	}
	out, err := NewMetadataRepoFromDB(r.pdb.Metadata).ResolveAssetNamesBulk(
		ctx, assetType, assetIDs, PreferredLangsForLocale("fr"),
	)
	if err != nil && isTableNotFoundErr(err) {
		return nil, nil
	}
	return out, err
}

// LoadModeTranslationsFR retourne les traductions FR depuis metadata.mode_name_tr.
// Calqué sur homeRepo.loadHomeModeNameTranslations — même table, même logique.
func (r *SquadRepo) LoadModeTranslationsFR(ctx context.Context, modeENs []string) (map[string]string, error) {
	if len(modeENs) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return nil, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(modeENs)), ",")
	q := fmt.Sprintf(`SELECT mode_en, name FROM mode_name_tr WHERE lang = 'fr' AND mode_en IN (%s)`, placeholders)
	args := make([]any, len(modeENs))
	for i, n := range modeENs {
		args[i] = n
	}
	rows, err := r.pdb.Metadata.Query(ctx, q, args...)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("LoadModeTranslationsFR: %w", err)
	}
	defer rows.Close()
	result := make(map[string]string, len(modeENs))
	for rows.Next() {
		var en, fr string
		if err := rows.Scan(&en, &fr); err != nil {
			continue
		}
		if strings.TrimSpace(fr) != "" {
			result[en] = fr
		}
	}
	return result, rows.Err()
}

// Ensure SquadRepo implements port.SquadRepository at compile time.
// (Vérification implicite via injection dans le service.)
