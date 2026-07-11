// Package service — engagement_squad_service.go : GetSquadSession (Mock 15 v2)
// + helpers matchBundle/computeTeammateMeanPace.
//
// Decoupe de engagement_player_service.go pour respecter la limite 500L
// (cf. arch-rules § Modularité).
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// GetSquadSession charge la session squad (Mock 15 v2). Calcule pour chaque
// match commun les means team-level + per-player paces.
//
// Reference plan : §6.6.1, §8.7.
func (s *PlayerEngagementService) GetSquadSession(
	ctx context.Context,
	matchIDs []string,
	teammates []domain.EngagementCoefficient,
) (*domain.SquadEngagementSession, error) {
	if len(matchIDs) == 0 {
		return &domain.SquadEngagementSession{
			Labels:  []string{},
			Players: []domain.SquadPlayerEngagement{},
		}, nil
	}
	slog.DebugContext(ctx, "PlayerEngagementService.GetSquadSession: computing",
		"xuid", s.xuid, "n_matches", len(matchIDs), "n_teammates", len(teammates))

	players := buildSquadPlayers(s.xuid, s.gamertag, teammates, len(matchIDs))
	session := &domain.SquadEngagementSession{
		Labels:         make([]string, 0, len(matchIDs)),
		MapNames:       make([]string, 0, len(matchIDs)),
		LobbyPerPlayer: make([]float64, 0, len(matchIDs)),
		TeamExpected:   make([]float64, 0, len(matchIDs)),
		TeamObserved:   make([]float64, 0, len(matchIDs)),
		Players:        players,
	}

	for i, mid := range matchIDs {
		bundle, ok := s.computeMatchBundle(ctx, mid, i)
		if !ok {
			continue
		}
		s.appendMatchToSession(ctx, session, bundle)
	}
	return session, nil
}

// buildSquadPlayers construit la liste des joueurs (main + teammates) avec
// les slices PaceObserved pre-allouees. Le main est toujours en premier.
func buildSquadPlayers(
	mainXUID, mainGamertag string,
	teammates []domain.EngagementCoefficient,
	matchCount int,
) []domain.SquadPlayerEngagement {
	players := make([]domain.SquadPlayerEngagement, 0, 1+len(teammates))
	players = append(players, domain.SquadPlayerEngagement{
		XUID:         mainXUID,
		Gamertag:     mainGamertag,
		PaceObserved: make([]float64, 0, matchCount),
	})
	for _, t := range teammates {
		if t.XUID == mainXUID {
			continue // évite le doublon si l'appelant inclut déjà le main player
		}
		players = append(players, domain.SquadPlayerEngagement{
			XUID:         t.XUID,
			Gamertag:     t.Gamertag,
			PaceObserved: make([]float64, 0, matchCount),
		})
	}
	return players
}

// appendMatchToSession ajoute un match au session output : labels + paces
// team-level + paces per-player.
func (s *PlayerEngagementService) appendMatchToSession(
	ctx context.Context,
	session *domain.SquadEngagementSession,
	bundle matchBundle,
) {
	summary := bundle.summary
	session.Labels = append(session.Labels, summary.Label)
	mapName := ""
	if summary.MapName != nil {
		mapName = *summary.MapName
	}
	session.MapNames = append(session.MapNames, mapName)
	session.LobbyPerPlayer = append(session.LobbyPerPlayer, summary.PaceLobby)
	session.TeamExpected = append(session.TeamExpected, summary.PaceAttendu)
	session.TeamObserved = append(session.TeamObserved, summary.PaceTeam)
	for j := range session.Players {
		var pace float64
		if session.Players[j].XUID == s.xuid {
			// Main player : reutilise le mean deja calcule par
			// computeMatchBundle (curve smoothing 90s identique au reste
			// de l'app). Evite la double-evaluation.
			pace = summary.PaceJoueur
		} else {
			// Coequipier : recompute avec son xuid en target sur les
			// memes events (pas de re-fetch DB). Le mean retourne est
			// coherent avec la methode utilisee pour le main player.
			pace = s.computeTeammateMeanPace(ctx, bundle, session.Players[j].XUID)
		}
		session.Players[j].PaceObserved = append(session.Players[j].PaceObserved, pace)
	}
}

// matchBundle regroupe le summary deja calcule + les events / context bruts
// pour permettre au caller de re-calculer le mean pour un autre xuid sans
// refaire les LoadEventsForMatch / LoadMatchEngagementContext.
type matchBundle struct {
	summary   domain.EngagementMatchSummary
	mctx      *port.MatchEngagementContext
	events    []canonical.HighlightEvent
	teamXUIDs map[string]bool
}

// computeMatchBundle est computeMatchSummary + retour des events/context pour
// permettre la reutilisation par GetSquadSession (overlay teammates).
func (s *PlayerEngagementService) computeMatchBundle(
	ctx context.Context,
	matchID string,
	index int,
) (matchBundle, bool) {
	mctx, err := s.repo.LoadMatchEngagementContext(ctx, matchID, s.xuid)
	if err != nil || mctx == nil || mctx.IsPvE {
		return matchBundle{}, false
	}
	events, _ := s.repo.LoadEventsForMatch(ctx, matchID)
	if len(events) == 0 {
		return matchBundle{}, false
	}
	teamXUIDs, _ := s.repo.LoadTeamXUIDs(ctx, matchID, mctx.TargetTeamID, s.xuid)
	input := s.buildInputForMatch(ctx, mctx, events, teamXUIDs, matchID)

	result, err := temporal.ComputeEngagementScore(input)
	if err != nil || len(result.EngagementCurve) == 0 {
		return matchBundle{}, false
	}
	summary := domain.EngagementMatchSummary{
		MatchID:         matchID,
		Label:           fmt.Sprintf("M%d", index+1),
		MapName:         mctx.MapName,
		StartedAt:       time.UnixMilli(mctx.StartTimeMS),
		PaceJoueur:      result.MeanPaceJoueur,
		PaceTeam:        result.MeanPaceTeam,
		PaceAttendu:     meanPace(result.EngagementCurve, func(p domain.EngagementPoint) float64 { return p.PaceAttendu }),
		PaceLobby:       result.MeanPaceLobby,
		EngagementScore: result.EngagementScore,
	}
	return matchBundle{
		summary:   summary,
		mctx:      mctx,
		events:    events,
		teamXUIDs: teamXUIDs,
	}, true
}

// computeTeammateMeanPace recompute le pace moyen d'un coequipier sur le
// meme match (events deja en memoire) — methode coherente avec celle du
// main player (mean de la curve smooth 90s, pas un simple count/duration).
//
// On reutilise la teamXUIDs deja chargee pour le main player ; pour les
// teammates, leur "team" est le main + autres teammates (mais on s'en fiche
// pour le calcul du PaceJoueur — seuls les events du teammate comptent).
func (s *PlayerEngagementService) computeTeammateMeanPace(
	ctx context.Context,
	bundle matchBundle,
	teammateXUID string,
) float64 {
	if teammateXUID == "" || len(bundle.events) == 0 {
		return 0
	}
	// Construire un teamXUIDs "vu du teammate" : tous les autres humains
	// presents dans bundle.teamXUIDs + le main player. On ne dispose pas du
	// team_id du teammate ici sans une nouvelle query, mais ce n'est pas
	// necessaire pour le PaceJoueur (qui ne depend que des events du
	// teammate, pas du partition team/lobby).
	mateTeamXUIDs := make(map[string]bool, len(bundle.teamXUIDs)+1)
	for k, v := range bundle.teamXUIDs {
		if v {
			mateTeamXUIDs[k] = true
		}
	}
	mateTeamXUIDs[s.xuid] = true
	delete(mateTeamXUIDs, teammateXUID) // exclure le teammate lui-meme de "son" team

	playerEvents, teamEvents, lobbyEvents := splitMatchEvents(bundle.events, teammateXUID, mateTeamXUIDs)
	durationMS := bundle.mctx.EndTimeMS - bundle.mctx.StartTimeMS
	input := temporal.EngagementScoreInput{
		PlayerEvents: playerEvents,
		TeamEvents:   teamEvents,
		LobbyEvents:  lobbyEvents,
		NTeam:        bundle.mctx.NTeam,
		NHumansLobby: bundle.mctx.NHumansLobby,
		XUID:         teammateXUID,
		MatchStartMS: 0,
		MatchEndMS:   durationMS,
		// Cold-start (aucun coef/bin fourni) : on ne lit que MeanPaceJoueur.
		Mode:       normalizeMode(bundle.mctx.IsRanked),
		IsTeamMode: bundle.mctx.IsTeamMode,
	}
	result, err := temporal.ComputeEngagementScore(input)
	if err != nil {
		slog.DebugContext(ctx, "computeTeammateMeanPace: compute failed",
			"match_id", bundle.mctx.MatchID, "teammate", teammateXUID, "err", err)
		return 0
	}
	return result.MeanPaceJoueur
}
