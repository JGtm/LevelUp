// Package service — session_page_blocks.go : LES BLOCS DU FILM DE LA COLONNE DE SESSION (plan
// `.ai/PLAN_SESSIONS_EMPRISE_2026-10-06.md`, lot S2) — leurs sources, et l'ordre dans lequel la page
// les pose sur la session affichée et, tiroir ouvert, sur la session comparée.
//
// UNE LECTURE DU RÉSUMÉ D'USAGE PAR SESSION (ADR 0036 I4) : films, joueurs et participants sont lus
// une fois par scope et partagés par l'Emprise, l'objectif et l'effectif de camp de la
// coordination.
package service

import (
	"context"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/squadagg"
)

// pageSessions — l'attribut `page` des journaux des blocs partagés (Emprise solo, vies).
const pageSessions = "sessions"

// sessionBlocksDeps — les sources des blocs du film, embarquées par SessionPageService. Chacune est
// optionnelle : absente, son bloc se retire ou dit pourquoi, jamais une erreur de page.
type sessionBlocksDeps struct {
	// sessionXUID : le joueur de la page pour l'Emprise, les vies, l'objectif et la coordination
	// (WithSessionEmprise, câblage inconditionnel).
	sessionXUID string
	// sessionUsageRepo : le résumé d'usage (vues _latest), câblé sous film.usage_summary ; nil =
	// titre sans résumé. repoRoot : les catalogues du titre (noms d'armes et de véhicules de
	// l'Emprise) ; vide = les objets gardent leur clé.
	sessionUsageRepo port.SessionUsageRepository
	repoRoot         string
	// empriseRepo : la feuille de match (frags aux armes spéciales), tous titres.
	empriseRepo port.SquadEmpriseRepository
	// vehicleRepo : la ressource véhicules, câblée sous film.vehicle_usage ; nil = non mesurée.
	vehicleRepo port.SquadVehicleRepository
	// emblemLoader : l'emblème du joueur de la page ; nil = initiale côté web.
	emblemLoader port.EmblemURLLoader
	// livesRepo : les vies (câblé sous film.kill_positions) ; radarRange : game_variant_name -> m.
	livesRepo  port.SoloLivesRepository
	radarRange map[string]int
	// formesObjectives : les colonnes d'objectif (câblé sous match.objective.stats) ; nil = pas de
	// feuille d'objectif.
	formesObjectives port.SquadFormesObjectiveRepository
	// roundsDecide : game_variant_name -> le résultat se lit en manches (ADR 0032).
	roundsDecide map[string]bool
}

// WithSessionUsageSummary injecte le résumé d'usage et la racine des catalogues du titre. Câblé sous
// film.usage_summary (jamais slug==) ; nil = titre sans résumé : l'Emprise ne garde que la feuille
// de match, l'objectif et l'effectif de camp de la coordination se retirent.
func (s *SessionPageService) WithSessionUsageSummary(repo port.SessionUsageRepository, repoRoot string) *SessionPageService {
	s.sessionUsageRepo = repo
	s.repoRoot = repoRoot
	return s
}

// WithSessionEmprise injecte la feuille de match et le joueur de la page. Câblage inconditionnel :
// la colonne est écrite par tous les titres, et le joueur sert aussi la coordination.
func (s *SessionPageService) WithSessionEmprise(repo port.SquadEmpriseRepository, xuid string) *SessionPageService {
	s.empriseRepo = repo
	s.sessionXUID = xuid
	return s
}

// WithSessionVehicleUsage injecte la ressource véhicules (câblée sous film.vehicle_usage).
func (s *SessionPageService) WithSessionVehicleUsage(repo port.SquadVehicleRepository) *SessionPageService {
	s.vehicleRepo = repo
	return s
}

// WithSessionEmblemLoader injecte le chargeur d'emblèmes (celui des fiches de l'Escouade).
func (s *SessionPageService) WithSessionEmblemLoader(l port.EmblemURLLoader) *SessionPageService {
	s.emblemLoader = l
	return s
}

// WithSessionLives injecte le lecteur des vies (câblé sous film.kill_positions).
func (s *SessionPageService) WithSessionLives(repo port.SoloLivesRepository) *SessionPageService {
	s.livesRepo = repo
	return s
}

// WithSessionRadarRange injecte la table des portées de radar du titre (game_variant_name -> m).
func (s *SessionPageService) WithSessionRadarRange(parVariante map[string]int) *SessionPageService {
	s.radarRange = parVariante
	return s
}

// WithSessionObjectives injecte les colonnes d'objectif de la feuille « Ma part à l'objectif ».
func (s *SessionPageService) WithSessionObjectives(repo port.SquadFormesObjectiveRepository) *SessionPageService {
	s.formesObjectives = repo
	return s
}

// WithRoundsDecide injecte la table des variantes qui se décident en manches (score des lignes).
func (s *SessionPageService) WithRoundsDecide(roundsDecide map[string]bool) *SessionPageService {
	s.roundsDecide = roundsDecide
	return s
}

// lecturesDesSessions — le résumé d'usage lu UNE fois par session : la session affichée et, tiroir
// ouvert, la session comparée (nil = non lu : titre sans résumé, ou pas de comparaison).
type lecturesDesSessions struct {
	courant, compare *squadagg.LecturesUsage
}

// attachSessionBlocks pose les blocs du film sur la session affichée et, tiroir ouvert
// (`sc.CompareMatches` non vide), sur la session comparée. `canon` : l'historique canonique déjà
// chargé par la page (cartes, résultats, modes des matchs).
func (s *SessionPageService) attachSessionBlocks(
	ctx context.Context, resp *domain.SessionPageResponse, sc sessionBlocksScope, canon []canonical.PlayerMatchRow,
) {
	lus := lecturesDesSessions{courant: s.lireUsageDeSession(ctx, sc.Matches)}
	if len(sc.CompareMatches) > 0 {
		lus.compare = s.lireUsageDeSession(ctx, sc.CompareMatches)
	}
	s.attachSessionCoordination(ctx, resp, sc, s.effectifsDeCamp(lus.courant), s.effectifsDeCamp(lus.compare))
	s.attachSessionEmprise(ctx, resp, sc, canon, lus)
	s.attachSessionLives(ctx, resp, sc)
	s.attachSessionFormes(ctx, resp, sc, canon, lus)
	s.attachSessionEmblem(ctx, resp)
}

// lireUsageDeSession fait les trois lectures du résumé d'usage d'une session ; nil sans résumé
// (titre sans film.usage_summary) ou sans match — chaque bloc dégrade alors comme il le fait seul.
func (s *SessionPageService) lireUsageDeSession(ctx context.Context, matches []legacymatch.StatsMatchRow) *squadagg.LecturesUsage {
	if s.sessionUsageRepo == nil || len(matches) == 0 {
		return nil
	}
	defer timing.FromContext(ctx).Section("usage_summary")()
	return squadagg.LireUsage(ctx, s.sessionUsageRepo, matchIDsFromStatsRows(matches))
}

// effectifsDeCamp — l'effectif de mon camp par match (sessionusage.BuildTeamContext, la définition
// des autres blocs, réserve R1) ; nil sans lecture réussie ou sans joueur : la coordination n'a
// alors pas de parité, jamais une parité inventée.
func (s *SessionPageService) effectifsDeCamp(lu *squadagg.LecturesUsage) map[string]int {
	if lu == nil || lu.Erreur() != nil || s.sessionXUID == "" {
		return nil
	}
	return sessionusage.BuildTeamContext(s.sessionXUID, lu.Participants).TeamSize
}

// lignesCanoniquesDe — les lignes canoniques des matchs d'une session, dans l'ordre de la session.
// Un match sans ligne canonique est omis (rien à nommer pour lui).
func lignesCanoniquesDe(canon []canonical.PlayerMatchRow, matches []legacymatch.StatsMatchRow) []canonical.PlayerMatchRow {
	parID := make(map[string]int, len(canon))
	for i := range canon {
		parID[canon[i].Summary.MatchID] = i
	}
	out := make([]canonical.PlayerMatchRow, 0, len(matches))
	for _, m := range matches {
		if i, ok := parID[m.MatchID]; ok {
			out = append(out, canon[i])
		}
	}
	return out
}
