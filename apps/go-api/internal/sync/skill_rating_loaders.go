// Package sync — skill_rating_loaders.go : chargement et persistance SQL pour LUSR.
//
// Sépare les accès DB (database/sql) de la logique algorithmique (skill_rating.go).
package sync

import (
	"database/sql"
	"fmt"
	"math"
	"time"
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
}

// lusrResult contient le résultat du calcul LUSR pour un match.
type lusrResult struct {
	MatchID         string
	RatingValue     float64
	RatingDeviation float64
	PlaylistGroup   string
}

// ── Chargeurs SQL ────────────────────────────────────────────────────────────

func loadLUSRMatchData(sharedDB *sql.DB, xuid string) ([]lusrMatchData, error) {
	rows, err := sharedDB.Query(`
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
		  AND mr.start_time IS NOT NULL
		  AND (mr.duration_seconds IS NULL OR mr.duration_seconds >= 30)
		ORDER BY mr.start_time ASC`, xuid)
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

func loadLUSRParticipants(sharedDB *sql.DB, matchIDs []string) (map[string][]lusrParticipant, error) {
	result := make(map[string][]lusrParticipant)
	if len(matchIDs) == 0 {
		return result, nil
	}

	// Build IN clause with placeholders.
	query := "SELECT match_id, xuid, team_id, COALESCE(kills_expected, 0) FROM match_participants WHERE match_id IN ("
	args := make([]interface{}, len(matchIDs))
	for i, id := range matchIDs {
		if i > 0 {
			query += ","
		}
		query += "?"
		args[i] = id
	}
	query += ")"

	rows, err := sharedDB.Query(query, args...)
	if err != nil {
		return result, fmt.Errorf("loadLUSRParticipants: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var p lusrParticipant
		var teamID sql.NullInt64
		if err := rows.Scan(&p.MatchID, &p.XUID, &teamID, &p.KillsExpected); err != nil {
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

func loadExistingRatingIDs(playerDB *sql.DB, ratingType string) map[string]bool {
	result := make(map[string]bool)
	rows, err := playerDB.Query("SELECT match_id FROM match_skill_rank WHERE rating_type = ?", ratingType)
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var mid string
		if rows.Scan(&mid) == nil {
			result[mid] = true
		}
	}
	return result
}

func loadExistingLUSRStates(playerDB *sql.DB) map[string]*PlayerState {
	states := make(map[string]*PlayerState)
	rows, err := playerDB.Query(`
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
	playerDB *sql.DB,
	results []lusrResult,
	existingCSR, existingLUSR map[string]bool,
	seedRatings map[string]float64,
) (int, error) {
	now := time.Now().UTC()
	prevRating := make(map[string]float64)
	for pg, r := range seedRatings {
		prevRating[pg] = r
	}

	updated := 0
	for _, r := range results {
		if existingCSR[r.MatchID] {
			continue
		}
		if existingLUSR[r.MatchID] {
			continue
		}

		ratingValue := r.RatingValue
		var delta *float64
		if prev, ok := prevRating[r.PlaylistGroup]; ok {
			rawDelta := ratingValue - prev
			if math.Abs(rawDelta) > LUSRMaxDelta {
				if rawDelta > 0 {
					rawDelta = LUSRMaxDelta
				} else {
					rawDelta = -LUSRMaxDelta
				}
				ratingValue = prev + rawDelta
			}
			delta = &rawDelta
		}
		prevRating[r.PlaylistGroup] = ratingValue

		tier, sub := GetTierForRating(ratingValue)
		var tierName, tierFR, tierLabel *string
		if tier != nil {
			tierName = &tier.Name
			tierFR = &tier.NameFR
			label := FormatTierLabel(ratingValue)
			tierLabel = &label
		}

		_, err := playerDB.Exec(`
			INSERT INTO match_skill_rank
				(match_id, rating_type, rating_value, rating_deviation,
				 tier, tier_fr, sub_tier, tier_label,
				 rating_delta, playlist_group, created_at, updated_at)
			VALUES (?, 'LUSR', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (match_id) DO UPDATE SET
				rating_type      = 'LUSR',
				rating_value     = EXCLUDED.rating_value,
				rating_deviation = EXCLUDED.rating_deviation,
				tier             = EXCLUDED.tier,
				tier_fr          = EXCLUDED.tier_fr,
				sub_tier         = EXCLUDED.sub_tier,
				tier_label       = EXCLUDED.tier_label,
				rating_delta     = EXCLUDED.rating_delta,
				playlist_group   = EXCLUDED.playlist_group,
				updated_at       = EXCLUDED.updated_at`,
			r.MatchID, ratingValue, r.RatingDeviation,
			tierName, tierFR, sub, tierLabel,
			delta, r.PlaylistGroup, now, now)
		if err != nil {
			continue
		}
		updated++
	}
	return updated, nil
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
