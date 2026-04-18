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
	"log"
	"path/filepath"
	"time"

	_ "github.com/duckdb/duckdb-go/v2" // driver DuckDB

	"levelup/go-api/internal/domain"
)

const (
	// historyPageSize est le nombre de matchs demandés par page API.
	historyPageSize = 25
	// leaseTimeout est le délai max pour acquérir un write lease.
	leaseTimeout = 10 * time.Second
)

// SyncEngine orchestre la synchronisation des données Halo d'un joueur.
type SyncEngine struct {
	gamertag     string
	xuid         string
	playerDBPath string
	sharedDBPath string
	tokens       *domain.HaloTokens
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
	return &SyncEngine{
		gamertag:     gamertag,
		xuid:         xuid,
		playerDBPath: filepath.Join(repoRoot, "data", "players", gamertag, "stats.duckdb"),
		sharedDBPath: filepath.Join(repoRoot, "data", "warehouse", "shared_matches_v2.duckdb"),
		tokens:       tokens,
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
	_ = ctx // reserved for future cancellation support

	// ─── Write leases (lecture seule suffit pour la détection) ───────────
	playerDB, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return nil, fmt.Errorf("RunBackfill OpenPlayerDB: %w", err)
	}
	defer playerDB.Close()

	sharedDB, err := OpenSharedDB(e.sharedDBPath)
	if err != nil {
		return nil, fmt.Errorf("RunBackfill OpenSharedDB: %w", err)
	}
	defer sharedDB.Close()

	return FindMatchesMissingData(playerDB, sharedDB, e.xuid, scope)
}

// run est le cœur du moteur de sync. isDelta=true → stop dès un match connu.
func (e *SyncEngine) run(ctx context.Context, opts domain.SyncOptions, isDelta bool) (domain.SyncResult, error) {
	result := domain.SyncResult{StartedAt: time.Now()}

	// ─── Write leases ──────────────────────────────────────────────────────────
	relPlayer, err := AcquireLease(e.playerDBPath, leaseTimeout)
	if err != nil {
		return result, fmt.Errorf("run: %w", err)
	}
	defer relPlayer()

	relShared, err := AcquireLease(e.sharedDBPath, leaseTimeout)
	if err != nil {
		return result, fmt.Errorf("run: %w", err)
	}
	defer relShared()

	// ─── Ouverture des DBs ─────────────────────────────────────────────────────
	playerDB, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return result, fmt.Errorf("run OpenPlayerDB: %w", err)
	}
	defer playerDB.Close()

	sharedDB, err := OpenSharedDB(e.sharedDBPath)
	if err != nil {
		return result, fmt.Errorf("run OpenSharedDB: %w", err)
	}
	defer sharedDB.Close()

	// ─── Match IDs déjà connus (player DB) ───────────────────────────────────
	known, err := loadKnownMatchIDs(playerDB)
	if err != nil {
		return result, fmt.Errorf("run loadKnownMatchIDs: %w", err)
	}

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

		entries, err := client.GetMatchHistory(ctx, e.gamertag, opts.MatchType, start, historyPageSize)
		if err != nil {
			result.AddWarning(fmt.Sprintf("GetMatchHistory(start=%d): %v", start, err))
			break
		}
		if len(entries) == 0 {
			break // fin de l'historique
		}

		allKnown := true
		for _, entry := range entries {
			if processed >= opts.MaxMatches {
				break
			}
			if known[entry.MatchID] {
				result.MatchesSkipped++
				if isDelta {
					// En mode delta, on s'arrête dès le premier match connu.
					goto done
				}
				continue
			}
			allKnown = false

			if err := e.processMatch(ctx, client, sharedDB, playerDB, &result, entry.MatchID, opts); err != nil {
				result.AddWarning(fmt.Sprintf("processMatch(%s): %v", entry.MatchID, err))
			} else {
				processed++
			}
		}

		if isDelta && allKnown {
			break
		}
		start += len(entries)
	}

done:
	// ─── Pipeline post-sync ─────────────────────────────────────────────────────
	if result.MatchesInserted > 0 {
		postResult := e.runPostSyncPipeline(ctx, playerDB, sharedDB, client)
		result.PostSync = &postResult
	}

	// ─── sync_meta ──────────────────────────────────────────────────────────────
	if err := SetSyncMeta(playerDB, "last_delta_sync", time.Now().UTC().Format(time.RFC3339)); err != nil {
		result.AddWarning(fmt.Sprintf("SetSyncMeta: %v", err))
	}

	result.FinishedAt = time.Now()
	result.DurationSeconds = result.FinishedAt.Sub(result.StartedAt).Seconds()
	return result, nil
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
	matchJSON, err := client.GetMatchStats(ctx, matchID)
	if err != nil {
		return fmt.Errorf("GetMatchStats: %w", err)
	}

	// ─── match_registry ────────────────────────────────────────────────────────
	reg, err := ExtractRegistry(matchJSON, e.gamertag)
	if err != nil {
		return fmt.Errorf("ExtractRegistry: %w", err)
	}
	if err := InsertRegistryIfNotExists(sharedDB, *reg); err != nil {
		return fmt.Errorf("InsertRegistry: %w", err)
	}

	// ─── match_participants ────────────────────────────────────────────────────
	if opts.WithParticipants {
		participants := ExtractParticipants(matchJSON)
		if err := InsertParticipants(sharedDB, participants); err != nil {
			return fmt.Errorf("InsertParticipants: %w", err)
		}
		result.ParticipantsDone += len(participants)

		// xuid_aliases pour tous les participants
		for _, p := range participants {
			if p.Gamertag != nil && *p.Gamertag != "" {
				_ = UpsertXUIDAlias(sharedDB, p.XUID, *p.Gamertag)
			}
		}
	}

	// ─── medals_earned ─────────────────────────────────────────────────────────
	if opts.WithMedals {
		medals := ExtractMedals(matchJSON)
		if err := InsertMedals(sharedDB, medals); err != nil {
			return fmt.Errorf("InsertMedals: %w", err)
		}
		result.MedalsInserted += len(medals)
	}

	// ─── player_match_enrichment (player DB) ───────────────────────────────────
	if err := UpsertPlayerEnrichment(playerDB, matchID, ""); err != nil {
		return fmt.Errorf("UpsertPlayerEnrichment: %w", err)
	}

	result.MatchesInserted++
	result.InsertedMatchIDs = append(result.InsertedMatchIDs, matchID)
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
	if n, err := batchComputePerformanceScores(playerDB, sharedDB, e.xuid); err != nil {
		log.Printf("[post-sync] WARN perf scores: %v", err)
	} else {
		r.PerfScoresComputed = n
	}

	// 2. LUSR (TrueSkill 2)
	if n, err := batchComputeLUSR(playerDB, sharedDB, e.xuid); err != nil {
		log.Printf("[post-sync] WARN LUSR: %v", err)
	} else {
		r.LUSRUpdated = n
	}

	// 3. Career rank
	if data, err := syncCareerRank(ctx, client, e.xuid); err != nil {
		log.Printf("[post-sync] WARN career: %v", err)
	} else if data != nil {
		if err := saveCareerRank(playerDB, data); err != nil {
			log.Printf("[post-sync] WARN career save: %v", err)
		} else {
			r.CareerSynced = true
		}
	}

	// 4. Aggregates (materialized views)
	if n, err := refreshAggregates(playerDB); err != nil {
		log.Printf("[post-sync] WARN aggregates: %v", err)
	} else {
		r.ViewsRefreshed = n
	}
	if n, err := refreshSharedViews(sharedDB); err != nil {
		log.Printf("[post-sync] WARN shared views: %v", err)
	} else {
		r.ViewsRefreshed += n
	}

	return r
}
