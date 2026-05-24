// Package sync — performance.go : calcul du score de performance relatif (0-100).
//
// Portage de src/analysis/_performance_relative.py + src/data/sync/_performance.py.
// Compare la performance d'un match à l'historique personnel du joueur.
//   - 50 = match dans ta moyenne
//   - 100 = meilleur match de ton historique
//   - 0 = pire match de ton historique
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"os"
	"sort"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/persist"
)

// ── Types ───────────────────────────────────────────────────────────────────

// historyRow contient les métriques d'un match historique pour le calcul de perf.
type historyRow struct {
	MatchID           string
	Kills             float64
	Deaths            float64
	Assists           float64
	KDA               float64
	Accuracy          float64
	TimePlayedSeconds float64
	PersonalScore     float64
	DamageDealt       float64
	DamageTaken       float64
	Rank              float64
	TeamMMR           float64
	EnemyMMR          float64
	KillsExpected     float64
	DeathsExpected    float64
	// Métriques dérivées (per-minute)
	KPM              float64
	DPMDeaths        float64
	APM              float64
	PSPM             float64
	DPMDamage        float64
	KillsVsExpected  float64
	DeathsVsExpected float64
	// Métriques combat yield v5 (ADR-0006) — enrichies après chargement SQL
	OffensiveConversion float64
	DefensiveResistance float64
	MedalExploitScore   float64
	// Chain est la chaîne de score de performance dérivée via GetPerformanceChain
	// (pair_name + flags is_ranked/is_firefight). Toujours non vide.
	Chain string
}

// matchMetrics contient les métriques per-minute d'un match unique.
type matchMetrics struct {
	KPM                 float64
	DPMDeaths           float64
	APM                 float64
	KDA                 float64
	Accuracy            *float64
	PSPM                *float64
	DPMDamage           *float64
	Rank                *float64
	TeamMMR             *float64
	EnemyMMR            *float64
	KillsVsExpected     *float64
	DeathsVsExpected    *float64
	OffensiveConversion *float64
	DefensiveResistance *float64
	MedalExploit        *float64
}

// ── Extraction des métriques ────────────────────────────────────────────────

func extractMatchMetrics(row *historyRow) *matchMetrics {
	duration := row.TimePlayedSeconds
	if duration <= 0 {
		duration = 600.0
	}
	minutes := duration / 60.0

	kills := row.Kills
	deaths := row.Deaths
	assists := row.Assists

	// KDA canonique (ADR 0006) — fallback si row.KDA absent (valeur 0).
	kda := row.KDA
	if kda == 0 {
		kda = analysis.KDA(int(kills), int(assists), int(deaths))
	}

	m := &matchMetrics{
		KPM:       kills / minutes,
		DPMDeaths: deaths / minutes,
		APM:       assists / minutes,
		KDA:       kda,
	}
	if row.Accuracy > 0 {
		v := row.Accuracy
		m.Accuracy = &v
	}
	if row.PersonalScore > 0 {
		v := row.PersonalScore / minutes
		m.PSPM = &v
	}
	if row.DamageDealt > 0 {
		v := row.DamageDealt / minutes
		m.DPMDamage = &v
	}
	if row.Rank > 0 {
		v := row.Rank
		m.Rank = &v
	}
	if row.TeamMMR > 0 {
		v := row.TeamMMR
		m.TeamMMR = &v
	}
	if row.EnemyMMR > 0 {
		v := row.EnemyMMR
		m.EnemyMMR = &v
	}
	if row.KillsExpected > 0 {
		v := kills / row.KillsExpected
		m.KillsVsExpected = &v
	}
	if row.DeathsExpected > 0 && deaths > 0 {
		v := row.DeathsExpected / math.Max(1.0, deaths)
		m.DeathsVsExpected = &v
	}
	// Combat yield v5 — précomputé dans historyRow
	if row.OffensiveConversion > 0 {
		v := row.OffensiveConversion
		m.OffensiveConversion = &v
	}
	if row.DefensiveResistance > 0 {
		v := row.DefensiveResistance
		m.DefensiveResistance = &v
	}
	if row.MedalExploitScore > 0 {
		v := row.MedalExploitScore
		m.MedalExploit = &v
	}
	return m
}

// ── Percentiles ─────────────────────────────────────────────────────────────

// percentileRank calcule le percentile d'une valeur dans une série triée (0-100).
func percentileRank(value float64, series []float64) float64 {
	if len(series) < 2 {
		return 50.0
	}
	count := 0
	for _, v := range series {
		if v <= value {
			count++
		}
	}
	return clampF(float64(count)/float64(len(series))*100.0, 0, 100)
}

// percentileRankInverse calcule le percentile inversé (moins = mieux).
func percentileRankInverse(value float64, series []float64) float64 {
	if len(series) < 2 {
		return 50.0
	}
	count := 0
	for _, v := range series {
		if v >= value {
			count++
		}
	}
	return clampF(float64(count)/float64(len(series))*100.0, 0, 100)
}

// ── Calcul du score de performance relatif ──────────────────────────────────

// computeRelativePerformanceScore calcule le score 0-100 d'un match
// par rapport à l'historique du joueur.
func computeRelativePerformanceScore(current *historyRow, history []historyRow) *float64 {
	if len(history) < MinMatchesForRelative {
		return nil
	}

	metrics := extractMatchMetrics(current)
	if metrics == nil {
		return nil
	}

	// Préparer les séries historiques par métrique.
	histMetrics := prepareHistoryMetrics(history)

	percentiles := make(map[string]float64)
	weightsUsed := make(map[string]float64)

	// Métriques standard (plus = mieux)
	standardMetrics := []string{
		MetricKeyKPM, MetricKeyAPM, MetricKeyKDA, MetricKeyAccuracy, MetricKeyPSPM, MetricKeyDPMDamage,
		MetricKeyKillsVsExpected, MetricKeyDeathsVsExpected,
		MetricKeyOffensiveConv, MetricKeyDefensiveResist, MetricKeyMedalExploit,
	}
	for _, key := range standardMetrics {
		val, ok := getMetricValue(metrics, key)
		if !ok {
			continue
		}
		series, ok2 := histMetrics[key]
		if !ok2 || len(series) == 0 {
			continue
		}
		percentiles[key] = percentileRank(val, series)
		weightsUsed[key] = RelativeWeights[key]
	}

	// Métrique inversée : dpm_deaths (moins = mieux)
	if series, ok := histMetrics[MetricKeyDPMDeaths]; ok && len(series) > 0 {
		percentiles[MetricKeyDPMDeaths] = percentileRankInverse(metrics.DPMDeaths, series)
		weightsUsed[MetricKeyDPMDeaths] = RelativeWeights[MetricKeyDPMDeaths]
	}

	// Rank performance (optionnel)
	if metrics.Rank != nil && metrics.TeamMMR != nil && metrics.EnemyMMR != nil {
		rankPerf := computeRankPerformance(*metrics.Rank, *metrics.TeamMMR, *metrics.EnemyMMR, histMetrics)
		if rankPerf != nil {
			percentiles[MetricKeyRankPerf] = *rankPerf
			weightsUsed[MetricKeyRankPerf] = RelativeWeights[MetricKeyRankPerf]
		}
	}

	if len(percentiles) == 0 {
		return nil
	}

	totalWeight := 0.0
	for _, w := range weightsUsed {
		totalWeight += w
	}
	if totalWeight <= 0 {
		return nil
	}

	score := 0.0
	for k, p := range percentiles {
		score += p * weightsUsed[k]
	}
	score /= totalWeight
	score = math.Round(score*10) / 10
	return &score
}

// computeRankPerformance calcule le percentile de rank_perf_diff.
func computeRankPerformance(rank, teamMMR, enemyMMR float64, histMetrics map[string][]float64) *float64 {
	deltaMMR := teamMMR - enemyMMR
	expectedRank := 4.5 - (deltaMMR/100.0)*0.5
	diff := expectedRank - rank // positif = mieux que prévu
	series, ok := histMetrics["rank_perf_diff"]
	if !ok || len(series) == 0 {
		return nil
	}
	p := percentileRank(diff, series)
	return &p
}

// getMetricValue extrait la valeur d'une métrique par clé.
func getMetricValue(m *matchMetrics, key string) (float64, bool) {
	switch key {
	case MetricKeyKPM:
		return m.KPM, true
	case MetricKeyDPMDeaths:
		return m.DPMDeaths, true
	case MetricKeyAPM:
		return m.APM, true
	case MetricKeyKDA:
		return m.KDA, true
	case MetricKeyAccuracy:
		if m.Accuracy != nil {
			return *m.Accuracy, true
		}
		return 0, false
	case MetricKeyPSPM:
		if m.PSPM != nil {
			return *m.PSPM, true
		}
		return 0, false
	case MetricKeyDPMDamage:
		if m.DPMDamage != nil {
			return *m.DPMDamage, true
		}
		return 0, false
	case MetricKeyKillsVsExpected:
		if m.KillsVsExpected != nil {
			return *m.KillsVsExpected, true
		}
		return 0, false
	case MetricKeyDeathsVsExpected:
		if m.DeathsVsExpected != nil {
			return *m.DeathsVsExpected, true
		}
		return 0, false
	case MetricKeyOffensiveConv:
		if m.OffensiveConversion != nil {
			return *m.OffensiveConversion, true
		}
		return 0, false
	case MetricKeyDefensiveResist:
		if m.DefensiveResistance != nil {
			return *m.DefensiveResistance, true
		}
		return 0, false
	case MetricKeyMedalExploit:
		if m.MedalExploit != nil {
			return *m.MedalExploit, true
		}
		return 0, false
	}
	return 0, false
}

// prepareHistoryMetrics calcule les séries de métriques per-minute à partir de l'historique.
func prepareHistoryMetrics(history []historyRow) map[string][]float64 {
	n := len(history)
	result := map[string][]float64{
		MetricKeyKPM:              make([]float64, 0, n),
		MetricKeyDPMDeaths:        make([]float64, 0, n),
		MetricKeyAPM:              make([]float64, 0, n),
		MetricKeyKDA:              make([]float64, 0, n),
		MetricKeyAccuracy:         make([]float64, 0, n),
		MetricKeyPSPM:             make([]float64, 0, n),
		MetricKeyDPMDamage:        make([]float64, 0, n),
		"rank_perf_diff":          make([]float64, 0, n),
		MetricKeyKillsVsExpected:  make([]float64, 0, n),
		MetricKeyDeathsVsExpected: make([]float64, 0, n),
		MetricKeyOffensiveConv:    make([]float64, 0, n),
		MetricKeyDefensiveResist:  make([]float64, 0, n),
		MetricKeyMedalExploit:     make([]float64, 0, n),
	}

	for _, row := range history {
		m := extractMatchMetrics(&row)
		if m == nil {
			continue
		}
		result[MetricKeyKPM] = append(result[MetricKeyKPM], m.KPM)
		result[MetricKeyDPMDeaths] = append(result[MetricKeyDPMDeaths], m.DPMDeaths)
		result[MetricKeyAPM] = append(result[MetricKeyAPM], m.APM)
		result[MetricKeyKDA] = append(result[MetricKeyKDA], m.KDA)
		if m.Accuracy != nil {
			result[MetricKeyAccuracy] = append(result[MetricKeyAccuracy], *m.Accuracy)
		}
		if m.PSPM != nil {
			result[MetricKeyPSPM] = append(result[MetricKeyPSPM], *m.PSPM)
		}
		if m.DPMDamage != nil {
			result[MetricKeyDPMDamage] = append(result[MetricKeyDPMDamage], *m.DPMDamage)
		}
		if m.Rank != nil && m.TeamMMR != nil && m.EnemyMMR != nil {
			delta := *m.TeamMMR - *m.EnemyMMR
			expected := 4.5 - (delta/100.0)*0.5
			result["rank_perf_diff"] = append(result["rank_perf_diff"], expected-*m.Rank)
		}
		if m.KillsVsExpected != nil {
			result[MetricKeyKillsVsExpected] = append(result[MetricKeyKillsVsExpected], *m.KillsVsExpected)
		}
		if m.DeathsVsExpected != nil {
			result[MetricKeyDeathsVsExpected] = append(result[MetricKeyDeathsVsExpected], *m.DeathsVsExpected)
		}
		if m.OffensiveConversion != nil {
			result[MetricKeyOffensiveConv] = append(result[MetricKeyOffensiveConv], *m.OffensiveConversion)
		}
		if m.DefensiveResistance != nil {
			result[MetricKeyDefensiveResist] = append(result[MetricKeyDefensiveResist], *m.DefensiveResistance)
		}
		if m.MedalExploit != nil {
			result[MetricKeyMedalExploit] = append(result[MetricKeyMedalExploit], *m.MedalExploit)
		}
	}

	// Trier chaque série pour les calculs de percentile.
	for k, s := range result {
		sort.Float64s(s)
		result[k] = s
	}
	return result
}

// ── Batch compute ───────────────────────────────────────────────────────────

// loadHistoryForPerf charge tous les matchs du joueur depuis shared DB pour le batch.
// Le champ damage_taken est utilisé pour defensive_resistance (combat yield v5).
// Chaque row est classée dans une chaîne via GetPerformanceChain (jamais vide).
func loadHistoryForPerf(ctx context.Context, sharedDB *sql.DB, xuid string) ([]historyRow, error) {
	rows, err := sharedDB.QueryContext(ctx, `
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
			mr.pair_name, COALESCE(mr.is_ranked, FALSE), COALESCE(mr.is_firefight, FALSE)
		FROM match_registry mr
		JOIN match_participants mp ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?
		  AND mr.start_time IS NOT NULL
		  AND COALESCE(mp.outcome, 0) != 4
		ORDER BY mr.start_time ASC`, xuid)
	if err != nil {
		return nil, fmt.Errorf("loadHistoryForPerf: %w", err)
	}
	defer rows.Close()

	var (
		history    []historyRow
		scanErrors int
	)
	for rows.Next() {
		var h historyRow
		var startTime time.Time
		var pairName sql.NullString
		var isRanked, isFirefight bool
		if err := rows.Scan(
			&h.MatchID, &startTime,
			&h.Kills, &h.Deaths, &h.Assists, &h.KDA,
			&h.Accuracy, &h.TimePlayedSeconds,
			&h.PersonalScore, &h.DamageDealt, &h.DamageTaken,
			&h.Rank, &h.TeamMMR, &h.EnemyMMR,
			&h.KillsExpected, &h.DeathsExpected,
			&pairName, &isRanked, &isFirefight,
		); err != nil {
			scanErrors++
			continue
		}
		// Calcul offline des métriques combat yield (pas de DB supplémentaire requise)
		h.OffensiveConversion, h.DefensiveResistance = computeCombatYield(lusrMatchData{
			Kills: h.Kills, Deaths: h.Deaths, Assists: h.Assists,
			DamageDealt: h.DamageDealt, DamageTaken: h.DamageTaken,
		})
		// Classification en chaîne — toujours non vide grâce au fallback arena_slayer.
		pn := ""
		if pairName.Valid {
			pn = pairName.String
		}
		h.Chain = GetPerformanceChain(pn, isRanked, isFirefight)
		history = append(history, h)
	}
	if scanErrors > 0 {
		// Cas anormal (schéma divergent ?). On a quand même chargé ce qu'on a pu.
		slog.Warn("loadHistoryForPerf: scan errors on history rows",
			"xuid", xuid, "scan_errors", scanErrors, "loaded_rows", len(history))
	}
	return history, rows.Err()
}

// batchComputePerformanceScores calcule les performance_score manquants ou tous si force=true.
// medalExploitByMatch : match_id → score médailles (nil = pas de données médailles).
// force : recalcule même si performance_score est déjà présent dans la même chaîne.
// Retourne le nombre de matchs mis à jour.
//
// Sémantique : chaque match est rattaché à une chaîne via GetPerformanceChain
// (6 chaînes possibles). Le percentile pondéré est calculé sur les 50 derniers
// matchs de la **même chaîne** uniquement. Un score n'est calculé qu'à partir
// du MinMatchesPerChainForRelative-ième match de la chaîne (pas de fallback
// global, préservation de la sémantique "relatif à ta chaîne").
// BatchComputePerformanceScores is the public entry point that recomputes
// the relative performance score for every match of a player. Exposed so
// that external callers (e.g. the OpenSpartan post-import service) can run
// the same recompute pass as the sync engine after a bulk import.
//
// medalExploit override is set to nil — callers that need the exploit-aware
// variant (sync engine) still use the unexported helper.
func BatchComputePerformanceScores(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string, force bool) (int, error) {
	return batchComputePerformanceScores(ctx, playerDB, sharedDB, xuid, nil, force)
}

func batchComputePerformanceScores(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string, medalExploitByMatch map[string]float64, force bool) (int, error) {
	allMatches, err := loadHistoryForPerf(ctx, sharedDB, xuid)
	if err != nil {
		return 0, err
	}
	if len(allMatches) == 0 {
		return 0, nil
	}

	// Filtrer les matchs marqués `is_excluded` côté playerDB : ils ne doivent ni
	// peser dans la fenêtre glissante ni recevoir de score.
	excluded, err := loadExcludedMatchIDs(ctx, playerDB)
	if err != nil {
		return 0, fmt.Errorf("batchComputePerformanceScores: %w", err)
	}
	if len(excluded) > 0 {
		before := len(allMatches)
		filtered := allMatches[:0]
		for _, m := range allMatches {
			if excluded[m.MatchID] {
				continue
			}
			filtered = append(filtered, m)
		}
		allMatches = filtered
		slog.Debug("batchComputePerformanceScores: matchs exclus filtrés",
			"xuid", xuid, "filtered", before-len(allMatches), "remaining", len(allMatches))
		if len(allMatches) == 0 {
			return 0, nil
		}
	}

	// Enrichir avec les scores médailles
	for i := range allMatches {
		if score, ok := medalExploitByMatch[allMatches[i].MatchID]; ok {
			allMatches[i].MedalExploitScore = score
		}
	}

	// Charger les matchs qui ont déjà un score ET la chaîne stockée (ignoré en mode force).
	// On skippe uniquement si la chaîne déjà stockée correspond à la chaîne actuelle calculée.
	// Si la classification a changé rétroactivement (cas rare), on recompute.
	existingChain := make(map[string]string)
	if !force {
		existRows, queryErr := playerDB.QueryContext(ctx,
			`SELECT match_id, performance_chain
			   FROM player_match_enrichment
			  WHERE performance_score IS NOT NULL`)
		if queryErr != nil {
			// Non bloquant : on continue en recompute-tout, mais on signale.
			slog.Warn("batchComputePerformanceScores: query existing scores failed — fallback recompute-all",
				"xuid", xuid, "err", queryErr)
		} else {
			defer existRows.Close()
			scanErrors := 0
			for existRows.Next() {
				var mid string
				var chain sql.NullString
				if err := existRows.Scan(&mid, &chain); err != nil {
					scanErrors++
					continue
				}
				if chain.Valid {
					existingChain[mid] = chain.String
				} else {
					// Score legacy sans chaîne stockée → recompute pour peupler.
					existingChain[mid] = ""
				}
			}
			if err := existRows.Err(); err != nil {
				slog.Warn("batchComputePerformanceScores: existing rows iteration error",
					"xuid", xuid, "err", err)
			}
			if scanErrors > 0 {
				slog.Warn("batchComputePerformanceScores: scan errors on existing scores",
					"xuid", xuid, "scan_errors", scanErrors)
			}
		}
	}

	const windowSize = 50
	now := time.Now().UTC()

	// Historique cumulé par chaîne (chronologique ASC). Pour chaque match, le
	// percentile est calculé sur la fenêtre des 50 derniers entrées de sa chaîne.
	chainHistory := make(map[string][]historyRow)

	// Phase 4.7 closure (2026-05-24) : default flipé ON. Accumulation in-RAM
	// + flush batch en fin de boucle. Set LEVELUP_POSTSYNC_INSERT_ONLY=0 pour
	// fallback legacy UPDATE-then-INSERT row-by-row (mode dégradé).
	batchMode := os.Getenv("LEVELUP_POSTSYNC_INSERT_ONLY") != "0"
	type perfUpdate struct {
		MatchID string
		Score   float64
		Chain   string
	}
	var pendingUpdates []perfUpdate

	// Stats par chaîne pour observabilité (utile pour diagnostiquer "pourquoi si peu
	// de matchs scorés ?" — distribution réelle des chaînes pour le joueur).
	var (
		updated        int
		execErrors     int
		skippedBelow   int // matchs ignorés car len(history) < MinMatchesPerChainForRelative
		skippedExist   int // matchs déjà scorés avec la bonne chaîne (mode !force)
		updatedByChain = make(map[string]int)
		totalByChain   = make(map[string]int)
	)

	for _, match := range allMatches {
		chain := match.Chain
		totalByChain[chain]++
		history := chainHistory[chain]

		// Skip si déjà calculé pour la MÊME chaîne (mode !force).
		shouldSkip := false
		if !force {
			if stored, ok := existingChain[match.MatchID]; ok && stored == chain {
				shouldSkip = true
				skippedExist++
			}
		}

		switch {
		case shouldSkip:
			// nothing to do
		case len(history) < MinMatchesPerChainForRelative:
			skippedBelow++
		default:
			// On ne calcule un score qu'après MinMatchesPerChainForRelative matchs dans la chaîne.
			start := len(history) - windowSize
			if start < 0 {
				start = 0
			}
			window := history[start:]

			score := computeRelativePerformanceScore(&match, window)
			if score != nil {
				// Phase 4.4 batch mode : accumuler en RAM, flush en fin de boucle.
				if batchMode {
					pendingUpdates = append(pendingUpdates, perfUpdate{
						MatchID: match.MatchID, Score: *score, Chain: chain,
					})
					updated++
					updatedByChain[chain]++
					chainHistory[chain] = append(history, match)
					continue
				}
				// Legacy path (LEVELUP_POSTSYNC_INSERT_ONLY=0) — UPSERT direct.
				// Phase 4.7 revert acad4603 : retour au pattern standard
				// ON CONFLICT DO UPDATE. Le bug ART est éliminé sur le path
				// default (batch mode), donc plus besoin du workaround
				// UPDATE-then-INSERT sur le path legacy.
				_, err := playerDB.ExecContext(ctx, `
					INSERT INTO player_match_enrichment (match_id, performance_score, performance_chain, updated_at)
					VALUES (?, ?, ?, ?)
					ON CONFLICT (match_id) DO UPDATE SET
						performance_score = EXCLUDED.performance_score,
						performance_chain = EXCLUDED.performance_chain,
						updated_at        = EXCLUDED.updated_at`,
					match.MatchID, *score, chain, now)
				if err != nil {
					execErrors++
					slog.Warn("batchComputePerformanceScores: UPSERT failed",
						"match_id", match.MatchID, "chain", chain, "xuid", xuid, "err", err)
				} else {
					updated++
					updatedByChain[chain]++
				}
			}
		}

		// Pousser le match courant dans l'historique de sa chaîne (toujours, même skip).
		chainHistory[chain] = append(history, match)
	}

	// Phase 4.4 batch mode : flush des updates accumulés en 1 single SQL.
	if batchMode && len(pendingUpdates) > 0 {
		rows := make([]persist.EnrichmentMultiColumnUpdate, 0, len(pendingUpdates))
		for _, u := range pendingUpdates {
			rows = append(rows, persist.EnrichmentMultiColumnUpdate{
				MatchID: u.MatchID,
				Fields: map[string]any{
					"performance_score": u.Score,
					"performance_chain": u.Chain,
				},
			})
		}
		p := persist.NewPostSyncEnrichmentPersister(playerDB)
		if err := p.BatchUpdateMulti(ctx, rows); err != nil {
			slog.ErrorContext(ctx, "batchComputePerformanceScores: BatchUpdateMulti échoué",
				"batch_size", len(rows), "err", err)
			execErrors += len(rows)
			updated = 0
		}
	}

	// Résumé final : observabilité de la distribution réelle.
	if updated > 0 || execErrors > 0 || len(totalByChain) > 0 {
		slog.Info("batchComputePerformanceScores: batch terminé",
			"xuid", xuid,
			"force", force,
			"total_matches", len(allMatches),
			"updated", updated,
			"updated_by_chain", updatedByChain,
			"total_by_chain", totalByChain,
			"skipped_below_threshold", skippedBelow,
			"skipped_existing", skippedExist,
			"exec_errors", execErrors,
			"min_per_chain_threshold", MinMatchesPerChainForRelative)
	}
	return updated, nil
}
