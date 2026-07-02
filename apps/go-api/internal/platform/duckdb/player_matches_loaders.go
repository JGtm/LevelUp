package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/port"
)

func (r *PlayerMatchesRepo) LobbySizesAtCompletion(ctx context.Context, matchIDs []string) (map[string]int, error) {
	out := make(map[string]int, len(matchIDs))
	if len(matchIDs) == 0 {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(matchIDs)), ",")
	q := fmt.Sprintf(`
		SELECT match_id, COUNT(*) AS n
		FROM match_participants
		WHERE present_at_completion AND match_id IN (%s)
		GROUP BY match_id`, placeholders)
	args := make([]any, len(matchIDs))
	for i, id := range matchIDs {
		args[i] = id
	}

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("lobby sizes shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("lobby sizes query: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, fmt.Errorf("lobby sizes scan: %w", err)
		}
		out[id] = n
	}
	return out, rows.Err()
}

// ObjectiveScores retourne, par match_id, la somme des scores PSA de catégorie
// "objective" du joueur (table personal_score_awards, DB player). Alimente l'axe
// Objective (+ le résiduel de l'axe Score) du profil de participation de la session.
// Dégradation silencieuse : table absente / joueur sans PSA → map vide.
func (r *PlayerMatchesRepo) ObjectiveScores(ctx context.Context, matchIDs []string) (map[string]int, error) {
	out := make(map[string]int, len(matchIDs))
	if len(matchIDs) == 0 || r.pdb == nil || r.pdb.XUID == "" {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var tableCount int
	if err := r.pdb.ReadDB().QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_name = 'personal_score_awards'
	`).Scan(&tableCount); err != nil || tableCount == 0 {
		return out, nil
	}

	ph := Placeholders(len(matchIDs))
	args := make([]any, 0, 1+len(matchIDs))
	args = append(args, r.pdb.XUID)
	for _, id := range matchIDs {
		args = append(args, id)
	}
	q := `SELECT match_id, COALESCE(SUM(award_score), 0)::INTEGER AS total
	      FROM personal_score_awards_latest
	      WHERE award_category = 'objective'
	        AND xuid = ?
	        AND match_id IN (` + ph + `)
	      GROUP BY match_id`

	rows, err := r.pdb.ReadDB().Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("objective scores query: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var total int
		if err := rows.Scan(&id, &total); err != nil {
			return nil, fmt.Errorf("objective scores scan: %w", err)
		}
		out[id] = total
	}
	return out, rows.Err()
}

func (r *PlayerMatchesRepo) loadSharedRows(ctx context.Context, filters port.PlayerMatchFilters) ([]playerMatchScanResult, error) {
	q, args, _, err := r.buildSharedQuery(filters)
	if err != nil {
		return nil, fmt.Errorf("build shared query: %w", err)
	}

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("shared query: %w", err)
	}
	defer rows.Close()

	var results []playerMatchScanResult
	for rows.Next() {
		s, err := scanSharedPlayerMatchRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan shared row: %w", err)
		}
		s.xuid = r.pdb.XUID
		s.gamertag = r.pdb.Gamertag
		results = append(results, s)
	}
	return results, rows.Err()
}

// loadEnrichmentsForMatches récupère player_match_enrichment via le helper
// partagé LoadPlayerMatchEnrichments (commit 9d.4).
func (r *PlayerMatchesRepo) loadEnrichmentsForMatches(ctx context.Context, matchIDs []string) (map[string]MatchEnrichment, error) {
	return LoadPlayerMatchEnrichments(ctx, r.pdb.Player, matchIDs)
}

// loadSkillRanksForMatches récupère match_skill_rank pour la liste de match_ids
// (étape 3 du split). Retourne une map indexée par match_id.
func (r *PlayerMatchesRepo) loadSkillRanksForMatches(ctx context.Context, matchIDs []string) (map[string]playerMatchSkillRankRow, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	query := fmt.Sprintf(playerMatchesSkillRankTpl, Placeholders(len(matchIDs)))
	// QueryRecovered (Phase 5 ART) : retry après Reopen si la handle player DB
	// est invalidée (FATAL DuckDB déclenché par un crash ART sur une autre table).
	rows, err := r.pdb.Player.QueryRecovered(ctx, query, ToAnySlice(matchIDs)...)
	if err != nil {
		return nil, fmt.Errorf("skill_rank query: %w", err)
	}
	defer rows.Close()

	out := make(map[string]playerMatchSkillRankRow, len(matchIDs))
	for rows.Next() {
		var (
			mid string
			s   playerMatchSkillRankRow
		)
		if err := rows.Scan(&mid, &s.ratingType, &s.ratingValue, &s.tier,
			&s.tierFR, &s.subTier, &s.delta, &s.playlistGroup, &s.expectedWinProb); err != nil {
			return nil, fmt.Errorf("skill_rank scan: %w", err)
		}
		out[mid] = s
	}
	return out, rows.Err()
}

// matchCSRMeta porte season_id + measurement_matches_remaining issus de
// match_csrs (shared DB), clé (match_id, xuid). Ces champs sont CSR-spécifiques
// (NULL pour les matchs social/LUSR sans row match_csrs) et alimentent le
// canonical.SkillSnapshot (graphes progression CSR/LUSR).
type matchCSRMeta struct {
	seasonID             sql.NullString
	measurementRemaining sql.NullInt64
}

// playerMatchesCSRMetaTpl : SQL pour l'étape 3b du split. Lit la dernière row
// match_csrs par match pour le joueur courant (append-only → QUALIFY latest).
// Source unique de vérité de season_id / measurement_matches_remaining ; ces
// colonnes n'existent pas sur match_skill_rank (cf. ADR refactor ART Phase 2).
const playerMatchesCSRMetaTpl = `
SELECT
    match_id,
    season_id,
    measurement_matches_remaining
FROM match_csrs_latest
WHERE match_id IN (%s) AND xuid = ?`

// loadMatchCSRMetaForMatches récupère season_id + measurement_matches_remaining
// depuis match_csrs (shared DB) pour la liste de match_ids du joueur courant
// (étape 3b du split). Retourne une map indexée par match_id ; les matchs sans
// CSR (social/LUSR) sont simplement absents de la map.
func (r *PlayerMatchesRepo) loadMatchCSRMetaForMatches(ctx context.Context, matchIDs []string) (map[string]matchCSRMeta, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	query := fmt.Sprintf(playerMatchesCSRMetaTpl, Placeholders(len(matchIDs)))
	args := append(ToAnySlice(matchIDs), r.pdb.XUID)

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("match_csrs meta query: %w", err)
	}
	defer rows.Close()

	out := make(map[string]matchCSRMeta, len(matchIDs))
	for rows.Next() {
		var (
			mid string
			m   matchCSRMeta
		)
		if err := rows.Scan(&mid, &m.seasonID, &m.measurementRemaining); err != nil {
			return nil, fmt.Errorf("match_csrs meta scan: %w", err)
		}
		out[mid] = m
	}
	return out, rows.Err()
}

// mergePlayerMatchRows assemble les rows finaux depuis les 4 sources :
// shared + enrichments + skill_ranks + csr_meta. Applique le filtre HadBotTeammate (player)
// post-merge, le re-tri éventuel par performance_score, et le LIMIT.
