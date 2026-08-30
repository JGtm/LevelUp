// Package service — home_service_enrichment.go : helpers d'enrichissement des
// RecentMatchItem (médailles + citations), favoris et lien rejeu 2D. Extrait de
// home_service.go (refactor god-file, revue 2026-06-02).
package service

import (
	"context"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

func buildFavoriteMatchListCanonical(
	rows []canonical.PlayerMatchRow,
	favoriteIDs map[string]bool,
	opts analysis.RecentMatchesOptions,
) []domain.RecentMatchItem {
	if len(favoriteIDs) == 0 {
		return nil
	}
	allItems := analysis.BuildRecentMatchesWithFavoritesFromCanonical(rows, len(rows), favoriteIDs, opts)
	var favorites []domain.RecentMatchItem
	for _, item := range allItems {
		if item.IsFavorite {
			favorites = append(favorites, item)
		}
	}
	return favorites
}

// WithReplay injecte le service de rejeu 2D — même contrat que MatchHistoryService :
// les tuiles de match de l'Accueil portent has_replay pour poser le lien vers la page
// de rejeu. Dégradation gracieuse si nil. Retourne le service (chaînage).
func (s *HomeService) WithReplay(svc port.ReplayService) *HomeService {
	s.replaySvc = svc
	return s
}

// replayAvailability liste les matchs ayant un artefact de rejeu — UN listing de
// dossier par requête. Service non câblé ou listing en échec (déjà journalisé par le
// service de rejeu) : ensemble vide, les tuiles se servent sans lien plutôt qu'en 500.
func (s *HomeService) replayAvailability(ctx context.Context) port.ReplayAvailability {
	if s.replaySvc == nil {
		return nil
	}
	set, err := s.replaySvc.AvailableSet(ctx)
	if err != nil {
		return nil
	}
	return set
}

// applyReplayAvailabilityToRecentItems pose HasReplay sur les tuiles de match (récents
// + favoris) depuis l'ensemble résolu une fois par requête. Ensemble vide/nil = no-op.
func applyReplayAvailabilityToRecentItems(replays port.ReplayAvailability, itemLists ...[]domain.RecentMatchItem) {
	if len(replays) == 0 {
		return
	}
	for _, items := range itemLists {
		for i := range items {
			items[i].HasReplay = replays.Has(items[i].MatchID)
		}
	}
}

// enrichMatchesWithMedals injecte les TopMedals (max 4, sÃ©lection par raretÃ©/count)
// dans chaque RecentMatchItem via un appel batch sur le repo.
func enrichMatchesWithMedals(ctx context.Context, repo port.HomeRepository, items []domain.RecentMatchItem) {
	if len(items) == 0 {
		return
	}
	matchIDs := make([]string, len(items))
	for i, item := range items {
		matchIDs[i] = item.MatchID
	}
	medalsMap, err := repo.LoadMatchMedals(ctx, matchIDs)
	if err != nil || len(medalsMap) == 0 {
		return
	}
	for i, item := range items {
		if all, ok := medalsMap[item.MatchID]; ok {
			items[i].TopMedals = selectTopMedals(all, 4)
		}
	}
}

// medalDifficultyWeight retourne le poids de tri d'une difficulte de medaille.
// Mythic > Legendary > Heroic > Normal (ou vide).
func medalDifficultyWeight(d string) int {
	switch d {
	case "Mythic":
		return 3
	case "Legendary":
		return 2
	case "Heroic":
		return 1
	default:
		return 0
	}
}

// selectTopMedals selectionne au plus n medailles, en privilegiant
// la rarete (Mythic > Legendary > Heroic > Normal) puis le count.
func selectTopMedals(medals []domain.RecentMatchMedal, n int) []domain.RecentMatchMedal {
	if len(medals) == 0 {
		return nil
	}
	sorted := make([]domain.RecentMatchMedal, len(medals))
	copy(sorted, medals)
	sort.Slice(sorted, func(i, j int) bool {
		wi := medalDifficultyWeight(sorted[i].Difficulty)
		wj := medalDifficultyWeight(sorted[j].Difficulty)
		if wi != wj {
			return wi > wj
		}
		return sorted[i].Count > sorted[j].Count
	})
	if len(sorted) <= n {
		return sorted
	}
	return sorted[:n]
}

// maxCitationSnippets est le nombre maximum de citations affichÃ©es par MatchCard.
const maxCitationSnippets = 3

// enrichMatchesWithCitations injecte les TopCitations (max 3, filtre citations dÃ©jÃ  masterisÃ©es)
// dans chaque RecentMatchItem via un appel batch sur le repo.
func enrichMatchesWithCitations(ctx context.Context, repo port.HomeRepository, items []domain.RecentMatchItem) {
	if len(items) == 0 {
		return
	}
	matchIDs := make([]string, len(items))
	for i, item := range items {
		matchIDs[i] = item.MatchID
	}
	citationsMap, err := repo.LoadMatchCitations(ctx, matchIDs)
	if err != nil || len(citationsMap) == 0 {
		return
	}
	for i, item := range items {
		if rows, ok := citationsMap[item.MatchID]; ok && len(rows) > 0 {
			items[i].TopCitations = analysis.BuildCitationSnippets(rows, maxCitationSnippets)
		}
	}
}

// enrichMatchesWithCommendations remplit le slot TopCitations à partir des
// commendations NATIVES par match (Halo 5 : shared.match_commendations) pour les
// items dont TopCitations est encore VIDE après enrichMatchesWithCitations.
//
// DÉCISION TITLE-AGNOSTIC (P7) : plutôt qu'un gating `slug == "halo_5"`, le wiring
// appelle citations PUIS commendations en fallback. Halo Infinite remplit TopCitations
// via le moteur de citations dérivé (LoadMatchCitations) → les commendations natives
// (table vide pour Infinite) sont un no-op. Halo 5 n'a pas de moteur de citations
// (citations.engine = not_exposed) → LoadMatchCitations renvoie vide → les
// commendations natives (commendations.native = supported) alimentent le MÊME slot,
// SANS changement frontend ni OpenAPI (réutilisation de MatchCitationSnippet).
//
// PROGRESSION (S6) : quand la commendation porte des paliers (tier_targets seedés
// depuis levels[].threshold) ET un cumul à vie (progress), on calcule
// ProgressPct/Cumulative/Tier*/IsNewlyMastered EXACTEMENT comme les citations Infinite
// (analysis.ComputeTierProgression, mécanique partagée). Sans paliers/cumul (Meta/Daily,
// ou définition pré-tier_targets), tout reste à zéro → anneau vide, dégradation propre
// confirmée côté CitationProgressRing. Name + ImageURL (icône CDN) + Delta (count gagné)
// suffisent au rendu minimal de la tuile.
func enrichMatchesWithCommendations(ctx context.Context, repo port.HomeRepository, items []domain.RecentMatchItem) {
	if len(items) == 0 {
		return
	}
	// Ne charger que pour les matchs dont TopCitations est encore vide (Infinite a
	// déjà rempli via citations dérivées → on ne réinterroge pas inutilement).
	matchIDs := make([]string, 0, len(items))
	for _, item := range items {
		if len(item.TopCitations) == 0 {
			matchIDs = append(matchIDs, item.MatchID)
		}
	}
	if len(matchIDs) == 0 {
		return
	}
	commMap, err := repo.LoadMatchCommendations(ctx, matchIDs)
	if err != nil || len(commMap) == 0 {
		return
	}
	for i, item := range items {
		if len(items[i].TopCitations) > 0 {
			continue // citations dérivées déjà présentes (Infinite) → ne pas écraser
		}
		rows, ok := commMap[item.MatchID]
		if !ok || len(rows) == 0 {
			continue
		}
		items[i].TopCitations = buildCommendationSnippets(rows, maxCitationSnippets)
	}
}

// buildCommendationSnippets projette des commendations natives par match en
// MatchCitationSnippet (slot réutilisé). Tri count DESC (déjà appliqué par le repo,
// re-trié ici par sûreté), max `limit`. Progrès/tier/masterisé calculés via la même
// mécanique pure que les citations Infinite (analysis.ComputeTierProgression) à partir
// du cumul à vie (Progress) + paliers (TierTargets) + delta du match (Count).
func buildCommendationSnippets(rows []domain.HomeMatchCommendationRaw, limit int) []domain.MatchCitationSnippet {
	if len(rows) == 0 || limit <= 0 {
		return nil
	}
	sorted := make([]domain.HomeMatchCommendationRaw, len(rows))
	copy(sorted, rows)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Count > sorted[j].Count })
	// Masquage des commendations déjà maîtrisées AVANT ce match — parité STRICTE avec
	// les citations Halo Infinite (analysis.BuildCitationSnippets) et la match-view
	// (match_view_canonical_citations). Le filtre AlreadyMastered s'applique AVANT la
	// borne `limit` (sinon une commendation maîtrisée occuperait un slot du top-N et
	// évincerait une commendation valide au-delà de la borne). On ne montre que celles
	// progressées/nouvellement maîtrisées CE match.
	out := make([]domain.MatchCitationSnippet, 0, limit)
	for _, r := range sorted {
		tiers := analysis.ParseTierTargets(r.TierTargets)
		tp := analysis.ComputeTierProgression(r.Progress, r.Count, tiers)
		if tp.AlreadyMastered {
			continue
		}
		var imgURL *string
		if r.IconURL != "" {
			u := r.IconURL
			imgURL = &u
		}
		out = append(out, domain.MatchCitationSnippet{
			Key:             r.ID,
			Name:            r.Name,
			ImageURL:        imgURL,
			Delta:           r.Count,
			ProgressPct:     tp.ProgressPct,
			IsNewlyMastered: tp.IsNewlyMastered,
			Cumulative:      r.Progress,
			TierIndex:       tp.TierIndex,
			TierCount:       tp.TierCount,
			NextTierTarget:  tp.NextTierTarget,
		})
		if len(out) >= limit {
			break // borne appliquée APRÈS le filtre (parité Infinite)
		}
	}
	return out
}

// GetBattlePass retourne les infos Battle Pass (live d'abord, cache DB en fallback).
// Appel live systÃ©matique pour garantir des donnÃ©es fraÃ®ches au rechargement de page.
// Si le live Ã©choue (tokens absents, API indisponible), le cache DB est retournÃ©.
// Si un PersistSink est configurÃ© et que le live rÃ©ussit, les donnÃ©es sont persistÃ©es
// de maniÃ¨re synchrone avant le retour (garantit que loadTrackSnapshots lit un rang Ã  jour).
