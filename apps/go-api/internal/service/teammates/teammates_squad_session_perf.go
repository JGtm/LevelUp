package teammates

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// squadPerfByMatch : la performance d'escouade de chaque match de `rows`, moyenne des scores
// des membres (joueur principal et coéquipiers choisis) qui en portent un, même règle que
// l'historique des matchs (attachSquadPerformance). Un match sans aucun score n'a pas
// d'entrée. Les scores des coéquipiers viennent de leurs propres lignes (squadLoader) ;
// sans chargeur, seule la note du joueur principal entre.
func (s *TeammatesService) squadPerfByMatch(
	ctx context.Context, rows []domain.SquadMatchRow, selectedGamertags []string,
) map[string]float64 {
	type acc struct {
		sum float64
		n   int
	}
	byMatch := make(map[string]*acc, len(rows))
	inScope := make(map[string]struct{}, len(rows))
	for _, m := range rows {
		if _, dup := inScope[m.MatchID]; dup {
			continue
		}
		inScope[m.MatchID] = struct{}{}
		if m.PerformanceScore != nil {
			byMatch[m.MatchID] = &acc{sum: *m.PerformanceScore, n: 1}
		}
	}
	if s.squadLoader != nil {
		for _, gt := range selectedGamertags {
			if gt == s.gamertag {
				continue // la note du joueur principal est déjà portée par ses lignes
			}
			teamRows, err := s.squadLoader.LoadFor(ctx, s.titleSlug, gt, port.PlayerMatchFilters{})
			if err != nil {
				slog.WarnContext(ctx, "teammates_session_perf_load_failed", "gamertag", gt, "err", err)
				continue
			}
			for _, r := range teamRows {
				if _, ok := inScope[r.Summary.MatchID]; !ok || r.Enrichment.PerformanceScore == nil {
					continue
				}
				a := byMatch[r.Summary.MatchID]
				if a == nil {
					a = &acc{}
					byMatch[r.Summary.MatchID] = a
				}
				a.sum += *r.Enrichment.PerformanceScore
				a.n++
			}
		}
	}
	out := make(map[string]float64, len(byMatch))
	for id, a := range byMatch {
		out[id] = a.sum / float64(a.n)
	}
	return out
}
