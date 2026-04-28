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
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
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
	metadataDBPath string
	tokens         *domain.HaloTokens
	// provider est utilisé pour résoudre l'access_token Xbox Live (achievements).
	// Nil si non défini (les achievements seront ignorés).
	provider auth.TokenProvider
	// resolver est utilisé pour le pré-warming des images d'achievements (optionnel).
	resolver assets.Resolver
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
	relPlayer, err := AcquireLeaseCtx(ctx, e.playerDBPath)
	if err != nil {
		return nil, fmt.Errorf("RunBackfill lease player: %w", err)
	}
	defer relPlayer()

	relShared, err := AcquireLeaseCtx(ctx, e.sharedDBPath)
	if err != nil {
		return nil, fmt.Errorf("RunBackfill lease shared: %w", err)
	}
	defer relShared()

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
func (e *SyncEngine) RunBackfillComebackBadges(ctx context.Context, forceAll bool) (int, error) {
	relPlayer, err := AcquireLeaseCtx(ctx, e.playerDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillComebackBadges lease player: %w", err)
	}
	defer relPlayer()

	relShared, err := AcquireLeaseCtx(ctx, e.sharedDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillComebackBadges lease shared: %w", err)
	}
	defer relShared()

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
	relPlayer, err := AcquireLeaseCtx(ctx, e.playerDBPath)
	if err != nil {
		slog.ErrorContext(ctx, "sync: lease player DB échouée", "gamertag", e.gamertag, "err", err)
		return result, fmt.Errorf("run: %w", err)
	}
	defer relPlayer()

	slog.DebugContext(ctx, "sync: acquisition lease shared DB", "gamertag", e.gamertag, "db", e.sharedDBPath)
	relShared, err := AcquireLeaseCtx(ctx, e.sharedDBPath)
	if err != nil {
		slog.ErrorContext(ctx, "sync: lease shared DB échouée", "gamertag", e.gamertag, "err", err)
		return result, fmt.Errorf("run: %w", err)
	}
	defer relShared()

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
	slog.DebugContext(ctx, "sync: DBs ouvertes", "gamertag", e.gamertag)

	// ─── Match IDs déjà connus (player DB) ───────────────────────────────────
	known, err := loadKnownMatchIDs(playerDB)
	if err != nil {
		slog.ErrorContext(ctx, "sync: chargement match_ids connus échoué", "gamertag", e.gamertag, "err", err)
		return result, fmt.Errorf("run loadKnownMatchIDs: %w", err)
	}
	slog.InfoContext(ctx, "sync: match_ids connus chargés", "gamertag", e.gamertag, "known_count", len(known))

	// ─── Client API ────────────────────────────────────────────────────────────
	client := NewHaloAPIClient(e.tokens.SpartanToken, e.tokens.ClearanceToken, opts.RequestsPerSecond)

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

			if err := e.processMatch(ctx, client, sharedDB, playerDB, &result, entry.MatchID, opts); err != nil {
				slog.WarnContext(ctx, "sync: processMatch échoué",
					"gamertag", e.gamertag, "match_id", entry.MatchID, "err", err,
				)
				result.AddWarning(fmt.Sprintf("processMatch(%s): %v", entry.MatchID, err))
			} else {
				processed++
				slog.InfoContext(ctx, "sync: match traité",
					"gamertag", e.gamertag, "match_id", entry.MatchID,
					"processed", processed, "inserted_total", result.MatchesInserted,
				)
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
	sharedDB, playerDB *sql.DB,
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
				_ = UpsertXUIDAlias(sharedDB, p.XUID, *p.Gamertag)
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
		if err := processHighlightEvents(ctx, client, sharedDB, matchID, result); err != nil {
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

// processHighlightEvents télécharge le chunk highlight events, le parse et
// insère les events + paires killer/victim dans la shared DB.
// Retourne une erreur uniquement en cas de défaillance fatale (non-nil = warning dans processMatch).
func processHighlightEvents(
	ctx context.Context,
	client HaloClient,
	sharedDB *sql.DB,
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
	aliasCount := 0
	for _, ev := range events {
		if ev.XUID != 0 && ev.Gamertag != "" {
			if uErr := UpsertXUIDAlias(sharedDB, strconv.FormatUint(ev.XUID, 10), ev.Gamertag); uErr == nil {
				aliasCount++
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
