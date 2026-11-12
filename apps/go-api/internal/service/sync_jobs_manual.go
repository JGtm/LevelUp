// Package service — sync_jobs_manual.go : syncs delta lancés à la demande (un joueur, ou tous) et leur
// cooldown anti-spam. Le sync manuel est un déclencheur à la demande de l'auto-sync : il en construit
// le même moteur (jetons du pool partagé), l'authentification ne dépend plus de la session.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/port"
)

// StartDeltaSync lance la sync delta d'un joueur. Contrairement à la sync initiale, elle n'est pas
// soumise à can_start_initial_sync.
func (s *SyncJobService) StartDeltaSync(ctx context.Context, caller port.SyncCaller, playerSlug string) (*domain.AsyncJobStatus, error) {
	if err := s.guardManualDeltaSync(ctx, caller, "delta:"+playerSlug); err != nil {
		return nil, err
	}
	if s.jobs.FindActiveInitialSync(playerSlug) != nil {
		return nil, &port.SyncJobError{Kind: port.SyncErrDeltaAlreadyActive}
	}

	_, evID := logging.WithEvent(ctx, "http.sync.delta:"+playerSlug)
	slog.InfoContext(ctx, "sync_jobs: StartDeltaSync démarré", "player_slug", playerSlug, "event", evID)

	prof, found, err := s.findPlayerProfile(playerSlug)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, &port.SyncJobError{Kind: port.SyncErrPlayerNotFound, PlayerSlug: playerSlug}
	}

	release, claimed := s.syncGate.TryClaimT(ctxkeys.TitleSlug(ctx), prof.Gamertag)
	if !claimed {
		return nil, &port.SyncJobError{Kind: port.SyncErrAlreadyRunning}
	}

	job := s.jobs.Create(domain.JobTypeInitialSync, playerSlug)
	snapshot := *job
	s.launchSyncJob(job.JobID, playerSlug, release, func(runCtx context.Context) (domain.SyncResult, error) {
		return s.pooledRunnerFor(runCtx, prof.Gamertag, prof.XUID).RunDelta(runCtx, domain.DefaultSyncOptions())
	})
	return &snapshot, nil
}

// StartSyncAll lance la sync delta de tous les joueurs configurés.
func (s *SyncJobService) StartSyncAll(ctx context.Context, caller port.SyncCaller) (*domain.AsyncJobStatus, error) {
	if err := s.guardManualDeltaSync(ctx, caller, "all"); err != nil {
		return nil, err
	}
	players, err := s.profiles.LoadPlayers()
	if err != nil {
		return nil, &port.SyncJobError{Kind: port.SyncErrProfilesLoad, Err: err}
	}
	if len(players) == 0 {
		return nil, &port.SyncJobError{Kind: port.SyncErrNoPlayers}
	}

	_, evID := logging.WithEvent(ctx, "http.sync.all")
	slog.InfoContext(ctx, "sync_jobs: StartSyncAll démarré", "player_count", len(players), "event", evID)

	job := s.jobs.Create(domain.JobTypeDeltaSyncAll, "all")
	snapshot := *job
	go s.syncAllPlayers(job.JobID, players)
	return &snapshot, nil
}

// syncAllPlayers synchronise les joueurs l'un après l'autre et consigne le résultat dans le job (échec
// dès qu'un joueur échoue). L'arrêt du serveur interrompt la boucle : la réservation étant posée puis
// relâchée par joueur, l'attente d'arrêt du coordinateur pourrait sinon rendre la main pendant qu'on
// démarre le joueur suivant.
func (s *SyncJobService) syncAllPlayers(jobID string, players []domain.PlayerSummary) {
	total := len(players)
	var succeeded, failed, coalesced int
	for i, p := range players {
		if s.serverCtx.Err() != nil {
			break
		}
		step := fmt.Sprintf("%s (%d/%d)", p.Gamertag, i+1, total)
		s.jobs.Update(jobID, func(j *domain.AsyncJobStatus) {
			j.CurrentStep = &step
			pct := (i * 100) / total
			j.ProgressPct = &pct
		})
		switch s.syncOnePlayer(p) {
		case syncPlayerCoalesced:
			coalesced++
		case syncPlayerFailed:
			failed++
		default:
			succeeded++
		}
	}

	summary := fmt.Sprintf("players=%d succeeded=%d failed=%d coalesced=%d", total, succeeded, failed, coalesced)
	status := domain.JobStatusSucceeded
	if failed > 0 {
		status = domain.JobStatusFailed
	}
	s.jobs.SetStatus(jobID, status, &summary)
}

// Issue de la sync d'un joueur dans une sync de tous.
type syncPlayerOutcome int

const (
	syncPlayerDone      syncPlayerOutcome = iota
	syncPlayerFailed                      // RunDelta a échoué
	syncPlayerCoalesced                   // déjà en vol via une autre source : sauté, sans compter un échec
)

// syncOnePlayer synchronise un joueur sous sa réservation, bornée par syncJobTimeout (qui borne
// l'attente du bail). La libération est différée : garantie même sur panique de RunDelta.
func (s *SyncJobService) syncOnePlayer(p domain.PlayerSummary) syncPlayerOutcome {
	release, claimed := s.syncGate.TryClaimT(p.TitleSlug, p.Gamertag)
	if !claimed {
		slog.DebugContext(s.serverCtx, "sync_jobs: joueur coalescé dans sync/all (déjà en vol via une autre source)",
			"gamertag", p.Gamertag)
		return syncPlayerCoalesced
	}
	defer release()
	runner := s.pooledRunnerFor(s.serverCtx, p.Gamertag, p.XUID)
	pCtx, cancel := context.WithTimeout(s.serverCtx, syncJobTimeout)
	defer cancel()
	if _, err := runner.RunDelta(pCtx, domain.DefaultSyncOptions()); err != nil {
		return syncPlayerFailed
	}
	return syncPlayerDone
}

// guardManualDeltaSync applique les pré-conditions du sync manuel delta : une session (port.SyncErrAuthRequired
// sinon ; l'authentification Halo elle-même vient du pool) puis le cooldown par clé (port.SyncErrCooldown
// avec le délai restant).
func (s *SyncJobService) guardManualDeltaSync(ctx context.Context, caller port.SyncCaller, key string) error {
	if !caller.Authenticated {
		return &port.SyncJobError{Kind: port.SyncErrAuthRequired}
	}
	if retry, ok := s.tryManualSyncCooldown(key); !ok {
		slog.InfoContext(ctx, "sync_jobs: sync manuel throttlé par cooldown",
			"key", key, "retry_after_s", int(retry.Seconds())+1)
		return &port.SyncJobError{Kind: port.SyncErrCooldown, RetryAfter: retry}
	}
	return nil
}

// tryManualSyncCooldown vérifie ET enregistre, atomiquement, le cooldown du sync manuel pour key : un
// sync de la même clé déclenché il y a moins du cooldown rend (délai restant, false) ; sinon l'instant
// courant est mémorisé et (0, true) rendu — toujours, si le cooldown est désactivé (<= 0). La fenêtre
// couvre la TENTATIVE : le marquage est posé même si la suite échoue (joueur introuvable, réservation
// refusée).
func (s *SyncJobService) tryManualSyncCooldown(key string) (time.Duration, bool) {
	if s.cooldown <= 0 {
		return 0, true
	}
	s.cooldownMu.Lock()
	defer s.cooldownMu.Unlock()
	if last, ok := s.lastManualAt[key]; ok {
		if elapsed := time.Since(last); elapsed < s.cooldown {
			return s.cooldown - elapsed, false
		}
	}
	s.lastManualAt[key] = time.Now()
	return 0, true
}
