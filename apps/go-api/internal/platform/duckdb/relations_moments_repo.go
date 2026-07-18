// Package duckdb — relations_moments_repo.go : lectures du sous-endpoint
// « Moments & Rivalités » (Phase 3a). Lecture seule sur le catalogue shared via
// SharedReader. Heatmap relation × heure (top-N) + timeline d'un rival.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// GetRelationsHeatmap retourne, pour les TOP-N relations (les plus de matchs
// communs, bots exclus), le nombre de matchs communs par heure (fuseau
// utilisateur). Le bucketing par heure est fait côté service (aggregateHeatmapHours).
//
// scope (Phase 2) : nil → tous les matchs ; vide → court-circuit ([], nil) ;
// non-vide → clause IN injectée dans my_history.
func (r *CareerRepo) GetRelationsHeatmap(ctx context.Context, scope []string, topN int) ([]domain.RelationHeatmapRawRow, error) {
	ctx, cancel := context.WithTimeout(ctx, careerEncountersTimeout)
	defer cancel()

	if scope != nil && len(scope) == 0 {
		return []domain.RelationHeatmapRawRow{}, nil
	}

	sqlText, args := r.buildHeatmapQuery(scope, topN)
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetRelationsHeatmap: shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetRelationsHeatmap: %w", err)
	}
	defer rows.Close()

	var out []domain.RelationHeatmapRawRow
	for rows.Next() {
		var row domain.RelationHeatmapRawRow
		if err := rows.Scan(&row.XUID, &row.Gamertag, &row.Hour, &row.Dow, &row.Count); err != nil {
			return nil, fmt.Errorf("CareerRepo.GetRelationsHeatmap scan: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// buildHeatmapQuery assemble Q29 + args selon le scope.
func (r *CareerRepo) buildHeatmapQuery(scope []string, topN int) (string, []any) {
	x := r.pdb.XUID
	// Masquage Campagne (Halo 5) : my_history sans registre → by-match-id, résolu
	// AVANT Sprintf (sans placeholder). No-op Infinite. Item backlog H1.
	tpl := resolveCampaignExclusionByMatchID(Q29RelationsHeatmapTpl, r.pdb.TitleSlug, "match_id")
	if len(scope) == 0 { // scope == nil ici (le cas vide est court-circuité en amont)
		sqlText := fmt.Sprintf(tpl, "")
		return sqlText, []any{x, x, topN}
	}
	inClause := " AND match_id IN (" + Placeholders(len(scope)) + ")"
	sqlText := fmt.Sprintf(tpl, inClause)
	args := make([]any, 0, 3+len(scope))
	args = append(args, x)                    // my_history.xuid
	args = append(args, ToAnySlice(scope)...) // my_history scope IN
	args = append(args, x, topN)              // encounters p.xuid<> + LIMIT
	return sqlText, args
}

// GetRivalTimeline retourne la séquence des N derniers duels (matchs communs
// joués EN ENNEMI) contre rivalXUID, ordonnée ancien→récent, avec les frags
// directionnels moi↔lui par match. Result est décidé title-aware (1=win,
// 2=loss, 0=autre) via outcomeSQLEq — jamais 2/3 en dur.
//
// scope (Phase 2) : nil → tous ; vide → court-circuit ; non-vide → IN injecté
// dans la CTE duel (kv.match_id) ET dans recent (r.match_id).
func (r *CareerRepo) GetRivalTimeline(ctx context.Context, rivalXUID string, scope []string, limit int) ([]domain.RelationDuelRawRow, error) {
	ctx, cancel := context.WithTimeout(ctx, careerEncountersTimeout)
	defer cancel()

	if scope != nil && len(scope) == 0 {
		return []domain.RelationDuelRawRow{}, nil
	}

	winExpr := outcomeSQLEq(ctx, "p1.outcome", canonical.OutcomeWin, "p1.outcome = 2")
	lossExpr := outcomeSQLEq(ctx, "p1.outcome", canonical.OutcomeLoss, "p1.outcome = 3")
	sqlText, args := r.buildRivalTimelineQuery(rivalXUID, scope, winExpr, lossExpr, limit)

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetRivalTimeline: shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetRivalTimeline: %w", err)
	}
	defer rows.Close()

	var out []domain.RelationDuelRawRow
	for rows.Next() {
		var (
			row       domain.RelationDuelRawRow
			startTime sql.NullTime
		)
		if err := rows.Scan(&row.MatchID, &startTime, &row.Result, &row.KillsOnRival, &row.DeathsByRival, &row.Mode, &row.MapName); err != nil {
			return nil, fmt.Errorf("CareerRepo.GetRivalTimeline scan: %w", err)
		}
		if startTime.Valid {
			row.StartTime = startTime.Time
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// buildRivalTimelineQuery assemble Q30 + args selon le scope.
func (r *CareerRepo) buildRivalTimelineQuery(rivalXUID string, scope []string, winExpr, lossExpr string, limit int) (string, []any) {
	me := r.pdb.XUID
	// Masquage Campagne (Halo 5) : recent joint match_registry aliasé "r" → forme
	// alias, résolue AVANT Sprintf (sans placeholder). No-op Infinite. Item backlog H1.
	tpl := resolveCampaignExclusion(Q30RivalTimelineTpl, r.pdb.TitleSlug, "r")
	if len(scope) == 0 { // scope == nil ici
		sqlText := fmt.Sprintf(tpl, "", winExpr, lossExpr, "")
		// duel CTE : me(kills), me(deaths), (me,rival), (rival,me) ; recent : me, rival ; LIMIT.
		args := []any{me, me, me, rivalXUID, rivalXUID, me, me, rivalXUID, limit}
		return sqlText, args
	}
	kvIn := " AND kv.match_id IN (" + Placeholders(len(scope)) + ")"
	rIn := " AND r.match_id IN (" + Placeholders(len(scope)) + ")"
	sqlText := fmt.Sprintf(tpl, kvIn, winExpr, lossExpr, rIn)

	args := make([]any, 0, 9+2*len(scope))
	args = append(args, me, me, me, rivalXUID, rivalXUID, me) // duel CTE 6 xuid
	args = append(args, ToAnySlice(scope)...)                 // kv.match_id IN
	args = append(args, me, rivalXUID)                        // recent p1/p2
	args = append(args, ToAnySlice(scope)...)                 // r.match_id IN
	args = append(args, limit)
	return sqlText, args
}
