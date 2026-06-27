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

	// Efficacité de combat (kills+assists)/max(1,deaths) — métrique INTERNE du
	// perf score, toujours recalculée ici. Ce n'est PAS le KDA affiché. On ne
	// lit plus row.KDA car il porte désormais le KDA natif Halo
	// (Kills+Assists/3−Deaths, possiblement négatif) destiné à l'affichage du
	// FDA, pas au scoring.
	kda := analysis.CombatEfficiency(int(kills), int(assists), int(deaths))

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
			   FROM player_match_enrichment_latest
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

	// Historique cumulé par chaîne (chronologique ASC). Pour chaque match, le
	// percentile est calculé sur la fenêtre des 50 derniers entrées de sa chaîne.
	chainHistory := make(map[string][]historyRow)

	// Phase 3 du refactor ART : seul le batch path est conservé (le legacy
	// ON CONFLICT DO UPDATE déclenchait le bug ART). Accumulation in-RAM
	// + flush batch en fin de boucle via PostSyncEnrichmentPersister.
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
				// Accumuler en RAM, flush via PostSyncEnrichmentPersister
				// (1 UPDATE multi-row en fin de boucle).
				pendingUpdates = append(pendingUpdates, perfUpdate{
					MatchID: match.MatchID, Score: *score, Chain: chain,
				})
				updated++
				updatedByChain[chain]++
				chainHistory[chain] = append(history, match)
				continue
			}
		}

		// Pousser le match courant dans l'historique de sa chaîne (toujours, même skip).
		chainHistory[chain] = append(history, match)
	}

	// Flush des updates accumulés en 1 single SQL.
	if len(pendingUpdates) > 0 {
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
		if _, err := p.BatchUpdateMulti(ctx, rows); err != nil {
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
