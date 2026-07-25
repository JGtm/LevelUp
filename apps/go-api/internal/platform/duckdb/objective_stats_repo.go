// Package duckdb — objective_stats_repo.go : agrégat (SUM) des stats objectifs
// par joueur sur un ensemble de matchs, servi aux KPI « Objectifs » des pages
// Synthèse et Escouade (PLAN_V72_OBJECTIVE_STATS.md, P4).
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
)

// ObjectiveStatsRepo lit shared.match_objective_stats_latest (vue append-only) sur
// le SharedReader du joueur. Injecté best-effort et gated par la capability
// match.objective.stats (registry_pages_home) — nil ⇒ pas de KPI objectifs.
type ObjectiveStatsRepo struct {
	pdb *PlayerDB
}

// NewObjectiveStatsRepo construit le repo à partir de la player DB (SharedReader).
func NewObjectiveStatsRepo(pdb *PlayerDB) *ObjectiveStatsRepo {
	return &ObjectiveStatsRepo{pdb: pdb}
}

// LoadAggregatedByXUID retourne, par xuid, le cumul des stats objectifs sur les
// matchs donnés (SUM sur match_objective_stats_latest, filtré xuid IN + match_id IN).
// Best-effort : une erreur de requête (vue absente sur une DB non migrée, etc.)
// dégrade en map vide + warn — jamais d'échec dur d'une page. Seuls les xuids avec
// au moins une ligne objectif figurent dans le résultat.
func (r *ObjectiveStatsRepo) LoadAggregatedByXUID(
	ctx context.Context, matchIDs, xuids []string,
) (map[string]*domain.ObjectiveAggregate, error) {
	out := map[string]*domain.ObjectiveAggregate{}
	if len(matchIDs) == 0 || len(xuids) == 0 {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return out, fmt.Errorf("ObjectiveStatsRepo: shared reader: %w", err)
	}
	defer release()

	args := make([]any, 0, len(xuids)+len(matchIDs))
	for _, x := range xuids {
		args = append(args, x)
	}
	for _, id := range matchIDs {
		args = append(args, id)
	}
	q := `SELECT xuid,
	             COALESCE(SUM(flag_captures), 0)::INTEGER,
	             COALESCE(SUM(flag_returns), 0)::INTEGER,
	             COALESCE(SUM(flag_steals), 0)::INTEGER,
	             COALESCE(SUM(time_as_flag_carrier_seconds), 0)::DOUBLE,
	             COALESCE(SUM(zone_captures), 0)::INTEGER,
	             COALESCE(SUM(zone_secures), 0)::INTEGER,
	             COALESCE(SUM(time_in_zones_seconds), 0)::DOUBLE,
	             COALESCE(SUM(skull_grabs), 0)::INTEGER,
	             COALESCE(SUM(time_as_skull_carrier_seconds), 0)::DOUBLE
	      FROM match_objective_stats_latest
	      WHERE xuid IN (` + Placeholders(len(xuids)) + `)
	        AND match_id IN (` + Placeholders(len(matchIDs)) + `)
	      GROUP BY xuid`

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		slog.WarnContext(ctx, "ObjectiveStatsRepo: query failed (best-effort)",
			"match_count", len(matchIDs), "xuid_count", len(xuids), "err", err)
		return out, nil
	}
	defer rows.Close()

	for rows.Next() {
		var xuid string
		var agg domain.ObjectiveAggregate
		if err := rows.Scan(
			&xuid,
			&agg.FlagCaptures, &agg.FlagReturns, &agg.FlagSteals, &agg.FlagCarrierSeconds,
			&agg.ZoneCaptures, &agg.ZoneSecures, &agg.ZoneSeconds,
			&agg.SkullGrabs, &agg.SkullCarrierSeconds,
		); err != nil {
			slog.WarnContext(ctx, "ObjectiveStatsRepo: scan failed", "err", err)
			continue
		}
		if agg.IsEmpty() {
			continue
		}
		a := agg
		out[xuid] = &a
	}
	return out, rows.Err()
}
