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
	"strconv"
	"strings"
	"sync"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
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
	globalDB, globalCleanup, err := openGlobalDB(e.globalDBPath)
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
	known, err := loadKnownMatchIDs(playerDB)
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

		for _, entry := range entries {
			if processed >= opts.MaxMatches {
				break
			}
			if known[entry.MatchID] {
				result.MatchesSkipped++
				if isDelta {
					slog.InfoContext(ctx, "sync: match connu rencontré — arrêt delta",
						"gamertag", e.gamertag, "match_id", entry.MatchID,
						"processed", processed, "skipped", result.MatchesSkipped,
					)
					goto done
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

		if isDelta && allKnown {
			break
		}
		start += len(entries)
	}

done:
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
	if err := SetSyncMeta(playerDB, "last_delta_sync", time.Now().UTC().Format(time.RFC3339)); err != nil {
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

// processMatch récupère, transforme et insère un match dans les deux DBs.
func (e *SyncEngine) processMatch(
	ctx context.Context,
	client HaloClient,
	sharedDB, playerDB, globalDB *sql.DB,
	result *domain.SyncResult,
	matchID string,
	opts domain.SyncOptions,
) error {
	start := time.Now()
	slog.DebugContext(ctx, "processMatch: début", "gamertag", e.gamertag, "match_id", matchID)

	matchJSON, err := client.GetMatchStats(ctx, matchID)
	if err != nil {
		slog.WarnContext(ctx, "processMatch: GetMatchStats échoué",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
		return fmt.Errorf("GetMatchStats: %w", err)
	}

	// ─── match_registry ────────────────────────────────────────────────────────
	reg, err := ExtractRegistry(matchJSON, e.gamertag)
	if err != nil {
		slog.WarnContext(ctx, "processMatch: ExtractRegistry échoué",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
		return fmt.Errorf("ExtractRegistry: %w", err)
	}
	// Enrichissement post-Extract : résout les UUIDs bruts en noms canoniques
	// via metadata.asset_translations[en-US] AVANT l'INSERT, pour ne pas
	// stocker `playlist_name = playlist_id` quand l'API Halo n'a pas retourné
	// de PublicName. Best-effort : nil metaDB → no-op (préserve le fallback
	// historique). Cf. thought_log 2026-05-09.
	if err := EnrichRegistryFromMetadata(ctx, e.metaDB, reg); err != nil {
		slog.WarnContext(ctx, "processMatch: EnrichRegistryFromMetadata non-bloquant",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
	}
	if err := InsertRegistryIfNotExists(sharedDB, *reg); err != nil {
		slog.ErrorContext(ctx, "processMatch: InsertRegistry échoué",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
		return fmt.Errorf("InsertRegistry: %w", err)
	}

	// ─── match_participants ────────────────────────────────────────────────────
	if opts.WithParticipants {
		participants := ExtractParticipants(matchJSON)

		// Garantir gamertag sur la row du joueur synchronisé.
		ensureGamertagForSelf(participants, e.xuid, e.gamertag)

		// Skill API (séparé du stats endpoint) : team_mmr, enemy_mmr, kills/deaths_expected.
		// Non-bloquant : un échec produit un warning mais le sync continue.
		if xuids := ParticipantXUIDs(participants); len(xuids) > 0 {
			skillData, skillErr := client.GetMatchSkill(ctx, matchID, xuids)
			if skillErr != nil {
				slog.WarnContext(ctx, "processMatch: GetMatchSkill échoué (continuing without skill)",
					"gamertag", e.gamertag, "match_id", matchID, "err", skillErr,
				)
				result.Warnings = append(result.Warnings, fmt.Sprintf("skill %s: %v", matchID, skillErr))
			} else if len(skillData) > 0 {
				participants = MergeSkillIntoParticipants(participants, skillData)
				slog.DebugContext(ctx, "processMatch: skill merged",
					"match_id", matchID, "players_with_skill", len(skillData),
				)
				// CSR par-match : pour les matchs classés, le payload skill
				// contient RankRecap.PostMatchCsr. On persiste côté player DB.
				// Non-bloquant : tout échec laisse le sync continuer.
				if row := ExtractCSRRowIfRanked(reg, skillData[e.xuid]); row != nil {
					if csrErr := UpsertCSRRow(playerDB, row); csrErr != nil {
						slog.WarnContext(ctx, "processMatch: UpsertCSRRow échoué",
							"gamertag", e.gamertag, "match_id", matchID, "err", csrErr,
						)
					} else {
						slog.DebugContext(ctx, "processMatch: CSR row écrite",
							"match_id", matchID, "tier", row.Tier, "tier_label", row.TierLabel,
						)
					}
				}
			}
		}

		if err := InsertParticipants(sharedDB, participants); err != nil {
			slog.ErrorContext(ctx, "processMatch: InsertParticipants échoué",
				"gamertag", e.gamertag, "match_id", matchID, "count", len(participants), "err", err,
			)
			return fmt.Errorf("InsertParticipants: %w", err)
		}
		result.ParticipantsDone += len(participants)

		aliased := 0
		for _, p := range participants {
			if p.Gamertag != nil && *p.Gamertag != "" {
				// P5.3 : écriture dans la DB globale xbox_aliases.duckdb.
				if globalDB != nil {
					_ = UpsertXUIDAlias(globalDB, p.XUID, *p.Gamertag)
				}
				aliased++
			}
		}
		slog.DebugContext(ctx, "processMatch: participants insérés",
			"match_id", matchID, "participants", len(participants), "aliases_upserted", aliased,
		)
	}

	// ─── medals_earned ─────────────────────────────────────────────────────────
	if opts.WithMedals {
		medals := ExtractMedals(matchJSON)
		if err := InsertMedals(sharedDB, medals); err != nil {
			slog.ErrorContext(ctx, "processMatch: InsertMedals échoué",
				"gamertag", e.gamertag, "match_id", matchID, "count", len(medals), "err", err,
			)
			return fmt.Errorf("InsertMedals: %w", err)
		}
		result.MedalsInserted += len(medals)
		slog.DebugContext(ctx, "processMatch: médailles insérées",
			"match_id", matchID, "medals", len(medals),
		)
	}

	// ─── highlight_events + killer_victim_pairs ──────────────────────────────────────
	if opts.WithHighlightEvents {
		if err := ProcessHighlightEvents(ctx, client, sharedDB, globalDB, matchID, result); err != nil {
			// Non-bloquant : on logge et on continue (pas de return).
			slog.WarnContext(ctx, "processMatch: highlight_events non chargés",
				"gamertag", e.gamertag, "match_id", matchID, "err", err,
			)
			result.Warnings = append(result.Warnings, fmt.Sprintf("highlight_events %s: %v", matchID, err))
		}
	}

	// ─── player_match_enrichment (player DB) ───────────────────────────────────
	if err := UpsertPlayerEnrichment(playerDB, matchID, ""); err != nil {
		slog.ErrorContext(ctx, "processMatch: UpsertPlayerEnrichment échoué",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
		return fmt.Errorf("UpsertPlayerEnrichment: %w", err)
	}

	// ─── personal_score_awards (player DB) ─────────────────────────────────────
	psaRows := ExtractPersonalScoreAwards(matchJSON, matchID, e.xuid)
	if len(psaRows) > 0 {
		if err := InsertPersonalScoreAwards(playerDB, matchID, e.xuid, psaRows); err != nil {
			slog.WarnContext(ctx, "processMatch: InsertPersonalScoreAwards échoué",
				"gamertag", e.gamertag, "match_id", matchID, "err", err,
			)
			result.Warnings = append(result.Warnings, fmt.Sprintf("psa %s: %v", matchID, err))
		}
	}

	result.MatchesInserted++
	result.InsertedMatchIDs = append(result.InsertedMatchIDs, matchID)
	slog.DebugContext(ctx, "processMatch: terminé",
		"gamertag", e.gamertag, "match_id", matchID,
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return nil
}

// ─── Fetch phases (parallel fetch + sequential insert) ───

// fetchedMatch contient les données extraites d'un GetMatchStats, prêtes pour insertion.
// Utilisé pour paralléliser les fetches tout en gardant les inserts séquentiels.
type fetchedMatch struct {
	MatchID        string
	Registry       *MatchRegistryRow
	Participants   []ParticipantRow
	Medals         []MedalRow
	PSA            []PersonalScoreAwardRow // PersonalScores du joueur courant (player DB)
	HighlightData  []byte                  // Raw highlight events chunk (ou nil si absent)
	FilmMajorVer   int
	HasHighlights  bool
	HighlightError error // Non-bloquant si présent
	SkillError     error // Non-bloquant si présent
	// CSRRow : ligne CSR à insérer côté player DB. Renseignée uniquement
	// pour les matchs classés dont le payload skill contient RankRecap.
	// Inséré dans insertFetchedMatch.
	CSRRow *MatchCSRRow
}

// fetchMatchData exécute le fetch et l'extraction pour un match (pur, sans DB).
// Retourne les données extraites prêtes pour insertion séquentielle.
func (e *SyncEngine) fetchMatchData(
	ctx context.Context,
	client HaloClient,
	matchID string,
	opts domain.SyncOptions,
) (*fetchedMatch, error) {
	matchJSON, err := client.GetMatchStats(ctx, matchID)
	if err != nil {
		slog.WarnContext(ctx, "sync: GetMatchStats échoué",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
		return nil, fmt.Errorf("GetMatchStats: %w", err)
	}

	fm := &fetchedMatch{
		MatchID: matchID,
	}

	// Extract registry (obligatoire).
	reg, err := ExtractRegistry(matchJSON, e.gamertag)
	if err != nil {
		slog.WarnContext(ctx, "sync: ExtractRegistry échoué",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
		return nil, fmt.Errorf("ExtractRegistry: %w", err)
	}
	fm.Registry = reg

	// Extract optionnels.
	if opts.WithParticipants {
		fm.Participants = ExtractParticipants(matchJSON)

		// Garantir gamertag sur la row du joueur synchronisé : l'API renvoie
		// parfois Gamertag/PlayerName vide pour le joueur appelant.
		ensureGamertagForSelf(fm.Participants, e.xuid, e.gamertag)

		// Skill API : team_mmr, enemy_mmr, kills/deaths_expected.
		// Endpoint séparé du stats — non-bloquant : un échec produit un warning.
		if xuids := ParticipantXUIDs(fm.Participants); len(xuids) > 0 {
			skillData, skillErr := client.GetMatchSkill(ctx, matchID, xuids)
			if skillErr != nil {
				fm.SkillError = fmt.Errorf("GetMatchSkill: %w", skillErr)
			} else if len(skillData) > 0 {
				fm.Participants = MergeSkillIntoParticipants(fm.Participants, skillData)
				// CSR par-match : extraction depuis RankRecap si match classé.
				// L'écriture en player DB est différée à insertFetchedMatch.
				fm.CSRRow = ExtractCSRRowIfRanked(fm.Registry, skillData[e.xuid])
			}
		}
	}
	if opts.WithMedals {
		fm.Medals = ExtractMedals(matchJSON)
	}
	// PersonalScores du joueur courant — toujours extraits (pas de flag dédié,
	// même cycle de vie que les participants). La table n'est pas dans shared :
	// l'insertion se fera côté playerDB dans insertFetchedMatch.
	fm.PSA = ExtractPersonalScoreAwards(matchJSON, matchID, e.xuid)
	if opts.WithHighlightEvents {
		data, filmMajorVer, found, err := client.GetHighlightEventsChunk(ctx, matchID)
		fm.HasHighlights = found
		fm.FilmMajorVer = filmMajorVer
		if err != nil {
			fm.HighlightError = fmt.Errorf("GetHighlightEventsChunk: %w", err)
		} else if found {
			fm.HighlightData = data
		}
	}

	return fm, nil
}

// insertFetchedMatch insère les données fetchées d'un match (séquentiel, order-preserving).
func (e *SyncEngine) insertFetchedMatch(
	ctx context.Context,
	sharedDB, playerDB, globalDB *sql.DB,
	result *domain.SyncResult,
	fm *fetchedMatch,
) error {
	// Registry (obligatoire).
	if err := InsertRegistryIfNotExists(sharedDB, *fm.Registry); err != nil {
		slog.ErrorContext(ctx, "sync: InsertRegistry échoué",
			"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err,
		)
		return fmt.Errorf("InsertRegistry: %w", err)
	}

	// Participants.
	if len(fm.Participants) > 0 {
		if fm.SkillError != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("skill %s: %v", fm.MatchID, fm.SkillError))
		}
		if err := InsertParticipants(sharedDB, fm.Participants); err != nil {
			slog.ErrorContext(ctx, "sync: InsertParticipants échoué",
				"gamertag", e.gamertag, "match_id", fm.MatchID, "count", len(fm.Participants), "err", err,
			)
			return fmt.Errorf("InsertParticipants: %w", err)
		}
		result.ParticipantsDone += len(fm.Participants)

		// Phase 2 du plan PLAN_BITMASKS_AUDIT_FIX : marquer le bit
		// participants pour que `levelup backfill --participants` ne re-traite
		// pas indéfiniment ce match.
		if markErr := MarkParticipantsDone(sharedDB, fm.MatchID); markErr != nil {
			slog.WarnContext(ctx, "sync: MarkParticipantsDone échoué",
				"match_id", fm.MatchID, "err", markErr)
		}

		// Phase 2 — skill bits : on ne marque que si l'API skill a renvoyé des
		// données (fm.SkillError nil ET team_mmr présent sur ≥1 participant).
		// MarkSkillLoaded filtre lui-même sur team_mmr IS NOT NULL côté SQL.
		if fm.SkillError == nil && hasAnyTeamMMR(fm.Participants) {
			if markErr := MarkSkillLoaded(sharedDB, fm.MatchID); markErr != nil {
				slog.WarnContext(ctx, "sync: MarkSkillLoaded échoué",
					"match_id", fm.MatchID, "err", markErr)
			}
		}

		// Upsert XUID aliases.
		aliased := 0
		for _, p := range fm.Participants {
			if p.Gamertag != nil && *p.Gamertag != "" {
				if globalDB != nil {
					_ = UpsertXUIDAlias(globalDB, p.XUID, *p.Gamertag)
				}
				aliased++
			}
		}
		slog.DebugContext(ctx, "sync: participants insérés",
			"match_id", fm.MatchID, "participants", len(fm.Participants), "aliases_upserted", aliased,
		)
	}

	// Medals.
	if len(fm.Medals) > 0 {
		if err := InsertMedals(sharedDB, fm.Medals); err != nil {
			slog.ErrorContext(ctx, "sync: InsertMedals échoué",
				"gamertag", e.gamertag, "match_id", fm.MatchID, "count", len(fm.Medals), "err", err,
			)
			return fmt.Errorf("InsertMedals: %w", err)
		}
		result.MedalsInserted += len(fm.Medals)
		slog.DebugContext(ctx, "sync: médailles insérées",
			"match_id", fm.MatchID, "medals", len(fm.Medals),
		)
	}

	// Highlight events.
	if fm.HasHighlights && fm.HighlightData != nil {
		if err := insertHighlightEventsFromData(ctx, sharedDB, globalDB, fm.MatchID, fm.HighlightData, fm.FilmMajorVer, result); err != nil {
			slog.WarnContext(ctx, "sync: highlight_events insertion échouée",
				"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err,
			)
			result.Warnings = append(result.Warnings, fmt.Sprintf("highlight_events %s: %v", fm.MatchID, err))
		}
	} else if fm.HighlightError != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("highlight_events %s: %v", fm.MatchID, fm.HighlightError))
	}

	// Player enrichment.
	if err := UpsertPlayerEnrichment(playerDB, fm.MatchID, ""); err != nil {
		slog.ErrorContext(ctx, "sync: UpsertPlayerEnrichment échoué",
			"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err,
		)
		return fmt.Errorf("UpsertPlayerEnrichment: %w", err)
	}

	// PersonalScoreAwards (player DB, par joueur synchronisé). Non-bloquant :
	// un échec produit un warning, le sync continue.
	if len(fm.PSA) > 0 {
		if err := InsertPersonalScoreAwards(playerDB, fm.MatchID, e.xuid, fm.PSA); err != nil {
			slog.WarnContext(ctx, "sync: InsertPersonalScoreAwards échoué",
				"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err,
			)
			result.Warnings = append(result.Warnings, fmt.Sprintf("psa %s: %v", fm.MatchID, err))
		}
	}

	// CSR par-match (player DB). Renseigné par fetchMatchData uniquement pour
	// les matchs classés dont RankRecap était présent. Non-bloquant.
	if fm.CSRRow != nil {
		if err := UpsertCSRRow(playerDB, fm.CSRRow); err != nil {
			slog.WarnContext(ctx, "sync: UpsertCSRRow échoué",
				"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err,
			)
			result.Warnings = append(result.Warnings, fmt.Sprintf("csr %s: %v", fm.MatchID, err))
		} else {
			slog.DebugContext(ctx, "sync: CSR row écrite",
				"match_id", fm.MatchID, "tier", fm.CSRRow.Tier, "tier_label", fm.CSRRow.TierLabel,
			)
		}
	}

	result.MatchesInserted++
	result.InsertedMatchIDs = append(result.InsertedMatchIDs, fm.MatchID)
	return nil
}

// insertHighlightEventsFromData parse et insère les highlight events à partir de données déjà fetchées.
// Helper utilisé par insertFetchedMatch pour injection de dépendance.
func insertHighlightEventsFromData(
	ctx context.Context,
	sharedDB, globalDB *sql.DB,
	matchID string,
	data []byte,
	filmMajorVersion int,
	result *domain.SyncResult,
) error {
	if len(data) == 0 {
		return nil // Pas de données — OK, pas d'erreur.
	}

	events, err := analysis.ParseHighlightEvents(data, filmMajorVersion)
	if err != nil {
		return fmt.Errorf("ParseHighlightEvents: %w", err)
	}
	if len(events) == 0 {
		// Anomalie : on a téléchargé un chunk non-vide mais le parser
		// n'a rien extrait. Avant le fix bit-aligné (mai 2026), ce cas
		// était silencieusement loggé en DEBUG et faisait perdre tout
		// l'historique highlight events. Désormais : WARN + compteur
		// expvar pour qu'une regression soit immédiatement visible.
		observability.IncCounter("highlight_events_parse_anomaly_total")
		slog.WarnContext(ctx, "highlight_events parse_anomaly: chunk non-vide mais 0 events extraits",
			"match_id", matchID,
			"film_version", filmMajorVersion,
			"data_size", len(data),
		)
		if result != nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("highlight_events parse_anomaly %s: chunk %d bytes v%d → 0 events", matchID, len(data), filmMajorVersion))
		}
		return nil
	}

	n, err := InsertHighlightEvents(sharedDB, matchID, events)
	if err != nil {
		return fmt.Errorf("InsertHighlightEvents: %w", err)
	}

	// Upsert XUID aliases from events.
	if globalDB != nil {
		for _, ev := range events {
			if ev.XUID != 0 && ev.Gamertag != "" {
				_ = UpsertXUIDAlias(globalDB, strconv.FormatUint(ev.XUID, 10), ev.Gamertag)
			}
		}
	}

	if n > 0 {
		result.EventsInserted += n
		_ = MarkEventsLoaded(sharedDB, matchID)
	}

	// Fix Phase 1bis (mai 2026) : ne marquer MBitKillerVictim que si l'insert
	// a réellement réussi. Avant, l'insert + le mark étaient appelés
	// inconditionnellement avec `_ =` qui swallowait l'erreur — bit menteur
	// dormant, masqué tant que les events n'arrivaient pas (parser cassé).
	if pairsErr := InsertKillerVictimPairsFromEvents(sharedDB, matchID, events); pairsErr != nil {
		slog.WarnContext(ctx, "InsertKillerVictimPairs échoué", "match_id", matchID, "err", pairsErr)
		if result != nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("killer_victim_pairs %s: %v", matchID, pairsErr))
		}
	} else {
		_ = MarkKillerVictimLoaded(sharedDB, matchID)
	}

	return nil
}

// ProcessHighlightEvents télécharge le chunk highlight events, le parse et
// insère les events + paires killer/victim dans la shared DB.
// Retourne une erreur uniquement en cas de défaillance fatale (non-nil = warning dans processMatch).
//
// Exposé (majuscule) pour les outils de replay : cmd/replay_highlight_events.
func ProcessHighlightEvents(
	ctx context.Context,
	client HaloClient,
	sharedDB, globalDB *sql.DB,
	matchID string,
	result *domain.SyncResult,
) error {
	data, filmMajorVersion, found, err := client.GetHighlightEventsChunk(ctx, matchID)
	if err != nil {
		return fmt.Errorf("GetHighlightEventsChunk: %w", err)
	}
	if !found || len(data) == 0 {
		slog.DebugContext(ctx, "processHighlightEvents: film absent ou chunk vide",
			"match_id", matchID, "found", found, "data_len", len(data),
		)
		// Marquer events_loaded=TRUE pour ne pas retenter à chaque sync : le
		// film 404 est définitif (Halo ne sauve pas le film de tous les matchs).
		if markErr := MarkEventsLoaded(sharedDB, matchID); markErr != nil {
			slog.DebugContext(ctx, "MarkEventsLoaded échoué (no-film)",
				"match_id", matchID, "err", markErr)
		}
		return nil
	}

	events, err := analysis.ParseHighlightEvents(data, filmMajorVersion)
	if err != nil {
		return fmt.Errorf("ParseHighlightEvents: %w", err)
	}
	if len(events) == 0 {
		// Anomalie : chunk téléchargé non-vide mais 0 event parsé.
		// Voir insertHighlightEventsFromData pour la justification.
		observability.IncCounter("highlight_events_parse_anomaly_total")
		slog.WarnContext(ctx, "highlight_events parse_anomaly: chunk non-vide mais 0 events extraits",
			"match_id", matchID,
			"film_version", filmMajorVersion,
			"data_size", len(data),
		)
		if result != nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("highlight_events parse_anomaly %s: chunk %d bytes v%d → 0 events", matchID, len(data), filmMajorVersion))
		}
		return nil
	}

	n, err := InsertHighlightEvents(sharedDB, matchID, events)
	if err != nil {
		return fmt.Errorf("InsertHighlightEvents: %w", err)
	}

	// Upsert les gamertags extraits depuis le film (source la plus fiable).
	// P5.3 : ecriture dans la DB globale xbox_aliases.
	aliasCount := 0
	if globalDB != nil {
		for _, ev := range events {
			if ev.XUID != 0 && ev.Gamertag != "" {
				if uErr := UpsertXUIDAlias(globalDB, strconv.FormatUint(ev.XUID, 10), ev.Gamertag); uErr == nil {
					aliasCount++
				}
			}
		}
	}

	if n > 0 {
		result.EventsInserted += n
		if markErr := MarkEventsLoaded(sharedDB, matchID); markErr != nil {
			slog.WarnContext(ctx, "MarkEventsLoaded échoué", "match_id", matchID, "err", markErr)
		}
	}

	pairsErr := InsertKillerVictimPairsFromEvents(sharedDB, matchID, events)
	if pairsErr != nil {
		slog.WarnContext(ctx, "InsertKillerVictimPairs échoué", "match_id", matchID, "err", pairsErr)
		// Non-bloquant : on continue.
	} else {
		if markErr := MarkKillerVictimLoaded(sharedDB, matchID); markErr != nil {
			slog.WarnContext(ctx, "MarkKillerVictimLoaded échoué", "match_id", matchID, "err", markErr)
		}
	}

	slog.DebugContext(ctx, "processHighlightEvents: terminé",
		"match_id", matchID,
		"film_version", filmMajorVersion,
		"events_parsed", len(events),
		"events_inserted", n,
		"aliases_upserted", aliasCount,
		"killer_victim_err", pairsErr,
	)
	return nil
}

// hasAnyTeamMMR retourne true si au moins un participant a team_mmr renseigné.
// Utilisé pour décider si MarkSkillLoaded doit être appelé après
// MergeSkillIntoParticipants (Phase 2 plan PLAN_BITMASKS_AUDIT_FIX).
func hasAnyTeamMMR(parts []ParticipantRow) bool {
	for _, p := range parts {
		if p.TeamMMR != nil {
			return true
		}
	}
	return false
}

// loadKnownMatchIDs retourne l'ensemble des match_ids déjà présents dans
// player_match_enrichment (player DB).
func loadKnownMatchIDs(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query("SELECT match_id FROM player_match_enrichment")
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
