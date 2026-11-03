// Package handlers — desktop_replay_regen.go : la régénération des rejeux en retard après une mise à jour de
// l'application de bureau (ADR 0045) : GET /api/v1/desktop/replay-regen rend le décompte et l'état de la file,
// POST .../start, .../pause, .../resume et .../rescan la pilotent.
//
// Les routes ne sont montées que sous la capacité ReplayRegen et exigent le cookie de la coquille (garde racine).
// Rien ne part sans le POST de l'utilisateur : le serveur ne lance jamais la file de lui-même.
package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/replayregen"
)

// Codes des refus de la file.
const (
	codeReplayRegenUnavailable = "replay_regen_unavailable"
	codeReplayRegenNotRunning  = "replay_regen_not_running"
)

// ReplayRegenService : ce que les routes demandent à la file (satisfait par *replayregen.Service).
type ReplayRegenService interface {
	Status() domain.ReplayRegenStatus
	Start() (domain.ReplayRegenStatus, error)
	Pause() (domain.ReplayRegenStatus, error)
	Resume() (domain.ReplayRegenStatus, error)
	Rescan() (domain.ReplayRegenStatus, error)
}

// DesktopReplayRegenHandler sert les routes de la régénération des rejeux.
type DesktopReplayRegenHandler struct {
	svc ReplayRegenService
}

// NewDesktopReplayRegenHandler crée le handler sur la file.
func NewDesktopReplayRegenHandler(svc ReplayRegenService) *DesktopReplayRegenHandler {
	return &DesktopReplayRegenHandler{svc: svc}
}

// Mount enregistre les cinq routes au point de montage de l'appelant.
func (h *DesktopReplayRegenHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/desktop/replay-regen", h.handleStatus,
		humacore.Op("getDesktopReplayRegen", "Replays behind the app version and state of the regeneration queue (desktop app)", "desktop"))
	huma.Post(api, "/desktop/replay-regen/start", h.handleStart,
		humacore.Op("postDesktopReplayRegenStart", "Start the replay regeneration queue (desktop app)", "desktop"),
		humacore.DefaultStatus(http.StatusAccepted))
	huma.Post(api, "/desktop/replay-regen/pause", h.handlePause,
		humacore.Op("postDesktopReplayRegenPause", "Pause the replay regeneration queue (desktop app)", "desktop"))
	huma.Post(api, "/desktop/replay-regen/resume", h.handleResume,
		humacore.Op("postDesktopReplayRegenResume", "Resume the replay regeneration queue (desktop app)", "desktop"))
	huma.Post(api, "/desktop/replay-regen/rescan", h.handleRescan,
		humacore.Op("postDesktopReplayRegenRescan", "Recount the replays behind the app version, in the background (desktop app)", "desktop"),
		humacore.DefaultStatus(http.StatusAccepted))
}

type replayRegenOutput struct {
	Body *domain.ReplayRegenStatus
}

func (h *DesktopReplayRegenHandler) handleStatus(_ context.Context, _ *struct{}) (*replayRegenOutput, error) {
	st := h.svc.Status()
	return &replayRegenOutput{Body: &st}, nil
}

func (h *DesktopReplayRegenHandler) handleStart(ctx context.Context, _ *struct{}) (*replayRegenOutput, error) {
	return h.command(ctx, h.svc.Start)
}

func (h *DesktopReplayRegenHandler) handlePause(ctx context.Context, _ *struct{}) (*replayRegenOutput, error) {
	return h.command(ctx, h.svc.Pause)
}

func (h *DesktopReplayRegenHandler) handleResume(ctx context.Context, _ *struct{}) (*replayRegenOutput, error) {
	return h.command(ctx, h.svc.Resume)
}

// handleRescan relance le décompte en arrière-plan (jamais lancé par la lecture de l'état) ; la réponse porte
// `checked: false` jusqu'à sa fin.
func (h *DesktopReplayRegenHandler) handleRescan(ctx context.Context, _ *struct{}) (*replayRegenOutput, error) {
	return h.command(ctx, h.svc.Rescan)
}

// command exécute une commande de la file et traduit ses refus : file absente (409 pour pause/reprise sans file),
// service non relié (503).
func (h *DesktopReplayRegenHandler) command(ctx context.Context, run func() (domain.ReplayRegenStatus, error)) (*replayRegenOutput, error) {
	st, err := run()
	switch {
	case err == nil:
		return &replayRegenOutput{Body: &st}, nil
	case errors.Is(err, replayregen.ErrNotRunning):
		return nil, humacore.NewError(http.StatusConflict, codeReplayRegenNotRunning, "no regeneration running")
	default:
		return nil, humacore.NewErrorFrom(ctx, http.StatusServiceUnavailable, codeReplayRegenUnavailable, err)
	}
}
