// Package ops — build_queue_priority.go : la priorité d'un job déjà en file, relevée quand le
// même match est redemandé plus haut (domain.BuildPriority*).
package ops

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
)

// raisePriorityLocked traite la remise en file d'un job déjà actif : un job EN FILE demandé à
// une priorité plus haute passe à cette priorité (un match d'abord vu par le rattrapage, puis
// demandé frais) ; sinon rien ne change. Jamais d'abaissement, jamais de retouche d'un job en
// cours. La date de mise en file d'origine est conservée — elle ordonne la file à priorité
// égale — et le travail résolu est celui de la demande, aux URL pré-signées plus fraîches.
func (s *MonitoringStore) raisePriorityLocked(
	ctx context.Context, w *dblease.LeasedWriter, existing scannedJob, priority int, payload string,
) (domain.BuildQueueJob, error) {
	if existing.Status != domain.JobStatusQueued || priority <= existing.Priority {
		return existing.BuildQueueJob, nil
	}
	if payload == "" {
		payload = existing.rawPayload // une demande sans travail résolu garde celui du job
	}
	if err := insertJobEvent(ctx, w, jobEvent{
		jobID: existing.JobID, jobType: existing.JobType, titleSlug: existing.TitleSlug, matchID: existing.MatchID,
		status: domain.JobStatusQueued, priority: priority, attempt: existing.Attempt,
		payload: payload, enqueuedAt: parseRFC3339(existing.EnqueuedAt),
	}); err != nil {
		return domain.BuildQueueJob{}, err
	}
	slog.InfoContext(ctx, "build queue: priorité du job relevée",
		"module", "monitoring", "job_id", existing.JobID, "match_id", existing.MatchID,
		"de", existing.Priority, "a", priority)
	existing.Priority = priority
	return existing.BuildQueueJob, nil
}
