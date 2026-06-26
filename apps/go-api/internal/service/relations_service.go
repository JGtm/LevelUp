// Package service — relations_service.go : orchestration du hub Communauté >
// Relations. Combine RelationsRepository (agrégats SQL) + analysis/relations
// (badges, catégorie, aperçu) → domain.RelationsPageResponse. Aucun SQL.
package service

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/relations"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// RelationsService orchestre le hub Relations.
type RelationsService struct {
	repo port.RelationsRepository
	now  func() time.Time // injectable pour les tests (badges temporels)
}

// NewRelationsService crée un RelationsService.
func NewRelationsService(repo port.RelationsRepository) *RelationsService {
	return &RelationsService{repo: repo, now: time.Now}
}

// withNow injecte une horloge déterministe (tests des badges temporels).
func (s *RelationsService) withNow(now func() time.Time) *RelationsService {
	s.now = now
	return s
}

// GetRelationsPage construit la page complète : aperçu (compteurs + binôme +
// bête noire) et liste des relations enrichies (badges + catégorie).
func (s *RelationsService) GetRelationsPage(ctx context.Context) (domain.RelationsPageResponse, error) {
	rawRows, err := s.repo.GetRelations(ctx)
	if err != nil {
		return domain.RelationsPageResponse{}, fmt.Errorf("RelationsService.GetRelationsPage: %w", err)
	}

	stats := make([]relations.RelationStats, 0, len(rawRows))
	for _, r := range rawRows {
		stats = append(stats, relationStatsFromRaw(r))
	}

	now := s.now()
	insights := make([]domain.RelationInsight, 0, len(rawRows))
	for i := range rawRows {
		insights = append(insights, buildRelationInsight(rawRows[i], stats[i], now))
	}

	return domain.RelationsPageResponse{
		Overview:  buildOverview(stats),
		Relations: insights,
	}, nil
}

// relationStatsFromRaw projette la ligne brute repo en stats d'analyse (pur),
// calculant les taux de victoire et le duel ratio.
func relationStatsFromRaw(r domain.RelationRawRow) relations.RelationStats {
	st := relations.RelationStats{
		XUID:            r.XUID,
		Gamertag:        r.Gamertag,
		TotalMatches:    r.TotalMatches,
		TeammateMatches: r.TeammateCount,
		EnemyMatches:    r.EnemyCount,
		TeammateWins:    r.TeammateWins,
		EnemyWins:       r.EnemyWins,
		KillsDealt:      r.KillsDealt,
		DeathsSuffered:  r.DeathsSuffered,
	}
	st.TeammateWinRate = relationWinRate(r.TeammateWins, r.TeammateLosses)
	st.EnemyWinRate = relationWinRate(r.EnemyWins, r.EnemyLosses)
	st.DuelRatio = duelRatio(r.KillsDealt, r.DeathsSuffered)
	if !r.FirstSeen.IsZero() {
		t := r.FirstSeen
		st.FirstSeen = &t
	}
	if !r.LastSeen.IsZero() {
		t := r.LastSeen
		st.LastSeen = &t
	}
	return st
}

// buildRelationInsight assemble le DTO d'une relation (stats + badges +
// catégorie) à partir de la ligne brute et des stats d'analyse.
func buildRelationInsight(r domain.RelationRawRow, st relations.RelationStats, now time.Time) domain.RelationInsight {
	insight := domain.RelationInsight{
		XUID:            r.XUID,
		Gamertag:        r.Gamertag,
		TotalMatches:    r.TotalMatches,
		TeammateMatches: r.TeammateCount,
		TeammateWins:    r.TeammateWins,
		TeammateWinRate: st.TeammateWinRate,
		EnemyMatches:    r.EnemyCount,
		EnemyWins:       r.EnemyWins,
		EnemyWinRate:    st.EnemyWinRate,
		AvgKDAWith:      r.AvgKDAWith,
		AvgKDAAgainst:   r.AvgKDAAgainst,
		KillsDealt:      r.KillsDealt,
		DeathsSuffered:  r.DeathsSuffered,
		DuelRatio:       st.DuelRatio,
		FirstSeenAt:     formatRFC3339(r.FirstSeen),
		LastSeenAt:      formatRFC3339(r.LastSeen),
		Category:        relations.Categorize(st),
		IsCore:          relations.IsCore(st),
		Badges:          projectBadges(relations.ComputeBadges(st, now)),
	}
	return insight
}

// buildOverview agrège les compteurs et sélectionne binôme + bête noire.
func buildOverview(stats []relations.RelationStats) domain.RelationsOverview {
	c := relations.ComputeCounts(stats)
	return domain.RelationsOverview{
		DistinctPlayers: c.DistinctPlayers,
		AlliesCount:     c.AlliesCount,
		RivalsCount:     c.RivalsCount,
		CoreCount:       c.CoreCount,
		TopAlly:         topRefToRef(relations.SelectTopAlly(stats)),
		TopNemesis:      topRefToRef(relations.SelectTopNemesis(stats)),
	}
}

// projectBadges projette []relations.Badge → []domain.RelationBadge.
func projectBadges(badges []relations.Badge) []domain.RelationBadge {
	out := make([]domain.RelationBadge, 0, len(badges))
	for _, b := range badges {
		out = append(out, domain.RelationBadge{
			LabelKey:   b.LabelKey,
			ColorToken: b.ColorToken,
			Style:      b.Style,
			Detail:     b.Detail,
		})
	}
	return out
}

// topRefToRef projette *relations.TopRef → *domain.RelationRef (nil-safe).
func topRefToRef(t *relations.TopRef) *domain.RelationRef {
	if t == nil {
		return nil
	}
	return &domain.RelationRef{Gamertag: t.Gamertag, WinRate: t.WinRate, Matches: t.Matches}
}

// relationWinRate : nil si W+L == 0, sinon ratio 0..1.
func relationWinRate(wins, losses int) *float64 {
	total := wins + losses
	if total == 0 {
		return nil
	}
	rate := analysis.WinRate(wins, total)
	return &rate
}

// duelRatio : kills/deaths. nil si deaths==0 ET kills==0 (aucune donnée de
// duel). Si deaths==0 et kills>0 → ratio = float64(kills) (domination totale).
func duelRatio(kills, deaths int) *float64 {
	if deaths == 0 && kills == 0 {
		return nil
	}
	var ratio float64
	if deaths > 0 {
		ratio = float64(kills) / float64(deaths)
	} else {
		ratio = float64(kills)
	}
	return &ratio
}

// formatRFC3339 : nil pour un time zéro, sinon RFC3339 UTC.
func formatRFC3339(t time.Time) *string {
	if t.IsZero() {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}
