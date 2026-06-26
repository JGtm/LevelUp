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
func (r *CareerRepo) GetRelations(ctx context.Context) ([]domain.RelationRawRow, error) {
	ctx, cancel := context.WithTimeout(ctx, careerEncountersTimeout)
	defer cancel()

	// PMT-5 : exprs win/loss title-aware (fallback "e.my_outcome = 2/3"
	// byte-identique Halo). Ordre des %s : win, loss, win, loss.
	winExpr := outcomeSQLEq(ctx, "e.my_outcome", canonical.OutcomeWin, "e.my_outcome = 2")
	lossExpr := outcomeSQLEq(ctx, "e.my_outcome", canonical.OutcomeLoss, "e.my_outcome = 3")
	sqlText := fmt.Sprintf(Q28RelationsTpl, winExpr, lossExpr, winExpr, lossExpr)

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetRelations: shared reader: %w", err)
	}
	defer release()

	x := r.pdb.XUID
	rows, err := db.QueryContext(ctx, sqlText, x, x, x, x, x, x, x)
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
