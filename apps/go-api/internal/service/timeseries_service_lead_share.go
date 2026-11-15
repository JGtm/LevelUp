// Package service — timeseries_service_lead_share.go : LA MAINMISE DES SÉRIES TEMPORELLES — la part de
// chaque match passée en tête du score, sa moyenne par mois (population solo complète) et par carte
// (matchs filtrés contre historique).
package service

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"time"

	"levelup/go-api/internal/analysis/leadshare"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
)

// WithLeadShare injecte la lecture de la Mainmise (câblée sous film.lead_share) et le joueur de la
// page. Nil : aucune ligne ni agrégat ne porte la mesure.
func (s *TimeseriesService) WithLeadShare(repo port.LeadShareRepository) *TimeseriesService {
	s.leadShareRepo = repo
	return s
}

// attachLeadShare pose la Mainmise sur les lignes de match, les cartes et les mois de la réponse.
// UNE lecture bornée par la liste des matchs du périmètre filtré et de la population solo complète
// (ADR 0036). Une lecture en échec est journalisée avant de dégrader : rien n'est posé, jamais un
// zéro.
func (s *TimeseriesService) attachLeadShare(
	ctx context.Context, resp *domain.TimeseriesPageResponse, filtered, historical []legacymatch.StatsMatchRow,
	groups domain.GroupesDeCartes,
) {
	if s.leadShareRepo == nil || s.playerXUID == "" {
		return
	}
	ids := idsDesMatchs(filtered, historical)
	if len(ids) == 0 {
		return
	}
	got, err := s.leadShareRepo.LoadLeadShare(ctx, ids, s.playerXUID)
	if err != nil {
		if !errors.Is(err, games.ErrCapabilityNotSupported) {
			slog.WarnContext(ctx, "timeseries: mainmise illisible, page sans la mesure", "err", err)
		}
		return
	}
	for i := range resp.MatchRows {
		if v, ok := got[resp.MatchRows[i].MatchID]; ok {
			share, changes := v.Share, v.Changes
			resp.MatchRows[i].LeadShare, resp.MatchRows[i].LeadChanges = &share, &changes
		}
	}
	loc := i18n.FromContext(ctx)
	current := leadShareByMap(filtered, got, groups, loc)
	historic := leadShareByMap(historical, got, groups, loc)
	for i := range resp.MapBreakdown {
		row := &resp.MapBreakdown[i]
		if a, ok := current[row.MapID]; ok {
			row.LeadShare, row.LeadShareMatches = &a.avg, a.n
		}
		if a, ok := historic[row.MapID]; ok {
			row.HistoricalLeadShare, row.HistoricalLeadShareMatches = &a.avg, a.n
		}
	}
	resp.LeadShareByMonth = leadShareByMonth(historical, got)
}

// idsDesMatchs rend les identifiants des deux populations, sans doublon, dans l'ordre de première
// apparition.
func idsDesMatchs(populations ...[]legacymatch.StatsMatchRow) []string {
	seen := map[string]struct{}{}
	var ids []string
	for _, rows := range populations {
		for i := range rows {
			if _, ok := seen[rows[i].MatchID]; ok {
				continue
			}
			seen[rows[i].MatchID] = struct{}{}
			ids = append(ids, rows[i].MatchID)
		}
	}
	return ids
}

// moyenneMainmise : la moyenne des parts mesurées et leur nombre.
type moyenneMainmise struct {
	avg float64
	n   int
}

// leadShareByMap rend, par clé de carte (`mapKeyOf`), la moyenne des parts des matchs MESURÉS. Une
// carte sans match mesuré est absente du résultat.
func leadShareByMap(
	rows []legacymatch.StatsMatchRow, got map[string]leadshare.PlayerShare, groups domain.GroupesDeCartes, loc i18n.Locale,
) map[string]moyenneMainmise {
	parts := map[string][]float64{}
	for i := range rows {
		v, ok := got[rows[i].MatchID]
		if !ok {
			continue
		}
		key, _ := mapKeyOf(rows[i], groups, loc)
		parts[key] = append(parts[key], v.Share)
	}
	out := make(map[string]moyenneMainmise, len(parts))
	for key, shares := range parts {
		avg, n := leadshare.Average(shares)
		out[key] = moyenneMainmise{avg: avg, n: n}
	}
	return out
}

// leadShareByMonth rend la Mainmise moyenne de chaque mois (UTC) qui compte au moins un match, du
// plus ancien au plus récent. Un mois sans match mesuré garde ses matchs et pas de moyenne.
func leadShareByMonth(rows []legacymatch.StatsMatchRow, got map[string]leadshare.PlayerShare) []domain.LeadSharePeriodPoint {
	type mois struct {
		start  time.Time
		shares []float64
		total  int
	}
	byMonth := map[string]*mois{}
	for i := range rows {
		t := rows[i].StartTime.UTC()
		key := t.Format("2006-01")
		m := byMonth[key]
		if m == nil {
			m = &mois{start: time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)}
			byMonth[key] = m
		}
		m.total++
		if v, ok := got[rows[i].MatchID]; ok {
			m.shares = append(m.shares, v.Share)
		}
	}
	out := make([]domain.LeadSharePeriodPoint, 0, len(byMonth))
	for key, m := range byMonth {
		p := domain.LeadSharePeriodPoint{
			Month: key, StartDate: m.start.Format("2006-01-02"), Matches: m.total, MeasuredMatches: len(m.shares),
		}
		if avg, n := leadshare.Average(m.shares); n > 0 {
			p.LeadShare = &avg
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartDate < out[j].StartDate })
	return out
}
