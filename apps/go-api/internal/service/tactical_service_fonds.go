package service

// tactical_service_fonds.go — LE CALAGE DU FOND PORTE PAR CHAQUE CARTE OUVRABLE DE LA GRILLE
// (plan backlog 2026-10-16, E7, D-10).
//
// POURQUOI ICI. La grille des cartes jouees montait une vignette par carte, et chaque vignette
// demandait SON calage par une requete (`GET /tactical/{map}/background`) : sur 46 cartes
// ouvrables, 46 requetes pour une reponse que le serveur sait deja composer avec la grille.
// Le calage est une donnee de REFERENCE du titre (versionnee, cuite avec le fond) ; le lire
// ici, dans la reponse de `/tactical/maps`, supprime les requetes sans changer la source :
// c'est le MEME `MapBackgroundForMap` que sert la route par carte.
//
// UNE CARTE SANS FOND N'EST PAS UNE ERREUR (toutes les cartes n'en ont pas) : son calage reste
// absent, sans trace d'erreur. Toute AUTRE panne est journalisee et laisse la carte sans calage
// — la vignette s'affiche alors sans fond, la grille n'est jamais arretee par un fond.

import (
	"context"
	"errors"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
)

// poserLesCalagesDeFond pose le calage du fond de chaque carte OUVRABLE de la page. Les cartes
// sous le plancher n'ont pas de vignette : leur fond n'est pas lu. Sans service de rejeu
// (`WithReplay` non cable), la page reste sans calage — aucune erreur, le titre n'a pas de fond.
// Rend le nombre de cartes qui en portent un.
func (s *TacticalService) poserLesCalagesDeFond(ctx context.Context, page *domain.TacticalMapsPage) int {
	if s.detail.rejeu == nil {
		return 0
	}
	defer timing.FromContext(ctx).Section("tactical_maps_fonds")()
	poses := 0
	for i := range page.Cartes {
		carte := &page.Cartes[i]
		if carte.SousPlancher {
			continue
		}
		fond, err := s.detail.rejeu.MapBackgroundForMap(ctx, carte.MapID)
		if errors.Is(err, port.ErrMapBackgroundNotAvailable) {
			continue
		}
		if err != nil {
			s.logger.WarnContext(ctx, "tactique: calage du fond d'une carte en echec",
				"player", s.xuid, "map_id", carte.MapID, "err", err)
			continue
		}
		if fond == nil {
			continue
		}
		calage := fond.Calibration
		carte.CalageFond = &calage
		poses++
	}
	return poses
}
