// Package service - TrendsService : endpoint POST /pages/trends (page Tendances).
//
// Le service charge les matchs du joueur (même lecture en cache que la page
// Séries temporelles), les aplatit, retient le contexte demandé et délègue tout
// le calcul au paquet pur analysis/trends.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis/trends"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync"
)

// TrendsService construit la page Tendances.
type TrendsService struct {
	matches   port.PlayerMatchesRepository
	titleSlug string
	gamertag  string
	loc       *time.Location
	now       func() time.Time
	caps      domain.TrendsCapabilities

	// Sources optionnelles (voir trends_service_sources.go) ; nil = bloc absent.
	playerXUID     string
	objectiveRoles objectiveRoleRowsLoader
	participants   port.SessionUsageRepository
	equipment      port.SessionUsageRepository
	medals         port.MedalsByXUIDRepository
	medalDefs      port.MedalDefinitionsRepository
}

// NewTrendsService crée un TrendsService (fuseau UTC, horloge système, aucune capacité).
func NewTrendsService(matches port.PlayerMatchesRepository, titleSlug, gamertag string) *TrendsService {
	return &TrendsService{
		matches:   matches,
		titleSlug: titleSlug,
		gamertag:  gamertag,
		loc:       time.UTC,
		now:       time.Now,
	}
}

// WithLocation fixe le fuseau des jours, semaines et mois ; nil garde UTC.
func (s *TrendsService) WithLocation(loc *time.Location) *TrendsService {
	if loc != nil {
		s.loc = loc
	}
	return s
}

// WithClock injecte l'horloge ; nil garde l'horloge système.
func (s *TrendsService) WithClock(now func() time.Time) *TrendsService {
	if now != nil {
		s.now = now
	}
	return s
}

// WithCapabilities fixe les capacités publiées dans la réponse.
func (s *TrendsService) WithCapabilities(c domain.TrendsCapabilities) *TrendsService {
	s.caps = c
	return s
}

// GetPage construit la page. Toute erreur de chargement est propagée enveloppée
// (games.ErrCapabilityNotSupported reste détectable par errors.Is).
func (s *TrendsService) GetPage(ctx context.Context, req domain.TrendsQueryRequest) (domain.TrendsPageResponse, error) {
	stop := timing.FromContext(ctx).Section("player_matches")
	rows, err := s.matches.LoadPlayerMatches(ctx, s.titleSlug, s.gamertag, port.PlayerMatchFilters{})
	stop()
	if err != nil {
		return domain.TrendsPageResponse{}, fmt.Errorf("TrendsService: chargement des matchs : %w", err)
	}

	all := trends.FromCanonical(rows, trends.FromOptions{Loc: s.loc, ChainOf: s.chainOf})
	solo := make([]trends.Match, 0, len(all))
	for i := range all {
		if !all[i].IsWithFriends {
			solo = append(solo, all[i])
		}
	}
	slog.DebugContext(ctx, "trends: matchs chargés",
		"titleSlug", s.titleSlug, "player", s.gamertag, "matches", len(rows), "retained", len(solo))

	now := s.now()
	src := s.readSources(ctx, trends.MatchIDsSince(solo, now.Add(-trendsSampleDays*24*time.Hour)), req.Locale)
	trends.Attach(solo, src.objectives, src.equipment)

	stop = timing.FromContext(ctx).Section("trends_build")
	resp := trends.BuildSolo(solo, trends.Options{
		Now:        now,
		Loc:        s.loc,
		GameType:   req.GameType,
		HpToKill:   games.EffectiveHpToKill(s.titleSlug),
		Medals:     src.medals,
		MedalNames: src.medalNames,
	})
	stop()

	resp.View = domain.TrendsViewSolo
	resp.Capabilities = s.caps
	if resp.Medals == nil {
		resp.Medals = []domain.TrendsMedalsBlock{}
	}
	return resp, nil
}

// chainOf classe un match en chaîne de performance (type de partie).
func (s *TrendsService) chainOf(pairName string, isRanked, isPvE bool) string {
	return sync.GetPerformanceChain(s.titleSlug, pairName, isRanked, isPvE)
}
