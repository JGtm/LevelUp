// Package service — engagement_player_service.go : PlayerEngagementService et
// helpers Phase 4 (handlers HTTP per-player) + Phase 4.b (Timeseries / Squad).
//
// Decoupe depuis engagement_score_service.go pour respecter la limite 500L
// par fichier (CLAUDE.md regle 14).
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// PlayerEngagementService est un wrapper avec xuid baked-in destine aux
// handlers HTTP. Charge metadata + events + history + coefs via le repo, puis
// appelle l'algo pur. Pas d'acces SQL direct, pas d'appel cross-service.
type PlayerEngagementService struct {
	repo port.EngagementScoreRepository
	xuid string
}

// NewPlayerEngagementService cree un service per-player.
func NewPlayerEngagementService(repo port.EngagementScoreRepository, xuid string) *PlayerEngagementService {
	return &PlayerEngagementService{repo: repo, xuid: xuid}
}

// GetMatchEngagement charge le contexte du match, recompute la courbe et le
// score (live), et retourne le resultat. Utilise par GET /matches/{id}/engagement.
func (s *PlayerEngagementService) GetMatchEngagement(
	ctx context.Context,
	matchID string,
) (*domain.EngagementScoreResult, error) {
	mctx, err := s.repo.LoadMatchEngagementContext(ctx, matchID, s.xuid)
	if err != nil {
		return nil, fmt.Errorf("PlayerEngagementService: load match context: %w", err)
	}
	if mctx == nil {
		return nil, ErrEngagementMatchNotFound
	}
	if mctx.IsPvE {
		return nil, ErrEngagementPvENotSupported
	}

	events, err := s.repo.LoadEventsForMatch(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("PlayerEngagementService: load events: %w", err)
	}
	teamXUIDs, err := s.repo.LoadTeamXUIDs(ctx, matchID, mctx.TargetTeamID, s.xuid)
	if err != nil {
		return nil, fmt.Errorf("PlayerEngagementService: load team xuids: %w", err)
	}

	input := s.buildInputForMatch(ctx, mctx, events, teamXUIDs, matchID)
	result, err := temporal.ComputeEngagementScore(input)
	if err != nil {
		return nil, fmt.Errorf("PlayerEngagementService: compute: %w", err)
	}
	return &result, nil
}

// GetEngagementProfile retourne tous les coefficients du joueur.
func (s *PlayerEngagementService) GetEngagementProfile(
	ctx context.Context,
) ([]domain.EngagementCoefficient, error) {
	if s.xuid == "" {
		return nil, errors.New("PlayerEngagementService.GetEngagementProfile: xuid required")
	}
	coefs, err := s.repo.LoadAllCoefficients(ctx, s.xuid)
	if err != nil {
		if errors.Is(err, port.ErrEngagementUnavailable) {
			return []domain.EngagementCoefficient{}, nil
		}
		return nil, fmt.Errorf("PlayerEngagementService.GetEngagementProfile: %w", err)
	}
	return coefs, nil
}

// GetTimeseries retourne les N derniers matchs PvP du joueur avec leurs
// paces calcules a la volee (Mock 11 sur Timeseries intensity tab).
//
// Reference plan : §6.6.3.
func (s *PlayerEngagementService) GetTimeseries(
	ctx context.Context,
	limit int,
) ([]domain.EngagementMatchSummary, error) {
	if s.xuid == "" {
		return nil, errors.New("PlayerEngagementService.GetTimeseries: xuid required")
	}
	if limit <= 0 {
		limit = 50
	}

	matchIDs, err := s.loadRecentPvPMatchIDs(ctx, limit)
	if err != nil {
		if errors.Is(err, port.ErrEngagementUnavailable) {
			return []domain.EngagementMatchSummary{}, nil
		}
		return nil, err
	}
	slog.DebugContext(ctx, "PlayerEngagementService.GetTimeseries: computing",
		"xuid", s.xuid, "limit", limit, "n_matches", len(matchIDs))

	out := make([]domain.EngagementMatchSummary, 0, len(matchIDs))
	for i, mid := range matchIDs {
		summary, ok := s.computeMatchSummary(ctx, mid, i)
		if ok {
			out = append(out, summary)
		}
	}
	slog.InfoContext(ctx, "PlayerEngagementService.GetTimeseries: done",
		"xuid", s.xuid, "n_returned", len(out))
	return out, nil
}

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

	session := &domain.SquadEngagementSession{
		Labels:         make([]string, 0, len(matchIDs)),
		LobbyPerPlayer: make([]float64, 0, len(matchIDs)),
		TeamExpected:   make([]float64, 0, len(matchIDs)),
		TeamObserved:   make([]float64, 0, len(matchIDs)),
		Players:        make([]domain.SquadPlayerEngagement, 0, len(teammates)),
	}
	for _, t := range teammates {
		session.Players = append(session.Players, domain.SquadPlayerEngagement{
			XUID:         t.XUID,
			Gamertag:     t.XUID,
			PaceObserved: make([]float64, 0, len(matchIDs)),
		})
	}

	for i, mid := range matchIDs {
		summary, ok := s.computeMatchSummary(ctx, mid, i)
		if !ok {
			continue
		}
		session.Labels = append(session.Labels, summary.Label)
		session.LobbyPerPlayer = append(session.LobbyPerPlayer, summary.PaceLobby)
		session.TeamExpected = append(session.TeamExpected, summary.PaceAttendu)
		session.TeamObserved = append(session.TeamObserved, summary.PaceTeam)
		for j := range session.Players {
			pace := s.computePlayerPace(ctx, mid, session.Players[j].XUID)
			session.Players[j].PaceObserved = append(session.Players[j].PaceObserved, pace)
		}
	}
	return session, nil
}

// =============================================================================
// Helpers prives
// =============================================================================

// buildInputForMatch construit l'input temporal a partir d'un MatchEngagementContext
// + events + teamXUIDs deja charges. Mutualise GetMatchEngagement et computeMatchSummary.
func (s *PlayerEngagementService) buildInputForMatch(
	ctx context.Context,
	mctx *port.MatchEngagementContext,
	events []canonical.HighlightEvent,
	teamXUIDs map[string]bool,
	matchID string,
) temporal.EngagementScoreInput {
	playerEvents, teamEvents, lobbyEvents := splitMatchEvents(events, s.xuid, teamXUIDs)
	modeCategory := normalizeMode(mctx.IsRanked)
	history, _ := s.loadHistorySafeByMode(ctx, modeCategory, matchID)
	coefTeam, coefLobby := s.loadCoefsSafe(ctx, modeCategory)
	return temporal.EngagementScoreInput{
		PlayerEvents:   playerEvents,
		TeamEvents:     teamEvents,
		LobbyEvents:    lobbyEvents,
		NTeam:          mctx.NTeam,
		NHumansLobby:   mctx.NHumansLobby,
		XUID:           s.xuid,
		MatchStartMS:   mctx.StartTimeMS,
		MatchEndMS:     mctx.EndTimeMS,
		History:        history,
		CoefTeamShare:  coefTeam,
		CoefLobbyShare: coefLobby,
		PersonalScore:  mctx.PersonalScore,
		Kills:          mctx.Kills,
		Assists:        mctx.Assists,
		Mode:           modeCategory,
		IsTeamMode:     mctx.IsTeamMode,
	}
}

// loadHistorySafeByMode wrappe LoadPlayerHistory en degradant gracieusement.
func (s *PlayerEngagementService) loadHistorySafeByMode(
	ctx context.Context,
	modeCategory, excludeMatchID string,
) ([]domain.HistoricalEngagementBrut, error) {
	filter := port.EngagementHistoryFilter{
		XUID:           s.xuid,
		ModeCategory:   modeCategory,
		Limit:          HistoryWindow,
		ExcludeMatchID: excludeMatchID,
	}
	history, err := s.repo.LoadPlayerHistory(ctx, filter)
	if err != nil {
		if errors.Is(err, port.ErrEngagementUnavailable) {
			return nil, nil
		}
		return nil, err
	}
	return history, nil
}

// loadCoefsSafe charge les coefs avec defaut neutre 1.0/1.0 en cold start.
func (s *PlayerEngagementService) loadCoefsSafe(
	ctx context.Context,
	modeCategory string,
) (coefTeam, coefLobby float64) {
	coef, err := s.repo.LoadEngagementCoefficient(ctx, s.xuid, modeCategory)
	if err != nil || coef == nil {
		return 1.0, 1.0
	}
	return coef.CoefTeamShare, coef.CoefLobbyShare
}

// loadRecentPvPMatchIDs liste les match_ids PvP recents via la repo (si elle
// expose la methode). Fallback slice vide si non disponible (mocks de test).
func (s *PlayerEngagementService) loadRecentPvPMatchIDs(
	ctx context.Context,
	limit int,
) ([]string, error) {
	type lister interface {
		ListRecentPvPMatchIDs(ctx context.Context, xuid string, limit int) ([]string, error)
	}
	if l, ok := s.repo.(lister); ok {
		return l.ListRecentPvPMatchIDs(ctx, s.xuid, limit)
	}
	return []string{}, nil
}

// computeMatchSummary recompute les means d'engagement pour un match donne.
// Retourne false si le match n'est pas calculable (PvE, match court, etc.).
func (s *PlayerEngagementService) computeMatchSummary(
	ctx context.Context,
	matchID string,
	index int,
) (domain.EngagementMatchSummary, bool) {
	mctx, err := s.repo.LoadMatchEngagementContext(ctx, matchID, s.xuid)
	if err != nil || mctx == nil || mctx.IsPvE {
		return domain.EngagementMatchSummary{}, false
	}
	events, _ := s.repo.LoadEventsForMatch(ctx, matchID)
	if len(events) == 0 {
		return domain.EngagementMatchSummary{}, false
	}
	teamXUIDs, _ := s.repo.LoadTeamXUIDs(ctx, matchID, mctx.TargetTeamID, s.xuid)
	input := s.buildInputForMatch(ctx, mctx, events, teamXUIDs, matchID)

	result, err := temporal.ComputeEngagementScore(input)
	if err != nil || len(result.EngagementCurve) == 0 {
		return domain.EngagementMatchSummary{}, false
	}
	return domain.EngagementMatchSummary{
		MatchID:         matchID,
		Label:           fmt.Sprintf("M%d", index+1),
		StartedAt:       time.UnixMilli(mctx.StartTimeMS),
		PaceJoueur:      meanPace(result.EngagementCurve, func(p domain.EngagementPoint) float64 { return p.PaceJoueur }),
		PaceTeam:        meanPace(result.EngagementCurve, func(p domain.EngagementPoint) float64 { return p.PaceTeam }),
		PaceAttendu:     meanPace(result.EngagementCurve, func(p domain.EngagementPoint) float64 { return p.PaceAttendu }),
		PaceLobby:       meanPace(result.EngagementCurve, func(p domain.EngagementPoint) float64 { return p.PaceLobby }),
		EngagementScore: result.EngagementScore,
	}, true
}

// computePlayerPace renvoie le pace mean d'un joueur (autre que self) sur un
// match donne. Pour Mock 15 v2 squad overlay.
func (s *PlayerEngagementService) computePlayerPace(
	ctx context.Context,
	matchID, xuid string,
) float64 {
	events, _ := s.repo.LoadEventsForMatch(ctx, matchID)
	if len(events) == 0 {
		return 0
	}
	mctx, _ := s.repo.LoadMatchEngagementContext(ctx, matchID, xuid)
	if mctx == nil {
		return 0
	}
	durationMin := float64(mctx.EndTimeMS-mctx.StartTimeMS) / 60_000.0
	if durationMin <= 0 {
		return 0
	}
	count := 0
	for _, e := range events {
		if e.XUID == xuid {
			count++
		}
	}
	return float64(count) / durationMin
}

// splitMatchEvents partitionne en player / team / lobby selon teamXUIDs explicites.
func splitMatchEvents(
	all []canonical.HighlightEvent,
	targetXUID string,
	teamXUIDs map[string]bool,
) (player, team, lobby []canonical.HighlightEvent) {
	player = make([]canonical.HighlightEvent, 0)
	team = make([]canonical.HighlightEvent, 0)
	lobby = all
	for _, e := range all {
		actor := e.XUID
		switch {
		case actor == targetXUID:
			player = append(player, e)
		case teamXUIDs[actor]:
			team = append(team, e)
		}
	}
	return player, team, lobby
}

// normalizeMode retourne PvP_ranked / PvP_unranked depuis is_ranked.
func normalizeMode(isRanked bool) string {
	if isRanked {
		return "PvP_ranked"
	}
	return "PvP_unranked"
}

// meanPace calcule la moyenne d'un champ extrait de la courbe.
func meanPace(curve []domain.EngagementPoint, getter func(domain.EngagementPoint) float64) float64 {
	if len(curve) == 0 {
		return 0
	}
	var sum float64
	for _, p := range curve {
		sum += getter(p)
	}
	return sum / float64(len(curve))
}

// ErrEngagementMatchNotFound signale un match inconnu pour le joueur cible.
var ErrEngagementMatchNotFound = errors.New("engagement: match not found for this player")

// ErrEngagementPvENotSupported signale qu'on tente de calculer sur un match PvE
// (non couvert v1, cf doc reflexion §3.4 perimetre).
var ErrEngagementPvENotSupported = errors.New("engagement: PvE not supported in v1")
