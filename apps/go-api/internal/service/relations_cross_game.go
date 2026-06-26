// Package service — relations_cross_game.go : injection ADDITIVE du badge
// cross-jeu sur le hub Relations (Phase 3b). BEST-EFFORT / LECTURE SEULE : le
// badge « Aussi sur {game} » est ajouté aux relations aussi croisées sur un
// AUTRE titre (xuid global, ADR 0008). Toute erreur d'accès cross-titre est
// avalée par le port (skip + log) ; l'endpoint /relations continue exactement
// comme avant si la dépendance est absente (crossGame == nil).
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/relations"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// appendCrossGameBadges enrichit (en place) les insights d'un badge cross-jeu
// pour chaque relation aussi croisée >= seuil sur un autre titre. No-op si
// aucune dépendance n'est injectée. Ne renvoie jamais d'erreur : le badge est
// best-effort et ne doit jamais dégrader la réponse Relations.
func (s *RelationsService) appendCrossGameBadges(ctx context.Context, insights []domain.RelationInsight) {
	if s.crossGame == nil || len(insights) == 0 {
		return
	}
	xuids := make([]string, 0, len(insights))
	for i := range insights {
		if insights[i].XUID != "" {
			xuids = append(xuids, insights[i].XUID)
		}
	}
	if len(xuids) == 0 {
		return
	}
	hits := s.crossGame.CooccurrencesByXUID(ctx, xuids)
	if len(hits) == 0 {
		return
	}
	added := 0
	for i := range insights {
		hit, ok := hits[insights[i].XUID]
		if !ok {
			continue
		}
		badge := relations.CrossGameBadge(hit.TitleDisplayName, hit.MatchesTogether)
		if badge == nil {
			continue
		}
		insights[i].Badges = append(insights[i].Badges, domain.RelationBadge{
			LabelKey:   badge.LabelKey,
			ColorToken: badge.ColorToken,
			Style:      badge.Style,
			Detail:     badge.Detail,
		})
		added++
	}
	slog.DebugContext(ctx, "relations: cross-game badges appended",
		"candidates", len(xuids), "hits", len(hits), "added", added)
}

// WithCrossGame injecte la dépendance cross-jeu (Phase 3b). Sans cette
// injection, le badge cross-jeu est inerte (chemin Phase 3a strictement
// inchangé). La dépendance encapsule l'énumération des autres titres via le
// TitleRegistry et la lecture best-effort de leur catalogue shared.
func (s *RelationsService) WithCrossGame(c port.CrossGameCooccurrence) *RelationsService {
	s.crossGame = c
	return s
}
