// Package v2 — cycle.go : implémentation concrète du CycleOrchestrator
// qui compose les 6 phases (D6.1 du plan ADR 0027).
//
// Compose RunDiscovery → RunDedup → RunFetchShared → RunFetchPlayer →
// RunPersist → RunPostSync avec logging structuré par phase, métriques
// expvar, et propagation des erreurs partielles dans CycleResult.
//
// Cette implémentation est testable en isolation avec des mocks pour
// chacune des 6 dépendances (KnownLoader, MatchListProvider,
// SharedMatchFetcher, PlayerEnrichmentFetcher, CycleBatchPersister,
// PostSyncRunner). Les adapters V1-bridge sont livrés en D6.2 à D6.4.
package v2

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
)

// CycleConfig regroupe les paramètres tunables de l'orchestrator.
//
// Valeurs par défaut (si 0) :
//   - FetchSharedParallelism      : 8  (limite errgroup Phase 3)
//   - FetchPlayerParallelism      : 4  (limite errgroup intra-player Phase 4)
//   - PostSyncParallelism         : 0  (illimité — len(players))
type CycleConfig struct {
	FetchSharedParallelism int
	FetchPlayerParallelism int
	PostSyncParallelism    int
}

// CycleOrchestratorImpl est l'implémentation concrète qui enchaîne les
// 6 phases. Construit avec NewCycleOrchestrator en injectant les 6
// dépendances + config.
type CycleOrchestratorImpl struct {
	loader         KnownLoader
	listProvider   MatchListProvider
	sharedFetcher  SharedMatchFetcher
	playerFetcher  PlayerEnrichmentFetcher
	persister      CycleBatchPersister
	postSyncRunner PostSyncRunner
	cfg            CycleConfig
	// snapshotProducer (optionnel) fige une version immuable en fin de cycle
	// (Phase 6bis). nil = aucun snapshot émis (nil-guard explicite, jamais de panic —
	// contrairement aux 6 dépendances obligatoires injectées par NewCycleOrchestrator).
	snapshotProducer SnapshotProducer
	// prestigeHook (optionnel) ré-évalue les défis Prestige actifs après le post-sync
	// de chaque joueur (Phase 6). Le chemin V2 n'appelle PAS engine.run() (il passe par
	// RunPostSyncForV2 directement), le hook engine ne fire donc jamais ici : il est
	// invoqué explicitement après RunPostSync, hors fenêtre RW (write-lease relâché en
	// fin de Phase 6), par PlayerSlug (= user_id des défis). nil = no-op.
	prestigeHook func(ctx context.Context, playerSlug, titleSlug string)
}

// WithSnapshotProducer câble le producteur de snapshot (Phase 6bis), optionnel et
// best-effort. Retourne l'orchestrator pour chaînage. Laisser non appelé désactive
// proprement la production de snapshot.
func (o *CycleOrchestratorImpl) WithSnapshotProducer(p SnapshotProducer) *CycleOrchestratorImpl {
	o.snapshotProducer = p
	return o
}

// WithPrestigeHook câble le hook Prestige post-sync (best-effort), invoqué en
// Phase 6 après le post-sync de chaque joueur (V2 ne passe pas par engine.run, cf.
// champ prestigeHook). Retourne l'orchestrator pour chaînage. Laisser non appelé
// désactive proprement la ré-évaluation Prestige côté V2 (nil-guard).
func (o *CycleOrchestratorImpl) WithPrestigeHook(hook func(ctx context.Context, playerSlug, titleSlug string)) *CycleOrchestratorImpl {
	o.prestigeHook = hook
	return o
}

// NewCycleOrchestrator construit un orchestrator avec les 6 dépendances
// injectées. Aucune n'est optionnelle : passer nil produit un panic au
// premier call (intentionnel, le câblage doit être complet ou ne pas
// activer V2 du tout via shouldUseV2).
//
// cfg avec valeurs zéro → defaults raisonnables (cf. CycleConfig).
func NewCycleOrchestrator(
	loader KnownLoader,
	listProvider MatchListProvider,
	sharedFetcher SharedMatchFetcher,
	playerFetcher PlayerEnrichmentFetcher,
	persister CycleBatchPersister,
	postSyncRunner PostSyncRunner,
	cfg CycleConfig,
) *CycleOrchestratorImpl {
	if cfg.FetchSharedParallelism == 0 {
		cfg.FetchSharedParallelism = 8
	}
	if cfg.FetchPlayerParallelism == 0 {
		cfg.FetchPlayerParallelism = 4
	}
	// PostSyncParallelism = 0 → illimité, c'est le comportement voulu
	return &CycleOrchestratorImpl{
		loader:         loader,
		listProvider:   listProvider,
		sharedFetcher:  sharedFetcher,
		playerFetcher:  playerFetcher,
		persister:      persister,
		postSyncRunner: postSyncRunner,
		cfg:            cfg,
	}
}

// Run exécute un cycle V2 complet. Implémente l'interface
// CycleOrchestrator du fichier orchestrator.go.
//
// Séquence stricte (sans parallélisme entre phases — chaque phase
// dépend du résultat de la précédente) :
//
//  1. Discovery   — parallèle N joueurs, read-only
//  2. Dedup       — single, pure function
//  3. FetchShared — errgroup(FetchSharedParallelism)
//  4. FetchPlayer — parallèle par joueur, errgroup interne
//  5. Persist     — single writer
//  6. PostSync    — parallèle N joueurs
//
// Erreurs : capturées par phase dans CycleResult, l'orchestrator ne
// retourne err != nil qu'en cas d'échec global non récupérable (ctx
// annulé, panic). Les erreurs par-joueur ou par-match sont collectées
// dans PerPlayer et les Warnings.
//
// Logging structuré avec sub-events par phase (event="sync.v2.cycle"
// + phase=...).
func (o *CycleOrchestratorImpl) Run(
	ctx context.Context,
	players []PlayerProfile,
) (CycleResult, error) {
	start := time.Now()
	res := CycleResult{
		StartedAt:      start,
		PerPlayer:      make(map[string]PlayerOutcome, len(players)),
		PhaseDurations: make(map[string]time.Duration, 6),
	}
	if len(players) == 0 {
		res.Duration = time.Since(start)
		slog.InfoContext(ctx, "sync.v2: cycle skipped — pas de joueurs", "event", "sync.v2.cycle.skip")
		return res, nil
	}

	// Préparer le lookup PlayerSlug → PlayerProfile pour Phase 3.
	playerBySlug := make(map[string]PlayerProfile, len(players))
	for _, p := range players {
		playerBySlug[p.PlayerSlug] = p
	}

	slog.InfoContext(ctx, "sync.v2: cycle démarré",
		"event", "sync.v2.cycle.start",
		"players", len(players),
	)
	observability.IncCounterT(ctxkeys.TitleSlug(ctx), "sync_v2_cycle_total")

	// ─── Phase 1 — Discovery ────────────────────────────────────────────
	disc, err := RunDiscovery(ctx, players, o.loader, o.listProvider)
	res.PhaseDurations[PhaseDiscovery] = disc.Duration
	observability.RecordDurationMST(ctxkeys.TitleSlug(ctx), "sync_v2_phase_duration_ms_discovery", disc.Duration.Milliseconds())
	if err != nil {
		res.Duration = time.Since(start)
		observability.IncCounterT(ctxkeys.TitleSlug(ctx), "sync_v2_cycle_error_discovery")
		return res, fmt.Errorf("phase discovery: %w", err)
	}
	slog.InfoContext(ctx, "sync.v2: phase discovery terminée",
		"event", "sync.v2.cycle.phase",
		"phase", PhaseDiscovery,
		"duration_ms", disc.Duration.Milliseconds(),
		"errors", len(disc.Errors),
	)

	// Capturer les erreurs Discovery dans PerPlayer.
	for slug, errVal := range disc.Errors {
		o.markFailed(res.PerPlayer, slug, playerBySlug, "discovery", errVal)
	}

	// ─── Phase 2 — Dedup ────────────────────────────────────────────────
	dedup := RunDedup(disc)
	res.PhaseDurations[PhaseDedup] = dedup.Duration
	res.UniqueMatches = len(dedup.UniqueMatches)
	observability.RecordDurationMST(ctxkeys.TitleSlug(ctx), "sync_v2_phase_duration_ms_dedup", dedup.Duration.Milliseconds())
	observability.AddIntT(ctxkeys.TitleSlug(ctx), "sync_v2_unique_matches_total", int64(len(dedup.UniqueMatches)))
	slog.InfoContext(ctx, "sync.v2: phase dedup terminée",
		"event", "sync.v2.cycle.phase",
		"phase", PhaseDedup,
		"unique_matches", len(dedup.UniqueMatches),
	)

	// ─── Phase 3 — Fetch shared ─────────────────────────────────────────
	fetched, err := RunFetchShared(ctx, dedup, playerBySlug, o.sharedFetcher, o.cfg.FetchSharedParallelism)
	res.PhaseDurations[PhaseFetchShared] = fetched.Duration
	observability.RecordDurationMST(ctxkeys.TitleSlug(ctx), "sync_v2_phase_duration_ms_fetch_shared", fetched.Duration.Milliseconds())
	observability.AddIntT(ctxkeys.TitleSlug(ctx), "sync_v2_fetch_shared_errors_total", int64(len(fetched.Errors)))
	if err != nil {
		res.Duration = time.Since(start)
		observability.IncCounterT(ctxkeys.TitleSlug(ctx), "sync_v2_cycle_error_fetch_shared")
		return res, fmt.Errorf("phase fetch_shared: %w", err)
	}
	slog.InfoContext(ctx, "sync.v2: phase fetch_shared terminée",
		"event", "sync.v2.cycle.phase",
		"phase", PhaseFetchShared,
		"duration_ms", fetched.Duration.Milliseconds(),
		"fetched", len(fetched.Matches),
		"errors", len(fetched.Errors),
	)

	// ─── Phase 4 — Fetch per-player ─────────────────────────────────────
	enrichments, err := RunFetchPlayer(ctx, players, dedup, o.playerFetcher, o.cfg.FetchPlayerParallelism)
	res.PhaseDurations[PhaseFetchPlayer] = enrichments.Duration
	observability.RecordDurationMST(ctxkeys.TitleSlug(ctx), "sync_v2_phase_duration_ms_fetch_player", enrichments.Duration.Milliseconds())
	if err != nil {
		res.Duration = time.Since(start)
		observability.IncCounterT(ctxkeys.TitleSlug(ctx), "sync_v2_cycle_error_fetch_player")
		return res, fmt.Errorf("phase fetch_player: %w", err)
	}
	slog.InfoContext(ctx, "sync.v2: phase fetch_player terminée",
		"event", "sync.v2.cycle.phase",
		"phase", PhaseFetchPlayer,
		"duration_ms", enrichments.Duration.Milliseconds(),
		"enriched_players", len(enrichments.Enrichments),
	)

	// ─── Phase 5 — Persist ──────────────────────────────────────────────
	persistRes := RunPersist(ctx, fetched, enrichments, playerBySlug, o.persister)
	res.PhaseDurations[PhasePersist] = persistRes.Duration
	observability.RecordDurationMST(ctxkeys.TitleSlug(ctx), "sync_v2_phase_duration_ms_persist", persistRes.Duration.Milliseconds())
	observability.AddIntT(ctxkeys.TitleSlug(ctx), "sync_v2_matches_persisted_total", int64(persistRes.MatchesPersisted))
	slog.InfoContext(ctx, "sync.v2: phase persist terminée",
		"event", "sync.v2.cycle.phase",
		"phase", PhasePersist,
		"duration_ms", persistRes.Duration.Milliseconds(),
		"matches_persisted", persistRes.MatchesPersisted,
		"enrichments_persisted", persistRes.EnrichmentsPersisted,
		"err", errorMsg(persistRes.Err),
	)
	if persistRes.Err != nil {
		// Persist échec global → on continue pas vers post-sync (rien
		// à post-syncher si rien n'a été écrit).
		res.Duration = time.Since(start)
		observability.IncCounterT(ctxkeys.TitleSlug(ctx), "sync_v2_cycle_error_persist")
		for _, p := range players {
			o.markFailed(res.PerPlayer, p.PlayerSlug, playerBySlug, "persist", persistRes.Err)
		}
		return res, fmt.Errorf("phase persist: %w", persistRes.Err)
	}

	// Mettre à jour PerPlayer avec les compteurs Phase 1-5.
	for _, p := range players {
		o.populateOutcome(res.PerPlayer, p, playerBySlug, disc, dedup)
	}

	// ─── Phase 6 — Post-sync ────────────────────────────────────────────
	// Calcule insertedByPlayer depuis fetched + dedup pour cibler les heals
	// weapon_kills/dominance sur les nouveaux matchs.
	insertedByPlayer := make(map[string][]string, len(players))
	for mID := range fetched.Matches {
		for _, slug := range dedup.ParticipantsByMatch[mID] {
			insertedByPlayer[slug] = append(insertedByPlayer[slug], mID)
		}
	}
	postRes, err := RunPostSync(ctx, players, o.postSyncRunner, o.cfg.PostSyncParallelism, insertedByPlayer)
	res.PhaseDurations[PhasePostSync] = postRes.Duration
	observability.RecordDurationMST(ctxkeys.TitleSlug(ctx), "sync_v2_phase_duration_ms_post_sync", postRes.Duration.Milliseconds())
	if err != nil {
		res.Duration = time.Since(start)
		observability.IncCounterT(ctxkeys.TitleSlug(ctx), "sync_v2_cycle_error_post_sync")
		return res, fmt.Errorf("phase post_sync: %w", err)
	}
	slog.InfoContext(ctx, "sync.v2: phase post_sync terminée",
		"event", "sync.v2.cycle.phase",
		"phase", PhasePostSync,
		"duration_ms", postRes.Duration.Milliseconds(),
	)

	// Merger les compteurs post-sync dans PerPlayer.
	for slug, pr := range postRes.PerPlayer {
		o.mergePostSyncOutcome(res.PerPlayer, slug, pr)
	}

	// ─── Hook Prestige (post-sync V2) ───────────────────────────────────
	// V2 ne passe pas par engine.run() (RunPostSyncForV2 appelle directement
	// runPostSyncPipeline), donc le hook engine ne fire jamais ici : on ré-évalue
	// explicitement les défis actifs par joueur. Hors fenêtre RW (write-lease shared
	// relâché en fin de RunPostSync) → instance directe non-lease, aucun deadlock
	// (invariant wire/prestige_setup.go). Best-effort : RunPostSync (côté hook) log et
	// n'échoue jamais le cycle. Identifiant = PlayerSlug (= user_id des défis).
	if o.prestigeHook != nil {
		for _, p := range players {
			pr, ok := postRes.PerPlayer[p.PlayerSlug]
			if !ok || pr.Err != nil {
				continue // post-sync joueur en échec → pas de ré-éval sur snapshot incohérent
			}
			o.prestigeHook(ctx, p.PlayerSlug, p.TitleSlug)
		}
	}

	// ─── Phase 6bis — Cut snapshot immuable ─────────────────────────────
	// Hors fenêtre RW (le write-lease shared est relâché à la fin de RunPostSync) →
	// les COPY de lecture ne stallent jamais. Best-effort + inconditionnel : un échec
	// de cut n'invalide pas le cycle, aucun flag ne laisse la feature à moitié OFF.
	o.cutSnapshot(ctx, players)

	res.Duration = time.Since(start)
	observability.RecordDurationMST(ctxkeys.TitleSlug(ctx), "sync_v2_cycle_duration_ms", res.Duration.Milliseconds())
	observability.IncCounterT(ctxkeys.TitleSlug(ctx), "sync_v2_cycle_success")

	// Observabilité : remonter explicitement la raison (FirstError) de chaque
	// joueur en échec. Sans ça, le détail par-joueur était silencieusement avalé
	// dans le snapshot diag (players:[]) et un échec récurrent restait invisible
	// dans les logs (incident 2026-06-01 : 3 joueurs "failed" sans aucune trace).
	for slug, out := range res.PerPlayer {
		if out.Status == "failed" {
			slog.WarnContext(ctx, "sync.v2: joueur en échec",
				"event", "sync.v2.cycle.player_failed",
				"player", slug,
				"gamertag", out.Gamertag,
				"reason", out.FirstError,
			)
		}
	}

	slog.InfoContext(ctx, "sync.v2: cycle terminé",
		"event", "sync.v2.cycle.done",
		"duration_ms", res.Duration.Milliseconds(),
		"unique_matches", res.UniqueMatches,
		"matches_persisted", persistRes.MatchesPersisted,
	)
	return res, nil
}

// markFailed positionne PerPlayer[slug] en status failed avec FirstError.
func (o *CycleOrchestratorImpl) markFailed(
	per map[string]PlayerOutcome,
	slug string,
	byProfile map[string]PlayerProfile,
	phase string,
	err error,
) {
	out := per[slug]
	if p, ok := byProfile[slug]; ok {
		out.Gamertag = p.Gamertag
		out.XUID = p.XUID
	}
	out.Status = "failed"
	if out.FirstError == "" {
		out.FirstError = fmt.Sprintf("%s: %v", phase, err)
	}
	per[slug] = out
}

// populateOutcome remplit les compteurs Phase 1-5 d'un joueur, sans
// écraser un status "failed" déjà positionné par markFailed.
func (o *CycleOrchestratorImpl) populateOutcome(
	per map[string]PlayerOutcome,
	p PlayerProfile,
	byProfile map[string]PlayerProfile,
	disc DiscoveryResult,
	dedup DedupResult,
) {
	out := per[p.PlayerSlug]
	out.Gamertag = p.Gamertag
	out.XUID = p.XUID
	if out.Status == "" {
		out.Status = "ok"
	}
	out.MatchesUnknown = len(disc.PerPlayer[p.PlayerSlug])
	// Compter les matchs assignés à ce joueur comme canonical fetcher,
	// pondéré : son MatchesInserted = matchs où ses participants
	// existent (proxy correct pour V2 où on persist en batch).
	for _, mID := range disc.PerPlayer[p.PlayerSlug] {
		participants := dedup.ParticipantsByMatch[mID]
		for _, s := range participants {
			if s == p.PlayerSlug {
				out.MatchesInserted++
				break
			}
		}
	}
	per[p.PlayerSlug] = out
	_ = byProfile // signature consistante, byProfile peut servir future
}

// mergePostSyncOutcome ajoute les warnings + propage l'erreur post-sync
// vers PerPlayer si applicable.
func (o *CycleOrchestratorImpl) mergePostSyncOutcome(
	per map[string]PlayerOutcome,
	slug string,
	pr PlayerPostSyncResult,
) {
	out := per[slug]
	out.Warnings = append(out.Warnings, pr.Warnings...)
	if pr.Err != nil {
		if out.Status == "ok" {
			out.Status = "partial" // post-sync échoué mais le reste OK
		}
		if out.FirstError == "" {
			out.FirstError = fmt.Sprintf("post_sync: %v", pr.Err)
		}
	}
	per[slug] = out
}

// cutSnapshot déclenche la Phase 6bis : production d'une version de snapshot immuable
// pour le titre courant à partir des joueurs du cycle. nil-guard explicite (producteur
// optionnel) ; échec loggé en WARN sans propagation (best-effort).
func (o *CycleOrchestratorImpl) cutSnapshot(ctx context.Context, players []PlayerProfile) {
	if o.snapshotProducer == nil {
		return
	}
	slug := ctxkeys.TitleSlug(ctx)
	gamertags := make([]string, 0, len(players))
	for _, p := range players {
		if p.Gamertag != "" {
			gamertags = append(gamertags, p.Gamertag)
		}
	}
	if err := o.snapshotProducer.CutSnapshot(ctx, slug, gamertags); err != nil {
		// module=snapshot → logs/snapshot.log (diagnostic centralisé du sous-système,
		// le package v2 routerait sinon vers logs/general.log).
		slog.WarnContext(ctx, "sync.v2: cut snapshot échoué (best-effort)",
			"module", "snapshot", "event", "sync.v2.snapshot.error", "err", err, "titleSlug", slug)
	}
}

func errorMsg(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
