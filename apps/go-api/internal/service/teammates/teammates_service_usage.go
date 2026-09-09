// Package teammates — teammates_service_usage.go : LE BLOC « SERVI OU GÂCHÉ »
// DE L'ÉQUIPEMENT SUR LA PAGE TEAMMATES (étape E6.1bis du
// PLAN_EQUIPEMENT_GACHIS_2026-09-09).
//
// CORRIGE LA PUBLICATION E6.1 : le lot E6.1 avait posé ce bloc sur
// domain.SquadPageV2Response (GET /pages/squad/v2), une réponse que la page
// Escouade réellement servie en production ne fetch jamais (elle appelle
// POST /pages/teammates — cf. SquadLayout.tsx / features/squad/queries.ts).
// Ce fichier corrige le tir : même assemblage (squadagg.BuildEquipmentUsageBlock,
// partagé avec la Synthèse), mais attaché à TeammatesPageResponse.
//
// SCOPE ET AMIS (décision E6.1bis) : le scope est `filteredMatches` de GetPage —
// les matchs du joueur principal après période/cascade/sessions, LA MÊME
// population que Options/MatchHistory/TotalMatches — jamais l'intersection
// escouade (allSquadRows) : cette page reste utile même sans coéquipier
// sélectionné. Les « amis » du bloc sont les coéquipiers SÉLECTIONNÉS
// (req.SelectedGamertags), exactement comme le faisait E6.1 sur SquadV2 — pas les
// amis globalement configurés (app_settings), qui sont la définition retenue par
// la Synthèse pour une raison différente (aucune sélection UI sur cette page-là).
package teammates

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/squadagg"
)

// WithEquipmentUsage injecte la source du résumé d'usage (vues _latest) — le MÊME
// repo que les pages Sessions, Synthèse et (jusqu'à E6.1bis) Squad V2. Câblé
// gated par film.usage_summary (registry, jamais slug==) ; nil ⇒ bloc servi avec
// Available=false et raison machine, réponse partielle propre.
func (s *TeammatesService) WithEquipmentUsage(repo port.SessionUsageRepository) *TeammatesService {
	s.sessionUsageRepo = repo
	return s
}

// loadEquipmentUsage publie le bloc sur le scope FILTRÉ de la page (voir
// commentaire de fichier). playerXUID est le sujet de la page (le joueur
// principal, paramètre de route de GetPage) ; selectedGamertags sont les
// coéquipiers sélectionnés dans l'UI, qui deviennent les « amis » du bloc.
func (s *TeammatesService) loadEquipmentUsage(
	ctx context.Context, playerXUID string,
	filteredMatches []legacymatch.SynthesisMatchRow, selectedGamertags []string,
) *domain.EquipmentUsageBlock {
	return squadagg.BuildEquipmentUsageBlock(ctx, squadagg.EquipmentUsageQuery{
		Repo:            s.sessionUsageRepo,
		PlayerXUID:      playerXUID,
		MatchIDs:        teammatesMatchIDs(filteredMatches),
		FriendGamertags: selectedGamertags,
	})
}

// teammatesMatchIDs — les identifiants d'un scope de SynthesisMatchRow, dans
// l'ordre. Miroir de synthesisMatchIDs (service/synthesis_service_usage.go) :
// packages disjoints (K3b), même littéral de boucle qu'un seul champ — pas de
// troisième copie à ce jour côté teammates (règle CLAUDE.md n°6, 2 copies).
func teammatesMatchIDs(rows []legacymatch.SynthesisMatchRow) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.MatchID)
	}
	return out
}
