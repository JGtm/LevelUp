// server_replay_generation.go — le montage de la génération à la demande du rejeu d'un match : deux routes sous le groupe
// du rejeu, seulement sous la capacité GenerateReplays (application de bureau, poste de développement). L'hébergé n'a ni
// la route ni la file : il garde l'action d'administration « construire le rejeu ».
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/port"
)

// mountReplayGeneration monte POST .../replay/generate et GET .../replay/generation sur le groupe du rejeu de
// /players/{player_slug} : les routes héritent du titre actif, de l'authentification des écritures, de la propriété du
// joueur (écriture : propriétaire ou administrateur) et de la capacité de rejeu du titre ; `participated` ajoute la
// garde de participation au match (404 match_not_participant).
func mountReplayGeneration(r chi.Router, cfg *config.AppConfig, newSvc handlers.ServiceFactory[port.ReplayGeneration],
	participated func(http.Handler) http.Handler, opts ...humacore.MountOption) {
	if !cfg.Capabilities.GenerateReplays {
		return
	}
	handlers.NewReplayGenerationHandler(newSvc).Mount(r.With(participated), opts...)
}
