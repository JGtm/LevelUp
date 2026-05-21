// Package sync — engine_postsync.go : pipeline post-sync + sync achievements.
//
// Extrait de engine.go (refactor 2026-05-21). Regroupe :
//   - runConditionalPostSync : branche post-sync selon matchs insérés ou heal.
//   - runPostSyncPipeline : pipeline complet 14+ étapes (heal stats/skill/events/
//     weapons, had_bot, sessions, perf/engagement/LUSR/citations, CSR snapshots,
//     friends recompute, aggregates, achievements).
//   - runCSRSnapshotSync : CSR snapshots best-effort si csrSeasonID renseigné.
//   - runAchievementsSync + RunAchievementsOnly : sync Xbox achievements via
//     TokenProvider (resolveAccessTokenFromDB → XSTS → SyncAchievements).
//   - hasMatchesNeedingScoreRefresh : heuristique heal-only path.
//   - resolveAccessTokenFromDB : lecture cache MSAL/refresh + fallback env.
//
// Comportement INCHANGÉ — pur déplacement.
//
// Voir engine.go (struct SyncEngine + run()) pour le contexte.
package sync

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/dblease"
)

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

	// Pas de nouveaux matchs : on tente d'abord un heal skill — si ça remplit
	// des champs (team_mmr, kills_expected), il faut quand même lancer le
	// pipeline post-sync complet pour recalculer perf/engagement/LUSR/citations
	// qui dépendent de ces champs.
	healed, healErr := healSkillForMissingMatches(ctx, sharedDB, client, e.xuid, 200)
	if healErr != nil {
		slog.WarnContext(ctx, "sync: skill heal échoué (no-insert path)", "gamertag", e.gamertag, "err", healErr)
	}
	// Détecter aussi les matchs avec scores manquants (engagement/perf NULL).
	// Si présents, on lance le PostSync complet pour les combler.
	needsScoreRefresh, _ := hasMatchesNeedingScoreRefresh(ctx, playerDB, sharedDB, e.xuid)
	if healed > 0 || needsScoreRefresh {
		slog.InfoContext(ctx, "sync: aucun match inséré — heal/scores → lancement post-sync complet",
			"gamertag", e.gamertag, "matches_healed", healed, "needs_score_refresh", needsScoreRefresh)
		return e.runPostSyncPipeline(ctx, playerDB, sharedDB, client, nil)
	}
	slog.DebugContext(ctx, "sync: aucun match inséré — refresh CSR + achievements seul (carrière live découplé)", "gamertag", e.gamertag)
	// Carrière (XP + Spartan ID) retirée du post-sync : service.CareerLiveService
	// la rafraîchit live à chaque chargement de /pages/home.
	e.runCSRSnapshotSync(ctx, playerDB, client)
	return domain.PostSyncResult{
		AchievementsSynced: e.runAchievementsSync(ctx, playerDB),
	}
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
) domain.PostSyncResult {
	var r domain.PostSyncResult

	// Sprint B1 commit 18 : event_id pour tracer le pipeline post-sync à
	// travers ses 14+ étapes (stats heal, skill heal, events heal, weapons,
	// bot teammate, sessions, perf scores, engagement, LUSR, citations, CSR,
	// friends, aggregates). Tous les sous-logs hériteront automatiquement.
	ctx, evID := logging.WithEvent(ctx, "sync.postSync:"+e.gamertag)
	slog.InfoContext(ctx, "post-sync: pipeline démarré",
		"gamertag", e.gamertag, "matches_inserted", len(insertedIDs), "event", evID)

	// -1.5 Stats re-extraction heal — comble max_killing_spree, grenade/melee/
	// power_weapon kills, time_played_seconds, avg_life_seconds, gamertag,
	// team_X_ps_score pour les matchs synchronisés avec un ancien binaire.
	// Détection via max_killing_spree IS NULL. Limit 10 pour amortir.
	if n, err := healStatsForRecentMatches(ctx, sharedDB, client, e.xuid, e.gamertag, 10); err != nil {
		slog.WarnContext(ctx, "post-sync: stats heal échoué", "gamertag", e.gamertag, "err", err)
	} else if n > 0 {
		slog.InfoContext(ctx, "post-sync: stats self-heal", "gamertag", e.gamertag, "matches_healed", n)
	}

	// -1. Skill self-heal — comble team_mmr/enemy_mmr/kills_expected/deaths_expected
	// pour les matchs synchronisés AVANT que GetMatchSkill ne soit câblé dans
	// processMatch (ou avec un échec transitoire). Idempotent : 0 appel API
	// si tout est déjà rempli. Doit tourner avant performance/LUSR qui
	// dépendent de team_mmr et kills_expected.
	if n, err := healSkillForMissingMatches(ctx, sharedDB, client, e.xuid, 200); err != nil {
		slog.WarnContext(ctx, "post-sync: skill heal échoué", "gamertag", e.gamertag, "err", err)
	} else if n > 0 {
		slog.InfoContext(ctx, "post-sync: skill self-heal", "gamertag", e.gamertag, "matches_healed", n)
	}

	// -0.5 Highlight events / killer_victim heal pour les matchs récents où
	// events_loaded=FALSE (matchs syncés avant que processHighlightEvents ne
	// soit câblé). Best-effort : films absents → 404 silencieux. globalDB nil
	// est OK : xuid_aliases déjà résolu pour ces matchs.
	// Limit 20 pour amortir : processHighlightEvents marque events_loaded=TRUE
	// même sur 404, donc converge en quelques syncs.
	if h, nf, err := healEventsForRecentMatches(ctx, sharedDB, nil, client, 20); err != nil {
		slog.WarnContext(ctx, "post-sync: events heal échoué", "gamertag", e.gamertag, "err", err)
	} else if h > 0 || nf > 0 {
		slog.InfoContext(ctx, "post-sync: events self-heal",
			"gamertag", e.gamertag, "healed", h, "no_film", nf)
	}

	// -0.4 Weapon kills heal pour les matchs récents où le pipeline n'a jamais
	// tourné (bit MBitWeaponKills absent dans match_registry). Dépend des
	// highlight_events ci-dessus (kills attribution lit highlight_events).
	// Limit 10 : weapon kills marque le bit MBitWeaponKills aussi sur no-film,
	// donc converge en quelques syncs.
	if h, nf, err := healWeaponKillsForRecentMatches(ctx, sharedDB, client, e.xuid, 10); err != nil {
		slog.WarnContext(ctx, "post-sync: weapon heal échoué", "gamertag", e.gamertag, "err", err)
	} else if h > 0 || nf > 0 {
		slog.InfoContext(ctx, "post-sync: weapon self-heal",
			"gamertag", e.gamertag, "healed", h, "no_film", nf)
	}

	// -0.3 had_bot_teammate — dérivé des participants (cheap SQL, pas d'API).
	// Idempotent : skip les rows déjà à TRUE.
	if n, err := computeAndPersistHadBotTeammate(ctx, playerDB, sharedDB, e.xuid); err != nil {
		slog.WarnContext(ctx, "post-sync: had_bot_teammate échoué", "gamertag", e.gamertag, "err", err)
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
		} else if n > 0 {
			r.SessionsAssigned = n
			slog.DebugContext(ctx, "post-sync: sessions recalculées", "gamertag", e.gamertag, "count", n)
		}
	}

	// 1. Performance scores
	slog.DebugContext(ctx, "post-sync: calcul perf scores", "gamertag", e.gamertag)
	if n, err := batchComputePerformanceScores(playerDB, sharedDB, e.xuid, nil, false); err != nil {
		slog.WarnContext(ctx, "post-sync: perf scores échoué", "gamertag", e.gamertag, "err", err)
	} else {
		r.PerfScoresComputed = n
		slog.DebugContext(ctx, "post-sync: perf scores calculés", "gamertag", e.gamertag, "count", n)
	}

	// 1.5 Engagement scores (Phase 3 plan engagement) — best-effort,
	// skip silencieux si migration Phase 2 non appliquee.
	slog.DebugContext(ctx, "post-sync: calcul engagement scores", "gamertag", e.gamertag)
	if n, err := batchComputeEngagementScores(ctx, playerDB, sharedDB, e.xuid, false); err != nil {
		slog.WarnContext(ctx, "post-sync: engagement scores échoué", "gamertag", e.gamertag, "err", err)
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
	} else if n > 0 {
		r.EngagementCoefsUpdated = n
		slog.DebugContext(ctx, "post-sync: engagement coefs mis à jour", "gamertag", e.gamertag, "count", n)
	}

	// 1.52 Assists model — OLS per-mode, skip silencieux si migration absente.
	// force=false : ne recalcule que si player_assists_model est vide (cold-start).
	// Un nouveau sync peut amener des données → on recalcule si table vide.
	if n, err := batchComputePlayerAssistsModel(ctx, playerDB, sharedDB, e.xuid, false); err != nil {
		slog.WarnContext(ctx, "post-sync: assists model échoué", "gamertag", e.gamertag, "err", err)
	} else if n > 0 {
		slog.InfoContext(ctx, "post-sync: assists model calculé", "gamertag", e.gamertag, "n_modes", n)
	}

	// 1.55 Weapon kills — pipeline film pour les matchs nouvellement insérés.
	// Best-effort : films absents (404/410) sont normaux pour les vieux matchs
	// et n'échouent pas le sync. Limité aux nouveaux matchs (insertedIDs) pour
	// éviter de re-traiter l'historique à chaque sync.
	if len(insertedIDs) > 0 {
		done, noFilm, werr := processWeaponKillsInline(ctx, sharedDB, client, e.xuid, insertedIDs)
		if werr != nil {
			slog.WarnContext(ctx, "post-sync: weapon kills échoué", "gamertag", e.gamertag, "err", werr)
		}
		r.WeaponKillsProcessed = done
		r.WeaponKillsNoFilm = noFilm
		if done > 0 || noFilm > 0 {
			slog.InfoContext(ctx, "post-sync: weapon kills",
				"gamertag", e.gamertag, "done", done, "no_film", noFilm)
		}
	}

	// 1.6 Citations (best-effort) — calcul des deltas pour les matchs absents
	// de match_citations. Skip silencieux si metadata.duckdb introuvable ou si
	// citation_mappings vide. Ne propage jamais d'erreur (le sync ne doit pas
	// echouer a cause des citations).
	if n, err := e.runPostSyncCitations(ctx, playerDB, sharedDB); err != nil {
		slog.WarnContext(ctx, "post-sync: citations échoué", "gamertag", e.gamertag, "err", err)
	} else if n > 0 {
		slog.InfoContext(ctx, "post-sync: citations calculées",
			"gamertag", e.gamertag, "match_count", n)
	}

	// 2. LUSR (TrueSkill 2) — best-effort medal data depuis metadata DB
	slog.DebugContext(ctx, "post-sync: calcul LUSR", "gamertag", e.gamertag)
	medalMap := e.loadMedalExploitMapBestEffort(ctx, sharedDB)
	if n, err := batchComputeLUSR(playerDB, sharedDB, e.xuid, medalMap, false); err != nil {
		slog.WarnContext(ctx, "post-sync: LUSR échoué", "gamertag", e.gamertag, "err", err)
	} else {
		r.LUSRUpdated = n
		slog.DebugContext(ctx, "post-sync: LUSR mis à jour", "gamertag", e.gamertag, "count", n)
	}

	// 3. Career rank — DÉCOUPLÉ du post-sync depuis 2026-05-14.
	// Le flow XP + Spartan ID est désormais géré par service.CareerLiveService
	// (throttle 5 min / 6 h + fallback DB per-field), appelé depuis HomeService
	// à chaque chargement de /pages/home. Voir .ai/thought_log.md.
	// domain.PostSyncResult.CareerSynced reste dans le struct (compat e2e tests)
	// mais n'est plus jamais positionné à true ici.

	// 3.1 CSR snapshots (best-effort, skip silencieux si csrSeasonID vide).
	// Maintenu dans le post-sync : le CSR ne bouge que sur fin de match ranked,
	// donc le déclencheur "nouveau match" reste pertinent.
	e.runCSRSnapshotSync(ctx, playerDB, client)

	// 3.5 Friends recompute is_with_friends (best-effort).
	// Avant l'étape 4 (aggregates) pour éviter un double-refresh : on passe
	// refreshAggregates=false, le refresh natif de l'engine couvre les UPDATEs.
	// Skip silencieux si pas de loader (legacy) ou liste vide.
	if e.friendsLoader != nil {
		if friends, ferr := e.friendsLoader(); ferr != nil {
			slog.WarnContext(ctx, "post-sync: friends loader échoué", "gamertag", e.gamertag, "err", ferr)
		} else if len(friends) > 0 {
			slog.DebugContext(ctx, "post-sync: friends recompute", "gamertag", e.gamertag, "friends_count", len(friends))
			fres, err := RecomputeIsWithFriendsCore(ctx, playerDB, sharedDB, e.xuid, friends, false)
			if err != nil {
				slog.WarnContext(ctx, "post-sync: friends recompute échoué", "gamertag", e.gamertag, "err", err)
			} else if fres.MatchesPromoted > 0 {
				r.MatchesPromotedFriends = fres.MatchesPromoted
				slog.InfoContext(ctx, "post-sync: matchs reclasses comme escouade-amis",
					"gamertag", e.gamertag,
					"promoted", fres.MatchesPromoted,
				)
			}
		}
	}

	// 4. Aggregates (materialized views)
	slog.DebugContext(ctx, "post-sync: refresh aggregates player", "gamertag", e.gamertag)
	if n, err := refreshAggregates(ctx, playerDB); err != nil {
		slog.WarnContext(ctx, "post-sync: aggregates échoué", "gamertag", e.gamertag, "err", err)
	} else {
		r.ViewsRefreshed = n
	}
	slog.DebugContext(ctx, "post-sync: refresh shared views", "gamertag", e.gamertag)
	if n, err := refreshSharedViews(ctx, sharedDB); err != nil {
		slog.WarnContext(ctx, "post-sync: shared views échoué", "gamertag", e.gamertag, "err", err)
	} else {
		r.ViewsRefreshed += n
	}

	// 5. Achievements Xbox (fire-and-forget, non bloquant en cas d'erreur token)
	r.AchievementsSynced = e.runAchievementsSync(ctx, playerDB)

	return r
}

// runCSRSnapshotSync récupère les classements CSR du joueur pour la saison courante
// et les persiste dans player_csr_snapshots. Best-effort : skippé si csrSeasonID vide.
func (e *SyncEngine) runCSRSnapshotSync(ctx context.Context, playerDB *sql.DB, client HaloClient) {
	if strings.TrimSpace(e.csrSeasonID) == "" {
		return
	}
	slog.DebugContext(ctx, "post-sync: sync CSR snapshots", "gamertag", e.gamertag, "season", e.csrSeasonID)
	n, err := syncPlayerCSRs(ctx, client, playerDB, e.xuid, e.csrSeasonID)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: CSR snapshots échoué", "gamertag", e.gamertag, "err", err)
		return
	}
	slog.DebugContext(ctx, "post-sync: CSR snapshots sauvegardés", "gamertag", e.gamertag, "count", n)
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

	// Ouvrir la DB metadata (lecture-écriture pour l'upsert).
	metadataDB, err := sql.Open("duckdb", e.metadataDBPath)
	if err != nil {
		slog.WarnContext(ctx, "achievements: ouverture metadata DB échouée",
			"gamertag", e.gamertag, "err", err)
		return false
	}
	defer metadataDB.Close() //nolint:errcheck
	metadataDB.SetMaxOpenConns(1)

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
