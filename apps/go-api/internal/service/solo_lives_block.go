// Package service — solo_lives_block.go : « ISOLEMENT » (vies près d'un coéquipier ou seules) sur une
// liste de matchs — pour un joueur (Séries temporelles : la fenêtre filtrée ; Sessions : une session)
// ou pour chaque joueur de l'équipe (Vue match : un match). Calcul :
// analysis/coordination.ViesPresOuSeul ; type publié : domain.TimeseriesLivesNearTeammate.
//
// Orchestration seule : UNE lecture bornée par les matchs et le ou les joueurs (port.SoloLivesRepository,
// port.CampLivesRepository, ADR 0036), la portée COURANTE du radar de chaque match par
// `mappings.PorteesDuRadarParMatch` (source unique), le calcul pur par joueur. Capability absente (repo
// non câblé ou table manquante) : bloc absent, journalisé en Debug ; lecture en échec : bloc absent,
// journalisé en Error — jamais une erreur de page, jamais un zéro inventé. Les journaux portent
// l'attribut `page`.
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

// viesQuery — ce que la page fournit à la lecture d'un joueur.
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

// viesCampQuery — la même lecture pour plusieurs joueurs (Vue match : les joueurs de l'équipe).
type viesCampQuery struct {
	// Page et Player (gamertag du joueur de la page) : journaux. XUIDs : les joueurs lus.
	Page     string
	Player   string
	XUIDs    []string
	Repo     port.CampLivesRepository
	Radar    map[string]int
	MatchIDs []string
}

// lireViesPresOuSeul rend le bloc des matchs `q.MatchIDs`, ou nil (capability absente, lecture en
// échec, aucune vie lue). L'appelant garantit une liste non vide et un joueur connu.
func lireViesPresOuSeul(ctx context.Context, q viesQuery) *domain.TimeseriesLivesNearTeammate {
	if q.Repo == nil {
		viesCapabilityAbsente(ctx, q.Page, q.Player, games.ErrCapabilityNotSupported)
		return nil
	}
	defer timing.FromContext(ctx).Section("lives")()
	lues, err := q.Repo.LoadLivesNearTeammate(ctx, q.MatchIDs, q.PlayerXUID)
	if !viesLuesOK(ctx, q.Page, q.Player, len(q.MatchIDs), err) {
		return nil
	}
	bloc := bilansDesVies(ctx, q.Page, q.Radar, len(q.MatchIDs), map[string]domain.ViesLues{q.PlayerXUID: lues})[q.PlayerXUID]
	if bloc.MatchesRead == 0 {
		slog.InfoContext(ctx, "vies_sans_vie", "page", q.Page, "player", q.Player, "matchs", len(q.MatchIDs))
		return nil
	}
	return &bloc
}

// lireViesDuCamp rend le bilan de chaque joueur de `q.XUIDs` (une entrée par joueur, à zéro s'il n'a
// aucune vie lue) et vrai si au moins un joueur a une vie lue ; (nil, false) : capability absente ou
// lecture en échec. UNE lecture pour tous les joueurs (ADR 0036 I4).
func lireViesDuCamp(ctx context.Context, q viesCampQuery) (map[string]domain.TimeseriesLivesNearTeammate, bool) {
	if q.Repo == nil {
		viesCapabilityAbsente(ctx, q.Page, q.Player, games.ErrCapabilityNotSupported)
		return nil, false
	}
	parJoueur, err := q.Repo.LoadLivesNearTeammateForPlayers(ctx, q.MatchIDs, q.XUIDs)
	if !viesLuesOK(ctx, q.Page, q.Player, len(q.MatchIDs), err) {
		return nil, false
	}
	bilans := bilansDesVies(ctx, q.Page, q.Radar, len(q.MatchIDs), parJoueur)
	for _, b := range bilans {
		if b.MatchesRead > 0 {
			return bilans, true
		}
	}
	slog.InfoContext(ctx, "vies_sans_vie", "page", q.Page, "player", q.Player, "matchs", len(q.MatchIDs), "joueurs", len(q.XUIDs))
	return bilans, false
}

// viesLuesOK journalise une lecture qui n'a pas abouti (capability absente : Debug ; échec : Error).
func viesLuesOK(ctx context.Context, page, player string, matchs int, err error) bool {
	switch {
	case errors.Is(err, games.ErrCapabilityNotSupported):
		viesCapabilityAbsente(ctx, page, player, err)
		return false
	case err != nil:
		slog.ErrorContext(ctx, "vies_en_echec", "page", page, "player", player, "matchs", matchs, "err", err)
		return false
	}
	return true
}

func viesCapabilityAbsente(ctx context.Context, page, player string, err error) {
	slog.DebugContext(ctx, "vies_capability_absente", "page", page, "player", player,
		"capability", string(games.CapFilmKillPositions), "err", err)
}

// bilansDesVies range les vies de chaque joueur (portée du radar de chaque match, calcul pur) et
// journalise un bilan par joueur.
func bilansDesVies(
	ctx context.Context, page string, radar map[string]int, matchs int, parJoueur map[string]domain.ViesLues,
) map[string]domain.TimeseriesLivesNearTeammate {
	out := make(map[string]domain.TimeseriesLivesNearTeammate, len(parJoueur))
	for xuid, lues := range parJoueur {
		rayons, sansRayon := mappings.PorteesDuRadarParMatch(radar, lues.Variantes)
		bloc, fragsEcartes := coordination.ViesPresOuSeul(lues, rayons)
		out[xuid] = bloc
		if bloc.MatchesRead == 0 {
			continue
		}
		slog.InfoContext(ctx, "vies",
			"page", page, "xuid", xuid, "matchs", matchs, "matchs_lus", bloc.MatchesRead,
			"matchs_sans_portee", bloc.MatchesWithoutRadar, "variantes_sans_portee", sansRayon,
			"pres", bloc.Near.Lives, "seul", bloc.Alone.Lives, "ecartees_sans_coequipier", bloc.ExcludedUnlocated,
			"ecartees_sans_portee", bloc.ExcludedNoRadar, "ecartees_journal_non_publiable", bloc.ExcludedUnpublishable,
			"frags_ecartes", fragsEcartes)
	}
	return out
}
