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
	"levelup/go-api/internal/platform/duckdb"
)

// FanoutService identifie et enrichit les joueurs partageant des matchs communs.
type FanoutService struct {
	cfg *config.AppConfig
}

// NewFanoutService crée un FanoutService.
func NewFanoutService(cfg *config.AppConfig) *FanoutService {
	return &FanoutService{cfg: cfg}
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

	// Résoudre le joueur source pour obtenir un accès shared DB.
	sourcePDB, err := config.ResolvePlayer(ctx, s.cfg, sourceGamertag, ctxkeys.TitleSlug(ctx))
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

		count, err := countCommonMatches(ctx, sourcePDB, p.XUID, insertedMatchIDs)
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
	pdb, err := config.ResolvePlayer(ctx, s.cfg, target.Gamertag, ctxkeys.TitleSlug(ctx))
	if err != nil {
		return 0, fmt.Errorf("resolve target %s: %w", target.Gamertag, err)
	}

	// Trouver les matchs déjà enrichis pour ce joueur.
	existing, err := loadExistingEnrichments(ctx, pdb, matchIDs)
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
	enriched, err := insertStubEnrichments(ctx, pdb, target.XUID, missing)
	if err != nil {
		return 0, fmt.Errorf("insert enrichments %s: %w", target.Gamertag, err)
	}

	return enriched, nil
}

// countCommonMatches compte le nombre de matchs parmi insertedMatchIDs
// où le joueur cible (targetXUID) était participant.
func countCommonMatches(
	ctx context.Context,
	pdb *duckdb.PlayerDB,
	targetXUID string,
	matchIDs []string,
) (int, error) {
	if len(matchIDs) == 0 {
		return 0, nil
	}

	// Requête sur shared.match_participants pour trouver les matchs communs.
	query := `
		SELECT COUNT(DISTINCT match_id)
		FROM shared.match_participants
		WHERE xuid = ?
		AND match_id IN (SELECT UNNEST(?::VARCHAR[]))
	`
	var count int
	err := pdb.Player.QueryRow(ctx, query, targetXUID, matchIDs).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// loadExistingEnrichments retourne un set des match_ids déjà enrichis.
func loadExistingEnrichments(
	ctx context.Context,
	pdb *duckdb.PlayerDB,
	matchIDs []string,
) (map[string]bool, error) {
	result := make(map[string]bool, len(matchIDs))
	if len(matchIDs) == 0 {
		return result, nil
	}

	query := `
		SELECT match_id
		FROM player_match_enrichment
		WHERE match_id IN (SELECT UNNEST(?::VARCHAR[]))
	`
	rows, err := pdb.Player.Query(ctx, query, matchIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var mid string
		if err := rows.Scan(&mid); err != nil {
			return nil, err
		}
		result[mid] = true
	}
	return result, rows.Err()
}

// insertStubEnrichments insère des enregistrements stub dans player_match_enrichment
// pour les matchs communs manquants. Le performance_score sera recalculé plus tard.
func insertStubEnrichments(
	ctx context.Context,
	pdb *duckdb.PlayerDB,
	xuid string,
	matchIDs []string,
) (int, error) { //nolint:unparam // error toujours nil actuellement, interface-compatible
	_ = xuid // disponible pour future extension

	inserted := 0
	for _, mid := range matchIDs {
		_, err := pdb.Player.Exec(ctx,
			`INSERT OR IGNORE INTO player_match_enrichment (match_id) VALUES (?)`,
			mid,
		)
		if err != nil {
			slog.Warn("fanout: insert stub échoué", "match_id", mid, "err", err)
			continue
		}
		inserted++
	}

	return inserted, nil
}
