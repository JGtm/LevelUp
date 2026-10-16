// Package sync — engine_postsync.go : pipeline post-sync + sync achievements.
//
// Extrait de engine.go (refactor 2026-05-21). Regroupe :
//   - runConditionalPostSync : branche post-sync selon matchs insérés ou heal.
//   - runPostSyncPipeline : pipeline complet, dans l'ordre : lignes d'enrichissement,
//     scoring, événements (1.54), PSA (1.56), escouade (3.5), domination (1.7), LUSR,
//     CSR (3.1), agrégats (4), vidage des caches de lecture, puis les étapes film lentes
//     (source du kill 1.57, rejeu 1.58), citations (1.6), médias, succès, préparation du
//     match. Citations et domination sont des étapes primaires, pas des heals : elles
//     tournent pour chaque nouveau match et comblent le backlog.
//   - runCSRSnapshotSync : CSR snapshots best-effort si csrSeasonID renseigné.
//   - runAchievementsSync + RunAchievementsOnly : sync Xbox achievements via
//     TokenProvider (resolveAchievementsAccessToken → XSTS → SyncAchievements).
//   - hasMatchesNeedingScoreRefresh : heuristique heal-only path.
//   - resolveAchievementsAccessToken : MultiUserTokenStore seul (ADR 0023).
//
// Voir engine.go (struct SyncEngine + run()) pour le contexte.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/logging"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync/snapshot"
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

// withSharedRead ouvre un segment de LECTURE shared COURT pour une étape du
// post-sync, et le relâche à la sortie de fn. Échec d'acquisition → WARN +
// trackFatalErr, l'étape est skippée (best-effort ; en mode pinned l'acquisition
// n'échoue jamais).
//
// Helper CANONIQUE : toute étape qui a besoin de lire shared passe par ici — y
// compris les étapes extraites (engine_postsync_films.go). Ne pas ré-écrire ce
// bloc ailleurs : le release DOIT précéder tout burst Write (garde anti-deadlock
// de SharedAccess.Write).
func (e *SyncEngine) withSharedRead(
	ctx context.Context,
	shared *SharedAccess,
	r *domain.PostSyncResult,
	step string,
	fn func(sharedDB *sql.DB),
) {
	roDB, release, aerr := shared.Read(ctx)
	if aerr != nil {
		slog.WarnContext(ctx, "post-sync: lecture shared indisponible — étape skippée",
			"gamertag", e.gamertag, "step", step, "err", aerr)
		trackFatalErr(r, step+" shared read", aerr)
		return
	}
	defer release()
	fn(roDB)
}

// runConditionalPostSync exécute le pipeline complet si des matchs ont été insérés,
// sinon rafraîchit au moins la carrière pour mettre à jour le snapshot joueur.
func (e *SyncEngine) runConditionalPostSync(
	ctx context.Context,
	playerDB *sql.DB,
	shared *SharedAccess,
	client HaloClient,
	matchesInserted int,
	insertedIDs []string,
) domain.PostSyncResult {
	if matchesInserted > 0 {
		slog.InfoContext(ctx, "sync: lancement pipeline post-sync", "gamertag", e.gamertag)
		return e.runPostSyncPipeline(ctx, playerDB, shared, client, insertedIDs)
	}

	// Pas de nouveaux matchs : heal skill décommissionné (2026-06-01, cf.
	// runPostSyncPipeline). On se contente de détecter les matchs avec scores
	// manquants (engagement/perf NULL) pour relancer le recalcul post-sync —
	// pur calcul dérivé, aucune écriture API/shared concurrente. Les pré-checks
	// sont des LECTURES : shared.Read (RO en mode burst, handle pinné sinon),
	// relâché AVANT le pipeline (garde anti-deadlock des bursts Write).
	needsScoreRefresh := false
	backlog := false
	if roDB, roRelease, roErr := shared.Read(ctx); roErr == nil {
		needsScoreRefresh, _ = hasMatchesNeedingScoreRefresh(ctx, playerDB, roDB, e.xuid)
		// Convergence : même sans nouveau match, le sync n'a pas "fini" tant qu'il
		// reste des matchs events/weapons incomplets (ex. matchs joués en escouade
		// insérés par le watcher d'un teammate). Idempotent + borné.
		backlog = needsScoreRefresh || hasConvergenceBacklog(ctx, playerDB, roDB, e.xuid)
		roRelease()
	} else {
		slog.WarnContext(ctx, "sync: pré-checks post-sync — lecture shared indisponible, chemin léger",
			"gamertag", e.gamertag, "err", roErr)
	}
	if backlog {
		slog.InfoContext(ctx, "sync: aucun match inséré — backlog scores/convergence → post-sync complet",
			"gamertag", e.gamertag, "needs_score_refresh", needsScoreRefresh)
		return e.runPostSyncPipeline(ctx, playerDB, shared, client, nil)
	}
	slog.DebugContext(ctx, "sync: aucun match inséré — refresh CSR + achievements seul (carrière live découplé)", "gamertag", e.gamertag)
	// Carrière (XP + Spartan ID) retirée du post-sync : service.CareerLiveService
	// la rafraîchit live à chaque chargement de /pages/home.
	res := domain.PostSyncResult{}
	lightStart := time.Now()
	if csrs, err := e.runCSRSnapshotSync(ctx, playerDB, client); err != nil {
		trackFatalErr(&res, "CSR snapshots", err)
	} else if len(csrs) > 0 {
		e.seedCatalogFromCSRs(ctx, csrs)
	}
	res.AchievementsSynced = e.runAchievementsSync(ctx, playerDB) == achievementsSynced
	res.DurationMs = time.Since(lightStart).Milliseconds()
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
		SELECT COUNT(*) FROM player_match_enrichment_latest
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
	playerDB *sql.DB,
	shared *SharedAccess,
	client HaloClient,
	insertedIDs []string,
) (r domain.PostSyncResult) {
	defer duckdbpkg.InvalidatePlayerReadCaches(ctx, e.xuid, e.titleSlug) // perf L5b : fin du post-sync = lectures joueur cachées périmées (posé en premier, exécuté en dernier)
	defer e.flushHomeCache(ctx)                                          // idem pour le cache Home (rappel du serveur, nil hors serveur)
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

	// Chronométrage P4 (dashboard monitoring) : un lap par bloc séquentiel.
	// finish() en defer → DurationMs posé même sur retour partiel post-panic
	// (r est un named return).
	clock := newPostSyncClock(&r, e.titleSlug)
	defer clock.finish()

	// Étape 1 contention (incrément 3) : plus AUCUNE matérialisation upfront du
	// writer — chaque étape ouvre le segment dont elle a besoin : Read RO court
	// (lectures shared, majorité des étapes) ou burst Write court (les 4 seules
	// écritures shared : intensity/events/weapons/psa). En mode pinned (V1),
	// Read/Write retournent le handle déjà tenu → comportement identique.
	//
	// Probe RC-A : en pinned uniquement (parité historique — le pipeline CONTINUE
	// en dégradé sur read-only). En burst, SharedAccess.Write sonde par burst et
	// échoue proprement étape par étape.
	if shared.isPinned {
		if err := assertSharedWritable(ctx, shared.pinned); err != nil {
			slog.ErrorContext(ctx, "post-sync: shared attaché en read-only (RC-A) — écritures shared vont échouer",
				"gamertag", e.gamertag, "err", err)
			trackFatalErr(&r, "shared read-only", err)
		}
	}
	// withSharedRead : segment de LECTURE court par étape (délègue au helper
	// canonique e.withSharedRead — même sémantique, un seul point de vérité
	// partagé avec les étapes extraites, cf. engine_postsync_films.go).
	withSharedRead := func(step string, fn func(sharedDB *sql.DB)) {
		e.withSharedRead(ctx, shared, &r, step, fn)
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
	enrichRows := 0
	withSharedRead("enrichment_rows", func(sharedDB *sql.DB) {
		if n, err := ensurePlayerEnrichmentRows(ctx, playerDB, sharedDB, e.xuid); err != nil {
			slog.WarnContext(ctx, "post-sync: ensure enrichment rows échoué",
				"gamertag", e.gamertag, "err", err)
			// Best-effort : on continue le pipeline même si ça échoue (les UPDATE
			// resteront no-op pour les matchs sans row, mais le reste du pipeline
			// peut quand même tourner pour les matchs qui ont déjà leurs rows).
		} else if n > 0 {
			enrichRows = n
			slog.InfoContext(ctx, "post-sync: enrichment rows créées",
				"gamertag", e.gamertag, "rows_created", n)
		}
	})
	clock.lap("enrichment_rows", enrichRows)

	// HEALS DÉCOMMISSIONNÉS (2026-06-01) — stats / skill / events / weapon.
	//
	// Le film highlights est disponible immédiatement ~99% du temps : le sync
	// PRIMAIRE (engine_fetch.go) récupère et persiste skill (GetMatchSkill +
	// MergeSkillIntoParticipants), participants (ExtractParticipants) et events
	// (GetHighlightEventsChunk → persistCombatCompletion) au 1er passage,
	// via le writer RW orchestré (persist.BatchBuilder → *Persister). Les heals
	// post-sync ne faisaient que dupliquer ce chemin sur une fenêtre non
	// maîtrisée — et `healSkillForMissingMatches` réécrivait match_participants
	// en `ON CONFLICT DO UPDATE`, ce qui corrompait l'ART (incident 2026-06-01).
	// Décision produit : aucun heal automatique. Le 1% résiduel (film absent au
	// 1er sync) se récupère via backfill CLI explicite et contrôlé.
	// Cf. ADR 0019 + thought_log 2026-06-01.

	// ORDRE DES ÉTAPES — ce que l'utilisateur lit d'abord passe avant ce qui télécharge un
	// film. Les étapes rapides qui portent la tuile du match (escouade, LUSR, CSR,
	// agrégats) précèdent la source du kill (1.57) et le rejeu (1.58), qui lisent des films
	// pendant des minutes. Dépendances qui fixent l'ordre :
	//   - 1.54 (événements) avant la domination (1.7) et le reste : ils lisent highlight_events ;
	//   - 1.56 (PSA) avant l'escouade et les citations ;
	//   - 1.57 avant le rejeu (1.58, qui retrouve le film sur disque), avant les citations
	//     (les citations par arme lisent ses lignes, calcul unique) et avant la préparation ;
	//   - le LUSR lit la passe crédit-seul de match_kill_events (écrite au premier écrit ou par
	//     1.54), pas la passe film : il n'attend pas 1.57.

	// -0.3 had_bot, 0 sessions, 1 perf, 1.5 engagement, 1.5.b coefs, 1.52 assists —
	// bloc de scoring enrichment (player DB), extrait pour réduire la taille de
	// runPostSyncPipeline (cf. revue D7-3).
	e.runScoringSteps(ctx, playerDB, shared, &r)
	clock.lap("scoring", r.EngagementScoresComputed)

	// 1.54 convergence events — étape « film » du post-sync, extraite dans convergence.go
	// (split COLLECT/FLUSH : le téléchargement du film ne tient plus le writer RW).
	//
	// L'ÉTAPE 1.55 (weapon kills) A ÉTÉ SUPPRIMÉE le 2026-09-01 avec son producteur :
	// elle corrélait les tirs de l'attaquant avec l'instant du kill, une méthode que
	// la source de dégât (1.57) remplace intégralement. Le numéro n'est pas réattribué.
	films := postSyncFilmSteps{engine: e, playerDB: playerDB, shared: shared, client: client, result: &r}
	films.runEventsConvergence(ctx)
	clock.lap("convergence_events", r.ConvergedEvents)

	// 1.56 PSA — convergence des personal_score_awards.
	e.runPSAConvergence(ctx, playerDB, shared, client, &r)
	clock.lap("convergence_psa", r.ConvergedPSA)

	// Pas de réécriture des noms du registre ici : map_name/pair_name/
	// playlist_name/game_variant_name sont résolus au sync PRIMAIRE via
	// EnrichRegistryFromMetadata. Les noms restés en identifiant convergent par
	// sync.BackfillRegistryNames, après le balayage périodique des noms d'assets
	// (serveur) ou par `levelup backfill-registry-names` (CLI).
	e.runCatalogRefresh(ctx, shared, &r, insertedIDs)

	// 3.5 Friends — recalcul convergent de is_with_friends (le premier écrit l'a déjà posé).
	e.runFriendsRecompute(ctx, playerDB, shared, &r)
	clock.lap("friends", int(r.MatchesPromotedFriends))

	// 1.7 Dominance flags (après 1.54 : la courbe de score vient de highlight_events).
	e.runDominanceFlags(ctx, playerDB, shared, &r, insertedIDs)
	clock.lap("dominance", r.DominanceFlagsComputed)

	// 2 / 2.5 / 2.6 — bloc LUSR (v1 + v2 shadow + sentinelle dual-row), extrait
	// pour réduire la taille de runPostSyncPipeline (cf. revue D7-3). Self-gate
	// sur la capability title.CapLUSR.
	e.runSkillRatingSteps(ctx, playerDB, shared, &r)
	clock.lap("skill_rating", r.LUSRUpdated)

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
	clock.lap("csr_snapshots", len(pendingCSRs))

	// 4. Aggregates (vues matérialisées player) + catalog seeding, en parallèle
	// via errgroup — extrait pour réduire runPostSyncPipeline (cf. revue D7-3).
	e.runAggregatesRefresh(ctx, playerDB, pendingCSRs, &r)
	clock.lap("aggregates", r.ViewsRefreshed)

	// Les lignes que les pages lisent sont à jour : vider les caches de lecture MAINTENANT,
	// avant la lecture des films (le `defer` de tête vide encore à la sortie).
	e.flushReadCaches(ctx)

	// 1.57 source du kill : décodage du kill-feed — la SEULE origine de `assist_known`.
	// AVANT 1.58, qui retrouve alors le film sur disque (cf. killcollector/postsync.go).
	clock.lap("kill_source", films.runKillSource(ctx, insertedIDs))
	// 1.58 artefacts de rejeu 2D (fil de l'eau LOCAL, lot 6 v7.5) : pont disque
	// filmcache + construction replaybuild des matchs insérés, fenêtre
	// replay_retention_months relue à chaque cycle. No-op si le hook n'est pas
	// installé (VPS web : jamais). Cf. convergence.go.
	films.runReplayArtifacts(ctx, insertedIDs)
	clock.lap("replay_artifacts", 0)

	// 1.6 Citations (après les films, cf. runCitations).
	e.runCitations(ctx, playerDB, shared, &r)
	clock.lap("citations", r.CitationsComputed)

	// 4.5 Media scan post-sync — indexe les captures présentes dans le dossier
	// du joueur et les associe aux matchs fraîchement insérés (best-effort).
	// ForceRescan=false : seuls les nouveaux fichiers sont traités → coût nul
	// si aucune nouvelle capture depuis la dernière sync.
	if e.mediaHook != nil {
		slog.DebugContext(ctx, "post-sync: scan médias", "gamertag", e.gamertag)
		e.mediaHook(ctx)
	}
	clock.lap("media_scan", 0)

	// 5. Achievements Xbox (fire-and-forget, non bloquant en cas d'erreur token)
	r.AchievementsSynced = e.runAchievementsSync(ctx, playerDB) == achievementsSynced
	clock.lap("achievements", 0)

	// 6. Snapshot readiness (Phase 2) — marque les matchs complets (toutes
	// dérivations terminales, ou terminalement-absentes / grâce dépassée) éligibles
	// au snapshot immuable. Best-effort : n'interrompt pas le pipeline ; réévalué à
	// chaque cycle convergent (aucun nouveau call site — runConditionalPostSync gère).
	// DERNIÈRE étape : elle juge de la complétude de toutes les dérivations ci-dessus.
	withSharedRead("snapshot_readiness", func(sharedDB *sql.DB) {
		if n, err := snapshot.EvaluateSnapshotReadiness(ctx, playerDB, sharedDB, e.xuid, e.titleSlug); err != nil {
			slog.WarnContext(ctx, "post-sync: snapshot readiness échoué", "gamertag", e.gamertag, "err", err)
			trackFatalErr(&r, "snapshot readiness", err)
		} else if n > 0 {
			r.SnapshotReadyMarked = n
			slog.InfoContext(ctx, "post-sync: snapshot readiness marqué", "gamertag", e.gamertag, "count", n)
		}
	})
	clock.lap("snapshot_readiness", r.SnapshotReadyMarked)

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
		created, failed, err := refreshAggregates(ctx, playerDB)
		viewsRefreshed.Add(int32(created))
		if err != nil {
			slog.WarnContext(ctx, "post-sync: aggregates partiellement échoué",
				"gamertag", e.gamertag, "views_created", created, "views_failed", failed, "err", err)
			aggregatesErr = err
			return nil //nolint:nilerr // best-effort, ne propage pas (cohérent avec ancien comportement)
		}
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
