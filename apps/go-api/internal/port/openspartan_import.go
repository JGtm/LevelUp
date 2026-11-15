package port

import (
	"context"
	"errors"

	"levelup/go-api/internal/openspartan"
)

// Erreurs de l'import d'une base OpenSpartan ; leur identité fixe le code d'échec du job.
var (
	// ErrOpenSpartanXUIDMismatch is returned when the OpenSpartan database's detected
	// owner XUID does not match the caller's session XUID. Critical for
	// security: prevents a user from importing another user's database.
	ErrOpenSpartanXUIDMismatch = errors.New("openspartan import: owner xuid does not match session")

	// ErrOpenSpartanLowConfidence is returned when the owner detection cannot reach at
	// least medium confidence. The caller can decide whether to retry with
	// an explicit hint or surface the error to the user.
	ErrOpenSpartanLowConfidence = errors.New("openspartan import: owner detection confidence too low")
)

// OpenSpartanImportOptions tunes the import behaviour.
type OpenSpartanImportOptions struct {
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

// OpenSpartanImportResult summarises one import run.
type OpenSpartanImportResult struct {
	DetectedOwnerXUID    string
	Confidence           openspartan.Confidence
	TotalMatches         int
	InsertedMatches      int
	InsertedMatchIDs     []string // populated as registry inserts succeed; consumed by post-import recompute
	InsertedParticipants int
	InsertedMedals       int
	InsertedHighlights   int
	InsertedCSRs         int // shared.match_csrs écrits depuis RankRecap (par-match CSR)
	InsertedAliases      int // alias de l'importeur mis à jour + alias de tiers proposés
	// SkippedForeignMatches : matchs du fichier où l'importeur ne joue pas (non écrits).
	SkippedForeignMatches int
	StashedFriends        int
	Errors                []OpenSpartanImportError
}

// OpenSpartanImportError records a single non-fatal failure during the import.
type OpenSpartanImportError struct {
	MatchID string
	Stage   string
	Err     string
}

// OpenSpartanImportService imports one OpenSpartan SQLite database into the shared match store;
// implémenté par service.OpenSpartanImportService.
type OpenSpartanImportService interface {
	// Import refuses a database whose detected owner is not expectedOwnerXUID
	// (ErrOpenSpartanXUIDMismatch, ErrOpenSpartanLowConfidence).
	Import(ctx context.Context, expectedOwnerXUID, dbPath string, opts OpenSpartanImportOptions) (OpenSpartanImportResult, error)
}

// OpenSpartanPostImportResult summarises one post-import recompute run.
type OpenSpartanPostImportResult struct {
	SessionsTouched     int
	PerfScoresTouched   int
	CSRProjected        int // lignes CSR projetées shared.match_csrs → player.match_skill_rank
	LUSRRecomputed      int // matchs LUSR recalculés (replay complet, inclut les matchs importés)
	CitationsBackfilled bool
	Errors              []OpenSpartanPostImportError
}

// OpenSpartanPostImportError records a single non-fatal failure during the recompute.
type OpenSpartanPostImportError struct {
	Stage   string
	MatchID string // optional — set when the error scopes to a single match
	Err     string
}

// OpenSpartanPostImportOptions tunes recompute behaviour. Sessions follow the title's session
// rule (settings.SessionOptions), never a per-import value.
type OpenSpartanPostImportOptions struct {
	TitleSlug       string // defaults to halo_infinite
	ForcePerfScores bool   // recompute even if score already exists with the same chain
}

// OpenSpartanPostImportService recomputes what derives from the imported matches (CSR, LUSR, sessions,
// perf scores, citations) for the player who imported them; implémenté par
// service.OpenSpartanPostImportService. An error is returned only on fatal initialisation failures;
// per-stage failures are surfaced in the result.
type OpenSpartanPostImportService interface {
	Run(ctx context.Context, xuid, gamertag string, matchIDs []string, opts OpenSpartanPostImportOptions) (OpenSpartanPostImportResult, error)
}
