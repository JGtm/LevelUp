// Package service — openspartan_import_service.go: orchestrates a one-shot
// import of an OpenSpartan SQLite database into the shared DuckDB.
//
// The service does NOT manage the SSO session nor the HTTP layer — those
// live in handlers. It receives a caller-supplied owner XUID, validates it
// against the database's own detected owner, then walks the matches and
// writes via the existing sync.Insert* helpers. Friends are stashed as JSON
// for the future MULTIUSER_ACL sprint to consume.
//
// PR 3.A scope: the service is testable end-to-end via integration tests.
// PR 3.B will add the HTTP endpoint, multipart upload handling, and job
// system integration.
package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/openspartan"
	"levelup/go-api/internal/openspartan/mapper"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/sync"
)

var (
	// ErrXUIDMismatch is returned when the OpenSpartan database's detected
	// owner XUID does not match the caller's session XUID. Critical for
	// security: prevents a user from importing another user's database.
	ErrXUIDMismatch = errors.New("openspartan import: owner xuid does not match session")

	// ErrLowConfidence is returned when the owner detection cannot reach at
	// least medium confidence. The caller can decide whether to retry with
	// an explicit hint or surface the error to the user.
	ErrLowConfidence = errors.New("openspartan import: owner detection confidence too low")
)

// ImportOptions tunes the import behaviour.
type ImportOptions struct {
	// DryRun, when true, counts what would be inserted but performs no writes.
	DryRun bool
	// Source labels the import in match_registry.first_sync_by.
	// Defaults to "openspartan_import" when empty.
	Source string
	// StashDir is the root directory under which the Friends JSON stash is
	// written. Defaults to "./data/players" when empty. The actual file path
	// is `<StashDir>/<ownerXUID>/stash/openspartan_friends.json`.
	StashDir string
	// OnProgress, when non-nil, is invoked after each parsed match with
	// (parsed, total) counts. Intended for job system integration.
	OnProgress func(parsed, total int)
}

// ImportResult summarises one import run.
type ImportResult struct {
	DetectedOwnerXUID    string
	Confidence           openspartan.Confidence
	TotalMatches         int
	InsertedMatches      int
	InsertedMatchIDs     []string // populated as registry inserts succeed; consumed by post-import recompute
	InsertedParticipants int
	InsertedMedals       int
	InsertedHighlights   int
	InsertedAliases      int
	StashedFriends       int
	Errors               []ImportError
}

// ImportError records a single non-fatal failure during the import.
type ImportError struct {
	MatchID string
	Stage   string
	Err     string
}

// OpenSpartanImportService orchestrates one import from an OpenSpartan
// SQLite database into the shared DuckDB.
//
// Sprint B1 commit 15 : acquisition on-demand du writer shared via le
// Provider, plutôt qu'un handle RW persistant ouvert au boot.
//
// Pourquoi : OpenSpartan import est un flow d'onboarding RARE (un user qui
// veut importer ses données depuis OpenSpartan vers LevelUp). Garder un
// handle RW shared ouvert pour la vie du process créait un conflit avec
// le Provider qui tient shared en RO en steady state ("different
// configuration" au boot).
//
// Modes :
//   - Production (provider != nil) : acquisition à chaque Import via
//     Provider.AcquireWriter. Coordination automatique avec auto_sync.
//   - Tests in-memory (legacyDB != nil) : utilise le handle injecté sans
//     coordination Provider. Utiliser uniquement pour les tests.
type OpenSpartanImportService struct {
	provider     sharedprovider.Provider // mode B-swap (production)
	legacyDB     *sql.DB                 // tests in-memory uniquement
	sharedDBPath string                  // utilisé en mode legacy si provider nil
	log          *slog.Logger
}

// NewOpenSpartanImportService construit un service en mode production —
// l'écriture shared passe par Provider.AcquireWriter à chaque Import.
// sharedDBPath conservé pour mode kill-switch (provider nil, dblease + open).
func NewOpenSpartanImportService(provider sharedprovider.Provider, sharedDBPath string) *OpenSpartanImportService {
	return &OpenSpartanImportService{
		provider:     provider,
		sharedDBPath: sharedDBPath,
		log:          slog.Default(),
	}
}

// NewOpenSpartanImportServiceForTest construit un service avec un handle
// shared in-memory injecté. PAS DE COORDINATION avec Provider — usage tests
// uniquement, où un sharedDB DuckDB in-memory est partagé entre setup et
// service. Ne PAS utiliser en production.
func NewOpenSpartanImportServiceForTest(sharedDB *sql.DB) *OpenSpartanImportService {
	return &OpenSpartanImportService{
		legacyDB: sharedDB,
		log:      slog.Default(),
	}
}

// SetLogger swaps the default slog logger. Useful for tests and integration.
func (s *OpenSpartanImportService) SetLogger(log *slog.Logger) {
	if log != nil {
		s.log = log
	}
}

// acquireShared retourne un *sql.DB pour les écritures shared + une fonction
// release à appeler via defer. Stratégie selon la configuration du service :
//   - Tests (legacyDB != nil) : retourne le handle injecté, release no-op.
//   - DryRun : aucun handle nécessaire (pas d'écriture), release no-op.
//   - Production (provider != nil) : Provider.AcquireWriter (coordonné B-swap).
//   - Fallback legacy (sharedDBPath != ""): AcquireSharedWriterStandalone.
func (s *OpenSpartanImportService) acquireShared(ctx context.Context, dryRun bool) (*sql.DB, func(), error) {
	if s.legacyDB != nil {
		return s.legacyDB, func() {}, nil
	}
	if dryRun {
		// Pas d'écriture en dryRun — retourne nil + no-op release, les helpers
		// ne dereferencent pas le handle sur le code path dryRun.
		return nil, func() {}, nil
	}
	return sync.AcquireSharedWriterStandalone(ctx, s.provider, s.sharedDBPath)
}

// Import is the entry point. expectedOwnerXUID comes from the caller's SSO
// session — the service refuses to import a database whose detected owner
// does not match.
func (s *OpenSpartanImportService) Import(
	ctx context.Context,
	expectedOwnerXUID string,
	dbPath string,
	opts ImportOptions,
) (ImportResult, error) {
	opts = withDefaults(opts)
	var result ImportResult

	s.log.Info("openspartan_import_started",
		"expected_xuid", expectedOwnerXUID, "db_path", dbPath, "dry_run", opts.DryRun)

	// Sprint B1 commit 15 : acquisition on-demand du writer shared.
	//   - Mode B-swap (provider != nil) : Provider.AcquireWriter coordonne
	//     avec auto_sync et les readers HTTP (PreSwap → DETACH pool, OpenRW,
	//     puis re-OpenRO post-Release). Aucun handle RW persistant.
	//   - Mode legacy (provider nil, sharedDBPath != "") : sync.AcquireSharedWriterStandalone
	//     (dblease + OpenSharedDB direct).
	//   - Mode tests (legacyDB != nil) : utilise le handle injecté tel quel.
	//   - DryRun : skip toute acquisition (pas d'écriture).
	sharedDB, releaseShared, err := s.acquireShared(ctx, opts.DryRun)
	if err != nil {
		return result, fmt.Errorf("openspartan import acquire shared: %w", err)
	}
	defer releaseShared()

	r, err := openspartan.Open(dbPath)
	if err != nil {
		s.log.Error("openspartan_import_open_failed", "db_path", dbPath, "err", err)
		return result, fmt.Errorf("open openspartan db: %w", err)
	}
	defer r.Close()

	if err := s.validateOwner(ctx, r, dbPath, expectedOwnerXUID, &result); err != nil {
		s.log.Warn("openspartan_import_owner_validation_failed",
			"detected_xuid", result.DetectedOwnerXUID, "confidence", result.Confidence.String(), "err", err)
		return result, err
	}

	total, err := r.MatchCount(ctx)
	if err != nil {
		return result, fmt.Errorf("match count: %w", err)
	}
	result.TotalMatches = total
	s.log.Info("openspartan_import_matches_to_process", "total", total)

	aliases, err := r.AliasMap(ctx)
	if err != nil {
		return result, fmt.Errorf("load aliases: %w", err)
	}
	resolver := func(xuid string) string { return aliases[xuid] }

	if err := s.importMatches(ctx, sharedDB, r, resolver, opts, &result); err != nil {
		return result, err
	}
	if err := s.importHighlights(ctx, sharedDB, r, opts, &result); err != nil {
		return result, err
	}
	s.importAliases(ctx, sharedDB, r, opts, &result)
	s.stashFriends(ctx, r, expectedOwnerXUID, opts, &result)

	s.log.Info("openspartan_import_completed",
		"inserted_matches", result.InsertedMatches,
		"inserted_participants", result.InsertedParticipants,
		"inserted_medals", result.InsertedMedals,
		"inserted_highlights", result.InsertedHighlights,
		"inserted_aliases", result.InsertedAliases,
		"stashed_friends", result.StashedFriends,
		"errors", len(result.Errors))
	return result, nil
}

// validateOwner runs DetectOwner and refuses the import if the XUID
// doesn't match the session's, or if confidence is below medium.
func (s *OpenSpartanImportService) validateOwner(
	ctx context.Context,
	r *openspartan.Reader,
	dbPath, expectedXUID string,
	result *ImportResult,
) error {
	detected, conf, err := r.DetectOwner(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("detect owner: %w", err)
	}
	result.DetectedOwnerXUID = detected
	result.Confidence = conf

	if conf == openspartan.ConfidenceNone || conf == openspartan.ConfidenceLow {
		return fmt.Errorf("%w: %s", ErrLowConfidence, conf)
	}
	if detected != expectedXUID {
		return fmt.Errorf("%w: detected=%s expected=%s", ErrXUIDMismatch, detected, expectedXUID)
	}
	return nil
}

// importMatches iterates every match in the reader, applies the mapper,
// and writes registry/participants/medals via the sync.* helpers.
func (s *OpenSpartanImportService) importMatches(
	ctx context.Context,
	sharedDB *sql.DB,
	r *openspartan.Reader,
	resolver func(string) string,
	opts ImportOptions,
	result *ImportResult,
) error {
	parsed := 0
	mapOpts := mapper.MapOptions{
		AliasResolver: resolver,
		Source:        opts.Source,
	}
	for pm, err := range r.Matches(ctx) {
		if err != nil {
			result.Errors = append(result.Errors, ImportError{Stage: "parse_match", Err: err.Error()})
			continue
		}
		parsed++
		if opts.OnProgress != nil {
			opts.OnProgress(parsed, result.TotalMatches)
		}
		s.writeOneMatch(sharedDB, pm, mapOpts, opts.DryRun, result)
	}
	return nil
}

// writeOneMatch maps + writes one match's contribution to the shared DB.
// Errors per stage are recorded but never abort the whole import.
func (s *OpenSpartanImportService) writeOneMatch(
	sharedDB *sql.DB,
	pm *openspartan.ParsedMatch,
	mapOpts mapper.MapOptions,
	dryRun bool,
	result *ImportResult,
) {
	mm, err := mapper.MapMatch(pm, mapOpts)
	if err != nil {
		result.Errors = append(result.Errors, ImportError{MatchID: pm.MatchID, Stage: "map", Err: err.Error()})
		return
	}
	if dryRun {
		result.InsertedMatches++
		result.InsertedParticipants += len(mm.Participants)
		result.InsertedMedals += len(mm.Medals)
		return
	}
	if err := sync.InsertRegistryIfNotExists(sharedDB, toSyncRegistry(mm.Registry)); err != nil {
		result.Errors = append(result.Errors, ImportError{MatchID: pm.MatchID, Stage: "insert_registry", Err: err.Error()})
		return
	}
	result.InsertedMatches++
	result.InsertedMatchIDs = append(result.InsertedMatchIDs, pm.MatchID)
	if err := sync.InsertParticipants(sharedDB, toSyncParticipants(mm.Participants)); err != nil {
		result.Errors = append(result.Errors, ImportError{MatchID: pm.MatchID, Stage: "insert_participants", Err: err.Error()})
	} else {
		result.InsertedParticipants += len(mm.Participants)
	}
	if err := sync.InsertMedals(sharedDB, toSyncMedals(mm.Medals)); err != nil {
		result.Errors = append(result.Errors, ImportError{MatchID: pm.MatchID, Stage: "insert_medals", Err: err.Error()})
	} else {
		result.InsertedMedals += len(mm.Medals)
	}
}

// importHighlights walks HighlightEvents and writes one event per row.
func (s *OpenSpartanImportService) importHighlights(
	ctx context.Context,
	sharedDB *sql.DB,
	r *openspartan.Reader,
	opts ImportOptions,
	result *ImportResult,
) error {
	for hl, err := range r.Highlights(ctx) {
		if err != nil {
			result.Errors = append(result.Errors, ImportError{Stage: "parse_highlight", Err: err.Error()})
			continue
		}
		row, err := mapper.MapHighlight(hl.MatchID, hl.RawJSON)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{MatchID: hl.MatchID, Stage: "map_highlight", Err: err.Error()})
			continue
		}
		if opts.DryRun {
			result.InsertedHighlights++
			continue
		}
		event := toAnalysisEvent(row)
		n, err := sync.InsertHighlightEvents(sharedDB, row.MatchID, []analysis.HighlightEvent{event})
		if err != nil {
			result.Errors = append(result.Errors, ImportError{MatchID: hl.MatchID, Stage: "insert_highlight", Err: err.Error()})
			continue
		}
		result.InsertedHighlights += n
	}
	return nil
}

// importAliases upserts every XuidAliases row into shared.xuid_aliases.
func (s *OpenSpartanImportService) importAliases(
	ctx context.Context,
	sharedDB *sql.DB,
	r *openspartan.Reader,
	opts ImportOptions,
	result *ImportResult,
) {
	rows, err := r.LoadXuidAliases(ctx)
	if err != nil {
		result.Errors = append(result.Errors, ImportError{Stage: "load_aliases_for_write", Err: err.Error()})
		return
	}
	if opts.DryRun {
		result.InsertedAliases = len(rows)
		return
	}
	for _, a := range rows {
		if err := sync.UpsertXUIDAlias(sharedDB, a.XUID, a.Gamertag); err != nil {
			result.Errors = append(result.Errors, ImportError{Stage: "upsert_alias", Err: err.Error()})
			continue
		}
		result.InsertedAliases++
	}
}

// stashFriends serialises the OpenSpartan Friends table to a JSON file
// under <StashDir>/<ownerXUID>/stash/openspartan_friends.json. No DuckDB
// table is created — the future MULTIUSER_ACL sprint will consume this.
func (s *OpenSpartanImportService) stashFriends(
	ctx context.Context,
	r *openspartan.Reader,
	ownerXUID string,
	opts ImportOptions,
	result *ImportResult,
) {
	friends, err := r.LoadFriends(ctx)
	if err != nil {
		result.Errors = append(result.Errors, ImportError{Stage: "load_friends", Err: err.Error()})
		return
	}
	if len(friends) == 0 {
		return
	}
	if opts.DryRun {
		result.StashedFriends = len(friends)
		return
	}
	if err := writeFriendsStash(opts.StashDir, ownerXUID, friends); err != nil {
		result.Errors = append(result.Errors, ImportError{Stage: "stash_friends", Err: err.Error()})
		return
	}
	result.StashedFriends = len(friends)
}

func writeFriendsStash(stashDir, ownerXUID string, friends []openspartan.FriendRow) error {
	dir := filepath.Join(stashDir, ownerXUID, "stash")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir stash: %w", err)
	}
	file := filepath.Join(dir, "openspartan_friends.json")
	payload := map[string]any{
		"owner_xuid":  ownerXUID,
		"imported_at": time.Now().UTC(),
		"friends":     friends,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal friends: %w", err)
	}
	return os.WriteFile(file, data, 0o644)
}

func withDefaults(opts ImportOptions) ImportOptions {
	if opts.Source == "" {
		opts.Source = "openspartan_import"
	}
	if opts.StashDir == "" {
		opts.StashDir = "./data/players"
	}
	return opts
}
