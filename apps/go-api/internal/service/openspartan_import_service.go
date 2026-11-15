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
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"levelup/go-api/internal/domain/highlightevent"
	"levelup/go-api/internal/openspartan"
	"levelup/go-api/internal/openspartan/mapper"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync"
)

// OpenSpartanImportService orchestrates one import from an OpenSpartan
// SQLite database into the shared DuckDB.
//
// L'écriture de la base partagée passe par port.OpenSpartanSharedAcquirer, pris à chaque Import (flow
// d'onboarding RARE) : le service ne tient aucun handle persistant, ce qui évite le conflit
// « different configuration » avec le fournisseur qui tient la base partagée en lecture seule en régime
// établi. En production l'acquisition est coordonnée avec auto_sync (Provider.AcquireWriter, bascule B-swap).
type OpenSpartanImportService struct {
	shared port.OpenSpartanSharedAcquirer
	log    *slog.Logger
}

// NewOpenSpartanImportService construit le service ; shared prend l'écriture de la base partagée à chaque
// Import non simulé (nil admis tant que seul DryRun est utilisé).
func NewOpenSpartanImportService(shared port.OpenSpartanSharedAcquirer) *OpenSpartanImportService {
	return &OpenSpartanImportService{shared: shared, log: slog.Default()}
}

// SetLogger swaps the default slog logger. Useful for tests and integration.
func (s *OpenSpartanImportService) SetLogger(log *slog.Logger) {
	if log != nil {
		s.log = log
	}
}

// acquireShared prend le magasin d'écriture partagé + sa fonction de libération (à appeler via defer).
// DryRun : aucune écriture, donc aucun magasin (nil, libération sans effet) ; les helpers ne le
// déréférencent pas sur ce chemin.
func (s *OpenSpartanImportService) acquireShared(ctx context.Context, dryRun bool) (port.OpenSpartanSharedStore, func(), error) {
	if dryRun {
		return nil, func() {}, nil
	}
	return s.shared.AcquireShared(ctx)
}

// Import is the entry point. expectedOwnerXUID comes from the caller's SSO
// session — the service refuses to import a database whose detected owner
// does not match.
func (s *OpenSpartanImportService) Import(
	ctx context.Context,
	expectedOwnerXUID string,
	dbPath string,
	opts port.OpenSpartanImportOptions,
) (port.OpenSpartanImportResult, error) {
	opts = withDefaults(opts)
	var result port.OpenSpartanImportResult

	s.log.Info("openspartan_import_started",
		"expected_xuid", expectedOwnerXUID, "db_path", dbPath, "dry_run", opts.DryRun)

	// Le fichier est validé AVANT toute bascule de la base partagée : un fichier qui n'est
	// pas une base OpenSpartan, ou d'un autre propriétaire, ne coûte aucune écriture.
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

	// Acquisition on-demand du writer shared, une fois le fichier validé et lu (DryRun : aucune
	// acquisition, pas d'écriture). Aucun handle RW persistant.
	store, releaseShared, err := s.acquireShared(ctx, opts.DryRun)
	if err != nil {
		return result, fmt.Errorf("openspartan import acquire shared: %w", err)
	}
	defer releaseShared()

	scope := newImportScope(expectedOwnerXUID, mapper.MapOptions{
		AliasResolver: func(xuid string) string { return aliases[xuid] },
		Source:        opts.Source,
	}, opts.DryRun)

	if err := s.importMatches(ctx, store, r, scope, opts, &result); err != nil {
		return result, err
	}
	if err := s.importHighlights(ctx, store, r, scope, &result); err != nil {
		return result, err
	}
	s.importOwnerAlias(ctx, store, r, scope, &result)
	s.stashFriends(ctx, r, expectedOwnerXUID, opts, &result)

	s.log.Info("openspartan_import_completed",
		"inserted_matches", result.InsertedMatches,
		"inserted_participants", result.InsertedParticipants,
		"inserted_medals", result.InsertedMedals,
		"inserted_csrs", result.InsertedCSRs,
		"inserted_highlights", result.InsertedHighlights,
		"inserted_aliases", result.InsertedAliases,
		"skipped_foreign_matches", result.SkippedForeignMatches,
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
	result *port.OpenSpartanImportResult,
) error {
	detected, conf, err := r.DetectOwner(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("detect owner: %w", err)
	}
	result.DetectedOwnerXUID = detected
	result.Confidence = conf

	if conf == openspartan.ConfidenceNone || conf == openspartan.ConfidenceLow {
		return fmt.Errorf("%w: %s", port.ErrOpenSpartanLowConfidence, conf)
	}
	if detected != expectedXUID {
		return fmt.Errorf("%w: detected=%s expected=%s", port.ErrOpenSpartanXUIDMismatch, detected, expectedXUID)
	}
	return nil
}

// importMatches iterates every match in the reader, applies the mapper,
// and writes registry/participants/medals via the sync.* helpers.
//
// aujourd'hui best-effort par-match (les erreurs sont accumulées dans result.Errors).
//
//nolint:unparam // err maintenu pour signature cohérente avec import* siblings ;
func (s *OpenSpartanImportService) importMatches(
	ctx context.Context,
	store port.OpenSpartanSharedStore,
	r *openspartan.Reader,
	scope *importScope,
	opts port.OpenSpartanImportOptions,
	result *port.OpenSpartanImportResult,
) error {
	parsed := 0
	for pm, err := range r.Matches(ctx) {
		if err != nil {
			result.Errors = append(result.Errors, port.OpenSpartanImportError{Stage: "parse_match", Err: err.Error()})
			continue
		}
		parsed++
		if opts.OnProgress != nil {
			opts.OnProgress(parsed, result.TotalMatches)
		}
		s.writeOneMatch(ctx, store, pm, scope, result)
	}
	return nil
}

// writeOneMatch maps + writes one match's contribution to the shared DB, if the
// match is in the import scope (valid id, importer among its participants —
// openspartan_import_scope.go). Errors per stage are recorded but never abort the
// whole import.
func (s *OpenSpartanImportService) writeOneMatch(
	ctx context.Context,
	store port.OpenSpartanSharedStore,
	pm *openspartan.ParsedMatch,
	scope *importScope,
	result *port.OpenSpartanImportResult,
) {
	if rejectMatchID(ctx, pm.MatchID, result) {
		return
	}
	mm, err := mapper.MapMatch(pm, scope.mapOpts)
	if err != nil {
		result.Errors = append(result.Errors, port.OpenSpartanImportError{MatchID: pm.MatchID, Stage: "map", Err: err.Error()})
		return
	}
	if !scope.admit(ctx, pm, &mm, result) {
		return
	}
	thirdPartyAliases := scope.thirdPartyAliases(mm.Participants)
	if scope.dryRun {
		result.InsertedAliases += len(thirdPartyAliases)
		result.InsertedMatches++
		result.InsertedParticipants += len(mm.Participants)
		result.InsertedMedals += len(mm.Medals)
		return
	}
	parts := toSyncParticipants(mm.Participants)
	syncReg := toSyncRegistry(mm.Registry, len(parts) > 0)
	// is_ranked : la règle unique du sync (D-10, sync.AnySkillShowsRankedMatch) appliquée au
	// RankRecap du payload skill. Posé AVANT l'insert registry — crucial pour que le recompute
	// LUSR EXCLUE bien ces matchs (LUSR = non classé) et pour les filtres.
	csrs, ranked := s.extractMatchCSRRows(pm, &syncReg, result)
	syncReg.IsRanked = ranked

	// Écriture per-match INSERT-only + atomique (ADR 0019/0026, anti-ART #23645) : registry + participants
	// + medals + match_csrs dans UNE transaction, jamais d'ON CONFLICT DO UPDATE per-helper. Idempotent :
	// si le match existe déjà, l'écriture est un no-op. Modèle identique au livesync H5.
	write := port.OpenSpartanMatchWrite{
		Registry:     syncReg,
		Participants: parts,
		Medals:       toSyncMedals(mm.Medals),
		CSRs:         toPortCSRs(csrs),
		Aliases:      thirdPartyAliases,
	}
	if err := store.PersistMatch(ctx, write); err != nil {
		result.Errors = append(result.Errors, port.OpenSpartanImportError{MatchID: pm.MatchID, Stage: "persist_match", Err: err.Error()})
		return
	}
	result.InsertedMatches++
	result.InsertedMatchIDs = append(result.InsertedMatchIDs, pm.MatchID)
	result.InsertedParticipants += len(mm.Participants)
	result.InsertedMedals += len(mm.Medals)
	result.InsertedCSRs += len(csrs)
	result.InsertedAliases += len(thirdPartyAliases)
}

// extractMatchCSRRows parse le payload skill OpenSpartan (table PlayerMatchStats,
// même forme que la réponse skill live) et en extrait les lignes CSR par-match
// pour shared.match_csrs, et dit si le match est classé (règle unique D-10 :
// InitialMeasurementMatches > 0). reg.IsRanked n'entre pas dans la décision (PlaylistName
// n'est pas résolu à l'import) ; un match social porte un RankRecap à zéro → 0 ligne.
// Le post-import reprojette ensuite ces lignes vers player.match_skill_rank (lu par
// l'UI). Best-effort : une erreur de parse est accumulée sans interrompre l'import.
func (s *OpenSpartanImportService) extractMatchCSRRows(
	pm *openspartan.ParsedMatch, reg *sync.MatchRegistryRow, result *port.OpenSpartanImportResult,
) ([]sync.SharedMatchCSRRow, bool) {
	if len(pm.RawPlayerStats) == 0 {
		return nil, false
	}
	skillMap, err := sync.ParseMatchSkillResponseJSON(pm.RawPlayerStats)
	if err != nil {
		result.Errors = append(result.Errors, port.OpenSpartanImportError{MatchID: pm.MatchID, Stage: "parse_csr", Err: err.Error()})
		return nil, false
	}
	return sync.ExtractAllSharedCSRRows(reg, skillMap), sync.AnySkillShowsRankedMatch(skillMap)
}

// importHighlights walks HighlightEvents and writes one event per row, for the
// matches retained by the import scope only (the importer's own matches).
//
// aujourd'hui best-effort par-highlight (les erreurs sont accumulées dans result.Errors).
//
//nolint:unparam // err maintenu pour signature cohérente avec import* siblings ;
func (s *OpenSpartanImportService) importHighlights(
	ctx context.Context,
	store port.OpenSpartanSharedStore,
	r *openspartan.Reader,
	scope *importScope,
	result *port.OpenSpartanImportResult,
) error {
	for hl, err := range r.Highlights(ctx) {
		if err != nil {
			result.Errors = append(result.Errors, port.OpenSpartanImportError{Stage: "parse_highlight", Err: err.Error()})
			continue
		}
		if !scope.matches[hl.MatchID] {
			continue // match hors périmètre (sans l'importeur, ou identifiant invalide)
		}
		row, err := mapper.MapHighlight(hl.MatchID, hl.RawJSON)
		if err != nil {
			result.Errors = append(result.Errors, port.OpenSpartanImportError{MatchID: hl.MatchID, Stage: "map_highlight", Err: err.Error()})
			continue
		}
		if scope.dryRun {
			result.InsertedHighlights++
			continue
		}
		event := toAnalysisEvent(row)
		n, err := store.InsertHighlightEvents(ctx, row.MatchID, []highlightevent.HighlightEvent{event})
		if err != nil {
			result.Errors = append(result.Errors, port.OpenSpartanImportError{MatchID: hl.MatchID, Stage: "insert_highlight", Err: err.Error()})
			continue
		}
		result.InsertedHighlights += n
	}
	return nil
}

// stashFriends serialises the OpenSpartan Friends table to a JSON file
// under <StashDir>/<ownerXUID>/stash/openspartan_friends.json. No DuckDB
// table is created — the future MULTIUSER_ACL sprint will consume this.
func (s *OpenSpartanImportService) stashFriends(
	ctx context.Context,
	r *openspartan.Reader,
	ownerXUID string,
	opts port.OpenSpartanImportOptions,
	result *port.OpenSpartanImportResult,
) {
	friends, err := r.LoadFriends(ctx)
	if err != nil {
		result.Errors = append(result.Errors, port.OpenSpartanImportError{Stage: "load_friends", Err: err.Error()})
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
		result.Errors = append(result.Errors, port.OpenSpartanImportError{Stage: "stash_friends", Err: err.Error()})
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

func withDefaults(opts port.OpenSpartanImportOptions) port.OpenSpartanImportOptions {
	if opts.Source == "" {
		opts.Source = "openspartan_import"
	}
	if opts.StashDir == "" {
		opts.StashDir = "./data/players"
	}
	return opts
}
