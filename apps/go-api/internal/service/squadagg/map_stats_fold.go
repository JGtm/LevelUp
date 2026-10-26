package squadagg

import "levelup/go-api/internal/domain"

// FoldMapStats replie les stats historiques par carte (clé = map_id, LoadMapStatsForSquad) en UNE entrée
// par groupe de cartes : clé = identifiant de base du groupe, parties et victoires sommées, performance
// moyenne pondérée par le nombre de matchs qui la fondent (PerfCount ; à défaut le total de parties). Sans
// groupe, la table est rendue telle quelle. Une carte hors groupe garde sa clé.
func FoldMapStats(stats map[string]domain.MapSquadStats, groups domain.GroupesDeCartes) map[string]domain.MapSquadStats {
	if groups.Vide() || len(stats) == 0 {
		return stats
	}
	type acc struct {
		wins, total, perfN int
		perfSum            float64
	}
	accs := make(map[string]*acc, len(stats))
	for id, s := range stats {
		key := groups.Groupe(id)
		a := accs[key]
		if a == nil {
			a = &acc{}
			accs[key] = a
		}
		a.wins += s.Wins
		a.total += s.Total
		if s.PerfAvg != nil {
			w := s.PerfCount
			if w == 0 {
				w = s.Total
			}
			a.perfN += w
			a.perfSum += *s.PerfAvg * float64(w)
		}
	}
	out := make(map[string]domain.MapSquadStats, len(accs))
	for key, a := range accs {
		s := domain.MapSquadStats{Wins: a.wins, Total: a.total, PerfCount: a.perfN}
		if a.perfN > 0 {
			v := a.perfSum / float64(a.perfN)
			s.PerfAvg = &v
		}
		out[key] = s
	}
	return out
}
