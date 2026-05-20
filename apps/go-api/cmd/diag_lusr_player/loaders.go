//go:build cgo

package main

import (
	"database/sql"
	"log"
	"strings"
	"time"
)

// matchData : équivalent local de sync.lusrMatchData.
type matchData struct {
	matchID        string
	startTime      time.Time
	pairName       string
	outcome        *int
	kills          float64
	deaths         float64
	assists        float64
	killsExpected  float64
	deathsExpected float64
	damageDealt    float64
	damageTaken    float64
	accuracy       float64
	teamID         *int
}

// participantData : équivalent local de sync.lusrParticipant.
type participantData struct {
	matchID       string
	xuid          string
	teamID        *int
	killsExpected float64
}

// loadMatches lit les matchs LUSR-éligibles pour un joueur (mêmes filtres que
// sync.loadLUSRMatchData : non-ranked, non-firefight, durée ≥ 30s).
func loadMatches(db *sql.DB, xuid string) []matchData {
	rows, err := db.Query(`
		SELECT
			mr.match_id, mr.start_time, COALESCE(mr.pair_name, ''),
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
		log.Fatalf("loadMatches(%s): %v", xuid, err)
	}
	defer rows.Close()

	var out []matchData
	for rows.Next() {
		var m matchData
		var outcome sql.NullInt64
		var teamID sql.NullInt64
		if err := rows.Scan(
			&m.matchID, &m.startTime, &m.pairName,
			&outcome, &m.kills, &m.deaths, &m.assists,
			&m.killsExpected, &m.deathsExpected,
			&m.damageDealt, &m.damageTaken,
			&m.accuracy, &teamID,
		); err != nil {
			continue
		}
		if outcome.Valid {
			v := int(outcome.Int64)
			m.outcome = &v
		}
		if teamID.Valid {
			v := int(teamID.Int64)
			m.teamID = &v
		}
		out = append(out, m)
	}
	return out
}

// loadParticipants charge les participants pour un ensemble de match_ids.
// Mêmes colonnes que sync.loadLUSRParticipants.
func loadParticipants(db *sql.DB, matchIDs []string) map[string][]participantData {
	out := make(map[string][]participantData)
	if len(matchIDs) == 0 {
		return out
	}
	// Pour éviter "binder parameter limit", chunk par 500.
	const chunk = 500
	for start := 0; start < len(matchIDs); start += chunk {
		end := start + chunk
		if end > len(matchIDs) {
			end = len(matchIDs)
		}
		batch := matchIDs[start:end]
		placeholders := strings.Repeat("?,", len(batch))
		placeholders = placeholders[:len(placeholders)-1]
		q := "SELECT match_id, xuid, team_id, COALESCE(kills_expected, 0) " +
			"FROM match_participants WHERE match_id IN (" + placeholders + ")"
		args := make([]interface{}, len(batch))
		for i, id := range batch {
			args[i] = id
		}
		rows, err := db.Query(q, args...)
		if err != nil {
			log.Fatalf("loadParticipants chunk[%d:%d]: %v", start, end, err)
		}
		for rows.Next() {
			var p participantData
			var teamID sql.NullInt64
			if err := rows.Scan(&p.matchID, &p.xuid, &teamID, &p.killsExpected); err != nil {
				continue
			}
			if teamID.Valid {
				v := int(teamID.Int64)
				p.teamID = &v
			}
			out[p.matchID] = append(out[p.matchID], p)
		}
		rows.Close()
	}
	return out
}
