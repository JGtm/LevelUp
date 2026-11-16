package port

import (
	"context"
	"time"

	"levelup/go-api/internal/domain"
)

// PlayerProfileService : ce que consomme le handler des routes /profile et /activity-calendar. Un
// joueur introuvable est un *ProgressionError de nature ProgErrPlayerNotFound. Implémenté par
// progression/profile.PlayerService.
type PlayerProfileService interface {
	// Build assemble le profil complet (sections A1, A2, B, C) sur la fenêtre de LOWESS donnée.
	Build(ctx context.Context, playerSlug, titleSlug string, windowDays int, now time.Time) (*domain.PlayerProfile, error)
	// ActivityCalendar rend le nombre de matchs par jour UTC entre `since` et `until`.
	ActivityCalendar(ctx context.Context, playerSlug string, since, until time.Time) (domain.ActivityCalendar, error)
}
