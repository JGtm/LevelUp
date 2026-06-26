// Package service — match_view_canonical_citations.go : voie CITATIONS de la
// Match View canonique (Halo 5, AXE B).
//
// Extrait de match_view_canonical.go (audit god files, limite projet 500 L).
// Couvre la résolution des commendations NATIVES d'un match servi live et leur
// projection vers l'onglet Citations avec PARITÉ Infinite (anneau de progression,
// masterisé pendant le match, masquage des pré-masterisées).
//
// Source primaire = shared.match_commendations via le CitationsRepo (progress
// cumulatif + tier_targets) ; fallback = les commendations brutes du détail live.
// NO-OP sur Infinite : ce chemin n'est emprunté que par les titres live-only.
package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// loadCanonicalCommendations résout les commendations NATIVES du match pour le
// viewer, INDÉPENDAMMENT du live. Source primaire = shared.match_commendations via
// le CitationsRepo (porte progress cumulatif + tier_targets → anneau + masterisé,
// parité Infinite). Fallback = les commendations brutes du détail live (sans
// progress : anneau vide, dégradation gracieuse) si le repo est absent / vide.
//
// NO-OP sur Infinite : ce chemin n'est emprunté que par les titres live-only (Halo
// 5) ; pour Infinite la table match_commendations est vide → repo vide → on retombe
// sur detail.Commendations (lui aussi vide) → onglet citations vide comme avant.
func (s *MatchViewService) loadCanonicalCommendations(ctx context.Context, detail *canonical.MatchDetail) []canonical.Commendation {
	if s.citationsRepo != nil && strings.TrimSpace(detail.MatchID) != "" && strings.TrimSpace(s.xuid) != "" {
		rows, err := s.citationsRepo.LoadMatchCommendationsRich(ctx, detail.MatchID, s.xuid)
		if err != nil {
			slog.WarnContext(ctx, "match_view: chargement commendations natives échoué (dégradation live)",
				"err", err, "match_id", detail.MatchID, "xuid", s.xuid)
		} else if len(rows) > 0 {
			return canonicalCommendationsFromRich(rows)
		}
	}
	return detail.Commendations
}

// canonicalCommendationsFromRich convertit les rows DB (progress + tier_targets)
// en canonical.Commendation enrichies. Donnée match-scoped déjà filtrée count>0.
func canonicalCommendationsFromRich(rows []domain.HomeMatchCommendationRaw) []canonical.Commendation {
	out := make([]canonical.Commendation, 0, len(rows))
	for _, r := range rows {
		var icon *string
		if r.IconURL != "" {
			u := r.IconURL
			icon = &u
		}
		out = append(out, canonical.Commendation{
			ID:          r.ID,
			Name:        r.Name,
			Count:       r.Count,
			IconURL:     icon,
			Progress:    r.Progress,
			TierTargets: r.TierTargets,
		})
	}
	return out
}

// buildCanonicalCitationsTab projette les commendations NATIVES (Halo 5) vers
// l'onglet Citations avec PARITÉ Infinite : anneau de progression (ProgressPct +
// TierIndex/TierCount), masterisé PENDANT le match (IsNewlyMastered → anneau doré
// + check côté front), et MASQUAGE des commendations masterisées AVANT le match
// (palier final déjà franchi → rien de neuf à montrer). Tri count DESC.
//
// Retourne aussi un flag « indisponible » (aucune commendation à montrer après
// filtrage) qui pilote partialReasonCitations : si des commendations restent, la
// section n'est plus signalée comme manquante.
func buildCanonicalCitationsTab(comms []canonical.Commendation) (domain.MatchCitationsTab, bool) {
	if len(comms) == 0 {
		return domain.MatchCitationsTab{}, true
	}
	sorted := make([]canonical.Commendation, len(comms))
	copy(sorted, comms)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Count > sorted[j].Count })

	native := make([]domain.MatchNativeCommendation, 0, len(sorted))
	for _, c := range sorted {
		tiers := analysis.ParseTierTargets(c.TierTargets)
		tp := analysis.ComputeTierProgression(c.Progress, c.Count, tiers)
		if tp.AlreadyMastered {
			// Masterisée AVANT ce match → masquée (parité Infinite : rien de neuf).
			continue
		}
		native = append(native, domain.MatchNativeCommendation{
			ID:              c.ID,
			Name:            c.Name,
			Count:           c.Count,
			IconURL:         c.IconURL,
			ProgressPct:     tp.ProgressPct,
			IsNewlyMastered: tp.IsNewlyMastered,
			Cumulative:      c.Progress,
			TierIndex:       tp.TierIndex,
			TierCount:       tp.TierCount,
			NextTierTarget:  tp.NextTierTarget,
		})
	}
	if len(native) == 0 {
		// Toutes pré-masterisées → onglet vide, section signalée manquante.
		return domain.MatchCitationsTab{}, true
	}
	return domain.MatchCitationsTab{NativeCommendations: native}, false
}
