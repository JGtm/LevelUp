// Package service — match_history_service_briefing_ranked.go : module « Classement »
// du bandeau de briefing de l'Explorer (mode Matchs).
//
// Émet une entrée PAR CHAÎNE de playlist (rating_type, playlist_group) présente dans
// le scope : paliers début/fin (premier / dernier match noté de la chaîne, via
// SkillTierLabel déjà résolu FR ; placement signalé par flags D-D) + variation nette
// de rating ramenée au match (co-signée avec la progression — DEC-RANK-BE). CSR =
// chaîne unique « ranked » (P-3) ; LUSR se scinde en ses chaînes. Le calcul pur vit
// dans analysis.ComputeRankProgressionByChain ; ce fichier ne fait que projeter les
// raw rows en samples puis mapper le résultat vers le DTO. Le bloc « attendu vs réel »
// a été retiré (décision produit 2026-07-16). Extrait de match_history_service_
// briefing.go pour rester sous le seuil de taille de fichier (CLAUDE.md §5).
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// buildBriefingRanked émet le module « Classement » : une entrée PAR CHAÎNE
// (rating_type, playlist_group) présente dans le scope. L'appelant a déjà gaté sur
// rankedCapable (capability match.skill.snapshot, exclut H5). Retourne nil si aucune
// chaîne rangée datée dans le scope.
func buildBriefingRanked(ctx context.Context, scope []domain.MatchHistoryRawRow) *domain.ExplorerBriefingRanked {
	samples := rankChainSamples(scope)
	if len(samples) == 0 {
		return nil
	}
	progs := analysis.ComputeRankProgressionByChain(samples)
	if len(progs) == 0 {
		return nil
	}
	kinds := make([]domain.ExplorerBriefingRankedKind, 0, len(progs))
	for i := range progs {
		p := &progs[i]
		if p.TierStartLabel == nil && !p.TierStartIsPlacement && p.TierEndLabel == nil && p.TierEndPlacementRemaining == nil {
			// Chaîne sans aucun palier : progression omise (dégradation best-effort
			// documentée, jamais d'erreur avalée — CLAUDE.md §3).
			slog.DebugContext(ctx, "briefing ranked: no tier label for chain",
				"kind", p.RatingType, "playlist_group", p.PlaylistGroup, "matches", p.Matches)
		}
		kinds = append(kinds, domain.ExplorerBriefingRankedKind{
			Kind:                      p.RatingType,
			PlaylistGroup:             p.PlaylistGroup,
			Matches:                   p.Matches,
			TierStartLabel:            p.TierStartLabel,
			TierEndLabel:              p.TierEndLabel,
			TierStartIsPlacement:      p.TierStartIsPlacement,
			TierEndPlacementRemaining: p.TierEndPlacementRemaining,
			DeltaPerMatch:             p.DeltaPerMatch,
		})
	}
	return &domain.ExplorerBriefingRanked{Kinds: kinds}
}

// rankChainSamples projette les raw rows rangées et datées du scope en samples pour
// l'algo pur (title-agnostic). Une row est retenue si elle porte un type de rating et
// une date (chronologie). PlaylistGroup / RatingValue / RatingDelta viennent de
// match_skill_rank_latest ; les flags de placement de match_history_placement.
func rankChainSamples(scope []domain.MatchHistoryRawRow) []analysis.RankChainSample {
	samples := make([]analysis.RankChainSample, 0, len(scope))
	for i := range scope {
		r := &scope[i]
		if r.SkillRatingType == nil || r.StartTime == nil {
			continue
		}
		samples = append(samples, analysis.RankChainSample{
			RatingType:     *r.SkillRatingType,
			PlaylistGroup:  r.PlaylistGroup,
			StartTime:      *r.StartTime,
			TierLabel:      r.SkillTierLabel,
			RatingValue:    r.RatingValue,
			RatingDelta:    r.RatingDelta,
			PlacementDone:  r.PlacementDone,
			PlacementTotal: r.PlacementTotal,
		})
	}
	return samples
}
