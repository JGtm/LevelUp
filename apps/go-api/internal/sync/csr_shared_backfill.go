// Package sync — csr_shared_backfill.go : backfill historique de
// shared.match_csrs (Option A du plan pipeline CSR).
//
// Le sync nominal écrit déjà shared.match_csrs à mesure que les matchs ranked
// sont synchronisés (via le batch Collect→Persist : batch.Shared.MatchCSRs,
// cf. buildBatchFromFetchedMatch → submitMatchAsBatch).
// Ce backfill rattrape les matchs ranked qui étaient déjà en DB AVANT le câblage
// Option A.
//
// Flow :
//  1. Lister les match_ids ranked où le joueur a participé (join match_participants).
//  2. Pour chaque match, compter les participants vs les rows match_csrs existantes.
//     Si tous présents et !force → skip (idempotence).
//  3. Sinon (et !dry-run) : fetch GetMatchSkill avec TOUS les xuids du match,
//     extraire les rows via ExtractAllSharedCSRRows, batch UPSERT.
//  4. Mode dry-run : compte uniquement (pas d'appel API ni d'écriture).
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/persist"
)

// SharedCSRBackfillOpts paramètre BackfillSharedCSRsFromAPI.
type SharedCSRBackfillOpts struct {
	// Force : ignore l'idempotence (re-fetch même les matchs déjà complets).
	Force bool
	// DryRun : compte les matchs nécessitant un backfill, ne fait aucun appel
	// API ni écriture. Idéal pour valider l'ampleur avant exécution.
	DryRun bool
}

// SharedCSRBackfillResult résume l'exécution.
type SharedCSRBackfillResult struct {
	// RankedMatches : matchs ranked du joueur trouvés dans le registry.
	RankedMatches int
	// AlreadyComplete : matchs avec match_csrs déjà rempli (N rows = N participants).
	// Comptés uniquement en mode !force.
	AlreadyComplete int
	// NeedBackfill : matchs avec rows match_csrs manquantes (ou tous si force).
	NeedBackfill int
	// Fetched : appels GetMatchSkill effectués (== NeedBackfill en non-dry-run
	// si pas d'erreur context-cancel).
	Fetched int
	// Inserted : nombre total de rows insérées/remplacées dans match_csrs.
	Inserted int
	// SkippedNoRankRecap : payload skill OK mais aucun PostMatchCsr → 0 rows
	// extraites (match classé côté registry mais drift API).
	SkippedNoRankRecap int
	// SkillErrors : erreurs GetMatchSkill (réseau, 401, 404) — non-bloquantes.
	SkillErrors int
	// UpsertErrors : erreurs UpsertSharedCSRs (DB lock, schéma) — non-bloquantes.
	UpsertErrors int
	// DryRun : reflète opts.DryRun pour le résumé.
	DryRun bool
}

// BackfillSharedCSRsFromAPI rattrape les rows shared.match_csrs manquantes pour
// les matchs ranked où le joueur xuid a participé. Idempotent par défaut.
func BackfillSharedCSRsFromAPI(
	ctx context.Context,
	client HaloClient,
	sharedDB *sql.DB,
	xuid string,
	opts SharedCSRBackfillOpts,
) (SharedCSRBackfillResult, error) {
	res := SharedCSRBackfillResult{DryRun: opts.DryRun}

	matches, err := loadRankedMatchesForSharedCSRBackfill(ctx, sharedDB, xuid)
	if err != nil {
		return res, fmt.Errorf("BackfillSharedCSRsFromAPI: load ranked: %w", err)
	}
	res.RankedMatches = len(matches)
	if len(matches) == 0 {
		slog.InfoContext(ctx, "BackfillSharedCSRsFromAPI: aucun match ranked pour ce joueur",
			"xuid", xuid)
		return res, nil
	}

	slog.InfoContext(ctx, "BackfillSharedCSRsFromAPI: démarrage",
		"xuid", xuid,
		"ranked_total", res.RankedMatches,
		"force", opts.Force,
		"dry_run", opts.DryRun,
	)

	const progressEveryN = 50
	for _, m := range matches {
		// Respect context (annulation utilisateur).
		if err := ctx.Err(); err != nil {
			slog.WarnContext(ctx, "BackfillSharedCSRsFromAPI: contexte annulé",
				"xuid", xuid, "progress", res.Fetched)
			return res, err
		}

		nParticipants, errCount := countMatchParticipants(ctx, sharedDB, m.MatchID)
		if errCount != nil {
			res.SkillErrors++ // catégorisation simple : erreur DB classée comme erreur de fetch
			continue
		}
		nExisting, errCount := countMatchCSRRows(ctx, sharedDB, m.MatchID)
		if errCount != nil {
			res.SkillErrors++
			continue
		}

		if !opts.Force && nExisting >= nParticipants && nParticipants > 0 {
			res.AlreadyComplete++
			continue
		}
		res.NeedBackfill++

		if opts.DryRun {
			// Mode dry-run : on logge l'écart, on n'appelle pas l'API.
			slog.DebugContext(ctx, "BackfillSharedCSRsFromAPI: would backfill",
				"match_id", m.MatchID, "participants", nParticipants, "existing_csr", nExisting,
				"gap", nParticipants-nExisting)
			continue
		}

		// Récupère TOUS les xuids participants pour l'appel /skill.
		xuids, errXUIDs := loadParticipantXUIDs(ctx, sharedDB, m.MatchID)
		if errXUIDs != nil || len(xuids) == 0 {
			res.SkillErrors++
			slog.WarnContext(ctx, "BackfillSharedCSRsFromAPI: loadParticipantXUIDs échoué",
				"match_id", m.MatchID, "err", errXUIDs)
			continue
		}

		skillData, skillErr := client.GetMatchSkill(ctx, m.MatchID, xuids)
		res.Fetched++
		if skillErr != nil {
			res.SkillErrors++
			slog.WarnContext(ctx, "BackfillSharedCSRsFromAPI: GetMatchSkill échoué",
				"match_id", m.MatchID, "xuid", xuid, "err", skillErr)
			continue
		}

		rows := ExtractAllSharedCSRRows(&m, skillData)
		if len(rows) == 0 {
			res.SkippedNoRankRecap++
			slog.DebugContext(ctx, "BackfillSharedCSRsFromAPI: aucun PostMatchCsr dans le payload",
				"match_id", m.MatchID)
			continue
		}

		if err := UpsertSharedCSRs(ctx, sharedDB, rows); err != nil {
			res.UpsertErrors++
			slog.WarnContext(ctx, "BackfillSharedCSRsFromAPI: UpsertSharedCSRs échoué",
				"match_id", m.MatchID, "rows", len(rows), "err", err)
			continue
		}
		res.Inserted += len(rows)

		if res.Fetched%progressEveryN == 0 {
			slog.InfoContext(ctx, "BackfillSharedCSRsFromAPI: progression",
				"xuid", xuid,
				"fetched", res.Fetched,
				"inserted", res.Inserted,
				"need_backfill", res.NeedBackfill,
				"errors", res.SkillErrors+res.UpsertErrors,
			)
		}
	}

	slog.InfoContext(ctx, "BackfillSharedCSRsFromAPI: terminé",
		"xuid", xuid,
		"ranked_total", res.RankedMatches,
		"already_complete", res.AlreadyComplete,
		"need_backfill", res.NeedBackfill,
		"fetched", res.Fetched,
		"inserted", res.Inserted,
		"skipped_no_recap", res.SkippedNoRankRecap,
		"skill_errors", res.SkillErrors,
		"upsert_errors", res.UpsertErrors,
		"dry_run", opts.DryRun,
	)
	return res, nil
}

// loadRankedMatchesForSharedCSRBackfill charge les matchs ranked où le joueur
// xuid a participé, avec season_id (peut être NULL pour les matchs antérieurs
// au backfill Phase 1). Triés ASC pour traiter chronologiquement.
//
// Pré-filtre (is_ranked OU nom contenant « ranked » OU playlist_id de la référence), identique à la fonction player-scoped :
// il choisit les CANDIDATS, la décision de classement appartient au payload skill
// (SkillShowsRankedMatch, via ExtractAllSharedCSRRows).
func loadRankedMatchesForSharedCSRBackfill(ctx context.Context, sharedDB *sql.DB, xuid string) ([]MatchRegistryRow, error) {
	ph, args := rankedCandidatePlaylistIDs()
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT DISTINCT r.match_id, r.start_time, r.season_id
		FROM match_registry r
		JOIN match_participants mp ON mp.match_id = r.match_id
		WHERE mp.xuid = ?
		  AND r.start_time IS NOT NULL
		  AND (
		    COALESCE(r.is_ranked, FALSE) = TRUE
		    OR STRPOS(LOWER(COALESCE(r.playlist_name, '')), 'ranked') > 0
		    OR STRPOS(LOWER(COALESCE(r.pair_name, '')), 'ranked') > 0
		    OR LOWER(COALESCE(r.playlist_id, '')) IN (`+ph+`)
		  )
		ORDER BY r.start_time ASC`, append([]any{xuid}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MatchRegistryRow
	for rows.Next() {
		var id string
		var start time.Time
		var sid sql.NullString
		if err := rows.Scan(&id, &start, &sid); err != nil {
			return nil, err
		}
		row := MatchRegistryRow{MatchID: id, StartTime: start}
		if sid.Valid {
			s := sid.String
			row.SeasonID = &s
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func countMatchParticipants(ctx context.Context, db *sql.DB, matchID string) (int, error) {
	var n int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM match_participants WHERE match_id = ?`, matchID).Scan(&n)
	return n, err
}

// nilerr explicite : table absente = 0 rows attendu (caller traite comme "tout à backfiller").
//
// **Lecture via match_csrs_latest** (Phase 2.F) : la table physique est
// append-only et peut contenir N versions par (match_id, xuid). Pour
// l'idempotence du backfill, on veut le compte fonctionnel "combien de
// (match_id, xuid) distincts existent", donné par la vue latest.
//
//nolint:unparam // err maintenu pour cohérence avec countMatchParticipants ;
func countMatchCSRRows(ctx context.Context, db *sql.DB, matchID string) (int, error) {
	var n int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM match_csrs_latest WHERE match_id = ?`, matchID).Scan(&n)
	if err != nil {
		// Table absente → 0 rows. On retourne (0, nil) pour ne pas casser le flow.
		// Le caller traitera comme "tout à backfiller".
		return 0, nil //nolint:nilerr
	}
	return n, nil
}

func loadParticipantXUIDs(ctx context.Context, db *sql.DB, matchID string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT DISTINCT xuid FROM match_participants WHERE match_id = ? AND xuid IS NOT NULL AND xuid <> ''`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var x string
		if err := rows.Scan(&x); err != nil {
			continue
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// ─── Rattrapage de is_ranked depuis le CSR (A4) ────────────────────────────────
// Rattrapage de `match_registry.is_ranked` depuis la
// présence du CSR par match (plan ajustements 2026-10-14, A4).
//
// # POURQUOI
//
// Le sync posait is_ranked par le nom de playlist, absent du payload : 876 matchs « Ranked » sur 877
// de 2026 sont restés `is_ranked = FALSE` (G5). Depuis la règle unique (csr_writes.go,
// décision D-10) le sync, l'import OpenSpartan et les rattrapages CSR posent ou écrivent selon le
// RankRecap du payload skill. Ce rattrapage répare les lignes DÉJÀ écrites : tout match qui porte
// une ligne dans `match_csrs_latest` a un CSR écrit selon cette règle (ou, avant elle, selon un
// nom de playlist « ranked » : même verdict), donc est classé.
//
// # CE QUI EST ÉCRIT, CE QUI NE L'EST JAMAIS
//
// FAUX/NULL vers VRAI, un match à la fois (persist.RegistryRankedPersister). Jamais de
// déclassement : un match classé sans CSR (CSR pas encore récupéré) reste classé ; il est compté
// (RankedWithoutCSR) pour que `backfill --csr` / `backfill --shared-csr` le rattrapent.
//
// # ORDRE D'EMPLOI
//
// Après `backfill --shared-csr` (qui écrit match_csrs selon la règle unique), pour que les matchs
// dont le CSR vient d'être retrouvé soient reclassés dans la même séquence. Idempotent.

// RegistryRankedOptions paramètre le rattrapage. DryRun : compter sans écrire.
type RegistryRankedOptions struct {
	DryRun bool
}

// RegistryRankedStats : compteurs du rattrapage, par match.
type RegistryRankedStats struct {
	DryRun bool
	// Candidates : matchs portant un CSR et dont is_ranked est faux ou NULL.
	Candidates int
	// Fixed : matchs passés à is_ranked = TRUE (en simulation : qui le seraient).
	Fixed int
	// AlreadyRanked : matchs portant un CSR et déjà classés AVANT ce passage.
	AlreadyRanked int
	// RankedWithoutCSR : matchs classés (is_ranked = TRUE) sans aucune ligne CSR. Indicatif, jamais
	// déclassés.
	RankedWithoutCSR int
	// Errors : écritures échouées (journalisées une à une).
	Errors int
}

const (
	qRegistryRankedCandidates = `
SELECT r.match_id
FROM match_registry r
WHERE COALESCE(r.is_ranked, FALSE) = FALSE
  AND r.match_id IN (SELECT match_id FROM match_csrs_latest WHERE rating_type = 'CSR')
ORDER BY r.match_id`

	qRegistryRankedCounts = `
WITH csr AS (SELECT DISTINCT match_id FROM match_csrs_latest WHERE rating_type = 'CSR')
SELECT
  COUNT(*) FILTER (WHERE COALESCE(r.is_ranked, FALSE) AND c.match_id IS NOT NULL),
  COUNT(*) FILTER (WHERE COALESCE(r.is_ranked, FALSE) AND c.match_id IS NULL)
FROM match_registry r
LEFT JOIN csr c ON c.match_id = r.match_id`
)

// BackfillRegistryIsRanked pose is_ranked = TRUE sur les matchs qui portent un CSR par match (cf.
// en-tête). sharedDB doit porter le writer de shared_matches_v2 hors simulation.
func BackfillRegistryIsRanked(ctx context.Context, sharedDB *sql.DB, opts RegistryRankedOptions) (RegistryRankedStats, error) {
	stats := RegistryRankedStats{DryRun: opts.DryRun}
	if sharedDB == nil {
		return stats, fmt.Errorf("BackfillRegistryIsRanked: sharedDB nil")
	}
	ids, err := loadRegistryRankedCandidates(ctx, sharedDB)
	if err != nil {
		return stats, err
	}
	stats.Candidates = len(ids)
	persister := persist.NewRegistryRankedPersister(sharedDB)
	for _, id := range ids {
		if opts.DryRun {
			stats.Fixed++
			continue
		}
		changed, err := persister.MarkMatchRanked(ctx, id)
		if err != nil {
			stats.Errors++
			slog.ErrorContext(ctx, "BackfillRegistryIsRanked: écriture échouée", "match_id", id, "err", err)
			continue
		}
		if changed {
			stats.Fixed++
		}
	}
	// Comptes de contrôle lus APRÈS les écritures : « classé avec CSR » inclut ce qu'on vient de
	// corriger ; on retranche Fixed pour rendre ce qui l'était déjà. En simulation, rien n'a changé.
	if err := sharedDB.QueryRowContext(ctx, qRegistryRankedCounts).
		Scan(&stats.AlreadyRanked, &stats.RankedWithoutCSR); err != nil {
		return stats, fmt.Errorf("BackfillRegistryIsRanked: comptes de contrôle: %w", err)
	}
	if !opts.DryRun {
		stats.AlreadyRanked -= stats.Fixed
	}
	slog.InfoContext(ctx, "BackfillRegistryIsRanked: terminé",
		"dry_run", opts.DryRun, "candidats", stats.Candidates, "corriges", stats.Fixed,
		"deja_classes", stats.AlreadyRanked, "classes_sans_csr", stats.RankedWithoutCSR, "errors", stats.Errors)
	return stats, nil
}

func loadRegistryRankedCandidates(ctx context.Context, sharedDB *sql.DB) ([]string, error) {
	rs, err := sharedDB.QueryContext(ctx, qRegistryRankedCandidates)
	if err != nil {
		return nil, fmt.Errorf("BackfillRegistryIsRanked: lecture des candidats: %w", err)
	}
	defer rs.Close()
	var out []string
	for rs.Next() {
		var id string
		if err := rs.Scan(&id); err != nil {
			return nil, fmt.Errorf("BackfillRegistryIsRanked: scan: %w", err)
		}
		out = append(out, id)
	}
	if err := rs.Err(); err != nil {
		return nil, fmt.Errorf("BackfillRegistryIsRanked: itération: %w", err)
	}
	return out, nil
}
