// Package service — explorer_service_relations.go : les agrégats relationnels de la
// section « matchs joués ensemble » de l'Explorer (repère des donuts, écart de frags
// cumulé, assistances), lus par le hub Relations.
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/timing"
)

// explorerFragGapTimelineLimit : nombre de duels (matchs en ennemi) conservés
// pour la courbe « écart de frags cumulé » de la section « matchs joués ensemble »
// — aligné sur momentsTimelineLimit du hub Relations (mêmes N derniers duels).
const explorerFragGapTimelineLimit = 20

// enrichEncounterRelations complète best-effort la section « matchs joués
// ensemble » : PlayerWinRate (repère « moyenne perso » des donuts, WR historique
// du joueur) + FragGapSeries (écart de frags cumulé duel par duel contre la cible)
// + Assists (« Part des assistances », cf. explorer_service_assists.go).
//
// commonIDs : les matchs communs déjà lus. Un duel est un match commun joué en ennemi :
// la timeline est lue sur ce périmètre (ADR 0036 I2), ce qui rend les mêmes duels que la
// lecture sur tout l'historique sans payer la fenêtre du kill-feed sur la table entière.
// no-op si stats nil (aucun match commun) ou provider non injecté. Chaque source
// est indépendante : un échec est loggé puis ignoré (dégradation gracieuse), la
// réponse reste servie sans le repère / le graphe.
func (s *ExplorerService) enrichEncounterRelations(ctx context.Context, stats *domain.ExplorerEncounterStats, otherXUID string, commonIDs []string) {
	if stats == nil || s.deps.Relations == nil {
		return
	}
	// WR historique perso (tout-temps) — repère des donuts. GetCoreEngagement avec
	// noyau vide ne calcule QUE le WR (la forme récente court-circuite).
	stopWinRate := timing.FromContext(ctx).Section("explorer_win_rate")
	eng, err := s.deps.Relations.GetCoreEngagement(ctx, nil, nil, 0)
	stopWinRate()
	if err != nil {
		slog.WarnContext(ctx, "explorer_player_win_rate_failed", "xuid", s.xuid, "err", err)
	} else {
		stats.PlayerWinRate = eng.PlayerWinRate
	}
	// Écart de frags cumulé — timeline des duels (matchs en ennemi contre la cible).
	if otherXUID != "" {
		stopDuels := timing.FromContext(ctx).Section("explorer_rival_timeline")
		duels, err := s.deps.Relations.GetRivalTimeline(ctx, otherXUID, commonIDs, explorerFragGapTimelineLimit)
		stopDuels()
		if err != nil {
			slog.WarnContext(ctx, "explorer_frag_gap_timeline_failed", "other_xuid", otherXUID, "err", err)
		} else {
			stats.FragGapSeries = buildExplorerFragGapSeries(duels)
		}
	}
	// Assistances échangées avec la cible + borne d'échelle des barres papillon
	// (bloc « Part des assistances »). Même best-effort que ci-dessus.
	s.enrichEncounterAssists(ctx, stats, otherXUID)
}
