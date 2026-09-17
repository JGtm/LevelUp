// Package service — match_view_encounter_assists.go : colonne « Assistances » de
// l'historique des rencontres de la vue match (assistances échangées sur tout
// l'historique commun, même agrégat que la page Relations).
//
// Fichier dédié : match_view_data_loaders.go et match_view_builders_team.go portent déjà
// la dette de taille gelée ; le chargement et l'attache vivent ici.
package service

import (
	"context"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/domain"
)

// loadEncounterAssists charge en parallèle (errgroup de loadMatchViewDataParallel) les
// assistances échangées avec les joueurs du match. Erreur loggée par goLoad, map nil →
// la colonne affiche « — ».
func (s *MatchViewService) loadEncounterAssists(gctx context.Context, g *errgroup.Group, matchID string, d *matchViewData) {
	goLoad(gctx, g, matchID, "encounter_assists", func() error {
		var e error
		d.encounterAssists, e = s.repo.GetMatchEncounterAssists(gctx, matchID, s.xuid)
		return e
	})
}

// attachEncounterAssists pose le bloc d'assistances sur chaque rencontre présente dans
// la map. Une entrée à zéro match mesuré n'est pas publiée (absent = non mesuré).
func attachEncounterAssists(rows []domain.MatchEncounterRow, byXUID map[string]domain.RelationAssists) {
	for i := range rows {
		a, ok := byXUID[rows[i].XUID]
		if !ok || a.MatchesMeasured == 0 {
			continue
		}
		rows[i].Assists = &a
	}
}
