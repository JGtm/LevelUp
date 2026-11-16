// Package service — session_page_lead_share.go : LA MAINMISE DE LA PAGE SESSIONS — la part de chaque
// match passée en tête du score, et sa moyenne sur la session affichée et la session comparée.
package service

import (
	"context"

	"levelup/go-api/internal/analysis/leadshare"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/leadshareload"
)

// WithSessionLeadShare injecte la lecture de la Mainmise (câblée sous film.lead_share). Nil : aucune
// ligne ni session ne porte la mesure, comme un titre qui ne la produit pas.
func (s *SessionPageService) WithSessionLeadShare(repo port.LeadShareRepository) *SessionPageService {
	s.leadShareRepo = repo
	return s
}

// attachSessionLeadShare pose `lead_share` et `lead_changes` sur les lignes de la session affichée et
// de la session comparée, puis la moyenne des matchs mesurés sur chaque session. UNE lecture pour
// les deux sessions, bornée par la liste de leurs matchs (ADR 0036). Une lecture en échec est
// journalisée avant de dégrader : rien n'est posé, jamais un zéro.
func (s *SessionPageService) attachSessionLeadShare(ctx context.Context, resp *domain.SessionPageResponse) {
	if s.leadShareRepo == nil || s.sessionXUID == "" {
		return
	}
	ids := make([]string, 0, len(resp.Matches)+len(resp.CompareMatches))
	for _, rows := range [][]domain.SessionDetailMatchRow{resp.Matches, resp.CompareMatches} {
		for i := range rows {
			ids = append(ids, rows[i].MatchID)
		}
	}
	if len(ids) == 0 {
		return
	}
	got, ok := leadshareload.Load(ctx, s.leadShareRepo, ids, s.sessionXUID,
		"session_page")
	if !ok {
		return
	}
	applyLeadShareToRows(resp.Matches, got)
	applyLeadShareToRows(resp.CompareMatches, got)
	applyLeadShareToEntry(resp.CurrentSession, resp.Matches)
	applyLeadShareToEntry(resp.CompareSession, resp.CompareMatches)
}

// applyLeadShareToRows pose la mesure sur les lignes mesurées ; une ligne absente de `got` reste
// sans champ.
func applyLeadShareToRows(rows []domain.SessionDetailMatchRow, got map[string]leadshare.PlayerShare) {
	for i := range rows {
		v, ok := got[rows[i].MatchID]
		if !ok {
			continue
		}
		rows[i].LeadShare, rows[i].LeadChanges = leadshareload.Pointers(v)
	}
}

// applyLeadShareToEntry pose la moyenne des lignes MESURÉES et leur nombre sur la session. Sans
// aucune ligne mesurée, la session ne porte rien.
func applyLeadShareToEntry(entry *domain.SessionCompareEntry, rows []domain.SessionDetailMatchRow) {
	if entry == nil {
		return
	}
	var shares []float64
	for i := range rows {
		if rows[i].LeadShare != nil {
			shares = append(shares, *rows[i].LeadShare)
		}
	}
	if avg, n := leadshare.Average(shares); n > 0 {
		entry.LeadShare, entry.LeadShareMatches = &avg, n
	}
}
