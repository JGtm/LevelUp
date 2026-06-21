package sync

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

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
		// OWNER-ONLY (fix 2026-06-08) : le post-sync de CE joueur ne persiste/avance
		// que SON propre état + watermark + ligne canonique, jamais ceux de ses
		// coéquipiers. Sinon (persist 2 équipes) le sync d'un coéquipier avançait le
		// watermark des autres → leur match partagé sautait à jamais sans ligne LUSR
		// (couplage cross-joueur). Owner-only rend chaque sync AUTONOME : un joueur
		// obtient ses lignes complètes seul, sans dépendre d'un backfill ni du sync
		// d'un autre. Même chemin que le backfill recovery, déjà validé.
		// Porte le titre de l'engine dans le ctx pour le seam LUSR title-aware
		// (GetLUSRChainForTitle). Infinite → classifier défaut ; titres dédiés (h5)
		// → leur classifier. Idempotent si le ctx le porte déjà.
		scoringCtx := ctxkeys.WithTitleSlug(ctx, e.titleSlug)
		if n, err := RunLUSRV2ShadowOwnerOnly(scoringCtx, playerDB, sharedDB, e.xuid); err != nil {
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
