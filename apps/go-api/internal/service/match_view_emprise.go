// Package service — match_view_emprise.go : L'EMPRISE SUR LA VUE MATCH (plan
// `.ai/V7.5/PLAN_MATCHVIEW_EMPRISE_2026-10-06.md`, lot M2 ; types publiés : domain/match_emprise.go).
//
// Orchestration seule. Un match, les joueurs de l'équipe du joueur de la page connus un par un :
// l'Emprise vient de l'assemblage commun aux Séries temporelles et à Sessions
// (`buildSoloEmpriseBlock`, avec la liste des joueurs de l'équipe) ; l'« Isolement » de chaque joueur
// de la lecture de l'équipe (`lireViesDuCamp`) ; le fait « journal des morts publiable » de Q21d, déjà
// chargée par la page. Chaque source dégrade seule (jamais une erreur de page).
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/squademprise"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
)

// matchViewPage — l'attribut `page` des journaux de l'Emprise et des vies de la Vue match.
const matchViewPage = "match_view"

// matchViewEmpriseDeps — les dépendances des blocs de l'Emprise, embarquées par MatchViewService.
type matchViewEmpriseDeps struct {
	// empriseRepo : la feuille de match (frags aux armes spéciales), câblée pour tout titre.
	empriseRepo port.SquadEmpriseRepository
	// usageRepo / repoRoot : le résumé d'usage (câblé sous `film.usage_summary`) ; nil = sans film.
	usageRepo port.SessionUsageRepository
	repoRoot  string
	// vehicleRepo : la ressource véhicules (câblée sous `film.vehicle_usage`) ; nil = non mesurée.
	vehicleRepo port.SquadVehicleRepository
	// livesRepo / radarRange : les vies de l'équipe (câblées sous `film.kill_positions`) et la portée
	// du radar par variante (`regulation.toml [radar_range_m]`).
	livesRepo  port.CampLivesRepository
	radarRange map[string]int
	// categories : les frags par catégorie de source du film (objet explosif, chute), pour les
	// « Outils de destruction » ; nil = les deux lignes rejoignent le reliquat.
	categories port.KillSourceCategoryRepository
}

// WithEmpriseSheet injecte la feuille de match de l'Emprise (câblage inconditionnel).
func (s *MatchViewService) WithEmpriseSheet(repo port.SquadEmpriseRepository) *MatchViewService {
	s.emprise.empriseRepo = repo
	return s
}

// WithEmpriseUsageSummary injecte le résumé d'usage et la racine du dépôt (catalogue d'armes, noms de
// véhicules) — câblé sous `film.usage_summary`.
func (s *MatchViewService) WithEmpriseUsageSummary(repo port.SessionUsageRepository, repoRoot string) *MatchViewService {
	s.emprise.usageRepo, s.emprise.repoRoot = repo, repoRoot
	return s
}

// WithEmpriseVehicles injecte la ressource véhicules (câblée sous `film.vehicle_usage`).
func (s *MatchViewService) WithEmpriseVehicles(repo port.SquadVehicleRepository) *MatchViewService {
	s.emprise.vehicleRepo = repo
	return s
}

// WithCampLives injecte la lecture des vies de l'équipe (câblée sous `film.kill_positions`).
func (s *MatchViewService) WithCampLives(repo port.CampLivesRepository) *MatchViewService {
	s.emprise.livesRepo = repo
	return s
}

// WithRadarRange injecte la portée du radar par variante (même table que l'onglet Tactique).
func (s *MatchViewService) WithRadarRange(radar map[string]int) *MatchViewService {
	s.emprise.radarRange = radar
	return s
}

// WithKillSourceCategories injecte les frags par catégorie de source du film.
func (s *MatchViewService) WithKillSourceCategories(repo port.KillSourceCategoryRepository) *MatchViewService {
	s.emprise.categories = repo
	return s
}

// matchEmpriseFields rend les blocs de l'Emprise du match. Sans ligne du joueur de la page au tableau
// des scores, rien : l'équipe n'est pas connue.
func (s *MatchViewService) matchEmpriseFields(
	ctx context.Context, matchID string, meta *domain.MatchMetaRaw, d matchViewData, suivis map[string]port.FriendMatchExtras,
) domain.MatchViewEmpriseFields {
	players := matchCampPlayers(d.scoreboard, s.xuid, suivis)
	if len(players) == 0 {
		slog.DebugContext(ctx, "emprise_sans_joueur_de_la_page", "page", matchViewPage, "match_id", matchID)
		return domain.MatchViewEmpriseFields{}
	}
	return domain.MatchViewEmpriseFields{
		Emprise:           s.matchEmpriseBlock(ctx, matchID, meta, players, killJournalState(d)),
		LivesNearTeammate: s.matchLives(ctx, matchID, players),
	}
}

// matchEmpriseBlock assemble l'Emprise d'UN match, fiches = les joueurs de l'équipe.
func (s *MatchViewService) matchEmpriseBlock(
	ctx context.Context, matchID string, meta *domain.MatchMetaRaw, players []domain.SessionUsageSquadPlayer, journal string,
) *domain.MatchEmpriseBlock {
	defer timing.FromContext(ctx).Section("match_emprise")()
	m := squademprise.Match{MatchID: matchID}
	if meta != nil && meta.StartTime != nil {
		m.StartTime = *meta.StartTime
	}
	solo := buildSoloEmpriseBlock(ctx, soloEmpriseQuery{
		Page: matchViewPage, Player: players[0].Gamertag, PlayerXUID: s.xuid,
		RepoRoot: s.emprise.repoRoot, TitleSlug: s.titleSlug, Locale: ctxkeys.Locale(ctx),
		Current: []squademprise.Match{m}, UsageRepo: s.emprise.usageRepo, EmpriseRepo: s.emprise.empriseRepo,
		VehicleRepo: s.emprise.vehicleRepo, Players: players,
	})
	return &domain.MatchEmpriseBlock{SquadEmpriseBlock: solo.SquadEmpriseBlock, KillJournal: journal}
}

// killJournalState rend l'état du journal des morts du match : indisponible quand la lecture de sa
// portée a échoué, publiable dès une mort publiable, non publiable sinon.
func killJournalState(d matchViewData) string {
	switch {
	case d.assistScopeFailed:
		return domain.MatchKillJournalUnavailable
	case d.assistScope.PublishableDeaths > 0:
		return domain.MatchKillJournalPublishable
	default:
		return domain.MatchKillJournalNotPublishable
	}
}

// matchLives rend l'« Isolement » de chaque joueur de l'équipe, dans l'ordre des fiches ; nil
// (capability absente, lecture en échec, aucune vie lue).
func (s *MatchViewService) matchLives(
	ctx context.Context, matchID string, players []domain.SessionUsageSquadPlayer,
) *domain.MatchLivesNearTeammate {
	defer timing.FromContext(ctx).Section("match_lives")()
	xuids := make([]string, 0, len(players))
	for _, p := range players {
		xuids = append(xuids, p.XUID)
	}
	bilans, lu := lireViesDuCamp(ctx, viesCampQuery{
		Page: matchViewPage, Player: players[0].Gamertag, XUIDs: xuids,
		Repo: s.emprise.livesRepo, Radar: s.emprise.radarRange, MatchIDs: []string{matchID},
	})
	if !lu {
		return nil
	}
	out := &domain.MatchLivesNearTeammate{Players: make([]domain.MatchLivesPlayer, 0, len(players))}
	for _, x := range xuids {
		out.Players = append(out.Players, domain.MatchLivesPlayer{XUID: x, TimeseriesLivesNearTeammate: bilans[x]})
	}
	return out
}

// matchCampPlayers rend les joueurs de l'équipe du joueur de la page présents à la fin, humains : le
// joueur de la page d'abord, puis les profils suivis (`suivis`), puis les autres — chaque groupe dans
// l'ordre du tableau des scores. Bots et partis n'ont pas de fiche (ils restent dans le reste de
// l'équipe des comptes). Sans ligne du joueur de la page : nil ; sans équipe (chacun pour soi) : le
// joueur de la page seul.
func matchCampPlayers(
	scoreboard []domain.ScoreboardRaw, me string, suivis map[string]port.FriendMatchExtras,
) []domain.SessionUsageSquadPlayer {
	var moi *domain.ScoreboardRaw
	for i := range scoreboard {
		if scoreboard[i].XUID == me && me != "" {
			moi = &scoreboard[i]
			break
		}
	}
	if moi == nil {
		return nil
	}
	lui := domain.SessionUsageSquadPlayer{XUID: moi.XUID, Gamertag: moi.Gamertag}
	if moi.TeamID == nil {
		return []domain.SessionUsageSquadPlayer{lui}
	}
	var suivisDeLEquipe, autres []domain.SessionUsageSquadPlayer
	for _, r := range scoreboard {
		if r.XUID == me || r.XUID == "" || r.IsBot || r.TeamID == nil || *r.TeamID != *moi.TeamID ||
			(r.LeftInProgress != nil && *r.LeftInProgress) {
			continue
		}
		p := domain.SessionUsageSquadPlayer{XUID: r.XUID, Gamertag: r.Gamertag}
		if _, suivi := suivis[r.XUID]; suivi {
			suivisDeLEquipe = append(suivisDeLEquipe, p)
		} else {
			autres = append(autres, p)
		}
	}
	out := make([]domain.SessionUsageSquadPlayer, 0, 1+len(suivisDeLEquipe)+len(autres))
	out = append(out, lui)
	return append(append(out, suivisDeLEquipe...), autres...)
}
