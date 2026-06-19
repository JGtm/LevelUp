// Package sync — convergence_backfill_events.go : backfill BACKGROUND
// NON-BLOQUANT des highlight_events manquants (films), pour les matchs déjà en
// base mais sans events (typiquement importés depuis OpenSpartan — vanilla
// OpenSpartan ne décode pas les films).
//
// Invariants pour ne pas bloquer l'app (cf. contrainte B-swap DuckDB : RO+RW
// interdits sur le même fichier in-process) :
//   - le FETCH réseau (lent) se fait HORS de tout lease RW ;
//   - la PERSISTANCE se fait par LOTS courts (acquire → écrire le lot → release),
//     fenêtre RW de l'ordre de la dizaine de ms (K transactions rapides) ;
//   - YIELD entre les lots pour que le pool RO se rattache et que l'app lise ;
//   - CESSION au sync live via Coordinator.TryClaim (le watcher temps-réel n'est
//     jamais bloqué ; auto/HTTP coalescent).
//
// Convergent + reprenable : la détection (events_loaded=false) se fait une fois
// par passe ; les matchs en erreur réseau restent events_loaded=false et sont
// repris à la passe suivante. Auto-bornant : un film 404 sur un match plus vieux
// que la fenêtre de retry est marqué no-film définitif (sort du retry set).
//
// Tout est composé des briques existantes (fetchHighlightChunkResilient,
// analysis.ParseHighlightEvents, persistCombatCompletion, isNoFilmDefinitive,
// EventsCompletionPersister.MarkNoFilmDefinitive) — zéro logique dupliquée.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/persist"
)

const (
	defaultConvergenceChunkSize     = 15
	defaultConvergenceYieldInterval = 2 * time.Second
)

// EventsConvergenceConfig regroupe les dépendances et réglages du backfill events.
// AcquireShared est injecté (closure) plutôt que (provider, path) pour rester
// testable en DuckDB in-memory — même philosophie que persist.SharedWriterFn.
type EventsConvergenceConfig struct {
	// Client fetche les chunks film (hors lease). Interface étroite → mockable.
	Client highlightChunkFetcher
	// AcquireShared ouvre le writer RW shared avec lease et retourne un release
	// à appeler après CHAQUE lot (fenêtre courte). En prod : closure sur
	// AcquireSharedWriterStandalone(provider, path).
	AcquireShared func(ctx context.Context) (*sql.DB, func(), error)
	// PlayerDB sert à la détection (FindMatchesMissingData prend player + shared).
	PlayerDB *sql.DB
	// Coord (optionnel) : cession au sync live via TryClaim. nil = pas de cession.
	Coord    *Coordinator
	Gamertag string
	XUID     string

	// MaxMatches borne le nombre de matchs traités par passe (0 = tous les incomplets).
	MaxMatches int
	// ChunkSize : matchs par fenêtre RW (défaut 15). YieldInterval : pause entre
	// lots pour laisser l'app respirer (défaut 2s).
	ChunkSize     int
	YieldInterval time.Duration
}

// EventsConvergenceResult résume une passe de backfill.
type EventsConvergenceResult struct {
	Detected      int  // matchs incomplets détectés (events_loaded=false)
	EventsWritten int  // matchs ayant reçu des events (film présent)
	NoFilmFinal   int  // matchs marqués no-film définitif (404 + trop vieux)
	Skipped       int  // erreurs réseau/parse/anomalie → repris la passe suivante
	Ceded         bool // a cédé à un sync live en cours de route
}

// RunEventsConvergenceBackfill exécute UNE passe : détecte les matchs sans events
// (récent→vieux), puis les traite par lots non-bloquants. Idempotent et
// reprenable. Retourne le résumé. Une erreur n'est renvoyée que sur échec de
// détection (acquisition shared impossible) ou ctx annulé.
func RunEventsConvergenceBackfill(ctx context.Context, cfg EventsConvergenceConfig) (EventsConvergenceResult, error) {
	var res EventsConvergenceResult
	if cfg.Client == nil || cfg.AcquireShared == nil {
		return res, fmt.Errorf("RunEventsConvergenceBackfill: Client et AcquireShared requis")
	}
	chunkSize := cfg.ChunkSize
	if chunkSize <= 0 {
		chunkSize = defaultConvergenceChunkSize
	}
	yield := cfg.YieldInterval
	if yield <= 0 {
		yield = defaultConvergenceYieldInterval
	}

	matchIDs, err := cfg.detectIncomplete(ctx)
	if err != nil {
		return res, fmt.Errorf("RunEventsConvergenceBackfill detect: %w", err)
	}
	if cfg.MaxMatches > 0 && len(matchIDs) > cfg.MaxMatches {
		matchIDs = matchIDs[:cfg.MaxMatches]
	}
	res.Detected = len(matchIDs)
	if len(matchIDs) == 0 {
		return res, nil
	}
	slog.InfoContext(ctx, "convergence backfill events: démarrage",
		"gamertag", cfg.Gamertag, "detected", res.Detected, "chunk", chunkSize)

	for i := 0; i < len(matchIDs); i += chunkSize {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		end := min(i+chunkSize, len(matchIDs))
		chunk := matchIDs[i:end]

		// Cession au live : si un sync tient déjà le joueur, on cède (les matchs
		// restants seront repris à la prochaine passe). Le watcher temps-réel
		// (Submit) n'est jamais bloqué par ce claim.
		release, ok := cfg.tryClaim()
		if !ok {
			res.Ceded = true
			slog.InfoContext(ctx, "convergence backfill events: cession au sync live",
				"gamertag", cfg.Gamertag, "remaining", len(matchIDs)-i)
			return res, nil
		}
		cfg.processChunk(ctx, chunk, &res)
		release()

		if end < len(matchIDs) {
			select {
			case <-ctx.Done():
				return res, ctx.Err()
			case <-time.After(yield):
			}
		}
	}
	slog.InfoContext(ctx, "convergence backfill events: passe terminée",
		"gamertag", cfg.Gamertag, "events_written", res.EventsWritten,
		"no_film_final", res.NoFilmFinal, "skipped", res.Skipped)
	return res, nil
}

// detectIncomplete liste les matchs sans events (events_loaded=false), récent→vieux,
// via le détecteur existant. Fenêtre shared courte (lecture).
func (cfg EventsConvergenceConfig) detectIncomplete(ctx context.Context) ([]string, error) {
	sharedDB, release, err := cfg.AcquireShared(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	scope := &SyncScope{Events: true}
	scope.Resolve()
	return FindMatchesMissingData(ctx, cfg.PlayerDB, sharedDB, cfg.XUID, scope)
}

// tryClaim cède au live si un Coordinator est fourni ; sinon claim toujours ok.
func (cfg EventsConvergenceConfig) tryClaim() (func(), bool) {
	if cfg.Coord == nil {
		return func() {}, true
	}
	return cfg.Coord.TryClaim(cfg.Gamertag)
}

// chunkFetch porte le résultat de fetch+parse d'un match (hors lease).
type chunkFetch struct {
	matchID string
	found   bool
	err     error
	events  []analysis.HighlightEvent
}

// processChunk fetche tout le lot HORS lease, puis persiste le lot en UNE fenêtre
// RW courte. Met à jour res (compteurs). Best-effort par match.
func (cfg EventsConvergenceConfig) processChunk(ctx context.Context, chunk []string, res *EventsConvergenceResult) {
	// ── Fetch + parse HORS lease (réseau lent + CPU pur) ──
	fetched := make([]chunkFetch, 0, len(chunk))
	for _, mid := range chunk {
		if ctx.Err() != nil {
			return
		}
		// startTime zéro → pas d'attente fresh-film (matchs de backfill = anciens).
		data, ver, found, err := fetchHighlightChunkResilient(ctx, cfg.Client, mid, time.Time{})
		f := chunkFetch{matchID: mid, found: found && len(data) > 0, err: err}
		if err == nil && f.found {
			if evs, perr := analysis.ParseHighlightEvents(data, ver); perr != nil {
				f.err = perr
			} else {
				f.events = evs
			}
		}
		fetched = append(fetched, f)
	}

	// ── Persist le lot en fenêtre RW courte ──
	sharedDB, release, err := cfg.AcquireShared(ctx)
	if err != nil {
		res.Skipped += len(fetched)
		slog.WarnContext(ctx, "convergence backfill events: acquire shared échoué (lot reporté)",
			"gamertag", cfg.Gamertag, "err", err)
		return
	}
	defer release()
	persister := persist.NewEventsCompletionPersister(sharedDB)
	for _, f := range fetched {
		switch {
		case f.err != nil:
			res.Skipped++ // réseau/parse → events_loaded reste false → repris
		case f.found && len(f.events) > 0:
			if _, e := persistCombatCompletion(ctx, sharedDB, f.matchID, f.events); e != nil {
				res.Skipped++
			} else {
				res.EventsWritten++
			}
		case f.found:
			res.Skipped++ // anomalie : chunk non-vide, 0 event parsé → repris
		default: // film absent (404)
			if isNoFilmDefinitive(ctx, sharedDB, f.matchID) {
				if e := persister.MarkNoFilmDefinitive(ctx, f.matchID, MBitEvents); e != nil {
					res.Skipped++
				} else {
					res.NoFilmFinal++
				}
			}
			// sinon : match récent, film pas encore propagé → laissé pour la passe suivante
		}
	}
}
