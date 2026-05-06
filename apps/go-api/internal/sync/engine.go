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
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/dblease"
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
	// prestigeHook est appelé après ingestion (best-effort, no-op si nil).
	// Reçoit (ctx, gamertag, titleSlug) — le hook se charge lui-même de
	// la résolution Prestige et du feature flag.
	prestigeHook func(ctx context.Context, gamertag, titleSlug string)
	// friendsLoader résout settings.FriendGamertags à la demande pour le
	// hook auto-recompute is_with_friends post-sync delta. Nil → feature off
	// (les nouveaux matchs resteront is_with_friends=FALSE jusqu'au prochain
	// recompute manuel via PATCH /settings ou CLI levelup recompute-friends).
	friendsLoader FriendsLoader
}

// NewSyncEngine crée un moteur de sync pour un joueur.
//
//   - repoRoot    : racine du repo (cfg.RepoRoot)
//   - gamertag    : gamertag Halo du joueur
//   - xuid        : XUID numérique (sans "xuid()")
//   - tokens      : tokens Halo frais obtenus après Device Code Flow
//   - provider    : TokenProvider pour résoudre l'access_token Xbox Live
func NewSyncEngine(
	repoRoot, gamertag, xuid string,
	tokens *domain.HaloTokens,
	provider auth.TokenProvider,
) *SyncEngine {
	pr := titlePkg.NewPathResolver(repoRoot)
	return &SyncEngine{
		gamertag:       gamertag,
		xuid:           xuid,
		titleSlug:      titlePkg.DefaultSlug,
		playerDBPath:   pr.PlayerDBPath(titlePkg.DefaultSlug, gamertag),
		sharedDBPath:   pr.SharedDBPath(titlePkg.DefaultSlug),
		globalDBPath:   pr.GlobalXuidAliasesDBPath(),
		metadataDBPath: pr.MetadataDBPath(titlePkg.DefaultSlug),
		tokens:         tokens,
		provider:       provider,
	}
}

// WithPrestigeHook attache un hook post-sync (best-effort).
//
// Le hook reçoit (ctx, gamertag, titleSlug) après que match_participants
// soit écrit. Il est responsable de gérer le feature flag et de ne jamais
// propager d'erreur (le sync ne doit pas échouer à cause de Prestige).
func (e *SyncEngine) WithPrestigeHook(hook func(ctx context.Context, gamertag, titleSlug string)) *SyncEngine {
	e.prestigeHook = hook
	return e
}

// WithResolver attache un Resolver pour le pré-warming des images d'achievements.
// Retourne le même engine pour permettre le chaînage.
func (e *SyncEngine) WithResolver(r assets.Resolver) *SyncEngine {
	e.resolver = r
	return e
}

// WithFriendsLoader attache un loader settings.FriendGamertags pour le hook
// auto-recompute is_with_friends post-sync delta. Sans ce hook, les nouveaux
// matchs sync restent is_with_friends=FALSE jusqu'au prochain recompute
// manuel (PATCH /settings ou CLI levelup recompute-friends).
//
// Le hook est idempotent (garde WHERE FALSE dans friends_recompute.go) et
// court-circuite si la liste est vide. Aucune erreur ne propage : un échec
// n'arrête pas le sync (best-effort).
func (e *SyncEngine) WithFriendsLoader(loader FriendsLoader) *SyncEngine {
	e.friendsLoader = loader
	return e
}

// RunDelta synchronise uniquement les matchs nouveaux depuis la dernière sync.
// S'arrête dès qu'un match connu est rencontré dans l'historique paginé.
// SetCustomClient injecte un client HaloClient personnalisé (ex: PooledHaloClient).
// Si défini, ce client sera utilisé à la place de NewHaloAPIClient.
func (e *SyncEngine) SetCustomClient(client HaloClient) {
	e.customClient = client
}

func (e *SyncEngine) RunDelta(ctx context.Context, opts domain.SyncOptions) (domain.SyncResult, error) {
	return e.run(ctx, opts, true)
}

// RunFull synchronise tous les matchs jusqu'à opts.MaxMatches (peu importe l'historique connu).
func (e *SyncEngine) RunFull(ctx context.Context, opts domain.SyncOptions) (domain.SyncResult, error) {
	return e.run(ctx, opts, false)
}

// RunBackfill détecte les matchs avec données manquantes et retourne la liste.
// Le scope doit être Resolve() avant appel. Retourne la liste des match_ids manquants.
func (e *SyncEngine) RunBackfill(ctx context.Context, scope *SyncScope) ([]string, error) {
	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return nil, fmt.Errorf("RunBackfill lease player: %w", err)
	}
	defer writerPlayer.Release()

	writerShared, err := dblease.AcquireWriterCtx(ctx, nil, e.sharedDBPath, dblease.KindSharedMatches)
	if err != nil {
		return nil, fmt.Errorf("RunBackfill lease shared: %w", err)
	}
	defer writerShared.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return nil, fmt.Errorf("RunBackfill OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	sharedHandle, err := OpenSharedDB(e.sharedDBPath)
	if err != nil {
		return nil, fmt.Errorf("RunBackfill OpenSharedDB: %w", err)
	}
	defer sharedHandle.Close()

	return FindMatchesMissingData(playerHandle.SQLDb(), sharedHandle.SQLDb(), e.xuid, scope)
}

// RunBackfillComebackBadges calcule et persiste le dominance_flag pour les
// matchs du joueur. Selectionne :
//   - tous les matchs si forceAll=true
//   - les matchs sans dominance_flag (ou flag=0) sinon
//
// Branche la fonction BackfillDominanceFlags (sync/comeback.go) au pipeline.
// Retourne le nombre de match_ids traites (et l'erreur infra si lease/open
// echoue).
//
// L'ingestion principale (RunDelta/RunFull) ne calcule PAS encore le flag a
// chaque match : ce backfill explicite est la voie d'entree pour peupler les
// dominance_flag (cf. PLAN_META_FOUNDATIONS_GO § 6.0.1, prerequis Phase 1
// pilote Squad/MatchView/Career).
// RunBackfillEngagementScores calcule et persiste le score d'engagement pour
// les matchs PvP du joueur (Phase 6 plan engagement). Si force=true, recalcule
// les scores existants ; sinon ne calcule que les manquants.
//
// Skip silencieux si la migration Phase 2 n'a pas ete appliquee (gating
// information_schema). Aucun appel API requis (calcul purement local depuis
// highlight_events deja synces).
func (e *SyncEngine) RunBackfillEngagementScores(ctx context.Context, force bool) (int, error) {
	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillEngagementScores lease player: %w", err)
	}
	defer writerPlayer.Release()

	writerShared, err := dblease.AcquireWriterCtx(ctx, nil, e.sharedDBPath, dblease.KindSharedMatches)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillEngagementScores lease shared: %w", err)
	}
	defer writerShared.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillEngagementScores OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	sharedHandle, err := OpenSharedDB(e.sharedDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillEngagementScores OpenSharedDB: %w", err)
	}
	defer sharedHandle.Close()

	n, err := batchComputeEngagementScores(ctx, playerHandle.SQLDb(), sharedHandle.SQLDb(), e.xuid, force)
	if err != nil {
		return n, err
	}
	// Recompute des coefficients en queue : on a possiblement ajoute des
	// paces en DB, donc la mediane est a rafraichir.
	if nCoefs, errCoefs := batchRecomputeCoefficients(ctx, playerHandle.SQLDb(), e.xuid); errCoefs != nil {
		slog.WarnContext(ctx, "RunBackfillEngagementScores: recompute coefs failed",
			"xuid", e.xuid, "err", errCoefs)
	} else if nCoefs > 0 {
		slog.InfoContext(ctx, "RunBackfillEngagementScores: coefs updated",
			"xuid", e.xuid, "n_modes", nCoefs)
	}
	return n, nil
}

// RunBackfillEngagementCoefficients recompute UNIQUEMENT les coefficients
// d'engagement du joueur depuis les paces deja persistees (~5ms par joueur,
// 0 re-scan des matchs). A activer via SyncScope.EngagementCoefficients.
//
// Utile pour rafraichir apres un ajustement de formule sans devoir relancer
// le compute des scores. Skip silencieux si la migration des paces n'est
// pas appliquee (cf. batchRecomputeCoefficients).
//
// Retourne le nombre de modes_category mis a jour (0 a 2).
func (e *SyncEngine) RunBackfillEngagementCoefficients(ctx context.Context) (int, error) {
	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillEngagementCoefficients lease player: %w", err)
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillEngagementCoefficients OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	return batchRecomputeCoefficients(ctx, playerHandle.SQLDb(), e.xuid)
}

func (e *SyncEngine) RunBackfillComebackBadges(ctx context.Context, forceAll bool) (int, error) {
	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillComebackBadges lease player: %w", err)
	}
	defer writerPlayer.Release()

	writerShared, err := dblease.AcquireWriterCtx(ctx, nil, e.sharedDBPath, dblease.KindSharedMatches)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillComebackBadges lease shared: %w", err)
	}
	defer writerShared.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillComebackBadges OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	sharedHandle, err := OpenSharedDB(e.sharedDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillComebackBadges OpenSharedDB: %w", err)
	}
	defer sharedHandle.Close()

	matchIDs, err := selectMatchesForComebackBadges(ctx, playerHandle.SQLDb(), sharedHandle.SQLDb(), e.xuid, forceAll)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillComebackBadges select: %w", err)
	}
	if len(matchIDs) == 0 {
		slog.InfoContext(ctx, "comeback-badges: aucun match a traiter",
			"player", e.gamertag, "force_all", forceAll)
		return 0, nil
	}

	slog.InfoContext(ctx, "comeback-badges: backfill en cours",
		"player", e.gamertag, "match_count", len(matchIDs), "force_all", forceAll)
	if err := BackfillDominanceFlags(ctx, sharedHandle.SQLDb(), playerHandle.SQLDb(), e.xuid, matchIDs); err != nil {
		return 0, fmt.Errorf("RunBackfillComebackBadges backfill: %w", err)
	}
	return len(matchIDs), nil
}

// selectMatchesForComebackBadges retourne les match_ids du joueur a traiter
// pour le backfill dominance_flag.
//
// Si forceAll=true : tous les matchs du joueur dans shared.match_participants.
// Sinon : uniquement les matchs ou player_match_enrichment.dominance_flag est
// nul ou egal a 0 (cas par defaut "manquant").
func selectMatchesForComebackBadges(
	ctx context.Context,
	playerDB, sharedDB *sql.DB,
	xuid string,
	forceAll bool,
) ([]string, error) {
	allIDs, err := loadAllMatchIDsForPlayer(ctx, sharedDB, xuid)
	if err != nil {
		return nil, fmt.Errorf("load all match_ids: %w", err)
	}
	if forceAll {
		return allIDs, nil
	}
	flagged, err := loadFlaggedMatchIDs(ctx, playerDB)
	if err != nil {
		return nil, fmt.Errorf("load flagged match_ids: %w", err)
	}
	flaggedSet := make(map[string]struct{}, len(flagged))
	for _, id := range flagged {
		flaggedSet[id] = struct{}{}
	}
	out := make([]string, 0, len(allIDs))
	for _, id := range allIDs {
		if _, ok := flaggedSet[id]; !ok {
			out = append(out, id)
		}
	}
	return out, nil
}

// loadAllMatchIDsForPlayer retourne tous les match_id du joueur (shared DB).
func loadAllMatchIDsForPlayer(ctx context.Context, sharedDB *sql.DB, xuid string) ([]string, error) {
	rows, err := sharedDB.QueryContext(ctx,
		`SELECT match_id FROM match_participants WHERE xuid = ? ORDER BY match_id`, xuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// loadFlaggedMatchIDs retourne les match_id deja peuples avec un flag
// non-nul et non-zero (player DB).
func loadFlaggedMatchIDs(ctx context.Context, playerDB *sql.DB) ([]string, error) {
	rows, err := playerDB.QueryContext(ctx,
		`SELECT match_id FROM player_match_enrichment
		 WHERE dominance_flag IS NOT NULL AND dominance_flag > 0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// run est le cœur du moteur de sync. isDelta=true → stop dès un match connu.
func (e *SyncEngine) run(ctx context.Context, opts domain.SyncOptions, isDelta bool) (domain.SyncResult, error) {
	result := domain.SyncResult{StartedAt: time.Now()}
	mode := "full"
	if isDelta {
		mode = "delta"
	}

	slog.InfoContext(ctx, "sync: démarrage",
		"gamertag", e.gamertag,
		"xuid", e.xuid,
		"mode", mode,
		"match_type", opts.MatchType,
		"max_matches", opts.MaxMatches,
		"with_participants", opts.WithParticipants,
		"with_medals", opts.WithMedals,
		"rps", opts.RequestsPerSecond,
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

	slog.DebugContext(ctx, "sync: acquisition lease shared DB", "gamertag", e.gamertag, "db", e.sharedDBPath)
	writerShared, err := dblease.AcquireWriterCtx(ctx, nil, e.sharedDBPath, dblease.KindSharedMatches)
	if err != nil {
		slog.ErrorContext(ctx, "sync: lease shared DB échouée", "gamertag", e.gamertag, "err", err)
		return result, fmt.Errorf("run: %w", err)
	}
	defer writerShared.Release()

	// ─── Ouverture des DBs ─────────────────────────────────────────────────────
	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		slog.ErrorContext(ctx, "sync: ouverture player DB échouée", "gamertag", e.gamertag, "db", e.playerDBPath, "err", err)
		return result, fmt.Errorf("run OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()
	playerDB := playerHandle.SQLDb()

	sharedHandle, err := OpenSharedDB(e.sharedDBPath)
	if err != nil {
		slog.ErrorContext(ctx, "sync: ouverture shared DB échouée", "gamertag", e.gamertag, "db", e.sharedDBPath, "err", err)
		return result, fmt.Errorf("run OpenSharedDB: %w", err)
	}
	defer sharedHandle.Close()
	sharedDB := sharedHandle.SQLDb()

	// P5.3 : DB globale xbox_aliases (mapping xuid→gamertag global Microsoft).
	globalDB, globalCleanup, err := openGlobalDB(e.globalDBPath)
	if err != nil {
		slog.WarnContext(ctx, "sync: ouverture global DB échouée — alias upsert désactivé",
			"db", e.globalDBPath, "err", err)
		globalDB = nil
	} else {
		defer globalCleanup()
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
		client = NewHaloAPIClient(e.tokens.SpartanToken, e.tokens.ClearanceToken, opts.RequestsPerSecond)
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
			"gamertag", e.gamertag, "start", start, "page_size", historyPageSize,
		)
		entries, err := client.GetMatchHistory(ctx, e.gamertag, opts.MatchType, start, historyPageSize)
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

		allKnown := true

		// ─── Phase 1 : Filtrer et préparer les matchs à fetcher ───
		var toFetch []string // MatchIDs à fetcher
		var fetchIndex []int // Index dans entries (pour order preservation)

		for i, entry := range entries {
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
			fetchIndex = append(fetchIndex, i)
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
	postResult := e.runConditionalPostSync(ctx, playerDB, sharedDB, client, result.MatchesInserted)
	if result.MatchesInserted > 0 || postResult.CareerSynced || postResult.AchievementsSynced {
		result.PostSync = &postResult
		slog.InfoContext(ctx, "sync: pipeline post-sync terminé",
			"gamertag", e.gamertag,
			"perf_scores", postResult.PerfScoresComputed,
			"lusr_updated", postResult.LUSRUpdated,
			"career_synced", postResult.CareerSynced,
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

// runConditionalPostSync exécute le pipeline complet si des matchs ont été insérés,
// sinon rafraîchit au moins la carrière pour mettre à jour le snapshot joueur.
func (e *SyncEngine) runConditionalPostSync(
	ctx context.Context,
	playerDB, sharedDB *sql.DB,
	client HaloClient,
	matchesInserted int,
) domain.PostSyncResult {
	if matchesInserted > 0 {
		slog.InfoContext(ctx, "sync: lancement pipeline post-sync", "gamertag", e.gamertag)
		return e.runPostSyncPipeline(ctx, playerDB, sharedDB, client)
	}

	slog.DebugContext(ctx, "sync: aucun match inséré — refresh carrière seul", "gamertag", e.gamertag)
	return domain.PostSyncResult{
		CareerSynced:       e.runCareerSync(ctx, playerDB, client),
		AchievementsSynced: e.runAchievementsSync(ctx, playerDB),
	}
}

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
	if err := InsertRegistryIfNotExists(sharedDB, *reg); err != nil {
		slog.ErrorContext(ctx, "processMatch: InsertRegistry échoué",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
		return fmt.Errorf("InsertRegistry: %w", err)
	}

	// ─── match_participants ────────────────────────────────────────────────────
	if opts.WithParticipants {
		participants := ExtractParticipants(matchJSON)
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
		if err := processHighlightEvents(ctx, client, sharedDB, globalDB, matchID, result); err != nil {
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
	HighlightData  []byte // Raw highlight events chunk (ou nil si absent)
	FilmMajorVer   int
	HasHighlights  bool
	HighlightError error // Non-bloquant si présent
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
	}
	if opts.WithMedals {
		fm.Medals = ExtractMedals(matchJSON)
	}
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
		if err := InsertParticipants(sharedDB, fm.Participants); err != nil {
			slog.ErrorContext(ctx, "sync: InsertParticipants échoué",
				"gamertag", e.gamertag, "match_id", fm.MatchID, "count", len(fm.Participants), "err", err,
			)
			return fmt.Errorf("InsertParticipants: %w", err)
		}
		result.ParticipantsDone += len(fm.Participants)

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

	_ = InsertKillerVictimPairsFromEvents(sharedDB, matchID, events)
	_ = MarkKillerVictimLoaded(sharedDB, matchID)

	return nil
}

// processHighlightEvents télécharge le chunk highlight events, le parse et
// insère les events + paires killer/victim dans la shared DB.
// Retourne une erreur uniquement en cas de défaillance fatale (non-nil = warning dans processMatch).
func processHighlightEvents(
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
		return nil // Film indisponible ou pas de chunk — pas d'erreur.
	}

	events, err := analysis.ParseHighlightEvents(data, filmMajorVersion)
	if err != nil {
		return fmt.Errorf("ParseHighlightEvents: %w", err)
	}
	if len(events) == 0 {
		slog.DebugContext(ctx, "processHighlightEvents: aucun event extrait",
			"match_id", matchID, "film_version", filmMajorVersion, "data_len", len(data),
		)
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

// runPostSyncPipeline exécute le pipeline post-sync :
// 1. Performance scores
// 2. LUSR (TrueSkill 2)
// 3. Career rank
// 4. Aggregates (materialized views)
func (e *SyncEngine) runPostSyncPipeline(
	ctx context.Context,
	playerDB, sharedDB *sql.DB,
	client HaloClient,
) domain.PostSyncResult {
	var r domain.PostSyncResult

	// 1. Performance scores
	slog.DebugContext(ctx, "post-sync: calcul perf scores", "gamertag", e.gamertag)
	if n, err := batchComputePerformanceScores(playerDB, sharedDB, e.xuid); err != nil {
		slog.WarnContext(ctx, "post-sync: perf scores échoué", "gamertag", e.gamertag, "err", err)
	} else {
		r.PerfScoresComputed = n
		slog.DebugContext(ctx, "post-sync: perf scores calculés", "gamertag", e.gamertag, "count", n)
	}

	// 1.5 Engagement scores (Phase 3 plan engagement) — best-effort,
	// skip silencieux si migration Phase 2 non appliquee.
	slog.DebugContext(ctx, "post-sync: calcul engagement scores", "gamertag", e.gamertag)
	if n, err := batchComputeEngagementScores(ctx, playerDB, sharedDB, e.xuid, false); err != nil {
		slog.WarnContext(ctx, "post-sync: engagement scores échoué", "gamertag", e.gamertag, "err", err)
	} else if n > 0 {
		r.EngagementScoresComputed = n
		slog.DebugContext(ctx, "post-sync: engagement scores calculés", "gamertag", e.gamertag, "count", n)
	}

	// 1.5.b Recompute des engagement coefficients depuis la mediane glissante
	// des paces persistees ci-dessus. Sans ce recompute, coef_team_share reste
	// a 1.0 (cold-start) → pace_attendu = pace_team → courbes superposees a
	// l'ecran (cf. .ai/V7/PLAN_ENGAGEMENT_IMPLEMENTATION.md §4.4).
	if n, err := batchRecomputeCoefficients(ctx, playerDB, e.xuid); err != nil {
		slog.WarnContext(ctx, "post-sync: engagement coefs échoué", "gamertag", e.gamertag, "err", err)
	} else if n > 0 {
		r.EngagementCoefsUpdated = n
		slog.DebugContext(ctx, "post-sync: engagement coefs mis à jour", "gamertag", e.gamertag, "count", n)
	}

	// 1.6 Citations (best-effort) — calcul des deltas pour les matchs absents
	// de match_citations. Skip silencieux si metadata.duckdb introuvable ou si
	// citation_mappings vide. Ne propage jamais d'erreur (le sync ne doit pas
	// echouer a cause des citations).
	if n, err := e.runPostSyncCitations(ctx, playerDB, sharedDB); err != nil {
		slog.WarnContext(ctx, "post-sync: citations échoué", "gamertag", e.gamertag, "err", err)
	} else if n > 0 {
		slog.InfoContext(ctx, "post-sync: citations calculées",
			"gamertag", e.gamertag, "match_count", n)
	}

	// 2. LUSR (TrueSkill 2)
	slog.DebugContext(ctx, "post-sync: calcul LUSR", "gamertag", e.gamertag)
	if n, err := batchComputeLUSR(playerDB, sharedDB, e.xuid); err != nil {
		slog.WarnContext(ctx, "post-sync: LUSR échoué", "gamertag", e.gamertag, "err", err)
	} else {
		r.LUSRUpdated = n
		slog.DebugContext(ctx, "post-sync: LUSR mis à jour", "gamertag", e.gamertag, "count", n)
	}

	// 3. Career rank
	r.CareerSynced = e.runCareerSync(ctx, playerDB, client)

	// 3.5 Friends recompute is_with_friends (best-effort).
	// Avant l'étape 4 (aggregates) pour éviter un double-refresh : on passe
	// refreshAggregates=false, le refresh natif de l'engine couvre les UPDATEs.
	// Skip silencieux si pas de loader (legacy) ou liste vide.
	if e.friendsLoader != nil {
		if friends, ferr := e.friendsLoader(); ferr != nil {
			slog.WarnContext(ctx, "post-sync: friends loader échoué", "gamertag", e.gamertag, "err", ferr)
		} else if len(friends) > 0 {
			slog.DebugContext(ctx, "post-sync: friends recompute", "gamertag", e.gamertag, "friends_count", len(friends))
			fres, err := RecomputeIsWithFriendsCore(ctx, playerDB, sharedDB, e.xuid, friends, false)
			if err != nil {
				slog.WarnContext(ctx, "post-sync: friends recompute échoué", "gamertag", e.gamertag, "err", err)
			} else if fres.MatchesPromoted > 0 {
				r.MatchesPromotedFriends = fres.MatchesPromoted
				slog.InfoContext(ctx, "post-sync: matchs reclasses comme escouade-amis",
					"gamertag", e.gamertag,
					"promoted", fres.MatchesPromoted,
				)
			}
		}
	}

	// 4. Aggregates (materialized views)
	slog.DebugContext(ctx, "post-sync: refresh aggregates player", "gamertag", e.gamertag)
	if n, err := refreshAggregates(playerDB); err != nil {
		slog.WarnContext(ctx, "post-sync: aggregates échoué", "gamertag", e.gamertag, "err", err)
	} else {
		r.ViewsRefreshed = n
	}
	slog.DebugContext(ctx, "post-sync: refresh shared views", "gamertag", e.gamertag)
	if n, err := refreshSharedViews(sharedDB); err != nil {
		slog.WarnContext(ctx, "post-sync: shared views échoué", "gamertag", e.gamertag, "err", err)
	} else {
		r.ViewsRefreshed += n
	}

	// 5. Achievements Xbox (fire-and-forget, non bloquant en cas d'erreur token)
	r.AchievementsSynced = e.runAchievementsSync(ctx, playerDB)

	return r
}

func (e *SyncEngine) runCareerSync(
	ctx context.Context,
	playerDB *sql.DB,
	client HaloClient,
) bool {
	slog.DebugContext(ctx, "post-sync: sync career rank", "gamertag", e.gamertag, "xuid", e.xuid)
	data, err := syncCareerRank(ctx, client, e.xuid)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: career rank échoué", "gamertag", e.gamertag, "err", err)
		return false
	}
	if data == nil {
		return false
	}
	if strings.TrimSpace(e.metadataDBPath) != "" {
		metaDB, err := openCareerMetadataDB(e.metadataDBPath)
		if err != nil {
			slog.WarnContext(ctx, "post-sync: ouverture metadata échouée", "gamertag", e.gamertag, "err", err)
		} else {
			defer metaDB.Close()
			if err := enrichCareerRankFromMetadata(metaDB.SQLDb(), data); err != nil {
				slog.WarnContext(ctx, "post-sync: enrichissement carrière metadata échoué", "gamertag", e.gamertag, "err", err)
			}
		}
	}
	if err := saveCareerRank(playerDB, data); err != nil {
		slog.WarnContext(ctx, "post-sync: sauvegarde career échouée", "gamertag", e.gamertag, "err", err)
		return false
	}
	slog.DebugContext(ctx, "post-sync: career rank sauvegardé",
		"gamertag", e.gamertag, "rank", data.CurrentRank, "rank_name", data.CurrentRankName,
	)
	return true
}

// runAchievementsSync récupère les achievements Xbox pour le joueur et les persiste.
// Retourne true si la sync a réussi, false en cas d'erreur (non bloquante).
// Nécessite e.provider non nil ; skippé silencieusement sinon.
func (e *SyncEngine) runAchievementsSync(ctx context.Context, playerDB *sql.DB) bool {
	if e.provider == nil {
		slog.DebugContext(ctx, "achievements: provider nil — sync ignorée", "gamertag", e.gamertag)
		return false
	}

	// Résoudre l'access_token depuis sync_meta DuckDB.
	accessToken, err := resolveAccessTokenFromDB(ctx, playerDB, e.gamertag, e.provider)
	if err != nil {
		slog.WarnContext(ctx, "achievements: échec résolution access_token",
			"gamertag", e.gamertag, "err", err)
		return false
	}
	if accessToken == "" {
		slog.InfoContext(ctx, "achievements: aucun access_token disponible — sync ignorée",
			"gamertag", e.gamertag)
		return false
	}

	// Obtenir un XSTS token pour Xbox Live.
	xstsResult, err := auth.AcquireXSTSForRTA(ctx, accessToken)
	if err != nil {
		slog.WarnContext(ctx, "achievements: échec acquisition XSTS",
			"gamertag", e.gamertag, "err", err)
		return false
	}

	// Ouvrir la DB metadata (lecture-écriture pour l'upsert).
	metadataDB, err := sql.Open("duckdb", e.metadataDBPath)
	if err != nil {
		slog.WarnContext(ctx, "achievements: ouverture metadata DB échouée",
			"gamertag", e.gamertag, "err", err)
		return false
	}
	defer metadataDB.Close() //nolint:errcheck
	metadataDB.SetMaxOpenConns(1)

	client := NewXboxHTTPClient(xstsResult)
	if err := SyncAchievements(ctx, client, e.resolver, metadataDB, playerDB, e.xuid); err != nil {
		slog.WarnContext(ctx, "achievements: sync échouée",
			"gamertag", e.gamertag, "err", err)
		return false
	}

	slog.InfoContext(ctx, "achievements: sync terminée avec succès", "gamertag", e.gamertag)
	return true
}

// RunAchievementsOnly synchronise uniquement les achievements Xbox du joueur,
// indépendamment du sync des matchs. Utilisé par le CLI sync-achievements pour
// le backfill admin one-shot. Best-effort : retourne false sur erreur (logguée).
//
// Acquiert le dblease sur la player DB pour éviter les collisions avec un sync
// concurrent. Le provider doit être non nil ; sinon retourne false silencieusement.
func (e *SyncEngine) RunAchievementsOnly(ctx context.Context) bool {
	if e.provider == nil {
		slog.WarnContext(ctx, "achievements: provider nil — sync ignorée",
			"gamertag", e.gamertag)
		return false
	}

	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		slog.ErrorContext(ctx, "achievements: lease player DB échoué",
			"gamertag", e.gamertag, "err", err)
		return false
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		slog.ErrorContext(ctx, "achievements: ouverture player DB échouée",
			"gamertag", e.gamertag, "err", err)
		return false
	}
	defer playerHandle.Close() //nolint:errcheck

	return e.runAchievementsSync(ctx, playerHandle.SQLDb())
}

// resolveAccessTokenFromDB lit le cache MSAL et le refresh token depuis sync_meta (DB déjà ouverte),
// puis tente TrySilentRefresh ou TryOAuthRefresh selon ce qui est disponible.
// Retourne ("", nil) si aucun token n'est disponible (non fatal).
//
//nolint:unparam // contrat documenté : second retour non-nil est réservé aux futures erreurs fatales (DB)
func resolveAccessTokenFromDB(
	ctx context.Context,
	playerDB *sql.DB,
	gamertag string,
	provider auth.TokenProvider,
) (string, error) {
	var cacheJSON, refreshToken string
	if err := playerDB.QueryRowContext(ctx,
		"SELECT value FROM sync_meta WHERE key = 'msal_token_cache'").Scan(&cacheJSON); err != nil {
		slog.DebugContext(ctx, "achievements: msal_token_cache absent", "gamertag", gamertag)
	}
	if err := playerDB.QueryRowContext(ctx,
		"SELECT value FROM sync_meta WHERE key = 'oauth_refresh_token'").Scan(&refreshToken); err != nil {
		slog.DebugContext(ctx, "achievements: oauth_refresh_token absent", "gamertag", gamertag)
	}

	// Fallback env var SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG>
	if refreshToken == "" && gamertag != "" {
		key := strings.ToUpper(strings.NewReplacer(" ", "_", "-", "_", ".", "_").Replace(gamertag))
		if v := os.Getenv("SPNKR_OAUTH_REFRESH_TOKEN_" + key); v != "" {
			refreshToken = v
		}
	}

	if cacheJSON != "" {
		token, err := provider.TrySilentRefresh(ctx, cacheJSON)
		if err == nil && token != "" {
			return token, nil
		}
	}

	if refreshToken != "" {
		token, err := provider.TryOAuthRefresh(ctx, refreshToken)
		if err == nil && token != "" {
			return token, nil
		}
	}

	return "", nil
}
