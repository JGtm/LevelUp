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
	"sort"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
)

// HistoryWindow est le nombre maximal de matchs utilises pour la baseline
// percentile (cf doc reflexion §6.2 et plan engagement §4.4).
const HistoryWindow = 200

// engagementWorkCap borne le nombre de matchs sur lesquels on lance le compute
// par-match (LoadMatchEngagementContext + LoadEventsForMatch + LoadTeamXUIDs +
// ComputeEngagementScore -> ~10-30ms par match). Au-dela, on garde les N plus
// recents et on positionne `TruncatedToRecent` dans la reponse. La cascade de
// binning (session/week/month) tourne ensuite sur ce sous-ensemble.
const engagementWorkCap = 200

// PlayerEngagementService est un wrapper avec xuid baked-in destine aux
// handlers HTTP. Charge metadata + events + history + coefs via le repo, puis
// appelle l'algo pur. Pas d'acces SQL direct, pas d'appel cross-service.
type PlayerEngagementService struct {
	repo     port.EngagementScoreRepository
	xuid     string
	gamertag string

	// playerMatchesRepo (optionnel) permet a GetTimeseries de resoudre la
	// liste de matchs via le pipeline canonical + filterStatsMatchRows, donc
	// d'honorer le FilterContextInput de la page Timeseries (period, cascade,
	// sessions, match_context). Si nil, GetTimeseries retombe sur le fast
	// path SQL via le repo (ListRecentPvPMatchIDs) sans filtres metier.
	playerMatchesRepo port.PlayerMatchesRepository
	titleSlug         string

	// engagementCap est le statut de la capability fine engagement.score du titre
	// (2e porte de degradation F7). Vide = titre historique valide (Infinite) traite
	// comme supported/validated. degraded → score servi AVEC calibration=provisional ;
	// not_exposed → GetMatchEngagement retourne ErrCapabilityNotSupported (503 propre).
	engagementCap games.CapabilityStatus

	// expectedMemo cache coef lobby + bins par mode_category le temps de vie du
	// service (E1 revue 2026-07). coef+bins ne dependent que de (xuid,
	// mode_category), et xuid est fige sur le service : dans GetTimeseries
	// (jusqu'a engagementWorkCap matchs) mode_category ne prend que 2 valeurs
	// (PvP_ranked / PvP_unranked) → au plus 4 lectures DB au lieu de ~3 par match
	// (~600 pour 200 matchs). Non concurrent : le service est cree par requete
	// (registry_pages.Engagement) et computeSummariesChronoAsc itere en sequentiel.
	expectedMemo map[string]expectedInputsEntry
}

// expectedInputsEntry memoize le resultat de loadExpectedInputs pour une
// mode_category (coef lobby global + flag de disponibilite + bins de reponse).
type expectedInputsEntry struct {
	coefLobby float64
	hasGlobal bool
	bins      *domain.EngagementResponseBins
}

// NewPlayerEngagementService cree un service per-player.
func NewPlayerEngagementService(repo port.EngagementScoreRepository, xuid, gamertag string) *PlayerEngagementService {
	return &PlayerEngagementService{repo: repo, xuid: xuid, gamertag: gamertag}
}

// WithPlayerMatchesRepo cable le loader canonical pour permettre a
// GetTimeseries de filtrer la liste de matchs via FilterContextInput.
func (s *PlayerEngagementService) WithPlayerMatchesRepo(
	repo port.PlayerMatchesRepository, titleSlug string,
) *PlayerEngagementService {
	s.playerMatchesRepo = repo
	s.titleSlug = titleSlug
	return s
}

// WithEngagementCapability injecte le statut de la capability fine engagement.score
// du titre (2e porte de degradation F7). Resolu par la factory title-aware ; le
// service reste agnostic (il porte un statut, pas la logique de capability).
func (s *PlayerEngagementService) WithEngagementCapability(status games.CapabilityStatus) *PlayerEngagementService {
	s.engagementCap = status
	return s
}

// calibrationForStatus mappe le statut de capability fine vers la valeur de
// calibration exposee par match (2e porte F7). Vide/supported → validated ;
// degraded → provisional. not_exposed est gate en amont (ErrCapabilityNotSupported).
func calibrationForStatus(status games.CapabilityStatus) string {
	if status == games.CapDegraded {
		return domain.CalibrationProvisional
	}
	return domain.CalibrationValidated
}

// GetMatchEngagement charge le contexte du match, recompute la courbe et le
// score (live), et retourne le resultat. Utilise par GET /matches/{id}/engagement.
func (s *PlayerEngagementService) GetMatchEngagement(
	ctx context.Context,
	matchID string,
) (*domain.EngagementScoreResult, error) {
	// Porte statique (1re moitie de la double porte F7) : un titre qui n'expose PAS
	// engagement.score (not_exposed) ne sert pas de score — degradation gracieuse
	// 503 propre (jamais un score cold-start silencieusement faux). Vide/degraded/
	// supported passent (degraded servira avec calibration=provisional plus bas).
	if s.engagementCap == games.CapNotExposed {
		return nil, fmt.Errorf("engagement.score: %w", games.ErrCapabilityNotSupported)
	}

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
		// Indisponibilite PAR MATCH (trop court / pas d'event exploitable pour le
		// joueur+coequipiers) -> sentinelle dediee mappee en 422 cote handler, et
		// non un 500 generique affiche "migration en cours" cote front.
		if errors.Is(err, temporal.ErrMatchTooShort) || errors.Is(err, temporal.ErrInsufficientData) {
			// Log diagnostic : compteurs d'events par segment pour pister la cause
			// racine (attribution xuid/team_id) quand un match parait pourtant peuple.
			slog.WarnContext(ctx, "engagement: courbe non calculable pour ce match",
				"match_id", matchID, "xuid", s.xuid,
				"n_player_events", len(input.PlayerEvents),
				"n_team_events", len(input.TeamEvents),
				"n_lobby_events", len(input.LobbyEvents),
				"n_team", mctx.NTeam, "target_team_id", mctx.TargetTeamID,
				"reason", err)
			return nil, ErrEngagementInsufficient
		}
		return nil, fmt.Errorf("PlayerEngagementService: compute: %w", err)
	}
	// 2e porte F7 : statut de calibration du titre (provisional si degraded).
	result.Calibration = calibrationForStatus(s.engagementCap)
	return &result, nil
}

// GetEngagementProfile retourne le profil engagement du joueur par categorie de
// mode : coef lobby global + bins de reponse par intensite (modele lobby-anchored
// v2). coef_team_share n'est plus expose (D5).
func (s *PlayerEngagementService) GetEngagementProfile(
	ctx context.Context,
) ([]domain.EngagementProfile, error) {
	if s.xuid == "" {
		return nil, errors.New("PlayerEngagementService.GetEngagementProfile: xuid required")
	}
	coefs, err := s.repo.LoadAllCoefficients(ctx, s.xuid)
	if err != nil {
		if errors.Is(err, port.ErrEngagementUnavailable) {
			return []domain.EngagementProfile{}, nil
		}
		return nil, fmt.Errorf("PlayerEngagementService.GetEngagementProfile: %w", err)
	}
	profiles := make([]domain.EngagementProfile, 0, len(coefs))
	for _, c := range coefs {
		p := domain.EngagementProfile{
			XUID:           c.XUID,
			Gamertag:       s.gamertag,
			ModeCategory:   c.ModeCategory,
			CoefLobbyShare: c.CoefLobbyShare,
			NMatches:       c.NMatches,
			LastUpdated:    c.LastUpdated,
		}
		// Bins best-effort : table absente (migration non appliquee) → profil
		// sans bins, jamais d'echec (degradation gracieuse).
		if bins, berr := s.repo.LoadResponseBins(ctx, s.xuid, c.ModeCategory); berr == nil && bins != nil {
			p.Bins = bins.Bins
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

// GetTimeseries retourne la serie temporelle d'engagement du joueur (Mock 11)
// avec granularite adaptative selon la densite filtree :
//
//   - len(filtered) <= limit                 -> granularity = "match"   (1 point = 1 match)
//   - sinon, len(by_session) <= limit        -> granularity = "session"
//   - sinon, len(by_week) <= limit           -> granularity = "week"
//   - sinon                                  -> granularity = "month"
//
// Le compute par-match (3 queries DB + ComputeEngagementScore) est borne a
// engagementWorkCap matchs les plus recents ; TruncatedToRecent est positionne
// dans la reponse si l'ensemble filtre depasse ce plafond.
//
// Si playerMatchesRepo n'est pas cable (mocks de test), retombe sur
// ListRecentPvPMatchIDs et force granularity = "match" (pas de binning sans
// row data complete).
//
// Reference plan : §6.6.3.
func (s *PlayerEngagementService) GetTimeseries(
	ctx context.Context,
	filters domain.FilterContextInput,
	limit int,
) (*domain.EngagementTimeseriesResponse, error) {
	if s.xuid == "" {
		return nil, errors.New("PlayerEngagementService.GetTimeseries: xuid required")
	}
	if limit <= 0 {
		limit = 50
	}

	rows, fallbackIDs, err := s.resolveFilteredRowsDesc(ctx, filters)
	if err != nil {
		if errors.Is(err, port.ErrEngagementUnavailable) {
			// Dégradation gracieuse (200 série vide) : rendre observable au lieu de
			// la masquer — un titre/joueur sans colonnes engagement ne doit pas être
			// un trou noir silencieux dans les logs.
			slog.WarnContext(ctx, "engagement timeseries indisponible (migration/colonnes absentes) → série vide",
				"xuid", s.xuid)
			return emptyEngagementTimeseries(), nil
		}
		return nil, err
	}

	total := len(rows)
	if rows == nil {
		total = len(fallbackIDs)
	}

	var truncatedTo *int
	if len(rows) > engagementWorkCap {
		n := engagementWorkCap
		truncatedTo = &n
		rows = rows[:engagementWorkCap]
	}
	if len(fallbackIDs) > engagementWorkCap {
		n := engagementWorkCap
		truncatedTo = &n
		fallbackIDs = fallbackIDs[:engagementWorkCap]
	}

	slog.DebugContext(ctx, "PlayerEngagementService.GetTimeseries: computing",
		"xuid", s.xuid, "limit", limit, "total_matches", total, "compute_n",
		len(rows)+len(fallbackIDs))

	summaries := s.computeSummariesChronoAsc(ctx, rows, fallbackIDs)

	// Cascade de granularite. Le binning n'a de sens qu'avec les `rows`
	// (besoin de session_label + StartTime). En fallback on reste sur "match".
	granularity := domain.EngagementGranularityMatch
	points := summaries
	if rows != nil && len(summaries) > limit {
		if sessionPts := aggregateEngagementBySession(summaries, rows); len(sessionPts) <= limit {
			granularity = domain.EngagementGranularitySession
			points = sessionPts
		} else if weekPts := rollupEngagementByPeriod(summaries, "week"); len(weekPts) <= limit {
			granularity = domain.EngagementGranularityWeek
			points = weekPts
		} else {
			granularity = domain.EngagementGranularityMonth
			points = rollupEngagementByPeriod(summaries, "month")
		}
	}

	slog.InfoContext(ctx, "PlayerEngagementService.GetTimeseries: done",
		"xuid", s.xuid, "granularity", granularity, "n_points", len(points),
		"total_matches", total)
	return &domain.EngagementTimeseriesResponse{
		Granularity:       granularity,
		Points:            points,
		TotalMatches:      total,
		TruncatedToRecent: truncatedTo,
	}, nil
}

// computeSummariesChronoAsc lance computeMatchSummary sur chaque match (rows
// ou fallback) en ordre chronologique croissant pour stabiliser les labels
// "M1, M2, ...". Les paces et MatchCount=1 sont prets pour le binning aval.
func (s *PlayerEngagementService) computeSummariesChronoAsc(
	ctx context.Context,
	rows []legacymatch.StatsMatchRow,
	fallbackIDs []string,
) []domain.EngagementMatchSummary {
	n := len(rows)
	if rows == nil {
		n = len(fallbackIDs)
	}
	out := make([]domain.EngagementMatchSummary, 0, n)
	// Iteration DESC -> chronologique ASC pour les labels.
	for i := n - 1; i >= 0; i-- {
		var matchID string
		if rows != nil {
			matchID = rows[i].MatchID
		} else {
			matchID = fallbackIDs[i]
		}
		summary, ok := s.computeMatchSummary(ctx, matchID, n-1-i)
		if !ok {
			continue
		}
		summary.MatchCount = 1
		out = append(out, summary)
	}
	return out
}

// emptyEngagementTimeseries renvoie une reponse vide bien typee — utilisee
// quand la migration engagement n'est pas appliquee (ErrEngagementUnavailable).
func emptyEngagementTimeseries() *domain.EngagementTimeseriesResponse {
	return &domain.EngagementTimeseriesResponse{
		Granularity: domain.EngagementGranularityMatch,
		Points:      []domain.EngagementMatchSummary{},
	}
}

// (GetSquadSession + matchBundle + computeTeammateMeanPace : voir
// engagement_squad_service.go.
// RecomputeCoefficients + RecomputeReport : voir engagement_admin_service.go.)

// =============================================================================
// Helpers prives partages
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
	coefLobby, hasGlobal, bins := s.loadExpectedInputs(ctx, modeCategory)
	// highlight_events.time_ms est relatif au debut du match (0 a durationMS),
	// pas un epoch UTC. On normalise donc les bornes a [0, duration].
	durationMS := mctx.EndTimeMS - mctx.StartTimeMS
	return temporal.EngagementScoreInput{
		PlayerEvents:       playerEvents,
		TeamEvents:         teamEvents,
		LobbyEvents:        lobbyEvents,
		NTeam:              mctx.NTeam,
		NHumansLobby:       mctx.NHumansLobby,
		XUID:               s.xuid,
		MatchStartMS:       0,
		MatchEndMS:         durationMS,
		History:            history,
		CoefLobbyShare:     coefLobby,
		HasGlobalLobbyCoef: hasGlobal,
		ResponseBins:       bins,
		PersonalScore:      mctx.PersonalScore,
		Kills:              mctx.Kills,
		Assists:            mctx.Assists,
		Mode:               modeCategory,
		IsTeamMode:         mctx.IsTeamMode,
		// Vecteur de signaux (masque de presence + signaux riches), derive de la
		// composition des events deja partitionnes (title-agnostic, la richesse
		// vient de ce que le titre a projete en amont dans highlight_events).
		Signals: temporal.SignalsFromEvents(playerEvents, lobbyEvents, durationMS),
		// Poids d'events PAR TITRE (F7) : constants.toml [engagement], defaut
		// byte-identique Infinite si non declare.
		Weights: games.EngagementWeightsFor(s.titleSlug),
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

// loadExpectedInputs charge les entrees de l'attendu ancre lobby : coef lobby
// global (fallback), flag de disponibilite du global, et bins de reponse.
//
// hasGlobal = (une row engagement_coefficients existe) : le recompute n'en
// persiste une que si >= MinMatchesForCoef samples valides, donc sa presence
// garantit un coef lobby global reel (pas le defaut cold-start 1.0). Degradation
// gracieuse : table/colonnes absentes → cold-start (coefLobby 1.0, bins nil).
func (s *PlayerEngagementService) loadExpectedInputs(
	ctx context.Context,
	modeCategory string,
) (coefLobby float64, hasGlobal bool, bins *domain.EngagementResponseBins) {
	// E1 : memoisation par mode_category. Lecture sur map nil = zero-value + ok
	// false (sur — pas de panic). Correct pour tout xuid fige : coef+bins ne
	// changent pas sous le service.
	if e, ok := s.expectedMemo[modeCategory]; ok {
		return e.coefLobby, e.hasGlobal, e.bins
	}
	coefLobby = 1.0
	if coef, err := s.repo.LoadEngagementCoefficient(ctx, s.xuid, modeCategory); err == nil && coef != nil {
		coefLobby = coef.CoefLobbyShare
		hasGlobal = true
	}
	if b, err := s.repo.LoadResponseBins(ctx, s.xuid, modeCategory); err == nil {
		bins = b
	}
	if s.expectedMemo == nil {
		s.expectedMemo = make(map[string]expectedInputsEntry, 2)
	}
	s.expectedMemo[modeCategory] = expectedInputsEntry{coefLobby: coefLobby, hasGlobal: hasGlobal, bins: bins}
	return coefLobby, hasGlobal, bins
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

// resolveFilteredRowsDesc retourne les matchs PvP du scope filtre, ordre
// chronologique decroissant (latest first). Le binning aval a besoin du
// StatsMatchRow complet (session_label, start_time).
//
//   - playerMatchesRepo cable + titleSlug/gamertag fournis : pipeline canonical
//   - filterStatsMatchRows (honore FilterContextInput).
//   - sinon : fast path SQL via ListRecentPvPMatchIDs ; rows == nil, fallbackIDs
//     porte uniquement les match_ids. Binning desactive en aval (granularity
//     reste "match").
func (s *PlayerEngagementService) resolveFilteredRowsDesc(
	ctx context.Context,
	filters domain.FilterContextInput,
) (rows []legacymatch.StatsMatchRow, fallbackIDs []string, err error) {
	if s.playerMatchesRepo == nil || s.titleSlug == "" || s.gamertag == "" {
		ids, err := s.loadRecentPvPMatchIDs(ctx, 1000)
		if err != nil {
			return nil, nil, err
		}
		// ListRecentPvPMatchIDs renvoie en ordre chronologique croissant ; on
		// inverse pour latest-first (cohérent avec rows desc).
		out := make([]string, len(ids))
		for i, id := range ids {
			out[len(ids)-1-i] = id
		}
		return nil, out, nil
	}
	canonicalRows, err := s.playerMatchesRepo.LoadPlayerMatches(
		ctx, s.titleSlug, s.gamertag, port.PlayerMatchFilters{},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("PlayerEngagementService.resolveFilteredRowsDesc: %w", err)
	}
	allRows := analysis.StatsMatchRowsFromCanonical(canonicalRows, games.EffectiveHpToKill(s.titleSlug))
	filtered := filterStatsMatchRows(allRows, filters)
	pvp := make([]legacymatch.StatsMatchRow, 0, len(filtered))
	for _, m := range filtered {
		if m.IsFirefight {
			continue
		}
		pvp = append(pvp, m)
	}
	sort.SliceStable(pvp, func(i, j int) bool {
		return pvp[i].StartTime.After(pvp[j].StartTime)
	})
	return pvp, nil, nil
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
		MapName:         mctx.MapName,
		StartedAt:       time.UnixMilli(mctx.StartTimeMS),
		PaceJoueur:      meanPace(result.EngagementCurve, func(p domain.EngagementPoint) float64 { return p.PaceJoueur }),
		PaceTeam:        meanPace(result.EngagementCurve, func(p domain.EngagementPoint) float64 { return p.PaceTeam }),
		PaceAttendu:     meanPace(result.EngagementCurve, func(p domain.EngagementPoint) float64 { return p.PaceAttendu }),
		PaceLobby:       meanPace(result.EngagementCurve, func(p domain.EngagementPoint) float64 { return p.PaceLobby }),
		EngagementScore: result.EngagementScore,
	}, true
}

// (computePlayerPace : supprime en Phase 7 plan engagement long-term — la
// methode count/duration etait incoherente avec PaceJoueur du main player
// (curve smoothing 90s) et faisait N×M lookups DB inutiles. Remplace par
// computeTeammateMeanPace qui reutilise les events deja en memoire et
// applique la meme methode de calcul que le main player.)

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

// ErrEngagementInsufficient signale que la courbe n'a pas pu etre calculee pour
// ce match precis (match trop court, ou aucun event exploitable pour le joueur
// et ses coequipiers). Distinct de ErrEngagementUnavailable (migration absente) :
// c'est une indisponibilite PAR MATCH, pas un probleme de schema -> 422, pas 503.
// Le front affiche un message neutre ("indisponible pour ce match"), jamais
// "migration en cours".
var ErrEngagementInsufficient = errors.New("engagement: insufficient data for this match")
