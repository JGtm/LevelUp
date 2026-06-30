// Package duckdb — MatchHistoryRepo : chargement paginé de l'historique des parties.
package duckdb

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// MatchHistoryRepo implémente port.MatchHistoryRepository.
type MatchHistoryRepo struct {
	pdb *PlayerDB
}

// NewMatchHistoryRepo crée un MatchHistoryRepo depuis un PlayerDB.
func NewMatchHistoryRepo(pdb *PlayerDB) *MatchHistoryRepo {
	return &MatchHistoryRepo{pdb: pdb}
}

// LoadAll charge tous les matchs du joueur avec les stats de participation.
//
// split+merge cross-DB. La query historique unique
// (shared.v_match_full ⨝ shared.match_participants ⨝ player_match_enrichment
// ⨝ match_skill_rank) est découpée en 3 round-trips :
//  1. SharedReader.Get : v_match_full ⨝ match_participants → 25 cols + team_id
//  2. pdb.Player : player_match_enrichment WHERE match_id IN (...)
//  3. pdb.Player : match_skill_rank WHERE match_id IN (...)
//  4. Merge Go : assemble MatchHistoryRawRow + calcule my/enemy_team_score.
func (r *MatchHistoryRepo) LoadAll(ctx context.Context) ([]domain.MatchHistoryRawRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	results, teamIDs, teamScores, err := r.loadSharedHistory(ctx)
	if err != nil {
		return nil, fmt.Errorf("MatchHistoryRepo.LoadAll: %w", err)
	}
	if len(results) == 0 {
		return results, nil
	}

	matchIDs := make([]string, 0, len(results))
	for _, m := range results {
		matchIDs = append(matchIDs, m.MatchID)
	}

	if err := r.mergeHistoryEnrichments(ctx, results, matchIDs); err != nil {
		return nil, fmt.Errorf("MatchHistoryRepo.LoadAll: %w", err)
	}
	if err := r.mergeHistorySkillRanks(ctx, results, matchIDs); err != nil {
		return nil, fmt.Errorf("MatchHistoryRepo.LoadAll: %w", err)
	}

	// Calcul my/enemy_team_score à partir de team_id et team_0/1_score.
	for i := range results {
		applyTeamScore(&results[i], teamIDs[i], teamScores[i])
	}

	// Enrichissement FR best-effort (mode/map/playlist) — même mécanisme que
	// FiltersRepo (filters_repo.go applyXxxFRTranslations), nécessaire car
	// match_registry.{map,pair,playlist}_name_fr peut être NULL.
	applyMatchHistoryFRTranslations(ctx, r.pdb, results)

	return results, nil
}

// teamScorePair retient les scores bruts par équipe d'un match pour le calcul
// my/enemy en Go.
type teamScorePair struct {
	team0 *int
	team1 *int
}

// loadSharedHistory exécute l'étape 1 du split LoadAll (SharedReader).
// Retourne aussi un parallel array de team_id et team_0/1_score (indexé sur
// results) pour permettre le calcul my/enemy en Go.
func (r *MatchHistoryRepo) loadSharedHistory(ctx context.Context) ([]domain.MatchHistoryRawRow, []*int, []teamScorePair, error) {
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, Q5SharedHistory, r.pdb.XUID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("shared query: %w", err)
	}
	defer rows.Close()

	var (
		results    []domain.MatchHistoryRawRow
		teamIDs    []*int
		teamScores []teamScorePair
	)
	for rows.Next() {
		var (
			m      domain.MatchHistoryRawRow
			teamID *int
			team0  *int
			team1  *int
		)
		if err := rows.Scan(
			&m.MatchID,
			&m.StartTime,
			&m.MapName,
			&m.MapNameFR,
			&m.PairName,
			&m.PairNameFR,
			&m.PlaylistNameEN,
			&m.PlaylistName,
			&m.MapID,
			&m.PairID,
			&m.PlaylistID,
			&m.SeasonID,
			&m.IsFirefight,
			&m.IsRanked,
			&m.Outcome,
			&m.TeamMMR,
			&m.EnemyMMR,
			&m.Kills,
			&m.Deaths,
			&m.Assists,
			&m.KDA,
			&m.Accuracy,
			&m.PersonalScore,
			&m.AverageLifeSeconds,
			&m.TimePlayedSeconds,
			&teamID,
			&team0,
			&team1,
			&m.GameVariantID,
			&m.GameVariantName,
		); err != nil {
			return nil, nil, nil, fmt.Errorf("scan: %w", err)
		}
		results = append(results, m)
		teamIDs = append(teamIDs, teamID)
		teamScores = append(teamScores, teamScorePair{team0: team0, team1: team1})
	}
	return results, teamIDs, teamScores, rows.Err()
}

// mergeHistoryEnrichments hydrate les champs player_match_enrichment dans rows.
func (r *MatchHistoryRepo) mergeHistoryEnrichments(ctx context.Context, rows []domain.MatchHistoryRawRow, matchIDs []string) error {
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
		if e.SessionID.Valid {
			s := e.SessionID.String
			rows[i].SessionID = &s
		}
		if e.SessionLabel.Valid {
			s := e.SessionLabel.String
			rows[i].SessionLabel = &s
		}
		rows[i].IsWithFriends = e.IsWithFriends
		rows[i].IsExcluded = e.IsExcluded
		if e.PerformanceScore.Valid {
			v := e.PerformanceScore.Float64
			rows[i].PerformanceScore = &v
		}
		rows[i].DominanceFlag = e.DominanceFlag
	}
	return nil
}

// mergeHistorySkillRanks hydrate les champs match_skill_rank dans rows.
func (r *MatchHistoryRepo) mergeHistorySkillRanks(ctx context.Context, rows []domain.MatchHistoryRawRow, matchIDs []string) error {
	query := fmt.Sprintf(Q5PlayerSkillRankHistoryTpl, Placeholders(len(matchIDs)))
	ctx2, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// QueryRecovered (Phase 5 ART) : retry après Reopen si la handle player
	// DB est invalidée (FATAL DuckDB).
	dbRows, err := r.pdb.Player.QueryRecovered(ctx2, query, ToAnySlice(matchIDs)...)
	if err != nil {
		return fmt.Errorf("skill_rank query: %w", err)
	}
	defer dbRows.Close()

	type skill struct {
		tier      *string
		tierFR    *string
		rating    *string
		tierLabel *string
		winProb   *float64
	}
	ranks := make(map[string]skill, len(matchIDs))
	for dbRows.Next() {
		var mid string
		var s skill
		if err := dbRows.Scan(&mid, &s.tier, &s.tierFR, &s.rating, &s.tierLabel, &s.winProb); err != nil {
			return fmt.Errorf("skill_rank scan: %w", err)
		}
		// match_skill_rank est append-only (N versions par match_id, CSR + LUSR).
		// La dernière version scannée gagne pour le tier ; mais expected_win_prob
		// n'est posé que sur les rows LUSR — on préserve toute valeur non-nil pour
		// ne pas l'effacer si une row CSR (winProb nil) est scannée après.
		if prev, ok := ranks[mid]; ok && s.winProb == nil {
			s.winProb = prev.winProb
		}
		ranks[mid] = s
	}
	if err := dbRows.Err(); err != nil {
		return fmt.Errorf("skill_rank rows: %w", err)
	}

	for i := range rows {
		s, ok := ranks[rows[i].MatchID]
		if !ok {
			continue
		}
		rows[i].SkillTier = s.tier
		rows[i].SkillTierFR = s.tierFR
		rows[i].SkillRatingType = s.rating
		rows[i].SkillTierLabel = s.tierLabel
		rows[i].SkillExpectedWinProb = s.winProb
	}
	return nil
}

// applyTeamScore calcule MyTeamScore/EnemyTeamScore depuis team_id et les
// scores team_0/team_1. Reproduit la sémantique CASE WHEN p.team_id = 0 du SQL.
func applyTeamScore(row *domain.MatchHistoryRawRow, teamID *int, scores teamScorePair) {
	if teamID == nil {
		row.MyTeamScore = scores.team0
		row.EnemyTeamScore = scores.team1
		return
	}
	if *teamID == 0 {
		row.MyTeamScore = scores.team0
		row.EnemyTeamScore = scores.team1
	} else {
		row.MyTeamScore = scores.team1
		row.EnemyTeamScore = scores.team0
	}
}

// LoadMapWinRates calcule le win_rate historique par carte (sur tous les matchs).
// Retourne map[map_name] -> {wins, total}.
func (r *MatchHistoryRepo) LoadMapWinRates(ctx context.Context) (map[string][2]int, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// PMT-5 : win title-aware (fallback "p.outcome = 2" byte-identique Halo).
	winExpr := outcomeSQLEq(ctx, "p.outcome", canonical.OutcomeWin, "p.outcome = 2")
	q := `
	SELECT r.map_name,
	       SUM(CASE WHEN ` + winExpr + ` THEN 1 ELSE 0 END) AS wins,
	       COUNT(*) AS total
	FROM match_registry r
	JOIN match_participants p ON r.match_id = p.match_id
	WHERE p.xuid = ? AND r.map_name IS NOT NULL
	GROUP BY r.map_name`

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadMapWinRates: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, q, r.pdb.XUID)
	if err != nil {
		return nil, fmt.Errorf("LoadMapWinRates: %w", err)
	}
	defer rows.Close()

	result := make(map[string][2]int)
	for rows.Next() {
		var mapName string
		var wins, total int
		if err := rows.Scan(&mapName, &wins, &total); err != nil {
			return nil, err
		}
		result[mapName] = [2]int{wins, total}
	}
	return result, rows.Err()
}
