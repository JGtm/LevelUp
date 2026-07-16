// Package api — post_sync_progression_queries.go : queries de support pour
// l'orchestrateur post-sync de la couche progression (V2 Ascension).
//
// Toutes les queries sont read-only. Depuis ADR 0016 (retrait final
// d'attachShared, P2), les queries cross-DB sont scindées en 2 phases :
// la partie shared (match_participants, match_registry) lue via SharedReader,
// la partie player (player_match_enrichment) lue sur pdb.Player, jointure
// faite côté Go.
package wire

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/progression/milestones"
	"levelup/go-api/internal/progression/records"
	"levelup/go-api/internal/progression/streaks"
)

// AccuracyThresholdForDays est le seuil utilisé pour le compte
// accuracy_threshold_days (milestone régularité). Décision §6 :
// « au moins 1 match du jour a accuracy >= 0.50 ».
const AccuracyThresholdForDays = 0.50

// Budgets de la lecture progression. Déclarés en var (pas const) pour être
// raccourcis dans les tests d'unité du retry (cf. post_sync_progression_retry_test.go).
var (
	// progressionLoadBudget borne la lecture complète des matchs de progression
	// (acquisition shared + queries). Généreux car le pipeline tourne hors du
	// chemin user-facing (post-sync / backfill), jamais sur une requête HTTP.
	progressionLoadBudget = 60 * time.Second

	// progressionSharedReadBudget borne l'acquisition résiliente d'une connexion
	// lecture shared (toutes tentatives confondues).
	progressionSharedReadBudget = 45 * time.Second

	// progressionSharedReadAttempt borne UNE tentative Get (le provider peut
	// attendre un retour RO le temps d'un swap ; au-delà on réessaie).
	progressionSharedReadAttempt = 10 * time.Second

	// progressionSharedReadBackoff sépare deux tentatives Get.
	progressionSharedReadBackoff = 500 * time.Millisecond
)

// acquireProgressionSharedRead acquiert une connexion lecture shared, résiliente
// à la fenêtre de swap RO↔RW d'un sync concurrent (ADR 0016) : pendant qu'un
// sync tient le writer RW, un Get lecteur attend le retour RO. Plutôt que de
// renoncer au premier dépassement, on réessaie avec backoff jusqu'à
// progressionSharedReadBudget.
//
// Le ctx est détaché du parent (context.WithoutCancel) : le post-sync est un
// traitement best-effort en arrière-plan qui ne doit pas être tué si le run de
// sync appelant se termine et annule son ctx juste après.
//
// Le caller DOIT appeler la fonction release retournée (typiquement via defer).
// Si err != nil, release est nil.
func acquireProgressionSharedRead(parent context.Context, reader duckdb.SharedReader) (*sql.DB, func(), error) {
	overall, cancelOverall := context.WithTimeout(context.WithoutCancel(parent), progressionSharedReadBudget)
	var lastErr error
	for attempt := 1; ; attempt++ {
		attemptCtx, cancelAttempt := context.WithTimeout(overall, progressionSharedReadAttempt)
		db, release, err := reader.Get(attemptCtx)
		if err == nil {
			return db, func() {
				release()
				cancelAttempt()
				cancelOverall()
			}, nil
		}
		cancelAttempt()
		lastErr = err
		select {
		case <-overall.Done():
			cancelOverall()
			return nil, nil, fmt.Errorf("shared reader unavailable after %d attempt(s): %w", attempt, lastErr)
		case <-time.After(progressionSharedReadBackoff):
		}
	}
}

// loadProgressionMatches lit les matchs récents avec les métriques nécessaires
// aux détecteurs streaks (KDA pour les types perf-based) et records.
//
// La fenêtre = derniers `lookbackDays` jours. Limite par défaut suffisante
// pour couvrir records 90d + streak walkBuckets.
//
// Sortie : 2 vues du même set de matchs — streaks.MatchActivity (timing +
// KDA pour predicates perf) et records.MatchInput (5 métriques pour PB).
type progressionMatchRow struct {
	matchID   string
	startTime time.Time
	kda       float64
	kills     float64
	deaths    float64
	assists   float64
	accuracy  float64
	personal  float64
	timeSec   float64
	dmgDealt  float64
	dmgTaken  float64
}

func loadProgressionMatches(ctx context.Context, pdb *duckdb.PlayerDB, lookbackDays int, now time.Time) ([]streaks.MatchActivity, []records.MatchInput, error) {
	if pdb == nil || pdb.Player == nil {
		return nil, nil, fmt.Errorf("loadProgressionMatches: player DB not attached")
	}
	since := now.AddDate(0, 0, -lookbackDays)
	// Détaché du parent + budget généreux : la résilience à la fenêtre de swap
	// RO↔RW est gérée par acquireProgressionSharedRead ci-dessous.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), progressionLoadBudget)
	defer cancel()

	loaded, matchIDs, err := loadProgressionSharedMatches(ctx, pdb, since)
	if err != nil {
		return nil, nil, err
	}

	perfByMatch, err := loadProgressionPerfScores(ctx, pdb, matchIDs)
	if err != nil {
		return nil, nil, err
	}

	activities, inputs := assembleProgressionResults(loaded, perfByMatch, games.EffectiveHpToKill(pdb.TitleSlug))
	return activities, inputs, nil
}

// loadProgressionSharedMatches charge participants + registry depuis shared via SharedReader.
func loadProgressionSharedMatches(ctx context.Context, pdb *duckdb.PlayerDB, since time.Time) ([]progressionMatchRow, []string, error) {
	sharedDB, release, err := acquireProgressionSharedRead(ctx, pdb.SharedReadDB())
	if err != nil {
		return nil, nil, fmt.Errorf("loadProgressionMatches: %w", err)
	}
	defer release()

	rows, err := sharedDB.QueryContext(ctx, `
		SELECT
			mp.match_id,
			mr.start_time,
			COALESCE(mp.kda, 0) AS kda,
			COALESCE(mp.kills, 0) AS kills,
			COALESCE(mp.deaths, 0) AS deaths,
			COALESCE(mp.assists, 0) AS assists,
			COALESCE(mp.accuracy, 0) AS accuracy,
			COALESCE(mp.personal_score, 0) AS personal_score,
			COALESCE(mp.time_played_seconds, 0) AS time_played_seconds,
			COALESCE(mp.damage_dealt, 0) AS damage_dealt,
			COALESCE(mp.damage_taken, 0) AS damage_taken
		FROM match_participants mp
		JOIN match_registry mr ON mp.match_id = mr.match_id
		WHERE mp.xuid = ? AND mr.start_time >= ?
		ORDER BY mr.start_time ASC
	`, pdb.XUID, since)
	if err != nil {
		return nil, nil, fmt.Errorf("query progression matches: %w", err)
	}
	defer rows.Close()

	var loaded []progressionMatchRow
	matchIDs := make([]string, 0)
	for rows.Next() {
		var (
			r         progressionMatchRow
			startTime sql.NullTime
		)
		if err := rows.Scan(&r.matchID, &startTime, &r.kda, &r.kills, &r.deaths, &r.assists, &r.accuracy, &r.personal, &r.timeSec, &r.dmgDealt, &r.dmgTaken); err != nil {
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
	return loaded, matchIDs, nil
}

// loadProgressionPerfScores charge performance_score depuis player_match_enrichment.
func loadProgressionPerfScores(ctx context.Context, pdb *duckdb.PlayerDB, matchIDs []string) (map[string]float64, error) {
	perfByMatch := make(map[string]float64, len(matchIDs))
	if len(matchIDs) == 0 {
		return perfByMatch, nil
	}
	placeholders := make([]string, len(matchIDs))
	args := make([]any, len(matchIDs))
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	pmeRows, err := pdb.Player.QueryRecovered(ctx, `
		SELECT match_id, COALESCE(performance_score, 0)
		FROM player_match_enrichment_latest
		WHERE match_id IN (`+joinProgressionPlaceholders(placeholders)+`)
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("query performance_score: %w", err)
	}
	defer pmeRows.Close()
	for pmeRows.Next() {
		var mid string
		var perf float64
		if err := pmeRows.Scan(&mid, &perf); err != nil {
			return nil, fmt.Errorf("scan performance_score: %w", err)
		}
		perfByMatch[mid] = perf
	}
	return perfByMatch, nil
}

// assembleProgressionResults compose les 2 vues (activities + inputs) à partir des rows + perf scores.
func assembleProgressionResults(loaded []progressionMatchRow, perfByMatch map[string]float64, effectiveHpToKill float64) ([]streaks.MatchActivity, []records.MatchInput) {
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
		// OC/DR via le calcul canonique analysis.ComputeCombatYieldFloat (K1a :
		// formules produit → analysis/) plutôt qu'un recalcul inline dupliqué.
		// Respecte AssistsExcludedFromYield ; byte-identique en mode par défaut. On
		// ne pose la clé que si la métrique est définie (dmg > 0) — un predicate
		// distingue « absent » de « 0 ».
		cy := analysis.ComputeCombatYieldFloat(r.kills, r.assists, r.dmgDealt, r.dmgTaken, r.deaths, effectiveHpToKill)
		matchStats := map[string]float64{"kda": r.kda}
		if r.dmgDealt > 0 {
			matchStats["oc"] = cy.OffensiveConversion
		}
		if r.dmgTaken > 0 && r.deaths > 0 {
			matchStats["dr"] = cy.DefensiveResistance
		}
		activities = append(activities, streaks.MatchActivity{
			PlayedAt: r.startTime,
			Stats:    matchStats,
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
	return activities, inputs
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
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), progressionLoadBudget)
	defer cancel()

	sharedDB, release, err := acquireProgressionSharedRead(ctx, pdb.SharedReadDB())
	if err != nil {
		return out, fmt.Errorf("loadPlayerStats: %w", err)
	}
	defer release()

	var (
		matchesPlayed int64
		wins          int64
		kills         int64
		headshots     int64
		assists       int64
	)
	// « wins » via le seam d'issues title-aware (K1a) plutôt qu'un `outcome = 2`
	// codé en dur : byte-identique pour halo_infinite, correct pour tout titre au
	// raw_code différent. Slug explicite (pdb.TitleSlug) car le ctx post-sync
	// détaché ne porte pas forcément le titleSlug.
	winExpr := duckdb.OutcomeSQLEqSlug(pdb.TitleSlug, "outcome", canonical.OutcomeWin, "outcome = 2")
	statsQuery := `
		SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN ` + winExpr + ` THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(kills), 0),
			COALESCE(SUM(headshot_kills), 0),
			COALESCE(SUM(assists), 0)
		FROM match_participants
		WHERE xuid = ?
	`
	if err := sharedDB.QueryRowContext(ctx, statsQuery, pdb.XUID).Scan(&matchesPlayed, &wins, &kills, &headshots, &assists); err != nil {
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

	// combat_precision_matches : matchs avec OC >= 0.83 (OffensiveConversionP80).
	// combat_endurance_matches : matchs avec DR >= 1.59 (DefensiveResistanceP80).
	// combat_excellence_matches : matchs avec OC >= 0.83 ET DR >= 1.59.
	const ocP80 = 0.83
	const drP80 = 1.59
	// effectiveHpToKill = baseline PV-pour-tuer du titre (225 Infinite ; 115 Halo 5).
	// Float de confiance issu de la config → injecté via Sprintf (%g), pas une
	// entrée utilisateur. Les seuils P80 restent en bind params (?).
	hp := games.EffectiveHpToKill(pdb.TitleSlug)
	combatMetricsQuery := fmt.Sprintf(`
		SELECT
			COUNT(CASE WHEN damage_dealt > 0
				AND %[1]g * (COALESCE(kills,0) + COALESCE(assists,0)/3.0) / damage_dealt >= ? THEN 1 END),
			COUNT(CASE WHEN damage_taken > 0 AND COALESCE(deaths,0) > 0
				AND damage_taken / (%[1]g * deaths) >= ? THEN 1 END),
			COUNT(CASE WHEN damage_dealt > 0 AND damage_taken > 0 AND COALESCE(deaths,0) > 0
				AND %[1]g * (COALESCE(kills,0) + COALESCE(assists,0)/3.0) / damage_dealt >= ?
				AND damage_taken / (%[1]g * deaths) >= ? THEN 1 END)
		FROM match_participants
		WHERE xuid = ?
	`, hp)
	var precisionMatches, enduranceMatches, excellenceMatches int64
	if err := sharedDB.QueryRowContext(ctx, combatMetricsQuery,
		ocP80, drP80, ocP80, drP80, pdb.XUID).Scan(&precisionMatches, &enduranceMatches, &excellenceMatches); err != nil {
		return out, fmt.Errorf("aggregate combat metrics: %w", err)
	}
	out.Metrics["combat_precision_matches"] = float64(precisionMatches)
	// Endurance / excellence dépendent de la résistance (damage_taken). Un titre
	// sans dégâts subis (ex. Halo 5) ne peut JAMAIS les atteindre (0 à vie) → on
	// n'émet PAS ces métriques pour lui (milestones masqués) plutôt que de les
	// bloquer à 0 inatteignable. La précision (OC) reste : elle ne dépend que des
	// dégâts infligés. Title-agnostic via games.ProvidesDamageTaken (config).
	if games.ProvidesDamageTaken(pdb.TitleSlug) {
		out.Metrics["combat_endurance_matches"] = float64(enduranceMatches)
		out.Metrics["combat_excellence_matches"] = float64(excellenceMatches)
	}
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
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), progressionLoadBudget)
	defer cancel()
	sharedDB, release, err := acquireProgressionSharedRead(ctx, pdb.SharedReadDB())
	if err != nil {
		return out, fmt.Errorf("loadComebackContext: %w", err)
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
