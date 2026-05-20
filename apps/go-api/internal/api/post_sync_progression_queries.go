// Package api — post_sync_progression_queries.go : queries de support pour
// l'orchestrateur post-sync de la couche progression (V2 Ascension).
//
// Toutes les queries sont read-only. Depuis ADR 0016 (retrait final
// d'attachShared, P2), les queries cross-DB sont scindées en 2 phases :
// la partie shared (match_participants, match_registry) lue via SharedReader,
// la partie player (player_match_enrichment) lue sur pdb.Player, jointure
// faite côté Go.
package api

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/progression/milestones"
	"levelup/go-api/internal/progression/records"
	"levelup/go-api/internal/progression/streaks"
)

// AccuracyThresholdForDays est le seuil utilisé pour le compte
// accuracy_threshold_days (milestone régularité). Décision §6 :
// « au moins 1 match du jour a accuracy >= 0.50 ».
const AccuracyThresholdForDays = 0.50

// loadProgressionMatches lit les matchs récents avec les métriques nécessaires
// aux détecteurs streaks (KDA pour les types perf-based) et records.
//
// La fenêtre = derniers `lookbackDays` jours. Limite par défaut suffisante
// pour couvrir records 90d + streak walkBuckets.
//
// Sortie : 2 vues du même set de matchs — streaks.MatchActivity (timing +
// KDA pour predicates perf) et records.MatchInput (5 métriques pour PB).
func loadProgressionMatches(ctx context.Context, pdb *duckdb.PlayerDB, lookbackDays int, now time.Time) ([]streaks.MatchActivity, []records.MatchInput, error) {
	if pdb == nil || pdb.Player == nil {
		return nil, nil, fmt.Errorf("loadProgressionMatches: player DB not attached")
	}
	since := now.AddDate(0, 0, -lookbackDays)
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// P2 (ADR 0016) : split cross-DB en 2 phases.
	// Phase A : shared via SharedReader (match_participants + match_registry).
	sharedDB, release, err := pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("loadProgressionMatches: shared reader: %w", err)
	}
	defer release()

	rows, err := sharedDB.QueryContext(ctx, `
		SELECT
			mp.match_id,
			mr.start_time,
			COALESCE(mp.kda, 0) AS kda,
			COALESCE(mp.kills, 0) AS kills,
			COALESCE(mp.accuracy, 0) AS accuracy,
			COALESCE(mp.personal_score, 0) AS personal_score,
			COALESCE(mp.time_played_seconds, 0) AS time_played_seconds
		FROM match_participants mp
		JOIN match_registry mr ON mp.match_id = mr.match_id
		WHERE mp.xuid = ? AND mr.start_time >= ?
		ORDER BY mr.start_time ASC
	`, pdb.XUID, since)
	if err != nil {
		return nil, nil, fmt.Errorf("query progression matches: %w", err)
	}
	defer rows.Close()

	type matchRow struct {
		matchID   string
		startTime time.Time
		kda       float64
		kills     float64
		accuracy  float64
		personal  float64
		timeSec   float64
	}
	var loaded []matchRow
	matchIDs := make([]string, 0)
	for rows.Next() {
		var (
			r         matchRow
			startTime sql.NullTime
		)
		if err := rows.Scan(&r.matchID, &startTime, &r.kda, &r.kills, &r.accuracy, &r.personal, &r.timeSec); err != nil {
			return nil, nil, fmt.Errorf("scan progression match: %w", err)
		}
		if !startTime.Valid {
			continue
		}
		r.startTime = startTime.Time
		loaded = append(loaded, r)
		matchIDs = append(matchIDs, r.matchID)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	// Phase B : performance_score depuis player_match_enrichment.
	perfByMatch := make(map[string]float64, len(matchIDs))
	if len(matchIDs) > 0 {
		placeholders := make([]string, len(matchIDs))
		args := make([]any, len(matchIDs))
		for i, id := range matchIDs {
			placeholders[i] = "?"
			args[i] = id
		}
		pmeRows, err := pdb.Player.Query(ctx, `
			SELECT match_id, COALESCE(performance_score, 0)
			FROM player_match_enrichment
			WHERE match_id IN (`+joinProgressionPlaceholders(placeholders)+`)
		`, args...)
		if err != nil {
			return nil, nil, fmt.Errorf("query performance_score: %w", err)
		}
		for pmeRows.Next() {
			var mid string
			var perf float64
			if err := pmeRows.Scan(&mid, &perf); err != nil {
				pmeRows.Close()
				return nil, nil, fmt.Errorf("scan performance_score: %w", err)
			}
			perfByMatch[mid] = perf
		}
		pmeRows.Close()
	}

	// Phase C : assembler activities + inputs.
	activities := make([]streaks.MatchActivity, 0, len(loaded))
	inputs := make([]records.MatchInput, 0, len(loaded))
	for _, r := range loaded {
		perfScore := perfByMatch[r.matchID]
		kpm := 0.0
		pspm := 0.0
		if r.timeSec > 0 {
			minutes := r.timeSec / 60
			kpm = r.kills / minutes
			pspm = r.personal / minutes
		}
		activities = append(activities, streaks.MatchActivity{
			PlayedAt: r.startTime,
			Stats:    map[string]float64{"kda": r.kda},
		})
		inputs = append(inputs, records.MatchInput{
			MatchID:  r.matchID,
			PlayedAt: r.startTime,
			Metrics: map[records.TrackedMetric]float64{
				records.MetricPerformanceScore: perfScore,
				records.MetricKDA:              r.kda,
				records.MetricKPM:              kpm,
				records.MetricAccuracy:         r.accuracy,
				records.MetricPSPM:             pspm,
			},
		})
	}
	return activities, inputs, nil
}

// joinProgressionPlaceholders compose une chaîne `?,?,...` pour clause IN.
func joinProgressionPlaceholders(placeholders []string) string {
	out := ""
	for i, p := range placeholders {
		if i > 0 {
			out += ","
		}
		out += p
	}
	return out
}

// loadPlayerStats agrège les compteurs cumulatifs nécessaires aux milestones.
// matches_played, wins, kills, headshots, assists + accuracy_threshold_days
// (nombre de jours distincts avec au moins 1 match accuracy >= 0.50).
//
// Convention « outcome=2 » = victoire (cf. legacymatch.Outcome et analyse).
func loadPlayerStats(ctx context.Context, pdb *duckdb.PlayerDB) (milestones.PlayerStats, error) {
	out := milestones.PlayerStats{Metrics: map[string]float64{}}
	if pdb == nil || pdb.Player == nil {
		return out, fmt.Errorf("loadPlayerStats: player DB not attached")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	sharedDB, release, err := pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return out, fmt.Errorf("loadPlayerStats: shared reader: %w", err)
	}
	defer release()

	var (
		matchesPlayed int64
		wins          int64
		kills         int64
		headshots     int64
		assists       int64
	)
	if err := sharedDB.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN outcome = 2 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(kills), 0),
			COALESCE(SUM(headshot_kills), 0),
			COALESCE(SUM(assists), 0)
		FROM match_participants
		WHERE xuid = ?
	`, pdb.XUID).Scan(&matchesPlayed, &wins, &kills, &headshots, &assists); err != nil {
		return out, fmt.Errorf("aggregate stats: %w", err)
	}

	// accuracy_threshold_days : COUNT(DISTINCT DATE(start_time)) where any match >= threshold.
	var accuracyDays int64
	if err := sharedDB.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT CAST(mr.start_time AS DATE))
		FROM match_participants mp
		JOIN match_registry mr ON mp.match_id = mr.match_id
		WHERE mp.xuid = ? AND mp.accuracy >= ?
	`, pdb.XUID, AccuracyThresholdForDays).Scan(&accuracyDays); err != nil {
		return out, fmt.Errorf("aggregate accuracy days: %w", err)
	}

	out.Metrics["matches_played"] = float64(matchesPlayed)
	out.Metrics["wins"] = float64(wins)
	out.Metrics["kills"] = float64(kills)
	out.Metrics["headshots"] = float64(headshots)
	out.Metrics["assists"] = float64(assists)
	out.Metrics["accuracy_threshold_days"] = float64(accuracyDays)
	return out, nil
}

// comebackContext capture les 2 derniers matchs pour décider d'une alerte
// `comeback_welcome` : pause >= 5j entre les 2 derniers + dernier match
// récent (≈ sync qui vient d'aboutir).
type comebackContext struct {
	LastMatchAt    *time.Time
	PrevMatchAt    *time.Time
	HasNewActivity bool
}

// loadComebackContext lit les 2 matchs les plus récents du joueur.
// HasNewActivity = true si le dernier match a moins de freshThreshold de
// décalage par rapport à now. PrevMatchAt sert de référence pour la pause.
func loadComebackContext(ctx context.Context, pdb *duckdb.PlayerDB, now time.Time, freshThreshold time.Duration) (comebackContext, error) {
	out := comebackContext{}
	if pdb == nil || pdb.Player == nil {
		return out, fmt.Errorf("loadComebackContext: player DB not attached")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	sharedDB, release, err := pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return out, fmt.Errorf("loadComebackContext: shared reader: %w", err)
	}
	defer release()
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT mr.start_time
		FROM match_participants mp
		JOIN match_registry mr ON mp.match_id = mr.match_id
		WHERE mp.xuid = ? AND mr.start_time IS NOT NULL
		ORDER BY mr.start_time DESC
		LIMIT 2
	`, pdb.XUID)
	if err != nil {
		return out, fmt.Errorf("query last matches: %w", err)
	}
	defer rows.Close()
	var times []time.Time
	for rows.Next() {
		var t sql.NullTime
		if err := rows.Scan(&t); err != nil {
			return out, err
		}
		if t.Valid {
			times = append(times, t.Time)
		}
	}
	if len(times) >= 1 {
		t := times[0]
		out.LastMatchAt = &t
		if now.Sub(t) <= freshThreshold {
			out.HasNewActivity = true
		}
	}
	if len(times) >= 2 {
		t := times[1]
		out.PrevMatchAt = &t
	}
	return out, rows.Err()
}
