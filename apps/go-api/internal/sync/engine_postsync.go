// Package sync — engine_postsync.go : pipeline post-sync + sync achievements.
//
// Extrait de engine.go (refactor 2026-05-21). Regroupe :
//   - runConditionalPostSync : branche post-sync selon matchs insérés ou heal.
//   - runPostSyncPipeline : pipeline complet 16 étapes (heals fallback,
//     had_bot, sessions, perf, engagement, assists, weapon kills,
//     citations (1.6), dominance_flag (1.7), LUSR, CSR, friends, aggregates,
//     achievements). Citations et dominance sont des étapes primaires du pipeline,
//     pas des heals : ils tournent pour chaque nouveau match et comblent le backlog.
//   - runCSRSnapshotSync : CSR snapshots best-effort si csrSeasonID renseigné.
//   - runAchievementsSync + RunAchievementsOnly : sync Xbox achievements via
//     TokenProvider (resolveAccessTokenFromDB → XSTS → SyncAchievements).
//   - hasMatchesNeedingScoreRefresh : heuristique heal-only path.
//   - resolveAccessTokenFromDB : lecture cache MSAL/refresh + fallback env.
//
// Voir engine.go (struct SyncEngine + run()) pour le contexte.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/dblease"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
)

// trackFatalErr collecte les erreurs FATAL DuckDB (IsInvalidatedError) dans
// r.FatalErrors pour propagation vers SyncResult.Errors. À appeler après
// chaque slog.WarnContext "post-sync: X échoué" du pipeline. Pas-op pour
// les erreurs non-fatales (les WARN restent informatifs).
//
// Sémantique (Phase 5 ART — Status sync honnête) : sans ce tracking, un
// FATAL sur le post-sync (LUSR/aggregates/citations/friends) reste loggé en
// WARN, le sync se termine avec status="success" et le monitoring ne voit
// pas la dégradation. Avec ce tracking, le status passe à "partial_success".
//
// `stepName` doit correspondre au libellé du WARN (ex. "LUSR", "citations",
// "aggregates") pour pouvoir corréler logs ↔ FatalErrors.
func trackFatalErr(r *domain.PostSyncResult, stepName string, err error) {
	if err == nil || !duckdbpkg.IsInvalidatedError(err) {
		return
	}
	r.FatalErrors = append(r.FatalErrors, fmt.Sprintf("%s: %v", stepName, err))
}

// runConditionalPostSync exécute le pipeline complet si des matchs ont été insérés,
// sinon rafraîchit au moins la carrière pour mettre à jour le snapshot joueur.
func (e *SyncEngine) runConditionalPostSync(
	ctx context.Context,
	playerDB, sharedDB *sql.DB,
	client HaloClient,
	matchesInserted int,
	insertedIDs []string,
) domain.PostSyncResult {
	if matchesInserted > 0 {
		slog.InfoContext(ctx, "sync: lancement pipeline post-sync", "gamertag", e.gamertag)
		return e.runPostSyncPipeline(ctx, playerDB, sharedDB, client, insertedIDs)
	}

	// Pas de nouveaux matchs : heal skill décommissionné (2026-06-01, cf.
	// runPostSyncPipeline). On se contente de détecter les matchs avec scores
	// manquants (engagement/perf NULL) pour relancer le recalcul post-sync —
	// pur calcul dérivé, aucune écriture API/shared concurrente.
	needsScoreRefresh, _ := hasMatchesNeedingScoreRefresh(ctx, playerDB, sharedDB, e.xuid)
	if needsScoreRefresh {
		slog.InfoContext(ctx, "sync: aucun match inséré — scores manquants → lancement post-sync complet",
			"gamertag", e.gamertag, "needs_score_refresh", needsScoreRefresh)
		return e.runPostSyncPipeline(ctx, playerDB, sharedDB, client, nil)
	}
	slog.DebugContext(ctx, "sync: aucun match inséré — refresh CSR + achievements seul (carrière live découplé)", "gamertag", e.gamertag)
	// Carrière (XP + Spartan ID) retirée du post-sync : service.CareerLiveService
	// la rafraîchit live à chaque chargement de /pages/home.
	res := domain.PostSyncResult{}
	if csrs, err := e.runCSRSnapshotSync(ctx, playerDB, client); err != nil {
		trackFatalErr(&res, "CSR snapshots", err)
	} else if len(csrs) > 0 {
		e.seedCatalogFromCSRs(ctx, csrs)
	}
	res.AchievementsSynced = e.runAchievementsSync(ctx, playerDB)
	return res
}

// hasMatchesNeedingScoreRefresh indique si au moins un match a des scores
// manquants (performance OR engagement IS NULL) parmi les matchs joués par
// ce joueur. Heuristique pour décider si runPostSyncPipeline doit tourner
// même quand aucun nouveau match n'a été inséré.
func hasMatchesNeedingScoreRefresh(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) (bool, error) {
	_ = sharedDB // signature future-proof si on veut joindre shared.match_participants
	_ = xuid
	var n int
	err := playerDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM player_match_enrichment
		WHERE engagement_score IS NULL OR performance_score IS NULL
		LIMIT 1
	`).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// runPostSyncPipeline exécute le pipeline post-sync :
// 1. Performance scores
// 2. LUSR (TrueSkill 2)
// 3. Career rank
// 4. Aggregates (materialized views)
func (e *SyncEngine) runPostSyncPipeline(
	ctx context.Context,
	playerDB, sharedDB *sql.DB,
	client HaloClient,
	insertedIDs []string,
) (r domain.PostSyncResult) {
	// Capture des panics du pipeline post-sync — avant ce defer un panic dans
	// n'importe quelle étape (perf scores, LUSR, citations, etc.) tuait
	// silencieusement tout le process server sans laisser de stack trace dans
	// les logs (cf. incident 2026-05-22, post-sync JGtm tué à 18:41:19 sans
	// trace). On capture, on logue le stack, et on retourne le résultat partiel
	// — le sync engine continue avec ce qui a réussi, le tick scheduler suivant
	// pourra retenter (toutes les étapes sont idempotentes).
	defer func() {
		if rec := recover(); rec != nil {
			slog.ErrorContext(ctx, "post-sync: PANIC récupéré",
				"gamertag", e.gamertag,
				"panic", fmt.Sprintf("%v", rec),
				"stack", string(debug.Stack()),
				"hint", "résultat partiel retourné, étapes idempotentes — prochain tick retentera",
			)
		}
	}()

	// Sprint B1 commit 18 : event_id pour tracer le pipeline post-sync à
	// travers ses étapes (bot teammate, sessions, perf scores, engagement,
	// LUSR, citations, CSR, friends, dominance, aggregates). Tous les sous-logs
	// hériteront automatiquement.
	ctx, evID := logging.WithEvent(ctx, "sync.postSync:"+e.gamertag)
	slog.InfoContext(ctx, "post-sync: pipeline démarré",
		"gamertag", e.gamertag, "matches_inserted", len(insertedIDs), "event", evID)

	// Fail-fast RC-A (defense-in-depth) : le post-sync écrit encore shared
	// (LUSR match_skill_rank, dominance match_registry). Si le handle reçu est
	// attaché en read-only (régression du bug RC-A 2026-06-01), on le détecte
	// immédiatement et on logue au lieu de laisser chaque étape échouer en
	// silence. Best-effort : nil/erreur de sonde n'interrompt pas le pipeline.
	if err := assertSharedWritable(ctx, sharedDB); err != nil {
		slog.ErrorContext(ctx, "post-sync: shared attaché en read-only (RC-A) — écritures shared vont échouer",
			"gamertag", e.gamertag, "err", err)
		trackFatalErr(&r, "shared read-only", err)
	}

	// -2 Ensure player_match_enrichment rows exist for all matches where the
	// player has a row in shared.match_participants (incident 2026-05-27).
	//
	// Sans cette étape, les joueurs qui apparaissent en shared via le sync
	// d'un teammate (cas escouade) n'ont jamais de row INSERT côté
	// player_match_enrichment, et tous les UPDATE qui suivent (perf, lusr,
	// sessions, citations, dominance) sont des no-op silencieux. Voir
	// ensure_enrichment_rows.go pour le contexte complet du bug.
	//
	// Idempotent : 0 row créée pour un joueur dont tous les enrichments
	// existent déjà (cas stationnaire JGtm post-sync).
	if n, err := ensurePlayerEnrichmentRows(ctx, playerDB, sharedDB, e.xuid); err != nil {
		slog.WarnContext(ctx, "post-sync: ensure enrichment rows échoué",
			"gamertag", e.gamertag, "err", err)
		// Best-effort : on continue le pipeline même si ça échoue (les UPDATE
		// resteront no-op pour les matchs sans row, mais le reste du pipeline
		// peut quand même tourner pour les matchs qui ont déjà leurs rows).
	} else if n > 0 {
		slog.InfoContext(ctx, "post-sync: enrichment rows créées",
			"gamertag", e.gamertag, "rows_created", n)
	}

	// HEALS DÉCOMMISSIONNÉS (2026-06-01) — stats / skill / events / weapon.
	//
	// Le film highlights est disponible immédiatement ~99% du temps : le sync
	// PRIMAIRE (engine_fetch.go) récupère et persiste skill (GetMatchSkill +
	// MergeSkillIntoParticipants), participants (ExtractParticipants) et events
	// (GetHighlightEventsChunk → insertHighlightEventsFromData) au 1er passage,
	// via le writer RW orchestré (persist.BatchBuilder → *Persister). Les heals
	// post-sync ne faisaient que dupliquer ce chemin sur une fenêtre non
	// maîtrisée — et `healSkillForMissingMatches` réécrivait match_participants
	// en `ON CONFLICT DO UPDATE`, ce qui corrompait l'ART (incident 2026-06-01).
	// Décision produit : aucun heal automatique. Le 1% résiduel (film absent au
	// 1er sync) se récupère via backfill CLI explicite et contrôlé.
	// Cf. ADR 0019 + thought_log 2026-06-01.

	// -0.3 had_bot, 0 sessions, 1 perf, 1.5 engagement, 1.5.b coefs, 1.52 assists —
	// bloc de scoring enrichment (player DB), extrait pour réduire la taille de
	// runPostSyncPipeline (cf. revue D7-3).
	e.runScoringSteps(ctx, playerDB, sharedDB, &r)

	// 1.55 Weapon kills — pipeline film pour les matchs nouvellement insérés.
	// Best-effort : films absents (404/410) sont normaux pour les vieux matchs
	// et n'échouent pas le sync. Limité aux nouveaux matchs (insertedIDs) pour
	// éviter de re-traiter l'historique à chaque sync.
	if len(insertedIDs) > 0 {
		done, noFilm, werr := processWeaponKillsInline(ctx, sharedDB, client, e.xuid, insertedIDs)
		if werr != nil {
			slog.WarnContext(ctx, "post-sync: weapon kills échoué", "gamertag", e.gamertag, "err", werr)
			trackFatalErr(&r, "weapon kills", werr)
		}
		r.WeaponKillsProcessed = done
		r.WeaponKillsNoFilm = noFilm
		if done > 0 || noFilm > 0 {
			slog.InfoContext(ctx, "post-sync: weapon kills",
				"gamertag", e.gamertag, "done", done, "no_film", noFilm)
		}
	}

	// Registry names heal DÉCOMMISSIONNÉ (2026-06-01) — map_name/pair_name/
	// playlist_name/game_variant_name sont résolus au sync PRIMAIRE via
	// EnrichRegistryFromMetadata (metadata saine). Le nettoyage one-shot des
	// GUID hérités d'un incident ART metadata se fait via `cmd/backfill_registry_names`
	// (CLI explicite), pas un heal post-sync automatique.

	// 1.6 Citations — pipeline primaire, pas un heal. Traite tous les matchs
	// absents de match_citations (LEFT JOIN IS NULL). Le sentinel "_processed"
	// (fix citations.go) empêche les matchs à 0 delta d'être re-traités.
	if n, err := e.runPostSyncCitations(ctx, playerDB, sharedDB); err != nil {
		slog.WarnContext(ctx, "post-sync: citations échoué", "gamertag", e.gamertag, "err", err)
		trackFatalErr(&r, "citations", err)
	} else if n > 0 {
		r.CitationsComputed = n
		slog.InfoContext(ctx, "post-sync: citations calculées", "gamertag", e.gamertag, "count", n)
	}

	// 1.7 Dominance flags — matchs nouvellement insérés + backlog (dominance_flag IS NULL).
	// Reconstruit la courbe score depuis highlight_events. Valeur 0 = calculé/sans dominance,
	// NULL = jamais calculé. UPSERT idempotent.
	{
		missingIDs, merr := selectMatchesMissingDominanceFlags(ctx, playerDB)
		if merr != nil {
			slog.WarnContext(ctx, "post-sync: dominance detect échoué", "gamertag", e.gamertag, "err", merr)
			trackFatalErr(&r, "dominance detect", merr)
		}
		allDominanceIDs := mergeUniqMatchIDs(insertedIDs, missingIDs)
		if len(allDominanceIDs) > 0 {
			if err := BackfillDominanceFlags(ctx, sharedDB, playerDB, e.xuid, allDominanceIDs); err != nil {
				slog.WarnContext(ctx, "post-sync: dominance flags échoué",
					"gamertag", e.gamertag, "err", err, "count", len(allDominanceIDs))
				trackFatalErr(&r, "dominance flags", err)
			} else {
				r.DominanceFlagsComputed = len(allDominanceIDs)
				slog.InfoContext(ctx, "post-sync: dominance flags calculés",
					"gamertag", e.gamertag, "count", r.DominanceFlagsComputed)
			}
		}
	}

	// 2 / 2.5 / 2.6 — bloc LUSR (v1 + v2 shadow + sentinelle dual-row), extrait
	// pour réduire la taille de runPostSyncPipeline (cf. revue D7-3). Self-gate
	// sur la capability title.CapLUSR.
	e.runSkillRatingSteps(ctx, playerDB, sharedDB, &r)

	// 3. Career rank — DÉCOUPLÉ du post-sync depuis 2026-05-14.
	// Le flow XP + Spartan ID est désormais géré par service.CareerLiveService
	// (throttle 5 min / 6 h + fallback DB per-field), appelé depuis HomeService
	// à chaque chargement de /pages/home. Voir .ai/thought_log.md.
	// domain.PostSyncResult.CareerSynced reste dans le struct (compat e2e tests)
	// mais n'est plus jamais positionné à true ici.

	// 3.1 CSR snapshots (best-effort, skip silencieux si csrSeasonID vide).
	// Maintenu dans le post-sync : le CSR ne bouge que sur fin de match ranked,
	// donc le déclencheur "nouveau match" reste pertinent.
	// Les CSRs sont capturés pour alimenter playlists_catalog en parallèle
	// à l'étape 4 (errgroup) — transparent, 0 latence supplémentaire.
	var pendingCSRs []PlayerPlaylistCSR
	if csrs, csrErr := e.runCSRSnapshotSync(ctx, playerDB, client); csrErr != nil {
		trackFatalErr(&r, "CSR snapshots", csrErr)
	} else {
		pendingCSRs = csrs
	}

	// 3.5 Friends recompute is_with_friends (best-effort).
	// Avant l'étape 4 (aggregates) pour éviter un double-refresh : on passe
	// refreshAggregates=false, le refresh natif de l'engine couvre les UPDATEs.
	// Skip silencieux si pas de loader (legacy) ou liste vide.
	if e.friendsLoader != nil {
		if friends, ferr := e.friendsLoader(); ferr != nil {
			slog.WarnContext(ctx, "post-sync: friends loader échoué", "gamertag", e.gamertag, "err", ferr)
			trackFatalErr(&r, "friends loader", ferr)
		} else if len(friends) > 0 {
			slog.DebugContext(ctx, "post-sync: friends recompute", "gamertag", e.gamertag, "friends_count", len(friends))
			fres, err := RecomputeIsWithFriendsCore(ctx, playerDB, sharedDB, e.xuid, friends, false)
			if err != nil {
				slog.WarnContext(ctx, "post-sync: friends recompute échoué", "gamertag", e.gamertag, "err", err)
				trackFatalErr(&r, "friends recompute", err)
			} else if fres.MatchesPromoted > 0 {
				r.MatchesPromotedFriends = fres.MatchesPromoted
				slog.InfoContext(ctx, "post-sync: matchs reclasses comme escouade-amis",
					"gamertag", e.gamertag,
					"promoted", fres.MatchesPromoted,
				)
			}
		}
	}

	// 4. Aggregates (vues matérialisées player) + catalog seeding, en parallèle
	// via errgroup — extrait pour réduire runPostSyncPipeline (cf. revue D7-3).
	e.runAggregatesRefresh(ctx, playerDB, pendingCSRs, &r)

	// 4.5 Media scan post-sync — indexe les captures présentes dans le dossier
	// du joueur et les associe aux matchs fraîchement insérés (best-effort).
	// ForceRescan=false : seuls les nouveaux fichiers sont traités → coût nul
	// si aucune nouvelle capture depuis la dernière sync.
	if e.mediaHook != nil {
		slog.DebugContext(ctx, "post-sync: scan médias", "gamertag", e.gamertag)
		e.mediaHook(ctx)
	}

	// 5. Achievements Xbox (fire-and-forget, non bloquant en cas d'erreur token)
	r.AchievementsSynced = e.runAchievementsSync(ctx, playerDB)

	return r
}

// runAggregatesRefresh rafraîchit les vues matérialisées player (refreshAggregates)
// et seede le catalog depuis les CSR en attente, en parallèle via errgroup (DBs
// différentes → pas de conflit ni de race ; ViewsRefreshed est atomic). Best-effort :
// les erreurs sont capturées par goroutine puis agrégées post-Wait via trackFatalErr.
func (e *SyncEngine) runAggregatesRefresh(ctx context.Context, playerDB *sql.DB, pendingCSRs []PlayerPlaylistCSR, r *domain.PostSyncResult) {
	slog.DebugContext(ctx, "post-sync: refresh aggregates+views (parallel)", "gamertag", e.gamertag)
	var viewsRefreshed atomic.Int32
	// Capture des erreurs en variables locales (une par goroutine) pour les
	// agréger post-Wait sans race.
	var aggregatesErr, sharedViewsErr error
	egRefresh := &errgroup.Group{}
	egRefresh.Go(func() error {
		n, err := refreshAggregates(ctx, playerDB)
		if err != nil {
			slog.WarnContext(ctx, "post-sync: aggregates échoué", "gamertag", e.gamertag, "err", err)
			aggregatesErr = err
			return nil //nolint:nilerr // best-effort, ne propage pas (cohérent avec ancien comportement)
		}
		viewsRefreshed.Add(int32(n))
		return nil
	})
	// Fix bug 2026-05-27 : refreshSharedViews retiré du post-sync runtime. Les vues
	// shared (v_gamertag_lookup, v_match_full) sont créées au boot via
	// EnsureSharedSchema(). En runtime, le sharedDB writer peut avoir un ATTACH RO
	// implicite (auto-attach DuckDB) qui ferait échouer le CREATE OR REPLACE VIEW.
	// Les VIEW survivent au close/reopen — inutile de les recréer après chaque sync.
	_ = sharedViewsErr // conservé pour minimiser le diff downstream
	// Catalog seeding en parallèle des aggregates — transparent, DB différente.
	if len(pendingCSRs) > 0 {
		csrsToSeed := pendingCSRs
		egRefresh.Go(func() error {
			e.seedCatalogFromCSRs(ctx, csrsToSeed)
			return nil
		})
	}
	_ = egRefresh.Wait()
	trackFatalErr(r, "aggregates", aggregatesErr)
	trackFatalErr(r, "shared views", sharedViewsErr)
	r.ViewsRefreshed = int(viewsRefreshed.Load())
}

// runScoringSteps exécute le bloc de scoring enrichment du post-sync sur la
// player DB : had_bot_teammate, sessions, performance scores, engagement scores,
// engagement coefficients, assists model. Best-effort : chaque sous-étape logue
// + trackFatalErr sur erreur sans interrompre le reste (toutes idempotentes).
func (e *SyncEngine) runScoringSteps(ctx context.Context, playerDB, sharedDB *sql.DB, r *domain.PostSyncResult) {
	// -0.3 had_bot_teammate — dérivé des participants (cheap SQL, pas d'API).
	// Idempotent : skip les rows déjà à TRUE.
	if n, err := computeAndPersistHadBotTeammate(ctx, playerDB, sharedDB, e.xuid); err != nil {
		slog.WarnContext(ctx, "post-sync: had_bot_teammate échoué", "gamertag", e.gamertag, "err", err)
		trackFatalErr(r, "had_bot_teammate", err)
	} else if n > 0 {
		slog.InfoContext(ctx, "post-sync: had_bot_teammate", "gamertag", e.gamertag, "rows_updated", n)
	}

	// 0. Session assignments — auto-recalc session_id pour les nouveaux matchs.
	// Best-effort : un échec ne bloque pas le pipeline. Les amis sont
	// résolus depuis le friendsLoader (settings.FriendGamertags). Sans loader
	// (legacy), on retombe en TeamChangeMode=teammates.
	{
		var friends []string
		if e.friendsLoader != nil {
			if fs, ferr := e.friendsLoader(); ferr == nil {
				friends = fs
			}
		}
		opts := analysis.DefaultSessionOptions()
		if n, err := recalculateSessionsInline(ctx, playerDB, sharedDB, e.xuid, opts, friends); err != nil {
			slog.WarnContext(ctx, "post-sync: sessions échoué", "gamertag", e.gamertag, "err", err)
			trackFatalErr(r, "sessions", err)
		} else if n > 0 {
			r.SessionsAssigned = n
			slog.DebugContext(ctx, "post-sync: sessions recalculées", "gamertag", e.gamertag, "count", n)
		}
	}

	// 1. Performance scores
	slog.DebugContext(ctx, "post-sync: calcul perf scores", "gamertag", e.gamertag)
	if n, err := batchComputePerformanceScores(ctx, playerDB, sharedDB, e.xuid, nil, false); err != nil {
		slog.WarnContext(ctx, "post-sync: perf scores échoué", "gamertag", e.gamertag, "err", err)
		trackFatalErr(r, "perf scores", err)
	} else {
		r.PerfScoresComputed = n
		slog.DebugContext(ctx, "post-sync: perf scores calculés", "gamertag", e.gamertag, "count", n)
	}

	// 1.5 Engagement scores (Phase 3 plan engagement) — best-effort,
	// skip silencieux si migration Phase 2 non appliquee.
	slog.DebugContext(ctx, "post-sync: calcul engagement scores", "gamertag", e.gamertag)
	if n, err := batchComputeEngagementScores(ctx, playerDB, sharedDB, e.xuid, false); err != nil {
		slog.WarnContext(ctx, "post-sync: engagement scores échoué", "gamertag", e.gamertag, "err", err)
		trackFatalErr(r, "engagement scores", err)
	} else if n > 0 {
		r.EngagementScoresComputed = n
		slog.DebugContext(ctx, "post-sync: engagement scores calculés", "gamertag", e.gamertag, "count", n)
	}

	// 1.5.b Recompute des engagement coefficients depuis la mediane glissante
	// des paces persistees ci-dessus. Sans ce recompute, coef_team_share reste
	// a 1.0 (cold-start) → pace_attendu = pace_team → courbes superposees a
	// l'ecran (cf. .ai/V7/PLAN_ENGAGEMENT_IMPLEMENTATION.md §4.4).
	if n, err := batchRecomputeCoefficients(ctx, playerDB, e.xuid); err != nil {
		slog.WarnContext(ctx, "post-sync: engagement coefs échoué", "gamertag", e.gamertag, "err", err)
		trackFatalErr(r, "engagement coefs", err)
	} else if n > 0 {
		r.EngagementCoefsUpdated = n
		slog.DebugContext(ctx, "post-sync: engagement coefs mis à jour", "gamertag", e.gamertag, "count", n)
	}

	// 1.52 Assists model — OLS per-mode, skip silencieux si migration absente.
	// force=false : ne recalcule que si player_assists_model est vide (cold-start).
	// Un nouveau sync peut amener des données → on recalcule si table vide.
	if n, err := batchComputePlayerAssistsModel(ctx, playerDB, sharedDB, e.xuid, false); err != nil {
		slog.WarnContext(ctx, "post-sync: assists model échoué", "gamertag", e.gamertag, "err", err)
		trackFatalErr(r, "assists model", err)
	} else if n > 0 {
		slog.InfoContext(ctx, "post-sync: assists model calculé", "gamertag", e.gamertag, "n_modes", n)
	}
}

// runSkillRatingSteps exécute le bloc LUSR du post-sync : v1 (TrueSkill 2),
// v2 shadow, et la sentinelle dual-row. Tout est gardé par la capability
// title.CapLUSR (gate sur e.titleSlug, autoritatif côté engine) — un titre sans
// CapLUSR saute proprement. Best-effort : chaque sous-étape logue + trackFatalErr
// sur erreur sans interrompre le reste (idempotent).
func (e *SyncEngine) runSkillRatingSteps(ctx context.Context, playerDB, sharedDB *sql.DB, r *domain.PostSyncResult) {
	lusrEnabled := slugHasLUSR(e.titleSlug)
	if !lusrEnabled {
		slog.DebugContext(ctx, "post-sync: LUSR skippé — capability absente",
			"gamertag", e.gamertag, "title_slug", e.titleSlug)
	}

	// 2. LUSR v1 (TrueSkill 2 closed-form, formule composite).
	// SKIPPÉ si LEVELUP_LUSR_CANONICAL=LUSR_V2 — alors v2 est canonical et
	// écrit directement dans rating_type='LUSR' via Stratégie C.
	if lusrEnabled && !IsLUSRV2Canonical() {
		slog.DebugContext(ctx, "post-sync: calcul LUSR v1", "gamertag", e.gamertag)
		medalMap := e.loadMedalExploitMapBestEffort(ctx, sharedDB)
		if n, err := batchComputeLUSR(ctx, playerDB, sharedDB, e.xuid, medalMap, false); err != nil {
			slog.WarnContext(ctx, "post-sync: LUSR v1 échoué", "gamertag", e.gamertag, "err", err)
			trackFatalErr(r, "LUSR", err)
		} else {
			r.LUSRUpdated = n
			slog.DebugContext(ctx, "post-sync: LUSR v1 mis à jour", "gamertag", e.gamertag, "count", n)
		}
	} else if lusrEnabled {
		slog.DebugContext(ctx, "post-sync: LUSR v1 skippé (canonical=LUSR_V2)", "gamertag", e.gamertag)
	}

	// 2.5 LUSR v2 — shadow mode (LEVELUP_LUSR_V2_ENABLED=1). Calcule en parallèle
	// du v1 et écrit dans player_skill_state_v2.
	//
	// Si LEVELUP_LUSR_CANONICAL=LUSR_V2, écrit AUSSI dans match_skill_rank
	// (rating_type='LUSR' slot historique) via Stratégie C — l'UI voit alors
	// le v2 sans modif des readers. Cf. ADR 0024. RunLUSRV2Shadow self-gate la
	// capability ; le garde ici évite juste l'appel inutile.
	if lusrEnabled && IsLUSRV2Enabled() {
		if n, err := RunLUSRV2Shadow(ctx, playerDB, sharedDB, e.xuid); err != nil {
			slog.WarnContext(ctx, "post-sync: LUSR v2 shadow échoué",
				"gamertag", e.gamertag, "err", err)
		} else if n > 0 {
			slog.InfoContext(ctx, "post-sync: LUSR v2 shadow OK",
				"gamertag", e.gamertag, "processed", n, "canonical", IsLUSRV2Canonical())
		}
	}

	// 2.6 Sentinelle dual-row (Sprint 2.C) — SEULEMENT en mode canonical (sinon la
	// table dual-row LUSR_V2 n'est pas censée exister). Détecte l'invariant
	// Stratégie C cassé (match avec LUSR_V2 sans LUSR). Read-only, idempotente,
	// timeout 30s pour ne jamais bloquer le post-sync. Toute incohérence →
	// slog.ErrorContext (auto-routé logs/sync.log) ; pas de notif externe.
	if lusrEnabled && IsLUSRV2Canonical() && playerDB != nil {
		sentCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		report, serr := RunDualRowSentinel(sentCtx, playerDB)
		cancel()
		switch {
		case serr != nil:
			slog.ErrorContext(ctx, "post-sync: sentinelle dual-row échouée",
				"err", serr, "gamertag", e.gamertag)
		case report.OnlyLUSRV2 > 0:
			slog.ErrorContext(ctx, "post-sync: sentinelle dual-row a détecté des incohérences",
				"gamertag", e.gamertag,
				"only_lusr_v2", report.OnlyLUSRV2,
				"sample", report.SampleInconsistent,
			)
		}
	}
}

// runCSRSnapshotSync récupère les classements CSR du joueur pour la saison courante
// et les persiste dans player_csr_snapshots. Best-effort : skippé si csrSeasonID vide
// (avec WARN explicite pour rendre cette régression de config visible aux ops).
// runCSRSnapshotSync retourne nil sur succès ou skip de config, ou l'erreur
// brute de syncPlayerCSRs en cas d'échec runtime. Le caller peut utiliser
// trackFatalErr pour propager au SyncResult si IsInvalidatedError.
// runCSRSnapshotSync récupère + persiste les CSR snapshots du joueur.
// Retourne la slice CSR pour que le caller puisse alimenter playlists_catalog
// en parallèle (cf. errgroup step 4 dans runPostSyncPipeline).
func (e *SyncEngine) runCSRSnapshotSync(ctx context.Context, playerDB *sql.DB, client HaloClient) ([]PlayerPlaylistCSR, error) {
	if strings.TrimSpace(e.csrSeasonID) == "" {
		// Visibilité explicite : sans cette config, player_csr_snapshots reste vide
		// éternellement et la home affiche "Aucun classement". Bug racine difficile
		// à diagnostiquer côté UI ; un WARN rend le silence visible aux ops.
		slog.WarnContext(ctx,
			"post-sync: CSR snapshot sync SKIPPED — csr_season_id non configuré "+
				"(ajouter le champ \"csr_season_id\" dans app_settings.json, ex. \"CsrSeason13-1\", "+
				"ou définir l'env var LEVELUP_CSR_SEASON_ID)",
			"gamertag", e.gamertag,
		)
		return nil, nil
	}
	slog.DebugContext(ctx, "post-sync: sync CSR snapshots", "gamertag", e.gamertag, "season", e.csrSeasonID)
	csrs, err := syncPlayerCSRs(ctx, client, playerDB, e.xuid, e.csrSeasonID)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: CSR snapshots échoué", "gamertag", e.gamertag, "err", err)
		return nil, err
	}
	slog.DebugContext(ctx, "post-sync: CSR snapshots sauvegardés", "gamertag", e.gamertag, "count", len(csrs))
	return csrs, nil
}

// runAchievementsSync récupère les achievements Xbox pour le joueur et les persiste.
// Retourne true si la sync a réussi, false en cas d'erreur (non bloquante).
// Nécessite e.provider non nil ; skippé silencieusement sinon.
func (e *SyncEngine) runAchievementsSync(ctx context.Context, playerDB *sql.DB) bool {
	if e.provider == nil {
		slog.DebugContext(ctx, "achievements: provider nil — sync ignorée", "gamertag", e.gamertag)
		return false
	}

	// Résoudre l'access_token depuis sync_meta DuckDB.
	accessToken, err := resolveAccessTokenFromDB(ctx, playerDB, e.gamertag, e.provider)
	if err != nil {
		slog.WarnContext(ctx, "achievements: échec résolution access_token",
			"gamertag", e.gamertag, "err", err)
		return false
	}
	if accessToken == "" {
		slog.InfoContext(ctx, "achievements: aucun access_token disponible — sync ignorée",
			"gamertag", e.gamertag)
		return false
	}

	// Obtenir un XSTS token pour Xbox Live.
	xstsResult, err := auth.AcquireXSTSForRTA(ctx, accessToken)
	if err != nil {
		slog.WarnContext(ctx, "achievements: échec acquisition XSTS",
			"gamertag", e.gamertag, "err", err)
		return false
	}

	// Phase 2 du PLAN_FIX_SYNC_RELIABILITY_2026-05-24 : cache duckdbpkg pour
	// l'ecriture RW de metadata (achievements upsert). Aligne le DSN avec
	// engine.go:249 et citations_backfill.go via OpenReadWriteShared
	// (cle "rw:"+path partagee).
	//
	// Fix race 2026-05-27 : `xbox_achievement_definitions` est globale (1 row
	// par achievement_id, 144 communs aux 4 joueurs). 2 SyncEngine en
	// parallèle (cf. Coordinator parallel_slots:2 ou scheduler errgroup) qui
	// faisaient un upsert (INSERT-OR-UPDATE via conflict clause) sur la même
	// row déclenchaient "TransactionContext Error: Conflict on update!" côté
	// DuckDB (vu dans logs/sync.log 23:58:11 Madina97294 et précédents).
	// Sérialisation applicative via dblease.AcquireWriterCtx(KindMetadata)
	// — bloque le 2ème caller jusqu'au Release du 1er, sans contention DuckDB.
	metadataLease, err := dblease.AcquireWriterCtx(ctx, nil, e.metadataDBPath, dblease.KindMetadata)
	if err != nil {
		slog.WarnContext(ctx, "achievements: acquisition lease metadata échouée",
			"gamertag", e.gamertag, "err", err)
		return false
	}
	defer metadataLease.Release()

	metadataHandle, err := duckdbpkg.OpenReadWriteShared(e.metadataDBPath)
	if err != nil {
		slog.WarnContext(ctx, "achievements: ouverture metadata DB échouée",
			"gamertag", e.gamertag, "err", err)
		return false
	}
	defer metadataHandle.Close()
	metadataDB := metadataHandle.SQLDb()

	client := NewXboxHTTPClient(xstsResult, titlePkg.XboxTitleIDFor(e.titleSlug))
	if err := SyncAchievements(ctx, client, e.resolver, metadataDB, playerDB, e.xuid, e.titleSlug); err != nil {
		slog.WarnContext(ctx, "achievements: sync échouée",
			"gamertag", e.gamertag, "err", err)
		return false
	}

	slog.InfoContext(ctx, "achievements: sync terminée avec succès", "gamertag", e.gamertag)
	return true
}

// RunAchievementsOnly synchronise uniquement les achievements Xbox du joueur,
// indépendamment du sync des matchs. Utilisé par le CLI sync-achievements pour
// le backfill admin one-shot. Best-effort : retourne false sur erreur (logguée).
//
// Acquiert le dblease sur la player DB pour éviter les collisions avec un sync
// concurrent. Le provider doit être non nil ; sinon retourne false silencieusement.
func (e *SyncEngine) RunAchievementsOnly(ctx context.Context) bool {
	if e.provider == nil {
		slog.WarnContext(ctx, "achievements: provider nil — sync ignorée",
			"gamertag", e.gamertag)
		return false
	}

	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		slog.ErrorContext(ctx, "achievements: lease player DB échoué",
			"gamertag", e.gamertag, "err", err)
		return false
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		slog.ErrorContext(ctx, "achievements: ouverture player DB échouée",
			"gamertag", e.gamertag, "err", err)
		return false
	}
	defer playerHandle.Close() //nolint:errcheck

	return e.runAchievementsSync(ctx, playerHandle.SQLDb())
}

// selectMatchesMissingDominanceFlags retourne les match_ids dont le dominance_flag
// est NULL (jamais calculé). La valeur 0 = "aucune dominance détectée" est un
// résultat valide et non re-traité.
func selectMatchesMissingDominanceFlags(ctx context.Context, playerDB *sql.DB) ([]string, error) {
	rows, err := playerDB.QueryContext(ctx,
		`SELECT match_id FROM player_match_enrichment WHERE dominance_flag IS NULL`)
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

// mergeUniqMatchIDs fusionne deux slices de match_ids en dédupliquant, a en tête.
func mergeUniqMatchIDs(a, b []string) []string {
	if len(b) == 0 {
		return a
	}
	seen := make(map[string]struct{}, len(a)+len(b))
	result := make([]string, 0, len(a)+len(b))
	for _, id := range a {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	for _, id := range b {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	return result
}

// seedCatalogFromCSRs ouvre metadata.duckdb en RW et appelle seedPlaylistsCatalog.
// Best-effort : log WARN si metadata inaccessible, n'interrompt jamais le pipeline.
func (e *SyncEngine) seedCatalogFromCSRs(ctx context.Context, csrs []PlayerPlaylistCSR) {
	if e.metadataDBPath == "" || len(csrs) == 0 {
		return
	}
	mh, err := duckdbpkg.OpenReadWriteShared(e.metadataDBPath)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: catalog seed désactivé (metadata inaccessible)",
			"gamertag", e.gamertag, "err", err)
		return
	}
	defer mh.Close()
	seedPlaylistsCatalog(ctx, mh.SQLDb(), csrs, e.titleSlug)
}

// resolveAccessTokenFromDB lit le cache MSAL et le refresh token depuis sync_meta (DB déjà ouverte),
// puis tente TrySilentRefresh ou TryOAuthRefresh selon ce qui est disponible.
// Retourne ("", nil) si aucun token n'est disponible (non fatal).
//
//nolint:unparam // contrat documenté : second retour non-nil est réservé aux futures erreurs fatales (DB)
func resolveAccessTokenFromDB(
	ctx context.Context,
	playerDB *sql.DB,
	gamertag string,
	provider auth.TokenProvider,
) (string, error) {
	var cacheJSON, refreshToken string
	if err := playerDB.QueryRowContext(ctx,
		"SELECT value FROM sync_meta WHERE key = 'msal_token_cache'").Scan(&cacheJSON); err != nil {
		slog.DebugContext(ctx, "achievements: msal_token_cache absent", "gamertag", gamertag)
	}
	if err := playerDB.QueryRowContext(ctx,
		"SELECT value FROM sync_meta WHERE key = 'oauth_refresh_token'").Scan(&refreshToken); err != nil {
		slog.DebugContext(ctx, "achievements: oauth_refresh_token absent", "gamertag", gamertag)
	}

	// Fallback env var SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG>
	if refreshToken == "" && gamertag != "" {
		key := strings.ToUpper(strings.NewReplacer(" ", "_", "-", "_", ".", "_").Replace(gamertag))
		if v := os.Getenv("SPNKR_OAUTH_REFRESH_TOKEN_" + key); v != "" {
			refreshToken = v
		}
	}

	if cacheJSON != "" {
		token, err := provider.TrySilentRefresh(ctx, cacheJSON)
		if err == nil && token != "" {
			return token, nil
		}
	}

	if refreshToken != "" {
		token, err := provider.TryOAuthRefresh(ctx, refreshToken)
		if err == nil && token != "" {
			return token, nil
		}
	}

	return "", nil
}
