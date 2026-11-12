package handlers

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
)

// internalCause : texte d'une erreur interne (chemin, requête) qui ne doit jamais atteindre un job.
const internalCause = `open C:\data\titles\halo_infinite\warehouse\shared_matches_v2.duckdb: IO Error: SELECT * FROM match_stats`

func failedJob(t *testing.T, store *jobs.Store, jobID string) *domain.AsyncJobStatus {
	t.Helper()
	job := store.Get(jobID)
	if job == nil || job.Status != domain.JobStatusFailed || job.Error == nil {
		t.Fatalf("job attendu en échec avec une erreur, obtenu %+v", job)
	}
	return job
}

func assertJobErrorFixed(t *testing.T, job *domain.AsyncJobStatus, wantCode string) {
	t.Helper()
	if job.Error.Code != wantCode {
		t.Errorf("code = %q, attendu %q", job.Error.Code, wantCode)
	}
	if want := domain.ErrorMessageForCode(wantCode); job.Error.Message != want {
		t.Errorf("message = %q, attendu le message fixe du code %q", job.Error.Message, want)
	}
	for _, leaked := range []string{"duckdb", "SELECT", `C:\`, "IO Error"} {
		if strings.Contains(job.Error.Message, leaked) {
			t.Errorf("le message du job laisse fuir %q : %q", leaked, job.Error.Message)
		}
	}
}

// L9-4 : un job en échec porte un code et un message FIXE ; la cause part au journal.
func TestAdminJobs_FailureMessageIsFixed(t *testing.T) {
	ctx := context.Background()
	boom := errors.New(internalCause)

	t.Run("convergence", func(t *testing.T) {
		store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
		h := NewAdminConvergenceActionHandler(func(context.Context, string, string) (map[string]any, error) {
			return nil, boom
		}, store, ctx, nil)
		job := store.Create(domain.JobTypePlayerConvergence, "p")
		h.runConvergence(ctx, job.JobID, "halo_infinite", "p")
		assertJobErrorFixed(t, failedJob(t, store, job.JobID), "convergence_failed")
	})
	t.Run("initial sync", func(t *testing.T) {
		store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
		h := NewAdminInitialSyncActionHandler(func(context.Context, string, string, int) (map[string]any, error) {
			return nil, boom
		}, store, ctx, nil)
		job := store.Create(domain.JobTypeAdminInitialSync, "p")
		h.runInitialSync(ctx, job.JobID, "halo_infinite", "p", 10)
		assertJobErrorFixed(t, failedJob(t, store, job.JobID), "initial_sync_failed")
	})
	t.Run("catalog drain", func(t *testing.T) {
		store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
		h := NewAdminCatalogDrainHandler(func(context.Context, string) (domain.CatalogUGCDrainResult, error) {
			return domain.CatalogUGCDrainResult{}, boom
		}, store, ctx)
		job := store.Create(domain.JobTypeCatalogUGCDrain, "")
		h.runDrain(ctx, job.JobID, "halo_infinite")
		assertJobErrorFixed(t, failedJob(t, store, job.JobID), "catalog_drain_failed")
	})
	t.Run("replay build", func(t *testing.T) {
		store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
		h := NewAdminReplayBuildActionHandler(func(context.Context, string, string) (map[string]any, error) {
			return nil, boom
		}, store, ctx)
		job := store.Create(domain.JobTypeReplayBuild, "")
		h.runBuild(ctx, job.JobID, "halo_infinite", "00000000-0000-0000-0000-000000000001")
		assertJobErrorFixed(t, failedJob(t, store, job.JobID), "replay_build_failed")
	})
	t.Run("import OpenSpartan", func(t *testing.T) {
		store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
		h := &OpenSpartanImportHandler{jobStore: store}
		job := store.Create(domain.JobTypeOther, "p")
		h.recordFailure(job.JobID, boom)
		assertJobErrorFixed(t, failedJob(t, store, job.JobID), "import_error")
	})
}
