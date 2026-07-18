// Package duckdb — career_repo_encounters.go : encounters globaux + rivals
// (top neměsis / top souffre-douleur) pour la page Carrière. Découpé de
// career_repo.go (god-file split, refactor 2026-05-27).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// GetTopEncountersGlobal retourne les 10 joueurs les plus croisés au niveau
// carrière, hors XUIDs présents dans excludeXUIDs (typiquement les amis
// configurés). Lit match_participants + killer_victim_pairs via SharedReader.
func (r *CareerRepo) GetTopEncountersGlobal(ctx context.Context, excludeXUIDs []string) ([]domain.MatchEncounterRow, []domain.EncounterStatsRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, careerEncountersTimeout)
	defer cancel()

	// Construit la clause d'exclusion friends. Si liste vide, %s = "".
	excludeClause := ""
	args := []any{r.pdb.XUID, r.pdb.XUID, r.pdb.XUID, r.pdb.XUID, r.pdb.XUID, r.pdb.XUID, r.pdb.XUID}
	if len(excludeXUIDs) > 0 {
		placeholders := strings.Repeat("?,", len(excludeXUIDs))
		placeholders = strings.TrimRight(placeholders, ",")
		excludeClause = " AND es.xuid NOT IN (" + placeholders + ")"
		for _, x := range excludeXUIDs {
			args = append(args, x)
		}
	}
	// PMT-5 : exprs win/loss title-aware (fallback "e.my_outcome = 2/3" byte-identique
	// Halo). Ordre des %s du template : win, loss, win, loss, puis excludeClause.
	winExpr := outcomeSQLEq(ctx, "e.my_outcome", canonical.OutcomeWin, "e.my_outcome = 2")
	lossExpr := outcomeSQLEq(ctx, "e.my_outcome", canonical.OutcomeLoss, "e.my_outcome = 3")
	// Masquage Campagne (Halo 5) : my_history ne joint pas match_registry → forme
	// sous-requête by-match-id (sans placeholder, résolue AVANT Sprintf). No-op Infinite.
	tpl := resolveCampaignExclusionByMatchID(Q26CareerTopEncountersTpl, r.pdb.TitleSlug, "match_id")
	sqlText := fmt.Sprintf(tpl, winExpr, lossExpr, winExpr, lossExpr, excludeClause)

	// migré vers SharedReader. La query est shared-only
	// (match_participants, match_registry, killer_victim_pairs, v_gamertag_lookup)
	// — tables/vues au niveau root du catalogue shared_matches_v2.duckdb.
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("CareerRepo.GetTopEncountersGlobal: shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("CareerRepo.GetTopEncountersGlobal: %w", err)
	}
	defer rows.Close()

	var encounters []domain.MatchEncounterRow
	var stats []domain.EncounterStatsRaw
	for rows.Next() {
		var (
			xuid                                                 string
			gamertag                                             string
			countTogether, allyCount, enemyCount                 int
			winsAsAlly, lossesAsAlly, winsVsEnemy, lossesVsEnemy int
			killsDealt, deathsSuffered                           int
			firstSeenAt, lastSeenAt                              sql.NullTime
		)
		if err := rows.Scan(
			&xuid, &gamertag, &countTogether,
			&allyCount, &enemyCount,
			&winsAsAlly, &lossesAsAlly, &winsVsEnemy, &lossesVsEnemy,
			&killsDealt, &deathsSuffered, &firstSeenAt, &lastSeenAt,
		); err != nil {
			return nil, nil, fmt.Errorf("CareerRepo.GetTopEncountersGlobal scan: %w", err)
		}
		enc := domain.MatchEncounterRow{
			XUID:          xuid,
			Gamertag:      gamertag,
			CountTogether: countTogether,
			IsAlly:        allyCount >= enemyCount,
		}
		if allyCount > 0 || enemyCount > 0 {
			a := allyCount
			e := enemyCount
			enc.AllyCount = &a
			enc.EnemyCount = &e
		}
		if winsAsAlly+lossesAsAlly > 0 {
			r := float64(winsAsAlly) / float64(winsAsAlly+lossesAsAlly)
			enc.WinrateAsAlly = &r
		}
		if winsVsEnemy+lossesVsEnemy > 0 {
			r := float64(winsVsEnemy) / float64(winsVsEnemy+lossesVsEnemy)
			enc.WinrateVsEnemy = &r
		}
		kd := killsDealt
		ds := deathsSuffered
		enc.KillsDealt = &kd
		enc.DeathsSuffered = &ds
		if lastSeenAt.Valid {
			t := lastSeenAt.Time
			enc.LastSeenAt = &t
		}
		encounters = append(encounters, enc)

		stats = append(stats, domain.EncounterStatsRaw{
			XUID:           xuid,
			AllyCount:      allyCount,
			EnemyCount:     enemyCount,
			WinsAsAlly:     winsAsAlly,
			LossesAsAlly:   lossesAsAlly,
			WinsVsEnemy:    winsVsEnemy,
			LossesVsEnemy:  lossesVsEnemy,
			KillsDealt:     killsDealt,
			DeathsSuffered: deathsSuffered,
		})
	}
	return encounters, stats, rows.Err()
}

// GetRivals retourne les top némésis (par deaths DESC) et top souffre-douleur
// (par frags DESC), 10 chacun, depuis killer_victim_pairs via SharedReader.
// Pas de seuil min — le ratio est calculé côté service.
//
// rivalsOrderColXxx : colonnes SQL acceptées par queryRivals.
const (
	rivalsOrderColFrags  = "frags"
	rivalsOrderColDeaths = "deaths"
)

func (r *CareerRepo) GetRivals(ctx context.Context) (nemeses, victims []domain.CareerRivalRawRow, err error) {
	ctx, cancel := context.WithTimeout(ctx, careerRivalsTimeout)
	defer cancel()

	nemeses, err = r.queryRivals(ctx, rivalsOrderColDeaths)
	if err != nil {
		return nil, nil, err
	}
	victims, err = r.queryRivals(ctx, rivalsOrderColFrags)
	if err != nil {
		return nil, nil, err
	}
	return nemeses, victims, nil
}

// queryRivals exécute Q27CareerRivalsTpl avec orderCol pour le tri (frags ou deaths).
func (r *CareerRepo) queryRivals(ctx context.Context, orderCol string) ([]domain.CareerRivalRawRow, error) {
	if orderCol != rivalsOrderColFrags && orderCol != rivalsOrderColDeaths {
		return nil, fmt.Errorf("CareerRepo.queryRivals: invalid order column %q", orderCol)
	}
	sqlText := fmt.Sprintf(Q27CareerRivalsTpl, orderCol)
	// migré vers SharedReader. Q27 est shared-only
	// (killer_victim_pairs + v_gamertag_lookup, tous root-level).
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.queryRivals(%s): shared reader: %w", orderCol, err)
	}
	defer release()

	rows, err := db.QueryContext(
		ctx, sqlText,
		r.pdb.XUID, r.pdb.XUID, r.pdb.XUID, r.pdb.XUID, r.pdb.XUID, r.pdb.XUID,
	)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.queryRivals(%s): %w", orderCol, err)
	}
	defer rows.Close()

	var results []domain.CareerRivalRawRow
	for rows.Next() {
		var row domain.CareerRivalRawRow
		if err := rows.Scan(&row.XUID, &row.Gamertag, &row.Frags, &row.Deaths, &row.MatchCount); err != nil {
			return nil, fmt.Errorf("CareerRepo.queryRivals(%s) scan: %w", orderCol, err)
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

// GetEncounters retourne les adversaires/coéquipiers fréquents.
func (r *CareerRepo) GetEncounters(ctx context.Context) ([]domain.EncounterRawRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetEncounters: %w", err)
	}
	defer release()

	// Masquage Campagne (Halo 5) : Q10 ne joint pas match_registry → forme
	// sous-requête by-match-id (p1.match_id). No-op Infinite. Item backlog H1.
	q := resolveCampaignExclusionByMatchID(Q10Encounters, r.pdb.TitleSlug, "p1.match_id")
	rows, err := db.QueryContext(ctx, q, r.pdb.XUID)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetEncounters: %w", err)
	}
	defer rows.Close()

	var results []domain.EncounterRawRow
	for rows.Next() {
		var e domain.EncounterRawRow
		if err := rows.Scan(
			&e.XUID, &e.Gamertag, &e.MatchCount, &e.AsTeammate, &e.AsEnemy, &e.AvgKDA,
		); err != nil {
			return nil, fmt.Errorf("CareerRepo.GetEncounters scan: %w", err)
		}
		results = append(results, e)
	}
	return results, rows.Err()
}
