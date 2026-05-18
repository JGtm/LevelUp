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
type OpenSpartanImportService struct {
	sharedDB *sql.DB
	log      *slog.Logger
}

// NewOpenSpartanImportService constructs a service bound to a shared
// DuckDB connection opened in read-write mode.
func NewOpenSpartanImportService(sharedDB *sql.DB) *OpenSpartanImportService {
	return &OpenSpartanImportService{
		sharedDB: sharedDB,
		log:      slog.Default(),
	}
}

// SetLogger swaps the default slog logger. Useful for tests and integration.
func (s *OpenSpartanImportService) SetLogger(log *slog.Logger) {
	if log != nil {
		s.log = log
	}
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

	r, err := openspartan.Open(dbPath)
	if err != nil {
		return result, fmt.Errorf("open openspartan db: %w", err)
	}
	defer r.Close()

	if err := s.validateOwner(ctx, r, dbPath, expectedOwnerXUID, &result); err != nil {
		return result, err
	}

	total, err := r.MatchCount(ctx)
	if err != nil {
		return result, fmt.Errorf("match count: %w", err)
	}
	result.TotalMatches = total

	aliases, err := r.AliasMap(ctx)
	if err != nil {
		return result, fmt.Errorf("load aliases: %w", err)
	}
	resolver := func(xuid string) string { return aliases[xuid] }

	if err := s.importMatches(ctx, r, resolver, opts, &result); err != nil {
		return result, err
	}
	if err := s.importHighlights(ctx, r, opts, &result); err != nil {
		return result, err
	}
	s.importAliases(ctx, r, opts, &result)
	s.stashFriends(ctx, r, expectedOwnerXUID, opts, &result)

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
		s.writeOneMatch(pm, mapOpts, opts.DryRun, result)
	}
	return nil
}

// writeOneMatch maps + writes one match's contribution to the shared DB.
// Errors per stage are recorded but never abort the whole import.
func (s *OpenSpartanImportService) writeOneMatch(
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
	if err := sync.InsertRegistryIfNotExists(s.sharedDB, toSyncRegistry(mm.Registry)); err != nil {
		result.Errors = append(result.Errors, ImportError{MatchID: pm.MatchID, Stage: "insert_registry", Err: err.Error()})
		return
	}
	result.InsertedMatches++
	if err := sync.InsertParticipants(s.sharedDB, toSyncParticipants(mm.Participants)); err != nil {
		result.Errors = append(result.Errors, ImportError{MatchID: pm.MatchID, Stage: "insert_participants", Err: err.Error()})
	} else {
		result.InsertedParticipants += len(mm.Participants)
	}
	if err := sync.InsertMedals(s.sharedDB, toSyncMedals(mm.Medals)); err != nil {
		result.Errors = append(result.Errors, ImportError{MatchID: pm.MatchID, Stage: "insert_medals", Err: err.Error()})
	} else {
		result.InsertedMedals += len(mm.Medals)
	}
}

// importHighlights walks HighlightEvents and writes one event per row.
func (s *OpenSpartanImportService) importHighlights(
	ctx context.Context,
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
		n, err := sync.InsertHighlightEvents(s.sharedDB, row.MatchID, []analysis.HighlightEvent{event})
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
		if err := sync.UpsertXUIDAlias(s.sharedDB, a.XUID, a.Gamertag); err != nil {
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
