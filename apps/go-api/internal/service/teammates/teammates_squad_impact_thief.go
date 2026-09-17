// Package teammates — teammates_squad_impact_thief.go : badge d'impact « Voleur » de la
// matrice teammates.07 (critère : analysis.ComputeThiefBadge).
package teammates

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis"
)

// loadThiefBadgesByMatch charge le journal des morts des matchs et rend, par match, le
// badge « Voleur » de l'escouade (nil si aucun vol). xuidToGT définit l'escouade.
//
// Titre sans décodeur de film ou matchs sans film : aucune ligne → aucun badge, sans
// erreur (le repo dégrade déjà en loggant).
func (s *TeammatesService) loadThiefBadgesByMatch(
	ctx context.Context,
	matchIDs []string,
	xuidToGT map[string]string,
) map[string]*analysis.ImpactBadge {
	if len(matchIDs) == 0 || len(xuidToGT) == 0 {
		return nil
	}
	friends := make(map[string]bool, len(xuidToGT))
	squadXUIDs := make([]string, 0, len(xuidToGT))
	for xuid := range xuidToGT {
		friends[xuid] = true
		squadXUIDs = append(squadXUIDs, xuid)
	}
	rows, err := s.repo.LoadSquadKillLog(ctx, matchIDs, squadXUIDs)
	if err != nil {
		slog.WarnContext(ctx, "teammates_impact_thief_load_failed", "matchs", len(matchIDs), "err", err)
		return nil
	}
	entriesByMatch := make(map[string][]analysis.KillLogEntry)
	for _, r := range rows {
		entriesByMatch[r.MatchID] = append(entriesByMatch[r.MatchID], analysis.KillLogEntry{
			TimeMS:          r.TimeMS,
			KillerXUID:      r.KillerXUID,
			VictimXUID:      r.VictimXUID,
			AssistXUID:      r.AssistXUID,
			KillerDamagePct: r.KillerDamagePct,
		})
	}
	out := make(map[string]*analysis.ImpactBadge, len(entriesByMatch))
	for mid, entries := range entriesByMatch {
		if b := analysis.ComputeThiefBadge(entries, friends); b != nil {
			out[mid] = b
		}
	}
	return out
}
