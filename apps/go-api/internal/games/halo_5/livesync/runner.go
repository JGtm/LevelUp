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
type Deps struct {
	NewSource   func(ctx context.Context) (halo5.CaptureSource, error)
	Capture     CaptureFunc
	Viewer      canonical.PlayerIdentity
	ResolveXUID func(gamertag string) string
	LoadKnown   func(ctx context.Context) (map[string]bool, error)
	PersistAll  func(ctx context.Context, batches []*persist.MatchBatch) (done []*persist.MatchBatch, errs []string)
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

	batches, stats, err := r.deps.Capture(ctx, src, r.deps.Viewer, r.deps.ResolveXUID,
		mapContains(known), halo5.CaptureOptions{MaxMatches: opts.MaxMatches})
	if err != nil {
		res.AddError("h5 collect: " + err.Error())
	}

	done, errs := r.deps.PersistAll(ctx, batches)
	res.Errors = append(res.Errors, errs...)
	tallyResult(&res, done, stats)

	r.logger.InfoContext(ctx, "h5 sync: cycle terminé",
		"gamertag", r.deps.Viewer.Gamertag,
		"seen", stats.MatchesSeen, "collected", stats.MatchesCollected, "inserted", res.MatchesInserted,
		"skipped", res.MatchesSkipped, "events_failed", stats.EventsFailed,
		"carnage_failed", stats.CarnageFailed, "status", res.Status())
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
