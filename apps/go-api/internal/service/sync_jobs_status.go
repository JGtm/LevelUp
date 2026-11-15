// Package service — sync_jobs_status.go : l'état de la synchronisation d'un joueur (port.SyncStatusService).
package service

import (
	"context"
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
	job := s.jobs.LatestSyncJob(playerSlug)
	status := &domain.PlayerSyncStatus{
		PlayerSlug:            playerSlug,
		Job:                   job,
		CooldownRetryAfterSec: remainingSeconds(s.manualSyncCooldownLeft(manualDeltaKey(playerSlug))),
	}
	status.Syncing = (job != nil && !job.IsTerminal()) || s.syncGate.IsInFlightT(ctxkeys.TitleSlug(ctx), prof.Gamertag)
	return status, nil
}

// remainingSeconds arrondit un délai restant à la seconde supérieure ; 0 pour un délai nul ou négatif.
func remainingSeconds(d time.Duration) int {
	if d <= 0 {
		return 0
	}
	return int(d.Seconds()) + 1
}
