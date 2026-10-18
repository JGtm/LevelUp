package sync

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/ops"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
)

func (e *SyncEngine) runScoringSteps(ctx context.Context, playerDB *sql.DB, shared *SharedAccess, r *domain.PostSyncResult) {
	// Étape 1 contention : les 6 étapes de scoring LISENT shared (les écritures
	// vont dans la player DB) — un seul segment Read couvre le bloc. La SEULE
	// écriture shared (match_intensity, engagement) est accumulée pendant le
	// compute et flushée en burst court APRÈS le release du Read (garde
	// anti-deadlock : jamais de Write pendant un Read en vol).
	var intensities []matchIntensityUpdate

	sharedDB, releaseRead, rerr := shared.Read(ctx)
	if rerr != nil {
		slog.WarnContext(ctx, "post-sync: scoring — lecture shared indisponible, bloc skippé",
			"gamertag", e.gamertag, "err", rerr)
		trackFatalErr(r, "scoring shared read", rerr)
		return
	}
	func() {
		defer releaseRead()
		e.runScoringStepsWithDB(ctx, playerDB, sharedDB, r, &intensities)
	}()

	// Burst d'écriture court : flush des intensités accumulées (best-effort,
	// même sémantique que l'ancien write inline).
	if len(intensities) > 0 {
		wdb, releaseWrite, werr := shared.Write(ctx, "engagement_intensity")
		if werr != nil {
			slog.WarnContext(ctx, "post-sync: burst match_intensity indisponible — flush skippé (retenté au prochain cycle)",
				"gamertag", e.gamertag, "count", len(intensities), "err", werr)
			return
		}
		defer releaseWrite()
		persistMatchIntensities(ctx, wdb, intensities)
	}
}

// runScoringStepsWithDB : corps historique du bloc scoring, inchangé — reçoit un
// handle shared de LECTURE et le collecteur d'intensités (write différé).
func (e *SyncEngine) runScoringStepsWithDB(ctx context.Context, playerDB, sharedDB *sql.DB, r *domain.PostSyncResult, intensities *[]matchIntensityUpdate) {
	ctx = e.pinTitle(ctx)
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
	// résolus depuis le friendsLoader (amis du joueur). Sans loader
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
	// skip silencieux si migration Phase 2 non appliquee. Les écritures
	// match_intensity (shared) sont ACCUMULÉES et flushées par le caller
	// en burst court après ce bloc de lecture.
	slog.DebugContext(ctx, "post-sync: calcul engagement scores", "gamertag", e.gamertag)
	if n, ups, err := batchComputeEngagementScores(ctx, playerDB, sharedDB, e.xuid, false); err != nil {
		slog.WarnContext(ctx, "post-sync: engagement scores échoué", "gamertag", e.gamertag, "err", err)
		trackFatalErr(r, "engagement scores", err)
	} else {
		*intensities = append(*intensities, ups...)
		if n > 0 {
			r.EngagementScoresComputed = n
			slog.DebugContext(ctx, "post-sync: engagement scores calculés", "gamertag", e.gamertag, "count", n)
		}
	}

	// 1.5.b Recompute des engagement coefficients depuis la mediane glissante
	// des paces persistees ci-dessus. Sans ce recompute, coef_team_share reste
	// a 1.0 (cold-start) → pace_attendu = pace_team → courbes superposees a
	// l'ecran (cf. .ai/archive/V7/PLAN_ENGAGEMENT_IMPLEMENTATION.md §4.4).
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

// runSkillRatingSteps exécute le bloc LUSR du post-sync : le LUSR v2 (seul modèle, ADR 0024)
// puis la sentinelle dual-row. Tout est gardé par la capability title.CapLUSR (gate sur
// e.titleSlug, autoritatif côté engine) — un titre sans CapLUSR saute proprement.
// Best-effort : chaque sous-étape logue sur erreur sans interrompre le reste (idempotent).
//
// Accès shared : le v2 écrit player_skill_state_v2 CÔTÉ SHARED, il reçoit donc le
// *SharedAccess (pas un handle de lecture) et fait sa sélection sous Read puis ses PROPRES
// rafales Write("lusr"). Ne JAMAIS lui passer un handle de lecture (régression read-only
// 2026-07-03) ni appeler cette étape sous un Read en vol (garde anti-deadlock).
func (e *SyncEngine) runSkillRatingSteps(ctx context.Context, playerDB *sql.DB, shared *SharedAccess, r *domain.PostSyncResult) {
	ctx = e.pinTitle(ctx)
	if !slugHasLUSR(e.titleSlug) {
		slog.DebugContext(ctx, "post-sync: LUSR skippé — capability absente",
			"gamertag", e.gamertag, "title_slug", e.titleSlug)
		return
	}

	// OWNER-ONLY (fix 2026-06-08) : le post-sync de CE joueur ne persiste/avance
	// que SON propre état + watermark + ligne canonique, jamais ceux de ses
	// coéquipiers (sinon couplage cross-joueur → matchs sautés sans ligne LUSR).
	// Porte le titre de l'engine dans le ctx pour le seam LUSR title-aware
	// (GetLUSRChainForTitle : Infinite → classifier défaut ; h5 → dédié). Écrit la note
	// (match_skill_rank, Stratégie C) ET les 8 composantes de performance
	// (lusr_component_history) des matchs notés ; les matchs exclus n'y entrent pas.
	scoringCtx := ctxkeys.WithTitleSlug(ctx, e.titleSlug)
	if n, err := RunLUSRV2ShadowOwnerOnly(scoringCtx, playerDB, shared, e.xuid,
		MedalExploitOption(e.metadataDBPath, e.xuid)); err != nil {
		slog.WarnContext(ctx, "post-sync: LUSR v2 échoué", "gamertag", e.gamertag, "err", err)
	} else if n > 0 {
		r.LUSRUpdated = n
		slog.InfoContext(ctx, "post-sync: LUSR v2 OK", "gamertag", e.gamertag, "processed", n)
	}

	// Sentinelle dual-row (playerDB-only).
	e.runDualRowSentinelBestEffort(ctx, playerDB)
}

// runDualRowSentinelBestEffort exécute la sentinelle dual-row (Sprint 2.C).
// Détecte l'invariant Stratégie C cassé (match avec LUSR_V2 sans LUSR). Read-only
// sur la PLAYER DB (n'a pas besoin de shared), idempotente, timeout 30s pour ne
// jamais bloquer le post-sync. Toute incohérence → slog.ErrorContext.
func (e *SyncEngine) runDualRowSentinelBestEffort(ctx context.Context, playerDB *sql.DB) {
	if playerDB == nil {
		return
	}
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

// runCSRSnapshotSync récupère les classements CSR du joueur pour la saison courante
// et les persiste dans player_csr_snapshots. Best-effort : skippé si csrSeasonID vide
// (avec WARN explicite pour rendre cette régression de config visible aux ops).
// runCSRSnapshotSync retourne nil sur succès ou skip de config, ou l'erreur
// brute de syncPlayerCSRs en cas d'échec runtime. Le caller peut utiliser
// trackFatalErr pour propager au SyncResult si IsInvalidatedError.
// runCSRSnapshotSync récupère + persiste les CSR snapshots du joueur.
// Retourne la slice CSR pour que le caller puisse alimenter playlists_catalog
// en parallèle (cf. errgroup step 4 dans runPostSyncPipeline).

// runPSAConvergence — étape 1.56 : convergence des personal_score_awards. Cas nominal :
// matchs delta-skippés (insérés en shared par un coéquipier) dont seul le traitement
// per-match écrivait les PSA (gate invariants psa_missing, 2026-06-10). Marqueur terminal
// psa_checked_at : chaque match n'est fetché qu'une fois, borné par convergenceHorizon.
func (e *SyncEngine) runPSAConvergence(ctx context.Context, playerDB *sql.DB, shared *SharedAccess, client HaloClient, r *domain.PostSyncResult) {
	psaWork := selectMatchesMissingPSA(ctx, playerDB)
	observability.AddIntT(ctxkeys.TitleSlug(ctx), "convergence_psa_pending_total", int64(len(psaWork)))
	if len(psaWork) == 0 {
		return
	}
	// Split 4b : fetch réseau + écritures PLAYER hors de tout lease shared ;
	// seul le flush des alias (shared.xuid_aliases) passe en burst court.
	n, pairs := convergePSACollect(ctx, playerDB, client, e.xuid, psaWork)
	if len(pairs) > 0 {
		if wdb, releaseW, werr := shared.Write(ctx, "psa_aliases"); werr != nil {
			slog.WarnContext(ctx, "post-sync: burst psa_aliases indisponible — alias non flushés (retentés au prochain cycle)",
				"gamertag", e.gamertag, "count", len(pairs), "err", werr)
		} else {
			// Corps du burst sous closure : releaseW en defer, donc la fenêtre RW est
			// rendue même si flushAliasPairs panique.
			func() {
				defer releaseW()
				flushAliasPairs(ctx, wdb, e.xuid, pairs)
			}()
		}
	}
	r.ConvergedPSA = n
	observability.AddIntT(ctxkeys.TitleSlug(ctx), "convergence_psa_processed_total", int64(n))
	slog.InfoContext(ctx, "post-sync: convergence PSA",
		"gamertag", e.gamertag, "selected", len(psaWork), "processed", n)
}

// runCatalogRefresh inscrit les nouvelles playlists/maps/paires/variantes dans les tables
// catalogue (zéro réseau) — chemin V1/CLI ; V2 gère le catalogue dans son persister (son
// engine post-sync n'a pas de metaDB : skip propre). CatalogRefreshFromRegistry est
// ART-safe (SELECT-then-write). Best-effort.
func (e *SyncEngine) runCatalogRefresh(ctx context.Context, shared *SharedAccess, r *domain.PostSyncResult, insertedIDs []string) {
	if e.assetPool == nil || e.metaDB == nil || len(insertedIDs) == 0 {
		return
	}
	e.withSharedRead(ctx, shared, r, "catalog_refresh", func(sharedDB *sql.DB) {
		if sharedDB == nil {
			return
		}
		if _, cerr := ops.CatalogRefreshFromRegistry(ctx, e.metaDB, sharedDB, e.titleSlug); cerr != nil {
			slog.WarnContext(ctx, "post-sync: refresh catalogue non-bloquant",
				"gamertag", e.gamertag, "err", cerr)
		}
	})
}

// runFriendsRecompute — étape 3.5 : recalcule is_with_friends (best-effort), autorité
// convergente (ami ajouté/retiré) même quand le premier écrit a déjà posé la valeur. Placée
// AVANT les agrégats : refreshAggregates=false, le refresh natif de l'engine couvre les
// écritures. Skip silencieux sans loader (legacy) ou avec liste vide.
func (e *SyncEngine) runFriendsRecompute(ctx context.Context, playerDB *sql.DB, shared *SharedAccess, r *domain.PostSyncResult) {
	if e.friendsLoader == nil {
		return
	}
	friends, ferr := e.friendsLoader()
	if ferr != nil {
		slog.WarnContext(ctx, "post-sync: friends loader échoué", "gamertag", e.gamertag, "err", ferr)
		trackFatalErr(r, "friends loader", ferr)
		return
	}
	if len(friends) == 0 {
		return
	}
	slog.DebugContext(ctx, "post-sync: friends recompute", "gamertag", e.gamertag, "friends_count", len(friends))
	e.withSharedRead(ctx, shared, r, "friends", func(sharedDB *sql.DB) {
		fres, err := RecomputeIsWithFriendsCore(ctx, playerDB, sharedDB, e.xuid, friends, false)
		if err != nil {
			slog.WarnContext(ctx, "post-sync: friends recompute échoué", "gamertag", e.gamertag, "err", err)
			trackFatalErr(r, "friends recompute", err)
		} else if fres.MatchesPromoted > 0 {
			r.MatchesPromotedFriends = fres.MatchesPromoted
			slog.InfoContext(ctx, "post-sync: matchs reclasses comme escouade-amis",
				"gamertag", e.gamertag,
				"promoted", fres.MatchesPromoted,
			)
		}
	})
}

// runDominanceFlags — étape 1.7 : matchs nouvellement insérés + backlog
// (dominance_flag IS NULL). Reconstruit la courbe score depuis highlight_events (donc APRÈS
// la convergence events 1.54). Valeur 0 = calculé/sans dominance, NULL = jamais calculé.
func (e *SyncEngine) runDominanceFlags(ctx context.Context, playerDB *sql.DB, shared *SharedAccess, r *domain.PostSyncResult, insertedIDs []string) {
	missingIDs, merr := selectMatchesMissingDominanceFlags(ctx, playerDB)
	if merr != nil {
		slog.WarnContext(ctx, "post-sync: dominance detect échoué", "gamertag", e.gamertag, "err", merr)
		trackFatalErr(r, "dominance detect", merr)
	}
	allDominanceIDs := mergeUniqMatchIDs(insertedIDs, missingIDs)
	if len(allDominanceIDs) == 0 {
		return
	}
	// Lecture shared (courbe score depuis highlight_events) ; écriture via
	// PostSyncEnrichmentPersister(playerDB) — segment Read suffit.
	e.withSharedRead(ctx, shared, r, "dominance", func(sharedDB *sql.DB) {
		if err := BackfillDominanceFlags(ctx, sharedDB, playerDB, e.xuid, allDominanceIDs); err != nil {
			slog.WarnContext(ctx, "post-sync: dominance flags échoué",
				"gamertag", e.gamertag, "err", err, "count", len(allDominanceIDs))
			trackFatalErr(r, "dominance flags", err)
		} else {
			r.DominanceFlagsComputed = len(allDominanceIDs)
			slog.InfoContext(ctx, "post-sync: dominance flags calculés",
				"gamertag", e.gamertag, "count", r.DominanceFlagsComputed)
		}
	})
}

// runCitations — étape 1.6 : pipeline primaire, pas un heal. Traite tous les matchs absents
// de match_citations (LEFT JOIN IS NULL). Le sentinel "_processed" empêche les matchs à
// 0 delta d'être re-traités. APRÈS la source du kill (1.57) : les citations par arme lisent
// ses lignes et leur calcul est unique.
func (e *SyncEngine) runCitations(ctx context.Context, playerDB *sql.DB, shared *SharedAccess, r *domain.PostSyncResult) {
	e.withSharedRead(ctx, shared, r, "citations", func(sharedDB *sql.DB) {
		if n, err := e.runPostSyncCitations(ctx, playerDB, sharedDB); err != nil {
			slog.WarnContext(ctx, "post-sync: citations échoué", "gamertag", e.gamertag, "err", err)
			trackFatalErr(r, "citations", err)
		} else if n > 0 {
			r.CitationsComputed = n
			slog.InfoContext(ctx, "post-sync: citations calculées", "gamertag", e.gamertag, "count", n)
		}
	})
}

// flushReadCaches vide les caches de lecture du joueur (lignes de filtres et historique
// canonique, L5b) : appelée dès que les lignes lues par les pages sont à jour (après les
// agrégats), AVANT les étapes film lentes — sans quoi une page lue pendant la lecture des
// films sert une version périmée jusqu'au TTL. Le vidage final du `defer` de
// runPostSyncPipeline reste en place pour ce que les étapes film écrivent ensuite. Sa
// position avant la source du kill est tenue par archlint/player_read_cache_invalidation_test.go.
func (e *SyncEngine) flushReadCaches(ctx context.Context) {
	duckdbpkg.InvalidatePlayerReadCaches(ctx, e.xuid, e.titleSlug)
}
