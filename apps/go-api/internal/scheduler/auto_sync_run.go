// Package scheduler — auto_sync_run.go : boucle de run périodique + cycle de sync
// par joueur (Run / RunOnce / RunOnceTrigger / syncPlayersConcurrent / syncPlayer /
// checkSyncPreconditions + le type syncOutcome). Extrait de auto_sync.go (K2c,
// 2026-07-06) pour ramener ce dernier sous 500 L. Déplacement pur, même package.
package scheduler

import (
	"context"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_5/livesync"
	"levelup/go-api/internal/observability/logging"
)

// Run démarre la boucle périodique. Doit être lancé dans une goroutine.
// Se termine proprement à l'annulation de ctx.
func (s *AutoSyncScheduler) Run(ctx context.Context) {
	interval := s.CurrentInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.InfoContext(ctx, "auto_sync: scheduler démarré",
		"interval", interval,
		"interval_hours", int(interval.Hours()),
		"pool_size", s.poolSizeSafe(),
	)

	for {
		select {
		case <-ticker.C:
			// Sprint B1 commit 17 : event_id sur chaque tick pour tracer le
			// cycle scheduler complet à travers les modules (scheduler →
			// auth → sync → provider → pool → handlers post-sync).
			tickCtx, tickID := logging.WithEvent(ctx, "scheduler.tick")
			cfg, err := s.settings.Load()
			if err != nil {
				slog.WarnContext(tickCtx, "auto_sync: lecture settings échouée — tick ignoré", "err", err)
				continue
			}
			if !cfg.SpnkrAutoSyncEnabled {
				slog.DebugContext(tickCtx, "auto_sync: désactivé dans les settings, tick ignoré")
				continue
			}
			if newInterval := resolveInterval(cfg.SpnkrAutoSyncIntervalMinutes, cfg.SpnkrAutoSyncIntervalHours); newInterval != interval {
				slog.InfoContext(tickCtx, "auto_sync: intervalle mis à jour", "old", interval, "new", newInterval)
				interval = newInterval
				ticker.Reset(interval)
			}
			slog.InfoContext(tickCtx, "auto_sync: tick démarré", "event", tickID)
			res := s.RunOnceTrigger(tickCtx, "tick")
			slog.InfoContext(tickCtx, "auto_sync: cycle terminé",
				"total", res.Total,
				"synced", res.Synced,
				"skipped", res.Skipped,
				"failed", res.Failed,
				"duration", res.Duration.Round(time.Millisecond),
			)
			if res.Failed > 0 {
				slog.WarnContext(tickCtx, "auto_sync: des joueurs ont échoué — consulter le snapshot diag",
					"failed_count", res.Failed,
				)
			}
			if res.Total > 0 && res.Skipped == res.Total {
				slog.WarnContext(tickCtx, "auto_sync: aucun joueur synchronisé — vérifier le pool",
					"pool_size", s.poolSizeSafe(),
					"hint", "voir GET /api/v1/_diag/auto-sync/snapshot pour le détail",
				)
			}
		case <-ctx.Done():
			slog.InfoContext(ctx, "auto_sync: arrêt du scheduler (contexte annulé)")
			return
		}
	}
}

// RunOnce exécute un cycle de sync pour tous les joueurs configurés.
// Peut être appelé manuellement (debug, endpoint admin) sans attendre un tick.
func (s *AutoSyncScheduler) RunOnce(ctx context.Context) *RunOnceResult {
	return s.RunOnceTrigger(ctx, "manual")
}

// RunOnceTrigger est RunOnce avec la provenance du déclenchement ("tick" pour
// la boucle périodique, "manual" pour HTTP/diag/job) — tracée dans
// l'historique des cycles du dashboard monitoring.
func (s *AutoSyncScheduler) RunOnceTrigger(ctx context.Context, trigger string) *RunOnceResult {
	start := time.Now()
	// Cumuls process-wide capturés AVANT le cycle : les deltas après cycle
	// attribuent au cycle sa fenêtre d'indispo lectures, ses 503, son temps
	// API et son temps d'écriture (corrélation dashboard monitoring P4).
	loadBefore := captureCycleLoad()
	res := &RunOnceResult{}

	players, err := s.cfg.LoadPlayers()
	if err != nil {
		slog.ErrorContext(ctx, "auto_sync: chargement des joueurs échoué", "err", err)
		return res
	}
	// Exclure les couples (joueur, titre) en pause (sync_enabled=false) : ils ne
	// sont plus rafraîchis. Filtre à la SOURCE → couvre V1 (boucle syncPlayer) ET
	// V2 (runOnceV2 reçoit la slice déjà filtrée).
	players = domain.SyncablePlayers(players)
	res.Total = len(players)

	// Détection de claim fuité : le cycle auto-sync sert de heartbeat. Un claim
	// du gate tenu anormalement longtemps signale un release jamais appelé (le
	// joueur ne serait plus jamais synchronisé). Cf. jauge expvar sync_gate_inflight
	// pour le signal temps-réel.
	s.warnStaleGateClaims(ctx)

	// Séparer les titres live-only (Halo 5+) du batch moteur : l'orchestrator V2
	// est mono-titre (Infinite) et ne sait pas router un titre live-only. Ces
	// joueurs passent TOUJOURS par syncPlayer, qui sélectionne le runner de façon
	// registry-driven (livesync.HandlesTitle → liveRunner), que V2 soit actif ou
	// non. Le batch V2 ne reçoit donc que les joueurs pilotés par le SyncEngine.
	var orchPlayers, livePlayers []domain.PlayerSummary
	for _, p := range players {
		if livesync.HandlesTitle(resolveTitleSlug(p)) {
			livePlayers = append(livePlayers, p)
		} else {
			orchPlayers = append(orchPlayers, p)
		}
	}

	// Dispatch (ADR 0027 — pipeline V1 supprimé au lot D1c) : le pipeline V2 est
	// l'UNIQUE moteur de sync du cycle dès qu'il est câblé. Les joueurs MOTEUR
	// (Infinite) passent par l'orchestrator V2 ; les joueurs LIVE-ONLY (Halo 5)
	// par syncPlayer (→ liveRunner) en parallèle. Plus de flag LEVELUP_SYNC_PIPELINE
	// ni de fallback automatique vers V1 : un échec V2 est loggé et re-tenté au
	// cycle suivant (pipeline append-only idempotent), jamais rejoué via l'ancien
	// chemin UPSERT ART-unsafe.
	if s.shouldUseV2() {
		v2Res, v2Err := s.runOnceV2(ctx, orchPlayers)
		if v2Err != nil {
			slog.ErrorContext(ctx, "auto_sync: cycle V2 en échec (pas de fallback V1 — re-tenté au prochain cycle)",
				"err", v2Err)
			v2Res = &RunOnceResult{} // runOnceV2 renvoie nil sur erreur globale
		}
		liveSynced, liveSkipped, liveFailed := s.syncPlayersConcurrent(ctx, livePlayers)
		v2Res.Synced += liveSynced
		v2Res.Skipped += liveSkipped
		v2Res.Failed += liveFailed
		v2Res.Total = len(players)
		// Duration couvre le cycle COMPLET (orchestrator + sync live-only) —
		// runOnceV2 ne mesure que l'orchestrator (parité avec le filet ci-dessous).
		v2Res.Duration = time.Since(start)
		s.storeCycleResult(ctx, v2Res, trigger, captureCycleLoad().deltaSince(loadBefore))
		s.notifyDiscordSyncCycle(ctx, trigger, start, v2Res)
		return v2Res
	}

	// Filet structurel : orchestrator V2 non câblé (prérequis boot manquants —
	// pool/queue/metaDB). On sync directement via syncPlayer (qui route Infinite ET
	// live-only). Ce n'est PAS l'ancien pipeline V1 flag-sélectionnable (supprimé),
	// mais une sécurité de démarrage — en prod l'orchestrator est toujours câblé.
	slog.WarnContext(ctx, "auto_sync: orchestrator V2 non câblé — sync directe via syncPlayer (filet de boot)",
		"player_count", res.Total)
	res.Synced, res.Skipped, res.Failed = s.syncPlayersConcurrent(ctx, players)

	// PLAN_SPARTAN_IDENTITY_REFACTOR §11 Phase 5 (2026-05-25) :
	// customRefresher.RefreshAll(ctx) supprimé. La customisation est désormais
	// rafraîchie en LIVE à chaque visite home (CareerLiveService.kickoff).

	res.Duration = time.Since(start)

	s.storeCycleResult(ctx, res, trigger, captureCycleLoad().deltaSince(loadBefore))
	s.notifyDiscordSyncCycle(ctx, trigger, start, res)

	return res
}

// syncPlayersConcurrent lance syncPlayer pour chaque joueur en parallèle (errgroup
// borné par la taille du pool de tokens) et agrège synced/skipped/failed. Extrait
// de la boucle de cycle (Phase 3.4) pour être réutilisé par le path V2 (joueurs
// live-only, non gérés par l'orchestrator) ET le path V1 (tous les joueurs).
//
// Best-effort : un échec syncPlayer n'annule pas les autres goroutines (l'erreur
// est déjà loggée + reflétée dans le compteur failed). Safety : syncPlayer met à
// jour s.playerOutcomes via recordOutcome (protégé par s.snapshotMu) ; les writes
// shared sont sérialisés par dblease + singleflight ; les compteurs locaux sont
// protégés par atomic.Int32.
func (s *AutoSyncScheduler) syncPlayersConcurrent(ctx context.Context, players []domain.PlayerSummary) (synced, skipped, failed int) {
	if len(players) == 0 {
		return 0, 0, 0
	}
	parallelism := s.poolSizeSafe()
	if parallelism < 1 {
		parallelism = 1
	}
	var syncedN, skippedN, failedN atomic.Int32
	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(parallelism)
	for _, p := range players {
		p := p
		eg.Go(func() error {
			switch s.syncPlayer(egCtx, p) {
			case outcomeOK:
				syncedN.Add(1)
			case outcomeSkipped:
				skippedN.Add(1)
			case outcomeFailed:
				failedN.Add(1)
			}
			return nil
		})
	}
	_ = eg.Wait()
	return int(syncedN.Load()), int(skippedN.Load()), int(failedN.Load())
}

// syncOutcome encode le résultat d'une tentative de sync par joueur.
type syncOutcome int

const (
	outcomeOK      syncOutcome = iota // sync delta réussie
	outcomeSkipped                    // pas de token / watcher actif / DB absente → ignoré sans erreur
	outcomeFailed                     // erreur bloquante pendant RunDelta
)

// syncPlayer lance une sync delta pour un joueur via le pool de tokens.
//
// Conditions de skip silencieux (outcome=skipped) :
//   - pool nil (aucun credential découvert au boot)
//   - joueur absent du pool (pas dans .env.local et pas dans sync_meta)
//   - watcher actif sur ce joueur (céder la priorité pour éviter conflit DB)
//   - DB joueur absente (sync initiale jamais effectuée)
//
// Enregistre toujours un PlayerOutcomeDetail (via defer) pour exposition via
// /api/v1/_diag/auto-sync/snapshot.
//
// présente) → exécution sync delta → enregistrement outcome. Splitter forcerait à
// dupliquer le state PlayerOutcomeDetail dans les sous-fonctions.
//
//nolint:funlen // Orchestrateur sync-per-player : guards (cooldown, watcher, DB
func (s *AutoSyncScheduler) syncPlayer(ctx context.Context, p domain.PlayerSummary) syncOutcome {
	// Sprint B1 commit 17 : event_id par joueur (sous-événement du tick).
	// Permet de filtrer logs/*.log pour reconstituer le timeline d'un user
	// donné indépendamment des autres en cours de sync.
	ctx, evID := logging.WithEvent(ctx, "scheduler.sync:"+p.Gamertag)
	slog.InfoContext(ctx, "auto_sync: traitement joueur démarré",
		"gamertag", p.Gamertag, "xuid", p.XUID, "event", evID)

	startedAt := time.Now()
	// Lecture du compteur zero-insert précédent AVANT toute exécution. La défer
	// finale appelle recordOutcome qui écrasera la valeur ; on garde la
	// précédente pour pouvoir l'incrémenter ou la conserver selon l'outcome.
	prevZeroInserts := s.previousZeroInsertCount(p.Gamertag)
	detail := PlayerOutcomeDetail{
		Gamertag:               p.Gamertag,
		XUID:                   p.XUID,
		AttemptedAt:            startedAt,
		ConsecutiveZeroInserts: prevZeroInserts, // défaut : préserver (cas skipped/failed)
	}
	var outcome syncOutcome
	defer func() {
		switch outcome {
		case outcomeOK:
			detail.Outcome = "ok"
		case outcomeSkipped:
			detail.Outcome = "skipped"
		case outcomeFailed:
			detail.Outcome = "failed"
		}
		detail.DurationMs = time.Since(startedAt).Milliseconds()
		s.recordOutcome(detail)
	}()

	if skipReason, ok := s.checkSyncPreconditions(ctx, p); !ok {
		detail.Reason = skipReason
		outcome = outcomeSkipped
		return outcome
	}

	// Gate cross-source : céder si un sync du même joueur est déjà en vol (watcher
	// ou HTTP). Posé APRÈS checkSyncPreconditions (skip économe sans claim) et
	// AVANT le RunDelta ; `defer release()` est la 1re ligne post-claim → couvre
	// tous les retours faillibles (runner nil, RunDelta err, succès). Un skip ne
	// marque JAMAIS le joueur à jour → re-tenté au prochain tick quand le claim se
	// libère. Complète l'ActivityChecker (déjà appliqué) pour le résidu TOCTOU.
	if s.SyncGate != nil {
		release, ok := s.SyncGate.TryClaimT(resolveTitleSlug(p), p.Gamertag)
		if !ok {
			slog.InfoContext(ctx, "auto_sync: sync déjà en vol (autre source) — tick différé",
				"gamertag", p.Gamertag)
			detail.Reason = "différé: sync en cours via autre source (watcher/HTTP)"
			outcome = outcomeSkipped
			return outcome
		}
		defer release()
	}

	// ──────────────────────────────────────────────────────────────────────
	// Sync delta via DeltaRunner (PooledHaloClient en production, mockable en test).
	// ──────────────────────────────────────────────────────────────────────
	// MT-11 / PMT-3 : pose le titre du joueur dans le ctx AVANT la factory, pour
	// que BuildEngine (→ NewSyncEngineForTitle) écrive dans les DB du bon titre
	// et que les sous-modules/logs héritent du slug.
	slug := resolveTitleSlug(p)
	ctx = ctxkeys.WithTitleSlug(ctx, slug)
	slog.InfoContext(ctx, "auto_sync: démarrage sync delta", "gamertag", p.Gamertag, "title_slug", slug)

	// Sélection registry-driven (jamais slug==) du runner. Les titres live-only
	// (Halo 5+) ont leur propre pipeline (fetch cryptum → persist shared) et leur
	// auth transite par le ctx (pas de PooledHaloClient interne) → branche dédiée.
	var runner DeltaRunner
	if livesync.HandlesTitle(slug) {
		r, runCtx, release, lerr := s.liveRunner(ctx, slug, p.Gamertag, p.XUID)
		if lerr != nil {
			slog.ErrorContext(ctx, "auto_sync: runner live indisponible",
				"gamertag", p.Gamertag, "title_slug", slug, "err", lerr)
			detail.Reason = "runner live indisponible: " + lerr.Error()
			detail.FirstError = lerr.Error()
			outcome = outcomeFailed
			return outcome
		}
		defer release()
		runner, ctx = r, runCtx
	} else {
		factory := s.RunnerFactory
		if factory == nil {
			factory = s.defaultRunnerFactory
		}
		runner = factory(ctx, p.Gamertag, p.XUID)
		if runner == nil {
			slog.ErrorContext(ctx, "auto_sync: RunnerFactory a retourné nil",
				"gamertag", p.Gamertag)
			detail.Reason = "RunnerFactory a retourné nil (pool absent ?)"
			outcome = outcomeFailed
			return outcome
		}
	}

	syncResult, err := runner.RunDelta(ctx, domain.DefaultSyncOptions())
	if err != nil {
		slog.ErrorContext(ctx, "auto_sync: RunDelta échoué",
			"gamertag", p.Gamertag, "err", err)
		detail.Reason = "RunDelta échoué: " + err.Error()
		detail.FirstError = err.Error()
		outcome = outcomeFailed
		return outcome
	}

	detail.MatchesInserted = syncResult.MatchesInserted
	detail.MatchesSkipped = syncResult.MatchesSkipped
	detail.MedalsInserted = syncResult.MedalsInserted
	detail.SyncStatus = syncResult.Status()
	detail.ErrorCount = len(syncResult.Errors)
	// Compteurs post-sync exposés au dashboard monitoring (copie défensive :
	// le snapshot survit au SyncResult du cycle).
	if syncResult.PostSync != nil {
		ps := *syncResult.PostSync
		detail.PostSync = &ps
	}

	// Counter zero-insert : reset si on insère ≥1 match, sinon incrément.
	// Sentinelle d'API stale / format URL incorrect / gamertag changé.
	if syncResult.MatchesInserted > 0 {
		detail.ConsecutiveZeroInserts = 0
	} else {
		detail.ConsecutiveZeroInserts = prevZeroInserts + 1
		if detail.ConsecutiveZeroInserts >= ConsecutiveZeroInsertWarnThreshold {
			slog.WarnContext(ctx, "auto_sync: zero-insert prolongé — sync delta réussie mais 0 nouveau match sur N cycles consécutifs",
				"gamertag", p.Gamertag,
				"xuid", p.XUID,
				"consecutive_zero_inserts", detail.ConsecutiveZeroInserts,
				"threshold", ConsecutiveZeroInsertWarnThreshold,
				"hint", "vérifier endpoint Halo + format URL + token resolved (probe /api/v1/_diag/auto-sync/probe)",
			)
		}
	}

	if len(syncResult.Errors) > 0 {
		detail.FirstError = syncResult.Errors[0]
		detail.Reason = "sync terminée avec erreurs partielles"
		slog.WarnContext(ctx, "auto_sync: sync terminée avec erreurs partielles",
			"gamertag", p.Gamertag,
			"inserted", syncResult.MatchesInserted,
			"skipped", syncResult.MatchesSkipped,
			"error_count", len(syncResult.Errors),
			"first_error", syncResult.Errors[0],
			"duration_s", syncResult.DurationSeconds,
			"status", syncResult.Status())
	} else {
		if syncResult.MatchesInserted > 0 {
			detail.Reason = "sync delta réussie"
		} else {
			detail.Reason = "sync delta réussie — 0 nouveau match (déjà à jour)"
		}
		slog.InfoContext(ctx, "auto_sync: sync delta réussie",
			"gamertag", p.Gamertag,
			"inserted", syncResult.MatchesInserted,
			"skipped", syncResult.MatchesSkipped,
			"medals_inserted", syncResult.MedalsInserted,
			"duration_s", syncResult.DurationSeconds,
			"status", syncResult.Status())
	}

	// Convergence backfill events (bornée) : rattrape les highlight_events
	// manquants (matchs importés OpenSpartan, films retardés / 404 transitoires).
	// S'exécute SOUS le claim déjà tenu (gate=nil → pas de re-claim) après la sync
	// delta réussie. Best-effort : n'affecte jamais l'outcome du tick.
	s.runEventsConvergencePass(ctx, p.Gamertag, p.XUID)

	outcome = outcomeOK
	return outcome
}

// checkSyncPreconditions vérifie les 4 préconditions de sync (pool initialisé,
// joueur dans pool, watcher inactif, DB présente). Retourne (raison_skip, false)
// si une précondition échoue, sinon ("", true).
func (s *AutoSyncScheduler) checkSyncPreconditions(ctx context.Context, p domain.PlayerSummary) (string, bool) {
	if s.pool == nil {
		slog.InfoContext(ctx, "auto_sync: pool nil, joueur ignoré", "gamertag", p.Gamertag)
		return "pool de tokens non initialisé (aucun credential découvert au boot)", false
	}
	if !s.pool.HasPlayer(p.Gamertag) {
		slog.InfoContext(ctx, "auto_sync: joueur absent du pool, ignoré",
			"gamertag", p.Gamertag,
			"hint", "authentifier le joueur (SSO Xbox) ou `go run ./cmd/token-capture/ <GT>`",
		)
		return "joueur absent du pool (aucun refresh token dans data/auth/watcher_tokens)", false
	}
	if s.ActivityChecker != nil && s.ActivityChecker.IsPlayerActive(p.Gamertag) {
		slog.InfoContext(ctx, "auto_sync: watcher actif sur ce joueur — tick cédé",
			"gamertag", p.Gamertag,
		)
		return "watcher actif (Watching/Syncing/Cooling) — tick cédé", false
	}
	// Slug porté par le profil (db_profiles.json title-scoped), fallback DefaultSlug.
	// MT-11 / PMT-3 : la dette « le moteur écrit encore en DefaultSlug » est RÉSORBÉE —
	// syncPlayer pose le slug dans le ctx → BuildEngine construit via
	// NewSyncEngineForTitle. La précondition os.Stat utilise le même helper.
	slug := resolveTitleSlug(p)
	// Les titres live-only (Halo 5+) écrivent en shared-only (pas de player DB ;
	// le shared est provisionné au boot via provisionAdditionalActiveTitles). La
	// précondition « player DB présente » ne s'applique donc pas — sinon ces
	// joueurs seraient skippés à jamais (os.Stat échoue toujours).
	if livesync.HandlesTitle(slug) {
		return "", true
	}
	dbPath := titlePkg.NewPathResolver(s.cfg.RepoRoot).PlayerDBPath(slug, p.Gamertag)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		slog.InfoContext(ctx, "auto_sync: DB joueur absente, joueur ignoré",
			"gamertag", p.Gamertag, "title_slug", slug, "db_path", dbPath,
			"hint", "lancer la sync initiale via POST /sync/initial pour créer la DB",
		)
		return "DB joueur absente — sync initiale jamais effectuée", false
	}
	return "", true
}
