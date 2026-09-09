// Package service — equipment_usage_block.go : L'ORCHESTRATION DU BLOC « SERVI OU
// GÂCHÉ » AU GRAIN PÉRIODE (étapes E5.5 et E6.1 du
// PLAN_EQUIPEMENT_GACHIS_2026-09-09).
//
// UN SEUL ASSEMBLAGE POUR DEUX PAGES. La Synthèse (scope = les matchs du joueur
// sur la période filtrée) et l'Escouade (scope = les matchs partagés de la
// composition) posent exactement la même question et lisent exactement les mêmes
// trois vues. Un builder par page aurait fait diverger deux fois la règle de camp
// et la résolution des amis — c'est le patron « helper de package » (jamais un
// service qui en appelle un autre : couplage horizontal, cf. skill arch-rules).
//
// AUCUN SQL ICI, AUCUNE REQUÊTE NEUVE AILLEURS : les trois lectures sont celles de
// la page Sessions (port.SessionUsageRepository → duckdb.SessionUsageRepo), qui
// prennent déjà un scope FERMÉ de match_id — la seule chose qui change d'une page
// à l'autre est la liste d'identifiants qu'on leur passe.
//
// BEST-EFFORT ET DIT : capability absente (repo non câblé) ou lecture en échec ⇒
// bloc présent avec une raison MACHINE, jamais un 500 ni un bloc muet. Les
// constantes de raison sont celles du bloc de session (domain.SessionUsage*) : un
// second vocabulaire pour les mêmes deux états n'aurait rien dit de plus.
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// equipmentUsageQuery — tout ce que l'assemblage demande à la page appelante : la
// source (nil ⇒ titre sans film.usage_summary), le joueur de la route, le scope
// FERMÉ de matchs, et les amis configurés qui pourront devenir la part « mes
// amis » des deux donuts.
type equipmentUsageQuery struct {
	Repo            port.SessionUsageRepository
	PlayerXUID      string
	MatchIDs        []string
	FriendGamertags []string
}

// buildEquipmentUsageBlock lit le résumé d'usage sur le scope et rend le bloc
// contractuel. Scope vide ⇒ nil (rien à publier, pas même une indisponibilité).
func buildEquipmentUsageBlock(ctx context.Context, q equipmentUsageQuery) *domain.EquipmentUsageBlock {
	if len(q.MatchIDs) == 0 {
		return nil
	}
	if q.Repo == nil || q.PlayerXUID == "" {
		return &domain.EquipmentUsageBlock{
			UnavailableReason: domain.SessionUsageUnsupported, MatchesTotal: len(q.MatchIDs),
		}
	}
	films, filmsErr := q.Repo.LoadUsageFilms(ctx, q.MatchIDs)
	players, playersErr := q.Repo.LoadUsagePlayers(ctx, q.MatchIDs)
	participants, partErr := q.Repo.LoadParticipants(ctx, q.MatchIDs)
	for _, err := range []error{filmsErr, playersErr, partErr} {
		if err != nil {
			slog.ErrorContext(ctx, "equipment usage: lecture du résumé d'usage en échec",
				"err", err, "match_count", len(q.MatchIDs))
			return &domain.EquipmentUsageBlock{
				UnavailableReason: domain.SessionUsageLoadFailed, MatchesTotal: len(q.MatchIDs),
			}
		}
	}

	tc := sessionusage.BuildTeamContext(q.PlayerXUID, participants)
	friends := sessionusage.ResolveScopeFriends(q.PlayerXUID, participants, q.FriendGamertags)
	in := sessionusage.OverviewInput{
		PlayerXUID: q.PlayerXUID,
		Matches:    sessionusage.BuildMatchInputs(q.MatchIDs, films, players, tc),
	}
	for _, f := range friends {
		in.FriendXUIDs = append(in.FriendXUIDs, f.XUID)
	}
	block := sessionusage.ComputeUsageOverview(in)
	block.TrackedPlayers = friends
	// La couverture des films n'est jamais totale : un scope entier sans match
	// mesuré est un état légitime (le bloc dit « 0/N »), mais il ne doit pas
	// passer en silence — c'est le premier symptôme d'une recuisson manquante.
	if block.MatchesMeasured == 0 {
		slog.InfoContext(ctx, "equipment usage: aucun match du scope n'est mesuré",
			"match_count", len(q.MatchIDs), "player_xuid", q.PlayerXUID)
	}
	return &block
}
