// Package service — fanout_service.go : enrichissement multi-joueur.
//
// Sprint 42 : après la sync du joueur A, identifie les joueurs B/C/D
// configurés dans db_profiles.json qui partagent des matchs communs
// et recalcule leur performance_score / session_id pour les matchs manquants.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// FanoutService identifie et enrichit les joueurs partageant des matchs communs.
type FanoutService struct {
	cfg     *config.AppConfig
	factory port.FanoutPlayerFactory
}

// NewFanoutService crée un FanoutService.
func NewFanoutService(cfg *config.AppConfig) *FanoutService {
	return &FanoutService{
		cfg:     cfg,
		factory: config.NewFanoutFactory(cfg),
	}
}

// BuildPlan identifie les joueurs configurés ayant des matchs communs
// avec les matchs nouvellement insérés.
func (s *FanoutService) BuildPlan(
	ctx context.Context,
	sourceGamertag string,
	insertedMatchIDs []string,
) (*domain.FanoutPlan, error) {
	if len(insertedMatchIDs) == 0 {
		return &domain.FanoutPlan{
			SourceGamertag: sourceGamertag,
			CreatedAt:      time.Now(),
		}, nil
	}

	players, err := s.cfg.LoadPlayers()
	if err != nil {
		return nil, fmt.Errorf("fanout BuildPlan: %w", err)
	}

	// Résoudre le joueur source via le port hexagonal.
	sourceRepo, err := s.factory.OpenForPlayer(ctx, sourceGamertag, ctxkeys.TitleSlug(ctx))
	if err != nil {
		return nil, fmt.Errorf("fanout BuildPlan resolve source: %w", err)
	}

	plan := &domain.FanoutPlan{
		SourceGamertag: sourceGamertag,
		MatchIDs:       insertedMatchIDs,
		CreatedAt:      time.Now(),
	}

	for _, p := range players {
		if p.Gamertag == sourceGamertag || p.IsDemo {
			continue
		}

		count, err := sourceRepo.CountCommonMatchesForXUID(ctx, p.XUID, insertedMatchIDs)
		if err != nil {
			slog.WarnContext(ctx, "fanout: erreur comptage matchs communs",
				"target", p.Gamertag, "err", err)
			continue
		}
		if count == 0 {
			continue
		}

		plan.Targets = append(plan.Targets, domain.FanoutTarget{
			Gamertag:     p.Gamertag,
			XUID:         p.XUID,
			CommonCount:  count,
			MissingCount: count, // sera affiné lors de l'exécution
		})
	}

	slog.InfoContext(ctx, "fanout plan construit",
		"source", sourceGamertag,
		"inserted_matches", len(insertedMatchIDs),
		"targets", len(plan.Targets),
	)

	return plan, nil
}

// Execute lance l'enrichissement pour tous les targets du plan.
// Pour chaque joueur cible, recalcule le performance_score des matchs communs
// manquants dans sa player_match_enrichment.
func (s *FanoutService) Execute(
	ctx context.Context,
	plan *domain.FanoutPlan,
) domain.FanoutResult {
	result := domain.FanoutResult{}
	if plan == nil || len(plan.Targets) == 0 {
		return result
	}

	for _, target := range plan.Targets {
		enriched, err := s.enrichTarget(ctx, target, plan.MatchIDs)
		if err != nil {
			result.Errors = append(result.Errors,
				fmt.Sprintf("%s: %v", target.Gamertag, err))
			continue
		}
		result.MatchesEnriched += enriched
		result.TargetsProcessed++

		slog.InfoContext(ctx, "fanout enrichi",
			"target", target.Gamertag,
			"matches_enriched", enriched,
		)
	}

	return result
}

// enrichTarget enrichit un joueur cible avec les matchs communs.
func (s *FanoutService) enrichTarget(
	ctx context.Context,
	target domain.FanoutTarget,
	matchIDs []string,
) (int, error) {
	repo, err := s.factory.OpenForPlayer(ctx, target.Gamertag, ctxkeys.TitleSlug(ctx))
	if err != nil {
		return 0, fmt.Errorf("resolve target %s: %w", target.Gamertag, err)
	}

	// Trouver les matchs déjà enrichis pour ce joueur.
	existing, err := repo.LoadExistingEnrichments(ctx, matchIDs)
	if err != nil {
		return 0, fmt.Errorf("load existing enrichments %s: %w", target.Gamertag, err)
	}

	missing := make([]string, 0, len(matchIDs))
	for _, mid := range matchIDs {
		if !existing[mid] {
			missing = append(missing, mid)
		}
	}

	if len(missing) == 0 {
		return 0, nil
	}

	// Insérer des enregistrements basiques pour les matchs manquants.
	// Le performance_score sera recalculé au prochain post-sync compute.
	enriched, err := repo.InsertStubEnrichments(ctx, target.XUID, missing)
	if err != nil {
		return 0, fmt.Errorf("insert enrichments %s: %w", target.Gamertag, err)
	}

	return enriched, nil
}
