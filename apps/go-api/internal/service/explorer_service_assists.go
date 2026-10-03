// Package service — explorer_service_assists.go : les assistances échangées avec la
// cible de l'Explorer (bloc « Part des assistances » de la section « matchs joués
// ensemble »).
//
// MÊME agrégat que la carte Binôme du hub Relations : un seul lecteur
// (`GetRelationAssists`), une seule doctrine (domain/relation_assists.go). L'Explorer
// n'affiche qu'UNE paire, mais lit la map entière des coéquipiers pour deux raisons :
//   - l'entrée de la cible en est extraite sans requête supplémentaire ;
//   - la borne de l'échelle logarithmique des barres papillon est le plus gros volume
//     d'un sens parmi TOUTES les relations mesurées. Se borner à la paire affichée
//     remplirait toujours la demi-barre — échec constaté et documenté côté web
//     (apps/web/src/features/_shared/assists/assistExchange.ts).
//
// Titre sans décodeur de film : aucune ligne mesurée, la map revient vide, le bloc
// affiche « — ». Aucun branchement par slug (ADR 0025).
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
)

// enrichEncounterAssists remplit Assists (paire joueur↔cible) et AssistVolumeMax
// (borne d'échelle) sur les stats de rencontre. Best-effort strict : une erreur du
// repo est loggée puis ignorée, la réponse reste servie sans le bloc.
func (s *ExplorerService) enrichEncounterAssists(ctx context.Context, stats *domain.ExplorerEncounterStats, otherXUID string) {
	if stats == nil || s.deps.Relations == nil || otherXUID == "" {
		return
	}
	byXUID, err := s.deps.Relations.GetRelationAssists(ctx, nil)
	if err != nil {
		slog.WarnContext(ctx, "explorer_relation_assists_failed",
			"xuid", s.xuid, "other_xuid", otherXUID, "err", err)
		return
	}
	stats.AssistVolumeMax = explorerAssistVolumeMax(byXUID)
	if a, ok := byXUID[otherXUID]; ok {
		stats.Assists = &a
	}
}

// explorerAssistVolumeMax : plus gros volume d'un sens (reçues ou données) parmi toutes
// les relations mesurées. Miroir Go de `assistVolumeMax` côté web. 0 si map vide.
func explorerAssistVolumeMax(byXUID map[string]domain.RelationAssists) int {
	maxVolume := 0
	for _, a := range byXUID {
		if a.Received.Total > maxVolume {
			maxVolume = a.Received.Total
		}
		if a.Given.Total > maxVolume {
			maxVolume = a.Given.Total
		}
	}
	return maxVolume
}
