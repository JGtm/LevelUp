// Package service — sync_jobs_run.go : exécution en arrière-plan d'un job de sync d'un joueur, son
// échec et ses notifications.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/dblease"
)

// Codes d'échec d'un job de sync ; la contention de bail (réessayable) se distingue d'un échec réel.
const (
	syncJobCodeDBBusy = "db_busy"
	syncJobCodeFailed = "sync_failed"
)

// syncErrorCoalesceWindow : fenêtre de coalescence des notifications sync_error — une panne durable
// donne UNE notification dont le compte s'incrémente, pas une par tentative.
const syncErrorCoalesceWindow = 6 * time.Hour

// Clés des paramètres de notification.
const (
	notifParamCount = "count"
	notifParamCode  = "code"
)

// launchSyncJob exécute run en arrière-plan sous le contexte de vie du serveur, borné par
// syncJobTimeout, puis consigne le résultat dans le job : succès (résumé, notification match_synced,
// finalisation du post-sync) ou échec (failSyncJob). release libère la réservation du joueur ; le
// post-sync est capturé AVANT la sync.
func (s *SyncJobService) launchSyncJob(jobID, playerSlug string, release func(), run func(ctx context.Context) (domain.SyncResult, error)) {
	go func() {
		defer release()
		bgCtx, cancel := context.WithTimeout(s.serverCtx, syncJobTimeout)
		defer cancel()
		var after func(ctx context.Context)
		if s.postSync != nil {
			if fin := s.postSync.BeforeSync(bgCtx, playerSlug); fin != nil {
				after = fin
			}
		}
		result, err := run(bgCtx)
		if err != nil {
			s.failSyncJob(bgCtx, playerSlug, jobID, err)
			return
		}
		summary := fmt.Sprintf("inserted=%d skipped=%d medals=%d duration=%.1fs status=%s",
			result.MatchesInserted, result.MatchesSkipped, result.MedalsInserted,
			result.DurationSeconds, result.Status())
		s.jobs.SetStatus(jobID, domain.JobStatusSucceeded, &summary)
		s.emitMatchSynced(bgCtx, playerSlug, result.MatchesInserted)
		if after != nil {
			after(bgCtx)
		}
	}()
}

// syncFailure classe l'échec d'un job de sync en un code stable et un message fixe : la contention de
// bail (un autre sync du même joueur est en cours, réessayable) se distingue d'un échec réel. La cause
// n'entre jamais dans l'erreur du job.
func syncFailure(err error) *domain.JobErrorDetail {
	if errors.Is(err, dblease.ErrDBLocked) {
		return domain.NewJobError(syncJobCodeDBBusy, true)
	}
	return domain.NewJobError(syncJobCodeFailed, false)
}

// failSyncJob consigne l'échec d'un job de sync : la cause au journal, le code et le message fixes
// dans le job, puis la notification sync_error.
func (s *SyncJobService) failSyncJob(ctx context.Context, slug, jobID string, err error) {
	fail := syncFailure(err)
	slog.ErrorContext(ctx, "sync: job échoué", "job_id", jobID, "player", slug, notifParamCode, fail.Code, "err", err)
	s.jobs.Update(jobID, func(j *domain.AsyncJobStatus) { j.Error = fail })
	s.jobs.SetStatus(jobID, domain.JobStatusFailed, nil)
	s.emitSyncError(ctx, slug, jobID, fail)
}

// emitMatchSynced émet une notification agrégée match_synced si des matchs ont été insérés.
func (s *SyncJobService) emitMatchSynced(ctx context.Context, slug string, inserted int) {
	if s.notifierFor == nil || inserted <= 0 {
		return
	}
	em, err := s.notifierFor(ctx, slug)
	if err != nil || em == nil {
		slog.WarnContext(ctx, "notifications: emitter factory failed", "slug", slug, "err", err)
		return
	}
	if err := em.Emit(ctx, notifications.EmitInput{
		Category: notifications.CategoryMatchSynced,
		// Plomberie, pas un accomplissement : sévérité info, hors du badge de la cloche.
		Severity: notifications.SeverityInfo,
		TitleKey: "notif.match_synced.title",
		BodyKey:  "notif.match_synced.body",
		Params:   map[string]any{notifParamCount: inserted},
		Source:   "sync_handler",
	}); err != nil {
		slog.WarnContext(ctx, "notifications: match_synced emit", "err", err)
	}
}

// emitSyncError émet une notification sync_error sur échec d'une sync. Sans acteur, la coalescence
// porte sur la catégorie seule ; aucune route cible n'est émise : le front route cette catégorie vers
// les réglages, et le contexte détaché d'un job ne porte pas le titre réel.
func (s *SyncJobService) emitSyncError(ctx context.Context, slug, jobID string, fail *domain.JobErrorDetail) {
	if s.notifierFor == nil {
		return
	}
	em, err := s.notifierFor(ctx, slug)
	if err != nil || em == nil {
		return
	}
	if err := em.EmitCoalesced(ctx, notifications.EmitInput{
		Category: notifications.CategorySyncError,
		Severity: notifications.SeverityError,
		TitleKey: "notif.sync_error.title",
		BodyKey:  "notif.sync_error.body",
		Params:   map[string]any{notifParamCode: fail.Code, "job_id": jobID},
		Source:   "sync_handler",
	}, syncErrorCoalesceWindow); err != nil {
		slog.WarnContext(ctx, "notifications: sync_error emit", "err", err)
	}
}
