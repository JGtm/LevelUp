package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	go_sync "levelup/go-api/internal/sync"
)

// codeEchecRecalculSessions : code du job quand au moins un joueur n'a pas pu être recalculé.
const codeEchecRecalculSessions = "sessions_recalc_failed"

// handlePostRecalculateSessions lance un recalcul des sessions pour tous les joueurs.
// POST /settings/sessions/recalculate — retourne un AsyncJobStatus (202). Le job finit en échec
// (relançable) dès qu'un joueur n'a pas pu être recalculé, comme la synchro de tous les joueurs :
// un succès annoncé alors que des joueurs ont échoué masque une base devenue inscriptible
// seulement par intermittence.
func (h *SettingsHandler) handlePostRecalculateSessions(ctx context.Context, _ *struct{}) (*asyncJobOutput, error) {
	players, err := h.cfg.LoadPlayers()
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "players_load_error",
			"Impossible de charger les joueurs configurés.")
	}

	job := h.jobStore.Create(domain.JobTypeSessionsRecalc, "")
	// Snapshot avant le go func() : la goroutine modifie in-place le job dans le store.
	jobSnapshot := *job

	go func() {
		step := "Recalcul des sessions en cours"
		h.jobStore.SetStatus(job.JobID, domain.JobStatusRunning, &step)

		total, joueurs, echecs := 0, 0, 0
		for _, p := range players {
			if p.XUID == "" {
				continue
			}
			joueurs++
			n, err := h.recalculerSessionsJoueur(p)
			if err != nil {
				echecs++
				slog.Warn("sessions recalculate: erreur joueur",
					"gamertag", p.Gamertag, "title", p.TitleSlug, "err", err)
				continue
			}
			total += n
		}
		h.terminerRecalculSessions(job.JobID, total, joueurs, echecs)
	}()

	return &asyncJobOutput{Body: &jobSnapshot}, nil
}

// recalculerSessionsJoueur recalcule les sessions d'un profil, sur les bases ET le provider shared
// DE SON TITRE : un profil Halo 5 se recalcule sur les bases de Halo 5, jamais sur celles du titre
// par défaut. Options per-titre (overlay PMT-4, le rythme de session dépend du titre ; overlay
// absent ⇒ valeurs globales) ; amis DU joueur recalculé (data/global/player_friends.json : les
// sessions d'escouade de chacun se calculent sur SA liste ; store absent ou lecture en échec →
// aucun ami, journalisé avant dégradation).
func (h *SettingsHandler) recalculerSessionsJoueur(p domain.PlayerSummary) (int, error) {
	opts := sessionComputeOptionsFor(h.settingsStore, h.cfg.TitleSettingsPath, p.TitleSlug)
	var friendGamertags []string
	if h.friendStore != nil {
		fs, ferr := h.friendStore.Get(p.XUID)
		if ferr != nil {
			slog.WarnContext(context.Background(), "sessions recalculate: lecture des amis échouée",
				"gamertag", p.Gamertag, "err", ferr)
		}
		friendGamertags = fs
	}
	playerDBPath, sharedDBPath := sessionsRecalcPaths(h.cfg, p)
	return go_sync.RecalculatePlayerSessions(
		ctxkeys.WithTitleSlug(context.Background(), p.TitleSlug), h.cfg.SharedProviderForTitle(p.TitleSlug),
		playerDBPath, sharedDBPath, p.XUID, opts, friendGamertags,
	)
}

// terminerRecalculSessions pose l'état final du job : succès si tous les joueurs ont été
// recalculés, échec relançable sinon. Le résultat compte joueurs, échecs et matchs mis à jour.
func (h *SettingsHandler) terminerRecalculSessions(jobID string, total, joueurs, echecs int) {
	done := fmt.Sprintf("Sessions recalculées (%d matchs mis à jour)", total)
	status := domain.JobStatusSucceeded
	var jobErr *domain.JobErrorDetail
	if echecs > 0 {
		status = domain.JobStatusFailed
		jobErr = domain.NewJobError(codeEchecRecalculSessions, true)
	}
	pct := 100
	h.jobStore.Update(jobID, func(j *domain.AsyncJobStatus) {
		j.ProgressPct = &pct
		j.Error = jobErr
		j.Result = map[string]any{"players": joueurs, "players_failed": echecs, "matches_updated": total}
	})
	h.jobStore.SetStatus(jobID, status, &done)
}
