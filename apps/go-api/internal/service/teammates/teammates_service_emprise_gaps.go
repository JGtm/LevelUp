package teammates

// teammates_service_emprise_gaps.go — « ANALYSE DU FILM EN COURS » / « FILM NON ANALYSÉ » de l'Emprise.
//
// Les sections de l'onglet Emprise lisent le résumé d'usage du film. Un match dont le film n'est pas
// encore décodé n'en a pas : sans cette distinction, la page retirait les sections et rien ne disait
// pourquoi (constat du 2026-10-17 : les cinq matchs du jour attendaient le verrou de décodage). Le bloc
// dit donc, pour son périmètre, combien de matchs sans résumé ont une analyse EN COURS (la règle unique
// `analysis.AnalysisPendingParts`, via le résolveur — jamais une seconde règle ici) et combien n'en ont
// pas. Le web choisit alors entre `AnalysisPendingSlot` et `EmptyStateNotice`.
//
// Un titre qui ne publie pas de résumé d'usage (`film_unsupported`) n'a rien à attendre : le bloc ne
// compte rien, la section reste absente (capability absente ≠ données manquantes).

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// WithAnalysisPending injecte la décision « analyse en cours » du bloc Emprise (nil : tout match sans
// résumé d'usage est compté « non analysé »). Un résolveur nil de type concret est sûr : il rend une
// table vide.
func (s *TeammatesService) WithAnalysisPending(r port.AnalysisPendingMatches) *TeammatesService {
	s.analysisPending = r
	return s
}

// noteFilmGaps pose MatchesFilmPending / MatchesFilmMissing sur le bloc publié.
func (s *TeammatesService) noteFilmGaps(ctx context.Context, block *domain.SquadEmpriseBlock) {
	if block.FilmUnavailable == domain.EmpriseFilmUnsupported {
		return
	}
	ids := matchesWithoutFilm(block)
	if len(ids) == 0 {
		return
	}
	var pending map[string][]string
	if s.analysisPending != nil {
		pending = s.analysisPending.ResolvePending(ctx, ids)
	}
	block.MatchesFilmPending, block.MatchesFilmMissing = splitFilmGaps(ids, pending)
}

// matchesWithoutFilm — les matchs du périmètre dont le résumé d'usage n'existe pas.
func matchesWithoutFilm(block *domain.SquadEmpriseBlock) []string {
	var ids []string
	for _, m := range block.Matches {
		if !m.HasFilm {
			ids = append(ids, m.MatchID)
		}
	}
	return ids
}

// splitFilmGaps répartit les matchs sans résumé : analyse en cours (présents dans `pending`) et les autres.
func splitFilmGaps(ids []string, pending map[string][]string) (inProgress, missing int) {
	for _, id := range ids {
		if len(pending[id]) > 0 {
			inProgress++
		} else {
			missing++
		}
	}
	return inProgress, missing
}
