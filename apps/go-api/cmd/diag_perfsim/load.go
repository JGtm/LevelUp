package main

// load.go — accès DuckDB STRICTEMENT en lecture seule.
//
// Piège rencontré sur les données réelles (consigné au rapport) : l'index ART
// idx_psa_match de la DB XxDaemonGamerxX est incohérent — un lookup indexé
// (`WHERE match_id = ?`) rend 2 lignes là où le scan complet en rend 4. Toutes
// les lectures de personal_score_awards se font donc en SCAN COMPLET SANS
// PRÉDICAT, la sélection (xuid, catégorie, génération) étant faite en Go.

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/sync/skill"
)

// openRO ouvre une DuckDB en lecture seule stricte.
func openRO(path string) (*sql.DB, error) {
	db, err := sql.Open("duckdb", path+"?access_mode=read_only")
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping %s: %w", path, err)
	}
	return db, nil
}

// universeSQL est le SQL de loadHistoryForPerf (performance_helpers.go:154-172)
// à DEUX différences près, nécessaires à l'analyse de purge :
//   - la clause `AND COALESCE(mp.outcome, 0) != 4` est retirée ;
//   - `COALESCE(mp.outcome, 0)` est ajouté à la projection.
//
// Le filtre outcome != 4 est réappliqué en Go (splitScorable) : l'ensemble
// scorable est donc identique au production, l'univers total en plus.
const universeSQL = `
	SELECT
		mr.match_id, mr.start_time,
		COALESCE(mp.kills, 0), COALESCE(mp.deaths, 0),
		COALESCE(mp.assists, 0), COALESCE(mp.kda, 0),
		COALESCE(mp.accuracy, 0),
		COALESCE(mp.time_played_seconds, 600),
		COALESCE(mp.personal_score, 0), COALESCE(mp.damage_dealt, 0),
		COALESCE(mp.damage_taken, 0),
		COALESCE(mp.rank, 0),
		COALESCE(mp.team_mmr, 0), COALESCE(mp.enemy_mmr, 0),
		COALESCE(mp.kills_expected, 0), COALESCE(mp.deaths_expected, 0),
		mr.pair_name, COALESCE(mr.is_ranked, FALSE), COALESCE(mr.is_firefight, FALSE),
		COALESCE(mp.outcome, 0)
	FROM match_registry mr
	JOIN match_participants mp ON mr.match_id = mp.match_id
	WHERE mp.xuid = ?
	  AND mr.start_time IS NOT NULL
	ORDER BY mr.start_time ASC`

// loadUniverse charge tous les matchs du joueur (DNF compris), triés ASC.
func loadUniverse(ctx context.Context, sharedDB *sql.DB, xuid string) ([]matchRow, error) {
	rows, err := sharedDB.QueryContext(ctx, universeSQL, xuid)
	if err != nil {
		return nil, fmt.Errorf("loadUniverse: %w", err)
	}
	defer rows.Close()

	var out []matchRow
	for rows.Next() {
		var h matchRow
		var pairName sql.NullString
		if err := rows.Scan(
			&h.MatchID, &h.StartTime,
			&h.Kills, &h.Deaths, &h.Assists, &h.KDA,
			&h.Accuracy, &h.TimePlayedSeconds,
			&h.PersonalScore, &h.DamageDealt, &h.DamageTaken,
			&h.Rank, &h.TeamMMR, &h.EnemyMMR,
			&h.KillsExpected, &h.DeathsExpected,
			&pairName, &h.IsRanked, &h.IsFirefight, &h.Outcome,
		); err != nil {
			return nil, fmt.Errorf("loadUniverse scan: %w", err)
		}
		if pairName.Valid {
			h.PairName = pairName.String
		}
		// Identique à loadHistoryForPerf:200-203.
		h.OffensiveConversion, h.DefensiveResistance = skill.ComputeCombatYield(skill.LusrMatchData{
			Kills: h.Kills, Deaths: h.Deaths, Assists: h.Assists,
			DamageDealt: h.DamageDealt, DamageTaken: h.DamageTaken,
		})
		out = append(out, h)
	}
	return out, rows.Err()
}

// loadExcludedMatchIDs réplique skill.LoadExcludedMatchIDs (exclusion_filter.go:27).
func loadExcludedMatchIDs(ctx context.Context, playerDB *sql.DB) (map[string]bool, error) {
	result := make(map[string]bool)
	rows, err := playerDB.QueryContext(ctx,
		`SELECT match_id FROM player_match_enrichment_latest WHERE COALESCE(is_excluded, FALSE) = TRUE`)
	if err != nil {
		return nil, fmt.Errorf("loadExcludedMatchIDs: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var mid string
		if err := rows.Scan(&mid); err != nil {
			return nil, fmt.Errorf("loadExcludedMatchIDs scan: %w", err)
		}
		result[mid] = true
	}
	return result, rows.Err()
}

// psaMatch porte le résultat de dédup des awards pour un match.
type psaMatch struct {
	ObjectiveScore float64
	Covered        bool
}

// psaStats documente la dédup constatée (item B0.2 du plan).
type psaStats struct {
	RowsTotal        int
	RowsOtherXUID    int
	RowsTombstone    int
	RowsStaleGen     int // lignes d'une génération < max pour leur (match_id, xuid)
	PairsMultiGen    int
	MatchesCovered   int
	MatchesWithObj   int
	ObjScoreLatest   float64
	ObjScoreNaiveAll float64 // somme sans dédup de génération (pour mesurer l'écart)
}

type psaRow struct {
	category string
	score    float64
	gen      int64
	tomb     bool
}

// loadObjectiveByMatch calcule, par match, la somme des award_score de catégorie
// `objective` en répliquant EN GO la sémantique de la vue
// personal_score_awards_latest (DENSE_RANK sur generation_id par (match_id,xuid),
// tombstones exclus, filtre xuid strict) — la MÊME que celle du loader de production
// skill.LoadObjectiveParticipation, qui lit la vue directement.
//
// Le scan complet sans prédicat est CONSERVÉ après la réparation d'index du lot 2 :
// il rend cet oracle indépendant de l'état des index ART, dont la cause de
// désynchronisation n'est pas élucidée. Une divergence outil/produit au lot 4
// signalerait donc une re-corruption, pas un écart de règle.
func loadObjectiveByMatch(ctx context.Context, playerDB *sql.DB, xuid string) (map[string]psaMatch, psaStats, error) {
	var st psaStats
	rows, err := playerDB.QueryContext(ctx, `
		SELECT match_id, xuid, COALESCE(award_category, ''), COALESCE(award_score, 0),
		       COALESCE(generation_id, 0), COALESCE(is_tombstone, FALSE)
		  FROM personal_score_awards`)
	if err != nil {
		return nil, st, fmt.Errorf("loadObjectiveByMatch: %w", err)
	}
	defer rows.Close()

	byMatch := map[string][]psaRow{}
	maxGen := map[string]int64{}
	for rows.Next() {
		var mid, rxuid string
		var r psaRow
		if err := rows.Scan(&mid, &rxuid, &r.category, &r.score, &r.gen, &r.tomb); err != nil {
			return nil, st, fmt.Errorf("loadObjectiveByMatch scan: %w", err)
		}
		st.RowsTotal++
		if rxuid != xuid {
			st.RowsOtherXUID++
			continue
		}
		if r.tomb {
			st.RowsTombstone++
		}
		byMatch[mid] = append(byMatch[mid], r)
		if g, ok := maxGen[mid]; !ok || r.gen > g {
			maxGen[mid] = r.gen
		}
	}
	if err := rows.Err(); err != nil {
		return nil, st, fmt.Errorf("loadObjectiveByMatch rows: %w", err)
	}
	return foldObjective(byMatch, maxGen, &st), st, nil
}

// foldObjective replie les lignes brutes en une valeur par match.
func foldObjective(byMatch map[string][]psaRow, maxGen map[string]int64, st *psaStats) map[string]psaMatch {
	out := make(map[string]psaMatch, len(byMatch))
	for mid, rs := range byMatch {
		top := maxGen[mid]
		multi := false
		var sum float64
		covered := false
		for _, r := range rs {
			if r.category == "objective" {
				st.ObjScoreNaiveAll += r.score
			}
			if r.gen != top {
				st.RowsStaleGen++
				multi = true
				continue
			}
			if r.tomb {
				continue
			}
			covered = true
			if r.category == "objective" {
				sum += r.score
			}
		}
		if multi {
			st.PairsMultiGen++
		}
		if !covered {
			continue
		}
		st.MatchesCovered++
		if sum != 0 {
			st.MatchesWithObj++
		}
		st.ObjScoreLatest += sum
		out[mid] = psaMatch{ObjectiveScore: sum, Covered: true}
	}
	return out
}

// storedRow est une ligne de player_match_enrichment_latest telle que stockée.
type storedRow struct {
	MatchID  string
	Score    *float64
	Chain    string
	Excluded bool
}

// loadStoredScores lit les notes DÉJÀ stockées (base de l'analyse de purge).
func loadStoredScores(ctx context.Context, playerDB *sql.DB) ([]storedRow, error) {
	rows, err := playerDB.QueryContext(ctx, `
		SELECT match_id, performance_score, COALESCE(performance_chain, ''),
		       COALESCE(is_excluded, FALSE)
		  FROM player_match_enrichment_latest`)
	if err != nil {
		return nil, fmt.Errorf("loadStoredScores: %w", err)
	}
	defer rows.Close()

	var out []storedRow
	for rows.Next() {
		var r storedRow
		var score sql.NullFloat64
		if err := rows.Scan(&r.MatchID, &score, &r.Chain, &r.Excluded); err != nil {
			return nil, fmt.Errorf("loadStoredScores scan: %w", err)
		}
		if score.Valid {
			v := score.Float64
			r.Score = &v
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
