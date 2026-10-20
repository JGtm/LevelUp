// Package service — tactical_service_voisines.go : LES LECTURES VOISINES, servies avec la
// lecture demandee parce qu'elles sortent de la MEME lecture de la base.
//
// « Morts », « frags », « solde » et « victoires − defaites » se rasterisent toutes sur les
// positions de kill de l'univers ; « temps » et « trajets » sur les memes sidecars. Une fois
// cette lecture en main (la partie chere : univers sur la liste blanche, positions ou
// fichiers), rasteriser les autres grilles ne coute que du calcul. Les servir ensemble rend
// le changement de lecture instantane cote web, sans une requete de plus — et sans lire la
// base N fois pour N lectures (ADR 0036, I4).
//
// « Isole » n'a pas de voisine : sa lecture (contexte de chaque mort) n'en sert aucune autre.
//
// UNE VOISINE EN ECHEC N'EMPORTE PAS LA LECTURE DEMANDEE : elle est journalisee puis omise,
// et le web la demandera lui-meme s'il en a besoin.
package service

import (
	"context"

	"levelup/go-api/internal/domain"
)

// questionsDesPositions : les lectures servies par une seule lecture des positions de kill.
var questionsDesPositions = []string{
	domain.TacticalQuestionMorts,
	domain.TacticalQuestionKills,
	domain.TacticalQuestionSolde,
	domain.TacticalQuestionGagne,
}

// questionsDesSidecars : les lectures servies par une seule lecture des sidecars.
var questionsDesSidecars = []string{
	domain.TacticalQuestionTemps,
	domain.TacticalQuestionRoutes,
}

// enteteDeVoisine : ce qu'une voisine reprend de la lecture demandee — meme carte, meme axe,
// meme univers, memes grappes. Ses cellules, son echelle et son cadre lui sont propres.
func enteteDeVoisine(out *domain.TacticalRaster, question string) domain.TacticalRaster {
	return domain.TacticalRaster{
		MapID:              out.MapID,
		Question:           question,
		Qui:                out.Qui,
		MatchsFiltres:      out.MatchsFiltres,
		MatchsRetenus:      out.MatchsRetenus,
		MatchsEnAttente:    out.MatchsEnAttente,
		MatchsNonCuisables: out.MatchsNonCuisables,
		Grappes:            out.Grappes,
	}
}

// voisinesDesPositions rasterise, sur les positions DEJA LUES, les lectures de positions autres
// que celle demandee.
func (s *TacticalService) voisinesDesPositions(ctx context.Context, out *domain.TacticalRaster,
	mesure domain.TacticalUnivers, lecture domain.TacticalPositions, dans predicatQui) []domain.TacticalRaster {
	voisines := make([]domain.TacticalRaster, 0, len(questionsDesPositions)-1)
	for _, q := range questionsDesPositions {
		if q == out.Question {
			continue
		}
		v := enteteDeVoisine(out, q)
		lue, err := rasteriserLaCible(mesure, lecture, q, dans)
		if err != nil {
			s.logger.ErrorContext(ctx, "tactique: lecture voisine en echec, servie sans elle",
				"player", s.xuid, "map_id", out.MapID, "question", out.Question, "voisine", q, "err", err)
			continue
		}
		remplirRaster(&v, lue.Raster, q)
		voisines = append(voisines, v)
	}
	return voisines
}

// voisinesDesSidecars somme, sur les sidecars DEJA LUS, les lectures d'artefact autres que celle
// demandee.
func (s *TacticalService) voisinesDesSidecars(ctx context.Context, out *domain.TacticalRaster,
	univers domain.TacticalUnivers, sidecars map[string]*domain.TacticalRasterSidecar,
	scope domain.TacticalScope, ignores int) []domain.TacticalRaster {
	voisines := make([]domain.TacticalRaster, 0, len(questionsDesSidecars)-1)
	for _, q := range questionsDesSidecars {
		if q == out.Question {
			continue
		}
		v := enteteDeVoisine(out, q)
		if err := s.remplirLectureArtefact(ctx, &v, univers, sidecars, scope, ignores); err != nil {
			s.logger.ErrorContext(ctx, "tactique: lecture voisine en echec, servie sans elle",
				"player", s.xuid, "map_id", out.MapID, "question", out.Question, "voisine", q, "err", err)
			continue
		}
		voisines = append(voisines, v)
	}
	return voisines
}
