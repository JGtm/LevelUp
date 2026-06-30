// Package livesync — câblage du sync LIVE Halo 5 (tranche 3).
//
// Orchestration du runner : fetch (capture h5 : matchs + events + carnage) →
// persist (shared DB du titre) → domain.SyncResult. INDEPENDANTE de l'acquisition
// du lease/DB (injectée via Deps) → runner unit-testable sans réseau ni DuckDB.
//
// Ne touche JAMAIS le moteur Infinite (SyncEngine.run) : les deux chemins
// convergent sur persist.MatchBatch + SharedPersister. Le runner REMPLACE le
// SyncEngine pour les couples du titre Halo 5 (sélection au BuildEngine/newEngineFor
// par le registre — jamais slug==), l'adapter Infinite reste byte-identique.
//
// Discipline lease : on tient le write-lease COURT (known-set, puis persist) et
// JAMAIS pendant le fetch réseau (CollectRecentMatches) — c'est Deps qui orchestre
// l'acquisition autour des deux moments.
package livesync

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/persist"
)

// CaptureFunc : signature de halo5.CollectRecentMatches (injectable → fake en test).
type CaptureFunc func(
	ctx context.Context,
	src halo5.CaptureSource,
	viewer canonical.PlayerIdentity,
	resolveXUID func(gamertag string) string,
	isKnown func(matchID string) bool,
	opts halo5.CaptureOptions,
) ([]*persist.MatchBatch, halo5.CaptureStats, error)

// Deps regroupe les dépendances injectées du runner (toutes mockables) :
//   - NewSource   : source live h5 depuis le contexte (SpartanToken). re-auth → erreur.
//   - Capture     : la collecte (fetch → []MatchBatch). nil → halo5.CollectRecentMatches.
//   - Viewer      : joueur consulté (self), owner du batch. XUID = xuid Xbox RÉSOLU.
//   - ResolveXUID : gamertag → xuid Xbox ("" toléré).
//   - LoadKnown   : match_ids déjà persistés (delta-stop). Lease COURT, relâché AVANT
//     le fetch réseau. nil → pas de delta (collecte complète, idempotente).
//   - PersistAll  : persiste les batches sous UN lease (acquis APRÈS le fetch) et
//     retourne les batches réellement écrits + les erreurs.
//   - PersistCSR  : hook OPTIONNEL post-sync (G4 Phase 1) — fetch le service record
//     arena via la source DÉJÀ construite (pas de 2e auth) → persiste le CSR par
//     playlist dans la player DB h5 (player_csr_snapshots, saison lifetime). nil →
//     pas de persistance CSR (runner unit-testable sans player DB). Best-effort :
//     une erreur n'avorte JAMAIS le cycle (remontée dans le SyncResult).
type Deps struct {
	NewSource  func(ctx context.Context) (halo5.CaptureSource, error)
	Capture    CaptureFunc
	Viewer     canonical.PlayerIdentity
	Resolver   func(ctx context.Context) XUIDResolver // LAZY (PeopleHub fait de l'auth) ; nil → tout ""
	LoadKnown  func(ctx context.Context) (map[string]bool, error)
	PersistAll func(ctx context.Context, batches []*persist.MatchBatch) (done []*persist.MatchBatch, errs []string)
	PersistCSR func(ctx context.Context, src halo5.CaptureSource) (int, error)
	// PostScore : hook OPTIONNEL post-sync appelé APRÈS persist quand des matchs ont
	// été insérés → reconstruit l'enrichment PAR JOUEUR (sessions/perf/engagement/
	// dominance/is_with_friends + LUSR) sur les nouveaux matchs, en INCRÉMENTAL
	// (force=false), comme runScoringSteps Infinite + écrit le CSR par match classé
	// (CurrentCsr du carnage, via src). Rend le sync live h5 autonome (plus besoin
	// d'un cmd/h5-enrich/h5-csr-match manuel après chaque sync). Best-effort : une
	// erreur n'avorte JAMAIS le cycle. nil → runner unit-testable sans player DB.
	PostScore func(ctx context.Context, src halo5.CaptureSource, insertedMatchIDs []string) error
	// HasEnrichmentBacklog : sonde OPTIONNELLE « reste-t-il des matchs en shared SANS
	// enrichment par-joueur ? » (CountSharedMatchesMissingEnrichment côté sync,
	// title-agnostic). Sert le FILET DE CONVERGENCE à 0 insert : un match du titre
	// inséré en shared par le sync d'un coéquipier (delta-skip chez ce joueur) doit
	// quand même déclencher PostScore — BackfillEnrichmentFromShared est un reconcile
	// full-shared (ensurePlayerEnrichmentRows). Mirroir de hasConvergenceBacklog
	// Infinite (engine_postsync.go). Garde bon-marché : appelée UNIQUEMENT à 0 insert.
	// nil → comportement strict insert>0 (runner unit-testable sans DuckDB).
	HasEnrichmentBacklog func(ctx context.Context) bool
	// RunAchievements : hook OPTIONNEL post-sync (best-effort) qui synchronise les
	// achievements Xbox du joueur pour ce titre — équivalent du runAchievementsSync
	// Infinite (engine_postsync.go), appelé à CHAQUE post-sync (les achievements ne
	// dépendent pas de l'insertion de matchs). RÉUTILISE le chemin title-aware
	// existant (SyncEngineForTitle.RunAchievementsOnly), gaté sur la capability
	// achievements du titre au CÂBLAGE (hook nil si absente). Best-effort : une
	// erreur n'avorte JAMAIS le cycle. nil → runner unit-testable sans réseau Xbox.
	RunAchievements func(ctx context.Context) error
	// NotifyFirstSync : hook OPTIONNEL (MT-19 / axe E) appelé APRÈS persist quand le
	// titre a des matchs (1er sync OU steady-state). Best-effort, HORS pipeline
	// progression/prestige (qui reste Infinite-only). L'idempotence DURABLE (une
	// seule notif « titre prêt » par titre, ré-essayée jusqu'au succès) est garantie
	// par l'impl (watermark sync_meta), PAS par ce hook. nil → runner unit-testable
	// sans notifications (no-op).
	NotifyFirstSync func(ctx context.Context, inserted int)
}

// Runner implémente scheduler.DeltaRunner (RunDelta) pour Halo 5.
type Runner struct {
	deps   Deps
	logger *slog.Logger
}

// NewRunner construit le runner. Capture nil → halo5.CollectRecentMatches.
func NewRunner(deps Deps, logger *slog.Logger) *Runner {
	if deps.Capture == nil {
		deps.Capture = halo5.CollectRecentMatches
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Runner{deps: deps, logger: logger}
}

// RunDelta exécute un cycle de sync delta Halo 5. Dégradation gracieuse : token
// absent / known-set KO / erreur de fetch n'avortent pas le cycle (best-effort,
// erreurs remontées dans le SyncResult). opts.MaxMatches borne la collecte.
func (r *Runner) RunDelta(ctx context.Context, opts domain.SyncOptions) (domain.SyncResult, error) {
	var res domain.SyncResult

	src, err := r.deps.NewSource(ctx)
	if err != nil {
		// Token absent/expiré = signal de re-auth, pas une panne dure : remontée
		// dans le résultat (Status=failure) sans casser le cycle global.
		r.logger.WarnContext(ctx, "h5 sync: source live indisponible (re-auth ?)",
			"gamertag", r.deps.Viewer.Gamertag, "err", err)
		res.AddError("h5 source: " + err.Error())
		return res, nil
	}

	var known map[string]bool
	if r.deps.LoadKnown != nil {
		if known, err = r.deps.LoadKnown(ctx); err != nil {
			// Best-effort : sans known-set on collecte sans delta-stop (idempotence
			// garantie en aval par match_registry — un re-collect est no-opé).
			r.logger.WarnContext(ctx, "h5 sync: known-set indisponible (collecte sans delta)", "err", err)
			res.AddError("h5 known-set: " + err.Error())
			known = nil
		}
	}

	// Resolver bâti LAZY ici (auth PeopleHub) — pas au câblage (hot path). nil-safe.
	var resolver XUIDResolver
	if r.deps.Resolver != nil {
		resolver = r.deps.Resolver(ctx)
	}
	resolveXUID := ResolveXUIDClosure(ctx, resolver, r.logger)

	batches, stats, err := r.deps.Capture(ctx, src, r.deps.Viewer, resolveXUID,
		mapContains(known), halo5.CaptureOptions{MaxMatches: opts.MaxMatches})
	if err != nil {
		res.AddError("h5 collect: " + err.Error())
	}

	done, errs := r.deps.PersistAll(ctx, batches)
	res.Errors = append(res.Errors, errs...)
	tallyResult(&res, done, stats)

	// Hook CSR post-sync (G4 Phase 1) : persiste le CSR arena par playlist dans la
	// player DB h5. RÉUTILISE la source live déjà construite (zéro 2e auth).
	// Best-effort : une erreur n'avorte pas le cycle.
	if r.deps.PersistCSR != nil {
		if n, cerr := r.deps.PersistCSR(ctx, src); cerr != nil {
			r.logger.WarnContext(ctx, "h5 sync: persistance CSR échouée (non bloquant)",
				"gamertag", r.deps.Viewer.Gamertag, "err", cerr)
			res.AddError("h5 csr: " + cerr.Error())
		} else if n > 0 {
			r.logger.InfoContext(ctx, "h5 sync: CSR arena persisté",
				"gamertag", r.deps.Viewer.Gamertag, "playlists", n)
		}
	}

	// Hook post-score : enrichment PAR JOUEUR (+ LUSR) des matchs nouvellement
	// persistés, en incrémental. Best-effort (n'avorte jamais le cycle).
	//
	// FILET DE CONVERGENCE à 0 insert : on déclenche AUSSI quand un backlog
	// d'enrichment existe (matchs du titre insérés en shared par le sync d'un
	// coéquipier, delta-skippés chez ce joueur → jamais enrichis). Sans ça, le strict
	// insert>0 laissait ces matchs sans enrichment à durée indéterminée. La sonde
	// HasEnrichmentBacklog (CountSharedMatchesMissingEnrichment, title-agnostic) n'est
	// consultée qu'à 0 insert (garde bon-marché). Mirroir de hasConvergenceBacklog
	// Infinite + du pattern NotifyFirstSync ci-dessous (away-case).
	if r.deps.PostScore != nil && shouldRunPostScore(ctx, res.MatchesInserted, r.deps.HasEnrichmentBacklog) {
		if err := r.deps.PostScore(ctx, src, res.InsertedMatchIDs); err != nil {
			r.logger.WarnContext(ctx, "h5 sync: post-score enrichment échoué (non bloquant)",
				"gamertag", r.deps.Viewer.Gamertag, "err", err)
			res.AddError("h5 post-score: " + err.Error())
		}
	}

	// Achievements Xbox post-sync (parité Infinite : runAchievementsSync à CHAQUE
	// post-sync, gate capability au câblage). Indépendant de l'insertion de matchs
	// (les achievements évoluent hors match). Best-effort : une erreur n'avorte pas
	// le cycle. RÉUTILISE le chemin title-aware (SyncEngineForTitle.RunAchievementsOnly).
	if r.deps.RunAchievements != nil {
		if err := r.deps.RunAchievements(ctx); err != nil {
			r.logger.WarnContext(ctx, "h5 sync: achievements échoué (non bloquant)",
				"gamertag", r.deps.Viewer.Gamertag, "err", err)
			res.AddError("h5 achievements: " + err.Error())
		}
	}

	// Notif « titre prêt » (MT-19 / axe E) : best-effort, HORS progression/prestige
	// (qui reste Infinite-only). Déclenchée dès que le titre a des matchs (known-set
	// non vide OU insert>0 ce cycle) — PAS seulement quand known-set est vide : si la
	// 1re émission échoue (lease saturé), le watermark n'avance pas et la notif est
	// ré-essayée au cycle suivant (le known-set, lui, n'est plus vide). L'idempotence
	// durable (1 seule notif par titre) est garantie par le notifier (watermark).
	// Couvre l'away-case (scheduler + watcher) en un point unique du funnel RunDelta.
	if r.deps.NotifyFirstSync != nil && (len(known) > 0 || res.MatchesInserted > 0) {
		r.deps.NotifyFirstSync(ctx, res.MatchesInserted)
	}

	r.logger.InfoContext(ctx, "h5 sync: cycle terminé",
		"gamertag", r.deps.Viewer.Gamertag,
		"seen", stats.MatchesSeen, "collected", stats.MatchesCollected, "inserted", res.MatchesInserted,
		"skipped", res.MatchesSkipped, "events_failed", stats.EventsFailed,
		"carnage_failed", stats.CarnageFailed, "roster_dropped", stats.RosterDropped,
		"warzone", stats.ExcludedWarzone, "campaign", stats.ExcludedCampaign, "status", res.Status())
	return res, nil
}

// tallyResult agrège les compteurs du SyncResult depuis les batches RÉELLEMENT
// persistés (done) + les stats de collecte.
func tallyResult(res *domain.SyncResult, done []*persist.MatchBatch, stats halo5.CaptureStats) {
	res.MatchesInserted = len(done)
	for _, b := range done {
		if b.Shared.Match != nil {
			res.InsertedMatchIDs = append(res.InsertedMatchIDs, b.Shared.Match.MatchID)
		}
		res.ParticipantsDone += len(b.Shared.Participants)
		res.MedalsInserted += len(b.Shared.Medals)
		res.EventsInserted += len(b.Shared.HighlightEvents)
	}
	if s := stats.MatchesSeen - res.MatchesInserted; s > 0 {
		res.MatchesSkipped = s // vus mais non insérés (delta connus + échecs persist)
	}
}

// mapContains adapte un set de match_ids connus en prédicat isKnown (nil → rien
// de connu → collecte complète).
func mapContains(known map[string]bool) func(string) bool {
	return func(id string) bool { return known[id] }
}

// shouldRunPostScore décide si le hook post-score (enrichment par-joueur) doit
// tourner ce cycle. Pure (testable sans DuckDB) : déclenche dès qu'au moins un
// match a été inséré ; à 0 insert, consulte la sonde de backlog (filet de
// convergence — matchs insérés en shared par un coéquipier, jamais enrichis chez
// ce joueur). Sonde nil OU faux à 0 insert → ne rien faire (comportement strict).
func shouldRunPostScore(ctx context.Context, inserted int, hasBacklog func(context.Context) bool) bool {
	if inserted > 0 {
		return true
	}
	return hasBacklog != nil && hasBacklog(ctx)
}
