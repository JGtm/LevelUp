// Package wire — registry_trends.go : fabrique du service de la page Tendances.
package wire

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
	"levelup/go-api/internal/service/teammates"
	"levelup/go-api/internal/sync"
)

// titleHasCapability indique si le titre déclare la capacité de titre c (jamais
// slug== — ratchet no_slug_comparison_test.go).
func (r *ServiceRegistry) titleHasCapability(slug string, c title.Capability) bool {
	d := title.DefaultRegistry().Get(slug)
	return d != nil && d.HasCapability(c)
}

// trendsLocation retourne le fuseau de l'utilisateur, UTC s'il est invalide.
func (r *ServiceRegistry) trendsLocation(ctx context.Context) *time.Location {
	loc, err := time.LoadLocation(r.cfg.UserTimezone)
	if err != nil {
		slog.WarnContext(ctx, "trends: fuseau utilisateur invalide, repli sur UTC",
			"timezone", r.cfg.UserTimezone, "err", err)
		return time.UTC
	}
	return loc
}

// trendsCapabilities : les capacités publiées viennent des déclarations du titre et de
// son adapter, jamais du slug. Communes aux vues Solo et Escouade.
func (r *ServiceRegistry) trendsCapabilities(pdb *duckdb.PlayerDB, caps games.CapabilityMap) domain.TrendsCapabilities {
	return domain.TrendsCapabilities{
		MMR:        r.titleHasCapability(pdb.TitleSlug, title.CapTeamMMR),
		LUSR:       r.titleHasCapability(pdb.TitleSlug, title.CapLUSR),
		CSR:        r.titleHasCapability(pdb.TitleSlug, title.CapRanked) && caps.Has(games.CapMatchSkillSnapshot),
		Objectives: caps.Has(games.CapMatchObjectiveStats),
		Equipment:  caps.Has(games.CapFilmUsageSummary),
	}
}

// Trends retourne un TrendsService pour le joueur (vue Solo). Le fuseau est celui de
// l'utilisateur (UTC s'il est invalide).
func (r *ServiceRegistry) Trends(ctx context.Context, slug string) (port.TrendsService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	caps := r.capabilitiesForPDB(pdb)
	svc := service.NewTrendsService(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag).
		WithLocation(r.trendsLocation(ctx)).
		WithPlayerXUID(pdb.XUID).
		WithMedals(duckdb.NewMedalsByXUIDRepo(pdb), duckdb.NewMedalDefinitionsRepo(pdb)).
		WithCapabilities(r.trendsCapabilities(pdb, caps))
	if caps.Has(games.CapMatchObjectiveStats) {
		svc.WithObjectives(duckdb.NewObjectiveStatsRepo(pdb), duckdb.NewSessionUsageRepo(pdb))
	}
	if caps.Has(games.CapFilmUsageSummary) {
		svc.WithEquipmentUsage(duckdb.NewSessionUsageRepo(pdb))
	}
	return svc, nil
}

// SquadTrendsCtx retourne le service de la vue Escouade, le xuid et le gamertag du
// joueur principal : le câblage de la page Escouade, plus les dépendances des tendances.
func (r *ServiceRegistry) SquadTrendsCtx(ctx context.Context, slug string) (port.SquadTrendsService, string, string, error) {
	// TeammatesCtx reste l'unique câblage du service (le garde-rail
	// registry_pages_home_teammates_wiring_test.go lit son corps) ; le type concret est
	// celui qu'il construit.
	base, xuid, gamertag, err := r.TeammatesCtx(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc, ok := base.(*teammates.TeammatesService)
	if !ok {
		return nil, "", "", fmt.Errorf("SquadTrendsCtx: TeammatesCtx rend %T, attendu *teammates.TeammatesService", base)
	}
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc.WithTrends(teammates.TrendsDeps{
		Loc: r.trendsLocation(ctx),
		Now: time.Now,
		ChainOf: func(pairName string, isRanked, isPvE bool) string {
			return sync.GetPerformanceChain(pdb.TitleSlug, pairName, isRanked, isPvE)
		},
		HpToKill:     games.EffectiveHpToKill(pdb.TitleSlug),
		Capabilities: r.trendsCapabilities(pdb, r.capabilitiesForPDB(pdb)),
	})
	return svc, xuid, gamertag, nil
}
