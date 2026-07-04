// Package sync — skill_rating_loaders.go : chargement et persistance SQL pour LUSR.
//
// Sépare les accès DB (database/sql) de la logique algorithmique (skill_rating.go).
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"levelup/go-api/internal/analysis"
)

// ── Structs de données ───────────────────────────────────────────────────────

// lusrMatchData contient les données d'un match pour le calcul LUSR.
type lusrMatchData struct {
	MatchID        string
	StartTime      time.Time
	PlaylistName   *string
	PairName       *string
	Outcome        *int
	Kills          float64
	Deaths         float64
	Assists        float64
	KillsExpected  float64
	DeathsExpected float64
	DamageDealt    float64
	DamageTaken    float64
	Accuracy       float64
	TeamID         *int
}

// lusrParticipant contient les données d'un participant pour le calcul LUSR.
type lusrParticipant struct {
	MatchID       string
	XUID          string
	TeamID        *int
	KillsExpected float64
	Kills         float64
	Deaths        float64
	DamageDealt   float64
}

// lusrResult contient le résultat du calcul LUSR pour un match.
//
// Components : breakdown des 8 composantes calculées pour ce match (clés =
// noms canoniques de CompositeWeights). Vide si le match a été seed (pas
// assez de matchs) ou si toutes les composantes étaient absentes.
// Persistée par upsertLUSRRatings dans `lusr_component_history` (V2 commit-1).
type lusrResult struct {
	MatchID         string
	RatingValue     float64
	RatingDeviation float64
	PlaylistGroup   string
	Components      map[string]float64
}

// ── Chargeurs SQL ────────────────────────────────────────────────────────────

func loadLUSRMatchData(ctx context.Context, sharedDB *sql.DB, xuid string) ([]lusrMatchData, error) {
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT
			mr.match_id, mr.start_time, mr.playlist_name, mr.pair_name,
			mp.outcome, COALESCE(mp.kills, 0), COALESCE(mp.deaths, 0),
			COALESCE(mp.assists, 0),
			COALESCE(mp.kills_expected, 0), COALESCE(mp.deaths_expected, 0),
			COALESCE(mp.damage_dealt, 0), COALESCE(mp.damage_taken, 0),
			COALESCE(mp.accuracy, 0), mp.team_id
		FROM match_registry mr
		JOIN match_participants mp ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?
		  AND COALESCE(mr.is_ranked, FALSE) = FALSE
		  AND COALESCE(mr.is_firefight, FALSE) = FALSE
		  -- title-generic : titres canoniques (Halo 5) → start_time_utc, start_time NULL.
		  AND COALESCE(mr.start_time_utc, mr.start_time) IS NOT NULL
		  AND (mr.duration_seconds IS NULL OR mr.duration_seconds >= 30)
		ORDER BY `+analysis.SQLStartTimeCanonical("mr")+` ASC`, xuid)
	if err != nil {
		return nil, fmt.Errorf("loadLUSRMatchData: %w", err)
	}
	defer rows.Close()

	var result []lusrMatchData
	for rows.Next() {
		var m lusrMatchData
		var outcome sql.NullInt64
		var teamID sql.NullInt64
		var plName, pairName sql.NullString
		if err := rows.Scan(
			&m.MatchID, &m.StartTime, &plName, &pairName,
			&outcome, &m.Kills, &m.Deaths, &m.Assists,
			&m.KillsExpected, &m.DeathsExpected,
			&m.DamageDealt, &m.DamageTaken,
			&m.Accuracy, &teamID,
		); err != nil {
			continue
		}
		if plName.Valid {
			m.PlaylistName = &plName.String
		}
		if pairName.Valid {
			m.PairName = &pairName.String
		}
		if outcome.Valid {
			v := int(outcome.Int64)
			m.Outcome = &v
		}
		if teamID.Valid {
			v := int(teamID.Int64)
			m.TeamID = &v
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func loadLUSRParticipants(ctx context.Context, sharedDB *sql.DB, matchIDs []string) (map[string][]lusrParticipant, error) {
	result := make(map[string][]lusrParticipant)
	if len(matchIDs) == 0 {
		return result, nil
	}

	// Build IN clause with placeholders.
	query := "SELECT match_id, xuid, team_id, COALESCE(kills_expected, 0), COALESCE(kills, 0), COALESCE(deaths, 0), COALESCE(damage_dealt, 0) FROM match_participants WHERE match_id IN ("
	args := make([]interface{}, len(matchIDs))
	for i, id := range matchIDs {
		if i > 0 {
			query += ","
		}
		query += "?"
		args[i] = id
	}
	query += ")"

	rows, err := sharedDB.QueryContext(ctx, query, args...)
	if err != nil {
		return result, fmt.Errorf("loadLUSRParticipants: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var p lusrParticipant
		var teamID sql.NullInt64
		if err := rows.Scan(&p.MatchID, &p.XUID, &teamID, &p.KillsExpected, &p.Kills, &p.Deaths, &p.DamageDealt); err != nil {
			continue
		}
		if teamID.Valid {
			v := int(teamID.Int64)
			p.TeamID = &v
		}
		result[p.MatchID] = append(result[p.MatchID], p)
	}
	return result, rows.Err()
}

// loadExistingRatingIDs renvoie l'ensemble des match_id ayant le rating_type donné.
// Toute erreur SQL/scan est propagée — un map vide silencieux désactiverait la garde
// en profondeur Go qui protège les CSR contre l'écrasement par LUSR.
func loadExistingRatingIDs(ctx context.Context, playerDB *sql.DB, ratingType string) (map[string]bool, error) {
	result := make(map[string]bool)
	rows, err := playerDB.QueryContext(ctx, "SELECT match_id FROM match_skill_rank WHERE rating_type = ?", ratingType)
	if err != nil {
		return nil, fmt.Errorf("loadExistingRatingIDs(%s): %w", ratingType, err)
	}
	defer rows.Close()
	for rows.Next() {
		var mid string
		if err := rows.Scan(&mid); err != nil {
			return nil, fmt.Errorf("loadExistingRatingIDs(%s) scan: %w", ratingType, err)
		}
		result[mid] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("loadExistingRatingIDs(%s) rows: %w", ratingType, err)
	}
	return result, nil
}

func loadExistingLUSRStates(ctx context.Context, playerDB *sql.DB) map[string]*PlayerState {
	states := make(map[string]*PlayerState)
	rows, err := playerDB.QueryContext(ctx, `
		SELECT msr.playlist_group, msr.rating_value, msr.rating_deviation
		FROM match_skill_rank msr
		JOIN (
			SELECT playlist_group, MAX(start_time) AS max_st
			FROM match_skill_rank
			WHERE rating_type = 'LUSR'
			GROUP BY playlist_group
		) last ON msr.playlist_group = last.playlist_group
		       AND msr.start_time = last.max_st
		WHERE msr.rating_type = 'LUSR'`)
	if err != nil {
		return states
	}
	defer rows.Close()
	for rows.Next() {
		var pg string
		var rv, rd sql.NullFloat64
		if rows.Scan(&pg, &rv, &rd) != nil {
			continue
		}
		s := NewPlayerState()
		if rv.Valid {
			s.MU = rv.Float64
		}
		if rd.Valid {
			s.Sigma = rd.Float64
		}
		states[pg] = s
	}
	return states
}

func upsertLUSRRatings(
	ctx context.Context,
	playerDB *sql.DB,
	results []lusrResult,
	existingCSR, existingLUSR map[string]bool,
	seedRatings map[string]float64,
) (int, error) {
	// Phase 3 du refactor ART : seul le batch path est conservé.
	// Le chemin legacy row-by-row (qui était INSERT pur depuis Phase 2.E
	// mais moins efficace) est désormais inaccessible en prod.
	// upsertLUSRRatingsBatch route via AppendOnlyLUSRPersister.Persist
	// (INSERT pur, bug ART impossible par construction).
	return upsertLUSRRatingsBatch(ctx, playerDB, results, existingCSR, existingLUSR, seedRatings)
}

// writeLUSRComponentHistory persiste les 8 composantes d'un match dans
// lusr_component_history en append-only (INSERT pur, jamais d'ON CONFLICT).
//
// La table est append-only (phase ART #23046) : N versions par
// (match_id, component_name) ; la lecture courante passe par la vue
// lusr_component_history_latest. Le mode force (recalcul) écrit simplement une
// nouvelle version plus récente (computed_at) — la vue la priorise. L'ancien
// ON CONFLICT DO UPDATE (delete+insert interne sur l'index ART) est éliminé.
//
// Best-effort : un échec sur 1 composante ne stoppe pas les autres.
func writeLUSRComponentHistory(
	ctx context.Context,
	playerDB *sql.DB,
	matchID string,
	components map[string]float64,
	now time.Time,
) error {
	weights := CompositeWeights
	for name, value := range components {
		weight := weights[name]
		_, err := playerDB.ExecContext(ctx, `
			INSERT INTO lusr_component_history (match_id, component_name, value, weight, computed_at)
			VALUES (?, ?, ?, ?, ?)
		`, matchID, name, value, weight, now)
		if err != nil {
			return fmt.Errorf("insert %s/%s: %w", matchID, name, err)
		}
	}
	return nil
}

// ── Helpers stats participants ────────────────────────────────────────────────

func computeMatchKEStats(participants []lusrParticipant) (float64, float64) {
	var keValues []float64
	for _, p := range participants {
		if p.KillsExpected > 0 {
			keValues = append(keValues, p.KillsExpected)
		}
	}
	if len(keValues) == 0 {
		return InitialMU, 1.0
	}
	n := float64(len(keValues))
	sum := 0.0
	for _, k := range keValues {
		sum += k
	}
	avg := sum / n
	if len(keValues) < 2 {
		return avg, 1.0
	}
	variance := 0.0
	for _, k := range keValues {
		variance += (k - avg) * (k - avg)
	}
	variance /= n
	std := math.Sqrt(variance)
	if std == 0 {
		std = 1.0
	}
	return avg, std
}

func splitParticipantKEs(playerTeamID *int, participants []lusrParticipant) ([]float64, []float64) {
	var teammateKEs, enemyKEs []float64
	if playerTeamID == nil {
		for _, p := range participants {
			if p.KillsExpected > 0 {
				enemyKEs = append(enemyKEs, p.KillsExpected)
			}
		}
		return teammateKEs, enemyKEs
	}
	for _, p := range participants {
		if p.TeamID != nil && *p.TeamID == *playerTeamID {
			if p.KillsExpected > 0 {
				teammateKEs = append(teammateKEs, p.KillsExpected)
			}
		} else {
			if p.KillsExpected > 0 {
				enemyKEs = append(enemyKEs, p.KillsExpected)
			}
		}
	}
	return teammateKEs, enemyKEs
}
