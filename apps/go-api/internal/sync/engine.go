// Package sync — engine.go : moteur de synchronisation delta/full.
//
// Portage du DuckDBSyncEngine Python (engine.py + mixins).
// Le moteur est instancié une fois par requête de sync et n'est pas réutilisable.
//
// Flux RunDelta :
//  1. Acquérir les write leases (player + shared)
//  2. Ouvrir les deux DBs en lecture/écriture
//  3. Charger les match_ids déjà connus (player_match_enrichment)
//  4. Paginer l'historique API jusqu'à maxMatches nouveaux ou fin
//  5. Pour chaque match nouveau : fetch stats → transform → insert shared + player
//  6. Mettre à jour sync_meta
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"

	"golang.org/x/sync/errgroup"
)

const (
	// historyPageSize est le nombre de matchs demandés par page API.
	historyPageSize = 25
)

// SyncEngine orchestre la synchronisation des données Halo d'un joueur.
// FriendsLoader retourne la liste courante des amis configurés (typiquement
// settings.FriendGamertags). Nil → feature désactivée (legacy / pas de wiring).
type FriendsLoader func() ([]string, error)

type SyncEngine struct {
	gamertag       string
	xuid           string
	titleSlug      string
	playerDBPath   string
	sharedDBPath   string
	globalDBPath   string // P5.3 : data/global/xbox_aliases.duckdb (mapping xuid→gamertag global)
	metadataDBPath string
	tokens         *domain.HaloTokens
	// provider est utilisé pour résoudre l'access_token Xbox Live (achievements).
	// Nil si non défini (les achievements seront ignorés).
	provider auth.TokenProvider
	// resolver est utilisé pour le pré-warming des images d'achievements (optionnel).
	resolver assets.Resolver
	// customClient est optionnel — si non-nil, utilisé à la place de NewHaloAPIClient.
	// Permet l'injection de PooledHaloClient ou autres implémentations HaloClient.
	customClient HaloClient
	// localFilmCache (optionnel) court-circuite l'API film en lisant le cache
	// disque hérité du projet Python. Utile pour récupérer manifestes +
	// chunks REPLICATION_DATA déjà téléchargés (~942 matchs en cache).
	localFilmCache *LocalFilmCache
	// prestigeHook est appelé après ingestion (best-effort, no-op si nil).
	// Reçoit (ctx, gamertag, titleSlug) — le hook se charge lui-même de
	// la résolution Prestige et du feature flag.
	prestigeHook func(ctx context.Context, gamertag, titleSlug string)
	// friendsLoader résout settings.FriendGamertags à la demande pour le
	// hook auto-recompute is_with_friends post-sync delta. Nil → feature off
	// (les nouveaux matchs resteront is_with_friends=FALSE jusqu'au prochain
	// recompute manuel via PATCH /settings ou CLI levelup recompute-friends).
	friendsLoader FriendsLoader
	// metaDB (optionnel) — connexion ouverte par run() au démarrage de la sync
	// pour permettre l'enrichissement post-Extract des MatchRegistryRow via
	// asset_translations (cf. EnrichRegistryFromMetadata, anti-régression UUIDs
	// bruts dans match_registry.playlist_name). Nil dans les tests unitaires
	// qui appellent processMatch directement → l'enrichissement devient no-op
	// et la sync reste fonctionnelle (UUID préservé comme avant).
	metaDB *sql.DB
	// csrSeasonID est l'identifiant de saison CSR courant (ex: "CsrSeason8").
	// Vide → runCSRSnapshotSync est skippé silencieusement.
	csrSeasonID string

	// sharedProvider (commit 8i) — si non-nil, le sync engine route ses
	// ouvertures RW de shared via Provider.AcquireWriter (mode B-swap).
	// Sinon, fallback OpenSharedDB direct (mode legacy, comportement
	// pre-sprint, conflit "different configuration" possible).
	//
	// Injecté via WithSharedProvider depuis main.go / scheduler en mode
	// flag-on. Cohérent avec PlayerPoolConfig.SharedReader côté pool.
	sharedProvider sharedprovider.Provider
}

// NewSyncEngine, WithPrestigeHook, WithResolver, WithSharedProvider,
// WithFriendsLoader, WithCSRSeasonID, SetCustomClient, SetLocalFilmCache :
// déplacés vers engine_options.go (refactor 2026-05-21).
//
// AcquireSharedWriterStandalone, AcquireMetadataWriterStandalone,
// AcquirePlayerWriterStandalone, e.acquireSharedWriter :
// déplacés vers engine_acquire.go (refactor 2026-05-21).

// RunDelta synchronise uniquement les matchs nouveaux depuis la dernière sync.
// S'arrête dès qu'un match connu est rencontré dans l'historique paginé.
func (e *SyncEngine) RunDelta(ctx context.Context, opts domain.SyncOptions) (domain.SyncResult, error) {
	return e.run(ctx, opts, true)
}

// RunFull synchronise tous les matchs jusqu'à opts.MaxMatches (peu importe l'historique connu).
func (e *SyncEngine) RunFull(ctx context.Context, opts domain.SyncOptions) (domain.SyncResult, error) {
	return e.run(ctx, opts, false)
}

// RunBackfill, RunBackfillEngagementScores, RunBackfillEngagementCoefficients,
// RunBackfillLUSR, RunBackfillCSR, RunBackfillPerf, RunBackfillComebackBadges,
// loadMedalExploitMap, loadMedalExploitMapBestEffort, selectMatchesForComebackBadges,
// loadAllMatchIDsForPlayer, loadFlaggedMatchIDs : déplacés vers engine_backfills.go
// (refactor 2026-05-21).

// run est le cœur du moteur de sync. isDelta=true → stop dès un match connu.
func (e *SyncEngine) run(ctx context.Context, opts domain.SyncOptions, isDelta bool) (domain.SyncResult, error) {
	result := domain.SyncResult{StartedAt: time.Now()}
	mode := "full"
	if isDelta {
		mode = "delta"
	}

	// Sprint B1 commit 16 : crée un event_id qui sera ajouté à TOUS les logs
	// émis depuis ce ctx (cf. ContextHandler) — permet de grep cross-module
	// dans logs/{sync,provider,pool,...}.log pour reconstituer le timeline
	// d'un sync donné.
	ctx, eventID := logging.WithEvent(ctx, "sync.Run"+strings.Title(mode))
	slog.InfoContext(ctx, "sync: démarrage",
		"gamertag", e.gamertag,
		"xuid", e.xuid,
		"mode", mode,
		"match_type", opts.MatchType,
		"max_matches", opts.MaxMatches,
		"with_participants", opts.WithParticipants,
		"with_medals", opts.WithMedals,
		"rps", opts.RequestsPerSecond,
		"event", eventID,
	)

	// B8 : validation fail-fast des options avant tout accès réseau ou DB.
	if err := opts.Validate(); err != nil {
		slog.ErrorContext(ctx, "sync: options invalides", "err", err, "gamertag", e.gamertag)
		return result, fmt.Errorf("run: options invalides: %w", err)
	}

	// ─── Write leases ──────────────────────────────────────────────────────────
	slog.DebugContext(ctx, "sync: acquisition lease player DB", "gamertag", e.gamertag, "db", e.playerDBPath)
	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		slog.ErrorContext(ctx, "sync: lease player DB échouée", "gamertag", e.gamertag, "err", err)
		return result, fmt.Errorf("run: %w", err)
	}
	defer writerPlayer.Release()

	// Sprint B1 commit 11b : le dblease shared est désormais pris par
	// acquireSharedWriter (Provider ou legacy). Ne PAS le prendre ici sinon
	// auto-deadlock (cf. provider.go:231 + sync.Mutex non-réentrant).

	// ─── Ouverture des DBs ─────────────────────────────────────────────────────
	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		slog.ErrorContext(ctx, "sync: ouverture player DB échouée", "gamertag", e.gamertag, "db", e.playerDBPath, "err", err)
		return result, fmt.Errorf("run OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()
	playerDB := playerHandle.SQLDb()

	// Commit 8i : route via Provider en mode B-swap (coordonne avec le pool
	// joueur via Subscribe). Fallback OpenSharedDB direct si Provider nil.
	sharedDB, releaseShared, err := e.acquireSharedWriter(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "sync: ouverture shared DB échouée", "gamertag", e.gamertag, "db", e.sharedDBPath, "err", err)
		return result, fmt.Errorf("run acquireSharedWriter: %w", err)
	}
	defer releaseShared()

	// P5.3 : DB globale xbox_aliases (mapping xuid→gamertag global Microsoft).
	globalDB, globalCleanup, err := openGlobalDB(ctx, e.globalDBPath)
	if err != nil {
		slog.WarnContext(ctx, "sync: ouverture global DB échouée — alias upsert désactivé",
			"db", e.globalDBPath, "err", err)
		globalDB = nil
	} else {
		defer globalCleanup()
	}

	// metaDB best-effort : utilisé par EnrichRegistryFromMetadata pour résoudre
	// les UUIDs bruts en noms canoniques EN avant l'INSERT match_registry.
	// Échec d'ouverture → enrichissement désactivé pour ce run, sync continue.
	if e.metadataDBPath != "" {
		metaDB, metaErr := sql.Open("duckdb", e.metadataDBPath+"?access_mode=read_only")
		if metaErr != nil {
			slog.WarnContext(ctx, "sync: ouverture metadata DB échouée — enrich registry désactivé",
				"db", e.metadataDBPath, "err", metaErr)
		} else {
			metaDB.SetMaxOpenConns(1)
			e.metaDB = metaDB
			defer func() {
				_ = metaDB.Close()
				e.metaDB = nil
			}()
		}
	}
	slog.DebugContext(ctx, "sync: DBs ouvertes", "gamertag", e.gamertag)

	// ─── Match IDs déjà connus (player DB) ───────────────────────────────────
	known, err := loadKnownMatchIDs(ctx, playerDB)
	if err != nil {
		slog.ErrorContext(ctx, "sync: chargement match_ids connus échoué", "gamertag", e.gamertag, "err", err)
		return result, fmt.Errorf("run loadKnownMatchIDs: %w", err)
	}
	slog.InfoContext(ctx, "sync: match_ids connus chargés", "gamertag", e.gamertag, "known_count", len(known))

	// ─── Client API ────────────────────────────────────────────────────────────
	var client HaloClient
	if e.customClient != nil {
		client = e.customClient
		slog.DebugContext(ctx, "sync: utilisation client personnalisé (pool)")
	} else {
		api := NewHaloAPIClient(e.tokens.SpartanToken, e.tokens.ClearanceToken, opts.RequestsPerSecond)
		if e.localFilmCache != nil {
			api = api.WithLocalFilmCache(e.localFilmCache)
			slog.InfoContext(ctx, "sync: cache film local actif", "gamertag", e.gamertag)
		}
		client = api
		slog.DebugContext(ctx, "sync: utilisation HaloAPIClient standard")
	}

	// ─── Pagination de l'historique ────────────────────────────────────────────
	processed := 0
	start := 0

	for processed < opts.MaxMatches {
		// Respecter le contexte d'annulation.
		if err := ctx.Err(); err != nil {
			break
		}

		slog.DebugContext(ctx, "sync: requête historique API",
			"gamertag", e.gamertag, "xuid", e.xuid, "start", start, "page_size", historyPageSize,
		)
		// L'endpoint /hi/players/{player}/matches exige strictement le format
		// xuid(NNN) (voir Grunt StatsModule.GetMatchHistory + SPNKr). Passer le
		// gamertag directement renvoie une réponse stale figée — symptôme du
		// "no inserts since 6 mai" diagnostiqué le 2026-05-20.
		entries, err := client.GetMatchHistory(ctx, fmt.Sprintf("xuid(%s)", e.xuid), opts.MatchType, start, historyPageSize)
		if err != nil {
			slog.WarnContext(ctx, "sync: GetMatchHistory échoué",
				"gamertag", e.gamertag, "start", start, "err", err,
			)
			result.AddWarning(fmt.Sprintf("GetMatchHistory(start=%d): %v", start, err))
			break
		}
		if len(entries) == 0 {
			slog.DebugContext(ctx, "sync: fin historique (page vide)", "gamertag", e.gamertag, "start", start)
			break // fin de l'historique
		}
		slog.DebugContext(ctx, "sync: page reçue",
			"gamertag", e.gamertag, "entries", len(entries), "start", start,
		)
		// Log INFO du 1er match retourné par l'API (seulement sur start=0).
		// Sentinelle de fraîcheur : si ce StartTime ne bouge pas entre 2 cycles
		// alors que le joueur a joué, on sait que l'API renvoie du stale
		// (cf. incident 2026-05-20, endpoint /hi/players/{gamertag}/matches sans
		// xuid(...) renvoyait du contenu figé).
		if start == 0 && len(entries) > 0 {
			slog.InfoContext(ctx, "sync: 1er match retourné par API",
				"gamertag", e.gamertag, "xuid", e.xuid,
				"first_match_id", entries[0].MatchID,
				"first_match_start_time", entries[0].StartTime,
			)
		}

		allKnown := true

		// ─── Phase 1 : Filtrer et préparer les matchs à fetcher ───
		var toFetch []string // MatchIDs à fetcher (l'ordre suit `entries`)
		// stopAfterFlush : on a rencontré un match connu en mode delta. On
		// arrête après avoir fetché/inséré les nouveaux déjà collectés —
		// ne PAS goto done direct sinon on perd les entries unknown qui
		// précèdent le connu dans la même page (bug 2026-05-21 : page
		// renvoyait [cd89b091 (new May 11), b8c1b220 (known May 6)] →
		// goto done sautait Phase 2 → cd89b091 jamais inséré).
		stopAfterFlush := false

		for _, entry := range entries {
			if processed >= opts.MaxMatches {
				break
			}
			if known[entry.MatchID] {
				result.MatchesSkipped++
				if isDelta {
					slog.InfoContext(ctx, "sync: match connu rencontré — arrêt delta après flush",
						"gamertag", e.gamertag, "match_id", entry.MatchID,
						"processed", processed, "skipped", result.MatchesSkipped,
						"pending_fetch", len(toFetch),
					)
					stopAfterFlush = true
					break
				}
				continue
			}
			allKnown = false
			toFetch = append(toFetch, entry.MatchID)
		}

		if len(toFetch) > 0 {
			// ─── Phase 2 : Fetch parallèle ───
			fetchedMatches := make([]*fetchedMatch, len(toFetch))
			fetchErrors := make([]error, len(toFetch))
			var mu sync.Mutex

			eg, egCtx := errgroup.WithContext(ctx)
			// Pas de SetLimit ici — RPS limité par HaloAPIClient.rateWait()
			for i, matchID := range toFetch {
				i, matchID := i, matchID // Capturer pour closure
				eg.Go(func() error {
					fm, err := e.fetchMatchData(egCtx, client, matchID, opts)
					mu.Lock()
					fetchedMatches[i] = fm
					fetchErrors[i] = err
					mu.Unlock()
					if err != nil {
						slog.WarnContext(egCtx, "sync: fetchMatchData échoué",
							"gamertag", e.gamertag, "match_id", matchID, "err", err,
						)
						result.AddWarning(fmt.Sprintf("fetchMatchData(%s): %v", matchID, err))
					}
					return nil // Non-fatal : continuer même si fetch échoue
				})
			}
			_ = eg.Wait() // Attendre tous les fetches (même si certains échouent)

			// ─── Phase 3 : Insert séquentiel (order-preserving) ───
			for i, fm := range fetchedMatches {
				if fetchErrors[i] != nil {
					// Fetch échoué, skip insert
					continue
				}
				if fm == nil {
					continue
				}

				if err := e.insertFetchedMatch(ctx, sharedDB, playerDB, globalDB, &result, fm); err != nil {
					slog.WarnContext(ctx, "sync: insertFetchedMatch échoué",
						"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err,
					)
					result.AddWarning(fmt.Sprintf("insertFetchedMatch(%s): %v", fm.MatchID, err))
				} else {
					processed++
					slog.InfoContext(ctx, "sync: match traité (parallèle)",
						"gamertag", e.gamertag, "match_id", fm.MatchID,
						"processed", processed, "inserted_total", result.MatchesInserted,
					)
				}
			}
		}

		if stopAfterFlush {
			// Match connu rencontré, mais les unknowns déjà collectés ont été
			// fetchés/insérés via Phase 2-3 ci-dessus. On peut sortir.
			break
		}
		if isDelta && allKnown {
			break
		}
		start += len(entries)
	}

	slog.InfoContext(ctx, "sync: boucle pagination terminée",
		"gamertag", e.gamertag, "mode", mode,
		"inserted", result.MatchesInserted, "skipped", result.MatchesSkipped,
		"warnings", len(result.Warnings),
	)

	// ─── Pipeline post-sync ─────────────────────────────────────────────────────
	postResult := e.runConditionalPostSync(ctx, playerDB, sharedDB, client, result.MatchesInserted, result.InsertedMatchIDs)
	if result.MatchesInserted > 0 || postResult.AchievementsSynced {
		result.PostSync = &postResult
		slog.InfoContext(ctx, "sync: pipeline post-sync terminé",
			"gamertag", e.gamertag,
			"perf_scores", postResult.PerfScoresComputed,
			"lusr_updated", postResult.LUSRUpdated,
			"views_refreshed", postResult.ViewsRefreshed,
			"achievements_synced", postResult.AchievementsSynced,
		)
	}

	// ─── sync_meta ──────────────────────────────────────────────────────────────
	if err := SetSyncMeta(ctx, playerDB, "last_delta_sync", time.Now().UTC().Format(time.RFC3339)); err != nil {
		result.AddWarning(fmt.Sprintf("SetSyncMeta: %v", err))
	}

	// ─── Hook Prestige (post-sync) ──────────────────────────────────────────────
	// Best-effort : ré-évalue les défis Prestige actifs après ingestion.
	// No-op si feature flag PRESTIGE_ENABLED off ou si le hook n'est pas câblé.
	// Le hook ne propage jamais d'erreur pour ne pas casser le sync.
	if e.prestigeHook != nil {
		e.prestigeHook(ctx, e.gamertag, e.titleSlug)
	}

	result.FinishedAt = time.Now()
	result.DurationSeconds = result.FinishedAt.Sub(result.StartedAt).Seconds()

	slog.InfoContext(ctx, "sync: terminé",
		"gamertag", e.gamertag, "mode", mode,
		"inserted", result.MatchesInserted,
		"skipped", result.MatchesSkipped,
		"medals", result.MedalsInserted,
		"participants", result.ParticipantsDone,
		"warnings", len(result.Warnings),
		"duration_s", fmt.Sprintf("%.2f", result.DurationSeconds),
		"status", result.Status(),
	)
	return result, nil
}

// runConditionalPostSync, hasMatchesNeedingScoreRefresh : déplacés vers
// engine_postsync.go (refactor 2026-05-21).

// processMatch : déplacé vers engine_process_match.go (refactor 2026-05-21).
// Legacy séquentiel encore utilisé par engine_e2e_test.go.

// fetchedMatch struct + fetchMatchData + insertFetchedMatch + hasAnyTeamMMR :
// déplacés vers engine_fetch.go (refactor 2026-05-21). Pipeline parallèle
// fetch (Phase 2 errgroup) + insert séquentiel (Phase 3).

// insertHighlightEventsFromData, ProcessHighlightEvents : déplacés vers
// engine_highlight_events.go (refactor 2026-05-21). Parse + insert events
// (path standalone via ProcessHighlightEvents pour outils de replay ; path
// in-line via insertHighlightEventsFromData depuis insertFetchedMatch).

// loadKnownMatchIDs retourne l'ensemble des match_ids déjà présents dans
// player_match_enrichment (player DB).
func loadKnownMatchIDs(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, "SELECT match_id FROM player_match_enrichment")
	if err != nil {
		// Table peut ne pas exister si le schéma vient d'être créé — OK.
		return map[string]bool{}, nil
	}
	defer rows.Close()

	known := make(map[string]bool, 256)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			known[id] = true
		}
	}
	return known, rows.Err()
}

// runPostSyncPipeline : déplacé vers engine_postsync.go (refactor 2026-05-21).
// Le pipeline post-sync enchaîne notamment : batchComputeEngagementScores
// (calcul des paces) PUIS batchRecomputeCoefficients (recompute du coef
// team_share depuis la médiane des paces) — voir engine_postsync.go étapes
// 1.5 et 1.5.b. Le hook recompute doit rester APRÈS le compute sinon le
// coef reste à 1.0 cold-start (cf. TestRegressionB5_RecomputeCoefHookWired).

// runCSRSnapshotSync, runAchievementsSync, RunAchievementsOnly,
// resolveAccessTokenFromDB : déplacés vers engine_postsync.go (refactor 2026-05-21).
