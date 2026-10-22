// Package service — match_view_replay_context.go : le contexte de match de la page de rejeu.
//
// La page de rejeu lit de la base quatre morceaux de la vue du match : l'en-tête, les lignes du
// tableau de score, les événements du fil (kills ET médailles) et les médias. Cette lecture ne
// charge que leurs sources (liste fermée : loadReplayContextDataParallel)
// et les assemble par les fonctions partagées avec la vue complète
// (match_view_shared_build.go) et les mêmes builders : rencontres, historique, citations,
// Emprise, Isolement, radar, attendus et extras des amis ne sont ni lus ni calculés.
package service

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/observability/timing"

	"golang.org/x/sync/errgroup"
)

// GetReplayContext retourne le contexte de match de la page de rejeu.
//
// Même préalable que GetMatchView : métadonnées bloquantes (absence → 404, panne → 500) puis
// garde de participation, avant tout chargement. Chaque morceau rendu est égal à son homologue
// de GetMatchView, à l'exception des champs de ligne que seuls les chargements omis alimentent
// (cf. domain.MatchReplayContext).
func (s *MatchViewService) GetReplayContext(ctx context.Context, matchID string) (domain.MatchReplayContext, error) {
	defer func(start time.Time) {
		observability.RecordDurationMS("match_replay_context_get", time.Since(start).Milliseconds())
	}(time.Now())
	stop := timing.FromContext(ctx).Section("replay_context_meta")
	meta, err := s.loadMatchMeta(ctx, matchID)
	stop()
	if err != nil {
		return domain.MatchReplayContext{}, err
	}
	if err := s.assertParticipation(ctx, matchID); err != nil {
		return domain.MatchReplayContext{}, err
	}
	d, err := s.loadReplayContextDataParallel(ctx, matchID)
	if err != nil {
		return domain.MatchReplayContext{}, err
	}
	defer timing.FromContext(ctx).Section("replay_context_assemble")()
	return s.buildReplayContextFromData(ctx, matchID, meta, d), nil
}

// loadReplayContextDataParallel lance en parallèle les chargements dont dépend un des quatre
// morceaux, et eux seuls. Mêmes chargeurs et même traitement des échecs que
// loadMatchViewDataParallel : une lecture en échec est journalisée et retenue, jamais fatale.
//
//nolint:funlen // routage errgroup linéaire, une entrée par source de la liste fermée D4.
func (s *MatchViewService) loadReplayContextDataParallel(ctx context.Context, matchID string) (matchViewData, error) {
	d := matchViewData{failures: newSourceFailures()}
	g, gctx := errgroup.WithContext(ctx)

	d.failures.goLoad(gctx, g, matchID, srcPlayerStats, func() error {
		var e error
		d.stats, e = s.repo.GetPlayerMatchStats(gctx, s.xuid, matchID)
		return e
	})
	d.failures.goLoadSection(gctx, g, matchID, "enrichment", func() error {
		var e error
		d.enrich, e = s.repo.GetMatchEnrichment(gctx, matchID)
		return e
	})
	d.failures.goLoad(gctx, g, matchID, srcScoreboard, func() error {
		var e error
		d.scoreboard, e = s.repo.GetMatchScoreboard(gctx, matchID)
		return e
	})
	d.failures.goLoad(gctx, g, matchID, srcEvents, func() error {
		var e error
		d.events, e = s.repo.GetMatchEvents(gctx, matchID)
		return e
	})
	d.failures.goLoadSection(gctx, g, matchID, "kv_pairs", func() error {
		var e error
		d.kvPairs, e = s.repo.GetMatchKVPairs(gctx, matchID)
		return e
	})
	d.failures.goLoadSection(gctx, g, matchID, "kill_sources", func() error {
		var e error
		d.killSources, e = s.repo.GetMatchKillSources(gctx, matchID)
		return e
	})
	d.failures.goLoadSection(gctx, g, matchID, "kill_assists", func() error {
		var e error
		d.killAssists, e = s.repo.GetMatchKillAssists(gctx, matchID)
		return e
	})
	d.failures.goLoadSection(gctx, g, matchID, "bulk_medals", func() error {
		var e error
		d.bulkMedals, e = s.repo.GetMatchBulkMedals(gctx, matchID)
		return e
	})
	d.failures.goLoadSection(gctx, g, matchID, "bulk_weapons", func() error {
		var e error
		d.bulkWeapons, e = s.repo.GetMatchBulkWeaponKills(gctx, matchID)
		return e
	})
	d.failures.goLoadSection(gctx, g, matchID, "skill_rank", func() error {
		var e error
		d.skillRank, e = s.repo.GetMatchSkillRank(gctx, matchID)
		return e
	})
	d.failures.goLoadSection(gctx, g, matchID, "shared_csrs", func() error {
		var e error
		d.sharedCSRs, e = s.repo.GetMatchSharedCSRs(gctx, matchID)
		return e
	})
	d.failures.goLoadSection(gctx, g, matchID, "media", func() error {
		var e error
		d.media, e = s.repo.GetMatchMedia(gctx, matchID)
		return e
	})

	if err := g.Wait(); err != nil {
		return matchViewData{}, err
	}
	// Requête abandonnée pendant le chargement : les lectures coupées ne sont pas des pannes,
	// on rend l'erreur de contexte plutôt qu'un contexte parsemé de sources vides.
	if cerr := ctx.Err(); cerr != nil {
		return matchViewData{}, fmt.Errorf("match_view: contexte du rejeu interrompu pour %s: %w", matchID, cerr)
	}
	return d, nil
}

// buildReplayContextFromData assemble les quatre morceaux dans l'ordre de buildMatchViewFromData :
// recalage T0, en-tête, événements (synthèse kv si les events ne portent aucun kill, puis
// décorations), lignes du tableau de score (libellés d'équipe compris), médias.
func (s *MatchViewService) buildReplayContextFromData(
	ctx context.Context, matchID string, meta *domain.MatchMetaRaw, d matchViewData,
) domain.MatchReplayContext {
	prepareMatchViewEvents(&d, matchID, meta)
	header, outcomes := s.assembleMatchHeader(ctx, matchID, meta, d)

	events, _ := applyKVSynthesisIfNeeded(d.events, nil, d.kvPairs, matchID)
	highlights := highlightEventList(events)
	s.decorateHighlightEvents(ctx, highlights, d)

	// Rencontres, némésis et extras des amis ne sont pas des entrées des LIGNES : seules
	// celles-ci sont gardées de l'onglet.
	team := buildTeamTabFull(d.scoreboard, nil, nil, nil, d.bulkMedals, d.bulkWeapons, s.xuid, s.titleSlug, d.enrich, d.skillRank, nil, d.sharedCSRs, s.assetURL, outcomes)
	s.applyTeamNames(ctx, team.Scoreboard)

	return domain.MatchReplayContext{
		Header:            header,
		Scoreboard:        team.Scoreboard,
		HighlightEvents:   highlights,
		Media:             buildMediaTab(d.media),
		SourceUnavailable: d.failures.anyFailed(),
	}
}
