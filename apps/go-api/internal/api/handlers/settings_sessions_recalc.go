package handlers

import (
	"context"
)

// handlePostRecalculateSessions lance un recalcul des sessions pour tous les joueurs.
// POST /settings/sessions/recalculate — retourne un AsyncJobStatus (202). Le service crée le job et le
// lance en arrière-plan ; il finit en échec (relançable) dès qu'un joueur n'a pas pu être recalculé.
func (h *SettingsHandler) handlePostRecalculateSessions(ctx context.Context, _ *struct{}) (*asyncJobOutput, error) {
	job, err := h.sessionsRecalc.StartSessionsRecalc(ctx)
	if err != nil {
		return nil, syncJobHTTPError(ctx, err)
	}
	return &asyncJobOutput{Body: job}, nil
}
