// Package service — relations_moments_service.go : orchestration de la section
// « Moments & Rivalités » (Phase 3a). Combine RelationsRepository (heatmap
// relation × heure + timeline rival) + analysis/relations (bucketing day-parts,
// WR glissant, écart de frags) → domain.RelationsMomentsResponse. Aucun SQL.
package service

import (
	"context"
	"fmt"
	"sort"

	"levelup/go-api/internal/analysis/relations"
	"levelup/go-api/internal/domain"
)

// Plafonds de lisibilité (réserve produit : ne pas surcharger). Constantes
// nommées (pas de magic number).
const (
	// momentsHeatmapTopN : nombre de relations dans le heatmap agrégé (aligné sur
	// le cap d'affichage front MAX_HEATMAP_ROWS = 12, RelationsMomentsHeatmap).
	momentsHeatmapTopN = 12
	// momentsMaxRivalries : nombre de cartes revanche (bête noire + autres).
	momentsMaxRivalries = 3
	// momentsRivalMinEnemyMatches : seuil minimal de matchs en ennemi pour une
	// carte revanche (sinon la frise/WR glissant ne sont pas parlants).
	momentsRivalMinEnemyMatches = 3
	// momentsTimelineLimit : nombre de duels conservés par rival (N derniers).
	momentsTimelineLimit = 20
)

// GetRelationsMoments construit la section « Moments & Rivalités » : heatmap
// agrégé relation × tranche horaire (top-N par matchs communs) + cartes revanche
// (top rivaux par matchs en ennemi). Même segmentation serveur que la page.
func (s *RelationsService) GetRelationsMoments(ctx context.Context, input domain.FilterContextInput) (domain.RelationsMomentsResponse, error) {
	scope, err := s.resolveScope(ctx, input)
	if err != nil {
		return domain.RelationsMomentsResponse{}, fmt.Errorf("RelationsService.GetRelationsMoments: scope: %w", err)
	}

	heatmap, heatmapDow, err := s.buildHeatmap(ctx, scope)
	if err != nil {
		return domain.RelationsMomentsResponse{}, err
	}
	rivalries, err := s.buildRivalries(ctx, scope)
	if err != nil {
		return domain.RelationsMomentsResponse{}, err
	}
	return domain.RelationsMomentsResponse{
		Heatmap:      heatmap,
		HeatmapDow:   heatmapDow,
		Rivalries:    rivalries,
		TopRelations: momentsHeatmapTopN,
	}, nil
}

// buildHeatmap récupère les comptes relation × (heure, jour) (top-N) et les agrège
// en deux vues : par heure (0..23) ET par jour de semaine.
func (s *RelationsService) buildHeatmap(ctx context.Context, scope []string) ([]domain.RelationHeatmapCell, []domain.RelationHeatmapDowCell, error) {
	raw, err := s.repo.GetRelationsHeatmap(ctx, scope, momentsHeatmapTopN)
	if err != nil {
		return nil, nil, fmt.Errorf("RelationsService.GetRelationsMoments: heatmap: %w", err)
	}
	return aggregateHeatmapHours(raw), aggregateHeatmapDow(raw), nil
}

// aggregateHeatmapHours replie les comptes par heure (0..23) par relation.
// Cellules vides omises. Tri stable (gamertag, hour).
func aggregateHeatmapHours(raw []domain.RelationHeatmapRawRow) []domain.RelationHeatmapCell {
	type key struct {
		xuid string
		hour int
	}
	agg := map[key]int{}
	meta := map[string]string{} // xuid → gamertag
	for _, r := range raw {
		k := key{xuid: r.XUID, hour: r.Hour}
		agg[k] += r.Count
		meta[r.XUID] = r.Gamertag
	}
	out := make([]domain.RelationHeatmapCell, 0, len(agg))
	for k, count := range agg {
		out = append(out, domain.RelationHeatmapCell{
			XUID:     k.xuid,
			Gamertag: meta[k.xuid],
			Hour:     k.hour,
			Count:    count,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Gamertag != out[j].Gamertag {
			return out[i].Gamertag < out[j].Gamertag
		}
		return out[i].Hour < out[j].Hour
	})
	return out
}

// aggregateHeatmapDow replie les comptes par JOUR DE SEMAINE (0=dimanche … 6=samedi)
// par relation, depuis les mêmes lignes brutes. Cellules vides omises. Tri stable
// (gamertag, jour).
func aggregateHeatmapDow(raw []domain.RelationHeatmapRawRow) []domain.RelationHeatmapDowCell {
	type key struct {
		xuid string
		dow  int
	}
	agg := map[key]int{}
	meta := map[string]string{} // xuid → gamertag
	for _, r := range raw {
		k := key{xuid: r.XUID, dow: r.Dow}
		agg[k] += r.Count
		meta[r.XUID] = r.Gamertag
	}
	out := make([]domain.RelationHeatmapDowCell, 0, len(agg))
	for k, count := range agg {
		out = append(out, domain.RelationHeatmapDowCell{
			XUID:      k.xuid,
			Gamertag:  meta[k.xuid],
			DayOfWeek: k.dow,
			Count:     count,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Gamertag != out[j].Gamertag {
			return out[i].Gamertag < out[j].Gamertag
		}
		return out[i].DayOfWeek < out[j].DayOfWeek
	})
	return out
}

// buildRivalries sélectionne les top rivaux (par matchs en ennemi) puis charge
// leur timeline et calcule les métriques de revanche.
func (s *RelationsService) buildRivalries(ctx context.Context, scope []string) ([]domain.RelationRivalry, error) {
	rawRows, err := s.repo.GetRelations(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("RelationsService.GetRelationsMoments: relations: %w", err)
	}
	rivals := selectTopRivals(rawRows, momentsMaxRivalries)
	out := make([]domain.RelationRivalry, 0, len(rivals))
	for _, rv := range rivals {
		duels, err := s.repo.GetRivalTimeline(ctx, rv.XUID, scope, momentsTimelineLimit)
		if err != nil {
			return nil, fmt.Errorf("RelationsService.GetRelationsMoments: timeline %s: %w", rv.XUID, err)
		}
		out = append(out, buildRivalry(rv, duels))
	}
	return out, nil
}

// selectTopRivals retourne les `max` relations avec le plus de matchs en ennemi
// (>= seuil), triées EnemyCount DESC puis xuid pour le déterminisme.
func selectTopRivals(rows []domain.RelationRawRow, max int) []domain.RelationRawRow {
	candidates := make([]domain.RelationRawRow, 0, len(rows))
	for _, r := range rows {
		if r.EnemyCount >= momentsRivalMinEnemyMatches {
			candidates = append(candidates, r)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].EnemyCount != candidates[j].EnemyCount {
			return candidates[i].EnemyCount > candidates[j].EnemyCount
		}
		return candidates[i].XUID < candidates[j].XUID
	})
	if len(candidates) > max {
		candidates = candidates[:max]
	}
	return candidates
}

// buildRivalry assemble la carte revanche : projection des duels + métriques.
func buildRivalry(rv domain.RelationRawRow, raw []domain.RelationDuelRawRow) domain.RelationRivalry {
	duels := make([]relations.Duel, 0, len(raw))
	entries := make([]domain.RelationDuelEntry, 0, len(raw))
	for _, d := range raw {
		dom := relations.ResultToDuel(d.Result)
		duels = append(duels, relations.Duel{
			Outcome:       dom,
			KillsOnRival:  d.KillsOnRival,
			DeathsByRival: d.DeathsByRival,
		})
		entries = append(entries, domain.RelationDuelEntry{
			MatchID:       d.MatchID,
			StartedAt:     formatRFC3339(d.StartTime),
			Outcome:       duelOutcomeLabel(dom),
			Mode:          d.Mode,
			MapName:       d.MapName,
			KillsOnRival:  d.KillsOnRival,
			DeathsByRival: d.DeathsByRival,
		})
	}
	m := relations.ComputeRivalryMetrics(duels)
	return domain.RelationRivalry{
		XUID:           rv.XUID,
		Gamertag:       rv.Gamertag,
		EnemyMatches:   rv.EnemyCount,
		Duels:          entries,
		RollingWinRate: m.RollingWinRate,
		RollingWindow:  relations.RollingWinRateWindow,
		RecentWinRate:  m.RecentWinRate,
		GlobalWinRate:  m.GlobalWinRate,
		CurrentStreak:  m.CurrentStreak,
		FragGap:        m.FragGap,
	}
}

// Libellés canonical de duel exposés tels quels dans le DTO (contrat API).
const (
	duelLabelWin   = "win"
	duelLabelLoss  = "loss"
	duelLabelOther = "other"
)

// duelOutcomeLabel : libellé canonical du duel pour le DTO ("win"|"loss"|"other").
func duelOutcomeLabel(d relations.DuelOutcome) string {
	switch d {
	case relations.DuelWin:
		return duelLabelWin
	case relations.DuelLoss:
		return duelLabelLoss
	default:
		return duelLabelOther
	}
}
