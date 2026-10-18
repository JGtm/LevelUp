package teammates

// teammates_service_emprise_gaps.go — le résolveur « analyse du film en cours » du bloc Emprise. La règle
// (`squadagg.NoteFilmGaps`) est commune à l'Escouade, aux Séries temporelles et à Sessions.

import "levelup/go-api/internal/port"

// WithAnalysisPending injecte la décision « analyse en cours » du bloc Emprise (nil : tout match sans
// résumé d'usage est compté « non analysé »). Un résolveur nil de type concret est sûr : il rend une
// table vide.
func (s *TeammatesService) WithAnalysisPending(r port.AnalysisPendingMatches) *TeammatesService {
	s.analysisPending = r
	return s
}
