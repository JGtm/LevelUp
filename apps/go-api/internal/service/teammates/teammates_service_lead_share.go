// Package teammates — teammates_service_lead_share.go : LA MAINMISE DE LA SECTION DYNAMIQUE — la part
// de chaque match du périmètre passée en tête du score par l'équipe du joueur principal.
package teammates

import (
	"context"
	"sort"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/leadshareload"
)

// WithLeadShare injecte la lecture de la Mainmise (câblée sous film.lead_share). Nil : la page ne
// porte pas la série, comme un titre qui ne la produit pas.
func (s *TeammatesService) WithLeadShare(repo port.LeadShareRepository) *TeammatesService {
	s.leadShareRepo = repo
	return s
}

// buildSquadLeadShare rend la série de la Mainmise du périmètre, du plus ancien au plus récent. UNE
// lecture bornée par la liste des matchs du périmètre (ADR 0036) ; l'escouade joue du même côté, la
// part est celle de l'équipe du joueur principal. Un match non mesuré reste dans la série, sans
// chiffre. Une lecture en échec est journalisée avant de dégrader : pas de série, jamais un zéro.
func (s *TeammatesService) buildSquadLeadShare(
	ctx context.Context, rows []domain.SquadMatchRow, mainXUID string,
) []domain.SquadLeadSharePoint {
	if len(rows) == 0 {
		return nil
	}
	ids := make([]string, 0, len(rows))
	for i := range rows {
		ids = append(ids, rows[i].MatchID)
	}
	got, ok := leadshareload.Load(ctx, s.leadShareRepo, ids, mainXUID, "squad: mainmise illisible, page sans la série")
	if !ok {
		return nil
	}
	ordered := append([]domain.SquadMatchRow(nil), rows...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].StartTime.Before(ordered[j].StartTime) })
	out := make([]domain.SquadLeadSharePoint, 0, len(ordered))
	for i := range ordered {
		r := ordered[i]
		pt := domain.SquadLeadSharePoint{
			MatchID: r.MatchID, StartTime: r.StartTime.UTC().Format(time.RFC3339), Outcome: r.Outcome,
			SessionLabel: r.SessionLabel, MapName: r.MapUI,
		}
		if v, ok := got[r.MatchID]; ok {
			pt.LeadShare, pt.LeadChanges = leadshareload.Pointers(v)
		}
		out = append(out, pt)
	}
	return out
}
