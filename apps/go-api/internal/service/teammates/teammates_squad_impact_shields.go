// Package teammates — teammates_squad_impact_shields.go : mesures de bouclier du film par match,
// source du rôle d'impact « Os de verre » (squadimpact.GlassJaw).
package teammates

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/squadimpact"
)

// loadShieldsByMatch lit les bris et recharges de bouclier des matchs et les rend par match puis
// par xuid. Un match absent du résultat n'est pas mesuré (sans rejeu, mode sans bouclier) et ne
// reçoit pas d'« Os de verre ». Lecture en échec : journalisée, aucun match mesuré, sans erreur
// (le reste de la matrice se publie).
func (s *TeammatesService) loadShieldsByMatch(
	ctx context.Context, matchIDs []string,
) map[string]map[string]squadimpact.ShieldCount {
	if len(matchIDs) == 0 {
		return nil
	}
	rows, err := s.repo.LoadMatchShields(ctx, matchIDs)
	if err != nil {
		slog.WarnContext(ctx, "teammates_impact_shields_load_failed", "matchs", len(matchIDs), "err", err)
		return nil
	}
	out := make(map[string]map[string]squadimpact.ShieldCount)
	for _, r := range rows {
		if out[r.MatchID] == nil {
			out[r.MatchID] = make(map[string]squadimpact.ShieldCount)
		}
		out[r.MatchID][r.XUID] = squadimpact.ShieldCount{Breaks: r.Breaks, Recharges: r.Recharges}
	}
	return out
}
