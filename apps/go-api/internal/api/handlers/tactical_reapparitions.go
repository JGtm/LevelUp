// Package handlers — tactical_reapparitions.go : la lecture « reapparitions adverses » de l'onglet Tactique.
//
//	POST /players/{player_slug}/tactical/{map_id}/reapparitions-adverses   (corps : perimetre + zone)
//
// Montee avec les autres lectures de l'onglet (TacticalHandler), sur le sous-routeur joueur : ownership
// (ADR 0029) et titre herites du groupe. C'est une lecture en POST (le perimetre est une liste de
// match_id), declaree comme telle par le prefixe `/tactical/` de `middleware.readOnlyPostPrefixes`.
//
// ZERO LOGIQUE ICI. Le handler valide la forme du map_id, delegue, traduit un refus en statut. Une
// lecture sans assez de matchs ou de réapparitions est un 200 dont `etat` vaut « vide » et dont
// `raison` dit pourquoi : ce n'est pas un refus.
package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
)

// mountReapparitionsAdverses enregistre la route sur l'API Huma du sous-routeur.
func (h *TacticalHandler) mountReapparitionsAdverses(api huma.API) {
	huma.Post(api, "/tactical/{map_id}/reapparitions-adverses", h.handleGetReapparitionsAdverses,
		humacore.Op("getTacticalReapparitionsAdverses",
			"Ou les adversaires reapparaissent quand l'equipe tient la zone choisie, avec la repartition sans condition en reference", "tactical"))
}

// tacticalReapparitionsAdversesBody : le perimetre des autres lectures, plus la zone choisie.
type tacticalReapparitionsAdversesBody struct {
	MatchIDs    []string `json:"match_ids,omitempty" doc:"Perimetre : les match_id retenus par la barre de filtres (resolus via /filters/match-ids). Liste vide ou absente = aucun match."`
	Coequipiers []string `json:"coequipiers,omitempty" doc:"XUIDs de la composition choisie (0 a 3). Restreint aux matchs ou TOUS y etaient dans mon equipe."`
	VolumeIndex int      `json:"volume_index" doc:"Indice de la zone nommee choisie dans le catalogue de la carte (zones[].volume_index de la lecture de placement)."`
}

type tacticalReapparitionsAdversesInput struct {
	PlayerSlug string `path:"player_slug"`
	MapID      string `path:"map_id"`
	Body       tacticalReapparitionsAdversesBody
}

type tacticalReapparitionsAdversesOutput struct {
	Body domain.TacticalReapparitionsAdverses
}

// handleGetReapparitionsAdverses rend la lecture pour la zone choisie. Meme validation du map_id et
// meme traduction d'erreurs que les autres lectures de l'onglet.
func (h *TacticalHandler) handleGetReapparitionsAdverses(ctx context.Context,
	in *tacticalReapparitionsAdversesInput) (*tacticalReapparitionsAdversesOutput, error) {
	mapID, ok := MapIDValide(in.MapID)
	if !ok {
		return nil, humacore.NewErrorFrom(ctx, http.StatusNotFound, "tactical_map_unknown",
			domain.ErrTacticalCarteInconnue)
	}
	svc, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewErrorFrom(ctx, http.StatusNotFound, "player_not_found", err)
	}
	lecture, err := svc.ReapparitionsAdverses(ctx, domain.TacticalReapparitionsAdversesRequest{
		MapID:       mapID,
		Scope:       scopeDepuis(in.Body.MatchIDs, in.Body.Coequipiers),
		VolumeIndex: in.Body.VolumeIndex,
	})
	if err != nil {
		return nil, mapTacticalError(ctx, err, "tactical.reapparitions_adverses")
	}
	return &tacticalReapparitionsAdversesOutput{Body: lecture}, nil
}
