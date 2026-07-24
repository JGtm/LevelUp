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
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/sync"
)

// OpenSpartanPostImportService runs the three recompute stages for a player
// after their OpenSpartan import succeeded.
//
// Sprint B1 commit 15 : plus de handle sharedDB persistant. Le Provider
// (cfg.SharedProvider) est utilisé à chaque appel via AcquireSharedWriterStandalone.
// Évite le conflit "different configuration" au boot du serveur.
type OpenSpartanPostImportService struct {
	cfg *config.AppConfig
	log *slog.Logger
}

// PostImportResult summarises one post-import recompute run.
type PostImportResult struct {
	SessionsTouched     int
	PerfScoresTouched   int
	CSRProjected        int // lignes CSR projetées shared.match_csrs → player.match_skill_rank
	LUSRRecomputed      int // matchs LUSR recalculés (replay complet, inclut les matchs importés)
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

// NewOpenSpartanPostImportService construit le service. Le handle shared est
// acquis à la demande via cfg.SharedProvider (commit 15) — pas de handle
// persistant au boot pour éviter le conflit "different configuration".
func NewOpenSpartanPostImportService(cfg *config.AppConfig) *OpenSpartanPostImportService {
	return &OpenSpartanPostImportService{
		cfg: cfg,
		log: slog.Default(),
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

	pr := titlePkg.NewPathResolver(s.cfg.RepoRoot)
	playerDBPath := config.PlayerDBPath(s.cfg, opts.TitleSlug, gamertag)
	sharedDBPath := config.SharedDBPath(s.cfg, opts.TitleSlug)
	metadataDBPath := pr.MetadataDBPath(opts.TitleSlug)
	pveDBPath := pr.SharedPVEDBPath(opts.TitleSlug)

	playerHandle, err := sync.OpenPlayerDB(playerDBPath)
	if err != nil {
		return PostImportResult{}, fmt.Errorf("OpenPlayerDB %s: %w", playerDBPath, err)
	}
	defer playerHandle.Close()
	playerDB := playerHandle.SQLDb()

	var result PostImportResult
	s.ensureEnrichmentRows(ctx, playerDB, matchIDs, &result)
	s.recomputeCSR(ctx, playerDB, sharedDBPath, xuid, &result)
	s.recomputeLUSR(ctx, playerDB, sharedDBPath, xuid, &result)
	s.recomputeSessions(ctx, playerDBPath, sharedDBPath, xuid, opts, &result)
	s.recomputePerfScores(ctx, playerDB, sharedDBPath, xuid, opts.ForcePerfScores, &result)
	s.recomputeCitations(ctx, citationRecomputeInputs{
		sharedDBPath:   sharedDBPath,
		metadataDBPath: metadataDBPath,
		pveDBPath:      pveDBPath,
		xuid:           xuid,
		matchIDs:       matchIDs,
	}, playerDB, &result)
	return result, nil
}

// recomputeCSR projette le CSR par-match du joueur depuis shared.match_csrs
// (écrit à l'import depuis RankRecap) vers player.match_skill_rank, que l'UI lit
// pour afficher le rang par match. Pur local, aucun appel API.
//
// Sprint B1 commit 15 : acquisition du shared writer à la demande via Provider
// (lecture seule ici, mais on réutilise le même helper que recomputePerfScores).
func (s *OpenSpartanPostImportService) recomputeCSR(
	ctx context.Context, playerDB *sql.DB, sharedDBPath, xuid string, result *PostImportResult,
) {
	sharedDB, releaseShared, err := sync.AcquireSharedWriterStandalone(ctxkeys.WithDBWriterLabel(ctx, "openspartan_post_import"), s.cfg.SharedProvider, sharedDBPath)
	if err != nil {
		result.Errors = append(result.Errors, PostImportError{Stage: "csr_acquire", Err: err.Error()})
		s.log.Warn("post_import_csr_acquire_failed", "xuid", xuid, "err", err)
		return
	}
	defer releaseShared()
	n, err := sync.BackfillCSRFromShared(ctx, sharedDB, playerDB, xuid)
	if err != nil {
		result.Errors = append(result.Errors, PostImportError{Stage: "csr", Err: err.Error()})
		s.log.Warn("post_import_csr_failed", "xuid", xuid, "err", err)
		return
	}
	result.CSRProjected = n
}

// recomputeLUSR rejoue le LUSR (v2) du joueur sur tout son historique pour
// intégrer les matchs importés (anciens), que le chemin live incrémental
// sauterait via le watermark. Pur local, aucun appel API. Best-effort.
func (s *OpenSpartanPostImportService) recomputeLUSR(
	ctx context.Context, playerDB *sql.DB, sharedDBPath, xuid string, result *PostImportResult,
) {
	sharedDB, releaseShared, err := sync.AcquireSharedWriterStandalone(ctxkeys.WithDBWriterLabel(ctx, "openspartan_post_import"), s.cfg.SharedProvider, sharedDBPath)
	if err != nil {
		result.Errors = append(result.Errors, PostImportError{Stage: "lusr_acquire", Err: err.Error()})
		s.log.Warn("post_import_lusr_acquire_failed", "xuid", xuid, "err", err)
		return
	}
	defer releaseShared()
	n, err := sync.RecomputeLUSRCanonicalForPlayer(ctx, playerDB, sharedDB, xuid)
	if err != nil {
		result.Errors = append(result.Errors, PostImportError{Stage: "lusr", Err: err.Error()})
		s.log.Warn("post_import_lusr_failed", "xuid", xuid, "err", err)
		return
	}
	result.LUSRRecomputed = n
}

// stagePrimeEnrichment : libellé de l'étape « prime enrichment » dans les
// PostImportError (constante pour éviter la répétition du littéral — goconst).
const stagePrimeEnrichment = "prime_enrichment"

// ensureEnrichmentRows primes player_match_enrichment with one baseline row
// (stage='live') per imported match_id. Required because the recompute stages
// (sessions, performance_score) source their work-list from PME rows.
//
// Append-only #23046 : pure INSERT (no ON CONFLICT — match_id n'est plus une PK).
// Idempotence via pré-filtre delta : seuls les matchs sans aucune row PME reçoivent
// la baseline (évite les doublons stage='live' sur ré-import).
func (s *OpenSpartanPostImportService) ensureEnrichmentRows(
	ctx context.Context, playerDB *sql.DB, matchIDs []string, result *PostImportResult,
) {
	if len(matchIDs) == 0 {
		return
	}
	existing := make(map[string]struct{}, len(matchIDs))
	rows, err := playerDB.QueryContext(ctx, `SELECT match_id FROM player_match_enrichment_latest`)
	if err != nil {
		result.Errors = append(result.Errors, PostImportError{Stage: stagePrimeEnrichment, Err: err.Error()})
		s.log.Warn("post_import_prime_enrichment_load_failed", "err", err)
		return
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			existing[id] = struct{}{}
		}
	}
	_ = rows.Close()

	stmt, err := playerDB.PrepareContext(ctx,
		`INSERT INTO player_match_enrichment (match_id, stage) VALUES (?, 'live')`)
	if err != nil {
		result.Errors = append(result.Errors, PostImportError{Stage: stagePrimeEnrichment, Err: err.Error()})
		s.log.Warn("post_import_prime_enrichment_prepare_failed", "err", err)
		return
	}
	defer stmt.Close()
	for _, id := range matchIDs {
		if _, ok := existing[id]; ok {
			continue
		}
		if _, err := stmt.ExecContext(ctx, id); err != nil {
			result.Errors = append(result.Errors, PostImportError{Stage: stagePrimeEnrichment, MatchID: id, Err: err.Error()})
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
		result.Errors = append(result.Errors, PostImportError{Stage: scopeSessions, Err: err.Error()})
		s.log.Warn("post_import_sessions_failed", "xuid", xuid, "err", err)
		return
	}
	result.SessionsTouched = n
}

// recomputePerfScores runs sync.BatchComputePerformanceScores on the player.
// Skipped silently when the player has fewer than MinMatchesPerChainForRelative
// matches per chain — that's encoded in the helper itself.
//
// Sprint B1 commit 15 : acquisition du shared writer à la demande via Provider.
func (s *OpenSpartanPostImportService) recomputePerfScores(
	ctx context.Context, playerDB *sql.DB, sharedDBPath, xuid string, force bool, result *PostImportResult,
) {
	sharedDB, releaseShared, err := sync.AcquireSharedWriterStandalone(ctxkeys.WithDBWriterLabel(ctx, "openspartan_post_import"), s.cfg.SharedProvider, sharedDBPath)
	if err != nil {
		result.Errors = append(result.Errors, PostImportError{Stage: "perf_scores_acquire", Err: err.Error()})
		s.log.Warn("post_import_perf_scores_acquire_failed", "xuid", xuid, "err", err)
		return
	}
	defer releaseShared()
	n, err := sync.BatchComputePerformanceScores(ctx, playerDB, sharedDB, xuid, force)
	if err != nil {
		result.Errors = append(result.Errors, PostImportError{Stage: "perf_scores", Err: err.Error()})
		s.log.Warn("post_import_perf_scores_failed", "xuid", xuid, "err", err)
		return
	}
	result.PerfScoresTouched = n
}

// citationRecomputeInputs regroupe les entrées de recomputeCitations (garde la
// signature ≤ 7 arguments après l'ajout de pveDBPath — BUG A / I7).
type citationRecomputeInputs struct {
	sharedDBPath   string
	metadataDBPath string
	pveDBPath      string // shared_pve.duckdb (Firefight) — lu en RO, dégradation gracieuse
	xuid           string
	matchIDs       []string
}

// recomputeCitations runs sync.BackfillMatchCitations for the given match
// IDs. Skipped if matchIDs is empty (nothing to do).
//
// Sprint B1 commit 15 : acquisition du shared writer à la demande via Provider.
// BUG A (I7) : shared_pve est désormais ouvert en RO et passé au pipeline pour
// que les citations pve_stat (Firefight) soient calculées ; absent → dégradé.
func (s *OpenSpartanPostImportService) recomputeCitations(
	ctx context.Context,
	in citationRecomputeInputs,
	playerDB *sql.DB,
	result *PostImportResult,
) {
	if len(in.matchIDs) == 0 {
		return
	}
	metaSQL, releaseMeta, err := sync.AcquireMetadataWriterStandalone(ctx, in.metadataDBPath)
	if err != nil {
		result.Errors = append(result.Errors, PostImportError{Stage: "open_metadata", Err: err.Error()})
		s.log.Warn("post_import_metadata_unavailable", "err", err)
		return
	}
	defer releaseMeta()
	sharedDB, releaseShared, err := sync.AcquireSharedWriterStandalone(ctxkeys.WithDBWriterLabel(ctx, "openspartan_post_import"), s.cfg.SharedProvider, in.sharedDBPath)
	if err != nil {
		result.Errors = append(result.Errors, PostImportError{Stage: "citations_acquire", Err: err.Error()})
		s.log.Warn("post_import_citations_acquire_failed", "xuid", in.xuid, "err", err)
		return
	}
	defer releaseShared()
	pveDB, releasePve := sync.OpenPveReadForCitations(ctx, in.pveDBPath)
	defer releasePve()
	if err := sync.BackfillMatchCitations(ctx, metaSQL, sharedDB, playerDB, pveDB, in.xuid, in.matchIDs); err != nil {
		result.Errors = append(result.Errors, PostImportError{Stage: "citations", Err: err.Error()})
		s.log.Warn("post_import_citations_failed", "xuid", in.xuid, "err", err)
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
