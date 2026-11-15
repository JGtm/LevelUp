package service

// match_view_lead_share.go — LA MAINMISE DE L'EN-TÊTE DE LA VUE DU MATCH : part de la durée jouée
// passée en tête du score par l'équipe du joueur de la page, et retournements.

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// WithLeadShare injecte la lecture de la Mainmise (câblée sous film.lead_share). Nil : l'en-tête ne
// porte jamais la mesure, comme un titre qui ne la produit pas.
func (s *MatchViewService) WithLeadShare(r port.LeadShareRepository) *MatchViewService {
	s.leadShareRepo = r
	return s
}

// applyHeaderLeadShare pose `lead_share` et `lead_changes` quand le match est mesuré. Une lecture en
// échec est journalisée avant de dégrader : le champ reste absent, jamais zéro. Une capability
// absente (`games.ErrCapabilityNotSupported`) est le cas normal d'un titre sans la mesure.
func (s *MatchViewService) applyHeaderLeadShare(ctx context.Context, h *domain.MatchViewHeader) {
	if s.leadShareRepo == nil || h == nil || h.MatchID == "" || s.xuid == "" {
		return
	}
	got, err := s.leadShareRepo.LoadLeadShare(ctx, []string{h.MatchID}, s.xuid)
	if err != nil {
		if !errors.Is(err, games.ErrCapabilityNotSupported) {
			slog.WarnContext(ctx, "match_view: mainmise illisible, en-tête sans la mesure",
				"match_id", h.MatchID, "err", err)
		}
		return
	}
	if v, ok := got[h.MatchID]; ok {
		share, changes := v.Share, v.Changes
		h.LeadShare, h.LeadChanges = &share, &changes
	}
}
