// Package service — leaderboard_service.go : page Classement multi-catégories.
//
// Sprint 54 E (origine) + refonte Classement : CSR mondial (snapshots Halo
// Waypoint) + classements de stats agrégées des joueurs croisés.
package service

import (
	"context"
	"log/slog"
	"strings"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/port"
)

// LeaderboardService orchestre le classement CSR.
type LeaderboardService struct {
	repo port.LeaderboardRepository
}

// NewLeaderboardService crée un LeaderboardService.
func NewLeaderboardService(repo port.LeaderboardRepository) *LeaderboardService {
	return &LeaderboardService{repo: repo}
}

// defaultLeaderboardLimit borne la taille par défaut d'une page de classement.
// Top 50 (2026-07-03) : classement mondial plus digeste + synchronisé avec
// duckdb.WorldLeaderboardTopN (profondeur enrichie). Le classement mondial masque
// en plus les joueurs privés/sans données (anti-join world_player_no_data).
const defaultLeaderboardLimit = 50

// GetPage construit la réponse du classement selon la catégorie demandée :
//   - "csr-world" (défaut) : classement CSR mondial (snapshots Halo Waypoint).
//   - kills/kda/accuracy/… : stats agrégées des joueurs croisés.
func (s *LeaderboardService) GetPage(ctx context.Context, req domain.LeaderboardRequest) (domain.LeaderboardResponse, error) {
	category := domain.LeaderboardCategory(req.Category)
	if category == "" {
		category = domain.LeaderboardCSRWorld
	}
	limit := req.Limit
	if limit <= 0 {
		limit = defaultLeaderboardLimit
	}

	// Slug effectif (fallback défaut). Gating capability (PMT-7) : un titre sans
	// world.leaderboard dégrade en vide + 200 (jamais 500), via le registre — pas
	// de comparaison de slug.
	titleSlug := req.TitleSlug
	if titleSlug == "" {
		titleSlug = titlePkg.DefaultSlug
	}
	// Entries non nil dès la construction : `entries` n'a pas d'omitempty, une
	// réponse vide doit sérialiser `[]` et jamais `null` (ratchet
	// TestDTOs_NoNilSlicesOnEmptyInput) — vrai pour TOUS les chemins de sortie
	// anticipée : titre sans capability, couple (saison, playlist) incomplet.
	resp := domain.LeaderboardResponse{
		Category: string(category), Season: req.Season, Playlist: req.Playlist, TitleSlug: titleSlug,
		Entries: []domain.LeaderboardEntry{},
	}
	if !titleHasWorldLeaderboard(titleSlug) {
		return resp, nil
	}

	// Le classement mondial est identifié par un COUPLE (saison, playlist) : sans les
	// deux, il n'y a pas de classement à servir. Le repo refuse ce cas par une erreur
	// (contrat interne, cf. GetCSRWorldLeaderboard) — mais côté HTTP, un paramètre
	// manquant n'est pas une panne serveur : on dégrade en réponse vide + 200, comme le
	// reste du domaine (titre sans capability). Le front, lui, se cale sur le catalogue.
	if category == domain.LeaderboardCSRWorld &&
		(strings.TrimSpace(req.Season) == "" || strings.TrimSpace(req.Playlist) == "") {
		slog.WarnContext(ctx, "classement mondial demandé sans saison ou sans playlist — réponse vide",
			"module", logging.ModuleLeaderboard, "titleSlug", titleSlug,
			"season", req.Season, "playlist", req.Playlist)
		return resp, nil
	}

	var (
		entries []domain.LeaderboardEntry
		err     error
	)
	if category == domain.LeaderboardCSRWorld {
		entries, err = s.repo.GetCSRWorldLeaderboard(ctx, titleSlug, req.Season, req.Playlist, limit)
	} else {
		entries, err = s.repo.GetStatLeaderboard(ctx, titleSlug, category, req.Playlist, req.Season, limit)
	}
	if err != nil {
		return domain.LeaderboardResponse{}, err
	}

	resp.Entries = entries
	resp.TotalLocal = len(entries)
	return resp, nil
}

// titleHasWorldLeaderboard : le titre supporte-t-il les classements mondiaux ?
// Capability-gated (jamais de comparaison de slug). Titre inconnu → false.
func titleHasWorldLeaderboard(slug string) bool {
	d := titlePkg.DefaultRegistry().Get(slug)
	return d != nil && d.HasCapability(titlePkg.CapWorldLeaderboard)
}

// GetCatalog retourne les saisons + playlists disponibles (sélecteurs dynamiques)
// pour le titre courant (ctx). Titre sans world.leaderboard → catalogue vide.
func (s *LeaderboardService) GetCatalog(ctx context.Context) (domain.LeaderboardCatalog, error) {
	titleSlug := ctxkeys.TitleSlug(ctx)
	if !titleHasWorldLeaderboard(titleSlug) {
		return domain.LeaderboardCatalog{}, nil
	}
	return s.repo.GetWorldLeaderboardCatalog(ctx, titleSlug)
}
