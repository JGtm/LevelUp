// Package api — post_sync_progression_queries.go : lectures de support pour
// l'orchestrateur post-sync de la couche progression (V2 Ascension).
//
// Les requêtes (read-only) vivent dans duckdb.PostSyncProgressionStore ; ce fichier fixe les budgets de temps,
// les seuils de comptage et compose les vues que les détecteurs consomment. Depuis ADR 0016 (retrait final
// d'attachShared, P2), les lectures cross-DB sont scindées en 2 phases : la partie shared (match_participants,
// match_registry) lue via SharedReader, la partie player (player_match_enrichment) lue sur pdb.Player,
// jointure faite côté Go.
package wire

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/progression/milestones"
	"levelup/go-api/internal/progression/records"
	"levelup/go-api/internal/progression/streaks"
)

// AccuracyThresholdForDays est le seuil utilisé pour le compte
// accuracy_threshold_days (milestone régularité). Décision §6 :
// « au moins 1 match du jour a accuracy >= 0.50 ».
const AccuracyThresholdForDays = 0.50

// progressionLoadBudget borne la lecture complète d'une famille de lectures de progression (acquisition
// shared + requêtes). Généreux car le pipeline tourne hors du chemin user-facing (post-sync / backfill),
// jamais sur une requête HTTP. La résilience à la fenêtre de swap RO↔RW est celle de duckdb.AcquireSharedRead.
const progressionLoadBudget = 60 * time.Second

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
	// Détaché du parent + budget généreux : la résilience à la fenêtre de swap
	// RO↔RW est gérée par duckdb.AcquireSharedRead dans le dépôt.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), progressionLoadBudget)
	defer cancel()

	var reader port.PostSyncProgressionReader = duckdb.NewPostSyncProgressionStore(pdb)
	loaded, err := reader.MatchRows(ctx, since)
	if err != nil {
		return nil, nil, err
	}
	matchIDs := make([]string, 0, len(loaded))
	for _, r := range loaded {
		matchIDs = append(matchIDs, r.MatchID)
	}

	perfByMatch, err := reader.PerformanceScores(ctx, matchIDs)
	if err != nil {
		return nil, nil, err
	}

	activities, inputs := assembleProgressionResults(loaded, perfByMatch, games.EffectiveHpToKill(pdb.TitleSlug))
	return activities, inputs, nil
}

// assembleProgressionResults compose les 2 vues (activities + inputs) à partir des rows + perf scores.
func assembleProgressionResults(loaded []port.ProgressionMatchRow, perfByMatch map[string]float64, effectiveHpToKill float64) ([]streaks.MatchActivity, []records.MatchInput) {
	activities := make([]streaks.MatchActivity, 0, len(loaded))
	inputs := make([]records.MatchInput, 0, len(loaded))
	for _, r := range loaded {
		perfScore := perfByMatch[r.MatchID]
		// Cadences par minute (frags, score personnel) : analysis.PerMinute, 0 quand la durée est inconnue.
		kpm := analysis.PerMinute(r.Kills, r.TimeSeconds)
		pspm := analysis.PerMinute(r.PersonalScore, r.TimeSeconds)
		// OC/DR via le calcul canonique analysis.ComputeCombatYieldFloat (K1a :
		// formules produit → analysis/) plutôt qu'un recalcul inline dupliqué.
		// Respecte AssistsExcludedFromYield ; byte-identique en mode par défaut. On
		// ne pose la clé que si la métrique est définie (dmg > 0) — un predicate
		// distingue « absent » de « 0 ».
		cy := analysis.ComputeCombatYieldFloat(r.Kills, r.Assists, r.DamageDealt, r.DamageTaken, r.Deaths, effectiveHpToKill)
		matchStats := map[string]float64{"kda": r.KDA}
		if r.DamageDealt > 0 {
			matchStats["oc"] = cy.OffensiveConversion
		}
		if r.DamageTaken > 0 && r.Deaths > 0 {
			matchStats["dr"] = cy.DefensiveResistance
		}
		activities = append(activities, streaks.MatchActivity{
			PlayedAt: r.StartTime,
			Stats:    matchStats,
		})
		inputs = append(inputs, records.MatchInput{
			MatchID:  r.MatchID,
			PlayedAt: r.StartTime,
			Metrics: map[records.TrackedMetric]float64{
				records.MetricPerformanceScore: perfScore,
				records.MetricKDA:              r.KDA,
				records.MetricKPM:              kpm,
				records.MetricAccuracy:         r.Accuracy,
				records.MetricPSPM:             pspm,
			},
		})
	}
	return activities, inputs
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

	// Seuils de comptage des MILESTONES combat. Volontairement DISTINCTS des
	// frontières élite analysis.OffensiveConversionP80 (0,90) et
	// analysis.DefensiveResistanceP80 (1,65), qui servent à normaliser l'échelle
	// visuelle des jauges (NormalizeForBar) : un palier de progression doit être
	// atteignable régulièrement, une frontière élite non. Ne PAS « réaligner » ces
	// seuils sur les constantes *P80 — ce serait un changement de difficulté des
	// paliers, pas une correction de cohérence. Source unique des deux seuils de
	// référence : analysis.Combat{OC,DR}ReferenceThreshold.
	//
	// combat_precision_matches  : matchs avec OC >= 0,83.
	// combat_endurance_matches  : matchs avec DR >= 1,59.
	// combat_excellence_matches : matchs avec OC >= 0,83 ET DR >= 1,59.
	agg, err := duckdb.NewPostSyncProgressionStore(pdb).Aggregates(ctx, port.ProgressionAggregateParams{
		AccuracyThreshold: AccuracyThresholdForDays,
		OCThreshold:       analysis.CombatOCReferenceThreshold,
		DRThreshold:       analysis.CombatDRReferenceThreshold,
		// effectiveHpToKill = baseline PV-pour-tuer du titre (225 Infinite ; 115 Halo 5).
		HPToKill: games.EffectiveHpToKill(pdb.TitleSlug),
	})
	if err != nil {
		return out, err
	}

	out.Metrics["matches_played"] = float64(agg.MatchesPlayed)
	out.Metrics["wins"] = float64(agg.Wins)
	out.Metrics["kills"] = float64(agg.Kills)
	out.Metrics["headshots"] = float64(agg.Headshots)
	out.Metrics["assists"] = float64(agg.Assists)
	out.Metrics["accuracy_threshold_days"] = float64(agg.AccuracyDays)
	out.Metrics["combat_precision_matches"] = float64(agg.PrecisionMatches)
	// Endurance / excellence dépendent de la résistance (damage_taken). Un titre
	// sans dégâts subis (ex. Halo 5) ne peut JAMAIS les atteindre (0 à vie) → on
	// n'émet PAS ces métriques pour lui (milestones masqués) plutôt que de les
	// bloquer à 0 inatteignable. La précision (OC) reste : elle ne dépend que des
	// dégâts infligés. Title-agnostic via games.ProvidesDamageTaken (config).
	if games.ProvidesDamageTaken(pdb.TitleSlug) {
		out.Metrics["combat_endurance_matches"] = float64(agg.EnduranceMatches)
		out.Metrics["combat_excellence_matches"] = float64(agg.ExcellenceMatches)
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
	times, err := duckdb.NewPostSyncProgressionStore(pdb).LastMatchTimes(ctx)
	if err != nil && len(times) == 0 {
		return out, err
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
	return out, err
}
