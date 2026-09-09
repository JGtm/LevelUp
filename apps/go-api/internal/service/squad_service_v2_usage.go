package service

// squad_service_v2_usage.go — LE BLOC « SERVI OU GÂCHÉ » DE LA PAGE ESCOUADE
// (étape E6.1 du PLAN_EQUIPEMENT_GACHIS_2026-09-09).
//
// MÊME PUBLICATION QUE LA SYNTHÈSE, AGRÉGÉE PAR JOUEUR SUIVI : le bloc, son
// assemblage et ses trois lectures sont ceux de equipment_usage_block.go — seul
// change le scope (les matchs PARTAGÉS de la composition) et la liste des joueurs
// suivis (les coéquipiers SÉLECTIONNÉS dans l'UI).
//
// POURQUOI LES COÉQUIPIERS SÉLECTIONNÉS PASSENT PAR LA RESTRICTION D'AMIS. Le
// résolveur du grain période (sessionusage.ResolveScopeFriends) prend une liste de
// gamertags et rend les alliés correspondants AVEC leur xuid. Sur l'Escouade cette
// liste est la sélection de l'utilisateur : le résolveur n'invente donc personne,
// il fait la jointure gamertag -> xuid contre match_participants et écarte le cas
// réel où un « coéquipier » sélectionné se trouve en face sur tous les matchs
// partagés (l'intersection porte sur les match_id, jamais sur le camp).

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// WithEquipmentUsage injecte la source du résumé d'usage (vues _latest) — le MÊME
// repo que les pages Sessions et Synthèse. Câblé gated par film.usage_summary
// (SquadV2Ctx, jamais slug==) ; nil ⇒ bloc servi avec Available=false et raison
// machine, réponse partielle propre.
func (s *SquadServiceV2) WithEquipmentUsage(repo port.SessionUsageRepository) *SquadServiceV2 {
	s.sessionUsageRepo = repo
	return s
}

// loadEquipmentUsage publie le bloc sur les matchs PARTAGÉS de la composition.
// Le joueur de la route est le main de la page ; ses coéquipiers suivis sont les
// gamertags sélectionnés.
func (s *SquadServiceV2) loadEquipmentUsage(
	ctx context.Context, in squadUsageScope,
) *domain.EquipmentUsageBlock {
	return buildEquipmentUsageBlock(ctx, equipmentUsageQuery{
		Repo:            s.sessionUsageRepo,
		PlayerXUID:      in.MainXUID,
		MatchIDs:        in.MatchIDs,
		FriendGamertags: in.TeammateGTs,
	})
}

// squadUsageScope — ce que la page Escouade connaît de son scope au moment où le
// bloc se calcule (le xuid du main est résolu par extractSquadXUIDs, il n'est pas
// injecté à la DI : le service Escouade travaille par gamertag).
type squadUsageScope struct {
	MainXUID    string
	MatchIDs    []string
	TeammateGTs []string
}
