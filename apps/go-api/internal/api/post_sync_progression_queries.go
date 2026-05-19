// Package api — post_sync_progression_queries.go : queries de support pour
// l'orchestrateur post-sync de la couche progression (V2 Ascension).
//
// Toutes les queries sont read-only et opèrent sur le PlayerDB du joueur.
// La connexion `Player` a `shared` attaché → les queries cross-DB peuvent
// se faire via le préfixe `shared.`.
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

	// player_match_enrichment.performance_score est local au joueur.
	// shared.match_participants donne KDA/accuracy/time_played/kills/personal_score.
	// On filtre par xuid sur match_participants pour ne pas tirer les autres
	// joueurs du même match.
	rows, err := pdb.Player.Query(ctx, `
		SELECT
			mp.match_id,
			mr.start_time,
			COALESCE(pme.performance_score, 0) AS performance_score,
			COALESCE(mp.kda, 0) AS kda,
			COALESCE(mp.kills, 0) AS kills,
			COALESCE(mp.accuracy, 0) AS accuracy,
			COALESCE(mp.personal_score, 0) AS personal_score,
			COALESCE(mp.time_played_seconds, 0) AS time_played_seconds
		FROM shared.match_participants mp
		JOIN shared.match_registry mr ON mp.match_id = mr.match_id
		LEFT JOIN player_match_enrichment pme ON pme.match_id = mp.match_id
		WHERE mp.xuid = ? AND mr.start_time >= ?
		ORDER BY mr.start_time ASC
	`, pdb.XUID, since)
	if err != nil {
		return nil, nil, fmt.Errorf("query progression matches: %w", err)
	}
	defer rows.Close()

	var activities []streaks.MatchActivity
	var inputs []records.MatchInput
	for rows.Next() {
		var (
			matchID                       string
			startTime                     sql.NullTime
			perfScore, kda, accuracy, kills, personalScore, timePlayed float64
		)
		if err := rows.Scan(&matchID, &startTime, &perfScore, &kda, &kills, &accuracy, &personalScore, &timePlayed); err != nil {
			return nil, nil, fmt.Errorf("scan progression match: %w", err)
		}
		if !startTime.Valid {
			continue
		}
		kpm := 0.0
		pspm := 0.0
		if timePlayed > 0 {
			minutes := timePlayed / 60
			kpm = kills / minutes
			pspm = personalScore / minutes
		}
		activities = append(activities, streaks.MatchActivity{
			PlayedAt: startTime.Time,
			Stats:    map[string]float64{"kda": kda},
		})
		inputs = append(inputs, records.MatchInput{
			MatchID:  matchID,
			PlayedAt: startTime.Time,
			Metrics: map[records.TrackedMetric]float64{
				records.MetricPerformanceScore: perfScore,
				records.MetricKDA:              kda,
				records.MetricKPM:              kpm,
				records.MetricAccuracy:         accuracy,
				records.MetricPSPM:             pspm,
			},
		})
	}
	return activities, inputs, rows.Err()
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

	var (
		matchesPlayed int64
		wins          int64
		kills         int64
		headshots     int64
		assists       int64
	)
	err := pdb.Player.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN outcome = 2 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(kills), 0),
			COALESCE(SUM(headshot_kills), 0),
			COALESCE(SUM(assists), 0)
		FROM shared.match_participants
		WHERE xuid = ?
	`, pdb.XUID).Scan(&matchesPlayed, &wins, &kills, &headshots, &assists)
	if err != nil {
		return out, fmt.Errorf("aggregate stats: %w", err)
	}

	// accuracy_threshold_days : COUNT(DISTINCT DATE(start_time)) where any match >= threshold.
	var accuracyDays int64
	if err := pdb.Player.QueryRow(ctx, `
		SELECT COUNT(DISTINCT CAST(mr.start_time AS DATE))
		FROM shared.match_participants mp
		JOIN shared.match_registry mr ON mp.match_id = mr.match_id
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
	rows, err := pdb.Player.Query(ctx, `
		SELECT mr.start_time
		FROM shared.match_participants mp
		JOIN shared.match_registry mr ON mp.match_id = mr.match_id
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
