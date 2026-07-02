// Package service — MatchEventsService : timeline canonique d'events d'un match
// (kill-feed / timeline, chargée on-demand).
//
// Orchestration (arch-rules) : combine l'adapter de titre (LoadMatchEvents →
// canonical.MatchEventTimeline) et un GamertagResolver (chokepoint canonique
// v_gamertag_lookup). Aucun SQL inline, aucune logique de titre par slug : la
// capability est portée par l'adapter (ErrCapabilityNotSupported propagée).
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// MatchEventsService implémente port.MatchEventsService.
type MatchEventsService struct {
	data     games.TitleDataAdapter
	resolver port.GamertagResolver // nil OK → enrichissement gamertag skippé
	logger   *slog.Logger
}

// NewMatchEventsService construit le service. data peut être nil (titre sans
// builder d'adapter enregistré) → GetMatchEvents retourne ErrCapabilityNotSupported.
// resolver peut être nil (ex. titre déjà gamertag-keyé comme Halo 5, ou tests).
func NewMatchEventsService(data games.TitleDataAdapter, resolver port.GamertagResolver) *MatchEventsService {
	return &MatchEventsService{data: data, resolver: resolver, logger: slog.Default()}
}

// WithLogger injecte un logger (sinon slog.Default()). Chaînable.
func (s *MatchEventsService) WithLogger(l *slog.Logger) *MatchEventsService {
	if l != nil {
		s.logger = l
	}
	return s
}

// GetMatchEvents charge la timeline canonique d'events du match via l'adapter du
// titre, puis enrichit les identités xuid-seules avec leur gamertag (chokepoint).
//
//   - adapter nil → ErrCapabilityNotSupported (le handler dégrade en 503) ;
//   - ErrCapabilityNotSupported / erreur adapter → propagée ;
//   - timeline nil (adapter laxiste) → timeline vide non nulle.
func (s *MatchEventsService) GetMatchEvents(ctx context.Context, matchID string, opts canonical.MatchEventOptions) (*canonical.MatchEventTimeline, error) {
	if s.data == nil {
		return nil, games.ErrCapabilityNotSupported
	}
	tl, err := s.data.LoadMatchEvents(ctx, matchID, opts)
	if err != nil {
		return nil, err
	}
	if tl == nil {
		return &canonical.MatchEventTimeline{MatchID: matchID}, nil
	}
	s.enrichGamertags(ctx, tl)
	return tl, nil
}

// enrichGamertags peuple PlayerIdentity.Gamertag (vide) des events via le
// chokepoint canonique. No-op si resolver nil, ou si aucune identité xuid-seule
// (ex. Halo 5 = déjà gamertag-keyé, XUID vide). Échec resolver = dégradation
// gracieuse (identité laissée sans gamertag, masquée au rendu front).
func (s *MatchEventsService) enrichGamertags(ctx context.Context, tl *canonical.MatchEventTimeline) {
	if s.resolver == nil {
		return
	}
	need := map[string]struct{}{}
	forEachMatchEventIdentity(tl, func(id *canonical.PlayerIdentity) {
		if id != nil && id.XUID != "" && id.Gamertag == "" {
			need[id.XUID] = struct{}{}
		}
	})
	if len(need) == 0 {
		return
	}
	xuids := make([]string, 0, len(need))
	for x := range need {
		xuids = append(xuids, x)
	}
	gamertags, err := s.resolver.ResolveGamertags(ctx, xuids)
	if err != nil {
		s.logger.WarnContext(ctx, "match_events_gamertag_resolve_failed",
			"match_id", tl.MatchID, "xuids", len(xuids), "err", err)
		return
	}
	forEachMatchEventIdentity(tl, func(id *canonical.PlayerIdentity) {
		if id != nil && id.XUID != "" && id.Gamertag == "" {
			if gt, ok := gamertags[id.XUID]; ok {
				id.Gamertag = gt
			}
		}
	})
}

// forEachMatchEventIdentity applique fn à chaque PlayerIdentity (pointeur) des
// events : Killer, Victim, Player. Les pointeurs nil sont passés tels quels (fn
// les ignore). Mutation in-place via les pointeurs partagés avec l'event.
func forEachMatchEventIdentity(tl *canonical.MatchEventTimeline, fn func(*canonical.PlayerIdentity)) {
	for i := range tl.Events {
		fn(tl.Events[i].Killer)
		fn(tl.Events[i].Victim)
		fn(tl.Events[i].Player)
	}
}
