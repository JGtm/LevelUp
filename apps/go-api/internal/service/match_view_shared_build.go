// Package service — match_view_shared_build.go : les étapes d'assemblage COMMUNES à la vue
// complète d'un match (buildMatchViewFromData) et au contexte du rejeu
// (buildReplayContextFromData). Chaque morceau que les deux servent (en-tête, événements du
// fil, médias, lignes du tableau de score) passe par ces fonctions et par les mêmes builders :
// aucune des deux réponses ne réécrit la logique de l'autre.
package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/observability"
)

// loadMatchMeta lit les métadonnées du match (Q11), préalable bloquant des deux réponses.
//
// L'ABSENCE ET LA PANNE NE SONT PAS LE MÊME 404. Absence (sql.ErrNoRows, préservé par le
// wrapping %w du repo) : match jamais synchronisé, domain.ErrNotFound (404 propre, aucun fetch
// live vers l'API du titre depuis ces pages). Panne technique (schéma en retard, timeout,
// verrou, I/O) : erreur rendue telle quelle (500) et journal ERREUR — la déguiser en « pas
// encore synchronisé » masquerait une panne totale de la page.
func (s *MatchViewService) loadMatchMeta(ctx context.Context, matchID string) (*domain.MatchMetaRaw, error) {
	meta, err := s.repo.GetMatchMeta(ctx, matchID)
	if err == nil {
		return meta, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		slog.InfoContext(ctx, "match_view: match absent du substrat local (pas encore synchronisé)",
			"match_id", matchID, "err", err)
		return nil, domain.ErrNotFound("match", matchID)
	}
	slog.Log(ctx, observability.LevelUnlessCanceled(ctx, err, slog.LevelError), "match_view: lecture des métadonnées en échec (pas une absence)",
		"match_id", matchID, "err", err)
	return nil, fmt.Errorf("match_view: métadonnées illisibles pour %s: %w", matchID, err)
}

// prepareMatchViewEvents recale les sources d'events au référentiel gameplay (T0 réel depuis
// meta.T0Ms, 0 si indisponible) en un POINT UNIQUE, avant tout builder qui les lit, et rend
// la durée (ms) qui sert au binning de la Dominance. Sans ce recalage, le fil (event_time_ms)
// et les décorations appariées par clé exacte (arme, assistant, victime) seraient sur
// l'horloge du film, countdown inclus.
func prepareMatchViewEvents(d *matchViewData, matchID string, meta *domain.MatchMetaRaw) int64 {
	durationMS := tugDurationMS(meta)
	var t0Ms int64
	if meta != nil && meta.T0Ms != nil {
		t0Ms = *meta.T0Ms
	}
	correctMatchViewEventsT0(d, matchID, timeline.BuildForMatchMs(durationMS, t0Ms))
	return durationMS
}

// assembleMatchHeader construit l'en-tête du match et rend le jeu d'outcomes du titre, résolu
// UNE fois pour l'en-tête et les lignes du tableau de score (la CLÉ canonique de l'issue,
// jamais un texte). Les `apply*` vivent ici et pas dans buildMatchHeader parce que leurs
// dépendances (table réglementaire, adapter sémantique, règle de score, rejeu, taxonomie des
// modes) sont portées par le service.
func (s *MatchViewService) assembleMatchHeader(
	ctx context.Context, matchID string, meta *domain.MatchMetaRaw, d matchViewData,
) (domain.MatchViewHeader, *mappings.OutcomeMappingSet) {
	header := buildMatchHeader(ctx, matchID, meta, d.stats, d.enrich, d.scoreboard, s.assetURL, s.isMatchFavorite(ctx, matchID))
	// Flag « Prolongation » : titre sans table réglementaire → no-op.
	applyMatchHeaderOvertime(&header, meta, s.regulationSeconds)
	outcomes := outcomesOf(s.semantic)
	applyMatchHeaderOutcomeKey(&header, outcomes)
	// Score en points ou en MANCHES ([rounds_decide]) ; table absente → points.
	applyMatchHeaderScore(&header, meta, d.stats, s.roundsDecide)
	// Lecture du bloc « Score dans le temps » ; règle absente → champ vide → courbe.
	applyMatchHeaderScoreTimeline(&header, meta, s.scoreTimelineKind)
	// Présence de l'artefact de rejeu 2D : un os.Stat, jamais une lecture.
	applyMatchHeaderReplay(ctx, &header, matchID, s.replaySvc)
	// « Analyse du film en cours » : lit ReplayAvailable posé juste au-dessus.
	s.applyHeaderAnalysisPending(ctx, &header)
	// Catégorie custom du mode (taxonomie injectée) : la garde Fiesta du rejeu s'y corrèle.
	applyMatchHeaderModeCategory(&header, meta, s.modeTaxonomy)
	return header, outcomes
}

// isMatchFavorite : lookup synchrone (PK player_slug+match_id). Dégradation gracieuse si le
// repo social n'est pas câblé ; un échec est journalisé avant de dégrader à false.
func (s *MatchViewService) isMatchFavorite(ctx context.Context, matchID string) bool {
	if s.socialRepo == nil || s.playerSlug == "" {
		return false
	}
	fav, err := s.socialRepo.IsMatchFavorite(ctx, s.playerSlug, matchID)
	if err != nil {
		slog.WarnContext(ctx, "match_view: IsMatchFavorite échoué",
			"match_id", matchID, "player", s.playerSlug, "err", err)
		return false
	}
	return fav
}

// decorateHighlightEvents pose sur les events du fil (EN PLACE) l'arme, l'assistant, la
// victime et l'équipe du tueur, puis l'identité des médailles. Ce sont des décorations, pas
// des entrées d'un calcul : elles se posent après l'assemblage des events.
func (s *MatchViewService) decorateHighlightEvents(ctx context.Context, events []domain.MatchHighlightEvent, d matchViewData) {
	decorateKillFeed(ctx, events, killFeedInputs{
		sources:    d.killSources,
		assists:    d.killAssists,
		victims:    d.kvPairsFeed,
		scoreboard: d.scoreboard,
		assetURL:   s.assetURL,
	})
	decorateMedalEvents(ctx, events, s.repo, s.assetURL)
}
