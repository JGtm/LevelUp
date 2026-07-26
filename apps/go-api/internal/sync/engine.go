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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/pool"
	"levelup/go-api/internal/platform/dblease"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/port"

	"golang.org/x/sync/errgroup"
)

const (
	// historyPageSize est le nombre de matchs demandés par page API.
	historyPageSize = 25

	// postsyncEventsBurstChunk / postsyncWeaponsBurstChunk : taille des lots des
	// étapes film du post-sync (events / weapon kills).
	//
	// Depuis le split COLLECT/FLUSH (v7.3, engine_postsync_films.go) le fetch film
	// n'est PLUS dans le burst : le writer n'est acquis que pour le flush SQL du
	// lot. Ces constantes ne bornent donc plus la durée du lease (elle ne dépend
	// que du coût des INSERT) mais la MÉMOIRE : nombre de films téléchargés/parsés
	// avant écriture. Valeurs conservées à l'identique — le pic mémoire d'un cycle
	// reste celui d'avant le split (weapons plus lourd → lot plus petit).
	postsyncEventsBurstChunk  = 3
	postsyncWeaponsBurstChunk = 2

	// syncFetchParallelism borne le nombre de fetchMatchData simultanés dans la
	// Phase 2 d'un cycle joueur. Volontairement < taille de pool typique (~7)
	// pour laisser des slots au trafic user-facing et éviter que N joueurs
	// synchronisés en parallèle ne saturent le pool + les exchanges XBL. Le pool
	// reste le plafond dur de la concurrence API réelle ; ceci lisse le burst.
	syncFetchParallelism = 4
)

// SyncEngine orchestre la synchronisation des données Halo d'un joueur.
// FriendsLoader retourne la liste courante des amis configurés (typiquement
// settings.FriendGamertags). Nil → feature désactivée (legacy / pas de wiring).
type FriendsLoader func() ([]string, error)

type SyncEngine struct {
	gamertag string
	xuid     string
	// repoRoot : racine du repo (cfg.RepoRoot). Sert à résoudre le dossier global
	// watcher_tokens (data/auth/watcher_tokens/) pour la résolution store-first de
	// l'access_token achievements (ADR 0023). Cf. resolveAchievementsAccessToken.
	repoRoot       string
	titleSlug      string
	playerDBPath   string
	sharedDBPath   string
	metadataDBPath string
	// pveDBPath : shared_pve.duckdb du titre (stats Firefight). Lu en RO pour les
	// citations pve_stat (BUG A / I7). Peut pointer un fichier inexistant sur un
	// titre sans Firefight → lecture dégradée gracieusement (OpenPveReadForCitations).
	pveDBPath    string
	syncCacheDir string // Phase 2 refactor Collect→Persist : data/sync_cache/ root (cf. PathResolver.SyncCacheDir)
	tokens       *domain.HaloTokens
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
	// qui appellent buildBatchFromFetchedMatch directement → l'enrichissement
	// devient no-op et la sync reste fonctionnelle (UUID préservé comme avant).
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

	// batchQueue (Phase 3 refactor Collect→Persist) — si non-nil,
	// submitMatchAsBatch passe par queue.Submit (WAL + worker async) au lieu
	// d'appeler les Persisters directement. À la fin du cycle, run() appelle
	// queue.Drain pour attendre que tous les batches soient ACKed (parité
	// observable avec le path sync).
	//
	// Si nil : submitMatchAsBatch reste synchrone (Phase 2.3 — direct
	// Persister.Persist sans WAL). Reset à nil = pas d'async layer.
	batchQueue *persist.BatchQueue

	// assetPool (optionnel) — POOL de tokens UNIFIÉ (auth/pool, la même source que
	// tous les syncs) pour la résolution autonome des noms d'assets au sync : si
	// non-nil, un pré-pass (resolveCycleAssets) peuple metadata.asset_translations
	// pour les assets neufs du cycle AVANT l'écriture registry (noms résolus dès le
	// 1er passage, sans heal/backfill/action admin). GameCMS exige un token Spartan.
	// Injecté via WithAssetNameResolution ; nil → feature off. Cf. assetnames_wiring.go.
	assetPool pool.Pool
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
// RunBackfillLUSRDryRun (LUSR v2 : le v1 RunBackfillLUSR est mort — cf.
// RecomputeLUSRCanonicalForPlayer), RunBackfillCSR, RunBackfillPerf,
// RunBackfillComebackBadges, loadMedalExploitMap, loadMedalExploitMapBestEffort,
// selectMatchesForComebackBadges, loadAllMatchIDsForPlayer, loadFlaggedMatchIDs :
// déplacés vers engine_backfills.go (refactor 2026-05-21).

// historyPaginationInputs regroupe l'état run()-local consommé par
// paginateAndPersistHistory (garde la signature ≤ 5 params — CLAUDE.md).
type historyPaginationInputs struct {
	client   HaloClient
	opts     domain.SyncOptions
	known    map[string]bool
	sharedDB *sql.DB
	playerDB *sql.DB
	isDelta  bool
}

// paginateAndPersistHistory parcourt l'historique paginé (GetMatchHistory) et, page par
// page, filtre les matchs connus, fetche en parallèle les inconnus (borné à
// syncFetchParallelism) puis les persiste séquentiellement (order-preserving). Arrêts :
// MaxMatches atteint, page vide, match connu rencontré en delta (APRÈS flush des unknowns
// déjà collectés de la page), ou page entièrement connue en delta. Extrait de run() (K2b,
// 2026-07-06) — mute `result` (compteurs + warnings).
//
// la découper disperserait l'état partagé processed/toFetch/stopAfterFlush et nuirait à
// la lisibilité du critère d'arrêt delta (bug 2026-05-21 : flush-avant-stop).
//
//nolint:funlen // boucle de pagination cohérente (filtre → fetch → persist par page) :
func (e *SyncEngine) paginateAndPersistHistory(ctx context.Context, in historyPaginationInputs, result *domain.SyncResult) {
	processed := 0
	start := 0

	for processed < in.opts.MaxMatches {
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
		entries, err := in.client.GetMatchHistory(ctx, fmt.Sprintf("xuid(%s)", e.xuid), in.opts.MatchType, start, historyPageSize)
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
			if processed >= in.opts.MaxMatches {
				break
			}
			if in.known[entry.MatchID] {
				result.MatchesSkipped++
				if in.isDelta {
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
			// Borne la concurrence du fan-out fetch : sans SetLimit, une page delta
			// initiale lançait une goroutine PAR match inconnu (des dizaines/centaines
			// d'un coup). Le pool cappe déjà l'API concurrente à sa taille, mais on
			// évite ici l'explosion de goroutines et on lisse la pression (pool +
			// exchanges XBL) pour laisser de la marge au trafic user-facing.
			eg.SetLimit(syncFetchParallelism)
			for i, matchID := range toFetch {
				i, matchID := i, matchID // Capturer pour closure
				eg.Go(func() error {
					fm, err := e.fetchMatchData(egCtx, in.client, matchID, in.opts)
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

			// ─── Pré-pass : résolution des noms d'assets (primary write) ───
			// Peuple metadata.asset_translations pour les assets neufs du cycle
			// AVANT la phase d'insert, pour qu'EnrichRegistryFromMetadata
			// (submitOrInsertMatch) écrive un vrai nom dès le 1er passage.
			// Best-effort, gated (assetPool nil → no-op).
			e.resolveCycleAssets(ctx, fetchedMatches)

			// ─── Phase 3 : Insert séquentiel (order-preserving) ───
			for i, fm := range fetchedMatches {
				if fetchErrors[i] != nil {
					// Fetch échoué, skip insert
					continue
				}
				if fm == nil {
					continue
				}

				if err := e.persistFetchedMatch(ctx, in.sharedDB, in.playerDB, result, fm); err != nil {
					slog.WarnContext(ctx, "sync: persistFetchedMatch échoué",
						"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err,
					)
					result.AddWarning(fmt.Sprintf("persistFetchedMatch(%s): %v", fm.MatchID, err))
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
		if in.isDelta && allKnown {
			break
		}
		start += len(entries)
	}
}

// run est le cÅ“ur du moteur de sync. isDelta=true → stop dès un match connu.
func (e *SyncEngine) run(ctx context.Context, opts domain.SyncOptions, isDelta bool) (domain.SyncResult, error) {
	result := domain.SyncResult{StartedAt: time.Now()}
	// mode + sa forme titre-case pour le nom d'event (remplace strings.Title,
	// déprécié Go 1.18+ et cassé sur l'Unicode). Seuls 2 modes existent.
	mode, modeTitle := "full", "Full"
	if isDelta {
		mode, modeTitle = "delta", "Delta"
	}

	// Sprint B1 commit 16 : crée un event_id qui sera ajouté à TOUS les logs
	// émis depuis ce ctx (cf. ContextHandler) — permet de grep cross-module
	// dans logs/{sync,provider,pool,...}.log pour reconstituer le timeline
	// d'un sync donné.
	ctx, eventID := logging.WithEvent(ctx, "sync.Run"+modeTitle)
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

	// Phase 4 plan stabilisation 2026-05-22 : capture du snapshot before-sync
	// pour la couche post-sync (delta notifications + pipeline progression V2).
	// Le runner est nil si non injecté (legacy / tests) — feature off.
	var postSyncFinalizer port.PostSyncFinalizer
	if e.postSyncRunner != nil && e.postSyncSlug != "" {
		postSyncFinalizer = e.postSyncRunner.BeforeSync(ctx, e.postSyncSlug)
	}

	// Le finalizer est invoqué en succès uniquement (finalizerArmed positionné
	// juste avant le `return result, nil` terminal) — sur erreur il ne tourne
	// pas, pour ne pas émettre de deltas sur un snapshot post-sync incohérent.
	//
	// CRITIQUE (fix auto-contention shared, ADR 0016) : le finalizer lit la
	// shared DB (pipeline progression V2 → loadProgressionMatches). Si on
	// l'appelle alors que CE sync tient encore le shared writer RW (le
	// `defer releaseShared()` / `defer postRls()` ne tournent qu'au retour),
	// le `SharedReadDB().Get` attend un retour RO que le même run empêche →
	// `context deadline exceeded` systématique → progression jamais persistée.
	//
	// On enregistre donc le finalizer en `defer` AVANT les leases writer :
	// grâce au LIFO, il s'exécute APRÈS tous les release() de writer, shared
	// repassée en RO. Le ctx est détaché (WithoutCancel) pour survivre à une
	// annulation du parent juste après le retour de run().
	var finalizerArmed bool
	if postSyncFinalizer != nil {
		defer func() {
			if !finalizerArmed {
				return
			}
			postSyncFinalizer(context.WithoutCancel(ctx))
		}()
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
	// Étape 0 attribution : label du détenteur pour la ventilation rw_window.
	sharedDB, releaseShared, err := e.acquireSharedWriter(ctxkeys.WithDBWriterLabel(ctx, "sync_v1_run"))
	if err != nil {
		slog.ErrorContext(ctx, "sync: ouverture shared DB échouée", "gamertag", e.gamertag, "db", e.sharedDBPath, "err", err)
		return result, fmt.Errorf("run acquireSharedWriter: %w", err)
	}
	defer releaseShared()

	// metaDB best-effort : utilisé par EnrichRegistryFromMetadata pour résoudre
	// les UUIDs bruts en noms canoniques EN avant l'INSERT match_registry.
	// Échec d'ouverture → enrichissement désactivé pour ce run, sync continue.
	if e.metadataDBPath != "" {
		// OpenReadForQuery RÉUTILISE le handle metadata déjà en cache (RW tenu par
		// main.go via OpenReadWriteShared, sinon RO) au lieu d'ouvrir un 2e handle.
		// Forcer OpenReadOnly ("ro:"+path) ici ouvrait une SECONDE instance DuckDB
		// alors que le serveur tient déjà metadata en RW ("rw:"+path) → échec
		// "Can't open ... with a different configuration" → e.metaDB nil →
		// EnrichRegistryFromMetadata ET la résolution des noms d'assets désactivés
		// silencieusement (cf. ADR 0016, thought_log "different configuration").
		// Le handle ainsi obtenu est RW en prod : il sert la LECTURE (enrich) ET
		// l'écriture basse-fréquence (asset_translations via ops.UpsertAssetTranslation,
		// SELECT-then-write ART-safe). En CLI/test sans handle partagé, retombe sur un
		// OpenReadOnly propre (lecture OK ; l'écriture résolution reste best-effort).
		metaSQL, releaseMeta, metaErr := duckdbpkg.OpenReadForQuery(e.metadataDBPath)
		if metaErr != nil {
			slog.WarnContext(ctx, "sync: ouverture metadata DB échouée — enrich registry désactivé",
				"db", e.metadataDBPath, "err", metaErr)
		} else {
			e.metaDB = metaSQL
			defer func() {
				releaseMeta()
				e.metaDB = nil
			}()
		}
	}
	slog.DebugContext(ctx, "sync: DBs ouvertes", "gamertag", e.gamertag)

	// ─── Match IDs déjà connus (player DB) ───────────────────────────────────
	// Known set étendu : player_match_enrichment ∪ shared.match_participants
	// WHERE xuid=e.xuid. Permet de skipper le fetch API quand un autre joueur
	// du même cycle a déjà ingéré le match en shared (cross-player dedup).
	known, err := loadKnownMatchIDs(ctx, playerDB, sharedDB, e.xuid)
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

	// Cache fetch intermédiaire (cf. REFACTOR_COLLECT_PERSIST §3.5 / §10 Q7).
	// Wrap le client avec un cache fichier sous data/sync_cache/{cycle_id}/.
	// Désactivable via LEVELUP_PERSIST_NO_FETCH_CACHE=1. cycle_id = eventID
	// du run (créé via logging.WithEvent en début de run).
	//
	// L'eventID contient `:` (format `sync.RunDelta:abc123`) qui est interdit
	// dans les noms de fichier/dossier sur Windows. On normalise `:` → `_`
	// pour le nom du dossier de cache.
	if os.Getenv("LEVELUP_PERSIST_NO_FETCH_CACHE") != "1" && e.syncCacheDir != "" {
		safeCycleID := strings.ReplaceAll(eventID, ":", "_")
		cacheDir := filepath.Join(e.syncCacheDir, safeCycleID)
		client = NewCachedHaloClient(client, FetchCacheConfig{CacheDir: cacheDir})
		slog.InfoContext(ctx, "sync: cache fetch intermédiaire actif",
			"gamertag", e.gamertag, "cache_dir", cacheDir)
	}

	// ─── Pagination de l'historique ────────────────────────────────────────────
	// Boucle filtre → fetch parallèle → persist par page, extraite en méthode (K2b).
	e.paginateAndPersistHistory(ctx, historyPaginationInputs{
		client: client, opts: opts, known: known,
		sharedDB: sharedDB, playerDB: playerDB, isDelta: isDelta,
	}, &result)

	slog.InfoContext(ctx, "sync: boucle pagination terminée",
		"gamertag", e.gamertag, "mode", mode,
		"inserted", result.MatchesInserted, "skipped", result.MatchesSkipped,
		"warnings", len(result.Warnings),
	)

	// ─── Drain async batch queue ────────────────────────────────────────
	// En mode async (batchQueue non-nil), attendre que tous les batches
	// soumis pendant cette boucle aient été persistés par les workers
	// AVANT le post-sync compute. Sinon, runConditionalPostSync lirait
	// des données qui ne sont pas encore en DB.
	// drainErr est déclaré au scope run() : il est relu plus bas par la
	// réconciliation post-Drain (le bloc `if e.batchQueue != nil` se ferme AVANT
	// le pipeline post-sync, donc un `:=` interne ne serait pas visible).
	var drainErr error
	// postShared : accès shared du pipeline post-sync (étape 1 contention).
	// Posé par le post-drain (bursts par défaut, pinned en rollback/legacy) ;
	// chemin non-batch → pinned sur le writer primaire (voir plus bas).
	var postShared *SharedAccess
	if e.batchQueue != nil {
		// Libérer les write leases AVANT Drain — le Worker (CombinedPersister)
		// doit acquérir ces mêmes leases pour persister. Sans release ici,
		// le Worker bloque indéfiniment → PendingCount reste > 0 → timeout 60s.
		//
		// writerPlayer.Release() est idempotent (sync.Once) → le defer reste safe.
		// releaseShared n'est pas idempotent : closure-swap pour éviter
		// le double-release depuis le defer.
		writerPlayer.Release()
		_ = playerHandle.Close()
		oldReleaseShared := releaseShared
		releaseShared = func() {} //nolint:ineffassign // closure-swap intentionnel : le defer lit la variable, pas la valeur
		oldReleaseShared()
		playerDB = nil // invalide après Close
		sharedDB = nil // invalide après release

		drainCtx, drainCancel := context.WithTimeout(ctx, 60*time.Second)
		drainErr = e.batchQueue.Drain(drainCtx)
		if drainErr != nil {
			slog.WarnContext(ctx, "sync: batch queue drain incomplet",
				"gamertag", e.gamertag, "err", drainErr)
			result.AddWarning(fmt.Sprintf("queue.Drain: %v", drainErr))
		}
		drainCancel()

		// Re-acquisition post-Drain pour le pipeline post-sync.
		// Echec → DBs restent nil → post-sync skippé proprement (pas de panic).
		//
		// Phase 7 PLAN_FIX_SYNC_RELIABILITY_2026-05-24 (Option C — Status quo
		// ameliore) : observabilite explicite sur le temps d'attente du
		// shared writer lease. En auto-sync parallele (pool=4), tous les
		// joueurs font la queue sur ce lease pour le post-sync (sessions,
		// LUSR upsert, dominance flags). Un wait > 30s signale une
		// serialisation pathologique qui justifierait Phase 7.A (batch
		// post-sync global) en PR dedie.
		postPH, phErr := OpenPlayerDB(e.playerDBPath)
		if phErr != nil {
			slog.WarnContext(ctx, "sync: post-drain OpenPlayerDB echoue — post-sync skippe",
				"gamertag", e.gamertag, "err", phErr)
		} else {
			defer postPH.Close()
			playerDB = postPH.SQLDb()
			if PostSyncBurstEnabled() && e.sharedProvider != nil {
				// Étape 1 contention (défaut) : PAS d'acquisition writer upfront —
				// le pipeline post-sync ouvre des segments Read RO et des bursts
				// Write courts (labels sync_v1_postsync/<étape>). Fenêtre RW ~0 en
				// stationnaire ; fin de la sérialisation post-sync inter-joueurs.
				postShared = NewBurstSharedAccess(
					func(c context.Context) (*sql.DB, func(), error) { return e.acquireSharedWriter(c) },
					e.sharedProvider.Get,
					"sync_v1_postsync")
				if wp, wpErr := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer); wpErr == nil {
					defer wp.Release()
				} else {
					slog.WarnContext(ctx, "sync: post-drain AcquireWriterCtx echoue",
						"gamertag", e.gamertag, "err", wpErr)
				}
			} else {
				// ROLLBACK (LEVELUP_POSTSYNC_BURST=0) ou mode legacy sans provider :
				// chemin pinned historique (writer tenu pendant tout le post-sync).
				leaseStart := time.Now()
				// Étape 0 attribution : détenteur LONG identifié — label dédié.
				postSDB, postRls, sErr := e.acquireSharedWriter(ctxkeys.WithDBWriterLabel(ctx, "sync_v1_postsync"))
				leaseWaitMs := time.Since(leaseStart).Milliseconds()
				if sErr != nil {
					slog.WarnContext(ctx, "sync: post-drain acquireSharedWriter echoue — post-sync skippe",
						"gamertag", e.gamertag, "err", sErr,
						"lease_wait_ms", leaseWaitMs)
					playerDB = nil
				} else {
					defer postRls()
					sharedDB = postSDB
					// Phase 7C : log explicite si attente > 1s (serialisation
					// notable). Au-dessus de 30s = goulot post-sync confirme.
					if leaseWaitMs > 1000 {
						slog.WarnContext(ctx, "sync: post-drain shared lease wait > 1s — serialisation post-sync",
							"gamertag", e.gamertag, "lease_wait_ms", leaseWaitMs)
					} else {
						slog.DebugContext(ctx, "sync: post-drain shared lease acquis",
							"gamertag", e.gamertag, "lease_wait_ms", leaseWaitMs)
					}
					if wp, wpErr := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer); wpErr == nil {
						defer wp.Release()
					} else {
						slog.WarnContext(ctx, "sync: post-drain AcquireWriterCtx echoue",
							"gamertag", e.gamertag, "err", wpErr)
					}
				}
			}
		}
	}

	// Réconciliation post-Drain : en mode async, MatchesInserted/InsertedMatchIDs
	// ont été peuplés au Submit (durabilité WAL), PAS à l'ACK (persistance DB).
	// Si le Drain a échoué (timeout/circuit-breaker), certains matchs comptés ne
	// sont PAS dans match_registry. On relit la vérité terrain pour ne pas lancer
	// le post-sync (notamment processWeaponKillsInline, ~285s/cycle de re-download
	// film) sur des matchs fantômes. Lecture seule (anti-ART). Les WAL non-ACKés
	// restent sur disque et seront rejoués au prochain boot (RecoverPending).
	if e.batchQueue != nil && drainErr != nil {
		if sharedDB != nil {
			reconcileInsertedAgainstRegistry(ctx, sharedDB, &result, e.gamertag)
		} else if postShared != nil {
			// Mode bursts : lecture seule via un segment Read court (le writer
			// n'est plus tenu à ce stade).
			if roDB, roRel, roErr := postShared.Read(ctx); roErr == nil {
				reconcileInsertedAgainstRegistry(ctx, roDB, &result, e.gamertag)
				roRel()
			} else {
				slog.WarnContext(ctx, "sync: reconcile post-drain — lecture shared indisponible",
					"gamertag", e.gamertag, "err", roErr)
			}
		}
	}

	// ─── Pipeline post-sync ─────────────────────────────────────────────────────
	// Étape 1 contention : postShared est posé par le post-drain (bursts par
	// défaut, pinned en rollback/legacy). Chemin non-batch : le writer primaire
	// est encore tenu → pinned (byte-identique).
	if postShared == nil && sharedDB != nil {
		postShared = NewPinnedSharedAccess(sharedDB)
	}
	var postResult domain.PostSyncResult
	if playerDB != nil && postShared != nil {
		postResult = e.runConditionalPostSync(ctx, playerDB, postShared, client, result.MatchesInserted, result.InsertedMatchIDs)
	} else if e.batchQueue != nil {
		slog.WarnContext(ctx, "sync: post-sync skippe — DBs indisponibles apres drain async",
			"gamertag", e.gamertag)
	}
	if result.MatchesInserted > 0 || postResult.AchievementsSynced {
		result.PostSync = &postResult
		slog.InfoContext(ctx, "sync: pipeline post-sync terminé",
			"gamertag", e.gamertag,
			"perf_scores", postResult.PerfScoresComputed,
			"lusr_updated", postResult.LUSRUpdated,
			"views_refreshed", postResult.ViewsRefreshed,
			"achievements_synced", postResult.AchievementsSynced,
			"citations_computed", postResult.CitationsComputed,
			"dominance_flags", postResult.DominanceFlagsComputed,
			"fatal_errors", len(postResult.FatalErrors),
		)
	}
	// Phase 5 ART — Status sync honnête : propager les FATAL DuckDB du
	// post-sync vers result.Errors pour que SyncResult.Status() retourne
	// "partial_success" au lieu de "success" quand une étape critique a
	// invalidé une DB. Les WARN existants restent intacts pour la trace ;
	// seuls les FATAL DB (IsInvalidatedError) sont promus en erreur sync.
	for _, fatalErr := range postResult.FatalErrors {
		result.AddError("post-sync " + fatalErr)
	}

	// ─── sync_meta ──────────────────────────────────────────────────────────────
	if playerDB != nil {
		if err := SetSyncMeta(ctx, playerDB, "last_delta_sync", time.Now().UTC().Format(time.RFC3339)); err != nil {
			result.AddWarning(fmt.Sprintf("SetSyncMeta: %v", err))
		}
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

	// Phase 4 plan stabilisation 2026-05-22 : le finalizer post-sync
	// (notifications delta + pipeline progression V2) est invoqué via le defer
	// armé en début de run — APRÈS la libération du shared writer RW (sinon la
	// lecture shared de la progression s'auto-bloque en attente du retour RO,
	// cf. ADR 0016). Ici on se contente d'armer le flag sur le chemin de succès.
	finalizerArmed = true
	return result, nil
}

// runConditionalPostSync, hasMatchesNeedingScoreRefresh : déplacés vers
// engine_postsync.go (refactor 2026-05-21).

// fetchedMatch struct + fetchMatchData + hasAnyTeamMMR : dans engine_fetch.go.
// Pipeline parallèle fetch (Phase 2 errgroup) + persist séquentiel (Phase 3 :
// persistFetchedMatch → submitMatchAsBatch INSERT-only).

// ProcessHighlightEvents : dans engine_highlight_events.go. Parse + insert
// events (path standalone exposé pour les outils de replay).

// reconcileInsertedAgainstRegistry retire de result.InsertedMatchIDs (et décrémente
// MatchesInserted, incrémente MatchesSkipped) les match_id comptés au Submit mais
// ABSENTS de match_registry — cas du mode async où le Drain a échoué et où certains
// batches n'ont pas été persistés. Lecture seule (anti-ART). No-op si sharedDB nil
// ou liste vide.
//
// Bénéfice du doute (aligné sur le pre-check Submit, engine_batch_path.go) : une
// erreur de requête n'est PAS une preuve d'absence → le match est CONSERVÉ. On ne
// retire que les fantômes confirmés absents (qErr == nil && !exists), pour ne pas
// priver à tort un match réellement persisté de son post-sync (films/scores) à
// cause d'une erreur transitoire DuckDB.
func reconcileInsertedAgainstRegistry(ctx context.Context, sharedDB *sql.DB, result *domain.SyncResult, gamertag string) {
	if sharedDB == nil || len(result.InsertedMatchIDs) == 0 {
		return
	}
	confirmed := make([]string, 0, len(result.InsertedMatchIDs))
	for _, mid := range result.InsertedMatchIDs {
		var exists bool
		qErr := sharedDB.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM match_registry WHERE match_id = ?)`, mid).Scan(&exists)
		if qErr != nil || exists {
			confirmed = append(confirmed, mid)
		}
	}
	dropped := len(result.InsertedMatchIDs) - len(confirmed)
	if dropped == 0 {
		return
	}
	slog.WarnContext(ctx, "sync: drain incomplet — matchs non confirmés retirés du post-sync",
		"gamertag", gamertag, "dropped", dropped, "confirmed", len(confirmed))
	result.InsertedMatchIDs = confirmed
	result.MatchesInserted -= dropped
	if result.MatchesInserted < 0 {
		result.MatchesInserted = 0
	}
	result.MatchesSkipped += dropped
}

// loadKnownMatchIDs retourne l'ensemble des match_ids déjà présents dans
// player_match_enrichment (player DB).
// loadKnownMatchIDs construit le set des match_id "connus" pour ce joueur.
//
// Source 1 — playerDB.player_match_enrichment : matchs déjà ingérés dans la
// pipeline complète (registry + participants + enrichment per-player).
//
// Source 2 — sharedDB.match_participants WHERE xuid=? : matchs déjà présents
// en shared pour ce joueur (typiquement parce qu'un AUTRE joueur du même
// cycle de sync a déjà fait le fetch + persist via le chemin batch — qui
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
	if rows, err := playerDB.QueryContext(ctx, "SELECT match_id FROM player_match_enrichment_latest"); err == nil {
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
	//
	// Phase 3 du PLAN_FIX_SYNC_RELIABILITY_2026-05-24 :
	//   - Cast defensif `xuid || ''` aligne sur recompute_after_art_rebuild.go:156.
	//     Empeche un mismatch silencieux si la colonne xuid drift en type sur un
	//     titre futur (UBIGINT vs VARCHAR), incident observe en prod 2026-05-24
	//     ou known_count=22 alors que total_matches=30 pour XxDaemonGamerxX.
	//   - Erreur query : warn explicite (plus de swallow silencieux) pour
	//     visibilite. Cas pathologique = known set partiel → 285s perdues sur
	//     un joueur inactif.
	if sharedDB != nil && strings.TrimSpace(xuid) != "" {
		rows, err := sharedDB.QueryContext(ctx,
			"SELECT DISTINCT match_id FROM match_participants WHERE xuid || '' = ?", xuid)
		if err != nil {
			slog.WarnContext(ctx,
				"loadKnownMatchIDs: shared.match_participants query failed — known set partiel (source 1 seule)",
				"xuid", xuid, "err", err)
		} else {
			addedFromShared := 0
			for rows.Next() {
				var id string
				if scanErr := rows.Scan(&id); scanErr == nil {
					if !known[id] {
						addedFromShared++
					}
					known[id] = true
				}
			}
			_ = rows.Close()
			slog.DebugContext(ctx, "loadKnownMatchIDs: source 2 (shared) ajoutee",
				"xuid", xuid, "added_from_shared", addedFromShared, "total_known", len(known))
		}
	}

	return known, nil
}

// runPostSyncPipeline : déplacé vers engine_postsync.go (refactor 2026-05-21).
// Le pipeline post-sync enchaîne notamment : batchComputeEngagementScores
// (calcul des paces) PUIS batchRecomputeCoefficients (recompute du coef
// team_share depuis la médiane des paces) — voir engine_postsync.go étapes
// 1.5 et 1.5.b. Le hook recompute doit rester APRÈS le compute sinon le
// coef reste à 1.0 cold-start (cf. TestRegressionB5_RecomputeCoefHookWired).

// runCSRSnapshotSync, runAchievementsSync, RunAchievementsOnly,
// resolveAchievementsAccessToken : déplacés vers engine_postsync.go (refactor 2026-05-21).
