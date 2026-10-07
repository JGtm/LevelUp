// Package service — solo_lives_block.go : « MES VIES : PRÈS D'UN COÉQUIPIER OU SEUL » sur une liste de
// matchs, commune aux Séries temporelles (la fenêtre filtrée) et à la page Sessions (une session).
// Calcul : analysis/coordination.ViesPresOuSeul ; type publié : domain.TimeseriesLivesNearTeammate.
//
// Orchestration seule : UNE lecture bornée par les matchs et le joueur (port.SoloLivesRepository,
// ADR 0036), la portée COURANTE du radar de chaque match par `mappings.PorteesDuRadarParMatch` (source
// unique), le calcul pur. Capability absente (repo non câblé ou table manquante) : bloc absent,
// journalisé en Debug ; lecture en échec : bloc absent, journalisé en Error — jamais une erreur de
// page, jamais un zéro inventé. Les journaux portent l'attribut `page`.
package service

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
)

// viesQuery — ce que la page fournit à la lecture.
type viesQuery struct {
	// Page : `timeseries` ou `sessions`, posé sur chaque journal. Player : gamertag du joueur.
	Page       string
	Player     string
	PlayerXUID string
	// Repo : nil = capability `film.kill_positions` absente. Radar : game_variant_name -> mètres.
	Repo     port.SoloLivesRepository
	Radar    map[string]int
	MatchIDs []string
}

// lireViesPresOuSeul rend le bloc des matchs `q.MatchIDs`, ou nil (capability absente, lecture en
// échec, aucune vie lue). L'appelant garantit une liste non vide et un joueur connu.
func lireViesPresOuSeul(ctx context.Context, q viesQuery) *domain.TimeseriesLivesNearTeammate {
	if q.Repo == nil {
		slog.DebugContext(ctx, "vies_capability_absente", "page", q.Page, "player", q.Player,
			"capability", string(games.CapFilmKillPositions), "err", games.ErrCapabilityNotSupported)
		return nil
	}
	defer timing.FromContext(ctx).Section("lives")()
	lues, err := q.Repo.LoadLivesNearTeammate(ctx, q.MatchIDs, q.PlayerXUID)
	switch {
	case errors.Is(err, games.ErrCapabilityNotSupported):
		slog.DebugContext(ctx, "vies_capability_absente", "page", q.Page, "player", q.Player,
			"capability", string(games.CapFilmKillPositions), "err", err)
		return nil
	case err != nil:
		slog.ErrorContext(ctx, "vies_en_echec", "page", q.Page, "player", q.Player, "matchs", len(q.MatchIDs), "err", err)
		return nil
	}
	rayons, sansRayon := mappings.PorteesDuRadarParMatch(q.Radar, lues.Variantes)
	bloc, fragsEcartes := coordination.ViesPresOuSeul(lues, rayons)
	if bloc.MatchesRead == 0 {
		slog.InfoContext(ctx, "vies_sans_vie", "page", q.Page, "player", q.Player, "matchs", len(q.MatchIDs))
		return nil
	}
	slog.InfoContext(ctx, "vies",
		"page", q.Page, "player", q.Player, "matchs", len(q.MatchIDs), "matchs_lus", bloc.MatchesRead,
		"matchs_sans_portee", bloc.MatchesWithoutRadar, "variantes_sans_portee", sansRayon,
		"pres", bloc.Near.Lives, "seul", bloc.Alone.Lives, "ecartees_sans_coequipier", bloc.ExcludedUnlocated,
		"ecartees_sans_portee", bloc.ExcludedNoRadar, "ecartees_journal_non_publiable", bloc.ExcludedUnpublishable,
		"frags_ecartes", fragsEcartes)
	return &bloc
}
