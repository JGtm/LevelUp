// Package port — achievements.go : interfaces pour les achievements Xbox.
package port

import (
	"context"

	"levelup/go-api/internal/domain"
)

// AchievementsRepository fournit la progression d'achievements par joueur.
// Implémenté par platform/duckdb.AchievementsRepo (lit player_achievements
// depuis stats.duckdb).
type AchievementsRepository interface {
	GetPlayerAchievements(ctx context.Context) ([]domain.PlayerAchievementRow, error)
}

// MetadataAchievementsRepository fournit le référentiel bilingue d'achievements.
// Interface séparée de MetadataRepository (interface segregation) : le service
// achievements ne dépend que des méthodes dont il a besoin, pas du référentiel
// saisons/waypoint complet. Implémenté par platform/duckdb.MetadataRepo
// (lit xbox_achievement_definitions depuis metadata.duckdb).
type MetadataAchievementsRepository interface {
	GetAchievementDefinitions(ctx context.Context) ([]domain.AchievementDefinitionRow, error)
}

// AchievementsService construit la réponse de la page Achievements.
// Implémenté par service.AchievementsService.
type AchievementsService interface {
	GetAchievementsPage(ctx context.Context) (domain.AchievementsPageResponse, error)
}
