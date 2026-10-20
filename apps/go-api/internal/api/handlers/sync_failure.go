package handlers

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
)

// syncCodeFailed : code d'un échec réel de sync ; la contention de bail porte codeDBBusy.
const syncCodeFailed = "sync_failed"

// syncFailure classe l'échec d'un job de sync en un code stable et un message fixe :
// la contention de bail (un autre sync du même joueur est en cours, réessayable) se
// distingue d'un échec réel. La cause n'entre jamais dans l'erreur du job.
func syncFailure(err error) *domain.JobErrorDetail {
	if errors.Is(err, dblease.ErrDBLocked) {
		return domain.NewJobError(codeDBBusy, true)
	}
	return domain.NewJobError(syncCodeFailed, false)
}

// failSyncJob consigne l'échec d'un job de sync : la cause au journal, le code et le
// message fixes dans le job, puis la notification sync_error.
func (h *SyncHandler) failSyncJob(ctx context.Context, slug, jobID string, err error) {
	fail := syncFailure(err)
	slog.ErrorContext(ctx, "sync: job échoué", "job_id", jobID, "player", slug, jsonKeyCode, fail.Code, "err", err)
	h.jobStore.Update(jobID, func(j *domain.AsyncJobStatus) { j.Error = fail })
	h.jobStore.SetStatus(jobID, domain.JobStatusFailed, nil)
	h.emitSyncError(ctx, slug, jobID, fail)
}
