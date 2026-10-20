package service

// tactical_service_vignettes.go — LE MINI-PLAN D'UNE VIGNETTE DE LA GRILLE (lot F,
// 2026-09-13, maquette 034b1915).
//
// # UNE SEULE LECTURE POUR TOUTE LA GRILLE
//
// `MortsParCarte` rend MES morts du perimetre, deja groupees par carte : une requete, pas
// une par vignette. La rasterisation, elle, est en memoire — quelques centaines de points
// par carte sur une grille grossiere.
//
// # AUCUNE ERREUR N'ARRETE LA GRILLE
//
// La grille des cartes jouees se lit sur le REGISTRE, qui ne depend d'aucun film. Les
// mini-plans, eux, dependent des positions mesurees : un titre qui ne les lit pas, ou une
// lecture qui echoue, laissent les vignettes AVEC LEUR SEUL FOND — c'est ce qu'elles
// montraient avant ce lot. L'echec est journalise, jamais avale, et jamais promu en panne
// de page.

import (
	"context"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain"
)

// peindreLesVignettes pose le mini-plan « ou je meurs » de chaque carte de la grille.
func (s *TacticalService) peindreLesVignettes(ctx context.Context, page *domain.TacticalMapsPage,
	scope domain.TacticalScope,
) {
	if len(page.Cartes) == 0 || !positionsDeKillLisibles(s.caps) {
		return
	}
	parCarte, err := s.repo.MortsParCarte(ctx, requeteDuScope(s.xuid, "", scope))
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: mini-plans des vignettes non servis",
			"player", s.xuid, "err", err)
		return
	}
	grille, err := tactical.NouvelleGrille(domain.TacticalTuilePasM)
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: pas de vignette refuse par la grille",
			"player", s.xuid, "pas_m", domain.TacticalTuilePasM, "err", err)
		return
	}
	peintes := 0
	for i := range page.Cartes {
		points := parCarte[page.Cartes[i].MapID]
		if len(points) == 0 {
			continue
		}
		if s.peindreUneVignette(ctx, &page.Cartes[i], grille, points) {
			peintes++
		}
	}
	s.logger.InfoContext(ctx, "tactique: mini-plans des vignettes",
		"player", s.xuid, "cartes", len(page.Cartes), "cartes_peintes", peintes,
		"pas_m", domain.TacticalTuilePasM)
}

// peindreUneVignette rasterise les morts d'UNE carte et habille sa vignette. Rend faux
// quand rien n'est peint (aucune cellule ne passe le plancher de matchs distincts).
func (s *TacticalService) peindreUneVignette(ctx context.Context, carte *domain.TacticalMapCard,
	grille tactical.Grille, points []domain.PositionSample,
) bool {
	// L'UNIVERS EST CELUI DES MATCHS QUI ONT PRODUIT CES POINTS, pas les `carte.Matchs` du
	// registre : le denominateur « par match » d'une vignette doit etre le nombre de matchs
	// MESURES, sans quoi la valeur baisserait avec la couverture de film au lieu du jeu
	// (correction G2, la meme que pour les lectures de placement).
	ids := matchsDesPoints(points)
	raster, err := tactical.Rasterise(grille, ids, points)
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: rasterisation d'une vignette en echec",
			"player", s.xuid, "map_id", carte.MapID, "err", err)
		return false
	}
	cellules := raster.Cellules()
	if len(cellules) == 0 {
		return false
	}
	carte.Cellules = cellules
	carte.Echelle = tactical.Echelle(cellules)
	carte.Bornes = raster.Bornes()
	carte.PasM = raster.PasM()
	return true
}

// matchsDesPoints rend les match_id DISTINCTS d'une liste de points, ordre d'apparition.
func matchsDesPoints(points []domain.PositionSample) []string {
	vus := make(map[string]bool, len(points))
	out := make([]string, 0, len(points))
	for _, p := range points {
		if vus[p.MatchID] {
			continue
		}
		vus[p.MatchID] = true
		out = append(out, p.MatchID)
	}
	return out
}
