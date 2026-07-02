package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"levelup/go-api/internal/ctxkeys"
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
