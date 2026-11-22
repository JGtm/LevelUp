// Package scheduler — auto_sync_run.go : le cycle de sync (RunOnce / RunOnceTrigger /
// syncPlayersConcurrent / syncPlayer / checkSyncPreconditions + le type syncOutcome). La boucle
// périodique vit dans auto_sync_loop.go. Extrait de auto_sync.go (K2c, 2026-07-06).
package scheduler

import (
	"context"
	"log/slog"
	"os"
	"sort"
	gosync "sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_5/livesync"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/platform/adminstate"
	"levelup/go-api/internal/port"
)

// RunOnce exécute un cycle de sync pour tous les joueurs configurés.
// Peut être appelé manuellement (debug, endpoint admin) sans attendre un tick.
func (s *AutoSyncScheduler) RunOnce(ctx context.Context) *RunOnceResult {
	return s.RunOnceTrigger(ctx, "manual")
}

// RunOnceTrigger est RunOnce avec la provenance du déclenchement ("tick" pour
// la boucle périodique, "manual" pour HTTP/diag/job) — tracée dans
// l'historique des cycles du dashboard monitoring.
//
// Jamais deux cycles à la fois : si un cycle tourne déjà, la demande est ignorée et journalisée, et le
// résultat rendu est vide (busy). Un cycle qui n'est pas le tick de la boucle de Run signale son issue à
// la boucle, qui recale alors sa cadence (replanFromCycle).
func (s *AutoSyncScheduler) RunOnceTrigger(ctx context.Context, trigger string) *RunOnceResult {
	if !s.cycleRunning.CompareAndSwap(false, true) {
		slog.InfoContext(ctx, "auto_sync: un cycle tourne déjà, demande ignorée", "trigger", trigger)
		return &RunOnceResult{busy: true}
	}
	defer s.cycleRunning.Store(false)
	res := s.runCycle(ctx, trigger)
	if trigger != adminstate.TriggerCron {
		s.signalReplan(res.debt)
	}
	return res
}

// runCycle exécute le cycle ; l'appelant tient le drapeau cycleRunning.
func (s *AutoSyncScheduler) runCycle(ctx context.Context, trigger string) *RunOnceResult {
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
	// Position de l'historique et cumul de films avant le cycle : la garde de progrès les compare à ceux d'après.
	watch := s.watchProgress(ctx, players)

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
			// runOnceV2 renvoie nil sur erreur globale : aucun joueur n'a pu dire où en est son historique.
			v2Res = &RunOnceResult{debt: cycleDebt{history: s.pendingHistoryOf(ctx, orchPlayers)}}
			s.recordFailedPasses(orchPlayers)
		}
		live := s.syncPlayersConcurrent(ctx, livePlayers)
		v2Res.debt.history = append(v2Res.debt.history, live.pending...)
		v2Res.debt.films = s.filmBacklogOf(players)
		v2Res.Synced += live.synced
		v2Res.Skipped += live.skipped
		v2Res.Failed += live.failed
		v2Res.addInserted(s.insertedSince(start))
		v2Res.debt.progressed = s.progressedSince(ctx, watch, players, v2Res)
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
	// Contrat (D-18, plan 8.2.2) : ce filet ne rattrape PAS l'historique de Halo Infinite. Le moteur V1
	// de syncPlayer est un delta pur (aucune lecture ni écriture de l'état de l'historique, jamais
	// d'historyPending) : le rattrapage appartient au cycle de l'orchestrator V2. Halo 5 rattrape
	// toujours, dans son runner.
	slog.WarnContext(ctx, "auto_sync: orchestrator V2 non câblé — sync directe via syncPlayer (filet de boot)",
		"player_count", res.Total)
	all := s.syncPlayersConcurrent(ctx, players)
	res.Synced, res.Skipped, res.Failed, res.debt.history = all.synced, all.skipped, all.failed, all.pending
	res.debt.films = s.filmBacklogOf(players)
	res.addInserted(s.insertedSince(start))
	res.debt.progressed = s.progressedSince(ctx, watch, players, res)

	// PLAN_SPARTAN_IDENTITY_REFACTOR §11 Phase 5 (2026-05-25) :
	// customRefresher.RefreshAll(ctx) supprimé. La customisation est désormais
	// rafraîchie en LIVE à chaque visite home (CareerLiveService.kickoff).

	res.Duration = time.Since(start)

	s.storeCycleResult(ctx, res, trigger, captureCycleLoad().deltaSince(loadBefore))
	s.notifyDiscordSyncCycle(ctx, trigger, start, res)

	return res
}

// playersTally agrège l'issue d'un lot de joueurs passés par syncPlayer.
type playersTally struct {
	synced, skipped, failed int
	// pending : gamertags dont l'historique reste incomplet après la passe (cadence de rattrapage), triés.
	pending []string
}

// syncPlayersConcurrent lance syncPlayer pour chaque joueur en parallèle (errgroup
// borné par la taille du pool de tokens) et agrège synced/skipped/failed. Extrait
// de la boucle de cycle (Phase 3.4) pour être réutilisé par le path V2 (joueurs
// live-only, non gérés par l'orchestrator) ET le path V1 (tous les joueurs).
//
// Les joueurs d'un titre live-only en rattrapage se partagent le budget du titre (planLiveCatchUp) : chacun
// reçoit sa part en opts.MaxMatches, et ceux que le budget ne couvre pas attendent le cycle suivant (ils
// restent en rattrapage, donc dans pending).
//
// Best-effort : un échec syncPlayer n'annule pas les autres goroutines (l'erreur
// est déjà loggée + reflétée dans le compteur failed). Safety : syncPlayer met à
// jour s.playerOutcomes via recordOutcome (protégé par s.snapshotMu) ; les writes
// shared sont sérialisés par dblease + singleflight ; les compteurs locaux sont
// protégés par atomic.Int32.
func (s *AutoSyncScheduler) syncPlayersConcurrent(ctx context.Context, players []domain.PlayerSummary) playersTally {
	var out playersTally
	if len(players) == 0 {
		return out
	}
	parallelism := s.poolSizeSafe()
	if parallelism < 1 {
		parallelism = 1
	}
	plan := s.planLiveCatchUp(ctx, players)
	var syncedN, skippedN, failedN atomic.Int32
	var pendingMu gosync.Mutex
	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(parallelism)
	for _, p := range players {
		p := p
		if plan.waits(p) {
			// Le budget de rattrapage du titre est réparti sans lui ce cycle : le joueur reste en rattrapage
			// et passera parmi les premiers au cycle suivant.
			skippedN.Add(1)
			pendingMu.Lock()
			out.pending = append(out.pending, p.Gamertag)
			pendingMu.Unlock()
			s.deferCatchUp(ctx, p)
			continue
		}
		opts := plan.optionsFor(p)
		eg.Go(func() error {
			outcome, hist := s.syncPlayerWithHistory(egCtx, p, opts)
			switch outcome {
			case outcomeOK:
				syncedN.Add(1)
			case outcomeSkipped:
				skippedN.Add(1)
			case outcomeFailed:
				failedN.Add(1)
			}
			if hist {
				pendingMu.Lock()
				out.pending = append(out.pending, p.Gamertag)
				pendingMu.Unlock()
			}
			return nil
		})
	}
	_ = eg.Wait()
	sort.Strings(out.pending)
	out.synced, out.skipped, out.failed = int(syncedN.Load()), int(skippedN.Load()), int(failedN.Load())
	return out
}

// syncOutcome encode le résultat d'une tentative de sync par joueur.
type syncOutcome int

const (
	outcomeOK      syncOutcome = iota // sync delta réussie
	outcomeSkipped                    // pas de token / watcher actif / DB absente → ignoré sans erreur
	outcomeFailed                     // erreur bloquante pendant RunDelta
)

// outcomeName est le nom d'une issue dans PlayerOutcomeDetail.Outcome.
func outcomeName(o syncOutcome) string {
	switch o {
	case outcomeOK:
		return "ok"
	case outcomeSkipped:
		return "skipped"
	case outcomeFailed:
		return statusFailed
	}
	return ""
}

// syncPlayer lance une sync delta pour un joueur, avec les options par défaut, et rend son issue (voir
// syncPlayerWithHistory).
func (s *AutoSyncScheduler) syncPlayer(ctx context.Context, p domain.PlayerSummary) syncOutcome {
	outcome, _ := s.syncPlayerWithHistory(ctx, p, domain.DefaultSyncOptions())
	return outcome
}

// syncPlayerWithHistory lance une sync delta pour un joueur via le pool de tokens et rend, avec
// son issue, si l'historique du joueur reste incomplet après la passe (cadence de rattrapage).
// Seul le runner Halo 5 le dit : le moteur V1 (Halo Infinite) est un delta pur et rend toujours faux
// (D-18) ; le rattrapage de Halo Infinite est celui du pipeline V2, hors de ce filet.
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
func (s *AutoSyncScheduler) syncPlayerWithHistory(ctx context.Context, p domain.PlayerSummary, opts domain.SyncOptions) (outcome syncOutcome, pending bool) {
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
	defer func() {
		detail.Outcome = outcomeName(outcome)
		detail.DurationMs = time.Since(startedAt).Milliseconds()
		s.recordOutcome(detail)
		s.recordPass(resolveTitleSlug(p), p.Gamertag, port.SyncPass{
			At: time.Now(), MatchesInserted: detail.MatchesInserted, Status: passStatusOfSync(outcome, detail),
		})
		// Une passe en échec n'a pas pu dire où en est l'historique : sans fin rangée, le cycle suivant
		// doit venir vite (cadence de rattrapage).
		if outcome == outcomeFailed {
			pending = s.historyIncomplete(ctx, p)
		}
	}()

	if skipReason, ok := s.checkSyncPreconditions(ctx, p); !ok {
		detail.Reason = skipReason
		outcome = outcomeSkipped
		return outcome, pending
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
			return outcome, pending
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
			return outcome, pending
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
			return outcome, pending
		}
	}

	syncResult, err := runner.RunDelta(ctx, opts)
	if err != nil {
		slog.ErrorContext(ctx, "auto_sync: RunDelta échoué",
			"gamertag", p.Gamertag, "err", err)
		detail.Reason = "RunDelta échoué: " + err.Error()
		detail.FirstError = err.Error()
		outcome = outcomeFailed
		return outcome, pending
	}

	pending = syncResult.HistoryPending
	detail.MatchesInserted = syncResult.MatchesInserted
	detail.CatchUpInserted = syncResult.CatchUpInserted
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
	return outcome, pending
}

// checkSyncPreconditions vérifie les préconditions de sync (pool initialisé, watcher
// inactif, DB présente). Retourne (raison_skip, false) si une précondition échoue,
// sinon ("", true).
//
// PAS de précondition « le joueur a son propre token » (D1, plan 2026-09-16) : le cycle
// sync par le POOL, et l'historique / les stats / les films / les CSR sont des endpoints
// publics que n'importe quel token du parc sert. Un profil suivi sans refresh token propre
// était auparavant sauté À VIE par le cycle alors que seul son rang de carrière lui est
// inaccessible — celui-ci se dégrade seul, par endpoint.
func (s *AutoSyncScheduler) checkSyncPreconditions(ctx context.Context, p domain.PlayerSummary) (string, bool) {
	if s.pool == nil {
		slog.InfoContext(ctx, "auto_sync: pool nil, joueur ignoré", "gamertag", p.Gamertag)
		return "pool de tokens non initialisé (aucun credential découvert au boot)", false
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
	// Une base joueur absente n'est plus un motif de saut : le moteur la crée et, sans match connu,
	// son delta récupère jusqu'au plafond (première synchronisation). Un profil
	// connecté après le démarrage se synchronise donc sans passer par l'assistant.
	dbPath := s.cfg.PathResolver().PlayerDBPath(slug, p.Gamertag)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		slog.InfoContext(ctx, "auto_sync: DB joueur absente — première synchronisation",
			"gamertag", p.Gamertag, "title_slug", slug, "db_path", dbPath)
	}
	return "", true
}
