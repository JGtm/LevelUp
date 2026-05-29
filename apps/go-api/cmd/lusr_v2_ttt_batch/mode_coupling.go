//go:build cgo

package main

// mode_coupling.go — Sprint 2.B : calcul + écriture de la matrice de couplage
// cross-mode (corrélation de Pearson des μ entre modes), en plus des stats
// empiriques de base. Stocke des rows mode_coupling_<source>_<target> dans
// lusr_hyperparams_v2 (playlist_group = source), relues par propagateCrossModeLeak.

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
)

// loadPlayerStatesByXUID charge les μ courants par joueur depuis
// player_skill_state_v2_latest, groupés par xuid pour l'estimation de matrice.
func loadPlayerStatesByXUID(ctx context.Context, db *sql.DB) (map[string][]skillv2.GroupState, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT xuid, playlist_group, mu FROM player_skill_state_v2_latest`)
	if err != nil {
		return nil, fmt.Errorf("query states: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	out := make(map[string][]skillv2.GroupState)
	for rows.Next() {
		var xuid, group string
		var mu float64
		if err := rows.Scan(&xuid, &group, &mu); err != nil {
			return nil, err
		}
		out[xuid] = append(out[xuid], skillv2.GroupState{Group: group, Mu: mu})
	}
	return out, rows.Err()
}

// printModeCouplingReport imprime la matrice de couplage (paires triées).
func printModeCouplingReport(matrix map[string]map[string]float64) {
	sources := make([]string, 0, len(matrix))
	for s := range matrix {
		sources = append(sources, s)
	}
	sort.Strings(sources)
	var sb strings.Builder
	sb.WriteString("=== Matrice de couplage cross-mode (Sprint 2.B) ===\n")
	seen := make(map[string]bool)
	for _, s := range sources {
		targets := make([]string, 0, len(matrix[s]))
		for t := range matrix[s] {
			targets = append(targets, t)
		}
		sort.Strings(targets)
		for _, t := range targets {
			key := s + "|" + t
			rev := t + "|" + s
			if seen[rev] {
				continue // déjà affiché (symétrique)
			}
			seen[key] = true
			sb.WriteString(fmt.Sprintf("  %s ↔ %s : %.3f\n", s, t, matrix[s][t]))
		}
	}
	if len(sources) == 0 {
		sb.WriteString("  (aucune paire éligible — pas assez de joueurs multi-modes)\n")
	}
	fmt.Println(sb.String())
}

// writeModeCoupling persiste chaque poids de couplage source→target dans
// lusr_hyperparams_v2 (playlist_group = source). La matrice étant symétrique,
// les deux sens sont écrits.
func writeModeCoupling(ctx context.Context, repo *duckdb.SkillV2Repo,
	matrix map[string]map[string]float64, source string) error {
	for src, targets := range matrix {
		for tgt, weight := range targets {
			if err := repo.UpsertHyperparam(ctx, domain.SkillV2Hyperparam{
				PlaylistGroup: src,
				Name:          skillv2.ModeCouplingHyperparamName(src, tgt),
				Value:         weight,
				Source:        source,
			}); err != nil {
				return fmt.Errorf("upsert coupling %s→%s: %w", src, tgt, err)
			}
		}
	}
	return nil
}
