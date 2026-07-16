package duckdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
)

func (r *SquadRepo) LoadMainTeamParticipants(ctx context.Context, mainXUID string, matchIDs []string) ([]domain.AllyParticipant, error) {
	if len(matchIDs) == 0 || mainXUID == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	placeholders := make([]string, len(matchIDs))
	args := make([]interface{}, 0, 1+len(matchIDs))
	args = append(args, mainXUID)
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	query := fmt.Sprintf(Q32bMainTeamParticipantsTemplate, strings.Join(placeholders, ","))

	// shared-only via SharedReader.
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadMainTeamParticipants: shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("LoadMainTeamParticipants: %w", err)
	}
	defer rows.Close()

	var result []domain.AllyParticipant
	for rows.Next() {
		var row domain.AllyParticipant
		if err := rows.Scan(
			&row.MatchID,
			&row.XUID,
			&row.Gamertag,
			&row.Kills,
			&row.Deaths,
			&row.Assists,
			&row.Outcome,
		); err != nil {
			return nil, fmt.Errorf("LoadMainTeamParticipants scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadSynthesisHeatmap charge les donnÃ©es heatmap mapÃ—mode (Q33).
func (r *SquadRepo) LoadSynthesisHeatmap(ctx context.Context, xuid string) ([]domain.SynthesisHeatmapRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// shared-only via SharedReader.
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadSynthesisHeatmap: shared reader: %w", err)
	}
	defer release()

	// PMT-5 : win title-aware (fallback "p.outcome = 2" byte-identique Halo).
	winExpr := outcomeSQLEq(ctx, "p.outcome", canonical.OutcomeWin, "p.outcome = 2")
	heatmapQ := resolveCampaignExclusion(fmt.Sprintf(Q33SynthesisHeatmap, winExpr), r.pdb.TitleSlug, "r")
	rows, err := db.QueryContext(ctx, heatmapQ, xuid)
	if err != nil {
		return nil, fmt.Errorf("LoadSynthesisHeatmap: %w", err)
	}
	defer rows.Close()

	var result []domain.SynthesisHeatmapRow
	for rows.Next() {
		var row domain.SynthesisHeatmapRow
		if err := rows.Scan(
			&row.MapName,
			&row.ModeName,
			&row.MatchCount,
			&row.Wins,
		); err != nil {
			return nil, fmt.Errorf("LoadSynthesisHeatmap scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadSynthesisMatches charge les matchs du joueur pour le calcul top_weeks (Q33b).
//
// split cross-DB en 3 étapes.
//
//	Étape 1 (SharedReader) : Q33bSynthesisSharedQuery — match_participants ⨝
//	  match_registry. 11 cols shared.
//	Étape 2 (pdb.Player) : player_match_enrichment WHERE match_id IN (...).
//	Étape 3 (Go) : merge LEFT JOIN — hydrate is_with_friends, performance_score,
//	  session_label.
func (r *SquadRepo) LoadSynthesisMatches(ctx context.Context, xuid string) ([]legacymatch.SynthesisMatchRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Étape 1 : shared.
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadSynthesisMatches: shared reader: %w", err)
	}
	defer release()

	synthQ := resolveCampaignExclusion(Q33bSynthesisSharedQuery, r.pdb.TitleSlug, "r")
	rows, err := db.QueryContext(ctx, synthQ, xuid)
	if err != nil {
		return nil, fmt.Errorf("LoadSynthesisMatches: %w", err)
	}
	defer rows.Close()

	var result []legacymatch.SynthesisMatchRow
	for rows.Next() {
		var row legacymatch.SynthesisMatchRow
		if err := rows.Scan(
			&row.MatchID,
			&row.StartTime,
			&row.Outcome,
			&row.Kills,
			&row.Deaths,
			&row.KDA,
			&row.Accuracy,
			&row.TimePlayedSecs,
			&row.AvgLifeSeconds,
			&row.IsRanked,
			&row.IsFirefight,
			&row.PlaylistName,
		); err != nil {
			return nil, fmt.Errorf("LoadSynthesisMatches scan: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return result, nil
	}

	// Étape 2 + 3 : enrichment merge.
	matchIDs := make([]string, 0, len(result))
	for _, m := range result {
		matchIDs = append(matchIDs, m.MatchID)
	}
	if err := r.mergeSynthesisEnrichments(ctx, result, matchIDs); err != nil {
		return nil, fmt.Errorf("LoadSynthesisMatches: %w", err)
	}
	return result, nil
}

// mergeSynthesisEnrichments hydrate is_with_friends + performance_score +
// session_label depuis player_match_enrichment (étape 2/3 du split).
func (r *SquadRepo) mergeSynthesisEnrichments(ctx context.Context, rows []legacymatch.SynthesisMatchRow, matchIDs []string) error {
	ctx2, cancel := context.WithTimeout(ctx, 10*time.Second)
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
		rows[i].IsWithFriends = e.IsWithFriends
		if e.PerformanceScore.Valid {
			v := e.PerformanceScore.Float64
			rows[i].PerformanceScore = &v
		}
		if e.SessionLabel.Valid {
			v := e.SessionLabel.String
			rows[i].SessionLabel = &v
		}
	}
	return nil
}

// LoadMapStatsForSquad calcule par carte (map_id) le winrate et la performance
// moyenne du joueur principal sur les matchs où TOUS les xuids du squad sont
// participants. Aucun filtre temporel — c'est l'historique complet "avec cette
// escouade exacte".
//
// split cross-DB en 3 étapes.
//
//	Étape 1 (SharedReader) : Q42MapStatsForSquadSharedTpl — retourne per-match
//	  rows (match_id, map_id, outcome) avec le CTE squad_matches (filtre
//	  cardinality du squad).
//	Étape 2 (pdb.Player) : SELECT match_id, performance_score FROM
//	  player_match_enrichment WHERE match_id IN (...).
//	Étape 3 (Go) : aggregation par map_id — total, wins, perf_avg.
//
// Comportement :
//   - squadXUIDs vide : retourne nil, nil (pas de squad sélectionné).
//   - squadXUIDs ne contenant que mainXUID : tombe sur les stats solo du main
//     (cas dégénéré utile pour le mode solo).
//   - mainXUID inclus dans squadXUIDs : pas de doublonnage côté SQL grâce au
//     COUNT(DISTINCT xuid) dans squad_matches.
//
// Retour : map keyée sur map_id (jamais vide ; clé absente = aucun match avec
// ce squad sur cette carte).
