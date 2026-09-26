// Package duckdb — MatchHistoryRepo : chargement paginé de l'historique des parties.
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
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
//  1. SharedReader.Get : v_match_full ⨝ match_participants → 31 cols (dont team_id/scores + durée)
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
	// Durée de jeu observée (socle du flag « Prolongation ») — merge BEST-EFFORT :
	// une colonne de participation absente ne doit jamais faire tomber la liste.
	r.mergeHistoryElapsedSeconds(ctx, results, matchIDs)

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
// my/enemy en Go — les points ET les manches, lus ensemble parce qu'ils se
// permutent ensemble (même team_id).
type teamScorePair struct {
	team0 *int
	team1 *int
	// rounds0 / rounds1 : manches gagnées par camp ; total : manches jouées.
	rounds0 *int
	rounds1 *int
	total   *int
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

	// Masquage READ-SIDE des matchs Campagne (Halo 5) : v_match_full est aliasé "r".
	// No-op pour Infinite (aucun mode masqué). Sans ceci, la LISTE de l'historique
	// et le compteur total remonteraient les ~287 matchs Campagne (item backlog H1).
	q := resolveCampaignExclusion(Q5SharedHistory, r.pdb.TitleSlug, "r")
	rows, err := db.QueryContext(ctx, q, r.pdb.XUID)
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
			m       domain.MatchHistoryRawRow
			teamID  *int
			team0   *int
			team1   *int
			rounds0 *int
			rounds1 *int
			total   *int
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
			&rounds0,
			&rounds1,
			&total,
			&m.GameVariantID,
			&m.GameVariantName,
			&m.DurationSeconds,
		); err != nil {
			return nil, nil, nil, fmt.Errorf("scan: %w", err)
		}
		results = append(results, m)
		teamIDs = append(teamIDs, teamID)
		teamScores = append(teamScores, teamScorePair{
			team0: team0, team1: team1, rounds0: rounds0, rounds1: rounds1, total: total,
		})
	}
	return results, teamIDs, teamScores, rows.Err()
}

// mergeHistoryElapsedSeconds hydrate ElapsedSeconds (durée de jeu observée) dans
// rows depuis shared.match_participants. Best-effort et sans valeur de retour :
// sur échec, les champs restent nil et l'Explorateur n'affiche simplement aucun
// badge « Prolongation » (le WARN est émis par l'estimateur partagé).
func (r *MatchHistoryRepo) mergeHistoryElapsedSeconds(ctx context.Context, rows []domain.MatchHistoryRawRow, matchIDs []string) {
	ctx2, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx2)
	if err != nil {
		slog.WarnContext(ctx2, "match_history: durée de jeu observée indisponible (shared reader)", "err", err)
		return
	}
	defer release()

	byMatch := LoadElapsedSecondsByMatch(ctx2, db, matchIDs)
	for i := range rows {
		if secs, ok := byMatch[rows[i].MatchID]; ok {
			v := secs
			rows[i].ElapsedSeconds = &v
		}
	}
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
		tier          *string
		tierFR        *string
		rating        *string
		tierLabel     *string
		winProb       *float64
		playlistGroup *string
		ratingValue   *float64
		ratingDelta   *float64
		subTier       *int
	}
	ranks := make(map[string]skill, len(matchIDs))
	for dbRows.Next() {
		var mid string
		var s skill
		if err := dbRows.Scan(&mid, &s.tier, &s.tierFR, &s.rating, &s.tierLabel, &s.winProb, &s.playlistGroup, &s.ratingValue, &s.ratingDelta, &s.subTier); err != nil {
			return fmt.Errorf("skill_rank scan: %w", err)
		}
		// Q5 lit désormais match_skill_rank_latest : 1 ligne/match (priorité
		// CSR>LUSR), plus de N versions à fusionner. Sur un match ranked la ligne
		// gagnante est CSR → expected_win_prob NULL (convention _latest déjà en
		// vigueur : Q22a, playerMatchesSkillRankTpl).
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
		rows[i].PlaylistGroup = s.playlistGroup
		rows[i].RatingValue = s.ratingValue
		rows[i].RatingDelta = s.ratingDelta
		rows[i].SubTier = s.subTier
	}
	return nil
}

// applyTeamScore calcule MyTeamScore/EnemyTeamScore ET les manches my/enemy depuis team_id.
// Reproduit la sémantique CASE WHEN p.team_id = 0 du SQL. Points et manches se permutent
// ENSEMBLE : les dissocier ferait afficher les manches d'un camp à côté des points de
// l'autre. `RoundsTotal` ne se permute pas — c'est une grandeur du match.
func applyTeamScore(row *domain.MatchHistoryRawRow, teamID *int, scores teamScorePair) {
	row.RoundsTotal = scores.total
	// LA FORME EST CELLE DU SQL, À LA BRANCHE PRÈS : `CASE WHEN team_id = 0 THEN … ELSE …`.
	// Ce n'est pas un détail de style — un `team_id >= 2` (FFA : Halo Infinite numérote un
	// camp par joueur) tombe dans le ELSE, et le chemin jumeau de l'escouade
	// (queries_squad.go, le même CASE en SQL) doit lui répondre la même chose. Tester
	// `== 1` au lieu de `!= 0` inverserait ces lignes-là et ferait diverger les deux
	// surfaces sur les mêmes matchs.
	if teamID == nil || *teamID == 0 {
		row.MyTeamScore, row.EnemyTeamScore = scores.team0, scores.team1
		row.MyRoundsWon, row.EnemyRoundsWon = scores.rounds0, scores.rounds1
		return
	}
	row.MyTeamScore, row.EnemyTeamScore = scores.team1, scores.team0
	row.MyRoundsWon, row.EnemyRoundsWon = scores.rounds1, scores.rounds0
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
	WHERE p.xuid = ?` + excludeCampaignClause(r.pdb.TitleSlug, "r") + ` AND r.map_name IS NOT NULL
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
