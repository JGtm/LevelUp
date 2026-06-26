// Package duckdb — relations_repo.go : agrégats du hub Communauté > Relations.
// Lecture seule sur le catalogue shared (match_participants + match_registry +
// killer_victim_pairs + v_gamertag_lookup) via SharedReader. Aucune écriture.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// GetRelations retourne TOUS les joueurs récurrents (>= 2 matchs communs) avec
// leurs agrégats allié/ennemi, KDA moyens, duel (kills/deaths) et bornes
// temporelles. Triés count_together DESC, xuid ASC. Lecture seule.
//
// scope (Phase 2) restreint l'agrégation à un sous-ensemble de match_id :
//   - scope == nil  → aucun filtre (template Q28 non scopé, byte-identique Phase 1)
//   - scope vide    → aucun match en périmètre → retour ([], nil) sans requête
//   - scope non-vide → clause IN injectée dans my_history + kv_stats
func (r *CareerRepo) GetRelations(ctx context.Context, scope []string) ([]domain.RelationRawRow, error) {
	ctx, cancel := context.WithTimeout(ctx, careerEncountersTimeout)
	defer cancel()

	// PMT-5 : exprs win/loss title-aware (fallback "e.my_outcome = 2/3"
	// byte-identique Halo). Ordre des %s : win, loss, win, loss.
	winExpr := outcomeSQLEq(ctx, "e.my_outcome", canonical.OutcomeWin, "e.my_outcome = 2")
	lossExpr := outcomeSQLEq(ctx, "e.my_outcome", canonical.OutcomeLoss, "e.my_outcome = 3")

	sqlText, args := r.buildRelationsQuery(scope, winExpr, lossExpr)
	if sqlText == "" {
		// scope non-nil et vide : aucun match → aucune relation.
		return []domain.RelationRawRow{}, nil
	}

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetRelations: shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetRelations: %w", err)
	}
	defer rows.Close()

	var out []domain.RelationRawRow
	for rows.Next() {
		row, scanErr := scanRelationRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// buildRelationsQuery assemble le SQL + les args positionnels selon le scope.
// Retourne ("", nil) si le scope est non-nil et vide (aucun match en périmètre).
func (r *CareerRepo) buildRelationsQuery(scope []string, winExpr, lossExpr string) (string, []any) {
	x := r.pdb.XUID
	if scope == nil {
		// Phase 1 : aucun filtre. 7 placeholders xuid.
		sqlText := fmt.Sprintf(Q28RelationsTpl, winExpr, lossExpr, winExpr, lossExpr)
		return sqlText, []any{x, x, x, x, x, x, x}
	}
	if len(scope) == 0 {
		return "", nil
	}
	// Scope non-vide : deux clauses IN (my_history + kv_stats).
	inClause := " AND match_id IN (" + Placeholders(len(scope)) + ")"
	kvInClause := " AND kv.match_id IN (" + Placeholders(len(scope)) + ")"
	sqlText := fmt.Sprintf(Q28RelationsScopedTpl,
		inClause, winExpr, lossExpr, winExpr, lossExpr, kvInClause)

	args := make([]any, 0, 7+2*len(scope))
	args = append(args, x)                    // my_history.xuid
	args = append(args, ToAnySlice(scope)...) // my_history scope IN
	// encounters p.xuid<> (1) + kv_stats : 3 CASE + 2 WHERE (5) = 6 xuid.
	args = append(args, x, x, x, x, x, x)
	args = append(args, ToAnySlice(scope)...) // kv_stats scope IN
	return sqlText, args
}

// scanRelationRow scanne une ligne de Q28RelationsTpl en domain.RelationRawRow.
func scanRelationRow(rows *sql.Rows) (domain.RelationRawRow, error) {
	var (
		row                       domain.RelationRawRow
		avgKDAWith, avgKDAAgainst sql.NullFloat64
		firstSeen, lastSeen       sql.NullTime
	)
	if err := rows.Scan(
		&row.XUID, &row.Gamertag, &row.TotalMatches,
		&row.TeammateCount, &row.EnemyCount,
		&row.TeammateWins, &row.TeammateLosses,
		&row.EnemyWins, &row.EnemyLosses,
		&row.KillsDealt, &row.DeathsSuffered,
		&avgKDAWith, &avgKDAAgainst,
		&firstSeen, &lastSeen,
	); err != nil {
		return domain.RelationRawRow{}, fmt.Errorf("CareerRepo.GetRelations scan: %w", err)
	}
	if avgKDAWith.Valid {
		v := avgKDAWith.Float64
		row.AvgKDAWith = &v
	}
	if avgKDAAgainst.Valid {
		v := avgKDAAgainst.Float64
		row.AvgKDAAgainst = &v
	}
	if firstSeen.Valid {
		row.FirstSeen = firstSeen.Time
	}
	if lastSeen.Valid {
		row.LastSeen = lastSeen.Time
	}
	return row, nil
}
