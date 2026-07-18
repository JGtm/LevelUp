// Package duckdb — relations_core_engagement_repo.go : agrégats joueur-centriques
// de la carte résumé du noyau dur (hub Communauté > Relations). Lecture seule sur
// le catalogue shared (match_participants + match_registry) via SharedReader.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// GetCoreEngagement retourne le WR global du joueur (référence du lift) + les
// issues des `limit` derniers matchs joués avec un membre du noyau (ancien→récent).
// scope : même contrat que GetRelations (nil = tous ; vide = court-circuit).
// coreXUIDs vide ⇒ RecentForm nil (le WR reste calculé). Lecture seule.
func (r *CareerRepo) GetCoreEngagement(ctx context.Context, coreXUIDs []string, scope []string, limit int) (domain.CoreEngagement, error) {
	ctx, cancel := context.WithTimeout(ctx, careerEncountersTimeout)
	defer cancel()

	// scope non-nil et vide : aucun match en périmètre → rien à agréger.
	if scope != nil && len(scope) == 0 {
		return domain.CoreEngagement{}, nil
	}

	wr, err := r.queryPlayerWinRate(ctx)
	if err != nil {
		return domain.CoreEngagement{}, err
	}
	form, err := r.queryRecentForm(ctx, QRelationsCoreFormTpl, coreXUIDs, scope, limit)
	if err != nil {
		return domain.CoreEngagement{}, err
	}
	return domain.CoreEngagement{PlayerWinRate: wr, RecentForm: form}, nil
}

// queryPlayerWinRate : WR HISTORIQUE décisif du joueur (tout-temps, nil si aucun
// match décisif). Référence stable du lift. Title-aware via outcomeSQLEq.
func (r *CareerRepo) queryPlayerWinRate(ctx context.Context) (*float64, error) {
	winExpr := outcomeSQLEq(ctx, "outcome", canonical.OutcomeWin, "outcome = 2")
	lossExpr := outcomeSQLEq(ctx, "outcome", canonical.OutcomeLoss, "outcome = 3")
	sqlText := fmt.Sprintf(QRelationsPlayerWinRateTpl, winExpr, winExpr, lossExpr) +
		excludeCampaignByMatchID(r.pdb.TitleSlug, "match_id")

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetCoreEngagement winrate: shared reader: %w", err)
	}
	defer release()

	var wins, decided sql.NullInt64
	if err := db.QueryRowContext(ctx, sqlText, r.pdb.XUID).Scan(&wins, &decided); err != nil {
		return nil, fmt.Errorf("CareerRepo.GetCoreEngagement winrate: %w", err)
	}
	if !decided.Valid || decided.Int64 == 0 {
		return nil, nil
	}
	rate := analysis.WinRate(int(wins.Int64), int(decided.Int64))
	return &rate, nil
}

// GetRelationRecentForm : voir port.RelationsRepository. Réutilise la requête
// de forme récente AVEC un noyau réduit à un seul xuid (le binôme).
func (r *CareerRepo) GetRelationRecentForm(ctx context.Context, xuid string, scope []string, limit int) ([]string, error) {
	if xuid == "" {
		return nil, nil
	}
	if scope != nil && len(scope) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, careerEncountersTimeout)
	defer cancel()
	return r.queryRecentForm(ctx, QRelationsCoreFormTpl, []string{xuid}, scope, limit)
}

// GetRelationEnemyRecentForm : voir port.RelationsRepository. Forme récente des
// matchs joués CONTRE `xuid` (équipe adverse) — sparkline « Derniers matchs » de
// la carte bête noire (miroir ennemi de GetRelationRecentForm).
func (r *CareerRepo) GetRelationEnemyRecentForm(ctx context.Context, xuid string, scope []string, limit int) ([]string, error) {
	if xuid == "" {
		return nil, nil
	}
	if scope != nil && len(scope) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, careerEncountersTimeout)
	defer cancel()
	return r.queryRecentForm(ctx, QRelationsEnemyFormTpl, []string{xuid}, scope, limit)
}

// queryRecentForm : issues ("win"|"loss"|"other") des `limit` derniers matchs du
// joueur satisfaisant `baseTpl` (AVEC un membre — QRelationsCoreFormTpl — ou
// CONTRE un adversaire — QRelationsEnemyFormTpl), retournées ancien→récent (sens
// de la frise). Les deux templates partagent le même ordre de placeholders.
func (r *CareerRepo) queryRecentForm(ctx context.Context, baseTpl string, xuids, scope []string, limit int) ([]string, error) {
	if len(xuids) == 0 || limit <= 0 {
		return nil, nil
	}
	winExpr := outcomeSQLEq(ctx, "outcome", canonical.OutcomeWin, "outcome = 2")
	lossExpr := outcomeSQLEq(ctx, "outcome", canonical.OutcomeLoss, "outcome = 3")

	scopeIn := ""
	args := []any{r.pdb.XUID}
	if scope != nil {
		scopeIn = " AND mp.match_id IN (" + Placeholders(len(scope)) + ")"
		args = append(args, ToAnySlice(scope)...)
	}
	args = append(args, r.pdb.XUID)           // exclusion self : c.xuid <> ?
	args = append(args, ToAnySlice(xuids)...) // c.xuid IN (…)
	args = append(args, limit)
	// Masquage Campagne (Halo 5) : my_matches est filtré via la forme sous-requête
	// by-match-id (sans placeholder, résolue AVANT Sprintf). No-op Infinite.
	tpl := resolveCampaignExclusionByMatchID(baseTpl, r.pdb.TitleSlug, "mp.match_id")
	sqlText := fmt.Sprintf(tpl, scopeIn, Placeholders(len(xuids)), winExpr, lossExpr)

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.queryRecentForm: shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.queryRecentForm: %w", err)
	}
	defer rows.Close()

	var recent []string // récent→ancien (ORDER BY DESC)
	for rows.Next() {
		var label string
		if err := rows.Scan(&label); err != nil {
			return nil, fmt.Errorf("CareerRepo.queryRecentForm scan: %w", err)
		}
		recent = append(recent, label)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Inverser → ancien→récent (gauche→droite de la sparkline).
	for i, j := 0, len(recent)-1; i < j; i, j = i+1, j-1 {
		recent[i], recent[j] = recent[j], recent[i]
	}
	return recent, nil
}
