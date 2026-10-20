// Package duckdb — career_repo_encounters.go : encounters globaux pour la page Carrière (les
// rivaux vivent dans career_repo_rivals.go). Découpé de career_repo.go (god-file split,
// refactor 2026-05-27).
//
// NOMS (lot perf L7, 2026-09-23) : Q26, Q27 et Q10 ne joignent plus v_gamertag_lookup, qui se
// matérialisait en entier à chaque lecture (1,7 à 3 s ; trois fois par ouverture de la page
// Carrière). Leurs lignes sont nommées par l'annuaire de la lecture (squad_repo_annuaire.go), sur
// les matchs de l'historique du joueur (QMatchsDuJoueurTpl) : même cascade que la vue.
//
// PÉRIMÈTRE (lot B du plan perf, 2026-09-27, ADR 0036 I2) : Q26 et Q27 lisent d'abord cette
// liste, la lient en UNE constante (clauseListeMatchs) sous la fenêtre `_latest` du kill-feed,
// puis la passent telle quelle à l'annuaire — une seule lecture de la liste par appel.
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
// lignes des lectures Carrière agrégées (D7.1 du plan perf), et la liste liée sous les fenêtres
// `_latest` des lectures d'historique complet (lot B). Paramètre : ?1 = xuid du joueur.
const QMatchsDuJoueurTpl = `SELECT DISTINCT match_id FROM match_participants WHERE xuid = ?` + campaignExclusionToken

// GetTopEncountersGlobal retourne les 10 joueurs les plus croisés au niveau
// carrière, hors XUIDs présents dans excludeXUIDs (typiquement les amis
// configurés). Lit match_participants + le kill-feed canonique via SharedReader.
func (r *CareerRepo) GetTopEncountersGlobal(ctx context.Context, excludeXUIDs []string) ([]domain.MatchEncounterRow, []domain.EncounterStatsRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, careerEncountersTimeout)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("CareerRepo.GetTopEncountersGlobal: shared reader: %w", err)
	}
	defer release()

	stop := timing.FromContext(ctx).Section("top_encounters")
	matchs, encounters, stats, err := r.lireTopEncounters(ctx, db, excludeXUIDs)
	stop()
	if err != nil {
		return nil, nil, fmt.Errorf("CareerRepo.GetTopEncountersGlobal: %w", err)
	}
	// Noms : l'annuaire de la lecture — les xuids du top, sur les matchs que Q26 vient d'agréger.
	stop = timing.FromContext(ctx).Section("top_encounters_annuaire")
	err = nommerLignes(ctx, db, matchs, encounters, accesLigne[domain.MatchEncounterRow]{
		xuid:   func(e domain.MatchEncounterRow) string { return e.XUID },
		nommer: func(e *domain.MatchEncounterRow, gt string) { e.Gamertag = gt },
	})
	stop()
	if err != nil {
		return nil, nil, fmt.Errorf("CareerRepo.GetTopEncountersGlobal: %w", err)
	}
	return encounters, stats, nil
}

// lireTopEncounters lit les matchs du joueur puis Q26 bornée à ces matchs. Sans match : aucune
// ligne, sans requête (`IN ()` n'est pas du SQL valide ; Q26 n'aurait rien rendu).
func (r *CareerRepo) lireTopEncounters(ctx context.Context, db *sql.DB, excludeXUIDs []string) (
	[]string, []domain.MatchEncounterRow, []domain.EncounterStatsRaw, error,
) {
	matchs, err := matchsDeLHistorique(ctx, db, r.historique())
	if err != nil || len(matchs) == 0 {
		return matchs, nil, nil, err
	}
	sqlText, args := r.topEncountersQuery(ctx, matchs, excludeXUIDs)
	encounters, stats, err := scanTopEncounters(ctx, db, sqlText, args)
	return matchs, encounters, stats, err
}

// topEncountersQuery assemble Q26 et ses arguments : la liste `matchs` (non vide) sous la
// fenêtre du kill-feed, puis l'exclusion des amis.
func (r *CareerRepo) topEncountersQuery(ctx context.Context, matchs, excludeXUIDs []string) (string, []any) {
	x := r.pdb.XUID
	liste, listeArg := clauseListeMatchs("kv.match_id", matchs)
	args := make([]any, 0, 8+len(excludeXUIDs))
	args = append(args, x, x, x, x, x, x, x, listeArg)
	// Construit la clause d'exclusion friends. Si liste vide, %s = "".
	excludeClause := ""
	if len(excludeXUIDs) > 0 {
		placeholders := strings.Repeat("?,", len(excludeXUIDs))
		placeholders = strings.TrimRight(placeholders, ",")
		excludeClause = " AND es.xuid NOT IN (" + placeholders + ")"
		for _, x := range excludeXUIDs {
			args = append(args, x)
		}
	}
	// PMT-5 : exprs win/loss title-aware (fallback "e.my_outcome = 2/3" byte-identique
	// Halo). Ordre des %s du template : win, loss, win, loss, liste, puis excludeClause.
	winExpr := outcomeSQLEq(ctx, "e.my_outcome", canonical.OutcomeWin, "e.my_outcome = 2")
	lossExpr := outcomeSQLEq(ctx, "e.my_outcome", canonical.OutcomeLoss, "e.my_outcome = 3")
	// Masquage Campagne (Halo 5) : my_history ne joint pas match_registry → forme
	// sous-requête by-match-id (sans placeholder, résolue AVANT Sprintf). No-op Infinite.
	tpl := resolveCampaignExclusionByMatchID(Q26CareerTopEncountersTpl, r.pdb.TitleSlug, "match_id")
	return fmt.Sprintf(tpl, winExpr, lossExpr, winExpr, lossExpr, " AND "+liste, excludeClause), args
}

// scanTopEncounters exécute Q26 et rend ses lignes, SANS nom (cf. GetTopEncountersGlobal). Le
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

// historique : le joueur sur les matchs duquel l'annuaire nomme une lecture agrégée
// (QMatchsDuJoueurTpl), et le titre qui décide l'exclusion Campagne.
type historique struct {
	xuid, titre string
}

// historique du joueur du repo (même titre que Q26 pour l'exclusion Campagne).
func (r *CareerRepo) historique() historique {
	return historique{xuid: r.pdb.XUID, titre: r.pdb.TitleSlug}
}

// nommerSurLHistorique nomme les lignes d'une lecture agrégée par l'annuaire de la lecture
// (squad_repo_annuaire.go), sur les matchs de l'historique `h`. Sans ligne : rien à lire.
func nommerSurLHistorique[T any](ctx context.Context, db *sql.DB, h historique, lignes []T, acces accesLigne[T]) error {
	if len(lignes) == 0 {
		return nil
	}
	matchs, err := matchsDeLHistorique(ctx, db, h)
	if err != nil {
		return err
	}
	return nommerLignes(ctx, db, matchs, lignes, acces)
}

// matchsDeLHistorique rend les matchs de QMatchsDuJoueurTpl (exclusion Campagne résolue par match).
func matchsDeLHistorique(ctx context.Context, db *sql.DB, h historique) ([]string, error) {
	q := resolveCampaignExclusionByMatchID(QMatchsDuJoueurTpl, h.titre, "match_id")
	rows, err := db.QueryContext(ctx, q, h.xuid)
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

// GetEncounters retourne les adversaires/coéquipiers fréquents (Q10), nommés par l'annuaire de
// la lecture sur l'historique du joueur (lot perf L7 : plus de jointure v_gamertag_lookup).
func (r *CareerRepo) GetEncounters(ctx context.Context) ([]domain.EncounterRawRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetEncounters: %w", err)
	}
	defer release()

	stop := timing.FromContext(ctx).Section("encounters")
	results, err := r.scanEncounters(ctx, db)
	stop()
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetEncounters: %w", err)
	}
	stop = timing.FromContext(ctx).Section("encounters_annuaire")
	err = nommerSurLHistorique(ctx, db, r.historique(), results, accesLigne[domain.EncounterRawRow]{
		xuid:   func(e domain.EncounterRawRow) string { return e.XUID },
		nommer: func(e *domain.EncounterRawRow, gt string) { e.Gamertag = gt },
	})
	stop()
	if err != nil {
		return nil, fmt.Errorf("CareerRepo.GetEncounters: %w", err)
	}
	return results, nil
}

// scanEncounters exécute Q10 et rend ses lignes, SANS nom (cf. GetEncounters).
func (r *CareerRepo) scanEncounters(ctx context.Context, db *sql.DB) ([]domain.EncounterRawRow, error) {
	// Masquage Campagne (Halo 5) : Q10 ne joint pas match_registry → forme
	// sous-requête by-match-id (p1.match_id). No-op Infinite. Item backlog H1.
	q := resolveCampaignExclusionByMatchID(Q10Encounters, r.pdb.TitleSlug, "p1.match_id")
	rows, err := db.QueryContext(ctx, q, r.pdb.XUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.EncounterRawRow
	for rows.Next() {
		var e domain.EncounterRawRow
		if err := rows.Scan(&e.XUID, &e.MatchCount, &e.AsTeammate, &e.AsEnemy, &e.AvgKDA); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		results = append(results, e)
	}
	return results, rows.Err()
}
