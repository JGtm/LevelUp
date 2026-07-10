// Package handlers — openspartan_import.go: POST /api/v1/import/openspartan.
//
// Accepts a multipart upload of an OpenSpartan SQLite database, persists it
// to a temp file, creates a job entry via the existing jobs.Store, and runs
// the import in a background goroutine that updates job progress as matches
// are parsed. The HTTP response returns immediately with the job_id;
// callers poll status via the existing GET /jobs/{job_id} endpoint.
//
// Security:
//   - Owner XUID is taken from the SSO session (sess.LinkedHaloIdentity.XUID),
//     never from the request payload. The service refuses if the detected
//     owner of the uploaded database does not match.
//   - Upload size capped at 1 GB.
//   - Demo mode short-circuits with 503.
//   - Temp file is deleted after the goroutine finishes, success or failure.
package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/openspartan"
	"levelup/go-api/internal/platform/jobs"
	"levelup/go-api/internal/service"
	"levelup/go-api/internal/util/pointers"
)

const (
	openSpartanMaxUpload     = int64(1 << 30) // 1 GiB
	openSpartanFormFieldName = "db"
)

// EventsConvergenceTrigger lance un backfill events immédiat pour un joueur après
// l'import (récupère les highlight_events depuis les films, réutilise le pool
// d'auth). Optionnel : nil → la convergence se fera au prochain cycle du scheduler
// (elle est reprise automatiquement). *scheduler.AutoSyncScheduler le satisfait.
type EventsConvergenceTrigger interface {
	TriggerEventsConvergence(ctx context.Context, gamertag, xuid string)
}

// OpenSpartanImportHandler wires the import service to the HTTP layer.
type OpenSpartanImportHandler struct {
	importSvc     *service.OpenSpartanImportService
	postImportSvc *service.OpenSpartanPostImportService
	convergence   EventsConvergenceTrigger
	jobStore      *jobs.Store
	tempDir       string
	stashDir      string
	demoMode      bool
}

// OpenSpartanImportConfig collects the dependencies needed by the handler.
type OpenSpartanImportConfig struct {
	ImportService *service.OpenSpartanImportService
	// PostImportService recomputes sessions/perf_score/citations after the
	// raw import. Optional — when nil, the recompute stage is skipped and
	// callers can run it out-of-band (e.g. via the sync engine).
	PostImportService *service.OpenSpartanPostImportService
	// Convergence (optionnel) déclenche un backfill events immédiat après l'import.
	// Nil → repris au prochain cycle scheduler.
	Convergence EventsConvergenceTrigger
	JobStore    *jobs.Store
	// TempDir is where the uploaded `.db` is staged before opening.
	// Defaults to os.TempDir() when empty.
	TempDir string
	// StashDir is the parent directory under which the Friends JSON stash
	// is written by the service. Defaults to "./data/players" when empty.
	StashDir string
	// DemoMode short-circuits the endpoint with 503 when true.
	DemoMode bool
}

// NewOpenSpartanImportHandler constructs an OpenSpartanImportHandler.
func NewOpenSpartanImportHandler(cfg OpenSpartanImportConfig) *OpenSpartanImportHandler {
	tempDir := cfg.TempDir
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	stashDir := cfg.StashDir
	if stashDir == "" {
		stashDir = "./data/players"
	}
	return &OpenSpartanImportHandler{
		importSvc:     cfg.ImportService,
		postImportSvc: cfg.PostImportService,
		convergence:   cfg.Convergence,
		jobStore:      cfg.JobStore,
		tempDir:       tempDir,
		stashDir:      stashDir,
		demoMode:      cfg.DemoMode,
	}
}

// StartImport handles POST /api/v1/import/openspartan.
//
// Expected request:
//   - method:       POST
//   - content-type: multipart/form-data
//   - field "db":   the OpenSpartan SQLite file (max 1 GiB)
//
// Responses:
//   - 202 Accepted + {job_id, status} on success queue
//   - 401 if no SSO session XUID
//   - 413 if upload exceeds the size limit
//   - 400 if multipart parsing or the `db` field is malformed
//   - 503 in demo mode
func (h *OpenSpartanImportHandler) StartImport(w http.ResponseWriter, r *http.Request) {
	if h.demoMode {
		writeError(r.Context(), w, http.StatusServiceUnavailable, "demo_mode",
			"import OpenSpartan désactivé en mode démo")
		return
	}
	if h.importSvc == nil || h.jobStore == nil {
		writeError(r.Context(), w, http.StatusServiceUnavailable, "import_not_configured",
			"service d'import non configuré côté serveur")
		return
	}

	sess := middleware.GetSession(r.Context())
	if sess == nil || sess.LinkedHaloIdentity == nil || sess.LinkedHaloIdentity.XUID == "" {
		writeError(r.Context(), w, http.StatusUnauthorized, "halo_auth_required",
			"connexion Xbox/Halo requise pour l'import OpenSpartan")
		return
	}
	expectedXUID := sess.LinkedHaloIdentity.XUID
	gamertag := sess.LinkedHaloIdentity.Gamertag

	r.Body = http.MaxBytesReader(w, r.Body, openSpartanMaxUpload)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(r.Context(), w, http.StatusRequestEntityTooLarge, "upload_too_large",
			fmt.Sprintf("fichier trop volumineux (max %d Mo)", openSpartanMaxUpload>>20))
		return
	}

	tmpPath, err := h.saveUploadedDB(r)
	if err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "upload_failed", err.Error())
		return
	}

	jobStatus := h.jobStore.Create(domain.JobTypeOpenSpartanImport, expectedXUID)
	go h.runImport(jobStatus.JobID, expectedXUID, gamertag, tmpPath)

	writeJSON(w, http.StatusAccepted, map[string]any{
		"job_id":      jobStatus.JobID,
		jsonKeyStatus: jobStatus.Status,
	})
}

// saveUploadedDB extracts the "db" multipart field into a uniquely-named
// file under the handler's temp directory and returns the absolute path.
func (h *OpenSpartanImportHandler) saveUploadedDB(r *http.Request) (string, error) {
	file, header, err := r.FormFile(openSpartanFormFieldName)
	if err != nil {
		return "", fmt.Errorf("champ multipart %q manquant: %w", openSpartanFormFieldName, err)
	}
	defer file.Close()
	if header.Size <= 0 {
		return "", errors.New("fichier uploadé vide")
	}
	if err := os.MkdirAll(h.tempDir, 0o755); err != nil {
		return "", fmt.Errorf("création du répertoire tmp: %w", err)
	}
	name := fmt.Sprintf("openspartan_%s.db", uuid.NewString())
	tmpPath := filepath.Join(h.tempDir, name)
	f, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("création fichier tmp: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, file); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("copie vers tmp: %w", err)
	}
	return tmpPath, nil
}

// runImport executes the import in a background goroutine and updates the
// job entry as it makes progress. The temp file is always deleted on exit.
// On success, the post-import recompute (sessions / perf_score / citations)
// is run inline before marking the job succeeded — its counts are merged
// into the final job result.
func (h *OpenSpartanImportHandler) runImport(jobID, expectedXUID, gamertag, tmpPath string) {
	defer func() { _ = os.Remove(tmpPath) }()

	ctx := context.Background()
	h.jobStore.SetStatus(jobID, domain.JobStatusRunning, pointers.Ptr("opening_database"))

	opts := service.ImportOptions{
		Source:     "openspartan_import",
		StashDir:   h.stashDir,
		OnProgress: h.makeProgressCallback(jobID),
	}
	result, err := h.importSvc.Import(ctx, expectedXUID, tmpPath, opts)
	if err != nil {
		h.recordFailure(jobID, err)
		return
	}

	post := h.runPostImport(ctx, jobID, expectedXUID, gamertag, result.InsertedMatchIDs)
	h.recordSuccess(jobID, result, post)

	// Backfill events immédiat (best-effort, tâche de fond) : récupère les
	// highlight_events depuis les films pour les matchs importés (combat details,
	// killer/victim, dominance). Non-bloquant pour l'onboarding (fetch hors-lease,
	// lots courts, cède au sync live). nil → repris au prochain cycle scheduler.
	if h.convergence != nil {
		go h.convergence.TriggerEventsConvergence(context.Background(), gamertag, expectedXUID)
	}
}

// runPostImport runs the recompute stages (sessions, perf_score, citations)
// for the player whose matches were just imported. Returns a non-nil
// PostImportResult even when the post-import service is not configured —
// the zero value signals "skipped" to the caller. Errors are logged but
// never bubble up: the import itself succeeded.
func (h *OpenSpartanImportHandler) runPostImport(
	ctx context.Context, jobID, xuid, gamertag string, matchIDs []string,
) service.PostImportResult {
	if h.postImportSvc == nil || gamertag == "" {
		return service.PostImportResult{}
	}
	h.jobStore.SetStatus(jobID, domain.JobStatusRunning, pointers.Ptr("recomputing_sessions_and_scores"))
	res, err := h.postImportSvc.Run(ctx, xuid, gamertag, matchIDs, service.PostImportOptions{})
	if err != nil {
		slog.Warn("openspartan_post_import_fatal", "job_id", jobID, "err", err)
	}
	return res
}

func (h *OpenSpartanImportHandler) makeProgressCallback(jobID string) func(parsed, total int) {
	return func(parsed, total int) {
		pct := 0
		if total > 0 {
			pct = parsed * 100 / total
		}
		h.jobStore.Update(jobID, func(s *domain.AsyncJobStatus) {
			s.ProgressPct = &pct
			s.MatchesDone = &parsed
			s.MatchesTotal = &total
		})
	}
}

func (h *OpenSpartanImportHandler) recordSuccess(
	jobID string, result service.ImportResult, post service.PostImportResult,
) {
	slog.Info("openspartan_import_succeeded",
		"job_id", jobID,
		"inserted_matches", result.InsertedMatches,
		"sessions_touched", post.SessionsTouched,
		"perf_scores_touched", post.PerfScoresTouched,
		"citations_backfilled", post.CitationsBackfilled,
		"import_errors", len(result.Errors),
		"post_import_errors", len(post.Errors),
	)
	now := time.Now().UTC()
	h.jobStore.Update(jobID, func(s *domain.AsyncJobStatus) {
		s.Status = domain.JobStatusSucceeded
		s.FinishedAt = &now
		s.Result = map[string]any{
			"detected_owner_xuid":   result.DetectedOwnerXUID,
			"confidence":            result.Confidence.String(),
			"total_matches":         result.TotalMatches,
			"inserted_matches":      result.InsertedMatches,
			"inserted_participants": result.InsertedParticipants,
			"inserted_medals":       result.InsertedMedals,
			"inserted_highlights":   result.InsertedHighlights,
			"inserted_aliases":      result.InsertedAliases,
			"stashed_friends":       result.StashedFriends,
			"errors_count":          len(result.Errors),
			"post_import": map[string]any{
				"sessions_touched":     post.SessionsTouched,
				"perf_scores_touched":  post.PerfScoresTouched,
				"citations_backfilled": post.CitationsBackfilled,
				"errors_count":         len(post.Errors),
			},
		}
	})
}

func (h *OpenSpartanImportHandler) recordFailure(jobID string, err error) {
	slog.Error("openspartan_import_failed", "job_id", jobID, "err", err)
	now := time.Now().UTC()
	h.jobStore.Update(jobID, func(s *domain.AsyncJobStatus) {
		s.Status = domain.JobStatusFailed
		s.FinishedAt = &now
		s.Error = &domain.JobErrorDetail{
			Code:      classifyImportError(err),
			Message:   err.Error(),
			Retryable: false,
		}
	})
}

func classifyImportError(err error) string {
	switch {
	case errors.Is(err, service.ErrXUIDMismatch):
		return "xuid_mismatch"
	case errors.Is(err, service.ErrLowConfidence):
		return "owner_low_confidence"
	case errors.Is(err, openspartan.ErrNotOpenSpartanDB):
		return "not_openspartan_db"
	default:
		return "import_error"
	}
}
