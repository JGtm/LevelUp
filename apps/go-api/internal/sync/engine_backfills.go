// Package sync — engine_backfills.go : routines de backfill manuel.
//
// Extrait de engine.go (refactor 2026-05-21). Regroupe tous les RunBackfill*
// du SyncEngine (engagement scores/coefs, LUSR, CSR, perf, comeback badges) +
// helpers privés associés (loadMedalExploitMap, selectMatchesForComebackBadges,
// loadAllMatchIDsForPlayer, loadFlaggedMatchIDs). Comportement INCHANGÉ — pur
// déplacement.
//
// Ces fonctions sont les points d'entrée des CLI `levelup backfill --*` et de
// l'admin RunBackfill HTTP. Elles partagent un pattern commun : prendre les
// leases player + shared, ouvrir les DBs et appeler le batch processor adéquat.
//
// Voir engine.go (struct SyncEngine) pour le contexte.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/platform/dblease"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
)

// RunBackfill détecte les matchs avec données manquantes et retourne la liste.
// Le scope doit être Resolve() avant appel. Retourne la liste des match_ids manquants.
func (e *SyncEngine) RunBackfill(ctx context.Context, scope *SyncScope) ([]string, error) {
	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return nil, fmt.Errorf("RunBackfill lease player: %w", err)
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return nil, fmt.Errorf("RunBackfill OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	// Sprint B1 commit 11b : acquireSharedWriter centralise lease + open
	// (Provider en B-swap, dblease+OpenSharedDB en legacy).
	sharedDB, releaseShared, err := e.acquireSharedWriter(ctxkeys.WithDBWriterLabel(ctx, "backfill_missing_data"))
	if err != nil {
		return nil, fmt.Errorf("RunBackfill: %w", err)
	}
	defer releaseShared()

	return FindMatchesMissingData(ctx, playerHandle.SQLDb(), sharedDB, e.xuid, scope)
}

// RunBackfillComebackBadges calcule et persiste le dominance_flag pour les
// matchs du joueur. Selectionne :
//   - tous les matchs si forceAll=true
//   - les matchs sans dominance_flag (ou flag=0) sinon
//
// Branche la fonction BackfillDominanceFlags (sync/comeback.go) au pipeline.
// Retourne le nombre de match_ids traites (et l'erreur infra si lease/open
// echoue).
//
// L'ingestion principale (RunDelta/RunFull) ne calcule PAS encore le flag a
// chaque match : ce backfill explicite est la voie d'entree pour peupler les
// dominance_flag (cf. PLAN_META_FOUNDATIONS_GO § 6.0.1, prerequis Phase 1
// pilote Squad/MatchView/Career).
// RunBackfillEngagementScores calcule et persiste le score d'engagement pour
// les matchs PvP du joueur (Phase 6 plan engagement). Si force=true, recalcule
// les scores existants ; sinon ne calcule que les manquants.
//
// Skip silencieux si la migration Phase 2 n'a pas ete appliquee (gating
// information_schema). Aucun appel API requis (calcul purement local depuis
// highlight_events deja synces).
func (e *SyncEngine) RunBackfillEngagementScores(ctx context.Context, force bool) (int, error) {
	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillEngagementScores lease player: %w", err)
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillEngagementScores OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	// Sprint B1 commit 11b : acquireSharedWriter centralise lease + open.
	sharedDB, releaseShared, err := e.acquireSharedWriter(ctxkeys.WithDBWriterLabel(ctx, "backfill_engagement"))
	if err != nil {
		return 0, fmt.Errorf("RunBackfillEngagementScores: %w", err)
	}
	defer releaseShared()

	n, intensities, err := batchComputeEngagementScores(ctx, playerHandle.SQLDb(), sharedDB, e.xuid, force)
	if err != nil {
		return n, err
	}
	// Writer déjà tenu (backfill) : flush direct — même sémantique qu'avant le
	// split 4a (le write était inline dans le compute).
	persistMatchIntensities(ctx, sharedDB, intensities)
	// Recompute des coefficients en queue : on a possiblement ajoute des
	// paces en DB, donc la mediane est a rafraichir.
	if nCoefs, errCoefs := batchRecomputeCoefficients(ctx, playerHandle.SQLDb(), e.xuid); errCoefs != nil {
		slog.WarnContext(ctx, "RunBackfillEngagementScores: recompute coefs failed",
			"xuid", e.xuid, "err", errCoefs)
	} else if nCoefs > 0 {
		slog.InfoContext(ctx, "RunBackfillEngagementScores: coefs updated",
			"xuid", e.xuid, "n_modes", nCoefs)
	}
	return n, nil
}

// RunBackfillEngagementCoefficients recompute UNIQUEMENT les coefficients
// d'engagement du joueur depuis les paces deja persistees (~5ms par joueur,
// 0 re-scan des matchs). A activer via SyncScope.EngagementCoefficients.
//
// Utile pour rafraichir apres un ajustement de formule sans devoir relancer
// le compute des scores. Skip silencieux si la migration des paces n'est
// pas appliquee (cf. batchRecomputeCoefficients).
//
// Retourne le nombre de modes_category mis a jour (0 a 2).
func (e *SyncEngine) RunBackfillEngagementCoefficients(ctx context.Context) (int, error) {
	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillEngagementCoefficients lease player: %w", err)
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillEngagementCoefficients OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	return batchRecomputeCoefficients(ctx, playerHandle.SQLDb(), e.xuid)
}

// RunBackfillLUSRDryRun simule un recompute LUSR sans écrire en DB. Compare
// l'état persisté actuel avec celui qui serait produit par un recompute force,
// agrégé par playlist_group. Utile pour valider l'impact d'un rebuild ART
// (cf. Phase 1 plan stabilisation 2026-05-22) avant d'engager l'écriture.
//
// Retourne un LUSRDryRunReport. Lock writer non requis (lecture seule) ;
// le caller doit s'assurer qu'aucun sync ne tourne en parallèle pour avoir
// un snapshot cohérent.
func (e *SyncEngine) RunBackfillLUSRDryRun(ctx context.Context) (*LUSRDryRunReport, error) {
	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return nil, fmt.Errorf("RunBackfillLUSRDryRun OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	// shared DB en read-only suffit pour le dry-run.
	sharedDB, releaseShared, err := e.acquireSharedWriter(ctxkeys.WithDBWriterLabel(ctx, "backfill_lusr_dryrun"))
	if err != nil {
		return nil, fmt.Errorf("RunBackfillLUSRDryRun: %w", err)
	}
	defer releaseShared()

	medalMap := e.loadMedalExploitMapBestEffort(ctx, sharedDB)
	return batchComputeLUSRPreview(ctx, playerHandle.SQLDb(), sharedDB, e.xuid, medalMap)
}

// RunFormulaSim simule les 5 variantes de formule LUSR sur les lastN derniers matchs.
// Lecture seule — aucune écriture DB. lastN=0 → tous les matchs.
func (e *SyncEngine) RunFormulaSim(ctx context.Context, lastN int) (*FormulaSimReport, error) {
	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return nil, fmt.Errorf("RunFormulaSim OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	sharedDB, releaseShared, err := e.acquireSharedWriter(ctxkeys.WithDBWriterLabel(ctx, "backfill_formula_sim"))
	if err != nil {
		return nil, fmt.Errorf("RunFormulaSim: %w", err)
	}
	defer releaseShared()

	medalMap := e.loadMedalExploitMapBestEffort(ctx, sharedDB)
	return RunFormulaSim(ctx, playerHandle.SQLDb(), sharedDB, e.xuid, medalMap, lastN)
}

// RecomputeLUSRCanonical recalcule le LUSR v2 canonique (TrueSkill2, ADR 0024)
// pour tous les matchs du joueur : reset watermark + replay complet via
// RecomputeLUSRCanonicalForPlayer. Remplace l'ancien RunBackfillLUSR v1 (CR C3 :
// deux chemins concurrents écrivaient match_skill_rank — v1 supprimé). Le replay
// v2 est toujours complet : il n'y a plus de paramètre `force`.
func (e *SyncEngine) RecomputeLUSRCanonical(ctx context.Context) (int, error) {
	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return 0, fmt.Errorf("RecomputeLUSRCanonical lease player: %w", err)
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return 0, fmt.Errorf("RecomputeLUSRCanonical OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	// acquireSharedWriter centralise lease + open : RecomputeLUSRCanonicalForPlayer
	// écrit une row sentinelle dans player_skill_state_v2 (shared DB).
	sharedDB, releaseShared, err := e.acquireSharedWriter(ctxkeys.WithDBWriterLabel(ctx, "backfill_lusr"))
	if err != nil {
		return 0, fmt.Errorf("RecomputeLUSRCanonical: %w", err)
	}
	defer releaseShared()

	return RecomputeLUSRCanonicalForPlayer(ctx, playerHandle.SQLDb(), sharedDB, e.xuid)
}

// RunBackfillCSR ré-importe les CSR par-match depuis l'API Halo skill pour
// tous les matchs classés du joueur qui n'ont pas encore de row CSR (ou tous
// si force=true). Cible : les matchs synchronisés AVANT la Phase B (sync
// nominal qui écrit déjà les CSR inline) et les cas où GetMatchSkill n'a pas
// retourné de RankRecap au moment du sync initial.
//
// Retourne le résumé d'exécution (matchs traités, restaurés, skippés, etc.).
func (e *SyncEngine) RunBackfillCSR(ctx context.Context, force bool) (CSRBackfillResult, error) {
	var empty CSRBackfillResult
	if e.tokens == nil || e.tokens.SpartanToken == "" {
		return empty, fmt.Errorf("RunBackfillCSR: tokens Halo absents (re-login requis)")
	}

	slog.InfoContext(ctx, "RunBackfillCSR: démarrage",
		"gamertag", e.gamertag, "xuid", e.xuid, "force", force)

	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return empty, fmt.Errorf("RunBackfillCSR lease player: %w", err)
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return empty, fmt.Errorf("RunBackfillCSR OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	// shared DB en read-only suffit : on ne fait que SELECT match_registry.
	// Sprint B1 commit 11b : passe par acquireSharedWriter pour cohérence
	// (Provider en B-swap, dblease+OpenSharedDB en legacy).
	sharedDB, releaseShared, err := e.acquireSharedWriter(ctxkeys.WithDBWriterLabel(ctx, "backfill_csr"))
	if err != nil {
		return empty, fmt.Errorf("RunBackfillCSR: %w", err)
	}
	defer releaseShared()

	var client HaloClient
	if e.customClient != nil {
		client = e.customClient
		slog.DebugContext(ctx, "RunBackfillCSR: utilisation client personnalisé (pool)")
	} else {
		client = NewHaloAPIClient(e.tokens.SpartanToken, e.tokens.ClearanceToken, 5)
		slog.DebugContext(ctx, "RunBackfillCSR: client Halo standard, 5 RPS")
	}

	res, err := BackfillCSRFromAPI(ctx, client, playerHandle.SQLDb(), sharedDB, e.xuid, force)
	if err != nil {
		slog.ErrorContext(ctx, "RunBackfillCSR: échec",
			"gamertag", e.gamertag, "err", err)
		return res, err
	}
	slog.InfoContext(ctx, "RunBackfillCSR: terminé",
		"gamertag", e.gamertag,
		"inserted", res.Inserted,
		"already_csr", res.AlreadyHadCSR,
		"skipped_no_recap", res.SkippedNoRankRecap,
		"skill_errors", res.SkillErrors,
	)
	return res, nil
}

// RunBackfillSharedCSR rattrape shared.match_csrs pour tous les matchs ranked
// où le joueur a participé. Wrapper sur BackfillSharedCSRsFromAPI avec
// acquisition du shared writer + résolution du client Halo (skip API si dry-run).
//
// Différent de RunBackfillCSR :
//   - RunBackfillCSR écrit player.match_skill_rank pour le joueur synchronisé.
//   - RunBackfillSharedCSR écrit shared.match_csrs pour TOUS les participants
//     du match (option A du plan : permet comparaisons cross-joueurs).
func (e *SyncEngine) RunBackfillSharedCSR(ctx context.Context, opts SharedCSRBackfillOpts) (SharedCSRBackfillResult, error) {
	var empty SharedCSRBackfillResult
	empty.DryRun = opts.DryRun

	// En non-dry-run, les tokens Halo sont indispensables pour appeler /skill.
	if !opts.DryRun && (e.tokens == nil || e.tokens.SpartanToken == "") {
		return empty, fmt.Errorf("RunBackfillSharedCSR: tokens Halo absents (re-login requis) — utiliser --dry-run sinon")
	}

	slog.InfoContext(ctx, "RunBackfillSharedCSR: démarrage",
		"gamertag", e.gamertag, "xuid", e.xuid, "force", opts.Force, "dry_run", opts.DryRun)

	sharedDB, releaseShared, err := e.acquireSharedWriter(ctxkeys.WithDBWriterLabel(ctx, "backfill_shared_csr"))
	if err != nil {
		return empty, fmt.Errorf("RunBackfillSharedCSR: %w", err)
	}
	defer releaseShared()

	var client HaloClient
	if !opts.DryRun {
		if e.customClient != nil {
			client = e.customClient
		} else {
			client = NewHaloAPIClient(e.tokens.SpartanToken, e.tokens.ClearanceToken, 5)
		}
	}
	// En dry-run, client peut rester nil — BackfillSharedCSRsFromAPI ne l'appelle pas.

	res, err := BackfillSharedCSRsFromAPI(ctx, client, sharedDB, e.xuid, opts)
	if err != nil {
		slog.ErrorContext(ctx, "RunBackfillSharedCSR: échec", "gamertag", e.gamertag, "err", err)
		return res, err
	}
	slog.InfoContext(ctx, "RunBackfillSharedCSR: terminé",
		"gamertag", e.gamertag,
		"ranked_total", res.RankedMatches,
		"already_complete", res.AlreadyComplete,
		"need_backfill", res.NeedBackfill,
		"fetched", res.Fetched,
		"inserted", res.Inserted,
		"skipped_no_recap", res.SkippedNoRankRecap,
		"skill_errors", res.SkillErrors,
		"upsert_errors", res.UpsertErrors,
		"dry_run", res.DryRun,
	)
	return res, nil
}

// RunBackfillPerf recalcule le performance score relatif pour tous les matchs du joueur.
// force=true : recalcule même si les matchs ont déjà un score (utile après changement de formule).
func (e *SyncEngine) RunBackfillPerf(ctx context.Context, force bool) (int, error) {
	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillPerf lease player: %w", err)
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillPerf OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	// Sprint B1 commit 11b : acquireSharedWriter centralise lease + open.
	sharedDB, releaseShared, err := e.acquireSharedWriter(ctxkeys.WithDBWriterLabel(ctx, "backfill_perf"))
	if err != nil {
		return 0, fmt.Errorf("RunBackfillPerf: %w", err)
	}
	defer releaseShared()

	medalMap := e.loadMedalExploitMapBestEffort(ctx, sharedDB)
	return batchComputePerformanceScores(ctx, playerHandle.SQLDb(), sharedDB, e.xuid, medalMap, force)
}

// loadMedalExploitMapBestEffort charge les scores d'exploit médailles depuis la metadata DB.
// Retourne nil en cas d'erreur (le LUSR/Perf fonctionne sans données médailles).
func (e *SyncEngine) loadMedalExploitMapBestEffort(ctx context.Context, sharedDB *sql.DB) map[string]float64 {
	return loadMedalExploitMap(ctx, e.metadataDBPath, sharedDB, e.xuid)
}

// loadMedalExploitMap : variante package-level réutilisable hors SyncEngine
// (ex: MatchRecomputer). Best-effort : retourne nil si la metadata DB est
// indisponible ou si le calcul échoue — perf/LUSR fonctionnent sans.
func loadMedalExploitMap(ctx context.Context, metadataDBPath string, sharedDB *sql.DB, xuid string) map[string]float64 {
	if metadataDBPath == "" {
		return nil
	}
	// Phase 2 PLAN_FIX_SYNC_RELIABILITY_2026-05-24 (site residuel detecte
	// par audit grep 2026-05-25) : passage par le cache duckdbpkg.OpenReadOnly
	// pour aligner le DSN avec les autres sites RO du sync engine. Empeche
	// le bug "Can't open a connection with a different configuration"
	// lorsque loadMedalExploitMap tourne en concurrence avec engine.go:249.
	metaHandle, err := duckdbpkg.OpenReadOnly(metadataDBPath)
	if err != nil {
		slog.DebugContext(ctx, "loadMedalExploitMap: ouverture metaDB échouée", "err", err)
		return nil
	}
	defer metaHandle.Close()
	metaDB := metaHandle.SQLDb()

	diffMap, err := LoadMedalDifficultyFromMeta(ctx, metaDB)
	if err != nil || len(diffMap) == 0 {
		slog.DebugContext(ctx, "loadMedalExploitMap: difficulty map vide", "err", err)
		return nil
	}
	result, err := ComputeMedalExploitByMatch(ctx, sharedDB, diffMap, xuid)
	if err != nil {
		slog.DebugContext(ctx, "loadMedalExploitMap: compute échoué", "err", err)
		return nil
	}
	return result
}

func (e *SyncEngine) RunBackfillComebackBadges(ctx context.Context, forceAll bool) (int, error) {
	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillComebackBadges lease player: %w", err)
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillComebackBadges OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	// Sprint B1 commit 11b : acquireSharedWriter centralise lease + open.
	sharedDB, releaseShared, err := e.acquireSharedWriter(ctxkeys.WithDBWriterLabel(ctx, "backfill_comeback"))
	if err != nil {
		return 0, fmt.Errorf("RunBackfillComebackBadges: %w", err)
	}
	defer releaseShared()

	matchIDs, err := selectMatchesForComebackBadges(ctx, playerHandle.SQLDb(), sharedDB, e.xuid, forceAll)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillComebackBadges select: %w", err)
	}
	if len(matchIDs) == 0 {
		slog.InfoContext(ctx, "comeback-badges: aucun match a traiter",
			"player", e.gamertag, "force_all", forceAll)
		return 0, nil
	}

	slog.InfoContext(ctx, "comeback-badges: backfill en cours",
		"player", e.gamertag, "match_count", len(matchIDs), "force_all", forceAll)
	if err := BackfillDominanceFlags(ctx, sharedDB, playerHandle.SQLDb(), e.xuid, matchIDs); err != nil {
		return 0, fmt.Errorf("RunBackfillComebackBadges backfill: %w", err)
	}
	return len(matchIDs), nil
}

// selectMatchesForComebackBadges retourne les match_ids du joueur a traiter
// pour le backfill dominance_flag.
//
// Si forceAll=true : tous les matchs du joueur dans shared.match_participants.
// Sinon : uniquement les matchs ou player_match_enrichment.dominance_flag est
// nul ou egal a 0 (cas par defaut "manquant").
func selectMatchesForComebackBadges(
	ctx context.Context,
	playerDB, sharedDB *sql.DB,
	xuid string,
	forceAll bool,
) ([]string, error) {
	allIDs, err := loadAllMatchIDsForPlayer(ctx, sharedDB, xuid)
	if err != nil {
		return nil, fmt.Errorf("load all match_ids: %w", err)
	}
	if forceAll {
		return allIDs, nil
	}
	flagged, err := loadFlaggedMatchIDs(ctx, playerDB)
	if err != nil {
		return nil, fmt.Errorf("load flagged match_ids: %w", err)
	}
	flaggedSet := make(map[string]struct{}, len(flagged))
	for _, id := range flagged {
		flaggedSet[id] = struct{}{}
	}
	out := make([]string, 0, len(allIDs))
	for _, id := range allIDs {
		if _, ok := flaggedSet[id]; !ok {
			out = append(out, id)
		}
	}
	return out, nil
}

// loadAllMatchIDsForPlayer retourne tous les match_id du joueur (shared DB).
func loadAllMatchIDsForPlayer(ctx context.Context, sharedDB *sql.DB, xuid string) ([]string, error) {
	rows, err := sharedDB.QueryContext(ctx,
		`SELECT match_id FROM match_participants WHERE xuid = ? ORDER BY match_id`, xuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// loadFlaggedMatchIDs retourne les match_id dont la dominance a DÉJÀ été calculée
// (dominance_flag NON-NULL, valeur 0 INCLUSE) — player DB.
//
// Append-only #23046 — IDEMPOTENCE : on inclut dominance_flag=0. Un match
// non-dominant (0 = ni domination ni humiliation ni comeback = la MAJORITÉ des
// matchs) recalculé donne TOUJOURS 0 ; le traiter comme « non calculé » le ferait
// ré-INSÉRER (stage='dominance', valeur 0) à chaque backfill admin non-force →
// croissance non bornée. Aligné sur le chemin per-sync (dominance_flag IS NULL =
// jamais calculé). Le re-calcul volontaire passe par forceAll=true.
func loadFlaggedMatchIDs(ctx context.Context, playerDB *sql.DB) ([]string, error) {
	rows, err := playerDB.QueryContext(ctx,
		`SELECT match_id FROM player_match_enrichment_latest
		 WHERE dominance_flag IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
