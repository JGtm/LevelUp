// Package duckdb — relations_repo.go : agrégats du hub Communauté > Relations.
// Lecture seule sur le catalogue shared (match_participants + match_registry +
// kill-feed canonique) via SharedReader. Aucune écriture.
//
// NOMS (lot A du plan perf « lectures par périmètre », 2026-09-26, ADR 0036 I1) : Q28 et Q28
// scopé ne joignent plus la vue canonique des noms (3,4-4,0 s par lecture, 2,4-2,7 s scopée à
// 30 matchs) ; GetRelations nomme ses lignes par l'annuaire en portée base
// (squad_repo_annuaire.go, nommerLignesPorteeBase) sur les matchs de la lecture : l'historique du
// joueur (QMatchsDuJoueurTpl), ou le périmètre scopé — la même liste que Q28 lie sous sa fenêtre.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/observability/timing"
)

// GetRelations retourne TOUS les joueurs récurrents (>= 2 matchs communs) avec
// leurs agrégats allié/ennemi, KDA moyens, duel (kills/deaths) et bornes
// temporelles. Triés count_together DESC, xuid ASC. Lecture seule.
//
// scope (Phase 2) restreint l'agrégation à un sous-ensemble de match_id :
//   - scope == nil  → l'historique du joueur (QMatchsDuJoueurTpl, exclusion Campagne comprise)
//   - scope vide    → aucun match en périmètre → retour ([], nil) sans requête
//   - scope non-vide → ces matchs
//
// Dans les deux cas la MÊME requête (Q28RelationsScopedTpl) lie la liste en constantes sous la
// fenêtre `_latest` du kill-feed (lot B du plan perf, 2026-09-27, ADR 0036 I2) ; l'annuaire nomme
// ensuite les lignes sur cette même liste, lue une fois.
func (r *CareerRepo) GetRelations(ctx context.Context, scope []string) ([]domain.RelationRawRow, error) {
	if scope != nil && len(scope) == 0 {
		// scope non-nil et vide : aucun match → aucune relation.
		return []domain.RelationRawRow{}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, careerEncountersTimeout)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetRelations: shared reader: %w", err)
	}
	defer release()

	stop := timing.FromContext(ctx).Section("relations")
	matchs, out, err := r.lireRelationsDuPerimetre(ctx, db, scope)
	stop()
	if err != nil {
		return nil, err
	}
	stop = timing.FromContext(ctx).Section("relations_annuaire")
	defer stop()
	if err := r.nommerRelations(ctx, db, matchs, out); err != nil {
		return nil, fmt.Errorf("CareerRepo.GetRelations: %w", err)
	}
	return out, nil
}

// lireRelationsDuPerimetre rend les matchs de la lecture (matchsDuPerimetre) et les lignes de
// Q28 sur ces matchs, sans nom. Un historique vide ne lit rien : Q28 n'aurait rien rendu.
func (r *CareerRepo) lireRelationsDuPerimetre(ctx context.Context, db *sql.DB, scope []string) ([]string, []domain.RelationRawRow, error) {
	matchs, err := r.matchsDuPerimetre(ctx, db, scope)
	if err != nil {
		return nil, nil, fmt.Errorf("CareerRepo.GetRelations: %w", err)
	}
	if len(matchs) == 0 {
		return nil, nil, nil
	}
	// PMT-5 : exprs win/loss title-aware (fallback "e.my_outcome = 2/3"
	// byte-identique Halo). Ordre des %s : win, loss, win, loss.
	winExpr := outcomeSQLEq(ctx, "e.my_outcome", canonical.OutcomeWin, "e.my_outcome = 2")
	lossExpr := outcomeSQLEq(ctx, "e.my_outcome", canonical.OutcomeLoss, "e.my_outcome = 3")
	sqlText, args := r.buildRelationsQuery(matchs, winExpr, lossExpr)
	out, err := lireRelations(ctx, db, sqlText, args)
	return matchs, out, err
}

// lireRelations exécute Q28 : lignes sans nom.
func lireRelations(ctx context.Context, db *sql.DB, sqlText string, args []any) ([]domain.RelationRawRow, error) {
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

// nommerRelations nomme les lignes par l'annuaire en portée base, sur les matchs de la lecture.
func (r *CareerRepo) nommerRelations(ctx context.Context, db *sql.DB, matchs []string, lignes []domain.RelationRawRow) error {
	if len(lignes) == 0 {
		return nil
	}
	return nommerLignesPorteeBase(ctx, db, matchs, lignes, accesLigne[domain.RelationRawRow]{
		xuid:   func(l domain.RelationRawRow) string { return l.XUID },
		nommer: func(l *domain.RelationRawRow, gt string) { l.Gamertag = gt },
	})
}

// matchsDuPerimetre : les matchs d'une lecture Relations (Q28, heatmap Q29) — le périmètre
// scopé, sinon l'historique du joueur (QMatchsDuJoueurTpl, exclusion Campagne comprise).
func (r *CareerRepo) matchsDuPerimetre(ctx context.Context, db *sql.DB, scope []string) ([]string, error) {
	if scope != nil {
		return scope, nil
	}
	return matchsDeLHistorique(ctx, db, r.historique())
}

// buildRelationsQuery assemble Q28 et ses args positionnels sur une liste de matchs NON VIDE
// (le périmètre scopé ou l'historique du joueur, cf. GetRelations).
func (r *CareerRepo) buildRelationsQuery(matchs []string, winExpr, lossExpr string) (string, []any) {
	x := r.pdb.XUID
	// Masquage Campagne (Halo 5) : my_history ne joint pas match_registry → forme
	// sous-requête by-match-id (sans placeholder, résolue AVANT Sprintf, ne décale
	// donc aucun args positionnel). No-op Infinite. Item backlog H1.
	// Deux clauses (my_history + kv_stats), la MÊME liste en une constante chacune
	// (clauseListeMatchs) : c'est la seconde qui borne la fenêtre du kill-feed.
	histo, histoArg := clauseListeMatchs("match_id", matchs)
	kv, kvArg := clauseListeMatchs("kv.match_id", matchs)
	tpl := resolveCampaignExclusionByMatchID(Q28RelationsScopedTpl, r.pdb.TitleSlug, "match_id")
	sqlText := fmt.Sprintf(tpl,
		" AND "+histo, winExpr, lossExpr, winExpr, lossExpr, " AND "+kv)
	// my_history.xuid, liste, encounters p.xuid<>, kv_stats 3 CASE + 2 WHERE, liste.
	return sqlText, []any{x, histoArg, x, x, x, x, x, x, kvArg}
}

// scanRelationRow scanne une ligne de Q28RelationsScopedTpl en domain.RelationRawRow.
func scanRelationRow(rows *sql.Rows) (domain.RelationRawRow, error) {
	var (
		row                       domain.RelationRawRow
		avgKDAWith, avgKDAAgainst sql.NullFloat64
		firstSeen, lastSeen       sql.NullTime
	)
	if err := rows.Scan(
		&row.XUID, &row.TotalMatches,
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
