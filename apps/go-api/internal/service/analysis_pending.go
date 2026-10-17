package service

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// AnalysisPendingResolver décide, pour un lot de matchs, lesquels ont une analyse de film
// non finie et pour quelles causes (`analysis.AnalysisPendingParts`). Il ne se construit QUE pour un titre qui porte la
// capability du film (porte posée au câblage, cf. wire.newAnalysisPendingResolver) : un
// titre sans elle n'a pas de résolveur, et le champ reste faux partout.
//
// UN lot par appel, jamais une lecture par match (ADR 0036).
type AnalysisPendingResolver struct {
	repo port.AnalysisProbeRepository
	// replayExpected : le titre produit des artefacts de rejeu (`film.replay_artifact`).
	replayExpected bool
	// now : horloge injectable (tests de la borne des 24 h).
	now func() time.Time
}

// NewAnalysisPendingResolver construit le résolveur. repo nil → nil (rien à résoudre).
func NewAnalysisPendingResolver(repo port.AnalysisProbeRepository, replayExpected bool) *AnalysisPendingResolver {
	if repo == nil {
		return nil
	}
	return &AnalysisPendingResolver{repo: repo, replayExpected: replayExpected, now: func() time.Time { return time.Now().UTC() }}
}

// WithClock remplace l'horloge (tests).
func (r *AnalysisPendingResolver) WithClock(now func() time.Time) *AnalysisPendingResolver {
	if r != nil && now != nil {
		r.now = now
	}
	return r
}

// ReplayLookup dit si un match a son artefact de rejeu, et si la réponse est fiable.
type ReplayLookup struct {
	// Known : la présence des artefacts a pu être lue (faux → condition « rejeu » ignorée).
	Known bool
	// Has : l'artefact du match existe.
	Has func(matchID string) bool
}

// Resolve rend, pour chaque match_id du lot dont l'analyse est en cours, ses causes (jamais
// vides : un match sans cause est absent de la table). Récepteur nil, lot vide ou lecture en
// échec : table vide (l'échec est journalisé en WARN, jamais un indicateur fabriqué sur une
// lecture ratée).
func (r *AnalysisPendingResolver) Resolve(ctx context.Context, matchIDs []string, lookup ReplayLookup) map[string][]string {
	if r == nil || len(matchIDs) == 0 {
		return nil
	}
	probes, err := r.repo.LoadAnalysisProbes(ctx, matchIDs)
	if err != nil {
		slog.WarnContext(ctx, "analysis_pending: lecture de l'avancement en échec — aucun indicateur",
			"err", err, "matches", len(matchIDs))
		return nil
	}
	now := r.now()
	pending := make(map[string][]string)
	for _, id := range matchIDs {
		probe, ok := probes[id]
		if !ok {
			continue
		}
		has := lookup.Known && lookup.Has != nil && lookup.Has(id)
		parts := analysis.AnalysisPendingParts(analysis.AnalysisPendingInput{
			Probe: probe, Now: now,
			ReplayExpected: r.replayExpected, ReplayKnown: lookup.Known, HasReplay: has,
		})
		if len(parts) > 0 {
			pending[id] = parts
		}
	}
	return pending
}

// ResolvePending est `Resolve` sans lecture du rejeu : la condition « rejeu manquant » n'est pas
// évaluée (pages qui n'attendent aucun artefact de rejeu, ex. l'Emprise de l'Escouade).
func (r *AnalysisPendingResolver) ResolvePending(ctx context.Context, matchIDs []string) map[string][]string {
	return r.Resolve(ctx, matchIDs, ReplayLookup{})
}

var _ port.AnalysisPendingMatches = (*AnalysisPendingResolver)(nil)

// WithAnalysisPending injecte le résolveur de l'indicateur d'analyse (nil : champ toujours
// faux). Retourne le service (chaînage).
func (s *HomeService) WithAnalysisPending(r *AnalysisPendingResolver) *HomeService {
	s.analysisPending = r
	return s
}

// applyAnalysisPendingToRecentItems pose AnalysisPending sur les tuiles de match (récents +
// favoris) en UNE lecture pour les deux listes. `replays` nil = ensemble non résolu.
func (s *HomeService) applyAnalysisPendingToRecentItems(ctx context.Context, replays port.ReplayAvailability, itemLists ...[]domain.RecentMatchItem) {
	if s.analysisPending == nil {
		return
	}
	var ids []string
	seen := make(map[string]struct{})
	for _, items := range itemLists {
		for _, it := range items {
			if _, dup := seen[it.MatchID]; dup {
				continue
			}
			seen[it.MatchID] = struct{}{}
			ids = append(ids, it.MatchID)
		}
	}
	pending := s.analysisPending.Resolve(ctx, ids, ReplayLookup{Known: replays != nil, Has: replays.Has})
	if len(pending) == 0 {
		return
	}
	for _, items := range itemLists {
		for i := range items {
			parts := pending[items[i].MatchID]
			items[i].AnalysisPending = len(parts) > 0
			items[i].AnalysisPendingParts = parts
		}
	}
}

// WithAnalysisPending injecte le résolveur de l'indicateur d'analyse (nil : champ toujours
// faux). Retourne le service (chaînage).
func (s *MatchViewService) WithAnalysisPending(r *AnalysisPendingResolver) *MatchViewService {
	s.analysisPending = r
	return s
}

// applyHeaderAnalysisPending pose AnalysisPending sur l'en-tête de la vue match. À appeler
// APRÈS applyMatchHeaderReplay : la condition « rejeu absent » lit header.ReplayAvailable, et
// elle n'est évaluée que si le service de rejeu est câblé (sinon l'absence ne dit rien).
func (s *MatchViewService) applyHeaderAnalysisPending(ctx context.Context, h *domain.MatchViewHeader) {
	if s.analysisPending == nil || h == nil || h.MatchID == "" {
		return
	}
	lookup := ReplayLookup{Known: s.replaySvc != nil, Has: func(string) bool { return h.ReplayAvailable }}
	parts := s.analysisPending.Resolve(ctx, []string{h.MatchID}, lookup)[h.MatchID]
	h.AnalysisPending = len(parts) > 0
	h.AnalysisPendingParts = parts
}
