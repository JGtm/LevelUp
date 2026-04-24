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
	"time"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
)

const (
	// historyPageSize est le nombre de matchs demandés par page API.
	historyPageSize = 25
)

// SyncEngine orchestre la synchronisation des données Halo d'un joueur.
type SyncEngine struct {
	gamertag       string
	xuid           string
	playerDBPath   string
	sharedDBPath   string
	metadataDBPath string
	tokens         *domain.HaloTokens
}

// NewSyncEngine crée un moteur de sync pour un joueur.
//
//   - repoRoot    : racine du repo (cfg.RepoRoot)
//   - gamertag    : gamertag Halo du joueur
//   - xuid        : XUID numérique (sans "xuid()")
//   - tokens      : tokens Halo frais obtenus après Device Code Flow
func NewSyncEngine(
	repoRoot, gamertag, xuid string,
	tokens *domain.HaloTokens,
) *SyncEngine {
	pr := titlePkg.NewPathResolver(repoRoot)
	return &SyncEngine{
		gamertag:       gamertag,
		xuid:           xuid,
		playerDBPath:   pr.PlayerDBPath(titlePkg.DefaultSlug, gamertag),
		sharedDBPath:   pr.SharedDBPath(titlePkg.DefaultSlug),
		metadataDBPath: pr.MetadataDBPath(titlePkg.DefaultSlug),
		tokens:         tokens,
	}
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
	if result.MatchesInserted > 0 || postResult.CareerSynced {
		result.PostSync = &postResult
		slog.InfoContext(ctx, "sync: pipeline post-sync terminé",
			"gamertag", e.gamertag,
			"perf_scores", postResult.PerfScoresComputed,
			"lusr_updated", postResult.LUSRUpdated,
			"career_synced", postResult.CareerSynced,
			"views_refreshed", postResult.ViewsRefreshed,
		)
	}

	// ─── sync_meta ──────────────────────────────────────────────────────────────
	if err := SetSyncMeta(playerDB, "last_delta_sync", time.Now().UTC().Format(time.RFC3339)); err != nil {
		result.AddWarning(fmt.Sprintf("SetSyncMeta: %v", err))
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
		CareerSynced: e.runCareerSync(ctx, playerDB, client),
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
