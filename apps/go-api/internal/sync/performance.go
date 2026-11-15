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
	"levelup/go-api/internal/sync/objectiveblend"
	"levelup/go-api/internal/sync/skill"
)

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
	// ObjectiveBlend : faits d'objectif et de soutien du match (paquet objectiveblend), posés
	// après chargement SQL par applyObjectiveFacts. nil = absent : titre sans
	// match_objective_stats (Halo 5), match sans bloc d'objectif.
	ObjectiveBlend *objectiveblend.Facts
	// Chain est la chaîne de score de performance dérivée via GetPerformanceChain
	// (pair_name + flags is_ranked/is_firefight). Toujours non vide.
	Chain string
}

// Paramètres de calcul : toute modification impose de changer skill.PerformanceScoreRev.
const (
	perfWindowSize              = 50    // derniers matchs de la chaîne qui servent de référence
	perfFallbackDurationSeconds = 600.0 // durée supposée d'un match sans temps joué
)

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

func extractMatchMetrics(row *historyRow) *matchMetrics {
	duration := row.TimePlayedSeconds
	if duration <= 0 {
		duration = perfFallbackDurationSeconds
	}

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
		KPM:       analysis.PerMinute(kills, duration),
		DPMDeaths: analysis.PerMinute(deaths, duration),
		APM:       analysis.PerMinute(assists, duration),
		KDA:       kda,
	}
	if row.Accuracy > 0 {
		v := row.Accuracy
		m.Accuracy = &v
	}
	if row.PersonalScore > 0 {
		v := analysis.PerMinute(row.PersonalScore, duration)
		m.PSPM = &v
	}
	if row.DamageDealt > 0 {
		v := analysis.PerMinute(row.DamageDealt, duration)
		m.DPMDamage = &v
	}
	if row.Rank > 0 {
		v := row.Rank
		m.Rank = &v
	}
	if analysis.MMRKnown(row.TeamMMR) {
		v := row.TeamMMR
		m.TeamMMR = &v
	}
	if analysis.MMRKnown(row.EnemyMMR) {
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

// standardPerfMetrics : métriques « plus = mieux », classées par percentile direct (dpm_deaths et
// rank_perf ont leur traitement). SUPERSET des chaînes : le profil de poids de la chaîne
// (WeightsForChain) décide lesquelles comptent.
var standardPerfMetrics = []string{
	MetricKeyKPM, MetricKeyAPM, MetricKeyKDA, MetricKeyAccuracy, MetricKeyPSPM, MetricKeyDPMDamage,
	MetricKeyKillsVsExpected, MetricKeyDeathsVsExpected,
	MetricKeyOffensiveConv, MetricKeyDefensiveResist, MetricKeyMedalExploit,
}

// computeRelativePerformanceScore calcule le score 0-100 d'un match contre l'historique du
// joueur. `weights` est le profil de poids de la CHAÎNE (WeightsForChain) : une métrique hors
// profil n'est ni calculée ni comptée dans la renormalisation.
func computeRelativePerformanceScore(current *historyRow, history []historyRow, weights map[string]float64) *float64 {
	score, ok := weightedPercentileMean(current, history, weights)
	if !ok {
		return nil
	}
	score = roundPerfScore(score)
	return &score
}

// objectiveBlendParams : les réglages de la note à objectif (paquet objectiveblend), fixés avec
// l'utilisateur le 2026-11-15 ; toute modification impose de changer skill.PerformanceScoreRev.
func objectiveBlendParams() objectiveblend.Params {
	return objectiveblend.DefaultParams(MinMatchesForRelative)
}

// computeChainPerformanceScore est la note d'un match selon sa chaîne : le score relatif du profil
// de la chaîne ; sur une chaîne de famille objectif, ce score est le COMBAT sur lequel l'objectif
// et le soutien de l'équipe se greffent (objectiveblend.Compute). Sans faits (Halo 5, match sans
// match_objective_stats) la métrique est absente : la note est le combat, poids renormalisés.
func computeChainPerformanceScore(chain string, current *historyRow, history []historyRow) *float64 {
	if !skill.IsObjectiveChain(chain) {
		return computeRelativePerformanceScore(current, history, WeightsForChain(chain))
	}
	combat, ok := weightedPercentileMean(current, history, WeightsForChain(chain))
	if !ok {
		return nil
	}
	facts := make([]*objectiveblend.Facts, len(history))
	for i := range history {
		facts[i] = history[i].ObjectiveBlend
	}
	score := roundPerfScore(objectiveblend.Compute(combat, current.ObjectiveBlend, facts, objectiveBlendParams()).Score)
	return &score
}

// roundPerfScore arrondit une note au dixième : l'arrondi unique de la note stockée.
func roundPerfScore(score float64) float64 { return math.Round(score*10) / 10 }

// weightedPercentileMean est la note NON ARRONDIE : la moyenne des percentiles des
// métriques du profil `weights`, pondérée par leurs poids et renormalisée par la somme
// des poids RETENUS. ok == false sous MinMatchesForRelative matchs d'historique, ou si
// aucune métrique du profil n'est calculable.
func weightedPercentileMean(current *historyRow, history []historyRow, weights map[string]float64) (float64, bool) {
	if len(history) < MinMatchesForRelative {
		return 0, false
	}
	metrics := extractMatchMetrics(current)
	if metrics == nil {
		return 0, false
	}
	histMetrics := prepareHistoryMetrics(history)
	percentiles, weightsUsed := collectWeightedPercentiles(metrics, histMetrics, weights)
	if len(percentiles) == 0 {
		return 0, false
	}
	totalWeight := 0.0
	for _, w := range weightsUsed {
		totalWeight += w
	}
	if totalWeight <= 0 {
		return 0, false
	}
	score := 0.0
	for k, p := range percentiles {
		score += p * weightsUsed[k]
	}
	return score / totalWeight, true
}

// collectWeightedPercentiles classe le match dans son historique, métrique par métrique (celles
// du profil de sa chaîne) ; rend les percentiles et poids retenus, l'appelant renormalise.
func collectWeightedPercentiles(m *matchMetrics, hist map[string][]float64, weights map[string]float64) (map[string]float64, map[string]float64) {
	percentiles := make(map[string]float64, len(weights))
	weightsUsed := make(map[string]float64, len(weights))

	// Métriques standard (plus = mieux).
	for _, key := range standardPerfMetrics {
		if w, inProfile := profileWeight(weights, key); inProfile {
			addStandardPercentile(m, hist, key, w, percentiles, weightsUsed)
		}
	}

	// Métrique inversée : dpm_deaths (moins = mieux).
	if w, inProfile := profileWeight(weights, MetricKeyDPMDeaths); inProfile {
		if series := hist[MetricKeyDPMDeaths]; len(series) > 0 {
			percentiles[MetricKeyDPMDeaths] = percentileRankInverse(m.DPMDeaths, series)
			weightsUsed[MetricKeyDPMDeaths] = w
		}
	}

	// Rank performance (optionnel).
	if w, inProfile := profileWeight(weights, MetricKeyRankPerf); inProfile {
		addRankPercentile(m, hist, w, percentiles, weightsUsed)
	}
	return percentiles, weightsUsed
}

// profileWeight rend le poids d'une métrique dans le profil de la chaîne courante.
// inProfile=false quand la métrique n'y figure pas ou qu'elle y porte un poids nul : elle n'est// alors ni calculée ni comptée dans la renormalisation.
func profileWeight(weights map[string]float64, key string) (float64, bool) {
	w, inProfile := weights[key]
	return w, inProfile && w != 0
}

// computeRankPerformance calcule le percentile de rank_perf_diff.
func BatchComputePerformanceScores(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string, force bool) (int, error) {
	return batchComputePerformanceScores(ctx, playerDB, sharedDB, xuid, nil, force)
}

// RecomputePerformanceScoresWithMedals recalcule les notes en incluant le bonus
// médailles — la variante que RunBackfillPerf (engine_backfills.go:352) applique
// au backfill, exposée SANS SyncEngine pour les binaires de recompute hors
// serveur (lot 4 du plan PERF_NOTE_OBJECTIFS).
//
// À préférer à BatchComputePerformanceScores pour tout recompute RÉEL : cette
// dernière passe medalExploit nil, donc la métrique medal_exploit y vaut 0 pour
// tous les matchs et les notes stockées divergeraient de celles que produit le
// post-sync (écart mesuré ~1,2-2,2 pt).
//
// metadataDBPath vide ou illisible → dégradation gracieuse (map nil, cf.
// loadMedalExploitMap). playerDB doit être ouvert en RW ; sharedDB est lu.
func RecomputePerformanceScoresWithMedals(ctx context.Context, playerDB, sharedDB *sql.DB,
	metadataDBPath, xuid string, force bool,
) (int, error) {
	medalMap := loadMedalExploitMap(ctx, metadataDBPath, sharedDB, xuid)
	return batchComputePerformanceScores(ctx, playerDB, sharedDB, xuid, medalMap, force)
}

func batchComputePerformanceScores(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string, medalExploitByMatch map[string]float64, force bool) (int, error) {
	allMatches, err := loadHistoryForPerf(ctx, sharedDB, xuid)
	if err != nil {
		return 0, err
	}
	if len(allMatches) == 0 {
		// Univers vide : toute note stockée est orpheline par construction (D-E).
		cleanupAllScoredAsOrphans(ctx, playerDB, xuid, nil)
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
		slog.DebugContext(ctx, "batchComputePerformanceScores: matchs exclus filtrés",
			"xuid", xuid, "filtered", before-len(allMatches), "remaining", len(allMatches))
		if len(allMatches) == 0 {
			// Tous les matchs exclus : idem, plus rien ne qualifie (D-E).
			cleanupAllScoredAsOrphans(ctx, playerDB, xuid, excluded)
			return 0, nil
		}
	}

	// Enrichir avec ce que la requête de l'historique ne porte pas : médailles + objectif.
	if err := applyPerfEnrichments(ctx, sharedDB, xuid, allMatches, medalExploitByMatch); err != nil {
		return 0, fmt.Errorf("batchComputePerformanceScores: %w", err)
	}

	// Charger les matchs qui ont déjà un score ET la chaîne stockée.
	// En mode !force, on skippe si la chaîne stockée correspond à la chaîne
	// actuelle calculée ; si la classification a changé rétroactivement (cas rare),
	// on recompute. Le chargement a lieu dans les DEUX modes (D-E) : la passe de
	// nettoyage des notes orphelines a besoin de l'ensemble des notes stockées à
	// chaque run, force compris.
	existingChain := loadScoredPerfChains(ctx, playerDB, xuid)

	// Historique cumulé par chaîne (chronologique ASC). Pour chaque match, le
	// percentile est calculé sur la fenêtre des 50 derniers entrées de sa chaîne.
	chainHistory := make(map[string][]historyRow)

	// Accumulation en RAM, flush batch en fin de boucle (PostSyncEnrichmentPersister, ADR 0019).
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
		flushErr       error // échec d'écriture des notes, rendu après nettoyage et résumé
		skippedBelow   int   // matchs ignorés car len(history) < MinMatchesPerChainForRelative
		skippedExist   int   // matchs déjà scorés avec la bonne chaîne (mode !force)
		updatedByChain = make(map[string]int)
		totalByChain   = make(map[string]int)
		// Ensembles de la passe de nettoyage (D-D/D-E) : les matchs qui qualifient
		// au terme de ce run, et ceux écartés par le seuil de leur chaîne (cause).
		qualified    = make(map[string]struct{})
		belowMatches = make(map[string]struct{})
	)

	for _, match := range allMatches {
		chain := match.Chain
		totalByChain[chain]++
		history := chainHistory[chain]

		// Le SEUIL prime sur le skip : un match sous le seuil de sa chaîne ne qualifie plus, même
		// avec une note stockée dans la BONNE chaîne (population de référence rétrécie) : sa note
		// est NULLée par la passe de nettoyage (D-D).
		belowThreshold := len(history) < MinMatchesPerChainForRelative

		// Skip si déjà calculé pour la MÊME chaîne (mode !force).
		shouldSkip := false
		if !force && !belowThreshold {
			if stored, ok := existingChain[match.MatchID]; ok && stored == chain {
				shouldSkip = true
				skippedExist++
				qualified[match.MatchID] = struct{}{}
			}
		}

		switch {
		case shouldSkip:
			// nothing to do : la note stockée est à jour (chaîne inchangée).
		case belowThreshold:
			skippedBelow++
			belowMatches[match.MatchID] = struct{}{}
		default:
			// On ne calcule un score qu'après MinMatchesPerChainForRelative matchs dans la chaîne.
			start := len(history) - perfWindowSize
			if start < 0 {
				start = 0
			}
			window := history[start:]

			// Le profil de poids suit la CHAÎNE du match ; les chaînes de famille objectif
			// greffent l'objectif et le soutien de l'équipe sur le combat.
			score := computeChainPerformanceScore(chain, &match, window)
			if score != nil {
				// Accumuler en RAM, flush via PostSyncEnrichmentPersister
				// (1 UPDATE multi-row en fin de boucle).
				pendingUpdates = append(pendingUpdates, perfUpdate{
					MatchID: match.MatchID, Score: *score, Chain: chain,
				})
				qualified[match.MatchID] = struct{}{}
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
			slog.ErrorContext(ctx, "batchComputePerformanceScores: BatchUpdateMulti échoué", "batch_size", len(rows), "err", err)
			execErrors += len(rows)
			updated = 0
			flushErr = fmt.Errorf("batchComputePerformanceScores: écriture des notes: %w", err)
		}
	}

	// Passe auto-nettoyante (D-E) — cf. runPerfCleanupPass.
	cleaned := runPerfCleanupPass(ctx, playerDB, xuid, execErrors, perfCleanupSets{
		scored: existingChain, qualified: qualified, below: belowMatches, excluded: excluded,
	})

	// Résumé final : observabilité de la distribution réelle.
	if updated > 0 || execErrors > 0 || len(totalByChain) > 0 {
		slog.InfoContext(ctx, "batchComputePerformanceScores: batch terminé",
			"xuid", xuid,
			"force", force,
			"total_matches", len(allMatches),
			"updated", updated,
			"updated_by_chain", updatedByChain,
			"total_by_chain", totalByChain,
			"skipped_below_threshold", skippedBelow,
			"skipped_existing", skippedExist,
			"cleaned", cleaned,
			"exec_errors", execErrors,
			"min_per_chain_threshold", MinMatchesPerChainForRelative)
	}
	return updated, flushErr
}
