// Package duckdb — career_repo_encounters.go : encounters globaux + rivals
// (top neměsis / top souffre-douleur) pour la page Carrière. Découpé de
// career_repo.go (god-file split, refactor 2026-05-27).
//
// NOMS (lot perf L7, 2026-09-23) : Q26 ne joint plus v_gamertag_lookup, qui se matérialisait
// en entier à chaque lecture (1,7 à 3 s). Ses lignes sont nommées par l'annuaire de la lecture
// (squad_repo_annuaire.go), sur les matchs de l'historique du joueur (QMatchsDuJoueurTpl) : même
// cascade que la vue.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/observability/timing"
)

// QMatchsDuJoueurTpl : les matchs de l'historique du joueur, exclusion Campagne comprise — le
// `my_history` de Q26. Ce sont les « matchs de la lecture » sur lesquels l'annuaire nomme les
// lignes des lectures Carrière agrégées (D7.1 du plan perf). Paramètre : ?1 = xuid du joueur.
const QMatchsDuJoueurTpl = `SELECT DISTINCT match_id FROM match_participants WHERE xuid = ?` + campaignExclusionToken

// GetTopEncountersGlobal retourne les 10 joueurs les plus croisés au niveau
// carrière, hors XUIDs présents dans excludeXUIDs (typiquement les amis
// configurés). Lit match_participants + le kill-feed canonique via SharedReader.
func (r *CareerRepo) GetTopEncountersGlobal(ctx context.Context, excludeXUIDs []string) ([]domain.MatchEncounterRow, []domain.EncounterStatsRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, careerEncountersTimeout)
	defer cancel()

	sqlText, args := r.topEncountersQuery(ctx, excludeXUIDs)
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("CareerRepo.GetTopEncountersGlobal: shared reader: %w", err)
	}
	defer release()

	stop := timing.FromContext(ctx).Section("top_encounters")
	encounters, stats, err := scanTopEncounters(ctx, db, sqlText, args)
	stop()
	if err != nil {
		return nil, nil, fmt.Errorf("CareerRepo.GetTopEncountersGlobal: %w", err)
	}
	// Noms : l'annuaire de la lecture — les xuids du top, sur l'historique que Q26 vient d'agréger.
	stop = timing.FromContext(ctx).Section("top_encounters_annuaire")
	err = nommerSurLHistorique(ctx, r, db, encounters, accesLigne[domain.MatchEncounterRow]{
		xuid:   func(e domain.MatchEncounterRow) string { return e.XUID },
		nommer: func(e *domain.MatchEncounterRow, gt string) { e.Gamertag = gt },
	})
	stop()
	if err != nil {
		return nil, nil, fmt.Errorf("CareerRepo.GetTopEncountersGlobal: %w", err)
	}
	return encounters, stats, nil
}

// topEncountersQuery assemble Q26 et ses arguments.
func (r *CareerRepo) topEncountersQuery(ctx context.Context, excludeXUIDs []string) (string, []any) {
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
	return fmt.Sprintf(tpl, winExpr, lossExpr, winExpr, lossExpr, excludeClause), args
}

// scanTopEncounters exécute Q26 et rend ses lignes, SANS nom (cf. nommerSurLHistorique). Le
// curseur est fermé au retour : l'annuaire relit la même connexion ensuite.
func scanTopEncounters(ctx context.Context, db *sql.DB, q string, args []any) ([]domain.MatchEncounterRow, []domain.EncounterStatsRaw, error) {
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var encounters []domain.MatchEncounterRow
	var stats []domain.EncounterStatsRaw
	for rows.Next() {
		var (
			st                      domain.EncounterStatsRaw
			countTogether           int
			firstSeenAt, lastSeenAt sql.NullTime
		)
		if err := rows.Scan(
			&st.XUID, &countTogether,
			&st.AllyCount, &st.EnemyCount,
			&st.WinsAsAlly, &st.LossesAsAlly, &st.WinsVsEnemy, &st.LossesVsEnemy,
			&st.KillsDealt, &st.DeathsSuffered, &firstSeenAt, &lastSeenAt,
		); err != nil {
			return nil, nil, fmt.Errorf("scan: %w", err)
		}
		encounters = append(encounters, encounterFromStats(st, countTogether, lastSeenAt))
		stats = append(stats, st)
	}
	return encounters, stats, rows.Err()
}

// encounterFromStats projette une ligne de Q26 en rencontre (sans nom : l'annuaire le pose).
func encounterFromStats(st domain.EncounterStatsRaw, countTogether int, lastSeenAt sql.NullTime) domain.MatchEncounterRow {
	enc := domain.MatchEncounterRow{
		XUID:          st.XUID,
		CountTogether: countTogether,
		IsAlly:        st.AllyCount >= st.EnemyCount,
	}
	if st.AllyCount > 0 || st.EnemyCount > 0 {
		a := st.AllyCount
		e := st.EnemyCount
		enc.AllyCount = &a
		enc.EnemyCount = &e
	}
	if st.WinsAsAlly+st.LossesAsAlly > 0 {
		r := float64(st.WinsAsAlly) / float64(st.WinsAsAlly+st.LossesAsAlly)
		enc.WinrateAsAlly = &r
	}
	if st.WinsVsEnemy+st.LossesVsEnemy > 0 {
		r := float64(st.WinsVsEnemy) / float64(st.WinsVsEnemy+st.LossesVsEnemy)
		enc.WinrateVsEnemy = &r
	}
	kd := st.KillsDealt
	ds := st.DeathsSuffered
	enc.KillsDealt = &kd
	enc.DeathsSuffered = &ds
	if lastSeenAt.Valid {
		t := lastSeenAt.Time
		enc.LastSeenAt = &t
	}
	return enc
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

// nommerSurLHistorique nomme les lignes d'une lecture Carrière agrégée par l'annuaire de la
// lecture (squad_repo_annuaire.go), sur les matchs de l'historique du joueur. Sans ligne : rien à lire.
func nommerSurLHistorique[T any](ctx context.Context, r *CareerRepo, db *sql.DB, lignes []T, acces accesLigne[T]) error {
	if len(lignes) == 0 {
		return nil
	}
	matchs, err := r.matchsDuJoueur(ctx, db)
	if err != nil {
		return err
	}
	return nommerLignes(ctx, db, matchs, lignes, acces)
}

// matchsDuJoueur rend les matchs de QMatchsDuJoueurTpl (exclusion Campagne résolue par match).
func (r *CareerRepo) matchsDuJoueur(ctx context.Context, db *sql.DB) ([]string, error) {
	q := resolveCampaignExclusionByMatchID(QMatchsDuJoueurTpl, r.pdb.TitleSlug, "match_id")
	rows, err := db.QueryContext(ctx, q, r.pdb.XUID)
	if err != nil {
		return nil, fmt.Errorf("matchs du joueur: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("matchs du joueur scan: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
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
