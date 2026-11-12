// Package service — sync_jobs_initial.go : sync initiale d'un joueur (onboarding).
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/port"
)

// Bornes de max_matches d'une sync initiale.
const (
	defaultInitialSyncMatches = 200
	maxInitialSyncMatches     = 2000
)

// CheckInitialSyncEnabled refuse la sync initiale quand l'instance l'interdit.
func (s *SyncJobService) CheckInitialSyncEnabled(ctx context.Context) error {
	appCfg, err := s.settingsStore.Load()
	if err != nil {
		return &port.SyncJobError{Kind: port.SyncErrSettingsLoad, Err: err}
	}
	if !appCfg.CanStartInitialSync {
		return &port.SyncJobError{Kind: port.SyncErrInitialDisabled}
	}
	return nil
}

// StartInitialSync lance la sync initiale d'un joueur. Le titre cible est celui de la requête, sinon
// celui du contexte. Le nombre de matchs, omis, vaut celui du profil (initial_max_matches) puis
// defaultInitialSyncMatches ; dans tous les cas il reste borné à maxInitialSyncMatches.
func (s *SyncJobService) StartInitialSync(ctx context.Context, caller port.SyncCaller, req domain.InitialSyncStartRequest) (*domain.AsyncJobStatus, error) {
	titleSlug := req.TitleSlug
	if titleSlug == "" {
		titleSlug = ctxkeys.TitleSlug(ctx)
	}
	bodyMaxOmitted := req.MaxMatches == 0

	_, evID := logging.WithEvent(ctx, "http.sync.initial:"+req.PlayerSlug)
	slog.InfoContext(ctx, "sync_jobs: StartInitialSync démarré",
		"player_slug", req.PlayerSlug, "title_slug", titleSlug, "max_matches", req.MaxMatches, "event", evID)

	// Un corps explicite hors bornes est rejeté d'emblée ; le défaut du profil s'applique après la
	// résolution du profil.
	if req.MaxMatches == 0 {
		req.MaxMatches = defaultInitialSyncMatches
	}
	if req.MaxMatches < 1 || req.MaxMatches > maxInitialSyncMatches {
		return nil, &port.SyncJobError{Kind: port.SyncErrInvalidMaxMatches}
	}
	if s.jobs.FindActiveInitialSync(req.PlayerSlug) != nil {
		return nil, &port.SyncJobError{Kind: port.SyncErrInitialAlreadyActive}
	}
	tokens, err := initialSyncTokens(ctx, caller)
	if err != nil {
		return nil, err
	}

	// Le couple (gamertag, xuid) et le nombre de matchs initiaux se lisent dans le profil POUR CE TITRE :
	// cela lève l'ambiguïté d'un gamertag suivi sous plusieurs titres.
	prof, found, err := s.findPlayerProfile(req.PlayerSlug, titleSlug)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, &port.SyncJobError{Kind: port.SyncErrPlayerNotFoundForTitle, PlayerSlug: req.PlayerSlug, TitleSlug: titleSlug}
	}
	if bodyMaxOmitted && prof.InitialMaxMatches > 0 {
		req.MaxMatches = min(prof.InitialMaxMatches, maxInitialSyncMatches)
	}

	// Réservation SYNCHRONE avant la création du job : un joueur déjà en vol (watcher ou auto-sync) est
	// refusé sans job. Posée avant le retour, elle est vue par l'attente d'arrêt du coordinateur.
	release, claimed := s.syncGate.TryClaimT(titleSlug, prof.Gamertag)
	if !claimed {
		return nil, &port.SyncJobError{Kind: port.SyncErrAlreadyRunning}
	}

	job := s.jobs.Create(domain.JobTypeInitialSync, req.PlayerSlug)
	// Instantané avant le lancement : le job du registre est modifié en place par la goroutine.
	snapshot := *job
	s.launchSyncJob(job.JobID, req.PlayerSlug, release, func(runCtx context.Context) (domain.SyncResult, error) {
		opts := domain.DefaultSyncOptions()
		opts.MaxMatches = req.MaxMatches
		// Un titre en suivi live seul (Halo 5) a son runner dédié ; sinon le moteur de Halo Infinite.
		runner, jobCtx := s.runnerFor(runCtx, titleSlug, prof.Gamertag, prof.XUID, tokens)
		return runner.RunDelta(jobCtx, opts)
	})
	return &snapshot, nil
}

// initialSyncTokens rend les jetons Halo de la session pour la sync initiale : port.SyncErrAuthRequired
// sans session, port.SyncErrHaloTokensMissing avec une session sans jetons Halo.
func initialSyncTokens(ctx context.Context, caller port.SyncCaller) (*domain.HaloTokens, error) {
	if !caller.Authenticated {
		return nil, &port.SyncJobError{Kind: port.SyncErrAuthRequired}
	}
	if caller.HaloTokens == nil {
		slog.InfoContext(ctx, "sync_jobs: sync initiale refusée, session sans jetons Halo")
		return nil, &port.SyncJobError{Kind: port.SyncErrHaloTokensMissing}
	}
	return caller.HaloTokens, nil
}
