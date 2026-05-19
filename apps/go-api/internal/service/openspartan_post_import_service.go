// Package service — openspartan_post_import_service.go: recomputes the v6
// derived fields (sessions, performance scores, citations) after a bulk
// OpenSpartan import has finished writing the raw rows.
//
// The post-import service touches the per-player stats.duckdb (creating it
// via sync.OpenPlayerDB if missing — typical at onboarding time) and reads
// citation_mappings from metadata.duckdb. Each stage runs best-effort:
// per-stage failure goes into PostImportResult.Errors without aborting the
// remaining stages.
package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	platform_duckdb "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync"
)

// OpenSpartanPostImportService runs the three recompute stages for a player
// after their OpenSpartan import succeeded.
type OpenSpartanPostImportService struct {
	cfg      *config.AppConfig
	sharedDB *sql.DB
	log      *slog.Logger
}

// PostImportResult summarises one post-import recompute run.
type PostImportResult struct {
	SessionsTouched     int
	PerfScoresTouched   int
	CitationsBackfilled bool
	Errors              []PostImportError
}

// PostImportError records a single non-fatal failure during the recompute.
type PostImportError struct {
	Stage   string
	MatchID string // optional — set when the error scopes to a single match
	Err     string
}

// PostImportOptions tunes recompute behaviour.
type PostImportOptions struct {
	TitleSlug         string // defaults to halo_infinite
	ForcePerfScores   bool   // recompute even if score already exists with the same chain
	SessionGapMinutes int    // gap that breaks a session, defaults to 15
}

// NewOpenSpartanPostImportService constructs a service bound to a shared
// DuckDB connection opened in read-write mode.
func NewOpenSpartanPostImportService(cfg *config.AppConfig, sharedDB *sql.DB) *OpenSpartanPostImportService {
	return &OpenSpartanPostImportService{
		cfg:      cfg,
		sharedDB: sharedDB,
		log:      slog.Default(),
	}
}

// Run executes the three recompute stages. Returns a non-nil error only on
// fatal initialisation failures (missing required inputs, player DB cannot
// be opened); per-stage failures are surfaced via PostImportResult.Errors.
func (s *OpenSpartanPostImportService) Run(
	ctx context.Context,
	xuid, gamertag string,
	matchIDs []string,
	opts PostImportOptions,
) (PostImportResult, error) {
	if xuid == "" || gamertag == "" {
		return PostImportResult{}, errors.New("post-import: xuid and gamertag are required")
	}
	opts = applyPostImportDefaults(opts)

	playerDBPath := config.PlayerDBPath(s.cfg, opts.TitleSlug, gamertag)
	sharedDBPath := config.SharedDBPath(s.cfg, opts.TitleSlug)
	metadataDBPath := titlePkg.NewPathResolver(s.cfg.RepoRoot).MetadataDBPath(opts.TitleSlug)

	playerHandle, err := sync.OpenPlayerDB(playerDBPath)
	if err != nil {
		return PostImportResult{}, fmt.Errorf("OpenPlayerDB %s: %w", playerDBPath, err)
	}
	defer playerHandle.Close()
	playerDB := playerHandle.SQLDb()

	var result PostImportResult
	s.ensureEnrichmentRows(ctx, playerDB, matchIDs, &result)
	s.recomputeSessions(ctx, playerDBPath, sharedDBPath, xuid, opts, &result)
	s.recomputePerfScores(playerDB, xuid, opts.ForcePerfScores, &result)
	s.recomputeCitations(ctx, metadataDBPath, playerDB, xuid, matchIDs, &result)
	return result, nil
}

// ensureEnrichmentRows primes player_match_enrichment with one placeholder
// row per imported match_id. Required because the recompute stages (sessions,
// performance_score) update those rows in place — without a prior INSERT
// they would silently no-op.
//
// Idempotent via ON CONFLICT DO NOTHING — a no-op for matches already
// enriched by a previous sync.
func (s *OpenSpartanPostImportService) ensureEnrichmentRows(
	ctx context.Context, playerDB *sql.DB, matchIDs []string, result *PostImportResult,
) {
	if len(matchIDs) == 0 {
		return
	}
	stmt, err := playerDB.PrepareContext(ctx,
		`INSERT INTO player_match_enrichment (match_id) VALUES (?) ON CONFLICT (match_id) DO NOTHING`)
	if err != nil {
		result.Errors = append(result.Errors, PostImportError{Stage: "prime_enrichment", Err: err.Error()})
		s.log.Warn("post_import_prime_enrichment_prepare_failed", "err", err)
		return
	}
	defer stmt.Close()
	for _, id := range matchIDs {
		if _, err := stmt.ExecContext(ctx, id); err != nil {
			result.Errors = append(result.Errors, PostImportError{Stage: "prime_enrichment", MatchID: id, Err: err.Error()})
			s.log.Warn("post_import_prime_enrichment_exec_failed", "match_id", id, "err", err)
			return
		}
	}
}

// recomputeSessions runs sync.RecalculatePlayerSessions. The helper acquires
// its own write leases on player and shared DBs internally.
func (s *OpenSpartanPostImportService) recomputeSessions(
	ctx context.Context,
	playerDBPath, sharedDBPath, xuid string,
	opts PostImportOptions,
	result *PostImportResult,
) {
	n, err := sync.RecalculatePlayerSessions(ctx, s.cfg.SharedProvider, playerDBPath, sharedDBPath, xuid,
		domain.SessionComputeOptions{
			GapMinutes:     opts.SessionGapMinutes,
			TeamChangeMode: "cluster",
		}, nil,
	)
	if err != nil {
		result.Errors = append(result.Errors, PostImportError{Stage: "sessions", Err: err.Error()})
		s.log.Warn("post_import_sessions_failed", "xuid", xuid, "err", err)
		return
	}
	result.SessionsTouched = n
}

// recomputePerfScores runs sync.BatchComputePerformanceScores on the player.
// Skipped silently when the player has fewer than MinMatchesPerChainForRelative
// matches per chain — that's encoded in the helper itself.
func (s *OpenSpartanPostImportService) recomputePerfScores(
	playerDB *sql.DB, xuid string, force bool, result *PostImportResult,
) {
	n, err := sync.BatchComputePerformanceScores(playerDB, s.sharedDB, xuid, force)
	if err != nil {
		result.Errors = append(result.Errors, PostImportError{Stage: "perf_scores", Err: err.Error()})
		s.log.Warn("post_import_perf_scores_failed", "xuid", xuid, "err", err)
		return
	}
	result.PerfScoresTouched = n
}

// recomputeCitations runs sync.BackfillMatchCitations for the given match
// IDs. Skipped if matchIDs is empty (nothing to do).
func (s *OpenSpartanPostImportService) recomputeCitations(
	ctx context.Context,
	metadataDBPath string,
	playerDB *sql.DB,
	xuid string,
	matchIDs []string,
	result *PostImportResult,
) {
	if len(matchIDs) == 0 {
		return
	}
	metaHandle, err := platform_duckdb.OpenReadWriteShared(metadataDBPath)
	if err != nil {
		result.Errors = append(result.Errors, PostImportError{Stage: "open_metadata", Err: err.Error()})
		s.log.Warn("post_import_metadata_unavailable", "err", err)
		return
	}
	defer metaHandle.Close()
	if err := sync.BackfillMatchCitations(ctx, metaHandle.SQLDb(), s.sharedDB, playerDB, xuid, matchIDs); err != nil {
		result.Errors = append(result.Errors, PostImportError{Stage: "citations", Err: err.Error()})
		s.log.Warn("post_import_citations_failed", "xuid", xuid, "err", err)
		return
	}
	result.CitationsBackfilled = true
}

func applyPostImportDefaults(opts PostImportOptions) PostImportOptions {
	if opts.TitleSlug == "" {
		opts.TitleSlug = titlePkg.DefaultSlug
	}
	if opts.SessionGapMinutes <= 0 {
		opts.SessionGapMinutes = 15
	}
	return opts
}
