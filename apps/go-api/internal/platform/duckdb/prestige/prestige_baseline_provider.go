// Package duckdb — BaselineProvider pour le titre Halo Infinite.
//
// Implémente prestige.BaselineProvider en lisant les match_participants
// existants depuis shared_matches_v2.duckdb et la métrique demandée.

package prestige

import (
	"context"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/prestige"
)

// Noms de colonnes match_participants utilisés par mapMetricToColumn.
// Centralisés pour éviter la duplication littérale (lint goconst).
const (
	metricColAccuracy = "accuracy"
	metricColKills    = "kills"
	metricColDeaths   = "deaths"
)

// HaloBaselineProvider lit les matchs récents d'un joueur Halo Infinite
// pour fournir baseline + percentile au module Prestige.
//
// La complexité réelle (filtres mode/playlist, jointures avec medals) est
// gardée minimale pour Phase 4. Phase 5/6 affineront selon les besoins UI.
//
// reçoit un duckdb.SharedReader (pas un *duckdb.DB) pour coordonner
// avec le SharedDBProvider (cycle RO↔RW).
type HaloBaselineProvider struct {
	reader duckdb.SharedReader
}

// NewHaloBaselineProvider construit le provider depuis un duckdb.SharedReader.
//
// Côté caller : passer pdb.SharedReadDB() pour bénéficier du Provider B-swap
// (fallback transparent vers LegacySharedReader(pdb.Shared) si Provider absent).
func NewHaloBaselineProvider(reader duckdb.SharedReader) *HaloBaselineProvider {
	return &HaloBaselineProvider{reader: reader}
}

// Compile-time assertion.
var _ prestige.BaselineProvider = (*HaloBaselineProvider)(nil)

// RecentMatches retourne les N derniers matchs PvP du joueur avec la métrique demandée.
//
// Mapping des FieldKey canoniques vers les colonnes de match_participants.
// Les métriques cumulées (medal:*, maps_played_distinct, etc.) ne sont pas
// gérées ici — l'evaluator les agrège via une requête dédiée Phase 5.
func (p *HaloBaselineProvider) RecentMatches(ctx context.Context, userID, _ string, metric string, window int) ([]prestige.MatchData, error) {
	col := mapMetricToColumn(metric)
	if col == "" {
		// Métrique non mappée : pas d'erreur, juste pas de matchs.
		// L'evaluator interprétera les []MatchData vides comme "pas de progrès".
		return nil, nil
	}
	if window <= 0 {
		window = 20
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	q := fmt.Sprintf(`
		SELECT mp.match_id, mr.start_time, %s
		FROM match_participants mp
		JOIN match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?
		ORDER BY mr.start_time DESC
		LIMIT ?
	`, col)

	db, release, err := p.reader.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("HaloBaselineProvider.RecentMatches: shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, q, userID, window)
	if err != nil {
		return nil, fmt.Errorf("HaloBaselineProvider.RecentMatches: %w", err)
	}
	defer rows.Close()

	var out []prestige.MatchData
	for rows.Next() {
		var m prestige.MatchData
		if err := rows.Scan(&m.MatchID, &m.StartedAt, &m.MetricValue); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// PopulationPercentile renvoie la position approximative de la cible
// dans la population active sur la métrique.
//
// Phase 4 : implémentation simpliste qui retourne (0.5, 0). popSize=0
// désactive le cap population côté palier, ce qui est cohérent tant
// qu'on n'a pas un vrai calcul agrégé. Phase 5/6 : remplacer par une
// vraie agrégation (peut-être via un rollup périodique).
func (p *HaloBaselineProvider) PopulationPercentile(ctx context.Context, _ string, metric string, target float64) (float64, int, error) {
	col := mapMetricToColumn(metric)
	if col == "" {
		return 0.5, 0, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Approximation : compter combien de joueurs ont une moyenne ≥ target
	// sur la métrique demandée (sur les 30 derniers jours).
	q := fmt.Sprintf(`
		WITH player_avgs AS (
			SELECT mp.xuid, AVG(%s) AS avg_metric, COUNT(*) AS n
			FROM match_participants mp
			JOIN match_registry mr ON mr.match_id = mp.match_id
			WHERE mr.start_time > NOW() - INTERVAL 30 DAY
			GROUP BY mp.xuid
			HAVING COUNT(*) >= 5
		)
		SELECT
			COALESCE(SUM(CASE WHEN avg_metric < ? THEN 1 ELSE 0 END), 0) AS below,
			COUNT(*) AS total
		FROM player_avgs
	`, col)

	db, release, err := p.reader.Get(ctx)
	if err != nil {
		// Erreur non bloquante : retourner valeurs neutres (cap désactivé).
		return 0.5, 0, nil
	}
	defer release()

	var below, total int
	if err := db.QueryRowContext(ctx, q, target).Scan(&below, &total); err != nil {
		// Erreur non bloquante : retourner valeurs neutres (cap désactivé).
		return 0.5, 0, nil
	}
	if total == 0 {
		return 0.5, 0, nil
	}
	return float64(below) / float64(total), total, nil
}

// mapMetricToColumn convertit un FieldKey canonique en nom de colonne
// match_participants. Retourne "" si la métrique n'a pas d'équivalent
// direct (cas cumulative qui demandent une autre source).
func mapMetricToColumn(metric string) string {
	// Normalisation : accepte aussi les noms en lowercase
	m := strings.TrimSpace(metric)
	switch m {
	case "FieldKDA", "kda":
		return "kda"
	case "FieldKDR", "kd":
		return "kd"
	case "FieldAccuracy", metricColAccuracy:
		return metricColAccuracy
	case "FieldKills", metricColKills:
		return metricColKills
	case "FieldDeaths", metricColDeaths:
		return metricColDeaths
	case "FieldAssists", "assists":
		return "assists"
	case "FieldHeadshotKills", "headshot_kills":
		return "headshot_kills"
	case "FieldMeleeKills", "melee_kills":
		return "melee_kills"
	case "FieldGrenadeKills", "grenade_kills":
		return "grenade_kills"
	case "FieldPowerWeaponKills", "power_weapon_kills":
		return "power_weapon_kills"
	case "FieldDamageDealt", "damage_dealt":
		return "damage_dealt"
	case "FieldPersonalScore", "personal_score":
		return "personal_score"
	case "FieldMaxKillingSpree", "max_killing_spree":
		return "max_killing_spree"
	}
	return ""
}
