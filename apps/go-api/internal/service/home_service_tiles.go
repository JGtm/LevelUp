// Package service — home_service_tiles.go : les tuiles de match de l'Accueil. UN constructeur
// (buildTiles) pour la tête de la page, les pages suivantes et les favoris (D-2) : une tuile
// servie par une page suivante ne peut pas différer de celle de la tête (traductions d'assets,
// placements, rejeu, analyse en cours, enrichissements par tuile).
package service

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/observability/timing"
)

// tilesInput regroupe ce que buildTiles lit.
type tilesInput struct {
	// allRows : l'historique canonique COMPLET. Les placements de performance se comptent par
	// chaîne sur tout l'historique (D-3), pas sur la fenêtre.
	allRows []canonical.PlayerMatchRow
	// window : les lignes dont on construit la tuile, dans l'ordre d'affichage.
	window      []canonical.PlayerMatchRow
	favoriteIDs map[string]bool
	locale      string
	// list nomme la liste dans les sections de durée `home_<étape>_<liste>`.
	list string
	// translated : les traductions d'assets sont déjà posées sur `window` (la tête les pose sur
	// tout l'historique, que les sessions et les faits marquants lisent aussi).
	translated bool
}

// tileOptions rassemble ce que la couche service injecte dans la projection des tuiles : locale,
// létalité du titre, résolveur de badge de palier, variantes décidées aux manches (ADR 0032). Les
// tuiles d'accueil lisent le score par la MÊME règle que la vue match et les tableaux.
func (s *HomeService) tileOptions(ctx context.Context, locale string) analysis.RecentMatchesOptions {
	return analysis.RecentMatchesOptions{
		Locale:            locale,
		EffectiveHpToKill: games.EffectiveHpToKill(ctxkeys.TitleSlug(ctx)),
		SkillBadgeURL:     s.skillBadgeResolver,
		RoundsDecide:      s.roundsDecide,
	}
}

// translateAssets remplit Labels["fr"] des AssetReference depuis metadata.asset_translations +
// mode_name_tr quand match_registry.{...}_name_fr est NULL. Sans cela, modes, cartes et playlists
// restent en anglais. Best-effort : un échec est journalisé, les libellés d'origine restent.
func (s *HomeService) translateAssets(ctx context.Context, rows []canonical.PlayerMatchRow) {
	defer timing.FromContext(ctx).Section("home_asset_translations")()
	if err := s.repo.EnrichCanonicalAssetTranslations(ctx, rows); err != nil {
		slog.WarnContext(ctx, "home: EnrichCanonicalAssetTranslations failed", "err", err)
	}
}

// buildTiles construit les tuiles de `in.window`, dans l'ordre reçu, enrichies. Rend toujours une
// tranche non nulle (un nil sérialiserait `null`).
func (s *HomeService) buildTiles(ctx context.Context, in tilesInput) []domain.RecentMatchItem {
	if !in.translated {
		s.translateAssets(ctx, in.window)
	}
	stop := timing.FromContext(ctx).Section("home_build_tiles")
	items := analysis.BuildRecentMatchesWithFavoritesFromCanonical(
		in.window, len(in.window), in.favoriteIDs, s.tileOptions(ctx, in.locale))
	stop()
	if items == nil {
		return []domain.RecentMatchItem{}
	}
	// Placement de chaîne de PERFORMANCE : même signal que l'Explorer (« En placement (X/Y) » au
	// lieu d'un vide ou d'un 0 fabriqué pour une perf structurellement absente).
	stop = timing.FromContext(ctx).Section("home_perf_placements")
	applyPerfPlacementsToRecentItems(ctx, in.allRows, items)
	stop()
	// Lien « Rejeu 2D » : présence d'artefact résolue en UN listing de dossier.
	replays := s.replayAvailability(ctx)
	applyReplayAvailabilityToRecentItems(replays, items)
	// Médailles, citations (puis commendations natives en repli, TITLE-AGNOSTIC P7) et frags
	// assistés, une section par étape.
	enrichTileList(ctx, s.repo, items, in.list)
	// Indicateur « analyse du film en cours » ; sans résolveur câblé (titre sans la capability du
	// film), le champ reste faux.
	s.applyAnalysisPendingToRecentItems(ctx, replays, items)
	return items
}

// GetHomeMatches rend une liste de tuiles : la suite de l'historique après le curseur, ou les
// favoris en entier. Les mêmes lignes canoniques que la tête (chargeur en cache), la même
// construction (buildTiles) ; seule la fenêtre diffère.
func (s *HomeService) GetHomeMatches(ctx context.Context, locale string, req domain.HomeMatchesRequest) (*domain.HomeMatchesPage, error) {
	defer func(start time.Time) {
		observability.RecordDurationMS("home_get_matches", time.Since(start).Milliseconds())
	}(time.Now())
	cursor, ok := analysis.ParseHomeTileCursor(req.Before)
	if !ok && req.Kind != domain.HomeMatchesKindFavorites {
		return nil, domain.ErrHomeCursorInvalid
	}
	rows, err := s.loadCanonicalRows(ctx)
	if err != nil {
		return nil, err
	}
	favoriteIDs := s.loadFavoriteIDs(ctx)
	ordered := analysis.OrderHomeTileRows(rows)
	win, list := s.matchesWindow(ordered, favoriteIDs, cursor, req)
	items := s.buildTiles(ctx, tilesInput{
		allRows: rows, window: win.Rows, favoriteIDs: favoriteIDs, locale: locale, list: list,
	})
	return &domain.HomeMatchesPage{Items: items, HasMore: win.HasMore, NextBefore: nextBefore(win)}, nil
}

// matchesWindow choisit la fenêtre d'une demande : les favoris en entier, sinon `limit` tuiles
// après le curseur (HomeTilesHeadSize par défaut, HomeTilesMaxPageSize au plus).
func (s *HomeService) matchesWindow(
	ordered []canonical.PlayerMatchRow, favoriteIDs map[string]bool, cursor analysis.HomeTileCursor, req domain.HomeMatchesRequest,
) (analysis.HomeTileWindow, string) {
	if req.Kind == domain.HomeMatchesKindFavorites {
		return analysis.HomeTileWindow{Rows: analysis.FavoriteHomeTileRows(ordered, favoriteIDs)}, homeSectionListFavorites
	}
	limit := req.Limit
	if limit <= 0 {
		limit = domain.HomeTilesHeadSize
	}
	if limit > domain.HomeTilesMaxPageSize {
		limit = domain.HomeTilesMaxPageSize
	}
	return analysis.WindowHomeTileRows(ordered, cursor, limit), homeSectionListRecent
}

// nextBefore : le curseur à rendre, vide quand la liste est épuisée.
func nextBefore(w analysis.HomeTileWindow) string {
	if !w.HasMore {
		return ""
	}
	return w.NextBefore
}

// loadFavoriteIDs lit les matchs favoris du joueur. Sans dépôt social câblé, ou en erreur
// (journalisée), aucun favori : les tuiles se servent sans étoile plutôt que de tomber.
func (s *HomeService) loadFavoriteIDs(ctx context.Context) map[string]bool {
	if s.socialRepo == nil || s.playerSlug == "" {
		return nil
	}
	defer timing.FromContext(ctx).Section("home_favorite_ids")()
	ids, err := s.socialRepo.GetFavoriteMatchIDs(ctx, s.playerSlug)
	if err != nil {
		slog.WarnContext(ctx, "home: GetFavoriteMatchIDs failed", "err", err)
		return nil
	}
	return ids
}

// lastSessionOutcomes rend les issues de la séquence « Résultats de la dernière session » : les
// tuiles de la dernière session connue (solo ou escouade) parmi les lignes ordonnées, sinon les
// 20 plus récentes. Calculée sur l'HISTORIQUE COMPLET (D-3) : la tête fenêtrée ne la porterait pas
// toujours. `known` = libellés des deux sessions résumées de la page.
func lastSessionOutcomes(ordered []canonical.PlayerMatchRow, known map[string]bool, opts analysis.RecentMatchesOptions) []domain.HomeSessionOutcome {
	label := ""
	for i := range ordered {
		if l := sessionLabelOf(&ordered[i]); l != "" && known[l] {
			label = l
			break
		}
	}
	var rows []canonical.PlayerMatchRow
	if label != "" {
		for i := range ordered {
			if sessionLabelOf(&ordered[i]) == label {
				rows = append(rows, ordered[i])
			}
		}
	} else {
		rows = ordered[:min(lastSessionFallbackTiles, len(ordered))]
	}
	items := analysis.BuildRecentMatchesWithFavoritesFromCanonical(rows, len(rows), nil, opts)
	out := make([]domain.HomeSessionOutcome, 0, len(items))
	for _, it := range items {
		out = append(out, domain.HomeSessionOutcome{MatchID: it.MatchID, OutcomeTone: it.OutcomeTone, Detail: it.Detail})
	}
	return out
}

// lastSessionFallbackTiles : sans session connue parmi les tuiles, la séquence montre les
// dernières parties jouées.
const lastSessionFallbackTiles = 20

// sessionLabelOf : libellé de session d'une ligne, "" s'il manque.
func sessionLabelOf(r *canonical.PlayerMatchRow) string {
	if r.Enrichment.SessionLabel == nil {
		return ""
	}
	return *r.Enrichment.SessionLabel
}
