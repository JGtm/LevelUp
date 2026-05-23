// Package sync â€” engine.go : moteur de synchronisation delta/full.
//
// Portage du DuckDBSyncEngine Python (engine.py + mixins).
// Le moteur est instanciÃ© une fois par requÃªte de sync et n'est pas rÃ©utilisable.
//
// Flux RunDelta :
//  1. AcquÃ©rir les write leases (player + shared)
//  2. Ouvrir les deux DBs en lecture/Ã©criture
//  3. Charger les match_ids dÃ©jÃ  connus (player_match_enrichment)
//  4. Paginer l'historique API jusqu'Ã  maxMatches nouveaux ou fin
//  5. Pour chaque match nouveau : fetch stats â†’ transform â†’ insert shared + player
//  6. Mettre Ã  jour sync_meta
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
	"levelup/go-api/internal/port"

	"golang.org/x/sync/errgroup"
)

const (
	// historyPageSize est le nombre de matchs demandÃ©s par page API.
	historyPageSize = 25
)

// SyncEngine orchestre la synchronisation des donnÃ©es Halo d'un joueur.
// FriendsLoader retourne la liste courante des amis configurÃ©s (typiquement
// settings.FriendGamertags). Nil â†’ feature dÃ©sactivÃ©e (legacy / pas de wiring).
type FriendsLoader func() ([]string, error)

type SyncEngine struct {
	gamertag       string
	xuid           string
	titleSlug      string
	playerDBPath   string
	sharedDBPath   string
	globalDBPath   string // P5.3 : data/global/xbox_aliases.duckdb (mapping xuidâ†’gamertag global)
	metadataDBPath string
	tokens         *domain.HaloTokens
	// provider est utilisÃ© pour rÃ©soudre l'access_token Xbox Live (achievements).
	// Nil si non dÃ©fini (les achievements seront ignorÃ©s).
	provider auth.TokenProvider
	// resolver est utilisÃ© pour le prÃ©-warming des images d'achievements (optionnel).
	resolver assets.Resolver
	// customClient est optionnel â€” si non-nil, utilisÃ© Ã  la place de NewHaloAPIClient.
	// Permet l'injection de PooledHaloClient ou autres implÃ©mentations HaloClient.
	customClient HaloClient
	// localFilmCache (optionnel) court-circuite l'API film en lisant le cache
	// disque hÃ©ritÃ© du projet Python. Utile pour rÃ©cupÃ©rer manifestes +
	// chunks REPLICATION_DATA dÃ©jÃ  tÃ©lÃ©chargÃ©s (~942 matchs en cache).
	localFilmCache *LocalFilmCache
	// prestigeHook est appelÃ© aprÃ¨s ingestion (best-effort, no-op si nil).
	// ReÃ§oit (ctx, gamertag, titleSlug) â€” le hook se charge lui-mÃªme de
	// la rÃ©solution Prestige et du feature flag.
	prestigeHook func(ctx context.Context, gamertag, titleSlug string)
	// friendsLoader rÃ©sout settings.FriendGamertags Ã  la demande pour le
	// hook auto-recompute is_with_friends post-sync delta. Nil â†’ feature off
	// (les nouveaux matchs resteront is_with_friends=FALSE jusqu'au prochain
	// recompute manuel via PATCH /settings ou CLI levelup recompute-friends).
	friendsLoader FriendsLoader
	// metaDB (optionnel) â€” connexion ouverte par run() au dÃ©marrage de la sync
	// pour permettre l'enrichissement post-Extract des MatchRegistryRow via
	// asset_translations (cf. EnrichRegistryFromMetadata, anti-rÃ©gression UUIDs
	// bruts dans match_registry.playlist_name). Nil dans les tests unitaires
	// qui appellent processMatch directement â†’ l'enrichissement devient no-op
	// et la sync reste fonctionnelle (UUID prÃ©servÃ© comme avant).
	metaDB *sql.DB
	// csrSeasonID est l'identifiant de saison CSR courant (ex: "CsrSeason8").
	// Vide â†’ runCSRSnapshotSync est skippÃ© silencieusement.
	csrSeasonID string

	// sharedProvider (commit 8i) â€” si non-nil, le sync engine route ses
	// ouvertures RW de shared via Provider.AcquireWriter (mode B-swap).
	// Sinon, fallback OpenSharedDB direct (mode legacy, comportement
	// pre-sprint, conflit "different configuration" possible).
	//
	// InjectÃ© via WithSharedProvider depuis main.go / scheduler en mode
	// flag-on. CohÃ©rent avec PlayerPoolConfig.SharedReader cÃ´tÃ© pool.
	sharedProvider sharedprovider.Provider

	// postSyncRunner (Phase 4 plan stabilisation 2026-05-22) — runner invoqué
	// avant + après chaque sync réussie pour émettre les notifications delta
	// (career_rank, citation_tier, threshold_crossed, etc.) ET lancer le
	// pipeline progression V2 Ascension (streaks/records/milestones/coach).
	//
	// Nil → feature off (legacy : avant ce sprint, seuls les syncs HTTP
	// déclenchaient ces hooks via SyncHandler.WithPostSyncDeltaHook ; auto-sync
	// et CLI sautaient TOUT le post-sync delta + progression).
	//
	// Injecté via WithPostSyncRunner depuis main.go (SyncHandler + scheduler).
	// postSyncSlug : identifiant URL du joueur, passé au runner.BeforeSync.
	postSyncRunner port.PostSyncRunner
	postSyncSlug   string

	// mediaHook est appelé à la fin du pipeline post-sync pour indexer les
	// médias présents dans le dossier captures/ et les associer aux matchs
	// fraîchement synchronisés. Best-effort : nil → feature off.
	// Injecté via WithMediaScanHook depuis scheduler + SyncHandler.
	mediaHook func(ctx context.Context)

	// batchMode (Phase 2.3 refactor Collect→Persist) — si true, la boucle
	// d'insertion utilise `submitMatchAsBatch` (chemin INSERT-only via
	// persist.SharedPersister + persist.PlayerPersister) au lieu de
	// `insertFetchedMatch` legacy (UPSERT direct). Feature-flag opt-in
	// activé par WithBatchPersistMode. Synchrone : pas de BatchQueue ni
	// worker async pour l'instant (Phase 3 ajoutera la couche queue).
	batchMode bool
}

// NewSyncEngine, WithPrestigeHook, WithResolver, WithSharedProvider,
// WithFriendsLoader, WithCSRSeasonID, SetCustomClient, SetLocalFilmCache :
// dÃ©placÃ©s vers engine_options.go (refactor 2026-05-21).
//
// AcquireSharedWriterStandalone, AcquireMetadataWriterStandalone,
// AcquirePlayerWriterStandalone, e.acquireSharedWriter :
// dÃ©placÃ©s vers engine_acquire.go (refactor 2026-05-21).

// RunDelta synchronise uniquement les matchs nouveaux depuis la derniÃ¨re sync.
// S'arrÃªte dÃ¨s qu'un match connu est rencontrÃ© dans l'historique paginÃ©.
func (e *SyncEngine) RunDelta(ctx context.Context, opts domain.SyncOptions) (domain.SyncResult, error) {
	return e.run(ctx, opts, true)
}

// RunFull synchronise tous les matchs jusqu'Ã  opts.MaxMatches (peu importe l'historique connu).
func (e *SyncEngine) RunFull(ctx context.Context, opts domain.SyncOptions) (domain.SyncResult, error) {
	return e.run(ctx, opts, false)
}

// RunBackfill, RunBackfillEngagementScores, RunBackfillEngagementCoefficients,
// RunBackfillLUSR, RunBackfillCSR, RunBackfillPerf, RunBackfillComebackBadges,
// loadMedalExploitMap, loadMedalExploitMapBestEffort, selectMatchesForComebackBadges,
// loadAllMatchIDsForPlayer, loadFlaggedMatchIDs : dÃ©placÃ©s vers engine_backfills.go
// (refactor 2026-05-21).

// run est le cÅ“ur du moteur de sync. isDelta=true â†’ stop dÃ¨s un match connu.
func (e *SyncEngine) run(ctx context.Context, opts domain.SyncOptions, isDelta bool) (domain.SyncResult, error) {
	result := domain.SyncResult{StartedAt: time.Now()}
	mode := "full"
	if isDelta {
		mode = "delta"
	}

	// Sprint B1 commit 16 : crÃ©e un event_id qui sera ajoutÃ© Ã  TOUS les logs
	// Ã©mis depuis ce ctx (cf. ContextHandler) â€” permet de grep cross-module
	// dans logs/{sync,provider,pool,...}.log pour reconstituer le timeline
	// d'un sync donnÃ©.
	ctx, eventID := logging.WithEvent(ctx, "sync.Run"+strings.Title(mode))
	slog.InfoContext(ctx, "sync: dÃ©marrage",
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

	// B8 : validation fail-fast des options avant tout accÃ¨s rÃ©seau ou DB.
	if err := opts.Validate(); err != nil {
		slog.ErrorContext(ctx, "sync: options invalides", "err", err, "gamertag", e.gamertag)
		return result, fmt.Errorf("run: options invalides: %w", err)
	}

	// Phase 4 plan stabilisation 2026-05-22 : capture du snapshot before-sync
	// pour la couche post-sync (delta notifications + pipeline progression V2).
	// Le runner est nil si non injecté (legacy / tests) — feature off.
	// Le finalizer sera invoqué en succès uniquement (chemin terminal de run,
	// ligne ~435 `return result, nil`) — sur erreur, le finalizer n'est jamais
	// appelé pour éviter d'émettre des deltas sur snapshot post-sync incohérent.
	var postSyncFinalizer port.PostSyncFinalizer
	if e.postSyncRunner != nil && e.postSyncSlug != "" {
		postSyncFinalizer = e.postSyncRunner.BeforeSync(ctx, e.postSyncSlug)
	}

	// â”€â”€â”€ Write leases â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
	slog.DebugContext(ctx, "sync: acquisition lease player DB", "gamertag", e.gamertag, "db", e.playerDBPath)
	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		slog.ErrorContext(ctx, "sync: lease player DB Ã©chouÃ©e", "gamertag", e.gamertag, "err", err)
		return result, fmt.Errorf("run: %w", err)
	}
	defer writerPlayer.Release()

	// Sprint B1 commit 11b : le dblease shared est dÃ©sormais pris par
	// acquireSharedWriter (Provider ou legacy). Ne PAS le prendre ici sinon
	// auto-deadlock (cf. provider.go:231 + sync.Mutex non-rÃ©entrant).

	// â”€â”€â”€ Ouverture des DBs â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		slog.ErrorContext(ctx, "sync: ouverture player DB Ã©chouÃ©e", "gamertag", e.gamertag, "db", e.playerDBPath, "err", err)
		return result, fmt.Errorf("run OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()
	playerDB := playerHandle.SQLDb()

	// Commit 8i : route via Provider en mode B-swap (coordonne avec le pool
	// joueur via Subscribe). Fallback OpenSharedDB direct si Provider nil.
	sharedDB, releaseShared, err := e.acquireSharedWriter(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "sync: ouverture shared DB Ã©chouÃ©e", "gamertag", e.gamertag, "db", e.sharedDBPath, "err", err)
		return result, fmt.Errorf("run acquireSharedWriter: %w", err)
	}
	defer releaseShared()

	// P5.3 : DB globale xbox_aliases (mapping xuidâ†’gamertag global Microsoft).
	globalDB, globalCleanup, err := openGlobalDB(ctx, e.globalDBPath)
	if err != nil {
		slog.WarnContext(ctx, "sync: ouverture global DB Ã©chouÃ©e â€” alias upsert dÃ©sactivÃ©",
			"db", e.globalDBPath, "err", err)
		globalDB = nil
	} else {
		defer globalCleanup()
	}

	// metaDB best-effort : utilisÃ© par EnrichRegistryFromMetadata pour rÃ©soudre
	// les UUIDs bruts en noms canoniques EN avant l'INSERT match_registry.
	// Ã‰chec d'ouverture â†’ enrichissement dÃ©sactivÃ© pour ce run, sync continue.
	if e.metadataDBPath != "" {
		metaDB, metaErr := sql.Open("duckdb", e.metadataDBPath+"?access_mode=read_only")
		if metaErr != nil {
			slog.WarnContext(ctx, "sync: ouverture metadata DB Ã©chouÃ©e â€” enrich registry dÃ©sactivÃ©",
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

	// â”€â”€â”€ Match IDs dÃ©jÃ  connus (player DB) â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
	// Known set étendu : player_match_enrichment ∪ shared.match_participants
	// WHERE xuid=e.xuid. Permet de skipper le fetch API quand un autre joueur
	// du même cycle a déjà ingéré le match en shared (cross-player dedup).
	known, err := loadKnownMatchIDs(ctx, playerDB, sharedDB, e.xuid)
	if err != nil {
		slog.ErrorContext(ctx, "sync: chargement match_ids connus Ã©chouÃ©", "gamertag", e.gamertag, "err", err)
		return result, fmt.Errorf("run loadKnownMatchIDs: %w", err)
	}
	slog.InfoContext(ctx, "sync: match_ids connus chargÃ©s", "gamertag", e.gamertag, "known_count", len(known))

	// â”€â”€â”€ Client API â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
	var client HaloClient
	if e.customClient != nil {
		client = e.customClient
		slog.DebugContext(ctx, "sync: utilisation client personnalisÃ© (pool)")
	} else {
		api := NewHaloAPIClient(e.tokens.SpartanToken, e.tokens.ClearanceToken, opts.RequestsPerSecond)
		if e.localFilmCache != nil {
			api = api.WithLocalFilmCache(e.localFilmCache)
			slog.InfoContext(ctx, "sync: cache film local actif", "gamertag", e.gamertag)
		}
		client = api
		slog.DebugContext(ctx, "sync: utilisation HaloAPIClient standard")
	}

	// â”€â”€â”€ Pagination de l'historique â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
	processed := 0
	start := 0

	for processed < opts.MaxMatches {
		// Respecter le contexte d'annulation.
		if err := ctx.Err(); err != nil {
			break
		}

		slog.DebugContext(ctx, "sync: requÃªte historique API",
			"gamertag", e.gamertag, "xuid", e.xuid, "start", start, "page_size", historyPageSize,
		)
		// L'endpoint /hi/players/{player}/matches exige strictement le format
		// xuid(NNN) (voir Grunt StatsModule.GetMatchHistory + SPNKr). Passer le
		// gamertag directement renvoie une rÃ©ponse stale figÃ©e â€” symptÃ´me du
		// "no inserts since 6 mai" diagnostiquÃ© le 2026-05-20.
		entries, err := client.GetMatchHistory(ctx, fmt.Sprintf("xuid(%s)", e.xuid), opts.MatchType, start, historyPageSize)
		if err != nil {
			slog.WarnContext(ctx, "sync: GetMatchHistory Ã©chouÃ©",
				"gamertag", e.gamertag, "start", start, "err", err,
			)
			result.AddWarning(fmt.Sprintf("GetMatchHistory(start=%d): %v", start, err))
			break
		}
		if len(entries) == 0 {
			slog.DebugContext(ctx, "sync: fin historique (page vide)", "gamertag", e.gamertag, "start", start)
			break // fin de l'historique
		}
		slog.DebugContext(ctx, "sync: page reÃ§ue",
			"gamertag", e.gamertag, "entries", len(entries), "start", start,
		)
		// Log INFO du 1er match retournÃ© par l'API (seulement sur start=0).
		// Sentinelle de fraÃ®cheur : si ce StartTime ne bouge pas entre 2 cycles
		// alors que le joueur a jouÃ©, on sait que l'API renvoie du stale
		// (cf. incident 2026-05-20, endpoint /hi/players/{gamertag}/matches sans
		// xuid(...) renvoyait du contenu figÃ©).
		if start == 0 && len(entries) > 0 {
			slog.InfoContext(ctx, "sync: 1er match retournÃ© par API",
				"gamertag", e.gamertag, "xuid", e.xuid,
				"first_match_id", entries[0].MatchID,
				"first_match_start_time", entries[0].StartTime,
			)
		}

		allKnown := true

		// â”€â”€â”€ Phase 1 : Filtrer et prÃ©parer les matchs Ã  fetcher â”€â”€â”€
		var toFetch []string // MatchIDs Ã  fetcher (l'ordre suit `entries`)
		// stopAfterFlush : on a rencontrÃ© un match connu en mode delta. On
		// arrÃªte aprÃ¨s avoir fetchÃ©/insÃ©rÃ© les nouveaux dÃ©jÃ  collectÃ©s â€”
		// ne PAS goto done direct sinon on perd les entries unknown qui
		// prÃ©cÃ¨dent le connu dans la mÃªme page (bug 2026-05-21 : page
		// renvoyait [cd89b091 (new May 11), b8c1b220 (known May 6)] â†’
		// goto done sautait Phase 2 â†’ cd89b091 jamais insÃ©rÃ©).
		stopAfterFlush := false

		for _, entry := range entries {
			if processed >= opts.MaxMatches {
				break
			}
			if known[entry.MatchID] {
				result.MatchesSkipped++
				if isDelta {
					slog.InfoContext(ctx, "sync: match connu rencontrÃ© â€” arrÃªt delta aprÃ¨s flush",
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
			// â”€â”€â”€ Phase 2 : Fetch parallÃ¨le â”€â”€â”€
			fetchedMatches := make([]*fetchedMatch, len(toFetch))
			fetchErrors := make([]error, len(toFetch))
			var mu sync.Mutex

			eg, egCtx := errgroup.WithContext(ctx)
			// Pas de SetLimit ici â€” RPS limitÃ© par HaloAPIClient.rateWait()
			for i, matchID := range toFetch {
				i, matchID := i, matchID // Capturer pour closure
				eg.Go(func() error {
					fm, err := e.fetchMatchData(egCtx, client, matchID, opts)
					mu.Lock()
					fetchedMatches[i] = fm
					fetchErrors[i] = err
					mu.Unlock()
					if err != nil {
						slog.WarnContext(egCtx, "sync: fetchMatchData Ã©chouÃ©",
							"gamertag", e.gamertag, "match_id", matchID, "err", err,
						)
						result.AddWarning(fmt.Sprintf("fetchMatchData(%s): %v", matchID, err))
					}
					return nil // Non-fatal : continuer mÃªme si fetch Ã©choue
				})
			}
			_ = eg.Wait() // Attendre tous les fetches (mÃªme si certains Ã©chouent)

			// â”€â”€â”€ Phase 3 : Insert sÃ©quentiel (order-preserving) â”€â”€â”€
			for i, fm := range fetchedMatches {
				if fetchErrors[i] != nil {
					// Fetch Ã©chouÃ©, skip insert
					continue
				}
				if fm == nil {
					continue
				}

				if err := e.submitOrInsertMatch(ctx, sharedDB, playerDB, globalDB, &result, fm); err != nil {
					slog.WarnContext(ctx, "sync: submitOrInsertMatch Ã©chouÃ©",
						"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err,
					)
					result.AddWarning(fmt.Sprintf("submitOrInsertMatch(%s): %v", fm.MatchID, err))
				} else {
					processed++
					slog.InfoContext(ctx, "sync: match traitÃ© (parallÃ¨le)",
						"gamertag", e.gamertag, "match_id", fm.MatchID,
						"processed", processed, "inserted_total", result.MatchesInserted,
					)
				}
			}
		}

		if stopAfterFlush {
			// Match connu rencontrÃ©, mais les unknowns dÃ©jÃ  collectÃ©s ont Ã©tÃ©
			// fetchÃ©s/insÃ©rÃ©s via Phase 2-3 ci-dessus. On peut sortir.
			break
		}
		if isDelta && allKnown {
			break
		}
		start += len(entries)
	}

	slog.InfoContext(ctx, "sync: boucle pagination terminÃ©e",
		"gamertag", e.gamertag, "mode", mode,
		"inserted", result.MatchesInserted, "skipped", result.MatchesSkipped,
		"warnings", len(result.Warnings),
	)

	// â”€â”€â”€ Pipeline post-sync â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
	postResult := e.runConditionalPostSync(ctx, playerDB, sharedDB, client, result.MatchesInserted, result.InsertedMatchIDs)
	if result.MatchesInserted > 0 || postResult.AchievementsSynced {
		result.PostSync = &postResult
		slog.InfoContext(ctx, "sync: pipeline post-sync terminÃ©",
			"gamertag", e.gamertag,
			"perf_scores", postResult.PerfScoresComputed,
			"lusr_updated", postResult.LUSRUpdated,
			"views_refreshed", postResult.ViewsRefreshed,
			"achievements_synced", postResult.AchievementsSynced,
		)
	}

	// â”€â”€â”€ sync_meta â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
	if err := SetSyncMeta(ctx, playerDB, "last_delta_sync", time.Now().UTC().Format(time.RFC3339)); err != nil {
		result.AddWarning(fmt.Sprintf("SetSyncMeta: %v", err))
	}

	// â”€â”€â”€ Hook Prestige (post-sync) â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
	// Best-effort : rÃ©-Ã©value les dÃ©fis Prestige actifs aprÃ¨s ingestion.
	// No-op si feature flag PRESTIGE_ENABLED off ou si le hook n'est pas cÃ¢blÃ©.
	// Le hook ne propage jamais d'erreur pour ne pas casser le sync.
	if e.prestigeHook != nil {
		e.prestigeHook(ctx, e.gamertag, e.titleSlug)
	}

	result.FinishedAt = time.Now()
	result.DurationSeconds = result.FinishedAt.Sub(result.StartedAt).Seconds()

	slog.InfoContext(ctx, "sync: terminÃ©",
		"gamertag", e.gamertag, "mode", mode,
		"inserted", result.MatchesInserted,
		"skipped", result.MatchesSkipped,
		"medals", result.MedalsInserted,
		"participants", result.ParticipantsDone,
		"warnings", len(result.Warnings),
		"duration_s", fmt.Sprintf("%.2f", result.DurationSeconds),
		"status", result.Status(),
	)

	// Phase 4 plan stabilisation 2026-05-22 : invoque le finalizer post-sync
	// (notifications delta + pipeline progression V2) capturé en début de run.
	// Best-effort : toute erreur dans le finalizer reste contenue côté runner.
	if postSyncFinalizer != nil {
		postSyncFinalizer(ctx)
	}
	return result, nil
}

// runConditionalPostSync, hasMatchesNeedingScoreRefresh : dÃ©placÃ©s vers
// engine_postsync.go (refactor 2026-05-21).

// processMatch : dÃ©placÃ© vers engine_process_match.go (refactor 2026-05-21).
// Legacy sÃ©quentiel encore utilisÃ© par engine_e2e_test.go.

// fetchedMatch struct + fetchMatchData + insertFetchedMatch + hasAnyTeamMMR :
// dÃ©placÃ©s vers engine_fetch.go (refactor 2026-05-21). Pipeline parallÃ¨le
// fetch (Phase 2 errgroup) + insert sÃ©quentiel (Phase 3).

// insertHighlightEventsFromData, ProcessHighlightEvents : dÃ©placÃ©s vers
// engine_highlight_events.go (refactor 2026-05-21). Parse + insert events
// (path standalone via ProcessHighlightEvents pour outils de replay ; path
// in-line via insertHighlightEventsFromData depuis insertFetchedMatch).

// loadKnownMatchIDs retourne l'ensemble des match_ids dÃ©jÃ  prÃ©sents dans
// player_match_enrichment (player DB).
// loadKnownMatchIDs construit le set des match_id "connus" pour ce joueur.
//
// Source 1 — playerDB.player_match_enrichment : matchs déjà ingérés dans la
// pipeline complète (registry + participants + enrichment per-player).
//
// Source 2 — sharedDB.match_participants WHERE xuid=? : matchs déjà présents
// en shared pour ce joueur (typiquement parce qu'un AUTRE joueur du même
// cycle de sync a déjà fait le fetch + insert via insertFetchedMatch — qui
// écrit toutes les rows participants y compris celle de notre xuid). Avant
// 2026-05-22 ce 2e check n'existait pas, donc Chocoboflor re-fetchait depuis
// Halo API 21 matchs déjà insérés par Madina dans le même tick (84 calls
// inutiles). Le post-sync (batchComputePerformanceScores, batchComputeLUSR,
// citations) UPSERT naturellement les rows enrichment manquantes pour les
// matchs déjà en shared.
//
// sharedDB peut être nil (cas tests / boot avant shared init) → seule la
// source 1 est consultée.
//
// aujourd'hui best-effort interne (tables manquantes = warning silencieux), mais
// future migration cross-DB pourrait remonter une erreur ici.
//
//nolint:unparam // err maintenu pour signature standard (caller engine.go:241 check),
func loadKnownMatchIDs(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) (map[string]bool, error) {
	known := make(map[string]bool, 512)

	// Source 1 : player_match_enrichment (per-player).
	if rows, err := playerDB.QueryContext(ctx, "SELECT match_id FROM player_match_enrichment"); err == nil {
		for rows.Next() {
			var id string
			if scanErr := rows.Scan(&id); scanErr == nil {
				known[id] = true
			}
		}
		_ = rows.Close()
	}
	// Note: erreur ignorée (table peut ne pas exister sur schéma frais).

	// Source 2 : shared.match_participants pour ce xuid (cross-player dedup).
	if sharedDB != nil && strings.TrimSpace(xuid) != "" {
		rows, err := sharedDB.QueryContext(ctx, "SELECT DISTINCT match_id FROM match_participants WHERE xuid = ?", xuid)
		if err == nil {
			for rows.Next() {
				var id string
				if scanErr := rows.Scan(&id); scanErr == nil {
					known[id] = true
				}
			}
			_ = rows.Close()
		}
		// Erreur ignorée idem : si la table n'existe pas (cas tests minimaux),
		// fallback gracieux sur la source 1 seule.
	}

	return known, nil
}

// runPostSyncPipeline : dÃ©placÃ© vers engine_postsync.go (refactor 2026-05-21).
// Le pipeline post-sync enchaÃ®ne notamment : batchComputeEngagementScores
// (calcul des paces) PUIS batchRecomputeCoefficients (recompute du coef
// team_share depuis la mÃ©diane des paces) â€” voir engine_postsync.go Ã©tapes
// 1.5 et 1.5.b. Le hook recompute doit rester APRÃˆS le compute sinon le
// coef reste Ã  1.0 cold-start (cf. TestRegressionB5_RecomputeCoefHookWired).

// runCSRSnapshotSync, runAchievementsSync, RunAchievementsOnly,
// resolveAccessTokenFromDB : dÃ©placÃ©s vers engine_postsync.go (refactor 2026-05-21).
