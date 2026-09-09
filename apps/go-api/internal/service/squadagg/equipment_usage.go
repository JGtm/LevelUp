// Package squadagg — equipment_usage.go : L'ORCHESTRATION DU BLOC « SERVI OU
// GÂCHÉ » AU GRAIN PÉRIODE (étapes E5.5, E6.1 puis E6.1bis du
// PLAN_EQUIPEMENT_GACHIS_2026-09-09).
//
// UN SEUL ASSEMBLAGE POUR PLUSIEURS PAGES. La Synthèse (scope = les matchs du
// joueur sur la période filtrée) et l'Escouade/Teammates (scope = les matchs
// filtrés de la page) posent exactement la même question et lisent exactement
// les mêmes trois vues. Un builder par page aurait fait diverger deux fois la
// règle de camp et la résolution des amis — c'est le patron « helper de
// package » (jamais un service qui en appelle un autre : couplage horizontal,
// cf. skill arch-rules).
//
// POURQUOI DANS squadagg ET PAS service (E6.1bis, 2026-09-09) : le service-root
// (package service) importe déjà internal/service/teammates (synthesis_service_usage.go,
// pour teammates.FriendGamertagsResolver). Si TeammatesService avait besoin
// d'appeler une fonction du package service, ce serait un cycle service→teammates→
// service. squadagg est une FEUILLE (ne dépend ni de service ni de teammates) déjà
// importée des deux côtés — même patron que BuildSquadHeader/IntersectByMatchID,
// déplacés ici pour la même raison (K3b). Le package service continue d'appeler
// cette fonction via l'alias de squadagg_reexport.go : zéro site d'appel changé.
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
package squadagg

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// EquipmentUsageQuery — tout ce que l'assemblage demande à la page appelante : la
// source (nil ⇒ titre sans film.usage_summary), le joueur de la route, le scope
// FERMÉ de matchs, et les amis (coéquipiers suivis/sélectionnés ou amis
// configurés selon la page appelante) qui pourront devenir la part « mes amis »
// des deux donuts.
type EquipmentUsageQuery struct {
	Repo            port.SessionUsageRepository
	PlayerXUID      string
	MatchIDs        []string
	FriendGamertags []string
}

// BuildEquipmentUsageBlock lit le résumé d'usage sur le scope et rend le bloc
// contractuel. Scope vide ⇒ nil (rien à publier, pas même une indisponibilité).
func BuildEquipmentUsageBlock(ctx context.Context, q EquipmentUsageQuery) *domain.EquipmentUsageBlock {
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
