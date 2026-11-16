package service

// match_view_lead_share.go — LA MAINMISE DE L'EN-TÊTE DE LA VUE DU MATCH : part de la durée jouée
// passée en tête du score par l'équipe du joueur de la page, et retournements.

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/leadshareload"
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
	got, ok := leadshareload.Load(ctx, s.leadShareRepo, []string{h.MatchID}, s.xuid,
		"match_view")
	if !ok {
		return
	}
	if v, found := got[h.MatchID]; found {
		h.LeadShare, h.LeadChanges = leadshareload.Pointers(v)
	}
}
