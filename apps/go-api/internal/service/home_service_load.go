// Package service — home_service_load.go : chargement des données brutes de la page d'accueil.
package service

import (
	"context"
	"fmt"
	"log/slog"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
)

// homePageData regroupe toutes les données brutes chargées en parallèle par fetchHomePageData.
//
// `canonicalRows` est l'HISTORIQUE complet du joueur (chargeur en cache) : les totaux de la page
// (héros, faits marquants, sessions, placements) le lisent en entier, seules les tuiles servies
// en sont une fenêtre (D-1, D-3).
type homePageData struct {
	canonicalRows  []canonical.PlayerMatchRow
	spartanIdent   *domain.HomeSpartanIdentityRow
	totalMatches   int
	playlistRanks  []domain.HomePlaylistRank
	favoriteIDs    map[string]bool
	favWeaponName  string
	favWeaponKills int
}

// loadCanonicalRows charge les lignes canoniques du joueur (ADR 0011) : playerMatchesRepo,
// titleSlug et gamertag sont REQUIS (câblés par l'injection de dépendances partout).
func (s *HomeService) loadCanonicalRows(ctx context.Context) ([]canonical.PlayerMatchRow, error) {
	defer timing.FromContext(ctx).Section("home_load_matches")()
	if s.playerMatchesRepo == nil || s.titleSlug == "" || s.gamertag == "" {
		return nil, fmt.Errorf("HomeService: PlayerMatchesRepo non câblé (le wiring DI est requis)")
	}
	rows, err := s.playerMatchesRepo.LoadPlayerMatches(
		ctx, s.titleSlug, s.gamertag, port.PlayerMatchFilters{},
	)
	if err != nil {
		return nil, err
	}
	slog.DebugContext(ctx, "home: loaded canonical", "rows", len(rows), "title_slug", s.titleSlug)
	return rows, nil
}

// fetchHomePageData charge toutes les données de la page d'accueil en parallèle.
// Les erreurs non-critiques sont absorbées (dégradation journalisée).
func (s *HomeService) fetchHomePageData(ctx context.Context, locale string) (homePageData, error) {
	var d homePageData
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		d.canonicalRows, err = s.loadCanonicalRows(gctx)
		return err
	})
	g.Go(func() error {
		defer timing.FromContext(gctx).Section("home_spartan_identity")()
		// Path live (CareerLiveService câblé) : flow découplé du post-sync matchs avec throttle
		// 5 min (XP) / 6 h (customisation) + fallback DB per-field garanti. Fallback :
		// repo.LoadSpartanIdentity (DB-only) pour les bootstraps sans careerLive.
		if s.careerLive != nil {
			// Sujet EXPLICITE = joueur de la PAGE (s.xuid = pdb.XUID), jamais le contexte ambiant
			// (finding ID4).
			row, err := s.careerLive.GetSpartanIdentityFor(gctx, s.xuid)
			if err != nil {
				slog.WarnContext(gctx, "home: CareerLive.GetSpartanIdentityFor failed", "err", err)
			}
			d.spartanIdent = row
			return nil
		}
		var err error
		d.spartanIdent, err = s.repo.LoadSpartanIdentity(gctx)
		if err != nil {
			slog.WarnContext(gctx, "home: LoadSpartanIdentity failed", "err", err)
		}
		return nil
	})
	g.Go(func() error {
		defer timing.FromContext(gctx).Section("home_count_matches")()
		// Fallback sur len(matches) après le Wait si la query échoue (totalMatches reste 0).
		d.totalMatches, _ = s.repo.CountPlayerMatches(gctx)
		return nil
	})
	g.Go(func() error {
		var err error
		defer timing.FromContext(gctx).Section("home_playlist_ranks")()
		d.playlistRanks, err = s.repo.LoadRecentPlaylistRanks(gctx, locale)
		if err != nil {
			slog.WarnContext(gctx, "home: LoadRecentPlaylistRanks failed", "err", err)
		}
		return nil
	})
	g.Go(func() error {
		defer timing.FromContext(gctx).Section("home_favorite_weapon")()
		if wName, wKills, err := s.repo.LoadFavoriteWeapon(gctx, locale); err == nil && wName != "" {
			d.favWeaponName = wName
			d.favWeaponKills = wKills
		}
		return nil
	})
	g.Go(func() error {
		d.favoriteIDs = s.loadFavoriteIDs(gctx)
		return nil
	})

	if err := g.Wait(); err != nil {
		return homePageData{}, err
	}
	if d.totalMatches == 0 {
		d.totalMatches = len(d.canonicalRows)
	}
	return d, nil
}
