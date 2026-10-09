// Package duckdb — squad_repo_assist_pairs.go : lecture des paires d'assistance INTERNES
// à l'escouade sur une sélection de matchs (Q32d).
//
// Fichier dédié plutôt qu'un ajout à squad_repo.go (444 lignes, proche du seuil de 500).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

// LoadSquadAssistPairs retourne les paires (assistant → tueur assisté) internes à
// l'escouade sur les matchs fournis, et le nombre de ces matchs où l'assistance EST
// mesurée.
//
// Dégradation gracieuse alignée sur Q21b/Q21c/Q21d — et pour la même raison de fond :
// un titre sans décodeur de film n'est pas une panne. Reader indisponible ou table
// absente d'une base non migrée rendent une couverture à ZÉRO, loggée ; le builder
// n'émet alors aucun bloc et la page ne rend rien de plus qu'avant.
//
// Entrées vides (aucun match, aucun xuid) : retour immédiat, aucune requête.
func (r *SquadRepo) LoadSquadAssistPairs(
	ctx context.Context,
	matchIDs, squadXUIDs []string,
) ([]domain.SquadAssistPairRaw, int, error) {
	if len(matchIDs) == 0 || len(squadXUIDs) == 0 {
		return nil, 0, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "teammates: paires d'assistance indisponibles (shared reader)",
			"matchs", len(matchIDs), "err", err)
		return nil, 0, nil
	}
	defer release()

	query := fmt.Sprintf(Q32dSquadAssistPairsTemplate,
		placeholderList(len(matchIDs)),
		placeholderList(len(matchIDs)),
		placeholderList(len(squadXUIDs)),
		placeholderList(len(squadXUIDs)),
	)
	// L'ordre des arguments suit STRICTEMENT celui des '%s' du gabarit : portée,
	// paires, assistants, tueurs.
	args := make([]interface{}, 0, 2*len(matchIDs)+2*len(squadXUIDs))
	args = appendStrArgs(args, matchIDs)
	args = appendStrArgs(args, matchIDs)
	args = appendStrArgs(args, squadXUIDs)
	args = appendStrArgs(args, squadXUIDs)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		slog.WarnContext(ctx, "teammates: paires d'assistance indisponibles (Q32d)",
			"matchs", len(matchIDs), "err", err)
		return nil, 0, nil
	}
	defer rows.Close()
	return scanSquadAssistPairs(rows)
}

// scanSquadAssistPairs lit le résultat de Q32d. Séparé du lecteur pour être testable sur
// une base DuckDB en mémoire, sans provider ni bail.
func scanSquadAssistPairs(rows *sql.Rows) ([]domain.SquadAssistPairRaw, int, error) {
	var (
		measured int
		out      []domain.SquadAssistPairRaw
	)
	for rows.Next() {
		var (
			m                int
			ax, kx           sql.NullString
			assistN, stolenN sql.NullInt64
		)
		if err := rows.Scan(&m, &ax, &kx, &assistN, &stolenN); err != nil {
			return nil, 0, fmt.Errorf("SquadRepo.LoadSquadAssistPairs scan: %w", err)
		}
		measured = m
		// Ligne de couverture SEULE : aucune paire interne à l'escouade sur la
		// sélection. On garde la couverture, on n'invente pas de paire.
		if !ax.Valid || !kx.Valid {
			continue
		}
		out = append(out, domain.SquadAssistPairRaw{
			AssistXUID:  ax.String,
			KillerXUID:  kx.String,
			AssistCount: int(assistN.Int64),
			StolenCount: int(stolenN.Int64),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, measured, nil
}

// placeholderList rend `?,?,?` pour n valeurs.
func placeholderList(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

// appendStrArgs pousse une tranche de chaînes dans la liste d'arguments positionnels.
func appendStrArgs(args []interface{}, values []string) []interface{} {
	for _, v := range values {
		args = append(args, v)
	}
	return args
}
