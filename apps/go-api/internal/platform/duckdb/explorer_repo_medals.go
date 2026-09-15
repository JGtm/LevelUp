// Package duckdb — explorer_repo_medals.go : agrégat « top médailles » de
// l'Explorer sur une liste de match_id explicite.
//
// Fichier séparé d'explorer_repo.go (déjà au-delà du seuil de 500 lignes,
// dette gelée par la baseline lint : on ne l'accroît pas).
package duckdb

import (
	"context"
	"fmt"
	"strings"

	"levelup/go-api/internal/domain"
)

// GetTopMedalsForMatches retourne le top `limit` médailles du joueur sur les
// matchs donnés : SUM(count) par medal_name_id dans shared.medals_earned, tri
// décroissant (départage par identifiant croissant pour un ordre déterministe).
//
// Renvoie des identifiants BRUTS (domain.RemoteMedalCount) : l'enrichissement
// label/description/image est fait plus haut, par le service, avec le MÊME
// builder que les médailles lifetime — aucun libellé n'est résolu ici.
//
// Retourne nil si xuid vide, matchIDs vide ou limit <= 0.
func (r *ExplorerRepo) GetTopMedalsForMatches(
	ctx context.Context, xuid string, matchIDs []string, limit int,
) ([]domain.RemoteMedalCount, error) {
	if strings.TrimSpace(xuid) == "" || len(matchIDs) == 0 || limit <= 0 {
		return nil, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(matchIDs)), ",")
	q := fmt.Sprintf(`
		SELECT medal_name_id, COALESCE(SUM(count), 0) AS total
		FROM medals_earned
		WHERE xuid = ? AND match_id IN (%s)
		GROUP BY medal_name_id
		ORDER BY total DESC, medal_name_id ASC
		LIMIT %d`, placeholders, limit) //nolint:gosec // limit int maîtrisé (constante appelante), placeholders générés

	args := make([]any, 0, 1+len(matchIDs))
	args = append(args, xuid)
	for _, mid := range matchIDs {
		args = append(args, mid)
	}

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("ExplorerRepo.GetTopMedalsForMatches: shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("ExplorerRepo.GetTopMedalsForMatches: query: %w", err)
	}
	defer rows.Close()

	var out []domain.RemoteMedalCount
	for rows.Next() {
		// medal_name_id est UBIGINT en base de test / BIGINT en prod : UBigint
		// absorbe les deux (cf. ubigint_scanner.go).
		var id UBigint
		var count int
		if scanErr := rows.Scan(&id, &count); scanErr != nil {
			return nil, fmt.Errorf("ExplorerRepo.GetTopMedalsForMatches: scan: %w", scanErr)
		}
		out = append(out, domain.RemoteMedalCount{NameID: id.Int64(), Count: count})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ExplorerRepo.GetTopMedalsForMatches: rows: %w", err)
	}
	return out, nil
}
