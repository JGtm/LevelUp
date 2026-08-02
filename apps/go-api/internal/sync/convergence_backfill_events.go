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
	"os"
	"strconv"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
)

const (
	defaultConvergenceChunkSize     = 15
	defaultConvergenceYieldInterval = 2 * time.Second
)

// ConvergenceLogModule route tous les logs du backfill events vers
// logs/convergence.log (tag slog `module`). Sous-système background qui s'étale
// sur sync (orchestrateur) + scheduler (passe/trigger) — un fichier unique le
// rend diagnosticable d'un coup (même précédent que ModuleCatalog / ModuleLab).
const ConvergenceLogModule = "convergence"

// convergenceClaimGate : sous-ensemble minimal de SyncGate/Coordinator pour la
// cession au sync live. TryClaim(gamertag) → (release, ok=false si déjà en vol).
// Passer nil quand le caller tient déjà le claim (ex. passe scheduler sous claim).
type convergenceClaimGate interface {
	TryClaim(gamertag string) (func(), bool)
}

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
	// Coord (optionnel) : cession au sync live via TryClaim. nil = pas de cession
	// (ex. le caller tient déjà le claim). *Coordinator et sync.SyncGate le satisfont.
	Coord    convergenceClaimGate
	Gamertag string
	XUID     string

	// MaxMatches borne le nombre de matchs traités par passe (0 = tous les incomplets).
	MaxMatches int
	// ChunkSize : matchs par fenêtre RW (défaut 15). YieldInterval : pause entre
	// lots pour laisser l'app respirer (défaut 2s).
	ChunkSize     int
	YieldInterval time.Duration

	// log : logger taggé module=convergence (→ logs/convergence.log), initialisé
	// par RunEventsConvergenceBackfill. Interne (non fourni par le caller).
	log *slog.Logger
}

// EventsConvergenceResult résume une passe de backfill.
type EventsConvergenceResult struct {
	Detected         int  // matchs incomplets détectés (events_loaded=false)
	EventsWritten    int  // matchs ayant reçu des events (film présent)
	NoFilmFinal      int  // matchs marqués no-film définitif (404 + trop vieux)
	Skipped          int  // erreurs réseau/parse/anomalie → repris la passe suivante
	DominanceUpdated int  // matchs dont le dominance_flag a été (re)calculé après events
	Ceded            bool // a cédé à un sync live en cours de route
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
	// Logger taggé : tous les logs de la passe atterrissent dans logs/convergence.log.
	cfg.log = slog.With("module", ConvergenceLogModule)

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
	cfg.log.InfoContext(ctx, "convergence backfill events: démarrage",
		"gamertag", cfg.Gamertag, "detected", res.Detected, "chunk", chunkSize)

	var eventMatchIDs []string // matchs ayant reçu des events → dominance ensuite
	for i := 0; i < len(matchIDs); i += chunkSize {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		end := min(i+chunkSize, len(matchIDs))
		chunk := matchIDs[i:end]

		// Cession au live : si un sync tient déjà le joueur, on cède (les matchs
		// restants seront repris à la prochaine passe). Le watcher temps-réel
		// (Submit) n'est jamais bloqué par ce claim. On break (pas return) pour
		// quand même calculer la dominance des matchs déjà traités.
		release, ok := cfg.tryClaim()
		if !ok {
			res.Ceded = true
			cfg.log.InfoContext(ctx, "convergence backfill events: cession au sync live",
				"gamertag", cfg.Gamertag, "remaining", len(matchIDs)-i)
			break
		}
		cfg.processChunk(ctx, chunk, &res, &eventMatchIDs)
		release()

		if end < len(matchIDs) {
			select {
			case <-ctx.Done():
				return res, ctx.Err()
			case <-time.After(yield):
			}
		}
	}

	// Dominance flags sur les matchs fraîchement pourvus d'events (REMONTADA,
	// DÉBÂCLE, etc. — dépendent de la courbe de score reconstruite depuis
	// highlight_events). Réutilise BackfillDominanceFlags ; no-op sans player DB.
	cfg.computeDominance(ctx, eventMatchIDs, &res)

	cfg.log.InfoContext(ctx, "convergence backfill events: passe terminée",
		"gamertag", cfg.Gamertag, "events_written", res.EventsWritten,
		"no_film_final", res.NoFilmFinal, "skipped", res.Skipped,
		"dominance_updated", res.DominanceUpdated, "ceded", res.Ceded)
	return res, nil
}

// computeDominance recalcule le dominance_flag des matchs qui viennent de recevoir
// leurs events (la courbe de score nécessite highlight_events). Best-effort :
// no-op si aucun match ou pas de player DB ; une erreur est loggée sans échouer
// la passe.
func (cfg EventsConvergenceConfig) computeDominance(ctx context.Context, matchIDs []string, res *EventsConvergenceResult) {
	if len(matchIDs) == 0 || cfg.PlayerDB == nil {
		return
	}
	sharedDB, release, err := cfg.AcquireShared(ctxkeys.WithDBWriterLabel(ctx, "events_convergence_dominance"))
	if err != nil {
		cfg.log.WarnContext(ctx, "convergence backfill events: acquire shared pour dominance échoué",
			"gamertag", cfg.Gamertag, "err", err)
		return
	}
	defer release()
	if err := BackfillDominanceFlags(ctx, sharedDB, cfg.PlayerDB, cfg.XUID, matchIDs); err != nil {
		cfg.log.WarnContext(ctx, "convergence backfill events: dominance flags échoué",
			"gamertag", cfg.Gamertag, "err", err, "count", len(matchIDs))
		return
	}
	res.DominanceUpdated = len(matchIDs)
}

// RunEventsConvergence lance une passe de backfill events pour le joueur de cet
// engine, en réutilisant son client poolé + provider (B-swap). gate (optionnel)
// permet la cession au sync live par lot — passer nil quand le caller tient déjà
// le claim (passe scheduler). maxMatches borne la passe (0 = tous les incomplets).
// No-op (erreur) si aucun client API n'est câblé (pool absent).
func (e *SyncEngine) RunEventsConvergence(ctx context.Context, gate convergenceClaimGate, maxMatches int) (EventsConvergenceResult, error) {
	if e.customClient == nil {
		return EventsConvergenceResult{}, fmt.Errorf("RunEventsConvergence: client API absent (pool non câblé)")
	}
	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return EventsConvergenceResult{}, fmt.Errorf("RunEventsConvergence OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()
	return RunEventsConvergenceBackfill(ctx, EventsConvergenceConfig{
		Client:        e.customClient,
		AcquireShared: e.acquireSharedWriter,
		PlayerDB:      playerHandle.SQLDb(),
		Coord:         gate,
		Gamertag:      e.gamertag,
		XUID:          e.xuid,
		MaxMatches:    maxMatches,
		// Réglages fins prod (0 → défauts de l'orchestrateur : 15 / 2s). L'env est
		// lu ici (entrée prod) pour garder RunEventsConvergenceBackfill pur/testable.
		ChunkSize:     convergenceEnvInt("LEVELUP_EVENTS_CONVERGENCE_CHUNK"),
		YieldInterval: time.Duration(convergenceEnvInt("LEVELUP_EVENTS_CONVERGENCE_YIELD_MS")) * time.Millisecond,
	})
}

// convergenceEnvInt lit un entier positif depuis l'env ; 0 si absent/invalide
// (l'orchestrateur applique alors son défaut).
func convergenceEnvInt(key string) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 0
}

// detectIncomplete liste les matchs sans events (events_loaded=false), récent→vieux,
// via le détecteur existant. Fenêtre shared courte (lecture).
func (cfg EventsConvergenceConfig) detectIncomplete(ctx context.Context) ([]string, error) {
	sharedDB, release, err := cfg.AcquireShared(ctxkeys.WithDBWriterLabel(ctx, "events_convergence_detect"))
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
// RW courte. Met à jour res (compteurs) et accumule dans eventMatchIDs les matchs
// ayant reçu des events (pour la dominance ensuite). Best-effort par match.
func (cfg EventsConvergenceConfig) processChunk(ctx context.Context, chunk []string, res *EventsConvergenceResult, eventMatchIDs *[]string) {
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
	sharedDB, release, err := cfg.AcquireShared(ctxkeys.WithDBWriterLabel(ctx, "events_convergence_write"))
	if err != nil {
		res.Skipped += len(fetched)
		cfg.log.WarnContext(ctx, "convergence backfill events: acquire shared échoué (lot reporté)",
			"gamertag", cfg.Gamertag, "err", err)
		return
	}
	defer release()
	// Anti-TOCTOU multi-joueurs : le fetch a eu lieu HORS lease — un post-sync
	// parallèle (coéquipier partageant le match) a pu converger ces events entre
	// la sélection et ce burst. Re-check sous le burst (sérialisé par dblease),
	// sinon persistCombatCompletion dupliquerait les highlight_events (INSERT
	// OR IGNORE = no-op de dédup en prod).
	var fetchedIDs []string
	for _, f := range fetched {
		if f.err == nil && f.found && len(f.events) > 0 {
			fetchedIDs = append(fetchedIDs, f.matchID)
		}
	}
	still := make(map[string]bool, len(fetchedIDs))
	for _, mid := range filterEventsStillMissing(ctx, sharedDB, fetchedIDs) {
		still[mid] = true
	}
	persister := persist.NewEventsCompletionPersister(sharedDB)
	for _, f := range fetched {
		switch {
		case f.err != nil:
			res.Skipped++ // réseau/parse → events_loaded reste false → repris
		case f.found && len(f.events) > 0:
			if !still[f.matchID] {
				res.Skipped++ // convergé par un sync parallèle entre fetch et burst
				continue
			}
			if _, e := persistCombatCompletion(ctx, sharedDB, f.matchID, f.events); e != nil {
				// LOGGÉ AVANT LA DÉGRADATION. Le match repart au retry set (events_loaded reste
				// false), donc l'échec est RATTRAPABLE — mais un `Skipped++` muet le rendait
				// indistinguable d'un match convergé par un sync parallèle. Une écriture qui
				// échoue à chaque passe boucle alors indéfiniment sans qu'aucune ligne de log
				// ni aucun compteur ne le dise.
				observability.AddIntT(ctxkeys.TitleSlug(ctx),
					"convergence_events_persist_failed_total", 1)
				cfg.log.ErrorContext(ctx, "convergence backfill events: persistance de la "+
					"complétion combat échouée (match repris à la passe suivante)",
					"gamertag", cfg.Gamertag, "match_id", f.matchID, "err", e)
				res.Skipped++
			} else {
				res.EventsWritten++
				*eventMatchIDs = append(*eventMatchIDs, f.matchID)
			}
		case f.found:
			// Anomalie : chunk non-vide, 0 event parsé. Définitif (vieux) →
			// events_empty (sort du retry set sans mentir sur events_loaded) ;
			// récent → repris (film peut-être encore instable).
			if isNoFilmDefinitive(ctx, sharedDB, f.matchID) {
				if e := persister.MarkEventsEmptyDefinitive(ctx, f.matchID, MBitEvents); e != nil {
					res.Skipped++
				} else {
					res.NoFilmFinal++ // définitivement résolu : aucun event exploitable
				}
			} else {
				res.Skipped++
			}
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
