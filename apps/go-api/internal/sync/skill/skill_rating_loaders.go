// Package sync — skill_rating_loaders.go : chargement et persistance SQL pour LUSR.
//
// Sépare les accès DB (database/sql) de la logique algorithmique (skill_rating.go).
package skill

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"levelup/go-api/internal/analysis"
)

// ── Structs de données ───────────────────────────────────────────────────────

// LusrMatchData contient les données d'un match pour le calcul LUSR.
type LusrMatchData struct {
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

// ── Chargeurs SQL ────────────────────────────────────────────────────────────

func loadLUSRMatchData(ctx context.Context, sharedDB *sql.DB, xuid string) ([]LusrMatchData, error) {
	return queryLUSRMatchData(ctx, sharedDB, xuid, "")
}

// queryLUSRMatchData charge les matchs LUSR-éligibles du joueur, en ordre chronologique ;
// matchID non vide restreint à ce match (zéro ou une ligne). Une seule définition de
// l'éligibilité SQL pour le simulateur et les composantes du LUSR v2.
func queryLUSRMatchData(ctx context.Context, sharedDB *sql.DB, xuid, matchID string) ([]LusrMatchData, error) {
	filter, args := "", []any{xuid}
	if matchID != "" {
		filter, args = " AND mr.match_id = ?", append(args, matchID)
	}
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
		  AND (mr.duration_seconds IS NULL OR mr.duration_seconds >= 30)`+filter+
		analysis.SQLExcludeCampaignByMatchIDCtx(ctx, "mr.match_id")+`
		ORDER BY `+analysis.SQLStartTimeCanonical("mr")+` ASC`, args...)
	if err != nil {
		return nil, fmt.Errorf("loadLUSRMatchData: %w", err)
	}
	defer rows.Close()

	var result []LusrMatchData
	for rows.Next() {
		var m LusrMatchData
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

// LoadExistingRatingIDs renvoie l'ensemble des match_id ayant le rating_type donné.
// Toute erreur SQL/scan est propagée — un map vide silencieux désactiverait la garde
// en profondeur Go qui protège les CSR contre l'écrasement par LUSR.
func LoadExistingRatingIDs(ctx context.Context, playerDB *sql.DB, ratingType string) (map[string]bool, error) {
	result := make(map[string]bool)
	rows, err := playerDB.QueryContext(ctx, "SELECT match_id FROM match_skill_rank WHERE rating_type = ?", ratingType)
	if err != nil {
		return nil, fmt.Errorf("LoadExistingRatingIDs(%s): %w", ratingType, err)
	}
	defer rows.Close()
	for rows.Next() {
		var mid string
		if err := rows.Scan(&mid); err != nil {
			return nil, fmt.Errorf("LoadExistingRatingIDs(%s) scan: %w", ratingType, err)
		}
		result[mid] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("LoadExistingRatingIDs(%s) rows: %w", ratingType, err)
	}
	return result, nil
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
