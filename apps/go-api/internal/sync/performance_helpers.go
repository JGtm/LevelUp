package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/persist"
)

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
		h.Chain = GetPerformanceChain(ctxkeys.TitleSlug(ctx), pn, isRanked, isFirefight)
		history = append(history, h)
	}
	if scanErrors > 0 {
		// Cas anormal (schéma divergent ?). On a quand même chargé ce qu'on a pu.
		slog.WarnContext(ctx, "loadHistoryForPerf: scan errors on history rows",
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
// (7 chaînes possibles). Le percentile pondéré est calculé sur les 50 derniers
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

// ── Hygiène des notes orphelines (lot 2 — décisions D-D / D-E) ──────────────
//
// **Pourquoi** : une note stockée survit à la disparition de son match de
// l'univers de calcul. Trois causes constatées sur le corpus (diagnostic
// 2026-08-27) : le match est devenu non terminé (outcome=4, exclu du SQL de
// chargement), il a été exclu manuellement (is_excluded — match_exclusion_service
// ne nettoie pas la note), ou la population de sa chaîne est repassée sous le
// seuil (exclusions, reclassification de famille). D-D tranche : purge sèche,
// score ET chaîne remis à NULL.
//
// **D-E** : le batch est AUTO-NETTOYANT — la passe tourne à chaque run (force
// compris), pas seulement lors d'une purge one-shot.
//
// **Écriture** : append-only (#23046) via PostSyncEnrichmentPersister — une row
// partielle stage='perf' portant NULL. La vue player_match_enrichment_latest
// rend bien ce NULL (merge-on-read PAR GROUPE : « si l'étape propriétaire a une
// row, sa valeur, NULL inclus » — cf. buildPMELatestViewSQL). Aucun UPDATE, aucun
// ON CONFLICT : le vecteur ART reste éliminé.
//
// **Idempotence** : une note NULLée ne matche plus le filtre
// `performance_score IS NOT NULL`, donc le run suivant ne la voit plus et ne
// réécrit rien.

// loadScoredPerfChains charge les matchs qui portent déjà une note, avec la
// chaîne stockée (chaîne NULL → "", note legacy à recalculer).
//
// Chargé dans les DEUX modes (D-E) : en mode !force il pilote le skip des matchs
// déjà à jour ; dans tous les cas il définit l'ensemble des notes stockées que la
// passe de nettoyage confronte à l'ensemble qualifié du run.
//
// Best-effort : une lecture en échec rend une map vide (le batch dégrade en
// recompute-all et ne nettoie rien) — jamais silencieusement, l'erreur est loguée.
func loadScoredPerfChains(ctx context.Context, playerDB *sql.DB, xuid string) map[string]string {
	scored := make(map[string]string)
	rows, err := playerDB.QueryContext(ctx,
		`SELECT match_id, performance_chain
		   FROM player_match_enrichment_latest
		  WHERE performance_score IS NOT NULL`)
	if err != nil {
		slog.WarnContext(ctx, "batchComputePerformanceScores: query existing scores failed — fallback recompute-all, aucun nettoyage",
			"xuid", xuid, "err", err)
		return scored
	}
	defer rows.Close()

	scanErrors := 0
	for rows.Next() {
		var mid string
		var chain sql.NullString
		if err := rows.Scan(&mid, &chain); err != nil {
			scanErrors++
			continue
		}
		if chain.Valid {
			scored[mid] = chain.String
		} else {
			// Score legacy sans chaîne stockée → recompute pour peupler.
			scored[mid] = ""
		}
	}
	if err := rows.Err(); err != nil {
		slog.WarnContext(ctx, "batchComputePerformanceScores: existing rows iteration error",
			"xuid", xuid, "err", err)
	}
	if scanErrors > 0 {
		slog.WarnContext(ctx, "batchComputePerformanceScores: scan errors on existing scores",
			"xuid", xuid, "scan_errors", scanErrors)
	}
	return scored
}

// perfCleanupSets porte les ensembles construits par le run courant, dont la
// confrontation désigne les notes orphelines. `below` et `excluded` ne servent
// qu'à ventiler la cause dans les logs.
type perfCleanupSets struct {
	scored    map[string]string   // notes stockées : match_id → chaîne stockée
	qualified map[string]struct{} // matchs qui qualifient au terme de ce run
	below     map[string]struct{} // matchs sous le seuil de leur chaîne
	excluded  map[string]bool     // matchs is_excluded (filtrés de l'univers)
}

// runPerfCleanupPass exécute la passe auto-nettoyante en fin de batch : toute
// note stockée hors de l'ensemble qualifié du run repasse à NULL (score +
// chaîne). Retourne le nombre de notes nettoyées.
//
// La passe est SAUTÉE si le flush des notes calculées a échoué (execErrors > 0) :
// on n'empile pas d'écritures sur une DB qui vient de refuser la précédente, et
// le prochain run rattrapera le nettoyage.
//
// Best-effort : un échec du nettoyage est logué et n'invalide pas les notes
// calculées par ce run.
func runPerfCleanupPass(ctx context.Context, playerDB *sql.DB, xuid string, execErrors int, sets perfCleanupSets) int {
	if execErrors > 0 {
		slog.WarnContext(ctx, "batchComputePerformanceScores: nettoyage des notes orphelines sauté (flush en erreur)",
			"xuid", xuid, "exec_errors", execErrors)
		return 0
	}
	cleaned, err := nullifyOrphanPerfScores(ctx, playerDB, xuid, sets)
	if err != nil {
		slog.ErrorContext(ctx, "batchComputePerformanceScores: nettoyage des notes orphelines échoué",
			"xuid", xuid, "err", err)
	}
	return cleaned
}

// nullifyOrphanPerfScores remet à NULL (score + chaîne) toute note stockée dont
// le match n'est pas dans l'ensemble qualifié du run. Retourne le nombre de
// lignes nettoyées.
//
// Ce que « non qualifié » couvre, par construction :
//   - outcome=4 (DNF) : exclu par le SQL de loadHistoryForPerf, donc jamais vu
//     par la boucle → jamais qualifié ;
//   - is_excluded : filtré de allMatches en amont ;
//   - sous le seuil de sa chaîne : classé dans `below` par la boucle ;
//   - match disparu de l'univers (purge, re-sync) : plus aucune ligne le porte.
func nullifyOrphanPerfScores(ctx context.Context, playerDB *sql.DB, xuid string, sets perfCleanupSets) (int, error) {
	orphans := make([]string, 0, len(sets.scored))
	var causeBelow, causeExcluded, causeOther int
	for mid := range sets.scored {
		if _, ok := sets.qualified[mid]; ok {
			continue
		}
		orphans = append(orphans, mid)
		switch {
		case sets.excluded[mid]:
			causeExcluded++
		case containsMatch(sets.below, mid):
			causeBelow++
		default:
			// DNF (jamais chargé) ou match sorti de l'univers.
			causeOther++
		}
	}
	if len(orphans) == 0 {
		return 0, nil
	}
	sort.Strings(orphans) // déterminisme des logs et des tests

	rows := make([]persist.EnrichmentMultiColumnUpdate, 0, len(orphans))
	for _, mid := range orphans {
		rows = append(rows, persist.EnrichmentMultiColumnUpdate{
			MatchID: mid,
			Fields: map[string]any{
				"performance_score": nil,
				"performance_chain": nil,
			},
		})
	}
	p := persist.NewPostSyncEnrichmentPersister(playerDB)
	if _, err := p.BatchUpdateMulti(ctx, rows); err != nil {
		return 0, fmt.Errorf("nullifyOrphanPerfScores (%d lignes): %w", len(rows), err)
	}

	slog.InfoContext(ctx, "batchComputePerformanceScores: notes orphelines nettoyées",
		"xuid", xuid,
		"cleaned", len(orphans),
		"cleaned_below_threshold", causeBelow,
		"cleaned_excluded", causeExcluded,
		"cleaned_other", causeOther)
	slog.DebugContext(ctx, "batchComputePerformanceScores: détail des notes nettoyées",
		"xuid", xuid, "match_ids", orphans)
	return len(orphans), nil
}

// containsMatch — appartenance à un ensemble de match_ids.
func containsMatch(set map[string]struct{}, matchID string) bool {
	_, ok := set[matchID]
	return ok
}

// cleanupAllScoredAsOrphans traite les deux sorties anticipées du batch (« aucun
// match qualifiable » : univers vide, ou tous les matchs exclus). Sans elle,
// l'invariant D-E aurait un trou — le run rendrait la main sans regarder les
// notes stockées, qui sont pourtant TOUTES orphelines dans ce cas.
// Best-effort : l'échec est logué, il n'invalide pas le batch.
func cleanupAllScoredAsOrphans(ctx context.Context, playerDB *sql.DB, xuid string, excluded map[string]bool) {
	scored := loadScoredPerfChains(ctx, playerDB, xuid)
	if len(scored) == 0 {
		return
	}
	if _, err := nullifyOrphanPerfScores(ctx, playerDB, xuid, perfCleanupSets{
		scored:    scored,
		qualified: map[string]struct{}{},
		below:     map[string]struct{}{},
		excluded:  excluded,
	}); err != nil {
		slog.ErrorContext(ctx, "batchComputePerformanceScores: nettoyage des notes orphelines échoué (aucun match qualifiable)",
			"xuid", xuid, "err", err)
	}
}
