// Package service — sync_jobs_backfill.go : lancement du rattrapage (backfill) des données manquantes
// d'un joueur. L'orchestration multi-phase vit dans BackfillOrchestrator ; ce fichier valide le joueur,
// câble le moteur et crée le job.
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/port"
	go_sync "levelup/go-api/internal/sync"
)

// StartBackfill lance le rattrapage d'un joueur. Un titre absent vaut le titre par défaut, résolu AVANT
// la recherche du joueur : le moteur ouvre les bases de ce titre, la recherche doit donc porter sur ses
// profils (un joueur suivi ailleurs seulement est refusé). Les jetons Halo de la session servent au
// rattrapage des armes.
func (s *SyncJobService) StartBackfill(ctx context.Context, caller port.SyncCaller, req domain.BackfillStartRequest) (*domain.AsyncJobStatus, error) {
	if req.Title == "" {
		req.Title = titlePkg.DefaultSlug
	}
	prof, found, err := s.findPlayerProfile(req.PlayerSlug, req.Title)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, &port.SyncJobError{Kind: port.SyncErrPlayerNotFound, PlayerSlug: req.PlayerSlug}
	}
	if s.jobs.FindActiveJob(domain.JobTypeBackfill, req.PlayerSlug) != nil {
		return nil, &port.SyncJobError{Kind: port.SyncErrBackfillAlreadyActive}
	}

	// Journal immédiat, sans muter le contexte de la requête (course de données avec le middleware chi).
	_, evID := logging.WithEvent(ctx, "http.backfill:"+req.PlayerSlug)
	slog.InfoContext(ctx, "sync_jobs: StartBackfill démarré",
		"player_slug", req.PlayerSlug, "gamertag", prof.Gamertag, "event", evID)

	tokens := caller.HaloTokens
	engine := s.newBackfillEngine(req.Title, prof, tokens)
	job := s.jobs.Create(domain.JobTypeBackfill, req.PlayerSlug)
	// Instantané avant le lancement : le job du registre est modifié en place par la goroutine.
	snapshot := *job
	orch := NewBackfillOrchestrator(engine, s.jobs, buildSyncScope(req), tokens, req.DryRun)
	go orch.Run(job.JobID)
	return &snapshot, nil
}

// newBackfillEngine construit le moteur DU TITRE demandé, sur le provider shared de CE titre (celui du
// boot est celui du titre par défaut : il ferait écrire un autre titre dans la mauvaise base). Le
// provider aligne le rattrapage sur l'auto-sync ; sans lui le moteur ouvrirait la base partagée en
// direct, en parallèle des syncs automatiques.
func (s *SyncJobService) newBackfillEngine(titleSlug string, prof domain.PlayerSummary, tokens *domain.HaloTokens) *go_sync.SyncEngine {
	engine := go_sync.NewSyncEngineForTitle(s.engineRoot(), titleSlug, prof.Gamertag, prof.XUID, tokens, nil).
		WithSessionOptionsFromConfig(s.cfg)
	if provider := s.cfg.SharedProviderForTitle(titleSlug); provider != nil {
		engine = engine.WithSharedProvider(provider)
	}
	return engine
}

// buildSyncScope construit le périmètre du moteur depuis la requête. Sans périmètre explicite (ni
// AllData), tout est activé ; les coefficients d'engagement ne s'activent jamais implicitement : ils
// sont recalculés en queue du rattrapage des scores d'engagement.
func buildSyncScope(req domain.BackfillStartRequest) *go_sync.SyncScope {
	scope := &go_sync.SyncScope{
		DetectionMode: "or",
		MaxMatches:    req.MaxMatches,
	}
	if req.AllData || !hasExplicitBackfillScope(req) {
		enableAllBackfillScope(scope)
	} else {
		applyRequestedBackfillScope(scope, req)
	}
	if req.ForceRescan {
		applyForcedBackfillScope(scope, req)
	}
	scope.Resolve()
	return scope
}

// hasExplicitBackfillScope : la requête désigne au moins un type de données.
func hasExplicitBackfillScope(req domain.BackfillStartRequest) bool {
	return req.Medals || req.Events || req.Skill ||
		req.PersonalScores || req.PerformanceScores ||
		req.Aliases || req.LUSR || req.CSR || req.EngagementScores ||
		req.EngagementCoefficients || req.ComebackBadges || req.Citations
}

// enableAllBackfillScope active tous les types de données.
func enableAllBackfillScope(scope *go_sync.SyncScope) {
	scope.AllData = true
	scope.Medals = true
	scope.Events = true
	scope.Skill = true
	scope.PersonalScores = true
	scope.PerformanceScores = true
	scope.Aliases = true
	scope.LUSR = true
	scope.CSR = true
	scope.EngagementScores = true
	scope.ComebackBadges = true
	scope.Citations = true
}

// applyRequestedBackfillScope active les seuls types demandés.
func applyRequestedBackfillScope(scope *go_sync.SyncScope, req domain.BackfillStartRequest) {
	scope.Medals = req.Medals
	scope.Events = req.Events
	scope.Skill = req.Skill
	scope.PersonalScores = req.PersonalScores
	scope.PerformanceScores = req.PerformanceScores
	scope.Aliases = req.Aliases
	scope.LUSR = req.LUSR
	scope.CSR = req.CSR
	scope.EngagementScores = req.EngagementScores
	scope.EngagementCoefficients = req.EngagementCoefficients
	scope.ComebackBadges = req.ComebackBadges
	scope.Citations = req.Citations
}

// applyForcedBackfillScope force le rescan des types demandés (tous avec AllData).
func applyForcedBackfillScope(scope *go_sync.SyncScope, req domain.BackfillStartRequest) {
	scope.ForceMedals = req.Medals || req.AllData
	scope.ForceEvents = req.Events || req.AllData
	scope.ForceSkill = req.Skill || req.AllData
	scope.ForcePersonalScores = req.PersonalScores || req.AllData
	scope.ForcePerformanceScores = req.PerformanceScores || req.AllData
	scope.ForceAliases = req.Aliases || req.AllData
	scope.ForceLUSR = req.LUSR || req.AllData
	scope.ForceCSR = req.CSR || req.AllData
	scope.ForceEngagementScores = req.EngagementScores || req.AllData
	scope.ForceEngagementCoefficients = req.EngagementCoefficients
	scope.ForceComebackBadges = req.ComebackBadges || req.AllData
	scope.ForceCitations = req.Citations || req.AllData
}
