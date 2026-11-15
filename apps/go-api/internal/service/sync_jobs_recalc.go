// Package service — sync_jobs_recalc.go : recalcul des sessions de tous les joueurs suivis.
package service

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
	go_sync "levelup/go-api/internal/sync"
)

// codeEchecRecalculSessions : code du job quand au moins un joueur n'a pas pu être recalculé.
const codeEchecRecalculSessions = "sessions_recalc_failed"

// StartSessionsRecalc lance le recalcul des sessions de tous les joueurs suivis, tous titres
// confondus. Le job finit en échec (relançable) dès qu'un joueur n'a pas pu être recalculé, comme la
// synchro de tous les joueurs : un succès annoncé alors que des joueurs ont échoué masquerait une base
// devenue inscriptible par intermittence seulement.
func (s *SyncJobService) StartSessionsRecalc(ctx context.Context) (*domain.AsyncJobStatus, error) {
	players, err := s.profiles.LoadPlayers()
	if err != nil {
		return nil, &port.SyncJobError{Kind: port.SyncErrPlayersLoad, Err: err}
	}
	job := s.jobs.Create(domain.JobTypeSessionsRecalc, "")
	// Instantané avant le lancement : le job du registre est modifié en place par la goroutine.
	snapshot := *job
	go s.recalculerSessions(job.JobID, players)
	return &snapshot, nil
}

// recalculerSessions recalcule les sessions de chaque profil puis pose l'état final du job.
func (s *SyncJobService) recalculerSessions(jobID string, players []domain.PlayerSummary) {
	step := "Recalcul des sessions en cours"
	s.jobs.SetStatus(jobID, domain.JobStatusRunning, &step)

	total, joueurs, echecs := 0, 0, 0
	for _, p := range players {
		// Un profil auth_only ne porte qu'un jeton : pas de base joueur à recalculer, et l'ouvrir la
		// créerait.
		if p.XUID == "" || p.AuthOnly {
			continue
		}
		joueurs++
		n, err := s.recalculerSessionsJoueur(p)
		if err != nil {
			echecs++
			slog.WarnContext(context.Background(), "sessions recalculate: erreur joueur",
				"gamertag", p.Gamertag, "title", p.TitleSlug, "err", err)
			continue
		}
		total += n
	}
	s.terminerRecalculSessions(jobID, total, joueurs, echecs)
}

// recalculerSessionsJoueur recalcule les sessions d'un profil, sur les bases ET le provider shared DE
// SON TITRE : un profil Halo 5 se recalcule sur les bases de Halo 5, jamais sur celles du titre par
// défaut. Règle de découpage du titre (settings.SessionOptions : réglage global + overlay du titre), la
// même que la synchro ; amis DU joueur recalculé (data/global/player_friends.json : les sessions
// d'escouade de chacun se calculent sur SA liste ; magasin absent ou lecture en échec -> aucun ami,
// journalisé avant dégradation).
func (s *SyncJobService) recalculerSessionsJoueur(p domain.PlayerSummary) (int, error) {
	ctx := ctxkeys.WithTitleSlug(context.Background(), p.TitleSlug)
	opts := settings_platform.SessionOptions(ctx, s.settingsStore, s.cfg.TitleSettingsPath(p.TitleSlug))
	var friendGamertags []string
	if s.friendStore != nil {
		fs, ferr := s.friendStore.Get(p.XUID)
		if ferr != nil {
			slog.WarnContext(ctx, "sessions recalculate: lecture des amis échouée",
				"gamertag", p.Gamertag, "err", ferr)
		}
		friendGamertags = fs
	}
	playerDBPath, sharedDBPath := sessionsRecalcPaths(s.cfg, p)
	return go_sync.RecalculatePlayerSessions(
		ctx, s.cfg.SharedProviderForTitle(p.TitleSlug),
		playerDBPath, sharedDBPath, p.XUID, opts, friendGamertags,
	)
}

// sessionsRecalcPaths rend la base joueur et la base partagée du TITRE d'un profil, pour le recalcul
// des sessions. Un titre vide (profil sans titre) vaut le titre par défaut, comme partout dans
// config.PlayerDBPath / config.SharedDBPath.
func sessionsRecalcPaths(cfg *config.AppConfig, p domain.PlayerSummary) (playerDBPath, sharedDBPath string) {
	return config.PlayerDBPath(cfg, p.TitleSlug, p.Gamertag), config.SharedDBPath(cfg, p.TitleSlug)
}

// terminerRecalculSessions pose l'état final du job : succès si tous les joueurs ont été recalculés,
// échec relançable sinon. Le résultat compte joueurs, échecs et matchs mis à jour.
func (s *SyncJobService) terminerRecalculSessions(jobID string, total, joueurs, echecs int) {
	done := fmt.Sprintf("Sessions recalculées (%d matchs mis à jour)", total)
	status := domain.JobStatusSucceeded
	var jobErr *domain.JobErrorDetail
	if echecs > 0 {
		status = domain.JobStatusFailed
		jobErr = domain.NewJobError(codeEchecRecalculSessions, true)
	}
	pct := 100
	s.jobs.Update(jobID, func(j *domain.AsyncJobStatus) {
		j.ProgressPct = &pct
		j.Error = jobErr
		j.Result = map[string]any{"players": joueurs, "players_failed": echecs, "matches_updated": total}
	})
	s.jobs.SetStatus(jobID, status, &done)
}
