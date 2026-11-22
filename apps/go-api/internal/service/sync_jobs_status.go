// Package service — sync_jobs_status.go : l'état de la synchronisation d'un joueur (port.SyncStatusService).
package service

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// PlayerSyncStatus rend où en est la synchronisation de playerSlug : en vol par n'importe quelle source
// (porte de déduplication), job actif ou dernier job conservé, délai restant du sync manuel. Lecture
// seule : ni le délai ni la porte ne sont modifiés.
func (s *SyncJobService) PlayerSyncStatus(ctx context.Context, playerSlug string) (*domain.PlayerSyncStatus, error) {
	prof, found, err := s.findPlayerProfile(playerSlug)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, &port.SyncJobError{Kind: port.SyncErrPlayerNotFound, PlayerSlug: playerSlug}
	}
	titleSlug := ctxkeys.TitleSlug(ctx)
	job := s.jobs.LatestSyncJob(playerSlug, titleSlug)
	status := &domain.PlayerSyncStatus{
		PlayerSlug:            playerSlug,
		Job:                   job,
		CooldownRetryAfterSec: remainingSeconds(s.manualSyncCooldownLeft(manualDeltaKey(playerSlug))),
	}
	status.Syncing = (job != nil && !job.IsTerminal()) || s.syncGate.IsInFlightT(titleSlug, prof.Gamertag)
	status.HistoryComplete = s.historyComplete(ctx, playerSlug, titleSlug)
	return status, nil
}

// historyComplete lit l'état de l'historique du joueur pour le titre. Une lecture impossible (lecteur non
// branché, base joueur absente ou en échec) se sert comme « non complet », après journalisation : le statut
// de la synchronisation reste servi.
func (s *SyncJobService) historyComplete(ctx context.Context, playerSlug, titleSlug string) bool {
	if s.history == nil {
		return false
	}
	complete, err := s.history.HistoryComplete(ctx, playerSlug, titleSlug)
	if err != nil {
		slog.WarnContext(ctx, "sync status: état de l'historique illisible", "player", playerSlug, "titleSlug", titleSlug, "err", err)
		return false
	}
	return complete
}

// remainingSeconds arrondit un délai restant à la seconde supérieure ; 0 pour un délai nul ou négatif.
func remainingSeconds(d time.Duration) int {
	if d <= 0 {
		return 0
	}
	return int(d.Seconds()) + 1
}

// createSyncJob crée un job de sync du joueur POUR un titre : le titre est écrit dans les métadonnées du job,
// car l'état du sync d'un joueur se lit par titre (PlayerSyncStatus) et un job d'un titre ne doit jamais
// s'afficher sur l'autre. L'instantané rendu porte le titre.
func (s *SyncJobService) createSyncJob(playerSlug, titleSlug string) *domain.AsyncJobStatus {
	job := s.jobs.Create(domain.JobTypeInitialSync, playerSlug)
	s.jobs.Update(job.JobID, func(j *domain.AsyncJobStatus) { j.Metadata.TitleSlug = titleSlug })
	job.Metadata.TitleSlug = titleSlug
	return job
}
