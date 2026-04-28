// Package service — engagement_score_service.go : orchestration du calcul et
// de la persistence du score d'engagement.
//
// Reference conceptuelle : .ai/REFLEXION_ENGAGEMENT_SCORE_INTRA_MATCH.md
// Plan d'implementation : .ai/PLAN_ENGAGEMENT_IMPLEMENTATION.md
//
// Architecture (arch-rules) :
//   - Le service combine 2 repos (events + engagement persistence) + l'algo
//     pur (analysis/temporal/engagement_score). Pas d'acces SQL direct.
//   - Pas d'appel a un autre service (pas de couplage horizontal).
//   - Capability gating : ErrEngagementUnavailable si la persistence n'est
//     pas disponible (migration Phase 2 non encore appliquee). Le service
//     peut tout de meme calculer la courbe en mode "ephemere" (sans persister).
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

// HistoryWindow est le nombre maximal de matchs utilises pour la baseline
// percentile (cf doc reflexion §6.2).
const HistoryWindow = 200

// EngagementScoreService orchestre le calcul + la persistence.
//
// Implemente port.EngagementScoreService.
type EngagementScoreService struct {
	repo       port.EngagementScoreRepository
	eventsRepo port.HighlightEventsRepository
}

// NewEngagementScoreService cree un service avec ses dependances.
func NewEngagementScoreService(
	repo port.EngagementScoreRepository,
	eventsRepo port.HighlightEventsRepository,
) *EngagementScoreService {
	return &EngagementScoreService{
		repo:       repo,
		eventsRepo: eventsRepo,
	}
}

// ComputeAndPersist calcule le score d'engagement pour un match et le persiste.
//
// Si force=false et qu'un score existe deja pour (xuid, match_id), retourne
// immediatement le resultat charge sans recalcul (skip cas idempotent du sync).
// Si force=true, recalcule et ecrase.
//
// Erreurs :
//   - port.ErrEngagementUnavailable : migration Phase 2 non appliquee
//   - temporal.ErrInvalidBoundaries / ErrMatchTooShort / ErrInsufficientData :
//     cas degeneres (cf doc Phase 1.2)
//   - autres erreurs : wrappees pour debug
func (s *EngagementScoreService) ComputeAndPersist(
	ctx context.Context,
	params port.MatchEngagementParams,
	force bool,
) (*domain.EngagementScoreResult, error) {
	if !force {
		if has, err := s.repo.HasEngagementScore(ctx, params.XUID, params.MatchID); err == nil && has {
			slog.DebugContext(ctx, "EngagementScoreService.ComputeAndPersist: skip (already computed)",
				"xuid", params.XUID, "match_id", params.MatchID)
			// Score deja present : on ne recharge pas le detail (ce serait
			// un round-trip inutile pour le sync). Retourne nil pour signaler
			// "no-op". L'appelant peut interpreter (ou re-charger via un
			// futur GetMatchEngagement).
			return nil, nil
		}
	}

	result, err := s.computeForParams(ctx, params)
	if err != nil {
		return nil, err
	}

	// Persistance : score + residu + confidence
	if err := s.repo.SaveEngagementScore(ctx, params.XUID, params.MatchID, *result); err != nil {
		if errors.Is(err, port.ErrEngagementUnavailable) {
			// Degrader : retourner le resultat calcule sans erreur (utile en
			// dev avant migration). Le sync pipeline a sa propre gestion.
			slog.WarnContext(ctx, "EngagementScoreService: persistence unavailable, returning ephemeral result",
				"xuid", params.XUID, "match_id", params.MatchID)
			return result, nil
		}
		return result, fmt.Errorf("EngagementScoreService.ComputeAndPersist: save score: %w", err)
	}

	// Persistance match_intensity : 1 valeur par match. SaveMatchIntensity
	// retourne ErrEngagementUnavailable dans la repo Phase 1.5 (necessite
	// Phase 3 sync engine). Acceptable : on ne fait pas planter.
	if result.MatchIntensity > 0 {
		if err := s.repo.SaveMatchIntensity(ctx, params.MatchID, result.MatchIntensity); err != nil {
			if !errors.Is(err, port.ErrEngagementUnavailable) {
				slog.ErrorContext(ctx, "EngagementScoreService: save match intensity failed",
					"match_id", params.MatchID, "err", err)
			}
		}
	}

	return result, nil
}

// GetMatchEngagement reconstruit le score + la courbe pour un match. La courbe
// est recalculee a la volee depuis les events (cf plan §9.4 stockage hybride).
//
// Cas typique : handler Match View tab "team" charge la courbe pour affichage.
func (s *EngagementScoreService) GetMatchEngagement(
	ctx context.Context,
	params port.MatchEngagementParams,
) (*domain.EngagementScoreResult, error) {
	return s.computeForParams(ctx, params)
}

// GetEngagementProfile retourne les coefficients perso d'un joueur, par
// categorie de mode. Iteration sur les categories connues.
//
// Phase 1.6 : implementation simple qui charge categorie par categorie. Si
// la liste des categories grossit, on ajoutera un endpoint de batch dans la
// repo (LoadAllCoefficients).
func (s *EngagementScoreService) GetEngagementProfile(
	ctx context.Context,
	xuid string,
) ([]domain.EngagementCoefficient, error) {
	if xuid == "" {
		return nil, errors.New("EngagementScoreService.GetEngagementProfile: xuid required")
	}

	categories := []string{
		"PvP_ranked",
		"PvP_unranked",
		// PvE non couvert v1 (cf doc reflexion §3.4 perimetre)
	}

	out := make([]domain.EngagementCoefficient, 0, len(categories))
	for _, cat := range categories {
		coef, err := s.repo.LoadEngagementCoefficient(ctx, xuid, cat)
		if err != nil {
			if errors.Is(err, port.ErrEngagementUnavailable) {
				// Migration non appliquee : retourner profil vide proprement.
				return out, nil
			}
			return nil, fmt.Errorf("EngagementScoreService.GetEngagementProfile: %w", err)
		}
		if coef != nil {
			out = append(out, *coef)
		}
	}
	return out, nil
}

// =============================================================================
// Helper interne : compute for given params
// =============================================================================

// computeForParams charge events + history + coef et calcule le score.
//
// Pas de persistence ici (ComputeAndPersist appelle ce helper puis persiste).
// La separation permet a GetMatchEngagement de reconstruire sans round-trip
// d'ecriture.
func (s *EngagementScoreService) computeForParams(
	ctx context.Context,
	params port.MatchEngagementParams,
) (*domain.EngagementScoreResult, error) {
	// 1. Charger les events du match (joueur + lobby).
	allEvents, err := s.loadMatchEvents(ctx, params.MatchID)
	if err != nil {
		return nil, fmt.Errorf("EngagementScoreService: load events: %w", err)
	}

	// 2. Partitionner : joueur cible / coequipiers / lobby (humains).
	playerEvents, teamEvents, lobbyEvents := partitionEvents(allEvents, params.XUID)

	// 3. Charger l'historique et les coefficients perso (ignorant
	// ErrEngagementUnavailable -> degrade en insufficient_history).
	history, _ := s.loadHistorySafe(ctx, params)
	coef, _ := s.loadCoefficientSafe(ctx, params)

	// 4. Construire l'input pour l'algo pur.
	input := temporal.EngagementScoreInput{
		PlayerEvents:  playerEvents,
		TeamEvents:    teamEvents,
		LobbyEvents:   lobbyEvents,
		NTeam:         params.NTeam,
		NHumansLobby:  params.NHumansLobby,
		XUID:          params.XUID,
		MatchStartMS:  params.MatchStartMS,
		MatchEndMS:    params.MatchEndMS,
		History:       history,
		PersonalScore: params.PersonalScore,
		Kills:         params.Kills,
		Assists:       params.Assists,
		Mode:          params.ModeCategory,
		IsTeamMode:    params.IsTeamMode,
	}
	if coef != nil {
		input.CoefTeamShare = coef.CoefTeamShare
		input.CoefLobbyShare = coef.CoefLobbyShare
	} else {
		// Cold start : pas de coef stocke. Defauts neutres = "fait sa part".
		input.CoefTeamShare = 1.0
		input.CoefLobbyShare = 1.0
	}

	// 5. Calculer.
	result, err := temporal.ComputeEngagementScore(input)
	if err != nil {
		return nil, fmt.Errorf("EngagementScoreService: compute: %w", err)
	}
	return &result, nil
}

// loadMatchEvents charge tous les events du match via le repo HighlightEvents.
//
// Note: la signature port.HighlightEventsRepository.LoadHighlightEvents
// requiert un slug. Le service amont (handler ou sync) doit disposer du slug
// dans le ctx ; pour Phase 1.6 simple, on lit slug depuis ctx via une helper
// (a wirer en Phase 3).
func (s *EngagementScoreService) loadMatchEvents(
	ctx context.Context,
	matchID string,
) ([]canonical.HighlightEvent, error) {
	slug := titleSlugFromContext(ctx)

	filter := port.HighlightEventFilters{
		MatchIDs: []string{matchID},
	}
	events, err := s.eventsRepo.LoadHighlightEvents(ctx, slug, filter)
	if err != nil {
		return nil, err
	}
	return events, nil
}

// loadHistorySafe wrappe LoadPlayerHistory en degradant gracieusement sur
// ErrEngagementUnavailable. Retourne une slice vide dans ce cas (-> le
// service produira "insufficient_history").
func (s *EngagementScoreService) loadHistorySafe(
	ctx context.Context,
	params port.MatchEngagementParams,
) ([]domain.HistoricalEngagementBrut, error) {
	filter := port.EngagementHistoryFilter{
		XUID:           params.XUID,
		ModeCategory:   params.ModeCategory,
		Limit:          HistoryWindow,
		ExcludeMatchID: params.MatchID,
	}
	history, err := s.repo.LoadPlayerHistory(ctx, filter)
	if err != nil {
		if errors.Is(err, port.ErrEngagementUnavailable) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "EngagementScoreService: history load failed",
			"xuid", params.XUID, "match_id", params.MatchID, "err", err)
		return nil, err
	}
	return history, nil
}

// loadCoefficientSafe wrappe LoadEngagementCoefficient en degradant
// gracieusement (cold start ou migration non appliquee).
func (s *EngagementScoreService) loadCoefficientSafe(
	ctx context.Context,
	params port.MatchEngagementParams,
) (*domain.EngagementCoefficient, error) {
	coef, err := s.repo.LoadEngagementCoefficient(ctx, params.XUID, params.ModeCategory)
	if err != nil {
		if errors.Is(err, port.ErrEngagementUnavailable) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "EngagementScoreService: coef load failed",
			"xuid", params.XUID, "err", err)
		return nil, err
	}
	return coef, nil
}

// =============================================================================
// Helpers prives
// =============================================================================

// partitionEvents separe les events en 3 slices : joueur cible, coequipiers,
// lobby complet (incluant ennemis, humains uniquement, bots filtres en amont
// par le repo HighlightEvents si la convention est respectee).
//
// Ici on ne dispose pas du teamID des joueurs (la struct HighlightEvent
// canonique ne porte pas TeamID directement). Phase 1.6 simple : on suppose
// que la separation team/lobby est faite par le caller (qui fournit
// MatchEngagementParams.NTeam et NHumansLobby) en amont. Cette partition
// retourne donc :
//   - playerEvents : events dont KillerXUID/VictimXUID/PlayerXUID == XUID
//   - lobbyEvents  : tous les events (= joueur + autres = lobby)
//   - teamEvents   : sous-ensemble des events des coequipiers
//
// Pour distinguer team de lobby, le caller doit fournir une projection des
// teamXUIDs (TODO Phase 3 quand on integre le sync). Phase 1.6 retourne :
// teamEvents = lobbyEvents \ playerEvents (approximation : tous les autres
// joueurs sont consideres comme coequipiers, ce qui est faux des qu'il y a
// une equipe ennemie). A raffiner en Phase 3 avec un parametre TeamXUIDs.
func partitionEvents(
	all []canonical.HighlightEvent,
	xuid string,
) (player, team, lobby []canonical.HighlightEvent) {
	player = make([]canonical.HighlightEvent, 0)
	team = make([]canonical.HighlightEvent, 0)
	lobby = all
	for _, e := range all {
		if eventActorXUID(e) == xuid {
			player = append(player, e)
		} else {
			team = append(team, e)
		}
	}
	return player, team, lobby
}

// eventActorXUID extrait le XUID acteur d'un event selon son type.
func eventActorXUID(e canonical.HighlightEvent) string {
	if e.KillerXUID != nil {
		return *e.KillerXUID
	}
	if e.VictimXUID != nil {
		return *e.VictimXUID
	}
	if e.PlayerXUID != nil {
		return *e.PlayerXUID
	}
	return e.XUID // legacy fallback
}

// titleSlugFromContext extrait le slug depuis le contexte (middleware
// TitleExtractor), avec fallback "halo_infinite".
//
// Phase 1.6 : implementation defensive. En production, ctxkeys.TitleSlug(ctx)
// est preferable mais cree une dependance circulaire potentielle ; on inline
// un lookup sur la cle connue.
func titleSlugFromContext(ctx context.Context) string {
	type ctxKey string
	const titleSlugKey ctxKey = "title_slug"

	if v := ctx.Value(titleSlugKey); v != nil {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return "halo_infinite"
}

// staleThreshold definit la duree au-dela de laquelle un coefficient est
// considere perime et merite un recalcul. Reference plan §4.4. Non utilise
// directement en Phase 1.6 mais expose pour les futurs schedulers.
var staleThreshold = 30 * 24 * time.Hour

// IsCoefficientStale retourne true si LastUpdated est trop ancien.
func IsCoefficientStale(coef domain.EngagementCoefficient) bool {
	return time.Since(coef.LastUpdated) > staleThreshold
}

// =============================================================================
// PlayerEngagementService — service per-player pour les handlers Phase 4
// =============================================================================

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
//
// Erreurs :
//   - port.ErrEngagementUnavailable : migration Phase 2 non appliquee
//   - apiNotFound (sentinel locale) : match introuvable / joueur absent du match
//   - autres : wrappees pour debug
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

	playerEvents, teamEvents, lobbyEvents := splitMatchEvents(events, s.xuid, teamXUIDs)

	modeCategory := normalizeMode(mctx.IsRanked)
	history, _ := s.loadHistorySafeByMode(ctx, modeCategory, matchID)
	coefTeam, coefLobby := s.loadCoefsSafe(ctx, modeCategory)

	input := temporal.EngagementScoreInput{
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

// ErrEngagementMatchNotFound signale un match inconnu pour le joueur cible.
var ErrEngagementMatchNotFound = errors.New("engagement: match not found for this player")

// ErrEngagementPvENotSupported signale qu'on tente de calculer sur un match PvE
// (non couvert v1, cf doc reflexion §3.4 perimetre).
var ErrEngagementPvENotSupported = errors.New("engagement: PvE not supported in v1")
