// Package duckdb — squad_repo_shields.go : bris de bouclier et recharges complètes par joueur et
// par match (Q32f), source du rôle d'impact « Os de verre » (analysis/squadimpact.GlassJaw).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
)

// Q32fMatchShieldsTemplate : les mesures de bouclier de la passe COURANTE du résumé d'usage des
// matchs fournis. Lecture par la vue `match_usage_players_latest` UNIQUEMENT (ADR 0026 : la table
// brute sert les lignes d'une passe périmée).
//
// NULL dit « non mesuré » (match sans rejeu, passe d'avant la mesure, mode sans bouclier), et c'est
// distinct de zéro : une ligne dont les deux colonnes sont NULL est écartée, de sorte qu'un match
// est mesuré dès qu'une de ses lignes porte une valeur. Une colonne NULL à côté de l'autre mesurée
// vaut zéro.
//
// Le '%s' positionnel est la liste des match_ids. Ne PAS utiliser directement — passer par
// LoadMatchShields().
const Q32fMatchShieldsTemplate = `
SELECT match_id, xuid, COALESCE(shield_breaks, 0), COALESCE(shield_recharges, 0)
FROM match_usage_players_latest
WHERE match_id IN (%s)
  AND (shield_breaks IS NOT NULL OR shield_recharges IS NOT NULL)
ORDER BY match_id, xuid`

// LoadMatchShields : cf. port.SquadRepository. Dégradation gracieuse alignée sur Q32e : reader
// indisponible, ou base sans résumé d'usage (titre sans décodeur de film, base non migrée) →
// (nil, nil), loggé ; aucun rôle « Os de verre » n'est alors attribué.
func (r *SquadRepo) LoadMatchShields(ctx context.Context, matchIDs []string) ([]domain.MatchShieldRow, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "teammates: boucliers du film indisponibles (shared reader)",
			"matchs", len(matchIDs), "err", err)
		return nil, nil
	}
	defer release()

	rows, err := db.QueryContext(ctx,
		fmt.Sprintf(Q32fMatchShieldsTemplate, placeholderList(len(matchIDs))), appendStrArgs(nil, matchIDs)...)
	if err != nil {
		slog.WarnContext(ctx, "teammates: boucliers du film indisponibles (Q32f)",
			"matchs", len(matchIDs), "err", err)
		return nil, nil
	}
	defer rows.Close()
	return scanMatchShields(rows)
}

// scanMatchShields lit le résultat de Q32f.
func scanMatchShields(rows *sql.Rows) ([]domain.MatchShieldRow, error) {
	var out []domain.MatchShieldRow
	for rows.Next() {
		var row domain.MatchShieldRow
		if err := rows.Scan(&row.MatchID, &row.XUID, &row.Breaks, &row.Recharges); err != nil {
			return nil, fmt.Errorf("SquadRepo.LoadMatchShields scan: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
