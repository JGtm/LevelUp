// Package service — home_service_page.go : GetHomePage, la page d'accueil Mission Control.
package service

import (
	"context"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/sync"
)

// applyPerfPlacementsToRecentItems renseigne PerfPlacementDone/Total sur les tuiles
// de match de l'Accueil (RecentMatchItem) dont la chaîne de PERFORMANCE est sous le
// seuil — MÊME algorithme que l'Explorer (computePerfPlacements), une seule source de
// vérité. La tuile affiche alors « En placement (X/Y) » au lieu d'un vide, JAMAIS un 0
// fabriqué pour une perf absente. Le comptage par chaîne porte sur TOUT l'historique
// (allRows) ; l'application sur les items visibles.
//
// Divergence assumée vs l'Explorer : la row canonique de l'Accueil ne porte pas
// `is_excluded` (exclusions manuelles, rares et hors placement) → l'éligibilité ne
// filtre que le DNF (miroir du COALESCE(outcome,0) != 4 de loadHistoryForPerf).
func applyPerfPlacementsToRecentItems(ctx context.Context, allRows []canonical.PlayerMatchRow, itemLists ...[]domain.RecentMatchItem) {
	if len(allRows) == 0 {
		return
	}
	slug := ctxkeys.TitleSlug(ctx)
	inputs := make([]perfPlacementInput, len(allRows))
	for i := range allRows {
		r := &allRows[i]
		pair := ""
		if r.Enrichment.PairName != nil {
			pair = *r.Enrichment.PairName
		}
		isRanked := r.Summary.IsRanked != nil && *r.Summary.IsRanked
		isFirefight := r.Summary.IsPvE != nil && *r.Summary.IsPvE
		inputs[i] = perfPlacementInput{
			matchID:  r.Summary.MatchID,
			chain:    sync.GetPerformanceChain(slug, pair, isRanked, isFirefight),
			eligible: r.Summary.Outcome != canonical.OutcomeDNF,
			hasScore: r.Enrichment.PerformanceScore != nil,
		}
	}
	placements := computePerfPlacements(inputs, sync.MinMatchesPerChainForRelative)
	if len(placements) == 0 {
		return
	}
	for _, items := range itemLists {
		for i := range items {
			dt, ok := placements[items[i].MatchID]
			if !ok {
				continue
			}
			d, t := dt[0], dt[1]
			items[i].PerfPlacementDone = &d
			items[i].PerfPlacementTotal = &t
		}
	}
}

// homeSessions regroupe les résumés de sessions de la page.
type homeSessions struct {
	solo, squad         *domain.SessionSummaryItem
	soloList, squadList []domain.SessionSummaryItem
}

// buildHomeSessions construit les résumés de sessions sur l'historique complet, puis renseigne
// les coéquipiers des sessions escouade (best-effort : alimente le deep-link card escouade ->
// /squad, no-op si non câblé).
func (s *HomeService) buildHomeSessions(ctx context.Context, rows []canonical.PlayerMatchRow, locale string, hp float64) homeSessions {
	stop := timing.FromContext(ctx).Section("home_build_sessions")
	h := homeSessions{
		solo:      analysis.BuildSessionSummaryFromCanonical(rows, false, locale, hp),
		squad:     analysis.BuildSessionSummaryFromCanonical(rows, true, locale, hp),
		soloList:  analysis.BuildSessionSummariesFromCanonical(rows, false, 20, locale, hp),
		squadList: analysis.BuildSessionSummariesFromCanonical(rows, true, 20, locale, hp),
	}
	stop()
	s.enrichSquadSessionsTeammates(ctx, h.squadList, rows)
	return h
}

// knownSessionLabels : les libellés des deux sessions résumées de la page.
func knownSessionLabels(h homeSessions) map[string]bool {
	known := map[string]bool{}
	for _, sess := range []*domain.SessionSummaryItem{h.solo, h.squad} {
		if sess != nil && sess.SessionLabel != "" {
			known[sess.SessionLabel] = true
		}
	}
	return known
}

// GetHomePage retourne la page d'accueil agrégée (hero card, highlights, tête des matchs
// récents, résumés de sessions solo et escouade).
//
// Les totaux (héros, faits marquants, sessions, séquence de la dernière session, placements)
// se calculent sur l'HISTORIQUE complet ; seule la liste de tuiles est une fenêtre de
// HomeTilesHeadSize (D-1, D-3). La suite se lit par GetHomeMatches.
func (s *HomeService) GetHomePage(ctx context.Context, gamertag, locale string) (*domain.HomePageResponse, error) {
	defer func(start time.Time) {
		observability.RecordDurationMS("home_get_page", time.Since(start).Milliseconds())
	}(time.Now())
	d, err := s.fetchHomePageData(ctx, locale)
	if err != nil {
		return nil, err
	}
	// Les sessions et les faits marquants lisent aussi les libellés : traductions sur TOUT
	// l'historique, avant tout calcul.
	s.translateAssets(ctx, d.canonicalRows)

	stop := timing.FromContext(ctx).Section("home_build_cards")
	// Baseline modèle de dégâts du titre courant (PV pour tuer) — 225 Infinite (byte-identique).
	hp := games.EffectiveHpToKill(ctxkeys.TitleSlug(ctx))
	hasRankedHistory, hasUnrankedHistory := analysis.InferHomeSkillHistoryFromCanonical(d.canonicalRows)
	hero := analysis.BuildHeroCardFromCanonical(d.canonicalRows, gamertag, d.totalMatches, locale, hp)
	highlights := analysis.BuildHighlightsFromCanonical(d.canonicalRows, locale)
	stop()
	if d.favWeaponName != "" {
		hero.KPIs.FavoriteWeaponName = d.favWeaponName
		hero.KPIs.FavoriteWeaponKills = d.favWeaponKills
	}
	sessions := s.buildHomeSessions(ctx, d.canonicalRows, locale, hp)

	ordered := analysis.OrderHomeTileRows(d.canonicalRows)
	win := analysis.WindowHomeTileRows(ordered, analysis.HomeTileCursor{}, domain.HomeTilesHeadSize)
	recent := s.buildTiles(ctx, tilesInput{
		allRows: d.canonicalRows, window: win.Rows, favoriteIDs: d.favoriteIDs,
		locale: locale, list: homeSectionListRecent, translated: true,
	})
	outcomes := lastSessionOutcomes(ordered, knownSessionLabels(sessions), s.tileOptions(ctx, locale))

	var rankCatalog *mappings.RankCatalog
	if s.semantic != nil {
		rankCatalog = s.semantic.Ranks()
	}
	// Garantit slices non-nil sur les champs JSON sans omitempty : un slice nil sérialise en JSON
	// `null` et crashe le front. Cf. testutil.RequireNoNilSlicesWithoutOmitempty.
	if highlights == nil {
		highlights = []domain.HighlightItem{}
	}
	return &domain.HomePageResponse{
		Hero:                    hero,
		SpartanIdentity:         analysis.BuildSpartanIdentity(d.spartanIdent, locale, rankCatalog),
		Highlights:              highlights,
		RecentMatches:           recent,
		RecentMatchesTotal:      len(ordered),
		RecentMatchesHasMore:    win.HasMore,
		RecentMatchesNextBefore: nextBefore(win),
		LastSessionOutcomes:     outcomes,
		SoloSession:             sessions.solo,
		SquadSession:            sessions.squad,
		SoloSessions:            sessions.soloList,
		SquadSessions:           sessions.squadList,
		HasRankedHistory:        hasRankedHistory,
		HasUnrankedHistory:      hasUnrankedHistory,
		RecentPlaylistRanks:     d.playlistRanks,
	}, nil
}
