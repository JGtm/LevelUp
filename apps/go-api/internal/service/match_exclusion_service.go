// Package service — MatchExclusionService : gestion des matchs non pertinents.
package service

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// MatchExclusionService implémente port.MatchExclusionService.
//
// Workflow SetExclusion :
//  1. Lecture match_registry → garde "match classé" + métadonnées
//  2. Lecture de l'état et décision sous le verrou d'écriture de la base joueur
//     (repo.SetExclusion) : sans changement d'état, rien n'est écrit
//  3. INSERT de is_excluded en player_match_enrichment (append-only), sous le même verrou
//  4. Recompute synchrone perf_score + LUSR (filtrage is_excluded déjà appliqué)
//
// Le recomputer est optionnel (peut être nil pour des tests bas niveau) ; si
// présent, son échec propage une erreur — l'utilisateur peut retenter : le flag est
// déjà persisté, et un recomputer qui sait que son dernier recalcul a échoué
// (RecomputePending) fait recalculer la demande répétée au lieu de la tenir pour un no-op.
type MatchExclusionService struct {
	repo       port.MatchExclusionRepository
	recomputer port.MatchRecomputer
}

// NewMatchExclusionService crée un MatchExclusionService.
// recomputer peut être nil (le service marquera l'exclusion sans recalcul).
func NewMatchExclusionService(
	repo port.MatchExclusionRepository,
	recomputer port.MatchRecomputer,
) *MatchExclusionService {
	return &MatchExclusionService{repo: repo, recomputer: recomputer}
}

// SetExclusion marque ou démarque un match comme non pertinent.
//
// Erreurs typées :
//   - domain.ErrMatchNotFound          → match_id absent de shared.match_registry
//   - domain.ErrRankedMatchNotExcludable → tentative d'exclure un match classé
func (s *MatchExclusionService) SetExclusion(ctx context.Context, matchID string, excluded bool) error {
	info, err := s.repo.GetMatchRegistryInfo(ctx, matchID)
	if err != nil {
		return err
	}
	if excluded && info.IsRanked {
		// Log Info : événement métier observable (refus garde), pas une erreur
		// applicative. Permet de mesurer combien d'utilisateurs heurtent la
		// règle (et donc combien la garde frontend les a déjà filtrés en amont).
		slog.InfoContext(ctx, "MatchExclusionService: exclusion refusée — match classé",
			"match_id", matchID, "pair_name", info.PairName)
		return domain.ErrRankedMatchNotExcludable
	}
	// Sans changement d'état : ni écriture (la table est append-only, chaque appel y
	// ajouterait une ligne) ni recalcul de tout l'historique du joueur. L'état est lu et
	// la décision prise par le dépôt, sous le verrou d'écriture de la base joueur.
	retry := s.recomputePending()
	changed, err := s.repo.SetExclusion(ctx, matchID, excluded)
	if err != nil {
		return fmt.Errorf("MatchExclusionService.SetExclusion upsert: %w", err)
	}
	if !changed && !retry {
		slog.DebugContext(ctx, "MatchExclusionService: état inchangé, rien à faire",
			"match_id", matchID, "excluded", excluded)
		return nil
	}
	if changed {
		slog.InfoContext(ctx, "MatchExclusionService: flag is_excluded mis à jour",
			"match_id", matchID, "excluded", excluded, "pair_name", info.PairName,
			"is_firefight", info.IsFirefight)
	}
	if s.recomputer == nil {
		return nil
	}
	if err := s.recomputer.RecomputeAfterExclusion(ctx, matchID); err != nil {
		slog.WarnContext(ctx, "MatchExclusionService: recompute échoué après UPSERT",
			"match_id", matchID, "excluded", excluded, "err", err)
		return fmt.Errorf("MatchExclusionService.SetExclusion recompute: %w", err)
	}
	return nil
}

// recomputePending dit si le dernier recalcul du joueur a échoué : le flag est alors déjà
// persisté mais l'historique n'a pas suivi, et la demande répétée doit recalculer.
func (s *MatchExclusionService) recomputePending() bool {
	p, ok := s.recomputer.(interface{ RecomputePending() bool })
	return ok && p.RecomputePending()
}
